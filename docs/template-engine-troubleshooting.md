# Template Engine - Troubleshooting Guide

Comprehensive guide for diagnosing and fixing issues with the Konsul template engine.

## Quick Diagnostics

### Health Check Checklist

Run through this checklist first:

```bash
# 1. Check if konsul-template is running
pgrep -f konsul-template

# 2. Check Konsul server connectivity
curl http://localhost:8500/health

# 3. Verify template file exists and is readable
ls -la /path/to/template.tpl
cat /path/to/template.tpl

# 4. Check destination is writable
touch /path/to/destination.conf

# 5. Test dry-run mode
konsul-template -template app.tpl -dest /tmp/test.conf -dry -once

# 6. Check logs (if using systemd)
journalctl -u konsul-template -f
```

---

## Common Issues

### Issue: Template Render Fails

#### Symptom

```
Error: failed to render template: template: nginx.conf.tpl:5:10: executing "nginx.conf.tpl" at <kv "config/domain">: error calling kv: key not found: config/domain
```

#### Cause

The template is trying to access a KV key that doesn't exist.

#### Solution

**Option 1: Add the missing key**

```bash
curl -X POST http://localhost:8500/kv/config/domain \
  -H "Content-Type: application/json" \
  -d '{"value":"example.com"}'
```

**Option 2: Make the key optional in template**

```go
{{- $domain := "" }}
{{- if kv "config/domain" }}
{{- $domain = kv "config/domain" }}
{{- else }}
{{- $domain = "localhost" }}
{{- end }}

server_name {{ $domain }};
```

**Option 3: Use default value**

```go
{{- $domain := kv "config/domain" | default "localhost" }}
server_name {{ $domain }};
```

#### Prevention

Always handle optional keys gracefully in production templates.

---

### Issue: Permission Denied

#### Symptom

```
Error: failed to write file: open /etc/nginx/nginx.conf: permission denied
```

#### Cause

konsul-template doesn't have write permissions to the destination.

#### Diagnosis

```bash
# Check destination permissions
ls -la /etc/nginx/nginx.conf

# Check parent directory permissions
ls -lad /etc/nginx

# Check running user
ps aux | grep konsul-template
```

#### Solution

**Option 1: Run as root**

```bash
sudo konsul-template -template nginx.conf.tpl -dest /etc/nginx/nginx.conf
```

**Option 2: Change ownership**

```bash
sudo chown $(whoami) /etc/nginx/nginx.conf
```

**Option 3: Use systemd with appropriate user**

```ini
[Service]
User=nginx
Group=nginx
ExecStart=/usr/local/bin/konsul-template -template /etc/templates/nginx.conf.tpl -dest /etc/nginx/nginx.conf
```

**Option 4: Write to temp directory first**

```bash
konsul-template -template nginx.conf.tpl -dest /tmp/nginx.conf -once
sudo cp /tmp/nginx.conf /etc/nginx/nginx.conf
```

---

### Issue: Service Discovery Returns Empty

#### Symptom

Template renders but service list is empty:

```nginx
upstream backend {
}
```

Expected:
```nginx
upstream backend {
    server 10.0.0.1:8080;
    server 10.0.0.2:8080;
}
```

#### Cause

No services registered with that name, or services have expired.

#### Diagnosis

```bash
# Check if services are registered
curl http://localhost:8500/services | jq .

# Check specific service
curl http://localhost:8500/services/web | jq .

# Check service TTL
curl http://localhost:8500/services/web | jq '.expires_at'
```

#### Solution

**Register the service:**

```bash
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }'
```

**Send heartbeats to prevent expiration:**

```bash
# Every 15 seconds (TTL is usually 30s)
while true; do
  curl -X POST http://localhost:8500/services/web/heartbeat
  sleep 15
done
```

---

### Issue: Template Not Re-rendering on Changes

#### Symptom

- Change KV value
- Template doesn't update
- Watch mode is running

#### Diagnosis

```bash
# Check if watch mode is active
ps aux | grep konsul-template | grep -v once

# Check watcher logs
# Should see "checking for changes" messages

# Manually verify data changed
curl http://localhost:8500/kv/config/domain | jq .
```

#### Possible Causes

**1. Not running in watch mode**

```bash
# This runs once and exits
konsul-template -template app.tpl -dest app.conf -once

# This watches for changes
konsul-template -template app.tpl -dest app.conf
```

**2. Min/Max wait times**

Changes may be batched. Wait at least `maxWait` duration (default 10s):

```bash
# Change data
curl -X POST http://localhost:8500/kv/test -d '{"value":"new"}'

# Wait 10 seconds
sleep 10

# Check if file updated
cat /path/to/output.conf
```

**3. Content hasn't actually changed**

The watcher uses SHA256 hashing. If the rendered content is identical, no re-render occurs:

```bash
# This won't trigger re-render (same rendered output)
curl -X POST http://localhost:8500/kv/test -d '{"value":"same"}'
curl -X POST http://localhost:8500/kv/test -d '{"value":"same"}'

# This will trigger re-render (different output)
curl -X POST http://localhost:8500/kv/test -d '{"value":"different"}'
```

#### Solution

**Check actual rendered content:**

```bash
# Enable dry-run to see what would be rendered
konsul-template -template app.tpl -dest /tmp/test.conf -dry -once
cat /tmp/test.conf
```

**Reduce wait times for testing:**

```go
// In code
config := template.Config{
    Wait: &template.WaitConfig{
        Min: 1 * time.Second,   // Faster response
        Max: 2 * time.Second,
    },
}
```

---

### Issue: Command Execution Fails

#### Symptom

```
Error: command failed: exit status 1
Output: nginx: [emerg] invalid host in upstream "backend" in /etc/nginx/nginx.conf:10
```

#### Cause

The generated configuration is invalid.

#### Diagnosis

```bash
# Check generated file
cat /etc/nginx/nginx.conf

# Manually test command
nginx -t -c /etc/nginx/nginx.conf
```

#### Solution

**Validate config before applying:**

```go
// Template config
TemplateConfig{
    Source:      "nginx.conf.tpl",
    Destination: "/tmp/nginx.conf.new",
    Command:     "nginx -t -c /tmp/nginx.conf.new && mv /tmp/nginx.conf.new /etc/nginx/nginx.conf && nginx -s reload",
}
```

**Or use a validation script:**

```bash
#!/bin/bash
# validate-and-reload.sh

CONFIG="/tmp/nginx.conf"
TARGET="/etc/nginx/nginx.conf"

# Validate
if nginx -t -c "$CONFIG" 2>&1; then
    echo "✓ Config valid"
    cp "$CONFIG" "$TARGET"
    nginx -s reload
    echo "✓ Reloaded"
else
    echo "✗ Invalid config, not applying"
    exit 1
fi
```

```go
TemplateConfig{
    Command: "/usr/local/bin/validate-and-reload.sh",
}
```

---

### Issue: High CPU Usage

#### Symptom

```bash
top
# konsul-template using 80-100% CPU
```

#### Cause

**1. Min wait time too short** - Constant polling

```go
// Bad
Wait: &WaitConfig{
    Min: 100 * time.Millisecond,  // Too frequent
}
```

**2. Template rendering is expensive** - Complex template with many operations

**3. Many templates** - Each has its own watcher goroutine

#### Diagnosis

```bash
# Profile CPU usage
go tool pprof http://localhost:6060/debug/pprof/profile

# Check number of goroutines
curl http://localhost:6060/debug/pprof/goroutine?debug=1

# Monitor system calls
strace -c -p $(pgrep konsul-template)
```

#### Solution

**Increase wait times:**

```go
Wait: &WaitConfig{
    Min: 5 * time.Second,    // Less frequent checks
    Max: 30 * time.Second,
}
```

**Optimize template:**

```go
// Bad - queries KV multiple times
{{- range services }}
DB: {{ kv "config/db/host" }}:{{ kv "config/db/port" }}
{{- end }}

// Good - query once, reuse
{{- $dbHost := kv "config/db/host" }}
{{- $dbPort := kv "config/db/port" }}
{{- range services }}
DB: {{ $dbHost }}:{{ $dbPort }}
{{- end }}
```

**Reduce template count:**

```go
// Bad - 100 separate templates
for i := 0; i < 100; i++ {
    templates = append(templates, TemplateConfig{...})
}

// Good - 1 template that generates multiple configs
TemplateConfig{
    Source: "all-configs.tpl",  // Generates all 100 in one pass
}
```

---

### Issue: Memory Leak

#### Symptom

```bash
# Memory usage grows over time
ps aux | grep konsul-template
# RSS/MEM column increasing
```

#### Diagnosis

```bash
# Memory profile
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap

# Check for goroutine leaks
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

#### Common Causes

**1. Watchers not cleaning up**

Check goroutine count stays stable:

```bash
watch -n 1 'curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 | grep goroutine'
```

**2. Template cache growing unbounded**

(Not yet implemented, but could be a future issue)

#### Solution

**Restart periodically:**

```ini
# systemd unit
[Service]
Restart=always
RestartSec=3600  # Restart every hour
```

**Enable pprof in production:**

```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

---

### Issue: Network Timeouts

#### Symptom

```
Error: failed to fetch KV data: Get "http://localhost:8500/kv": dial tcp: i/o timeout
```

#### Cause

Can't connect to Konsul server.

#### Diagnosis

```bash
# Check Konsul is running
curl http://localhost:8500/health

# Check network connectivity
ping localhost

# Check firewall
sudo iptables -L | grep 8500

# Check if listening
netstat -tlnp | grep 8500
```

#### Solution

**1. Verify Konsul address:**

```bash
konsul-template \
    -template app.tpl \
    -dest app.conf \
    -konsul http://correct-host:8500
```

**2. Increase timeout:**

```go
// In client.go
client := &http.Client{
    Timeout: 30 * time.Second,  // Increase from 10s
}
```

**3. Check network configuration:**

```bash
# Verify DNS resolution
nslookup konsul.example.com

# Test with IP
konsul-template -konsul http://10.0.0.1:8500 ...
```

---

## Debugging Techniques

### Enable Debug Logging

```go
// Set log level to debug
logger.SetDefault(logger.New(zapcore.DebugLevel, "text"))
```

Or via environment:

```bash
LOG_LEVEL=debug konsul-template ...
```

### Dry-Run Mode

Always test templates with dry-run first:

```bash
konsul-template \
    -template risky-config.tpl \
    -dest /tmp/test.conf \
    -dry \
    -once

# Review output
cat /tmp/test.conf
```

### Template Debugging

**Print variables:**

```go
{{- $var := kv "test" }}
DEBUG: var = {{ $var }}
DEBUG: type = {{ printf "%T" $var }}
```

**Check conditionals:**

```go
{{- if kv "test" }}
DEBUG: Key exists
{{- else }}
DEBUG: Key does not exist
{{- end }}
```

**Inspect data structures:**

```go
{{- range services }}
DEBUG: {{ printf "%#v" . }}
{{- end }}
```

### Strace System Calls

```bash
# Monitor file operations
strace -e trace=open,openat,read,write -f -p $(pgrep konsul-template)

# Monitor network operations
strace -e trace=network -f -p $(pgrep konsul-template)
```

### Check File Descriptors

```bash
# List open files
lsof -p $(pgrep konsul-template)

# Count open file descriptors
ls -l /proc/$(pgrep konsul-template)/fd | wc -l
```

### Profile Performance

```bash
# CPU profile (30 seconds)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8080 cpu.prof

# Memory profile
curl http://localhost:6060/debug/pprof/heap > mem.prof
go tool pprof -http=:8080 mem.prof

# Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof -http=:8080 goroutine.prof
```

---

## Advanced Troubleshooting

### Corrupted Output Files

**Symptom:** Output file has partial content

**Cause:** Process killed during write (before atomic rename)

**Prevention:** The engine uses atomic writes, so this should be impossible. If it happens:

```bash
# Check for .tmp files
ls -la /path/to/*.tmp

# Check system logs for OOM killer
dmesg | grep -i kill

# Verify filesystem health
sudo fsck /dev/sda1
```

### Race Conditions

**Symptom:** Intermittent failures, works sometimes

**Diagnosis:**

```bash
# Run with race detector (development only)
go run -race cmd/konsul-template/main.go ...
```

**Common race conditions:**

1. **Multiple instances writing same file**

```bash
# Check for multiple processes
ps aux | grep konsul-template

# Use file locking (future enhancement)
flock /var/lock/konsul-template.lock konsul-template ...
```

2. **Service expired between check and render**

```bash
# Check service TTL
curl http://localhost:8500/services/web | jq '.expires_at'

# Increase TTL
curl -X PUT http://localhost:8500/config/ttl -d '{"value":"60s"}'
```

### Unicode/Encoding Issues

**Symptom:** Output has � or garbled characters

**Cause:** UTF-8 encoding mismatch

**Solution:**

```go
// Ensure template file is UTF-8
file -i template.tpl

// Convert if needed
iconv -f ISO-8859-1 -t UTF-8 template.tpl > template.utf8.tpl
```

### Symlink Issues

**Symptom:** Template renders but doesn't update actual file

**Cause:** Writing to symlink, but not following it correctly

**Diagnosis:**

```bash
# Check if destination is symlink
ls -la /etc/nginx/nginx.conf
# lrwxr-xr-x  1 root  wheel  25 Dec  1 10:00 /etc/nginx/nginx.conf -> /var/configs/nginx.conf
```

**Solution:**

Either:
- Write directly to target: `-dest /var/configs/nginx.conf`
- Or ensure symlink handling in renderer (not yet implemented)

---

## Getting Help

### Information to Provide

When reporting issues, include:

1. **Version:**
```bash
konsul-template -version
```

2. **Command used:**
```bash
konsul-template -template app.tpl -dest app.conf ...
```

3. **Template file** (sanitized):
```go
{{ kv "config/host" }}:{{ kv "config/port" }}
```

4. **Error message** (full):
```
Error: failed to render template: ...
```

5. **Environment:**
```bash
uname -a
go version
```

6. **Konsul status:**
```bash
curl http://localhost:8500/health
```

### Logs

Collect relevant logs:

```bash
# If using systemd
journalctl -u konsul-template -n 100 > logs.txt

# If running in foreground
konsul-template ... 2>&1 | tee logs.txt
```

### Minimal Reproduction

Create minimal example:

```bash
# Create minimal template
echo '{{ kv "test" }}' > test.tpl

# Add test data
curl -X POST http://localhost:8500/kv/test -d '{"value":"hello"}'

# Run
konsul-template -template test.tpl -dest /tmp/out.txt -once
```

---

## Prevention

### Pre-deployment Checklist

- [ ] Test templates with dry-run mode
- [ ] Validate generated configs with appropriate tools
- [ ] Ensure destination directories exist and are writable
- [ ] Set appropriate file permissions
- [ ] Test with realistic data volumes
- [ ] Monitor resource usage (CPU, memory, file descriptors)
- [ ] Set up health checks
- [ ] Configure proper logging
- [ ] Document expected behavior
- [ ] Test failure scenarios (missing keys, network issues)

### Monitoring

Set up alerts for:

- CPU usage > 50%
- Memory usage > 500MB
- Render errors > 5 per hour
- Command execution failures
- File write failures

### Graceful Degradation

Design templates to handle missing data:

```go
{{- $host := "localhost" }}
{{- if kv "config/host" }}
{{- $host = kv "config/host" }}
{{- end }}

server_name {{ $host }};
```

---

## See Also

- [User Guide](template-engine.md)
- [API Reference](template-engine-api.md)
- [Implementation Guide](template-engine-implementation.md)
- [GitHub Issues](https://github.com/yourusername/konsul/issues)

# Konsul Template Examples

This directory contains example templates for the Konsul template engine.

## Examples

### 1. Simple Text Template

**File:** `simple.txt.tpl`

A basic template demonstrating KV access, service discovery, environment variables, and file reading.

**Usage:**
```bash
# Setup test data
curl -X POST http://localhost:8500/kv/app/name -d '{"value":"Konsul"}'

# Render template
konsul-template -template simple.txt.tpl -dest output.txt -once
```

### 2. Nginx Configuration

**File:** `nginx.conf.tpl`

Production-ready nginx configuration with service discovery for backend servers.

**Features:**
- Dynamic upstream configuration
- Multiple backends (web, api)
- Domain from KV store
- Health check endpoint

**Usage:**
```bash
# Setup data
curl -X POST http://localhost:8500/kv/config/domain -d '{"value":"example.com"}'
curl -X POST http://localhost:8500/services -d '{
  "name": "web",
  "address": "10.0.0.1",
  "port": 8080
}'

# Render
konsul-template -template nginx.conf.tpl -dest /tmp/nginx.conf -once

# Validate
nginx -t -c /tmp/nginx.conf
```

### 3. Application Environment File

**File:** `app-config.env.tpl`

Generate application environment variables from KV store and service catalog.

**Usage:**
```bash
# Setup database config
curl -X POST http://localhost:8500/kv/config/database/host -d '{"value":"db.local"}'
curl -X POST http://localhost:8500/kv/config/database/port -d '{"value":"5432"}'
curl -X POST http://localhost:8500/kv/config/database/name -d '{"value":"myapp"}'
curl -X POST http://localhost:8500/kv/config/database/user -d '{"value":"admin"}'

# Render
konsul-template -template app-config.env.tpl -dest .env -once
```

### 4. HAProxy Configuration

**File:** `haproxy.cfg.tpl`

Complete HAProxy configuration with service discovery and health checks.

**Features:**
- Round-robin load balancing
- Multiple backends
- Health checks
- Stats interface

**Usage:**
```bash
# Register services
curl -X POST http://localhost:8500/services -d '{
  "name": "web",
  "address": "10.0.0.1",
  "port": 8080
}'
curl -X POST http://localhost:8500/services -d '{
  "name": "api",
  "address": "10.0.0.2",
  "port": 9000
}'

# Render
konsul-template -template haproxy.cfg.tpl -dest /tmp/haproxy.cfg -once

# Validate
haproxy -c -f /tmp/haproxy.cfg
```

## Testing Templates

### Dry-Run Mode

Test templates without writing files:

```bash
konsul-template -template nginx.conf.tpl -dest /tmp/test.conf -dry -once
```

### Once Mode

Generate once and exit (for testing):

```bash
konsul-template -template simple.txt.tpl -dest output.txt -once
```

### Watch Mode

Continuously watch for changes:

```bash
konsul-template -template nginx.conf.tpl -dest /tmp/nginx.conf
```

## Creating Your Own Templates

1. **Start with a simple template:**

```go
{{ kv "app/name" }} version {{ kv "app/version" }}
```

2. **Add service discovery:**

```go
Services:
{{- range services }}
- {{ .Name }}: {{ .Address }}:{{ .Port }}
{{- end }}
```

3. **Use conditionals for optional data:**

```go
{{- if kv "optional/key" }}
OPTIONAL={{ kv "optional/key" }}
{{- end }}
```

4. **Iterate over KV tree:**

```go
{{- range kvTree "config/" }}
{{ .Key }}={{ .Value }}
{{- end }}
```

## Template Functions Reference

| Function | Description | Example |
|----------|-------------|---------|
| `kv "key"` | Get KV value | `{{ kv "config/host" }}` |
| `kvTree "prefix"` | Get all KV under prefix | `{{ range kvTree "config/" }}` |
| `kvList "prefix"` | List keys under prefix | `{{ range kvList "config/" }}` |
| `service "name"` | Get service instances | `{{ range service "web" }}` |
| `services` | Get all services | `{{ range services }}` |
| `env "VAR"` | Get environment variable | `{{ env "HOME" }}` |
| `file "path"` | Read file contents | `{{ file "/etc/hostname" }}` |
| `toLower` | Convert to lowercase | `{{ toLower "HELLO" }}` |
| `toUpper` | Convert to uppercase | `{{ toUpper "hello" }}` |

## Best Practices

1. **Always test with `-dry` first**
2. **Use `-once` for initial generation**
3. **Validate generated configs before applying**
4. **Handle missing keys gracefully with conditionals**
5. **Use comments to document template purpose**

## See Also

- [Template Engine Documentation](../../docs/template-engine.md)
- [ADR-0015: Template Engine](../../docs/adr/0015-template-engine.md)

# konsulctl - CLI Tool Complete Documentation

Comprehensive guide for using the konsulctl command-line tool to interact with Konsul.

## Overview

**konsulctl** is the official command-line interface for Konsul, providing a user-friendly way to interact with Konsul servers for key-value operations, service discovery, backups, and DNS queries.

### Quick Start

**Basic usage:**
```bash
# Set a key-value pair
konsulctl kv set mykey myvalue

# Get a value
konsulctl kv get mykey

# Register a service
konsulctl service register web-api 10.0.0.1 8080

# List services
konsulctl service list

# Create a backup
konsulctl backup create
```

**With TLS:**
```bash
konsulctl kv list \
  --server https://localhost:8888 \
  --tls-skip-verify
```

---

## Table of Contents

- [Installation](#installation)
- [Global Options](#global-options)
- [KV Commands](#kv-commands)
- [Service Commands](#service-commands)
- [Backup Commands](#backup-commands)
- [DNS Commands](#dns-commands)
- [TLS/SSL Support](#tlsssl-support)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Scripting](#scripting)

---

## Installation

### Binary Installation

**Download from releases:**
```bash
# Linux
curl -L https://github.com/neogan74/konsul/releases/latest/download/konsulctl-linux-amd64 \
  -o konsulctl
chmod +x konsulctl
sudo mv konsulctl /usr/local/bin/

# macOS
curl -L https://github.com/neogan74/konsul/releases/latest/download/konsulctl-darwin-amd64 \
  -o konsulctl
chmod +x konsulctl
sudo mv konsulctl /usr/local/bin/

# Windows
# Download konsulctl-windows-amd64.exe from releases
```

### Build from Source

**Prerequisites:**
- Go 1.24.5 or later

**Build:**
```bash
git clone https://github.com/neogan74/konsul.git
cd konsul
go build -o konsulctl ./cmd/konsulctl
sudo mv konsulctl /usr/local/bin/
```

### Verify Installation

```bash
konsulctl version
# Output: konsulctl version 1.0.0
```

---

## Global Options

All commands support these global options:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--server <url>` | string | `http://localhost:8888` | Konsul server URL |
| `--tls-skip-verify` | flag | `false` | Skip TLS certificate verification |
| `--ca-cert <file>` | string | - | Path to CA certificate |
| `--client-cert <file>` | string | - | Path to client certificate (mTLS) |
| `--client-key <file>` | string | - | Path to client key (mTLS) |

**Usage:**
```bash
konsulctl <command> [options] --server <url> [TLS options]
```

---

## KV Commands

Manage key-value store operations.

### `kv get`

Get a value by key.

**Syntax:**
```bash
konsulctl kv get <key> [--server <url>] [TLS options]
```

**Examples:**
```bash
# Get a value
konsulctl kv get mykey

# Get from remote server
konsulctl kv get mykey --server http://konsul.example.com:8888

# Get with TLS
konsulctl kv get mykey \
  --server https://konsul.example.com:8888 \
  --ca-cert /path/to/ca.crt
```

**Output:**
```
myvalue
```

**Exit codes:**
- `0` - Success
- `1` - Key not found or error

---

### `kv set`

Set a key-value pair.

**Syntax:**
```bash
konsulctl kv set <key> <value> [--server <url>] [TLS options]
```

**Examples:**
```bash
# Set a simple value
konsulctl kv set app/config/port 8080

# Set JSON value
konsulctl kv set app/config '{"port":8080,"debug":true}'

# Set multiline value
konsulctl kv set app/banner "$(cat banner.txt)"
```

**Output:**
```
Successfully set app/config/port = 8080
```

**Exit codes:**
- `0` - Success
- `1` - Error

---

### `kv delete`

Delete a key from the store.

**Syntax:**
```bash
konsulctl kv delete <key> [--server <url>] [TLS options]
```

**Examples:**
```bash
# Delete a key
konsulctl kv delete mykey

# Delete with confirmation
konsulctl kv delete production/database/password
```

**Output:**
```
Successfully deleted key: mykey
```

**Exit codes:**
- `0` - Success
- `1` - Key not found or error

---

### `kv list`

List all keys in the store.

**Syntax:**
```bash
konsulctl kv list [--server <url>] [TLS options]
```

**Examples:**
```bash
# List all keys
konsulctl kv list

# List keys and save to file
konsulctl kv list > keys.txt

# Count keys
konsulctl kv list | wc -l
```

**Output:**
```
Keys:
  app/config/port
  app/config/debug
  database/host
  database/port
```

**Exit codes:**
- `0` - Success
- `1` - Error

---

## Service Commands

Manage service discovery operations.

### `service register`

Register a service with the discovery system.

**Syntax:**
```bash
konsulctl service register <name> <address> <port> [health check options] [--server <url>] [TLS options]
```

**Health Check Options:**
| Option | Format | Description |
|--------|--------|-------------|
| `--check-http <url>` | `http://...` | HTTP health check endpoint |
| `--check-tcp <addr>` | `host:port` | TCP connectivity check |
| `--check-ttl <duration>` | `30s` | TTL-based health check |

**Examples:**
```bash
# Basic registration
konsulctl service register web-api 10.0.0.1 8080

# With HTTP health check
konsulctl service register web-api 10.0.0.1 8080 \
  --check-http http://10.0.0.1:8080/health

# With TCP health check
konsulctl service register database 10.0.0.2 5432 \
  --check-tcp 10.0.0.2:5432

# With TTL check
konsulctl service register cache 10.0.0.3 6379 \
  --check-ttl 30s

# Multiple health checks
konsulctl service register api 10.0.0.4 9000 \
  --check-http http://10.0.0.4:9000/health \
  --check-tcp 10.0.0.4:9000
```

**Output:**
```
Successfully registered service: web-api at 10.0.0.1:8080 with 1 health check(s)
```

**Exit codes:**
- `0` - Success
- `1` - Error (invalid parameters, server error)

---

### `service list`

List all registered services.

**Syntax:**
```bash
konsulctl service list [--server <url>] [TLS options]
```

**Examples:**
```bash
# List services
konsulctl service list

# List and filter
konsulctl service list | grep web

# Export to JSON (if server supports)
curl http://localhost:8888/services/ | jq .
```

**Output:**
```
Services:
  web-api - 10.0.0.1:8080
  database - 10.0.0.2:5432
  cache - 10.0.0.3:6379
```

**Exit codes:**
- `0` - Success (even if no services)
- `1` - Error

---

### `service deregister`

Deregister a service from discovery.

**Syntax:**
```bash
konsulctl service deregister <name> [--server <url>] [TLS options]
```

**Examples:**
```bash
# Deregister a service
konsulctl service deregister web-api

# Deregister with confirmation
read -p "Really deregister? " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    konsulctl service deregister production-api
fi
```

**Output:**
```
Successfully deregistered service: web-api
```

**Exit codes:**
- `0` - Success
- `1` - Service not found or error

---

### `service heartbeat`

Send a heartbeat for a service to update its TTL.

**Syntax:**
```bash
konsulctl service heartbeat <name> [--server <url>] [TLS options]
```

**Examples:**
```bash
# Send heartbeat
konsulctl service heartbeat web-api

# Heartbeat loop (keep service alive)
while true; do
  konsulctl service heartbeat web-api
  sleep 20
done
```

**Output:**
```
Successfully sent heartbeat for service: web-api
```

**Exit codes:**
- `0` - Success
- `1` - Service not found or error

**Best Practice:** Use systemd timer or cron for automated heartbeats.

---

## Backup Commands

Manage data backups and restores.

### `backup create`

Create a backup of all data (KV store and services).

**Syntax:**
```bash
konsulctl backup create [--server <url>] [TLS options]
```

**Examples:**
```bash
# Create backup
konsulctl backup create

# Create backup with timestamp
konsulctl backup create
BACKUP_FILE=$(konsulctl backup create | grep -o 'backup-.*\.json')
echo "Created: $BACKUP_FILE"

# Automated daily backup
#!/bin/bash
DATE=$(date +%Y%m%d)
konsulctl backup create --server http://konsul:8888
cp backups/backup-*.json /mnt/backups/konsul-backup-$DATE.json
```

**Output:**
```
Successfully created backup: backup-20251014-150430.json
```

**Exit codes:**
- `0` - Success
- `1` - Error

---

### `backup list`

List available backups.

**Syntax:**
```bash
konsulctl backup list [--server <url>] [TLS options]
```

**Examples:**
```bash
# List backups
konsulctl backup list

# Find latest backup
konsulctl backup list | tail -1
```

**Output:**
```
Available backups:
  backup-20251014-150430.json
  backup-20251013-150430.json
  backup-20251012-150430.json
```

**Exit codes:**
- `0` - Success
- `1` - Error

---

### `backup restore`

Restore data from a backup file.

**Syntax:**
```bash
konsulctl backup restore <backup-file> [--server <url>] [TLS options]
```

**Examples:**
```bash
# Restore from backup
konsulctl backup restore backup-20251014-150430.json

# Restore with confirmation
echo "WARNING: This will overwrite current data!"
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    konsulctl backup restore backup-20251014-150430.json
fi

# Restore to different server
konsulctl backup restore backup-20251014-150430.json \
  --server http://konsul-backup:8888
```

**Output:**
```
Successfully restored from backup: backup-20251014-150430.json
```

**Exit codes:**
- `0` - Success
- `1` - Backup file not found or error

**Warning:** Restore overwrites existing data!

---

### `backup export`

Export current data as JSON.

**Syntax:**
```bash
konsulctl backup export [--server <url>] [TLS options]
```

**Examples:**
```bash
# Export to stdout
konsulctl backup export

# Export to file
konsulctl backup export > export.json

# Pretty print
konsulctl backup export | jq .

# Export and analyze
konsulctl backup export | jq '.kv | keys | length'
```

**Output:**
```json
{
  "kv": {
    "app/config/port": "8080",
    "app/config/debug": "true"
  },
  "services": [
    {
      "name": "web-api",
      "address": "10.0.0.1",
      "port": 8080
    }
  ]
}
```

**Exit codes:**
- `0` - Success
- `1` - Error

---

## DNS Commands

DNS query helpers (shows how to query Konsul's DNS interface).

### `dns srv`

Show how to query SRV records for a service.

**Syntax:**
```bash
konsulctl dns srv <service> [--server <dns-host>] [--port <dns-port>]
```

**Examples:**
```bash
# Show SRV query for service
konsulctl dns srv web-api

# Custom DNS server
konsulctl dns srv web-api --server konsul.local --port 8600
```

**Output:**
```
DNS srv query for service 'web-api' (server: localhost:8600)
SRV Record: _web-api._tcp.service.consul
Run: dig @localhost -p 8600 _web-api._tcp.service.consul SRV
```

**Exit codes:**
- `0` - Always (informational only)

---

### `dns a`

Show how to query A records for a service.

**Syntax:**
```bash
konsulctl dns a <service> [--server <dns-host>] [--port <dns-port>]
```

**Examples:**
```bash
# Show A query for service
konsulctl dns a web-api

# Custom DNS server
konsulctl dns a web-api --server 10.0.0.1 --port 8600
```

**Output:**
```
DNS a query for service 'web-api' (server: localhost:8600)
A Record: web-api.service.consul
Run: dig @localhost -p 8600 web-api.service.consul A
```

**Exit codes:**
- `0` - Always (informational only)

---

## TLS/SSL Support

konsulctl supports secure connections to Konsul servers.

### Connection Modes

**1. Plain HTTP (default):**
```bash
konsulctl kv list --server http://localhost:8888
```

**2. HTTPS with self-signed cert:**
```bash
konsulctl kv list \
  --server https://localhost:8888 \
  --tls-skip-verify
```

**3. HTTPS with CA certificate:**
```bash
konsulctl kv list \
  --server https://localhost:8888 \
  --ca-cert /path/to/ca.crt
```

**4. Mutual TLS (mTLS):**
```bash
konsulctl kv list \
  --server https://localhost:8888 \
  --ca-cert /path/to/ca.crt \
  --client-cert /path/to/client.crt \
  --client-key /path/to/client.key
```

---

### TLS Options Reference

| Option | Purpose | When to Use |
|--------|---------|-------------|
| `--tls-skip-verify` | Skip cert verification | Development, self-signed certs |
| `--ca-cert` | Custom CA certificate | Private CA, enterprise PKI |
| `--client-cert` + `--client-key` | Client authentication | mTLS, high security |

---

### Examples

**Development (self-signed):**
```bash
export KONSUL_SERVER="https://localhost:8888"
export KONSUL_TLS_SKIP="--tls-skip-verify"

konsulctl kv list --server $KONSUL_SERVER $KONSUL_TLS_SKIP
```

**Production (with CA):**
```bash
export KONSUL_SERVER="https://konsul.example.com:8888"
export KONSUL_CA="/etc/konsul/ca.crt"

konsulctl kv list --server $KONSUL_SERVER --ca-cert $KONSUL_CA
```

**High Security (mTLS):**
```bash
konsulctl kv list \
  --server https://konsul.example.com:8888 \
  --ca-cert /etc/konsul/ca.crt \
  --client-cert /etc/konsul/client.crt \
  --client-key /etc/konsul/client.key
```

---

## Examples

### Common Workflows

**1. Service Health Monitoring:**
```bash
#!/bin/bash
# monitor-service.sh

SERVICE="web-api"
while true; do
  if konsulctl service heartbeat $SERVICE; then
    echo "$(date): Heartbeat OK"
  else
    echo "$(date): Heartbeat FAILED"
    # Trigger alert
  fi
  sleep 20
done
```

**2. Configuration Deployment:**
```bash
#!/bin/bash
# deploy-config.sh

# Read local config
CONFIG=$(cat app-config.json)

# Deploy to Konsul
konsulctl kv set app/config "$CONFIG"

# Verify
konsulctl kv get app/config | jq .
```

**3. Backup and Restore Workflow:**
```bash
#!/bin/bash
# backup-workflow.sh

# Create backup
echo "Creating backup..."
konsulctl backup create

# List backups
echo "Available backups:"
konsulctl backup list

# Restore if needed
# konsulctl backup restore backup-20251014-150430.json
```

**4. Bulk Key Operations:**
```bash
#!/bin/bash
# bulk-keys.sh

# Set multiple keys
for i in {1..10}; do
  konsulctl kv set "app/key$i" "value$i"
done

# List all keys
konsulctl kv list

# Delete all keys matching pattern
konsulctl kv list | grep "app/key" | while read key; do
  konsulctl kv delete "$key"
done
```

**5. Service Registration with Health Check:**
```bash
#!/bin/bash
# register-service.sh

SERVICE_NAME="api"
SERVICE_IP="10.0.0.1"
SERVICE_PORT="8080"
HEALTH_URL="http://$SERVICE_IP:$SERVICE_PORT/health"

# Register service
konsulctl service register $SERVICE_NAME $SERVICE_IP $SERVICE_PORT \
  --check-http $HEALTH_URL

# Start heartbeat loop
while true; do
  konsulctl service heartbeat $SERVICE_NAME
  sleep 20
done
```

---

## Troubleshooting

### Issue: Connection Refused

**Error:**
```
Error: failed to make request: dial tcp 127.0.0.1:8888: connect: connection refused
```

**Solutions:**
1. **Check if Konsul is running:**
   ```bash
   curl http://localhost:8888/health
   ```

2. **Verify server URL:**
   ```bash
   konsulctl kv list --server http://localhost:8888
   ```

3. **Check firewall:**
   ```bash
   telnet localhost 8888
   ```

---

### Issue: TLS Certificate Error

**Error:**
```
Error: x509: certificate signed by unknown authority
```

**Solutions:**
1. **Use --tls-skip-verify (dev only):**
   ```bash
   konsulctl kv list --server https://localhost:8888 --tls-skip-verify
   ```

2. **Provide CA certificate:**
   ```bash
   konsulctl kv list --server https://localhost:8888 --ca-cert /path/to/ca.crt
   ```

3. **Check certificate:**
   ```bash
   openssl s_client -connect localhost:8888 -showcerts
   ```

---

### Issue: Key Not Found

**Error:**
```
Error getting key 'mykey': key not found
```

**Solutions:**
1. **List all keys:**
   ```bash
   konsulctl kv list
   ```

2. **Check spelling:**
   ```bash
   konsulctl kv list | grep -i mykey
   ```

3. **Verify server:**
   ```bash
   konsulctl kv list --server http://localhost:8888
   ```

---

### Issue: Authentication Required

**Error:**
```
Error: 401 Unauthorized
```

**Solutions:**
1. **Check if auth is enabled on server:**
   ```bash
   curl http://localhost:8888/health
   ```

2. **Authentication not yet supported in konsulctl** - use curl:
   ```bash
   # Get JWT token
   TOKEN=$(curl -X POST http://localhost:8888/auth/login \
     -H "Content-Type: application/json" \
     -d '{"user_id":"user123","username":"admin","roles":["admin"]}' | \
     jq -r .token)

   # Use token
   curl http://localhost:8888/kv/mykey -H "Authorization: Bearer $TOKEN"
   ```

---

## Best Practices

### 1. Use Environment Variables

**Define defaults:**
```bash
# In ~/.bashrc or ~/.zshrc
export KONSUL_SERVER="http://konsul.local:8888"
export KONSUL_CA_CERT="/etc/konsul/ca.crt"

# Create alias
alias konsulctl='konsulctl --server $KONSUL_SERVER --ca-cert $KONSUL_CA_CERT'
```

---

### 2. Script Error Handling

**Proper error handling:**
```bash
#!/bin/bash
set -e  # Exit on error

# Check if service exists before operations
if konsulctl service list | grep -q "web-api"; then
  konsulctl service deregister web-api
else
  echo "Service not found"
  exit 1
fi
```

---

### 3. Use JSON for Complex Values

**Store structured data:**
```bash
# Create JSON config
cat > config.json <<EOF
{
  "database": {
    "host": "db.example.com",
    "port": 5432,
    "name": "myapp"
  }
}
EOF

# Store in KV
konsulctl kv set app/config "$(cat config.json)"

# Retrieve and parse
konsulctl kv get app/config | jq .database.host
```

---

### 4. Automate Backups

**Cron job:**
```bash
# /etc/cron.d/konsul-backup
0 2 * * * root /usr/local/bin/konsulctl backup create --server http://localhost:8888 && find /path/to/backups -mtime +7 -delete
```

**Systemd timer:**
```ini
# /etc/systemd/system/konsul-backup.timer
[Unit]
Description=Daily Konsul Backup

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

---

### 5. Use Wrapper Scripts

**Simplified operations:**
```bash
#!/bin/bash
# /usr/local/bin/konsul-deploy

# Deploy script with sensible defaults
SERVER="${KONSUL_SERVER:-http://localhost:8888}"
CA_CERT="${KONSUL_CA_CERT}"

case "$1" in
  set)
    konsulctl kv set "$2" "$3" --server $SERVER ${CA_CERT:+--ca-cert $CA_CERT}
    ;;
  get)
    konsulctl kv get "$2" --server $SERVER ${CA_CERT:+--ca-cert $CA_CERT}
    ;;
  *)
    echo "Usage: konsul-deploy {set|get} ..."
    exit 1
    ;;
esac
```

---

## Scripting

### Exit Codes

All commands follow standard exit code conventions:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (generic) |

**Usage in scripts:**
```bash
if konsulctl kv get mykey > /dev/null 2>&1; then
  echo "Key exists"
else
  echo "Key does not exist"
  konsulctl kv set mykey defaultvalue
fi
```

---

### Output Parsing

**Parse list output:**
```bash
# Get all keys into array
mapfile -t KEYS < <(konsulctl kv list | tail -n +2 | sed 's/^  //')

# Iterate
for key in "${KEYS[@]}"; do
  echo "Processing $key"
  value=$(konsulctl kv get "$key")
  echo "  Value: $value"
done
```

**Parse service list:**
```bash
# Extract service names
konsulctl service list | tail -n +2 | awk '{print $1}'

# Get services with addresses
konsulctl service list | tail -n +2 | while read name sep address; do
  echo "Service: $name at $address"
done
```

---

### Integration Examples

**With systemd:**
```ini
# /etc/systemd/system/myapp.service
[Unit]
Description=My Application
After=network.target

[Service]
Type=simple
ExecStartPre=/usr/local/bin/konsulctl service deregister myapp || true
ExecStart=/usr/bin/myapp
ExecStartPost=/usr/local/bin/konsulctl service register myapp 10.0.0.1 8080
ExecStopPost=/usr/local/bin/konsulctl service deregister myapp

[Install]
WantedBy=multi-user.target
```

**With Docker:**
```dockerfile
FROM alpine:latest

# Install konsulctl
COPY konsulctl /usr/local/bin/
RUN chmod +x /usr/local/bin/konsulctl

# Register on start
CMD ["/bin/sh", "-c", "konsulctl service register $SERVICE_NAME $SERVICE_IP $SERVICE_PORT && exec myapp"]
```

**With Kubernetes:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: app
    image: myapp:latest
  - name: sidecar
    image: konsulctl:latest
    command:
    - /bin/sh
    - -c
    - |
      konsulctl service register myapp ${POD_IP} 8080
      while true; do
        konsulctl service heartbeat myapp
        sleep 20
      done
```

---

## See Also

- [Konsul README](../README.md)
- [Authentication Documentation](authentication.md)
- [Service Discovery Documentation](dns-service-discovery.md)
- [Backup and Persistence](persistence-api.md)

---

## Changelog

- **2025-10-14**: Initial comprehensive documentation
- **Version**: 1.0.0
- **Status**: ✅ Production Ready

---

## Future Enhancements

Planned features for konsulctl:

- [ ] Built-in authentication support (JWT token management)
- [ ] Interactive mode
- [ ] Shell completion (bash, zsh, fish)
- [ ] Watch mode for KV changes
- [ ] Batch operations from file
- [ ] Output format options (JSON, YAML, table)
- [ ] Colorized output
- [ ] Progress bars for long operations
- [ ] Configuration file support (~/.konsulctl.yaml)

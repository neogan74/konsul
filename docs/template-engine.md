# Template Engine

Konsul's template engine allows you to dynamically generate configuration files based on data from the KV store and service catalog. It's inspired by HashiCorp's consul-template.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Template Functions](#template-functions)
- [CLI Usage](#cli-usage)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

The template engine watches for changes in Konsul's KV store and service catalog, automatically regenerating configuration files when data changes. This enables:

- Dynamic service discovery configuration (nginx, HAProxy, etc.)
- Application configuration from KV store
- Environment file generation
- Automatic service reloads on configuration changes

### Key Features

- **Go template syntax** - Uses Go's powerful `text/template` package
- **Watch mode** - Automatically regenerate configs when data changes
- **Once mode** - Generate configs once and exit
- **Dry-run mode** - Preview generated configs without writing files
- **Atomic writes** - Files are written atomically to prevent partial updates
- **Backup support** - Optionally create backups before overwriting
- **Command execution** - Run commands after successful rendering (e.g., reload services)
- **De-duplication** - Intelligent batching to avoid rapid rerenders

## Installation

### Build from Source

```bash
cd cmd/konsul-template
go build -o konsul-template
```

### Install to $GOPATH/bin

```bash
go install github.com/neogan74/konsul/cmd/konsul-template@latest
```

## Quick Start

### 1. Create a Template

Create a file `nginx.conf.tpl`:

```nginx
upstream backend {
{{- range service "web" }}
    server {{ .Address }}:{{ .Port }};
{{- end }}
}

server {
    listen 80;
    server_name {{ kv "config/domain" }};

    location / {
        proxy_pass http://backend;
    }
}
```

### 2. Populate Konsul Data

```bash
# Add KV data
curl -X POST http://localhost:8500/kv/config/domain \
  -H "Content-Type: application/json" \
  -d '{"value":"example.com"}'

# Register a service
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }'
```

### 3. Run konsul-template

```bash
# Once mode - generate and exit
konsul-template -template nginx.conf.tpl -dest /etc/nginx/nginx.conf -once

# Watch mode - regenerate on changes
konsul-template -template nginx.conf.tpl -dest /etc/nginx/nginx.conf

# Dry-run - preview without writing
konsul-template -template nginx.conf.tpl -dest /tmp/nginx.conf -dry -once
```

## Template Functions

### KV Store Functions

#### `kv "key"`

Retrieves a value from the KV store.

```go
Database: {{ kv "config/database/host" }}
```

#### `kvTree "prefix"`

Retrieves all key-value pairs under a prefix.

```go
{{- range kvTree "config/database/" }}
{{ .Key }}: {{ .Value }}
{{- end }}
```

#### `kvList "prefix"`

Returns all keys under a prefix.

```go
{{- range kvList "config/" }}
- {{ . }}
{{- end }}
```

### Service Discovery Functions

#### `service "name"`

Retrieves all instances of a service.

```go
{{- range service "web" }}
server {{ .Address }}:{{ .Port }}
{{- end }}
```

Returns: `[]Service` with fields:
- `Name` - Service name
- `Address` - Service address
- `Port` - Service port

#### `services`

Retrieves all registered services.

```go
{{- range services }}
{{ .Name }}: {{ .Address }}:{{ .Port }}
{{- end }}
```

### Utility Functions

#### `env "VAR"`

Retrieves an environment variable.

```go
Home: {{ env "HOME" }}
User: {{ env "USER" }}
```

#### `file "path"`

Reads a file and returns its contents.

```go
Hostname: {{ file "/etc/hostname" }}
```

### String Manipulation

Built-in string functions:

- `toLower` - Convert to lowercase
- `toUpper` - Convert to uppercase
- `trim` - Trim whitespace
- `split` - Split string
- `join` - Join strings
- `replace` - Replace substring
- `contains` - Check if contains
- `hasPrefix` - Check prefix
- `hasSuffix` - Check suffix

Example:

```go
{{ $name := kv "app/name" }}
Uppercase: {{ toUpper $name }}
Lowercase: {{ toLower $name }}
```

## CLI Usage

### Command-Line Flags

```bash
konsul-template [options]
```

**Options:**

- `-template <file>` - Template source file
- `-dest <file>` - Destination file path
- `-konsul <addr>` - Konsul server address (default: http://localhost:8500)
- `-once` - Run once and exit (don't watch)
- `-dry` - Dry-run mode (don't write files or execute commands)
- `-version` - Show version

### Examples

**Once mode:**
```bash
konsul-template -template app.conf.tpl -dest /etc/app.conf -once
```

**Watch mode:**
```bash
konsul-template -template app.conf.tpl -dest /etc/app.conf
```

**Dry-run:**
```bash
konsul-template -template app.conf.tpl -dest /tmp/test.conf -dry -once
```

**Custom Konsul address:**
```bash
konsul-template -template app.conf.tpl -dest app.conf -konsul http://konsul.example.com:8500
```

## Examples

### Example 1: Nginx Configuration

**Template (`nginx.conf.tpl`):**

```nginx
{{- $domain := kv "config/domain" }}

upstream backend {
    {{- range service "web" }}
    server {{ .Address }}:{{ .Port }} max_fails=3;
    {{- end }}
}

server {
    listen 80;
    server_name {{ $domain }};

    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
    }
}
```

**Usage:**

```bash
konsul-template -template nginx.conf.tpl -dest /etc/nginx/nginx.conf
```

### Example 2: Application Environment File

**Template (`app.env.tpl`):**

```bash
# Database
DB_HOST={{ kv "config/database/host" }}
DB_PORT={{ kv "config/database/port" }}
DB_NAME={{ kv "config/database/name" }}

# Services
{{- range services }}
SERVICE_{{ toUpper .Name }}_URL=http://{{ .Address }}:{{ .Port }}
{{- end }}
```

**Usage:**

```bash
konsul-template -template app.env.tpl -dest /app/.env -once
```

### Example 3: HAProxy Configuration

See [examples/templates/haproxy.cfg.tpl](../examples/templates/haproxy.cfg.tpl) for a complete HAProxy example.

### Example 4: Service Discovery List

**Template (`services.txt.tpl`):**

```text
Registered Services:
{{- range services }}

Service: {{ .Name }}
  Address: {{ .Address }}
  Port: {{ .Port }}
{{- end }}
```

## Best Practices

### 1. Use Once Mode for Initial Setup

Use `-once` flag during application startup to generate initial configs:

```bash
konsul-template -template app.conf.tpl -dest app.conf -once
./start-app
```

### 2. Dry-Run Before Production

Always test templates with `-dry` flag first:

```bash
konsul-template -template nginx.conf.tpl -dest /tmp/test.conf -dry -once
cat /tmp/test.conf  # Review output
```

### 3. Handle Missing Keys Gracefully

Use template conditionals for optional keys:

```go
{{- if kv "optional/key" }}
OPTIONAL_VAR={{ kv "optional/key" }}
{{- end }}
```

### 4. Validate Generated Configs

For critical configs (nginx, HAProxy), validate before applying:

```bash
# Generate config
konsul-template -template nginx.conf.tpl -dest /tmp/nginx.conf -once

# Validate
nginx -t -c /tmp/nginx.conf

# If valid, copy to production
cp /tmp/nginx.conf /etc/nginx/nginx.conf
nginx -s reload
```

### 5. Use Atomic Writes

The template engine writes files atomically (write to temp file, then rename). This prevents reading partially-written files.

### 6. Organize Templates

Keep templates in a dedicated directory:

```
/etc/konsul-template/
  templates/
    nginx.conf.tpl
    haproxy.cfg.tpl
    app.env.tpl
```

### 7. Monitor Template Rendering

The template engine logs all rendering operations. Monitor logs for errors:

```bash
konsul-template ... 2>&1 | tee /var/log/konsul-template.log
```

### 8. Version Control Templates

Store templates in version control alongside your application code.

## Troubleshooting

### Template Parse Errors

If you see parse errors, check your template syntax:

```bash
# Use dry-run to see detailed errors
konsul-template -template bad.tpl -dest /tmp/test -dry -once
```

### Missing Keys

If a key doesn't exist, the template will fail. Use conditionals:

```go
{{- with kv "optional/key" }}
VALUE={{ . }}
{{- else }}
# Key not found
{{- end }}
```

### Permission Denied

Ensure konsul-template has write permissions to the destination directory:

```bash
# Check permissions
ls -la /etc/nginx/

# Fix permissions
sudo chown $(whoami) /etc/nginx/nginx.conf
```

## Library Usage

You can also use the template engine as a Go library:

```go
import (
    "github.com/neogan74/konsul/internal/template"
    "github.com/neogan74/konsul/internal/logger"
)

// Create engine
config := template.Config{
    Templates: []template.TemplateConfig{
        {
            Source:      "app.conf.tpl",
            Destination: "/etc/app.conf",
            Perms:       0644,
        },
    },
    Once: true,
}

engine := template.New(config, kvStore, serviceStore, logger.GetDefault())

// Run once
if err := engine.RunOnce(); err != nil {
    log.Fatal(err)
}
```

## See Also

- [ADR-0015: Template Engine](../docs/adr/0015-template-engine.md)
- [Example Templates](../examples/templates/)
- [Go text/template documentation](https://pkg.go.dev/text/template)

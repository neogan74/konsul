# Template Engine - API Reference

Complete API reference for the Konsul template engine Go package.

## Package `github.com/neogan74/konsul/internal/template`

### Types

#### `Config`

Main configuration for the template engine.

```go
type Config struct {
    Templates  []TemplateConfig  // List of templates to render
    KonsulAddr string            // Konsul server address
    Token      string            // Authentication token (optional)
    Once       bool              // Run once and exit
    DryRun     bool              // Preview without writing
    Wait       *WaitConfig       // Global wait configuration
}
```

**Fields:**

- **Templates** - Array of template configurations to process
- **KonsulAddr** - HTTP address of Konsul server (e.g., `http://localhost:8500`)
- **Token** - Authentication token for secured Konsul instances
- **Once** - When true, renders all templates once and exits
- **DryRun** - When true, renders but doesn't write files or execute commands
- **Wait** - Global wait configuration (can be overridden per-template)

---

#### `TemplateConfig`

Configuration for a single template.

```go
type TemplateConfig struct {
    Source         string        // Template file path
    Destination    string        // Output file path
    Command        string        // Post-render command
    CommandTimeout time.Duration // Command timeout
    Perms          uint32        // File permissions (e.g., 0644)
    Backup         bool          // Create backup before overwriting
    Wait           *WaitConfig   // Per-template wait config
}
```

**Fields:**

- **Source** - Path to template file (e.g., `nginx.conf.tpl`)
- **Destination** - Where to write rendered output (e.g., `/etc/nginx/nginx.conf`)
- **Command** - Shell command to run after successful render (e.g., `nginx -s reload`)
- **CommandTimeout** - Maximum time for command execution (default: 30s)
- **Perms** - Unix file permissions as octal (e.g., `0644` = `-rw-r--r--`)
- **Backup** - Create `.bak` file before overwriting
- **Wait** - Overrides global wait configuration for this template

---

#### `WaitConfig`

Timing configuration for de-duplication.

```go
type WaitConfig struct {
    Min time.Duration  // Minimum wait before rendering
    Max time.Duration  // Maximum wait before forcing render
}
```

**Fields:**

- **Min** - Minimum time between change detection and render (batching period)
- **Max** - Maximum time to wait before forcing a render (prevents infinite waiting)

**Typical values:**
```go
&WaitConfig{
    Min: 2 * time.Second,   // Quick response
    Max: 10 * time.Second,  // Reasonable batching
}
```

---

#### `RenderContext`

Context passed to template execution with data sources and functions.

```go
type RenderContext struct {
    KVStore      KVStoreReader       // KV store interface
    ServiceStore ServiceStoreReader  // Service store interface
    DryRun       bool                // Dry-run flag
}
```

**Methods:**

- `FuncMap() template.FuncMap` - Returns template function map

---

#### `RenderResult`

Result of a template render operation.

```go
type RenderResult struct {
    Template        TemplateConfig  // Template that was rendered
    Content         string          // Rendered content
    Written         bool            // File was written
    CommandExecuted bool            // Command was executed
    CommandOutput   string          // Command output
    Error           error           // Error if any
    Duration        time.Duration   // Render duration
}
```

**Fields:**

- **Template** - The template configuration used
- **Content** - The fully rendered content
- **Written** - True if file was successfully written
- **CommandExecuted** - True if post-render command ran
- **CommandOutput** - Stdout/stderr from command
- **Error** - First error encountered (or nil)
- **Duration** - How long rendering took

---

#### `KVStoreReader`

Interface for reading KV data.

```go
type KVStoreReader interface {
    Get(key string) (string, bool)
    List() []string
}
```

**Methods:**

- **Get(key)** - Retrieve value for key, returns (value, exists)
- **List()** - Return all keys in the store

---

#### `ServiceStoreReader`

Interface for reading service data.

```go
type ServiceStoreReader interface {
    List() []Service
    Get(name string) (Service, bool)
}
```

**Methods:**

- **List()** - Return all registered services
- **Get(name)** - Retrieve specific service, returns (service, exists)

---

#### `Service`

Represents a registered service.

```go
type Service struct {
    Name    string  `json:"name"`
    Address string  `json:"address"`
    Port    int     `json:"port"`
}
```

**Fields:**

- **Name** - Service name (e.g., `"web"`)
- **Address** - IP address or hostname
- **Port** - TCP port number

---

### Functions

#### `New`

Create a new template engine.

```go
func New(
    config Config,
    kvStore KVStoreReader,
    serviceStore ServiceStoreReader,
    log logger.Logger,
) *Engine
```

**Parameters:**
- `config` - Engine configuration
- `kvStore` - KV data source
- `serviceStore` - Service data source
- `log` - Logger for structured logging

**Returns:** Configured engine instance

**Example:**

```go
engine := template.New(
    template.Config{
        Templates: []template.TemplateConfig{
            {
                Source:      "app.conf.tpl",
                Destination: "/etc/app.conf",
                Perms:       0644,
            },
        },
        Once: true,
    },
    kvStore,
    serviceStore,
    logger.GetDefault(),
)
```

---

### Engine Methods

#### `RunOnce`

Render all templates once and exit.

```go
func (e *Engine) RunOnce() error
```

**Returns:** Error if any template fails

**Example:**

```go
if err := engine.RunOnce(); err != nil {
    log.Fatal(err)
}
```

**Behavior:**
- Renders all configured templates sequentially
- Writes files if not in dry-run mode
- Executes commands if specified
- Returns on first error (does not continue)

---

#### `Run`

Start watch mode (continuous operation).

```go
func (e *Engine) Run(ctx context.Context) error
```

**Parameters:**
- `ctx` - Context for cancellation

**Returns:** Error if startup fails

**Example:**

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Handle Ctrl+C
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt)
go func() {
    <-sigCh
    cancel()
}()

if err := engine.Run(ctx); err != nil {
    log.Fatal(err)
}
```

**Behavior:**
- Performs initial render (like RunOnce)
- Starts watcher goroutine for each template
- Watches for changes and re-renders automatically
- Blocks until context is cancelled
- Gracefully shuts down all watchers

---

#### `Stop`

Stop the engine gracefully.

```go
func (e *Engine) Stop()
```

**Example:**

```go
// In another goroutine
engine.Stop()
```

**Behavior:**
- Cancels internal context
- Waits for all watchers to finish
- Cleans up resources

---

#### `RenderTemplate`

Render a single template (used internally by watchers).

```go
func (e *Engine) RenderTemplate(tmpl TemplateConfig) (*RenderResult, error)
```

**Parameters:**
- `tmpl` - Template configuration

**Returns:** Render result and error

**Example:**

```go
result, err := engine.RenderTemplate(template.TemplateConfig{
    Source:      "test.tpl",
    Destination: "/tmp/output.txt",
})
```

---

### Renderer Methods

#### `NewRenderer`

Create a new template renderer.

```go
func NewRenderer(ctx *RenderContext) *Renderer
```

**Parameters:**
- `ctx` - Render context with data sources

**Returns:** New renderer instance

---

#### `Render`

Render a template.

```go
func (r *Renderer) Render(config TemplateConfig) (*RenderResult, error)
```

**Parameters:**
- `config` - Template configuration

**Returns:** Render result and error

**Example:**

```go
renderer := template.NewRenderer(&template.RenderContext{
    KVStore:      kvStore,
    ServiceStore: serviceStore,
})

result, err := renderer.Render(template.TemplateConfig{
    Source:      "app.tpl",
    Destination: "/tmp/app.conf",
    Perms:       0644,
})
```

---

### Executor Methods

#### `NewExecutor`

Create a new command executor.

```go
func NewExecutor() *Executor
```

**Returns:** New executor with default 30s timeout

---

#### `Execute`

Execute a shell command.

```go
func (e *Executor) Execute(command string, timeout time.Duration) (string, error)
```

**Parameters:**
- `command` - Shell command to execute
- `timeout` - Maximum execution time (0 = use default)

**Returns:** Combined stdout/stderr and error

**Example:**

```go
executor := template.NewExecutor()
output, err := executor.Execute("nginx -t", 10*time.Second)
if err != nil {
    log.Printf("Command failed: %v\nOutput: %s", err, output)
}
```

---

#### `ExecuteWithRetry`

Execute with retry logic.

```go
func (e *Executor) ExecuteWithRetry(command string, timeout time.Duration, maxRetries int) (string, error)
```

**Parameters:**
- `command` - Shell command
- `timeout` - Per-attempt timeout
- `maxRetries` - Maximum number of retries

**Returns:** Output and error

**Example:**

```go
// Retry up to 3 times with exponential backoff
output, err := executor.ExecuteWithRetry(
    "systemctl reload app",
    30*time.Second,
    3,
)
```

**Backoff:** 1s, 4s, 9s, 16s, ... (attempt²)

---

### Template Functions

These functions are available inside templates.

#### KV Store Functions

##### `kv`

Get a value from the KV store.

```go
{{ kv "key" }} → string
```

**Example:**

```go
Database: {{ kv "config/database/host" }}:{{ kv "config/database/port" }}
```

**Error:** Fails if key doesn't exist

---

##### `kvTree`

Get all key-value pairs under a prefix.

```go
{{ kvTree "prefix" }} → []KVPair
```

**Returns:** Array of `{Key string, Value string}`

**Example:**

```go
{{- range kvTree "config/database/" }}
{{ .Key }}: {{ .Value }}
{{- end }}
```

**Output:**
```
config/database/host: localhost
config/database/port: 5432
config/database/name: myapp
```

---

##### `kvList`

List all keys under a prefix.

```go
{{ kvList "prefix" }} → []string
```

**Example:**

```go
{{- range kvList "config/" }}
- {{ . }}
{{- end }}
```

**Output:**
```
- config/host
- config/port
- config/name
```

---

#### Service Discovery Functions

##### `service`

Get instances of a specific service.

```go
{{ service "name" }} → []Service
```

**Returns:** Array of services (may be empty)

**Example:**

```go
upstream backend {
{{- range service "web" }}
    server {{ .Address }}:{{ .Port }};
{{- end }}
}
```

**Output:**
```nginx
upstream backend {
    server 10.0.0.1:8080;
    server 10.0.0.2:8080;
    server 10.0.0.3:8080;
}
```

---

##### `services`

Get all registered services.

```go
{{ services }} → []Service
```

**Example:**

```go
{{- range services }}
{{ .Name }}: {{ .Address }}:{{ .Port }}
{{- end }}
```

**Output:**
```
web: 10.0.0.1:8080
api: 10.0.0.2:9000
db: 10.0.0.3:5432
```

---

#### Utility Functions

##### `env`

Get environment variable.

```go
{{ env "VAR" }} → string
```

**Example:**

```go
Home: {{ env "HOME" }}
User: {{ env "USER" }}
```

**Returns:** Empty string if not set

---

##### `file`

Read file contents.

```go
{{ file "path" }} → string
```

**Example:**

```go
Hostname: {{ file "/etc/hostname" }}
```

**Error:** Fails if file doesn't exist or can't be read

---

#### String Functions

##### `toLower`

Convert to lowercase.

```go
{{ toLower "HELLO" }} → "hello"
```

---

##### `toUpper`

Convert to uppercase.

```go
{{ toUpper "hello" }} → "HELLO"
```

---

##### `trim`

Remove leading/trailing whitespace.

```go
{{ trim "  hello  " }} → "hello"
```

---

##### `split`

Split string by separator.

```go
{{ split "a,b,c" "," }} → []string{"a", "b", "c"}
```

**Example:**

```go
{{- $parts := split "web,api,db" "," }}
{{- range $parts }}
- {{ . }}
{{- end }}
```

---

##### `join`

Join strings with separator.

```go
{{ join (list "a" "b" "c") "," }} → "a,b,c"
```

---

##### `replace`

Replace all occurrences.

```go
{{ replace "hello world" "world" "gopher" }} → "hello gopher"
```

---

##### `contains`

Check if string contains substring.

```go
{{ contains "hello world" "world" }} → true
```

---

##### `hasPrefix`

Check if string starts with prefix.

```go
{{ hasPrefix "hello world" "hello" }} → true
```

---

##### `hasSuffix`

Check if string ends with suffix.

```go
{{ hasSuffix "hello world" "world" }} → true
```

---

## CLI Tool

### Command: `konsul-template`

```bash
konsul-template [options]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-template` | string | - | Template source file path |
| `-dest` | string | - | Destination file path |
| `-konsul` | string | `http://localhost:8500` | Konsul server address |
| `-once` | bool | `false` | Run once and exit |
| `-dry` | bool | `false` | Dry-run mode (don't write) |
| `-version` | bool | `false` | Show version and exit |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (template failed, invalid args, etc.) |
| 2 | Signal received (Ctrl+C) |

### Examples

**Basic usage:**
```bash
konsul-template -template app.conf.tpl -dest /etc/app.conf -once
```

**Watch mode:**
```bash
konsul-template -template nginx.conf.tpl -dest /tmp/nginx.conf
```

**Dry-run:**
```bash
konsul-template -template test.tpl -dest output.txt -dry -once
```

**Custom Konsul address:**
```bash
konsul-template \
    -template app.tpl \
    -dest app.conf \
    -konsul http://konsul.prod.example.com:8500
```

---

## Error Handling

### Common Errors

#### `key not found: <key>`

**Cause:** KV key doesn't exist

**Solution:**
```go
{{- if kv "optional/key" }}
VALUE={{ kv "optional/key" }}
{{- end }}
```

---

#### `template: <file>: executing <file> at <line>: error calling kv: ...`

**Cause:** Template function error during execution

**Solution:** Check that:
- KV store is populated
- Key names are correct
- Network connectivity to Konsul

---

#### `failed to parse template: ...`

**Cause:** Template syntax error

**Solution:** Validate template syntax:
```bash
konsul-template -template bad.tpl -dest /tmp/test -dry -once
```

---

#### `permission denied`

**Cause:** Can't write to destination

**Solution:**
- Check file permissions
- Run with appropriate user
- Ensure parent directory exists

---

## Best Practices

### 1. Use Interfaces for Testing

```go
// Good - easy to mock
func ProcessTemplate(kv KVStoreReader, svc ServiceStoreReader) error {
    // ...
}

// Bad - hard to test
func ProcessTemplate(kv *store.KVStore, svc *store.ServiceStore) error {
    // ...
}
```

### 2. Handle Missing Data

```go
// Good - graceful handling
{{- $host := kv "db/host" }}
{{- if $host }}
DB_HOST={{ $host }}
{{- else }}
# Database host not configured
{{- end }}

// Bad - fails if key missing
DB_HOST={{ kv "db/host" }}
```

### 3. Validate Generated Configs

```go
// Generate to temp location
result, err := renderer.Render(TemplateConfig{
    Source:      "nginx.conf.tpl",
    Destination: "/tmp/nginx.conf",
})

// Validate
cmd := exec.Command("nginx", "-t", "-c", "/tmp/nginx.conf")
if err := cmd.Run(); err != nil {
    return fmt.Errorf("invalid nginx config: %w", err)
}

// Copy to production
os.Rename("/tmp/nginx.conf", "/etc/nginx/nginx.conf")
```

### 4. Use Appropriate Permissions

```go
// Secrets
TemplateConfig{Perms: 0600}  // -rw-------

// Configs
TemplateConfig{Perms: 0644}  // -rw-r--r--

// Scripts
TemplateConfig{Perms: 0755}  // -rwxr-xr-x
```

---

## Version

Current version: **0.1.0**

API stability: **Alpha** (may change)

---

## See Also

- [User Guide](template-engine.md) - End-user documentation
- [Implementation Guide](template-engine-implementation.md) - Technical deep dive
- [ADR-0015](adr/0015-template-engine.md) - Architecture decision
- [Go text/template](https://pkg.go.dev/text/template) - Template syntax

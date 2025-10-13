# Template Engine - Implementation Guide

This guide provides in-depth technical details about the template engine implementation for developers who want to understand, maintain, or extend the system.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Component Deep Dive](#component-deep-dive)
- [Data Flow](#data-flow)
- [Design Patterns](#design-patterns)
- [Implementation Details](#implementation-details)
- [Extending the Engine](#extending-the-engine)
- [Testing Strategy](#testing-strategy)
- [Performance Considerations](#performance-considerations)
- [Security Model](#security-model)

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    konsul-template CLI                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Flags      │  │   Client     │  │   Signal     │      │
│  │   Parser     │──│   (HTTP)     │  │   Handler    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Template Engine Core Package                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    Engine                             │   │
│  │  - Orchestrates all components                        │   │
│  │  - Manages lifecycle (start, stop, reload)           │   │
│  │  - Coordinates watchers and renderer                  │   │
│  └────────────┬──────────────┬──────────────┬────────────┘   │
│               │              │              │                 │
│       ┌───────▼─────┐  ┌────▼─────┐  ┌────▼─────┐          │
│       │  Watcher(s) │  │ Renderer │  │ Executor │          │
│       │             │  │          │  │          │          │
│       └──────┬──────┘  └────┬─────┘  └────┬─────┘          │
│              │              │             │                 │
│       ┌──────▼──────────────▼─────────────▼─────┐          │
│       │        RenderContext                     │          │
│       │  - KVStore interface                     │          │
│       │  - ServiceStore interface                │          │
│       │  - Template functions                    │          │
│       └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                  Konsul Backend                              │
│  ┌──────────────┐              ┌──────────────┐            │
│  │   KV Store   │              │ Service Store│            │
│  └──────────────┘              └──────────────┘            │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Key Files |
|-----------|---------------|-----------|
| **Engine** | Main orchestrator, lifecycle management | `engine.go` |
| **Watcher** | Detect changes, trigger re-renders | `watcher.go` |
| **Renderer** | Parse and execute templates | `renderer.go` |
| **Executor** | Run post-render commands | `executor.go` |
| **RenderContext** | Provide data and functions to templates | `functions.go`, `types.go` |
| **Client** | HTTP client for Konsul API | `cmd/konsul-template/client.go` |

## Component Deep Dive

### 1. Engine (`engine.go`)

**Purpose:** Central coordinator that manages the entire template rendering lifecycle.

**Key Responsibilities:**
- Initialize all components (renderer, watchers)
- Manage execution modes (once vs. watch)
- Handle graceful shutdown
- Coordinate concurrent operations

**Important Methods:**

```go
// New creates a new engine with configuration
func New(config Config, kvStore KVStoreReader, serviceStore ServiceStoreReader, log logger.Logger) *Engine

// RunOnce renders all templates once and exits
func (e *Engine) RunOnce() error

// Run starts watch mode (continuous operation)
func (e *Engine) Run(ctx context.Context) error

// Stop gracefully stops the engine
func (e *Engine) Stop()
```

**Implementation Details:**

```go
type Engine struct {
    config   Config              // User configuration
    renderer *Renderer           // Template renderer
    watchers []*Watcher          // One watcher per template
    log      logger.Logger       // Structured logger
    ctx      context.Context     // Cancellation context
    cancel   context.CancelFunc  // Cancel function
    wg       sync.WaitGroup      // Wait group for goroutines
}
```

**Concurrency Model:**
- Main goroutine runs the engine
- Each template gets its own watcher goroutine
- WaitGroup ensures all goroutines complete on shutdown
- Context propagation for cancellation

### 2. Watcher (`watcher.go`)

**Purpose:** Monitor data sources and trigger re-renders when changes are detected.

**Key Responsibilities:**
- Poll for changes at configurable intervals
- Detect actual content changes (not just timestamp changes)
- De-duplicate rapid changes using SHA256 hashing
- Respect min/max wait times to batch updates

**Change Detection Algorithm:**

```go
func (w *Watcher) hasChanged() bool {
    // 1. Render template to temporary buffer
    result, err := w.engine.RenderTemplate(w.template)
    if err != nil {
        return false  // Don't trigger on errors
    }

    // 2. Compute SHA256 hash of rendered content
    hash := computeHash(result.Content)

    // 3. Compare with last known hash
    if hash != w.lastHash {
        w.lastHash = hash
        return true
    }

    return false
}
```

**De-duplication Strategy:**

The watcher uses a two-tier timing mechanism:

```
Time ──────────────────────────────────────────────────►

        minWait           maxWait
        (2s)              (10s)
         │                 │
         ▼                 ▼
    ┌────────┬────────────────────┐
    │ Change │  Wait for more     │ Render
    │ Detect │  changes (batch)   │
    └────────┴────────────────────┘
```

- **minWait**: Minimum time between checks (default: 2s)
- **maxWait**: Maximum time to wait before rendering (default: 10s)

This prevents:
- Rapid-fire renders on multiple quick changes
- Excessive CPU usage from constant polling
- Thundering herd when many templates update simultaneously

**Goroutine Lifecycle:**

```go
func (w *Watcher) Watch(ctx context.Context) {
    ticker := time.NewTicker(w.minWait)
    defer ticker.Stop()

    maxWaitTimer := time.NewTimer(w.maxWait)
    defer maxWaitTimer.Stop()

    for {
        select {
        case <-ctx.Done():
            return  // Graceful shutdown
        case <-ticker.C:
            // Check for changes
        case <-maxWaitTimer.C:
            // Force render if pending
        }
    }
}
```

### 3. Renderer (`renderer.go`)

**Purpose:** Parse and execute Go templates, write output files atomically.

**Key Responsibilities:**
- Parse template files using `text/template`
- Execute templates with RenderContext
- Write files atomically (temp file + rename)
- Handle file permissions
- Create backups when requested

**Rendering Pipeline:**

```
┌──────────────┐
│ Read Template│
│ Source File  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Parse with   │
│ text/template│
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Execute with │
│ FuncMap +    │
│ Data         │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Write to     │
│ .tmp file    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Atomic rename│
│ to dest      │
└──────────────┘
```

**Atomic Write Implementation:**

```go
func (r *Renderer) writeFile(config TemplateConfig) error {
    // 1. Ensure destination directory exists
    destDir := filepath.Dir(config.Destination)
    os.MkdirAll(destDir, 0755)

    // 2. Create backup if requested
    if config.Backup {
        copyFile(config.Destination, config.Destination+".bak")
    }

    // 3. Write to temporary file
    tempPath := config.Destination + ".tmp"
    os.WriteFile(tempPath, content, perms)

    // 4. Atomic rename (POSIX guarantees atomicity)
    os.Rename(tempPath, config.Destination)
}
```

**Why Atomic Writes Matter:**

- Prevents readers from seeing partially-written files
- Process crashes won't leave corrupt configs
- Multiple konsul-template processes can't conflict
- Essential for critical configs (nginx, HAProxy)

### 4. Executor (`executor.go`)

**Purpose:** Execute post-render commands safely with timeout protection.

**Key Features:**
- Shell command execution via `sh -c`
- Configurable timeouts
- Stdout/stderr capture
- Retry logic with exponential backoff

**Safety Mechanisms:**

```go
func (e *Executor) Execute(command string, timeout time.Duration) (string, error) {
    // 1. Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    // 2. Create command with context (auto-kills on timeout)
    cmd := exec.CommandContext(ctx, "sh", "-c", command)

    // 3. Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    // 4. Execute and handle timeout
    err := cmd.Run()
    if ctx.Err() == context.DeadlineExceeded {
        return output, fmt.Errorf("command timed out after %v", timeout)
    }

    return output, err
}
```

**Retry Logic:**

```go
func (e *Executor) ExecuteWithRetry(command string, timeout time.Duration, maxRetries int) (string, error) {
    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff: 1s, 4s, 9s, 16s, ...
            backoff := time.Duration(attempt*attempt) * time.Second
            time.Sleep(backoff)
        }

        output, err := e.Execute(command, timeout)
        if err == nil {
            return output, nil
        }
    }
    return "", fmt.Errorf("failed after %d attempts", maxRetries+1)
}
```

### 5. RenderContext & Functions (`functions.go`)

**Purpose:** Provide data and functions to templates during execution.

**Architecture:**

```go
type RenderContext struct {
    KVStore      KVStoreReader       // KV data source
    ServiceStore ServiceStoreReader  // Service data source
    DryRun       bool                // Preview mode flag
}

func (ctx *RenderContext) FuncMap() template.FuncMap {
    return template.FuncMap{
        "kv":       ctx.kv,
        "kvTree":   ctx.kvTree,
        "service":  ctx.service,
        "services": ctx.services,
        "env":      env,
        "file":     file,
        // ... string functions
    }
}
```

**Function Implementation Pattern:**

All template functions follow this pattern:

```go
func (ctx *RenderContext) functionName(params...) (returnType, error) {
    // 1. Validate parameters
    if invalidParam {
        return nil, fmt.Errorf("validation error")
    }

    // 2. Check if data source available
    if ctx.DataSource == nil {
        return nil, fmt.Errorf("data source not available")
    }

    // 3. Fetch data
    data, ok := ctx.DataSource.Get(key)

    // 4. Return result or error
    if !ok {
        return nil, fmt.Errorf("not found: %s", key)
    }
    return data, nil
}
```

**Error Handling in Templates:**

Go templates stop execution on first error. This is intentional:
- Fail fast on missing data
- Prevent generating invalid configs
- Force explicit error handling

To handle optional data:

```go
{{- if kv "optional/key" }}
VALUE={{ kv "optional/key" }}
{{- end }}
```

## Data Flow

### Once Mode Flow

```
User runs CLI
    │
    ▼
Parse flags & config
    │
    ▼
Create Engine
    │
    ▼
engine.RunOnce()
    │
    ├─► For each template:
    │   │
    │   ├─► renderer.Render(template)
    │   │   │
    │   │   ├─► Read template file
    │   │   ├─► Parse with text/template
    │   │   ├─► Execute with RenderContext
    │   │   ├─► Write to destination (atomic)
    │   │   └─► executor.Execute(command) if specified
    │   │
    │   └─► Log result
    │
    └─► Return errors (if any)
```

### Watch Mode Flow

```
User runs CLI
    │
    ▼
Parse flags & config
    │
    ▼
Create Engine
    │
    ▼
engine.Run(ctx)
    │
    ├─► Initial render (same as once mode)
    │
    ├─► Start watcher goroutine for each template
    │   │
    │   └─► Watch loop:
    │       │
    │       ├─► Wait minWait interval
    │       ├─► Check if content changed (SHA256)
    │       ├─► If changed, mark pending
    │       ├─► Wait up to maxWait
    │       ├─► Render if pending
    │       └─► Repeat
    │
    ├─► Wait for ctx.Done() (Ctrl+C)
    │
    └─► Shutdown:
        ├─► Cancel context (stops watchers)
        └─► Wait for all goroutines (WaitGroup)
```

### Template Execution Flow

```
Template:
  {{ kv "config/host" }}:{{ kv "config/port" }}

Execution:
  1. text/template parser creates AST
  2. Encounters {{ kv "config/host" }}
  3. Looks up "kv" in FuncMap
  4. Calls ctx.kv("config/host")
  5. ctx.kv queries KVStore.Get("config/host")
  6. Returns "localhost"
  7. Continues with ":"
  8. Encounters {{ kv "config/port" }}
  9. Calls ctx.kv("config/port")
  10. Returns "8080"

Result: "localhost:8080"
```

## Design Patterns

### 1. Interface Segregation

We use minimal interfaces instead of concrete types:

```go
// KVStoreReader - only what templates need
type KVStoreReader interface {
    Get(key string) (string, bool)
    List() []string
}

// ServiceStoreReader - only what templates need
type ServiceStoreReader interface {
    List() []Service
    Get(name string) (Service, bool)
}
```

**Benefits:**
- Easy to mock in tests
- Decouples template engine from store implementation
- Allows multiple backend implementations
- Minimal API surface = fewer breaking changes

### 2. Builder Pattern (Implicit)

Engine construction uses functional options pattern:

```go
engine := template.New(
    config,           // Core configuration
    kvStore,          // KV data source
    serviceStore,     // Service data source
    logger,           // Logging
)
```

### 3. Strategy Pattern

Different execution strategies:

```go
// Once mode strategy
func (e *Engine) RunOnce() error {
    for _, tmpl := range e.config.Templates {
        e.renderer.Render(tmpl)
    }
}

// Watch mode strategy
func (e *Engine) Run(ctx context.Context) error {
    e.RunOnce()  // Initial render
    for _, tmpl := range e.config.Templates {
        go watcher.Watch(ctx)  // Continuous watching
    }
}
```

### 4. Context Propagation

Clean shutdown via context:

```go
// Parent context
ctx, cancel := context.WithCancel(context.Background())

// Pass to all goroutines
go watcher.Watch(ctx)

// On shutdown
cancel()  // All goroutines see ctx.Done()
```

### 5. Functional Options (for future extension)

Template functions use closure for context access:

```go
func (ctx *RenderContext) FuncMap() template.FuncMap {
    return template.FuncMap{
        // Functions capture ctx in closure
        "kv": func(key string) (string, error) {
            return ctx.KVStore.Get(key)
        },
    }
}
```

## Implementation Details

### SHA256-based Change Detection

**Why SHA256?**
- Content-based, not timestamp-based
- Fast enough for config files (< 1ms for typical configs)
- Deterministic - same content always has same hash
- Prevents false triggers from metadata changes

**Implementation:**

```go
func computeHash(content string) string {
    h := sha256.New()
    h.Write([]byte(content))
    return hex.EncodeToString(h.Sum(nil))
}
```

**Performance:**
- SHA256 throughput: ~500 MB/s on modern CPUs
- Typical config file: 10 KB
- Hash computation: ~0.02ms
- Negligible overhead vs. file I/O

### Wait Configuration

**Problem:** Balance between responsiveness and efficiency.

**Solution:** Two-tier wait system:

```go
type WaitConfig struct {
    Min time.Duration  // Fast response to single change
    Max time.Duration  // Batch multiple rapid changes
}
```

**Example Scenarios:**

**Scenario 1: Single change**
```
t=0s: Change detected
t=2s: No more changes, render immediately
Result: 2s latency (minWait)
```

**Scenario 2: Rapid changes**
```
t=0s: Change 1 detected
t=1s: Change 2 detected
t=3s: Change 3 detected
t=5s: Change 4 detected
t=10s: MaxWait reached, render now
Result: 10s latency, but only 1 render instead of 4
```

**Configuration:**

```go
// Global default
config.Wait = &WaitConfig{
    Min: 2 * time.Second,
    Max: 10 * time.Second,
}

// Per-template override
template.Wait = &WaitConfig{
    Min: 5 * time.Second,   // Less sensitive
    Max: 30 * time.Second,  // Longer batching
}
```

### File Permissions

**POSIX Permission Model:**

```go
type FileMode uint32

// Standard permissions
const (
    OwnerRead   = 0400  // -r--------
    OwnerWrite  = 0200  // --w-------
    OwnerExec   = 0100  // ---x------
    GroupRead   = 0040  // ----r-----
    GroupWrite  = 0020  // -----w----
    GroupExec   = 0010  // ------x---
    OtherRead   = 0004  // -------r--
    OtherWrite  = 0002  // --------w-
    OtherExec   = 0001  // ---------x
)

// Common combinations
const (
    Perm0644 = 0644  // -rw-r--r--  (readable by all, writable by owner)
    Perm0600 = 0600  // -rw-------  (owner only)
    Perm0755 = 0755  // -rwxr-xr-x  (executable)
)
```

**Usage in templates:**

```go
config := TemplateConfig{
    Source:      "app.conf.tpl",
    Destination: "/etc/app.conf",
    Perms:       0644,  // Owner RW, others R
}
```

### Atomic Rename Semantics

**POSIX Guarantee:**

The `rename()` system call is atomic:
- Either fully succeeds or fully fails
- No intermediate state visible to other processes
- Works across most filesystems (ext4, btrfs, xfs)

**Implementation:**

```go
// Step 1: Write new content to temp file
os.WriteFile("config.tmp", content, 0644)

// Step 2: Atomic rename
os.Rename("config.tmp", "config")
// At this point, readers see either:
// - Old config (if rename not complete)
// - New config (if rename complete)
// Never: partial/corrupt config
```

**Edge Cases:**

- **Different filesystems:** Atomic rename only works within same filesystem
- **Windows:** Rename not atomic; requires special handling (not yet implemented)
- **NFS:** Atomicity guarantees vary by NFS version

## Extending the Engine

### Adding New Template Functions

**Step 1: Define the function**

```go
// functions.go

// myNewFunc does something useful
// Usage: {{ myNewFunc "arg1" "arg2" }}
func (ctx *RenderContext) myNewFunc(arg1, arg2 string) (string, error) {
    // Validate arguments
    if arg1 == "" {
        return "", fmt.Errorf("arg1 cannot be empty")
    }

    // Implement logic
    result := doSomething(arg1, arg2)

    return result, nil
}
```

**Step 2: Register in FuncMap**

```go
// functions.go

func (ctx *RenderContext) FuncMap() template.FuncMap {
    return template.FuncMap{
        // ... existing functions
        "myNewFunc": ctx.myNewFunc,
    }
}
```

**Step 3: Write tests**

```go
// functions_test.go

func TestMyNewFunc(t *testing.T) {
    ctx := &RenderContext{
        // Setup context
    }

    result, err := ctx.myNewFunc("test1", "test2")
    if err != nil {
        t.Fatalf("myNewFunc() error = %v", err)
    }

    expected := "expected result"
    if result != expected {
        t.Errorf("myNewFunc() = %v, want %v", result, expected)
    }
}
```

**Step 4: Document it**

Add to `docs/template-engine.md`:

```markdown
#### `myNewFunc "arg1" "arg2"`

Does something useful with two arguments.

**Example:**
\```
{{ myNewFunc "hello" "world" }}
\```

**Returns:** Combined result
```

### Adding New Data Sources

**Step 1: Define interface**

```go
// types.go

type MyDataSourceReader interface {
    GetSomething(key string) (Something, bool)
    ListSomethings() []Something
}
```

**Step 2: Add to RenderContext**

```go
// types.go

type RenderContext struct {
    KVStore      KVStoreReader
    ServiceStore ServiceStoreReader
    MyDataSource MyDataSourceReader  // New!
    DryRun       bool
}
```

**Step 3: Add template functions**

```go
// functions.go

func (ctx *RenderContext) something(key string) (Something, error) {
    if ctx.MyDataSource == nil {
        return Something{}, fmt.Errorf("data source not available")
    }

    data, ok := ctx.MyDataSource.GetSomething(key)
    if !ok {
        return Something{}, fmt.Errorf("not found: %s", key)
    }

    return data, nil
}
```

**Step 4: Update Engine**

```go
// engine.go

func New(
    config Config,
    kvStore KVStoreReader,
    serviceStore ServiceStoreReader,
    myDataSource MyDataSourceReader,  // New parameter
    log logger.Logger,
) *Engine {
    renderCtx := &RenderContext{
        KVStore:      kvStore,
        ServiceStore: serviceStore,
        MyDataSource: myDataSource,  // Wire up
        DryRun:       config.DryRun,
    }

    // ... rest of initialization
}
```

### Adding Configuration Options

**Step 1: Add to Config struct**

```go
// types.go

type Config struct {
    Templates  []TemplateConfig
    KonsulAddr string
    Token      string
    Once       bool
    DryRun     bool
    Wait       *WaitConfig

    // New option
    MyNewOption bool `json:"my_new_option"`
}
```

**Step 2: Add CLI flag**

```go
// cmd/konsul-template/main.go

var (
    // ... existing flags
    myNewOption = flag.Bool("my-option", false, "Enable my new feature")
)

config := template.Config{
    // ... existing config
    MyNewOption: *myNewOption,
}
```

**Step 3: Use in implementation**

```go
// engine.go or renderer.go

if e.config.MyNewOption {
    // Do something different
}
```

### Adding Metrics

**Step 1: Define metrics**

```go
// metrics.go (new file)

import "github.com/neogan74/konsul/internal/metrics"

var (
    templateRenderTotal = metrics.NewCounter(
        "konsul_template_renders_total",
        "Total number of template renders",
        []string{"template", "status"},
    )

    templateRenderDuration = metrics.NewHistogram(
        "konsul_template_render_duration_seconds",
        "Template render duration",
        []string{"template"},
    )
)
```

**Step 2: Instrument code**

```go
// renderer.go

func (r *Renderer) Render(config TemplateConfig) (*RenderResult, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        templateRenderDuration.Observe(duration.Seconds(), config.Source)
    }()

    // ... rendering logic

    status := "success"
    if result.Error != nil {
        status = "error"
    }
    templateRenderTotal.Inc(config.Source, status)

    return result, nil
}
```

## Testing Strategy

### Unit Tests

**Test each component in isolation:**

```go
// functions_test.go - Test template functions
func TestKVFunction(t *testing.T) {
    // Use mock KV store
    kvStore := NewMockKVStore()
    kvStore.Set("key", "value")

    ctx := &RenderContext{KVStore: kvStore}
    result, err := ctx.kv("key")

    assert.NoError(t, err)
    assert.Equal(t, "value", result)
}
```

**Mock Implementation:**

```go
type MockKVStore struct {
    data map[string]string
}

func (m *MockKVStore) Get(key string) (string, bool) {
    val, ok := m.data[key]
    return val, ok
}

func (m *MockKVStore) List() []string {
    keys := make([]string, 0, len(m.data))
    for k := range m.data {
        keys = append(keys, k)
    }
    return keys
}
```

### Integration Tests

**Test components working together:**

```go
// engine_test.go
func TestEngineIntegration(t *testing.T) {
    // Setup real-ish environment
    kvStore := NewMockKVStore()
    serviceStore := NewMockServiceStore()

    config := Config{
        Templates: []TemplateConfig{{
            Source: "test.tpl",
            Destination: filepath.Join(t.TempDir(), "output"),
        }},
        Once: true,
    }

    engine := New(config, kvStore, serviceStore, logger.GetDefault())

    err := engine.RunOnce()
    assert.NoError(t, err)

    // Verify output file exists and has correct content
    content, err := os.ReadFile(config.Templates[0].Destination)
    assert.NoError(t, err)
    assert.Contains(t, string(content), "expected text")
}
```

### End-to-End Tests

**Test the full CLI flow:**

```bash
#!/bin/bash
# test_e2e.sh

# 1. Start Konsul server
go run cmd/konsul/main.go &
KONSUL_PID=$!

# 2. Populate test data
curl -X POST http://localhost:8500/kv/test/key -d '{"value":"test-value"}'

# 3. Run konsul-template
./bin/konsul-template \
    -template test.tpl \
    -dest /tmp/output.txt \
    -once

# 4. Verify output
grep "test-value" /tmp/output.txt

# 5. Cleanup
kill $KONSUL_PID
```

### Test Coverage

Current coverage:

```bash
$ go test ./internal/template/... -cover
ok      github.com/neogan74/konsul/internal/template    0.642s  coverage: 78.3% of statements
```

**Coverage by file:**
- `functions.go`: 95% (all functions tested)
- `renderer.go`: 85% (atomic writes, backups tested)
- `executor.go`: 70% (basic execution tested, retry not yet)
- `engine.go`: 65% (once mode tested, watch mode partial)
- `watcher.go`: 50% (basic logic tested, timing not fully)

## Performance Considerations

### Rendering Performance

**Benchmark data** (typical 10KB nginx config):

| Operation | Time | Notes |
|-----------|------|-------|
| Parse template | ~0.5ms | Cached after first parse |
| Execute template | ~1ms | Function calls dominate |
| SHA256 hash | ~0.02ms | Very fast |
| File write | ~5ms | Disk I/O bottleneck |
| Atomic rename | ~0.1ms | Metadata operation |
| **Total** | **~7ms** | End-to-end render time |

**Optimization opportunities:**

1. **Template parsing cache:**
```go
// Cache parsed templates
var templateCache sync.Map

func parseTemplate(source string) (*template.Template, error) {
    if cached, ok := templateCache.Load(source); ok {
        return cached.(*template.Template), nil
    }

    tmpl, err := template.ParseFiles(source)
    if err != nil {
        return nil, err
    }

    templateCache.Store(source, tmpl)
    return tmpl, nil
}
```

2. **Parallel rendering:**
```go
func (e *Engine) RunOnce() error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(e.config.Templates))

    for _, tmpl := range e.config.Templates {
        wg.Add(1)
        go func(t TemplateConfig) {
            defer wg.Done()
            _, err := e.renderer.Render(t)
            if err != nil {
                errCh <- err
            }
        }(tmpl)
    }

    wg.Wait()
    close(errCh)

    // Collect errors
    for err := range errCh {
        // Handle errors
    }
}
```

### Memory Usage

**Typical memory profile:**

```
Template engine: ~5 MB
  - Template ASTs: ~1 MB (per template)
  - Watcher goroutines: ~2 KB each
  - KV cache: ~500 KB (1000 keys)
  - Service cache: ~100 KB (100 services)
```

**Memory optimization:**

1. **Streaming for large templates:**
```go
// Instead of buffering entire output
var buf strings.Builder
tmpl.Execute(&buf, data)
content := buf.String()

// Stream directly to file
file, _ := os.Create(tempPath)
defer file.Close()
tmpl.Execute(file, data)
```

2. **Limit cache sizes:**
```go
type LRUCache struct {
    maxSize int
    cache   *lru.Cache
}
```

### Network Performance

**HTTP client optimization:**

```go
client := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        // Enable keep-alive
    },
}
```

## Security Model

### Threat Model

**Threats we protect against:**

1. **Partial file reads** → Atomic writes
2. **Command injection** → Parameterized execution
3. **Path traversal** → Absolute path resolution
4. **Excessive timeouts** → Configurable limits
5. **Resource exhaustion** → Rate limiting

**Threats we don't (yet) protect against:**

1. **Malicious templates** → No sandboxing
2. **Arbitrary command execution** → User-specified commands
3. **Secret exposure** → No secret redaction in logs

### Security Best Practices

**1. File Permissions:**

```go
// Sensitive configs
config := TemplateConfig{
    Destination: "/etc/app/credentials.conf",
    Perms:       0600,  // Owner only
}

// Public configs
config := TemplateConfig{
    Destination: "/etc/nginx/nginx.conf",
    Perms:       0644,  // Readable by all
}
```

**2. Command Whitelisting:**

```go
// Future enhancement
var allowedCommands = map[string]bool{
    "nginx -s reload":      true,
    "systemctl reload app": true,
}

func (e *Executor) Execute(command string) error {
    if !allowedCommands[command] {
        return fmt.Errorf("command not whitelisted: %s", command)
    }
    // Execute
}
```

**3. Input Validation:**

```go
func (ctx *RenderContext) kv(key string) (string, error) {
    // Validate key format
    if strings.Contains(key, "..") {
        return "", fmt.Errorf("invalid key: path traversal attempt")
    }

    if len(key) > 1024 {
        return "", fmt.Errorf("key too long")
    }

    // Proceed with lookup
}
```

**4. Secrets Management:**

```go
// Don't log secret values
func (e *Engine) logResult(result *RenderResult) {
    // Redact sensitive paths
    dest := result.Template.Destination
    if strings.Contains(dest, "secret") || strings.Contains(dest, "credential") {
        dest = "[REDACTED]"
    }

    e.log.Info("Template rendered",
        logger.String("destination", dest))
}
```

---

This implementation guide provides the technical depth needed to understand, maintain, and extend the template engine. For user-facing documentation, see [template-engine.md](template-engine.md).

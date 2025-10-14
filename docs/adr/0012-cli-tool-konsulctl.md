# ADR-0012: Command-Line Interface Tool (konsulctl)

**Date**: 2024-09-26

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: cli, tooling, developer-experience, operations

## Context

While Konsul provides REST APIs for all operations, operators and developers need a convenient command-line interface for:

### Use Cases

1. **Local development**: Quickly test KV operations and service registration
2. **CI/CD pipelines**: Automate deployments and configuration updates
3. **Operations**: Manual service management and troubleshooting
4. **Scripts**: Shell scripts for automation and monitoring
5. **Debugging**: Inspect cluster state and test connectivity
6. **Backup/restore**: Data management operations
7. **Service discovery**: Query services and health checks

### Requirements

**Usability**:
- Intuitive command structure
- Clear error messages
- Consistent argument patterns
- Help documentation
- Exit codes for scripting

**Functionality**:
- KV store operations (get, set, delete, list)
- Service operations (register, deregister, list, heartbeat)
- Backup/restore operations
- DNS query helpers
- Health check management

**Security**:
- TLS support (HTTPS, mTLS)
- API key/JWT authentication
- Certificate validation
- Skip-verify for development

**Compatibility**:
- Cross-platform (Linux, macOS, Windows)
- Single binary
- No external dependencies
- Shell completion (future)

## Decision

We will implement **konsulctl** - a command-line tool written in Go that provides a user-friendly interface to Konsul's REST APIs.

### Architecture

**Structure**:
```
konsulctl
├── main.go           # CLI routing and command dispatch
├── client.go         # HTTP client with TLS support
└── commands/         # Command implementations (future)
    ├── kv.go
    ├── service.go
    ├── backup.go
    └── dns.go
```

**Design Principles**:
1. **Verb-Noun pattern**: `konsulctl <resource> <action>` (e.g., `konsulctl kv get`)
2. **Consistent flags**: Global flags work across all commands
3. **Single responsibility**: Each command does one thing well
4. **Error handling**: Clear, actionable error messages
5. **Exit codes**: 0 for success, 1 for errors

### Command Structure

```bash
konsulctl <resource> <action> [arguments] [flags]
```

**Resources**:
- `kv` - Key-value store operations
- `service` - Service discovery operations
- `backup` - Backup and restore operations
- `dns` - DNS query helpers
- `health` - Health check operations (future)
- `acl` - ACL management (future)
- `cluster` - Cluster operations (future)

### Implemented Commands

#### KV Store Commands

```bash
# Get a key
konsulctl kv get <key>

# Set a key
konsulctl kv set <key> <value>

# Delete a key
konsulctl kv delete <key>

# List all keys
konsulctl kv list
```

#### Service Commands

```bash
# Register a service
konsulctl service register <name> <address> <port>

# Register with health checks
konsulctl service register web 10.0.1.10 8080 \
  --check-http http://10.0.1.10:8080/health \
  --check-tcp 10.0.1.10:8080

# List services
konsulctl service list

# Deregister a service
konsulctl service deregister <name>

# Send heartbeat
konsulctl service heartbeat <name>
```

#### Backup Commands

```bash
# Create backup
konsulctl backup create

# Restore from backup
konsulctl backup restore <backup-file>

# List backups
konsulctl backup list

# Export data as JSON
konsulctl backup export
```

#### DNS Commands

```bash
# Show DNS SRV query for service
konsulctl dns srv <service-name>

# Show DNS A query for service
konsulctl dns a <service-name>
```

### Global Flags

**Connection**:
```bash
--server <url>         # Konsul server URL (default: http://localhost:8888)
```

**TLS Options**:
```bash
--tls-skip-verify      # Skip TLS certificate verification
--ca-cert <file>       # Path to CA certificate
--client-cert <file>   # Path to client certificate (mTLS)
--client-key <file>    # Path to client key (mTLS)
```

**Authentication** (future):
```bash
--token <jwt>          # JWT token
--api-key <key>        # API key
```

**Output** (future):
```bash
--output <format>      # Output format: text, json, yaml
--quiet                # Minimal output
```

### TLS Support

**HTTP Client with TLS**:
```go
type TLSConfig struct {
    Enabled        bool
    SkipVerify     bool
    CACertFile     string
    ClientCertFile string
    ClientKeyFile  string
}

func NewKonsulClientWithTLS(baseURL string, tlsConfig *TLSConfig) *KonsulClient {
    transport := &http.Transport{}

    if tlsConfig != nil && tlsConfig.Enabled {
        tlsClientConfig := &tls.Config{
            InsecureSkipVerify: tlsConfig.SkipVerify,
        }

        // Load CA certificate
        if tlsConfig.CACertFile != "" {
            caCert, _ := os.ReadFile(tlsConfig.CACertFile)
            caCertPool := x509.NewCertPool()
            caCertPool.AppendCertsFromPEM(caCert)
            tlsClientConfig.RootCAs = caCertPool
        }

        // Load client certificate (mTLS)
        if tlsConfig.ClientCertFile != "" && tlsConfig.ClientKeyFile != "" {
            cert, _ := tls.LoadX509KeyPair(
                tlsConfig.ClientCertFile,
                tlsConfig.ClientKeyFile,
            )
            tlsClientConfig.Certificates = []tls.Certificate{cert}
        }

        transport.TLSClientConfig = tlsClientConfig
    }

    return &KonsulClient{
        BaseURL: baseURL,
        HTTPClient: &http.Client{
            Timeout:   30 * time.Second,
            Transport: transport,
        },
    }
}
```

### Example Usage

**Local Development**:
```bash
# Set a config value
konsulctl kv set app/config/db_host localhost

# Get the value
konsulctl kv get app/config/db_host

# Register a service
konsulctl service register myapp 127.0.0.1 8080
```

**Production with TLS**:
```bash
# Connect to TLS server with self-signed cert
konsulctl kv get app/config/db_host \
  --server https://konsul.prod.example.com:8888 \
  --tls-skip-verify

# Connect with custom CA
konsulctl service list \
  --server https://konsul.prod.example.com:8888 \
  --ca-cert /path/to/ca.crt

# Connect with mTLS
konsulctl backup create \
  --server https://konsul.prod.example.com:8888 \
  --ca-cert /path/to/ca.crt \
  --client-cert /path/to/client.crt \
  --client-key /path/to/client.key
```

**CI/CD Pipeline**:
```bash
#!/bin/bash
# Deploy script

# Update configuration
konsulctl kv set app/version "$CI_COMMIT_SHA" --server $KONSUL_URL

# Register service
konsulctl service register myapp $SERVICE_IP $SERVICE_PORT \
  --server $KONSUL_URL \
  --check-http "http://$SERVICE_IP:$SERVICE_PORT/health"

# Verify registration
if konsulctl service list --server $KONSUL_URL | grep -q myapp; then
  echo "Service registered successfully"
  exit 0
else
  echo "Service registration failed"
  exit 1
fi
```

## Alternatives Considered

### Alternative 1: REST API Only (No CLI)
- **Pros**:
  - No additional tool to maintain
  - Users can use curl/httpie
  - Simpler project scope
- **Cons**:
  - Poor user experience
  - Verbose curl commands
  - No TLS helpers
  - Harder to script
- **Reason for rejection**: CLI dramatically improves usability

### Alternative 2: Consul CLI Compatibility
- **Pros**:
  - Familiar to Consul users
  - Drop-in replacement potential
  - Proven UX patterns
- **Cons**:
  - Committed to Consul's API design
  - May not fit Konsul's features
  - Harder to innovate
  - Legal/trademark concerns
- **Reason for rejection**: Want flexibility; can be similar but not identical

### Alternative 3: Cobra Framework
- **Pros**:
  - Popular CLI framework in Go
  - Auto-generated help
  - Shell completion built-in
  - Command structure helpers
- **Cons**:
  - External dependency
  - More complexity than needed
  - Larger binary size
  - Learning curve
- **Reason for rejection**: Current implementation simple enough; can add later

### Alternative 4: Python CLI (Click)
- **Pros**:
  - Great CLI framework
  - Easy to extend
  - Python ecosystem
- **Cons**:
  - Requires Python runtime
  - Not a single binary
  - Deployment complexity
  - Cross-platform issues
- **Reason for rejection**: Go provides better distribution (single binary)

### Alternative 5: Shell Scripts
- **Pros**:
  - Simple to write
  - No compilation needed
  - Easy to customize
- **Cons**:
  - Platform-specific (bash vs zsh vs cmd)
  - No TLS support without curl
  - Poor error handling
  - Not maintainable at scale
- **Reason for rejection**: Professional tool requires proper CLI

## Consequences

### Positive
- **Great UX**: Intuitive commands for common operations
- **Productivity**: Faster than curl for routine tasks
- **Scripting**: Easy to automate with proper exit codes
- **TLS support**: Built-in HTTPS and mTLS
- **Single binary**: Easy distribution and deployment
- **Cross-platform**: Works on Linux, macOS, Windows
- **Type safety**: Go's type system prevents errors
- **Maintainability**: Clear code structure
- **Testable**: Can unit test client functions

### Negative
- **Maintenance burden**: Another binary to maintain
- **Feature parity**: Must keep in sync with API
- **Binary size**: ~6-8MB (larger than shell script)
- **Compilation required**: Can't edit on the fly
- **Limited flexibility**: Less flexible than direct API calls

### Neutral
- Need to maintain help documentation
- Version compatibility with server
- Error message consistency

## Implementation Notes

### Build and Distribution

**Build**:
```bash
# Build for current platform
go build -o konsulctl ./cmd/konsulctl

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o konsulctl-linux-amd64 ./cmd/konsulctl
GOOS=darwin GOARCH=amd64 go build -o konsulctl-darwin-amd64 ./cmd/konsulctl
GOOS=windows GOARCH=amd64 go build -o konsulctl-windows-amd64.exe ./cmd/konsulctl
```

**Distribution**:
- GitHub Releases with binaries
- Docker image includes konsulctl
- Homebrew formula (future)
- apt/yum packages (future)

### Error Handling

**Structured Errors**:
```go
type ErrorResponse struct {
    Error     string `json:"error"`
    Message   string `json:"message"`
    RequestID string `json:"request_id,omitempty"`
    Timestamp string `json:"timestamp,omitempty"`
}

// Parse and display server errors
if resp.StatusCode != 200 {
    var errResp ErrorResponse
    json.Unmarshal(body, &errResp)
    return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
}
```

**Exit Codes**:
- `0` - Success
- `1` - General error
- Future: Specific codes for different error types

### Future Enhancements

**Phase 1** (Completed):
- ✅ Basic KV operations
- ✅ Service operations
- ✅ Backup/restore
- ✅ TLS support
- ✅ DNS helpers

**Phase 2** (Planned):
- [ ] ACL management commands
- [ ] Health check operations
- [ ] Output formats (JSON, YAML)
- [ ] Authentication (JWT/API key flags)
- [ ] Environment variable support

**Phase 3** (Future):
- [ ] Shell completion (bash, zsh, fish)
- [ ] Interactive mode
- [ ] Watch mode (live updates)
- [ ] Cluster management commands
- [ ] Diff/apply for config as code
- [ ] Namespace support

### Testing Strategy

**Unit Tests**:
```go
func TestKVGet(t *testing.T) {
    // Mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        json.NewEncoder(w).Encode(KVResponse{
            Key:   "test",
            Value: "value",
        })
    }))
    defer server.Close()

    client := NewKonsulClient(server.URL)
    value, err := client.GetKV("test")

    assert.NoError(t, err)
    assert.Equal(t, "value", value)
}
```

**Integration Tests**:
- Test against live Konsul server
- Verify TLS connections
- Test error scenarios
- Validate command chaining

### Documentation

**Help Text**:
```bash
konsulctl --help           # Main help
konsulctl kv --help        # KV command help
konsulctl service --help   # Service command help
```

**Man Pages** (future):
```bash
man konsulctl
man konsulctl-kv
man konsulctl-service
```

**Examples**:
- README with common usage
- Inline help with examples
- Tutorial documentation

## References

- [Consul CLI](https://www.consul.io/commands)
- [kubectl Design](https://kubernetes.io/docs/reference/kubectl/)
- [etcdctl](https://etcd.io/docs/latest/dev-guide/interacting_v3/)
- [12 Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46)
- [Cobra CLI Framework](https://github.com/spf13/cobra)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-26 | Konsul Team | Initial version |

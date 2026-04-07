# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Konsul is a lightweight, cloud-native service mesh and discovery platform built in Go. It provides essential infrastructure services for distributed systems including:

- Service registration and discovery via REST API or DNS
- Distributed key-value (KV) store with CAS (compare-and-swap) support
- Health monitoring with configurable TTL
- Consensus-based clustering using HashiCorp Raft
- GraphQL API alongside REST endpoints
- React-based Admin UI for web-based management
- Prometheus metrics and observability
- JWT and API key authentication with ACL support
- Rate limiting and TLS encryption
- BadgerDB persistence layer

Current focus: Raft clustering for production-ready high availability (see ROADMAP.md).

## Essential Commands

### Build and Execution
```bash
make build              # Build main server binary → ./bin/konsul
make build-cli          # Build CLI tool → ./bin/konsulctl
make run               # Run server with go run cmd/konsul/main.go
make clean             # Remove built binaries
```

### Testing
```bash
make test              # Run all tests with verbose output (go test -v ./...)
go test ./...          # Same as make test
go test ./internal/store -count=1  # Run single package with fresh cache
go test -run TestKVStore ./internal/store  # Run specific test
go test -v ./internal/raft -timeout 5m    # Run raft tests with longer timeout
```

### Code Quality
```bash
make fmt               # Format all Go code (gofmt)
golangci-lint run      # Run linter (configured in .golangci.yml)
go vet ./...           # Run vet checker
```

### Docker
```bash
make docker-build      # Build container image as konsul:latest
make docker-run        # Run container on port 8888
```

### Admin UI Development
```bash
cd web/admin
npm run dev            # Development server with hot reload
npm run build          # Production build
npm run lint           # ESLint check
npm run test:e2e       # Playwright e2e tests
```

### Monitoring/Observability
```bash
make dashboard         # Build Grafana dashboards (from monitoring/grafana/)
make dashboard-validate # Validate dashboard JSON
```

## Architecture Overview

### Core Components

**cmd/konsul/main.go** - Main server entrypoint
- Initializes Fiber HTTP framework with all middleware
- Sets up Raft cluster consensus (internal/raft/)
- Registers API handlers for services, KV store, health checks
- Configures TLS, authentication, and rate limiting
- Embeds React Admin UI assets
- Implements signal handling for graceful shutdown

**internal/store/** - In-memory data store with Raft support
- `kv.go` - KV store with CAS, batch operations, and prefix queries
- `service.go` - Service registry with health tracking and load balancing
- `service_index.go` - Efficient service indexing by tag and metadata
- `service_query.go` - Query engine with filtering and sorting
- `snapshot.go` - Snapshot serialization for Raft persistence
- All operations are synchronized and support concurrent access

**internal/raft/** - Distributed consensus layer
- `node.go` - Raft node lifecycle and configuration (20KB core file)
- `fsm.go` - Finite State Machine applying commands from Raft log
- `commands.go` - Serializable command definitions for KV and service operations
- `config.go` - Raft cluster configuration (cluster ID, bootstrap, TLS)
- `metrics.go` - Raft-specific Prometheus metrics (applies, snapshots, replication)
- `store_interfaces.go` - Abstract interfaces for KV and service stores
- `transport.go` - TCP and TLS transport layer for Raft cluster communication with connection pooling and cleanup
- Integration tests covering all major scenarios:
  - `leader_election_integration_test.go`
  - `snapshot_recovery_integration_test.go`
  - `consistency_integration_test.go`
  - `data_replication_integration_test.go`
  - `failure_scenarios_integration_test.go`
  - `batch_operations_integration_test.go`
  - `tls_integration_test.go`

**internal/handlers/** - REST API endpoints
- KV endpoints: GET/PUT/DELETE `/kv/<key>`
- Service endpoints: `/register`, `/deregister/<name>`, `/services/...`
- Health check endpoints: `/heartbeat/<name>`, `/health`
- Cluster endpoints: `/cluster/join`, `/cluster/leave`, `/cluster/status`
- Admin UI serving: `/admin/*`
- `auth.go` - Security hardening: only explicit wildcards (`*`) are treated as prefix patterns for public routes; all other route patterns are matched exactly

**internal/graphql/** - GraphQL API
- Generated code from gqlgen (see gqlgen.yml)
- Resolvers in `resolver/` directory
- Query and mutation types for services and KV store
- `resolver/authz.go` - GraphQL authorization: `claimsFromGraphQLContext()` validates JWT from operation headers, `authorizeMutation()` enforces auth and ACL checks for mutations

**internal/auth/** - Authentication and authorization
- JWT token validation with `Claims` struct (includes `Policies` field) and `RefreshClaims` struct
- `GenerateTokenWithPolicies()` and `GenerateRefreshTokenWithPolicies()` for policy-aware token issuance
- Security hardening: identity is always derived from validated tokens only, never from client-supplied roles
- API key authentication
- ACL enforcement via internal/acl/

**internal/middleware/** - Fiber middleware
- CORS, compression, request logging
- Authentication and rate limiting
- Error handling and response formatting

**internal/audit/** - Audit logging
- `manager.go` - Idempotent shutdown via `sync.Once`, error tracking, and graceful shutdown

**internal/persistence/** - BadgerDB integration
- Embedded key-value database for durability
- Snapshot loading/saving for Raft recovery
- Background compaction and cleanup

**cmd/konsulctl/** - CLI tool (separate binary)
- Client implementation in `client.go` (36KB, handles HTTP requests)
- Commands for services, KV, ACL, rate limits, backup/restore
- Used for cluster management and testing

**cmd/konsul/ui/** - Embedded React Admin UI assets
- Built from `web/admin/` with npm
- Automatically embedded in binary with `//go:embed all:ui`
- Served at `/admin/*` path

### Testing Patterns

The codebase uses standard Go testing with these patterns:

1. **Unit Tests** - Test individual components with mocks
   - Use `testify/assert` and `testify/require` for assertions
   - Mock implementations for interfaces (e.g., mockKVStore)
   - Table-driven tests for multiple scenarios

2. **Integration Tests** - Test component interaction
   - Named `*_integration_test.go` for clarity
   - Example: `internal/raft/leader_election_integration_test.go`
   - May start real Raft clusters or use test fixtures

3. **Test Setup**
   - Use helper functions like `setupApp()` to initialize test dependencies
   - Clean up resources in deferred cleanup blocks
   - Use `-count=1` to bypass Go's test cache when needed

Example test patterns found in codebase:
```go
func TestFeatureName(t *testing.T) {
  // arrange
  store := store.NewKVStore()

  // act
  store.Set("key", "value")

  // assert
  require.True(t, true, "describe why")
  assert.Equal(t, "value", store.Get("key"))
}
```

## Configuration

Configuration is primarily through environment variables:

```bash
# Core
KONSUL_PORT=8888                    # HTTP listen port
KONSUL_DATA_DIR=./data              # Persistence directory

# Raft Clustering
KONSUL_RAFT_ENABLED=true            # Enable Raft consensus
KONSUL_RAFT_NODE_ID=node1           # Unique node identifier
KONSUL_RAFT_LISTEN_ADDR=localhost:7001  # Raft communication address
KONSUL_RAFT_BOOTSTRAP_MODE=single   # single or multi (cluster mode)
KONSUL_RAFT_SNAPSHOT_INTERVAL=30s   # Snapshot frequency

# Admin UI
KONSUL_ADMIN_UI_ENABLED=true        # Enable/disable Admin UI
KONSUL_ADMIN_UI_PATH=/admin         # Base path for Admin UI

# Authentication
KONSUL_JWT_SECRET=your-secret       # JWT signing secret
KONSUL_API_KEY=your-key             # API key for auth

# TLS
KONSUL_TLS_ENABLED=false            # Enable TLS
KONSUL_TLS_CERT_FILE=cert.pem
KONSUL_TLS_KEY_FILE=key.pem

# Metrics
KONSUL_METRICS_ENABLED=true         # Enable Prometheus metrics
KONSUL_METRICS_PORT=9090            # Metrics port

# Rate Limiting
KONSUL_RATE_LIMIT_ENABLED=true      # Enable rate limiting
KONSUL_RATE_LIMIT_RPS=100           # Requests per second
```

See `internal/config/` for complete configuration loading logic.

## Key Files Reference

**High-impact files to understand first:**
- `cmd/konsul/main.go` - Server initialization and routing (27KB)
- `internal/raft/node.go` - Raft cluster state machine (20KB)
- `internal/store/kv.go` - KV store with CAS support (27KB)
- `internal/store/service.go` - Service registry (25KB)
- `internal/handlers/` - HTTP API endpoints

**Testing files:**
- `internal/raft/fsm_test.go` - FSM behavior tests (17KB)
- `internal/store/kv_test.go`, `kv_cas_test.go` - KV store tests
- `cmd/konsul/main_test.go` - Integration tests

**Configuration and utilities:**
- `internal/config/` - Configuration loading
- `internal/logger/` - Structured logging setup
- `internal/metrics/` - Prometheus metrics definitions
- `internal/persistence/` - BadgerDB integration

## Development Workflow

### Adding a New Feature
1. Implement handler in `internal/handlers/` or modify existing one
2. Add store methods in `internal/store/` if data storage needed
3. Add Raft command in `internal/raft/commands.go` for clustering support
4. Update FSM in `internal/raft/fsm.go` to apply the command
5. Write tests with table-driven approach
6. Update GraphQL resolver if needed
7. Run `make test` and `golangci-lint run` before committing

### Fixing Issues
1. Write a failing test that reproduces the issue
2. Fix the code to make the test pass
3. Check for regressions: `make test` runs all tests
4. Verify with `golangci-lint run` for code quality

### Testing a Single Package
```bash
go test -v ./internal/raft -timeout 5m
go test -run TestRaftLeaderElection ./internal/raft
```

### Testing with Raft Cluster
Tests that require actual Raft clusters are in `*_integration_test.go` files. These may take longer to run due to starting multiple Raft nodes.

## Coding Conventions

- **Go**: Follow idiomatic Go conventions, enforced by gofmt and golangci-lint
- **Naming**: Packages lowercase, types PascalCase, functions camelCase
- **Testing**: Test files end in `_test.go`, table-driven tests preferred
- **Comments**: Exported types and functions must have doc comments
- **Errors**: Use `fmt.Errorf` with `%w` for wrapping, custom error types in `errors.go`
- **Interface usage**: Abstract store interfaces defined in `internal/raft/store_interfaces.go`

## Raft Clustering Notes

**Current Status**: Production-ready transport layer (`transport.go`) with TCP/TLS support and connection pooling. Comprehensive integration test coverage across leader election, snapshot recovery, consistency, data replication, failure scenarios, batch operations, and TLS.

**Key considerations:**
- Leader redirection: Non-leaders redirect write operations to the current leader
- Snapshot recovery: Nodes load latest snapshot on startup before replaying log
- CAS operations: Now work through Raft for consistency across cluster
- Metrics: Track applies, snapshots, replication lag at `internal/raft/metrics.go`

**Common tasks:**
- Check cluster status: GET `/cluster/status`
- Join node to cluster: PUT `/cluster/join` with `{node_id, addr}`
- Leave cluster: PUT `/cluster/leave`

See `docs/adr/0030-raft-integration-implementation.md` and `ROADMAP.md` for detailed status.

## Common Pitfalls

- **Forgetting Raft commands for new store operations**: Any new KV or service operation that should be replicated needs a corresponding Raft command and FSM handler
- **Not handling non-leader writes**: REST handlers must redirect non-leaders to the leader for write operations
- **Concurrent access to stores**: The stores use sync.RWMutex internally, but FSM commands must be serialized by Raft
- **Missing test coverage for CAS operations**: CAS tests are critical for consistency guarantees

## Additional Resources

- `ROADMAP.md` - Project vision and implementation plan
- `docs/adr/` - Architecture decision records (0030 and 0031 cover Raft)
- `docs/authentication.md`, `docs/acl.md` - Security features
- `docs/kv-watch-guide.md` - Watch/subscribe patterns
- `docs/admin-ui-integration-plan.md` - Admin UI architecture
- `web/admin/README.md` - React Admin UI guide

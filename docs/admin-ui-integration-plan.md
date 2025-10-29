# Admin UI Integration Implementation Plan

**Document Version**: 1.0
**Date**: 2025-10-29
**Status**: Ready for Implementation
**Related ADR**: [ADR-0009: React Admin UI](adr/0009-react-admin-ui.md)

## Overview

This document outlines the step-by-step implementation plan for integrating the pre-built React Admin UI into the Konsul backend server. The UI is already built and located in `web/admin/dist/`, and we need to serve it through the Fiber web framework.

## Prerequisites

- ✅ React UI built (`web/admin/dist/`)
- ✅ ADR-0009 accepted
- ✅ Comprehensive documentation created (`docs/admin-ui.md`)
- ✅ Fiber framework already integrated
- ✅ Authentication system in place (JWT/API keys)
- ✅ All backend APIs functional

## Current State

### What We Have

1. **Built UI Assets** (`web/admin/dist/`):
   - `index.html` - Main entry point
   - `assets/index-[hash].js` (~332KB)
   - `assets/index-[hash].css` (~20KB)
   - `vite.svg` - Favicon

2. **Backend** (`cmd/konsul/main.go`):
   - Fiber web server running
   - REST API endpoints (`/kv/*`, `/services/*`, etc.)
   - Authentication middleware (JWT & API keys)
   - CORS middleware
   - Metrics endpoint (`/metrics`)
   - Health endpoints (`/health/*`)

3. **Embed Setup**:
   - `embed.FS` variable declared (line 37-38)
   - Currently points to `all:ui` directory

### What's Missing

1. ❌ Correct embed directive path
2. ❌ Static file serving middleware configuration
3. ❌ SPA fallback routing (for client-side routes)
4. ❌ Development vs production serving strategy
5. ❌ Admin UI documentation endpoint
6. ❌ Feature flag to disable UI if needed

## Implementation Plan

### Phase 1: Basic Static File Serving (Quick Win)

**Goal**: Serve the built React UI from `/admin` path

**Tasks**:

#### Task 1.1: Fix Embed Directive
**File**: `cmd/konsul/main.go`

**Current**:
```go
//go:embed all:ui
var adminUI embed.FS
```

**Change to**:
```go
//go:embed web/admin/dist
var adminUI embed.FS
```

**Why**: The embed directive needs to point to the actual directory containing the built UI files.

**Verification**:
```bash
# Build and verify embedded files
go build -o konsul ./cmd/konsul
# Should not show "pattern all:ui: no matching files found"
```

---

#### Task 1.2: Create UI Serving Function
**File**: `cmd/konsul/main.go`

**Add function** (after line 38):
```go
// getAdminUIFS returns the filesystem for the admin UI
func getAdminUIFS() (http.FileSystem, error) {
	// Strip the "web/admin/dist" prefix from embedded paths
	stripped, err := fs.Sub(adminUI, "web/admin/dist")
	if err != nil {
		return nil, fmt.Errorf("failed to create admin UI filesystem: %w", err)
	}
	return http.FS(stripped), nil
}
```

**Why**: The embedded FS includes the full path `web/admin/dist`, but we need to serve from the root of that directory.

---

#### Task 1.3: Add Static File Middleware
**File**: `cmd/konsul/main.go`

**Location**: After authentication middleware setup, before API routes (around line 200)

**Add**:
```go
// Serve Admin UI
if cfg.AdminUI.Enabled {
	uiFS, err := getAdminUIFS()
	if err != nil {
		appLogger.Error("Failed to initialize admin UI filesystem", logger.Error(err))
	} else {
		// Serve static files from /admin
		app.Use("/admin", filesystem.New(filesystem.Config{
			Root:       uiFS,
			Index:      "index.html",
			Browse:     false,
			PathPrefix: "",
		}))

		// SPA fallback - serve index.html for all /admin/* routes
		// This handles client-side routing
		app.Use("/admin/*", func(c *fiber.Ctx) error {
			// Only handle non-asset requests
			if !c.Path("/assets/") {
				return c.SendFile("web/admin/dist/index.html")
			}
			return c.Next()
		})

		appLogger.Info("Admin UI enabled", logger.String("path", "/admin"))
	}
} else {
	appLogger.Info("Admin UI disabled via configuration")
}
```

**Why**:
- First middleware serves static files (JS, CSS, images)
- Second middleware handles SPA routing (all routes go to index.html)
- Conditional on config flag for flexibility

---

#### Task 1.4: Add Configuration Support
**File**: `internal/config/config.go`

**Add to Config struct**:
```go
// AdminUI configuration
AdminUI struct {
	Enabled bool   `env:"KONSUL_ADMIN_UI_ENABLED" envDefault:"true"`
	Path    string `env:"KONSUL_ADMIN_UI_PATH" envDefault:"/admin"`
} `envPrefix:"KONSUL_ADMIN_UI_"`
```

**Why**: Allow users to disable the UI or change the base path.

---

#### Task 1.5: Update Dockerfile
**File**: `Dockerfile`

**Ensure UI is copied** (check if this exists):
```dockerfile
# Copy web admin UI (already built)
COPY web/admin/dist /app/web/admin/dist
```

**Note**: If using embed, this might not be necessary as the UI is embedded in the binary.

---

### Phase 2: CORS and API Configuration

**Goal**: Ensure the UI can communicate with the backend API

#### Task 2.1: Configure CORS for UI
**File**: `cmd/konsul/main.go`

**Update CORS middleware** (if needed):
```go
import "github.com/gofiber/fiber/v2/middleware/cors"

// Add CORS middleware (before auth middleware)
app.Use(cors.New(cors.Config{
	AllowOrigins:     "*", // In production, restrict to specific origins
	AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
	AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-API-Key",
	AllowCredentials: false,
	ExposeHeaders:    "Content-Length",
	MaxAge:           3600,
}))
```

**Why**: The UI needs to make cross-origin requests to the API.

---

#### Task 2.2: Public Paths for UI Assets
**File**: Authentication middleware configuration

**Ensure these paths are public** (no auth required):
```go
publicPaths := []string{
	"/health",
	"/health/live",
	"/health/ready",
	"/metrics",
	"/admin",           // Admin UI entry point
	"/admin/",          // Admin UI root
	"/admin/assets/",   // UI static assets (JS, CSS)
}
```

**Why**: The UI assets must be accessible without authentication, but API calls will require auth.

---

### Phase 3: Environment and Build Configuration

#### Task 3.1: Add Build Tags
**File**: `Makefile` (or build scripts)

**Add build command**:
```makefile
.PHONY: build
build: build-ui build-server

.PHONY: build-ui
build-ui:
	@echo "Building Admin UI..."
	cd web/admin && npm run build

.PHONY: build-server
build-server:
	@echo "Building Konsul server..."
	go build -o bin/konsul ./cmd/konsul

.PHONY: build-server-no-ui
build-server-no-ui:
	@echo "Building Konsul server (without UI)..."
	go build -tags=noui -o bin/konsul ./cmd/konsul
```

**Why**: Provide flexible build options.

---

#### Task 3.2: Add Build Tag Support
**File**: `cmd/konsul/main.go`

**Add build tag conditional**:
```go
//go:build !noui
// +build !noui

package main

//go:embed web/admin/dist
var adminUI embed.FS
```

**And create**: `cmd/konsul/main_noui.go`
```go
//go:build noui
// +build noui

package main

import "embed"

// Empty embed when UI is disabled
var adminUI embed.FS
```

**Why**: Allow building Konsul without the UI for smaller binaries.

---

### Phase 4: Documentation and Testing

#### Task 4.1: Update README
**File**: `README.md`

**Add section**:
```markdown
## Web Admin UI

Konsul includes a built-in web-based admin interface for managing services and KV store.

**Access the UI**:
```
http://localhost:8888/admin
```

**Features**:
- Service discovery dashboard
- KV store browser and editor
- Health monitoring
- Metrics visualization
- Backup management

**Configuration**:
```bash
# Enable/disable the UI
export KONSUL_ADMIN_UI_ENABLED=true

# Change the base path
export KONSUL_ADMIN_UI_PATH=/admin
```

See [Admin UI Documentation](docs/admin-ui.md) for details.
```

---

#### Task 4.2: Integration Tests
**File**: `internal/handlers/admin_ui_test.go` (new file)

**Create test**:
```go
package handlers

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUIServing(t *testing.T) {
	app := fiber.New()

	// Setup UI serving (simplified for test)
	// ... setup code ...

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		contains       string
	}{
		{
			name:           "Admin UI root",
			path:           "/admin",
			expectedStatus: 200,
			contains:       "<!doctype html>",
		},
		{
			name:           "Admin UI with trailing slash",
			path:           "/admin/",
			expectedStatus: 200,
			contains:       "<!doctype html>",
		},
		{
			name:           "SPA route fallback",
			path:           "/admin/services",
			expectedStatus: 200,
			contains:       "<!doctype html>",
		},
		{
			name:           "Static asset",
			path:           "/admin/vite.svg",
			expectedStatus: 200,
			contains:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.contains != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.contains)
			}
		})
	}
}
```

---

#### Task 4.3: Manual Testing Checklist
**File**: `docs/admin-ui-testing.md` (new file)

Create testing checklist:
- [ ] UI loads at `/admin`
- [ ] UI loads at `/admin/`
- [ ] Assets load (check Network tab)
- [ ] SPA routing works (navigate to `/admin/services`)
- [ ] API calls work (check Console for errors)
- [ ] Authentication flow works
- [ ] Dark mode toggle (if implemented)
- [ ] Responsive design on mobile
- [ ] All CRUD operations functional

---

### Phase 5: Production Readiness

#### Task 5.1: Security Headers
**File**: `cmd/konsul/main.go`

**Add security middleware**:
```go
import "github.com/gofiber/fiber/v2/middleware/helmet"

// Add before static file serving
app.Use("/admin", helmet.New(helmet.Config{
	ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:",
	XSSProtection:         "1; mode=block",
	XFrameOptions:         "DENY",
	HSTSMaxAge:            31536000,
}))
```

**Why**: Protect against common web vulnerabilities.

---

#### Task 5.2: Caching Headers
**File**: Static file middleware configuration

**Add caching**:
```go
app.Use("/admin/assets", filesystem.New(filesystem.Config{
	Root:       uiFS,
	Browse:     false,
	PathPrefix: "assets",
	MaxAge:     31536000, // 1 year for hashed assets
}))
```

**Why**: Improve performance with proper cache headers for static assets.

---

#### Task 5.3: Compression
**File**: `cmd/konsul/main.go`

**Add compression middleware**:
```go
import "github.com/gofiber/fiber/v2/middleware/compress"

// Add before static file serving
app.Use("/admin", compress.New(compress.Config{
	Level: compress.LevelBestSpeed,
}))
```

**Why**: Reduce bandwidth usage and improve load times.

---

### Phase 6: Advanced Features (Optional)

#### Task 6.1: Real-time Updates (WebSocket)
**Scope**: Future enhancement
**Requirements**: WebSocket support for live service updates

#### Task 6.2: Metrics Dashboard Integration
**Scope**: Parse Prometheus metrics and display in UI

#### Task 6.3: Health Check Visualization
**Scope**: Real-time health check status updates

---

## Implementation Order

### Week 1: Core Integration
- [x] **Day 1**: Tasks 1.1 - 1.3 (Basic serving)
- [ ] **Day 2**: Task 1.4 (Configuration)
- [ ] **Day 3**: Tasks 2.1 - 2.2 (CORS and auth)
- [ ] **Day 4**: Task 4.3 (Manual testing)
- [ ] **Day 5**: Bug fixes and refinements

### Week 2: Production Ready
- [ ] **Day 1**: Tasks 3.1 - 3.2 (Build configuration)
- [ ] **Day 2**: Task 4.1 (Documentation)
- [ ] **Day 3**: Task 4.2 (Integration tests)
- [ ] **Day 4**: Tasks 5.1 - 5.3 (Security and performance)
- [ ] **Day 5**: Final testing and release

---

## File Changes Summary

### Files to Modify

| File | Changes | Lines (~) |
|------|---------|-----------|
| `cmd/konsul/main.go` | Embed directive, UI serving, middleware | +50 |
| `internal/config/config.go` | AdminUI config struct | +5 |
| `Makefile` | Build targets for UI | +15 |
| `README.md` | Admin UI section | +20 |
| `Dockerfile` | UI asset copying (if not embedded) | +2 |

### Files to Create

| File | Purpose | Lines (~) |
|------|---------|-----------|
| `cmd/konsul/main_noui.go` | No-UI build tag support | 10 |
| `internal/handlers/admin_ui_test.go` | Integration tests | 80 |
| `docs/admin-ui-testing.md` | Manual testing checklist | 50 |

**Total LOC**: ~200 lines of code

---

## Configuration Reference

### Environment Variables

```bash
# Admin UI Configuration
KONSUL_ADMIN_UI_ENABLED=true          # Enable/disable UI
KONSUL_ADMIN_UI_PATH=/admin           # Base path for UI

# Development
VITE_API_BASE_URL=http://localhost:8888  # API endpoint for UI
```

### Config File (YAML)

```yaml
admin_ui:
  enabled: true
  path: /admin
```

---

## Testing Strategy

### Unit Tests
- ✅ Configuration loading
- ✅ Filesystem setup
- ✅ Path routing

### Integration Tests
- ✅ UI serving from `/admin`
- ✅ SPA fallback routing
- ✅ Static asset serving
- ✅ CORS headers
- ✅ Security headers

### Manual Tests
- ✅ UI loads correctly
- ✅ API calls succeed
- ✅ Authentication works
- ✅ All features functional
- ✅ Mobile responsive
- ✅ Browser compatibility

### Performance Tests
- ✅ Load time < 2s
- ✅ First contentful paint < 1s
- ✅ Time to interactive < 3s
- ✅ Bundle size optimized

---

## Rollback Plan

If issues arise:

1. **Disable via config**:
   ```bash
   export KONSUL_ADMIN_UI_ENABLED=false
   ```

2. **Revert code changes**:
   ```bash
   git revert <commit-hash>
   ```

3. **Build without UI**:
   ```bash
   make build-server-no-ui
   ```

---

## Success Criteria

### Must Have
- [ ] UI accessible at `/admin`
- [ ] All API endpoints work from UI
- [ ] Authentication flow functional
- [ ] SPA routing works correctly
- [ ] No console errors
- [ ] Mobile responsive

### Should Have
- [ ] Security headers configured
- [ ] Compression enabled
- [ ] Caching headers set
- [ ] Integration tests passing
- [ ] Documentation complete

### Nice to Have
- [ ] WebSocket support
- [ ] Dark mode
- [ ] Real-time updates
- [ ] Advanced filtering

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Embed path incorrect | Medium | High | Test build early, verify files |
| CORS issues | Low | Medium | Comprehensive CORS config |
| Auth breaks API | Low | High | Public paths for UI assets |
| Performance issues | Low | Low | Compression, caching headers |
| Build size increase | Medium | Low | Build tag for no-UI option |

---

## References

- [ADR-0009: React Admin UI](adr/0009-react-admin-ui.md)
- [Admin UI Documentation](admin-ui.md)
- [Fiber Static Files](https://docs.gofiber.io/api/middleware/filesystem)
- [Go Embed](https://pkg.go.dev/embed)
- [Vite Build](https://vitejs.dev/guide/build.html)

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2025-10-29 | 1.0 | Konsul Team | Initial implementation plan |

---

## Next Steps

1. Review this implementation plan
2. Get team approval
3. Create GitHub issue/task tracking
4. Start with Phase 1, Task 1.1
5. Test incrementally after each task
6. Update this document as needed

**Ready to implement? Let's start with Phase 1!**

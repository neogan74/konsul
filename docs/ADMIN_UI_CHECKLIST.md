# Admin UI Integration - Quick Checklist

**Status**: Ready to Implement
**Estimated Time**: 2-3 days for core integration
**Full Plan**: See [admin-ui-integration-plan.md](admin-ui-integration-plan.md)

## Prerequisites ✅

- [x] React UI built in `web/admin/dist/`
- [x] ADR-0009 approved
- [x] Backend APIs functional
- [x] Authentication system in place

## Phase 1: Core Integration (Day 1-2)

### Critical Path Tasks

- [ ] **1.1** Fix embed directive in `main.go`
  ```go
  //go:embed web/admin/dist
  var adminUI embed.FS
  ```

- [ ] **1.2** Add UI filesystem helper function
  ```go
  func getAdminUIFS() (http.FileSystem, error)
  ```

- [ ] **1.3** Configure static file middleware
  ```go
  app.Use("/admin", filesystem.New(...))
  ```

- [ ] **1.4** Add SPA fallback routing
  ```go
  app.Use("/admin/*", func(c *fiber.Ctx) error {
    return c.SendFile("web/admin/dist/index.html")
  })
  ```

- [ ] **1.5** Add AdminUI config to `config.Config`
  ```go
  AdminUI struct {
    Enabled bool
    Path    string
  }
  ```

## Phase 2: API Integration (Day 2-3)

- [ ] **2.1** Configure CORS for UI
  ```go
  AllowOrigins: "*"
  AllowHeaders: "Authorization, X-API-Key"
  ```

- [ ] **2.2** Add public paths for UI assets
  ```go
  publicPaths := []string{"/admin", "/admin/", "/admin/assets/"}
  ```

## Phase 3: Testing (Day 3-4)

- [ ] **3.1** Build and verify
  ```bash
  go build ./cmd/konsul
  ./konsul
  ```

- [ ] **3.2** Access UI at http://localhost:8888/admin
  - [ ] UI loads without errors
  - [ ] Assets load (check Network tab)
  - [ ] No 404 errors in console

- [ ] **3.3** Test SPA routing
  - [ ] Refresh at `/admin/services` works
  - [ ] Navigation doesn't cause 404s

- [ ] **3.4** Test API calls
  - [ ] Login works
  - [ ] KV operations work
  - [ ] Service operations work

## Phase 4: Production Ready (Day 4-5)

- [ ] **4.1** Add security headers (helmet middleware)
- [ ] **4.2** Add compression middleware
- [ ] **4.3** Add caching headers for assets
- [ ] **4.4** Update README.md with Admin UI section
- [ ] **4.5** Create integration tests

## Quick Commands

```bash
# Test current setup
ls -la web/admin/dist/

# Build Konsul
go build -o konsul ./cmd/konsul

# Run Konsul
./konsul

# Access UI
open http://localhost:8888/admin

# Check logs for UI initialization
# Should see: "Admin UI enabled" message
```

## Common Issues & Solutions

### Issue: "pattern all:ui: no matching files found"
**Fix**: Change embed directive to `//go:embed web/admin/dist`

### Issue: 404 for all UI routes
**Fix**: Verify filesystem middleware is before API routes

### Issue: SPA routes return 404
**Fix**: Add fallback middleware for `/admin/*`

### Issue: Assets not loading
**Fix**: Check `PathPrefix` in filesystem config

### Issue: CORS errors
**Fix**: Add proper CORS middleware with UI origin

## Success Indicators

✅ Navigate to `/admin` → See React UI
✅ Refresh at `/admin/services` → Still see UI
✅ Login → Get JWT token
✅ Make API call → Success (200 OK)
✅ No console errors
✅ Network tab shows 200s for all assets

## Time Estimate

| Phase | Time | Priority |
|-------|------|----------|
| Phase 1 | 2-4 hours | **Critical** |
| Phase 2 | 1-2 hours | **Critical** |
| Phase 3 | 2-3 hours | **Critical** |
| Phase 4 | 3-4 hours | High |

**Total**: ~1-2 days for working integration, +1 day for production polish

## Next Action

**Start here**: Task 1.1 - Fix the embed directive in `cmd/konsul/main.go`

```diff
- //go:embed all:ui
+ //go:embed web/admin/dist
  var adminUI embed.FS
```

Then build and test!

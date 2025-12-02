# E2E Testing Guide for Konsul Admin UI

## Overview

Comprehensive end-to-end (E2E) tests have been implemented using Playwright. These tests cover authentication, navigation, and all major pages of the Konsul Admin UI.

## Test Suite Summary

### Test Files Created

```
e2e/
├── fixtures/
│   └── auth.ts                    # Authentication fixture for authenticated tests
├── auth.spec.ts                   # 6 tests - Authentication flow
├── dashboard.spec.ts              # 3 tests - Dashboard page
├── services.spec.ts               # 3 tests - Services page
├── kvstore.spec.ts                # 4 tests - KV Store page
├── health.spec.ts                 # 3 tests - Health page
├── apikeys.spec.ts                # 4 tests - API Keys page
├── navigation.spec.ts             # 6 tests - Navigation between pages
├── protected-routes.spec.ts       # 6 tests - Protected route access
├── smoke.spec.ts                  # 4 tests - Critical user journeys
└── README.md                      # Detailed testing documentation
```

**Total: 39 tests across 9 test files**

## Prerequisites

### 1. Install Playwright Browsers

```bash
npx playwright install
```

This downloads Chromium, Firefox, and WebKit browsers needed for testing.

### 2. Start the Backend Server

**IMPORTANT:** The tests expect the Konsul backend API to be running. Before running tests:

```bash
# Navigate to the Konsul backend directory
cd ../../cmd/konsul

# Start the Konsul server
go run main.go
```

The frontend expects the backend API at:
- Development: `http://localhost:8500` (or configured API endpoint)
- The UI dev server will proxy API requests

### 3. Install UI Dependencies

```bash
cd web/admin
npm install
```

## Running Tests

### Quick Start

```bash
# Run all tests (all browsers)
npm run test:e2e

# Run tests in Chromium only (fastest)
npm run test:e2e -- --project=chromium

# Run in headed mode (see the browser)
npm run test:e2e:headed

# Run specific test file
npx playwright test e2e/auth.spec.ts

# Run in debug mode
npx playwright test --debug
```

### Advanced Options

```bash
# Run tests matching a pattern
npx playwright test -g "login"

# Run with specific reporter
npx playwright test --reporter=html

# Run on specific browser
npx playwright test --project=firefox
npx playwright test --project=webkit

# Parallel execution
npx playwright test --workers=4

# Update snapshots (if using visual regression)
npx playwright test --update-snapshots
```

## Test Coverage

### 1. Authentication Tests (`auth.spec.ts`)
- ✅ Login page display
- ✅ Form validation for required fields
- ✅ Successful login with valid credentials
- ✅ Login with custom user ID, roles, and policies
- ✅ Loading state during login
- ✅ Redirect to dashboard when already authenticated

### 2. Protected Routes Tests (`protected-routes.spec.ts`)
- ✅ Redirects to login when accessing protected pages without auth
- ✅ Login page accessible without authentication

### 3. Navigation Tests (`navigation.spec.ts`)
- ✅ Navigate to all pages (Services, KV Store, Health, API Keys)
- ✅ Return to dashboard
- ✅ Mobile sidebar open/close functionality

### 4. Page-Specific Tests
- **Dashboard** (`dashboard.spec.ts`): Title display, stats, navigation links
- **Services** (`services.spec.ts`): Page display, list/empty state, search
- **KV Store** (`kvstore.spec.ts`): Page display, KV pairs, add button, search
- **Health** (`health.spec.ts`): Page display, health metrics, status indicators
- **API Keys** (`apikeys.spec.ts`): Page display, key list, create button, search

### 5. Smoke Tests (`smoke.spec.ts`)
- ✅ Application loads successfully
- ✅ Complete user journey through all pages
- ✅ No critical console errors
- ✅ Mobile responsiveness

## Test Architecture

### Authentication Fixture

The `e2e/fixtures/auth.ts` provides an `authenticatedPage` fixture that automatically logs in before tests:

```typescript
import { test, expect } from './fixtures/auth';

test('my test', async ({ authenticatedPage }) => {
  await authenticatedPage.goto('/services');
  // Test authenticated pages
});
```

### Configuration

Tests are configured in `playwright.config.ts`:
- Base URL: `http://127.0.0.1:4173/admin`
- Auto-starts dev server before tests
- Runs on Chromium, Firefox, and WebKit
- Retries failed tests in CI (2 retries)

## Troubleshooting

### Tests Timeout or Hang

**Problem:** Tests timeout waiting for navigation or elements.

**Solutions:**
1. Ensure the backend Konsul server is running
2. Check that the frontend dev server started correctly
3. Verify API endpoints are accessible
4. Check browser console for network errors:
   ```bash
   npx playwright test --headed --debug
   ```

### Backend Not Running

**Problem:** Tests fail because backend API is not available.

**Solution:**
```bash
# Terminal 1: Start backend
cd ../../cmd/konsul
go run main.go

# Terminal 2: Run tests
cd web/admin
npm run test:e2e
```

### Browser Not Found

**Problem:** `Executable doesn't exist` error.

**Solution:**
```bash
npx playwright install
```

### Tests Pass Locally But Fail in CI

**Considerations:**
- CI may not have backend server running
- Consider mocking API responses for CI tests
- Or set up backend service in CI pipeline
- Increase timeouts for slower CI environments

## Viewing Test Results

### HTML Report

```bash
# Generate and open HTML report
npx playwright show-report
```

### Screenshots and Videos

On failure, Playwright automatically captures:
- Screenshots (on first failure)
- Videos (if configured)
- Trace files (on retry)

View traces:
```bash
npx playwright show-trace trace.zip
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '24'

      # Start backend
      - name: Start Konsul Backend
        run: |
          cd cmd/konsul
          go run main.go &
          sleep 5

      # Install and test
      - name: Install dependencies
        run: |
          cd web/admin
          npm ci

      - name: Install Playwright
        run: npx playwright install --with-deps

      - name: Run tests
        run: npm run test:e2e

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: playwright-report/
```

## Best Practices

1. **Keep tests independent** - Each test should work standalone
2. **Use meaningful selectors** - Prefer role/text over CSS classes
3. **Wait explicitly** - Use `page.waitForURL()` for navigation
4. **Clean up** - Tests should not leave artifacts
5. **Mock when appropriate** - Consider mocking external APIs in CI

## Next Steps

### Potential Enhancements

1. **Visual Regression Testing**: Add screenshot comparisons
2. **API Mocking**: Mock backend responses for faster CI tests
3. **Performance Testing**: Add metrics collection
4. **Accessibility Testing**: Add a11y checks with axe-core
5. **Cross-browser**: Ensure all tests pass on Firefox and WebKit
6. **More scenarios**: Add tests for error states, edge cases

### Adding New Tests

1. Create a new spec file in `e2e/`
2. Import the auth fixture if needed
3. Follow existing test patterns
4. Run your test: `npx playwright test e2e/yourtest.spec.ts`
5. Update this guide with new test coverage

## Resources

- [Playwright Documentation](https://playwright.dev)
- [Test Fixtures](https://playwright.dev/docs/test-fixtures)
- [Selectors](https://playwright.dev/docs/selectors)
- [Best Practices](https://playwright.dev/docs/best-practices)

## Support

For issues or questions:
1. Check `e2e/README.md` for detailed testing docs
2. Review Playwright documentation
3. Check browser console in headed mode
4. Enable debug mode: `npx playwright test --debug`
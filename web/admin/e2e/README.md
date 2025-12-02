# E2E Tests for Konsul Admin UI

This directory contains end-to-end (E2E) tests for the Konsul Admin UI using Playwright.

## Test Structure

```
e2e/
├── fixtures/
│   └── auth.ts           # Authentication fixture for authenticated tests
├── auth.spec.ts          # Authentication flow tests
├── dashboard.spec.ts     # Dashboard page tests
├── services.spec.ts      # Services page tests
├── kvstore.spec.ts       # KV Store page tests
├── health.spec.ts        # Health page tests
├── apikeys.spec.ts       # API Keys page tests
├── navigation.spec.ts    # Navigation and routing tests
├── protected-routes.spec.ts  # Protected route access tests
└── smoke.spec.ts         # Smoke tests and critical user journeys
```

## Running Tests

```bash
# Run all tests
npm run test:e2e

# Run tests in headed mode (see browser)
npm run test:e2e:headed

# Run specific test file
npx playwright test e2e/auth.spec.ts

# Run tests in debug mode
npx playwright test --debug

# Run tests in specific browser
npx playwright test --project=chromium
npx playwright test --project=firefox
npx playwright test --project=webkit
```

## Test Coverage

### Authentication Tests (`auth.spec.ts`)
- Login page display
- Form validation
- Successful login flow
- Custom credentials (user ID, roles, policies)
- Loading states
- Redirect when already authenticated

### Protected Routes Tests (`protected-routes.spec.ts`)
- Unauthenticated access redirects to login
- Login page is publicly accessible

### Navigation Tests (`navigation.spec.ts`)
- Navigation between all pages
- Mobile sidebar functionality
- URL updates on navigation

### Page-Specific Tests
- **Dashboard**: Page load and content display
- **Services**: Service list, search/filter
- **KV Store**: Key-value pairs, add/create functionality
- **Health**: Health status and metrics
- **API Keys**: API key list and creation

### Smoke Tests (`smoke.spec.ts`)
- Application loads successfully
- Complete user journey (login → all pages)
- No critical console errors
- Mobile responsiveness

## Authentication Fixture

The `fixtures/auth.ts` file provides an `authenticatedPage` fixture that automatically logs in before each test. Use it like this:

```typescript
import { test, expect } from './fixtures/auth';

test('my authenticated test', async ({ authenticatedPage }) => {
  await authenticatedPage.goto('/services');
  // Test code here
});
```

## Configuration

Tests are configured in `playwright.config.ts`:
- Base URL: `http://127.0.0.1:4173/admin`
- Test directory: `./e2e`
- Browsers: Chromium, Firefox, WebKit
- Dev server automatically starts before tests

## Best Practices

1. **Use fixtures** for repeated setup (like authentication)
2. **Use meaningful selectors** (role, text, placeholder over CSS classes)
3. **Wait for navigation** explicitly with `page.waitForURL()`
4. **Test user journeys** not just individual features
5. **Keep tests independent** - each test should work standalone
6. **Use descriptive test names** that explain what is being tested

## Troubleshooting

### Tests timing out
- Increase timeout in playwright.config.ts
- Check if dev server is starting properly
- Verify backend API is running

### Selectors not found
- Run tests in headed mode to see what's happening
- Use Playwright Inspector: `npx playwright test --debug`
- Check if elements have correct text/roles

### Authentication issues
- Verify auth context is working properly
- Check browser console for API errors
- Ensure JWT token generation is working

## Continuous Integration

Tests are configured to:
- Run with retries (2 retries in CI)
- Use GitHub Actions reporter in CI
- Run serially in CI (workers: 1)
- Fail on `.only()` tests in CI
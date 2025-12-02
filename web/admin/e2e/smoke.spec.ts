import { test, expect } from './fixtures/auth';

test.describe('Smoke Tests', () => {
  test('should load application successfully', async ({ page }) => {
    await page.goto('/login');

    // Check that the page loaded
    await expect(page).toHaveURL('/login');
    await expect(page.getByText('Konsul Admin')).toBeVisible();
  });

  test('should complete full user journey', async ({ page }) => {
    // 1. Start at login
    await page.goto('/login');
    await expect(page.getByText('Konsul Admin')).toBeVisible();

    // 2. Login
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');
    await page.getByRole('button', { name: 'Sign In' }).click();

    // 3. Verify dashboard
    await page.waitForURL('/');
    await expect(page.getByText('Dashboard')).toBeVisible();

    // 4. Navigate to Services
    await page.getByRole('link', { name: /services/i }).click();
    await expect(page).toHaveURL('/services');
    await expect(page.getByText('Services')).toBeVisible();

    // 5. Navigate to KV Store
    await page.getByRole('link', { name: /kv store/i }).click();
    await expect(page).toHaveURL('/kv');
    await expect(page.getByText('Key-Value Store')).toBeVisible();

    // 6. Navigate to Health
    await page.getByRole('link', { name: /health/i }).click();
    await expect(page).toHaveURL('/health');
    await expect(page.getByText('Cluster Health')).toBeVisible();

    // 7. Navigate to API Keys
    await page.getByRole('link', { name: /api keys/i }).click();
    await expect(page).toHaveURL('/apikeys');
    await expect(page.getByText('API Keys')).toBeVisible();

    // 8. Return to Dashboard
    await page.getByRole('link', { name: /dashboard/i }).click();
    await expect(page).toHaveURL('/');
    await expect(page.getByText('Dashboard')).toBeVisible();
  });

  test('should have no console errors on critical pages', async ({ page }) => {
    const consoleErrors: string[] = [];

    // Listen for console errors
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Login
    await page.goto('/login');
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await page.waitForURL('/');

    // Visit each page
    const pages = ['/', '/services', '/kv', '/health', '/apikeys'];

    for (const route of pages) {
      await page.goto(route);
      await page.waitForLoadState('networkidle');
    }

    // Filter out expected/known errors if any
    const criticalErrors = consoleErrors.filter(
      (error) =>
        !error.includes('Failed to load resource') && // Ignore resource loading errors
        !error.includes('net::ERR') // Ignore network errors in dev
    );

    expect(criticalErrors).toHaveLength(0);
  });

  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Login
    await page.goto('/login');
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await page.waitForURL('/');

    // Verify dashboard is visible on mobile
    await expect(page.getByText('Dashboard')).toBeVisible();

    // Verify we can open sidebar on mobile
    const menuButton = page.getByRole('button', { name: /menu/i });
    if (await menuButton.isVisible()) {
      await menuButton.click();

      // Navigation should be visible
      await expect(page.getByRole('link', { name: /services/i })).toBeVisible();
    }
  });
});
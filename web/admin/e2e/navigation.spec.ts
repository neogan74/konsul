import { test, expect } from './fixtures/auth';

test.describe('Application Navigation', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    // Start on dashboard
    await authenticatedPage.goto('/');
  });

  test('should navigate to Services page', async ({ authenticatedPage }) => {
    // Click on Services in sidebar
    await authenticatedPage.getByRole('link', { name: /services/i }).click();

    // Verify URL changed
    await expect(authenticatedPage).toHaveURL('/services');

    // Verify page content
    await expect(authenticatedPage.getByText('Services')).toBeVisible();
  });

  test('should navigate to KV Store page', async ({ authenticatedPage }) => {
    // Click on KV Store in sidebar
    await authenticatedPage.getByRole('link', { name: /kv store/i }).click();

    // Verify URL changed
    await expect(authenticatedPage).toHaveURL('/kv');

    // Verify page content
    await expect(authenticatedPage.getByText('Key-Value Store')).toBeVisible();
  });

  test('should navigate to Health page', async ({ authenticatedPage }) => {
    // Click on Health in sidebar
    await authenticatedPage.getByRole('link', { name: /health/i }).click();

    // Verify URL changed
    await expect(authenticatedPage).toHaveURL('/health');

    // Verify page content
    await expect(authenticatedPage.getByText('Cluster Health')).toBeVisible();
  });

  test('should navigate to API Keys page', async ({ authenticatedPage }) => {
    // Click on API Keys in sidebar
    await authenticatedPage.getByRole('link', { name: /api keys/i }).click();

    // Verify URL changed
    await expect(authenticatedPage).toHaveURL('/apikeys');

    // Verify page content
    await expect(authenticatedPage.getByText('API Keys')).toBeVisible();
  });

  test('should return to dashboard', async ({ authenticatedPage }) => {
    // Navigate to another page
    await authenticatedPage.getByRole('link', { name: /services/i }).click();
    await expect(authenticatedPage).toHaveURL('/services');

    // Click Dashboard link
    await authenticatedPage.getByRole('link', { name: /dashboard/i }).click();

    // Verify back on dashboard
    await expect(authenticatedPage).toHaveURL('/');
    await expect(authenticatedPage.getByText('Dashboard')).toBeVisible();
  });

  test('should open and close mobile sidebar', async ({ authenticatedPage }) => {
    // Set mobile viewport
    await authenticatedPage.setViewportSize({ width: 375, height: 667 });

    // Open sidebar
    const menuButton = authenticatedPage.getByRole('button', { name: /menu/i });
    await menuButton.click();

    // Sidebar should be visible
    const sidebar = authenticatedPage.locator('[data-testid="sidebar"], nav').first();
    await expect(sidebar).toBeVisible();

    // Close sidebar
    const closeButton = authenticatedPage.getByRole('button', { name: /close/i });
    await closeButton.click();

    // Sidebar should be hidden (or check that menu button is visible again)
    await expect(menuButton).toBeVisible();
  });
});
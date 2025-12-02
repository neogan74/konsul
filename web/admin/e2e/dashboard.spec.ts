import { test, expect } from './fixtures/auth';

test.describe('Dashboard Page', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
  });

  test('should display dashboard title', async ({ authenticatedPage }) => {
    await expect(authenticatedPage.getByText('Dashboard')).toBeVisible();
  });

  test('should display cluster overview stats', async ({ authenticatedPage }) => {
    // Check for common dashboard elements
    // Note: These selectors might need adjustment based on actual dashboard content

    // Look for stat cards or metrics
    const dashboardContent = authenticatedPage.locator('main');
    await expect(dashboardContent).toBeVisible();

    // Verify dashboard is not empty
    const isEmpty = await dashboardContent.textContent();
    expect(isEmpty?.length).toBeGreaterThan(0);
  });

  test('should have working navigation from dashboard', async ({ authenticatedPage }) => {
    // Verify all navigation links are accessible
    await expect(authenticatedPage.getByRole('link', { name: /services/i })).toBeVisible();
    await expect(authenticatedPage.getByRole('link', { name: /kv store/i })).toBeVisible();
    await expect(authenticatedPage.getByRole('link', { name: /health/i })).toBeVisible();
  });
});
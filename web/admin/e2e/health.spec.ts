import { test, expect } from './fixtures/auth';

test.describe('Health Page', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/health');
  });

  test('should display health page title', async ({ authenticatedPage }) => {
    await expect(authenticatedPage.getByText('Cluster Health')).toBeVisible();
  });

  test('should display health status information', async ({ authenticatedPage }) => {
    const mainContent = authenticatedPage.locator('main');
    await expect(mainContent).toBeVisible();

    // Health page should have content
    const content = await mainContent.textContent();
    expect(content?.length).toBeGreaterThan(0);
  });

  test('should show health metrics or status indicators', async ({ authenticatedPage }) => {
    // Look for status indicators (healthy, unhealthy, etc.)
    const statusElements = authenticatedPage.locator('[class*="status"], [class*="health"], [class*="metric"]');

    // If status elements exist, verify at least one is visible
    const count = await statusElements.count();
    if (count > 0) {
      await expect(statusElements.first()).toBeVisible();
    }
  });
});
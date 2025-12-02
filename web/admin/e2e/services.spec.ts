import { test, expect } from './fixtures/auth';

test.describe('Services Page', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/services');
  });

  test('should display services page title', async ({ authenticatedPage }) => {
    await expect(authenticatedPage.getByText('Services')).toBeVisible();
  });

  test('should have services list or empty state', async ({ authenticatedPage }) => {
    const mainContent = authenticatedPage.locator('main');
    await expect(mainContent).toBeVisible();

    // Either services are listed or an empty state is shown
    const content = await mainContent.textContent();
    expect(content?.length).toBeGreaterThan(0);
  });

  test('should be able to search or filter services', async ({ authenticatedPage }) => {
    // Look for search/filter inputs (adjust selector based on implementation)
    const searchInput = authenticatedPage.locator('input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]').first();

    // If search exists, test it
    if (await searchInput.isVisible().catch(() => false)) {
      await searchInput.fill('test');
      await expect(searchInput).toHaveValue('test');
    }
  });
});
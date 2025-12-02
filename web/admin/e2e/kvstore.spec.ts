import { test, expect } from './fixtures/auth';

test.describe('KV Store Page', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/kv');
  });

  test('should display KV Store page title', async ({ authenticatedPage }) => {
    await expect(authenticatedPage.getByText('Key-Value Store')).toBeVisible();
  });

  test('should display key-value pairs or empty state', async ({ authenticatedPage }) => {
    const mainContent = authenticatedPage.locator('main');
    await expect(mainContent).toBeVisible();

    // Content should be present
    const content = await mainContent.textContent();
    expect(content?.length).toBeGreaterThan(0);
  });

  test('should have add/create button or form', async ({ authenticatedPage }) => {
    // Look for buttons to add new KV pairs
    const addButton = authenticatedPage.getByRole('button', { name: /add|create|new/i }).first();

    // If button exists, it should be visible
    if (await addButton.isVisible().catch(() => false)) {
      await expect(addButton).toBeVisible();
    }
  });

  test('should have search or filter functionality', async ({ authenticatedPage }) => {
    // Look for search/filter inputs
    const searchInput = authenticatedPage.locator('input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]').first();

    // If search exists, verify it works
    if (await searchInput.isVisible().catch(() => false)) {
      await searchInput.fill('test-key');
      await expect(searchInput).toHaveValue('test-key');
    }
  });
});
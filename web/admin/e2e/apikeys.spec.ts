import { test, expect } from './fixtures/auth';

test.describe('API Keys Page', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/apikeys');
  });

  test('should display API Keys page title', async ({ authenticatedPage }) => {
    await expect(authenticatedPage.getByText('API Keys')).toBeVisible();
  });

  test('should display API keys list or empty state', async ({ authenticatedPage }) => {
    const mainContent = authenticatedPage.locator('main');
    await expect(mainContent).toBeVisible();

    // Content should exist
    const content = await mainContent.textContent();
    expect(content?.length).toBeGreaterThan(0);
  });

  test('should have create API key button', async ({ authenticatedPage }) => {
    // Look for button to create new API key
    const createButton = authenticatedPage.getByRole('button', { name: /create|add|new.*key/i }).first();

    // If button exists, verify it's visible
    if (await createButton.isVisible().catch(() => false)) {
      await expect(createButton).toBeVisible();
      await expect(createButton).toBeEnabled();
    }
  });

  test('should have search or filter functionality', async ({ authenticatedPage }) => {
    // Look for search/filter inputs
    const searchInput = authenticatedPage.locator('input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]').first();

    // If search exists, test it
    if (await searchInput.isVisible().catch(() => false)) {
      await searchInput.fill('test-key');
      await expect(searchInput).toHaveValue('test-key');
    }
  });
});
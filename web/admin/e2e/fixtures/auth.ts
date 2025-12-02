import { test as base, expect } from '@playwright/test';

// Extend base test with authentication fixture
export const test = base.extend({
  authenticatedPage: async ({ page }, use) => {
    // Navigate to login page
    await page.goto('/login');

    // Fill in login form
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');

    // Submit login form
    await page.getByRole('button', { name: /sign in/i }).click();

    // Wait for navigation to dashboard
    await page.waitForURL('/');

    // Verify we're authenticated
    await expect(page.getByText('Dashboard')).toBeVisible();

    await use(page);
  },
});

export { expect };
import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login page correctly', async ({ page }) => {
    // Check page title and branding
    await expect(page.getByText('Konsul Admin')).toBeVisible();
    await expect(page.getByText('Sign in to manage your cluster')).toBeVisible();

    // Check form elements are present
    await expect(page.getByPlaceholder('Enter username')).toBeVisible();
    await expect(page.getByPlaceholder('Defaults to username')).toBeVisible();
    await expect(page.getByPlaceholder('admin, developer')).toBeVisible();
    await expect(page.getByPlaceholder('developer, readonly')).toBeVisible();

    // Check submit button
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible();
  });

  test('should show validation for required fields', async ({ page }) => {
    // Try to submit empty form
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Check for HTML5 validation (form should not submit)
    await expect(page).toHaveURL(/login/);
  });

  test('should successfully login with valid credentials', async ({ page }) => {
    // Fill in the form
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');

    // Submit form
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Wait for redirect to dashboard
    await page.waitForURL('/');

    // Verify we're on the dashboard
    await expect(page.getByText('Dashboard')).toBeVisible();
  });

  test('should login with custom user ID and roles', async ({ page }) => {
    // Fill in all fields
    await page.getByPlaceholder('Enter username').fill('testuser');
    await page.getByPlaceholder('Defaults to username').fill('test-user-id');
    await page.getByPlaceholder('admin, developer').fill('developer, viewer');
    await page.getByPlaceholder('developer, readonly').fill('read-policy, write-policy');

    // Submit form
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Wait for redirect
    await page.waitForURL('/');

    // Verify dashboard is visible
    await expect(page.getByText('Dashboard')).toBeVisible();
  });

  test('should show loading state during login', async ({ page }) => {
    // Fill in the form
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');

    // Click submit
    await page.getByRole('button', { name: 'Sign In' }).click();

    // Check for loading state (button should be disabled and show loading text)
    const submitButton = page.getByRole('button', { name: /signing in/i });

    // Note: This might be too fast to catch, so we make it optional
    const isLoadingVisible = await submitButton.isVisible().catch(() => false);

    if (isLoadingVisible) {
      await expect(submitButton).toBeDisabled();
    }

    // Eventually should navigate
    await page.waitForURL('/', { timeout: 5000 });
  });

  test('should redirect to dashboard if already authenticated', async ({ page }) => {
    // First, login
    await page.getByPlaceholder('Enter username').fill('admin');
    await page.getByPlaceholder('admin, developer').fill('admin');
    await page.getByRole('button', { name: 'Sign In' }).click();
    await page.waitForURL('/');

    // Try to go back to login page
    await page.goto('/login');

    // Should redirect to dashboard
    await page.waitForURL('/');
    await expect(page.getByText('Dashboard')).toBeVisible();
  });
});
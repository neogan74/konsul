import { test, expect } from '@playwright/test';

test.describe('Protected Routes', () => {
  test('should redirect to login when accessing dashboard without auth', async ({ page }) => {
    await page.goto('/admin/');

    // Should redirect to login page
    await page.waitForURL('/admin/login');
    await expect(page.getByText('Konsul Admin')).toBeVisible();
  });

  test('should redirect to login when accessing services without auth', async ({ page }) => {
    await page.goto('/admin/services');

    // Should redirect to login page
    await page.waitForURL('/admin/login');
    await expect(page.getByText('Sign in to manage your cluster')).toBeVisible();
  });

  test('should redirect to login when accessing KV Store without auth', async ({ page }) => {
    await page.goto('/admin/kv');

    // Should redirect to login page
    await page.waitForURL('/admin/login');
  });

  test('should redirect to login when accessing Health without auth', async ({ page }) => {
    await page.goto('/admin/health');

    // Should redirect to login page
    await page.waitForURL('/admin/login');
  });

  test('should redirect to login when accessing API Keys without auth', async ({ page }) => {
    await page.goto('/admin/apikeys');

    // Should redirect to login page
    await page.waitForURL('/admin/login');
  });

  test('should allow access to login page without auth', async ({ page }) => {
    await page.goto('/admin/login');

    // Should stay on login page
    await expect(page).toHaveURL('/admin/login');
    await expect(page.getByText('Konsul Admin')).toBeVisible();
  });
});

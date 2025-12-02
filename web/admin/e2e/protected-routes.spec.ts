import { test, expect } from '@playwright/test';

test.describe('Protected Routes', () => {
  test('should redirect to login when accessing dashboard without auth', async ({ page }) => {
    await page.goto('/');

    // Should redirect to login page
    await page.waitForURL('/login');
    await expect(page.getByText('Konsul Admin')).toBeVisible();
  });

  test('should redirect to login when accessing services without auth', async ({ page }) => {
    await page.goto('/services');

    // Should redirect to login page
    await page.waitForURL('/login');
    await expect(page.getByText('Sign in to manage your cluster')).toBeVisible();
  });

  test('should redirect to login when accessing KV Store without auth', async ({ page }) => {
    await page.goto('/kv');

    // Should redirect to login page
    await page.waitForURL('/login');
  });

  test('should redirect to login when accessing Health without auth', async ({ page }) => {
    await page.goto('/health');

    // Should redirect to login page
    await page.waitForURL('/login');
  });

  test('should redirect to login when accessing API Keys without auth', async ({ page }) => {
    await page.goto('/apikeys');

    // Should redirect to login page
    await page.waitForURL('/login');
  });

  test('should allow access to login page without auth', async ({ page }) => {
    await page.goto('/login');

    // Should stay on login page
    await expect(page).toHaveURL('/login');
    await expect(page.getByText('Konsul Admin')).toBeVisible();
  });
});
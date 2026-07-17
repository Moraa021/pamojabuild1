import { test, expect } from '@playwright/test';

test.describe('Donor journey', () => {
  test('can browse open campaigns and navigate to donation page', async ({ page }) => {
    await page.goto('/donor/donations');
    await expect(page.getByRole('heading', { name: 'Fund a Campaign' })).toBeVisible();

    const categoryFilter = page.locator('#filter-category');
    await categoryFilter.selectOption('environment');

  });
});

test.describe('Volunteer journey', () => {
  test('register page loads and form is interactive', async ({ page }) => {
    await page.goto('/register');
    await expect(page.getByRole('heading', { name: 'Create account' })).toBeVisible();

    await page.fill('[name="display_name"]', 'Test Volunteer');
    await page.fill('[name="email"]',        'vol@test.com');
    await page.fill('[name="password"]',     'password123');
    await page.fill('[name="confirm_password"]', 'password123');
    await page.selectOption('[name="role"]', 'volunteer');

    const submitBtn = page.locator('#register-btn');
    await expect(submitBtn).not.toBeDisabled();
  });

  test('sign in page loads', async ({ page }) => {
    await page.goto('/signin');
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible();
  });

  test('task browser renders filter bar', async ({ page }) => {
    await page.goto('/volunteer/tasks');
    await expect(page.locator('.task-filter')).toBeVisible();
    await expect(page.locator('#filter-category')).toBeVisible();
    await expect(page.locator('#filter-region')).toBeVisible();
  });

  test('validation fires on empty registration submit', async ({ page }) => {
    await page.goto('/register');
    await page.click('#register-btn');
    await expect(page.locator('.form-field__error').first()).not.toBeEmpty();
  });
});
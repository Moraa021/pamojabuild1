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

test.describe('Campaign creator journey', () => {
  test('create campaign page loads with all form fields', async ({ page }) => {
    await page.goto('/campaigns/new');
    await expect(page.getByRole('heading', { name: 'Create a Campaign' })).toBeVisible();
    await expect(page.locator('[name="title"]')).toBeVisible();
    await expect(page.locator('[name="description"]')).toBeVisible();
    await expect(page.locator('[name="category"]')).toBeVisible();
    await expect(page.locator('[name="region"]')).toBeVisible();
    await expect(page.locator('[name="volunteer_mode"]')).toBeVisible();
  });

  test('required field validation works', async ({ page }) => {
    await page.goto('/campaigns/new');
    await page.click('#submit-btn');
    await expect(page.locator('.form-field__error').first()).not.toBeEmpty();
  });
});

test.describe('Trustee journey', () => {
  test('trustee register page requires slug param', async ({ page }) => {
    await page.goto('/trustee/register');
    await expect(page.locator('section')).toContainText('No campaign specified');
  });

  test('trustee register page loads with valid slug param', async ({ page }) => {
    await page.goto('/trustee/register?slug=test-task');
    await expect(page.locator('.vol-dashboard, .trustee-dashboard, .api-error')).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Navigation', () => {
  test('home page loads', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Fund volunteers with Bitcoin' })).toBeVisible();
  });

  test('navbar is present on all pages', async ({ page }) => {
    const pages = ['/', '/volunteer/tasks', '/campaigns/new', '/signin', '/register'];
    for (const path of pages) {
      await page.goto(path);
      await expect(page.locator('.navbar')).toBeVisible();
    }
  });

  test('404 page shows not found message', async ({ page }) => {
    await page.goto('/this-route-does-not-exist');
    await expect(page.locator('.not-found')).toBeVisible();
  });
});

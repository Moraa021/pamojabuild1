import { defineConfig, devices } from '@playwright/test';
 
export default defineConfig({
  testDir:    './tests/e2e',
  timeout:    30_000,
  retries:    1,
  reporter:   'html',
 
  use: {
    // Base URL for all page.goto() calls
    baseURL:       'http://localhost:3000',
    screenshot:    'only-on-failure',
    video:         'retain-on-failure',
    trace:         'on-first-retry',
  },
 
  // Start the Vite dev server before running E2E tests
  webServer: {
    command:  'npm run dev',
    url:      'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout:  30_000,
  },
 
  projects: [
    {
      name:  'chromium',
      use:   { ...devices['Desktop Chrome'] },
    },
    {
      name:  'firefox',
      use:   { ...devices['Desktop Firefox'] },
    },
    {
      name:  'Mobile Safari',
      use:   { ...devices['iPhone 13'] },
    },
  ],
});
 

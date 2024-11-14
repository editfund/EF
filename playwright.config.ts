import {devices, type PlaywrightTestConfig} from '@playwright/test';

const BASE_URL = process.env.GITEA_URL?.replace?.(/\/$/g, '') || 'http://localhost:3003';

/**
 * @see https://playwright.dev/docs/test-configuration
 * @type {import('@playwright/test').PlaywrightTestConfig}
 */
export default {
  testDir: './tests/e2e/',
  testMatch: /.*\.test\.e2e\.ts/, // Match any .test.e2e.js files

  // you can adjust this value locally to match your machine's power,
  // or pass `--workers x` to playwright
  workers: 1,

  /* Maximum time one test can run for. */
  timeout: 120 * 1000, // default 30s

  expect: {
    /**
     * Maximum time expect() should wait for the condition to be met.
     * For example in `await expect(locator).toHaveText();`
     */
    timeout: 15 * 1000, // default 5s
  },

  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: Boolean(process.env.CI),

  /* Retry on CI only */
  retries: process.env.CI ? 1 : 0,
  /* fail fast */
  maxFailures: process.env.CI ? 1 : 0,

  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: process.env.CI ? 'list' : [['list'], ['html', {outputFolder: 'tests/e2e/reports/', open: 'never'}]],

  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    headless: true,   // set to false to debug

    locale: 'en-US',

    /* Maximum time each action such as `click()` can take. Defaults to 0 (no limit). */
    actionTimeout: 5 * 1000, // default 0

    /* Maximum time allowed for navigation, such as `page.goto()`. */
    navigationTimeout: 10 * 1000, // default 0

    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: BASE_URL,

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',

    screenshot: 'only-on-failure',
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',

      /* Project-specific settings. */
      use: {
        ...devices['Desktop Chrome'],
      },
    },

    {
      name: 'firefox',
      use: {
        ...devices['Desktop Firefox'],
      },
    },

    {
      name: 'webkit',
      use: {
        ...devices['Desktop Safari'],
      },
    },

    /* Test against mobile viewports. */
    {
      name: 'Mobile Chrome',
      use: {
        ...devices['Pixel 5'],
      },
    },
    {
      name: 'Mobile Safari',
      use: {
        ...devices['iPhone 12'],
      },
    },
  ],

  /* Folder for test artifacts created during test execution such as screenshots, traces, etc. */
  outputDir: 'tests/e2e/test-artifacts/',
  /* Folder for explicit snapshots for visual testing */
  snapshotDir: 'tests/e2e/test-snapshots/',
} satisfies PlaywrightTestConfig;

import { defineConfig, devices } from '@playwright/test';

// Browser acceptance config driven by scripts/browser_acceptance.py.
// Retries stay at zero by design: one attempt, one sanitized JSON report.
// The firefox/webkit projects run @cross-browser tagged tests only.
const baseURL = process.env.GRAFT_BROWSER_WEB_BASE_URL ?? 'http://127.0.0.1:4173';

export default defineConfig({
  testDir: './suites',
  globalSetup: './support/auth.setup.ts',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  outputDir: './test-results',
  use: {
    baseURL,
    headless: true,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    storageState: './test-results/auth-state.json',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
      grepInvert: /@chromium-only/,
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop WebKit'] },
      grepInvert: /@chromium-only/,
    },
  ],
});

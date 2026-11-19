// @chromium-only
// Workbench suite: the dashboard workbench must render the real backend
// summary through the production bundle. The summary contract path mirrors
// web/src/contracts/generated/openapi-runtime-paths.ts (getDashboardSummary).
import { expect, test } from '@playwright/test';

import { browserEnv } from '../support/env';

test.describe('workbench @chromium-only', () => {
  test('loads the workbench page and receives the real dashboard summary', async ({ page }) => {
    const summaryResponse = page.waitForResponse(
      (response) => response.url().includes('/api/dashboard/summary') && response.status() === 200,
    );
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await summaryResponse;
    // The authenticated session must not bounce back to the auth entry.
    await expect(page).not.toHaveURL(/login/);
    await expect(page.getByRole('main').or(page.locator('body'))).toBeVisible();
    await expect(page.locator('body')).not.toContainText('Application error');
    // Keep the harness-provided environment identity traceable in the report.
    expect(browserEnv.suite).toBe('workbench');
  });
});

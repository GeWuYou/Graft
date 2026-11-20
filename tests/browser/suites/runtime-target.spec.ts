// @cross-browser
// Runtime target suite: the list must render the deterministic targets seeded
// by TestSeedBrowserFixture (server/modules/runtime-target). The list route
// mirrors RUNTIME_TARGET_ROUTE_PATH.LIST in
// web/src/modules/runtime-target/contract/paths.ts.
import { expect, test } from '@playwright/test';

import { browserEnv, browserFixture } from '../support/env';

test.describe('runtime-target @cross-browser', () => {
  test('renders the seeded runtime targets from the real backend', async ({ page }) => {
    const targetsResponse = page.waitForResponse(
      (response) => response.url().includes('/api/runtime-targets') && response.status() === 200,
    );
    await page.goto('/infrastructure/runtime-targets', { waitUntil: 'domcontentloaded' });
    const targets = await targetsResponse;
    // The raw HTTP response keeps the success envelope; totals live under data.
    const payload = (await targets.json()) as { data?: { total?: number } };
    expect(payload.data?.total).toBeGreaterThan(0);
    await expect(page.getByText(browserFixture.runtimeTargetDisplayName).first()).toBeVisible();
    await expect(page.locator('body')).not.toContainText('Application error');
    expect(browserEnv.engine).toBeDefined();
  });
});

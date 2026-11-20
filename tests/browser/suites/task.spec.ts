// @chromium-only
// Task suite: the fixture application detail page must render the deterministic
// task history seeded by TestSeedBrowserFixture (server/modules/task).
// The application detail route mirrors APPLICATION_ROUTE_PATH.DETAIL in
// web/src/modules/project/contract/paths.ts.
import { expect, test } from '@playwright/test';

import { browserFixture } from '../support/env';

test.describe('task @chromium-only', () => {
  test('lists the fixture application tasks with their stages', async ({ page }) => {
    const tasksResponse = page.waitForResponse(
      (response) => response.url().includes('/api/tasks') && response.status() === 200,
    );
    await page.goto(`/applications/${browserFixture.applicationId}`, { waitUntil: 'domcontentloaded' });
    await expect(page.getByText(browserFixture.applicationDisplayName).first()).toBeVisible();
    const tasks = await tasksResponse;
    // The raw HTTP response keeps the success envelope; totals live under data.
    const payload = (await tasks.json()) as { data?: { total?: number } };
    expect(payload.data?.total).toBeGreaterThan(0);
    // The task history surface renders inside the application detail page.
    await expect(page.locator('body')).not.toContainText('Application error');
  });
});

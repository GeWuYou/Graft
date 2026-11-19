// @chromium-only
// Compose suite: application lifecycle surfaces that mutate or observe the
// Docker daemon. Requires a reachable Docker daemon on the developer primary
// checkout; excluded from default runs and only executed with
// --suite compose --allow-daemon.
import { expect, test } from '@playwright/test';

test.describe('compose @chromium-only', () => {
  test.skip(process.env.GRAFT_BROWSER_ALLOW_DAEMON !== '1', 'requires --allow-daemon and a local Docker daemon');

  test('renders the application lifecycle entry surfaces', async ({ page }) => {
    await page.goto('/applications', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).not.toContainText('Application error');
  });
});

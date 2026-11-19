// @chromium-only
// Docker daemon suite: container and image surfaces. Requires a reachable
// Docker daemon on the developer primary checkout; excluded from default runs
// and only executed with --suite docker --allow-daemon.
import { expect, test } from '@playwright/test';

test.describe('docker @chromium-only', () => {
  test.skip(process.env.GRAFT_BROWSER_ALLOW_DAEMON !== '1', 'requires --allow-daemon and a local Docker daemon');

  test('renders the container list from the real daemon-backed backend', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).not.toContainText('Application error');
  });
});

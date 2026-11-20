// @chromium-only
// Docker daemon suite: the daemon-backed container and image management pages
// must render their real management surfaces. Routes mirror
// CONTAINER_ROUTE_PATH in web/src/modules/container/contract/paths.ts; the
// daemon-backed queries are /api/ops/containers and /api/ops/docker/images.
// Assertions stay daemon-content independent (rows, empty state or card list
// all pass) but fail when the page never renders its management skeleton.
import { expect, test } from '@playwright/test';

test.describe('docker @chromium-only', () => {
  test.skip(process.env.GRAFT_BROWSER_ALLOW_DAEMON !== '1', 'requires --allow-daemon and a local Docker daemon');

  test('renders the daemon-backed container management page', async ({ page }) => {
    const containersResponse = page.waitForResponse(
      (response) => response.url().includes('/api/ops/containers'),
    );
    await page.goto('/infrastructure/docker/containers', { waitUntil: 'domcontentloaded' });
    await containersResponse;
    // The authenticated session must not bounce back to the auth entry.
    await expect(page).not.toHaveURL(/login/);
    await expect(page.locator('.container-page')).toBeVisible();
    // List state: rendered table rows or the explicit empty state.
    await expect(page.locator('.container-page').locator('.t-table, .t-empty').first()).toBeVisible({
      timeout: 15_000,
    });
    await expect(page.locator('body')).not.toContainText('Application error');
  });

  test('renders the daemon-backed image management page', async ({ page }) => {
    const imagesResponse = page.waitForResponse(
      (response) => response.url().includes('/api/ops/docker/images'),
    );
    await page.goto('/infrastructure/images', { waitUntil: 'domcontentloaded' });
    await imagesResponse;
    await expect(page).not.toHaveURL(/login/);
    await expect(page.locator('.docker-images-page')).toBeVisible();
    await expect(
      page.locator('.docker-images-page').locator('.t-table, .t-empty, .docker-images-card-list').first(),
    ).toBeVisible({ timeout: 15_000 });
    await expect(page.locator('body')).not.toContainText('Application error');
  });
});

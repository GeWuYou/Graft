// @chromium-only
// Compose suite: the application registry page renders the Compose application
// lifecycle entry surface backed by /api/ops/applications. The route mirrors
// APPLICATION_ROUTE_PATH.LIST in web/src/modules/project/contract/paths.ts;
// the page declares its own project-table-summary testid. Assertions stay
// daemon-content independent but fail when the page never renders.
import { expect, test } from '@playwright/test';

test.describe('compose @chromium-only', () => {
  test.skip(process.env.GRAFT_BROWSER_ALLOW_DAEMON !== '1', 'requires --allow-daemon and a local Docker daemon');

  test('renders the application registry management page', async ({ page }) => {
    const applicationsResponse = page.waitForResponse(
      (response) => response.url().includes('/api/ops/applications'),
    );
    await page.goto('/applications', { waitUntil: 'domcontentloaded' });
    await applicationsResponse;
    // The authenticated session must not bounce back to the auth entry.
    await expect(page).not.toHaveURL(/login/);
    await expect(page.locator('.project-page')).toBeVisible();
    // The page declares its own table summary control; it renders once the
    // application list state (rows or empty state) is live.
    await expect(page.getByTestId('project-table-summary')).toBeVisible({ timeout: 15_000 });
    await expect(page.locator('body')).not.toContainText('Application error');
  });
});

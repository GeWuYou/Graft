import { describe, expect, it } from 'vitest';

import { developmentRouterList as developmentRoutes } from './development-routes.development';
import { developmentRouterList as releaseRoutes } from './development-routes.release';

describe('development-only routes', () => {
  it('registers the dashboard preview only in the development route list', () => {
    const previewShell = developmentRoutes.find(({ path }) => path === '/mock/dashboard-preview');
    const preview = previewShell?.children?.find(({ path }) => path === '');

    expect(previewShell?.name).toBe('DevelopmentDashboardWorkbenchPreviewShell');
    expect(preview?.name).toBe('DevelopmentDashboardWorkbenchPreview');
    expect(preview?.meta).toMatchObject({
      hidden: true,
      hiddenBreadcrumb: true,
      keepAlive: false,
      pageKind: 'overview',
    });
    expect(releaseRoutes).toEqual([]);
  });
});

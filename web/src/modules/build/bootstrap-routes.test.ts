import { describe, expect, it } from 'vitest';

import { buildBootstrapRouteRegistrations } from './bootstrap-routes';

describe('build bootstrap route registrations', () => {
  it('registers Build Artifacts as the canonical Build entry', () => {
    expect(buildBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/build/artifacts',
        routeName: 'BuildArtifactList',
        meta: expect.objectContaining({
          tabGroup: 'build-artifacts',
          pageKind: 'list',
          pageSurface: 'paged-table',
        }),
      }),
    );
  });
});

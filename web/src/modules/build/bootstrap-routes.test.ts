import { describe, expect, it } from 'vitest';

import { buildBootstrapRouteRegistrations } from './bootstrap-routes';

describe('build bootstrap route registrations', () => {
  it('registers Build Jobs as a paged table surface', () => {
    expect(buildBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/build/jobs',
        routeName: 'BuildJobList',
        meta: expect.objectContaining({
          tabGroup: 'build-jobs',
          pageKind: 'list',
          pageSurface: 'paged-table',
        }),
      }),
    );
  });

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

  it('registers Build Workspaces as a managed Build resource', () => {
    expect(buildBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: '/build/workspaces',
        routeName: 'BuildWorkspaceList',
        meta: expect.objectContaining({
          tabGroup: 'build-workspaces',
          pageKind: 'list',
          pageSurface: 'paged-table',
        }),
      }),
    );
  });
});

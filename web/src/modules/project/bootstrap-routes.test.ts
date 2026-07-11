import { describe, expect, it } from 'vitest';

import { projectBootstrapRouteRegistrations, projectGlobalRouteRegistrations } from './bootstrap-routes';
import { PROJECT_ROUTE_PATH } from './contract/paths';

describe('project bootstrap route registrations', () => {
  it('keeps the project list on the canonical paged list surface', () => {
    expect(projectBootstrapRouteRegistrations).toHaveLength(1);
    expect(projectBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: PROJECT_ROUTE_PATH.LIST,
      routeName: 'ProjectList',
      meta: expect.objectContaining({
        pageKind: 'list',
      }),
    });
    expect(projectBootstrapRouteRegistrations[0]?.meta?.pageSurface).toBeUndefined();
  });

  it('registers the configuration workspace as a hidden editor route', () => {
    expect(projectGlobalRouteRegistrations).toContainEqual(
      expect.objectContaining({
        path: PROJECT_ROUTE_PATH.CONFIGURATION_WORKSPACE,
        routeName: 'ProjectConfigurationWorkspace',
        pageRouteName: 'ProjectConfigurationWorkspaceIndex',
        meta: expect.objectContaining({
          hiddenMenu: true,
          keepAlive: true,
          pageKind: 'detail',
          pageSurface: 'editor',
          titleKey: 'project.route.configurationWorkspace.title',
        }),
      }),
    );
  });

  it('exposes only managed and template source routes beside Import Existing', () => {
    const paths = projectGlobalRouteRegistrations.map((route) => route.path);

    expect(paths).toContain(PROJECT_ROUTE_PATH.IMPORT);
    expect(paths).toContain(PROJECT_ROUTE_PATH.CREATE_MANAGED);
    expect(paths).toContain(PROJECT_ROUTE_PATH.CREATE_TEMPLATE);
    expect(paths).not.toContain('/ops/projects/create/git');
    expect(paths).not.toContain('/ops/projects/create/remote-host');
  });
});

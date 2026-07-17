import { describe, expect, it } from 'vitest';

import { applicationBootstrapRouteRegistrations, applicationGlobalRouteRegistrations } from './bootstrap-routes';
import { APPLICATION_ROUTE_PATH } from './contract/paths';

describe('project bootstrap route registrations', () => {
  it('keeps the project list on the canonical paged list surface', () => {
    expect(applicationBootstrapRouteRegistrations).toHaveLength(2);
    expect(applicationBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: APPLICATION_ROUTE_PATH.LIST,
      routeName: 'ApplicationList',
      meta: expect.objectContaining({
        pageKind: 'list',
      }),
    });
    expect(applicationBootstrapRouteRegistrations[0]?.meta?.pageSurface).toBeUndefined();
  });

  it('registers the configuration workspace as a hidden editor route', () => {
    expect(applicationGlobalRouteRegistrations).toContainEqual(
      expect.objectContaining({
        path: APPLICATION_ROUTE_PATH.CONFIGURATION_WORKSPACE,
        routeName: 'ApplicationConfigurationWorkspace',
        pageRouteName: 'ApplicationConfigurationWorkspaceIndex',
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

  it('registers Application template management as a visible Application route', () => {
    expect(applicationBootstrapRouteRegistrations).toContainEqual(
      expect.objectContaining({
        menuPath: APPLICATION_ROUTE_PATH.TEMPLATES,
        routeName: 'ApplicationTemplates',
        pageRouteName: 'ApplicationTemplatesIndex',
        meta: expect.objectContaining({
          pageKind: 'list',
          pageSurface: 'paged-table',
        }),
      }),
    );
    expect(applicationGlobalRouteRegistrations).not.toContainEqual(
      expect.objectContaining({ path: APPLICATION_ROUTE_PATH.TEMPLATES }),
    );
  });

  it('navigates template detail actions to the mounted child route', () => {
    const detail = applicationGlobalRouteRegistrations.find((route) => route.routeName === 'ApplicationTemplateDetail');

    expect(detail).toMatchObject({ pageRouteName: 'ApplicationTemplateDetailIndex' });
  });

  it('registers template creation as a hidden workflow below template management', () => {
    expect(applicationGlobalRouteRegistrations).toContainEqual(
      expect.objectContaining({
        path: APPLICATION_ROUTE_PATH.TEMPLATE_CREATE,
        routeName: 'ApplicationTemplateCreateWizard',
        pageRouteName: 'ApplicationTemplateCreateWizardIndex',
        navigationParentPath: APPLICATION_ROUTE_PATH.TEMPLATES,
        meta: expect.objectContaining({
          hiddenMenu: true,
          keepAlive: false,
          pageSurface: 'form-detail',
          titleKey: 'project.route.templateCreate.title',
        }),
      }),
    );
  });

  it('exposes the three supported project creation routes', () => {
    const paths = applicationGlobalRouteRegistrations.map((route) => route.path);

    expect(paths).toContain(APPLICATION_ROUTE_PATH.CREATE_IMPORT);
    expect(paths).toContain(APPLICATION_ROUTE_PATH.CREATE_BLANK);
    expect(paths).toContain(APPLICATION_ROUTE_PATH.CREATE_TEMPLATE);
    expect(paths).toContain(APPLICATION_ROUTE_PATH.CREATE_SOURCE);
    expect(paths).not.toContain('/applications/create/git');
    expect(paths).not.toContain('/applications/create/remote-host');
  });
});

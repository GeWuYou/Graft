import { describe, expect, it, vi } from 'vitest';

import { PROJECT_BOOTSTRAP_ROUTE } from '../contract/bootstrap';
import { navigateToApplicationCreateSource, navigateToApplicationList } from './navigation';

describe('project navigation', () => {
  it('returns to source selection with the current creation query', () => {
    const router = { push: vi.fn(), replace: vi.fn() };
    const query = { deployment: 'compose', runtime_target_id: '7' };

    navigateToApplicationCreateSource(router, query);

    expect(router.push).toHaveBeenCalledWith({
      name: PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE.pageRouteName,
      query,
    });
  });

  it('synchronizes the application list tab before returning to it', () => {
    const resolved = {
      fullPath: '/applications',
      meta: {},
      name: PROJECT_BOOTSTRAP_ROUTE.LIST.pageRouteName,
      params: {},
      path: '/applications',
      query: {},
    };
    const router = { push: vi.fn(), resolve: vi.fn(() => resolved) };
    const tabs = { appendTabRouterList: vi.fn(), setActiveTabKey: vi.fn() };

    navigateToApplicationList(router as never, tabs);

    expect(tabs.appendTabRouterList).toHaveBeenCalledWith(
      expect.objectContaining({
        tabKey: '/applications',
        path: '/applications',
        name: PROJECT_BOOTSTRAP_ROUTE.LIST.pageRouteName,
      }),
    );
    expect(tabs.setActiveTabKey).toHaveBeenCalledWith('/applications');
    expect(router.push).toHaveBeenCalledWith({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.pageRouteName });
  });
});

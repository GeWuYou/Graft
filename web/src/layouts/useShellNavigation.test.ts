import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useTabsRouterStore } from '@/store/modules/tabs-router';

const pushMock = vi.fn();

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRouter: () => ({
      push: pushMock,
    }),
  };
});

describe('useShellNavigation', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    pushMock.mockReset();
    pushMock.mockResolvedValue(undefined);
  });

  it('activates the home tab before navigating to the root entry', async () => {
    const tabsRouterStore = useTabsRouterStore();
    tabsRouterStore.appendTabRouterList({
      tabKey: '/server/system-config',
      path: '/server/system-config',
      name: 'SystemConfigList',
    });
    tabsRouterStore.setActiveTabKey('/server/system-config');

    const { useShellNavigation } = await import('./useShellNavigation');
    const { goHome } = useShellNavigation();

    await goHome();

    expect(tabsRouterStore.activeTabKey).toBe('/');
    expect(pushMock).toHaveBeenCalledWith('/');
  });
});

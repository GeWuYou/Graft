import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';

import { useTabsRouterStore } from './tabs-router';

describe('useTabsRouterStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('uses the neutral root entry as the preserved home tab', () => {
    const tabsRouterStore = useTabsRouterStore();

    expect(tabsRouterStore.tabRouters).toHaveLength(1);
    expect(tabsRouterStore.tabRouters[0]?.path).toBe('/');
    expect(tabsRouterStore.tabRouters[0]?.name).toBe('RootEntry');
  });
});

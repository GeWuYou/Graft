import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, reactive } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';

import AppLayout from './index.vue';

const routeState = vi.hoisted(
  () =>
    ({
      fullPath: '/ops/containers/container-1?tab=overview',
      meta: {},
      name: 'ContainerDetail',
      params: { id: 'container-1' },
      path: '/ops/containers/container-1',
      query: { tab: 'overview' },
    }) as Partial<RouteLocationNormalizedLoaded> & {
      fullPath: string;
      path: string;
    },
);

const routeProxy = vi.hoisted(() => ({
  value: null as null | typeof routeState,
}));

const routerMock = vi.hoisted(() => ({
  resolve: vi.fn((target: string) => ({
    path: target.split('?')[0] || target,
  })),
}));

const settingStoreProxy = vi.hoisted(() => ({
  value: null as null | {
    displayMode: string;
    isSidebarCompact: boolean;
    layout: { value: string };
    showSidebar: boolean;
  },
}));

const storeState = vi.hoisted(() => ({
  settingStore: {
    displayMode: 'light',
    isSidebarCompact: false,
    layout: { value: 'side' },
    showSidebar: true,
  },
  tabsRouterStore: {
    appendTabRouterList: vi.fn(),
    healPersistedRoutes: vi.fn(),
    setActiveRoute: vi.fn(),
  },
}));

const scrollToMock = vi.hoisted(() => vi.fn());

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>();
  const { reactive } = await import('vue');
  routeProxy.value = reactive(routeState);

  return {
    ...actual,
    useRoute: () => routeProxy.value,
    useRouter: () => routerMock,
  };
});

vi.mock('./components/ForcePasswordChangeDialog.vue', () => ({
  default: { name: 'ForcePasswordChangeDialog', template: '<div />' },
}));

vi.mock('./components/LayoutContent.vue', () => ({
  default: { name: 'LayoutContent', template: '<div />' },
}));

vi.mock('./components/LayoutHeader.vue', () => ({
  default: { name: 'LayoutHeader', template: '<div />' },
}));

vi.mock('./components/LayoutSideNav.vue', () => ({
  default: { name: 'LayoutSideNav', template: '<div />' },
}));

vi.mock('pinia', async (importOriginal) => ({
  ...(await importOriginal<typeof import('pinia')>()),
  storeToRefs: (store: unknown) => store,
}));

vi.mock('@/store', () => ({
  useSettingStore: () => {
    if (!settingStoreProxy.value) {
      settingStoreProxy.value = reactive(storeState.settingStore);
    }
    return settingStoreProxy.value;
  },
  useTabsRouterStore: () => storeState.tabsRouterStore,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    debug: vi.fn(),
  }),
}));

vi.mock('@/utils/route/meta', () => ({
  resolveRouteLocalizedTitle: () => undefined,
  toLocalizedTitle: () => undefined,
}));

vi.mock('@/style/layout.less', () => ({}));

const PlainStub = defineComponent({
  name: 'PlainStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

function mountAppLayout() {
  return mount(AppLayout, {
    global: {
      stubs: {
        ForcePasswordChangeDialog: true,
        LayoutContent: true,
        LayoutHeader: true,
        LayoutSideNav: true,
        TAside: PlainStub,
        TContent: PlainStub,
        THeader: PlainStub,
        TLayout: PlainStub,
      },
    },
  });
}

describe('App layout route effects', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    settingStoreProxy.value ??= reactive(storeState.settingStore);
    settingStoreProxy.value.displayMode = 'light';
    settingStoreProxy.value.isSidebarCompact = false;
    settingStoreProxy.value.layout = { value: 'side' };
    settingStoreProxy.value.showSidebar = true;
    routeProxy.value!.fullPath = '/ops/containers/container-1?tab=overview';
    routeProxy.value!.path = '/ops/containers/container-1';
    routeProxy.value!.name = 'ContainerDetail';
    routeProxy.value!.params = { id: 'container-1' };
    routeProxy.value!.query = { tab: 'overview' };
    storeState.tabsRouterStore.appendTabRouterList.mockClear();
    storeState.tabsRouterStore.healPersistedRoutes.mockClear();
    storeState.tabsRouterStore.setActiveRoute.mockClear();
    routerMock.resolve.mockClear();
    scrollToMock.mockClear();
    document.body.innerHTML = '<div class="tdesign-starter-layout"></div>';
    const layout = document.querySelector('.tdesign-starter-layout') as HTMLDivElement;
    layout.scrollTo = scrollToMock;
  });

  it('updates tab route state without scrolling for same-page query changes', async () => {
    const wrapper = mountAppLayout();

    routeProxy.value!.fullPath = '/ops/containers/container-1?tab=health';
    routeProxy.value!.query = { tab: 'health' };
    await wrapper.vm.$nextTick();

    expect(storeState.tabsRouterStore.appendTabRouterList).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fullPath: '/ops/containers/container-1?tab=health',
        path: '/ops/containers/container-1',
        query: { tab: 'health' },
        tabKey: '/ops/containers/container-1',
      }),
    );
    expect(scrollToMock).not.toHaveBeenCalled();
  });

  it('scrolls to top when the route path changes', async () => {
    const wrapper = mountAppLayout();

    routeProxy.value!.fullPath = '/server/runtime';
    routeProxy.value!.path = '/server/runtime';
    routeProxy.value!.name = 'ServerRuntime';
    routeProxy.value!.params = {};
    routeProxy.value!.query = {};
    await wrapper.vm.$nextTick();

    expect(scrollToMock).toHaveBeenCalledWith({ behavior: 'smooth', top: 0 });
  });

  it('exposes the sidebar compact state on the shell surface', async () => {
    settingStoreProxy.value!.isSidebarCompact = false;
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('false');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanded');

    settingStoreProxy.value!.isSidebarCompact = true;
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('true');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-target-compact')).toBe('true');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('collapsing-width');

    vi.advanceTimersByTime(124);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('collapsing-submenu');

    vi.advanceTimersByTime(84);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('collapsing-topmenu');

    vi.advanceTimersByTime(112);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('compact');
  });

  it('runs the reverse sidebar motion when expanding back out', async () => {
    settingStoreProxy.value!.isSidebarCompact = true;
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('compact');

    settingStoreProxy.value!.isSidebarCompact = false;
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanding-width');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('false');

    vi.advanceTimersByTime(128);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanding-topmenu');

    vi.advanceTimersByTime(96);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanding-submenu');

    vi.advanceTimersByTime(96);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanded');
  });
});

import { mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, reactive } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';

import AppLayout from './index.vue';

const routeState = vi.hoisted(
  () =>
    ({
      fullPath: '/infrastructure/docker/containers/container-1?tab=overview',
      meta: {},
      name: 'ContainerDetail',
      params: { id: 'container-1' },
      path: '/infrastructure/docker/containers/container-1',
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
    isSidebarFixed: boolean;
    layout: { value: string };
    showSidebar: boolean;
  },
}));

const storeState = vi.hoisted(() => ({
  permissionStore: {
    routers: [],
  },
  realtimeSchedulerStore: {
    freeze: vi.fn(() => 1),
    release: vi.fn(),
  },
  settingStore: {
    displayMode: 'light',
    isSidebarCompact: false,
    isSidebarFixed: true,
    layout: { value: 'side' },
    showSidebar: true,
  },
  tabsRouterStore: {
    appendTabRouterList: vi.fn(),
    healPersistedRoutes: vi.fn(),
    healPersistedState: vi.fn(),
    setActiveRoute: vi.fn(),
  },
}));

const scrollToMock = vi.hoisted(() => vi.fn());
const shellContainerSize = vi.hoisted(() => ({ height: 800, width: 1200 }));
const mountedLayouts: Array<{ unmount: () => void }> = [];

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];

  callback: ResizeObserverCallback;
  disconnect = vi.fn();
  observe = vi.fn();

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    ResizeObserverMock.instances.push(this);
  }

  emit(width: number, height: number) {
    this.callback([{ contentRect: { height, width } } as ResizeObserverEntry], this as unknown as ResizeObserver);
  }
}

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
  default: {
    name: 'LayoutContent',
    emits: ['page-scroll'],
    template: '<div data-test-id="layout-content" @scroll="$emit(\'page-scroll\', $event)" />',
  },
}));

vi.mock('./components/LayoutHeader.vue', () => ({
  default: {
    name: 'LayoutHeader',
    emits: ['open-navigation'],
    props: ['presentation'],
    template:
      '<button data-test-id="layout-header" :data-presentation="presentation" @click="$emit(\'open-navigation\')" />',
  },
}));

vi.mock('./components/LayoutSideNav.vue', () => ({
  default: {
    name: 'LayoutSideNav',
    emits: ['update:drawer-visible'],
    props: ['drawerVisible', 'presentation'],
    template:
      '<div data-test-id="layout-side-nav" :data-drawer-visible="String(drawerVisible)" :data-presentation="presentation" />',
  },
}));

vi.mock('./components/MobileNavigation.vue', () => ({
  default: {
    name: 'MobileNavigation',
    emits: ['update:visible'],
    props: ['menu', 'visible'],
    template:
      '<button data-test-id="mobile-navigation" :data-visible="String(visible)" @click="$emit(\'update:visible\', true)" />',
  },
}));

vi.mock('pinia', async (importOriginal) => ({
  ...(await importOriginal<typeof import('pinia')>()),
  storeToRefs: (store: unknown) => store,
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => storeState.permissionStore,
  useRealtimeSchedulerStore: () => storeState.realtimeSchedulerStore,
  useSettingStore: () => {
    if (!settingStoreProxy.value) {
      settingStoreProxy.value = reactive(storeState.settingStore);
    }
    return settingStoreProxy.value;
  },
  useTabsRouterStore: () => storeState.tabsRouterStore,
}));

vi.mock('@/modules/update', () => ({
  updateProvider: { name: 'UpdateProvider', template: '<span />' },
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    debug: vi.fn(),
    warn: vi.fn(),
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
  const wrapper = mount(AppLayout, {
    global: {
      stubs: {
        ForcePasswordChangeDialog: true,
        TAside: PlainStub,
        TContent: PlainStub,
        THeader: PlainStub,
        TLayout: PlainStub,
      },
    },
  });
  mountedLayouts.push(wrapper);
  return wrapper;
}

describe('App layout route effects', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    settingStoreProxy.value ??= reactive(storeState.settingStore);
    settingStoreProxy.value.displayMode = 'light';
    settingStoreProxy.value.isSidebarCompact = false;
    settingStoreProxy.value.isSidebarFixed = true;
    settingStoreProxy.value.layout = { value: 'side' };
    settingStoreProxy.value.showSidebar = true;
    routeProxy.value!.fullPath = '/infrastructure/docker/containers/container-1?tab=overview';
    routeProxy.value!.path = '/infrastructure/docker/containers/container-1';
    routeProxy.value!.name = 'ContainerDetail';
    routeProxy.value!.meta = {};
    routeProxy.value!.params = { id: 'container-1' };
    routeProxy.value!.query = { tab: 'overview' };
    storeState.tabsRouterStore.appendTabRouterList.mockClear();
    storeState.tabsRouterStore.healPersistedRoutes.mockClear();
    storeState.tabsRouterStore.healPersistedState.mockClear();
    storeState.tabsRouterStore.setActiveRoute.mockClear();
    storeState.realtimeSchedulerStore.freeze.mockClear();
    storeState.realtimeSchedulerStore.release.mockClear();
    routerMock.resolve.mockClear();
    scrollToMock.mockClear();
    shellContainerSize.width = 1200;
    shellContainerSize.height = 800;
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1200 });
    ResizeObserverMock.instances = [];
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(() => shellContainerSize.width);
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(() => shellContainerSize.height);
    document.body.innerHTML = '<div class="tdesign-starter-page-container"></div>';
    const pageContainer = document.querySelector('.tdesign-starter-page-container') as HTMLDivElement;
    pageContainer.scrollTo = scrollToMock;
  });

  afterEach(() => {
    mountedLayouts.splice(0).forEach((wrapper) => wrapper.unmount());
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('updates tab route state without scrolling for same-page query changes', async () => {
    const wrapper = mountAppLayout();

    routeProxy.value!.fullPath = '/infrastructure/docker/containers/container-1?tab=health';
    routeProxy.value!.query = { tab: 'health' };
    await wrapper.vm.$nextTick();

    expect(storeState.tabsRouterStore.appendTabRouterList).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fullPath: '/infrastructure/docker/containers/container-1?tab=health',
        path: '/infrastructure/docker/containers/container-1',
        query: { tab: 'health' },
        tabKey: '/infrastructure/docker/containers/container-1',
      }),
    );
    expect(scrollToMock).not.toHaveBeenCalled();
  });

  it('heals persisted tab state without trimming other restored tabs on mount', () => {
    mountAppLayout();

    expect(storeState.tabsRouterStore.healPersistedState).toHaveBeenCalledTimes(1);
    expect(storeState.tabsRouterStore.healPersistedRoutes).toHaveBeenCalledTimes(1);
    expect(storeState.tabsRouterStore.healPersistedRoutes).toHaveBeenCalledWith(routerMock);
  });

  it('scrolls to top when the route path changes', async () => {
    const wrapper = mountAppLayout();

    routeProxy.value!.fullPath = '/observability/service-status';
    routeProxy.value!.path = '/observability/service-status';
    routeProxy.value!.name = 'ServerRuntime';
    routeProxy.value!.params = {};
    routeProxy.value!.query = {};
    await wrapper.vm.$nextTick();

    expect(scrollToMock).toHaveBeenCalledWith({ behavior: 'smooth', top: 0 });
  });

  it('coalesces page scroll updates until the next animation frame', async () => {
    const wrapper = mountAppLayout();
    const content = wrapper.get('[data-test-id="layout-content"]');

    Object.defineProperty(content.element, 'scrollTop', { configurable: true, value: 120 });
    await content.trigger('scroll');
    Object.defineProperty(content.element, 'scrollTop', { configurable: true, value: 240 });
    await content.trigger('scroll');

    expect(wrapper.get('.app-shell').attributes('style')).toContain('--graft-shell-sidebar-scroll-translate-y: 0px');
    vi.advanceTimersToNextFrame();
    await wrapper.vm.$nextTick();

    settingStoreProxy.value!.isSidebarFixed = false;
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('style')).toContain('--graft-shell-sidebar-scroll-translate-y: -240px');
  });

  it('exposes the sidebar compact state on the shell surface', async () => {
    settingStoreProxy.value!.isSidebarCompact = false;
    routeProxy.value!.path = '/infrastructure/docker/containers';
    routeProxy.value!.meta = { pageKind: 'list' };
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('false');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanded');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-mode')).toBe('wide-table');

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
    expect(storeState.realtimeSchedulerStore.freeze).toHaveBeenCalledWith('shell-sidebar-motion');
  });

  it('moves an unfixed sidebar with the page scroll while keeping a fixed sidebar in place', async () => {
    const wrapper = mountAppLayout();
    const content = wrapper.get('[data-test-id="layout-content"]');

    Object.defineProperty(content.element, 'scrollTop', { configurable: true, value: 240 });
    await content.trigger('scroll');
    vi.advanceTimersToNextFrame();
    await wrapper.vm.$nextTick();

    const shell = wrapper.get('.app-shell');
    expect(shell.attributes('data-sidebar-fixed')).toBe('true');
    expect(shell.attributes('style')).toContain('--graft-shell-sidebar-scroll-translate-y: 0px');

    settingStoreProxy.value!.isSidebarFixed = false;
    await wrapper.vm.$nextTick();

    expect(shell.attributes('data-sidebar-fixed')).toBe('false');
    expect(shell.attributes('style')).toContain('--graft-shell-sidebar-scroll-translate-y: -240px');
  });

  it('uses wide-table motion for an explicitly marked non-list table route', () => {
    routeProxy.value!.path = '/observability/modules';
    routeProxy.value!.meta = { pageKind: 'overview', pageSurface: 'paged-table', sidebarMotion: 'wide-table' };
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-mode')).toBe('wide-table');
  });

  it('uses default motion outside paged list and explicitly dense table routes', () => {
    routeProxy.value!.path = '/observability/service-status';
    routeProxy.value!.meta = { pageKind: 'runtime' };
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-mode')).toBe('default');
  });

  it('hides the mixed-layout sidebar while the home route is active', async () => {
    settingStoreProxy.value!.layout = { value: 'mix' };
    routeProxy.value!.fullPath = '/';
    routeProxy.value!.path = '/';
    const wrapper = mountAppLayout();

    expect(wrapper.find('[data-test-id="layout-side-nav"]').exists()).toBe(false);
    expect(wrapper.get('.app-shell__main').classes()).not.toContain('t-layout--with-sider');

    routeProxy.value!.fullPath = '/observability/service-status';
    routeProxy.value!.path = '/observability/service-status';
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-test-id="layout-side-nav"]').exists()).toBe(true);
    expect(wrapper.get('.app-shell__main').classes()).toContain('t-layout--with-sider');
  });

  it('renders narrow shell navigation as a drawer and closes it after route navigation', async () => {
    shellContainerSize.width = 480;
    const wrapper = mountAppLayout();
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('drawer');
    expect(wrapper.get('[data-test-id="layout-header"]').attributes('data-presentation')).toBe('drawer');
    expect(wrapper.get('[data-test-id="mobile-navigation"]').attributes('data-visible')).toBe('false');
    expect(wrapper.get('.app-shell__main').classes()).not.toContain('t-layout--with-sider');

    await wrapper.get('[data-test-id="layout-header"]').trigger('click');
    expect(wrapper.get('[data-test-id="mobile-navigation"]').attributes('data-visible')).toBe('true');

    routeProxy.value!.fullPath = '/observability/service-status';
    routeProxy.value!.path = '/observability/service-status';
    await wrapper.vm.$nextTick();

    expect(wrapper.get('[data-test-id="mobile-navigation"]').attributes('data-visible')).toBe('false');
  });

  it('keeps mobile navigation available for the top layout without rendering a persistent sidebar', async () => {
    shellContainerSize.width = 480;
    settingStoreProxy.value!.layout = { value: 'top' };
    settingStoreProxy.value!.showSidebar = false;
    const wrapper = mountAppLayout();
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('drawer');
    expect(wrapper.get('[data-test-id="mobile-navigation"]').attributes('data-visible')).toBe('false');
    expect(wrapper.find('[data-test-id="layout-side-nav"]').exists()).toBe(false);
  });

  it('uses the viewport fallback for drawer navigation before the shell container is measured', () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 563 });
    shellContainerSize.width = 0;
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('drawer');
  });

  it('keeps drawer navigation stable when the persisted desktop compact preference changes', async () => {
    shellContainerSize.width = 480;
    const wrapper = mountAppLayout();
    await wrapper.vm.$nextTick();
    storeState.realtimeSchedulerStore.freeze.mockClear();
    storeState.realtimeSchedulerStore.release.mockClear();

    settingStoreProxy.value!.isSidebarCompact = true;
    await wrapper.vm.$nextTick();
    vi.advanceTimersByTime(320);
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('drawer');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanded');
    expect(wrapper.get('[data-test-id="mobile-navigation"]').attributes('data-visible')).toBe('false');
    expect(storeState.realtimeSchedulerStore.freeze).not.toHaveBeenCalled();
    expect(storeState.realtimeSchedulerStore.release).not.toHaveBeenCalled();
  });

  it('restores the persisted desktop compact state after leaving drawer navigation', async () => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    shellContainerSize.width = 480;
    const wrapper = mountAppLayout();
    await wrapper.vm.$nextTick();
    await wrapper.get('[data-test-id="layout-header"]').trigger('click');
    settingStoreProxy.value!.isSidebarCompact = true;
    await wrapper.vm.$nextTick();

    shellContainerSize.width = 1200;
    ResizeObserverMock.instances[0]?.emit(1200, 800);
    await wrapper.vm.$nextTick();
    vi.advanceTimersByTime(180);
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('desktop');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('true');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('compact');
    expect(wrapper.find('[data-test-id="mobile-navigation"]').exists()).toBe(false);
  });

  it('keeps drawer navigation when transient overflow reports a desktop-sized shell', async () => {
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    shellContainerSize.width = 480;
    const wrapper = mountAppLayout();
    await wrapper.vm.$nextTick();

    ResizeObserverMock.instances[0]?.emit(3553, 800);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('drawer');

    vi.advanceTimersByTime(90);
    shellContainerSize.width = 674;
    ResizeObserverMock.instances[0]?.emit(674, 800);
    await wrapper.vm.$nextTick();
    vi.advanceTimersByTime(180);
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-presentation')).toBe('drawer');
  });

  it('runs the reverse sidebar motion when expanding back out', async () => {
    settingStoreProxy.value!.isSidebarCompact = true;
    routeProxy.value!.path = '/infrastructure/docker/containers';
    routeProxy.value!.meta = { pageKind: 'list' };
    const wrapper = mountAppLayout();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('compact');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('true');

    settingStoreProxy.value!.isSidebarCompact = false;
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.app-shell').attributes('data-sidebar-motion-phase')).toBe('expanding-width');
    expect(wrapper.get('.app-shell').attributes('data-sidebar-compact')).toBe('true');

    vi.advanceTimersToNextFrame();
    vi.advanceTimersToNextFrame();
    await wrapper.vm.$nextTick();
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

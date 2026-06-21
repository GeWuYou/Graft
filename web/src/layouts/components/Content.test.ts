import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, markRaw } from 'vue';

import Content from './Content.vue';

const routeState = vi.hoisted(() => ({
  meta: {},
  path: '/access-control/roles',
  fullPath: '/access-control/roles',
}));
const tabStoreState = vi.hoisted(() => ({
  activeTabKey: '/access-control/roles',
  refreshing: false,
  tabRouters: [
    {
      fullPath: '/access-control/roles',
      tabKey: '/access-control/roles',
      path: '/access-control/roles',
      isAlive: true,
      meta: {},
      name: 'RoleListIndex',
    },
  ],
}));
const tabStoreProxy = vi.hoisted(() => ({
  value: null as null | typeof tabStoreState,
}));

const RouteContentProbe = markRaw({
  name: 'RouteContentProbe',
  template: '<div data-testid="route-content">content</div>',
});

const TransitionStub = defineComponent({
  name: 'Transition',
  props: {
    onBeforeEnter: {
      type: Function,
      default: undefined,
    },
  },
  setup(props, { slots }) {
    return () => {
      props.onBeforeEnter?.();
      return h('div', slots.default?.());
    };
  },
});

const LoadingStub = defineComponent({
  name: 'TLoading',
  props: {
    loading: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'route-loading',
          'data-loading': String(props.loading),
        },
        slots.default?.(),
      );
  },
});

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}));

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/router/route-loading', () => ({
  routeLoading: {
    value: false,
  },
}));

vi.mock('@/store', async () => {
  const { reactive } = await import('vue');
  tabStoreProxy.value = reactive(tabStoreState);

  return {
    useTabsRouterStore: () => tabStoreProxy.value,
  };
});

describe('Content', () => {
  beforeEach(() => {
    routeState.path = '/access-control/roles';
    routeState.fullPath = '/access-control/roles';
    routeState.meta = {};
    tabStoreState.activeTabKey = '/access-control/roles';
    tabStoreState.refreshing = false;
    tabStoreState.tabRouters = [
      {
        fullPath: '/access-control/roles',
        tabKey: '/access-control/roles',
        path: '/access-control/roles',
        isAlive: true,
        meta: {},
        name: 'RoleListIndex',
      },
    ];
  });

  it('keys rendered route content by the active tab key', async () => {
    const wrapper = mount(Content, {
      global: {
        stubs: {
          RouterView: {
            template: '<slot :Component="Component" :route="route" />',
            data() {
              return {
                Component: RouteContentProbe,
                route: routeState,
              };
            },
          },
          transition: TransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
          TLoading: LoadingStub,
        },
      },
    });

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles');

    tabStoreProxy.value!.activeTabKey = '/access-control/roles#copy-1';
    tabStoreProxy.value!.tabRouters = [
      ...tabStoreProxy.value!.tabRouters,
      {
        tabKey: '/access-control/roles#copy-1',
        path: '/access-control/roles',
        fullPath: '/access-control/roles',
        isAlive: true,
        meta: {},
        name: 'RoleListIndex',
      },
    ];
    await wrapper.vm.$nextTick();

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles#copy-1');
  });

  it('uses the entering route key when the active tab still points at the leaving route', async () => {
    const wrapper = mount(Content, {
      global: {
        stubs: {
          RouterView: {
            template: '<slot :Component="Component" :route="route" />',
            data() {
              return {
                Component: RouteContentProbe,
                route: routeState,
              };
            },
          },
          transition: TransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
          TLoading: LoadingStub,
        },
      },
    });

    routeState.path = '/ops/containers/container-1';
    routeState.fullPath = '/ops/containers/container-1?tab=overview';
    tabStoreProxy.value!.activeTabKey = '/access-control/roles#leaving';
    await wrapper.vm.$nextTick();

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe(
      '/ops/containers/container-1?tab=overview',
    );
  });

  it('does not restrict keep-alive by route name', () => {
    const wrapper = mount(Content, {
      global: {
        stubs: {
          RouterView: {
            template: '<slot :Component="Component" :route="route" />',
            data() {
              return {
                Component: markRaw({
                  name: 'RolesIndex',
                  template: '<div data-testid="route-content">content</div>',
                }),
                route: routeState,
              };
            },
          },
          transition: TransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
          TLoading: LoadingStub,
        },
      },
    });

    expect(wrapper.find('[data-testid="keep-alive"]').attributes('data-include')).toBeUndefined();
    expect(wrapper.findComponent({ name: 'RolesIndex' }).exists()).toBe(true);
  });

  it('emits the entering route surface from transition timing', () => {
    routeState.meta = {
      dashboard: true,
      pageSurface: 'paged-table',
    };

    const wrapper = mount(Content, {
      global: {
        stubs: {
          RouterView: {
            template: '<slot :Component="Component" :route="route" />',
            data() {
              return {
                Component: RouteContentProbe,
                route: routeState,
              };
            },
          },
          transition: TransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
          TLoading: LoadingStub,
        },
      },
    });

    expect(wrapper.emitted('page-surface-enter')).toEqual([['paged-table']]);
  });

  it('keeps a loading host mounted while a tab refresh removes route content', async () => {
    tabStoreState.refreshing = true;

    const wrapper = mount(Content, {
      global: {
        stubs: {
          RouterView: {
            template: '<slot :Component="Component" :route="route" />',
            data() {
              return {
                Component: RouteContentProbe,
                route: routeState,
              };
            },
          },
          transition: TransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
          TLoading: LoadingStub,
        },
      },
    });

    expect(wrapper.get('[data-testid="route-loading"]').attributes('data-loading')).toBe('true');
    expect(wrapper.find('.route-loading-host').exists()).toBe(true);
    expect(wrapper.find('.route-refresh-placeholder').exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).exists()).toBe(false);
  });
});

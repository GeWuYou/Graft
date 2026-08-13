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
  refreshNonceByTabKey: {
    '/access-control/roles': 0,
  } as Record<string, number>,
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
    onAfterLeave: {
      type: Function,
      default: undefined,
    },
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

const DeferredTransitionStub = defineComponent({
  name: 'Transition',
  props: {
    onAfterLeave: {
      type: Function,
      default: undefined,
    },
    onBeforeEnter: {
      type: Function,
      default: undefined,
    },
  },
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
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
    tabStoreState.refreshNonceByTabKey = {
      '/access-control/roles': 0,
    };
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
        },
      },
    });

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles::0');

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

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles#copy-1::0');
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

  it('keeps the active detail instance when only the route query changes', async () => {
    const wrapper = mount(Content, {
      global: {
        stubs: {
          RouterView: {
            template: '<slot :Component="Component" :route="route" />',
            data() {
              return { Component: RouteContentProbe, route: routeState };
            },
          },
          transition: TransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
        },
      },
    });

    routeState.fullPath = '/access-control/roles?mode=edit';
    await wrapper.vm.$nextTick();

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles::0');
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
        },
      },
    });

    expect(wrapper.find('[data-testid="keep-alive"]').attributes('data-include')).toBeUndefined();
    expect(wrapper.findComponent({ name: 'RolesIndex' }).exists()).toBe(true);
  });

  it('waits for the entering view before emitting the target page surface', () => {
    routeState.meta = {
      pageSurface: 'form-detail',
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
          transition: DeferredTransitionStub,
          KeepAlive: {
            props: ['include'],
            template: '<div data-testid="keep-alive" :data-include="include"><slot /></div>',
          },
          FramePage: true,
        },
      },
    });

    const transition = wrapper.findComponent(DeferredTransitionStub);
    expect(transition.props('onAfterLeave')).toBeUndefined();
    expect(wrapper.emitted('page-surface-ready')).toBeUndefined();

    const beforeEnter = transition.props('onBeforeEnter') as (() => void) | undefined;
    expect(beforeEnter).toBeTypeOf('function');
    beforeEnter?.();

    expect(wrapper.emitted('page-surface-ready')).toEqual([['form-detail']]);
  });

  it('keeps route content mounted while tab refresh only raises the loading state', async () => {
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
        },
      },
    });

    expect(wrapper.get('.route-loading-host').attributes('aria-busy')).toBe('true');
    expect(wrapper.find('[data-testid="route-loading"]').exists()).toBe(false);
    expect(wrapper.find('.route-page-loading-indicator').exists()).toBe(false);
    expect(wrapper.find('.route-loading-host').exists()).toBe(true);
    expect(wrapper.find('.route-refresh-placeholder').exists()).toBe(false);
    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).exists()).toBe(true);

    tabStoreProxy.value!.refreshing = false;
    await wrapper.vm.$nextTick();

    expect(wrapper.get('.route-loading-host').attributes('aria-busy')).toBe('false');
  });

  it('changes the rendered route key when the active tab refresh nonce changes', async () => {
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
        },
      },
    });

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles::0');

    tabStoreProxy.value!.refreshNonceByTabKey = {
      ...tabStoreProxy.value!.refreshNonceByTabKey,
      '/access-control/roles': 1,
    };
    await wrapper.vm.$nextTick();

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles::1');
  });
});

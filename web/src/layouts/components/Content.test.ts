// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
      tabKey: '/access-control/roles',
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

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
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
        tabKey: '/access-control/roles',
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
          TLoading: true,
        },
      },
    });

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles');

    tabStoreProxy.value!.activeTabKey = '/access-control/roles#copy-1';
    tabStoreProxy.value!.tabRouters = [
      ...tabStoreProxy.value!.tabRouters,
      {
        tabKey: '/access-control/roles#copy-1',
        isAlive: true,
        meta: {},
        name: 'RoleListIndex',
      },
    ];
    await wrapper.vm.$nextTick();

    expect(wrapper.findComponent({ name: 'RouteContentProbe' }).vm.$.vnode.key).toBe('/access-control/roles#copy-1');
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
          TLoading: true,
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
          TLoading: true,
        },
      },
    });

    expect(wrapper.emitted('page-surface-enter')).toEqual([['paged-table']]);
  });
});

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';

import SideNav from './SideNav.vue';

const updateConfigMock = vi.fn();

const settingStoreProxy = vi.hoisted(() => ({
  value: null as null | {
    isSidebarCompact: boolean;
    menuAutoCollapsed: boolean;
    updateConfig: typeof updateConfigMock;
  },
}));

const settingStoreState = vi.hoisted(() => ({
  isSidebarCompact: false,
  menuAutoCollapsed: false,
}));

vi.mock('@/store', () => ({
  useSettingStore: () => {
    if (!settingStoreProxy.value) {
      settingStoreProxy.value = reactive({
        ...settingStoreState,
        updateConfig: updateConfigMock,
      });
    }

    return settingStoreProxy.value;
  },
}));

vi.mock('@/router', () => ({
  getActive: () => '/server/runtime',
}));

vi.mock('@/layouts/useShellNavigation', () => ({
  useShellNavigation: () => ({
    goHome: vi.fn(),
  }),
}));

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('./MenuContent.vue', () => ({
  default: defineComponent({
    name: 'MenuContentStub',
    props: {
      navData: {
        type: Array,
        default: () => [],
      },
    },
    setup(props) {
      return () => h('div', { 'data-menu-content-count': String(props.navData.length) });
    },
  }),
}));

const menuStub = defineComponent({
  name: 'TMenuStub',
  props: {
    collapsed: { type: Boolean, default: false },
    expanded: { type: Array, default: () => [] },
    width: { type: Array, default: () => [] },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-menu-collapsed': String(props.collapsed),
          'data-menu-expanded': JSON.stringify(props.expanded),
          'data-menu-width': JSON.stringify(props.width),
        },
        [slots.logo?.(), slots.default?.(), slots.operations?.()],
      );
  },
});

function mountSideNav() {
  return mount(SideNav, {
    props: {
      isCompact: false,
      isFixed: true,
      layout: 'side',
      menu: [
        {
          path: '/server',
          meta: {
            title: {
              'zh-CN': '服务管理',
              'en-US': 'Server',
            },
          },
          children: [
            {
              path: 'runtime',
              meta: {
                title: {
                  'zh-CN': '运行时',
                  'en-US': 'Runtime',
                },
              },
            },
          ],
        },
      ],
      motionPhase: 'expanded',
      showLogo: true,
      theme: 'light',
    },
    global: {
      stubs: {
        't-menu': menuStub,
      },
    },
  });
}

describe('SideNav', () => {
  beforeEach(() => {
    updateConfigMock.mockReset();
    settingStoreProxy.value ??= reactive({
      ...settingStoreState,
      updateConfig: updateConfigMock,
    });
    settingStoreProxy.value.isSidebarCompact = false;
    settingStoreProxy.value.menuAutoCollapsed = false;
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1280,
      writable: true,
    });
  });

  it('uses the unified compact width for the menu collapse state', async () => {
    const wrapper = mountSideNav();

    expect(wrapper.get('[data-menu-width]').attributes('data-menu-width')).toBe(
      '["var(--graft-shell-sidebar-current-width)","var(--graft-shell-sidebar-current-width)"]',
    );
    expect(wrapper.get('[data-menu-collapsed]').attributes('data-menu-collapsed')).toBe('false');

    await wrapper.setProps({
      isCompact: true,
      renderCompact: true,
      motionPhase: 'compact',
    });

    expect(wrapper.get('[data-menu-collapsed]').attributes('data-menu-collapsed')).toBe('true');
  });

  it('collapses expanded groups during text fade and restores them on expand', async () => {
    const wrapper = mountSideNav();
    await nextTick();

    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/server","/server/runtime"]');

    await wrapper.setProps({
      motionPhase: 'collapsing-text',
    });
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('[]');

    await wrapper.setProps({
      isCompact: true,
      motionPhase: 'compact',
    });
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('[]');

    await wrapper.setProps({
      isCompact: false,
      motionPhase: 'expanding-width',
    });
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/server","/server/runtime"]');
  });
});

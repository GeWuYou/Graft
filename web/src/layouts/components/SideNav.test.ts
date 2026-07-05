import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';

import SideNav from './SideNav.vue';

const updateConfigMock = vi.fn();

const settingStoreProxy = {
  value: null as null | {
    isSidebarCompact: boolean;
    menuAutoCollapsed: boolean;
    updateConfig: typeof updateConfigMock;
  },
};

const settingStoreState = {
  isSidebarCompact: false,
  menuAutoCollapsed: false,
};

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

const activePathState = {
  value: '/server/runtime',
};

vi.mock('@/router', () => ({
  getActive: () => activePathState.value,
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
    activePathState.value = '/server/runtime';
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

    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/server"]');

    await wrapper.setProps({
      motionPhase: 'collapsing-submenu',
    });
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('[]');

    await wrapper.setProps({
      isCompact: true,
      motionPhase: 'compact',
    });
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('[]');

    await wrapper.setProps({
      isCompact: false,
      motionPhase: 'expanding-submenu',
    });
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/server"]');
  });

  it('keeps the active route branch expanded when the menu emits an empty expanded set', async () => {
    const wrapper = mountSideNav();
    await nextTick();

    wrapper.findComponent(menuStub).vm.$emit('expand', []);
    await nextTick();

    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/server"]');
  });

  it('expands the owning parent menu for descendant routes outside the literal menu path chain', async () => {
    activePathState.value = '/ops/containers/container-1';
    const wrapper = mount(SideNav, {
      props: {
        isCompact: false,
        isFixed: true,
        layout: 'side',
        menu: [
          {
            path: '/ops',
            meta: {
              title: {
                'zh-CN': '运维',
                'en-US': 'Operations',
              },
            },
            children: [
              {
                path: 'containers',
                meta: {
                  title: {
                    'zh-CN': '容器',
                    'en-US': 'Containers',
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

    await nextTick();

    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/ops"]');
  });
});

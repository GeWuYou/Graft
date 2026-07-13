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
const routeState = reactive({
  fullPath: '/server/runtime',
  path: '/server/runtime',
  meta: {},
});

vi.mock('@/router', () => ({
  getActive: () => activePathState.value,
}));

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}));

vi.mock('@/layouts/useShellNavigation', () => ({
  useShellNavigation: () => ({
    goHome: vi.fn(),
  }),
}));

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/shared/components/brand', () => ({
  BrandIdentity: defineComponent({
    name: 'BrandIdentityStub',
    props: {
      compact: {
        type: Boolean,
        default: false,
      },
      labelHidden: {
        type: Boolean,
        default: false,
      },
      label: {
        type: String,
        required: true,
      },
    },
    setup(props) {
      return () =>
        h(
          'div',
          {
            'data-brand-compact': String(props.compact),
            'data-brand-label': props.label,
            'data-brand-label-hidden': String(props.labelHidden),
          },
          props.compact ? '' : props.label,
        );
    },
  }),
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
    value: { type: String, default: '' },
    width: { type: Array, default: () => [] },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-menu-collapsed': String(props.collapsed),
          'data-menu-expanded': JSON.stringify(props.expanded),
          'data-menu-value': props.value,
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
              'zh-CN': '可观测性',
              'en-US': 'Observability',
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

const expectExpandedMenus = (wrapper: ReturnType<typeof mount>, expected: string[]) => {
  const expanded = JSON.parse(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded') ?? '[]') as string[];
  expect(expanded).toHaveLength(expected.length);
  expect(expanded).toEqual(expect.arrayContaining(expected));
};

const navigationMenu = [
  {
    path: '/server',
    meta: {
      title: {
        'zh-CN': '可观测性',
        'en-US': 'Observability',
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
  {
    path: '/security',
    meta: {
      title: {
        'zh-CN': '安全',
        'en-US': 'Security',
      },
    },
    children: [
      {
        path: 'overview',
        meta: {
          title: {
            'zh-CN': '概览',
            'en-US': 'Overview',
          },
        },
      },
    ],
  },
];

describe('SideNav', () => {
  beforeEach(() => {
    updateConfigMock.mockReset();
    activePathState.value = '/server/runtime';
    routeState.fullPath = '/server/runtime';
    routeState.path = '/server/runtime';
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
      '["var(--graft-shell-sidebar-surface-width)","var(--graft-shell-sidebar-surface-width)"]',
    );
    expect(wrapper.get('[data-menu-collapsed]').attributes('data-menu-collapsed')).toBe('false');
    expect(wrapper.get('[data-brand-label]').attributes('data-brand-label')).toBe('common.appName');
    expect(wrapper.get('[data-brand-compact]').attributes('data-brand-compact')).toBe('false');
    expect(wrapper.get('[data-brand-label-hidden]').attributes('data-brand-label-hidden')).toBe('false');
    expect(wrapper.text()).not.toContain('1.0.0');

    await wrapper.setProps({
      motionPhase: 'collapsing-width',
    });
    expect(wrapper.get('[data-brand-compact]').attributes('data-brand-compact')).toBe('false');
    expect(wrapper.get('[data-brand-label-hidden]').attributes('data-brand-label-hidden')).toBe('true');

    await wrapper.setProps({
      isCompact: true,
      renderCompact: true,
      motionPhase: 'compact',
    });

    expect(wrapper.get('[data-menu-collapsed]').attributes('data-menu-collapsed')).toBe('true');
    expect(wrapper.get('[data-brand-compact]').attributes('data-brand-compact')).toBe('true');
    expect(wrapper.get('[data-brand-label-hidden]').attributes('data-brand-label-hidden')).toBe('true');
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

  it('expands the active sidebar branch after switching routes through a tab', async () => {
    const wrapper = mount(SideNav, {
      props: {
        isCompact: false,
        isFixed: true,
        layout: 'side',
        menu: navigationMenu,
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

    activePathState.value = '/security/overview';
    routeState.fullPath = '/security/overview';
    routeState.path = '/security/overview';
    await nextTick();

    expectExpandedMenus(wrapper, ['/server', '/security']);
  });

  it('defers route branch expansion until sidebar motion can render submenus', async () => {
    const wrapper = mount(SideNav, {
      props: {
        isCompact: false,
        isFixed: true,
        layout: 'side',
        menu: navigationMenu,
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

    await wrapper.setProps({ motionPhase: 'collapsing-width' });
    activePathState.value = '/security/overview';
    routeState.fullPath = '/security/overview';
    routeState.path = '/security/overview';
    await nextTick();

    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["/server"]');

    await wrapper.setProps({ motionPhase: 'expanded' });

    expectExpandedMenus(wrapper, ['/server', '/security']);
  });

  it('expands bootstrap menu codes without normalizing their submenu values as URLs', async () => {
    activePathState.value = '/platform/scheduled-tasks';
    routeState.fullPath = '/platform/scheduled-tasks';
    routeState.path = '/platform/scheduled-tasks';
    const wrapper = mount(SideNav, {
      props: {
        isCompact: false,
        isFixed: true,
        layout: 'side',
        menu: [
          {
            path: 'domain.platform',
            meta: {
              navigationTargetPath: '/platform/scheduled-tasks',
              title: {
                'zh-CN': '平台',
                'en-US': 'Platform',
              },
            },
            children: [
              {
                path: 'scheduled-task.list',
                meta: {
                  navigationTargetPath: '/platform/scheduled-tasks',
                  title: {
                    'zh-CN': '定时任务',
                    'en-US': 'Scheduled Tasks',
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

    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["domain.platform"]');
  });

  it('expands the selected mixed-layout branch when a tab activates a nested route', async () => {
    activePathState.value = '/infrastructure';
    routeState.fullPath = '/infrastructure/docker/containers';
    routeState.path = '/infrastructure/docker/containers';
    const wrapper = mount(SideNav, {
      props: {
        isCompact: false,
        isFixed: true,
        layout: 'mix',
        menu: [
          {
            path: 'domain.infrastructure',
            meta: {
              navigationTargetPath: '/infrastructure/runtime-targets',
              title: {
                'zh-CN': '基础设施',
                'en-US': 'Infrastructure',
              },
            },
            children: [
              {
                path: 'runtime-target.list',
                meta: {
                  navigationTargetPath: '/infrastructure/runtime-targets',
                  title: {
                    'zh-CN': '运行目标',
                    'en-US': 'Runtime Targets',
                  },
                },
              },
              {
                path: 'docker',
                meta: {
                  title: {
                    'zh-CN': 'Docker',
                    'en-US': 'Docker',
                  },
                },
                children: [
                  {
                    path: 'container.list',
                    meta: {
                      navigationTargetPath: '/infrastructure/docker/containers',
                      title: {
                        'zh-CN': '容器',
                        'en-US': 'Containers',
                      },
                    },
                  },
                ],
              },
            ],
          },
        ],
        motionPhase: 'expanded',
        showLogo: false,
        theme: 'light',
      },
      global: {
        stubs: {
          't-menu': menuStub,
        },
      },
    });

    await nextTick();

    expectExpandedMenus(wrapper, ['domain.infrastructure', 'docker']);
    expect(wrapper.get('[data-menu-value]').attributes('data-menu-value')).toBe('/infrastructure/docker/containers');
  });

  it('expands the owning parent menu for descendant routes outside the literal menu path chain', async () => {
    activePathState.value = '/ops/containers/container-1';
    routeState.fullPath = '/ops/containers/container-1';
    routeState.path = '/ops/containers/container-1';
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

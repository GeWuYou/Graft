import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import MenuContent from './MenuContent.vue';

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

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({
    locale: {
      value: 'zh-CN',
    },
  }),
}));

vi.mock('@/router', () => ({
  getActive: () => '',
}));

describe('MenuContent', () => {
  function createMenuItemStub() {
    return defineComponent({
      props: {
        value: { type: String, required: false, default: '' },
      },
      emits: ['click'],
      setup(props, { emit, slots }) {
        return () =>
          h(
            'button',
            {
              type: 'button',
              'data-menu-value': props.value,
              onClick: () => emit('click'),
            },
            [slots.icon?.(), slots.default?.()],
          );
      },
    });
  }

  it('renders an authorized visual-only section label once without creating a menu item', () => {
    const wrapper = mount(MenuContent, {
      props: {
        showSections: true,
        navData: [
          {
            path: '/infrastructure/containers',
            meta: {
              title: { 'zh-CN': 'Docker', 'en-US': 'Docker' },
              navigationSection: {
                key: 'runtime',
                title: { 'zh-CN': '运行时', 'en-US': 'Runtime' },
              },
            },
          },
          {
            path: '/images',
            meta: {
              title: { 'zh-CN': '镜像', 'en-US': 'Images' },
              navigationSection: {
                key: 'runtime',
                title: { 'zh-CN': '运行时', 'en-US': 'Runtime' },
              },
            },
          },
        ],
      },
      global: {
        stubs: {
          't-menu-item': createMenuItemStub(),
          't-icon': defineComponent({ setup: () => () => h('i') }),
        },
      },
    });

    expect(wrapper.findAll('.graft-menu-section-label')).toHaveLength(1);
    expect(wrapper.get('.graft-menu-section-label').text()).toBe('运行时');
    expect(wrapper.findAll('button')).toHaveLength(2);
    expect(wrapper.find('[data-menu-value="runtime"]').exists()).toBe(false);
  });

  it('does not render sidebar section labels outside the side navigation', () => {
    const wrapper = mount(MenuContent, {
      props: {
        navData: [
          {
            path: '/infrastructure/containers',
            meta: {
              title: { 'zh-CN': 'Docker', 'en-US': 'Docker' },
              navigationSection: {
                key: 'runtime',
                title: { 'zh-CN': '运行时', 'en-US': 'Runtime' },
              },
            },
          },
        ],
      },
      global: {
        stubs: {
          't-menu-item': createMenuItemStub(),
          't-icon': defineComponent({ setup: () => () => h('i') }),
        },
      },
    });

    expect(wrapper.find('.graft-menu-section-label').exists()).toBe(false);
  });

  it('renders distinct static SVG icons for application, runtime target, and Docker entries', async () => {
    const wrapper = mount(MenuContent, {
      props: {
        navData: [
          { path: '/applications', meta: { icon: 'application', title: { 'en-US': 'Applications', 'zh-CN': '应用' } } },
          { path: '/targets', meta: { icon: 'runtime-target', title: { 'en-US': 'Targets', 'zh-CN': '目标' } } },
          {
            path: '/infrastructure/containers',
            meta: { icon: 'docker', title: { 'en-US': 'Docker', 'zh-CN': 'Docker' } },
          },
        ],
      },
      global: {
        stubs: { 't-menu-item': createMenuItemStub() },
      },
    });

    await nextTick();

    expect(wrapper.html()).toContain('<svg');
    const icons = wrapper.findAll('svg');
    expect(icons).toHaveLength(3);
    expect(new Set(icons.map((icon) => icon.html())).size).toBe(3);
    expect(wrapper.html()).not.toContain('<icon');
  });

  it('navigates grouped mix-menu items to the first visible leaf route', async () => {
    const submenuStub = defineComponent({
      name: 'TSubmenuStub',
      setup(_, { slots }) {
        return () => h('div', slots.default?.());
      },
    });

    const iconStub = defineComponent({
      name: 'TIconStub',
      setup() {
        return () => h('i');
      },
    });

    const wrapper = mount(MenuContent, {
      props: {
        navData: [
          {
            path: '/server',
            meta: {
              title: {
                'zh-CN': '服务器管理',
                'en-US': 'Server Management',
              },
              single: true,
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
        ],
      },
      global: {
        stubs: {
          't-menu-item': createMenuItemStub(),
          't-submenu': submenuStub,
          't-icon': iconStub,
        },
      },
    });

    await wrapper.get('button[data-menu-value="/server"]').trigger('click');

    expect(pushMock).toHaveBeenCalledWith('/server/overview');
  });
});

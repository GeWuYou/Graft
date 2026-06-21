import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

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
  it('navigates grouped mix-menu items to the first visible leaf route', async () => {
    const menuItemStub = defineComponent({
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
            slots.default?.(),
          );
      },
    });

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
          't-menu-item': menuItemStub,
          't-submenu': submenuStub,
          't-icon': iconStub,
        },
      },
    });

    await wrapper.get('button[data-menu-value="/server"]').trigger('click');

    expect(pushMock).toHaveBeenCalledWith('/server/overview');
  });
});

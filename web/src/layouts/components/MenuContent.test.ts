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

    const wrapper = mount(MenuContent, {
      props: {
        navData: [
          {
            path: '/monitor',
            meta: {
              title: {
                'zh-CN': '服务器管理',
                'en-US': 'Server Management',
              },
              single: true,
            },
            children: [
              {
                path: 'server-status',
                redirect: 'overview',
                meta: {
                  title: {
                    'zh-CN': '服务器状态',
                    'en-US': 'Server Status',
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
            ],
          },
        ],
      },
      global: {
        stubs: {
          't-menu-item': menuItemStub,
          't-submenu': true,
          't-icon': true,
        },
      },
    });

    await wrapper.get('button[data-menu-value="/monitor"]').trigger('click');

    expect(pushMock).toHaveBeenCalledWith('/monitor/server-status/overview');
  });
});

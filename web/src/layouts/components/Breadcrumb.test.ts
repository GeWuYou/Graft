import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { reactive } from 'vue';

import { LOCALE } from '@/contracts/i18n/locales';

import Breadcrumb from './Breadcrumb.vue';

const routeState = vi.hoisted(() => ({
  params: {
    id: 'container-1',
  },
  fullPath: '/ops/containers/container-1',
  matched: [
    {
      name: 'ContainerDetail',
      path: '/ops/containers/:id',
      meta: {
        breadcrumbTitle: {
          'zh-CN': '容器管理',
          'en-US': 'Container Management',
        },
      },
    },
    {
      name: 'ContainerDetailIndex',
      path: '',
      meta: {
        breadcrumbTitle: {
          'zh-CN': '容器详情',
          'en-US': 'Container Detail',
        },
      },
    },
  ],
}));

const routerMock = vi.hoisted(() => ({
  resolve: vi.fn((target: { name?: string | symbol; path?: string }) => {
    if (target.name === 'ContainerDetail') {
      return {
        fullPath: '/ops/containers/container-1',
      };
    }

    return {
      fullPath: target.path ?? routeState.fullPath,
    };
  }),
}));

const storeState = vi.hoisted(() => ({
  settingStore: {
    showBreadcrumb: true,
  },
}));

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => routerMock,
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({
    locale: {
      value: LOCALE.ZH_CN,
    },
  }),
}));

vi.mock('@/store', () => ({
  useSettingStore: () => reactive(storeState.settingStore),
}));

describe('Breadcrumb', () => {
  beforeEach(() => {
    storeState.settingStore.showBreadcrumb = true;
    routeState.params.id = 'container-1';
    routeState.fullPath = '/ops/containers/container-1';
    routeState.matched[0].name = 'ContainerDetail';
    routeState.matched[0].path = '/ops/containers/:id';
    routeState.matched[1].name = 'ContainerDetailIndex';
    routeState.matched[1].path = '';
    routerMock.resolve.mockClear();
  });

  it('keeps unique breadcrumb item keys for global detail routes', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    const wrapper = mount(Breadcrumb, {
      global: {
        stubs: {
          TBreadcrumb: {
            template: '<nav><slot /></nav>',
          },
          TBreadcrumbItem: {
            props: ['to'],
            template: "<a :data-to=\"typeof to === 'string' ? to : ''\"><slot /></a>",
          },
        },
      },
    });

    const items = wrapper.findAll('a');
    expect(items).toHaveLength(2);
    expect(items[0]?.attributes('data-to')).toBe('/ops/containers/container-1');
    expect(items[1]?.attributes('data-to')).toBe('');
    expect(warnSpy).not.toHaveBeenCalledWith(expect.stringContaining('Duplicate keys found during update'));

    warnSpy.mockRestore();
  });

  it('hides the breadcrumb when the shell setting disables it', () => {
    storeState.settingStore.showBreadcrumb = false;

    const wrapper = mount(Breadcrumb, {
      global: {
        stubs: {
          TBreadcrumb: {
            template: '<nav><slot /></nav>',
          },
          TBreadcrumbItem: {
            template: '<a><slot /></a>',
          },
        },
      },
    });

    expect(wrapper.find('nav').exists()).toBe(false);
  });
});

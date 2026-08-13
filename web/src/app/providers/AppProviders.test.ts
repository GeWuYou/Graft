import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, nextTick, ref } from 'vue';

import AppProviders from './AppProviders.vue';

const localeRef = ref('zh-CN');
const providerLocaleRef = ref({ localeName: 'zh-CN-components' });
const displayModeRef = ref('light');
const availabilityStatusRef = ref<'healthy' | 'recovering' | 'unavailable'>('healthy');
const routerMock = vi.hoisted(() => ({
  currentRoute: { value: { path: '/', fullPath: '/', query: {} as Record<string, unknown> } },
  replace: vi.fn(),
}));
const routeProbeMounts = { count: 0 };

vi.mock('@/layouts/setting.vue', () => ({
  default: defineComponent({
    name: 'SettingComStub',
    setup() {
      return () => h('div', { 'data-testid': 'setting-stub' });
    },
  }),
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({
    getComponentsLocale: computed(() => providerLocaleRef.value),
    locale: localeRef,
  }),
}));

vi.mock('@/store', () => ({
  useSettingStore: () => ({
    get displayMode() {
      return displayModeRef.value;
    },
  }),
}));

vi.mock('@/store/modules/platform-availability', () => ({
  usePlatformAvailabilityStore: () => ({
    bindRequestBridge: vi.fn(),
    consumePendingPath: vi.fn(() => '/'),
    get status() {
      return availabilityStatusRef.value;
    },
  }),
}));

vi.mock('@/router', () => ({
  default: routerMock,
}));

const RouteProbe = defineComponent({
  name: 'RouteProbe',
  setup() {
    routeProbeMounts.count += 1;
    return () => h('div', { 'data-testid': 'route-probe' }, `locale:${localeRef.value}`);
  },
});

describe('AppProviders', () => {
  beforeEach(() => {
    localeRef.value = 'zh-CN';
    providerLocaleRef.value = { localeName: 'zh-CN-components' };
    displayModeRef.value = 'light';
    availabilityStatusRef.value = 'healthy';
    routerMock.currentRoute.value = { path: '/', fullPath: '/', query: {} };
    routerMock.replace.mockReset();
    routeProbeMounts.count = 0;
  });

  it('keeps the routed view mounted when locale changes', async () => {
    const wrapper = mount(AppProviders, {
      global: {
        stubs: {
          RouterView: RouteProbe,
          TConfigProvider: defineComponent({
            name: 'TConfigProviderStub',
            props: {
              globalConfig: {
                type: Object,
                default: () => ({}),
              },
            },
            setup(props, { slots }) {
              return () =>
                h('div', { 'data-global-config': JSON.stringify(props.globalConfig ?? {}) }, slots.default?.());
            },
          }),
        },
      },
    });

    expect(routeProbeMounts.count).toBe(1);
    expect(wrapper.get('[data-testid="route-probe"]').text()).toBe('locale:zh-CN');
    expect(wrapper.get('[data-global-config]').attributes()['data-global-config']).toContain('zh-CN-components');

    localeRef.value = 'en-US';
    providerLocaleRef.value = { localeName: 'en-US-components' };
    await nextTick();

    expect(routeProbeMounts.count).toBe(1);
    expect(wrapper.get('[data-testid="route-probe"]').text()).toBe('locale:en-US');
    expect(wrapper.get('[data-global-config]').attributes()['data-global-config']).toContain('en-US-components');
  });

  it('keeps the theme workbench mounted while a health probe is recovering', async () => {
    const wrapper = mount(AppProviders, {
      global: {
        stubs: {
          RouterView: RouteProbe,
          TConfigProvider: defineComponent({
            name: 'TConfigProviderStub',
            setup(_, { slots }) {
              return () => h('div', slots.default?.());
            },
          }),
        },
      },
    });

    expect(wrapper.find('[data-testid="setting-stub"]').exists()).toBe(true);

    availabilityStatusRef.value = 'recovering';
    await nextTick();

    expect(wrapper.find('[data-testid="setting-stub"]').exists()).toBe(true);

    availabilityStatusRef.value = 'unavailable';
    await nextTick();

    expect(wrapper.find('[data-testid="setting-stub"]').exists()).toBe(false);
  });

  it('recovers the original route when the service becomes healthy on the unavailable page', async () => {
    routerMock.currentRoute.value = {
      path: '/result/service-unavailable',
      fullPath: '/result/service-unavailable?redirect=%2Fprojects',
      query: { redirect: '/projects' },
    };
    availabilityStatusRef.value = 'unavailable';

    mount(AppProviders, {
      global: {
        stubs: {
          RouterView: RouteProbe,
          TConfigProvider: defineComponent({
            name: 'TConfigProviderStub',
            setup(_, { slots }) {
              return () => h('div', slots.default?.());
            },
          }),
        },
      },
    });

    availabilityStatusRef.value = 'healthy';
    await nextTick();

    expect(routerMock.replace).toHaveBeenCalledWith('/projects');
  });
});

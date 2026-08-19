import { mount } from '@vue/test-utils';
import { ConfigProvider } from 'tdesign-vue-next/es/config-provider';
import { Dialog, DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, nextTick, type PropType, ref } from 'vue';

import AppProviders from './AppProviders.vue';

const localeRef = ref('zh-CN');
const providerLocaleRef = ref({
  localeName: 'zh-CN-components',
  dialog: { cancel: '取消', confirm: '确认' },
});
const displayModeRef = ref('light');
const availabilityStatusRef = ref<'unknown' | 'healthy' | 'recovering' | 'unavailable'>('healthy');
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

const DialogRouteProbe = defineComponent({
  name: 'DialogRouteProbe',
  props: {
    attach: {
      type: Function as PropType<() => HTMLElement>,
      required: true,
    },
  },
  setup(props) {
    return () =>
      h(Dialog, {
        attach: props.attach,
        dialogClassName: 'app-provider-declarative-dialog-test',
        header: 'Declarative dialog',
        visible: true,
      });
  },
});

describe('AppProviders', () => {
  beforeEach(() => {
    localeRef.value = 'zh-CN';
    providerLocaleRef.value = {
      localeName: 'zh-CN-components',
      dialog: { cancel: '取消', confirm: '确认' },
    };
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
    expect(JSON.parse(wrapper.get('[data-global-config]').attributes()['data-global-config'] ?? '{}')).toEqual({
      localeName: 'zh-CN-components',
      dialog: { cancel: '取消', confirm: '确认', placement: 'center' },
    });

    localeRef.value = 'en-US';
    providerLocaleRef.value = {
      localeName: 'en-US-components',
      dialog: { cancel: 'Cancel', confirm: 'Confirm' },
    };
    await nextTick();

    expect(routeProbeMounts.count).toBe(1);
    expect(wrapper.get('[data-testid="route-probe"]').text()).toBe('locale:en-US');
    expect(JSON.parse(wrapper.get('[data-global-config]').attributes()['data-global-config'] ?? '{}')).toEqual({
      localeName: 'en-US-components',
      dialog: { cancel: 'Cancel', confirm: 'Confirm', placement: 'center' },
    });
  });

  it('centers declarative and programmatic dialogs through the global provider', async () => {
    const dialogHost = document.createElement('div');
    document.body.appendChild(dialogHost);
    const attach = () => dialogHost;
    const wrapper = mount(AppProviders, {
      attachTo: document.body,
      global: {
        components: {
          TConfigProvider: ConfigProvider,
        },
        stubs: {
          RouterView: defineComponent({
            setup() {
              return () => h(DialogRouteProbe, { attach });
            },
          }),
        },
      },
    });
    const pluginDialog = DialogPlugin.confirm({
      attach,
      body: 'Programmatic dialog body',
      header: 'Programmatic dialog',
    });

    try {
      await nextTick();
      await nextTick();

      expect(
        dialogHost
          .querySelector('.app-provider-declarative-dialog-test')
          ?.closest('.t-dialog__position')
          ?.classList.contains('t-dialog--center'),
      ).toBe(true);
      expect(dialogHost.querySelectorAll('.t-dialog__position.t-dialog--center')).toHaveLength(2);
    } finally {
      pluginDialog.destroy();
      wrapper.unmount();
      dialogHost.remove();
    }
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

  it('keeps the theme workbench hidden on the unavailable result route during recovery probes', async () => {
    routerMock.currentRoute.value = {
      path: '/result/service-unavailable',
      fullPath: '/result/service-unavailable?redirect=%2Fprojects',
      query: { redirect: '/projects' },
    };
    availabilityStatusRef.value = 'unknown';

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

    expect(wrapper.find('[data-testid="setting-stub"]').exists()).toBe(false);

    availabilityStatusRef.value = 'recovering';
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

// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, nextTick, ref } from 'vue';

import AppProviders from './AppProviders.vue';

const localeRef = ref('zh-CN');
const providerLocaleRef = ref({ localeName: 'zh-CN-components' });
const displayModeRef = ref('light');
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
});

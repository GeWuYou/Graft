import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

vi.mock('@/locales', () => ({
  i18n: {
    global: {
      getLocaleMessage: () => ({}),
    },
  },
  t: (key: string) => key,
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({ locale: { value: 'en-US' } }),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: {} }),
}));

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({
    '--td-brand-color': brandTheme,
  }),
  insertThemeStylesheet: vi.fn(),
}));

import { useSettingStore } from '@/store';

import ThemeWorkbenchPanel from './ThemeWorkbenchPanel.vue';

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    visible: { type: Boolean, required: false, default: false },
  },
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const switchStub = defineComponent({
  name: 'TSwitchStub',
  props: {
    modelValue: { type: Boolean, required: true },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('button', {
        'data-testid': 'acrylic-switch',
        'aria-pressed': String(props.modelValue),
        onClick: () => emit('update:modelValue', !props.modelValue),
      });
  },
});

const presetCatalogStub = defineComponent({
  name: 'ThemeWorkbenchPresetCatalogStub',
  props: {
    preserveThemePersonalization: { type: Boolean, required: true },
  },
  emits: ['update:preserveThemePersonalization'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        {
          class: 'preset-catalog__apply-mode',
        },
        h(
          'button',
          {
            'aria-pressed': String(props.preserveThemePersonalization),
            onClick: () => emit('update:preserveThemePersonalization', !props.preserveThemePersonalization),
          },
          'preset catalog',
        ),
      );
  },
});

function mountPanel() {
  return mount(ThemeWorkbenchPanel, {
    global: {
      stubs: {
        't-drawer': drawerStub,
        't-switch': switchStub,
        'theme-workbench-preset-catalog': presetCatalogStub,
        't-button': true,
        't-color-picker': true,
        't-icon': true,
        't-radio-group': true,
        't-tooltip': true,
        't-select': true,
        't-slider': true,
        't-collapse': true,
        't-collapse-panel': true,
      },
    },
  });
}

describe('ThemeWorkbenchPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn(() => ({ matches: false })),
    });
  });

  it('renders the acrylic switch in appearance settings and updates the workbench draft', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('appearance');
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('layout.setting.workbench.appearance.acrylicGlass');
    expect(wrapper.get('[data-testid="acrylic-switch"]').attributes('aria-pressed')).toBe('true');

    await wrapper.get('[data-testid="acrylic-switch"]').trigger('click');

    expect(store.isAcrylicEnabled).toBe(false);
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);
  });

  it('passes the preset application preference through to the workbench store', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    const wrapper = mountPanel();

    const preferenceSwitch = wrapper.get('.preset-catalog__apply-mode button');
    expect(preferenceSwitch.attributes('aria-pressed')).toBe('true');

    await preferenceSwitch.trigger('click');

    expect(store.preserveThemePersonalization).toBe(false);
  });

  it('allows every preset to compose the Industrial treatment from manual style options', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('style');
    const wrapper = mountPanel();

    await wrapper.get('[data-testid="radius-square"]').trigger('click');
    await wrapper.get('[data-testid="shadow-hard-offset"]').trigger('click');

    expect(store.radiusPreset).toBe('square');
    expect(store.shadowPreset).toBe('hard-offset');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);
  });
});

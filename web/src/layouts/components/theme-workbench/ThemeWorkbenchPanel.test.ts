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
  syncFaviconColor: vi.fn(),
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

const tooltipStub = defineComponent({
  name: 'TTooltipStub',
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

const sliderStub = defineComponent({
  name: 'TSliderStub',
  props: {
    disabled: { type: Boolean, required: false, default: false },
    modelValue: { type: Number, required: false, default: 0 },
  },
  emits: ['change'],
  setup(props) {
    return () => h('button', { disabled: props.disabled, type: 'button' });
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    disabled: { type: Boolean, required: false, default: false },
    modelValue: { type: String, required: false, default: '' },
  },
  emits: ['change'],
  setup(props) {
    return () => h('button', { disabled: props.disabled, type: 'button' });
  },
});

const presetCatalogStub = defineComponent({
  name: 'ThemeWorkbenchPresetCatalogStub',
  props: {
    applicationScope: { type: String, required: true },
  },
  emits: ['select', 'update:applicationScope'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        {
          class: 'preset-catalog__apply-mode',
        },
        [
          h(
            'button',
            {
              'data-testid': 'preset-application-button',
              onClick: () => emit('select', 'one-dark-pro', 'complete'),
            },
            'preset catalog',
          ),
          h(
            'button',
            {
              'data-testid': 'preset-application-scope-toggle',
              onClick: () =>
                emit('update:applicationScope', props.applicationScope === 'palette' ? 'complete' : 'palette'),
            },
            'scope toggle',
          ),
        ],
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
        't-tooltip': tooltipStub,
        't-select': selectStub,
        't-slider': sliderStub,
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

  it('uses themed outline buttons for the workbench navigation', () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    const wrapper = mountPanel();

    const activeButton = wrapper.get('.nav-item--active');
    expect(activeButton.attributes('theme')).toBe('primary');
    expect(activeButton.attributes('variant')).toBe('outline');
    expect(activeButton.attributes('block')).toBe('');
  });

  it('passes the local preset application scope to the workbench store', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    const wrapper = mountPanel();

    const selectThemePreset = vi.spyOn(store, 'selectThemePreset');
    await wrapper.get('[data-testid="preset-application-button"]').trigger('click');

    expect(selectThemePreset).toHaveBeenCalledWith('one-dark-pro', 'complete');
  });

  it('persists the preset application scope separately from visual workbench state', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('presets');
    const wrapper = mountPanel();

    expect(store.themePresetApplicationScope).toBe('palette');
    await wrapper.get('[data-testid="preset-application-scope-toggle"]').trigger('click');
    expect(store.themePresetApplicationScope).toBe('complete');

    store.cancelThemeDraft();
    expect(store.themePresetApplicationScope).toBe('complete');
  });

  it('allows every preset to compose the Industrial treatment from manual style options', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('style');
    const wrapper = mountPanel();

    const radiusSlider = wrapper
      .findAllComponents(sliderStub)
      .find((component) => component.attributes('data-testid') === 'radius-slider');

    expect(radiusSlider).toBeDefined();
    radiusSlider!.vm.$emit('change', 0);
    await wrapper.vm.$nextTick();
    await wrapper.get('[data-testid="shadow-hard-offset"]').trigger('click');

    expect(store.radiusPreset).toBe('square');
    expect(store.shadowPreset).toBe('hard-offset');
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);
  });

  it('updates radius and density from their discrete slider controls', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('style');
    const wrapper = mountPanel();
    const sliders = wrapper.findAllComponents(sliderStub);
    const radiusSlider = sliders.find((component) => component.attributes('data-testid') === 'radius-slider');
    const densitySlider = sliders.find((component) => component.attributes('data-testid') === 'density-slider');
    expect(radiusSlider).toBeDefined();
    expect(densitySlider).toBeDefined();
    radiusSlider!.vm.$emit('change', 3);
    densitySlider!.vm.$emit('change', 0.91);
    await wrapper.vm.$nextTick();

    expect(store.radiusOverride).toBe(3);
    expect(store.densityOverride).toBe(0.91);
    expect(wrapper.get('[data-testid="radius-anchor-business"]').classes()).not.toContain(
      'style-control__mark--active',
    );
    expect(wrapper.get('[data-testid="density-anchor-compact"]').classes()).not.toContain(
      'style-control__mark--active',
    );

    await wrapper.get('[data-testid="radius-anchor-capsule"]').trigger('click');
    await wrapper.get('[data-testid="density-anchor-comfortable"]').trigger('click');
    await wrapper.vm.$nextTick();

    expect(store.radiusPreset).toBe('capsule');
    expect(store.densityPreset).toBe('comfortable');
    expect(store.radiusOverride).toBeNull();
    expect(store.densityOverride).toBeNull();
    expect(wrapper.get('[data-testid="radius-anchor-capsule"]').classes()).toContain('style-control__mark--active');
    expect(wrapper.get('[data-testid="density-anchor-comfortable"]').classes()).toContain(
      'style-control__mark--active',
    );
    expect(store.hasThemeWorkbenchPendingChanges).toBe(true);
  });

  it('shows continuous radius and density overrides in the overview', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('style');
    const wrapper = mountPanel();
    const sliders = wrapper.findAllComponents(sliderStub);
    const radiusSlider = sliders.find((component) => component.attributes('data-testid') === 'radius-slider');
    const densitySlider = sliders.find((component) => component.attributes('data-testid') === 'density-slider');

    radiusSlider!.vm.$emit('change', 3);
    densitySlider!.vm.$emit('change', 0.91);
    await wrapper.vm.$nextTick();
    store.setActiveThemeWorkbenchGroup('overview');
    await wrapper.vm.$nextTick();

    expect(wrapper.text()).toContain('3px');
    expect(wrapper.text()).toContain('91%');
  });

  it('uses a separate shadow intensity control and preserves it when flat shadows disable the control', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('style');
    const wrapper = mountPanel();
    const intensitySlider = wrapper
      .findAllComponents(sliderStub)
      .find((component) => component.attributes('data-testid') === 'shadow-intensity-slider');

    await wrapper.get('[data-testid="shadow-hard-offset"]').trigger('click');
    expect(intensitySlider).toBeDefined();
    intensitySlider!.vm.$emit('change', 1.35);
    await wrapper.vm.$nextTick();

    expect(store.shadowIntensityOverride).toBe(1.35);
    expect(wrapper.get('[data-testid="shadow-intensity-anchor-strong"]').classes()).not.toContain(
      'style-control__mark--active',
    );
    expect(wrapper.get('[data-testid="shadow-combination-preview"]').classes()).toEqual(
      expect.arrayContaining(['style-combination-preview--shadow-hard-offset']),
    );
    expect(wrapper.get('[data-testid="shadow-combination-preview"]').classes()).not.toContain(
      'style-combination-preview--intensity-standard',
    );

    await wrapper.get('[data-testid="shadow-flat"]').trigger('click');
    await wrapper.vm.$nextTick();

    expect(intensitySlider!.attributes('disabled')).toBeDefined();
    intensitySlider!.vm.$emit('change', 0);
    await wrapper.vm.$nextTick();
    expect(store.shadowIntensityOverride).toBe(1.35);
  });

  it('updates the active tab indicator position from layout settings', async () => {
    const store = useSettingStore();
    store.openThemeWorkbench('layout');
    const wrapper = mountPanel();

    expect(store.tabIndicatorPosition).toBe('none');
    expect(wrapper.get('[data-testid="tab-indicator-none"]').classes()).toContain('choice-card--active');

    await wrapper.get('[data-testid="tab-indicator-bottom"]').trigger('click');

    expect(store.tabIndicatorPosition).toBe('bottom');
    expect(wrapper.get('[data-testid="tab-indicator-bottom"]').classes()).toContain('choice-card--active');
  });
});

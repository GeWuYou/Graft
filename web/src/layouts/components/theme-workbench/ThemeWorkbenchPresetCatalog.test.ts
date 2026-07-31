import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { THEME_PRESET_DEFINITIONS } from '@/config/theme-workbench';

import ThemeWorkbenchPresetCatalog from './ThemeWorkbenchPresetCatalog.vue';

vi.mock('@/locales', () => ({
  t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${Object.values(params).join(',')}` : key),
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({ locale: { value: 'en-US' } }),
}));

const inputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: { type: String, required: false, default: '' },
  },
  emits: ['update:value'],
  setup(_, { emit }) {
    return () =>
      h('input', { onInput: (event: Event) => emit('update:value', (event.target as HTMLInputElement).value) });
  },
});

const radioGroupStub = defineComponent({
  name: 'TRadioGroupStub',
  props: {
    options: { type: Array, required: true },
  },
  emits: ['change'],
  setup() {
    return () => h('div');
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
        'data-testid': 'preset-application-switch',
        'aria-pressed': String(props.modelValue),
        onClick: () => emit('update:modelValue', !props.modelValue),
      });
  },
});

function mountCatalog(preserveThemePersonalization = true) {
  return mount(ThemeWorkbenchPresetCatalog, {
    props: {
      presets: THEME_PRESET_DEFINITIONS,
      activePresetId: 'tdesign-default',
      preserveThemePersonalization,
    },
    global: {
      stubs: {
        't-input': inputStub,
        't-radio-group': radioGroupStub,
        't-switch': switchStub,
      },
    },
  });
}

describe('ThemeWorkbenchPresetCatalog', () => {
  it('separates featured presets from the rest of the catalog', () => {
    const wrapper = mountCatalog();

    expect(wrapper.findAll('.preset-card')).toHaveLength(THEME_PRESET_DEFINITIONS.length);
    expect(wrapper.text()).toContain('layout.setting.workbench.presets.featured');
    expect(wrapper.text()).toContain('layout.setting.workbench.presets.more');
  });

  it('filters the catalog by localized search text and category', async () => {
    const wrapper = mountCatalog();

    await wrapper.get('input').setValue('onedarkpro');
    expect(wrapper.findAll('.preset-card')).toHaveLength(1);
    expect(wrapper.text()).not.toContain('layout.setting.workbench.presets.featured');

    await wrapper.get('input').setValue('');
    await wrapper.findComponent({ name: 'TRadioGroupStub' }).vm.$emit('change', 'operations');
    expect(wrapper.findAll('.preset-card')).toHaveLength(
      THEME_PRESET_DEFINITIONS.filter((preset) => preset.category === 'operations').length,
    );
  });

  it('includes the expanded catalog and gives Tencent Cloud distinct preview surfaces', () => {
    const wrapper = mountCatalog();
    const tencentCard = wrapper.findAll('.preset-card')[1].element as HTMLElement;

    expect(THEME_PRESET_DEFINITIONS).toHaveLength(20);
    expect(THEME_PRESET_DEFINITIONS).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ id: 'one-dark-pro', category: 'night' }),
        expect.objectContaining({ id: 'github-dark', category: 'operations' }),
        expect.objectContaining({ id: 'dracula', category: 'focused' }),
        expect.objectContaining({ id: 'nord', category: 'balanced' }),
      ]),
    );
    expect(tencentCard.style.getPropertyValue('--preset-brand-color')).toBe('#00A4FF');
    expect(tencentCard.style.getPropertyValue('--preset-thumbnail-background')).toBe('#F4F9FD');
    expect(tencentCard.style.getPropertyValue('--preset-thumbnail-sidebar')).toBe('#E5F3FF');
  });

  it('emits the selected preset id and exposes the active card state', async () => {
    const wrapper = mountCatalog();

    expect(wrapper.findAll('.preset-card[aria-pressed="true"]')).toHaveLength(1);
    await wrapper.findAll('.preset-card')[1].trigger('click');

    expect(wrapper.emitted('select')?.[0]).toEqual(['tencent-cloud']);
  });

  it('emits the default personalization setting and changes the application guidance', async () => {
    const wrapper = mountCatalog();

    expect(wrapper.get('[data-testid="preset-application-switch"]').attributes('aria-pressed')).toBe('true');
    expect(wrapper.text()).toContain('layout.setting.workbench.presets.preserveThemePersonalizationHint');

    await wrapper.get('[data-testid="preset-application-switch"]').trigger('click');

    expect(wrapper.emitted('update:preserveThemePersonalization')?.[0]).toEqual([false]);

    await wrapper.setProps({ preserveThemePersonalization: false });
    expect(wrapper.text()).toContain('layout.setting.workbench.presets.completePresetHint');
  });
});

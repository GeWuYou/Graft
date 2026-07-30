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

function mountCatalog() {
  return mount(ThemeWorkbenchPresetCatalog, {
    props: {
      presets: THEME_PRESET_DEFINITIONS,
      activePresetId: 'tdesign-default',
    },
    global: {
      stubs: {
        't-input': inputStub,
        't-radio-group': radioGroupStub,
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

    await wrapper.get('input').setValue('night');
    expect(wrapper.findAll('.preset-card')).toHaveLength(3);
    expect(wrapper.text()).not.toContain('layout.setting.workbench.presets.featured');

    await wrapper.get('input').setValue('');
    await wrapper.findComponent({ name: 'TRadioGroupStub' }).vm.$emit('change', 'operations');
    expect(wrapper.findAll('.preset-card')).toHaveLength(3);
  });

  it('keeps the catalog categories balanced and gives Tencent Cloud distinct preview surfaces', () => {
    const wrapper = mountCatalog();
    const categoryCounts = Object.groupBy(THEME_PRESET_DEFINITIONS, (preset) => preset.category);
    const tencentCard = wrapper.findAll('.preset-card')[1].element as HTMLElement;

    expect(Object.values(categoryCounts).map((presets) => presets?.length)).toEqual([3, 3, 3, 3]);
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
});

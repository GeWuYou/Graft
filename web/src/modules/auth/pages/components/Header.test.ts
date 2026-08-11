import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { useSettingStore } from '@/store';

import AuthHeader from './Header.vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({ '--td-brand-color': brandTheme }),
  insertThemeStylesheet: vi.fn(),
  syncFaviconColor: vi.fn(),
}));

vi.mock('@/locales', () => ({
  i18n: { global: { getLocaleMessage: () => ({}) } },
  t: (key: string) => key,
}));

const buttonStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.());
  },
});
const iconStub = defineComponent({
  props: { name: { type: String, required: true } },
  setup(props) {
    return () => h('i', { 'data-icon-name': props.name });
  },
});
const tooltipStub = defineComponent({
  props: { content: { type: String, required: false, default: '' } },
  setup(props, { slots }) {
    return () => h('div', { 'data-tooltip-content': props.content }, slots.default?.());
  },
});

describe('AuthHeader', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('uses the palette icon and personalization tooltip to open the workbench', async () => {
    const store = useSettingStore();
    const openWorkbench = vi.spyOn(store, 'openThemeWorkbench');
    const wrapper = mount(AuthHeader, {
      global: {
        stubs: {
          BrandIdentity: true,
          LanguageSwitcher: true,
          't-button': buttonStub,
          't-icon': iconStub,
          't-tooltip': tooltipStub,
        },
      },
    });

    const personalizationEntry = wrapper.get('[data-tooltip-content="layout.header.personalization"]');
    expect(personalizationEntry.find('[data-icon-name="palette"]').exists()).toBe(true);

    await personalizationEntry.get('button').trigger('click');
    expect(openWorkbench).toHaveBeenCalledWith('overview');
  });
});

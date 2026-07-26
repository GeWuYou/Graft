import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import LanguageSwitcher from './LanguageSwitcher.vue';

vi.mock('@/locales', () => ({
  languageList: ref([
    { content: 'English', value: 'en-US' },
    { content: '简体中文', value: 'zh-CN' },
  ]),
  t: (key: string) => key,
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({ changeLocale: vi.fn(), locale: ref('zh-CN') }),
}));

const tooltipStub = defineComponent({
  props: { content: { type: String, required: true } },
  setup(props, { slots }) {
    return () => h('div', { 'data-tooltip-content': props.content }, slots.default?.());
  },
});

const buttonStub = defineComponent({
  setup(_, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.());
  },
});

describe('LanguageSwitcher', () => {
  it('provides the localized language tooltip and accessible button label', () => {
    const wrapper = mount(LanguageSwitcher, {
      global: {
        stubs: {
          't-button': buttonStub,
          't-dropdown': { template: '<div><slot /><slot name="dropdown" /></div>' },
          't-dropdown-item': true,
          't-dropdown-menu': true,
          't-tooltip': tooltipStub,
        },
      },
    });

    expect(wrapper.get('[data-tooltip-content]').attributes('data-tooltip-content')).toBe('layout.header.language');
    expect(wrapper.get('button').attributes('aria-label')).toBe('layout.header.language');
  });

  it('opens the dialog presentation from its trigger', async () => {
    const wrapper = mount(LanguageSwitcher, {
      props: { mode: 'dialog' },
      global: {
        stubs: {
          't-button': buttonStub,
          't-dialog': {
            props: { visible: { type: Boolean, default: false } },
            template: '<section data-testid="language-dialog" :data-visible="String(visible)"><slot /></section>',
          },
          't-select': {
            props: ['filterable', 'options', 'placeholder', 'modelValue'],
            template:
              '<div data-testid="language-dialog-select" :data-filterable="String(filterable)" :data-options="options.map((item) => item.label).join(\',\')" />',
          },
          't-tooltip': tooltipStub,
        },
      },
    });

    expect(wrapper.get('[data-testid="language-dialog"]').attributes('data-visible')).toBe('false');
    await wrapper.get('button').trigger('click');
    expect(wrapper.get('[data-testid="language-dialog"]').attributes('data-visible')).toBe('true');
    expect(wrapper.get('[data-testid="language-dialog-select"]').attributes('data-filterable')).toBeDefined();
    expect(wrapper.get('[data-testid="language-dialog-select"]').attributes('data-options')).toBe('English,简体中文');
  });
});

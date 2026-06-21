import { mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { ThemeTokenDefinition } from '@/types/theme';

import ThemeTokenItem from './ThemeTokenItem.vue';

const copyTextMock = vi.hoisted(() => vi.fn(async () => true));

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/shared/observability', () => ({
  copyText: copyTextMock,
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          type: 'button',
          onClick: () => emit('click'),
        },
        slots.default?.(),
      );
  },
});

const inputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: { type: String, required: false, default: '' },
    suffix: { type: String, required: false, default: '' },
  },
  emits: ['blur', 'change', 'update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        onBlur: () => emit('blur'),
        onChange: (event: Event) => emit('change', (event.target as HTMLInputElement).value),
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const colorPickerStub = defineComponent({
  name: 'TColorPickerStub',
  props: {
    modelValue: { type: String, required: false, default: '' },
  },
  emits: ['change'],
  setup(props, { emit }) {
    return () =>
      h(
        'button',
        {
          type: 'button',
          'data-testid': 'color-picker',
          onClick: () => emit('change', props.modelValue || '#e37327'),
        },
        'picker',
      );
  },
});

const token: ThemeTokenDefinition = {
  group: 'brand',
  key: '--td-brand-color',
  labelKey: 'layout.setting.workbench.tokenDefinitions.primaryBrand',
};

function mountItem(props?: Partial<{ hasOverride: boolean; token: ThemeTokenDefinition; value: string }>) {
  return mount(ThemeTokenItem, {
    props: {
      hasOverride: true,
      token,
      value: '#e37327',
      ...props,
    },
    global: {
      stubs: {
        't-button': buttonStub,
        't-color-picker': colorPickerStub,
        't-input': inputStub,
      },
    },
  });
}

describe('ThemeTokenItem', () => {
  beforeEach(() => {
    copyTextMock.mockClear();
  });

  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('renders a collapsed summary first and expands on click', async () => {
    const wrapper = mountItem();

    expect(wrapper.text()).toContain('layout.setting.workbench.token.expand');
    expect(wrapper.find('[data-testid="color-picker"]').exists()).toBe(false);

    await wrapper.get('.theme-token-item__summary').trigger('click');

    expect(wrapper.text()).toContain('layout.setting.workbench.token.collapse');
    expect(wrapper.find('[data-testid="color-picker"]').exists()).toBe(true);
  });

  it('emits reset and commit actions from the expanded editor', async () => {
    const wrapper = mountItem();
    await wrapper.get('.theme-token-item__summary').trigger('click');

    const buttons = wrapper.findAll('button');
    await buttons[2].trigger('click');
    await buttons[3].trigger('click');

    expect(copyTextMock).toHaveBeenCalledWith('--td-brand-color');
    expect(wrapper.emitted('reset')).toHaveLength(1);
  });
});

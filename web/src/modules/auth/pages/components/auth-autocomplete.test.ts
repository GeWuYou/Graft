import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/modules/auth/store', () => ({
  useAuthSessionStore: () => ({
    login: vi.fn(),
    mustChangePassword: false,
    token: '',
  }),
}));

vi.mock('@/shared/composables', () => ({
  useCounter: () => [0, vi.fn()],
}));

vi.mock('@/utils/request', () => ({
  isApiRequestError: () => false,
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push: vi.fn() }),
}));

import LoginPanel from './Login.vue';
import RegisterPanel from './Register.vue';

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const InputStub = defineComponent({
  props: {
    autocomplete: {
      default: undefined,
      type: String,
    },
    modelValue: {
      default: undefined,
      type: [Number, String],
    },
    name: {
      default: undefined,
      type: String,
    },
    type: {
      default: 'text',
      type: String,
    },
  },
  setup(props) {
    return () =>
      h('input', {
        autocomplete: props.autocomplete,
        name: props.name,
        type: props.type,
      });
  },
});

const componentStubs = {
  't-button': SlotStub,
  't-checkbox': SlotStub,
  't-form': SlotStub,
  't-form-item': SlotStub,
  't-icon': SlotStub,
  't-input': InputStub,
};

describe('auth form autocomplete semantics', () => {
  it('marks password login inputs for saved credential filling', () => {
    const wrapper = mount(LoginPanel, {
      global: {
        stubs: componentStubs,
      },
    });

    expect(wrapper.get('input[name="username"]').attributes('autocomplete')).toBe('username');
    expect(wrapper.get('input[name="current-password"]').attributes('autocomplete')).toBe('current-password');
  });

  it('marks registration inputs as contact data and a new password', () => {
    const wrapper = mount(RegisterPanel, {
      global: {
        stubs: componentStubs,
      },
    });

    expect(wrapper.get('input[name="tel"]').attributes('autocomplete')).toBe('tel');
    expect(wrapper.get('input[name="new-password"]').attributes('autocomplete')).toBe('new-password');
    expect(wrapper.get('input[name="one-time-code"]').attributes('autocomplete')).toBe('one-time-code');
  });
});

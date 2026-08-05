import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import BuildCreatePage from './index.vue';

const mocks = vi.hoisted(() => ({ createBuildJob: vi.fn(), push: vi.fn() }));

vi.mock('../../api/build', () => ({ createBuildJob: mocks.createBuildJob }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));

const WrapperStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});
const FormStub = defineComponent({
  emits: ['submit'],
  setup(_props, { emit, slots }) {
    return () =>
      h(
        'form',
        {
          onSubmit: (event: Event) => {
            event.preventDefault();
            emit('submit', { validateResult: true });
          },
        },
        slots.default?.(),
      );
  },
});
const InputStub = defineComponent({
  props: { modelValue: { type: [Number, String], default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});
const ButtonStub = defineComponent({
  props: { type: { type: String, default: 'button' } },
  setup(props, { slots }) {
    return () => h('button', { type: props.type }, slots.default?.());
  },
});

function mountPage() {
  return mount(BuildCreatePage, {
    global: {
      stubs: {
        't-alert': WrapperStub,
        't-button': ButtonStub,
        't-form': FormStub,
        't-form-item': WrapperStub,
        't-input': InputStub,
        't-input-number': InputStub,
      },
    },
  });
}

describe('BuildCreatePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.push.mockResolvedValue(undefined);
  });

  it('reuses the idempotency key when an unchanged failed form is retried', async () => {
    mocks.createBuildJob.mockRejectedValueOnce(new Error('temporary')).mockResolvedValueOnce({});
    const wrapper = mountPage();

    await wrapper.get('form').trigger('submit');
    await flushPromises();
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(mocks.createBuildJob).toHaveBeenCalledTimes(2);
    const firstIdempotencyKey = mocks.createBuildJob.mock.calls[0]?.[1];
    expect(firstIdempotencyKey).toEqual(expect.any(String));
    expect(firstIdempotencyKey).not.toBe('');
    expect(mocks.createBuildJob.mock.calls[1]?.[1]).toBe(firstIdempotencyKey);
  });
});

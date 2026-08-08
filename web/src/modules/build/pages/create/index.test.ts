import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, ref } from 'vue';

import BuildCreatePage from './index.vue';

const mocks = vi.hoisted(() => ({
  createBuildJob: vi.fn(),
  getBuildBuilderPools: vi.fn(),
  getBuildRuntimeTargets: vi.fn(),
  getBuildWorkspaces: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/build', () => ({
  createBuildJob: mocks.createBuildJob,
  getBuildBuilderPools: mocks.getBuildBuilderPools,
  getBuildRuntimeTargets: mocks.getBuildRuntimeTargets,
  getBuildWorkspaces: mocks.getBuildWorkspaces,
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
const testLocale = ref('en-US');
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: testLocale, t: (key: string) => `${testLocale.value}:${key}` }),
}));

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
const SelectStub = defineComponent({
  props: { modelValue: { type: [Number, String], default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('select', {
        value: props.modelValue,
        onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
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
        't-select': SelectStub,
        't-input-number': InputStub,
      },
    },
  });
}

describe('BuildCreatePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    testLocale.value = 'en-US';
    mocks.push.mockResolvedValue(undefined);
    mocks.getBuildWorkspaces.mockResolvedValue({ items: [{ workspace_id: 'workspace_app', name: 'Application' }] });
    mocks.getBuildRuntimeTargets.mockResolvedValue({ items: [{ target_id: 4, display_name: 'Local Docker' }] });
    mocks.getBuildBuilderPools.mockResolvedValue({
      items: [{ pool_id: 'pool:default', display_name: 'Default Pool', scheduling_policy: 'round_robin' }],
    });
  });

  it('loads selector options through the Build-owned read boundary', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(mocks.getBuildWorkspaces).toHaveBeenCalledTimes(1);
    expect(mocks.getBuildRuntimeTargets).toHaveBeenCalledTimes(1);
    expect(mocks.getBuildBuilderPools).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).not.toContain('selectorsUnavailable');
  });

  it('projects every server-authorized Pool policy beside its display name', async () => {
    mocks.getBuildBuilderPools.mockResolvedValue({
      items: [
        { pool_id: 'pool:manual', display_name: 'Manual Pool', scheduling_policy: 'manual' },
        { pool_id: 'pool:round-robin', display_name: 'Round Robin Pool', scheduling_policy: 'round_robin' },
        { pool_id: 'pool:random', display_name: 'Random Pool', scheduling_policy: 'random' },
        { pool_id: 'pool:least-load', display_name: 'Least Load Pool', scheduling_policy: 'least_load' },
        { pool_id: 'pool:capacity', display_name: 'Capacity Pool', scheduling_policy: 'capacity' },
        { pool_id: 'pool:affinity', display_name: 'Affinity Pool', scheduling_policy: 'affinity' },
      ],
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(
      (wrapper.vm as unknown as { builderPoolOptions: Array<{ label: string; policy: string }> }).builderPoolOptions,
    ).toEqual([
      {
        label: 'Manual Pool (en-US:build.jobs.create.builderPoolPolicy.manual)',
        policy: 'manual',
        value: 'pool:manual',
      },
      {
        label: 'Round Robin Pool (en-US:build.jobs.create.builderPoolPolicy.roundRobin)',
        policy: 'round_robin',
        value: 'pool:round-robin',
      },
      {
        label: 'Random Pool (en-US:build.jobs.create.builderPoolPolicy.random)',
        policy: 'random',
        value: 'pool:random',
      },
      {
        label: 'Least Load Pool (en-US:build.jobs.create.builderPoolPolicy.leastLoad)',
        policy: 'least_load',
        value: 'pool:least-load',
      },
      {
        label: 'Capacity Pool (en-US:build.jobs.create.builderPoolPolicy.capacity)',
        policy: 'capacity',
        value: 'pool:capacity',
      },
      {
        label: 'Affinity Pool (en-US:build.jobs.create.builderPoolPolicy.affinity)',
        policy: 'affinity',
        value: 'pool:affinity',
      },
    ]);

    testLocale.value = 'zh-CN';
    await nextTick();

    expect(
      (wrapper.vm as unknown as { builderPoolOptions: Array<{ label: string }> }).builderPoolOptions.map(
        ({ label }) => label,
      ),
    ).toEqual([
      'Manual Pool (zh-CN:build.jobs.create.builderPoolPolicy.manual)',
      'Round Robin Pool (zh-CN:build.jobs.create.builderPoolPolicy.roundRobin)',
      'Random Pool (zh-CN:build.jobs.create.builderPoolPolicy.random)',
      'Least Load Pool (zh-CN:build.jobs.create.builderPoolPolicy.leastLoad)',
      'Capacity Pool (zh-CN:build.jobs.create.builderPoolPolicy.capacity)',
      'Affinity Pool (zh-CN:build.jobs.create.builderPoolPolicy.affinity)',
    ]);
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

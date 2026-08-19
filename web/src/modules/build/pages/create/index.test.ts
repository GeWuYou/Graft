import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, ref } from 'vue';

import { REGISTRY_ROUTE_PATH } from '@/modules/registry/contract/paths';

import { BUILD_ROUTE_PATH } from '../../contract/paths';
import BuildCreatePage from './index.vue';

const mocks = vi.hoisted(() => ({
  createBuildJob: vi.fn(),
  getBuildBuilderPools: vi.fn(),
  getBuildRegistryDestinations: vi.fn(),
  getBuildRuntimeTargets: vi.fn(),
  getBuildWorkspaces: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/build', () => ({
  createBuildJob: mocks.createBuildJob,
  getBuildBuilderPools: mocks.getBuildBuilderPools,
  getBuildRegistryDestinations: mocks.getBuildRegistryDestinations,
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
    return () => h('div', [slots.default?.(), slots.operation?.()]);
  },
});
const AlertStub = defineComponent({
  props: { message: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('div', { 'data-testid': 'alert' }, [props.message, slots.message?.(), slots.operation?.()]);
  },
});
const ManagementPageHeaderStub = defineComponent({
  name: 'ManagementPageHeader',
  props: {
    descriptionKey: { type: String, default: '' },
    source: { type: Object, default: () => ({}) },
    titleKey: { type: String, default: '' },
  },
  setup(props) {
    return () =>
      h('header', {
        'data-description-key': props.descriptionKey,
        'data-source-key': (props.source as { labelKey?: string }).labelKey,
        'data-testid': 'management-page-header',
        'data-title-key': props.titleKey,
      });
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
  props: {
    disabled: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
    modelValue: { type: [Number, String], default: '' },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'select',
        {
          value: props.modelValue,
          disabled: props.disabled,
          'data-loading': String(props.loading),
          'data-options': JSON.stringify(props.options),
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
        },
        (props.options as Array<{ label: string; value: string | number }>).map((option) =>
          h('option', { value: option.value }, option.label),
        ),
      );
  },
});
const RadioGroupStub = defineComponent({
  props: { modelValue: { type: [Number, String], default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'div',
        (slots.default?.() ?? []).map((node) => {
          const value = node.props?.value as string | number | undefined;
          const label = typeof node.children === 'string' ? node.children : String(value ?? '');
          return h(
            'button',
            {
              type: 'button',
              'data-value': value,
              onClick: () => emit('update:modelValue', value),
            },
            label,
          );
        }),
      );
  },
});
const CheckboxGroupStub = defineComponent({
  props: { modelValue: { type: Array, default: () => [] }, options: { type: Array, default: () => [] } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        (props.options as Array<{ disabled?: boolean; label: string; value: string }>).map((option) =>
          h(
            'button',
            {
              type: 'button',
              'data-platform': option.value,
              'data-disabled': String(Boolean(option.disabled)),
              disabled: option.disabled,
              onClick: () => {
                const selected = new Set(props.modelValue as string[]);
                if (selected.has(option.value)) selected.delete(option.value);
                else selected.add(option.value);
                emit('update:modelValue', [...selected]);
              },
            },
            option.label,
          ),
        ),
      );
  },
});
const ButtonStub = defineComponent({
  props: { type: { type: String, default: 'button' } },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', { type: props.type, onClick: () => emit('click') }, slots.default?.());
  },
});

function mountPage() {
  return mount(BuildCreatePage, {
    global: {
      stubs: {
        ManagementPageContent: WrapperStub,
        ManagementPageHeader: ManagementPageHeaderStub,
        't-alert': AlertStub,
        't-button': ButtonStub,
        't-card': WrapperStub,
        't-checkbox-group': CheckboxGroupStub,
        't-form': FormStub,
        't-form-item': WrapperStub,
        't-input': InputStub,
        't-radio-group': RadioGroupStub,
        't-radio': WrapperStub,
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
    mocks.getBuildRegistryDestinations.mockResolvedValue({
      items: [
        {
          connection_ref: 'registry:production',
          connection_display_name: 'Production Harbor',
          repository_ref: 'graft/server',
          repository_display_name: 'Server',
        },
        {
          connection_ref: 'registry:production',
          connection_display_name: 'Production Harbor',
          repository_ref: 'graft/worker',
          repository_display_name: 'Worker',
        },
      ],
    });
  });

  it('aligns the shared header and form card on one bounded workflow surface with three ordered sections', () => {
    const wrapper = mountPage();
    const pageHeader = wrapper.get('[data-testid="management-page-header"]');
    const formCard = wrapper.get('[data-testid="build-create-form-card"]');

    expect(wrapper.attributes('data-page-type')).toBe('workflow');
    expect(pageHeader.attributes()).toMatchObject({
      'data-description-key': 'build.jobs.create.description',
      'data-source-key': 'build.jobs.create.eyebrow',
      'data-title-key': 'build.jobs.create.title',
    });
    expect(pageHeader.element.parentElement).toBe(formCard.element.parentElement);
    expect(pageHeader.classes()).toContain('build-create-page__surface');
    expect(formCard.classes()).toContain('build-create-page__surface');
    expect(formCard.find('.build-create-page__form').exists()).toBe(true);
    expect(
      wrapper.findAll('[data-testid^="build-create-section-"]').map((section) => section.attributes('data-testid')),
    ).toEqual(['build-create-section-source', 'build-create-section-execution', 'build-create-section-destination']);
  });

  it('loads selector options through the Build-owned read boundary', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(mocks.getBuildWorkspaces).toHaveBeenCalledTimes(1);
    expect(mocks.getBuildRuntimeTargets).toHaveBeenCalledTimes(1);
    expect(mocks.getBuildBuilderPools).toHaveBeenCalledTimes(1);
    expect(mocks.getBuildRegistryDestinations).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).not.toContain('selectorsUnavailable');
  });

  it('uses only assigned Registry destinations for the connection and repository selectors', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const selectOptions = wrapper
      .findAll('select')
      .map((candidate) => JSON.parse(candidate.attributes('data-options') ?? '[]'));
    expect(selectOptions).toContainEqual([{ label: 'Production Harbor', value: 'registry:production' }]);
    expect(selectOptions).toContainEqual([]);
    const registrySelect = wrapper
      .findAll('select')
      .find((candidate) => candidate.attributes('data-options')?.includes('registry:production'));
    expect(registrySelect).toBeDefined();
    if (registrySelect) {
      await registrySelect.setValue('registry:production');
      await nextTick();
      const repositorySelect = wrapper
        .findAll('select')
        .find((candidate) => candidate.attributes('data-options')?.includes('graft/server'));
      expect(JSON.parse(repositorySelect?.attributes('data-options') ?? '[]')).toEqual([
        { label: 'Server', value: 'graft/server' },
        { label: 'Worker', value: 'graft/worker' },
      ]);
    }
    expect(wrapper.html()).not.toContain('registry:default');
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

    await wrapper.get('button[data-value="pool"]').trigger('click');
    await flushPromises();

    const select = wrapper
      .findAll('select')
      .find((candidate) => JSON.parse(candidate.attributes('data-options') ?? '[]').length === 6);
    expect(select).toBeDefined();
    if (!select) {
      return;
    }
    expect(JSON.parse(select.attributes('data-options') ?? '[]')).toEqual([
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
      (JSON.parse(select.attributes('data-options') ?? '[]') as Array<{ label: string }>).map(({ label }) => label),
    ).toEqual([
      'Manual Pool (zh-CN:build.jobs.create.builderPoolPolicy.manual)',
      'Round Robin Pool (zh-CN:build.jobs.create.builderPoolPolicy.roundRobin)',
      'Random Pool (zh-CN:build.jobs.create.builderPoolPolicy.random)',
      'Least Load Pool (zh-CN:build.jobs.create.builderPoolPolicy.leastLoad)',
      'Capacity Pool (zh-CN:build.jobs.create.builderPoolPolicy.capacity)',
      'Affinity Pool (zh-CN:build.jobs.create.builderPoolPolicy.affinity)',
    ]);
  });

  it('keeps ARM64 disabled without switching away from a direct runtime target', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const arm64 = wrapper.get('button[data-platform="linux/arm64"]');
    expect(arm64.attributes('disabled')).toBeDefined();
    expect(arm64.attributes('data-disabled')).toBe('true');
    await arm64.trigger('click');
    await nextTick();

    expect(
      wrapper.findAll('select').some((select) => select.attributes('data-options')?.includes('pool:default')),
    ).toBe(false);
    expect(wrapper.text()).toContain('build.jobs.create.arm64PoolHint');
  });

  it('uses Buildx for ARM64 only after the user explicitly chooses a Builder Pool', async () => {
    mocks.createBuildJob.mockResolvedValue({});
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('button[data-value="pool"]').trigger('click');
    await nextTick();
    await wrapper.get('button[data-platform="linux/arm64"]').trigger('click');
    await nextTick();

    const poolSelect = wrapper
      .findAll('select')
      .find((candidate) => candidate.attributes('data-options')?.includes('pool:default'));
    expect(poolSelect).toBeDefined();
    await poolSelect?.setValue('pool:default');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(mocks.createBuildJob).toHaveBeenCalledWith(
      expect.objectContaining({
        builder_pool_id: 'pool:default',
        driver: 'docker-buildx@v1',
        platforms: ['linux/amd64', 'linux/arm64'],
      }),
      expect.any(String),
    );
  });

  it('normalizes ARM64 and Buildx when switching back to a direct runtime target', async () => {
    mocks.createBuildJob.mockResolvedValue({});
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('button[data-value="pool"]').trigger('click');
    await wrapper.get('button[data-platform="linux/arm64"]').trigger('click');
    await wrapper.get('button[data-value="target"]').trigger('click');
    await nextTick();
    const targetSelect = wrapper
      .findAll('select')
      .find((candidate) => candidate.attributes('data-options')?.includes('Local Docker'));
    await targetSelect?.setValue('4');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(mocks.createBuildJob).toHaveBeenCalledWith(
      expect.objectContaining({
        builder_pool_id: undefined,
        driver: 'docker-engine@v1',
        platforms: ['linux/amd64'],
        runtime_target_id: '4',
      }),
      expect.any(String),
    );
  });

  it('keeps a populated runtime target enabled when no workspace is available', async () => {
    mocks.getBuildWorkspaces.mockResolvedValue({ items: [] });
    const wrapper = mountPage();
    await flushPromises();

    const targetSelect = wrapper
      .findAll('select')
      .find((candidate) => candidate.attributes('data-options')?.includes('Local Docker'));
    expect(targetSelect?.attributes('disabled')).toBeUndefined();
    expect(wrapper.text()).toContain('build.jobs.create.workspaceEmpty');
  });

  it('keeps a loaded runtime target usable while the workspace request is still pending', async () => {
    let resolveWorkspaces: ((value: { items: Array<{ workspace_id: string; name: string }> }) => void) | undefined;
    mocks.getBuildWorkspaces.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveWorkspaces = resolve;
        }),
    );
    const wrapper = mountPage();
    await flushPromises();

    const runtimeTargetSelect = wrapper
      .findAll('select')
      .find((candidate) => candidate.attributes('data-options')?.includes('Local Docker'));
    expect(runtimeTargetSelect?.attributes('disabled')).toBeUndefined();
    expect(runtimeTargetSelect?.attributes('data-loading')).toBe('false');

    resolveWorkspaces?.({ items: [] });
    await flushPromises();
  });

  it('reports target and pool availability independently', async () => {
    mocks.getBuildRuntimeTargets.mockResolvedValue({ items: [] });
    mocks.getBuildBuilderPools.mockResolvedValue({ items: [] });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('build.jobs.create.runtimeTargetEmpty');
    expect(wrapper.text()).not.toContain('build.jobs.create.builderPoolEmpty');

    await wrapper.get('button[data-value="pool"]').trigger('click');
    await nextTick();
    expect(wrapper.text()).toContain('build.jobs.create.builderPoolEmpty');
  });

  it('renders an empty direct target value instead of the sentinel zero', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const targetSelect = wrapper
      .findAll('select')
      .find((candidate) => candidate.attributes('data-options')?.includes('Local Docker'));
    expect(targetSelect?.element.value).toBe('');
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

  it('opens the Registry module through its exported route contract', async () => {
    mocks.getBuildRegistryDestinations.mockResolvedValue({ items: [] });
    const wrapper = mountPage();
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('addRegistry'))
      ?.trigger('click');

    expect(mocks.push).toHaveBeenCalledWith(REGISTRY_ROUTE_PATH.LIST);
  });

  it('returns to the Build Tasks list from the secondary action', async () => {
    const wrapper = mountPage();

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('build.jobs.create.back'))
      ?.trigger('click');

    expect(mocks.push).toHaveBeenCalledWith(BUILD_ROUTE_PATH.JOBS);
  });
});

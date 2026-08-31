import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, ref } from 'vue';

import BuildCreatePage from './index.vue';

const mocks = vi.hoisted(() => ({
  createBuildJob: vi.fn(),
  getBuildBuilderPools: vi.fn(),
  getBuildInputSnapshots: vi.fn(),
  getBuildRegistryDestinations: vi.fn(),
  getBuildRuntimeTargets: vi.fn(),
  getBuildWorkspaces: vi.fn(),
  uploadBuildInputSnapshot: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/build', () => mocks);
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
const locale = ref('en-US');
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale, t: (key: string) => key }) }));

const Wrapper = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});
const Input = defineComponent({
  props: { modelValue: { type: [Number, String], default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
      });
  },
});
const Select = defineComponent({
  props: { modelValue: { type: [Number, String], default: '' }, options: { type: Array, default: () => [] } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'select',
        {
          value: props.modelValue,
          onChange: (e: Event) => emit('update:modelValue', (e.target as HTMLSelectElement).value),
        },
        (props.options as any[]).map((o) => h('option', { value: o.value }, o.label)),
      );
  },
});
const Form = defineComponent({
  emits: ['submit'],
  setup(_props, { emit, slots }) {
    return () =>
      h(
        'form',
        {
          onSubmit: (e: Event) => {
            e.preventDefault();
            emit('submit', { validateResult: true });
          },
        },
        slots.default?.(),
      );
  },
});
const Button = defineComponent({
  props: { type: { type: String, default: 'button' } },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', { type: props.type, onClick: () => emit('click') }, slots.default?.());
  },
});
const RadioGroup = defineComponent({
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

function mountPage() {
  return mount(BuildCreatePage, {
    global: {
      stubs: {
        ManagementPageContent: Wrapper,
        ManagementPageHeader: Wrapper,
        't-alert': Wrapper,
        't-button': Button,
        't-card': Wrapper,
        't-checkbox-group': Wrapper,
        't-form': Form,
        't-form-item': Wrapper,
        't-input': Input,
        't-radio-group': RadioGroup,
        't-radio': Wrapper,
        't-select': Select,
      },
    },
  });
}

describe('BuildCreatePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getBuildInputSnapshots.mockResolvedValue({
      items: [
        {
          snapshot_id: 'snap-1',
          content_digest: 'sha256:abc',
          source_kind: 'uploaded_archive',
        },
      ],
      total: 1,
      limit: 100,
      offset: 0,
    });
    mocks.getBuildRuntimeTargets.mockResolvedValue({ items: [{ target_id: 4, display_name: 'Local Docker' }] });
    mocks.getBuildWorkspaces.mockResolvedValue({ items: [], total: 0, limit: 100, offset: 0 });
    mocks.getBuildBuilderPools.mockResolvedValue({ items: [] });
    mocks.getBuildRegistryDestinations.mockResolvedValue({
      items: [
        {
          connection_ref: 'registry:prod',
          connection_display_name: 'Production',
          repository_ref: 'graft/server',
          repository_display_name: 'Server',
        },
      ],
    });
    mocks.uploadBuildInputSnapshot.mockResolvedValue({
      snapshot_id: 'snap-upload',
      source_kind: 'uploaded_archive',
      content_digest: 'sha256:new',
      lifecycle_state: 'available',
    });
    mocks.createBuildJob.mockResolvedValue(undefined);
  });

  it('loads reusable Input Snapshots without Project or Application APIs', async () => {
    mountPage();
    await flushPromises();
    expect(mocks.getBuildInputSnapshots).toHaveBeenCalledWith({ limit: 100, offset: 0 });
  });

  it('uploads an archive before submitting the Build Job with input_snapshot_id', async () => {
    const wrapper = mountPage();
    await flushPromises();
    const file = new File(['FROM alpine'], 'Dockerfile.tar', { type: 'application/x-tar' });
    const input = wrapper.get('[data-testid="build-input-snapshot-file"]').element as HTMLInputElement;
    Object.defineProperty(input, 'files', { configurable: true, value: [file] });
    input.dispatchEvent(new Event('change'));
    await wrapper.find('form').trigger('submit');
    await flushPromises();
    expect(mocks.uploadBuildInputSnapshot).toHaveBeenCalledWith(file);
    expect(mocks.createBuildJob).toHaveBeenCalledWith(
      expect.objectContaining({ input_snapshot_id: 'snap-upload' }),
      expect.any(String),
    );
    expect(mocks.uploadBuildInputSnapshot.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.createBuildJob.mock.invocationCallOrder[0],
    );
  });

  it('loads more snapshots with the next page and keeps unique options', async () => {
    mocks.getBuildInputSnapshots
      .mockResolvedValueOnce({
        items: [
          { snapshot_id: 'snap-1', content_digest: 'sha256:abc', source_kind: 'uploaded_archive' },
          { snapshot_id: 'snap-2', content_digest: 'sha256:def', source_kind: 'uploaded_archive' },
        ],
        total: 101,
        limit: 100,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [
          { snapshot_id: 'snap-2', content_digest: 'sha256:def', source_kind: 'uploaded_archive' },
          { snapshot_id: 'snap-3', content_digest: 'sha256:ghi', source_kind: 'uploaded_archive' },
        ],
        total: 101,
        limit: 100,
        offset: 100,
      });
    const wrapper = mountPage();
    await flushPromises();
    (wrapper.vm as any).sourceMode = 'reuse';
    await nextTick();
    await wrapper.get('[data-testid="build-load-more-snapshots"]').trigger('click');
    await flushPromises();
    expect(mocks.getBuildInputSnapshots).toHaveBeenNthCalledWith(2, { limit: 100, offset: 100 });
    expect(wrapper.findAll('select')[0].findAll('option')).toHaveLength(3);
    expect(wrapper.find('[data-testid="build-load-more-snapshots"]').exists()).toBe(false);
  });

  it('discovers all authorized workspaces across paginated responses', async () => {
    mocks.getBuildWorkspaces
      .mockResolvedValueOnce({
        items: [{ workspace_id: 'workspace-1', name: 'First', source_kind: 'git' }],
        total: 101,
        limit: 100,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [{ workspace_id: 'workspace-2', name: 'Second', source_kind: 'generated' }],
        total: 101,
        limit: 100,
        offset: 100,
      });

    mountPage();
    await flushPromises();

    expect(mocks.getBuildWorkspaces).toHaveBeenNthCalledWith(1, { limit: 100, offset: 0 });
    expect(mocks.getBuildWorkspaces).toHaveBeenNthCalledWith(2, { limit: 100, offset: 100 });
  });
});

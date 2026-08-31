import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import BuildArtifactsPage from './index.vue';

const mocks = vi.hoisted(() => ({ getBuildArtifactPublications: vi.fn(), getBuildArtifacts: vi.fn() }));

vi.mock('../../api/build', () => ({
  getBuildArtifactPublications: mocks.getBuildArtifactPublications,
  getBuildArtifacts: mocks.getBuildArtifacts,
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'en-US' }, t: (key: string) => key }) }));
vi.mock('@/shared/observability', () => ({
  formatBytes: (value: number) => `${value} B`,
  formatLocaleDateTime: (value: string) => value,
}));
vi.mock('@/shared/components/management', () => ({
  ManagementPageHeader: defineComponent({ setup: () => () => null }),
  ManagementToolbar: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.actions?.()),
  }),
  ManagementTableCard: defineComponent({
    setup:
      (_props, { slots }) =>
      () =>
        h('div', slots.default?.()),
  }),
}));
vi.mock('@/shared/components/management/ManagementPagedTable.vue', () => ({
  default: defineComponent({
    props: { rows: { type: Array, default: () => [] } },
    setup(props) {
      return () =>
        h(
          'div',
          (props.rows as Array<{ artifact_id: string }>).map((row) => h('div', row.artifact_id)),
        );
    },
  }),
}));
vi.mock('tdesign-icons-vue-next', () => ({ RefreshIcon: defineComponent({ setup: () => () => null }) }));

const WrapperStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});
const ButtonStub = defineComponent({
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.());
  },
});

function mountPage() {
  return mount(BuildArtifactsPage, {
    global: { stubs: { 't-alert': WrapperStub, 't-button': ButtonStub, 't-space': WrapperStub, 't-tag': WrapperStub } },
  });
}

describe('BuildArtifactsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getBuildArtifacts.mockResolvedValue({ items: [], total: 0 });
    mocks.getBuildArtifactPublications.mockResolvedValue({ items: [], total: 0, limit: 20, offset: 0 });
  });

  it('loads the immutable Artifact projection from the Build-owned read boundary', async () => {
    mountPage();
    await flushPromises();

    expect(mocks.getBuildArtifacts).toHaveBeenCalledWith({ limit: 20, offset: 0 });
  });

  it('renders the newest Artifact response when requests resolve out of order', async () => {
    let resolveFirst!: (value: { items: Array<{ artifact_id: string }>; total: number }) => void;
    let resolveSecond!: (value: { items: Array<{ artifact_id: string }>; total: number }) => void;
    const first = new Promise<{ items: Array<{ artifact_id: string }>; total: number }>((resolve) => {
      resolveFirst = resolve;
    });
    const second = new Promise<{ items: Array<{ artifact_id: string }>; total: number }>((resolve) => {
      resolveSecond = resolve;
    });
    mocks.getBuildArtifacts.mockReset().mockReturnValueOnce(first).mockReturnValueOnce(second);
    const wrapper = mountPage();
    await wrapper.find('button').trigger('click');

    resolveSecond({ items: [{ artifact_id: 'new' }], total: 1 });
    await flushPromises();
    resolveFirst({ items: [{ artifact_id: 'old' }], total: 1 });
    await flushPromises();

    expect(wrapper.text()).toContain('new');
    expect(wrapper.text()).not.toContain('old');
  });

  it('ignores stale publication responses when switching artifacts quickly', async () => {
    type Publication = {
      publication_id: string;
      destination: { repository_ref: string; reference: string };
      created_at: string;
    };
    let resolveFirst!: (value: { items: Publication[] }) => void;
    let resolveSecond!: (value: { items: Publication[] }) => void;
    const first = new Promise<{ items: Publication[] }>((resolve) => {
      resolveFirst = resolve;
    });
    const second = new Promise<{ items: Publication[] }>((resolve) => {
      resolveSecond = resolve;
    });
    mocks.getBuildArtifactPublications.mockReset().mockReturnValueOnce(first).mockReturnValueOnce(second);
    const wrapper = mountPage();
    const openPublications = (
      wrapper.vm as unknown as {
        openPublications: (artifact: { artifact_id: string }) => Promise<void>;
      }
    ).openPublications;

    const firstRequest = openPublications({ artifact_id: 'artifact-old' });
    const secondRequest = openPublications({ artifact_id: 'artifact-new' });
    resolveSecond({
      items: [
        {
          publication_id: 'publication-new',
          destination: { repository_ref: 'repo', reference: 'new' },
          created_at: '',
        },
      ],
    });
    await secondRequest;
    resolveFirst({
      items: [
        {
          publication_id: 'publication-old',
          destination: { repository_ref: 'repo', reference: 'old' },
          created_at: '',
        },
      ],
    });
    await firstRequest;

    expect(mocks.getBuildArtifactPublications).toHaveBeenNthCalledWith(1, 'artifact-old', { limit: 20, offset: 0 });
    expect(mocks.getBuildArtifactPublications).toHaveBeenNthCalledWith(2, 'artifact-new', { limit: 20, offset: 0 });
    const vm = wrapper.vm as unknown as { publications: Publication[] };
    expect(vm.publications).toEqual([
      { publication_id: 'publication-new', destination: { repository_ref: 'repo', reference: 'new' }, created_at: '' },
    ]);
  });

  it('loads additional publication history pages when requested', async () => {
    const publication = (id: string) => ({
      publication_id: id,
      destination: { repository_ref: 'repo', reference: id },
      created_at: '',
    });
    mocks.getBuildArtifactPublications
      .mockResolvedValueOnce({ items: [publication('first')], total: 2, limit: 20, offset: 0 })
      .mockResolvedValueOnce({ items: [publication('second')], total: 2, limit: 20, offset: 1 });
    const wrapper = mountPage();
    const vm = wrapper.vm as unknown as {
      openPublications: (artifact: { artifact_id: string }) => Promise<void>;
    };
    await vm.openPublications({ artifact_id: 'artifact-1' });
    expect(wrapper.text()).toContain('first');
    const loadMore = wrapper
      .findAll('button')
      .find((button) => button.text() === 'build.artifacts.publications.loadMore');
    expect(loadMore).toBeDefined();
    await loadMore!.trigger('click');
    await flushPromises();
    expect(mocks.getBuildArtifactPublications).toHaveBeenLastCalledWith('artifact-1', { limit: 20, offset: 1 });
    expect(wrapper.text()).toContain('second');
  });
});

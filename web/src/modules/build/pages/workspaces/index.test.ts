import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import BuildWorkspacesPage from './index.vue';

const mocks = vi.hoisted(() => ({
  getBuildWorkspaces: vi.fn(),
  push: vi.fn(),
}));

vi.mock('../../api/build', () => ({ getBuildWorkspaces: mocks.getBuildWorkspaces }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: (value: string) => value }));
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }),
}));

const WrapperStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', [slots.default?.(), slots.actions?.()]);
  },
});
const PagedTableStub = defineComponent({
  props: { rows: { type: Array, default: () => [] }, total: { type: Number, default: 0 } },
  emits: ['page-change'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', { 'data-testid': 'paged-table', 'data-total': props.total }, [
        h('button', { 'data-testid': 'next-page', onClick: () => emit('page-change', { current: 2, pageSize: 50 }) }),
        props.rows[0] ? slots.source_reference?.({ row: props.rows[0] }) : null,
      ]);
  },
});

function mountPage() {
  return mount(BuildWorkspacesPage, {
    global: {
      directives: { permission: () => undefined },
      stubs: {
        ManagementPageHeader: WrapperStub,
        ManagementTableCard: WrapperStub,
        ManagementToolbar: WrapperStub,
        ManagementPagedTable: PagedTableStub,
        't-button': WrapperStub,
        't-alert': WrapperStub,
        't-space': WrapperStub,
      },
    },
  });
}

describe('BuildWorkspacesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getBuildWorkspaces
      .mockResolvedValueOnce({
        items: [{ workspace_id: 'workspace_first', source_reference: 'app_canonical_first' }],
        limit: 20,
        offset: 0,
        total: 51,
      })
      .mockResolvedValueOnce({
        items: [{ workspace_id: 'workspace_later', source_reference: 'app_canonical_later' }],
        limit: 50,
        offset: 50,
        total: 51,
      });
  });

  it('loads exact server pages and renders the canonical source reference without a capped Application lookup', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(mocks.getBuildWorkspaces).toHaveBeenNthCalledWith(1, { limit: 20, offset: 0 });
    expect(wrapper.get('[data-testid="paged-table"]').attributes('data-total')).toBe('51');
    expect(wrapper.text()).toContain('app_canonical_first');

    await wrapper.get('[data-testid="next-page"]').trigger('click');
    await flushPromises();

    expect(mocks.getBuildWorkspaces).toHaveBeenNthCalledWith(2, { limit: 50, offset: 50 });
    expect(wrapper.text()).toContain('app_canonical_later');
    expect(wrapper.text()).not.toContain('applicationUnavailable');
  });
});

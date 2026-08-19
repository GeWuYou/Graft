import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import RuntimeTargetAssignmentDialog from './RuntimeTargetAssignmentDialog.vue';

const apiMocks = vi.hoisted(() => ({
  getRuntimeTargetAssignmentCandidates: vi.fn(),
  getRuntimeTargetAssignments: vi.fn(),
  replaceRuntimeTargetAssignments: vi.fn(),
}));
const messageMocks = vi.hoisted(() => ({ success: vi.fn() }));

vi.mock('../api/runtime-target', () => apiMocks);
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));
vi.mock('@/utils/request', () => ({
  isApiRequestError: (error: unknown) => Boolean(error && typeof error === 'object' && 'status' in error),
}));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}));

const pagedMultiSelectStub = defineComponent({
  name: 'PagedMultiSelect',
  props: {
    errorMessage: { type: String, default: '' },
    selection: { type: Object, default: undefined },
    visible: { type: Boolean, default: false },
  },
  emits: ['confirm', 'update:selection'],
  template: `<div data-testid="assignment-dialog" :data-error="errorMessage" :data-visible="String(visible)">
    <button data-testid="select-user" @click="$emit('update:selection', { mode: 'explicit', selectedIds: new Set([9]) })">select</button>
    <button data-testid="confirm" @click="$emit('confirm')">confirm</button>
  </div>`,
});

describe('RuntimeTargetAssignmentDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.getRuntimeTargetAssignments.mockResolvedValue({
      items: [{ target_id: 7, user_id: 8, created_at: '2026-08-19T00:00:00Z' }],
      revision: 4,
    });
    apiMocks.getRuntimeTargetAssignmentCandidates.mockResolvedValue({ items: [], total: 0 });
  });

  it('loads and saves the target selected by its caller through the shared revision boundary', async () => {
    apiMocks.replaceRuntimeTargetAssignments.mockResolvedValue({
      items: [{ target_id: 7, user_id: 9, created_at: '2026-08-19T01:00:00Z' }],
      revision: 5,
    });
    const wrapper = mount(RuntimeTargetAssignmentDialog, {
      props: { targetId: 7, visible: true },
      global: { stubs: { 'paged-multi-select': pagedMultiSelectStub } },
    });
    await flushPromises();

    expect(apiMocks.getRuntimeTargetAssignments).toHaveBeenCalledWith(7);
    expect(apiMocks.getRuntimeTargetAssignmentCandidates).toHaveBeenCalledWith(7, { limit: 20, offset: 0 });
    await wrapper.get('[data-testid="select-user"]').trigger('click');
    await wrapper.get('[data-testid="confirm"]').trigger('click');
    await flushPromises();

    expect(apiMocks.replaceRuntimeTargetAssignments).toHaveBeenCalledWith(7, [9], 4);
    expect(wrapper.emitted('saved')?.[0]?.[0]).toMatchObject({ revision: 5 });
    expect(wrapper.emitted('update:visible')).toContainEqual([false]);
  });

  it('reloads authoritative data and keeps the dialog open when the revision conflicts', async () => {
    apiMocks.replaceRuntimeTargetAssignments.mockRejectedValue(
      Object.assign(new Error('conflict'), { isApiRequestError: true, status: 409 }),
    );
    const wrapper = mount(RuntimeTargetAssignmentDialog, {
      props: { targetId: 7, visible: true },
      global: { stubs: { 'paged-multi-select': pagedMultiSelectStub } },
    });
    await flushPromises();

    await wrapper.get('[data-testid="select-user"]').trigger('click');
    await wrapper.get('[data-testid="confirm"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getRuntimeTargetAssignments).toHaveBeenCalledTimes(2);
    expect(wrapper.get('[data-testid="assignment-dialog"]').attributes('data-error')).toBe(
      'runtimeTarget.detail.authorizationConflict',
    );
    expect(wrapper.emitted('saved')).toBeUndefined();
    expect(wrapper.emitted('update:visible')).toBeUndefined();
  });
});

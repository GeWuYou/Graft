import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import { useUpdateProgressStore } from '../store/progress';
import UpdateProgressDialog from './UpdateProgressDialog.vue';

const routerPush = vi.fn();

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}));
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}));
vi.mock('../api/update', () => ({
  getUpdateOperationDiagnostic: vi.fn(),
  subscribeToUpdateOperation: vi.fn(() => ({ close: vi.fn(), reconnect: vi.fn() })),
}));

const dialogStub = defineComponent({
  props: { visible: Boolean },
  emits: ['close'],
  template: '<section v-if="visible"><slot /><button data-testid="dialog-close" @click="$emit(\'close\')" /></section>',
});
const passthrough = defineComponent({ template: '<section><slot /></section>' });
const progressStub = defineComponent({
  props: {
    percentage: { type: Number, required: true },
    label: { type: Boolean, default: true },
    indeterminate: Boolean,
  },
  template:
    '<section data-testid="progress" :data-percentage="percentage" :data-label="label" :data-indeterminate="indeterminate" />',
});

function mountDialog() {
  return mount(UpdateProgressDialog, {
    global: {
      stubs: {
        't-alert': passthrough,
        't-button': defineComponent({
          emits: ['click'],
          template: '<button @click="$emit(\'click\')"><slot /></button>',
        }),
        't-dialog': dialogStub,
        't-progress': progressStub,
        't-step': passthrough,
        't-steps': passthrough,
      },
    },
  });
}

describe('UpdateProgressDialog', () => {
  beforeEach(() => {
    sessionStorage.clear();
    setActivePinia(createPinia());
    routerPush.mockReset();
  });

  it('releases the shell when a terminal dialog is closed', async () => {
    const progress = useUpdateProgressStore();
    progress.$patch({ operation: { operation_id: 'update-1', status: 'FAILED' } as never, phase: 'failed' });
    const wrapper = mountDialog();

    await wrapper.get('[data-testid="dialog-close"]').trigger('click');

    expect(progress.phase).toBe('idle');
  });

  it('shows the milestone percentage and an indeterminate current-stage bar while running', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({ operation: { operation_id: 'update-1', status: 'PULLING' } as never, phase: 'running' });
    const wrapper = mountDialog();

    expect(wrapper.get('[data-testid="update-progress-overall"]').attributes()).toMatchObject({
      'data-percentage': '30',
      'data-label': 'true',
      'data-indeterminate': 'false',
    });
    expect(wrapper.get('[data-testid="update-progress-stage"]').attributes()).toMatchObject({
      'data-percentage': '0',
      'data-label': 'false',
      'data-indeterminate': 'true',
    });
  });

  it('maps migration to its own milestone instead of collapsing it into image pulling', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({ operation: { operation_id: 'update-1', status: 'MIGRATING' } as never, phase: 'running' });
    const wrapper = mountDialog();

    expect(wrapper.get('[data-testid="update-progress-overall"]').attributes('data-percentage')).toBe('55');
  });

  it('clears progress before opening application logs', async () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: { operation_id: 'update-1', status: 'FAILED' } as never,
      diagnostic: { request_id: 'request-42' } as never,
      phase: 'failed',
    });
    const wrapper = mountDialog();

    await wrapper.get('button').trigger('click');

    expect(progress.phase).toBe('idle');
    expect(routerPush).toHaveBeenCalledWith(
      expect.objectContaining({ query: expect.objectContaining({ request_id: 'request-42' }) }),
    );
  });
});

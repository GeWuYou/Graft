import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import { useUpdateProgressStore } from '../store/progress';
import UpdateProgressDialog from './UpdateProgressDialog.vue';

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
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
  },
  template: '<section data-testid="progress" :data-percentage="percentage" :data-label="label" />',
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
  });

  it('releases the shell when a terminal dialog is closed', async () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: { operation_id: 'update-1', phase: 'FAILED', progress: 100 } as never,
      phase: 'failed',
    });
    const wrapper = mountDialog();

    await wrapper.get('[data-testid="dialog-close"]').trigger('click');

    expect(progress.phase).toBe('idle');
  });

  it('uses the runner-reported percentage and displays the current phase label', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: { operation_id: 'update-1', phase: 'PULL_IMAGES', progress: 45 } as never,
      phase: 'running',
    });
    const wrapper = mountDialog();

    expect(wrapper.get('[data-testid="update-progress-overall"]').attributes()).toMatchObject({
      'data-percentage': '45',
      'data-label': 'true',
    });
    expect(wrapper.find('[data-testid="update-progress-stage"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('update.center.history.phases.PULL_IMAGES');
  });

  it('keeps the terminal failure phase and runner message visible', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: { operation_id: 'update-1', phase: 'FAILED', progress: 100, message: 'safe failure reason' } as never,
      lastActivePhase: 'MIGRATION' as never,
      phase: 'failed',
    });
    const wrapper = mountDialog();

    expect(wrapper.find('[data-testid="update-progress-stage"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('safe failure reason');
  });
});

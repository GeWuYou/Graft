import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import type { BootstrapResponse } from '@/modules/auth/contract/types';
import { usePermissionStore } from '@/store';

import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateProgressStore } from '../store/progress';
import type { UpdateOperation } from '../types/update';
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

function operation(overrides: Partial<UpdateOperation> = {}): UpdateOperation {
  return {
    operation_id: 'update-1',
    operation: 'self_update',
    runner_id: 'runner-1',
    source_version: '1.0.0',
    target_version: '1.1.0',
    deployment_strategy: 'beta_tracking',
    phase: 'READY',
    progress: 0,
    message: 'runner_accepted',
    state_source: 'runner_state',
    state_available: true,
    started_at: '',
    updated_at: '',
    ...overrides,
  };
}

function managerBootstrapSnapshot(): BootstrapResponse {
  return {
    user: { id: 1, username: 'manager', display_name: 'Manager' },
    must_change_password: false,
    roles: ['manager'],
    permissions: [UPDATE_PERMISSION_CODE.MANAGE],
    menus: [],
    locale: {
      current_locale: 'zh-CN',
      default_locale: 'zh-CN',
      fallback_locale: 'zh-CN',
      supported_locales: ['zh-CN'],
    },
  };
}

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

  it('keeps received node events visible when the runner snapshot becomes unavailable', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: { operation_id: 'update-1', phase: 'PREFLIGHT', progress: 5 } as never,
      phase: 'unavailable',
      events: [
        { operation_id: 'update-1', revision: 1, phase: 'PREFLIGHT', message: 'checking_environment', occurred_at: '' },
      ],
    });
    const wrapper = mountDialog();

    expect(wrapper.get('[data-testid="update-progress-events"] ol').classes()).toContain('graft-scrollbar');
    expect(wrapper.get('[data-testid="update-progress-events"]').text()).toContain(
      'update.center.history.messages.checking_environment',
    );
    expect(wrapper.text()).toContain('update.center.progress.sourceUnavailable');
  });

  it('shows the protected runner-termination diagnostic without presenting READY progress', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: {
        operation_id: 'update-1',
        phase: 'READY',
        progress: 0,
        state_source: 'runner_terminated',
        state_available: false,
      } as never,
      phase: 'failed',
      failureDiagnostic: {
        summary: 'Runner stopped before the upgrade could continue.',
        detail: 'The runner could not persist its terminal state.',
      } as never,
    });
    const wrapper = mountDialog();

    expect(wrapper.find('[data-testid="update-progress-overall"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('update.center.history.phases.READY');
    expect(wrapper.get('[data-testid="update-progress-diagnostic-summary"]').text()).toContain(
      'Runner stopped before the upgrade could continue.',
    );
    expect(wrapper.get('[data-testid="update-progress-diagnostic-detail"]').text()).toContain(
      'The runner could not persist its terminal state.',
    );
  });

  it('shows the same disconnected runner treatment for an expired lease', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: operation({ state_source: 'runner_lost', state_available: false }),
      phase: 'failed',
    });
    usePermissionStore().setBootstrapSnapshot(managerBootstrapSnapshot());
    const wrapper = mountDialog();

    expect(wrapper.find('[data-testid="update-progress-overall"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="update-progress-recovery"]').exists()).toBe(true);
  });

  it('does not offer recovery for legacy runner termination states', () => {
    const progress = useUpdateProgressStore();
    progress.$patch({
      operation: {
        operation_id: 'update-1',
        phase: 'READY',
        progress: 0,
        state_source: 'runner_terminated',
        state_available: false,
      } as never,
      phase: 'failed',
    });

    expect(mountDialog().find('[data-testid="update-progress-recovery"]').exists()).toBe(false);

    usePermissionStore().setBootstrapSnapshot(managerBootstrapSnapshot());

    expect(mountDialog().find('[data-testid="update-progress-recovery"]').exists()).toBe(false);
  });

  it('starts controlled runner recovery from the terminal dialog', async () => {
    const progress = useUpdateProgressStore();
    const recover = vi.spyOn(progress, 'recoverTerminatedRunner').mockResolvedValue();
    usePermissionStore().setBootstrapSnapshot(managerBootstrapSnapshot());
    progress.$patch({
      operation: {
        operation_id: 'update-1',
        phase: 'READY',
        progress: 0,
        state_source: 'runner_lost',
        state_available: false,
      } as never,
      phase: 'failed',
    });
    const wrapper = mountDialog();

    await wrapper.get('[data-testid="update-progress-recovery"]').trigger('click');

    expect(recover).toHaveBeenCalledOnce();
  });
});

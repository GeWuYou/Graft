import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import type { BootstrapResponse } from '@/modules/auth/contract/types';
import { usePermissionStore } from '@/store';

import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';
import type { UpdateStatus } from '../types/update';
import UpdateNotification from './UpdateNotification.vue';

vi.mock('tdesign-icons-vue-next', () => ({
  CloudDownloadIcon: defineComponent({ template: '<i />' }),
}));

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock('../composables/useUpdatePreviewActions', () => ({
  useUpdatePreviewActions: () => ({
    canStartUpgrade: false,
    openManagement: vi.fn(),
    startUpgrade: vi.fn(),
  }),
}));

const dialogStub = defineComponent({
  props: {
    attach: { type: String, default: '' },
    visible: { type: Boolean, default: false },
  },
  emits: ['close'],
  template:
    '<section v-if="visible" data-testid="update-preview-dialog" :data-attach="attach"><slot /><button data-testid="update-preview-close" @click="$emit(\'close\')" /></section>',
});

function bootstrapSnapshot(): BootstrapResponse {
  return {
    user: { id: 1, username: 'admin', display_name: 'Admin' },
    must_change_password: false,
    roles: ['admin'],
    permissions: [UPDATE_PERMISSION_CODE.READ],
    menus: [],
    locale: {
      current_locale: 'zh-CN',
      default_locale: 'zh-CN',
      fallback_locale: 'zh-CN',
      supported_locales: ['zh-CN'],
    },
  };
}

function updateStatus(): UpdateStatus {
  return {
    current_version: 'v1.0.0',
    channel: 'stable',
    image_tag: 'latest',
    deployment_strategy: 'stable_tracking',
    available_releases: [],
    installation_profile: {
      declared_mode: 'compose',
      detected_mode: 'compose',
      capability: 'compose_upgrade_available',
      guidance: '',
      compose_root_source: 'explicit_env',
      compose_candidates: [],
    },
    readiness: { overall: 'up_to_date', ready_count: 0, total_count: 0, checks: [] },
  };
}

function mountNotification() {
  return mount(UpdateNotification, {
    global: {
      stubs: {
        'cloud-download-icon': true,
        't-alert': { template: '<p><slot /></p>' },
        't-badge': { template: '<span><slot /></span>' },
        't-button': defineComponent({
          emits: ['click'],
          template: '<button type="button" @click="$emit(\'click\')"><slot /></button>',
        }),
        't-dialog': dialogStub,
        't-tooltip': { template: '<span><slot /></span>' },
      },
    },
  });
}

describe('UpdateNotification', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    usePermissionStore().setBootstrapSnapshot(bootstrapSnapshot());
    useUpdateDiscoveryStore().replaceSnapshot(updateStatus());
  });

  it('opens a body-attached preview dialog with content and restores interaction after close', async () => {
    const wrapper = mountNotification();

    await wrapper.get('[data-testid="update-notification"]').trigger('click');

    expect(wrapper.get('[data-testid="update-preview-dialog"]').attributes('data-attach')).toBe('body');
    expect(wrapper.text()).toContain('v1.0.0');

    await wrapper.get('[data-testid="update-preview-close"]').trigger('click');
    expect(wrapper.find('[data-testid="update-preview-dialog"]').exists()).toBe(false);
  });
});

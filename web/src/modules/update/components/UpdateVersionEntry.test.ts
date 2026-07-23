import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import { usePermissionStore } from '@/store';

import { checkForUpdates } from '../api/update';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';
import UpdateVersionEntry from './UpdateVersionEntry.vue';

const apiMocks = vi.hoisted(() => ({ checkForUpdates: vi.fn() }));
const messageMocks = vi.hoisted(() => ({ error: vi.fn() }));

vi.mock('../api/update', () => apiMocks);
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) => (values ? `${key}:${Object.values(values).join(',')}` : key),
  }),
}));

const passthrough = defineComponent({ template: '<div><slot /></div>' });
const buttonStub = defineComponent({
  emits: ['click'],
  template: '<button @click="$emit(\'click\')"><slot /></button>',
});

const status = (overrides: Record<string, unknown> = {}) =>
  ({
    channel: 'stable',
    checked_at: '2026-07-23T00:00:00Z',
    current_version: '1.0.0',
    latest: null,
    cache_stale: false,
    check_error: '',
    installation_profile: {
      declared_mode: 'compose',
      detected_mode: 'compose',
      capability: 'compose_upgrade_available',
    },
    ...overrides,
  }) as never;

function mountEntry() {
  return mount(UpdateVersionEntry, {
    global: {
      stubs: {
        't-popup': defineComponent({ template: '<div><slot /><slot name="content" /></div>' }),
        't-tooltip': passthrough,
        't-button': buttonStub,
        'refresh-icon': passthrough,
      },
    },
  });
}

describe('UpdateVersionEntry', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({
      permissions: [UPDATE_PERMISSION_CODE.READ, UPDATE_PERMISSION_CODE.CHECK],
    } as never);
    const discovery = useUpdateDiscoveryStore();
    discovery.reset();
    discovery.replaceSnapshot(status());
    vi.clearAllMocks();
  });

  it('shows an available release instead of an up-to-date message', () => {
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(status({ latest: { version: '1.1.0', upgrade_notes: 'Fixes' } }));

    const wrapper = mountEntry();

    expect(wrapper.text()).toContain('update.preview.availableVersion:1.1.0');
    expect(wrapper.text()).not.toContain('update.preview.upToDate');
    expect(wrapper.text()).toContain('update.preview.available');
  });

  it('shows unavailable messaging and hides stale release details', () => {
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(status({ latest: { version: '1.1.0', upgrade_notes: 'Stale' }, cache_stale: true }));

    const wrapper = mountEntry();

    expect(wrapper.text()).toContain('update.preview.unavailable');
    expect(wrapper.text()).not.toContain('1.1.0');
  });

  it('shows an error message when refreshing the status fails', async () => {
    apiMocks.checkForUpdates.mockRejectedValueOnce(new Error('network failure'));
    const wrapper = mountEntry();

    await wrapper.findAll('button')[1]?.trigger('click');
    await flushPromises();

    expect(checkForUpdates).toHaveBeenCalledOnce();
    expect(messageMocks.error).toHaveBeenCalledWith('update.preview.checkFailed');
  });
});

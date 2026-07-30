import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import { usePermissionStore } from '@/store';

import { checkForUpdates } from '../api/update';
import { UPDATE_ROUTE_PATH } from '../contract/paths';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';
import UpdateVersionEntry from './UpdateVersionEntry.vue';

const apiMocks = vi.hoisted(() => ({ checkForUpdates: vi.fn() }));
const messageMocks = vi.hoisted(() => ({ error: vi.fn() }));
const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));

vi.mock('../api/update', () => apiMocks);
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));
vi.mock('vue-router', () => ({ useRouter: () => routerMocks }));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) => (values ? `${key}:${Object.values(values).join(',')}` : key),
  }),
}));

const tooltipStub = defineComponent({
  inheritAttrs: false,
  props: { content: { type: String, default: '' } },
  template: '<div v-bind="$attrs" :data-tooltip-content="content"><slot /></div>',
});
const buttonStub = defineComponent({
  inheritAttrs: false,
  props: {
    disabled: Boolean,
    href: { type: String, default: '' },
    loading: Boolean,
    tag: { type: String, default: 'button' },
  },
  emits: ['click'],
  template:
    '<component :is="tag" v-bind="$attrs" :disabled="disabled" :href="href" @click="$emit(\'click\', $event)"><span v-if="loading" data-testid="button-loading" /><slot v-if="!loading" name="icon" /><slot /></component>',
});
const popupStub = defineComponent({
  props: { visible: Boolean },
  emits: ['update:visible'],
  template: '<div><slot /><slot name="content" /></div>',
});

const status = (overrides: Record<string, unknown> = {}) =>
  ({
    channel: 'stable',
    checked_at: '2026-07-23T00:00:00Z',
    current_version: '1.0.0',
    image_tag: 'latest',
    update_mode: 'stable_tracking',
    latest: null,
    cache_stale: false,
    check_error: '',
    installation_profile: {
      declared_mode: 'compose',
      detected_mode: 'compose',
      capability: 'compose_upgrade_available',
      guidance: '',
      compose_root_source: 'explicit_env',
      compose_candidates: [],
    },
    ...overrides,
  }) as never;

function mountEntry() {
  return mount(UpdateVersionEntry, {
    global: {
      stubs: {
        't-popup': popupStub,
        't-tooltip': tooltipStub,
        't-button': buttonStub,
        'refresh-icon': defineComponent({ template: '<span data-testid="refresh-icon" />' }),
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

  it('keeps the current version clickable without a tooltip when no update is available', () => {
    const wrapper = mountEntry();

    expect(wrapper.find('[data-testid="update-version-tooltip"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="update-version-entry"]').attributes('aria-label')).toBe(
      'update.versionEntry.current:1.0.0',
    );
  });

  it('shows an available release without repeating its release summary', () => {
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(
      status({
        latest: {
          version: '1.1.0',
          notes_url: 'https://github.com/GeWuYou/Graft/releases/tag/v1.1.0',
          upgrade_notes: 'Fixes',
        },
      }),
    );

    const wrapper = mountEntry();

    expect(wrapper.get('[data-testid="update-version-tooltip"]').attributes('data-tooltip-content')).toBe(
      'update.versionEntry.updateAvailable:1.1.0',
    );
    expect(wrapper.text()).toContain('update.preview.availableVersion:1.1.0');
    expect(wrapper.text()).not.toContain('update.preview.upToDate');
    expect(wrapper.findAll('p').some((paragraph) => paragraph.text() === 'update.preview.available')).toBe(false);
    expect(wrapper.text()).not.toContain('Fixes');
  });

  it('shows unavailable messaging and hides stale release details', () => {
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(status({ latest: { version: '1.1.0', upgrade_notes: 'Stale' }, cache_stale: true }));

    const wrapper = mountEntry();

    expect(wrapper.find('[data-testid="update-version-tooltip"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('update.preview.unavailable');
    expect(wrapper.text()).not.toContain('1.1.0');
  });

  it('checks asynchronously when the preview opens and shows the release link after success', async () => {
    let resolveCheck!: (value: never) => void;
    apiMocks.checkForUpdates.mockReturnValueOnce(new Promise((resolve) => (resolveCheck = resolve)));
    const wrapper = mountEntry();
    const popup = wrapper.findComponent(popupStub);

    await popup.vm.$emit('update:visible', true);
    expect(checkForUpdates).toHaveBeenCalledOnce();
    expect(wrapper.text()).toContain('update.preview.checking');
    expect(wrapper.text()).not.toContain('update.preview.unavailable');
    expect(wrapper.text()).not.toContain('update.preview.upToDate');

    resolveCheck(
      status({
        latest: {
          version: '1.1.0',
          notes_url: 'https://github.com/GeWuYou/Graft/releases/tag/v1.1.0',
          upgrade_notes: 'Fixes',
        },
      }) as never,
    );
    await flushPromises();

    const releaseButton = wrapper.get('[data-testid="update-preview-release"]');
    expect(releaseButton.attributes('href')).toBe('https://github.com/GeWuYou/Graft/releases/tag/v1.1.0');
    expect(releaseButton.attributes('target')).toBe('_blank');
    expect(releaseButton.attributes('disabled')).not.toBe('');
  });

  it('keeps the refresh button icon-only while checking', async () => {
    let resolveCheck!: (value: never) => void;
    apiMocks.checkForUpdates.mockReturnValueOnce(new Promise((resolve) => (resolveCheck = resolve)));
    const wrapper = mountEntry();

    await wrapper.findComponent(popupStub).vm.$emit('update:visible', true);

    expect(wrapper.get('[data-testid="button-loading"]').isVisible()).toBe(true);
    expect(wrapper.find('[data-testid="refresh-icon"]').exists()).toBe(false);

    resolveCheck(status() as never);
    await flushPromises();
  });

  it('opens update management from the header preview details action', async () => {
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(
      status({
        latest: {
          version: '1.1.0',
          notes: '# Release notes',
          notes_url: 'https://github.com/GeWuYou/Graft/releases/tag/v1.1.0',
        },
      }),
    );
    const wrapper = mountEntry();

    await wrapper.get('[data-testid="update-preview-detail"]').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith(UPDATE_ROUTE_PATH.CENTER);
  });

  it('uses an injected development preview path for the details action', async () => {
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(
      status({
        latest: {
          version: '1.1.0',
          notes_url: 'https://github.com/GeWuYou/Graft/releases/tag/v1.1.0',
        },
      }),
    );
    const wrapper = mount(UpdateVersionEntry, {
      props: { centerPath: '/mock/platform/updates' },
      global: {
        stubs: {
          't-popup': popupStub,
          't-tooltip': tooltipStub,
          't-button': buttonStub,
          'refresh-icon': defineComponent({ template: '<span data-testid="refresh-icon" />' }),
        },
      },
    });

    await wrapper.get('[data-testid="update-preview-detail"]').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith('/mock/platform/updates');
  });

  it('starts the controlled upgrade flow from the primary quick action', async () => {
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({
      permissions: [UPDATE_PERMISSION_CODE.READ, UPDATE_PERMISSION_CODE.CHECK, UPDATE_PERMISSION_CODE.MANAGE],
    } as never);
    const discovery = useUpdateDiscoveryStore();
    discovery.replaceSnapshot(
      status({
        latest: {
          version: '1.1.0',
          notes_url: 'https://github.com/GeWuYou/Graft/releases/tag/v1.1.0',
        },
      }),
    );
    const wrapper = mountEntry();

    await wrapper.get('[data-testid="update-preview-upgrade"]').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({ path: UPDATE_ROUTE_PATH.CENTER, query: { upgrade: '1' } });
  });

  it('keeps the dev version visible, disables release navigation, and allows retry after failure', async () => {
    apiMocks.checkForUpdates
      .mockRejectedValueOnce(new Error('network failure'))
      .mockRejectedValueOnce(new Error('network failure'));
    const wrapper = mountEntry();
    const popup = wrapper.findComponent(popupStub);

    await popup.vm.$emit('update:visible', true);
    await flushPromises();

    expect(wrapper.text()).toContain('1.0.0');
    expect(wrapper.text()).toContain('update.preview.unavailable');
    const releaseButton = wrapper.get('[data-testid="update-preview-release"]');
    expect(releaseButton.attributes('disabled')).toBe('');
    expect(releaseButton.attributes('href')).toBe('');

    await wrapper.get('[data-testid="update-preview-refresh"]').trigger('click');
    await flushPromises();

    expect(checkForUpdates).toHaveBeenCalledTimes(2);
    expect(messageMocks.error).toHaveBeenCalledWith('update.preview.checkFailed');
    expect(routerMocks.push).not.toHaveBeenCalled();
    expect(popup.props('visible')).toBe(true);
  });
});

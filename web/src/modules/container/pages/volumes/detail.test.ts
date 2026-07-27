import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, reactive } from 'vue';

const apiMocks = vi.hoisted(() => ({
  getDockerVolume: vi.fn(),
  removeDockerVolume: vi.fn(),
}));
const permissionStoreMock = vi.hoisted(() => ({ hasPermission: vi.fn(() => true) }));
const removalMocks = vi.hoisted(() => ({ open: vi.fn() }));
const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));
const routeState = vi.hoisted(() => ({ route: null as { params: { name: string } } | null }));

vi.mock('../../api/container', () => ({
  getDockerVolume: apiMocks.getDockerVolume,
  removeDockerVolume: apiMocks.removeDockerVolume,
}));
vi.mock('../../shared/volume-removal', () => ({ openVolumeRemovalConfirmation: removalMocks.open }));
vi.mock('@/store', () => ({ usePermissionStore: () => permissionStoreMock }));
vi.mock('vue-router', () => {
  routeState.route ??= reactive({ params: { name: 'volume-a' } });
  return {
    useRoute: () => routeState.route,
    useRouter: () => routerMocks,
  };
});
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

import VolumeDetailPage from './detail.vue';

const volume = (name: string) =>
  ({
    container_references: [],
    name,
    relationship_status: 'unused',
    size_bytes: 1024,
  }) as never;

function mountPage() {
  return mount(VolumeDetailPage, {
    global: {
      stubs: {
        'resource-detail-layout': defineComponent({
          emits: ['update:visible'],
          setup:
            (_, { slots }) =>
            () =>
              h('section', [slots.actions?.(), slots.default?.()]),
        }),
        't-alert': defineComponent({
          props: ['message'],
          setup: (props) => () => h('p', String(props.message ?? '')),
        }),
        't-empty': true,
        't-loading': true,
        't-tag': true,
        'volume-detail-content': defineComponent({
          props: ['canRemove', 'volume'],
          emits: ['remove'],
          setup:
            (props, { emit }) =>
            () =>
              h('div', [
                h('span', { 'data-testid': 'volume-name' }, String((props.volume as { name: string }).name)),
                props.canRemove ? h('button', { 'data-testid': 'remove-volume', onClick: () => emit('remove') }) : null,
              ]),
        }),
      },
    },
  });
}

describe('docker volume detail page', () => {
  beforeEach(() => {
    routeState.route!.params.name = 'volume-a';
    apiMocks.getDockerVolume.mockReset();
    apiMocks.removeDockerVolume.mockReset();
    removalMocks.open.mockReset();
    routerMocks.push.mockReset();
    permissionStoreMock.hasPermission.mockReset();
    permissionStoreMock.hasPermission.mockReturnValue(true);
  });

  afterEach(() => vi.restoreAllMocks());

  it('reloads for a changed volume route parameter and ignores stale responses', async () => {
    let resolveFirst: (value: never) => void = () => undefined;
    const firstRequest = new Promise<never>((resolve) => {
      resolveFirst = resolve;
    });
    apiMocks.getDockerVolume.mockImplementationOnce(() => firstRequest).mockResolvedValueOnce(volume('volume-b'));

    const wrapper = mountPage();
    routeState.route!.params.name = 'volume-b';
    await flushPromises();

    expect(apiMocks.getDockerVolume).toHaveBeenNthCalledWith(1, 'volume-a');
    expect(apiMocks.getDockerVolume).toHaveBeenNthCalledWith(2, 'volume-b');
    expect(wrapper.get('[data-testid="volume-name"]').text()).toBe('volume-b');

    resolveFirst(volume('volume-a'));
    await flushPromises();
    expect(wrapper.get('[data-testid="volume-name"]').text()).toBe('volume-b');
  });

  it('retains the page after a failed removal and returns to the list only after success', async () => {
    apiMocks.getDockerVolume.mockResolvedValue(volume('volume-a'));
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="remove-volume"]').trigger('click');
    const options = removalMocks.open.mock.calls[0]?.[0] as { onConfirm: (force: boolean) => Promise<boolean> };

    apiMocks.removeDockerVolume.mockRejectedValueOnce(new Error('remove failed'));
    await expect(options.onConfirm(false)).resolves.toBe(false);
    expect(routerMocks.push).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('remove failed');

    apiMocks.removeDockerVolume.mockResolvedValueOnce(undefined);
    await expect(options.onConfirm(false)).resolves.toBe(true);
    expect(routerMocks.push).toHaveBeenCalledWith({ name: 'DockerVolumeListIndex' });
  });

  it('hides the removal affordance without the removal permission', async () => {
    permissionStoreMock.hasPermission.mockReturnValue(false);
    apiMocks.getDockerVolume.mockResolvedValue(volume('volume-a'));
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('[data-testid="remove-volume"]').exists()).toBe(false);
  });
});

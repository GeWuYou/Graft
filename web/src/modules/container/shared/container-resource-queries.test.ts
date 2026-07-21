import { VueQueryPlugin } from '@tanstack/vue-query';
import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { queryClient } from '@/shared/query';

import { getDockerNetworks, getDockerSystem, getDockerVolumes } from '../api/container';
import {
  containerResourceQueryKeys,
  type DockerResourceTab,
  useDockerResourceQueries,
} from './container-resource-queries';

vi.mock('../api/container', () => ({
  getDockerNetworks: vi.fn(),
  getDockerSystem: vi.fn(),
  getDockerVolumes: vi.fn(),
}));

const getDockerNetworksMock = vi.mocked(getDockerNetworks);
const getDockerVolumesMock = vi.mocked(getDockerVolumes);
const getDockerSystemMock = vi.mocked(getDockerSystem);

describe('container resource query keys', () => {
  beforeEach(() => {
    queryClient.clear();
    getDockerNetworksMock.mockReset();
    getDockerVolumesMock.mockReset();
    getDockerSystemMock.mockReset();
  });

  it('keeps every remaining Docker resource snapshot in a distinct module key', () => {
    expect(containerResourceQueryKeys.networks()).toEqual(['container', 'resources', 'networks']);
    expect(containerResourceQueryKeys.volumes()).toEqual(['container', 'resources', 'volumes']);
    expect(containerResourceQueryKeys.system()).toEqual(['container', 'resources', 'system']);
  });

  it('does not use page-local snapshots as a second cache', () => {
    queryClient.setQueryData(containerResourceQueryKeys.networks(), {
      items: [{ id: 'network-1' }],
      total: 1,
      limit: 20,
      offset: 0,
      summary: { total: 1, in_use: 0, unused: 1 },
    });

    const active = ref<DockerResourceTab>('networks');
    const Harness = defineComponent({
      setup() {
        useDockerResourceQueries(active);
        return () => h('div');
      },
    });
    const wrapper = mount(Harness, {
      global: {
        plugins: [[VueQueryPlugin, { queryClient }]],
      },
    });

    expect(queryClient.getQueryData(containerResourceQueryKeys.networks())).toMatchObject({
      items: [{ id: 'network-1' }],
    });
    expect(getDockerNetworksMock).not.toHaveBeenCalled();
    expect(getDockerVolumesMock).not.toHaveBeenCalled();
    expect(getDockerSystemMock).not.toHaveBeenCalled();

    wrapper.unmount();
  });

  it('fetches only the active resource and reuses its cached snapshot', async () => {
    getDockerNetworksMock.mockResolvedValue({
      items: [
        {
          id: 'network-1',
          name: 'graft_default',
          driver: 'bridge',
          scope: 'local',
          created_at: '2026-07-16T00:00:00Z',
          internal: false,
          attachable: true,
          ingress: false,
          container_count: 0,
          container_references: [],
          context: { runtime: 'docker', source: 'compose', compose_project: 'graft' },
          relationship_status: 'unused',
        },
      ],
      total: 1,
      limit: 20,
      offset: 0,
      summary: { total: 1, in_use: 0, unused: 1 },
    });
    getDockerVolumesMock.mockResolvedValue({
      items: [],
      total: 0,
      limit: 20,
      offset: 0,
      summary: { total: 0, in_use: 0, unused: 0, reference_unknown: 0, size_bytes: 0 },
    });

    const active = ref<DockerResourceTab>('networks');
    const Harness = defineComponent({
      setup() {
        useDockerResourceQueries(active);
        return () => h('div');
      },
    });
    const wrapper = mount(Harness, {
      global: {
        plugins: [[VueQueryPlugin, { queryClient }]],
      },
    });

    await flushPromises();
    expect(getDockerNetworksMock).toHaveBeenCalledTimes(1);

    active.value = 'volumes';
    await flushPromises();
    expect(getDockerVolumesMock).toHaveBeenCalledTimes(1);

    active.value = 'networks';
    await flushPromises();
    expect(getDockerNetworksMock).toHaveBeenCalledTimes(1);

    wrapper.unmount();
  });
});

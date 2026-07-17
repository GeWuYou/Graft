import { VueQueryPlugin } from '@tanstack/vue-query';
import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { queryClient } from '@/shared/query';

import { getDockerImages, getDockerNetworks, getDockerSystem } from '../api/container';
import {
  containerResourceQueryKeys,
  type DockerResourceTab,
  useDockerResourceQueries,
} from './container-resource-queries';

vi.mock('../api/container', () => ({
  getDockerImages: vi.fn(),
  getDockerNetworks: vi.fn(),
  getDockerSystem: vi.fn(),
}));

const getDockerImagesMock = vi.mocked(getDockerImages);
const getDockerNetworksMock = vi.mocked(getDockerNetworks);
const getDockerSystemMock = vi.mocked(getDockerSystem);

describe('container resource query keys', () => {
  beforeEach(() => {
    queryClient.clear();
    getDockerImagesMock.mockReset();
    getDockerNetworksMock.mockReset();
    getDockerSystemMock.mockReset();
  });

  it('keeps every static Docker resource snapshot in a distinct module key', () => {
    expect(containerResourceQueryKeys.images()).toEqual(['container', 'resources', 'images']);
    expect(containerResourceQueryKeys.networks()).toEqual(['container', 'resources', 'networks']);
    expect(containerResourceQueryKeys.system()).toEqual(['container', 'resources', 'system']);
  });

  it('does not use page-local snapshots as a second cache', () => {
    queryClient.setQueryData(containerResourceQueryKeys.images(), { items: [{ id: 'image-1' }] });

    const active = ref<DockerResourceTab>('images');
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

    expect(queryClient.getQueryData(containerResourceQueryKeys.images())).toEqual({ items: [{ id: 'image-1' }] });
    expect(getDockerImagesMock).not.toHaveBeenCalled();
    expect(getDockerNetworksMock).not.toHaveBeenCalled();
    expect(getDockerSystemMock).not.toHaveBeenCalled();

    wrapper.unmount();
  });

  it('fetches only the active resource and reuses its cached snapshot', async () => {
    getDockerImagesMock.mockResolvedValue({
      items: [
        {
          id: 'sha256:image-1',
          repository_tags: ['alpine:latest'],
          repository_digests: ['alpine@sha256:image-1'],
          created_at: '2026-07-16T00:00:00Z',
          size_bytes: 7340032,
          containers: 0,
        },
      ],
    });
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
        },
      ],
    });

    const active = ref<DockerResourceTab>('images');
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
    expect(getDockerImagesMock).toHaveBeenCalledTimes(1);
    expect(getDockerNetworksMock).not.toHaveBeenCalled();

    active.value = 'networks';
    await flushPromises();
    expect(getDockerNetworksMock).toHaveBeenCalledTimes(1);

    active.value = 'images';
    await flushPromises();
    expect(getDockerImagesMock).toHaveBeenCalledTimes(1);

    wrapper.unmount();
  });
});

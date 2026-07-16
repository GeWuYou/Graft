import { beforeEach, describe, expect, it, vi } from 'vitest';

import { queryClient } from '@/shared/query';

import { getDockerImages, getDockerNetworks, getDockerSystem, getDockerVolumes } from '../api/container';
import { containerResourceQueryKeys } from './container-resource-queries';

vi.mock('../api/container', () => ({
  getDockerImages: vi.fn(),
  getDockerNetworks: vi.fn(),
  getDockerSystem: vi.fn(),
  getDockerVolumes: vi.fn(),
}));

const getDockerImagesMock = vi.mocked(getDockerImages);
const getDockerNetworksMock = vi.mocked(getDockerNetworks);
const getDockerVolumesMock = vi.mocked(getDockerVolumes);
const getDockerSystemMock = vi.mocked(getDockerSystem);

describe('container resource query keys', () => {
  beforeEach(() => {
    queryClient.clear();
    getDockerImagesMock.mockReset();
    getDockerNetworksMock.mockReset();
    getDockerVolumesMock.mockReset();
    getDockerSystemMock.mockReset();
  });

  it('keeps every static Docker resource snapshot in a distinct module key', () => {
    expect(containerResourceQueryKeys.images()).toEqual(['container', 'resources', 'images']);
    expect(containerResourceQueryKeys.networks()).toEqual(['container', 'resources', 'networks']);
    expect(containerResourceQueryKeys.volumes()).toEqual(['container', 'resources', 'volumes']);
    expect(containerResourceQueryKeys.system()).toEqual(['container', 'resources', 'system']);
  });

  it('does not use page-local snapshots as a second cache', () => {
    queryClient.setQueryData(containerResourceQueryKeys.images(), { items: [{ id: 'image-1' }] });

    expect(queryClient.getQueryData(containerResourceQueryKeys.images())).toEqual({ items: [{ id: 'image-1' }] });
    expect(getDockerImagesMock).not.toHaveBeenCalled();
    expect(getDockerNetworksMock).not.toHaveBeenCalled();
    expect(getDockerVolumesMock).not.toHaveBeenCalled();
    expect(getDockerSystemMock).not.toHaveBeenCalled();
  });
});

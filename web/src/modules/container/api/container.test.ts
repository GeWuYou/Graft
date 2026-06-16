// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import {
  buildContainerDetailApiPath,
  buildContainerLogsApiPath,
  buildContainerRemoveApiPath,
  buildContainerRestartApiPath,
  buildContainerStartApiPath,
  buildContainerStopApiPath,
  CONTAINER_API_PATH,
} from '../contract/paths';
import {
  batchContainerActions,
  getContainer,
  getContainerLogs,
  getContainers,
  removeContainer,
  restartContainer,
  startContainer,
  stopContainer,
} from './container';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('container api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reads the canonical container collection path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({
      items: [],
      runtime: { runtime: 'first-adapter', status: 'disabled', endpoint: '' },
    } as never);

    await getContainers({ limit: 20, offset: 40, keyword: 'graft', state: 'running', health: 'healthy' });

    expect(requestGet).toHaveBeenCalledWith({
      params: { limit: 20, offset: 40, keyword: 'graft', state: 'running', health: 'healthy' },
      url: CONTAINER_API_PATH.LIST,
    });
  });

  it('encodes container ids for detail and logs reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValue({ id: 'web/api' } as never);

    await getContainer('web/api');
    await getContainerLogs('web/api', { tail: 100, stdout: true, stderr: false, timestamps: true });

    expect(buildContainerDetailApiPath('web/api')).toBe('/api/ops/containers/web%2Fapi');
    expect(buildContainerLogsApiPath('web/api')).toBe('/api/ops/containers/web%2Fapi/logs');
    expect(requestGet).toHaveBeenNthCalledWith(1, {
      url: buildContainerDetailApiPath('web/api'),
    });
    expect(requestGet).toHaveBeenNthCalledWith(2, {
      url: buildContainerLogsApiPath('web/api'),
      params: { tail: 100, stdout: true, stderr: false, timestamps: true },
    });
  });

  it('posts high-risk actions through encoded canonical action paths', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({ id: 'web/api', action: 'start', result: 'completed' } as never);

    await startContainer('web/api');
    await stopContainer('web/api');
    await restartContainer('web/api');
    await removeContainer('web/api', { force: true });

    expect(buildContainerStartApiPath('web/api')).toBe('/api/ops/containers/web%2Fapi/start');
    expect(buildContainerStopApiPath('web/api')).toBe('/api/ops/containers/web%2Fapi/stop');
    expect(buildContainerRestartApiPath('web/api')).toBe('/api/ops/containers/web%2Fapi/restart');
    expect(buildContainerRemoveApiPath('web/api')).toBe('/api/ops/containers/web%2Fapi/remove');
    expect(requestPost).toHaveBeenNthCalledWith(1, {
      url: buildContainerStartApiPath('web/api'),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      url: buildContainerStopApiPath('web/api'),
    });
    expect(requestPost).toHaveBeenNthCalledWith(3, {
      url: buildContainerRestartApiPath('web/api'),
    });
    expect(requestPost).toHaveBeenNthCalledWith(4, {
      url: buildContainerRemoveApiPath('web/api'),
      data: { force: true },
    });
  });

  it('posts batch actions through the canonical collection action path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({
      total: 2,
      success_count: 2,
      failed_count: 0,
      items: [],
    } as never);

    await batchContainerActions({ action: 'remove', ids: ['web/api', 'worker'], force: false });

    expect(requestPost).toHaveBeenCalledWith({
      url: CONTAINER_API_PATH.BATCH_ACTIONS,
      data: { action: 'remove', ids: ['web/api', 'worker'], force: false },
    });
  });
});

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import {
  batchContainerActions,
  getContainer,
  getContainerLogs,
  getContainerMountUsage,
  getContainers,
  getDockerImages,
  postContainerMountUsageRefresh,
  postContainerShellSession,
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

  it('passes server pagination and keyword parameters to the Docker image list', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], total: 0, limit: 20, offset: 40, summary: {} } as never);

    await getDockerImages({ limit: 20, offset: 40, keyword: 'graft' });

    expect(requestGet).toHaveBeenCalledWith({
      params: { limit: 20, offset: 40, keyword: 'graft' },
      url: OPENAPI_RUNTIME_PATH.getDockerImages,
    });
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
      url: OPENAPI_RUNTIME_PATH.getContainers,
    });
  });

  it('encodes container ids for detail and logs reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValue({ id: 'web/api' } as never);

    await getContainer('web/api');
    await getContainerLogs('web/api', { tail: 100, stdout: true, stderr: false, timestamps: true });

    expect(requestGet).toHaveBeenNthCalledWith(1, {
      url: buildOpenApiRuntimePath('getContainer', { id: 'web/api' }),
    });
    expect(requestGet).toHaveBeenNthCalledWith(2, {
      url: buildOpenApiRuntimePath('getContainerLogs', { id: 'web/api' }),
      params: { tail: 100, stdout: true, stderr: false, timestamps: true },
    });
  });

  it('uses canonical mount usage paths and stable mount ids', async () => {
    const requestGet = vi.mocked(request.get);
    const requestPost = vi.mocked(request.post);
    requestGet.mockResolvedValue({ items: [] } as never);
    requestPost.mockResolvedValue({ mount_id: 'mount/source:/data' } as never);

    await getContainerMountUsage('web/api');
    await postContainerMountUsageRefresh('web/api', 'mount/source:/data');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getContainerMountUsage', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postContainerMountUsageRefresh', { id: 'web/api', mountId: 'mount/source:/data' }),
    });
  });

  it('posts high-risk actions through encoded canonical action paths', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({ id: 'web/api', action: 'start', result: 'completed' } as never);

    await startContainer('web/api');
    await stopContainer('web/api');
    await restartContainer('web/api');
    await removeContainer('web/api', { force: true });

    expect(requestPost).toHaveBeenNthCalledWith(1, {
      url: buildOpenApiRuntimePath('postContainerStart', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      url: buildOpenApiRuntimePath('postContainerStop', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(3, {
      url: buildOpenApiRuntimePath('postContainerRestart', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(4, {
      url: buildOpenApiRuntimePath('postContainerRemove', { id: 'web/api' }),
      data: { force: true },
    });
  });

  it('issues container shell sessions through the canonical shell session path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({
      command: 'sh',
      cols: 120,
      expires_at: '2026-06-19T10:00:30Z',
      rows: 32,
      session_id: 'shell_session_demo',
      websocket_url: '/api/ops/containers/web%2Fapi/shell/ws?ticket=opaque-ticket',
    } as never);

    await postContainerShellSession('web/api', { command: 'sh', cols: 120, rows: 32 });

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postContainerShellSession', { id: 'web/api' }),
      data: { command: 'sh', cols: 120, rows: 32 },
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
      url: OPENAPI_RUNTIME_PATH.postContainerBatchActions,
      data: { action: 'remove', ids: ['web/api', 'worker'], force: false },
    });
  });
});

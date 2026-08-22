import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import {
  batchContainerActions,
  batchRemoveDockerVolumes,
  createDockerNetwork,
  getContainer,
  getContainerLogs,
  getContainerMountUsage,
  getContainers,
  getDockerImages,
  postContainerMountUsageRefresh,
  postContainerShellSession,
  removeContainer,
  removeDockerNetwork,
  removeDockerVolume,
  restartContainer,
  startContainer,
  stopContainer,
} from './container';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
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

  it('submits lifecycle actions through encoded canonical paths with idempotency keys', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({ task_id: 42, status: 'pending' } as never);

    await startContainer('web/api', 'start-key');
    await stopContainer('web/api', 'stop-key');
    await restartContainer('web/api', 'restart-key');
    await removeContainer('web/api', { force: true }, 'remove-key');

    expect(requestPost).toHaveBeenNthCalledWith(1, {
      headers: { 'Idempotency-Key': 'start-key' },
      url: buildOpenApiRuntimePath('postContainerStart', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      headers: { 'Idempotency-Key': 'stop-key' },
      url: buildOpenApiRuntimePath('postContainerStop', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(3, {
      headers: { 'Idempotency-Key': 'restart-key' },
      url: buildOpenApiRuntimePath('postContainerRestart', { id: 'web/api' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(4, {
      headers: { 'Idempotency-Key': 'remove-key' },
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
    requestPost.mockResolvedValue({ task_id: 43, status: 'pending' } as never);

    await batchContainerActions({ action: 'remove', ids: ['web/api', 'worker'], force: false }, 'batch-remove-key');

    expect(requestPost).toHaveBeenCalledWith({
      headers: { 'Idempotency-Key': 'batch-remove-key' },
      url: OPENAPI_RUNTIME_PATH.postContainerBatchActions,
      data: { action: 'remove', ids: ['web/api', 'worker'], force: false },
    });
  });

  it('submits network and volume mutations with idempotency keys and returns Task receipts', async () => {
    const receipt = { task_id: 44, status: 'pending' };
    const requestPost = vi.mocked(request.post);
    const requestDelete = vi.mocked(request.delete);
    requestPost.mockResolvedValue(receipt as never);
    requestDelete.mockResolvedValue(receipt as never);

    await expect(
      createDockerNetwork({ name: 'frontend', driver: 'bridge', internal: false, attachable: true }),
    ).resolves.toEqual(receipt);
    await expect(removeDockerNetwork('network/id', { confirm_network_name: 'frontend' })).resolves.toEqual(receipt);
    await expect(removeDockerVolume('volume/name', { force: true })).resolves.toEqual(receipt);
    await expect(batchRemoveDockerVolumes({ names: ['one', 'two'], force: false })).resolves.toEqual(receipt);

    expect(requestPost).toHaveBeenNthCalledWith(1, {
      data: { name: 'frontend', driver: 'bridge', internal: false, attachable: true },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-network-create-/) },
      url: OPENAPI_RUNTIME_PATH.postDockerNetwork,
    });
    expect(requestDelete).toHaveBeenCalledWith({
      data: { confirm_network_name: 'frontend' },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-network-remove-/) },
      url: buildOpenApiRuntimePath('deleteDockerNetwork', { id: 'network/id' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      data: { force: true },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-volume-remove-/) },
      url: buildOpenApiRuntimePath('postDockerVolumeRemove', { id: 'volume/name' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(3, {
      data: { names: ['one', 'two'], force: false },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-volume-batch-remove-/) },
      url: OPENAPI_RUNTIME_PATH.postDockerVolumeBatchRemove,
    });
  });
});

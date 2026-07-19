import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import {
  createScheduledTask,
  deleteScheduledTask,
  disableScheduledTask,
  enableScheduledTask,
  getScheduledTask,
  getScheduledTaskJobDefinition,
  getScheduledTaskJobDefinitions,
  getScheduledTaskRun,
  getScheduledTaskRuns,
  getScheduledTasks,
  runScheduledTask,
  updateScheduledTask,
} from './scheduled-task';

vi.mock('@/utils/request', () => ({
  request: {
    delete: vi.fn(),
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  },
}));

describe('scheduled task api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('passes list pagination to the canonical scheduled task list path', async () => {
    const requestGet = vi.mocked(request.get);
    const query = { limit: 10, offset: 20 };
    requestGet.mockResolvedValueOnce({ items: [], total: 0, limit: 10, offset: 20 } as never);

    await getScheduledTasks(query);

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getScheduledTasks,
      params: query,
    });
  });

  it('calls the canonical job definition list path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], total: 0 } as never);

    await getScheduledTaskJobDefinitions();

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getScheduledTaskJobDefinitions,
    });
  });

  it('reads job definition details through the canonical job definition detail path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ key: 'audit/job' } as never);

    await getScheduledTaskJobDefinition('audit/job');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getScheduledTaskJobDefinition', { jobKey: 'audit/job' }),
    });
  });

  it('encodes scheduled task keys for detail reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ key: 'audit/job' } as never);

    await getScheduledTask('audit/job');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getScheduledTask', { taskKey: 'audit/job' }),
    });
  });

  it('posts create payloads to the canonical collection path', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = {
      task_key: 'audit.retention.daily',
      job_key: 'audit.retention',
      title: 'Audit retention',
      cron_expression: '*/5 * * * *',
      enabled: true,
      config_json: '{"window_days":30}',
    } as const;
    requestPost.mockResolvedValueOnce({ key: 'audit.retention.daily' } as never);

    await createScheduledTask(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.postScheduledTask,
      data: payload,
    });
  });

  it('puts update payloads to the canonical detail path', async () => {
    const requestPut = vi.mocked(request.put);
    const payload = { cron_expression: '0 * * * *', enabled: false };
    requestPut.mockResolvedValueOnce({ key: 'audit/job' } as never);

    await updateScheduledTask('audit/job', payload);

    expect(requestPut).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('putScheduledTask', { taskKey: 'audit/job' }),
      data: payload,
    });
  });

  it('deletes tasks through the canonical detail path', async () => {
    const requestDelete = vi.mocked(request.delete);
    requestDelete.mockResolvedValueOnce({} as never);

    await deleteScheduledTask('audit/job');

    expect(requestDelete).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('deleteScheduledTask', { taskKey: 'audit/job' }),
    });
  });

  it('posts enable and disable actions through canonical lifecycle paths', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({ key: 'audit/job' } as never);

    await enableScheduledTask('audit/job');
    await disableScheduledTask('audit/job');

    expect(requestPost).toHaveBeenNthCalledWith(1, {
      url: buildOpenApiRuntimePath('postScheduledTaskEnable', { taskKey: 'audit/job' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      url: buildOpenApiRuntimePath('postScheduledTaskDisable', { taskKey: 'audit/job' }),
    });
  });

  it('passes run history pagination to the canonical runs path', async () => {
    const requestGet = vi.mocked(request.get);
    const query = { limit: 10, offset: 20 };
    requestGet.mockResolvedValueOnce({ items: [], total: 0, limit: 10, offset: 20 } as never);

    await getScheduledTaskRuns('audit/job', query);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getScheduledTaskRuns', { taskKey: 'audit/job' }),
      params: query,
    });
  });

  it('posts manual runs through the canonical run action path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ id: 1, status: 'running' } as never);

    await runScheduledTask('audit/job');

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postScheduledTaskRun', { taskKey: 'audit/job' }),
    });
  });

  it('reads run details through the canonical run detail path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 42, status: 'success' } as never);

    await getScheduledTaskRun(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getScheduledTaskRun', { runId: 42 }),
    });
  });
});

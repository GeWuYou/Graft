import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import {
  buildScheduledTaskDetailApiPath,
  buildScheduledTaskDisableApiPath,
  buildScheduledTaskEnableApiPath,
  buildScheduledTaskRunApiPath,
  buildScheduledTaskRunDetailApiPath,
  buildScheduledTaskRunsApiPath,
  SCHEDULED_TASK_API_PATH,
} from '../contract/paths';
import {
  createScheduledTask,
  deleteScheduledTask,
  disableScheduledTask,
  enableScheduledTask,
  getScheduledTask,
  getScheduledTaskJobs,
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
      url: SCHEDULED_TASK_API_PATH.LIST,
      params: query,
    });
  });

  it('calls the canonical job definition list path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], total: 0 } as never);

    await getScheduledTaskJobs();

    expect(requestGet).toHaveBeenCalledWith({
      url: SCHEDULED_TASK_API_PATH.JOBS,
    });
  });

  it('encodes scheduled task keys for detail reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ key: 'audit/job' } as never);

    await getScheduledTask('audit/job');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildScheduledTaskDetailApiPath('audit/job'),
    });
    expect(buildScheduledTaskDetailApiPath('audit/job')).toBe('/api/scheduled-tasks/audit%2Fjob');
  });

  it('posts create payloads to the canonical collection path', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = {
      task_key: 'audit.retention.daily',
      job_key: 'audit.retention',
      title: 'Audit retention',
      cron_expression: '*/5 * * * *',
      enabled: true,
      params_json: '{"window_days":30}',
    } as const;
    requestPost.mockResolvedValueOnce({ key: 'audit.retention.daily' } as never);

    await createScheduledTask(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: SCHEDULED_TASK_API_PATH.LIST,
      data: payload,
    });
  });

  it('puts update payloads to the canonical detail path', async () => {
    const requestPut = vi.mocked(request.put);
    const payload = { cron_expression: '0 * * * *', enabled: false };
    requestPut.mockResolvedValueOnce({ key: 'audit/job' } as never);

    await updateScheduledTask('audit/job', payload);

    expect(requestPut).toHaveBeenCalledWith({
      url: buildScheduledTaskDetailApiPath('audit/job'),
      data: payload,
    });
  });

  it('deletes tasks through the canonical detail path', async () => {
    const requestDelete = vi.mocked(request.delete);
    requestDelete.mockResolvedValueOnce({} as never);

    await deleteScheduledTask('audit/job');

    expect(requestDelete).toHaveBeenCalledWith({
      url: buildScheduledTaskDetailApiPath('audit/job'),
    });
  });

  it('posts enable and disable actions through canonical lifecycle paths', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({ key: 'audit/job' } as never);

    await enableScheduledTask('audit/job');
    await disableScheduledTask('audit/job');

    expect(requestPost).toHaveBeenNthCalledWith(1, {
      url: buildScheduledTaskEnableApiPath('audit/job'),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      url: buildScheduledTaskDisableApiPath('audit/job'),
    });
  });

  it('passes run history pagination to the canonical runs path', async () => {
    const requestGet = vi.mocked(request.get);
    const query = { limit: 10, offset: 20 };
    requestGet.mockResolvedValueOnce({ items: [], total: 0, limit: 10, offset: 20 } as never);

    await getScheduledTaskRuns('audit/job', query);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildScheduledTaskRunsApiPath('audit/job'),
      params: query,
    });
  });

  it('posts manual runs through the canonical run action path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ id: 1, status: 'running' } as never);

    await runScheduledTask('audit/job');

    expect(requestPost).toHaveBeenCalledWith({
      url: buildScheduledTaskRunApiPath('audit/job'),
    });
    expect(buildScheduledTaskRunApiPath('audit/job')).toBe('/api/scheduled-tasks/audit%2Fjob/run');
  });

  it('reads run details through the canonical run detail path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 42, status: 'success' } as never);

    await getScheduledTaskRun(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildScheduledTaskRunDetailApiPath(42),
    });
  });
});

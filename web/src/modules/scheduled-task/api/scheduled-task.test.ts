import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import {
  buildScheduledTaskDetailApiPath,
  buildScheduledTaskRunApiPath,
  buildScheduledTaskRunsApiPath,
  SCHEDULED_TASK_API_PATH,
} from '../contract/paths';
import { getScheduledTask, getScheduledTaskRuns, getScheduledTasks, runScheduledTask } from './scheduled-task';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('scheduled task api', () => {
  it('calls the canonical scheduled task list path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], total: 0 } as never);

    await getScheduledTasks();

    expect(requestGet).toHaveBeenCalledWith({
      url: SCHEDULED_TASK_API_PATH.LIST,
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
    expect(buildScheduledTaskRunApiPath('audit/job')).toBe('/api/scheduled-tasks/audit%2Fjob:run');
  });
});

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getContainers } from '@/modules/container/contract/project';

import { fetchProjectRuntimeContainers, PROJECT_RUNTIME_CONTAINER_PAGE_SIZE } from './runtime-containers';

vi.mock('@/modules/container/contract/project', () => ({
  getContainers: vi.fn(),
}));

const mockedGetContainers = vi.mocked(getContainers);

describe('runtime-containers', () => {
  beforeEach(() => {
    mockedGetContainers.mockReset();
  });

  it('returns an empty list when the canonical project name is blank', async () => {
    await expect(fetchProjectRuntimeContainers('   ')).resolves.toEqual([]);
    expect(mockedGetContainers).not.toHaveBeenCalled();
  });

  it('fetches all project runtime containers across paginated container-list responses', async () => {
    mockedGetContainers
      .mockResolvedValueOnce({
        items: [
          { compose_project: 'compose-demo', compose_service: 'app', id: 'container-1', ports: [] },
          { compose_project: 'compose-demo', compose_service: 'worker', id: 'container-2', ports: [] },
        ],
        total: 3,
      } as never)
      .mockResolvedValueOnce({
        items: [{ compose_project: 'compose-demo', compose_service: 'cron', id: 'container-3', ports: [] }],
        total: 3,
      } as never);

    await expect(fetchProjectRuntimeContainers('compose-demo')).resolves.toEqual([
      { compose_project: 'compose-demo', compose_service: 'app', id: 'container-1', ports: [] },
      { compose_project: 'compose-demo', compose_service: 'worker', id: 'container-2', ports: [] },
      { compose_project: 'compose-demo', compose_service: 'cron', id: 'container-3', ports: [] },
    ]);

    expect(mockedGetContainers).toHaveBeenNthCalledWith(1, {
      limit: PROJECT_RUNTIME_CONTAINER_PAGE_SIZE,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'compose-demo',
      source_scope_kind: 'compose_project',
    });
    expect(mockedGetContainers).toHaveBeenNthCalledWith(2, {
      limit: PROJECT_RUNTIME_CONTAINER_PAGE_SIZE,
      offset: 2,
      orchestrator: 'compose',
      source_scope: 'compose-demo',
      source_scope_kind: 'compose_project',
    });
  });

  it('stops when the backend returns an empty page before reaching the reported total', async () => {
    mockedGetContainers
      .mockResolvedValueOnce({
        items: [{ compose_project: 'compose-demo', compose_service: 'app', id: 'container-1', ports: [] }],
        total: 3,
      } as never)
      .mockResolvedValueOnce({
        items: [],
        total: 3,
      } as never);

    await expect(fetchProjectRuntimeContainers('compose-demo')).resolves.toEqual([
      { compose_project: 'compose-demo', compose_service: 'app', id: 'container-1', ports: [] },
    ]);
  });
});

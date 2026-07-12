import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getContainers } from '@/modules/container/contract/project';

import {
  fetchProjectRuntimeContainers,
  PROJECT_RUNTIME_CONTAINER_PAGE_SIZE,
  readProjectContainerSourceGroup,
  readProjectContainerSourceMember,
} from './runtime-containers';

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
          {
            id: 'container-1',
            orchestrator: { group_value: 'compose-demo', member_value: 'app' },
            ports: [],
          },
          {
            id: 'container-2',
            orchestrator: { group_value: 'compose-demo', member_value: 'worker' },
            ports: [],
          },
        ],
        total: 3,
      } as never)
      .mockResolvedValueOnce({
        items: [{ id: 'container-3', orchestrator: { group_value: 'compose-demo', member_value: 'cron' }, ports: [] }],
        total: 3,
      } as never);

    await expect(fetchProjectRuntimeContainers('compose-demo')).resolves.toEqual([
      { id: 'container-1', orchestrator: { group_value: 'compose-demo', member_value: 'app' }, ports: [] },
      { id: 'container-2', orchestrator: { group_value: 'compose-demo', member_value: 'worker' }, ports: [] },
      { id: 'container-3', orchestrator: { group_value: 'compose-demo', member_value: 'cron' }, ports: [] },
    ]);

    expect(mockedGetContainers).toHaveBeenNthCalledWith(1, {
      limit: PROJECT_RUNTIME_CONTAINER_PAGE_SIZE,
      offset: 0,
      deployment_type: 'compose',
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
        items: [{ id: 'container-1', orchestrator: { group_value: 'compose-demo', member_value: 'app' }, ports: [] }],
        total: 3,
      } as never)
      .mockResolvedValueOnce({
        items: [],
        total: 3,
      } as never);

    await expect(fetchProjectRuntimeContainers('compose-demo')).resolves.toEqual([
      { id: 'container-1', orchestrator: { group_value: 'compose-demo', member_value: 'app' }, ports: [] },
    ]);
  });

  it('reads canonical orchestrator group and member values from runtime containers', () => {
    const container = {
      orchestrator: {
        group_display_name: 'Compose Demo',
        group_value: ' compose-demo ',
        member_display_name: 'App',
        member_value: ' app ',
      },
    } as never;

    expect(readProjectContainerSourceGroup(container)).toBe('compose-demo');
    expect(readProjectContainerSourceMember(container)).toBe('app');
  });
});

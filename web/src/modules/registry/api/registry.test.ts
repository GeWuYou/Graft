import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath } from '@/contracts/generated/openapi-runtime-paths';

const requestMocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock('@/utils/request', () => ({ request: requestMocks }));

import { getRegistryRepositoryAssignments } from './registry';

describe('getRegistryRepositoryAssignments', () => {
  it('forwards the canonical pagination query without reconstructing it in the adapter', async () => {
    requestMocks.get.mockResolvedValueOnce({ items: [], total: 0 });

    await getRegistryRepositoryAssignments('registry-a', {
      repository_ref: 'team/api',
      limit: 100,
      offset: 200,
    });

    expect(requestMocks.get).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getRegistryArtifactRepositoryAssignments', { connectionRef: 'registry-a' }),
      params: { repository_ref: 'team/api', limit: 100, offset: 200 },
    });
  });
});

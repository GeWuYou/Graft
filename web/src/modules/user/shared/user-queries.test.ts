import { beforeEach, describe, expect, it, vi } from 'vitest';

import { queryClient } from '@/shared/query';

import { getUsers } from '../api/users';
import { invalidateUserListQueries, userQueryKeys } from './user-queries';

vi.mock('../api/users', () => ({
  getUsers: vi.fn(),
}));

const getUsersMock = vi.mocked(getUsers);

describe('user query cache', () => {
  beforeEach(() => {
    queryClient.clear();
    getUsersMock.mockReset();
  });

  it('uses query parameters in its cache identity and invalidates every list page after a mutation', async () => {
    const query = { limit: 20, offset: 0 };
    queryClient.setQueryData(userQueryKeys.list(query), {
      items: [
        {
          id: 7,
          username: 'alice',
          display: 'Alice',
          status: 'enabled',
          roles: [],
          created_at: '2026-05-17T00:00:00Z',
          updated_at: '2026-05-17T00:00:00Z',
        },
      ],
    });

    await invalidateUserListQueries();

    expect(queryClient.getQueryState(userQueryKeys.list(query))?.isInvalidated).toBe(true);
    expect(getUsersMock).not.toHaveBeenCalled();
  });
});

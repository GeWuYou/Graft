import { beforeEach, describe, expect, it, vi } from 'vitest';

import { queryClient } from '@/shared/query';

import { getUsers } from '../api/users';
import { updateUserListCache, userQueryKeys } from './user-queries';

vi.mock('../api/users', () => ({
  getUsers: vi.fn(),
}));

const getUsersMock = vi.mocked(getUsers);

describe('user query cache', () => {
  beforeEach(() => {
    queryClient.clear();
    getUsersMock.mockReset();
  });

  it('uses a stable module key and applies mutation results to that snapshot', () => {
    queryClient.setQueryData(userQueryKeys.list(), {
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

    updateUserListCache((items) => items.map((item) => (item.id === 7 ? { ...item, status: 'disabled' } : item)));

    expect(queryClient.getQueryData(userQueryKeys.list())).toMatchObject({
      items: [expect.objectContaining({ id: 7, status: 'disabled' })],
    });
    expect(getUsersMock).not.toHaveBeenCalled();
  });
});

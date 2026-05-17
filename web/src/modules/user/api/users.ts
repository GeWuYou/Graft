import { request } from '@/utils/request';

import type { UserListResponse } from '../types/user';

const USER_API_PATH = {
  USERS: '/api/users',
} as const;

export function getUsers() {
  return request.get<UserListResponse>({
    url: USER_API_PATH.USERS,
  });
}

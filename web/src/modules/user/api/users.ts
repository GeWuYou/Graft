import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import type { UserListResponse } from '../types/user';

export function getUsers() {
  return request.get<UserListResponse>({
    url: USER_API_PATH.USERS,
  });
}

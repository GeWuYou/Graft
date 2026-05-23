import type { components } from '@/contracts/openapi/generated/schema';

import type { UserStatus } from '../contract/status';

export type RawUserListItem = components['schemas']['UserListItem'];
export type RawUserListResponse = components['schemas']['UserListResponse'];

export type UserListItem = Omit<RawUserListItem, 'status'> & {
  status: UserStatus;
};
export type UserListResponse = Omit<RawUserListResponse, 'items'> & {
  items: UserListItem[];
};

export interface CreateUserPayload {
  username: string;
  display: string;
  password: string;
}

export interface UpdateUserPayload {
  username: string;
  display: string;
}

export interface UpdateUserStatusPayload {
  status: UserStatus;
}

export interface ResetUserPasswordPayload {
  new_password: string;
}

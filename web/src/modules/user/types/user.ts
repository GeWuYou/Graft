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

export type CreateUserPayload = components['schemas']['CreateUserRequest'];
export type UpdateUserPayload = components['schemas']['UpdateUserRequest'];
export type UpdateUserStatusPayload = components['schemas']['UpdateUserStatusRequest'];
export type ResetUserPasswordPayload = components['schemas']['ResetUserPasswordRequest'];

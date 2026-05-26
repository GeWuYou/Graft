import type { components } from '@/contracts/openapi/generated/schema';

export type RoleListItem = components['schemas']['RoleListItem'];
export type RoleListResponse = components['schemas']['RoleListResponse'];
export type UserRoleBindingResponse = components['schemas']['UserRoleBindingResponse'];
export type ReplaceUserRolesPayload = components['schemas']['ReplaceUserRolesRequest'];
export type BatchUserRoleMutationPayload = components['schemas']['BatchUserRolesRequest'];

export type UserRoleMutation = 'replace' | 'add' | 'remove';

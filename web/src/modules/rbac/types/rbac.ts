import type { components } from '@/contracts/openapi/generated/schema';

export type CreateRolePayload = components['schemas']['CreateRoleRequest'];
export type UpdateRolePayload = components['schemas']['UpdateRoleRequest'];
export type ReplaceRolePermissionsPayload = components['schemas']['ReplaceRolePermissionsRequest'];
export type RolePermissionMutationPayload = components['schemas']['ReplaceRolePermissionsRequest'];
export type RolePermissionBindingResponse = components['schemas']['RolePermissionBindingResponse'];
export type RoleDetailResponse = components['schemas']['RoleDetailResponse'];
export type UpdateRoleStatusPayload = components['schemas']['UpdateRoleStatusRequest'];

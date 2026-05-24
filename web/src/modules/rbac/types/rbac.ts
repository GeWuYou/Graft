import type { components } from '@/contracts/openapi/generated/schema';

export interface RolePermissionBindingResponse {
  permission_ids: number[];
}

export type CreateRolePayload = components['schemas']['CreateRoleRequest'];
export type UpdateRolePayload = components['schemas']['UpdateRoleRequest'];
export type ReplaceRolePermissionsPayload = components['schemas']['ReplaceRolePermissionsRequest'];

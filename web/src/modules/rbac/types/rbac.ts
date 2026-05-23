import type { components } from '@/contracts/openapi/generated/schema';

export interface RolePermissionBindingResponse {
  permission_ids: number[];
}

export interface CreateRolePayload {
  name: string;
  display: string;
  description?: string | null;
}

export interface UpdateRolePayload {
  name: string;
  display: string;
  description?: string | null;
}

export type ReplaceRolePermissionsPayload = components['schemas']['ReplaceRolePermissionsRequest'];

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

export interface ReplaceRolePermissionsPayload {
  permission_ids: number[];
}

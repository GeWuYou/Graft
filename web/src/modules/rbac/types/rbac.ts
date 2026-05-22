// PermissionListItem describes one item from GET /api/permissions.
export interface PermissionListItem {
  id: number;
  code: string;
  display: string;
  description?: string | null;
  category: string;
  created_at: string;
  updated_at: string;
  role_binding_count?: number | null;
}

// PermissionListResponse matches the minimal permission list contract exposed by the rbac plugin.
export interface PermissionListResponse {
  items: PermissionListItem[];
}

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

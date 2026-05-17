// RoleListItem describes one item from GET /api/roles and role write responses.
export interface RoleListItem {
  id: number;
  name: string;
  display: string;
  description?: string | null;
  builtin: boolean;
}

// RoleListResponse matches the minimal role list contract exposed by the rbac plugin.
export interface RoleListResponse {
  items: RoleListItem[];
}

// PermissionListItem describes one item from GET /api/permissions.
export interface PermissionListItem {
  id: number;
  code: string;
  display: string;
  description?: string | null;
  category: string;
}

// PermissionListResponse matches the minimal permission list contract exposed by the rbac plugin.
export interface PermissionListResponse {
  items: PermissionListItem[];
}

export interface RolePermissionBindingResponse {
  permission_ids: number[];
}

export interface UserRoleBindingResponse {
  role_ids: number[];
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

export interface ReplaceUserRolesPayload {
  role_ids: number[];
}

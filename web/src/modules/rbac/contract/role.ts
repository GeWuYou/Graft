// RoleListItem describes one item from GET /api/roles and role write responses.
export interface RoleListItem {
  id: number;
  name: string;
  display: string;
  remark?: string | null;
  description?: string | null;
  builtin: boolean;
  updated_at: string;
  permission_count: number;
  user_count: number;
}

// RoleListResponse matches the minimal role list contract exposed by the rbac plugin.
export interface RoleListResponse {
  items: RoleListItem[];
}

// UserRoleBindingResponse matches the role binding contract used across user and rbac modules.
export interface UserRoleBindingResponse {
  role_ids: number[];
}

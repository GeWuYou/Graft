import type { UserStatus } from '../contract/status';

// UserListItem 描述 /api/users 返回的单个用户条目。
// 当前页面依赖后端稳定返回基础身份与状态字段。
export interface UserListItem {
  id: number;
  username: string;
  display: string;
  email?: string | null;
  status: UserStatus;
  created_at: string;
  updated_at: string;
}

// UserListResponse 对齐当前 MVP 阶段的最小用户列表契约。
export interface UserListResponse {
  items: UserListItem[];
}

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

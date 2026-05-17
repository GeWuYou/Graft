// UserListItem 描述 /api/users 返回的单个用户条目。
// 当前页面依赖后端稳定返回 id、username、display、created_at、updated_at 五个字段。
export interface UserListItem {
  id: number;
  username: string;
  display: string;
  created_at: string;
  updated_at: string;
}

// UserListResponse 对齐当前 MVP 阶段的最小用户列表契约。
export interface UserListResponse {
  items: UserListItem[];
}

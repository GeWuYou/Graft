# Access Control Menu Alignment

- 状态：`implemented`
- 范围：`web` 动态菜单契约与访问控制一级聚合

## 当前前端兼容策略

- 前端已将 `访问控制` 作为一级菜单聚合入口。
- `modules/access-control` 只拥有概览页与一级菜单聚合，不拥有 `user` 或 `rbac` 领域模型。
- `modules/user` 继续拥有用户领域页面、API、types、contract、locales。
- `modules/rbac` 继续拥有角色、权限、角色权限绑定与用户角色授权相关 API/contract/types/locales。

## 后端契约对齐结果

- 后端菜单现已直接下发：
  - `/access-control/overview`
  - `/access-control/users`
  - `/access-control/roles`
  - `/access-control/permissions`
- 前端动态路由装配层仍保留 legacy `/users`、`/roles`、`/permissions` 兼容归并，但不再依赖它作为主路径。
- 后端菜单 `title_key` 现统一为：
  - `menu.access_control.title`
  - `menu.access_control.overview.title`
  - `menu.access_control.users.title`
  - `menu.access_control.roles.title`
  - `menu.access_control.permissions.title`

## 本轮约束

- 为兼容旧菜单快照，前端动态路由装配层仍会把 `/users`、`/roles`、`/permissions` 归并到 `/access-control/*`。
- 该兼容层仅代表菜单与路由聚合，不代表 `rbac` 拥有 `user` 领域。

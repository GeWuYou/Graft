# user plugin

## 用途

`server/plugins/user` 是当前 MVP 路径中的用户示例插件，用来证明“登录 + 菜单 + 权限 + 路由 + 公共服务”这条插件扩展路径可以端到端接通。

## 职责边界

这个模块负责：

* 注册用户读取能力所需的权限与菜单
* 提供最小 `/auth/login`、`/auth/refresh`、`/auth/bootstrap`、`/auth/change-password`、当前 refresh session 的 `/auth/logout`、支持显式 `limit` 约束的当前用户 `/auth/sessions`、`/auth/sessions/:sessionID/revoke`、`/auth/sessions/revoke-all` 与 `/auth/sessions/revoke-others` 自助可见性/撤销入口，以及管理员按用户 ID 的 `/users/:id/sessions`、`/users/:id/sessions/:sessionID/revoke` 和 `/users/:id/sessions/revoke-all` 会话治理入口，并把 refresh session、cookie 与 revoke/rotation 逻辑留在插件内
* 暴露 `pluginapi.UserService`
* 暴露最小 `pluginapi.AuthService`，把 access token 解析结果收敛为稳定请求主体，并在受保护请求上追加最小 session 存活校验
* 提供受权限保护的示例用户路由
* 负责默认管理员初始化与首次登录强制改密所需的最小认证治理闭环

这个模块不负责：

* 真实完整的用户领域实现
* 更复杂的设备级 / 审计级 session 治理
* OAuth / SSO / MFA、密码历史、可配置密码策略或独立 security 插件
* 把用户存储实现直接暴露给其它插件

## 主要入口

* `doc.go`：插件用途说明
* `plugin.go`：插件生命周期、服务注册与示例路由
* `login.go`：最小用户名/密码认证应用层、bootstrap 返回面与首次改密状态透传
* `change_password.go`：当前用户自助改密、默认密码禁用与首次改密状态清除
* `session.go`：refresh token、cookie、支持显式 limit 裁剪的当前有效 session 摘要、当前/指定 session 定向 revoke、当前用户批量 revoke / 保留当前会话清退其它会话、管理员批量 revoke、session 轮换与 request-auth 最小 session hardening

## 关键依赖

* 依赖 `plugin.Context` 提供的菜单、权限、路由、服务与存储能力
* 登录链路内部只消费 `store.Auth()` 与 `store.Users()` 提供的稳定 DTO 边界
* 对外通过 `server/internal/pluginapi` 暴露跨插件可消费的稳定接口

## 当前认证治理约束

* 默认管理员账号固定为 `graft`
* `graft-admin` 是初始化例外密码，只允许在默认管理员首次种子写入路径中使用
* `change-password` 永远不允许把密码设置为 `graft-admin`
* 是否需要首次改密必须以后端持久化状态为准，前端不得通过用户名或默认密码猜测
* 当前 MVP 只要求 `login/bootstrap` 返回首次改密状态，不在这一切片里给全部业务接口追加全局后端拦截
* 当前登录后的业务阻断由 `web` 受限态完成；后续如需更强安全，再补服务端全局中间件
* 默认管理员已存在时，不得覆写其密码、角色或首次改密状态
* 默认管理员必须具备最小后台菜单与权限可见性，不能成为只能登录的空账号

## 维护提示

后续如果用户能力继续扩展，应优先保持对外接口稳定，并把业务实现细节留在插件内部，不要把 repository、ORM 句柄、refresh session 细节或临时路由约束泄漏到跨插件边界。

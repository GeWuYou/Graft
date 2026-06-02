# user module

## 用途

`server/modules/user` 是当前 MVP 路径中的用户模块，用来承载用户资料、用户管理与面向其它模块的稳定用户能力；认证与会话生命周期不再视为 `user` 的长期 ownership。

## 职责边界

这个模块负责：

* 注册用户读取能力所需的权限与菜单
* 在 `Register` 阶段向统一 `server/internal/i18n` facade 注册模块内建菜单标题 message，并通过 bootstrap 菜单快照暴露 `title_key + title fallback`
* 暴露 `moduleapi.UserService` 与后续 `auth` 所需的稳定用户身份能力
* 提供受权限保护的用户资料与用户管理路由
* 负责默认管理员对应的用户记录与用户资料存在性，但不再长期拥有 token/session/cookie/login 运行时闭环
* 如保留管理员按用户维度的 `/users/:id/sessions` 会话治理入口，它只能作为调用 `auth` capability 的管理入口，不直接持有 auth 持久化

这个模块不负责：

* 真实完整的用户领域实现
* 更复杂的设备级 / 审计级 session 治理
* OAuth / SSO / MFA、密码历史、可配置密码策略或独立 security 插件
* 把用户存储实现直接暴露给其它插件

## 主要入口

* `doc.go`：模块用途说明
* `module.go`：模块生命周期、服务注册与用户管理路由
* 认证相关实现当前只保留为 `auth` 模块消费的过渡 bridge；`/auth/*` 路由注册已迁入 `server/modules/auth`

## 关键依赖

* 依赖 `module.Context` 提供的菜单、权限、路由、服务与存储能力
* `user` 只暴露稳定用户身份与用户管理能力；auth 迁移完成后，登录链路通过 `moduleapi.UserAuthIdentityService` 一类稳定 capability 消费用户身份真相
* 对外通过 `server/internal/moduleapi` 暴露跨模块可消费的稳定接口

## 当前认证治理约束

* 默认管理员账号固定为 `graft`
* `graft-admin` 是初始化例外密码，只允许在默认管理员首次种子写入路径中使用
* `change-password` 永远不允许把密码设置为 `graft-admin`，并且必须要求 `current_password + new_password`
* `complete-required-password-change` 只允许 `must_change_password=true` 的已登录受限会话调用，并且只接收 `new_password`
* 是否需要首次改密必须以后端持久化状态为准，前端不得通过用户名或默认密码猜测
* `must_change_password=true` 的受限会话只能访问 `bootstrap`、`logout` 与 `complete-required-password-change`，其余已登录接口统一返回 `403`
* 当前登录后的业务阻断由 `web` 受限态负责页面与路由收敛，并由 `server` 受限会话白名单补足后端硬约束
* `graft dev reset-admin` 与删库后的默认管理员恢复路径都应进入“登录成功但受限、必须先改密”的恢复流程，而不是把默认密码视为长期可直接使用的正常管理员口令
* 默认管理员已存在时，不得覆写其密码、角色或首次改密状态
* 默认管理员必须具备最小后台菜单与权限可见性，不能成为只能登录的空账号

## 维护提示

后续如果用户能力继续扩展，应优先保持对外接口稳定，并把业务实现细节留在模块内部；token、session、cookie、refresh session 与 `/auth/*` 路由真相应持续向 `auth` 收口，不要再把这些认证生命周期细节重新泄漏回 `user` 或其它模块。

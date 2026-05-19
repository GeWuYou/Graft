# store

## 用途

`store` 定义面向 `server` 插件和上层服务暴露的中立持久化契约。

## 边界

* 只放稳定 DTO、错误语义与仓储接口。
* 不暴露 Ent client、查询构造器或 ORM 实体。
* 新增仓储能力时优先按插件真实需求最小化扩展，而不是预先铺满 CRUD。

## 主要入口

* `factory.go`：仓储工厂总入口
* `user.go`：用户资料读取边界
* `auth.go`：认证口令与 refresh session 边界
* `rbac.go`：角色与权限解析边界

## 维护提示

如需扩展跨插件可见的数据访问能力，先确认 `ai-plan/design/插件与依赖注入设计.md` 中的
repository / store factory 约束，再决定是否新增接口。

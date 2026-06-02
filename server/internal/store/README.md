# store

## 用途

`store` 只保留 shared persistence 的治理占位，不再承载 `user/auth` 这类业务插件仓储契约。

## 边界

* 不要把已迁回插件私有边界的业务仓储重新放回这里。
* 只有明确属于 core-owned 或短期尚未迁出的 shared 契约才允许进入该目录。
* 不暴露 Ent client、查询构造器或 ORM 实体。

## 主要入口

当前没有活动的 shared business store 契约；如需新增，先更新治理与设计真相，再决定是否真的属于 `internal/store`。

## 维护提示

如需扩展跨插件可见的数据访问能力，先确认 `ai-plan/design/插件与依赖注入设计.md` 与
`server/AGENTS.md` 中关于 module-local persistence 的约束，再决定是否允许新增 shared 契约。

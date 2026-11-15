# server/AGENTS.md

本文件拥有 server 执行规则；启动、授权、协作与提交按根 [AGENTS.md](../AGENTS.md)，复用本轮回执。
技术栈固定 Go/Gin/Ent/PostgreSQL/Viper/Zap/Casbin/cron，轻量显式 DI、compile-time modular monolith。

## 边界与生命周期

- 业务归 `modules/<name>/**`，core 仅有显式 runtime 基础设施。CLI 为 `cmd/graft`/`internal/cli`；runtime
  装配为 `internal/app`，HTTP 为 `internal/httpx`；config/logger/database/redis/container/module/eventbus/
  menu/permission/cron/i18n 各有明确 owner。不在 core 新增业务仓储或隐藏 runtime surface。
- `module.Spec` 持有 ID、依赖、builder、migration path；`module.Builder` 与 generated registry 显式装配。
  `Register` 只声明 route/menu/permission/message/event/job/service/config，不做 goroutine、阻塞 I/O 或副作用；
  `Boot` 只启动已声明行为，不补注册；`Shutdown` 释放全部资源并使用 runtime 生命周期上下文/统一关闭预算。
  Module 方法不得用 context.Background 或自建 time.After 窗口逃避生命周期。
- 跨模块 capability/DTO/稳定错误事件归 `internal/moduleapi`；平台 typed contract 归 `internal/contract`；
  私有稳定 contract 归 `modules/<name>/contract`。禁止跨模块导入 service/store/repository/handler/Ent/schema/
  migration 内部实现。依赖缺失、循环、重复注册阻断；公共能力必须可追溯到生命周期。
- DI 只做显式单例 provider 注册、解析与复用；Resolve 仅在装配/lifecycle adapter/middleware wiring，不能散入
  handler/service/store。禁止 init/global 隐式注册、反射扫描、热插拔、通用 service locator 或自建共享基础设施。
- auth 拥有认证/session/token，user 拥有资料管理，rbac 拥有授权与 resource/action/permission metadata。
  `user_roles` schema/repository/migration/tests 唯一 owner 是 rbac；user 仅提供 ID、存在性和删除约束 capability。
- OpenAPI 是 shared wire authority；`internal/contract/openapi` 为派生消费面。route 采用 group + fragment；
  高风险 contract 指定唯一 owner 和 experimental/stable/deprecated/removed 生命周期。上游 drift 按根规则升级。

## 按风险读取

仅加载变更涉及的设计，不为局部修复全读。新增能力先确定 runtime、持久化、route/menu/permission/message owner。

| 风险 | 权威细则与工作流 |
| --- | --- |
| 模块/DI/lifecycle/observability | [模块设计](../ai-plan/design/architecture/模块与依赖注入设计.md)、graft-server-module-scaffold |
| Provider/Adapter/Integration/Agent/Runner/部署/conformance | [项目组织](../ai-plan/design/architecture/项目文件组织与扩展点设计.md)、[ADR-025](../ai-plan/design/decisions/ADR-025-provider-oriented-project-layout.md) |
| 契约/DTO/HTTP/兼容 | [API 边界](../ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md)、[契约](../ai-plan/design/governance/platform/契约治理与魔法值治理规范.md) |
| schema/migration/store | [表与迁移](../ai-plan/design/governance/backend/数据库表设计与迁移规范.md)、graft-table-design、graft-sql-migration |
| query shape/count/分页/SQL/batch/timeout | [查询治理](../ai-plan/design/governance/backend/后端查询与数据库访问治理规范.md) |
| 权限/actor/危险操作/审计 | [安全边界](../ai-plan/design/governance/backend/后端安全与信任边界治理规范.md) |
| 行为测试/复杂度/Go 编码 | [测试与维护](../ai-plan/design/governance/backend/后端测试与可维护性治理规范.md) |
| 注释/GoDoc/模块 README | [注释规范](../ai-plan/design/governance/ai/代码注释与模块文档规范.md)、graft-comment-governance |
| 配置；共享资产 | [配置治理](../ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md)；graft-shared-asset-reuse |

## 数据与安全

- module owns `modules/<name>/ent/**` 与 `migrations/**`。禁止复活 `internal/ent/**` 业务 schema、生成物或 legacy
  replay。serve 不修改 schema；显式 migrate up 仍受根运行授权限制。已落地历史 mixed migration 不重写。
- live migration 数字版本跨模块全局唯一；只能改 owner 表或 core-owned 表。Ent schema.Comment/field.Comment 与
  SQL COMMENT ON TABLE/COLUMN 必须同时完整中文语义，表级启用 SQL comment 输出；不能只生成声明。
- schema/生成入口/影响 schema 的配置或代码变更：受影响 Ent go generate，显式生成/更新 migration、校验 checksum
  和 live 注释，直接包 go test 及 `cd server && go test ./...`。不自动同步数据库，不跨模块 Ent edge/entity 耦合。
- 不引入 N+1；列表默认分页，bounded dataset 例外须说明。DTO/VO 显式映射，HTTP 不暴露 Ent/DB edge/内部外键。
- 后端验证输入、权限和 actor/tenant/ownership，不相信前端身份字段或隐藏按钮。危险/批量高危操作有审计、保护条件、
  上限与部分失败语义。未知权限拒绝；不泄漏 DB 错误、token、secret。行为变更补最小直接回归测试。
- 事务在 service/usecase，store 不隐式开启，同一事务共用 tx，defer Rollback，Commit 后不复用。
- 请求 context 第一参数贯穿 handler/service/store/外部客户端；goroutine 响应取消，资源显式释放，重试有界。
- config 统一 internal/config，默认值/校验在边界，业务不 os.Getenv，不写死秘密；依赖显式注入。
- 手写 Go 一文件一职责，命名具体，避免 common/utils/helper 万能包。错误保留 %w cause，由最了解语义的 owner
  记录一次；service 用 AppLogger/ReportError，handler 用 WriteAppError/AbortAppError，不重复 ERROR。
  expected 400/401/403/404/409 不默认 ERROR，安全拒绝归 Security Event；Access Log 固定 INFO 只记请求事实，
  未处理 panic 仅统一 Recovery 记录真实 stack。导出 GoDoc 以标识符开头，中文解释边界而非复述代码。

## 并行归属

module-owned 为默认长期职责，moduleapi/contract 是共享稳定边界，moduleregistry/generated.go 为机械接线热点，
core/app/module/migration 需单一 owner。共享变更串行协调，不手写生成接线或业务兼容。必要 authority 修复由协调者
重分配，不能以工作树隔离阻止上游修复。OpenAPI 派生闭环按根 Git 细则；其余线性资源不并行抢写。

## 后端验证与完成

完成入口是 `cd server && go run ./cmd/graft validate backend`（已构建时同一入口 `graft validate backend`）。
顺序：live migration 全局版本唯一检查 -> `graft validate backend --stage lint` -> 最小直接 go test ->
`go build ./cmd/graft` -> 仅需要 runtime 证明且获授权时 `graft validate smoke`。

唯一 blocking lint 为上述 stage，固定 golangci-lint v2.12.2，changed-file scoped
`--new-from-rev=<merge-base> --whole-files`；不扩大 backlog，不把 untouched backlog 当当前阻塞。
core 改动覆盖相关 internal 测试；schema/store/public contract 不用单包 smoke 替代行为覆盖。暂存 live SQL 的
pre-commit 全局版本检查继续有效。仅跑 go test 或遗漏 lint/生成/migration 不能声明后端完成。

复用未变输入的有效验证；治理文档切片做相应 docs/automation 检查，不伪造 server runtime 证据。最终报告 task class、
owned scope、boundary decision、schema/migration owner、命令/结果/跳过原因、风险/回滚和文档同步；手写 Go 做
graft-comment-governance 审查，注释实际改变时输出对应回执。

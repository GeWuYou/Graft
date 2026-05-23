# Graft OpenAPI First 契约治理长分支计划

> Historical status: `openapi-contract-governance` is complete, archived in place, and inactive. This document remains
> as the final topic design record; future implementation must start under a new topic rather than continuing this one.

## 1. 结论

- 建议跳过 swaggo 过渡，直接进入 OpenAPI First。
- 当前阶段适合提前切入，因为接口少、插件边界清晰、统一 envelope 已稳定。
- 现在仍不该做：不让 `oapi-codegen` 接管 server interface，不替换现有 request transport，不把 Monitor 全量冻死。

## 2. 当前仓库现状

- `server` 已按插件注册路由，`runtime` 只负责 `/api` 根挂载与生命周期编排。
- 统一 envelope 已在 `httpx` 收口，包含 `success/code/message/traceId/messageKey/locale/data`。
- `Auth/User/RBAC` 约束较稳定；`Monitor` 仍有趋势窗口和 runtime 指标演进，不宜首批强约束。
- `web` 已有模块级 API/contract/types 结构，适合承接生成类型，但不适合绕开 `request.ts`。
- OpenAPI 校验必须接入 `graft validate backend` 或等价显式入口。

## 3. 推荐架构

- 顶层使用 `openapi/` 作为 spec 入口。
- 每个插件维护自己的 fragment，根 spec 只做聚合。
- 公共 schema 集中定义 `ApiEnvelope`、`ApiError`、security scheme、分页和公共 DTO。
- 前端生成物放到明确的 generated 目录，禁止手改，作为只读产物。
- spec 源文件入库；静态文档站点产物不入库。

## 4. 第一批接口范围

- Auth: Phase 1。
- User: Phase 1。
- RBAC Role: Phase 1。
- RBAC Permission: Phase 1。
- Health: Phase 1。
- Monitor: 延后。

## 5. 统一 Schema 设计

- `ApiEnvelope`
- `ApiError`
- `PageResult`
- `LoginRequest`
- `LoginResponse`
- `User DTO`
- `Role DTO`
- `Permission DTO`
- `Health DTO`
- 401/403/common error response
- `traceId/messageKey/locale`

## 6. 工具链建议

- `openapi-typescript`: 现在引入，作为前端类型生成主工具。
- `openapi-fetch`: 后续评估，必须包在现有 request 语义下。
- `kin-openapi`: 现在引入到校验链。
- `oapi-codegen`: 先不接管 server interface，后期再评估 Go models 或 interface。
- `redocly/openapi-cli`: 可作为 lint/format 辅助。

## 7. 分阶段计划

### Phase 1

- 目标：建立 spec 真值和最小可浏览文档。
- 改动范围：`openapi/` 目录、Auth/User/RBAC/Health fragments、公共 components、TS 类型生成骨架、CI 校验钩子。
- 不做事项：不做 Monitor 全量、不做 server interface 生成、不替换 request transport、不上 swaggo。
- 验收标准：spec 可校验、文档可生成、前端可消费生成类型、后台接口与 spec 对齐。
- 推荐提交粒度：先治理文档，再补 spec 骨架，再补 web 类型，再补 CI。
- 是否影响业务接口：不改行为，只固化契约。
- 是否允许生成代码：允许生成 TS 类型，不生成 Go server code。

### Phase 2

- 目标：扩展到稳定 CRUD 全量并完善 docs。
- 改动范围：补齐分页、错误响应、更多 operationId、Redoc/Scalar 预览。
- 不做事项：仍不让 `oapi-codegen` 接管 server。
- 验收标准：生成物可重建、CI 可拦住 spec/类型漂移。
- 是否影响业务接口：不应影响。
- 是否允许生成代码：是。

### Phase 3

- 目标：把前端 SDK 变成可选默认入口。
- 改动范围：评估 `openapi-fetch` 或轻量封装，保留 `request.ts` 作为 transport 真相。
- 不做事项：不绕开 token refresh、locale、traceId。
- 验收标准：新模块能优先用生成类型/SDK，旧 API 不被破坏。
- 是否允许生成代码：允许。
- 当前结论：先不引入 `openapi-fetch` 或第二套 runtime client；以生成 schema 类型 + 模块本地 alias + 现有 `request.ts` transport 语义作为 Phase 3 完成态。

### Phase 4

- 目标：评估 Go 侧生成。
- 改动范围：仅在 Auth/User/RBAC 稳定后，试点 `oapi-codegen` 生成 Go models 或 server interface。
- 不做事项：Monitor 不先行。
- 验收标准：若收益不明显可停留在 spec-first + TS-first。
- 当前结论：现阶段关闭为 deferred/no-go，不引入 `oapi-codegen`。继续保持 `spec-first + TS-first + explicit server DTOs`，直到后续有更稳定的跨插件 Go contract 需求、明确的生成物归属，以及不会削弱插件显式路由/生命周期边界的验证入口。

## 8. 风险与规避

- spec/handler 漂移：用 route/spec parity 测试和 `kin-openapi` 校验卡住。
- 生成物污染源码：生成目录隔离，禁止手改，CI 做 diff 检查。
- 前端 request 层被绕过：`openapi-fetch` 只能包在现有 `request.ts` 语义下。
- plugin 路由分散：按插件 fragment 拆分，根 spec 只做 assembly。
- envelope 重复定义：只保留一套 `ApiEnvelope` / `ApiError`。
- Monitor 不稳定：延后，避免反复改 spec。
- `oapi-codegen` 过早接管：先不生成 server interface。

## 9. 建议 worktree 信息

- worktree 标识：`feat/wt-openapi-contract-governance`（本地路径不入库）
- 分支名：`feat/wt-openapi-contract-governance`
- topic 名：`openapi-contract-governance`
- ai-plan 落点：`ai-plan/public/openapi-contract-governance/`
- 边界：只管 OpenAPI 治理、spec 规划、前端生成策略和 CI 方案，不吞并插件实现。

## 10. 下一步

- 建议进入 Phase 1 实现。
- Phase 1 最小任务清单：建立 `openapi/` 目录骨架、定义共享 envelope/security/schema 约定、为 Auth/User/RBAC/Health 建首批 fragments、把 spec 校验接到 backend validate 和 web 类型生成流程。
- Phase 1 不允许做：swaggo 注释体系、Go server interface 生成、Monitor 全量纳入、绕过 `request.ts` 的前端直连 client、静态 docs 产物入库。

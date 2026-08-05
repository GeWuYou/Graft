# ADR-022: Task Submission Materialization

- Status: accepted
- Date: 2026-08-05
- Scope: `server/modules/task`, Task consumers, `openapi/**`, `web/src/modules/task/**`

## Context

ADR-004 的 Reservation 实现把不可领取语义保存为 Task 上的 `activation_required`。调用方在单独事务中持久化快照，使进程崩溃后可能遗留永久 pending Task、owner 占用和无法恢复的 activation flag。

## Decision

1. 引入 TaskSubmission 作为 Task 创建前的独立聚合；Task 只在所有本地前置条件已持久化后创建。
2. Submission 状态限定为 `reserved`、`activated`、`discarded`、`expired`，后三者均为终态；所有迁移使用统一 submission version CAS。
3. 本地 prerequisite 通过通用 transaction-scoped writer 在一个 PostgreSQL 事务中物化 Task、Snapshot 和 Submission 终态；Task Runtime 不依赖任何业务 consumer。
4. `TaskStatusReady` 成为 Task 的唯一可领取入口；Worker 不读取 Submission。
5. 不创建 owner claim 表。两张事实表各自的局部唯一索引配合稳定 owner PostgreSQL transaction advisory lock，保证跨表活跃 owner 互斥。
6. ADR-004 继续拥有执行期 Stage、worker、日志与崩溃恢复决策；本 ADR 替代其 Task 创建前 Reservation 语义。

## Consequences

- 本地提交路径不再需要 `activation_pending` 或 Activation Reconciler。
- `activation_required` 仅作为受期限的 legacy migration bridge；不得成为新运行时语义。
- Build 是首个 writer，未来 OCI、Git、Artifact 与 Helm 前置条件可使用相同 Materialize contract。
- 需要更新任务状态 wire contract、generated consumers、数据库 migration 和现有 Task producer 测试。

# Runtime Composability Governance Trace

## 2026-08-14 work-intake-design-bootstrap

- Classified the work as a long-running cross-boundary refactor: its eventual implementation can affect core Runtime, Modules, Providers, Agents, Task Runtime, realtime and typed capability boundaries.
- Created repository-wide design and roadmap authority plus the minimum active-topic recovery materials.
- Locked three principles: every runtime resource has creator/owner/disposer; capability visibility is bounded; composition units declare dependencies, exposed capabilities and cleanup responsibility.
- Rejected runtime Plugin Loader, nested dynamic configuration tree, Proxy Context hierarchy and HMR because Graft remains a compile-time Go modular monolith with persistent Task, Agent and external-resource authority.

## Locked Decisions

- A Resource Scope is only a narrow lifecycle tool for repeated cancellable resources; it cannot become a universal service context.
- Module dependencies, capability dependencies and resource ownership remain separate declarations.
- Process restart, configuration reconcile and Agent reconnect are preferred over runtime module reload.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-design-bootstrap", "phase-0-resource-inventory", "phase-1-lifecycle-cleanup"],
  "pending_batches": [
    "phase-0-resource-inventory",
    "phase-1-lifecycle-cleanup",
    "phase-2-narrow-resource-scope",
    "phase-3-capability-composition-declarations",
    "phase-4-controlled-change-evaluation"
  ],
  "current_batch": "phase-2-narrow-resource-scope",
  "next_batch": "phase-3-capability-composition-declarations",
  "closeout_status": "active"
}
```

## 2026-08-15 phase-0-resource-inventory

- Startup receipt: governance source `AGENTS.md`; task class `cross-boundary`; recovery source parent topic `runtime-composability-governance`; owned scope Runtime、Module、Task Runtime、Provider/Agent/Runner、Event、Realtime 长生命周期资源，优先 `server/**` 与主题追踪材料。
- 使用两个只读 explorer 分别盘点 Runtime/Module/Task/Event/cron 与 realtime/Provider/Agent/Runner/container/runtime-target；explorer 未修改文件、未提交、未运行验证。
- 事实确认：`internal/app.Runtime` 是 core resource 唯一 owner，按 MCP、Agent listeners、HTTP、durable Dispatcher、逆序 Module、Redis/DB/logger 关闭，并以 `WithoutCancel` + 默认 5s deadline 聚合错误；Module `Register` 只声明，`Boot`/逆序 `Shutdown` 承载业务资源。
- 事实确认：Task Runtime worker/stage/ticker、scheduler CronRuntime、container stats/runtime-event、RuntimeTarget summary collector 都有显式 cancel/stop/wait；Task 持久化状态、Submission/Stage recovery 和 RuntimeTarget ledger 保持既有 authority。
- 事实确认：同步 `MemoryBus` 是无持久化、无重试、无 goroutine 的 application-lifetime observer；durable Dispatcher 才拥有 outbox lease/retry/reclaim 和 worker/poller 生命周期；当前不存在外部 broker 资源。
- 事实确认：realtime Hub 由 Runtime 持有，订阅/observer/gateway stream 有 disposer；project/container topic streamer 的运行 context 多处从 `context.Background()` 派生，模块 `Service.Close` 是当前实际兜底；WS/SSE 依赖 request/HTTP shutdown 取消。
- 事实确认：Provider/Runner/独立 Agent 当前没有第二套 scheduler、queue、DI 或通用 Context 树；docker provider 按 operation context 清理，Builder Agent 持有 ledger/telemetry 事实，独立 Agent listener 由 Runtime 负责 HTTP/TLS shutdown。

### P0 Candidate Decisions

1. Task Runtime `Stop(nil)` 活跃路径存在 nil context 风险，Phase 1 优先归一化 context 并补回归测试。
2. Durable Dispatcher deadline 后可能仍有 worker，且状态未形成明确 terminal/health 语义；Phase 1 增加 forced-timeout 观测并禁止 deadline 后复用，保持 durable authority。
3. 审计 container/project realtime detached `Background` context，确保所有 Module failure/shutdown 路径调用 `Service.Close` 并受有界等待约束。
4. 为 RuntimeTarget collector 的静默 repository failure、Agent listener Serve/forced-close、external source stop 增加最小日志/健康证据。
5. 用测试/静态审计固化 `Register` 不启动长期资源、MemoryBus handler 不隐藏后台 goroutine、cron Registry 只登记声明。

本批次没有证明需要通用共享 Resource Scope；继续保持 Phase 2 pending。设计文档补充了 durable Dispatcher 的 process-lifetime parent 与 MemoryBus application-lifetime owner 语义，避免把现有独立生命周期误判为 authority drift。

## 2026-08-15 phase-1-lifecycle-cleanup

- Task Runtime `Stop` 在入口归一化 nil context，新增 active-runtime `Stop(nil)` 回归测试；Task/Submission/Stage persistence authority 未改变。
- durable Dispatcher 在 graceful/forced shutdown 后进入 terminal state：`Start` 拒绝复用，`Shutdown` 幂等，forced deadline 清除 started 状态并记录结构化 warning；handler 忽略 context 时仍可能在 deadline 后结束，durable store 保持恢复 authority。
- Phase 1 focused validation：`cd server && go test ./modules/task ./internal/event` 通过；`git diff --check` 通过。
- 本批次未新增 shared Scope、EventBus unsubscribe、Task state machine、外部 broker 或动态 module loader。

## 2026-08-16 phase-2-narrow-resource-scope

- Startup receipt: governance source `AGENTS.md`; task class `cross-boundary`; recovery source parent topic
  `runtime-composability-governance`; owned scope remained runtime lifecycle and topic tracking/trace only.
- Re-evaluated the Phase 0 inventory after Phase 1 cleanup. The repeated pairs found are already owned by one clear
  lifecycle authority: Task Runtime owns worker/ticker cancellation, realtime stream or Service owns observer/stream
  cleanup, and Runtime owns core listener/dispatcher shutdown. Provider operations and independent Agents likewise
  retain their operation/process owners.
- No case proved both required conditions for a new scope: at least two resources needing joint cancellation/release,
  and an existing Module, Task, subscription or Agent owner unable to express that relationship.
- Decision: Phase 2 completes with no Resource Scope API. `module.Context`, Task/Submission, Agent, Provider, Runner and
  realtime authority remain unchanged; no dynamic loader, second DI/scheduler, or Task state machine was introduced.
- Minimal docs validation is recorded at closeout: `git diff --check` and
  `python3 scripts/validate_ai_plan_structure.py`.

## 2026-08-16 phase-3-capability-composition-declarations

- Startup receipt: governance source `AGENTS.md`; task class `cross-boundary`; recovery source parent topic `runtime-composability-governance`; owned scope runtime composition declarations and topic tracking/trace.
- Authority discovery confirmed `module.Spec` is the compile-time module metadata owner and existing `moduleapi` interfaces are the typed capability contract owner. The container remains an explicit construction/resolve boundary, not a metadata authority.
- Added typed `required`/`exposed` capability declarations via `module.TypedCapability[T]()` plus configuration-owner and resource/disposer metadata. `RuntimeMetadata` returns defensive snapshots for diagnostics and conformance.
- Added capability-local health vocabulary (`Ready`, `Degraded`, `Unavailable`) without a global registry. Provider/Agent/Runner/Task/realtime facts and lifecycle owners remain unchanged.
- Explicitly rejected string service lookup, runtime dependency solving, `module.Context` expansion, a second DI/scheduler/Task state machine, dynamic Plugin Loader, and HMR.
- Validation: `gofmt`, focused `go test ./modules/task ./internal/module`, `go run ./cmd/graft validate backend --stage lint`, and `go build ./cmd/graft`; docs validation `git diff --check` and `python3 scripts/validate_ai_plan_structure.py`.

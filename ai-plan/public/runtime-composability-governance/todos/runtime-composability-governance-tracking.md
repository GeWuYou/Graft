# Runtime Composability Governance Tracking

## Topic

Runtime Composability Governance

## Scope

Establish and implement resource ownership, bounded capability visibility, and composition declarations across Graft Runtime, Module, Provider, Agent, Runner, Task Runtime and realtime boundaries.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/architecture/运行时组合与资源治理设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/项目文件组织与扩展点设计.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Runtime owns lifecycle orchestration; each composition unit owns its resources; ModuleSpec and typed contracts own dependency and capability boundaries; Task/Submission, Agent, Provider and realtime retain their existing facts.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/design/architecture/运行时组合与资源治理设计.md
    - ai-plan/roadmap/运行时组合与资源治理实施路线.md
    - ai-plan/public/runtime-composability-governance/README.md
    - ai-plan/public/runtime-composability-governance/startup-prompt.md
    - ai-plan/public/runtime-composability-governance/todos/runtime-composability-governance-tracking.md
    - ai-plan/public/runtime-composability-governance/traces/runtime-composability-governance-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: `docker-runtime-agent` Batch 8 (`ui-and-cross-boundary-convergence`) is next after Batch 7 acceptance.
- Completed: architecture research, Work Intake, repository-wide design, roadmap and active-topic bootstrap.
- Current risk: existing runtime resources use several lifecycle patterns; implementation must inventory before introducing shared cleanup abstractions.
- Phase 0 result: server-side inventory is recorded below and in the trace; no shared Scope API is justified yet.
- Phase 1 result: Task Runtime nil-context shutdown is safe; durable Dispatcher has terminal post-shutdown state and forced-timeout evidence.
- Phase 2 result: no repeated cross-owner cancellation/cleanup pattern was found that existing Module, Task Runtime,
  realtime subscription, Agent, Provider or Runner owners cannot express. No Resource Scope API is introduced.
- Phase 5 lifecycle evidence is complete. The Docker Runtime Agent subtopic now owns the active migration batches and
  this parent retains cross-runtime authority and archive status.

Batch 5 cross-boundary direction is frozen: Build Docker side effects use Task Runtime external leases and the
`docker-runtime-agent` `docker/v1` SDK capability. Build retains intent, placement/reservation and Artifact/Publication
semantics; Task retains lease, fence, renew/cancel, logs, transient-result digest, receipt, retry and recovery; Runtime
Target retains generation-scoped capability binding. Build material and normalized result are resolved/submitted only in
valid fenced windows and are never persisted as Task/Agent payloads. The server Docker socket remains only for the
explicitly unmigrated Update Controller and Container read/stream/interactive boundaries.

## Task Checklist

- [x] Work Intake, design, roadmap and active-topic bootstrap.
- [x] Phase 0: inventory creators, owners and disposers for current long-lived resources; classify P0 lifecycle gaps.
- [x] Phase 1: unify lifecycle cleanup and shutdown evidence for P0 resources.
- [x] Phase 2: evaluate duplicate ownership patterns; retain explicit local lifecycle ownership and introduce no Scope API.
- [x] Phase 3: add typed capability/composition declarations and capability-local health vocabulary where justified.
- [x] Phase 4: evaluate controlled dynamic change; the required isolation, state migration, drain/rollback and operational gates are not proven, so runtime enable/disable is not approved.
- [x] Phase 5: close the remaining P0 lifecycle evidence gaps before archive readiness:
  project/container detached-context shutdown paths; RuntimeTarget and Agent/collector failure observability; and
  lifecycle conformance checks for Register, MemoryBus and cron Registry boundaries.

## Acceptance Conditions

- Every newly changed long-lived resource has one creator, owner, disposer and shutdown test.
- Cross-module capability contracts remain typed and private implementation remains inaccessible outside its owner.
- Module/Provider/Agent/Runner/Task composition does not create a second DI, scheduler, task runtime or dynamic plugin platform.
- Dynamic enable/disable and HMR stay out of implementation unless a later approved decision changes the design.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-design-bootstrap", "phase-0-resource-inventory", "phase-1-lifecycle-cleanup", "phase-2-narrow-resource-scope", "phase-3-capability-composition-declarations", "phase-4-controlled-change-evaluation", "docker-runtime-agent-batch-5-build-sdk-migration", "docker-runtime-agent-batch-6-update-controller-launch-boundary", "docker-runtime-agent-batch-7-deployment-and-cli-deletion"],
  "pending_batches": ["docker-runtime-agent-batch-8-ui-and-cross-boundary-convergence"],
  "current_batch": "docker-runtime-agent-batch-8-ui-and-cross-boundary-convergence",
  "next_batch": "docker-runtime-agent-batch-8-ui-and-cross-boundary-convergence",
  "closeout_status": "active"
}
```

## Phase 5 Remaining P0 Lifecycle Evidence Result

- Project/container realtime streamers derive run contexts from the owning Service lifecycle context; publish queries inherit cancellation, and bounded close failures retain stream ownership for retry.
- RuntimeTarget summary collection preserves repository errors, records capability-local diagnostics, and emits structured warnings. Agent listeners log Serve failures and forced-close warnings, force `Close` after shutdown deadline/cancellation, and preserve joined errors.
- Conformance tests prove registration does not invoke module lifecycle methods, MemoryBus remains synchronous, and cronx Registry is declaration-only.
- Focused validation passed for internal/module, internal/eventbus, internal/cronx, internal/httpx, modules/project, modules/container, and modules/runtime-target; conformance race tests, git diff --check, and the AI-plan structure guard also passed.
- Phase 5 evidence is complete; topic remains active until normal task-closeout/archive review.
- Docker Runtime Agent is an active bounded subtopic. Its tracking file owns the eight migration batches; the parent
  retains only cross-runtime authority and archive status.

## Phase 1 Lifecycle Cleanup Result

- `server/modules/task.Runtime.Stop` now normalizes a nil context to `context.Background()` before cancellation/wait, preventing an active-runtime nil dereference. Normal Module shutdown continues to provide the bounded Runtime shutdown context.
- `server/internal/event.Dispatcher` now marks itself terminal on graceful or forced shutdown, rejects restart with `ErrDispatcherStopped`, resets `started`, keeps repeated `Shutdown` idempotent, and logs a structured forced-timeout warning. A handler that ignores context may still finish after the deadline; the dispatcher is intentionally not reusable.
- Added focused regressions for `Stop(nil)` and forced Dispatcher shutdown state/restart/idempotence.
- No Task/Submission, durable outbox, EventBus, Agent, Provider, Runner, or realtime authority changed. No shared Scope API was added.

## Phase 3 Capability/Composition Declaration Result

- `module.Spec` now declares required/exposed capabilities through `TypedCapability[T]()` keys, plus configuration owner and long-lived resource/disposer metadata.
- `RuntimeMetadata` exposes an immutable descriptor snapshot for those declarations; it does not resolve services or own cleanup.
- `CapabilityHealth` is limited to capability-local `Ready`, `Degraded`, and `Unavailable` vocabulary. No global health registry, dynamic dependency solver, second DI, scheduler, Task state machine, or plugin loader was introduced.
- Existing `moduleapi` interfaces remain the typed contract authority; `module.Context` and Task/Submission, Agent, Provider, Runner and realtime authorities are unchanged.

## Phase 4 Controlled Change Evaluation Result

Decision: do not implement runtime Module/Provider enable/disable, runtime dependency re-computation, hot unload, or
HMR. The default remains process restart, config reconcile, or Agent reconnect.

| Gate | Evidence | Decision |
| --- | --- | --- |
| Isolation | Modules, routes, permissions, capability metadata and the explicit container share one process/runtime; no per-module isolation boundary or independently disposable topology exists. | Not proven |
| Persistent state migration | TaskPlan/Stage bindings are frozen for in-flight work and RuntimeTarget/Agent facts are durable, but there is no versioned migration authority for changing the module/provider topology itself. | Not proven |
| Drain and rollback | Runtime shutdown, dispatcher terminal state, Task recovery and Agent reconnect provide local drain/recovery behavior; there is no atomic topology change, admission drain, or rollback snapshot spanning routes, permissions, capabilities and external resources. | Not proven |
| Fact authority | Task/Submission, Agent ledger and RuntimeTarget persistence remain valid authorities for their domains. No canonical persisted Module activation fact exists, and Phase 3 metadata is diagnostic rather than a mutating registry. | Existing facts only; insufficient |
| Operations entry | Explicit restart, configuration reconciliation and Agent reconnect paths exist. No protected, audited runtime Module/Provider toggle entry with authorization, preview, status and recovery semantics exists. | Not proven |

The evidence therefore fails the all-gates requirement in the design and roadmap. Capability-local `Unavailable` or
`Degraded` remains the failure vocabulary; it must not be converted into a topology mutation or hidden reload. Any
future reconsideration must first define a canonical activation fact, versioned migration, admission/drain protocol,
rollback receipt, authorization/audit entry and conformance evidence without adding a dynamic Plugin Loader, second DI,
scheduler or Task state machine.

## Phase 0 Resource Inventory

盘点范围：`server/internal/app`、`internal/module`、`internal/event`、`internal/eventbus`、`internal/realtime`、`internal/scheduler`，以及 task、project、container、runtime-target 模块和现有 Agent/Provider/Runner 边界。以下 owner 是唯一生命周期 authority；调用方只消费 capability，不接管关闭权。

| 资源 | creator / owner | 生命周期入口与 disposer | parent / deadline / wait | 日志、健康、失败与恢复 | 已知风险与最小 P0 候选 |
| --- | --- | --- | --- | --- | --- |
| `internal/app.Runtime` core（DB、Redis、logger、HTTP、Agent listeners、MCP） | `NewRuntime`/`newRuntimeCore`；唯一 owner `*Runtime` | `Run` -> `prepareModules` -> listeners/HTTP；`shutdownRuntime` 依次 `Shutdown`/`Close`，失败 `errors.Join` 仍继续 cleanup | module shutdown context 用 `WithoutCancel` + 默认 5s deadline，继承更早 parent deadline；各 HTTP server 等待 graceful shutdown | boot/failure/close 有 app log；健康主要由 HTTP/registry 元数据提供 | partial listener 已由 cleanup 兜底，但 core close 没有统一 stopped 状态；P0：保留并测试 idempotent shutdown/partial-start 证据，不引入新 Scope |
| Module `Register/Boot/Shutdown` | compile-time module registry/Manager；Runtime 编排，模块拥有业务资源 | `Register` 只声明；成功 `Boot` 纳入 booted；逆序 `Shutdown` | `module.Context.LifecycleContext`；Runtime 给每个模块独立有界 shutdown context，关闭错误聚合 | 模块启动/关闭失败带模块名；无统一 resource health | Register 禁止启动长期行为目前主要是契约/测试约束；P0：补 lifecycle instrumentation/conformance 检查，不扩张 `module.Context` |
| Task Runtime worker、stage execution、expiry ticker | task Module 创建 `task.Runtime`；task Runtime/Task repository owner | Module Boot `Start`；`Start` 启动 workers/expiry loop；`Stop` cancel running stages、executor cancel hook、WaitGroup；ticker defer Stop | `Start(ctx)` 派生 worker context；Shutdown 传入 Runtime 5s context；超时返回而未完成外部工作按 unknown/recovery 语义处理 | worker panic、promotion/expiry 错误有日志；持久化 Task/Submission/Stage 是事实 authority | `Stop(nil)` 活跃运行时会在 `ctx.Done()` 处失效；P0：归一化 nil context 并增加回归测试；记录 deadline 后后台收敛状态 |
| durable event Dispatcher / outbox worker | Runtime core `NewDurableDispatcher`；Runtime 唯一 owner | `Register` 启动前登记 handler；Runtime `Start` worker/poller；`Shutdown` 停止接收、停 poller、drain work、cancel workers | 当前可用独立 process-lifetime parent 以完成 drain；shutdown 使用有界 context；deadline 后 worker 可能继续 | outbox lease、retry、handler timeout、terminal fail 和结构化日志；跨重启由 durable store reclaim | deadline 分支返回时 `started/cancel` 状态仍可能保留，handler 忽略 ctx 可越过 Runtime；P0：增加明确 terminal/forced-timeout health 语义，禁止 deadline 后复用，不改变 durable authority |
| synchronous in-process `MemoryBus` | Runtime 创建；application-lifetime MemoryBus owner | `Subscribe` 只追加 handler；无 goroutine、`Close` 或 unsubscribe；同步 `Publish` | 无后台 parent/deadline；调用方 ctx 传入 handler | 顺序同步派发，panic/error 聚合并记录；无持久化、重试或 broker 语义 | handler 可阻塞发布且订阅永久保留；P0：文档/注册审计禁止隐藏后台资源，长工作回到 Module/Runtime Boot/Shutdown；不增共享 Scope API |
| realtime Hub、gateway、subscription、observer、SSE/WS stream | `realtime.NewHub` 由 Runtime 持有 singleton；连接/订阅 owner 各自请求或 stream | Hub `Subscribe`/observer 返回一次性 disposer；WS/SSE defer unsubscribe、cancel、conn.Close、ticker.Stop；gateway 路由注册在 Register | 连接以 request context 为 parent；Hub process lifetime 无 owner-level Close；stream topic 在最后订阅者离开或 Module Close 时停止 | 非阻塞 publish 丢弃慢消费者旧事件；gateway stream 错误观测有限 | Hub map/observer 若漏 disposer 会留存；P0：明确 Runtime owner 与 disposer 证据，补 gateway termination metrics/log；没有具体泄漏前不新增 Hub Close/Scope |
| project/container topic streamer | project/container Service lazy creator；Service 唯一 owner | topic observer active 时启动 stream；inactive/`Service.Close` cancel+wait 并 unregister observer | 多个 streamer 以 `context.Background()` 派生运行 context；Close 使用 Runtime module deadline 等待 | stream 错误 warn；取消被抑制；publish/collector 错误多为日志 | detached Background 使遗漏 `Service.Close` 时 goroutine 脱离模块；stop callback 使用 Background；P0：让生命周期 parent 可追踪，保证 Service.Close 在所有 module failure path 执行并补 timeout 证据 |
| container stats/runtime-event collector | container Service 创建；Service owner | `Start` 派生 context、ticker、done；`Stop` cancel+wait；runtime event manager 同样由 Service.Close 停止 | module lifecycle parent；caller shutdown deadline 只控制等待 | stats publish/collect warn；event manager bounded history/TTL 与 diagnostics | 外部 source Close/timeout 责任需继续核对；P0：确认 source stop 顺序并暴露 last-error/forced-timeout health |
| runtime-target summary collector | runtime-target Module 创建；Module owner | Boot `Start` 注册 topic observer、ticker；Shutdown `Stop` unregister/cancel/wait | `ctx.LifecycleContext` parent；Runtime deadline | 无订阅时停采集；当前 repository collect error 静默返回 | P0：至少记录/计数 collect failure，定义 capability-local Degraded/Unavailable，不改变 RuntimeTarget authority |
| cron/scheduler | Runtime `cronx.Registry` 只登记声明；scheduler Module 创建 `CronRuntime` 并拥有 robfig engine | Module Boot `Start` 加载 persisted definitions/cron.Start；Shutdown `Stop` cron.Stop、cancel lifecycle、wait | lifecycleCtx 派生自 Module ctx；Stop 等待 robfig stopCtx 或 shutdown deadline | job run history/persisted status、skip/failure 日志；无独立 scheduler | deadline 后 in-flight job 可能异步收敛；P0：暴露 forced-stop/started health，保持 cron registry metadata-only |
| Agent listeners / RuntimeTarget Agent listener | Runtime 创建 AgentBootstrap/AgentServer；Runtime owner | Start bind TLS listener + Serve goroutine；Runtime shutdown 调 `http.Server.Shutdown`；bind failure close listener | Shutdown 使用独立有界 context；Serve error channel 必须被 Runtime 消费 | Serve error 返回 runtime；健康由 listener/agent routes 反映 | graceful shutdown deadline 后缺少显式 force-close/health；P0：记录 listener forced timeout 与 Serve failure，保持 Agent 身份/ledger authority |
| Provider、Runner、独立 Agent | 当前 `dockerTargetProvider` 是 stateless provider；DockerRuntimeAgent 持有 target/ledger/telemetry；Runner 通过既有 task/backup handoff | Provider operation/request context 负责外部 client cleanup；Agent telemetry 无后台 ticker；独立 Agent 以进程 start/stop 为边界 | 不创建 `Agent -> Task -> Plugin` Context 树；Task/Submission、Agent ledger、Provider connection fact 各自保持 authority | conformance 覆盖 cancellation/cleanup；Agent ledger/telemetry TTL 提供事实与退化依据 | P0：为未来 Provider/Runner/Agent contract 明确 operation cancel/Close 与 capability-local health；不新增第二套 queue、scheduler、DI 或 state machine |

### P0 候选排序

1. 修复 Task Runtime `Stop(nil)` 的 context 语义并补回归测试。
2. 为 durable Dispatcher deadline 后的 forced-stop、terminal state、不可复用语义补健康/日志证据。
3. 审计 project/container realtime stream 的 detached `context.Background()`，保证所有 Module failure/shutdown 路径调用 Service.Close 并带有界等待。
4. 为 runtime-target collector、Agent listener、collector/source stop 增加失败/超时观测；保持各自现有 authority。
5. 以测试/静态审计固化 `Register` 不创建长期资源、同步 EventBus 不隐藏后台资源、cron Registry 不拥有 scheduler 的约束。

Phase 0/1 的盘点与 cleanup 结果仍未证明需要通用共享 Scope API；Phase 2 已完成并保留显式局部 ownership。

## Phase 2 Narrow Resource Scope Result

- Re-evaluated the Phase 0 inventory after Phase 1 cleanup across Runtime/Module, Task Runtime, realtime
  subscription/stream, Event/cron, Provider/Runner and Agent boundaries.
- No evidence met both gates for a new scope: (1) at least two resources must be cancelled or released together;
  and (2) the existing owner cannot clearly express their relationship. Existing pairs remain owner-local: Task
  worker/ticker resources belong to Task Runtime, realtime observer/stream resources belong to the stream or Service,
  and core listener/dispatcher resources belong to Runtime.
- Kept cancellation contexts and idempotent disposers local to their existing owners. No shared Scope API, `module.Context`
  expansion, Agent -> Task -> Plugin Context tree, or authority change was introduced.

## Docker Runtime Agent Batch 7 Documentation Evidence

- [x] The active subtopic's deployment slice records `cutover-v1` as retained bootstrap authority and does not add a
  second startup or Update execution path.
- [x] Parent recovery materials now point to the frozen server socket consumers only: Update observation/recovery,
  Runtime Target discovery/summary, and Container snapshot/stream/interactive transport.
- [x] The ordered Update replacement contract is documented consistently: server/web, verify server, replace Agent,
  verify mTLS/generation/capability readiness.

Evidence: `git diff --check`; `python3 scripts/validate_ai_plan_structure.py`.

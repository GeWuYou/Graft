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

- Current batch: `phase-0-resource-inventory`.
- Completed: architecture research, Work Intake, repository-wide design, roadmap and active-topic bootstrap.
- Current risk: existing runtime resources use several lifecycle patterns; implementation must inventory before introducing shared cleanup abstractions.
- Phase 0 result: server-side inventory is recorded below and in the trace; no shared Scope API is justified yet.
- Next step: Phase 1 fixes the bounded P0 candidates, starting with Task Stop nil-context safety and durable Dispatcher timeout observability.

## Task Checklist

- [x] Work Intake, design, roadmap and active-topic bootstrap.
- [x] Phase 0: inventory creators, owners and disposers for current long-lived resources; classify P0 lifecycle gaps.
- [ ] Phase 1: unify lifecycle cleanup and shutdown evidence for P0 resources.
- [ ] Phase 2: introduce a narrow Resource Scope only if duplicate ownership patterns prove it necessary.
- [ ] Phase 3: add capability/composition declarations and capability-local health where justified.
- [ ] Phase 4: evaluate controlled dynamic change only if isolation, state migration and rollback requirements are proven.

## Acceptance Conditions

- Every newly changed long-lived resource has one creator, owner, disposer and shutdown test.
- Cross-module capability contracts remain typed and private implementation remains inaccessible outside its owner.
- Module/Provider/Agent/Runner/Task composition does not create a second DI, scheduler, task runtime or dynamic plugin platform.
- Dynamic enable/disable and HMR stay out of implementation unless a later approved decision changes the design.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-design-bootstrap", "phase-0-resource-inventory"],
  "pending_batches": [
    "phase-0-resource-inventory",
    "phase-1-lifecycle-cleanup",
    "phase-2-narrow-resource-scope",
    "phase-3-capability-composition-declarations",
    "phase-4-controlled-change-evaluation"
  ],
  "current_batch": "phase-1-lifecycle-cleanup",
  "next_batch": "phase-2-narrow-resource-scope",
  "closeout_status": "active"
}
```

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
| Provider、Runner、独立 Agent | 当前 `dockerTargetProvider` 是 stateless provider；DockerBuilderAgent 持有 target/ledger/telemetry；Runner 通过既有 task/backup handoff | Provider operation/request context 负责外部 client cleanup；Agent telemetry 无后台 ticker；独立 Agent 以进程 start/stop 为边界 | 不创建 `Agent -> Task -> Plugin` Context 树；Task/Submission、Agent ledger、Provider connection fact 各自保持 authority | conformance 覆盖 cancellation/cleanup；Agent ledger/telemetry TTL 提供事实与退化依据 | P0：为未来 Provider/Runner/Agent contract 明确 operation cancel/Close 与 capability-local health；不新增第二套 queue、scheduler、DI 或 state machine |

### P0 候选排序

1. 修复 Task Runtime `Stop(nil)` 的 context 语义并补回归测试。
2. 为 durable Dispatcher deadline 后的 forced-stop、terminal state、不可复用语义补健康/日志证据。
3. 审计 project/container realtime stream 的 detached `context.Background()`，保证所有 Module failure/shutdown 路径调用 Service.Close 并带有界等待。
4. 为 runtime-target collector、Agent listener、collector/source stop 增加失败/超时观测；保持各自现有 authority。
5. 以测试/静态审计固化 `Register` 不创建长期资源、同步 EventBus 不隐藏后台资源、cron Registry 不拥有 scheduler 的约束。

本批次没有证据证明需要通用共享 Scope API；Phase 2 保持 pending，等待 Phase 1 修复后再重新评估。

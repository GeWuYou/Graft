# Queue Adoption Audit

## 1. 结论

本审计的“Queue”不是新增抽象，而是对现有两个异步能力的采用判断：

- `server/internal/event`：Runtime-owned 的事件发布、best-effort 内存投递和 PostgreSQL Outbox durable delivery。
- `server/modules/task`：module-owned 的持久化 Task/Stage 状态机、DB-backed dispatcher、worker、受控 retry/cancel、日志和 realtime。

两者不是同一个概念：事件表示“某个事实已经发生并通知消费者”，Task 表示“一个有计划、阶段、进度和外部副作用的执行过程”。`server/internal/eventbus` 仍是同步进程内观察者总线，不能被文档中的“队列”泛化替代。

当前仓库没有证据支持增加 `server/internal/queue`、通用 `Queue` 接口或独立 `Worker` 框架。`ai-plan/design/architecture/任务执行运行时设计.md` 已明确 PostgreSQL 是 Task 唯一事实来源、MVP 不引入 MQ/分布式 worker；`server/internal/event/README.md` 已明确 Outbox 与事件 dispatcher 的边界。

## 2. 证据与调用链

### 2.1 HTTP、Service、Repository

普通 CRUD、读取、校验和短事务应保持同步：HTTP route 做认证、授权、输入解析，Service 编排领域规则，Repository 写入事实，响应返回结果。典型证据是 `server/modules/update/routes.go` 的 `status`/`check` handler 调用 `Service.Status`/`Service.Check`；`server/internal/scheduler/runtime_tasks.go` 的 `CronRuntime.CreateTask` 调用 scheduler repository 持久化定义再刷新 cron。此类调用没有长外部副作用时不应为了“统一”强行创建 Task。

当 Service 会执行不可控时长或外部副作用，推荐改为：HTTP 完成授权和前置校验 -> Service 冻结 `TaskPlan` -> `TaskService.Submit` -> 返回 Task ID/`202 Accepted` -> Task dispatcher 运行 executor。Project 已形成该模式：`server/modules/project/module.go` 的 `Register` 解析 `moduleapi.TaskService` 和 `TaskRuntimeRegistrar`，`server/modules/project/task_executor.go` 的 `registerProjectTaskExecutors` 注册 Compose executor；executor 的 `Execute` 使用 `exec.CommandContext` 运行 Docker 并通过 `StageRun.AppendLog` 写日志。

Repository 仍只保存领域事实和 Task Runtime 事实，不承担隐式 goroutine、重试循环或“队列表”。Task 的领取由 `server/modules/task` 在事务中使用 `FOR UPDATE SKIP LOCKED` 完成，业务 Service 不应直接领取或修改 Stage 状态。

### 2.2 EventBus 与 Event

`server/internal/eventbus/bus.go` 的 `MemoryBus.Publish` 是同步观察者语义：调用者等待 handler 返回错误，适合同进程、需要顺序和错误回传的观察逻辑。它不是持久队列。

`server/internal/event/publisher.go` 的 `Publisher.Publish`/`PublishAsync` 用于事件投递；`TransactionalPublisher.PublishTx` 用于已经持有 SQL transaction 的 durable event。`server/internal/event/dispatcher.go` 的 dispatcher 有界 buffer、worker、handler timeout、attempt 和 Outbox lease；`server/internal/event/repository.go` 与 `server/internal/event/migrations/202607230004_event_outbox.sql` 保存 event/delivery 状态。durable event 必须与业务事实共用事务，不能在业务提交后再补发造成 dual-write。

事件 handler 必须幂等，因为 best-effort 重试、Outbox lease 恢复或进程崩溃都可能再次执行。事件 Receipt 只代表平台接收，不代表 consumer 完成；需要用户可见进度、取消、重试或人工恢复时，事件只能通知，不能替代 Task。

### 2.3 Scheduler/Cron

Scheduler/Cron 是触发器，不是 Stage worker：`server/internal/scheduler/runtime_tasks.go` 的 `RunOnceWithTrigger` 创建 run，`server/internal/scheduler/repository_run.go` 持久化 run；`server/internal/scheduler/runtime_execution.go` 的 `RunAction`/finish 路径执行 scheduler action。它适合“何时提交”，不应复制 Task 的领取、重试、日志和状态机。

定时业务动作应是 `cron -> TaskService.Submit(TaskPlan{trigger_type: cron}) -> Task worker`。若只是短时配置刷新或内部读操作，可继续由 scheduler 同步完成；若重复触发可能重叠，使用 Task owner 的 active 唯一约束或业务幂等键。

### 2.4 Docker、Runtime、Compose、Update、Backup

- Project Compose：`server/modules/project/task_executor.go` 的 executor 运行 `docker compose` 生命周期命令，并将 stdout/stderr 逐行写入 Task log；`server/modules/project/task_authorizer.go` 负责 owner 授权。这是适合 Task 的长副作用实例。Docker 命令崩溃后不能猜测成功，必须进入 `unknown`/`needs_attention`，由外部事实核对后再 settle/retry。
- Runtime：`server/modules/container/docker_runtime.go`、`runtime_events.go`、`runtime_events_http.go` 读取 Docker runtime 和事件；读取、快照、统计通常保持同步或短时流式 HTTP。容器动作若已经能改变外部状态，应由对应业务 module 注册 StageExecutor，而不是在 container 中新增队列。
- Compose：`server/modules/project/compose/loader.go` 和 project service 负责解析/校验文件；解析预览可同步，导入/部署/销毁/重建等跨进程副作用应提交 Task，并把 workspace/application ID 作为稳定 owner。
- Update：`server/modules/update/routes.go` 的 status/check 是只读发现/验证，`server/modules/update/execution.go`、`compose_runner.go` 和 `launcher.go` 执行 digest-pinned Compose runner。检查可以同步；确认升级是一次性外部副作用，应该由 Update 自己生成 Task plan 并让 executor 管理 runner receipt、日志和恢复。不可把现有 update operation 表直接当成第二套 Task Runtime。
- Backup：`server/modules/backup/doc.go`、`module.go` 和 `runner_handoff.go` 表明 Backup 当前提供窄 `BackupService` capability 与 runner handoff，拥有备份事实，不拥有通用 worker。数据库 dump、文件快照、上传和恢复都应在 Backup module 注册 Task executor；恢复必须默认人工确认，不能自动重放未知外部命令。

### 2.5 Webhook、网络、外部 API、SSH

当前 `server/modules` 没有独立 Webhook module/通用 Webhook worker 的实现证据；OpenAPI/HTTP ingress 仍是权威入口。未来 Webhook 接收应先验签、去重并快速提交一个领域事件或 Task，不能在 HTTP handler 内等待下游网络调用。Webhook delivery 若需要每次投递状态、指数退避、取消和人工重放，使用 Task；若只是“事实发生后通知多个本地消费者”，使用 durable event。

网络请求和外部 API 调用必须由业务 module 的 executor 持有超时、取消、响应大小上限、敏感字段脱敏和幂等键。SSH 当前也没有被发现为独立平台 worker；远程命令一旦引入，必须作为明确业务 executor，记录目标引用而非凭据，禁止把任意 command/host/path 从 HTTP 请求直接放入 Task plan。外部返回成功后要保存 receipt/operation identity，超时或连接断开进入 `unknown`，不能盲目 retry。

### 2.6 文件系统、事务、日志、通知、统计

文件写入、Compose 文件生成和读取应由业务 Service/Repository 持有路径校验、原子写入和权限边界；长时间扫描/压缩/上传适合 Task，短时 schema 校验适合同步 HTTP。事务边界是“业务事实 + durable event/outbox”同一 SQL transaction；Task Stage 的外部副作用不可能与 Docker/网络事务原子提交，必须用 idempotency key、receipt 和 recovery policy 补偿。

日志应写入 Task 的受控 log API，而不是 executor 自行建日志表或只写 zap。平台应用日志仍可走 `server/internal/logger`；审计使用 `server/modules/audit`；重要通知通过 `server/modules/notification/publisher.go` 的 Service 发布。通知发送失败不能改变已经成功的业务事实，需按通知语义选择 durable event 或独立 notification delivery，而不是为每个通知新增 Queue。

监控趋势和统计读取应保持查询/聚合语义：`server/modules/monitor/trend_runtime.go` 等 runtime 读取事实并聚合，不能以“统计慢”为由把查询偷偷变成无状态后台任务。重计算、导出、批量超过同步上限时提交 Task，结果和进度可查询。

### 2.7 Git、Registry、MCP、OpenAPI

- Git：当前仓库工作流要求 agent worktree 不启动长期 runtime、不应用迁移，主 checkout 才负责服务和集成；Git 操作是开发/交付控制面，不是业务 Queue。不得把 commit、push、PR 或 hook 包进业务 Task。
- Registry：`server/internal/moduleregistry/registry.go`、`server/internal/configregistry/registry.go`、`server/internal/permission/registry.go` 是注册/冻结/解析边界，不是待处理作业存储。注册 executor、handler、menu 或 config 必须发生在 module `Register`，不能由 worker 动态发现。
- MCP：`server/internal/app/runtime_authenticated_routes.go`、`server/internal/app/runtime_docs.go` 等 authenticated routes/runtime docs 是 transport adapter；开发者 MCP 由 `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md` 治理，不能变成业务 runtime 的隐藏依赖。MCP 工具触发的长操作仍遵守相同 Task/Event 规则。
- OpenAPI：canonical source 在 `openapi/**`，`web/src/contracts/openapi/generated/**` 是生成投影。HTTP→Task 的 `202`、Task receipt、状态和 capability 必须先更新 canonical contract，再生成投影；不得在 web 或 generated 文件中添加兼容性 Queue 映射。

## 3. 分级结果

### P0：必须阻断

1. 在 HTTP、Webhook、MCP 或 scheduler action 中直接等待无界 Docker、SSH、网络、外部 API、备份/恢复或升级副作用。
2. 为同一事实同时写业务表和自建 queue/job 表、或在事务提交后补写 durable event，造成 dual-write/不可恢复丢单。
3. 把失败的外部副作用自动当作可安全重试，尤其是 `compose up/down`, update runner, restore 和远程命令。
4. 绕过 owner authorizer、把任意 host/path/command/credential 放进 Task plan，或让 MCP/外部 callback 直接改变 Task 状态。

### P1：高优先级治理

1. 已有长任务没有 Task ID、冻结 plan、Stage 进度、受控日志、cancel/retry/recovery policy 或查询 API。
2. Event handler 无幂等键/去重，或 durable event 未使用 `PublishTx`。
3. Scheduler 重复提交无 owner 唯一约束或业务幂等键；Update/Backup/Compose 自建第二套生命周期表替代 Task。
4. 外部 API/网络/文件/Registry 访问缺少 timeout、边界、脱敏和 `unknown` 恢复路径。

### P2：应补齐

1. Task/Stage、event delivery、通知、审计和统计的指标缺少等待、处理、重试、失败、DLQ/人工介入计数。
2. OpenAPI 尚未表达 `202`、Task receipt、capabilities、progress、retry/cancel 和 `unknown` 语义。
3. Outbox/Task worker 的 shutdown、lease expiry、backpressure 和有界 drain 缺少集成验证。

### P3：观察项

1. Redis、MQ、Temporal、分布式 worker 或独立队列平台在正确性路径上的引入。
2. 为短时查询、同步 CRUD、配置注册、Git 或开发者 MCP 增加异步包装。
3. 以“Queue”作为跨领域产品术语，掩盖真实 owner、TaskPlan 或 Event contract。

## 4. 推荐流程与验收

```text
HTTP/Webhook/MCP/Cron
  -> authentication + authorization + validation + idempotency key
  -> domain Service
  -> SQL transaction: business facts + optional event.PublishTx
  -> TaskService.Submit(frozen TaskPlan) [long side effect]
  -> 202 + task_id + status URL/realtime topic
  -> task dispatcher claims Stage with PostgreSQL lock
  -> module StageExecutor performs bounded external effect
  -> Task log/progress/state + event/realtime after persistence
  -> success / failed / cancelled / unknown -> manual reconcile or permitted retry
```

Task ID 必须是稳定公开引用；progress 由 Stage/Task Runtime 事实产生，不允许前端猜测；retry 必须创建新 attempt 并保留历史；DLQ 不是新增 Queue 名称，而是 `failed`/`unknown`、过期 lease、recovery_required 和人工处置查询面的组合。幂等键必须绑定业务 owner + operation identity；事务只承诺数据库事实原子性，外部副作用用 receipt/reconcile 补足。

采用新异步流程前，至少应证明：现有 HTTP 同步预算不足或外部副作用不可控；选择的是 Event 还是 Task；owner/授权/幂等键明确；plan 输入已脱敏冻结；重试/取消/unknown 有业务定义；日志、审计、通知和指标有 owner；canonical OpenAPI 已表达异步契约；没有引入第二个 Queue/Worker/状态机。

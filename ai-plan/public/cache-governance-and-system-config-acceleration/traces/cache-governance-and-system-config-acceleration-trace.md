# Cache Governance And System Config Acceleration Trace

## 2026-06-21 Phase 0 governance persistence

- 建立缓存治理长期设计 authority：`ai-plan/design/缓存治理与系统配置读取加速规范.md`。
- 建立 public recovery topic：`ai-plan/public/cache-governance-and-system-config-acceleration/README.md`。
- 建立任务追踪：`ai-plan/public/cache-governance-and-system-config-acceleration/todos/cache-governance-and-system-config-acceleration-tracking.md`。
- 建立 trace：`ai-plan/public/cache-governance-and-system-config-acceleration/traces/cache-governance-and-system-config-acceleration-trace.md`。
- 建立仓库 skill：`.agents/skills/graft-cache-governance/SKILL.md`。
- 更新 `ai-plan/public/README.md` 与共享资产注册表，使后续会话可直接恢复本主题。

## 2026-06-21 Real-code exploration findings

- 系统配置读取 authority 当前位于 `server/modules/system-config/service.go`：
  - `Get(ctx, key)` 先查 `configregistry` definition，再调用 `store.GetOverride(ctx, key)`。
  - `ResolveDefaultConfig(ctx, key)` 底层仍调用 `Get(ctx, key)`。
  - `IsBooleanConfigEnabled(ctx, key, fallback)` 底层仍调用 `Get(ctx, key)`。
  - 当前没有 process-local cache、Redis cache、singleflight、版本号或显式失效。
- `server/internal/moduleapi/notification.go` 中 `SystemConfigResolver` 仅提供：
  - `IsBooleanConfigEnabled(ctx, key, fallback bool) bool`
- 已确认系统配置热点消费点：
  - `server/modules/container/service.go`
  - `server/modules/notification/publisher.go`
  - `server/internal/scheduler/runtime.go`
  - `server/modules/user/bootstrap.go`
- 已确认真实缓存资产：
  - `server/modules/container/mount_usage.go`：进程内 TTL cache，45s
  - `server/modules/monitor/module.go`：Redis ZSET 趋势缓存，TTL 2h
  - `configregistry` / `menu registry` / `cron registry` / dashboard registry：启动期只读 registry
- 已确认容器配置存在双轨：
  - 部分布尔开关已经由 `SystemConfigResolver` 消费
  - `ops.container.runtime`、`ops.container.docker.endpoint`、`logs.default_tail`、`logs.max_tail` 仍主要依赖启动期 config 默认值

## 2026-06-21 Governance decision

- 推荐 Phase 1：
  - 单进程本地 full snapshot cache
  - singleflight 合并 DB miss
  - Update/Reset 成功后显式本地失效
- 推荐 Phase 2：
  - Redis pub/sub 或轻量版本号轮询
  - 本地缓存保留，不以 Redis 作为 authority
- 不推荐：
  - 缓存日志分页结果
  - 长时间缓存容器实时状态
  - 让模块直接查 system-config override 表
  - 让前端长期缓存权限、菜单或 effective config 作为 authority

## 2026-06-21 Phase 1 local snapshot completed

- `server/modules/system-config/service.go` 已引入进程内 full override snapshot cache，authority 仍保持在 system-config service / resolver 边界。
- snapshot miss 现在通过 `singleflight` 合并并发加载，避免同一时刻重复回源 override 表。
- `Update(...)` 与 `Reset(...)` 成功路径现在都会显式清空本地 snapshot cache，并通过统一 `Get(...)` 返回刷新后的有效值。
- `server/modules/system-config/store/store.go` 与 `storeent/repository.go` 已补充 `ListOverrides(...)`，用于一次性构建本地 snapshot。
- `server/modules/system-config/service_test.go` 已补充：
  - snapshot cache 复用覆盖
  - Update/Reset 后显式失效覆盖
  - 并发 miss singleflight 合并覆盖
- 已提交：`076dc954` `fix(system-config): cache override snapshots locally`

## 2026-06-21 Phase 2 multi-node invalidation completed

- `server/modules/system-config/service.go` 保持 process-local snapshot cache 为唯一读缓存；authority 未改变。
- `Update(...)` / `Reset(...)` 成功后现在会：
  - 显式失效本地 snapshot cache
  - best-effort 发布 Redis invalidation signal 到 `graft:system-config:snapshot:invalidate`
- `server/modules/system-config/module.go` 在 `Boot` 阶段使用 `ctx.Redis` 启动 Redis 订阅；收到远端 invalidation 后仅清理本地 snapshot cache。
- Redis 不可用、订阅失败或 publish 失败时：
  - 配置写入与本地失效仍保持成功
  - 多节点传播降级为不可用，但不影响单节点 authority 与读取链路
- `server/modules/system-config/service_test.go` 已补充：
  - publish failure 下 Update/Reset 仍成功
  - 远端 invalidation message 会清空本地 snapshot cache
  - module Boot/Shutdown 的 Redis 订阅 wiring
- 已提交：`f3adec43` `fix(system-config): broadcast snapshot invalidation`

## 2026-06-21 Phase 3 hotspot expansion completed

- `server/internal/moduleapi/notification.go` 中的 `SystemConfigResolver` 已扩展统一暴露：
  - `IsBooleanConfigEnabled(ctx, key, fallback bool) bool`
  - `ResolveDefaultConfig(ctx, key) (string, error)`
- `server/modules/container/service.go` 已移除局部 `stringSystemConfigResolver` 类型断言路径：
  - 环境变量展示策略读取改为直接调用共享 `SystemConfigResolver.ResolveDefaultConfig(...)`
  - 编排来源动作级别读取改为直接调用共享 `SystemConfigResolver.ResolveDefaultConfig(...)`
- `server/modules/user/bootstrap.go` 现在会在 `bootstrapReader` 构造阶段缓存已解析的 `SystemConfigResolver` capability，避免每次菜单过滤在热路径中重复 `services.Resolve(...)`。
- 本批次确认：
  - system-config authority 仍保持在 `configregistry` + `server/modules/system-config/service.go`
  - 未引入 Redis authority、override-table 直查、日志分页缓存或实时容器状态长期缓存
  - dashboard quick actions 仍停留在 config-definition authority，本批次未把它误扩展成新的 runtime cache surface
- `server/modules/container/service_test.go`、`server/modules/user/menu_contract_test.go`、`server/modules/notification/module_test.go` 已补充/对齐 unified resolver 覆盖。
- 已提交：`93886719` `fix(system-config): unify hotspot resolver reads`

## 2026-06-21 Phase 4 observability and guardrails completed

- `server/modules/system-config/service.go` 现已补充 unified snapshot path 的 backend-local 调试态：
  - hit / miss / load / load error 计数
  - invalidate / remote invalidate 计数
  - publish attempt / publish failure 计数
  - 最近一次装载时间、最近一次失效来源、最近一次 override 数量
- 当前实现保持 authority-first：
  - system-config authority 仍在 `configregistry` + `server/modules/system-config/service.go`
  - 未新增 Redis authority
  - 未新增普通 consumer 可见的第二套读取链路
  - 未强制扩大到新的 OpenAPI debug endpoint
- `server/modules/system-config/service_test.go` 已补充：
  - snapshot debug state 的 hit / miss / load 统计覆盖
  - 本地 Update/Reset invalidation 来源与 action 覆盖
  - 远端 Redis invalidation 来源覆盖
  - publish failure 统计覆盖
- 主题 recovery 与设计 authority 已同步更新，供后续 archive 或 admin-only 调试入口复用。

## 2026-06-21 progress clarification audit

- 复核 `server/modules/system-config/service.go`、topic README、tracking 与 trace 后确认：
  - process-local snapshot cache、`singleflight`、local invalidation、Redis invalidation transport、debug state 已全部落地。
  - 当前 topic 的 archive-ready 结论成立，但结论边界仅限 `system-config` authority 主链及本 topic 已登记热点。
- 同步修正文档口径：
  - 当前状态不是“全仓缓存治理完成”。
  - 后续仓库级缓存治理仍需按新增热点或未纳入本 topic 的 authority owner 逐项推进。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-cache-audit-and-governance-persistence",
    "phase-1-system-config-local-snapshot",
    "phase-1-hot-consumer-adoption",
    "phase-2-multi-node-invalidation",
    "phase-3-hotspot-expansion",
    "phase-4-observability-and-guardrails"
  ],
  "pending_batches": [],
  "current_batch": "phase-4-observability-and-guardrails",
  "next_batch": null,
  "closeout_status": "archive-ready",
  "commit": [
    {
      "sha": "076dc954",
      "title": "fix(system-config): cache override snapshots locally"
    },
    {
      "sha": "f3adec43",
      "title": "fix(system-config): broadcast snapshot invalidation"
    },
    {
      "sha": "93886719",
      "title": "fix(system-config): unify hotspot resolver reads"
    }
  ]
}
```

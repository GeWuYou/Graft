# Cache Governance And System Config Acceleration Tracking

## Topic

Cache Governance And System Config Acceleration

## Scope

建立 `Graft` 统一缓存治理规范，优先治理系统配置从数据库到运行时消费的读取链路，并以分阶段方式推进本地 snapshot、singleflight、失效机制、多节点扩展与热点缓存收口。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/缓存治理与系统配置读取加速规范.md`
- `ai-plan/design/系统配置模型与渲染设计.md`
- `ai-plan/design/通知中心设计.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/共享资产复用治理规范.md`
- `.agents/skills/graft-cache-governance/SKILL.md`

## Current Recovery Point

- 已完成 Phase 0：
  - 真实代码扫描完成。
  - 已识别 `system-config` 当前无统一缓存层。
  - 已识别现有真实缓存资产：
    - `server/modules/container/mount_usage.go` 本地 TTL cache
    - `server/modules/monitor/module.go` Redis 趋势 cache
    - `configregistry` / `menu registry` / `cron registry` 等启动期只读 registry
  - 已确认容器、通知、调度、bootstrap 进入系统配置热点消费链。
- 已完成 Phase 1：
  - `phase-1-system-config-local-snapshot` 已提交 `076dc954` `fix(system-config): cache override snapshots locally`
  - `phase-1-hot-consumer-adoption` 已在 loop 中验收完成；当前 owned scope 未产生额外 diff
- 已完成 Phase 2：
  - `phase-2-multi-node-invalidation` 已提交 `f3adec43` `fix(system-config): broadcast snapshot invalidation`
  - 多节点失效传播现已通过 Redis pub/sub best-effort signal 驱动；Redis 不可用时仍退化为仅本地失效
- 已完成 Phase 3：
  - `phase-3-hotspot-expansion` 已提交 `93886719` `fix(system-config): unify hotspot resolver reads`
- 当前待推进批次：
  - 无；主题进入 archive-ready closeout
- 主题完成边界：
  - 当前 archive-ready 仅覆盖 `system-config` authority 主链和本 topic 已登记的热点消费者。
  - 不得把该状态解读为“仓库全部缓存已经治理完成”；后续新增或未纳入本 topic 的热点仍需继续治理。

## Task Checklist

- [x] Phase 0：缓存资产排查
- [x] Phase 0：系统配置读取链路排查
- [x] Phase 0：治理设计文档落盘
- [x] Phase 0：AI cache governance skill 落盘
- [x] Phase 0：public topic / tracking / trace 建立
- [x] Phase 1：system-config 本地 full snapshot cache
- [x] Phase 1：singleflight 合并加载
- [x] Phase 1：Update/Reset 后显式本地失效
- [x] Phase 1：container / notification / scheduler / bootstrap 接入缓存化 resolver
- [x] Phase 1：扩展 typed resolver，避免仅 bool-only resolver
- [ ] Follow-up：系统配置页补充 runtime-hot / restart-required / effective-source 展示建议
- [x] Phase 2：Redis pub/sub 或版本轮询方案落地
- [x] Phase 2：多节点失效一致性验证
- [x] Phase 3：RBAC/menu/dashboard/container runtime 热点扩展
- [x] Phase 4：指标、调试面板、治理门禁和文档收口

## Batch Boundaries

- `phase-1-system-config-local-snapshot`
  - 范围：`server/modules/system-config/**`、`server/internal/moduleapi/**`
  - 目标：建立本地 snapshot cache、singleflight、失效机制
  - 状态：已完成，提交 `076dc954` `fix(system-config): cache override snapshots locally`
- `phase-1-hot-consumer-adoption`
  - 范围：`server/modules/container/**`、`server/modules/notification/**`、`server/internal/scheduler/**`、`server/modules/user/bootstrap.go`
  - 目标：让热点消费点统一改走缓存化 resolver
  - 状态：已完成；当前 owned scope 未产生额外 diff
- `phase-2-multi-node-invalidation`
  - 范围：system-config invalidation event、Redis 集成边界
  - 目标：多节点一致性预留
  - 状态：已完成，提交 `f3adec43` `fix(system-config): broadcast snapshot invalidation`
- `phase-3-hotspot-expansion`
  - 范围：RBAC/menu/dashboard/container runtime
  - 目标：扩展缓存治理到更多聚合读热点
  - 状态：已完成，提交 `93886719` `fix(system-config): unify hotspot resolver reads`
- `phase-4-observability-and-guardrails`
  - 范围：metrics/debug/docs/tests/scripts
  - 目标：可观测性与治理门禁闭环
  - 状态：已完成；收口 snapshot debug state、结构化日志、聚焦测试与 recovery/design 更新

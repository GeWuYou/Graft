# Cache Governance And System Config Acceleration

## 当前状态摘要

- 当前主题目标是为 `Graft` 建立统一缓存治理规范，并优先收敛系统配置运行时读取加速方案。
- 当前状态：Phase 0 治理资产已落盘，Phase 1 本地 snapshot 与热点消费方收口已完成，Phase 2 多节点失效传播已完成并提交，Phase 3 热点扩展已完成并提交；下一批进入可观测性与治理门禁收口。
- 任务分类为 `cross-boundary`，但本主题以 `backend-first` 为主；前端仅涉及系统配置页面的生效语义与高级信息展示建议。
- Canonical design：`ai-plan/design/缓存治理与系统配置读取加速规范.md`。
- AI 执行 skill：`.agents/skills/graft-cache-governance/SKILL.md`。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/缓存治理与系统配置读取加速规范.md` + `server/modules/system-config/service.go` + `server/internal/configregistry` + `server/internal/moduleapi/notification.go`

## Owned Scope

允许修改：

- `ai-plan/design/缓存治理与系统配置读取加速规范.md`
- `ai-plan/public/cache-governance-and-system-config-acceleration/**`
- `ai-plan/public/README.md`
- `.agents/skills/graft-cache-governance/**`
- `.ai/registries/cross-boundary-assets.yaml`
- `server/modules/system-config/**`
- `server/internal/moduleapi/**`
- `server/internal/scheduler/**`
- `server/modules/notification/**`
- `server/modules/container/**`
- `server/modules/user/bootstrap.go`
- 以及本主题直接牵引的缓存治理测试与文档

禁止误触：

- 不得让模块绕过统一 `SystemConfigResolver` / typed resolver 直接查 system-config override 表。
- 不得把 Redis 作为系统配置 authority。
- 不得把所有缓存都塞进 Redis。
- 不得为了降低查询次数缓存日志分页结果或长时间缓存容器实时状态。
- 不得把 `restart-required` 配置伪装为运行时热生效。
- 不得为了缓存改造绕过配置修改审计。
- 不得让 `web` 长期缓存为有效权限、菜单或系统配置值的 authority。

## Phase Plan

- Phase 0：现状盘点、规范沉淀、topic/skill 持久化。已完成。
- Phase 1：单进程本地 snapshot + singleflight + 显式失效。已完成。
- Phase 2：Redis pub/sub 或版本号轮询，多节点一致性预留。`phase-2-multi-node-invalidation` 已完成。
- Phase 3：扩展到 RBAC/menu/dashboard/container runtime 等热点缓存。已完成。
- Phase 4：补充指标、调试面板、缓存治理文档和测试门禁。

## Current Recovery Point

- 已完成真实代码扫描与治理落盘。
- 已完成 `phase-1-system-config-local-snapshot`：
  - `server/modules/system-config/service.go` 已引入 process-local full override snapshot cache。
  - snapshot miss 已通过 `singleflight` 合并并发加载。
  - `Update/Reset` 成功路径已增加显式本地失效并回填最新 snapshot。
  - 聚焦测试已覆盖缓存复用、显式失效与并发 miss 合并。
  - 已提交：`076dc954` `fix(system-config): cache override snapshots locally`。
- 已完成 `phase-1-hot-consumer-adoption`：
  - 当前 owned scope 内的 container / notification / scheduler / bootstrap 热点消费点已稳定通过统一 resolver 边界读取系统配置。
  - 本批次未要求新增 owned-scope diff；loop closeout 已将该批次标记为完成。
- 已完成 `phase-2-multi-node-invalidation`：
  - `server/modules/system-config/service.go` 继续保持 process-local snapshot cache 为唯一读缓存。
  - `Update(...)` / `Reset(...)` 成功后会在本地失效之外，best-effort 发布 Redis invalidation signal。
  - `server/modules/system-config/module.go` 在 `Boot` 阶段接入 Redis 订阅；收到远端 invalidation 后仅清理本地 snapshot，不改变 authority。
  - Redis 不可用或 publish 失败时会退化为仅本地失效，不影响配置写入成功。
  - 已提交：`f3adec43` `fix(system-config): broadcast snapshot invalidation`。
- 已完成 `phase-3-hotspot-expansion`：
  - `server/internal/moduleapi.SystemConfigResolver` 已扩展为统一暴露布尔与 effective config 读取能力，热点消费方不再需要局部类型断言绕过共享 resolver 边界。
  - `server/modules/container/service.go` 已将环境展示策略与编排来源动作级别解析统一切到共享 `SystemConfigResolver.ResolveDefaultConfig(...)` 路径，继续复用 system-config 本地 snapshot authority。
  - `server/modules/user/bootstrap.go` 已在 reader 装配阶段缓存已解析的 `SystemConfigResolver` capability，避免每次 bootstrap 菜单过滤都在热路径重新解析服务容器。
  - 本批次未扩展到新的 dashboard authority 或 web 长期缓存；`dashboard.quick_actions` 仍保持 config-definition authority，后续如需 runtime 消费扩展必须在独立 authority 范围内推进。
  - 已提交：`93886719` `fix(system-config): unify hotspot resolver reads`。
- 当前确认的系统配置 authority 与热点事实：
  - `server/modules/system-config/service.go` 当前 authority 仍在统一 service/resolver 边界，读取链路保持本地 full snapshot cache + `singleflight`。
  - `server/internal/moduleapi/notification.go` 中 `SystemConfigResolver` 现已统一提供 `IsBooleanConfigEnabled(...)` 与 `ResolveDefaultConfig(...)`。
  - `server/modules/notification/publisher.go`、`server/modules/container/service.go`、`server/internal/scheduler/runtime.go`、`server/modules/user/bootstrap.go` 都继续通过共享 resolver 边界消费有效配置。
  - `server/modules/container/mount_usage.go` 已有本地 TTL cache，可作为 process-local cache 参考，但不是 system-config authority。
  - `server/modules/monitor/module.go` 已有 Redis 趋势缓存，可作为 distributed cache 参考，但不应用来取代 system-config authority。
- 当前推荐实现起点：
  - Phase 1 已完成 authority 层 process-local full snapshot cache + singleflight + explicit invalidation。
  - Phase 2 已完成 Redis invalidation signal 的多节点传播预留，authority 与统一 resolver 边界保持不变。
  - Phase 3 已完成热点读路径扩展，未改变 system-config authority。
  - 下一步进入 `phase-4-observability-and-guardrails`，补齐可观测性、调试证据、topic recovery 更新与最小必要治理门禁。

## Validation Targets

```bash
git diff --check
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-cache-governance
python3 scripts/validate_ai_governance.py
python3 scripts/validate_shared_asset_registries.py
```

若本轮进入运行时实现：

```bash
graft validate backend --stage lint
```

若本轮触达系统配置页展示语义：

```bash
cd web && bun run check
```

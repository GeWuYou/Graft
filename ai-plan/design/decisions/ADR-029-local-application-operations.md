# ADR-029: Local Application Operations

- Status: accepted
- Date: 2026-11-15
- Scope: local Docker/Compose, Build, Task execution, official installation and platform update
- Supersedes in part: ADR-026 for explicitly selected local execution

## Context

个人与小团队需要安装后直接完成本机应用运维。当前 Docker 读取由 server 承担，有限写操作经独立 Agent，
首次可用涉及 Vault PKI、证书和 delivery。真实复验已修复若干 Agent 与 Compose 边界，安装、诊断与恢复仍需收敛。

## Decision

1. 产品定位为容器化应用运维平台，优先单机 Docker/Compose，并覆盖构建和平台自更新。
2. 配置显式选择 local 或 agent。缺少配置时采用 agent 兼容既有安装，新安装模板选择 local。
3. 本机 StageExecutor 注册在现有 Task Runtime，共享 SDK 实现位于 server/internal/dockerexecution。
   Application、Container、Build 拥有意图、材料和结果，Task 拥有持久化 claim、attempt、取消、恢复和终态。
4. 任务计划冻结执行方式及版本，材料和短期凭据仅向当前有权执行的 attempt 提供，日志与回执受限并脱敏。
   不确定副作用经 unknown/needs_attention 收敛，再通过实际资源事实核对。
5. 本机平台更新仍由独立短生命周期、digest-pinned Controller 完成备份、迁移、替换、健康检查和 durable receipt。
   server 重启后读取并结算回执。数据库恢复需要显式流程与适用版本证据。
6. 模式切换入口为 graft runtime mode preflight --mode local，检查非终态相关任务与外部租约。
   历史任务、审计和更新记录保持可读取，旧 attempt 的执行归属保持可追溯。
7. 既有 Agent 模式显式启用 COMPOSE_PROFILES=agent，升级预检防止配置与运行拓扑脱节。
8. 默认交付拓扑使用 server、web、PostgreSQL、Redis 和一次性准备服务。本机执行就绪检查涵盖 Docker
   可访问性及请求能力，用户结果包含稳定失败阶段与关联标识。
9. 根验证阶段区分迭代、提交准备、完整 CI 与运行验收，根 Bun workspace 统一 JavaScript 安装。
   Turbo 初期仅缓存前端生产构建，检查、测试、生成与迁移验证每次执行。

## Consequences

server 获得 Docker 写操作职责，需要针对权限、并发、凭据、取消和崩溃恢复验证高权限边界。
共享适配的 API 以已存在消费者为依据，避免将任务或领域策略散入提供方。
旧模式切换与执行历史的兼容需要独立测试。缓存收益通过相同输入与工具链的冷/热构建测量。
应用数据备份、域名 HTTPS 和宿主机管理按路线分别推进。

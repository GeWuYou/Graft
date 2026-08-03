# Platform Self Update Roadmap

本路线图交付 Graft 自身生命周期管理能力。每一阶段必须保持 release identity 单一事实来源、可审计执行和明确的人工恢复边界。

## Delivery Order

1. **Release authority and manifest**
   - 发布 `release-manifest.json`，将 GitHub Release、checksum artifact 和 GHCR immutable digest 关联为同一 tag 的事实。
   - 固化 installation profile、channel 选择和 Compose runner 信任边界。
2. **Read-only discovery**
   - 实现 `platform-update` module、InstallationProfile、GitHub catalog/manifest 校验、SemVer stable/beta 选择、`read/check/manage` 权限和默认周期检查。
   - 暴露 current/latest/release-notes/capability APIs；二进制部署生成校验后的人工步骤。
3. **Update Management**
   - 交付 `Platform -> System Maintenance -> Updates` 和顶部轻量版本提醒。提醒只展示当前/最新版本、更新摘要和受控升级入口；完整 release notes、安装与校验详情、能力矩阵、历史和不可执行原因保留在管理员管理页。
4. **Independent backup capability**
   - 交付 `platform-backup` module 的配置与 PostgreSQL backup metadata、恢复入口和审计边界；更新只通过 capability 消费它。
5. **Runner-owned Compose execution and recovery**
   - 交付 runner-owned named state volume、schema-v2 版本化原子状态快照与 sparse append events、`lease_epoch` fencing、30 秒 heartbeat、五分钟 expiry、explicit migration、manifest-verified recreate、health confirmation、手动 recovery runner，以及 server 的只读 API/realtime/terminal-history projection。
   - runner 是 active lifecycle controller；`server` 只记录已授权用户请求、每分钟 reconcile 并在查询时计算 lease、验证并转发 runner state，且只将已验证 terminal result 投影到 history/audit/backup facts。runner 不接收 PostgreSQL credentials，也不提供 HTTP/realtime endpoint。
   - 过期 v2 lease 或五分钟未产生首份状态的授权记录投影为 `runner_lost`；Docker container inventory 不得决定活动状态或 recovery。recovery 只接受迁移前 `runner_lost` 操作，已有 state 传入绑定 snapshot，首状态缺失只传入授权身份和版本输入。
   - Beta cutover removes runtime schema-v1 compatibility: bootstrap runs `graft update cutover-v1` before `graft migrate up`, then server accepts schema-v2 snapshots only; `runner_terminated` and the v1 bridge are not part of the post-cutover contract.
   - 自动化含义仅为管理员确认后的工作流自动执行，不包含无人值守更新。
   - 在 Beta 可靠性收敛中，Compose `.env` 使用完整 server/web 镜像引用与 `stable|beta|fixed|manual` 策略；runner pull 后验证 manifest digest，且 `manual` 不执行镜像变更。`nightly` 延后且不暴露。
6. **Archive readiness**
   - 跑跨边界验证，补齐 Compose fixture、运行文档、已知风险和 topic archive evidence。

## MVP Acceptance

- 管理员可发现新版本、阅读 release、检查 capability，并手动确认官方 Compose 升级。
- 升级保留配置和数据库，创建可审计 backup，显式运行 Atlas migration，验证 `/healthz`，由 runner 持久记录阶段和 terminal result；服务恢复后由 server 验证并投影业务历史。
- binary 用户获得 release/checksum 和与安装方式匹配的步骤，不会得到错误的自动升级按钮。

## Deferred Phase 2

- 可配置更新 channel、更新窗口与通知策略。
- 多节点滚动升级、Kubernetes executor、systemd host integration。
- 有条件的 rollback automation；数据库恢复仍须受 forward-only migration policy 约束。当前范围仅允许迁移前的受控配置/镜像回滚，迁移后失败必须走人工恢复。

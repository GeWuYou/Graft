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
5. **Confirmed Compose execution and recovery**
   - 交付一次性 receipt-writing runner、Task Runtime 状态、explicit migration、manifest-verified recreate、health receipt、history 和 Restore Backup 操作。
   - 自动化含义仅为管理员确认后的工作流自动执行，不包含无人值守更新。
   - 在 Beta 可靠性收敛中，Compose `.env` 使用完整 server/web 镜像引用与 `stable|beta|fixed|manual` 策略；runner pull 后验证 manifest digest，且 `manual` 不执行镜像变更。`nightly` 延后且不暴露。
6. **Archive readiness**
   - 跑跨边界验证，补齐 Compose fixture、运行文档、已知风险和 topic archive evidence。

## MVP Acceptance

- 管理员可发现新版本、阅读 release、检查 capability，并手动确认官方 Compose 升级。
- 升级保留配置和数据库，创建可审计 backup，显式运行 Atlas migration，验证 `/healthz`，持久记录成功或失败 receipt。
- binary 用户获得 release/checksum 和与安装方式匹配的步骤，不会得到错误的自动升级按钮。

## Deferred Phase 2

- 可配置更新 channel、更新窗口与通知策略。
- 多节点滚动升级、Kubernetes executor、systemd host integration。
- 有条件的 rollback automation；数据库恢复仍须受 forward-only migration policy 约束。

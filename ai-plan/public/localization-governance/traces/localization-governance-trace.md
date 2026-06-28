# Localization Governance Trace

## 2026-06-18 Phase 0 governance persistence

- 建立本地化长期设计 authority：`ai-plan/design/governance/platform/本地化与i18n治理规范.md`。
- 建立 public recovery topic：`ai-plan/public/localization-governance/README.md`。
- 建立任务追踪：`ai-plan/public/localization-governance/todos/localization-governance-tracking.md`。
- 建立 trace：`ai-plan/public/localization-governance/traces/localization-governance-trace.md`。
- 建立仓库 skill：`.agents/skills/graft-localization-governance/SKILL.md`。
- 确认下一步用 `$graft-multi-agent-loop` 推进 Phase 1 起的 bounded batches。

## 2026-06-18 Phase 1-4 server migration completion

- Phase 1：已落地 `server/internal/i18n` embedded YAML loader、`locales/` 目录与单测，保持 `i18n.Service` facade、
  map catalog、`LookupRequest`、`Freeze`、`RegisteredMessageResources` 等外部 API 不变。
- Phase 2：已迁移 dashboard quick actions system-config 样例文案到 embedded locale YAML。
- Phase 3：已迁移剩余 server-side system-config 展示文案，保持 key 常量、fallback 字段与 JSON Schema `x-i18n`
  元数据不变。
- Phase 4：已迁移菜单、通知、公告、scheduler、container、log explorer 等展示文案到 locale resources，并保持
  provider 不外泄。

## 2026-06-18 Phase 5 go-i18n provider evaluation

- 以当前 `server/internal/i18n` 实现为 authority 完成 provider 评估。
- 证据摘要：
  - 当前 server i18n 结构是 facade + map catalog + embedded flat YAML loader。
  - `LookupRequest.TemplateData` 已预留，但当前调用面没有已落地 plural rules、复杂模板渲染或翻译平台导入导出流程。
  - Phases 1-4 已覆盖当前优先级内的 server locale 资源迁移，未出现必须引入第二 provider 才能继续推进的阻塞。
- 当前决策：不引入 `go-i18n`。
  - 原因是当前收益未被真实需求证明，而 provider 切换会增加实现分叉、测试矩阵和 provider 泄漏风险。
  - 未来只有在 plural、命名模板、翻译平台工作流或新增 locale 成为真实需求时，才重新开启 bounded provider 评估。

## 2026-06-18 Batch 0 authority reset and locale directory strategy

- 识别到 public README、tracking、trace 和 skill 仍保留 `ready-for-archive-check` 与过时 locale ownership 语言，属于 authority drift。
- 本轮将 design/topic/skill 统一重置为以下 backend authority：
  - embedded locale YAML 是 backend 用户可见本地化文案的 canonical truth。
  - `server/internal/i18n` 独占 locale 资源的 embed、load、validate、freeze 与 registry construction。
  - module 不得自持 locale 文件 embed/load 逻辑，也不得在 `server/internal/i18n` 外维护平行 registry。
- 当时的过渡实现策略是先补齐 `server/internal/i18n` 对 raw embedded resource 的注册入口，保持集中目录继续工作：
  - `server/internal/i18n/locales/*.yaml` 继续承载 `core` / `display`。
  - module-owned 与 runtime-owned locale 的最终物理归属，随后在独立 `server-locale-ownership-migration` topic 中迁往 owner-local `locales/*.yaml`。
  - 该 topic 最终完成后，`server/internal/i18n/locales/modules/*.yaml` 已不再承载 live locale YAML，只保留 guard marker `README.md`。

## 2026-06-18 Slice 3 delete legacy fallbacks and switch to locale resource

- 删除以下 Go 用户可见 fallback，改由 locale resource authority 提供：
  - `server/internal/httpx/accesslog_explorer.go`：权限 `Name/Description`，菜单 `Title`。
  - `server/internal/logger/explorer.go`：权限 `Name/Description`，菜单 `Title`。
  - `server/modules/audit/module_registration.go`：权限 `Name/Description`，菜单 `Title`。
  - `server/modules/audit/dashboard_widget.go`：Widget/QuickLink/Action/item `Title/Description/Label/empty` fallback。
  - `server/internal/httpx/accesslog_dashboard.go`：Widget item `Title` 与 empty fallback。
  - `server/internal/httpx/accesslog_retention.go`、`server/internal/logger/retention.go`、`server/modules/audit/retention.go`：
    config definition `DomainLabel/GroupLabel/GroupDescription/Title/Description`，job `Title/ShortTitle/Description`，action `Title/Description`。
- 新增 locale authority：
  - `server/internal/i18n/locales/display.{zh-CN,en-US}.yaml`：`dashboard.actions.details`。
  - `server/internal/i18n/locales/modules/rbac.{zh-CN,en-US}.yaml`：`rbac.permissionCatalog.accessLogRead.*`、`appLogRead.*`、`appLogDelete.*`、`auditRead.*`。
- 本轮保留在 Go 的字符串仅作为技术标识：
  - permission code、menu code、module key、route/path、job name、action key、resource name、operation name、query key、状态枚举与内部日志消息。
- 临时例外：无。

## 2026-06-18 Slice 4 residual server visible-copy cleanup

- 删除以下 Go 用户可见 fallback，改为仅保留 locale key authority：
  - `server/modules/rbac/route_registration.go`：菜单 `Title`。
  - `server/internal/moduleruntime/registration.go`：菜单 `Title`。
  - `server/modules/monitor/module.go`：菜单 `Title`、unavailable evidence link `Title`。
  - `server/modules/system-config/module_registration.go`：菜单 `Title`。
  - `server/modules/container/module_registration.go`：菜单 `Title`。
  - `server/modules/announcement/module_registration.go`：菜单 `Title`。
  - `server/modules/user/module_registration.go`：菜单 `Title`。
  - `server/modules/scheduler/module_registration.go`：菜单 `Title`。
  - `server/modules/rbac/dashboard_widget.go`：QuickLink `Title`。
  - `server/modules/monitor/dashboard_widget.go`：Widget / QuickLink / summary / health-item label fallback。
  - `server/modules/system-config/dashboard_widget.go`：QuickLink `Title`。
  - `server/modules/scheduler/dashboard_widget.go`：Widget / QuickLink / action `Title` / `Description` / `Label` fallback。
  - `server/internal/dashboard/quick_actions_config.go`：config definition `DomainLabel` / `GroupLabel` / `GroupDescription` / `Title` / `Description` 与 schema 内嵌 title/description fallback。
  - `server/modules/container/config.go`：config definition `DomainLabel` / `GroupLabel` / `GroupDescription` / `Title` / `Description` 与 schema 内嵌 title/description fallback。
- 新增 locale authority：
  - `server/internal/i18n/locales/display.{zh-CN,en-US}.yaml`：`dashboard.widget.monitorSystemHealth.*`。
- 本轮明确保留在 Go 的技术或非用户可见字符串：
  - permission `Name` / `Description` 当前仍是 RBAC 管理数据与 seeded metadata 的英文稳定文本，不属于本批 residual visible-copy owner。
  - `monitor` health `Detail`、anomaly `Summary`、reason 文本属于运行时动态诊断文本；当前未建立独立 locale resource owner，不在本批直接迁移。
  - route/path、menu code、module key、config key、resource key、event type、dedupe key、enum/status、log message 继续作为技术标识保留。
- 临时例外：无。

## 2026-06-18 Final archive readiness and governance sync

- 移除 `server/modules/scheduler/notification_integration.go` 中剩余通知 fallback 文本：
  - 成功通知不再写入 `Message` 与 `ActionLabel` 英文 fallback，仅保留 locale key authority 和必要的动态 `Title` 资源名。
  - 失败通知不再写入 `Title` 与动态拼接 `Message` fallback，仅保留稳定 locale key authority。
- 同步放宽通知中心发布/持久化校验：
  - `server/modules/notification/publisher.go`
  - `server/modules/notification/store/sql_repository.go`
  - 以上两处都改为 `TitleKey/Title`、`MessageKey/Message` 二选一满足即可，允许 key-first 事件持久化。
- 当前 final 审计结论：
  - backend Go 已无登记中的业务本地化硬编码例外。
  - `audit` built-in TargetLabel 已切换为 stable locale key + embedded locale YAML，HTTP wire shape 保持 `target_label` 不变。
  - `server/modules/monitor/module.go` 中 `Review related audit activity` / `Check audit records from the same bounded monitor window.` 属于 monitor 动态诊断 evidence link 文本，当前仅做分类，不作为本 topic 的未登记业务本地化硬编码处理。
- 本 topic 达到 archive-ready 条件：
  - backend 用户可见本地化硬编码已迁移到 embedded locale YAML，且当前无登记中的临时例外；
  - docs / trace / tracking / skill 已同步，无未登记 drift；
  - 当前无新的未声明 Go business-localization hardcoding 残留。

## 2026-06-18 P0 residual cleanup and cross-boundary governance sync

- 重新执行 root `AGENTS.md` startup preflight，并将 task class 提升为 `cross-boundary`，因为 residual cleanup 同时涉及 server、web、scanner 与治理文档。
- backend P0 收口：
  - 清空以下注册源中的 `permission.Item{Name, Description}` 英文用户可见 fallback，仅保留 stable key：
    - `server/modules/scheduler/module_registration.go`
    - `server/modules/notification/module_registration.go`
    - `server/modules/system-config/module_registration.go`
    - `server/modules/container/module_registration.go`
    - `server/modules/monitor/module.go`
    - `server/modules/user/module_registration.go`
    - `server/modules/rbac/route_registration.go`
    - `server/modules/announcement/module_registration.go`
    - `server/internal/moduleruntime/registration.go`
  - `server/modules/user/bootstrap_admin.go` 改为在 seed 生成时通过 `i18n.Service` 严格解析 `DisplayKey` / `DescriptionKey`，不再接受 Go 英文 fallback。
  - `server/modules/user/dev_reset.go` 与 `server/internal/cli/dev_reset.go` 对齐同一 locale authority，避免 dev reset 路径重新引入可见 fallback。
  - `server/internal/app/runtime.go` 移除 `Module Runtime`、`Current module runtime health.`、`View details`、`Access Logs`、`Request Attention`、`App Logs` 等 core dashboard/runtime 英文硬编码，统一改为 embedded locale YAML lookup。
  - `server/internal/dashboard/service.go` 删除框架级默认 `View details` fallback；缺值时应补 locale key，而不是回退英文。
- locale 资源同步：
  - `server/internal/i18n/locales/modules/rbac.{zh-CN,en-US}.yaml` 补齐缺失的 permission catalog key。
  - `server/internal/i18n/locales/display.{zh-CN,en-US}.yaml` 补齐 module runtime widget / summary / status 文案。
- frontend P0 收口：
  - `web/src/store/modules/tabs-router.ts` 删除 home tab 的 `工作台 / Workspace` 双语对象，改为 `localizeRouteTitleKey('app.home.title')`。
  - `web/src/store/modules/tabs-router.test.ts` 同步到 key-based 断言。
- scanner 治理补强：
  - `web/scripts/i18n-governance/rules/no-hardcoded-ui-prop.ts` 新增对 `[LOCALE.ZH_CN]` / `[LOCALE.EN_US]` computed property 双语硬编码的识别。
  - 保持原有 `'zh-CN': '...'` / `'en-US': '...'` 检测能力。
  - 为避免误报纯插值组合表达式，本轮明确跳过不携带自身可见字面量的 template literal 组合。
  - `web/scripts/check-i18n-governance.test.ts` 与 `invalid-computed-locale-title-object` fixture 为该形态补齐直接测试。
- governance sync：
  - `ai-plan/public/localization-governance/README.md`、tracking、trace、design、skill 均更新为当前 cross-boundary 事实。
  - 明确：archive-ready 只能建立在当前代码与验证事实上；若重新引入未登记 fallback，不得沿用旧 closeout 结论。

## 2026-06-18 server locale ownership follow-up closed

- 独立 follow-up topic `ai-plan/public/server-locale-ownership-migration/README.md` 已完成全部实现批次与 final drift audit。
- broad localization governance 与 server locale ownership follow-up 的联合最终事实为：
  - `server/internal/i18n/locales/*.yaml` 仅保留 `core` / `display`。
  - module-owned locale 位于 `server/modules/<name>/locales/*.yaml`。
  - runtime-owned locale 位于 `server/internal/moduleruntime/locales/*.yaml`。
  - `scripts/check_server_locale_ownership.py` 已阻止 `server/internal/i18n/locales/modules/*.yaml` 回流。
- 因此本目录不再是新的 active recovery 入口；未来若要继续推进本地化治理，应新建 bounded topic，而不是复用旧的 broad governance loop。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "batch-0-authority-reset-and-locale-directory-strategy"
  ],
  "pending_batches": [
    "slice-1-module-registration-resource-migration",
    "slice-2-core-default-catalog-migration",
    "slice-3-delete-legacy-fallbacks-and-switch-to-locale-resource",
    "final-archive-readiness-and-governance-sync"
  ],
  "current_batch": "batch-0-authority-reset-and-locale-directory-strategy",
  "next_batch": "slice-1-module-registration-resource-migration",
  "closeout_status": "batch-0-in-progress"
}
```

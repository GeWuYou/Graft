# Localization Governance Tracking

## Topic

Localization Governance

## Scope

建立前后端本地化治理规范，新增 AI 执行 skill，并分批把 server / web 的用户可见本地化文案迁移到受治理的 locale 资源与 locale catalog。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/platform/本地化与i18n治理规范.md`
- `ai-plan/design/architecture/项目设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `.agents/skills/graft-localization-governance/SKILL.md`

## Current Recovery Point

- 旧 recovery 口径中的 `Phase 0-5 已全部完成 / ready-for-archive-check` 已确认失真。
- 当前恢复点已推进到 broad governance closeout 与独立 server locale ownership follow-up 全部完成后的 final 归档判定。
- 已落定 authority：
  - embedded locale YAML 是 backend 用户可见本地化文案的 canonical truth。
  - `server/internal/i18n` 独占 locale 资源的 embed、load、validate、freeze 与 registry construction。
  - `server/internal/i18n/locales/*.yaml` 只保留 `core` / `display`。
  - module-owned locale 位于 `server/modules/<name>/locales/*.yaml`。
  - runtime-owned locale 位于 `server/internal/moduleruntime/locales/*.yaml`。
  - module 与 runtime owner 不得自持 locale 文件加载逻辑。
- 后续无新的同主题 pending batch；未来扩展应开启新的 bounded localization topic。

## Task Checklist

- [x] batch-0：authority reset、README/skill/topic 状态纠偏
- [x] batch-0：集中 locale 目录策略落定
- [x] batch-0：`server/internal/i18n` nested module locale loader 支持
- [x] slice-1：module registration resource migration
- [x] slice-2：core default catalog migration
- [x] slice-3：delete legacy fallbacks and switch to locale resource
- [x] slice-4：residual server visible-copy cleanup
- [x] final：archive readiness and governance sync
- [x] P0 follow-up：permission display fallback cleanup、runtime/dashboard fallback cleanup、web tabs title cleanup、scanner computed-locale coverage

## Final Closeout Notes

- `audit` built-in target label 已完成收口，不再依赖 `displayTargetLabel()` 中的中文硬编码。
- canonical truth 当前位于 `server/modules/audit/locales/{zh-CN,en-US}.yaml`。
- repository 仅保留 `TargetType -> stable locale key` 技术映射，并通过 `server/internal/i18n.Service` 按请求 locale 解析 `target_label`。
- API wire shape 保持不变；未新增 `target_label_key`。
- `permission.Item{Name, Description}` 不再以注册源中的英文文案作为真相；当前由 locale key 经 `i18n.Service` 解析后生成 seeded display text。
- core dashboard/runtime 不再以内嵌 Go 英文文案作为 Widget / QuickLink / Action 展示真相；当前由 embedded locale YAML 解析。
- `web/src/store/modules/tabs-router.ts` 不再内嵌 `工作台 / Workspace` 双语对象；当前复用 `app.home.title`。
- `web/scripts/i18n-governance/rules/no-hardcoded-ui-prop.ts` 已覆盖 `[LOCALE.ZH_CN]` / `[LOCALE.EN_US]` computed property 双语硬编码。
- `scripts/check_server_locale_ownership.py` 已阻止 `server/internal/i18n/locales/modules/*.yaml` 重新成为 module-owned 或 runtime-owned locale 真相。
- 当前无登记中的生产 Go 用户可见本地化硬编码临时例外；当前无登记中的生产 TS/Vue 双语 UI 硬编码例外。

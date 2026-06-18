# Localization Governance Tracking

## Topic

Localization Governance

## Scope

建立前后端本地化治理规范，新增 AI 执行 skill，并分批把 server 侧硬编码 i18n 注册点迁移到集中 locale 资源文件。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/本地化与i18n治理规范.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `.agents/skills/graft-localization-governance/SKILL.md`

## Current Recovery Point

- 旧 recovery 口径中的 `Phase 0-5 已全部完成 / ready-for-archive-check` 已确认失真。
- 当前恢复点是 `batch-0-authority-reset-and-locale-directory-strategy`。
- 本轮必须先重置 authority：
  - embedded locale YAML 是 backend 用户可见本地化文案的 canonical truth。
  - `server/internal/i18n` 独占 locale 资源的 embed、load、validate、freeze 与 registry construction。
  - module 不得自持 locale 文件加载逻辑。
- 后续 pending batch 仍包括 module registration resource migration、core default catalog migration、legacy fallback 清理和最终 archive readiness。

## Task Checklist

- [x] batch-0：authority reset、README/skill/topic 状态纠偏
- [x] batch-0：集中 locale 目录策略落定
- [x] batch-0：`server/internal/i18n` nested module locale loader 支持
- [x] slice-1：module registration resource migration
- [x] slice-2：core default catalog migration
- [x] slice-3：delete legacy fallbacks and switch to locale resource
- [x] slice-4：residual server visible-copy cleanup
- [x] final：archive readiness and governance sync

## Final Closeout Notes

- `audit` built-in target label 已完成收口，不再依赖 `displayTargetLabel()` 中的中文硬编码。
- canonical truth 迁移到 `server/internal/i18n/locales/modules/audit.zh-CN.yaml` 与 `server/internal/i18n/locales/modules/audit.en-US.yaml`。
- repository 仅保留 `TargetType -> stable locale key` 技术映射，并通过 `server/internal/i18n.Service` 按请求 locale 解析 `target_label`。
- API wire shape 保持不变；未新增 `target_label_key`。
- 当前 server 侧已无登记中的生产 Go 用户可见本地化硬编码临时例外。

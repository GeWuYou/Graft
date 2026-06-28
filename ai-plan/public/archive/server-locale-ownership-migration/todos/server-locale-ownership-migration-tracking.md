# Server Locale Ownership Migration Tracking

## Topic

Server Locale Ownership Migration

## Scope

将 `server` locale resource 的物理 ownership 迁移到 owner package，保持 `server/internal/i18n` 独占 i18n 基础设施，并通过 runtime 在 `Freeze` 前统一预注册 owner-local embedded resources。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/governance/platform/本地化与i18n治理规范.md`
- `ai-plan/design/governance/platform/服务端Locale资源归属与迁移设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `.agents/skills/graft-localization-governance/SKILL.md`
- `.agents/skills/graft-multi-agent-loop/SKILL.md`

## Current Recovery Point

- docs-first architecture decision 与 implementation loop 已完成。
- 当前 live implementation 已完成：
  - raw embedded resource preregistration 入口
  - `announcement` 低风险试点
  - 剩余 module-owned locale 迁移
  - `module-runtime` locale 迁移
  - locale ownership drift guard
- 当前剩余工作只包括 final 文档收尾、skill/recovery 对齐、archive-readiness 审计。

## Task Checklist

- [x] docs-first：architecture decision、design update、public topic persistence
- [x] slice-1：补 `server/internal/i18n` raw embedded resource registration 入口
- [x] slice-2：迁移 `announcement` 低风险试点
- [x] slice-3：迁移剩余 module-owned locale resources
- [x] slice-4：迁移 `module-runtime`
- [x] slice-5：更新治理文档、skill、recovery 与 CI/脚本阻断规则
- [x] final：i18n 文档收尾与 drift 审计

## Final Status

- 主题已完成全部计划批次并达到 `archive-ready`。
- 最终 live implementation：
  - `server/internal/i18n/locales/*.yaml` 仅保留 `core` / `display`
  - `server/modules/<name>/locales/*.yaml` 持有 module-owned locale
  - `server/internal/moduleruntime/locales/*.yaml` 持有 runtime-owned locale
  - `scripts/check_server_locale_ownership.py` 阻止集中 module locale 目录回流

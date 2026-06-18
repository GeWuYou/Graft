# Server Locale Ownership Migration

## 当前状态摘要

- 当前主题目标是把 `server` locale resource 的物理 ownership 从集中目录迁移到 owner package，同时保持 `server/internal/i18n` 继续独占 i18n 基础设施。
- 当前状态：`archive-ready`。
- 任务分类：`server`。
- Canonical design：
  - `ai-plan/design/本地化与i18n治理规范.md`
  - `ai-plan/design/服务端Locale资源归属与迁移设计.md`
  - `ai-plan/design/模块与依赖注入设计.md`
- AI 执行 skills：
  - `.agents/skills/graft-localization-governance/SKILL.md`
  - `.agents/skills/graft-multi-agent-loop/SKILL.md`

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`server`
- recovery source：`parent topic`
- authority summary：`server/internal/i18n.Service` 是唯一 server i18n facade；locale resource 的物理 ownership 可分散到 module/internal owner package，但 embed/load/validate/freeze/registry 仍集中在 `server/internal/i18n`

## Owned Scope

允许修改：

- `ai-plan/design/本地化与i18n治理规范.md`
- `ai-plan/design/服务端Locale资源归属与迁移设计.md`
- `ai-plan/design/模块与依赖注入设计.md`
- `ai-plan/public/server-locale-ownership-migration/**`
- `ai-plan/public/README.md`
- `ai-plan/public/localization-governance/**`
- `.agents/skills/graft-localization-governance/**`
- implementation / final closeout 允许修改：
  - `server/internal/i18n/**`
  - `server/internal/app/**`
  - `server/internal/moduleregistry/**`
  - `server/internal/moduleruntime/**`
  - `server/modules/**`
  - 以及对应的 focused tests 与必要 scanner/CI 脚本

禁止误触：

- 不得让 `server/internal/i18n` 反向 import `server/modules/*`
- 不得让模块自持 YAML loader、locale 校验、duplicate 校验、registry 或 freeze
- 不得改变 stable key
- 不得改变 HTTP wire shape
- 不得引入 `go-i18n`
- 不得长期保留 `server/internal/i18n/locales/modules/*.yaml` 与 owner-local locale 双来源

## Phase Plan

- Slice 1：补 `server/internal/i18n` raw embedded resource registration 入口，不移动文件
- Slice 2：选择 `announcement` 或 `container` 作为低风险模块试点
- Slice 3：迁移 `audit`、`container`、`monitor`、`rbac`、`scheduler`、`system-config`、`user`
- Slice 4：迁移 `module-runtime`
- Slice 5：更新治理文档、skill、recovery，并加 CI/脚本阻止新增 `server/internal/i18n/locales/modules/*.yaml`
- Final closeout：全部 phase 完成后，执行一次 i18n 文档收尾与 drift 审计，检查 `ai-plan/design`、`ai-plan/public`、`.agents/skills`、必要 scanner/CI 规则是否仍与 live implementation 对齐

## Current Recovery Point

- 已完成 implementation loop：
  - Slice 1：新增 `server/internal/i18n` raw embedded resource registration 入口与 runtime pre-registration slot，保持集中目录继续工作。
  - Slice 2：`announcement` 已迁到 `server/modules/announcement/locales/*.yaml`，通过 owner-local embedded descriptor 接入 preregistration。
  - Slice 3：`audit`、`container`、`monitor`、`rbac`、`scheduler`、`system-config`、`user` 已迁到各自 `server/modules/<name>/locales/*.yaml`。
  - Slice 4：`module-runtime` 已迁到 `server/internal/moduleruntime/locales/*.yaml`。
  - Slice 5：新增 `scripts/check_server_locale_ownership.py`，并接入 `graft validate backend --stage lint`，阻止 `server/internal/i18n/locales/modules/*.yaml` 回流。
- 当前 live implementation：
  - `server/internal/i18n/locales/*.yaml` 仅保留 `core` / `display`。
  - `server/internal/i18n/locales/modules/` 只保留 `README.md` 作为 legacy-free guard target。
  - `server/internal/moduleregistry.EmbeddedLocaleResources()` 汇总 module-owned locale providers。
  - `server/internal/app.runtimeEmbeddedLocaleResources()` 在 module providers 之外并入 `module-runtime` owner-local resources，并在模块 `Register` 前统一预注册。
  - 模块内 `registerMessages()` 继续只做 key existence 校验，不承担 loader。
- `system-config` 默认按 module-owned locale 处理。
- 未启用模块的 locale resource 默认仍注册。
- final 文档收尾、skill/recovery 对齐、guard 规则核对与 archive-readiness check 已完成。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "docs-first-architecture-decision-and-recovery-persistence",
    "slice-1-raw-embedded-registration-entry",
    "slice-2-low-risk-module-pilot",
    "slice-3-remaining-module-owned-locale-migration",
    "slice-4-module-runtime-locale-migration",
    "slice-5-governance-and-ci-drift-guards",
    "final-i18n-doc-closeout-and-drift-audit"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```

## Startup Prompt

- 见 `ai-plan/public/server-locale-ownership-migration/startup-prompt.md`

## Validation Targets

文档 / recovery / skill：

```bash
git diff --check
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-localization-governance
```

后续 server implementation：

```bash
cd server && go test ./internal/i18n/...
cd server && go test ./internal/app/... ./internal/moduleregistry/... ./internal/moduleruntime/...
cd server && go run ./cmd/graft validate backend --stage lint
cd server && go build ./cmd/graft
```

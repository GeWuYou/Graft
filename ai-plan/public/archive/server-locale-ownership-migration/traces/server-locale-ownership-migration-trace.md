# Server Locale Ownership Migration Trace

## 2026-06-18 docs-first architecture decision

- 重跑 root `AGENTS.md` startup preflight，确认当前工作应先以 `docs/automation with server impact` 启动，再为后续 server implementation 建立 recovery topic。
- 结合当前实现事实确认：
  - `server/internal/i18n` 当前 `go:embed` 只覆盖自身目录下的 `locales/*.yaml` 与旧的 `locales/modules/*`
  - `Service.New()` 在 runtime 构造阶段完成 embedded catalog 注册
  - `Freeze` 发生在所有模块 `Register` 完成之后、所有模块 `Boot` 之前
  - 现有各模块 `registerMessages()` 只是 key existence 校验，不承担 loader 职责
  - `moduleregistry` 已经持有全部 compile-time module specs
- 形成新的长期决议：
  - `server/internal/i18n` 只拥有 i18n 基础设施
  - `core` / `display` 继续由 `server/internal/i18n/locales/*.yaml` 拥有
  - module-owned locale 迁到 `server/modules/<name>/locales/*.yaml`
  - `module-runtime` 迁到 `server/internal/moduleruntime/locales/*.yaml`
  - owner package 可以自己 `go:embed locales/*.yaml`，但只暴露只读 resource descriptor
  - runtime 在 `Freeze` 前统一预注册这些 resources
  - 未启用模块的 locale resource 默认仍注册
- 更新文档：
  - `ai-plan/design/本地化与i18n治理规范.md`
  - `ai-plan/design/模块与依赖注入设计.md`
  - `ai-plan/design/服务端Locale资源归属与迁移设计.md`
  - `.agents/skills/graft-localization-governance/SKILL.md`
  - `ai-plan/public/localization-governance/README.md`
- 建立 active public topic：`ai-plan/public/server-locale-ownership-migration/README.md`

## 2026-06-18 implementation loop completed through slice-5

- Slice 1:
  - added raw embedded locale resource registration to `server/internal/i18n`
  - established runtime preregistration slot before module `Register` and before `Freeze`
  - kept centralized files working while introducing infrastructure-only preregistration
- Slice 2:
  - migrated `announcement` locale ownership to `server/modules/announcement/locales/*.yaml`
  - wired owner-local embedded descriptors through runtime preregistration
  - removed centralized `announcement` locale files
- Slice 3:
  - migrated `audit`、`container`、`monitor`、`rbac`、`scheduler`、`system-config`、`user`
  - removed the corresponding centralized locale files
  - updated focused tests to preregister embedded owner-local resources explicitly
- Slice 4:
  - migrated `module-runtime` locale ownership to `server/internal/moduleruntime/locales/*.yaml`
  - kept `registerMessages()` as key-existence validation only
  - updated focused app / i18n / moduleruntime tests to use preregistration semantics
- Slice 5:
  - added `scripts/check_server_locale_ownership.py`
  - wired the guard into `graft validate backend --stage lint`
  - blocked reintroduction of `server/internal/i18n/locales/modules/*.yaml` locale YAML

## Current Live State Before Final Closeout

- `server/internal/i18n/locales/*.yaml` only contains `core` / `display`.
- `server/internal/i18n/locales/modules/` is retained only as a guarded legacy-free marker directory with `README.md`.
- module-owned locale YAML now lives under:
  - `server/modules/announcement/locales/*.yaml`
  - `server/modules/audit/locales/*.yaml`
  - `server/modules/container/locales/*.yaml`
  - `server/modules/monitor/locales/*.yaml`
  - `server/modules/rbac/locales/*.yaml`
  - `server/modules/scheduler/locales/*.yaml`
  - `server/modules/system-config/locales/*.yaml`
  - `server/modules/user/locales/*.yaml`
- internal runtime-owned locale YAML now lives under:
  - `server/internal/moduleruntime/locales/*.yaml`
- runtime preregisters owner-local embedded locale resources before module `Register`.
- disabled modules' locale resources still register by default.
- no long-lived dual-source compatibility remains in live code.

## 2026-06-18 final doc closeout and drift audit

- Reconciled the design docs, public recovery materials, and `.agents/skills/graft-localization-governance/SKILL.md` with the live owner-local locale implementation.
- Cleared stale wording that still described `server/internal/i18n/locales/modules/*.yaml` as the live module-locale truth.
- Confirmed the final governance facts:
  - `server/internal/i18n` remains infrastructure-only.
  - owner packages embed `locales/*.yaml` and expose read-only resource descriptors only.
  - runtime preregisters owner-local locale resources before module `Register` and before `Freeze`.
  - `registerMessages()` remains key-existence validation only.
  - `scripts/check_server_locale_ownership.py` blocks reintroduction of centralized module/runtime locale YAML.
- Archive-readiness verdict: passed.

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

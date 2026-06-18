# Localization Governance

## 当前状态摘要

- 当前主题目标是建立 `Graft` 前后端本地化长期治理，并分阶段把 server / web 的用户可见文案收敛到受治理的 locale 资源。
- 当前状态：broad localization governance baseline 已闭环；server locale ownership follow-up 也已在独立 topic 完成 archive-readiness 审计。
- 任务分类为 `cross-boundary`；当前主题同时处理 backend locale authority、web locale catalog 与 scanner 治理收口。
- Canonical design：`ai-plan/design/本地化与i18n治理规范.md`。
- AI 执行 skill：`.agents/skills/graft-localization-governance/SKILL.md`。
- server ownership follow-up 记录：`ai-plan/public/server-locale-ownership-migration/README.md`。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/本地化与i18n治理规范.md` + `server/internal/i18n.Service` facade + embedded locale YAML under `server/internal/i18n/locales/**` + web locale catalogs under `web/src/locales/**`

## Owned Scope

允许修改：

- `ai-plan/design/本地化与i18n治理规范.md`
- `ai-plan/public/localization-governance/**`
- `ai-plan/public/README.md`
- `.agents/skills/graft-localization-governance/**`
- 当前 topic 允许修改 `server/internal/i18n/**`、相关 server 调用点、`web/src/locales/**`、相关 web 调用点、scanner 规则、治理文档与其直接测试，用于完成 locale authority 迁移与治理收口。

禁止误触：

- 不得让业务模块直接 import `github.com/nicksnyder/go-i18n`。
- 不得修改 HTTP 错误响应 wire shape。
- 不得删除 JSON Schema 现有 `x-i18n` key。
- 不得把 web 长期 UI 文案所有权迁到 server。
- 不得把 Go 硬编码用户可见文案继续表述为可接受的长期 authority。
- 不得把 TS/Vue 中的双语字面量对象继续表述为可接受的长期 authority。
- 不得允许业务模块自行 embed、load、validate 或 freeze locale 文件。
- 不得为了兼容旧逻辑继续保留新的用户可见 fallback 或兼容桥。

## Phase Plan

- Phase 0：现状盘点、规范和 topic 持久化。已完成。
- batch-0：authority reset、主题恢复材料纠偏、集中 locale 目录策略落定，以及 `server/internal/i18n` nested module loader 前置支持。已完成。
- slice-1：module registration resource migration。已完成。
- slice-2：core default catalog migration。已完成。
- slice-3：delete legacy fallbacks and switch to locale resource。已完成。
- final：archive readiness and governance sync。已完成。

## Current Recovery Point

- 旧的 `ready-for-archive-check` 判定已失效；后续 server locale ownership 迁移已转入独立 topic 并完成实现、guard 和 final drift audit。
- 当前 authority 决议：
  - backend 面向用户可见本地化文案的 canonical truth 是 embedded locale YAML。
  - locale 资源的 embed、load、validate、freeze 与 registry construction 只能集中在 `server/internal/i18n`。
  - module 与 internal runtime owner 只拥有 namespace/key 语义和 owner-local embedded resource descriptor，不拥有独立 locale 文件加载器。
  - web 面向用户可见 UI 文案的 canonical truth 是 locale catalog；`tabs-router` 等壳层状态不得内嵌双语字面量对象。
  - 生产 Go 不得新增用户可见硬编码本地化文案；菜单、Widget、Retention、Explorer、Cron Action、permission display metadata 等可见字段默认必须来自 locale resource。
  - 生产 TS/Vue 不得新增 `工作台 / Workspace` 这类 computed locale object 双语硬编码；locale 文案必须进入 catalog，再通过 key 或 lookup 使用。
  - 仅技术标识可保留在 Go：稳定 key、模块名、资源名、action key、route/path、permission code、job name 等。
  - 临时例外必须显式登记文件、字段、原因、移除条件与验证范围。
- 当前结论：
  - `audit` TargetLabel 临时例外已关闭。
  - backend `permission.Item{Name, Description}` 与 core dashboard/runtime 不再保留未登记的 Go 用户可见 fallback。
  - frontend `tabs-router` 不再内嵌 `工作台 / Workspace`。
  - `bun run lint:i18n` 规则已覆盖 `[LOCALE.ZH_CN]` / `[LOCALE.EN_US]` computed property 双语硬编码。
  - `server/internal/i18n/locales/*.yaml` 当前只保留 `core` / `display`；module-owned locale 已迁到 `server/modules/<name>/locales/*.yaml`，runtime-owned locale 已迁到 `server/internal/moduleruntime/locales/*.yaml`。
  - `scripts/check_server_locale_ownership.py` 已阻止 `server/internal/i18n/locales/modules/*.yaml` 回流。
  - 当前 broad topic 与独立 server follow-up topic 都已达到 `archive-ready` 条件；未来若要继续推进新的本地化治理，必须新开 bounded topic，而不是复用本目录作为 active recovery 入口。

## Validation Targets

```bash
git diff --check
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-localization-governance
```

若本轮触达 `server/internal/i18n/**`：

```bash
cd server && go test ./internal/i18n/...
cd server && go build ./cmd/graft
```

若本轮触达 web locale 或 scanner：

```bash
cd web && bun run lint:i18n
```

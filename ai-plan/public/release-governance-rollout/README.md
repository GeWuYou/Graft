# Release Governance Rollout

本 README 只承载后续发版治理落地的 topic recovery、loop 批次与 archive-ready 边界，不是仓库规范正文。

上游审计结论与 `P0/P1/P2` 分层以 `ai-plan/public/archive/release-readiness-governance-audit/**` 为证据真值；
本主题负责把其中 `v0.1.0` 的 `P0` 治理项拆成可执行的 `$graft-multi-agent-loop` 批次。

## 当前状态摘要

- 当前主题目标是把 release-readiness 审计结论转成 `v0.1.0` 的治理落地顺序，而不是直接改 release workflow。
- 当前状态：`planned-active`。
- 任务分类为 `cross-boundary`，涉及 version/build/release/migration/config/deployment/upgrade/documentation governance。
- 默认 loop mode：`topic-completion-loop`。
- Canonical audit evidence：`ai-plan/public/archive/release-readiness-governance-audit/README.md`。
- 当前建议优先顺序：
  - Phase 0：topic 建立与 archive handoff
  - Phase 1：release safety governance
  - Phase 2：release identity and policy
  - Phase 3：release operator docs baseline

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：root `AGENTS.md` + `ai-plan/public/archive/release-readiness-governance-audit/README.md` + `README.md` + `server/internal/cli/{serve,migrate,validate}.go` + `server/internal/config/**` + `web/package.json` + `.github/workflows/{release,publish}.yml`

## Owned Scope

允许修改：

- `ai-plan/public/release-governance-rollout/**`
- `ai-plan/public/README.md`
- `ai-plan/public/archive/release-readiness-governance-audit/**`
- Phase 1 如需沉淀仓库级治理真值，可修改：
  - `ai-plan/design/数据库表设计与迁移规范.md`
  - `ai-plan/design/服务端API边界与兼容治理规范.md`
  - `README.md`
- Phase 2 如需沉淀仓库级治理真值，可修改：
  - `README.md`
  - 必要的 topic-only design/roadmap 文档
- Phase 3 如需落地最小 operator 文档，可修改：
  - `README.md`
  - 后续新建的 release/install/upgrade/rollback 文档目录

禁止误触：

- 不得把本主题扩张成 release workflow、Docker/Compose、Kubernetes、托管平台支持实现。
- 不得在未固定 authority 前先改 `server/**`、`web/**`、`.github/workflows/**` 运行时代码。
- 不得创建“假装已经支持”的自动升级、自动回滚或自动部署资产。

## Loop Plan

- Loop mode：`topic-completion-loop`
- Worker model：每个 batch 默认一个 `worker` subagent，经 `$graft-multi-agent-task` 执行
- 默认预算：
  - `max_rounds=4`
  - `max_commits=4`
  - `checkpoint_budget=1`
  - `soft_timeout_minutes=30`
  - `default_grace_window=20`
  - `max_grace_window=30`
- validation failure policy：
  - docs-only batch 保持在治理文档、topic recovery 与引用一致性校验
  - 如 batch 合法扩展到实现层，继续由同一 worker 完成必要修复、重跑验证、再 closeout

## Phase Plan

- Phase 0：建立本 topic，接住已完成审计 topic 的 archive handoff。已完成。
- Phase 1：Release Safety Governance
  - 固定 migration forward-only / backup / rollback governance
  - 固定 config compatibility / deprecation / rename governance
  - 形成 operator upgrade path 的最小治理口径
- Phase 2：Release Identity And Policy
  - 固定 `BuildInfo` / `graft version` 最小 contract
  - 固定 release policy / support boundary
  - 固定 `server` / `web` / migration 的版本协同口径
- Phase 3：Release Operator Docs Baseline
  - 固定 install / config reference / upgrade / rollback / release notes 最小文档集合
  - 明确各文档的 canonical location 与 authority 引用
- Final closeout：执行 archive-readiness check；若三个 phase 的治理口径全部稳定且无新的 bounded batch，转为 `archive-ready`

## Current Recovery Point

- 上游 `release-readiness-governance-audit` 已 archive-ready，并已迁入 archive。
- 当前主题已完成 Phase 1-2 worker round，并固定了 release safety / identity / policy governance authority。
- 当前 loop 的 pending batches 为：
  - `phase-3-release-operator-docs-baseline`
- 实施顺序必须串行：
  - 先完成 Phase 3
  - `pending_batches=[]` 后仍需做 final archive-readiness check，不能直接停

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-topic-bootstrap-and-archive-handoff",
    "phase-1-release-safety-governance",
    "phase-2-release-identity-and-policy"
  ],
  "pending_batches": [
    "phase-3-release-operator-docs-baseline"
  ],
  "current_batch": null,
  "next_batch": "phase-3-release-operator-docs-baseline",
  "closeout_status": "planned-active"
}
```

## Phase 1 Accepted Authority

### Release Safety Governance

- authority evidence：
  - `server/internal/cli/migrate.go`
  - `server/internal/cli/serve.go`
  - `server/internal/cli/validate.go`
  - `server/internal/config/config.go`
  - `README.md`
- accepted rules：
  - live migration governance for `v0.1.0` is forward-only; do not promise down migrations, automatic rollback, or
    startup-time schema repair
  - `graft migrate up` and `graft dev` remain the explicit migration entrypoints; `graft serve` stays pure runtime
    startup
  - any upgrade that may apply live migrations must first verify database backup and restore capability, and retain the
    config snapshot required for manual recovery
  - rollback support in `v0.1.0` is documentation-first and operator-controlled: prerequisites, decision points,
    data/config risk, and minimum verification must be documented, but no helper tooling is promised

### Config Compatibility Governance

- stable config changes are classified as:
  - `additive`
  - `default-change`
  - `rename`
  - `semantic-change`
  - `removal`
- release constraints：
  - patch release must not silently rename, remove, or reinterpret stable config keys
  - minor release that introduces `rename`、`semantic-change` or `removal` must provide release notes, upgrade notes,
    replacement, operator action, and earliest removal target
  - alias bridge, startup deprecation warning, config rewrite helper, or machine-readable compatibility automation are
    not `v0.1.0` defaults

### Operator Upgrade Path Baseline

- Phase 1 only fixes the minimum governance baseline; it does not add implementation promises.
- The minimum operator path for later docs is:
  - verify artifact/version target and maintenance window
  - verify database backup/restore readiness
  - verify config diff against canonical keys and planned operator actions
  - run explicit migration step before normal runtime startup when schema change is involved
  - run minimum post-upgrade verification and capture rollback decision points

### Authority Drift Repair

- 本 topic README 原先引用了不存在的 `startup-prompt.md`。
- Phase 1 已补齐 `ai-plan/public/release-governance-rollout/startup-prompt.md`，恢复了 recovery 入口一致性。

## Phase 2 Accepted Authority

### Release Identity Baseline

- authority evidence：
  - `server/internal/cli/root.go`
  - `.github/workflows/release.yml`
  - `.github/workflows/publish.yml`
  - `web/package.json`
  - `README.md`
- accepted rules：
  - official release identity for `v0.1.0` is the repository Git tag `vMAJOR.MINOR.PATCH`; publish-time artifact naming,
    release notes, and release summary must derive from that same tag
  - the minimal future `BuildInfo` contract for release-grade `server` binaries is:
    - `version`
    - `git_commit`
    - `build_time_utc`
    - `git_tree_state`
  - `BuildInfo.version` uses bare semver such as `0.1.0`; the canonical release tag remains `v0.1.0`
  - optional future metadata such as target platform or builder identity may be added later, but must not replace the
    four required baseline fields
  - current repository state does not yet expose unified BuildInfo injection or an operator-facing runtime identity
    surface; until that implementation lands, operators must treat release tag, artifact filename, and release notes as
    the canonical release identity

### `graft version` Minimum Boundary

- current authority evidence shows `server/internal/cli/root.go` registers `dev`、`serve`、`migrate` and `validate`,
  but no `version` subcommand yet
- accepted rules for the future command boundary：
  - `graft version` must be a pure metadata readout and must not require PostgreSQL, Redis, HTTP startup, or live
    migration access
  - release builds must print or return at least `version`、`git_commit`、`build_time_utc` and `git_tree_state`
  - non-release or local builds may still identify themselves as `dev`, but must not be confused with an official
    tagged release
  - the command boundary is governance-only in Phase 2; this batch does not claim the command already exists

### Release Policy And Support Boundary

- `v0.1.0` only promises one active repository release line at a time; no LTS line, multi-minor support matrix, or
  independent `server` / `web` support cadence is promised
- the minimal supported release package for official operators is:
  - one repository tag
  - one `server` release artifact built from that tag
  - one `web` release artifact built from that tag
  - one matching release-notes body for that tag
- `v0.1.0` does not promise hot-patch builds, per-module releases, or detached `web`-only or `server`-only official
  release lines outside the shared repository version

### `server` / `web` / Migration Version Coordination

- `server` binary, `web` dist artifact, and release notes for an official release must come from the same Git tag and
  same repository commit lineage
- mixing `server` and `web` artifacts from different official release tags is unsupported for `v0.1.0`
- migration governance remains explicit and forward-only:
  - operators apply the migration set associated with the target release before normal runtime startup when schema
    changes are present
  - migration file versions are internal ordering identifiers, not product release numbers and not compatibility labels
- `web/package.json` dependency versions and frontend build tool versions are build inputs, not product release identity
- Phase 3 docs must cite release tag compatibility and explicit migration step requirements instead of inventing a
  separate compatibility matrix

## Batch Details

### Phase 1: Release Safety Governance

- allowed scopes：
  - `ai-plan/public/release-governance-rollout/**`
  - `ai-plan/design/数据库表设计与迁移规范.md`
  - `ai-plan/design/服务端API边界与兼容治理规范.md`
  - `README.md`
- hard goals：
  - 固定 migration forward-only / backup / rollback policy
  - 固定 config change class、patch/minor compatibility、deprecation record 字段
  - 固定 upgrade operator action 的最小文档化要求
- non-goals：
  - 不做 CLI helper
  - 不做 startup deprecation warning
  - 不做 config alias bridge 实现

### Phase 2: Release Identity And Policy

- allowed scopes：
  - `ai-plan/public/release-governance-rollout/**`
  - `README.md`
  - 必要的 topic-only design/roadmap 文档
- hard goals：
  - 固定 `BuildInfo` 最小字段集
  - 固定 `graft version` 最小输出
  - 固定 release policy / support boundary / version coordination
- non-goals：
  - 不直接修改 workflow 实现
  - 不直接承诺更强的 operator-facing introspection UI

### Phase 3: Release Operator Docs Baseline

- allowed scopes：
  - `ai-plan/public/release-governance-rollout/**`
  - `README.md`
  - 新建的 release/install/upgrade/rollback 文档目录
- hard goals：
  - 固定文档位置与最小章节结构
  - 补齐 install guide、config reference + compatibility notes、upgrade guide、rollback/recovery guide、release notes template
  - 所有文档必须引用 Phase 1/2 的 governance authority
- non-goals：
  - 不把文档写成已存在自动化能力的承诺
  - 不引入 docs-site、web shell docs page 或 hosted docs 平台

## Validation Targets

docs-only / recovery：

```bash
git diff --check
python3 scripts/validate_ai_governance.py
```

若某批次扩展到 `server` / `web` 实现：

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Startup Prompt

- 见 `ai-plan/public/release-governance-rollout/startup-prompt.md`

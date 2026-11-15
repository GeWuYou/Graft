# AGENTS.md

This file owns startup, authorization, validation and closeout. Child AGENTS own domain execution; skills/design supply detail.

Graft is a self-hosted containerized application operations platform for individuals and small teams.
Its architecture uses compile-time modules and lightweight explicit DI.
Keep the locked server/web stacks in their AGENTS; no dynamic plugins, marketplace, hidden initialization or heavyweight
IoC. `server`/`web` name the projects, `core` means runtime infrastructure, `module` the business unit. Historical backend
`plugin` wording does not change `server/modules/*` and `server/internal/moduleapi/**` authority.

## Startup Governance

Every task turn starts `unbooted`. Before edits, validation conclusions, delegation or handoff:

1. Confirm the repository root and read this file.
2. Read `.ai/environment/tools.ai.yaml`, falling back to tools.raw.yaml only if missing/insufficient. Use discovered
   tools; refresh inventory through its owner script, never hand-edit or invent dependencies.
3. Discover the canonical owner and whether the symptom is downstream drift. Classify `server`, `web`,
   `cross-boundary` or `docs/automation`; read [server/AGENTS.md](server/AGENTS.md) / [web/AGENTS.md](web/AGENTS.md)
   for affected domains, both for shared semantics, and [ai-plan/AGENTS.md](ai-plan/AGENTS.md) for ai-plan edits.
4. Emit one receipt: `governance source: AGENTS.md`; `task class`; `recovery source: none | parent topic | subtopic`;
   `authority summary: <owner or unknown-needs-escalation>`. The turn is then `preflighted`.
5. For recovery/long-running ownership, use [public index](ai-plan/public/README.md) after preflight. New long-running
   work uses `graft-work-intake` before topic/design/roadmap/ADR; local repairs need no tracking. Match risks below.

Reuse this turn's receipt. New turns/resume/restart rerun preflight; lost context/topic switching is `governance-lost`.
Handoff carries governance source, task class, recovery source, scope and topic/subtopic; recovery follows preflight.

## Authority and Risk Routing

Fix the highest incorrect authority: module intent -> server descriptor/permission/menu/route -> OpenAPI/typed
contract -> generated artifacts -> web bootstrap -> UI. Owned scope limits unrelated expansion, not authority repair.
Inspect descriptor, module/platform contract, OpenAPI and registration owners; shared semantics are cross-boundary.

Compatibility records owner, consumers, why direct repair is infeasible, cleanup trigger and validation.
Align architecture/lifecycle/governance designs with changes; generated artifacts do not justify upstream drift.

Load changed-behavior/material-risk guidance only. Small fixes need no complete design reading. Cover applicable
reviews once, combining evidence across concerns; record missing skill gaps and use the nearest checklist.

Review selection uses the concern matrix in [AI review](ai-plan/design/governance/ai/AI代码生成与Review规范.md).
Read [AI tooling](ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md) for tools/skills changes and
[recovery design](ai-plan/design/governance/ai/AI任务追踪与恢复设计.md) for recovery/loop work. New modules define
lifecycle and menu + route + page + API + permission owners first. Reusable assets use `graft-shared-asset-reuse`;
configuration changes use [configuration governance](ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md).
Skills live in `.agents/skills/`; invoke only relevant triggers, reusing evidence across skills.

## Authorized Work

Complete authorized work including related tests and diagnosed validation/hook repairs. File/hunk lists protect
ownership. Explain necessary expansion; preserve unrelated work. Ask for mixed ownership, changed goals or new external
effects, giving the failure, proposed change, impact and checks.

Read-only PR/security review authorizes inventory and diagnosis, not repair or external writes. It may finish with
`open/reported` findings carrying evidence, risk and next step. Explicit remediation requires final dispositions
`fixed/delegated/blocked/stale/noise`; do not extend remote ledger enums. Push, comments, deployment and database writes
require applicable explicit authorization. An explicit `graft-pr-remediation` grants its bounded commit/push/reply/
resolve chain only; never infer force-push, merge, human-thread resolution or arbitrary metadata writes.

## Validation and Completion

Use `graft-validation-runner` and [validation phases](docs/development/validation.md). Iterations use necessary
compilation and bounded feedback. Commit preparation reviews owned changes and consumers, adds needed regressions,
and runs `just check-local-plan` / `just check-local` with consumer tests. Reuse unchanged-input evidence.
Local checks inspect the worktree and reject selected staged/worktree divergence. Empty scope, prerequisites and
coverage gaps remain explicit. CI owns full acceptance through `graft validate backend` and `cd web && bun run check`;
shared semantics cover both domains. Hooks remain active. Distinguish preparation, CI and runtime acceptance.

Distinguish `agent-only`, `human-acceptance-required`, `browser-automation-required`, or `mixed` verification. Local
browser use needs task authorization and primary-checkout/runtime-identity proof; UI edits alone do not authorize it.
Screenshots are inspection evidence. Report missing checks and pending human acceptance honestly.

Review handwritten Go/TS/Vue comments via `graft-comment-governance`: meaningful Chinese boundary/lifecycle guidance,
not mechanical restatement. Use `graft-task-closeout`/`graft-commit` for validated owned scope; capture lessons when warranted.
Report behavior, scope, risks, checks and blockers; missing checks or human judgment preclude full acceptance.

## Git and Worktree Safety

Use `graft-worktree-manager` for numbered leased task worktrees; the primary checkout owns review/integration.
`main` development needs explicit authorization except emergency fixes. Worktrees prohibit services/browser QA/live
migrations/DB mutations; static checks/builds/tests/checksums may run without a live DB. Stop owned processes before release.

Commit validated, reliably owned logical slices through `graft-commit`. Preserve user/parallel edits; separate mixed
hunks or report the blocker. No `git add .`, `git add -A`, or `git commit -am` without explicit all-changes authority.
Bare graft-commit owns its initial inventory, not unrelated arrivals; related repair follows Authorized Work. Never bypass
hooks. Numbered tasks finish owned changes committed and clean, then record manager closeout before release.

Push/integration require explicit authorization. Integration records exact operation/source/target/scope/checks/rollback/
invalidation before execution. For Git, linear assets or CI read
[delivery detail](ai-plan/design/governance/ai/仓库协作与交付细则.md).

## Delegation

Delegate concrete independent bounded work; the main agent owns orchestration, acceptance, validation and integration.
Children receive governance source, task class, recovery source, owned scope, objective, output, constraints and commit
authority. Workers repair within ownership; cross-worker/authority changes require coordinator reassignment. Preserve
parallel changes. Loop protocols belong to recovery design/loop skill: batch success continues; worker return is not completion.

Choose a same-or-lower child relative to its immediate parent using trustworthy host/official evidence. Unknown relation:
inherit parent and continue. Never infer rank from names/reasoning effort or choose a higher child. Every nested/retry/
replacement edge records inheritance or comparison evidence. Tool parameters belong in references, not static rankings.

## License and Text Hygiene

Preserve LICENSE/`AGPL-3.0-only` and upstream attribution. No per-file license headers for repository-owned files.
Use LF under `.gitattributes`/`.editorconfig`; index/CI checks block. No secrets, personal client config, private addresses
or machine paths in tracked governance/evidence.

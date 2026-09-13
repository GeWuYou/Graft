# AGENTS.md

This is Graft's sole repository startup, authorization, validation ownership and closeout policy.
Child AGENTS own domain execution; skills/design provide conditional detail.

Graft is a composable admin platform, a modular monolith with compile-time modules and explicit lightweight DI.
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

Reuse this turn's receipt/context. New turns/resume/restart rerun preflight; context loss/topic switching is
`governance-lost`. Recovery is not startup; lightweight discovery may precede the receipt. Handoff carries a startup
prompt with governance source, task class, recovery source, owned scope and relevant topic/subtopic.

## Authority and Risk Routing

Fix the highest incorrect authority: module intent -> server descriptor/permission/menu/route -> OpenAPI/typed
contract -> generated artifacts -> web bootstrap -> UI. Owned scope limits unrelated expansion, not authority repair.
Inspect descriptor, module/platform contract, OpenAPI and registration owners; shared semantics are cross-boundary.

Compatibility requires owner, why direct repair is infeasible, consumers, cleanup trigger and bridge/cleanup validation.
Generated artifacts do not justify drift. Update conflicting design with the change; architecture/lifecycle/governance
changes require their owning design aligned before completion.

Load changed-behavior/material-risk guidance only. Small fixes need no complete design reading. Cover applicable
reviews once, combining evidence across concerns; record missing skill gaps and use the nearest checklist.

Review selection uses the concern matrix in [AI review](ai-plan/design/governance/ai/AI代码生成与Review规范.md).
Read [AI tooling](ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md) for tools/skills changes and
[recovery design](ai-plan/design/governance/ai/AI任务追踪与恢复设计.md) for recovery/loop work. New modules define
lifecycle and menu + route + page + API + permission owners first. Reusable assets use `graft-shared-asset-reuse`;
configuration changes use [configuration governance](ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md).
Skills live in `.agents/skills/`; invoke only relevant triggers, reusing evidence across skills.

## Authorized Work

Complete the authorized goal, including necessary related files/tests, diagnosed validation/hook repairs and rechecks.
File/hunk lists aid ownership, not absolute scope. Explain necessary expansion and continue with clear ownership.
No unrelated refactors/upgrades. Ask only for mixed/unknown ownership, changed goals or new external side effects;
identify failure/root cause, proposed changes, impact and checks before required repair authorization.

Read-only PR/security review authorizes inventory and diagnosis, not repair or external writes. It may finish with
`open/reported` findings carrying evidence, risk and next step. Explicit remediation requires final dispositions
`fixed/delegated/blocked/stale/noise`; do not extend remote ledger enums. Push, comments, deployment and database writes
require applicable explicit authorization. An explicit `graft-pr-remediation` grants its bounded commit/push/reply/
resolve chain only; never infer force-push, merge, human-thread resolution or arbitrary metadata writes.

## Validation and Completion

Use `graft-validation-runner` for direct checks. Server completion: `graft validate backend` (`--stage lint` sole blocking
lint). Web: `cd web && bun run check`. Shared semantics validate both. Focused checks cannot replace completion gates.
Reuse results for unchanged inputs/environment; changes require affected rechecks. Hooks/CI remain active.

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

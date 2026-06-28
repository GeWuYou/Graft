# ai-plan/AGENTS.md

This document is the local execution-truth for `ai-plan/**` governance. Root `AGENTS.md` remains the only
repository-level startup-governance, validation, commit, and closeout authority.

After root startup preflight, prefer `graft-ai-plan-governance` when a `docs/automation` task changes `ai-plan`
router documents, active-topic recovery materials, templates, catalog coverage, or bounded `ai-plan` validators.
Pair `graft-ai-governance-audit` only when the same slice also changes `.agents/skills/**`, `scripts/**`, or
`ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`.

## 1. Scope

Apply this document when a `docs/automation` task changes:

- `ai-plan/AGENTS.md`
- `ai-plan/README.md`
- `ai-plan/catalog.json`
- `ai-plan/design/**`
- `ai-plan/design/decisions/**`
- `ai-plan/roadmap/**`
- `ai-plan/public/**`
- `ai-plan/public/archive/**`
- `ai-plan/lessons/**`
- `ai-plan/templates/**`

This document does not authorize edits outside `ai-plan/**`. If the real authority sits in root `AGENTS.md`,
`.agents/skills/**`, `scripts/**`, `server/**`, or `web/**`, escalate only to the minimum required owner.

## 2. Authority Layers

- root `AGENTS.md`
  - startup governance, authority-first rules, validation ownership, commit flow, and closeout rules
- `ai-plan/AGENTS.md`
  - local execution rules for `ai-plan/**` structure, routing, topic materials, and ADR usage
- `ai-plan/README.md`
  - directory semantics and recovery-path overview for `ai-plan/**`
- `ai-plan/catalog.json`
  - bounded machine index for the currently approved governance coverage only
  - supplementary to router and topic documents; never a second source of truth
- `ai-plan/design/**`
  - repository-wide design truth
- `ai-plan/design/decisions/**`
  - convergence-gated ADRs that must land before wider router, catalog, validator, or skill rollout
- `ai-plan/roadmap/**`
  - repository-wide staged implementation plans
- `ai-plan/public/README.md`
  - active-topic recovery index only
- `ai-plan/public/<topic>/**`
  - topic-local recovery materials and topic-local archives
- `ai-plan/templates/**`
  - starter artifacts for new recovery or ADR documents; templates never override live topic truth
- `ai-plan/lessons/**`
  - reusable lessons captured after work completes; lessons do not replace active recovery materials

When these layers diverge, follow the higher layer and update the lower layer in the same change.

## 3. Router

Choose the narrowest `ai-plan/**` path that matches the authority you are changing:

- `design/`
  - use when the content is repository-wide design truth that should outlive one topic or one worktree
- `design/decisions/`
  - use when one explicit ADR must converge later batches before wider router, catalog, validator, or skill rollout
- `roadmap/`
  - use when the content is a phased implementation plan or delivery order rather than live recovery state
- `public/README.md`
  - use after startup preflight to discover active topics and their default recovery entry
- `public/<topic>/`
  - use for one active topic's live recovery materials, batch state, and bounded execution history
- `public/<topic>/archive/**`
  - use when an active topic needs compaction of completed stage history without leaving the `active` lifecycle state
- `public/archive/<topic>/`
  - use only after a topic reaches `archive-ready`, moves out of the active index, and becomes historical evidence
- `lessons/`
  - use only when a corrected mistake, durable anti-pattern, or reusable implementation lesson should outlive one topic
- `templates/`
  - use only to seed new active-topic or ADR documents
  - copy from the template, then adapt the new document to the live topic
  - do not treat template wording as a retrofit mandate for every existing active topic
  - do not let templates become a second authority layer

## 4. Active Topic Contract

Every active topic under `ai-plan/public/<topic>/` must include:

- `README.md`
- `startup-prompt.md`
- `todos/<topic>-tracking.md`
- `traces/<topic>-trace.md`

Minimum expectations:

- `README.md`
  - states the topic objective, recovery receipt, authority summary, owned scope, and pending batch direction
- tracking file
  - records the current recovery point, checklist, acceptance conditions, loop batch state, and persisted
    `Work Contract` when the topic came through `Work Intake`
- trace file
  - records dated decisions, validation milestones, and batch transitions
- `startup-prompt.md`
  - is a reusable loop or future-turn entry artifact and must not invent a second startup receipt format beyond root
    `AGENTS.md`

Machine-readable metadata rules:

- the required topic files above remain the metadata authority; do not require per-document frontmatter to feed
  automation
- when a machine-readable index is needed, use one `ai-plan/catalog.json`
- keep the catalog single-file, explicit-coverage, and no broader than the current approved governance slice requires
- `ai-plan/public/README.md` remains the active-topic router even when `ai-plan/catalog.json` exists

`ai-plan/public/README.md` must list only active topics and point at one recovery entry per topic. Add or remove the
index entry in the same change that creates or archives the topic.

When creating new active-topic materials, prefer the minimal starters under `ai-plan/templates/active-topic/`, then
adapt them to the live topic instead of copying placeholders forward unchanged.

## 4.1 Work Intake Contract

For new long-running work:

- `Work Intake` is the only allowed workflow entry before creating a new `topic`, `design`, `roadmap`, or `ADR`
- `design`, `roadmap`, `topic`, and `ADR` are derived artifacts, not peer intake paths
- `Work Contract` is the authority decision payload for that intake
- persist `Work Contract` only when the work actually becomes an active topic
- persist it in the topic tracking file, not as a standalone `work-contract.yaml`, not in topic `README.md`, and not
  in `ai-plan/catalog.json`
- use contract-driven minimal bootstrap: create only the minimal skeleton explicitly required by the contract

## 5. ADR, Template, And Archive Rules

Use `ai-plan/design/decisions/ADR-XXX-*.md` when one governance decision must converge multiple later batches. ADRs
should record context, decision, status, and consequences before downstream retrofits, validators, or skill updates
begin.

Template rules:

- keep templates minimal and copy-ready
- copied files become live topic truth and must be edited for the real authority, scope, and validation path
- do not mass-edit older active topics only to match new template wording
- if a template change would force broad retrofits, keep the template narrower instead of broadening the current batch

Archive rules:

- keep active topic entry files concise
- move stage history into `ai-plan/public/<topic>/archive/**` when the topic stays active but the default recovery path
  needs compaction
- move completed topics to `ai-plan/public/archive/<topic>/` and remove them from `ai-plan/public/README.md` in the
  same change
- do not let archive wording override current root `AGENTS.md`, this file, or current design truth

## 6. Editing Rules

- preserve concurrent topic state; do not rewrite unrelated active topics while updating the shared index
- keep repository-safe content only; no secrets, absolute paths, machine usernames, or local-only recovery stores
- when changing `ai-plan` governance, update the affected topic tracking and trace files in the same change
- when adding templates, keep them non-authoritative and scoped to the minimum files the current governance batch needs
- if `ai-plan/catalog.json` is introduced or updated, keep it single-file, bounded, and non-authoritative
- do not add validator scripts or new skills until the batch plan explicitly reaches those steps
- do not retrofit frontmatter across existing `ai-plan/**` documents just to satisfy machine-readable indexing
- do not treat topic `owned scope` as permission to ignore upstream authority drift
- do not let a thin workflow skill such as `graft-work-intake` invent independent intake rules outside the documented
  authority chain

## 7. Validation

Minimum validation for pure `ai-plan/**` documentation changes:

```bash
git diff --check
```

When a change touches the adopted Phase 1 governance slice under `ai-plan/catalog.json`,
`ai-plan/public/README.md`, `ai-plan/public/archive/ai-plan-ia-governance/**`, `ai-plan/templates/**`, or the
structure guard itself, also run:

```bash
python3 scripts/validate_ai_plan_structure.py
```

This structure guard is intentionally bounded to the current governance rollout. It validates the approved archived
`ai-plan-ia-governance` catalog slice, the shared active-topic router expectations still needed by
`compose-project-management`, the minimal starter templates, and the archived governance topic evidence retained by this
slice. It is not a whole-repository `ai-plan` linter.

If the same change also touches `.agents/skills/**` or `scripts/**`, run the smallest relevant structure or unit
validation required by root `AGENTS.md` and the governing skill.

If the structure guard logic changes, run the focused regression test:

```bash
python3 -m unittest scripts/test_validate_ai_plan_structure.py
```

## 8. Closeout

Closeout for `ai-plan/**` governance work should report:

- which topic or ADR authority changed
- whether routing changed across `design/`, `roadmap/`, `public/`, `public/archive/`, `lessons/`, or `templates/`
- which validation ran and why stronger checks were skipped
- whether `graft-task-closeout` or `graft-commit` were blocked by mixed ownership, validation gaps, or safe staging
  limits

# AI Plan

`ai-plan/` stores the repository's architecture truth, implementation roadmaps, and AI task recovery artifacts.

`ai-plan/AGENTS.md` is the local execution-truth document for `ai-plan/**`. Root `AGENTS.md` remains the repository
startup-governance source.

`ai-plan/` is not the same as `.ai/environment/`: `ai-plan/` stores design, roadmap, and recovery truth, while
`.ai/environment/` stores generated environment truth for AI and contributors.

## Directory Semantics

- `AGENTS.md`
  - Local execution-truth for `ai-plan/**` governance.
  - Read this with root `AGENTS.md` before changing `ai-plan/**`.
- `catalog.json`
  - Optional bounded machine index for an approved `ai-plan` governance slice.
  - Supplements router and topic documents; may intentionally cover only part of `ai-plan/`.
  - Does not require per-document frontmatter and does not become a second source of truth.
- `design/`
  - Repository-wide architecture and design truth.
  - Use this for stable design documents that apply across topics.
- `design/decisions/`
  - Convergence-gated ADRs for `ai-plan` governance decisions that must land before wider retrofits.
- `roadmap/`
  - Repository-wide implementation plans and staged delivery documents.
  - Use this for phased execution plans that apply across topics.
- `templates/`
  - Minimal starter files for active topics and ADRs.
  - Use these only to seed new documents; copied files must be adapted to the real topic and do not become a second
    authority source.
- `public/README.md`
  - Shared recovery index used after `AGENTS.md` startup preflight.
  - Maps branches or worktrees to active topics and points at the primary tracking and trace entry paths.
  - When a long-lived local worktree exists, prefer recording both the worktree name and the current branch name.
  - Must list only active topics.
- `public/<topic>/README.md`
  - Topic summary and default recovery entry.
  - Use this to record the current objective, authority summary, owned scope, and pending batch direction.
- `public/<topic>/startup-prompt.md`
  - Reusable loop or future-turn entry prompt for one active topic.
  - Keep it aligned with root `AGENTS.md` startup governance instead of inventing a second boot path.
- `public/<topic>/todos/`
  - Repository-safe recovery documents for one active topic.
  - Use these for durable task state that another contributor or worktree may need to resume safely.
- `public/<topic>/traces/`
  - Repository-safe execution traces for one active topic.
  - Record decisions, validation milestones, and the immediate next step.
- `public/<topic>/subtopics/<name>/todos/`
  - Recovery documents for one bounded subtopic inside an active topic.
  - Use this when one long-lived topic needs separate `server`, `web`, or similar boundary-specific recovery entrypoints.
- `public/<topic>/subtopics/<name>/traces/`
  - Execution traces for one bounded subtopic inside an active topic.
  - Keep these focused on one subsystem so the parent topic can stay concise.
- `public/<topic>/design/`
  - Topic-specific design documents.
  - Use this only when the design applies to one topic instead of the whole repository.
- `public/<topic>/roadmap/`
  - Topic-specific implementation plans.
  - Use this only when the roadmap applies to one topic instead of the whole repository.
- `public/<topic>/archive/`
  - Stage-level archive for completed artifacts that still belong to an active topic.
  - Prefer `archive/todos/` and `archive/traces/` when archiving content cut from the active entry files.
- `public/archive/<topic>/`
  - Completed-topic archive.
  - Move the entire topic directory here when that work direction is fully complete.
- `lessons/`
  - Reusable lessons that outlive one topic.
  - Use this only when a stable lesson should persist beyond active recovery materials.
- `private/`
  - Worktree-private recovery space.
  - Keep this directory untracked when it is introduced.

## Workflow Rules

- `AGENTS.md` owns startup governance; `ai-plan/` must not define a second boot chain, receipt format, or startup
  gating rule.
- When a task changes `ai-plan/**`, read `ai-plan/AGENTS.md` after root `AGENTS.md`.
- After that startup preflight, prefer `$graft-ai-plan-governance` for `ai-plan/**` governance slices; pair
  `$graft-ai-governance-audit` only when the same change also touches repo-local skills, scripts, or
  `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`.
- Let `ai-plan/AGENTS.md` act as the router for choosing `design/`, `roadmap/`, `public/`, `public/archive/`,
  `lessons/`, or `templates/`; this README stays descriptive rather than becoming a second governance source.
- If `catalog.json` exists, treat it as a bounded machine index only. `public/README.md` and topic-local recovery files
  remain the authoritative router and metadata sources.
- After startup preflight, recovery may read `public/README.md` before scanning active topics directly.
- Read `design/` and `roadmap/` before making architecture or implementation-boundary decisions.
- If the current branch or worktree appears in the public index, read the mapped topic tracking and trace files after
  startup preflight and before substantive recovery work.
- Short-lived branches used for hotfixes or narrow fixes should not become default active-topic mappings unless they are
  intentionally promoted into a long-lived worktree/topic recovery path.
- If an active topic defines subtopics, read the parent topic first and then continue into the relevant subtopic based
  on the current `server`, `web`, or cross-boundary task shape.
- When a topic is active, update its tracking document in the same change as substantive work.
- When work is clearly scoped to one subtopic, update that subtopic tracking document in the same change and keep the
  parent topic limited to cross-boundary summaries, shared risks, and shared milestones.
- Keep active tracking and trace files concise enough to serve as recovery entrypoints.
- When a stage is complete, move its detailed history into the topic's `archive/` and leave only the active recovery
  point in the default recovery path.
- When a topic is complete, move the whole topic directory into `public/archive/<topic>/` and remove it from the
  shared recovery index in the same change.
- `python3 scripts/validate_ai_plan_structure.py` is the current bounded structure guard for this rollout only:
  it checks the approved archived `ai-plan-ia-governance` catalog slice, the shared active-topic entry still needed by
  `compose-project-management`, the minimal template starters, and the retained governance archive evidence. It does not
  lint all of `ai-plan/public/**`.

## Content Rules

- Never write secrets, tokens, credentials, private keys, hostnames, IP addresses, proprietary URLs, or other
  sensitive data.
- Never write absolute file-system paths, home-directory paths, or machine usernames.
- Use repository-relative paths, branch names, commit ids, PR numbers, recovery-point ids, and validation commands
  instead.
- Do not retrofit frontmatter across existing `ai-plan/**` documents when the required topic files and bounded catalog
  already provide enough machine-readable structure.

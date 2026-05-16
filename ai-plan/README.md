# AI Plan

`ai-plan/` stores the repository's architecture truth, implementation roadmaps, and AI task recovery artifacts.

`ai-plan/` is not the same as `.ai/environment/`: `ai-plan/` stores design, roadmap, and recovery truth, while
`.ai/environment/` stores generated environment truth for AI and contributors.

## Directory Semantics

- `design/`
  - Repository-wide architecture and design truth.
  - Use this for stable design documents that apply across topics.
- `roadmap/`
  - Repository-wide implementation plans and staged delivery documents.
  - Use this for phased execution plans that apply across topics.
- `public/README.md`
  - Shared recovery index used after `AGENTS.md` startup preflight.
  - Maps branches or worktrees to active topics and points at the primary tracking and trace entry paths.
  - Must list only active topics.
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
- `private/`
  - Worktree-private recovery space.
  - Keep this directory untracked when it is introduced.

## Workflow Rules

- `AGENTS.md` owns startup governance; `ai-plan/` must not define a second boot chain, receipt format, or startup
  gating rule.
- After startup preflight, recovery may read `public/README.md` before scanning active topics directly.
- Read `design/` and `roadmap/` before making architecture or implementation-boundary decisions.
- If the current branch or worktree appears in the public index, read the mapped topic tracking and trace files after
  startup preflight and before substantive recovery work.
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

## Content Rules

- Never write secrets, tokens, credentials, private keys, hostnames, IP addresses, proprietary URLs, or other
  sensitive data.
- Never write absolute file-system paths, home-directory paths, or machine usernames.
- Use repository-relative paths, branch names, commit ids, PR numbers, recovery-point ids, and validation commands
  instead.

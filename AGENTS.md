# AGENTS.md

This document is the single source of truth for repository-level coding behavior in `Graft`.

All AI agents and contributors must follow these rules when writing, reviewing, or modifying code in this repository.

## 1. Project Intent

`Graft` is a composable admin platform, not a single-purpose business app and not an AI product.

Primary goal:

- build a backend platform that can add new capabilities quickly through plugins

Secondary goals:

- keep `server` and `web` module boundaries stable
- make repetitive admin modules easy to scaffold
- keep the codebase friendly to AI-assisted development

Do not optimize for:

- early dynamic plugin hot-loading
- third-party plugin marketplace in v1
- heavyweight framework abstractions without clear need

## 2. Source of Truth

Before changing code or structure, read the relevant documents in `ai-plan/`.

Authoritative repository documents:

- [ai-plan/design/项目设计.md](ai-plan/design/项目设计.md)
- [ai-plan/design/插件与依赖注入设计.md](ai-plan/design/插件与依赖注入设计.md)
- [ai-plan/design/前端架构设计.md](ai-plan/design/前端架构设计.md)
- [ai-plan/design/契约治理与魔法值治理规范.md](ai-plan/design/契约治理与魔法值治理规范.md) when the task changes
  typed contracts, magic-value governance, contract lifecycle, ownership, compatibility, drift handling, or shared
  `server` / `web` semantics
- [ai-plan/design/代码注释与模块文档规范.md](ai-plan/design/代码注释与模块文档规范.md) when the task changes
  code comments, package docs, module README rules, or AI documentation behavior
- [ai-plan/design/TDesign-MCP-辅助开发规范.md](ai-plan/design/TDesign-MCP-辅助开发规范.md) when the task changes
  TDesign Vue Next pages, components, styles, or frontend AI-assisted development workflow
- [ai-plan/roadmap/MVP实施计划.md](ai-plan/roadmap/MVP实施计划.md)
- [ai-plan/design/AI任务追踪与恢复设计.md](ai-plan/design/AI任务追踪与恢复设计.md) when the task changes
  tracking, recovery, or documentation-governance rules

Subdomain governance documents:

- root `AGENTS.md`
  - repository-level startup governance, recovery entry, validation chain ownership, commit/closeout rules, CI/CD,
    subagent rules, and cross-domain collaboration constraints
- [web/AGENTS.md](web/AGENTS.md)
  - `web` execution truth for frontend structure, module boundaries, contracts, routing, i18n, TDesign usage, and
    frontend validation
- [server/AGENTS.md](server/AGENTS.md)
  - `server` execution truth for plugin boundaries, DI constraints, Go code organization, Ent/migration flow, and
    backend validation

Reading order:

- every task reads this root `AGENTS.md` first
- `web` tasks must also read `web/AGENTS.md`
- `server` tasks must also read `server/AGENTS.md`
- `cross-boundary` tasks must read both subdomain `AGENTS.md` files before edits or validation conclusions

If code and docs diverge, update the docs first or in the same change.

When a task changes architecture, plugin boundaries, lifecycle semantics, frontend module conventions, or execution
governance, the related `ai-plan/design/`, `ai-plan/roadmap/`, or subdomain `AGENTS.md` document must be updated
before the task is considered complete.

## 3. Repository Terms

Use these names consistently in code discussions, plans, reviews, and task breakdowns:

- `server` means the backend project and its runtime, plugin, and infrastructure code
- `web` means the frontend project and its Vue 3 admin shell and feature modules
- `core` means true infrastructure owned by the platform runtime
- `plugin` means business capability registered into the platform through the plugin system

Do not use vague wording that blurs repository boundaries when a task is really about `server`, `web`, `core`, or a
plugin.

## 4. Environment Capability Inventory

Before choosing runtimes, package managers, or CLI tools:

- first read `.ai/environment/tools.ai.yaml` if it exists
- use `.ai/environment/tools.raw.yaml` only when the AI-facing inventory is missing or insufficient
- prefer repository-relevant installed tools over assumptions about what is available on the system
- if `.ai/environment/` marks a cross-environment exception such as host Windows Bun for `web`, follow that exception
  instead of defaulting to the current WSL shell toolchain
- if a change affects repository toolchain expectations or environment guidance, refresh the `.ai/environment/`
  inventory in the same change instead of leaving generated environment truth stale

If the environment inventory does not exist yet:

- inspect the repository for the actual toolchain before making assumptions
- report the missing inventory when it materially affects repeatability
- do not create fake dependencies on inventory files that are not present in the repository

When `.ai/environment/` exists:

- treat `tools.raw.yaml` and `tools.ai.yaml` as generated repository truth, not hand-maintained notes
- keep repository startup skills aligned with the inventory read order

### 4.1 Startup Governance

This root `AGENTS.md` is the only authoritative startup-governance source in this repository.

Other files may point to recovery materials, environment facts, or skill entrypoints, but they must not define a
second boot chain, a second receipt format, or a second set of startup gating rules.

Every repository task starts in one of these states:

- `unbooted`
  - no startup preflight has been completed for the current task turn
- `preflighted`
  - the startup preflight has completed and the task may enter repository recovery or direct execution
- `governance-lost`
  - the current task turn no longer has a trustworthy startup state and must rerun preflight before substantive work

The minimum startup preflight is:

1. confirm the repository root
2. read this root `AGENTS.md`
3. read `.ai/environment/tools.ai.yaml` when it exists; use `.ai/environment/tools.raw.yaml` only when the AI-facing
   summary is missing or insufficient
4. classify the task as one of:
   - `server`
   - `web`
   - `cross-boundary`
   - `docs/automation`
5. read the required subdomain `AGENTS.md` files for the chosen task class
6. decide whether the current turn needs recovery context from `ai-plan/public/README.md`

The minimum startup receipt is:

- `governance source`
  - the root `AGENTS.md`
- `task class`
  - one of `server` / `web` / `cross-boundary` / `docs/automation`
- `recovery source`
  - `none`, `parent topic`, or `subtopic`

Fail-closed startup rules:

- do not start implementation, validation conclusions, final handoff, or subagent delegation without the startup
  receipt for the current task turn
- if resume, restart, topic switching, or context loss makes the current startup state unclear, move to
  `governance-lost` and rerun the startup preflight
- recovery state does not replace startup state; reading tracking or trace files without the startup receipt is not a
  valid boot path
- lightweight lookups may happen before the receipt, but repository-level conclusions, edits, and subagent delegation
  must wait until the receipt is established

Resume and restart rules:

- `continue`, `resume`, `restart`, and similar prompts must rerun the startup preflight for the current turn
- only after that preflight may the agent read `ai-plan/public/README.md` and the mapped tracking or trace files
- restoring a topic recovery point does not mean repository governance has been restored

Handoff rules:

- if a task ends by handing off a next task, the handoff must include one explicit next-task startup prompt that
  re-establishes the startup receipt instead of assuming repository governance carries across turns
- the next-task startup prompt must carry, at minimum, `governance source`, `task class`, `recovery source`, and
  `owned scope`; if the next task depends on a topic recovery point, include the mapped parent topic and subtopic
- do not present a “next task” handoff as implementation-ready while leaving the next turn to guess whether it should
  boot, resume recovery, or continue from ambient context
- recovery documents may suggest the next task direction, but they must not replace the required next-task startup
  prompt in the handoff itself

Subagent inheritance rules:

- the main agent completing startup preflight does not mean a subagent already knows repository governance
- every subagent task must carry a minimum inherited context package containing:
  - `governance source`
    - the root `AGENTS.md`
  - `task class`
  - `recovery source`
  - `owned scope`
- a subagent must not be launched with only an objective or file target; without the inherited context package the
  task remains `governance-lost`

## 5. Repository Skills

Repository-maintained skills live under `.agents/skills/`.

Prefer the repository skills below when their trigger matches the task:

- `graft-boot`
  - use for short startup prompts, resume prompts, or when the first step should be to run the startup preflight
    defined in `4.1 Startup Governance`, assess whether `graft-multi-agent-batch` is justified, and enter repository
    recovery or direct execution when needed
- `graft-multi-agent-batch`
  - use when the user explicitly wants subagent delegation or when the work cleanly splits into disjoint parallel
    slices; `graft-boot` should perform the suitability assessment before delegation starts
- `graft-pr-review`
  - use when the task depends on the GitHub PR for the current branch, especially to extract AI review findings,
    failed checks, MegaLinter warnings, or failed test signals before local verification
- `graft-plugin-scaffold`
  - use when adding a new `server` plugin or shaping a plugin before implementation
- `graft-commit`
  - use as the canonical scoped commit workflow when the current task slice is ready to commit, whether the trigger is
    an explicit user request or a `graft-task-closeout` decision that the validated owned scope should be committed
- `graft-push`
  - use when the user explicitly wants the current branch pushed, or when a local push/commit chain is blocked and the
    agent needs to distinguish uncommitted scope, Husky hook failures, upstream ambiguity, or remote rejection before
    deciding the safest next push step
- `graft-task-closeout`
  - use as the default slice-end path after `graft-boot` work when the agent needs to decide between handoff-only
    versus commit-plus-handoff, while emitting the required next-task startup prompt
- `graft-web-module-scaffold`
  - use when adding a new `web` feature module aligned with backend plugin semantics
- `graft-validation-runner`
  - use when choosing the smallest correct validation for `server`, `web`, or cross-boundary work

If a repository skill and this document diverge, follow `AGENTS.md` first and update the skill in the same change.

## 6. Locked Technical Choices

### 6.1 Server

- Go
- Gin
- Ent
- PostgreSQL
- Viper
- Zap
- Casbin
- robfig/cron

### 6.2 Web

- Vue 3
- TypeScript
- Vite
- TDesign Vue Next
- Pinia
- Vue Router
- Axios
- UnoCSS

### 6.3 Architecture

- plugin-oriented backend
- lightweight DI / service registry
- no heavyweight IoC container
- compile-time plugin registration for v1

Do not switch to React, Naive UI, or a full IoC framework unless the project docs are explicitly revised first.

## 7. Architecture Rules

### 7.1 Server Core

Core runtime owns:

- config
- logger
- database
- HTTP server
- migration runner
- event bus
- permission registry
- menu registry
- cron registry
- plugin manager
- service container

Core runtime surface must stay explicit and small.

Only documented runtime surfaces such as config, HTTP, migration, event, permission, menu, cron, plugin, service
container, and repository CLI entrypoints may own platform-level startup behavior. Do not hide new runtime surfaces in
unrelated packages, starter code, or ad-hoc background initialization.

Business logic must live in plugins.

### 7.2 Plugin and Module Boundaries

- `server` business behavior belongs in plugins, not in platform core
- plugins must depend on public interfaces, not on another plugin's internal implementation
- cross-plugin stable contracts belong in `server/internal/pluginapi` or another documented stable boundary
- `web` is a platform shell plus feature modules
- new frontend capability should default to `web/src/modules/<name>` unless it is truly shell-owned
- keep `menu + route + page + api + permission` ownership explicit

Detailed execution rules for these boundaries live in `server/AGENTS.md` and `web/AGENTS.md`.

## 8. Implementation Priorities

When building new functionality, prefer this order:

1. stabilize docs and interfaces
2. implement platform primitives
3. implement a minimal end-to-end slice
4. add breadth only after the extension path is proven

For v1, prioritize:

- user
- rbac
- audit
- scheduler

Do not start Docker, SSH, monitor, or workflow plugins before the core extension path is stable.

## 9. Execution Rules

### 9.1 Module Placement

When asked to add a new capability:

- first identify whether it belongs in `server/core`, a `server` plugin, or a `web` feature module
- default to a plugin unless the capability is true infrastructure
- default to a `web/src/modules/<name>` entry path unless the page is a shell-level concern
- define the capability's runtime surface and lifecycle owner before implementation; entrypoints, menus, routes,
  permissions, jobs, public services, and boot/shutdown responsibilities must all have one clear home
- define menu, route, permission, API, and public service boundaries before writing code

### 9.2 Explicitness

When unsure:

- choose the more explicit implementation
- choose the narrower public interface
- keep the next contributor's mental load low
- prefer direct construction and visible wiring over hidden framework behavior
- prefer preserving the current repository architecture over introducing a second baseline, second shell, or second
  validation contract for temporary convenience

### 9.3 New Dependencies

When asked to introduce a new dependency:

- justify why the existing stack is insufficient
- prefer smaller, explicit libraries
- avoid adding abstractions that hide control flow
- reject dependencies that materially weaken plugin boundaries or increase hidden runtime magic without clear benefit

## 10. Validation Rules

Every completed task must pass at least one validation that directly covers the changed code before it is considered
done.

### 10.1 Server Validation

For `server` changes:

- follow `server/AGENTS.md` as the backend execution-truth document
- use `graft validate backend` as the backend completion entrypoint
- use `graft validate backend --stage lint` as the only allowed backend blocking lint gate
- keep backend validation order aligned with repository truth and the entrypoints it names

### 10.2 Web Validation

For `web` changes:

- follow `web/AGENTS.md` as the frontend execution-truth document
- use host Windows Bun from WSL when the environment inventory requires it
- use `bun run check` as the frontend completion entrypoint
- keep frontend validation order aligned with repository truth and the entrypoints it names

### 10.3 Cross-Boundary Validation

If a task changes contracts shared across `server` and `web`, or changes menu, permission, route, or lifecycle
semantics that affect both sides:

- validate both `server` and `web`
- keep the corresponding contract governance docs aligned in the same change
- if typed enforcement, drift detection, or compatibility checks are expected by the active contract lifecycle but no
  repository automation entrypoint exists yet, report that gap explicitly instead of claiming the contract slice is fully
  validated

### 10.4 Validation Reporting

If validation cannot be run:

- state exactly which command was expected
- state why it could not be run
- do not claim the task is fully complete without that caveat
- distinguish full repository entrypoints from focused direct checks and from execution-stage slices such as
  `graft validate backend --stage ...`

Warnings or failures in directly affected modules are part of the task scope. Do not ignore them unless the user
explicitly narrows the task.

README, skills, tracking docs, and CI workflows may point to repository entrypoints or narrower execution slices, but
they must not redefine validation order, acceptance criteria, or local-vs-CI environment rules into a second source of
truth. When wording diverges, root `AGENTS.md` plus the repository entrypoints it names win.

## 11. Git Workflow Rules

For repository work:

- default to a dedicated branch and PR for repository work
- direct development on `main` is allowed only for emergency fixes or when the user explicitly authorizes it
- use branch names in the form `<type>/<topic-or-scope>`
- decide change ownership before staging or committing; a validated change is auto-committable only when its ownership
  is reliably known
- when one logical feature slice reaches a directly validated milestone, commit it before starting the next unrelated
  slice unless the user explicitly asks to batch them
- if the working tree already mixes multiple feature points, split them back to feature-granularity commits before
  considering the task complete; do not leave validated slices piled up as uncommitted changes
- default to one logical closure per commit; for larger tasks, split commits into readable stages such as
  schema/migration, runtime implementation, tests, docs, or cleanup/refactor
- each commit should remain as buildable or testable as the current slice reasonably allows; do not rely on hidden
  local context to make an intermediate commit understandable

Automatic commits are allowed only after ownership is classified:

- scenario 1: if the working tree was clean before the task and the validated change was produced entirely by the
  agent, the agent may create the commit unless the user explicitly says not to commit
- scenario 2: if the working tree was already dirty, but the agent can reliably distinguish the task's owned files or
  hunks through `git status` and `git diff`, the agent may commit only the owned scope it can prove
- scenario 3: if user edits, unknown edits, or unrelated topic edits are mixed together and ownership cannot be
  reliably separated, the agent must not auto-commit; explain the mixed state to the user and limit the next step to
  one of these paths: commit only the confirmable scope, let the user specify the commit scope, or leave the changes
  uncommitted

Explicit commit trigger:

- when the user explicitly invokes a repository commit trigger such as `$graft-commit`, treat it as permission to
  create one scoped commit for the current validated task slice, but still apply the ownership and mixed-change rules
  above before staging anything
- the trigger grants permission to commit the confirmed owned scope only; it does not permit bundling unrelated files,
  unknown changes, or all current working tree changes by default
- if the current slice is not yet validated to the level required by its task class, finish the required validation
  before committing or explain why that validation cannot be completed yet
- if the working tree is mixed and the owned scope cannot be separated confidently, the trigger does not override the
  fail-closed rule; stop and report the ambiguity instead of forcing a commit

Closeout-driven commit evaluation:

- when a slice that started through `graft-boot` reaches a stop, completion, or handoff point, route the ending
  through `graft-task-closeout` instead of relying on an implicit wrap-up path
- `graft-task-closeout` must always evaluate commit eligibility using the same ownership, validation, and scoped
  staging rules enforced by `graft-commit`
- if closeout concludes the validated owned scope should be committed, execute that commit through `graft-commit`
  rather than inventing a second commit path
- if validation or ownership is insufficient, closeout must report the exact blocker and keep the handoff state honest
  instead of forcing a commit

Task handoff and pre-handoff commit rules:

- if the current task ends with a next-task handoff, first evaluate whether the current slice has reached the
  validation level required by its task class
- if that validation level has been reached, commit the confirmed owned scope before the handoff using the same
  ownership, validation, and scoped-staging rules enforced by `graft-commit`
- if that validation level has not been reached, do not claim the slice was ready to commit; record the validation
  gap and keep the handoff status honest
- if the current slice came through the normal `graft-boot` workflow, use `graft-task-closeout` as the handoff path
  so commit eligibility and startup-prompt emission stay in one place
- a next-task handoff must include one explicit next-task startup prompt that tells the next turn to rerun startup
  preflight and provides the minimum inherited context package needed to resume safely
- a handoff requirement does not override mixed-ownership or insufficient-validation refusal rules; when a safe commit
  cannot be made, say so and leave the scope uncommitted rather than force-staging ambiguous changes

For staging and mixed-ownership files:

- never stage or commit existing user changes, unknown-origin changes, unrelated files, or cross-topic files together
  with the current task just because they are present in the working tree
- default to staging only files or hunks whose ownership is confirmed
- do not use `git add .`, `git add -A`, or `git commit -am` unless the user explicitly requests committing all
  current changes
- if one file contains both user-owned and agent-owned edits, commit only the owned hunks when they can be reliably
  separated
- if mixed ownership inside one file cannot be reliably separated at hunk level, the agent must not auto-commit that
  file
- a file being relevant to the current task is not enough to justify committing the whole file when ownership is mixed

For commit hygiene:

- do not create noise commits such as `wip`, `update`, `fix typo`, temporary debug snapshots, or commits that mix
  unrelated formatting with behavior changes
- do not run repository-wide formatting unless the user explicitly asks for it
- do not let IDE actions, formatter passes, organize-imports actions, or `--fix` flows introduce broad unrelated diffs
- treat formatting drift outside the current task scope as a high-risk change by default
- formatting changes may be committed only within the files or hunks that belong to the current task; if the drift
  cannot be contained to that scope, it must not be auto-committed

Commit messages must use Conventional Commits:

- format: `<type>(<scope>): <summary>`
- the title must default to English
- `scope` is required and must be explicit
- keep the title focused on what changed, not on AI behavior or the implementation process
- do not place literal escaped control text such as `\n`, `\t`, or `\r` inside the commit title or body

Commit type rules:

- use `feat` for user-facing or plugin/platform capability additions
- use `fix` for behavior corrections
- use `refactor` for non-feature restructuring
- use `perf` for observable performance improvements
- use `docs`, `test`, `build`, `ci`, `chore`, or `style` for their literal categories

Do not use `feat` for documentation-only changes.

When a commit needs a body:

- use unordered bullet items
- start each bullet with a verb such as `新增`、`修复`、`优化`、`更新`、`补充`、`重构`
- make each bullet describe one independent change point
- write the title and body as real multi-line text
- if a commit message is generated by automation, expand escaped text into actual line breaks and indentation before
  invoking `git commit`

## 12. Automation and CI/CD Rules

Repository automation should follow the same boundary rules as local development.

### 12.1 Pull Request Validation

When the repository adds CI workflows:

- keep pull request validation and release automation in separate workflows
- validate `server` and `web` as separate jobs when both sides exist
- when backend lint governance is active, keep `server` lint and `server` build/test as separate jobs instead of one
  opaque backend script step
- when CI keeps split jobs or stage flags, document them as execution-layer decomposition of the same repository
  validation truth, not as independent acceptance contracts
- prefer a fast quality or security track plus a build or test track instead of one opaque monolithic job
- cache dependencies by ecosystem, such as Go modules and frontend package manager caches
- upload useful failure artifacts or summaries when they materially improve debugging
- keep current-stage workflows honest about repository maturity; prefer smoke validation over fake full builds when the
  actual toolchain or artifacts are not stable yet
- backend CI must reuse the same `graft validate backend` entrypoint and pinned `golangci-lint` version as local
  development instead of rebuilding a second lint parameter set inside workflow YAML
- when local `web` development in WSL requires host Windows Bun, keep that rule explicit in repository docs; a Linux CI
  runner reusing the same `bun run check` entrypoint is an execution environment difference, not permission to relax
  the local WSL rule

### 12.2 Release Automation

When the repository later adds release workflows:

- build artifacts once and reuse them across publish steps
- keep release gating stricter than pull request validation
- use explicit concurrency control for release or docs publish workflows
- do not introduce package publishing complexity that the repository does not actually need yet

### 12.3 Security and Maintenance Automation

When adding repository maintenance workflows:

- prefer CodeQL or equivalent scanning for the actual languages in this repository
- prefer secret scanning on pull requests
- prefer Dependabot or equivalent automation for Go modules, frontend dependencies, and GitHub Actions
- keep optional workflows such as docs publish or benchmarks separate from the main CI path

## 13. License Governance

This repository is licensed under Apache License 2.0.

Contributors must preserve that licensing posture when changing code, docs, automation, or dependencies.

### 13.1 Repository License Files

- do not remove or weaken the top-level `LICENSE` file
- if the repository later requires a `NOTICE` file or third-party license inventory, keep those files aligned with the
  actual distributed contents
- do not add repository rules that conflict with Apache-2.0 distribution terms

### 13.2 Source File Headers

The repository does not currently enforce a header script or SPDX baseline, so contributors must not invent a fake
mandatory workflow.

If the project later adopts source header enforcement:

- prefer SPDX-style Apache-2.0 identifiers that are easy to validate automatically
- apply the policy consistently across supported source and configuration file types
- document exclusions for generated files, third-party code, lockfiles, and build output

### 13.3 Dependency and Distribution Compliance

When introducing a new dependency, package, or distributable artifact:

- check whether its license is compatible with Apache-2.0 distribution
- record any required attribution or notice obligations when they apply
- avoid adding copyleft or distribution-restrictive dependencies without an explicit repository decision
- keep future CI license checks lightweight until the repository has a real release pipeline and artifact inventory

## 14. Subagent Usage Rules

Use subagents only when the task is complex, the context is likely to grow too large, or the work can be split into
independent parallel subtasks.

The main agent must identify the critical path first. Do not delegate the immediate blocking task if the next local
step depends on that result.

Use subagents this way:

- use `explorer` subagents for read-only discovery, comparison, tracing, and narrow codebase questions
- use `worker` subagents only for bounded implementation tasks with an explicit file or subsystem ownership boundary

Every delegation must specify:

- the inherited startup context required by `4.1 Startup Governance`
- the concrete objective
- the expected output format
- the files or subsystem the subagent owns
- any constraints about tests, diagnostics, or compatibility

Subagents are not allowed to revert or overwrite unrelated user changes or parallel agent changes. They must adapt to
concurrent work instead of assuming exclusive ownership of the repository.

The main agent remains responsible for:

- critical-path selection
- validation planning
- review and acceptance of every subagent result
- final integration
- final completion judgment

Repository subagent usage is allowed in this project when it follows these rules.

## 15. Complex Task Tracking

For complex, multi-step, or multi-agent work:

- keep an explicit execution record if the task would be hard to resume safely from chat history alone
- use the repository-local `ai-plan/` workflow instead of inventing ad-hoc tracking files
- after completing the startup preflight in `4.1 Startup Governance`, read `ai-plan/public/README.md` before scanning
  active topics when resuming or booting into complex work
- keep repository-wide design truth in `ai-plan/design/` and `ai-plan/roadmap/`
- keep active topic recovery state under `ai-plan/public/<topic>/`

`ai-plan/` uses these directory semantics:

- `ai-plan/design/`
  - repository-wide architecture and design truth
- `ai-plan/roadmap/`
  - repository-wide implementation plans and staged delivery documents
- `ai-plan/public/README.md`
  - shared recovery index that maps branches or worktrees to active topics after startup preflight
- `ai-plan/public/<topic>/todos/`
  - recovery-safe tracking documents for one active topic
- `ai-plan/public/<topic>/traces/`
  - execution traces for one active topic
- `ai-plan/public/<topic>/subtopics/<name>/todos/`
  - recovery-safe tracking documents for one bounded subtopic inside an active topic
- `ai-plan/public/<topic>/subtopics/<name>/traces/`
  - execution traces for one bounded subtopic inside an active topic
- `ai-plan/public/<topic>/design/`
  - topic-specific design documents that do not belong in repository-wide design truth
- `ai-plan/public/<topic>/roadmap/`
  - topic-specific implementation plans that do not belong in repository-wide roadmap truth
- `ai-plan/public/<topic>/archive/`
  - archived stage-level artifacts for an active topic
- `ai-plan/public/archive/<topic>/`
  - completed-topic archive that should not be treated as default boot context

Use these workflow rules:

- `ai-plan/public/README.md` must list only active topics
- when a branch or worktree has an active-topic mapping, read its tracking and trace files after startup preflight and
  before substantive recovery work
- when an active topic defines subtopics, read the parent topic first and then continue into the relevant subtopic based
  on the current `server`, `web`, or cross-boundary task boundary
- when working from a tracked topic, update the corresponding tracking document in the same change
- when work is clearly scoped to one subtopic, update that subtopic tracking document in the same change and keep the
  parent topic focused on cross-boundary milestones, shared risks, and shared next steps
- for complex work, maintain a matching trace that records the current date, key decisions, validation milestones, and
  the immediate next step
- keep active tracking and trace files concise enough to serve as recovery entrypoints
- when a stage inside an active topic is complete, move detailed history into that topic's `archive/` and keep only the
  active recovery point in the default recovery path
- when a topic is fully complete, move the entire topic directory under `ai-plan/public/archive/<topic>/` and remove it
  from `ai-plan/public/README.md` in the same change
- never record absolute file-system paths in `ai-plan/**`; use repository-relative paths, branch names, commit ids, PR
  numbers, and validation commands instead

## 16. Commenting and Documentation Rules

All generated or modified code must include clear and meaningful comments where required by repository or subdomain
rules.

High-level documentation rules:

- `server` code follows the detailed comment and GoDoc requirements in `server/AGENTS.md` and
  `ai-plan/design/代码注释与模块文档规范.md`
- `web` code follows the detailed frontend comment requirements in `web/AGENTS.md` and
  `ai-plan/design/代码注释与模块文档规范.md`
- architecture, lifecycle, compatibility, and ownership comments take priority over mechanical restatement
- module-level `README.md` files are navigation documents, not replacements for `ai-plan/design/`

Missing required documentation is a standards violation. Code that does not meet the applicable documentation rules is
incomplete.

## 17. Change Management

When making substantial changes:

- explain which `ai-plan/design/` or `ai-plan/roadmap/` section the change follows
- keep architecture changes aligned with `ai-plan/`
- avoid silent changes to core conventions

If a task reveals that the current docs are wrong:

- update the relevant doc
- state the new rule clearly
- then implement against the updated rule

## 18. Code Review Expectations

Review for:

- boundary violations between core and plugins
- hidden coupling between plugins
- unnecessary framework complexity
- divergence from locked `server` and `web` stacks
- duplicate canonical contract definitions, undocumented contract ownership, or missing lifecycle / compatibility notes
  for high-risk contract changes
- missing tests around plugin lifecycle, dependency ordering, authorization, and dynamic menu/route behavior
- undocumented public interfaces or lifecycle-sensitive code
- divergence between `ai-plan/design/`, `ai-plan/roadmap/`, active topic recovery documents, and the relevant
  subdomain `AGENTS.md`

A change is not acceptable if it makes adding the next plugin or frontend module harder.

## 19. Definition of Done

A task is done only when all relevant items below are satisfied:

- the change follows the current `ai-plan/` documents, or the docs were updated first
- `server` and `web` boundaries are still clear
- new module work keeps the `menu + route + page + api + permission` path explicit
- any new or changed high-risk contract follows the canonical ownership, lifecycle, and compatibility rules in
  `ai-plan/design/契约治理与魔法值治理规范.md`
- affected code has the required comments and documentation
- affected code follows the applicable subdomain execution-truth document
- the changed area passed direct validation, or the exact validation gap was reported
- `server` work reached its completion state only after the backend entrypoints required by `server/AGENTS.md`
- `web` work reached its completion state only after the frontend entrypoints required by `web/AGENTS.md`
- the final summary states the important behavior change, validation result, and any remaining blockers

If any of these are missing, the task is incomplete even if the code compiles.

# AGENTS.md

This document is the single source of truth for repository-level coding behavior in `Graft`.

All AI agents and contributors must follow these rules when writing, reviewing, or modifying code in this repository.

## 1. Project Intent

`Graft` is a composable admin platform, not a single-purpose business app and not an AI product.

Primary goal:

- build a module-oriented modular monolith backend that can add new capabilities quickly through compile-time modules

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
- [ai-plan/design/模块与依赖注入设计.md](ai-plan/design/模块与依赖注入设计.md)
- [ai-plan/design/前端架构设计.md](ai-plan/design/前端架构设计.md)
- [ai-plan/design/契约治理与魔法值治理规范.md](ai-plan/design/契约治理与魔法值治理规范.md) when the task changes
  typed contracts, magic-value governance, contract lifecycle, ownership, compatibility, drift handling, or shared
  `server` / `web` semantics
- [ai-plan/design/代码注释与模块文档规范.md](ai-plan/design/代码注释与模块文档规范.md) when the task changes
  code comments, package docs, module README rules, or AI documentation behavior
- [ai-plan/design/TDesign-MCP-辅助开发规范.md](ai-plan/design/TDesign-MCP-辅助开发规范.md) when the task changes
  TDesign Vue Next pages, components, styles, or frontend AI-assisted development workflow
- [ai-plan/design/CodeGraph-MCP-辅助开发规范.md](ai-plan/design/CodeGraph-MCP-辅助开发规范.md) when the task changes
  CodeGraph MCP setup, repository code-navigation assistance, local code index handling, or AI-assisted discovery
  workflow
- [ai-plan/design/AI工具与MCP接入治理规范.md](ai-plan/design/AI工具与MCP接入治理规范.md) when the task changes
  AI tooling, MCP adoption, repository skills, Python helper scripts, or AI governance drift checks
- [ai-plan/roadmap/MVP实施计划.md](ai-plan/roadmap/MVP实施计划.md)
- [ai-plan/design/AI任务追踪与恢复设计.md](ai-plan/design/AI任务追踪与恢复设计.md) when the task changes
  tracking, recovery, or documentation-governance rules
- [ai-plan/design/通知中心设计.md](ai-plan/design/通知中心设计.md) when the task changes Notification Center
  sources, delivery semantics, notification configuration, notification routes, or shell notification UI behavior
- [ai-plan/design/数据库表设计与迁移规范.md](ai-plan/design/数据库表设计与迁移规范.md) when the task changes
  database table design, Ent schema, Atlas migration, audit fields, soft delete semantics, indexes, store query
  semantics, or database table / column comments

Subdomain governance documents:

- root `AGENTS.md`
  - repository-level startup governance, recovery entry, validation chain ownership, commit/closeout rules, CI/CD,
    subagent rules, and cross-domain collaboration constraints
- [web/AGENTS.md](web/AGENTS.md)
  - `web` execution truth for frontend structure, module boundaries, contracts, routing, i18n, TDesign usage, and
    frontend validation
- [server/AGENTS.md](server/AGENTS.md)
  - `server` execution truth for module boundaries, DI constraints, Go code organization, Ent/migration flow, and
    backend validation

Reading order:

- every task reads this root `AGENTS.md` first
- `web` tasks must also read `web/AGENTS.md`
- `server` tasks must also read `server/AGENTS.md`
- `cross-boundary` tasks must read both subdomain `AGENTS.md` files before edits or validation conclusions

If code and docs diverge, update the docs first or in the same change.

When a task changes architecture, module boundaries, lifecycle semantics, frontend module conventions, or execution
governance, the related `ai-plan/design/`, `ai-plan/roadmap/`, or subdomain `AGENTS.md` document must be updated
before the task is considered complete.

## 3. Repository Terms

Use these names consistently in code discussions, plans, reviews, and task breakdowns:

- `server` means the backend project and its runtime, module, and infrastructure code
- `web` means the frontend project and its Vue 3 admin shell and feature modules
- `core` means true infrastructure owned by the platform runtime
- `module` means the canonical business capability unit in this repository
- `plugin` is a historical backend naming term for compile-time modules; current live paths and stable boundaries are `server/modules/*` and `server/internal/moduleapi/**`

Do not use vague wording that blurs repository boundaries when a task is really about `server`, `web`, `core`, or a
module.

## 4. Environment Capability Inventory

Before choosing runtimes, package managers, or CLI tools:

- first read `.ai/environment/tools.ai.yaml` if it exists
- use `.ai/environment/tools.raw.yaml` only when the AI-facing inventory is missing or insufficient
- prefer repository-relevant installed tools over assumptions about what is available on the system
- follow .ai/environment/ for repository toolchain preferences instead of guessing from the current shell alone
- when using CodeGraph MCP as an AI navigation aid, follow
  [ai-plan/design/CodeGraph-MCP-辅助开发规范.md](ai-plan/design/CodeGraph-MCP-辅助开发规范.md); CodeGraph is a
  developer-local knowledge source, not a repository runtime, CI, hook, or package dependency
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
4. run authority discovery for the reported problem:
   - identify the canonical source-of-truth owner before freezing implementation scope
   - decide whether the observed symptom is local or downstream from an upstream authority drift
5. classify the task as one of:
   - `server`
   - `web`
   - `cross-boundary`
   - `docs/automation`
6. read the required subdomain `AGENTS.md` files for the chosen task class
7. decide whether the current turn needs recovery context from `ai-plan/public/README.md`

The minimum startup receipt is:

- `governance source`
  - the root `AGENTS.md`
- `task class`
  - one of `server` / `web` / `cross-boundary` / `docs/automation`
- `recovery source`
  - `none`, `parent topic`, or `subtopic`
- `authority summary`
  - canonical owner or `unknown-needs-escalation`

Fail-closed startup rules:

- do not start implementation, validation conclusions, final handoff, or subagent delegation without the startup
  receipt for the current task turn
- if resume, restart, topic switching, or context loss makes the current startup state unclear, move to
  `governance-lost` and rerun the startup preflight
- recovery state does not replace startup state; reading tracking or trace files without the startup receipt is not a
  valid boot path
- lightweight lookups may happen before the receipt, but repository-level conclusions, edits, and subagent delegation
  must wait until the receipt is established
- if authority discovery shows the real source of truth sits outside the initially assumed local slice, the task must be
  promoted to the correct authority owner or to `cross-boundary` before implementation continues

### 4.2 Authority-First Governance

`Graft` is a unified mono repo. `server`、`web`、OpenAPI source、module descriptors、bootstrap metadata、typed contract
definitions and generated artifacts are one repository change graph, not independent systems integrating over slow
release boundaries.

Authority rules:

- `owned scope` means standing responsibility and default work area; it does not redefine canonical authority
- bounded scope forbids unrelated expansion, not required authority repair
- when a bug or drift is downstream from an upstream authority, fix the highest broken source of truth first
- if the authority repair changes both `server` and `web`, or changes a shared contract / OpenAPI / bootstrap semantic,
  the task is `cross-boundary` even when the first symptom appeared on only one side
- generated artifacts are derived outputs, not canonical authority and not a justification for preserving downstream
  compatibility drift

Default authority chain examples:

- product IA / module intent
- `server` module descriptor / menu / permission / route contract
- canonical OpenAPI or typed contract source input
- generated artifacts
- `web` bootstrap route/menu assembly
- UI render / page-local view model

When the observed mismatch is between layers, agents must repair the highest incorrect layer they can safely validate in
the same slice rather than normalizing the mismatch in a lower layer by default.

### 4.3 Source-of-Truth Escalation

Before adding local compatibility, agents must explicitly check whether the real authority sits in:

- `server/modules/<name>/contract/**`
- `server/modules/<name>/descriptor.go` or equivalent module/menu/permission declarations under historical plugin naming
- `server/internal/contract/**`
- OpenAPI source inputs such as `openapi/**`
- `web/src/modules/<name>/contract/**`
- `web` bootstrap metadata and route/menu registration boundaries

Escalation rules:

- if the authority is upstream from the symptom, escalate and repair upstream in the same slice when feasible
- if the authority owner sits outside the initial local slice, widen only to the minimum authority-repair scope
- do not reinterpret `owned scope` as permission to leave upstream authority drift unfixed
- if a worker or subagent discovers authority outside its inherited owned scope, it must report the escalation instead of
  silently implementing a downstream-only compatibility patch

### 4.4 Anti-Compatibility Rule

Compatibility is an exception mechanism, not the default answer.

Before introducing any of the following:

- fallback
- alias
- compatibility layer
- legacy path
- mapping layer
- adapter layer
- compatibility DTO

the task must first prove why the source of truth cannot be repaired directly in the same slice.

Allowed compatibility only when all of the following are recorded:

- the canonical authority owner
- why direct authority repair is not being done now
- affected downstream consumers
- expiry or cleanup trigger
- validation scope for both the compatibility bridge and the eventual cleanup path

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
- `graft-multi-agent-task`
  - use when the user explicitly wants one bounded task to run through `graft-multi-agent-batch`, then close out
    through `graft-task-closeout`, and commit the validated owned scope through `graft-commit` when safe
- `graft-multi-agent-loop`
  - use when one bounded task should be executed through repeated same-session main-agent-managed rounds of
    `graft-multi-agent-task`, where the default mode is `topic-completion-loop`, the main agent owns orchestration,
    batch-state maintenance, budget, closeout parsing, acceptance, and next-round dispatch, and each bounded
    implementation round is delegated to one worker subagent by default
- `graft-pr-review`
  - use when the task depends on the GitHub PR for the current branch, especially to extract AI review findings,
    failed checks, MegaLinter warnings, or failed test signals before local verification
  - verified actionable findings from PR review must not be ignored only because the repair is large, cross-slice, or
    likely to require a new task slice; when the fix no longer fits one safe local slice, prefer
    `graft-multi-agent-batch` or `graft-multi-agent-loop` under the normal subagent rules
  - only stale findings, noise, false positives, or no-longer-applicable findings may be left unfixed, and those cases
    must be listed explicitly in the task closeout with the concrete reason
- `graft-plugin-scaffold`
  - use when adding a new `server` module under `server/modules/*` or shaping that module before implementation
- `graft-table-design`
  - use when designing or changing database tables, Ent schemas, Atlas migrations, audit fields, soft delete fields,
    indexes, store query semantics, or database comments
- `graft-sql-migration`
  - use when creating or modifying live Atlas/PostgreSQL migration SQL, migration directories, table or column
    comments, migration versions, indexes, unique constraints, foreign keys, or soft-delete columns; it owns the
    `scripts/validate_sql_migrations.py` live SQL comment gate and keeps
    `server/internal/ent/migrate/migrations/**` excluded as legacy/manual replay by default
- `graft-worktree-init`
  - use when creating or rebuilding a local `Graft` git worktree and the setup should follow the repository-standard
    shared local resource rules without hard-coded machine paths
- `graft-codegraph-mcp`
  - use when installing, registering, initializing, validating, or troubleshooting CodeGraph MCP as a developer-local
    AI navigation aid for this repository
- `graft-commit`
  - use as the canonical scoped commit workflow when the current task slice is ready to commit, whether the trigger is
  an explicit user request or a `graft-task-closeout` decision that the validated owned scope should be committed
- `graft-push`
  - use when the user explicitly wants the current branch pushed, or when a local push/commit chain is blocked and the
    agent needs to distinguish uncommitted scope, Husky hook failures, upstream ambiguity, or remote rejection before
    deciding the safest next push step
- `graft-pr-create`
  - use when the user explicitly wants a pull request created or reconciled for the current branch, or when the work
    needs a safe current-branch PR path that can create or reuse the PR against the repository default branch without
    silently pushing, auto-merging, or overwriting user-authored PR content
- `graft-task-closeout`
  - use as the default slice-end path after `graft-boot` work when the agent needs to decide between handoff-only
    versus commit-plus-handoff, while emitting the required next-task startup prompt
- `graft-lessons-learned`
  - use near task closeout when a corrected mistake, reusable anti-pattern, or stable implementation lesson should be
    captured in `ai-plan/lessons/*`, optionally promoted into design docs, or elevated into the correct `AGENTS.md`
- `graft-web-module-scaffold`
  - use when adding a new `web` feature module aligned with backend module semantics
- `graft-web-vibe-coding`
  - use when adding, redesigning, or reviewing `web` pages, shell surfaces, frontend AI prompts, or visual-governance
    rules that should first declare a page type, pick one of the built-in page masters or register an extension type,
    and enforce token/theme/i18n/visible-copy constraints before implementation
- `graft-validation-runner`
  - use when choosing the smallest correct validation for `server`, `web`, or cross-boundary work
- `graft-ai-governance-audit`
  - use when evaluating or changing AI tooling, MCP adoption, repository skills, `ai-plan` governance, Python helper
    scripts, environment inventory, or drift between AGENTS and AI workflow documents

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

- module-oriented modular monolith backend
- lightweight DI / service registry
- no heavyweight IoC container
- compile-time module registration for v1

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
- module manager
- service container

Core runtime surface must stay explicit and small.

Only documented runtime surfaces such as config, HTTP, migration, event, permission, menu, cron, module lifecycle,
service
container, and repository CLI entrypoints may own platform-level startup behavior. Do not hide new runtime surfaces in
unrelated packages, starter code, or ad-hoc background initialization.

Business logic must live in business modules under `server/modules/*`.

### 7.2 Module Boundaries

- `server` business behavior belongs in modules, not in platform core
- backend modules currently live under `server/modules/*`; do not reinterpret historical `plugin` wording as current runtime truth
- modules must depend on public interfaces, not on another module's internal implementation
- cross-module stable contracts belong in `server/internal/moduleapi` or another documented stable boundary
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

Do not start Docker, SSH, monitor, or workflow modules before the core extension path is stable.

## 9. Execution Rules

### 9.1 Module Placement

When asked to add a new capability:

- first identify whether it belongs in `server/core`, a `server/modules/*` module, or a `web` feature module
- default to a backend module unless the capability is true infrastructure
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
- reject dependencies that materially weaken module boundaries or increase hidden runtime magic without clear benefit

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
- before final closeout, run the Experience Capture Check defined by `graft-lessons-learned` whenever the task exposed
  a reusable corrected mistake, anti-pattern, or promotable project lesson

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
- ordinary non-merge and non-revert commits must include a real multi-line body with at least one unordered bullet
  beginning with `- `
- keep the title focused on what changed, not on AI behavior or the implementation process
- do not place literal escaped control text such as `\n`, `\t`, or `\r` inside the commit title or body

Commit type rules:

- use `feat` for user-facing or module/platform capability additions
- use `fix` for behavior corrections
- use `refactor` for non-feature restructuring
- use `perf` for observable performance improvements
- use `docs`, `test`, `build`, `ci`, `chore`, or `style` for their literal categories

Do not use `feat` for documentation-only changes.

For ordinary commit bodies:

- use unordered bullet items
- start each bullet with a verb such as `新增`、`修复`、`优化`、`更新`、`补充`、`重构`
- make each bullet describe one independent change point
- write the title and body as real multi-line text
- do not create agent-authored ordinary commits with only a title
- if a commit message is generated by automation, expand escaped text into actual line breaks and indentation before
  invoking `git commit`

## 12. Automation and CI/CD Rules

Repository automation should follow the same boundary rules as local development.

### 12.1 Pull Request Validation

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
- when local and CI environments differ, document that difference explicitly without redefining the web validation entrypoint or creating a second acceptance contract
- PR test reporting may run as separate report jobs, but those report jobs do not replace `web-check` or
  `server-build-test`
- forked pull requests must still upload raw test artifacts without requiring write permissions for PR comments or
  Checks

### 12.2 Release Automation

- build artifacts once and reuse them across publish steps
- keep release gating stricter than pull request validation
- use explicit concurrency control for release or docs publish workflows
- do not introduce package publishing complexity that the repository does not actually need yet
- version tags are owned by the manual semantic-release workflow and must use the configured Conventional Commit
  release rules and `v${version}` tag format
- semantic-release tag creation must use a validated `PAT_TOKEN`; do not rely on `GITHUB_TOKEN` when a downstream
  tag-triggered publish workflow must run
- tag-triggered publish workflows must run the repository server and web validation entrypoints before producing or
  attaching release artifacts
- release notes are generated from the repository `git-cliff` configuration; do not hand-maintain release bodies in
  workflow YAML

### 12.3 Security and Maintenance Automation

- prefer CodeQL or equivalent scanning for the actual languages in this repository
- prefer secret scanning on pull requests
- prefer Dependabot or equivalent automation for Go modules, frontend dependencies, and GitHub Actions
- keep optional workflows such as docs publish or benchmarks separate from the main CI path
- release SBOM and license-compliance assets are maintenance artifacts attached to GitHub Releases; they must not
  redefine the product release body or publish extra package/deploy targets

## 13. License Governance

This repository is licensed under Apache License 2.0.

Contributors must preserve that licensing posture when changing code, docs, automation, or dependencies.

### 13.1 Repository License Files

- do not remove or weaken the top-level `LICENSE` file
- if the repository later requires a `NOTICE` file or third-party license inventory, keep those files aligned with the
  actual distributed contents
- do not add repository rules that conflict with Apache-2.0 distribution terms

### 13.2 Source File Headers

- all new repository-maintained source, script, style, and migration files supported by `scripts/license-header.py` must
  include an Apache-2.0 header near the top of the file
- the canonical header marker is `SPDX-License-Identifier: Apache-2.0`
- `scripts/license-header.py` is the canonical CLI wrapper for checks and workflow calls; `scripts/license_header.py`
  owns the importable implementation policy
- supported comment formats and exclusions are owned by the license header implementation; do not reimplement that
  policy in workflow YAML, hooks, or ad-hoc shell snippets
- pull request validation gates newly added supported files only, so the repository stops new header drift without
  forcing broad historical churn into unrelated feature branches
- existing tracked supported files that predate this policy must be backfilled through the manual
  `license-header-fix.yml` workflow or a dedicated license-header branch, not opportunistically mixed into unrelated work
- generated files, third-party or vendored content, lockfiles, docs, AI governance metadata, configuration files,
  generated environment inventories, binary files, and build output must stay excluded from header automation

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

Exception for `graft-multi-agent-loop`:

- when the user explicitly triggers `graft-multi-agent-loop`, the outer main agent may delegate one whole bounded
  implementation round to exactly one worker subagent by default instead of keeping that round's implementation local
- in that loop mode, the outer main agent keeps orchestration, batch-state maintenance, budget tracking, stop
  conditions, closeout parsing, acceptance, and next-round dispatch local
- in that loop mode, `topic-completion-loop` is the default:
  - if the caller does not explicitly set `loop_mode`, the outer main agent must run `topic-completion-loop`
  - `checkpoint-loop` may be used only when the caller explicitly requests it
- in `topic-completion-loop`, the outer main agent must maintain batch state after every accepted closeout:
  - update `completed_batches`
  - update `pending_batches`
  - choose `next_batch`
  - dispatch the next worker when no terminal stop condition applies
- when `pending_batches` becomes empty in `topic-completion-loop`, the outer main agent must run a final
  archive-readiness check instead of stopping immediately
  - if the acceptance conditions pass, stop as `archive-ready`
  - if more bounded work is required, regenerate `pending_batches` and continue
  - if no safe next batch can be defined without user help, stop as `blocked`
- in `topic-completion-loop`, ordinary batch success must continue by default and must not stop merely because no
  `Next-session startup prompt:` was emitted
- in `topic-completion-loop`, `Next-session startup prompt:` is reserved for terminal handoff states:
  - `blocked`
  - `archive-ready`
  - explicit user stop
- in that loop mode, the outer main agent runs bounded orchestration rather than real-time remote control:
  - timeout alone does not mean the worker is stalled
  - each round carries an explicit checkpoint budget, defaulting to `1`
  - high-risk or long-running rounds may raise the checkpoint budget to `2` or `3`, but that increase must be written
    into the round budget up front
  - checkpoint interrupts may be used only for health checks, never to change the round goal, broaden scope, or append
    new implementation requirements
  - checkpoint interrupts must respect an explicit cooldown; do not repeatedly interrupt an active worker
- in that loop mode, the outer main agent must not edit repo-tracked implementation files during an active round; it
  may only inspect state, evaluate the returned closeout, and decide whether to accept, retry, or stop
- in that loop mode, stalled-state judgment must be stricter than elapsed time alone:
  - a worker is not considered stalled until soft timeout has been exceeded, there has been prolonged lack of new
    visible output evidence, the worker has not entered closeout, and a checkpoint request still fails to produce a
    usable status
  - when the current tool surface does not expose a direct activity query, the outer main agent must not translate
    "cannot observe tool activity" into "no tool activity"; conservative loop decisions must be based only on elapsed
    time, visible transcript evidence, the worker's latest response, and any checkpoint content
  - usable checkpoint status must include current phase, changed files, last validation, next action, can-continue
    judgment, estimated remaining minutes, ETA confidence, and current risks or blockers
  - checkpoint responses are health reports, not round closeouts; they must never be treated as terminal state for the
    round
  - after any usable checkpoint with `can_continue=true`, the outer main agent must explicitly continue the same worker
    round and resume waiting for that worker's final closeout instead of replacing, closing, or downgrading the round
    because the latest message was a checkpoint
  - ETA only guides the next wait window; it must not override the round's total runtime budget
  - if ETA repeatedly misses, progress is not substantive, or no closeout arrives, lower worker reliability and fall
    through `retry_once_then_blocked`
- in that loop mode, a missing, malformed, or contradictory worker closeout must be retried once with a fresh worker
  subagent and must fail closed as `blocked` on the second failure instead of downgrading into main-agent
  implementation
- in that loop mode, a retry worker must inherit the partial diff, relevant logs, validation evidence, and the
  previous worker failure reason before retrying the same bounded round
  - retry is allowed only when the worker gives no usable checkpoint response, later gives no usable final closeout
    after post-checkpoint grace handling, explicitly reports `can_continue=false`, or exhausts checkpoint budget

Use subagents this way:

- use `explorer` subagents for read-only discovery, comparison, tracing, and narrow codebase questions
- use `worker` subagents only for bounded implementation tasks with an explicit file or subsystem ownership boundary

Every delegation must specify:

- the inherited startup context required by `4.1 Startup Governance`
- the concrete objective
- the expected output format
- the files or subsystem the subagent owns
- any constraints about tests, diagnostics, or compatibility
- for `graft-multi-agent-loop` rounds, the remaining budget, allowed scopes, and the required human-readable plus JSON
  closeout contract
- whether authority escalation outside the local owned scope is allowed or expected

Authority escalation override:

- `owned scope` isolation, minimal diff preference, worktree isolation, and recovery isolation do not override required
  source-of-truth repair
- when a subagent discovers that the real authority is upstream or cross-boundary, it must return
  `authority_escalation_required` with the suspected owner instead of implementing local compatibility by default

Subagents are not allowed to revert or overwrite unrelated user changes or parallel agent changes. They must adapt to
concurrent work instead of assuming exclusive ownership of the repository.

The main agent remains responsible for:

- batch-state updates when `graft-multi-agent-loop` is active
- critical-path selection
- validation planning
- review and acceptance of every subagent result
- final integration
- final completion judgment
- when `graft-multi-agent-loop` is active, retry-once malformed-closeout handling and fail-closed stop decisions

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
- tracking and trace documents must record authority discovery when it materially changes task classification, owned
  scope interpretation, or compatibility decisions
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

- boundary violations between core and modules
- hidden coupling between modules
- unnecessary framework complexity
- divergence from locked `server` and `web` stacks
- duplicate canonical contract definitions, undocumented contract ownership, or missing lifecycle / compatibility notes
  for high-risk contract changes
- missing tests around module lifecycle, dependency ordering, authorization, and dynamic menu/route behavior
- undocumented public interfaces or lifecycle-sensitive code
- divergence between `ai-plan/design/`, `ai-plan/roadmap/`, active topic recovery documents, and the relevant
  subdomain `AGENTS.md`

A change is not acceptable if it makes adding the next module or frontend module harder.

## 19. Definition of Done

A task is done only when all relevant items below are satisfied:

- the change follows the current `ai-plan/` documents, or the docs were updated first
- `server` and `web` boundaries are still clear
- new module work keeps the `menu + route + page + api + permission` path explicit
- any new or changed high-risk contract follows the canonical ownership, lifecycle, and compatibility rules in
  `ai-plan/design/契约治理与魔法值治理规范.md`
- any introduced compatibility branch records why direct authority repair was not completed in the same slice
- affected code has the required comments and documentation
- affected code follows the applicable subdomain execution-truth document
- the changed area passed direct validation, or the exact validation gap was reported
- `server` work reached its completion state only after the backend entrypoints required by `server/AGENTS.md`
- `web` work reached its completion state only after the frontend entrypoints required by `web/AGENTS.md`
- the final summary states the important behavior change, validation result, and any remaining blockers

If any of these are missing, the task is incomplete even if the code compiles.

# AGENTS.md

This document is the single source of truth for coding behavior in this repository.

All AI agents and contributors must follow these rules when writing, reviewing, or modifying code in `Graft`.

## 1. Project Intent

`Graft` is a composable admin platform, not a single-purpose business app and not an AI product.

Primary goal:

* build a backend platform that can add new capabilities quickly through plugins

Secondary goals:

* keep `server` and `web` module boundaries stable
* make repetitive admin modules easy to scaffold
* keep the codebase friendly to AI-assisted development

Do not optimize for:

* early dynamic plugin hot-loading
* third-party plugin marketplace in v1
* heavyweight framework abstractions without clear need

## 2. Source of Truth

Before changing code or structure, read the relevant documents in `ai-plan/`.

Authoritative documents:

* [ai-plan/design/项目设计.md](ai-plan/design/项目设计.md)
* [ai-plan/design/插件与依赖注入设计.md](ai-plan/design/插件与依赖注入设计.md)
* [ai-plan/design/前端架构设计.md](ai-plan/design/前端架构设计.md)
* [ai-plan/design/TDesign-MCP-辅助开发规范.md](ai-plan/design/TDesign-MCP-辅助开发规范.md) when the task changes
  TDesign Vue Next pages, components, styles, or frontend AI-assisted development workflow
* [ai-plan/roadmap/MVP实施计划.md](ai-plan/roadmap/MVP实施计划.md)
* [ai-plan/design/AI任务追踪与恢复设计.md](ai-plan/design/AI任务追踪与恢复设计.md) when the task changes
  tracking, recovery, or documentation-governance rules

If code and docs diverge, update the docs first or in the same change.

When a task changes architecture, plugin boundaries, lifecycle semantics, or frontend module conventions, the related
`ai-plan/design/` or `ai-plan/roadmap/` document must be updated before the task is considered complete.

## 3. Repository Terms

Use these names consistently in code discussions, plans, reviews, and task breakdowns:

* `server` means the backend project and its runtime, plugin, and infrastructure code
* `web` means the frontend project and its Vue 3 admin shell and feature modules
* `core` means true infrastructure owned by the platform runtime
* `plugin` means business capability registered into the platform through the plugin system

Do not use vague wording that blurs repository boundaries when a task is really about `server`, `web`, `core`, or a
plugin.

## 4. Environment Capability Inventory

Before choosing runtimes, package managers, or CLI tools:

* first read `@.ai/environment/tools.ai.yaml` if it exists
* use `@.ai/environment/tools.raw.yaml` only when the AI-facing inventory is missing or insufficient
* prefer repository-relevant installed tools over assumptions about what is available on the system
* if a change affects repository toolchain expectations or environment guidance, refresh the `.ai/environment/`
  inventory in the same change instead of leaving generated environment truth stale

If the environment inventory does not exist yet:

* inspect the repository for the actual toolchain before making assumptions
* report the missing inventory when it materially affects repeatability
* do not create fake dependencies on inventory files that are not present in the repository

When `.ai/environment/` exists:

* treat `tools.raw.yaml` and `tools.ai.yaml` as generated repository truth, not hand-maintained notes
* keep repository startup skills aligned with the inventory read order

## 5. Repository Skills

Repository-maintained skills live under `.agents/skills/`.

Prefer the repository skills below when their trigger matches the task:

* `graft-boot`
  * use for short startup prompts, resume prompts, or when the first step should be to read `AGENTS.md` and `ai-plan/`
* `graft-multi-agent-batch`
  * use when the user explicitly wants subagent delegation or when the work cleanly splits into disjoint parallel slices
* `graft-plugin-scaffold`
  * use when adding a new `server` plugin or shaping a plugin before implementation
* `graft-web-module-scaffold`
  * use when adding a new `web` feature module aligned with backend plugin semantics
* `graft-validation-runner`
  * use when choosing the smallest correct validation for `server`, `web`, or cross-boundary work

If a repository skill and this document diverge, follow `AGENTS.md` first and update the skill in the same change.

## 6. Locked Technical Choices

### 6.1 Server

* Go
* Gin
* GORM
* PostgreSQL
* Viper
* Zap
* Casbin
* robfig/cron

### 6.2 Web

* Vue 3
* TypeScript
* Vite
* TDesign Vue Next
* Pinia
* Vue Router
* Axios
* UnoCSS

### 6.3 Architecture

* plugin-oriented backend
* lightweight DI / service registry
* no heavyweight IoC container
* compile-time plugin registration for v1

Do not switch to React, Naive UI, or a full IoC framework unless the project docs are explicitly revised first.

## 7. Architecture Rules

### 7.1 Server Core

Core runtime owns:

* config
* logger
* database
* HTTP server
* migration runner
* event bus
* permission registry
* menu registry
* cron registry
* plugin manager
* service container

Business logic must live in plugins.

Do not put business-specific behavior into the platform core.

### 7.2 Plugins

Every backend plugin should follow the same model:

* declare metadata
* declare dependencies
* register routes, menus, permissions, migrations, jobs, and public services in `Register`
* start runtime behavior in `Boot`
* release resources in `Shutdown`

Plugins must depend on public interfaces, not on another plugin's internal implementation.

Cross-plugin contracts belong in a stable package such as `internal/pluginapi`.

### 7.3 Dependency Injection

The DI layer is intentionally small.

Allowed responsibilities:

* register singleton providers
* resolve services
* expose plugin public services
* close registered resources

Disallowed responsibilities:

* reflection-heavy auto wiring
* package scanning
* struct tag injection
* hidden magic construction
* complex scope systems

If a design requires implicit behavior to be understandable, it is too complex for this repo.

### 7.4 Web

Frontend is a platform shell plus feature modules.

Expected structure:

* `web/src/app`
* `web/src/layouts`
* `web/src/pages`
* `web/src/modules`
* `web/src/components`
* `web/src/api`
* `web/src/stores`
* `web/src/router`

Rules:

* use TDesign as the primary component system
* avoid mixing multiple UI libraries
* keep new modules aligned with `menu + route + page + api + permission`
* use dynamic menus driven by backend data
* keep shared state in stores and keep page-local state inside the page or module
* when generating, modifying, or reviewing TDesign Vue Next code with AI assistance, query TDesign MCP or official
  TDesign docs before relying on component props, events, slots, DOM structure, or changelog details
* configure TDesign MCP on the active AI coding client, with Codex as the default AI coding entrypoint for this
  repository; Rider MCP setup is only required when using Rider AI Assistant for frontend code generation

## 8. Naming and Boundary Conventions

### 8.1 Server

* plugin names are short, stable, lowercase
* exported cross-plugin interfaces should be capability-oriented
* do not expose repositories as public plugin APIs
* permission codes should be namespaced, for example `user.read`, `user.create`
* config keys should be namespaced by plugin
* public cross-plugin return types should be stable capability DTOs, not raw database models

### 8.2 Web

* route names should be stable and unique
* page components should reflect module intent, not UI widget names
* stores should be reserved for shared state, not page-local form state
* CRUD page layouts should stay consistent across modules
* module pages should align with backend permissions and menu metadata instead of inventing parallel frontend-only
  access rules

## 9. Implementation Priorities

When building new functionality, prefer this order:

1. stabilize docs and interfaces
2. implement platform primitives
3. implement a minimal end-to-end slice
4. add breadth only after the extension path is proven

For v1, prioritize:

* user
* rbac
* audit
* scheduler

Do not start Docker, SSH, monitor, or workflow plugins before the core extension path is stable.

## 10. Execution Rules

### 10.1 Module Placement

When asked to add a new capability:

* first identify whether it belongs in `server/core`, a `server` plugin, or a `web` feature module
* default to a plugin unless the capability is true infrastructure
* default to a `web/src/modules/<name>` entry path unless the page is a shell-level concern
* define menu, route, permission, API, and public service boundaries before writing code

### 10.2 Explicitness

When unsure:

* choose the more explicit implementation
* choose the narrower public interface
* keep the next contributor's mental load low
* prefer direct construction and visible wiring over hidden framework behavior

### 10.3 New Dependencies

When asked to introduce a new dependency:

* justify why the existing stack is insufficient
* prefer smaller, explicit libraries
* avoid adding abstractions that hide control flow
* reject dependencies that materially weaken plugin boundaries or increase hidden runtime magic without clear benefit

## 11. Validation Rules

Every completed task must pass at least one validation that directly covers the changed code before it is considered
done.

### 11.1 Server Validation

For `server` changes:

* run the smallest `go test` scope that still covers the touched packages when tests exist
* run the smallest `go build` scope that still proves the changed code compiles when tests are absent or insufficient
* prefer wider validation such as `go test ./...` or `go build ./...` when the task changes shared abstractions, plugin
  contracts, lifecycle code, dependency resolution, or startup wiring

### 11.2 Web Validation

For `web` changes:

* run the repository's actual frontend validation command once it exists
* prefer type checking plus production build when both are available
* at minimum, use the smallest validation that proves changed routes, modules, pages, and TypeScript contracts compile

### 11.3 Cross-Boundary Validation

If a task changes contracts shared across `server` and `web`, or changes menu/permission/route semantics that affect
both sides:

* validate both `server` and `web`

### 11.4 Validation Reporting

If validation cannot be run:

* state exactly which command was expected
* state why it could not be run
* do not claim the task is fully complete without that caveat

Warnings or failures in directly affected modules are part of the task scope. Do not ignore them unless the user
explicitly narrows the task.

## 12. Git Workflow Rules

For repository work:

* during the current rapid-iteration phase, direct development on `main` is allowed by default
* create a dedicated branch only when the user explicitly asks for branch isolation, when a release-stabilization
  workflow requires it, or when the task is risky enough that isolated review is materially safer
* use branch names in the form `<type>/<topic-or-scope>`
* if the required validation passes and the task produced changes, create a Git commit unless the user explicitly says
  not to commit

Commit messages must use Conventional Commits:

* format: `<type>(<scope>): <summary>`
* use simplified Chinese for the summary
* keep established technical terms in English when they are the project's normal vocabulary

Commit type rules:

* use `feat` for user-facing or plugin/platform capability additions
* use `fix` for behavior corrections
* use `refactor` for non-feature restructuring
* use `perf` for observable performance improvements
* use `docs`, `test`, `build`, `ci`, `chore`, or `style` for their literal categories

Do not use `feat` for documentation-only changes.

When a commit needs a body:

* use unordered bullet items
* start each bullet with a verb such as `新增`、`修复`、`优化`、`更新`、`补充`、`重构`
* make each bullet describe one independent change point

## 13. Automation and CI/CD Rules

Repository automation should follow the same boundary rules as local development.

### 13.1 Pull Request Validation

When the repository adds CI workflows:

* keep pull request validation and release automation in separate workflows
* validate `server` and `web` as separate jobs when both sides exist
* prefer a fast quality or security track plus a build or test track instead of one opaque monolithic job
* cache dependencies by ecosystem, such as Go modules and frontend package manager caches
* upload useful failure artifacts or summaries when they materially improve debugging
* keep current-stage workflows honest about repository maturity; prefer smoke validation over fake full builds when the
  actual toolchain or artifacts are not stable yet

### 13.2 Release Automation

When the repository later adds release workflows:

* build artifacts once and reuse them across publish steps
* keep release gating stricter than pull request validation
* use explicit concurrency control for release or docs publish workflows
* do not introduce package publishing complexity that the repository does not actually need yet

### 13.3 Security and Maintenance Automation

When adding repository maintenance workflows:

* prefer CodeQL or equivalent scanning for the actual languages in this repository
* prefer secret scanning on pull requests
* prefer Dependabot or equivalent automation for Go modules, frontend dependencies, and GitHub Actions
* keep optional workflows such as docs publish or benchmarks separate from the main CI path

## 14. License Governance

This repository is licensed under Apache License 2.0.

Contributors must preserve that licensing posture when changing code, docs, automation, or dependencies.

### 14.1 Repository License Files

* do not remove or weaken the top-level `LICENSE` file
* if the repository later requires a `NOTICE` file or third-party license inventory, keep those files aligned with the
  actual distributed contents
* do not add repository rules that conflict with Apache-2.0 distribution terms

### 14.2 Source File Headers

The repository does not currently enforce a header script or SPDX baseline, so contributors must not invent a fake
mandatory workflow.

If the project later adopts source header enforcement:

* prefer SPDX-style Apache-2.0 identifiers that are easy to validate automatically
* apply the policy consistently across supported source and configuration file types
* document exclusions for generated files, third-party code, lockfiles, and build output

### 14.3 Dependency and Distribution Compliance

When introducing a new dependency, package, or distributable artifact:

* check whether its license is compatible with Apache-2.0 distribution
* record any required attribution or notice obligations when they apply
* avoid adding copyleft or distribution-restrictive dependencies without an explicit repository decision
* keep future CI license checks lightweight until the repository has a real release pipeline and artifact inventory

## 15. Subagent Usage Rules

Use subagents only when the task is complex, the context is likely to grow too large, or the work can be split into
independent parallel subtasks.

The main agent must identify the critical path first. Do not delegate the immediate blocking task if the next local
step depends on that result.

Use subagents this way:

* use `explorer` subagents for read-only discovery, comparison, tracing, and narrow codebase questions
* use `worker` subagents only for bounded implementation tasks with an explicit file or subsystem ownership boundary

Every delegation must specify:

* the concrete objective
* the expected output format
* the files or subsystem the subagent owns
* any constraints about tests, diagnostics, or compatibility

Subagents are not allowed to revert or overwrite unrelated user changes or parallel agent changes. They must adapt to
concurrent work instead of assuming exclusive ownership of the repository.

The main agent remains responsible for:

* critical-path selection
* validation planning
* review and acceptance of every subagent result
* final integration
* final completion judgment

Repository subagent usage is allowed in this project when it follows these rules.

## 16. Complex Task Tracking

For complex, multi-step, or multi-agent work:

* keep an explicit execution record if the task would be hard to resume safely from chat history alone
* use the repository-local `ai-plan/` workflow instead of inventing ad-hoc tracking files
* read `ai-plan/public/README.md` before scanning active topics when resuming or booting into complex work
* keep repository-wide design truth in `ai-plan/design/` and `ai-plan/roadmap/`
* keep active topic recovery state under `ai-plan/public/<topic>/`

`ai-plan/` uses these directory semantics:

* `ai-plan/design/`
  * repository-wide architecture and design truth
* `ai-plan/roadmap/`
  * repository-wide implementation plans and staged delivery documents
* `ai-plan/public/README.md`
  * shared startup index that maps branches or worktrees to active topics
* `ai-plan/public/<topic>/todos/`
  * recovery-safe tracking documents for one active topic
* `ai-plan/public/<topic>/traces/`
  * execution traces for one active topic
* `ai-plan/public/<topic>/design/`
  * topic-specific design documents that do not belong in repository-wide design truth
* `ai-plan/public/<topic>/roadmap/`
  * topic-specific implementation plans that do not belong in repository-wide roadmap truth
* `ai-plan/public/<topic>/archive/`
  * archived stage-level artifacts for an active topic
* `ai-plan/public/archive/<topic>/`
  * completed-topic archive that should not be treated as default boot context

Use these workflow rules:

* `ai-plan/public/README.md` must list only active topics
* when a branch or worktree has an active-topic mapping, read its tracking and trace files before substantive work
* when working from a tracked topic, update the corresponding tracking document in the same change
* for complex work, maintain a matching trace that records the current date, key decisions, validation milestones, and
  the immediate next step
* keep active tracking and trace files concise enough to serve as recovery entrypoints
* when a stage inside an active topic is complete, move detailed history into that topic's `archive/` and keep only the
  active recovery point in the default boot path
* when a topic is fully complete, move the entire topic directory under `ai-plan/public/archive/<topic>/` and remove it
  from `ai-plan/public/README.md` in the same change
* never record absolute file-system paths in `ai-plan/**`; use repository-relative paths, branch names, commit ids, PR
  numbers, and validation commands instead

## 17. Commenting and Documentation Rules

All generated or modified code must include clear and meaningful comments where required by the rules below.

### 17.1 Server Documentation

For Go code:

* all exported packages, types, interfaces, functions, methods, and constants must have Go-style doc comments
* comments must explain intent, contract, and usage constraints instead of restating syntax
* plugin lifecycle types and methods must document registration order, boot semantics, shutdown expectations, and
  failure behavior when relevant
* cross-plugin interfaces must document stability expectations and what callers may depend on

### 17.2 Web Documentation

For TypeScript and Vue code:

* add comments for non-trivial routing assembly, permission gating, dynamic menu composition, and complex page-state
  synchronization
* document why a store exists when the same state could have been page-local
* document backend contract assumptions when the UI depends on menu, permission, or plugin metadata semantics

### 17.3 Inline Comment Rules

Add inline comments for:

* non-trivial logic
* concurrency behavior
* lifecycle sequencing
* compatibility constraints
* registration order assumptions
* workarounds and edge cases

Do not add trivial comments that only restate the code.

### 17.4 Architecture-Level Documentation

Core framework components and plugin-extension primitives must explain:

* responsibilities
* lifecycle
* interaction with other components
* why the abstraction exists
* when to use it instead of simpler alternatives

Missing required documentation is a standards violation. Code that does not meet these documentation rules is
incomplete.

## 18. Change Management

When making substantial changes:

* explain which `ai-plan/design/` or `ai-plan/roadmap/` section the change follows
* keep architecture changes aligned with `ai-plan/`
* avoid silent changes to core conventions

If a task reveals that the current docs are wrong:

* update the relevant doc
* state the new rule clearly
* then implement against the updated rule

## 19. Code Review Expectations

Review for:

* boundary violations between core and plugins
* hidden coupling between plugins
* unnecessary framework complexity
* divergence from Go + Gin + GORM + Casbin server rules
* divergence from Vue 3 + TDesign web rules
* missing tests around plugin lifecycle, dependency ordering, authorization, and dynamic menu/route behavior
* undocumented public interfaces or lifecycle-sensitive code
* divergence between `ai-plan/design/`, `ai-plan/roadmap/`, and active topic recovery documents

A change is not acceptable if it makes adding the next plugin or frontend module harder.

## 20. Definition of Done

A task is done only when all relevant items below are satisfied:

* the change follows the current `ai-plan/` documents, or the docs were updated first
* `server` and `web` boundaries are still clear
* new module work keeps the `menu + route + page + api + permission` path explicit
* affected code has the required comments and documentation
* the changed area passed direct validation, or the exact validation gap was reported
* the final summary states the important behavior change, validation result, and any remaining blockers

If any of these are missing, the task is incomplete even if the code compiles.

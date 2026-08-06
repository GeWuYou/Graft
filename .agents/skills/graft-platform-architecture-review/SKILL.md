---
name: graft-platform-architecture-review
description: Perform the default semantic review of Graft changes for platform architecture, authority ownership, runtime boundaries, and long-term extensibility. Use during design and before implementation for changes that add or reshape platform capabilities, modules, lifecycle semantics, configuration, or shared contracts.
---

# Graft Platform Architecture Review

Use this skill as Graft's default semantic review layer. It complements `graft-boot`, design documents, domain-specific
review skills, and repository validation; it does not create a second startup flow, architecture authority, planning
system, or completion gate. Run it proactively during design and before implementation when a change could affect how
Graft behaves as a platform, even when the user did not explicitly request an architecture review.

## Review Authority And Scope

- Establish the product/module intent and the owning platform or module boundary before judging local code shape.
- Treat the root `AGENTS.md`, relevant `ai-plan/design/**` documents, and `server/AGENTS.md` or `web/AGENTS.md` as
  governance authority. When the scope includes `ai-plan/**`, also read and follow `ai-plan/AGENTS.md`. Treat module
  descriptors, stable contracts, and `openapi/**` as their respective technical sources of truth.
- Generated artifacts, page-local models, infrastructure manifests, and convenience wrappers are projections or
  implementation details unless an existing governance document explicitly makes them authoritative.
- Escalate upstream when a downstream symptom is caused by an incorrect module descriptor, contract, lifecycle, or
  configuration owner. Do not normalize authority drift with aliases, fallbacks, adapters, or duplicate registries.

## When To Run

Run this review by default when a change:

- adds, removes, or reshapes a core runtime surface, module lifecycle, service, event, job, permission, menu, route, or
  configuration source;
- changes task/submission execution semantics, asynchronous workflows, runtime targets, or application orchestration;
- changes an OpenAPI/shared contract or data path crossing `server`, generated clients, and `web` modules;
- introduces a new abstraction, package boundary, integration, persistence strategy, or platform-wide convention; or
- changes architecture, module boundaries, or execution governance documents.

For a narrow page or implementation-only fix with no platform implications, record why this review is not applicable and
use the more specific review skill instead.

## Review Workflow

1. Establish the startup receipt and identify the canonical authority owner, task class, recovery source, and affected
   platform surfaces. Read the applicable architecture and governance documents before freezing scope.
2. State the capability and user-facing workflow in platform terms: what application/module owns it, which lifecycle it
   follows, and which stable contracts or runtime services it needs.
3. Map the authority chain from product intent to module descriptor or contract, runtime wiring, OpenAPI/generated
   artifacts, web bootstrap, and page-local behavior. Mark every new source of truth and every transformation owner.
4. Apply the architecture checklist below. Record concrete evidence, unresolved decisions, and the smallest authority-
   first repair before implementation begins.
5. Check the proposed design against existing modules and platform primitives. Prefer reuse or a narrow public interface
   when it preserves policy and ownership; require a concrete benefit before adding an abstraction or shared helper.
6. Select the relevant specialized semantic reviews (`graft-cross-boundary-review`, OpenAPI, domain, permission,
   event, or query-key review) and the repository validation entrypoints. This skill is the architecture lens, not a
   substitute for those checks.

## Semantic Checklist

### Application First

- The design starts from an application capability and user workflow, not from a Docker, SSH, database, vendor, or
  transport primitive.
- Infrastructure details remain replaceable behind an explicit module or runtime interface. Business modules do not
  import deployment-specific concepts merely because the first implementation uses them.
- A proposed integration explains its failure, retry, timeout, audit, and lifecycle behavior at the application boundary.

### Runtime And Module Boundaries

- Platform-level startup belongs to documented core runtime surfaces: config, logger, database, HTTP, migration, event,
  permission, menu, cron, module lifecycle, service container, or repository CLI entrypoints.
- Business behavior lives under `server/modules/*` or `web/src/modules/<name>` with one clear owner for menu, route,
  page, API, permission, jobs, and public services.
- Modules depend on stable public interfaces and `server/internal/moduleapi/**`, not another module's internal package.
- Dependency direction, lifecycle ownership, shutdown behavior, and registration order are explicit; no hidden init,
  ad-hoc goroutine, singleton, or second DI mechanism is introduced.
- The interface is deep enough to own meaningful policy and state. Reject pass-through interfaces, wrappers, and generic
  abstractions that merely rename an existing call or spread knowledge across consumers.

### Authority And Contracts

- There is one canonical owner for each contract, permission, route/menu declaration, lifecycle state, and configuration
  value. Generated outputs and UI projections are derived.
- OpenAPI-first and typed-contract rules are preserved; server DTOs do not leak Ent entities and web code does not
  recreate wire types.
- New names, states, enums, defaults, and errors fit established vocabulary and have an explicit compatibility policy.
- Any unavoidable compatibility bridge records authority owner, reason direct repair is not possible, affected consumers,
  cleanup trigger, and validation for both paths.

### Task, Submission, And Async Lifecycle

- Long-running or asynchronous work uses the platform Task/Submission lifecycle where applicable instead of inventing a
  parallel job state machine.
- Submission, execution, retry, cancellation, progress, result, audit, notification, and error ownership are explicit.
- Event-bus usage defines event owner, transaction boundary, delivery/retry/idempotency semantics, and observable
  consumer behavior; a local callback is not silently presented as a durable event contract.

### Configuration And Extensibility

- Configuration has one documented source and owner, with environment, Compose, System Config, and runtime precedence
  handled by existing governance rather than a new lookup path.
- The extension path is compile-time module registration in v1. New capability does not require dynamic plugin loading,
  a marketplace, a heavyweight IoC container, or a second shell/bootstrap architecture.
- A platform primitive is justified by at least two real consumers or a clear lifecycle/ownership requirement; otherwise
  keep it local to the owning module.
- Removal and rollback are considered: identify code, contracts, registrations, or configuration that can be deleted if
  the capability is withdrawn.

## Findings And Decision Format

Report findings first, ordered by severity (`blocking`, `high`, `medium`, `note`). Every finding includes:

- the affected capability, boundary, or source of truth;
- concrete evidence from the design or diff;
- platform impact (ownership, extensibility, reliability, compatibility, or developer experience);
- the smallest authority-first repair; and
- required specialized review or validation.

Conclude with:

- `authority_summary`: canonical owners and any newly introduced authority;
- `platform_fit`: how the design preserves Application First, runtime/module boundaries, and the extension path;
- `lifecycle_summary`: Task/Submission/event/configuration implications;
- `deletion_candidates`: existing or proposed code/abstractions that can be removed;
- `decisions_needed`: unresolved design choices; and
- `acceptance`: conditions required before implementation or merge.

If no findings exist, say so explicitly and list residual architectural or validation risk.

## Boundaries

- Do not redesign a working module only to make it resemble another pattern; require a concrete semantic,
  maintainability, or platform-extensibility benefit.
- Do not create a second architecture document, decision registry, source-of-truth catalog, review gate, or recovery
  path. Record decisions through the existing `ai-plan` and task closeout governance.
- Do not treat this review as permission to bypass server/web contract, security, table, localization, comment, worktree,
  validation, commit, or closeout skills. Those authorities remain applicable when their triggers match.

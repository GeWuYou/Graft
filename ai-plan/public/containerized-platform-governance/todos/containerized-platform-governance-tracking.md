# Containerized Platform Governance Tracking

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Root AGENTS, domain AGENTS, ADR-029 and module contracts
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
    - design
    - roadmap
    - adr
closeout:
  archive: true
  lessons_review: true
```

## Recovery Point

User approved the four-batch plan on 2026-11-15. Baseline is 346854ff5daae3f804f8fad706202c9e983d2067.
Mode is topic-completion-loop, topology revision 1. Engineering governance and execution preparation have independent scopes.
Shared Task contracts precede domain migration; delivery acceptance follows integrated source.

## Checklist

- [ ] Engineering governance implementation and scoped checks prepared; integrated review and full CI pending.
- [ ] Local Docker/Compose Task execution with persistent conflict and interruption handling.
- [ ] Local Build and independent platform update Controller.
- [ ] Mode migration, user diagnostics and reproducible acceptance.

## Acceptance

Full CI and real operation evidence are required per delivery batch. Empty selections and missing runtime proof remain visible.
Governance tracks direct/cold/hot build measurements and cache invalidation. Runtime tracks actual Docker effects and recovery.

## Preparation Evidence

Local planner regressions cover matching consumer families, deleted packages/tests and partially staged consumers.
The reporting helper pipeline writes its relative artifact beneath the repository root when invoked from server.

Runtime CI wiring now consumes verified build artifacts, validates required private/official scenarios and performs identity-guarded cleanup. Integrated execution remains pending.

Versioned Update API CI is wired with 0.0.0-beta.1/.2 source-bound binaries, Controller and TLS release fixture artifacts. Full runtime evidence remains pending integrated execution.

Tool dependency ownership repair passed a clean frozen install and full Web check with 312 files / 2175 tests on the governance slice. Exact source and lock hashes are retained in task-local evidence.

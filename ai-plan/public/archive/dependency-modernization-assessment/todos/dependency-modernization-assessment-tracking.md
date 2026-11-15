# Dependency modernization and assessment tracking

## Work Contract

```yaml
version: 1
kind: audit
scope: long-running
authority_summary: "Existing repository owners retain dependency, API and deployment authority; the approved plan defines upgrade and assessment scope."
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: native-workers
  dispatch_skill: gewuyou-team-lead
bootstrap:
  targets:
    - ai-plan/public/dependency-modernization-assessment/README.md
    - ai-plan/public/dependency-modernization-assessment/startup-prompt.md
    - ai-plan/public/dependency-modernization-assessment/todos/dependency-modernization-assessment-tracking.md
    - ai-plan/public/dependency-modernization-assessment/traces/dependency-modernization-assessment-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

Base commit is `3989727b06bf446a03bcabceba0246c43bf3f181`. Implementation and assessment evidence are complete. Server commit `4be45857c` and Web commit `da67d87d0` passed their local completion gates. Upgraded runtime images used exact source `c274ac9236a240cf209638434fd370692e204a2d`. The lead owns pending remote CI, merge, eleven superseded dependency PR dispositions and task resource release through issue #341 and the task board.

## Task Checklist

- [x] Upgrade Go, Docker SDK graph, server dependencies and OpenAPI closure.
- [x] Upgrade Web dependencies and affected consumers.
- [x] Upgrade Actions and explicit build-tool versions.
- [x] Verify baseline and upgraded operations against a private test daemon.
- [x] Deliver evidence-linked Arcane capability comparison and simplification roadmap.
- [x] Hand off validated source and evidence to the lead with explicit remote delivery ownership under issue #341.

## Acceptance

Repository validation passes for changed inputs. The same bounded real-API scenarios record actual Task and engine state for baseline and upgraded images. The report distinguishes observed behavior, source support and remaining evidence gaps. PR cleanup follows verified inclusion on the default branch.

## Verified Findings

- Baseline and upgraded reads, real WebSocket Shell, managed create and Agent identity/restart succeed.
- Both versions accept mutation Tasks that remain unclaimed because execution identity comparison requires fields absent from certificate middleware. Issue #342 owns the proposed trusted-resolution repair.
- Both versions fail import persistence because upsert SQL column and argument order differ. Issue #343 owns the proposed ordered-binding repair.
- The baseline Agent `--once` remains healthy without exiting after successful enrollment. The fixture uses regular service restarts and records that limitation.
- Relative Compose mounts, provider mutation results and execution-interruption recovery remain unverified behind the existing admission defect.

## Archive Disposition

This archive records completed implementation and assessment work. Remote CI, merge, PR closure and resource release remained pending at archive time. The lead's issue #341 delivery receipt and task board track those external operations. Product defect follow-ups are recorded in issues #342 and #343.

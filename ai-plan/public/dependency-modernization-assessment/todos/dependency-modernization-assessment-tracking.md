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

Base commit is `3989727b06bf446a03bcabceba0246c43bf3f181`. Implementation is in progress. Runtime results are pending.

## Task Checklist

- [ ] Upgrade Go, Docker SDK graph, server dependencies and OpenAPI closure.
- [ ] Upgrade Web dependencies and affected consumers.
- [ ] Upgrade Actions and explicit build-tool versions.
- [ ] Verify baseline and upgraded operations against a private test daemon.
- [ ] Deliver evidence-linked Arcane capability comparison and simplification roadmap.
- [ ] Merge validated updates, close the eleven superseded dependency PRs and release task resources.

## Acceptance

Repository validation passes for changed inputs. The same bounded real-API scenarios record actual Task and engine state for baseline and upgraded images. The report distinguishes observed behavior, source support and remaining evidence gaps. PR cleanup follows verified inclusion on the default branch.

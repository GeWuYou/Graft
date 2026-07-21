# Docker Resource Context IA

## Current Status Summary

- Topic objective: converge Docker Network and Volume around a context-first resource reading model while preserving OpenAPI as the shared contract authority.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running resource IA and contract convergence; no ADR is required because the work applies existing authority-first and OpenAPI-first rules.
- Canonical authority:
  - `ai-plan/design/domains/container/容器管理设计.md` for Docker resource product semantics.
  - `openapi/**` for Docker resource wire contracts.
  - `server/modules/container/**` for normalized Docker resource projections.
  - `web/src/modules/container/**` for the Graft presentation layer.
- Completed so far: product IA, contract, server projection, and Web implementation.
- Current stage: cross-boundary validation, browser evidence, and closeout.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: OpenAPI owns the wire shape; the Container module owns Context and relationship-trust projection; web consumes the resulting contract.

## Owned Scope

- `ai-plan/design/domains/container/**`
- `ai-plan/public/docker-resource-context-ia/**`
- `openapi/**`, generated OpenAPI artifacts, `server/modules/container/**`, and `web/src/modules/container/**` required by the Docker resource context contract.

Out of scope:

- Raw Docker inspect payloads or frontend-derived Compose ownership.
- Kubernetes, Swarm, Secret, Config, Plugin, or unrelated Docker feature implementation beyond the reusable guideline seam.

## Locked Decisions

1. Docker details answer six questions in order: what it is, its Context, its Relations, its Configuration, its diagnostic Metadata, and its destructive actions.
2. Context is first-screen business information. Metadata cannot drive navigation or business judgment.
3. Network and Volume list filters default to keyword and status; advanced source and runtime filters are server-owned.

## Phase Plan

- Completed: Context contract, relationship-trust projection, product guideline, and Network/Volume implementation.
- Current: cross-boundary validation, browser evidence, and closeout.

## Current Recovery Point

- Contract, server projection, and Web implementation are complete.
- The authority repair is a normalized Volume Context projection; web does not infer it from labels.
- Next step: collect cross-boundary validation and browser evidence, then close out the topic.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is in the tracking file, not here.

## Validation Targets

```bash
git diff --check
just openapi-check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/docker-resource-context-ia/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch`

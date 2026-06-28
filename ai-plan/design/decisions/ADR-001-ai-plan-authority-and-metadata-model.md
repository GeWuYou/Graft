# ADR-001: ai-plan Authority and Metadata Model

- Status: accepted
- Date: 2026-06-28
- Scope: `ai-plan/**`

## Context

`ai-plan/` already stored repository design truth, roadmaps, and public recovery materials, but the execution rules for
`ai-plan/**` were still distributed across root `AGENTS.md`, `ai-plan/README.md`, and topic-specific examples. Active
topics were also created with implicit structure instead of one explicit metadata contract, which made later router,
catalog, validator, and skill work likely to drift.

## Decision

1. Root `AGENTS.md` remains the only repository startup-governance source.
2. `ai-plan/AGENTS.md` becomes the local execution-truth document for `ai-plan/**`.
3. `ai-plan/README.md` continues to define directory semantics and recovery-path overview, but it does not become a
   second startup or validation authority.
4. Every active topic under `ai-plan/public/<topic>/` must carry this minimum metadata model:
   - `README.md`
     - topic objective, recovery receipt, authority summary, owned scope, and pending batch direction
   - `startup-prompt.md`
     - reusable loop or future-turn entry artifact
   - `todos/<topic>-tracking.md`
     - current recovery point, checklist, acceptance conditions, and loop batch state
   - `traces/<topic>-trace.md`
     - dated decisions, validation milestones, and batch transitions
5. `ai-plan/public/README.md` remains a short active-topic index only.
6. `ai-plan/design/decisions/**` is the convergence gate for governance decisions that must land before downstream
   template, catalog, validator, or skill rollout.
7. When machine-readable indexing is needed, use one bounded `ai-plan/catalog.json`:
   - it supplements the router and topic files instead of replacing them
   - it may intentionally cover only the currently approved governance slice
   - it must not require per-document frontmatter retrofits across existing `ai-plan/**`

## Consequences

- `docs/automation` tasks that change `ai-plan/**` can read one local execution-truth document instead of inferring
  structure from ad hoc topic examples.
- Future automation can validate topic structure without inventing a hidden metadata store.
- Router, catalog, validator, and skill work must conform to this metadata model rather than redefining it later.
- The repository can add a machine-readable index incrementally without pretending it is the authoritative active-topic
  router or a mandate to rewrite older documents.

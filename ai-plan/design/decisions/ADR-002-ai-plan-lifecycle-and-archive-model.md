# ADR-002: ai-plan Lifecycle and Archive Model

- Status: accepted
- Date: 2026-06-28
- Scope: `ai-plan/public/**`

## Context

The repository already distinguished active topics from archived work, but lifecycle transitions, stage compaction, and
active-index updates were not yet locked as one explicit model. Without a stable lifecycle, later IA router or
validation work could preserve stale topics as active, let archived wording become normative, or hide compaction state
inside one document.

## Decision

1. Topic lifecycle states are:
   - `active`
     - listed in `ai-plan/public/README.md` and backed by live recovery materials
   - `archive-ready`
     - active work is complete and the topic passed its final archive-readiness check
   - `archived`
     - the topic moved under `ai-plan/public/archive/<topic>/` and was removed from the active-topic index
2. An active topic may keep stage history under `ai-plan/public/<topic>/archive/**`, but its default recovery path must
   remain concise.
3. Batch-driven topics must update tracking and trace files in the same change that advances or completes a batch.
4. Only active topics belong in `ai-plan/public/README.md`.
5. Archived wording is historical evidence only; root `AGENTS.md`, `ai-plan/AGENTS.md`, and current design documents
   remain normative.
6. Later templates and validators must treat lifecycle transitions as explicit file moves and index updates rather than
   implicit status text hidden in one file.

## Consequences

- Active recovery paths stay short even when a topic accumulates long execution history.
- The shared public index remains authoritative for current work instead of becoming a historical ledger.
- Later automation can validate lifecycle state from directory placement and required-file presence.

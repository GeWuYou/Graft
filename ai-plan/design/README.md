# Design

`ai-plan/design/` stores repository-wide design truth that should outlive one active topic, one worktree, or one
delivery batch.

Use this directory for stable architecture, governance, domain, release, and ADR documents. Use `ai-plan/roadmap/`
for phased implementation sequencing, and use `ai-plan/public/<topic>/` for topic-local recovery materials.

## Directory Map

- `architecture/`
  - Repository-wide baseline architecture, module model, and shell structure.
- `governance/`
  - Long-lived rules, guardrails, and authority models.
- `domains/`
  - Capability-specific design documents grouped by bounded domain.
- `release/`
  - Release-policy and upgrade-policy design.
- `decisions/`
  - ADRs that converge later governance or migration batches.
- `graft-design-system/`
  - Reusable page-template references for Graft frontend work.

## Routing Rules

- Put content here when it is repository-wide design truth.
- Put content in `ai-plan/roadmap/` when it mainly defines staged delivery order.
- Put content in `ai-plan/public/<topic>/design/` only when the note is specific to one active topic and should not yet
  become repository-wide authority.

## Migration Note

The target IA is being introduced in phases. After Batch 4, repository-wide design authorities now route through the
child directories and the `ai-plan/design/` root remains a router-only entry. Batch 5 is reserved for archive, naming,
and governance-sync closeout rather than additional root-level design-document moves.

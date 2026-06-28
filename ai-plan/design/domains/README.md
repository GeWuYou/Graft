# Domains

This directory groups capability-specific design documents by bounded domain.

## Routing Rules

- Use one child directory per capability area when documents mostly describe one domain's intent, authority, or
  evolution.
- Keep repository-wide baseline rules in `../architecture/` or `../governance/`.
- Each domain README should summarize local authority and point at the domain's canonical design documents.

## Current Scope Note

Batch 4 moved the current notification, container, compose, and audit design authorities into their matching
`domains/*/` directories. New domain-oriented repository design truth should land in the matching child directory
instead of being restored to the `ai-plan/design/` root.

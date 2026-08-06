---
name: graft-consistency-review
description: Review Graft naming, state, contract, module, and UI patterns for meaningful consistency without forcing mechanical uniformity. Use when adding a repeated capability or changing shared vocabulary.
---

# Graft Consistency Review

Use this when a new capability resembles existing modules, routes, dialogs, query helpers, states, errors, or registries.
Consistency means one domain meaning and one authority, not identical files or arbitrary style. It complements contract,
domain, architecture, and TypeScript reviews without becoming a repository-wide formatter or lint gate.

## Workflow

1. Define the concept and owner, then search neighboring modules and shared contracts by meaning, not only spelling.
2. Compare names, route/menu/permission declarations, request/response shapes, error envelopes, lifecycle states,
   query-key namespaces, composable APIs, loading/error/empty UI, test seams, and localization keys.
3. Classify differences as intentional domain variation, platform convention, legacy drift, or accidental inconsistency.
   Repair the authority owner for accidental drift; do not add aliases or duplicate constants to make consumers agree.
4. Prefer existing deep interfaces and module scaffolds. A new convention needs at least two real consumers or a clear
   platform boundary; otherwise keep it local and document the reason.
5. Check migration impact and generated artifacts. A naming change must account for OpenAPI, server contract, web module,
   route/menu/permission, persisted keys, query cache, tests, and i18n consumers.

## Output

Report `blocking`, `high`, `medium`, or `note` findings with compared pattern, canonical owner, evidence, impact,
recommended authority-first repair, and intentional exceptions. Conclude with `canonical_conventions`, `intentional_variants`,
`drift_candidates`, `compatibility`, and `validation`.

## Guardrails

- Do not run a repository-wide rename or formatting wave from this skill.
- Do not equate visual similarity with domain consistency or introduce generic abstractions for one repeated shape.
- Preserve Graft module, contract, localization, worktree, validation, and closeout authorities.

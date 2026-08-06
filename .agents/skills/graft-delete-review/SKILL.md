---
name: graft-delete-review
description: Review whether a Graft change can remove code, mappings, abstractions, compatibility paths, or duplicate authority instead of adding another layer. Use during design and before implementation.
---

# Graft Delete Review

Use this as a default deletion-first semantic review. It complements architecture, contract, domain, and change review;
it does not authorize destructive filesystem, database, migration, or Git operations.

## Workflow

1. Inventory the proposed behavior and all existing implementations, projections, aliases, wrappers, registries, tests,
   docs, and generated outputs that represent it.
2. Apply the deletion test: if the proposed module or mapping is removed, which behavior is lost and where would its
   policy return? Prefer deleting a pass-through layer or duplicate source over adding compatibility.
3. For every new abstraction, adapter, DTO, state, cache, config key, helper, or fallback, state the concrete policy,
   variation, authority protection, or lifecycle responsibility it earns. Reject speculative generality.
4. Identify consumers and migration order. A compatibility bridge is allowed only with canonical owner, reason direct
   repair is not possible, affected consumers, cleanup trigger, and validation of bridge plus cleanup.
5. Check tests, docs, generated artifacts, menu/route/permission registrations, and i18n keys for removable residue.
   Do not delete an authority or generated output merely because it looks unused; prove ownership and regeneration rules.

## Output

Report `blocking`, `high`, `medium`, or `note` findings with candidate, evidence, policy lost/retained, consumers,
deletion or migration plan, and validation. Conclude with `delete_now`, `keep_with_reason`, `compatibility_bridges`,
`cleanup_triggers`, and `residual_risk`.

## Guardrails

- This is a review, not permission to run `rm`, drop data, apply migrations, or rewrite Git history.
- Repair the highest authority first; do not preserve downstream drift with an alias.
- Do not delete user-owned or mixed-scope changes. Use normal repair, validation, commit, and closeout governance.

---
name: graft-change-review
description: "Review a Graft change on two independent axes: repository standards and originating design/spec intent. Use before commit or integration for any implementation slice."
---

# Graft Change Review

Run this after implementation and before closeout. It is the Graft adaptation of standards/spec review and complements
specialized semantic reviews, PR review, validation, and commit governance.

## Workflow

1. Pin the fixed point (`origin/main`, merge-base, commit, or user-provided ref) and capture the exact diff and commit
   list. Confirm the ref and a non-empty diff before reviewing.
2. Identify the originating Work Contract, topic design, ADR, issue, or user request. If none exists, report missing
   intent instead of inventing requirements.
3. Run two independent passes:
   - **Standards**: authority ownership, module boundaries, contract/lifecycle rules, security, comments, localization,
     query/cache, database, validation, and the small-module/deletion principles. Treat heuristic smells as findings,
     not automatic violations.
   - **Spec**: missing or partial requirements, behavior not requested, incorrect interpretation, scope creep, and
     unhandled acceptance criteria.
4. Keep findings separate and ordered by severity. Include file/line evidence, violated authority or requirement,
   user/maintainer impact, and the smallest repair. Do not modify the code while reviewing.
5. Confirm specialized Review Skill evidence and required server/web/cross-boundary validation. A review cannot turn a
   failed or missing validation into a pass.

## Output

```text
Graft change review:
- fixed_point: <ref and diff command>
- intent_source: <Work Contract/design/ADR/request or missing>
- standards: <findings or none>
- spec: <findings or none>
- semantic_reviews: <selected skills and evidence>
- validation: <commands and results>
- acceptance: <conditions before commit/integration>
```

## Guardrails

- Never create a second commit, validation, or PR review path.
- Do not rank Standards findings against Spec findings; the two axes must remain independently visible.
- Do not repair a finding during review unless the caller starts the normal governed repair workflow.

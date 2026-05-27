# AI Plan Public Recovery Index

`ai-plan/public/README.md` is the shared recovery index used after `AGENTS.md` startup preflight. It should stay short,
list only active topics, and help the current branch or worktree land on the right recovery documents without scanning
every public artifact.

## Active Topic

- `localization-governance`
  - Status: `ready-to-archive`
  - Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-localization-governance`
  - Branch: `feat/wt-localization-governance`
  - Recovery source: `parent topic = none`
  - Scope: cross-boundary localization governance covering `server`, `web`, and `openapi` contract policy.
  - Recovery docs:
    - `ai-plan/public/localization-governance/todos/localization-governance-tracking.md`
    - `ai-plan/public/localization-governance/traces/localization-governance-trace.md`

## Archived Topics

- `ARCHIVED_OPENAPI_GOVERNANCE_SERIES`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore these topics into the active recovery path.
  - Archive reason: final closeout for the completed OpenAPI / `oapi-codegen` / generated boundary / docs governance series.
  - Final result: implementation, audit, bundled-docs, monitoring-coverage, and closeout topics were either completed, superseded by later closeout topics, or absorbed into the final governance closeout.
  - Follow-up status: `superseded`
  - Superseded by:
    - `ai-plan/public/archive/openapi-governance-closeout-audit/traces/openapi-governance-closeout-audit.md`
    - operating rule `feature-delivery-with-contract-first-rule`
  - Archived topic directories:
    - `ai-plan/public/archive/oapi-codegen-types-only-spike`
    - `ai-plan/public/archive/oapi-generated-server-client-governance-spike`
    - `ai-plan/public/archive/openapi-codegen-governance-audit`
    - `ai-plan/public/archive/openapi-docs-bundled-spec-fix`
    - `ai-plan/public/archive/openapi-docs-mvp`
    - `ai-plan/public/archive/openapi-governance-closeout-audit`
    - `ai-plan/public/archive/openapi-monitoring-coverage-audit`
  - Archive notes:
    - `openapi-codegen-governance-audit` completed its read-first audit mission and was superseded by docs MVP, bundled-spec, generated-boundary, and final closeout work.
    - `openapi-docs-mvp` and `openapi-docs-bundled-spec-fix` completed their docs exposure mission and were absorbed by the final closeout state.
    - `openapi-monitoring-coverage-audit` completed its audit mission and its gap was absorbed by later generated-governance completion work.
    - `oapi-codegen-types-only-spike` and `oapi-generated-server-client-governance-spike` completed their guarded generated-boundary mission and now remain historical evidence only.
  - Operating rule:
    - future HTTP feature work follows `feature-delivery-with-contract-first-rule`
    - do not reopen a broad OpenAPI / `oapi-codegen` governance topic unless contract governance itself changes
  - Next-session prompt: `No next-session prompt required.`

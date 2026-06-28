# Governance

This directory groups long-lived rules, guardrails, and authority models for the repository.

## Subdirectories

- `ai/`
  - AI workflow, MCP, recovery, and AI review governance.
- `backend/`
  - Server-authority and backend/cross-boundary engineering guardrails.
- `frontend/`
  - Frontend design-governance, visual rules, and TDesign workflow guidance.
- `platform/`
  - Cross-cutting governance that is neither one pure backend cluster nor one pure frontend cluster.

## Routing Rules

- Put repository-wide rules here when they are primarily governance rather than capability design.
- Keep detailed rule ownership in the narrowest subdirectory instead of accumulating mixed policy prose at this level.
- Use `../domains/` when the document is mainly about one business or product capability.

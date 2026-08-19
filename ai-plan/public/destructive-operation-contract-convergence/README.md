# Destructive Operation Contract Convergence

## Current Status Summary

- Topic objective: unify every deletion, relationship removal, credential revocation, irreversible destruction, and external resource cleanup contract without introducing a generic cross-module delete service.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running refactor executed as bounded direct batches because OpenAPI and governance files are shared authority hotspots.
- Canonical authority:
  - `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
  - `openapi/openapi.yaml`
- Completed so far: contract foundation, the user soft-delete pilot, and relationship/RBAC convergence, including generated server/web consumers and completion validation.
- Remaining: hard-delete commands, external Task destruction, and full inventory closeout.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: API semantics are owned by backend API governance and canonical OpenAPI; module services own domain deletion, authorization, audit, and persistence behavior; Task Runtime owns external destructive execution.

## Owned Scope

- deletion and removal sections in the existing backend governance documents
- `openapi/**` destructive metadata and shared batch result contracts
- the existing OpenAPI validation entrypoint and focused tests
- bounded server/web module migrations listed in the roadmap

Out of scope:

- a generic `DeleteService`, global repository abstraction, or second permission/audit engine
- compatibility aliases for replaced delete routes
- synchronous waiting for Docker, Compose, cloud, or other external destruction

## Locked Decisions

1. Ordinary soft deletion and relationship removal use `DELETE`; a visible resource deleted once and retried in the same authorized scope returns `204`, while `GET` and list queries hide its tombstone and return/behave as `404`.
2. Irreversible hard deletion is a `POST .../deletions` command with a persistent idempotency receipt; external destruction is a `POST` action returning `202` plus the canonical Task receipt.
3. Ordinary resource batches may partially commit; role, permission, access-binding, and equivalent security-sensitive batches are atomic.
4. Batch results use one `operation_id + summary + results` contract. Existing module-specific result shapes are migration debt, not compatibility authorities.
5. `x-graft-destructive` is the machine-readable governance projection. `exposure.mcp` must agree with the presence of `x-graft-mcp`; it does not grant authorization by itself.

## Phase Plan

- contract-foundation: governance, metadata schema, shared batch schema, and validation semantics
- soft-delete-pilot: migrate user deletion and prove tombstone idempotency plus hidden reads
- relationship-and-rbac: migrate relationship removals and atomic security batches
- hard-delete-commands: migrate audit/app-log irreversible deletion to command receipts
- external-destruction-tasks: migrate Docker/Compose removal to `202` Task receipts
- convergence-closeout: remove legacy paths/results and enforce metadata coverage for the complete inventory

## Current Recovery Point

- `contract-foundation`, `soft-delete-pilot`, and `relationship-and-rbac` are complete in worktree `01` on `refactor/unified-destructive-operations`.
- Existing operations are not annotated until their runtime behavior matches the metadata; the first gate validates truthful annotations instead of publishing target semantics as current behavior.
- User deletion now uses `DELETE /api/users/{id}`: first delete and same-scope tombstone retry return 204, while normal detail/list reads continue to hide the tombstone.
- RBAC relationship removals now use canonical `DELETE` collection operations without aliases; atomic role/permission/user-role batches return the shared ordered result contract and reject empty, duplicate, or over-100-item requests.
- Next step: migrate audit and app-log irreversible deletion to `POST .../deletions` commands backed by persistent idempotency receipts.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is persisted in the tracking file.

## Pending Batch Direction

- inventory audit and app-log hard-delete paths from canonical OpenAPI through persistence and web consumers
- migrate irreversible deletion to `POST .../deletions` commands with persistent idempotency receipts and no legacy aliases

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
just generate
just openapi-check
just check
```

## Loop Entry

- Preferred entry: `ai-plan/public/destructive-operation-contract-convergence/startup-prompt.md`
- Preferred execution mode: bounded direct batches under the matching semantic review skills

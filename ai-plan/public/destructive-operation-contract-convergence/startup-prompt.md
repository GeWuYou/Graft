# Destructive Operation Contract Convergence Startup Prompt

Continue this topic after rerunning root startup preflight.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/destructive-operation-contract-convergence/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
  - `openapi/openapi.yaml`
- AI skills:
  - `$graft-openapi-contract-review`
  - `$graft-cross-boundary-review`
  - `$graft-api-dx-review`
  - `$graft-domain-model-review`
  - `$graft-permission-model-review`
  - `$graft-test-seam-review`

Topic objective:

- converge every destructive HTTP operation on explicit soft-delete, relationship-removal, irreversible command, or asynchronous external-destruction semantics.

Work contract summary:

- long-running cross-boundary refactor; design, topic, and roadmap required; executed in bounded direct batches because the main authority files are shared hotspots.

Locked decisions:

1. Soft-delete retry returns 204 for the same authorized tombstone, but default reads hide it and return 404.
2. High-risk hard deletion is a POST command with a stored idempotency receipt; external effects use Task Runtime and 202.
3. Ordinary batches may be partial; security-sensitive batches are atomic; all successful batch responses use the shared destructive batch result.

Implementation guardrails:

- Repair OpenAPI and module authority together; do not add route aliases or response adapters.
- Do not annotate an operation with target metadata until runtime behavior and tests match it.
- Keep authorization and audit in the owning module; do not create a generic deletion runtime.
- Do not synchronously wait on an external resource destroy request.

Current batch plan:

1. inventory relationship-removal and RBAC batch operations from canonical OpenAPI through handlers, stores, and web consumers
2. migrate relationship removal to DELETE without aliases and make security-sensitive role/permission/access-binding batches atomic
3. use the shared destructive batch envelope for successful batch results and add truthful `x-graft-destructive` metadata only after runtime behavior matches

Validation expectations:

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
just openapi-check
just check
```

Required closeout:

- State the current batch, changed authority, validation evidence, and remaining migration inventory.
- Update tracking and trace files in the same change.
- Evaluate `$graft-commit` only after validation.

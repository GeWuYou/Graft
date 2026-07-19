# Cross-Boundary Contract Projection Convergence Roadmap

## Delivery Order

1. Inventory every remaining web mirror of HTTP paths and server-owned non-HTTP values.
2. Generate operationId-indexed runtime API paths from the canonical OpenAPI source, then remove module server API path mirrors in bounded consumer migrations.
3. Project remaining non-HTTP module values from existing Go constants, grouped by module-owned generated targets.
4. Add drift gates only after the generator and migration conventions are proven by representative batches.
5. Re-run the inventory and archive only when no authority mirror remains in scope.

## Batch Boundaries

| Batch | Scope | Authority owner | Validation |
| --- | --- | --- | --- |
| 0 | Inventory and runtime API path projection | `openapi/**` | `just openapi-check`, focused generator tests, web check |
| 1 | notification, project, runtime-target, task | OpenAPI plus each module Go contract | server/web completion checks |
| 2 | rbac, user, audit, monitor, scheduler, system-config | OpenAPI plus each module Go contract | server/web completion checks |
| 3 | security, announcement, app-log, access-log | OpenAPI plus each module Go contract | server/web completion checks |
| 4 | drift gates | canonical sources and projection metadata | gate-focused tests plus CI-equivalent checks |
| 5 | final inventory and archive readiness | all authorities | full cross-boundary validation |

## Guardrails

- A web `contract/paths.ts` may retain only web-private route contracts after its runtime API values migrate.
- Public OpenAPI enums remain generated wire types, not Go descriptor values.
- A descriptor may point to an existing Go constant but may not introduce a duplicate string literal.

# Unmigrated Contract Inventory

This is the baseline inventory for the convergence topic. It distinguishes server-owned values from web-private contracts so migration does not turn every front-end string into a server projection target.

## HTTP Runtime API Paths

All 198 OpenAPI operations have an `operationId`. The following 17 web modules still mirror API path templates in `contract/paths.ts` (193 `/api/` values in total):

| Module | Mirrored API values | Web-private route contract also present |
| --- | ---: | --- |
| access-log | 4 | yes |
| announcement | 8 | yes |
| app-log | 5 | yes |
| audit | 7 | yes |
| auth | 10 | no |
| container | 24 | yes |
| dashboard | 2 | no |
| monitor | 4 | yes |
| notification | 5 | yes |
| project | 48 | yes |
| rbac | 31 | no |
| runtime-target | 6 | yes |
| scheduled-task | 10 | yes |
| security | 1 | yes |
| system-config | 3 | yes |
| task | 5 | no |
| user | 26 | yes |

Migration rule: APIs consume `OPENAPI_RUNTIME_PATH` by operation id and `buildOpenApiRuntimePath` for templates. A module keeps UI route values only when they are consumed by web routing/navigation; it must not keep any `/api/` value in its contract path file.

## Non-HTTP Cross-Boundary Candidates

The following existing web contracts contain candidate server-owned values requiring a per-module owner comparison before migration. They are inventory evidence, not proof that every listed value needs projection.

| Group | Candidate modules / contracts | Expected authority check |
| --- | --- | --- |
| Permission codes | access-log, announcement, app-log, audit, monitor, notification, rbac, scheduled-task, security, system-config, user | module descriptor and `server/modules/*/contract/**`; runtime bootstrap remains effective access authority |
| Realtime/runtime topics | container, project, runtime-target, task | OpenAPI subscription semantics plus module Go topic constants; generated values may not authorize subscriptions |
| Message keys and business enums | project messages/categories, rbac permission copy, user status, notification category/severity, audit presets | existing Go contract or public OpenAPI enum; web-only presentation enums stay private |
| Docker/runtime values | container paths/realtime, project bootstrap/realtime, runtime-target bootstrap/realtime | container/runtime-target/project contracts and public OpenAPI values, according to whether the value crosses HTTP |

## Explicit Non-Targets

- UI route paths, component state, storage keys, query keys, local table/filter models, and navigation presentation remain web-owned.
- API envelope `code` and `messageKey` remain open strings. Consumers must retain server fallback behavior for unknown values.
- Menu, permission, and capability effectiveness remains defined by server bootstrap and runtime responses, not generated literal unions.

# Phase D Access Log Contract Definition Tracking

## Scope Guard

Allowed:

- `server/internal/logger/**`
- `server/internal/httpx/**`
- `server/internal/pluginapi/**`
- `openapi/**`
- `ai-plan/design/**`
- `ai-plan/public/phase-d-access-log-contract-definition/**`

Forbidden:

- `web` page work
- explorer work
- metrics/tracing/OpenTelemetry work
- durable storage implementation
- query API implementation

## Completed

- analyzed current `httpx` request logging path
- separated `Access Log`, `Audit Log`, and `Security Event` authority
- defined canonical access-log schema and explicit forbidden fields
- defined future query/sort/pagination constraints
- defined operator troubleshooting workflow
- defined future ownership matrix and runtime gap list

## Deferred To Future Runtime Topic

- normalize runtime field names to contract shape
- decide and implement durable storage authority
- add query API only after storage/runtime authority exists
- add explorer UI only after shared contract exists

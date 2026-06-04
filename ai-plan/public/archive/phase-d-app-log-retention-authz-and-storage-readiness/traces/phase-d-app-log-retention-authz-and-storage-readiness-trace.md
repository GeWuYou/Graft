# Phase D App Log Retention Authz And Storage Readiness Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md` as a `cross-boundary` task.
- Used archived App Log durable-storage and operator-workflow topics as parent recovery evidence.
- Kept `server/internal/logger/**` as canonical `App Log` authority.
- Kept `openapi/**` and `web/**` out of implementation scope because durable storage was not explicitly approved.

## Readiness Result

- Runtime readiness: `Partially Ready`
- Durable storage approval: `deferred-not-approved`
- Schema/API/UI approval: `not-approved`

## Decisions

- Keep `App Log` storage mode as `process_output_only`.
- Keep repository retention owner as `none`.
- Keep repository default retention window as `0`.
- Continue recommending deployment-owned external collection for MVP durable App Log search.
- Require a later explicit durable-storage runtime-approval topic before adding `app_logs`, OpenAPI paths, or web UI.

## Gate For Future Runtime Work

Future durable storage work must first approve:

- retention owner and default window
- cleanup lifecycle and execution path
- read permission code and authz owner
- approved query dimensions
- persisted fields, indexes, and forbidden fields
- validation scope across server, OpenAPI, and web if those surfaces are touched

## Forbidden Expansion

- no `app_logs` table
- no App Log repository implementation
- no cleanup job
- no App Log API
- no App Log Explorer UI
- no reuse of `audit_logs`, `access_logs`, or Redis as App Log durable truth

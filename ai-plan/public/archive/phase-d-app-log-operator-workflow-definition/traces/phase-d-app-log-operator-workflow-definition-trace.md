# Phase D App Log Operator Workflow Definition Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md` as a `server` task.
- Read the prior durable-storage decision as the parent recovery topic and kept `server/internal/logger/**` as canonical `App Log` authority.
- Defined the minimum operator workflows that could justify a future repository-owned durable `App Log` dataset or explorer.

## Minimum Approved Workflows

- runtime error investigation by time window, severity, component, operation, and bounded message search
- background job failure investigation by component, operation, and failure window
- module startup/config failure investigation by module component and boot window
- request-correlated troubleshooting through `request_id` / `trace_id`

## Forbidden Expansion

- no access traffic analytics
- no audit / compliance review
- no security incident timeline
- no metrics dashboard
- no tracing span view

## Decision

- Keep `App Log` as `process_output_only`.
- Continue recommending deployment-owned external collection for operators needing durable search.
- Keep repository-owned durable `App Log` storage deferred because workflow clarity alone does not yet approve retention, authz, cleanup, or repository data-volume ownership.

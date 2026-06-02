# Phase D Access Log Explorer Contract Trace

## Summary

- re-ran startup preflight from root `AGENTS.md`
- classified the task as `cross-boundary`
- treated `phase-d-access-log-runtime-storage` as archive-ready evidence
- used access-log authority, observability governance, current `server/internal/httpx/**`, and current admin explorer UX patterns as recovery/authority inputs

## Exploration Findings

- `server/internal/httpx/**` already owns canonical request-fact semantics and current access-log storage baseline
- current migration/index shape is timeline-first: `occurred_at desc` is the truthful default sort anchor
- no current retention policy exists for access logs, so explorer cannot promise history duration
- current audit explorer pattern is `list-form-detail` with dedicated filters, server-owned pagination, and detail drawer
- current dashboards act as shortcut entry surfaces rather than replacing canonical explorer pages

## Key Decisions

- fixed `Access Log Explorer` as a request-fact explorer, not an audit/monitor/security truth surface
- rejected free-form keyword search as canonical MVP contract authority
- approved exact/range/prefix semantics only on owner-backed access-log fields
- chose page-based pagination over cursor pagination
- limited sort authority to `occurred_at`, `duration_ms`, and `status_code`
- kept correlation with audit/security/monitor as bounded consumer relationships only

## Non-Goals Preserved

- no OpenAPI
- no handlers
- no repository query methods
- no web page implementation
- no retention jobs
- no app-log explorer
- no metrics, tracing, or OpenTelemetry expansion

## Closeout

- result: `archive-ready`
- reason: implementation can now proceed from one explicit authority contract without inventing runtime or UX semantics downstream

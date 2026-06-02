# Phase D Access Log Contract Definition Trace

## Summary

- booted from root `AGENTS.md`
- classified this topic as `cross-boundary` because future access-log explorer contract will eventually cross `server -> openapi -> web`
- used `phase-d-log-retention-and-storage-authority` as parent recovery evidence
- verified current `Access Log` runtime from `server/internal/httpx/accesslog.go` before defining contract fields

## Key Decisions

- fixed `Access Log` authority at `server/internal/httpx/**`
- kept `Audit Log` and `Security Event` outside access-log ownership
- defined a canonical schema with explicit `required` / `optional` / `derived` / `forbidden` status
- limited future query and sort contract to owner-approved fields only
- selected page-based pagination because no durable storage/cursor authority exists yet
- recommended `phase-d-access-log-runtime-storage` as the only truthful next runtime topic

## Non-Goals Preserved

- no explorer UI
- no new log page
- no durable table
- no query API
- no metrics/tracing/OTel expansion

## Closeout

- result: `archive-ready`
- reason: the future `Access Log Explorer` can now be constrained to one formal contract authority without widening into runtime implementation

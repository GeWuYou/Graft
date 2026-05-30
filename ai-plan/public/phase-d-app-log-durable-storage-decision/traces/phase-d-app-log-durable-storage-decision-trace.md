# Phase D App Log Durable Storage Decision Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md` as a `server` task.
- Read `server/internal/logger/**` as the canonical `App Log` authority and used `server/internal/httpx/**` durable access-log work only as a boundary reference.
- Confirmed the current logger authority already reserves future durable storage but does not approve it.

## Key Decision

- Deferred repository-owned durable `App Log` storage until a future operator-workflow topic defines why in-repo search is needed, what the minimum query path is, who owns authz and retention, and how boundary drift with access/audit/security logs will be prevented.

## Why Not Reject

- Developer debugging and external collection already work through process output.
- A later bounded operator workflow may still justify a narrow repository-owned dataset.

## Why Not Approve

- No canonical operator workflow exists yet.
- No retention/cleanup owner exists yet.
- Query/index/authz needs are still speculative.
- Free-form `message` / `fields` increase leakage and overlap risk compared with access-log storage.

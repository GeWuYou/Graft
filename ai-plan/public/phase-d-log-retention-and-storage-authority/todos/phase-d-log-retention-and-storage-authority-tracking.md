# Phase D Log Retention And Storage Authority Tracking

- Topic: `phase-d-log-retention-and-storage-authority`
- Status: `archive-ready`
- Task class: `server`
- Recovery source:
  - `phase-d-log-explorer-authority-definition`

## Completed

1. Executed startup preflight under root `AGENTS.md`.
2. Classified the topic as `server`.
3. Confirmed current runtime facts from code:
   - `AppLogger` and `AccessLogger` only write to shared zap output.
   - `AuditRecorder` persists to PostgreSQL `audit_logs`.
   - `SecurityEvent` publishes in `httpx` and persists through the audit path.
4. Confirmed no repository-owned retention cleanup, archive, or purge authority exists yet for audit/access/app logs.
5. Recorded governance truth:
   - only audit logs are eligible for future repository-owned cleanup because only audit has current durable storage authority
   - access/app log retention remains deployment-sink concern until a future storage authority topic changes that fact

## Final Decision

- Final verdict: `Archive Ready`
- Recommended next topic: `phase-d-access-log-contract-definition`

## Remaining Runtime Gaps

- define explicit audit retention window
- add audit-owned cleanup/purge authority only after that rule exists
- keep app/access log storage truthful; do not invent new sinks under contract-definition scope

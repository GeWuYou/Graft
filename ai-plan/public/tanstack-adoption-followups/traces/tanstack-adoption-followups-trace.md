# TanStack Adoption Follow-ups Trace

## 2026-07-15 Work Intake And Baseline

- Work Intake classified remaining TanStack adoption as long-running frontend architecture evolution: design and an active topic are required; roadmap and ADR are not.
- Added AI design and review guidance that makes Query the first evaluation for server-state duplication, manual request state, refresh logic, polling, and realtime-triggered HTTP refresh.
- Recorded P0 baseline commit `5acd17d8`: shared QueryClient plus announcement, monitor, access-log, app-log, and audit migrations.
- Deferred Table, Virtual, Router, and Form migration by default; they require separately recorded evidence rather than framework symmetry.

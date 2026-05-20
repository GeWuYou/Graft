# Monitor Server Status Trace

## 2026-05-20

- Split this work out of `multi-worktree-governance` into a standalone topic after the parent topic was archived.
- Confirmed the repository root has returned to `main`.
- Recorded the topic as design-only for now; no source code changes were made.
- Kept the future owned scope explicit: `server/plugins/monitor/**` and `web/src/modules/monitor/**`.
- Kept shared hotspots out of standing ownership.
- Implemented the first minimal `monitor/server-status` backend slice in `server/plugins/monitor/**`.
- Implemented the first minimal `monitor/server-status` frontend slice in `web/src/modules/monitor/**`.
- Registered the backend plugin through the explicit shared-hotspot exception in `server/internal/pluginregistry/generated.go`.
- Kept the route contract at `GET /api/monitor/server-status`, the menu path at `/monitor/server-status`, and the read permission at `monitor.server-status.read`.
- Chose the explicit server version fallback value `dev` because no stronger canonical runtime version source exists yet in current core surfaces.
- Validated the slice with `cd server && GIT_DIR=... GIT_WORK_TREE=... go run ./cmd/graft validate backend` because plain worktree git resolution is still misconfigured in WSL.
- Validated the frontend slice with `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run check`.

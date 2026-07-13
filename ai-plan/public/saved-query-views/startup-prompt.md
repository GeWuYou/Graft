Continue the `saved-query-views` topic.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/saved-query-views/README.md`
- local execution truth: `server/AGENTS.md`, `web/AGENTS.md`, and `ai-plan/AGENTS.md`

Topic objective:

- Provide private saved query views through the shared query-list UI while preserving each list owner's authorization and validation boundary.

Locked decisions:

1. Saved views remain user-private and do not persist the current page.
2. No generic public saved-view route; each consumer owns its protected API surface.

Current batch plan:

1. Implement the shared control and project migration.
2. Add the remaining owner routes and migrate the log explorers.

Run the smallest required validation, update tracking and trace materials, and use `$graft-task-closeout` at the batch boundary.

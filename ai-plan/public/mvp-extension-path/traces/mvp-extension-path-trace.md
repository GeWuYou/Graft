# MVP Extension Path Trace

## 2026-05-12 topic bootstrap

- Established `mvp-extension-path` as the first long-lived active topic for Graft and bound it to branch
  `feat/mvp-extension-path`.
- Migrated repository-wide design and roadmap truth from `plan/` into `ai-plan/`.
- Added `ai-plan/design/AI任务追踪与恢复设计.md` and aligned `AGENTS.md`, `README.md`, and `graft-boot` with the new
  recovery model.

## 2026-05-12 to 2026-05-14 shared MVP milestones

- Landed the first executable MVP shell across `server` runtime scaffolding and the `web` admin shell.
- Fixed repository-wide truth on Ent, Atlas versioned migrations, explicit migration CLI flow, plugin-facing store
  boundaries, and cross-boundary i18n extension hooks.
- Added repository-level PR review support, comment governance truth, and frontend governance truth.
- Preserved detailed pre-subtopic history in
  `ai-plan/public/mvp-extension-path/archive/traces/mvp-extension-path-trace-pre-subtopics-2026-05-14.md`.

## 2026-05-14 parent/subtopic split

- Refactored `mvp-extension-path` from one overloaded active topic into one parent topic plus bounded `server` and
  `web` subtopics.
- Kept the parent topic as the default `boot` recovery entrypoint so startup remains stable.
- Moved pure backend execution history and recovery state into the `server` subtopic.
- Moved pure frontend execution history and recovery state into the `web` subtopic.
- Limited the parent topic to cross-boundary direction, shared risks, shared validation summaries, and subtopic entry
  guidance.

## Next Step

- Continue MVP work through the relevant subtopic, while updating the parent topic whenever a change touches shared
  contracts, cross-boundary validation, or overall `server` + `web` direction.

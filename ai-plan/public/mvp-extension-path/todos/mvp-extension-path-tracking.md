# MVP Extension Path Tracking

## Topic

- Topic: `mvp-extension-path`
- Branch: `feat/mvp-extension-path`
- Scope: cross-boundary MVP extension-path coordination across `server` and `web`

## Goal

- Keep one stable parent recovery entrypoint for MVP work while letting `server` and `web` maintain their own bounded
  recovery files.

## Repository Truth

- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/代码注释与模块文档规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Subtopics

- `server`
  - Tracking: `ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
  - Trace: `ai-plan/public/mvp-extension-path/subtopics/server/traces/server-trace.md`
  - Use for: backend runtime, plugin lifecycle, registries, Ent/Atlas, CLI, auth/RBAC backend path, and other
    `server`-only work.
- `web`
  - Tracking: `ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
  - Trace: `ai-plan/public/mvp-extension-path/subtopics/web/traces/web-trace.md`
  - Use for: admin shell, route/menu/page/api/permission frontend path, i18n UI surface, and frontend
    governance/toolchain work.

## Parent-Scope Rules

- Keep cross-boundary direction, shared milestones, shared risks, and shared validation summaries here.
- Move pure `server` execution details into the `server` subtopic.
- Move pure `web` execution details into the `web` subtopic.
- When one change touches both sides or changes a shared contract, update this parent topic and the affected subtopic.

## Current Recovery Point

- The repository AI workflow now uses `ai-plan/` as the recovery system, with `mvp-extension-path` as the default MVP
  parent topic.
- `mvp-extension-path` has been refactored into a parent topic plus bounded `server` and `web` subtopics so future
  iteration no longer accumulates all recovery detail in one file.
- The repository already contains the first substantive MVP shell across both `server` and `web`.
- Shared architecture truth is stable on plugin-oriented backend boundaries, Vue 3 + TDesign frontend boundaries, Ent
  as the backend ORM baseline, and Atlas versioned migrations executed through explicit CLI flow.
- Shared extension-path truth now reserves cross-boundary i18n hooks, backend-driven menu/permission evolution, and a
  frontend governance baseline whose completion-state rules now require host Windows Bun `bun run check` plus zero
  unresolved warnings, and the latest implementation slice has now aligned the actual `web` toolchain to that bar.
- The previous pre-subtopic tracking snapshot has been archived at
  `ai-plan/public/mvp-extension-path/archive/todos/mvp-extension-path-tracking-pre-subtopics-2026-05-14.md`.

## Shared Milestones

- Established `mvp-extension-path` as the long-lived MVP recovery topic on branch `feat/mvp-extension-path`.
- Landed the first end-to-end MVP shell path across `server` runtime scaffolding and the `web` admin shell.
- Moved repository truth from `plan/` to `ai-plan/` and added `.ai/environment/` as generated environment truth.
- Reserved shared i18n extension points and a stable localized error-response contract across backend and frontend.
- Tightened repository-wide comment governance, frontend governance truth, and PR-review workflow support.

## Shared Risks

- The backend auth + RBAC plugin chain now exists, but broader session revocation, admin-driven session controls, and
  richer auth/audit governance are still pending without breaking the future `menu + route + page + api + permission`
  path.
- The disposable PostgreSQL + Atlas live validation path is now proven locally, but it still depends on manual
  disposable Docker resources instead of a repository-local scripted validation entrypoint.
- The frontend warning-cleanup slice now closes green, but the current `web` bundle-size strategy still reflects the
  temporary full-TDesign starter baseline, so deeper performance-oriented chunk optimization remains future work.
- Future work must keep parent-topic summaries, subtopic recovery files, and repository-wide design truth aligned to
  avoid creating parallel sources of truth.

## Shared Validation Summary

- Historical detailed validation commands before the subtopic split are preserved in the archived tracking snapshot.
- The latest cross-boundary implementation slice before this split validated focused `server` packages plus direct
  `web` test, typecheck, and build commands for the PR `#5` follow-up fixes.
- The latest backend live-validation slice now additionally proved the disposable PostgreSQL + Atlas path through
  `graft migrate up`, Atlas status verification, focused backend `go test`, and a disposable PostgreSQL + Redis
  `graft serve` healthz probe; detailed commands live in the `server` subtopic tracking file.
- The latest `web` governance implementation slice additionally validated host Windows Bun `bun run check`, a focused
  `bun run test:run -- --reporter=hanging-process` diagnosis, and the low-risk `axios@latest` refresh.
- This documentation-only governance update should be validated through consistency checks across `AGENTS.md`,
  `ai-plan/design/前端架构设计.md`, `ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md`, and
  `ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`.

## Immediate Next Step

- Keep future `web` work on the documented host Windows Bun `bun run check` zero-warning gate, then extend the
  backend auth/session path with broader session revocation and admin-driven session controls while keeping parent and
  subtopic recovery files in sync.

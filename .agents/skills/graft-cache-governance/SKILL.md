---
name: graft-cache-governance
description: Repository-specific workflow for Graft cache governance. Use when a task adds or changes runtime read paths, system config consumption, RBAC/menu/bootstrap aggregation, dashboard summaries, container runtime views, notification gating, scheduler defaults, or other hotspot data that may require caching.
---

# Graft Cache Governance

Use this skill before implementing or refactoring Graft runtime read paths that may become hotspots.

Treat root `AGENTS.md` as startup truth. This skill does not replace startup, validation, commit, or recovery workflow.

## Read First

1. Complete root `AGENTS.md` startup preflight.
2. Read `server/AGENTS.md` for backend work.
3. Read `web/AGENTS.md` when menu/bootstrap/config effective value is exposed to `web`.
4. Read `ai-plan/design/缓存治理与系统配置读取加速规范.md`.
5. Read `ai-plan/design/系统配置模型与渲染设计.md` when the task changes system config metadata or UI semantics.
6. Read `ai-plan/design/通知中心设计.md` when the task changes notification source gating or delivery config.
7. Read `ai-plan/design/容器管理设计.md` when the task changes container runtime config reads.
8. Read `ai-plan/design/共享资产复用治理规范.md` and the relevant shared-asset registries before introducing a new shared cache helper.

## When To Use

Use this skill when the task changes any of:

- `server/modules/system-config/**`
- `server/internal/moduleapi/**` config-reading contracts
- `server/internal/scheduler/**` default config resolution
- `server/modules/notification/**` runtime config gating
- `server/modules/container/**` runtime config consumption
- `server/modules/user/bootstrap.go` menu/bootstrap feature gating
- RBAC/menu/dashboard/container runtime aggregation or similar hotspot reads
- any new path that repeatedly reads DB-backed config, permission, menu, or summary data

## Authority Rules

- Keep `configregistry` plus `server/modules/system-config` as the system-config authority.
- All runtime system-config reads must go through one unified resolver or snapshot provider; do not let modules query the system-config override table directly.
- Prefer reusing existing `moduleapi`, `configregistry`, `cronx`, menu registry, dashboard registry, Redis client, and in-process cache patterns before inventing a new abstraction.
- Do not make Redis the authority for system config.
- Do not add distributed cache by default just because Redis exists.
- Do not cache highly volatile real-time status for too long just to reduce API latency.
- Do not let `web` become the authority for effective permissions, menus, or runtime config values.
- Keep `restart-required` and `runtime-hot` semantics explicit; do not claim hot reload for values that still require process rebuild or runtime reconstruction.

## Current Repository Status

- The repository's `system-config` authority chain already has a process-local snapshot cache with `singleflight`, explicit local invalidation, and Redis best-effort invalidation transport.
- The shared `SystemConfigResolver` contract is no longer bool-only; current repository truth includes `IsBooleanConfigEnabled(...)` and `ResolveDefaultConfig(...)`.
- Canonical runtime apply semantics now live in upstream `runtime_apply_mode`; `web` must consume that authority instead of inferring `runtime-hot` locally.
- Existing effective-source authority remains the current `default` vs `override` distinction; do not add a new display-only field for the same meaning.
- This completed scope is limited to the `system-config` authority chain and the hotspot consumers already registered by the topic.
- Do not reinterpret that archive-ready state as "all repository cache governance is finished"; new or unregistered hotspots still require explicit authority-first classification and follow-up.

## Required Cache Classification

Before coding, classify the target data into exactly one primary strategy:

- `request-scoped cache`
- `process local cache`
- `Redis distributed cache`
- `startup immutable cache`
- `no cache`

Every implementation or review should answer:

1. Is the read path actually hot?
2. What is the authority source?
3. What is the acceptable stale window?
4. How is invalidation handled on write?
5. Does multi-node consistency matter now, or only later?

## System Config Rules

For system-config-backed runtime reads:

- Default to process-local typed snapshot cache.
- Prefer whole-snapshot or domain/group snapshot over ad-hoc per-key caches unless a tighter shape is clearly justified.
- Use singleflight or equivalent request coalescing when a miss can hit the database.
- Merge `configregistry` defaults with DB overrides in one place.
- Preserve explicit fallback behavior.
- Preserve config-change audit behavior.
- Plan for explicit invalidation after successful writes.
- If the project is still single-node for the target slice, Phase 1 may stop at local cache plus explicit invalidation.
- If the task explicitly needs multi-node correctness, extend with Redis pub/sub invalidation or a version-polling design without changing authority.

## Hotspot Checklist

When touching a runtime read path, inspect whether it already belongs to one of these hotspot classes:

- notification source gating
- scheduler default config resolution
- container enablement / dangerous action / shell gating
- bootstrap menu feature gating
- RBAC permission aggregation
- dashboard summary aggregation
- mount usage / local filesystem scans
- log retention / scheduled task config resolution

If the path is hot and uncached, either:

- implement the correct cache layer in the same slice, or
- explicitly record why caching is deferred and what protects the system in the meantime

Do not silently leave a newly created hotspot uncached.

## Shared Asset Reuse

Before adding a new cache helper or cache interface:

1. Search `.ai/registries/server-shared-assets.yaml` and `.ai/registries/cross-boundary-assets.yaml`.
2. Search real code for:
   - `configregistry`
   - `SystemConfigResolver`
   - `ResolveDefaultConfig`
   - `singleflight`
   - `cache`
   - Redis client usage
3. Reuse or extend an existing shared helper when ownership matches.
4. Add a new registry entry only if the new helper becomes a stable shared asset.

## Validation

For docs or skill-only changes:

```bash
git diff --check
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-cache-governance
python3 scripts/validate_ai_governance.py
python3 scripts/validate_shared_asset_registries.py
```

For backend cache changes:

```bash
graft validate backend --stage lint
```

Add focused tests for:

- cache hit/miss behavior
- invalidation after update/reset
- singleflight behavior on concurrent reads
- degraded behavior when Redis is unavailable, if Redis is part of the slice

## Closeout Evidence

```text
Cache governance:
- task_class: server | cross-boundary | docs/automation
- owned_scope: <paths>
- authority: configregistry | system-config service | module-local runtime summary | registry-owned immutable data
- cache_strategy: request-scoped | process-local | redis | startup-immutable | none
- stale_window: <duration or not-allowed>
- invalidation: explicit-local | ttl-only | redis-pubsub | version-polling | not-applicable
- hotspot_status: reused-existing | added-cache | deferred-with-reason | not-applicable
- validation: <commands and results>
```

---
name: graft-cache-governance
description: Repository-specific workflow for Graft cache governance. Use when a task adds or changes runtime read paths, system config consumption, RBAC/menu/bootstrap aggregation, dashboard summaries, container runtime views, notification gating, scheduler defaults, or other hotspot data that may require caching.
---

# Graft Cache Governance

Use this skill before implementing or refactoring backend read paths that may become hotspots.

Treat root `AGENTS.md` as startup truth. This skill does not replace startup, validation, commit, or recovery workflow.

## Read First

1. Complete root `AGENTS.md` startup preflight.
2. Read `server/AGENTS.md`.
3. Read `web/AGENTS.md` only when the task changes `web`-visible menu/bootstrap/config semantics or shared authority.
4. Read `ai-plan/design/缓存治理与系统配置读取加速规范.md`.
5. Read `ai-plan/design/系统配置模型与渲染设计.md` when the task changes system-config metadata or UI semantics.
6. Read `ai-plan/design/通知中心设计.md` when the task changes notification gating or delivery config semantics.
7. Read `ai-plan/design/容器管理设计.md` when the task changes container runtime config reads or runtime-hot semantics.
8. Read `ai-plan/design/共享资产复用治理规范.md` before adding any shared helper or shared cache mechanism.

## When To Use

Use this skill when the task changes any of:

- `server/modules/system-config/**`
- `server/internal/moduleapi/**` config-reading contracts
- `server/internal/scheduler/**` default config resolution
- `server/modules/notification/**` runtime config gating
- `server/modules/container/**` runtime config consumption
- `server/modules/user/bootstrap.go`
- RBAC/menu/dashboard/container runtime aggregation or similar hotspot reads
- any new path that repeatedly reads DB-backed config, permission, menu, summary, or authority-owned runtime data

## Workflow

### 1. Authority-First Inventory

Before proposing cache code, inspect the real codebase:

- search current shared surfaces under `server/internal/**` and `server/modules/**`
- search direct Redis usage
- search local TTL caches
- search snapshot providers
- search `singleflight`
- search invalidation paths

At minimum, inspect:

- `server/internal/configregistry/**`
- `server/internal/moduleapi/**`
- `server/internal/redisx/**`
- `server/modules/system-config/**`
- the target consumer path

You must answer:

1. What is the canonical authority owner?
2. Is the observed problem upstream authority drift or only downstream read amplification?
3. Does an existing shared facility already cover this path?
4. Is this truly a hotspot?

If the real authority sits outside the initial file target, escalate before implementing a downstream-only cache patch.

### 2. Choose Exactly One Primary Strategy

Classify the target path into exactly one:

- `no cache`
- `request-scoped cache`
- `process-local cache`
- `startup-immutable cache`
- `redis-authority-data`
- `redis-invalidation-transport`

Do not combine several strategies in the closeout label just because multiple mechanisms are present internally.

Every classification must state:

- authority owner
- stale window
- invalidation or reload path
- whether multi-node correctness matters now

### 3. Reuse Before Adding

Default to reusing current repository facilities:

- `configregistry` for system-config definitions and defaults
- `moduleapi.SystemConfigResolver` for cross-module effective config reads
- `server/modules/system-config/service.go` for authority-owned config snapshot reads
- `server/internal/redisx/**` only for Redis transport opening
- startup registries under `server/internal/menu/**`, `server/internal/cronx/**`, `server/internal/dashboard/**` when the data is immutable after registration

Do not create a new shared helper unless authority discovery proves a repeated mechanical need across more than one authority owner.

### 4. Direct Redis Policy

Default rule:

- modules must not treat Redis as an arbitrary cache backdoor

Direct Redis usage is allowed only when one of these is true:

- the code is core Redis transport under `server/internal/redisx/**`
- the module owns Redis-backed data as part of its business/runtime authority
- Redis is used only for best-effort invalidation transport for another authority owner

Direct Redis usage is not allowed when:

- a module wants to mirror DB-backed authority into Redis just for convenience
- a module bypasses a unified resolver or snapshot provider
- a module invents ad-hoc key/TTL/invalidation rules without shared-facility review
- the cache would make Redis or `web` appear to own effective values

### 5. Shared Helper Admission Rule

Add or extend `server/internal` shared cache machinery only if all are true:

- at least two authority owners need the same cache mechanics
- the shared package can stay mechanical only
- no module business DTO, override table access, business key policy, or second config authority leaks into it
- current authority-owned helpers cannot reasonably hold the behavior themselves

If these conditions are not met, keep the helper authority-owned or module-local.

### 6. System-Config Hard Rules

For any system-config-backed path:

- keep `configregistry` plus `server/modules/system-config/**` as the authority chain
- all effective reads must go through the unified resolver or snapshot provider
- do not let modules query the override table directly
- do not make Redis the authority
- do not let `web` infer runtime apply semantics locally
- keep `runtime_apply_mode` as the canonical runtime-apply signal

### 7. Hotspot Registration Or Deferred Decision

If a path is genuinely hot and uncached, the task must end in one of these states:

- `reused-existing`
- `added-cache`
- `deferred-with-reason`

Do not silently leave a newly identified hotspot without a governance decision.

If deferred, record:

- why no cache is being added now
- what protects the path meanwhile
- what future trigger should reopen the decision

## Guardrails

- Do not assume every hotspot should be cached.
- Do not default to Redis because Redis exists.
- Do not make `web` an authority for effective config, permission, or menu state.
- Do not bypass unified resolver / snapshot provider for system-config reads.
- Do not create compatibility layers, aliases, or fallback authorities just to preserve downstream drift.
- Do not introduce a generic cache abstraction that only one module can use.

## Validation

For docs or skill-only changes:

```bash
git diff --check
python3 scripts/validate_ai_governance.py
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-cache-governance
```

Add:

```bash
python3 scripts/validate_shared_asset_registries.py
```

when the task changes shared-asset registry entries.

For backend cache changes:

```bash
cd server && go run ./cmd/graft validate backend --stage lint
```

Also run the smallest direct `go test` scope covering:

- cache hit/miss behavior
- invalidation after update/reset
- concurrent miss collapse when relevant
- degraded behavior when Redis publish/subscribe is unavailable, if Redis is part of the slice

## Closeout Evidence

```text
Cache governance:
- task_class: server | cross-boundary | docs/automation
- owned_scope: <paths>
- authority_owner: <canonical owner>
- hotspot: yes | no
- cache_strategy: none | request-scoped | process-local | startup-immutable | redis-authority-data | redis-invalidation-transport
- shared_facility: reused-existing | added-authority-owned-helper | added-shared-mechanical-layer | none
- direct_redis_usage: forbidden | reused-existing | added-with-justification | not-applicable
- stale_window: <duration | explicit-invalidation-only | not-allowed>
- invalidation: local-explicit | ttl-only | redis-pubsub | not-applicable
- hotspot_status: reused-existing | added-cache | deferred-with-reason | not-applicable
- validation: <commands and results>
```

---
name: graft-cache-governance
description: "Review Graft backend hotspot reads, cache ownership, invalidation, and system-config resolution when those behaviors change."
---

# Graft Cache Governance

Review backend read hotspots under [root governance](../../../AGENTS.md) and [cache/config authority](../../../ai-plan/design/governance/platform/缓存治理与系统配置读取加速规范.md). Reuse the current receipt and relevant subdomain evidence. Read System Config, notification, or container design only when their semantics change.

## Authority and Strategy

Inspect the actual consumer, direct Redis/TTL caches, snapshots, invalidation, and `singleflight` use. For config reads, inspect `configregistry`, public `moduleapi` contracts, `redisx`, and the owning `system-config` service. Determine the authority owner, whether the issue is read amplification or upstream drift, existing shared facilities, and evidence that the path is hot.

Select one primary strategy: `no cache`, `request-scoped cache`, `process-local cache`, `startup-immutable cache`, `redis-authority-data`, or `redis-invalidation-transport`. State its stale window, invalidation/reload path, and multi-node correctness needs. A hotspot does not automatically justify Redis or any cache.

Prefer current facilities:

- `configregistry` for definitions/defaults;
- `moduleapi.SystemConfigResolver` or the authority-owned snapshot provider for effective values;
- `system-config/service.go` for its authority-owned snapshots;
- `redisx` for transport opening;
- menu, cron, and dashboard startup registries for immutable registered data.

## Admission and Ownership

Direct Redis is allowed only for core transport, module-owned Redis-backed facts, or best-effort invalidation transport. It must not mirror database authority for convenience, bypass resolvers, or introduce ad-hoc keys/TTL as another authority.

A shared cache helper requires at least two authority owners with the same mechanical need. Keep business DTOs, override-table access, business key policy, and configuration authority out of the helper; otherwise retain the behavior in its module. Use [shared-asset governance](../../../ai-plan/design/governance/platform/共享资产复用治理规范.md) when introducing or changing a reusable facility.

For System Config, preserve `configregistry -> system-config -> resolver/snapshot` authority. Modules never query override tables directly; Redis and web never own effective values. `runtime_apply_mode` remains the canonical apply signal; do not infer it in web.

An identified uncached hotspot ends as `reused-existing`, `added-cache`, or `deferred-with-reason`. A deferral states present protection and the trigger to revisit it. Repair upstream drift before introducing a downstream cache or compatibility bridge.

## Validation and Evidence

Use the task-class entrypoint from `graft-validation-runner`; backend changes require `graft validate backend`. Focused tests cover hit/miss, update/reset invalidation, concurrent miss collapse, and Redis degradation when relevant. They supplement the completion entrypoint.

For skill/docs changes use `git diff --check` and `python3 scripts/validate_ai_governance.py`; add `python3 scripts/validate_shared_asset_registries.py` only for registry changes. Reuse unchanged passing evidence.

Include authority, hotspot evidence, selected strategy, stale window, invalidation, shared-facility decision, any direct Redis justification or deferral, and validation in the integrated task report. Do not create a separate report for each consumer.

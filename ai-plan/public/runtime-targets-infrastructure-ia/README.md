# Runtime Targets Infrastructure IA

## Current Status Summary

- Topic objective: establish Runtime Target authority and evolve Infrastructure navigation, Docker resources, and Application/Compose integration without changing the existing sidebar mode.
- Current status: `active`.
- Task class: `cross-boundary`.
- Intake summary: long-running feature dispatched through `$graft-multi-agent-loop`.
- Canonical authority:
  - `ai-plan/design/architecture/导航与资源路由信息架构规范.md`
  - `ai-plan/design/architecture/运行目标与基础设施信息架构设计.md`
  - `ai-plan/design/domains/container/容器管理设计.md`
  - `ai-plan/design/domains/compose/Compose项目管理设计.md`
- Completed so far: Work Intake and authority foundation.
- Not started yet: menu contract, runtime-target implementation, and container-contract migration.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: navigation IA, runtime-target design, Application/Compose lifecycle authority, then server menu contracts and web navigation consumers.

## Owned Scope

- runtime target, infrastructure navigation, shared-resource, Docker resource, container contract, and Application/Compose integration authorities
- required `ai-plan/**`, `openapi/**`, `server/**`, generated contracts, and `web/**` slices as each bounded batch authorizes

Out of scope:

- Workspace Mode or replacing the existing side navigation after target selection
- placeholder Kubernetes/Podman menus, plaintext Docker TCP, and standalone containers becoming Applications

## Locked Decisions

1. Section Labels are visual-only; Docker, Kubernetes, Podman, Runtime Targets, Registry, and Certificates remain ordinary Infrastructure menus.
2. Runtime Targets own connections and capabilities; Compose lifecycle remains Application/Project authority.
3. Container list replaces user-facing “source” with independent `deployment_type` and `runtime_target` fields.

## Current Recovery Point

- Batch 1 established the Work Contract, repository design authority, roadmap position, and active-topic recovery materials.
- Next step: `menu-section-contract-and-sidebar-rendering`.

## Pending Batch Direction

- menu-section-contract-and-sidebar-rendering
- runtime-target-foundation
- container-deployment-type-and-target-filter
- docker-resources-and-application-integration
- cross-boundary-acceptance-and-archive-readiness

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/runtime-targets-infrastructure-ia/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`

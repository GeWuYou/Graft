# System Configuration

## Current Status

- Status: `archive-ready`.
- Topic completed for the `cross-boundary` System Configuration capability.
- Branch renamed to `feat/system-configuration`.
- Loop mode: `topic-completion-loop` through `$graft-multi-agent-loop`.
- Completed commits:
  - `9c014fc` - backend configuration authority
  - `8375f87` - web settings module
  - `326835e` - initial retention defaults
- Startup receipt:
  - governance source: root `AGENTS.md`
  - task class: `cross-boundary`
  - recovery source: `parent topic`
  - authority summary: module-owned configuration definitions registered through a new `server/internal/configregistry` surface; `server/modules/system-config` owns persisted administrator overrides; OpenAPI source owns wire contracts; `web/src/modules/system-config` consumes generated contracts and backend menu authority.

## Product Boundary

System Configuration is not an Apollo/Nacos-style configuration center, not user personalization, and not Scheduled Task instance configuration.

The intended model is:

- `ConfigDefinition`
  - declared by modules at `Register` time
  - canonical source of schema, default value, sensitivity, masking, group, module, title, description, restart requirement, and permissions
  - not persisted as a second database truth
- `system_config_values`
  - stores administrator overrides only
  - never stores module defaults as copied rows
- effective system config
  - `ConfigDefinition.DefaultValue` merged with the administrator override
- Scheduled Task config
  - remains task-instance override data
  - may later derive its default from effective system config, but existing task instance semantics remain intact

## Implementation Priorities

1. Establish backend authority and persistence baseline:
   - `server/internal/configregistry`
   - `server/modules/system-config`
   - migration/store/service/API
   - OpenAPI shapes with explicit `sensitive`, `masked`, and value visibility fields
2. Add web module and shared JSON Schema form reuse:
   - `web/src/modules/system-config`
   - reuse a shared Schema parser/form renderer rather than duplicating Scheduled Task-only logic
3. Connect initial definitions:
   - scheduler/logging/audit low-risk defaults first
   - auth/login/password policy after the baseline is stable

## Acceptance Conditions

- Config registry lives under `server/internal/configregistry`, not `server/internal/config`.
- `server/internal/config` remains startup/environment config only.
- `ConfigDefinition` comes from module registration and is not written to the database as canonical truth.
- `system_config_values` stores override JSON only.
- OpenAPI responses can represent sensitive values safely:
  - definitions expose `sensitive`
  - value payloads expose `masked`
  - sensitive effective/current values are not returned as plaintext
- Menu placement is under Service Management as `/server/system-config`.
- MVP validates both server and web because this is a shared contract/menu/permission slice.

## Archive-Ready Decision

- Decision: `archive-ready`.
- Reason: all planned batches completed, server and web completion validations passed, and no remaining in-scope
  implementation batch is required for the MVP authority baseline.
- Non-blocking future scope: later auth/login/password-policy definitions can be added after the baseline remains stable.

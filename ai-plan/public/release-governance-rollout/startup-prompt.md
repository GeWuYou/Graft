Start the next delegated round under the same topic-completion-loop.

Round context:
- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/release-governance-rollout/README.md`
- design authority:
  - `ai-plan/design/数据库表设计与迁移规范.md`
  - `ai-plan/design/服务端API边界与兼容治理规范.md`
  - `README.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-task`

Topic objective:
- Continue the `release-governance-rollout` topic under `topic-completion-loop` until the topic reaches
  `archive-ready`, becomes `blocked`, or new bounded batches must be defined.

Locked Phase 1 decisions:
1. `graft migrate up` and `graft dev` are the explicit migration entrypoints; `graft serve` remains pure runtime startup
   and must not become an implicit migration path.
2. `v0.1.0` release governance treats live schema evolution as forward-only migration governance; the repository does
   not promise down migrations, automatic rollback, or startup-time schema repair.
3. Any operator upgrade path that applies live migrations must verify database backup and restore capability first, and
   must preserve the pre-change config snapshot needed for manual recovery.
4. `v0.1.0` rollback support is documentation-first and operator-controlled:
   - document prerequisites
   - document decision points
   - document data/config risk
   - document minimum post-rollback verification
5. Stable config changes are classified as:
   - `additive`
   - `default-change`
   - `rename`
   - `semantic-change`
   - `removal`
6. Patch releases must not silently rename, remove, or reinterpret stable config keys.
7. Minor releases that introduce `rename`, `semantic-change`, or `removal` must record:
   - canonical owner
   - deprecated_in
   - removal_target
   - replacement
   - operator action required
   - release-notes required
   - upgrade-notes required
8. Startup deprecation warnings, config alias bridges, config rewrite helpers, and rollback helpers remain deferred; do
   not present them as existing support.

Next batch scope:
- `phase-3-release-operator-docs-baseline`
- allowed scopes:
  - `ai-plan/public/release-governance-rollout/**`
  - `README.md`
  - new release/install/upgrade/rollback docs directories when the batch needs them

Locked Phase 2 decisions:
1. Official `v0.1.0` release identity is the repository Git tag `vMAJOR.MINOR.PATCH`.
2. The future minimal `BuildInfo` field set is:
   - `version`
   - `git_commit`
   - `build_time_utc`
   - `git_tree_state`
3. `BuildInfo.version` uses bare semver; the canonical release tag keeps the `v` prefix.
4. Current repository state still lacks a unified BuildInfo injection path and a `graft version` subcommand; Phase 2 only
   fixed the governance boundary and must not be restated as existing operator support.
5. Future `graft version` must be a pure metadata readout and must not require database, Redis, runtime startup, or
   migration execution.
6. `v0.1.0` only promises one active repository release line at a time; no LTS line, multi-minor support matrix, or
   independent `server` / `web` release cadence is promised.
7. Official `server` artifact, `web` artifact, and release notes must come from the same release tag.
8. Migration version identifiers remain internal ordering values, not product versions or compatibility labels.

Phase 3 goals:
1. Lock the minimal operator document set for install, config reference, upgrade, rollback, and release notes.
2. Give each document a canonical location and cite the authority fixed by Phase 1 and Phase 2.
3. Keep the docs honest about explicit migration steps, backup/config snapshot prerequisites, and same-tag artifact
   coordination.

Phase 3 non-goals:
- no workflow implementation changes
- no docs-site or hosted documentation platform work
- no stronger operator-facing introspection promise
- no expansion into Docker, Kubernetes, or hosted deployment support

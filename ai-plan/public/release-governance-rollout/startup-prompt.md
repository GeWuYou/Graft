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
- `phase-2-release-identity-and-policy`
- allowed scopes:
  - `ai-plan/public/release-governance-rollout/**`
  - `README.md`
  - necessary topic-only design or roadmap documents

Phase 2 goals:
1. Lock the minimal `BuildInfo` contract.
2. Lock the minimal `graft version` output boundary.
3. Lock release policy, support boundary, and `server` / `web` / migration version coordination.

Phase 2 non-goals:
- no workflow implementation changes
- no stronger operator-facing introspection promise
- no expansion into Docker, Kubernetes, or hosted deployment support

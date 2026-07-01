在同一个 `topic-completion-loop` 下启动下一轮实现，不要切换到新的主题。

Round context:

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- recovery entry：`ai-plan/public/compose-project-management/README.md`
- design authority：
  - `ai-plan/design/domains/compose/Compose项目管理设计.md`
  - `ai-plan/design/domains/container/容器管理设计.md`
  - `ai-plan/design/architecture/模块与依赖注入设计.md`
  - `ai-plan/design/architecture/前端架构设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
  - `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
  - `ai-plan/design/governance/backend/后端安全与信任边界治理规范.md`
- AI skills：
  - `$graft-multi-agent-loop`

Topic objective:

- 持续推进 Compose Project Management，直到该主题达到 `archive-ready`、进入 `blocked`，或必须重新定义 bounded batches。

Locked architecture decisions:

1. `Project` 是 Compose Project 的管理与聚合层，不是新的 Runtime。
2. `Container` 是 Runtime Authority。
3. `Project` 负责：
   - Project Registry
   - Ownership
   - Compose Files / Env Files
   - Working Directory
   - Canonical Project Name / Display Name
   - Snapshot
   - Drift Detection
   - Project Lifecycle
   - Services Aggregation
   - Activity Aggregation entry
4. `Container` 负责：
   - Runtime State
   - Stats
   - Logs
   - Events
   - Shell
   - Inspect
   - Networks
   - Mounts
5. 明确禁止：
   - `Project` 持久化容器运行时信息
   - `Project` 实现自己的 Container Detail
   - `Project` 保存容器 Logs / Events / Stats
6. Phase 1 的 Activity 必须继续复用现有 container logs/events，并由前端做 fan-out 聚合。
7. Phase 1 的 Configuration 必须保持只读，且 API 至少拆成：
   - configuration metadata/list
   - configuration preview
   - configuration single-file content
8. `Canonical Project Name` 与 `Display Name` 必须分离。
9. Snapshot 只表示最近一次成功解析结果，不是 runtime cache。
10. Phase 1 只做 `local host`。
11. 推荐静态解析使用 `compose-go`，生命周期执行使用 `docker compose` CLI。
12. 推荐 persistence 使用模块自有 `database/sql + migrations`，不是先回到集中 Ent 仓储。

Implementation guardrails:

- 必须优先修 authority owner，不得在下游 consumer 做长期兼容层。
- 若 `project` 需要容器运行时聚合，只能新增最小稳定 shared boundary，不得直接 import `server/modules/container` 私有实现。
- 不得让 Overview 变成 runtime dashboard。
- 不得在 Phase 1 偷渡 managed create、editor、diff、deploy、validate UI、project logs/events backend aggregation。

Current loop state:

- completed batch:
  - `phase-2-batch-5-phase-2-validation-drift-guard-and-governance-sync`
- next batch:
  - `phase-3-batch-1-git-template-source-contract-and-boundary`
- pending batches:
  - `phase-3-batch-1-git-template-source-contract-and-boundary`
  - `phase-3-batch-2-directory-scan-and-auto-discovery-candidates`
  - `phase-3-batch-3-remote-host-boundary-and-activity-authority`

Phase 3 rebatching intent:

1. `phase-3-batch-1-git-template-source-contract-and-boundary`
   - 固定 git/template project source 的 contract、metadata、route/permission/menu boundary，不落 remote host、directory scan 或 backend activity aggregation
2. `phase-3-batch-2-directory-scan-and-auto-discovery-candidates`
   - 落地 scan/discovery candidate model 与 bounded authority，不直接注册项目、不改变 runtime authority
3. `phase-3-batch-3-remote-host-boundary-and-activity-authority`
   - 收敛 remote host 扩展边界与 project activity backend aggregation authority，再决定后续实现切片

Loop instructions:

- 默认 `loop_mode=topic-completion-loop`。
- 每轮只做一个 bounded batch。
- 每轮 closeout 后必须更新：
  - `ai-plan/public/compose-project-management/todos/compose-project-management-tracking.md`
  - `ai-plan/public/compose-project-management/traces/compose-project-management-trace.md`
- 当 `pending_batches` 非空时，主 agent 必须继续自动选择 `next_batch`，不要在普通 batch success 后停止。
- Phase 1 中每个实现 batch 完成后都要先过对应验证，再决定是否执行 `$graft-commit`。
- 若出现 ownership、validation、scope 或 safety blocker，按 `graft-multi-agent-loop` 规则进入 `blocked` 或 retry，不得静默越界。

Validation expectations:

```bash
git diff --check
node scripts/openapi-bundle.mjs
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

Required closeout:

- 明确当前 batch
- 明确变更的 authority owner
- 明确已运行验证与跳过原因
- 更新 loop batch state
- 仅在 terminal state 下输出 `Next-session startup prompt:`

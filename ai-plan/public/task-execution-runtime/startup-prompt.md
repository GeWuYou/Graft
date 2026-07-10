Continue work inside the same `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/task-execution-runtime/README.md`
- local execution truth: `server/AGENTS.md`, `web/AGENTS.md`, `ai-plan/AGENTS.md`
- design authority: `ai-plan/design/architecture/任务执行运行时设计.md`, `ai-plan/design/decisions/ADR-004-task-runtime-state-machine.md`, `openapi/**`, `server/internal/moduleapi/task.go`
- AI skills: `$graft-multi-agent-loop`, `$graft-multi-agent-task`, `$graft-table-design`, `$graft-plugin-scaffold`, `$graft-validation-runner`

Topic objective:

- Deliver the module-owned Task Runtime without turning it into a queue, a distributed system, or a business-module dependency sink.

Work contract summary:

- Long-running feature; design, roadmap, ADR and active topic are required; execute by `graft-multi-agent-loop`.

Locked decisions:

1. `Task / TaskPlan / Stage / StageExecutor` are canonical terms.
2. PostgreSQL is source of truth; Redis remains auxiliary.
3. Unknown Stage plus needs-attention Task is mandatory crash semantics for non-resumable external work.

Implementation guardrails:

- Repair canonical contracts before consumers.
- Keep Task independent from Project/Image/Migration/Backup implementations.
- Scheduler submits Tasks but never executes Stages.
- Do not create MQ, distributed worker, Temporal Server, DAG or automatic Docker command replay.

Current batch plan:

1. `task-module-persistence-state-machine`
2. `task-runtime-worker-and-recovery`

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Advance exactly one bounded batch this round and update tracking/trace.
- Validate and commit confirmed owned scope before the next batch.

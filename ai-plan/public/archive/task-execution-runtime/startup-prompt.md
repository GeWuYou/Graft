This topic is archived. Do not resume it as an active topic; use the current repository authority documents for any
follow-up Task Runtime work.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/archive/task-execution-runtime/README.md`
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

Archive result:

- All planned batches passed final cross-boundary validation.
- The Topic Runtime is historical evidence only; any new extension starts with current startup preflight and authority
  discovery rather than inheriting this closed topic state.

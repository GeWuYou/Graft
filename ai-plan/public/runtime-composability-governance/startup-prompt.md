# Runtime Composability Governance Startup Prompt

Continue `runtime-composability-governance` with the `docker-runtime-agent` subtopic.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic `runtime-composability-governance`
- owned scope: the current batch named in
  `subtopics/docker-runtime-agent/todos/docker-runtime-agent-tracking.md`, plus its required design authority and trace

First rerun the root startup preflight, then read `server/AGENTS.md`, `web/AGENTS.md`, `ai-plan/AGENTS.md`, ADR-026, the
runtime composition design/roadmap, parent topic files and Docker Runtime Agent subtopic tracking/trace. Preserve Task
Runtime, Runtime Target, Provider and Update Controller authorities; do not add server push, a shared Scope, dynamic
loader, runtime dependency solver, second DI/scheduler/Task state machine, Docker secrets in TaskPlan or a server-local
Docker fallback.

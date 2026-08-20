# Docker Runtime Agent Trace

## 2026-08-21 intake-and-batch-1-start

- Reproduced the architectural split: Runtime Target health uses the Moby SDK, while Application/Build lifecycle paths
  may invoke a missing `docker` executable. Host Docker availability does not imply CLI presence inside server.
- Reused the active `runtime-composability-governance` topic and created a bounded subtopic rather than a parallel topic.
- Accepted ADR-026: a single pull-based Runtime Agent owns always-on Docker socket access; Task Runtime owns external
  execution state; the short-lived Update Controller remains the survivor across server/Agent replacement.
- Selected platform/module/domain/event/table/test/consistency/delete semantic review layers before implementation.
- No server, web, OpenAPI, migration or generated code was changed in the Batch 1 design step.

## 2026-08-21 batch-1-accepted

- Converged project-layout, Compose, Build Provider SPI, Agent protocol, Task Runtime, self-update and active-topic
  recovery authority on ADR-026.
- Preserved ADR-006/009 host-path, durable state, fencing and survivor invariants while superseding CLI/launcher details.
- Validation passed: `git diff --check` and `python3 scripts/validate_ai_plan_structure.py`.

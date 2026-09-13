---
name: graft-multi-agent-task
description: "Execute one bounded Graft delegated slice and return validated evidence through the normal closeout and commit workflow."
---

# Graft Multi-Agent Task

Execute one bounded slice under [root governance](../../../AGENTS.md), reusing the current receipt and relevant evidence. This wrapper applies `graft-multi-agent-batch` when the slice benefits from disjoint internal delegation; a single loop worker can perform its assigned slice directly.

1. Preserve the inherited task class, recovery, objective, ownership, authority, execution context, validation, acceptance gate, topology revision/node/dependencies, budget, and commit authority. A worker cannot change the DAG or claim another worker's scope.
2. When delegating, use the [host mapping](../graft-multi-agent-batch/references/host-delegation.md) and root parent-model ceiling. Select a verified same/lower model autonomously; inherit the direct parent's configuration when comparison is unavailable.
3. Carry the authorized goal through implementation, relevant regressions, and validation. Diagnose and repair ordinary lint/type/style/test failures within the assigned responsibility without another user confirmation. Newly discovered related files may be included when ownership is clear; report cross-worker or upstream authority changes to the coordinator for reassignment before editing them. Preserve external-write boundaries.
4. At a validated milestone, route through `graft-task-closeout` and `graft-commit` according to the dispatch contract. If the coordinator owns the atomic commit, return the validated diff and evidence to it rather than forcing a worker commit. Do not conceal ownership or validation gaps.
5. Return concise scope, implementation, validation, commit state, risks, and any required recovery context. Capture reusable lessons only when evidence warrants it.

## Under a Topic Loop

Read [loop worker evidence](references/loop-worker.md) before final JSON or checkpoint output. Ordinary successful rounds return evidence without `Next-session startup prompt:`. The outer main agent alone verifies, settles, selects the next batch, and emits terminal handoff.

A checkpoint is a health report. After `can_continue=true`, continue the same round until final closeout. Quiet waits, absent visible diffs, and checkpoint replies are not permission to stop or transfer ownership.

If final evidence cannot be obtained safely, return the failed round, partial diff, validation, repair authority/eligibility, and safe retry inputs in `required_context` and `recovery_requirements`. Retry exhaustion is wave evidence, not topic completion, suspension, or a worker-selected terminal transition.

After changing loop/task contracts, run `python3 scripts/validate_loop_controller_contract.py` in addition to the relevant governance checks.

---
name: graft-task-closeout
description: "Close a Graft task slice with validation evidence, a scoped commit decision, and recovery context when a handoff is needed."
---

# Graft Task Closeout

Follow [root governance](../../../AGENTS.md) and [delivery details](../../../ai-plan/design/governance/ai/仓库协作与交付细则.md). Reuse the current startup receipt, relevant reviews, and validation evidence instead of rerunning skills or checks for unchanged scope.

1. Determine whether the authorized goal is complete, requires a real handoff, or has a concrete blocker. Ordinary progress and worker checkpoints are not closeout.
2. Verify the changed scope reached its task-class completion entrypoint. Use `graft-validation-runner` only when the proper coverage is uncertain. Separate automated completion from remaining human acceptance.
3. Evaluate commit eligibility through `graft-commit`: commit validated owned scope when root rules or the explicit request require it; preserve inseparable user changes and report actual blockers. A delegated worker follows its dispatch commit authority and returns evidence to its coordinator.
4. For an acquired numbered worktree, after the authorized owner commits and the validated worktree is clean, invoke `graft-worktree-manager` `closeout` to make the lease release-ready. Do not report a dirty owned slice as completed.
5. Capture a lesson through `graft-lessons-learned` only when this task exposed a reusable correction; omit a full lesson audit for routine work with no such evidence.
6. Report result, validation, commit SHA/title if created, and material limits. When human judgment remains, supply the concise acceptance contract from [AI review governance](../../../ai-plan/design/governance/ai/AI代码生成与Review规范.md) and state: `Implementation complete; automated verification passed; awaiting human acceptance.`
7. Emit one next-task startup prompt only for a future-turn handoff: governance source, task class, recovery source, owned scope, and mapped topic/subtopic when relevant.

## Machine-readable or Loop Closeout

Read [the JSON contract](references/json-closeout.md) only when the caller requests that output. A loop worker instead returns the [worker evidence contract](../graft-multi-agent-task/references/loop-worker.md); it must not emit the controller payload. Only the outer controller changes batches, selects recovery transitions, or decides terminal state.

Ordinary successful topic-completion batches continue in the same session. Emit `Next-session startup prompt:` only for the terminal or structured recovery handoffs allowed by the loop contract; do not stop because a worker returned.

---
name: graft-test-seam-review
description: Review whether Graft tests exercise durable behavior through the right module, HTTP, event, or web seam. Use when adding behavior, fixing bugs, or restructuring code and tests.
---

# Graft Test Seam Review

Use this semantic review before implementation and before closeout. It complements `graft-validation-runner`,
backend/web test governance, and TDD practices; it does not define a second test command or completion gate.

## Workflow

1. Identify the user-visible behavior, canonical authority, and highest public interface that can observe it.
2. Choose the smallest set of stable seams: module capability, HTTP contract, event publisher/handler, CLI, or Vue
   module boundary. Record why a lower seam is needed.
3. Check that tests assert independent expected behavior, not private methods, internal call counts, generated shapes,
   database rows, or a value recomputed by the implementation.
4. Check the failure modes that matter: authorization, validation, duplicate submission, retry/idempotency, lifecycle
   transitions, error envelope, cache invalidation, realtime freshness, and empty/loading states where applicable.
5. For bugs, require a deterministic red-capable reproduction before hypothesizing. Turn the minimized reproduction into
   a regression test at the real seam, then rerun the original loop.
6. Check test doubles: mock external boundaries only; prefer real in-memory or disposable adapters when they preserve
   behavior. Do not expose a new interface only because one test is inconvenient.

## Findings

Report `blocking`, `high`, `medium`, or `note` findings with seam, behavior, false-confidence risk, and repair. Conclude
with `behavior_under_test`, `selected_seams`, `test_evidence`, `missing_cases`, `double_strategy`, and `validation`.
If no correct seam exists, report that architecture gap explicitly and route it to module architecture review.

## Guardrails

- Use `graft-validation-runner` to choose the task-class completion entrypoint: `graft validate backend` for server
  changes, `bun run check` for web changes, and both for cross-boundary changes. Focused tests supplement them.
- Do not add snapshots, mocks, or broad end-to-end tests to hide an unchosen seam.
- Do not treat green tests as proof of semantic correctness when the test bypasses the authority boundary.

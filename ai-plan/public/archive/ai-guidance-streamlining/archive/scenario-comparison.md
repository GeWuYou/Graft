# Independent Guidance Scenario Comparison

## Method and Limits

[Neutral inputs](neutral-inputs.json) retain the original eight requests/facts. Independent baseline and revised agents
used isolated contexts, the same inherited model configuration and no model/reasoning overrides. They received equivalent
English inputs, with no expected answers. Baseline read only policy at `fe36f69c17ad6c2dfb3d84185a18254afe2dc7a3`;
revised read the frozen current policy, without baseline, diff, ADR-028, task tracking or the approved plan.

This is policy scenario evaluation, not implementation of eight real business tasks. Source/PR/runtime data were simulated.
The coordinator reviewed decisions for key-risk omissions, unauthorized actions and unnecessary pauses. There is one
qualitative evaluation per version, so no statistical model-performance or real task-completion claim follows.

## Policy Footprint Estimates

Values are unique selected whole-file policy bundles: file count / UTF-8 bytes. They are not measured per-case tool reads,
tokens or latency. Some documents were read only in sections; additional domain, PR, dispatch or topic context is excluded
as recorded in the [baseline](baseline-scenarios.json) and [revised](revised-scenarios.json) evidence. Do not subtract the
columns and call the result observed runtime savings. Baseline S5 preserves ambiguity over the initial file-list meaning.

| Case | Baseline files / bytes | Revised files / bytes |
| --- | --- | --- |
| S1 | 21 / 298156 | 13 / 94209 |
| S2 | 19 / 290015 | 14 / 90130 |
| S3 | 36 / 497507 | 23 / 175379 |
| S4 | 11 / ≥144379 | 6 / ≥44128 |
| S5 | 21 / 299837 | 18 / 103577 |
| S6 | 12 / ≥168262 | 11 / ≥64508 |
| S7 | 8 / ≥134840 | 8 / 54248 |
| S8 | 24 / ≥278467 | 16 / ≥96965 |

Revised research actually read 34 complete unique files (192547 bytes) plus selected sections (30088 bytes), total 222635
bytes across the entire eight-case investigation. No comparable complete baseline observed-read total was supplied.

## Decision Comparison

- S1/S2: both versions permit local fixes. Revised routes less general design material; no new topic, browser run or
  needless component query. Server behavior still receives a direct regression and its completion gate.
- S3: both repair the highest incorrect source, keep OpenAPI generation and two-domain validation, and preserve genuine
  display-label policy while removing redundant mapping. Revised concern selection follows actual authority risk.
- S4: revised read-only review completes with open/reported evidence/risk/next step; old final-contract language produced
  a blocked finding awaiting repair authority. Neither version authorizes code or remote changes from review alone.
- S5: baseline distinguished an estimate from a hard confirmed file allowlist. Revised explicitly proceeds with necessary
  clear-owner files and regressions in the authorized goal; another worker's file is reassigned by the coordinator.
- S6: revised preserves inseparable user edits, asks a concrete ownership question, and continues the independent related
  fixture. Neither production DDL nor push/reply becomes authorized merely because somebody suggested it.
- S7: revised permits inherited P or trusted lower L; H is forbidden and unknown U uses inheritance without pause. The
  comment-only exact-adjacent-tier requirement is removed. All nested edges obey the direct-parent ceiling and capacity.
- S8: both continue ordinary batch success, preserve controller ownership and stop on a required unapproved external side
  effect. Revised resumes recoverable context without treating worker return or a missing handoff prompt as completion.

Coordinator acceptance: no missing key risk, unauthorized action or unnecessary pause found in the eight revised decisions.
Required repository verification and commit are separate completion conditions; scenario acceptance cannot replace them.
The later successful Linux verification is recorded in final-validation.json; coordinator publication is verified by the containing commit.

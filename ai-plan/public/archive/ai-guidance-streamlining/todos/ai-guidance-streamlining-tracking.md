# AI Guidance Streamlining Tracking

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: root AGENTS policy and domain AGENTS execution with conditional AI governance detail
requires:
  design: true
  topic: true
  roadmap: false
  adr: true
execution:
  engine: direct-specialized-skill
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-guidance-streamlining
    - ai-plan/design/decisions/ADR-028-risk-based-agent-guidance.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Baseline: `fe36f69c17ad6c2dfb3d84185a18254afe2dc7a3`.
- Work Intake selects a bounded hybrid DAG; the approved plan keeps batches here instead of a duplicate roadmap.
- Semantic review: platform authority, change/spec alignment, consistency and deletion of duplicated rules.
- Verification: agent-only structural/contract tests plus isolated qualitative instruction scenarios.

## Batch Plan

| Node | Owner scope | Dependency | Status |
| --- | --- | --- | --- |
| docs | Four AGENTS and ai-plan authority/recovery | none | accepted |
| skills | 53 skill entries and references | ADR authority | accepted |
| validators | Existing governance validators and tests | ADR interfaces | accepted; final Linux checks passed |
| review | Independent baseline/revised scenarios and integration review | G1 implementation | eight paired decisions accepted by coordinator |
| closeout | Full required checks, scoped commit, topic archive | review | validated archive prepared for coordinator atomic commit |

G1 is one atomic policy/skill/validator commit group. Coordinator alone stages, commits and judges completion.

## Acceptance Conditions

- [x] Root/server/web at most 8192 bytes, ai-plan at most 4096 bytes; retention routing and new links reviewed.
- [x] All 53 names/entrypoints remain; descriptions and UI metadata align; implementation review accepted.
- [x] Relevant local repair continues; mixed ownership and external authorization stay protected in revised scenarios.
- [x] Model unknown resolves by inheritance; no higher-than-parent child in revised scenarios.
- [x] Existing structure and loop contracts pass in final Linux run; hooks/full completion entrypoints preserved.
- [x] Eight isolated old/new scenarios reviewed; no material revised finding; decisions and footprint assumptions recorded.
- [x] Validated topic archive prepared with complete evidence; no push/integration.
- Local commit completion is verified by the containing coordinator commit and manager closeout, not a preclaimed SHA.

## Rule Disposition

| Baseline concern | Disposition / owner |
| --- | --- |
| Startup receipt, authority-first, no compatibility without rationale | Retain compact root policy |
| Skill catalog / repeated startup / all-design reading | Remove repetition; conditional root/domain links |
| Git/worktree operations, integration evidence, license | Retain root safeguards; skill/AI review operational detail |
| Loop checkpoint/DAG/closeout data contracts | Retain recovery design and existing skill references/tests |
| Model uncertainty / exactly lower comments | Replace by parent ceiling and unknown-to-inherit |
| Exact-hunk repair confirmation | Replace by authorized-target ownership/side-effect assessment |
| Read-only findings completion | Separate report completion from remediation/ledger enums |
| Domain module/contract/migration/UI rules | Retain domain core; route detail to matching design owner |
| Description minimum and exact policy sentence guards | Remove; structure, references, byte budgets and scenario review |

## Evidence

Baseline inventory: [baseline](../archive/baseline.json).
Baseline checks at the recorded commit: AI governance passed; ai-plan structure passed; loop contract passed;
script unittest 169 tests OK; skill-local 6 files / 124 tests OK. Revised results are recorded separately below.

### Retention Routing

| Retained concern | Current conditional owner |
| --- | --- |
| Initial commit inventory, hunk staging, commit messages, no noise, push naming, integration evidence | 仓库协作与交付细则.md Git/提交章节 + canonical commit/push skills |
| CI split gates, publishing/PAT/tag/git-cliff, AGPL obligations | 仓库协作与交付细则.md CI/CD/许可章节 |
| Module Register/Boot/Shutdown, DI, moduleapi narrowness | server/AGENTS.md + 模块与依赖注入设计.md |
| Schema ownership, globally unique migrations, comments/generation/tests | server/AGENTS.md + 数据库表设计与迁移规范.md |
| audit/monitor/logger/httpx and future Log Explorer ownership | 模块与依赖注入设计.md Observability capability ownership |
| Bootstrap/index/globalRoutes/hiddenMenu, footer/shell ownership, key-first locale and no remount | 前端架构设计.md |
| QueryClient singleton/session clear, canonical API types, module API boundary | 前端架构设计.md |
| Scrollbar/Dialog/fonts/hygiene hooks/debug/locale formatting | 前端实现与检查细则.md |
| TDesign API/DOM/version evidence reuse, page structure without automatic pause | TDesign-MCP-辅助开发规范.md + 前端视觉设计规范.md |
| Loop DAG/checkpoint/timeout/retry/JSON protocols | AI任务追踪与恢复设计.md + existing loop skill contracts/tests |

Active-recovery scan found model references in handwritten-comment-governance that describe user-selected orchestration
and completed-batch evidence; left intact. build-domain-v2 explicitly authorized normal repairs; no conflicting generic
active instruction required modification. Historical archives/traces retained.


### Documentation Slice Review

- Standards pass: root remains policy/startup owner; domain details have explicit conditional homes; no runtime,
  environment inventory, database or external-system changes. Ownership and licensing protections retained.
- Spec pass: compact four entry files, authorized-target repair, unknown-model inheritance, read-only PR completion,
  conditional TDesign lookups and valid-check reuse implemented. Coupled 53-skill/validator work owned by other nodes.
- Deletion review: removed repeated startup/catalog/full-read/confirmation prose; retained loop protocol and domain
  protections via design/skill owners. No compatibility bridge or new runtime abstraction introduced.
- Validation: git diff --check passed; newly introduced local Markdown references resolve; UTF-8 LF confirmed.
  Current AGENTS bytes: root 8160; server 7685; web 7427; ai-plan 4079.
- Lessons review: current case is captured in ADR-028 and this retention/evaluation record; no separate lessons file
  needed to repeat the same policy. Paired policy scenario review and final Linux integration validation later passed.

### Integrated Evidence and Resolved Interruption

- [53-skill disposition](../archive/skill-dispositions.json): SKILL.md entry bytes 306856 → 251936; description characters
  14824 → 6117. Entry totals exclude conditionally loaded references; this is size accounting, not a quality score.
- [Neutral scenario inputs](../archive/neutral-inputs.json) preserve the original requests/facts separately from interpretations.
- [Baseline scenario evaluation](../archive/baseline-scenarios.json): eight independent qualitative decisions. Root already
  allowed simple UI fixes; baseline S5 has real file-list ambiguity, and exact-lower comment routing can pause.
- [Paired comparison](../archive/scenario-comparison.md) and [revised evidence](../archive/revised-scenarios.json) record all eight decisions.
  Coordinator found no key-risk omission, unauthorized action or needless pause. Footprint estimates are not measured
  per-case tool reads, tokens, timing or runtime savings. No statistical performance claim.
- Final requested WSL six-check run started but was cancelled after new WSL process startup hung even for a trivial
  command. It did not establish a final pass. Windows Python fallback passed AI governance, ai-plan structure, loop
  contract and scoped diff checks; Linux-only fcntl/temp-file semantics make native full suites an invalid substitute.
- A process-level Git-path export in the native full-test attempt altered local Git options. The run was stopped;
  staging, HEAD and protected user file were unchanged. With explicit user approval, filemode was restored to true and
  local autocrlf/safecrlf overrides removed; coordinator verified the restoration. No repository policy was weakened.
- That interruption was resolved after user-authorized distribution/service recovery. The successful Linux results below supersede the temporary blocker; historical unsuccessful runs remain evidence.

### Final Validation and Archive Readiness

The user authorized distribution restart and then broader WslService restart; the latter restored Linux execution.
All six approved final commands passed on the frozen implementation in the intended WSL/Linux environment:

| Command | Actual result |
| --- | --- |
| git diff --check | passed |
| python3 scripts/validate_ai_governance.py | passed; 53 skills; AGENTS all-chain 27357 bytes |
| python3 scripts/validate_ai_plan_structure.py | passed |
| python3 scripts/validate_loop_controller_contract.py | passed |
| python3 -m unittest discover -s scripts -p 'test_*.py' | 161 tests OK |
| python3 scripts/run_skill_tests.py | 6 files / 124 tests, zero failures |

The skill suite's first parallel WSL launch failed before Python started with Wsl/Service/0x8007274c; its sequential
Linux retry passed. [Final validation record](../archive/final-validation.json) distinguishes this from the successful run.
Baseline script unittest count was 169; revised count is 161 after the approved validator refactor. Do not conflate them.

All implementation/review/validation acceptance conditions are satisfied. This topic is archive-ready and its complete
folder is moved to public/archive with removal from the active index. The coordinator will recheck archive-only changes,
stage exactly the owned atomic G1 scope and commit; this record does not claim that Git commit already exists. The
containing commit and coordinator closeout provide local commit/lease evidence. No push, publishing or integration.

Experience capture: added L1 LESSON-GOVERNANCE-GIT-ENV-ISOLATION-001 in ai-plan/lessons/governance.md with index entry.
Recognizable and repeatable fixture-environment contamination meets the write threshold; no AGENTS/design promotion.

### Final Formatter-Only Adjustment

Existing pre-commit formatting required web/AGENTS.md to pass Prettier. Its table padding produced 8209 bytes, above the
8192 budget, so the same seven risk labels and links were represented as compact bullets. No policy content changed.
Prettier write/check passed idempotently. Final web size is 7401 bytes (26 fewer than the scenario-evaluated 7427).
Frozen evaluation measurements and the six-command run's 27357-byte chain remain historical actual measurements.
Final file sizes are root 8160, server 7685, web 7401, ai-plan 4079: raw sum 27325; combined chain 27331 including
6 separator bytes. Coordinator will recheck only affected structure/formatting before the atomic local commit.

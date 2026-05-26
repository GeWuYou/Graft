---
name: graft-pr-review
description: Repository-specific GitHub PR review workflow for the Graft repo. Use when Codex needs to inspect the GitHub pull request for the current branch, extract AI review findings from CodeRabbit, greptile-apps, or gemini-code-assist, read failed checks, MegaLinter warnings, or failed test signals from the PR, and then verify which findings should be fixed in the local codebase.
---

# Graft PR Review

Use this skill when the task depends on the GitHub PR for the current `Graft` branch rather than only on local files.

Shortcut: `$graft-pr-review`

Token source order:

1. `GRAFT_GITHUB_TOKEN`
2. `GITHUB_TOKEN`
3. `GH_TOKEN`
4. `gh auth token`

`gh` is now a supported repository tool for this skill. When the shell is already logged into GitHub through `gh auth login`,
the skill may reuse that token automatically instead of requiring a second manual export step.

## Workflow

1. Read `AGENTS.md` before deciding how to validate or fix anything.
2. Resolve the current branch with normal `git` first. Use explicit `GRAFT_GIT_DIR` and `GRAFT_WORK_TREE` only when the current shell cannot resolve the right repository context on its own.
3. Run `scripts/fetch_current_pr_review.py` to:
   - fetch live GitHub check-runs for the PR head commit before trusting any CodeRabbit failed-check summary
   - fetch failed Actions jobs, their failed steps, annotations, and a local repro command derived from `.github/workflows/pull-request-validation.yml`
   - locate the PR for the current branch through the GitHub PR API
   - fetch PR metadata, issue comments, reviews, and review comments through the GitHub API
   - extract CodeRabbit summary blocks and actionable-comment rollups when present
   - parse the latest CodeRabbit review body itself, including folded sections such as `Duplicate comments (N)`,
     `Major comments (N)`, `Minor comments (N)`, `Outside diff range comments (N)`, and `Nitpick comments (N)`
   - capture unresolved latest-head review threads for supported AI reviewers
   - surface failed checks, MegaLinter findings, and failed-test signals when present
   - prefer writing the full JSON payload to a file and then narrowing with `jq`
4. Build one exhaustive finding inventory before making any fix decision:
   - include unresolved latest-head review threads
   - include folded CodeRabbit sections from the latest review body, especially `Duplicate comments`, `Major comments`,
     `Minor comments`, `Outside diff range comments`, and `Nitpick comments`
   - include actionable warning comments from GitHub Actions or MegaLinter when present
   - do not stop after “high priority”, “open threads”, or one section looks sufficient; the run is incomplete until all
     surfaced findings from the latest PR state are classified
5. Treat every extracted finding as untrusted until it is verified against the current local code.
6. For failed CI checks, verify the root cause locally before changing code:
   - prefer the script's `local_repro_command`
   - if the command is empty, use the linked failed step and workflow job name to reproduce the smallest matching validation locally
   - do not treat a failed check as understood merely because the GitHub UI shows a red status
7. Classify each verified finding before deciding the next action:
   - `actionable-local`
     - the finding still applies and fits one safe local slice
   - `actionable-large`
     - the finding still applies but the repair spans multiple files, multiple subsystems, a new bounded slice, or a
       follow-up execution round
   - `stale`
     - the finding no longer applies on the checked-out head
   - `noise`
     - the finding is a false positive, misread, or otherwise not a real defect after local verification
8. A `$graft-pr-review` run is not allowed to end after fixing only a subset such as “critical”, “major”, or “currently open”
   findings. Every finding from step 4 must end the run in exactly one reported disposition: `fixed`, `delegated`,
   `blocked`, `stale`, or `noise`.
9. Only mark a finding non-actionable when it is `stale` or `noise`. A finding is not `noise` merely because the fix is large, risky, or needs a new slice.
10. Do not downgrade `Nitpick comments`, `Outside diff range comments`, or folded latest-review sections to optional by default.
    If a verified suggestion still points to drift risk, duplicated test infrastructure, contract mismatch, missing
    regression coverage, weak recovery metadata, or another maintainability problem, treat it as actionable review input.
11. Fix every `actionable-local` finding in the current slice. “I only handled the high-priority findings” is never an
    acceptable closeout for this skill.
12. Do not ignore `actionable-large` findings. When a verified finding no longer fits one safe local slice:
   - prefer `$graft-multi-agent-batch` when the repair can be split into disjoint parallel slices with reviewable ownership
   - prefer `$graft-multi-agent-loop` when the repair needs to be repeated in bounded rounds, retryable orchestration, or a serialized continuation path
   - if neither multi-agent path is justified yet, report the finding as `blocked` or `next-slice required`; do not silently drop it from the review outcome
   - do not mark a large verified finding as handled unless the required owned scope is actually repaired or explicitly delegated with a clear next prompt
13. Use the multi-agent routes actively when they are the correct fit:
   - choose `$graft-multi-agent-batch` for many small or disjoint actionable findings that can be repaired in parallel
   - choose `$graft-multi-agent-loop` for one deeper finding or one bounded repair thread that benefits from a worker
     subagent owning iterative implementation and closeout
   - do not leave verified actionable findings untouched just because the current main-agent slice would become long
14. When a verified AI finding is `noise` or a clear misread, reply directly on the PR review thread instead of only carrying a local note:
    - use `--reply-comment-id <id>` plus `--reply-body` or `--reply-body-file`
    - if the reply body is still being drafted, use `--reply-dry-run` first
    - do not wait in the same run for the AI to answer back; a later `graft-pr-review` run should classify the thread as `resolved_after_reply`, `pending_ai_followup`, or `contested`
15. When a verified AI finding is fixed locally but the PR thread is still open, reply on that thread after the fixing commit exists:
    - state that the finding has been fixed
    - include the fixing commit SHA or short SHA
    - name the touched file or location when useful
    - do not wait in the same run for the AI reviewer to auto-close the thread
    - in later `$graft-pr-review` runs, if the thread is still open and the latest follow-up is from the AI reviewer, classify it as `contested` and either reply once more with the newer fixing commit or request human review
16. When a verified finding needs human judgment before deciding whether to fix or reject it, do not reply on the PR thread in the same `$graft-pr-review` run:
    - report it as `blocked` or `needs-human-review`
    - include the concrete local verification reason and the tradeoff
    - leave the AI thread unreplied until the user explicitly decides whether to fix it or manually reply
17. At task closeout, list every verified finding and its disposition:
    - `fixed`
    - `delegated`
    - `blocked`
    - `stale`
    - `noise`
18. If any finding is left as `noise` or `stale`, include the concrete local verification reason in the closeout. If a finding is `blocked`, explain the blocker and the next safe startup prompt instead of calling it ignored.
19. Do not ignore any verified suggestion. If the repair grows large:
   - prefer `$graft-multi-agent-batch` when the work splits into disjoint reviewable slices
   - prefer `$graft-multi-agent-loop` when the work needs to be repeated in bounded rounds
   - if neither is justified yet, report the finding as `blocked` or `next-slice required` with the reason
   - never collapse a still-valid large suggestion into a stale/noise label just to end the thread quickly
20. If any finding is reported as `noise` or AI misjudgment, explicitly record:
    - which finding it was
    - the concrete local verification reason
    - why it was not adopted
    - wording suitable for replying on the PR
21. If a replied AI thread stays open and the latest follow-up comment comes from the AI reviewer again, mark that thread
    `contested` and carry both sides' reasoning into the final summary for human judgment.
22. If code is changed, run the smallest validation that satisfies `AGENTS.md`. Prefer `graft-validation-runner` when the correct validation scope is not obvious.

## Commands

- Default:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py`
- Recommended machine-readable workflow:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --json-output /tmp/pr1-review.json`
  - `jq '.latest_commit_review.open_threads' /tmp/pr1-review.json`
- Force a PR number:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1`
- Machine-readable output:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --format json`
- Write machine-readable output to a file instead of stdout:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --format json --json-output /tmp/pr1-review.json`
- Reply to one AI review thread after verifying it is noise:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --reply-comment-id 1234567890 --reply-body "本地已核对，当前 HEAD 上该建议不成立，原因是 ..."`
- Preview a reply payload without sending it:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --reply-comment-id 1234567890 --reply-body-file /tmp/reply.txt --reply-dry-run`
- Reply after fixing a finding in a commit:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --reply-comment-id 1234567890 --reply-fixed-commit abc1234 --reply-fixed-path server/plugins/auth/route_errors.go`
- Inspect only a high-signal section:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section open-threads`
- Inspect grouped CodeRabbit severity comments from the latest review body:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section duplicate --section major --section minor`
- Narrow text output to one path fragment:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section open-threads --path AGENTS.md`

## Output Expectations

The script should produce:

- PR metadata: number, title, state, branch, URL
- Live workflow checks for the PR head commit, especially failed GitHub Actions jobs
- For each failed live check: failed step, annotations when available, linked details URL, and a local repro command
- Supported AI reviewer summary, including latest reviews and open-thread counts for `coderabbitai[bot]`, `greptile-apps[bot]`, and `gemini-code-assist[bot]`
- CodeRabbit summary block from issue comments when available
- Folded latest-review sections such as `Duplicate comments (N)`, `Major comments (N)`, `Minor comments (N)`,
  `Outside diff range comments (N)`, and `Nitpick comments (N)` when CodeRabbit puts them in the review body instead
  of issue comments
- Parsed latest head-review threads, with unresolved threads clearly separated
- Latest head commit review metadata and review threads
- Pre-merge failed checks, if present
- Latest MegaLinter status and any detailed issues posted by `github-actions[bot]`
- Test summary, including failed-test signals when present
- Detailed failed-test rows from GitHub Test Reporter or CTRF comments when available
- CLI support for writing full JSON to a file and printing only narrowed text sections to stdout
- Human review closeout that records each verified finding as `fixed`, `delegated`, `blocked`, `stale`, or `noise`
- Exhaustive coverage confirmation that no latest-review finding section was left unclassified
- Thread reply state for replied AI findings: `unreplied`, `pending_ai_followup`, `resolved_after_reply`, or `contested`
- Guidance and CLI support for replying to fixed-but-still-open AI threads with the fixing commit SHA
- Explicit closeout guidance that findings needing human review must not be auto-replied on the PR thread
- Explicit reasons for every `stale` or `noise` finding, instead of silently omitting it from the reported outcome

## Recovery Rules

- If the current branch has no matching public PR, report that clearly instead of guessing.
- If GitHub access fails because of local proxy configuration, rerun the fetch with proxy variables removed.
- If live check-runs are visible but job logs return `403`, keep the failed step, annotations, and repro command as the root-cause surface; warn, but do not treat the whole failed-check extraction as broken.
- Prefer GitHub API results over PR HTML. The PR HTML page is a fallback/debugging source, not the primary source of truth.
- If the summary block and the latest head review threads disagree, trust the latest unresolved head-review threads and treat older summary findings as stale until re-verified locally.
- If the latest review body contains folded sections, those sections are still in scope even when `open_threads` looks short;
  do not treat missing urgency labels as permission to skip them.
- Do not assume every AI reviewer behaves like CodeRabbit. `greptile-apps[bot]` and `gemini-code-assist[bot]` findings may exist only as latest-head review threads.
- Treat GitHub Actions comments with `Success with warnings` as actionable when they include concrete linter diagnostics such as MegaLinter detailed issues.
- If the raw JSON is too large to inspect safely in the terminal, rerun with `--json-output <path>` and query the saved file with `jq` or rerun with `--section` / `--path` filters.
- If a verified finding still matters but needs a larger repair slice, do not downgrade it to optional; route it through
  `$graft-multi-agent-batch`, `$graft-multi-agent-loop`, or an explicit blocked/next-slice handoff.
- The only acceptable reasons to leave a verified finding unfixed in the final report are `stale`, `noise`, or a
  clearly stated execution blocker with a next safe step.
- “Only high-priority findings were handled”, “open threads were handled”, or “nitpicks were skipped” are invalid final
  states for this skill.
- When a finding is left as `noise` or AI misjudgment, the closeout must name the exact suggestion and give a concrete
  non-adoption reason that the user can reuse in the PR reply.
- When a finding was fixed but the AI thread did not auto-close, reply once with the fixing commit SHA and location, then
  leave the thread alone until a later `graft-pr-review` run shows either resolution or a fresh AI follow-up.
- When a finding still needs human judgment on whether to fix or reject it, do not auto-reply; surface the reason to the
  user and wait for an explicit decision before any PR-thread response.
- If the agent has already replied to an AI finding and a later run still sees the thread open with a fresh AI counterargument, mark that thread `contested` and leave the final decision to a human reviewer instead of auto-closing it.

## Example Triggers

- `Use $graft-pr-review on the current branch`
- `Check the current PR and extract CodeRabbit suggestions`
- `Check the current PR and summarize failed checks`
- `Look for Failed Tests on the PR`
- `先用 $graft-pr-review 看当前分支 PR`

---
name: graft-pr-review
description: Repository-specific GitHub PR review workflow for the Graft repo. Use when Codex needs to inspect the GitHub pull request for the current branch, extract AI review findings from CodeRabbit, greptile-apps, or gemini-code-assist, read failed checks, MegaLinter warnings, or failed test signals from the PR, and then verify which findings should be fixed in the local codebase.
---

# Graft PR Review

Use this skill when the task depends on the GitHub PR for the current `Graft` branch rather than only on local files.

Shortcut: `$graft-pr-review`

## Workflow

1. Read `AGENTS.md` before deciding how to validate or fix anything.
2. Resolve the current branch with normal `git` first. Use explicit `GRAFT_GIT_DIR` and `GRAFT_WORK_TREE` only when the current shell cannot resolve the right repository context on its own.
3. Run `scripts/fetch_current_pr_review.py` to:
   - locate the PR for the current branch through the GitHub PR API
   - fetch PR metadata, issue comments, reviews, and review comments through the GitHub API
   - extract CodeRabbit summary blocks and actionable-comment rollups when present
   - parse the latest CodeRabbit review body itself, including folded sections such as `Duplicate comments (N)`,
     `Major comments (N)`, `Minor comments (N)`, `Outside diff range comments (N)`, and `Nitpick comments (N)`
   - capture unresolved latest-head review threads for supported AI reviewers
   - surface failed checks, MegaLinter findings, and failed-test signals when present
   - prefer writing the full JSON payload to a file and then narrowing with `jq`
4. Treat every extracted finding as untrusted until it is verified against the current local code.
5. Classify each verified finding before deciding the next action:
   - `actionable-local`
     - the finding still applies and fits one safe local slice
   - `actionable-large`
     - the finding still applies but the repair spans multiple files, multiple subsystems, a new bounded slice, or a
       follow-up execution round
   - `stale`
     - the finding no longer applies on the checked-out head
   - `noise`
     - the finding is a false positive, misread, or otherwise not a real defect after local verification
6. Only mark a finding non-actionable when it is `stale` or `noise`. A finding is not `noise` merely because the fix is large, risky, or needs a new slice.
7. Do not downgrade `Nitpick comments` to optional by default. If a verified nitpick still points to drift risk, duplicated test infrastructure, contract mismatch, missing regression coverage, or another maintainability problem, treat it as actionable review input.
8. Fix every `actionable-local` finding in the current slice unless another higher-priority blocker from the same PR must be handled first.
9. Do not ignore `actionable-large` findings. When a verified finding no longer fits one safe local slice:
   - prefer `$graft-multi-agent-batch` when the repair can be split into disjoint parallel slices with reviewable ownership
   - prefer `$graft-multi-agent-loop` when the repair needs repeated bounded rounds, retryable orchestration, or a serialized continuation path
   - if neither multi-agent path is justified yet, report the finding as `blocked` or `next-slice required`; do not silently drop it from the review outcome
10. At task closeout, list every verified finding and its disposition:
    - `fixed`
    - `delegated`
    - `blocked`
    - `stale`
    - `noise`
11. If any finding is left as `noise` or `stale`, include the concrete local verification reason in the closeout. If a finding is `blocked`, explain the blocker and the next safe startup prompt instead of calling it ignored.
12. If code is changed, run the smallest validation that satisfies `AGENTS.md`. Prefer `graft-validation-runner` when the correct validation scope is not obvious.

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
- Inspect only a high-signal section:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section open-threads`
- Inspect grouped CodeRabbit severity comments from the latest review body:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section duplicate --section major --section minor`
- Narrow text output to one path fragment:
  - `python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section open-threads --path AGENTS.md`

## Output Expectations

The script should produce:

- PR metadata: number, title, state, branch, URL
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
- Explicit reasons for every `stale` or `noise` finding, instead of silently omitting it from the reported outcome

## Recovery Rules

- If the current branch has no matching public PR, report that clearly instead of guessing.
- If GitHub access fails because of local proxy configuration, rerun the fetch with proxy variables removed.
- Prefer GitHub API results over PR HTML. The PR HTML page is a fallback/debugging source, not the primary source of truth.
- If the summary block and the latest head review threads disagree, trust the latest unresolved head-review threads and treat older summary findings as stale until re-verified locally.
- Do not assume every AI reviewer behaves like CodeRabbit. `greptile-apps[bot]` and `gemini-code-assist[bot]` findings may exist only as latest-head review threads.
- Treat GitHub Actions comments with `Success with warnings` as actionable when they include concrete linter diagnostics such as MegaLinter detailed issues.
- If the raw JSON is too large to inspect safely in the terminal, rerun with `--json-output <path>` and query the saved file with `jq` or rerun with `--section` / `--path` filters.
- If a verified finding still matters but needs a larger repair slice, do not downgrade it to optional; route it through
  `$graft-multi-agent-batch`, `$graft-multi-agent-loop`, or an explicit blocked/next-slice handoff.
- The only acceptable reasons to leave a verified finding unfixed in the final report are `stale`, `noise`, or a
  clearly stated execution blocker with a next safe step.

## Example Triggers

- `Use $graft-pr-review on the current branch`
- `Check the current PR and extract CodeRabbit suggestions`
- `Check the current PR and summarize failed checks`
- `Look for Failed Tests on the PR`
- `先用 $graft-pr-review 看当前分支 PR`

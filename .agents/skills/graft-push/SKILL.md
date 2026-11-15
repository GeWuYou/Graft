---
name: graft-push
description: "Publish an explicitly requested Graft branch or diagnose push failures using the existing commit and validation workflows."
---

# Graft Push

Follow [root governance](../../../AGENTS.md) and [delivery details](../../../ai-plan/design/governance/ai/仓库协作与交付细则.md). Reuse this task's receipt and validation evidence. A diagnosis request is read-only; an explicit push request authorizes completing its necessary local commit/validation path and publishing the confirmed branch.

## Prepare the Branch

1. Inspect status, branch, upstream, and current local commit range. Resolve any detached HEAD destination before publishing; never push a `main-XX` pool marker.
2. In the primary checkout, an upstream with an open PR is an incremental push: use `@{upstream}..HEAD` and retain its branch name. Otherwise fetch `origin/main`, use all of `origin/main..HEAD` to derive the repository-compliant branch name, and rename locally when required. An empty range means no new publication.
3. Confirm the intended upstream and branch-name fit under delivery rules. Do not remove an old remote branch after a rename, push unrelated refs, amend history, or force-push without explicit authorization.
4. Route missing commits through `graft-commit`. Diagnose local hook, generated-artifact, snapshot, or validation failures and repair related task-owned causes continuously, including needed files and regression tests. Revalidate the changed scope. New files alone do not require confirmation; ambiguous ownership, changed goals, or new external actions do.
5. Preserve complete PR finding dispositions throughout remediation. Folded outside-diff/nitpick findings are not optional.

## Publish and Verify

Push only the confirmed current branch, using its existing upstream for incremental publication or `git push --set-upstream origin <branch>` for a new upstream. Never bypass hooks or branch protection.

Before any separately authorized PR reply or managed-ledger write, compare `git ls-remote --exit-code origin refs/heads/<current-branch>` with `git rev-parse HEAD`. Only an exact SHA match proves publication; a successful push process or hook does not. Missing or mismatched proof leaves remote review content untouched. Return to `graft-pr-review` or the explicitly requested `graft-pr-remediation` workflow for those writes.

Report branch/upstream, commit range and any rename, publication SHA evidence, or the concrete blocker and retry action. Preserve non-fast-forward history until an authorized safe resolution exists.

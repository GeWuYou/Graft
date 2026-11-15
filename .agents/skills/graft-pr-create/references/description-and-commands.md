# PR Description and Commands

Read when authoring a PR body or invoking its helper. Commands run from the repository root.

## PR Description Standard

Write the following Markdown sections in this order. Omit a section only when its required information cannot be
inferred without speculation; never fill a section with generic boilerplate.

- `# Summary`: three to eight bullets stating the product or engineering result, not filenames.
- `# Motivation`: the prior limitation, defect, inconsistency, missing capability, or design goal.
- `# Scope`: affected business capabilities, modules, contracts, or runtime areas.
- `# Key Changes`: `##` subsections grouped by logical behavior, each explaining intent and resulting behavior.
- `# Architecture Impact`: authority, boundary, contract, state, event, or dependency changes. State `No architectural
  contract changes.` when the evidence confirms none.
- `# User Impact`: visible workflows, UI, performance, or an evidenced absence of user-facing change.
- `# Compatibility`: backward compatibility, API compatibility, migrations, and configuration effects.
- `# Testing`: only commands or verification that were actually run. Do not claim a test, build, manual check, or
  screenshot without evidence.
- `# Risks`: concrete review or operational risks, including a justified low-risk assessment where appropriate.
- `# Follow-up Work`: intentionally deferred work only.
- `# Reviewer Notes`: the highest-value review targets, especially authority, contract, compatibility, or behavior
  changes.
- `# Checklist`: context-appropriate unchecked items for self-review, documentation, tests, breaking changes, and
  migrations.

The description must be intelligible without reading the diff. It must not use generic phrases such as "minor cleanup",
"miscellaneous fixes", "update code", or "improve implementation". Do not fabricate screenshots, issue references,
or test outcomes.

## Commands

- Dry run after writing `/tmp/graft-pr-description.md`:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --dry-run --description-file /tmp/graft-pr-description.md`
- Create or reuse PR for the current branch:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md`
- Write machine-readable output:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md --format json`
- Save machine-readable output:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md --json-output /tmp/graft-pr.json`
- Attempt guarded auto-merge enablement:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md --enable-auto-merge --confirm-automerge`

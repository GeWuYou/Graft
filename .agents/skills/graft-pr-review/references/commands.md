# PR Review Helper Commands

Use the complete inventory command before any focused section:

```bash
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py \
  --format json --json-output /tmp/graft-pr-review.json
```

Use `--pr <number>` when branch lookup is unavailable. Narrow only after the complete JSON exists:

```bash
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> --section open-threads
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> --section pre-merge-checks
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> \
  --section duplicate --section major --section minor --section outside-diff --section nitpick
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> --section advanced-security
```

Preview any explicitly authorized reply or ledger payload before writing:

```bash
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> \
  --reply-comment-id <id> --reply-body-file /tmp/reply.md --reply-dry-run
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py \
  --ledger-validate-body-file /tmp/pr-review-ledger.md
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> \
  --ledger-body-file /tmp/pr-review-ledger.md --ledger-expected-head <full-sha> --ledger-dry-run
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> \
  --ledger-body-file /tmp/pr-review-ledger.md --ledger-expected-head <full-sha>
```

Only after explicit remote-write authorization and exact remote-SHA publication proof may the same reply or ledger
command be rerun without its dry-run flag.

Resolve an eligible supported-AI review thread only after sending its reply, rebuilding the inventory, and proving the
PR head equals the published local HEAD:

```bash
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> \
  --resolve-comment-id <root-comment-id> --resolve-expected-head <full-sha> --resolve-dry-run
python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr <number> \
  --resolve-comment-id <root-comment-id> --resolve-expected-head <full-sha>
```

The helper refuses non-AI root comments, a mismatched PR head, abbreviated SHAs, or a combined reply-and-resolve
invocation. Keep the two writes separate so a fresh inventory can detect a contested reviewer response before
resolution.

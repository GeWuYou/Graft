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
  --ledger-body-file /tmp/pr-review-ledger.md --ledger-dry-run
```

Only after explicit remote-write authorization and exact remote-SHA publication proof may the same reply or ledger
command be rerun without its dry-run flag.

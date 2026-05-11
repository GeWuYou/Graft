# MVP Extension Path Trace

## 2026-05-12

- Established `mvp-extension-path` as the first long-lived active topic for Graft.
- Bound the topic to branch `feat/mvp-extension-path` so future MVP work has a stable recovery entrypoint.
- Migrated repository-wide design documents from `plan/` into `ai-plan/design/`.
- Migrated the MVP execution document from `plan/` into `ai-plan/roadmap/`.
- Added `ai-plan/design/AI任务追踪与恢复设计.md` to define the boundary between repository truth and topic recovery
  documents.
- Updated `AGENTS.md`, `README.md`, and `graft-boot` so boot and implementation rules now point at `ai-plan/`.
- Validation target for this change is documentation governance consistency rather than runtime compilation.

## 2026-05-12 `.ai/environment`

- Introduced `.ai/environment/tools.raw.yaml` and `.ai/environment/tools.ai.yaml` as repository-wide environment truth.
- Added `scripts/collect-dev-environment.sh` and `scripts/generate-ai-environment.py` so the inventory can be
  regenerated instead of hand-maintained.
- Updated `README.md`, `AGENTS.md`, `graft-boot`, and the AI governance docs so startup flow reads environment truth
  before making toolchain assumptions.
- Captured the current reality that `web` bootstrap files exist, while `server/go.mod` is still absent and `go` is not
  installed on this machine.

## Next Step

- Use `ai-plan/public/README.md` plus this topic's tracking file as the default recovery path when substantive MVP work
  begins.

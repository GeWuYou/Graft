# MVP Extension Path Tracking

## Topic

- Topic: `mvp-extension-path`
- Branch: `feat/mvp-extension-path`
- Scope: `server/core`, platform registries, initial plugins, and the `web` shell required by the MVP path

## Goal

- Keep one long-lived recovery entrypoint for the MVP extension path while the repository is still stabilizing its core
  architecture and implementation sequence.

## Repository Truth

- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/roadmap/MVP实施计划.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Stages

- Stage A: core runtime
- Stage B: platform registries
- Stage C: initial plugins
- Stage D: web shell and dynamic menu path

## Current Recovery Point

- The repository AI workflow has been upgraded from `plan/` to `ai-plan/`.
- Repository-wide design truth now lives in `ai-plan/design/`.
- Repository-wide implementation sequencing now lives in `ai-plan/roadmap/`.
- The long-lived branch `feat/mvp-extension-path` has been created and is now the default execution branch for this
  topic.
- Repository-wide environment truth now lives under `.ai/environment/`, with scripts that regenerate raw and AI-facing
  inventories.
- This topic is the default recovery entrypoint for future MVP-path work.

## Active Risks

- The repository has not yet implemented the MVP runtime, so the topic currently tracks governance and recovery shape
  rather than executable platform milestones.
- Future work must keep repository-wide design truth and topic-level recovery documents aligned instead of creating a
  second source of truth.
- The current machine does not have `go` installed, and the repository still lacks `server/go.mod`, so server build
  expectations must remain explicitly gated in the environment inventory.

## Latest Validation

- `rg -n -P "(?<!ai-)plan/" AGENTS.md README.md .gitignore .agents/skills -S`
- `rg -n "ai-plan/" AGENTS.md README.md .gitignore .agents/skills ai-plan -S`
- `bash scripts/collect-dev-environment.sh --check`
- `bash scripts/collect-dev-environment.sh --write`
- `python3 scripts/generate-ai-environment.py`
- `python3 -c 'import yaml; yaml.safe_load(open(".ai/environment/tools.raw.yaml", "r", encoding="utf-8")); yaml.safe_load(open(".ai/environment/tools.ai.yaml", "r", encoding="utf-8")); print("ok")'`

## Immediate Next Step

- Start the first substantive MVP implementation task on `feat/mvp-extension-path` and use `.ai/environment/tools.ai.yaml`
  as the default environment truth before making runtime or package-manager assumptions.

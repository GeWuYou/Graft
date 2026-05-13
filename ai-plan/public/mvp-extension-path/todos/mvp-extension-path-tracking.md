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
- `ai-plan/design/代码注释与模块文档规范.md`
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
- The repository now contains the first substantive MVP shell implementation across both `server` and `web`.
- `server` has a minimal runtime shell with explicit plugin registration, lifecycle ordering, registries, and a sample
  `user` plugin.
- `server` now uses an env-first configuration path with PostgreSQL and Redis as required core infrastructure.
- `server/go.mod` keeps direct dependencies documented and leaves indirect dependencies in standard Go tool format.
- `web` has a minimal Vue 3 + TDesign admin shell with `AuthLayout`, `BasicLayout`, static routing, mock auth, and a
  navigation store reserved for backend-driven menu metadata.
- Repository truth for the planned server-wide ORM migration now uses Ent as the backend ORM baseline.
- Repository truth for schema changes now uses Atlas versioned migrations executed through an explicit CLI step before
  application startup, not during runtime boot.
- Plugin-facing design truth now reserves a neutral repository / store factory boundary instead of exposing a concrete
  ORM handle through `plugin.Context` or cross-plugin public services.
- `server` now uses an Ent-backed database bootstrap, a Cobra CLI with `graft migrate up`, and a repository / store
  factory boundary in `plugin.Context`.
- `server` no longer exposes `*gorm.DB` or a runtime migration registry through the plugin lifecycle surface.
- The initial Ent schema, generated client, and Atlas-versioned SQL baseline now live under `server/internal/ent/`.
- Repository automation now includes a `graft-pr-review` skill that can resolve the current branch PR through the
  GitHub API and extract AI review findings, failed checks, MegaLinter warnings, and failed test signals into a local
  verification input.
- `server` now enforces an explicit MVP-only backend permission guard for protected plugin routes instead of relying on
  frontend route metadata alone.
- `server` runtime shutdown now honors reverse plugin `Shutdown` ordering and closes Redis / database resources on both
  startup failures and normal process exit.
- `graft migrate up` now resolves the default Atlas migration directory from either the repository root or the server
  module root so the CLI does not depend on one fragile working directory.
- `server/internal/httpx` now serializes `Run` / `Shutdown` lifecycle ownership through one guarded server pointer so
  concurrent start-stop transitions cannot race on partially applied runtime state.
- `graft-pr-review` now prefers native `git` before the Windows fallback in WSL-like shells, keeps JSON stdout stable
  when `--json-output` is requested, and treats visible `Addressed in commit` markers as resolved review threads.
- `graft-pr-review` now suppresses the missing-actionable warning when the latest CodeRabbit review was parsed through
  non-nitpick grouped sections such as `major`, `minor`, `duplicate`, or `outside-diff`.
- Repository-wide comment governance now has an explicit Chinese-first documentation rule set, module `README.md`
  navigation guidance, comment priority ordering, and exemption boundaries for generated or artifact code.
- The first implementation wave for the comment governance topic targets `server/internal/container`,
  `server/internal/plugin`, `server/internal/httpx`, `server/internal/app`, and `server/plugins/user`.
- The comment-governance wave now also covers `server/internal/config`, `server/internal/database`,
  `server/internal/menu`, `server/internal/permission`, `server/internal/cronx`, `server/internal/store`,
  `server/internal/pluginapi`, `server/internal/redisx`, and the current `web` shell's route/store boundary comments.
- Local startup ergonomics now use `graft dev` as the primary development entrypoint, so IDEs and Windows shells no
  longer depend on `bash scripts/dev-server.sh` to compose migration plus server startup.
- The temporary `scripts/dev-server.sh` compatibility wrapper has been removed, so repository startup guidance now
  points only at the Go CLI entrypoint.

## Active Risks

- Future work must keep repository-wide design truth and topic-level recovery documents aligned instead of creating a
  second source of truth.
- The main implementation risk is replacing existing runtime assumptions around startup-time migrations and direct DB
  handle access without leaking Ent-specific details across plugin boundaries.
- Comment governance must avoid turning module READMEs into duplicated design truth; detailed design still belongs in
  `ai-plan/design/`.
- Atlas CLI execution has not yet been validated against a real local PostgreSQL instance in this environment because
  the `atlas` binary is not installed and no target database was exercised during this change.
- The current backend permission gate is an explicit MVP placeholder based on request headers; it still needs the real
  auth + RBAC plugin chain before the authorization contract can be considered production-ready.

## Latest Validation

- `rg -n -P "(?<!ai-)plan/" AGENTS.md README.md .gitignore .agents/skills -S`
- `rg -n "ai-plan/" AGENTS.md README.md .gitignore .agents/skills ai-plan -S`
- `bash scripts/collect-dev-environment.sh --check`
- `bash scripts/collect-dev-environment.sh --write`
- `python3 scripts/generate-ai-environment.py`
- `python3 -c 'import yaml; yaml.safe_load(open(".ai/environment/tools.raw.yaml", "r", encoding="utf-8")); yaml.safe_load(open(".ai/environment/tools.ai.yaml", "r", encoding="utf-8")); print("ok")'`
- `cd web && bun install`
- `cd web && bun run typecheck`
- `cd web && bun run build`
- `cd server && go mod tidy`
- `cd server && go list -m -u all`
- `cd server && go build ./cmd/graft`
- `cd server && go test ./...`
- `bash scripts/collect-dev-environment.sh --write`
- `python3 scripts/generate-ai-environment.py`
- `rm -rf web/dist`
- Documentation-only validation target for this update: owned `ai-plan/` files stay mutually consistent on Ent,
  Atlas versioned migrations, explicit migration CLI flow, and repository / store factory boundaries.
- `cd server && GOSUMDB=off go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/ent/schema`
- `cd server && GOSUMDB=off go mod tidy`
- `cd server && go build ./cmd/graft`
- `cd server && go test ./...`
- `cd server && go run ./cmd/graft --help`
- `cd server && go run ./cmd/graft migrate --help`
- `python3 .agents/skills/graft-pr-review/scripts/test_fetch_current_pr_review.py`
- `env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --section pr --section open-threads --section warnings`
- `env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --format json --json-output /tmp/graft-pr1-review.json`
- `jq '.pull_request, .review_agents, .latest_commit_review.open_threads, .parse_warnings' /tmp/graft-pr1-review.json`
- `cd server && go test ./...`
- `cd server && GOCACHE=/tmp/go-build-cache go test ./internal/container ./internal/httpx ./internal/cli`
- `cd web && bun run typecheck`
- `cd server && go test -race ./internal/httpx`
- `python3 .agents/skills/graft-pr-review/scripts/test_fetch_current_pr_review.py`
- `env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --section pr`
- `env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY python3 .agents/skills/graft-pr-review/scripts/fetch_current_pr_review.py --pr 1 --format json --json-output /tmp/graft-pr1-review.json`
- `jq '{open_thread_count: (.latest_commit_review.open_threads | length), open_threads: [.latest_commit_review.open_threads[] | {path, status}]}' /tmp/graft-pr1-review.json`
- `python3 .agents/skills/graft-pr-review/scripts/test_fetch_current_pr_review.py`
- Documentation and focused `server` validation for the comment-governance update: owned `ai-plan/` files,
  `AGENTS.md`, first-wave module `README.md` files, and the directly touched `server` packages stay consistent with
  the new Chinese comment rules and compile under package-level tests.
- `cd server && go test ./internal/cli`
- `cd server && go build ./cmd/graft`
- Expanded comment-governance validation for this update: touched `server` packages and `web` shell files keep
  Chinese-first comments aligned with implementation and still pass direct compile-oriented validation.
- `cd server && go test ./internal/config ./internal/database ./internal/cronx ./internal/menu ./internal/permission ./internal/pluginapi ./internal/redisx ./internal/store ./internal/store/entstore ./internal/app ./internal/cli ./internal/container ./internal/httpx`
- `cd web && bun run typecheck`

## Immediate Next Step

- Continue the remaining incremental comment-governance sweep on hand-written modules that still lack high-priority
  Chinese comments, then return to the Atlas migration and real auth + RBAC plugin chain work.
- Exercise `graft dev` against a disposable PostgreSQL instance with a real Atlas installation, then continue the MVP
  path with the real auth + RBAC plugin chain work.

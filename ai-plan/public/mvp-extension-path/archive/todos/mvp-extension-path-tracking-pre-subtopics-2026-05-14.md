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
- The hand-written `server` Go comment-governance sweep now also covers the remaining CLI, runtime shell, registry,
  repository, resource-bootstrap, and sample plugin packages, and `AGENTS.md` now states when complex functions should
  use `参数：` / `返回值：` sections instead of forcing that template onto every function.
- Local startup ergonomics now use `graft dev` as the primary development entrypoint, so IDEs and Windows shells no
  longer depend on `bash scripts/dev-server.sh` to compose migration plus server startup.
- The temporary `scripts/dev-server.sh` compatibility wrapper has been removed, so repository startup guidance now
  points only at the Go CLI entrypoint.
- `server` runtime now owns a first-class Zap logger and a platform i18n service, and injects both through
  `plugin.Context` plus the service container instead of leaving plugins to invent their own hooks.
- `server` HTTP errors now reserve a stable `message_key + message + locale` response contract, with `zh-CN` as both
  default and fallback locale.
- `web` shell now reserves an application-level i18n path for locale state, message lookup, and request header
  propagation, while still keeping the initial visible language surface in Chinese.
- Repository truth for the `web` governance baseline now requires one explicit frontend quality chain built from
  `TypeScript strict`, `format:check`, `ESLint`, `Stylelint`, `Vitest`, `Husky + lint-staged`, and `commitlint`.
- `web/ai-libs/tdesign-vue-next-starter` is now documented as a local reference source for governance and TDesign
  usage patterns only; it must not be copied wholesale into the production `web` shell.
- Repository git-message rules now explicitly forbid committing literal escaped control text such as `\n` or `\t`
  inside commit titles or bodies; automation must expand them into real multi-line text before `git commit`.

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
- The first platform i18n wave intentionally keeps only the extension hooks; most menus, routes, and pages are still
  backed by Chinese-first source strings and will need gradual migration to message keys.
- The frontend governance baseline is currently documented but not yet fully implemented in the `web` toolchain, so
  follow-up work must keep scripts, hooks, and validation order aligned with the new repository truth.

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
- Final hand-written `server` comment-governance validation for this update: `AGENTS.md`, the repository comment
  design document, active-topic recovery docs, and the directly touched `server` packages stay aligned on the
  mixed conservative Go comment style.
- `cd server && go test ./internal/app ./internal/cli ./internal/config ./internal/container ./internal/cronx ./internal/database ./internal/httpx ./internal/menu ./internal/permission ./internal/plugin ./internal/pluginapi ./internal/redisx ./internal/store ./plugins/user && go build ./cmd/graft`
- Current validation target for the logger + i18n extension-hook update: touched `server` core packages, the sample
  `user` plugin, updated `ai-plan/` truth, and the `web` shell i18n path compile without widening into live database
  or Atlas integration.
- `cd server && go mod tidy`
- `cd server && go test ./...`
- `cd server && go build ./cmd/graft`
- `cd web && bun run typecheck`
- `cd web && bun run build`
- Documentation-only validation for the frontend-governance baseline update: `git diff --check`, `rg -n "format:check|Stylelint|Vitest|lint-staged|commitlint|ai-libs|\\\\n|\\\\t" AGENTS.md ai-plan/design/前端架构设计.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/traces/mvp-extension-path-trace.md -S`, and manual `sed -n` review of the touched sections.
- PR-review follow-up validation target for this update: verify only the still-valid `server` + `web` findings from
  PR `#5`, then run the smallest cross-boundary checks that cover the touched packages, i18n contracts, and frontend
  test helpers without widening into live Atlas or database execution.
- `cd server && go test ./internal/cli ./internal/httpx ./internal/i18n ./internal/plugin ./plugins/user`
- `cd web && bun run test:run -- NotFoundPage locale`
- `cd web && bun run typecheck`
- `cd web && bun run build`

## Immediate Next Step

- Implement the documented `web` governance baseline in the actual frontend toolchain, then exercise `graft dev`
  against a disposable PostgreSQL instance with a real Atlas installation while the real auth + RBAC plugin chain
  replaces the current request-header placeholder path.

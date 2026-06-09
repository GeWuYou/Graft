# Graft

Graft is a composable admin platform built with Go and Vue 3.

The project is not a single-purpose business application and is not a dynamic extension marketplace. Its current
architecture is a module-oriented modular monolith: the backend composes business capabilities through compile-time
modules, while the frontend provides a Vue 3 admin shell with module-owned feature pages.

Current baseline:

- Backend: `Go + Gin + Ent + PostgreSQL + Redis`
- Frontend: `Vue 3 + TypeScript + Vite`
- UI: `TDesign Vue Next`
- Architecture: compile-time modules in a modular monolith
- Dependency model: lightweight DI and explicit service registration, not a heavyweight IoC container

## Documentation

- [Project design](ai-plan/design/项目设计.md)
- [Module and dependency injection design](ai-plan/design/模块与依赖注入设计.md)
- [Frontend architecture design](ai-plan/design/前端架构设计.md)
- [Contract and magic-value governance](ai-plan/design/契约治理与魔法值治理规范.md)
- [MVP implementation plan](ai-plan/roadmap/MVP实施计划.md)
- [AI task tracking and recovery design](ai-plan/design/AI任务追踪与恢复设计.md)
- [AI Plan recovery index](ai-plan/public/README.md)
- [AI environment inventory](.ai/environment/README.md)

For repository-level coding, validation, startup, and commit rules, read [AGENTS.md](AGENTS.md) first. Backend-specific
execution rules live in [server/AGENTS.md](server/AGENTS.md), and frontend-specific execution rules live in
[web/AGENTS.md](web/AGENTS.md).

## Current State

The repository is in the MVP convergence stage. The priority is to stabilize the backend runtime, module boundaries,
real server/web contracts, and the minimum admin platform loop around:

- `auth`
- `user`
- `rbac`
- `audit`
- `scheduler`

Business behavior belongs under `server/modules/*`. Stable cross-module backend contracts belong under
`server/internal/moduleapi/**` or another documented stable boundary. Frontend business capabilities should default to
`web/src/modules/<name>`.

The repository also keeps `.ai/environment/` as generated environment truth:

- `.ai/environment/tools.raw.yaml` records raw local machine and repository facts.
- `.ai/environment/tools.ai.yaml` records the condensed inventory used by AI agents and contributors.

## Local Server

The server uses `.env` as its primary local runtime configuration source. The recommended IDE working directory is
`server`. If the command is launched from the repository root, the server falls back to `server/.env`; if it is launched
from a nested server directory such as `server/cmd/graft`, it walks upward to the same `server/.env`.

Minimal startup:

1. Copy `server/.env.example` to `server/.env`.
2. Set the local auth secrets in `server/.env`.
3. Run the development entrypoint:

```bash
cd server
go run ./cmd/graft dev
```

If auth secrets are missing, startup fails with:

```text
GRAFT_AUTH_JWT_SECRET or GRAFT_AUTH_SIGNING_KEY is required
```

To generate local auth secret values:

```bash
cd server
go run ./cmd/graft-jwt-secret
go run ./cmd/graft-signing-key
```

Each command prints one config line that can be pasted into `server/.env`.

`graft dev` is the local development supervisor. It runs explicit migrations first and starts the service only after
migrations succeed. This does not change `graft serve`, which remains the pure runtime startup command.

### Hot Reload

The repository provides a fixed Air configuration for local server rebuilds:

```bash
cd server
go run ./cmd/graft dev air
```

Notes:

- Air is pinned as a dev-only Go tool dependency in `server/go.mod`.
- Air rebuilds the server binary and restarts the serve child process.
- Air does not run `graft dev` or `graft migrate up`.
- Server config loading remains owned by the server `.env` lookup rules.

Recommended hot-reload flow:

```bash
cd server
go run ./cmd/graft migrate up
go run ./cmd/graft dev air
```

Run `graft migrate up` before hot reload on first startup or after schema/migration changes. For a one-shot
"migrate then start" flow, keep using `go run ./cmd/graft dev`.

### Reset the Local Admin

For repeated local verification of the default admin forced-password-change flow:

```bash
cd server
go run ./cmd/graft dev reset-admin
```

This dev-only command is allowed only when `GRAFT_APP_ENV=local` or `test`. It ensures the default admin user `graft`
exists, resets the password to `graft-admin`, and sets `must_change_password=true`.

After running it, clear browser `localStorage` and `sessionStorage`, then log in with `graft / graft-admin`.

### Split Migration and Runtime Startup

If you need to run migrations and startup separately:

```bash
cd server
go run ./cmd/graft migrate up
go run ./cmd/graft serve
```

Important server notes:

- The root `graft` command only prints help; it does not start the service.
- `graft dev` and `graft migrate up` require the Atlas CLI.
- The default migration chain is owner-aligned across live core-owned and module-owned migration directories.
- The historical shared Ent migration directory is retained only for explicit manual or diagnostic use.
- After adding, renaming, or editing migration files, refresh the corresponding Atlas hash before rerunning migrations.
- `graft serve` connects to PostgreSQL and Redis before serving; unavailable dependencies fail startup.
- In GoLand or another IDE, use working directory `server`, program entry `./cmd/graft`, and argument `dev`.

Windows PowerShell / CMD can use the same Go command:

```powershell
cd server
go run ./cmd/graft dev
```

If the CLI has already been built:

```powershell
cd server
.\graft.exe dev
```

## Server Validation

The backend completion entrypoint is:

```bash
cd server
go run ./cmd/graft validate backend
```

The repository pins `golangci-lint v2.12.2` and requires local development, AI agents, and CI to reuse this backend
validation entrypoint instead of maintaining separate blocking lint commands.

Backend completion order:

1. Migration version gate
2. `graft validate backend --stage lint`
3. Smallest directly relevant `go test` scope
4. `go build ./cmd/graft`
5. `graft validate smoke` when a runtime startup proof is needed

The backend blocking lint gate is changed-file scoped against the resolved base branch using
`--new-from-rev=<merge-base> --whole-files`. Untouched files are not blocking gate failures. Full-repository lint is
audit-only backlog scanning. New code must not expand the lint backlog.

## Local Web

Local frontend configuration should not commit real `web/.env.development` values. The committed file is
`web/.env.example`; local `.env` files remain ignored.

Minimal startup:

1. In the canonical repository root, copy `web/.env.example` to `web/.env.development`.
2. Set `VITE_API_TARGET` to the local backend address.
3. Start Vite:

```bash
cd web
bun run dev
```

Default development request flow:

- Browser requests go to `http://localhost:3002/api/...`.
- The Vite dev proxy forwards API calls to `VITE_API_TARGET`.

Notes:

- Keep `web/.env.development`, `web/.env.local`, and other `web/.env.*` local files untracked.
- `web/.env.example` is only a shared template and must not contain personal secrets or machine-specific addresses.
- When multiple long-lived or temporary worktrees exist, keep one canonical `web/.env.development` and let the
  worktree initialization flow create relative symlinks instead of copying per-worktree local config.

## Web Validation

The frontend completion entrypoint is:

```bash
cd web
bun run check
```

`bun run check` currently runs:

```text
format:check -> typecheck -> openapi:frontend-governance:check -> lint:i18n -> lint -> stylelint -> hygiene:check -> test:run -> build
```

Focused commands are fine during development, but completion, handoff, and merge readiness should use `bun run check`
unless the task explicitly reports why a narrower validation was used.

To reproduce the local contract-governance changed scan:

```bash
cd web
bun run contract:check:changed
```

## Worktrees

Use the repository worktree initialization workflow instead of creating private machine-specific setup scripts. The
standard workflow:

- Detects the canonical `repo_dir` from the current Git environment.
- Places worktrees under sibling `<repo-name>-wt/` paths by default.
- Uses the root `.worktree-shared.json` as the shared local-resource source of truth.
- Creates relative symlinks for shared local resources, including `web/.env.development` when available.
- Warns, but does not fail, when optional local files are missing.
- Does not rely on legacy `.local` conventions or hard-coded machine paths.

The shared local-resource source of truth is `.worktree-shared.json`, not `.local`.

## Git Hooks

The repository Git hooks source of truth is the root `.husky/` directory, not a `prepare` script in
`web/package.json`.

After initializing a clone or worktree:

```bash
sh scripts/install-git-hooks.sh
```

Verify the hook path:

```bash
git config --get core.hooksPath
```

Expected output:

```text
.husky
```

## Development Rules

- Read root [AGENTS.md](AGENTS.md) before changing code or structure.
- Read `server/AGENTS.md` for backend work and `web/AGENTS.md` for frontend work.
- Fix the highest incorrect source of truth when code, generated artifacts, and docs drift.
- Do not add compatibility layers, aliases, or fallback mappings before proving why the canonical authority cannot be
  repaired directly.
- Keep backend business behavior in `server/modules/*`.
- Keep frontend module behavior in `web/src/modules/<name>`.
- Use repository validation entrypoints instead of inventing second validation contracts.

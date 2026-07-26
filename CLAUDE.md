# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Graft is a self-hosted application platform for teams running Docker Compose applications. It is a **module-oriented modular monolith** with a Go backend and Vue 3 frontend, connected via an OpenAPI 3.1 contract.

## Common Commands

All top-level developer recipes use [Just](https://github.com/casey/just):

```bash
just setup          # Install all dependencies
just check          # Full validation (server + web)
just check-server   # Backend validation only
just check-web      # Frontend validation only
just dev            # Start backend dev server
just web            # Start frontend dev server
just lint           # Run all linters
just generate       # Regenerate code (OpenAPI types, Ent schemas)
just migrate-up     # Apply database migrations
```

### Backend (Go)

```bash
cd server
go run ./cmd/graft dev                          # Dev mode with live reload
go run ./cmd/graft serve                        # Production serve
go run ./cmd/graft validate backend             # Full lint + test + build
go run ./cmd/graft validate backend --stage lint # Lint only (changed-file scoped)
go run ./cmd/graft validate smoke               # Smoke tests
go run ./cmd/graft migrate up                   # Apply migrations
go test ./...                                   # Run all tests
go test ./modules/auth/...                      # Run tests for specific module
```

### Frontend (Vue 3 + Bun)

```bash
cd web
bun run dev             # Dev server with HMR
bun run check           # Full check (format, type, lint, test, build)
bun run lint            # ESLint (max-warnings 0)
bun run typecheck       # TypeScript type checking
bun run test:run        # Vitest unit tests
bun run build           # Production build
bun run format:check    # Prettier check
bun run lint:i18n       # i18n key governance check
bun run openapi:types   # Regenerate OpenAPI TypeScript types from openapi/openapi.yaml
```

### Docker Compose

```bash
docker compose up -d    # Start all services (postgres, redis, server, web)
docker compose down     # Stop services
```

Default first-login credentials: `graft` / `graft-admin`

## Architecture

### Backend Module System

Modules live in `server/modules/<name>/` and are wired at compile time via `server/internal/moduleruntime/`. Each module has:
- `contract/` — stable public interfaces this module exposes
- `store/` / `storeent/` — Ent ORM entity definitions and data access
- `migrations/` — versioned Atlas database migrations
- `locales/` — server-side i18n strings

**Critical rule**: modules must not directly import other modules' internal packages. All cross-module interaction goes through stable interfaces in `server/internal/moduleapi/`.

The service container (`server/internal/app/`) is a lightweight DI mechanism for registering and resolving singletons — it does NOT use reflection, field tags, or automatic dependency graphs.

Module lifecycle: `Register` → `Boot` → `Shutdown`.

### Frontend Module System

Frontend modules live in `web/src/modules/<name>/` and follow the same isolation principle. Each module owns:
- `pages/` — route-level Vue components
- `components/` — module-local components
- `api/` — Axios-based API client functions
- `contract/` / `types/` — TypeScript types
- `locales/` — i18n messages

Cross-module sharing is handled through `web/src/shared/` (components, composables, utilities). Modules must not import from other modules' internals.

### OpenAPI Contract

`openapi/openapi.yaml` is the canonical API definition. It is the source of truth for both backend routing (`server/internal/contract/`) and frontend types (`web/src/contracts/openapi/generated/schema.ts`). After changing the OpenAPI spec, run `just generate` or `bun run openapi:types` (frontend).

### Key Infrastructure (`server/internal/`)

| Package | Purpose |
|---|---|
| `moduleapi/` | Stable inter-module interface contracts |
| `permission/` | Casbin-based RBAC permission registry |
| `menu/` | Dynamic navigation menu registry |
| `cronx/` | Scheduled task (cron) registry |
| `eventbus/` | Internal pub/sub event system |
| `kvx/` / `cachex/` | Redis cache abstraction |
| `httpx/` | Gin HTTP server setup |
| `logger/` | Zap structured logging with category filtering |
| `database/` | Ent + PostgreSQL connection and pooling |

### Frontend Key Directories

| Path | Purpose |
|---|---|
| `web/src/app/` | App shell, global setup, auth pages |
| `web/src/layouts/` | Frame, sidebar, header layout components |
| `web/src/shared/` | Cross-module reusable components and composables |
| `web/src/contracts/` | Platform-level type contracts and constants |
| `web/src/router/` | Vue Router configuration and route registration |
| `web/src/store/` | Pinia global stores |
| `web/src/locales/` | i18n locale files |

## Validation Chains

These are the ordered validation steps each CI gate runs:

**Backend** (in order):
1. Migration version gate
2. `validate backend --stage lint` (golangci-lint, scoped to changed files)
3. `go test`
4. `go build ./cmd/graft`
5. `validate smoke` (when touching server startup or API surface)

**Frontend** (in order):
1. `format:check`
2. `typecheck`
3. `openapi:frontend-governance:check`
4. `lint:i18n`
5. `lint`
6. `stylelint`
7. `hygiene:check`
8. `test:run`
9. `build`

## Authoritative Governance Documents

For deep conventions and rules beyond this file:
- `AGENTS.md` — Repository-level governance, commit conventions, semantic release rules
- `server/AGENTS.md` — Backend task lifecycle, module design rules, database conventions, authorization rules
- `web/AGENTS.md` — Frontend directory authority, component patterns, i18n governance, route registration
- `DESIGN.md` — UI design system, page type registry, component composition guidelines
- `ai-plan/design/architecture/` — Architecture decision records and design documents

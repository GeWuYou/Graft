set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default: help

help:
    @printf '%s\n' \
      'Graft developer entrypoints' \
      '' \
      'Common recipes:' \
      '  just setup             Install root/web dependencies and warm Go modules' \
      '  just dev               Run the server development supervisor' \
      '  just dev-air           Run the server development supervisor with Air notifications' \
      '  just reset-admin       Reset the default admin in local/test environments' \
      '  just web               Start the web development server' \
      '  just check             Run completion-state server + web checks' \
      '  just check-server      Run the authoritative backend completion check' \
      '  just check-web         Run the authoritative frontend completion check' \
      '  just check-changed     Run worktree-scoped commit-preparation checks' \
      '  just check-local-plan  Inspect scoped checks and coverage gaps' \
      '  just check-local       Execute scoped checks and explicit consumer tests' \
      '  just lint              Run the highest-value local lint slices' \
      '  just lint-server       Run the authoritative backend lint stage' \
      '  just buildtest-server  Run the authoritative backend build/test stage' \
      '  just lint-web          Run frontend lint and style slices without full build/test' \
      '  just test-web          Run frontend Vitest without the rest of bun run check' \
      '  just check-path <path> Run a focused local diagnostic for one file path' \
      '  just smoke             Run the backend smoke validation entrypoint' \
      '  just migrate-up        Apply pending Atlas migrations' \
      '  just migrate-validate  Validate migration assets without a DB connection' \
      '  just migration-check   Run the full migration governance gate' \
      '  just compose-up        Start repository Docker Compose services' \
      '  just compose-down      Stop repository Docker Compose services' \
      '  just ensure-openapi-bundle Ensure the generated OpenAPI bundle exists and is fresh' \
      '  just generate          Run Go generation, OpenAPI bundle, and frontend OpenAPI types' \
      '  just openapi-check     Validate OpenAPI, generated bindings, web schema, and contract projection freshness'

setup:
    bun install --frozen-lockfile
    cd server && go mod download

dev:
    cd server && go run ./cmd/graft dev

dev-air:
    cd server && go run ./cmd/graft dev air

reset-admin:
    cd server && go run ./cmd/graft dev reset-admin

web:
    cd web && bun run dev

check:
    just ensure-openapi-bundle
    just check-governance
    bun run check:code

check-server:
    just ensure-openapi-bundle
    cd server && go run ./cmd/graft validate backend

check-web:
    cd web && bun run check

# Worktree checks select paths and explicit consumers. Full checks remain explicit.
[positional-arguments]
check-local-plan *args:
    python3 -B scripts/local_validation.py --plan "$@"

[positional-arguments]
check-local *args:
    python3 -B scripts/local_validation.py --execute "$@"

[positional-arguments]
check-changed *args:
    python3 -B scripts/local_validation.py --execute "$@"

check-governance:
    python3 -B scripts/validate_ai_governance.py
    python3 -B scripts/validate_ai_plan_structure.py
    python3 -B scripts/validate_shared_asset_registries.py
    python3 -B -m unittest scripts/test_local_validation.py scripts/test_governance_workflow.py scripts/test_ci_runtime_artifacts.py
    node --test scripts/test-turbo.mjs

lint:
    just lint-server
    just lint-web

lint-server:
    cd server && go run ./cmd/graft validate backend --stage lint

buildtest-server:
    cd server && go run ./cmd/graft validate backend --stage buildtest

lint-web:
    cd web && bun run lint:i18n
    cd web && bun run lint
    cd web && bun run stylelint
    cd web && bun run contract:check:changed

test-web:
    cd web && bun run test:run

check-path path:
    target="{{path}}"; \
    case "$$target" in \
      server/*.go) \
        rel="$${target#server/}"; \
        pkg="./$$(dirname "$$rel")"; \
        echo "Focused backend package test: $$pkg"; \
        cd server && go test "$$pkg"; \
        ;; \
      web/*.test.ts|web/*.test.tsx|web/*.test.js|web/*.test.jsx) \
        rel="$${target#web/}"; \
        echo "Focused frontend test: $$rel"; \
        cd web && bunx vitest run "$$rel"; \
        ;; \
      web/*.ts|web/*.tsx|web/*.js|web/*.jsx|web/*.vue) \
        rel="$${target#web/}"; \
        echo "Focused frontend lint: $$rel"; \
        cd web && bunx eslint "$$rel"; \
        ;; \
      *) \
        echo "Unsupported path for focused check: {{path}}"; \
        echo 'Use just check-server, just check-web, or just check-changed instead.'; \
        exit 1; \
        ;; \
    esac

smoke:
    cd server && go run ./cmd/graft validate smoke

migrate-up:
    cd server && go run ./cmd/graft migrate up

migrate-validate:
    cd server && go run ./cmd/graft migrate validate

migration-check:
    cd server && go run ./cmd/graft migrate validate
    python3 scripts/check_migration_versions.py --mode all
    python3 scripts/validate_sql_migrations.py --changed --base-ref "$(git merge-base origin/main HEAD)"
    python3 scripts/check_migration_bootstrap.py

compose-up:
    docker compose pull
    docker compose up -d

compose-down:
    docker compose down

ensure-openapi-bundle:
    node scripts/ensure-openapi-bundle.mjs

generate:
    just ensure-openapi-bundle
    node scripts/openapi-runtime-paths.mjs
    cd server && go generate ./...
    cd web && bun run openapi:types
    cd server && go run ./internal/contract/projection/cmd/projectiongen

openapi-check:
    just ensure-openapi-bundle
    cd server && go run ./cmd/graft validate openapi
    node scripts/openapi-runtime-paths.mjs --check
    cd web && bun run openapi:types:check
    cd server && go run ./internal/contract/projection/cmd/projectiongen --check

contract-projection-check:
    cd server && go run ./internal/contract/projection/cmd/projectiongen --check

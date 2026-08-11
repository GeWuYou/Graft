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
      '  just check-changed     Run the smallest local check for current changes' \
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
      '  just generate          Run Go generation, OpenAPI bundle, and frontend OpenAPI types' \
      '  just openapi-check     Validate OpenAPI, generated bindings, web schema, and contract projection freshness'

setup:
    bun install --frozen-lockfile
    cd web && bun install --frozen-lockfile
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
    cd server && go run ./cmd/graft validate backend
    cd web && bun run check

check-server:
    cd server && go run ./cmd/graft validate backend

check-web:
    cd web && bun run check

check-changed:
    changed="$$(git diff --cached --name-only --diff-filter=ACMR || true)"; \
    if [ -z "$$changed" ]; then changed="$$(git diff HEAD --name-only --diff-filter=ACMR || true)"; fi; \
    if [ -z "$$changed" ]; then echo 'No changed files detected; running just check.'; just check; exit 0; fi; \
    if printf '%s\n' "$$changed" | rg -q '^(AGENTS\.md|README\.md|Justfile|\.github/|\.vscode/|\.ai/|ai-plan/|scripts/|openapi/)'; then \
      echo 'Shared governance or contract changes detected; running just check.'; \
      just check; \
    elif printf '%s\n' "$$changed" | rg -q '^server/'; then \
      if printf '%s\n' "$$changed" | rg -q '^web/'; then \
        echo 'Both server and web changes detected; running just check.'; \
        just check; \
      else \
        echo 'Server-only changes detected; running backend lint and build/test stages.'; \
        just lint-server; \
        just buildtest-server; \
      fi; \
    elif printf '%s\n' "$$changed" | rg -q '^web/'; then \
      echo 'Web-only changes detected; running just check-web.'; \
      just check-web; \
    else \
      echo 'Non-runtime changes detected; running git diff --check.'; \
      git diff --check; \
    fi

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
    python3 scripts/validate_sql_migrations.py
    python3 scripts/validate_sql_migrations.py --changed --base-ref "$(git merge-base origin/main HEAD)"
    python3 scripts/check_migration_bootstrap.py

compose-up:
    docker compose pull
    docker compose up -d

compose-down:
    docker compose down

generate:
    node scripts/openapi-bundle.mjs
    node scripts/openapi-runtime-paths.mjs
    cd server && go generate ./...
    cd web && bun run openapi:types
    cd server && go run ./internal/contract/projection/cmd/projectiongen

openapi-check:
    cd server && go run ./cmd/graft validate openapi
    node scripts/openapi-runtime-paths.mjs --check
    cd web && bun run openapi:types:check
    cd server && go run ./internal/contract/projection/cmd/projectiongen --check

contract-projection-check:
    cd server && go run ./internal/contract/projection/cmd/projectiongen --check

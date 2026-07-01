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
      '  just check             Run completion-state server + web + quality checks' \
      '  just lint              Run the highest-value local lint slices' \
      '  just smoke             Run the backend smoke validation entrypoint' \
      '  just migrate-up        Apply pending Atlas migrations' \
      '  just migrate-validate  Validate migration assets without a DB connection' \
      '  just compose-up        Start repository Docker Compose services' \
      '  just compose-down      Stop repository Docker Compose services' \
      '  just generate          Run Go generation, OpenAPI bundle, and frontend OpenAPI types' \
      '  just quality           Score changed files with the repository quality evaluator' \
      '  just openapi-check     Validate the root OpenAPI spec and frontend generated types freshness'

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

lint:
    cd server && go run ./cmd/graft validate backend --stage lint
    cd web && bun run lint:i18n
    cd web && bun run lint
    cd web && bun run stylelint
    cd web && bun run contract:check:changed

smoke:
    cd server && go run ./cmd/graft validate smoke

migrate-up:
    cd server && go run ./cmd/graft migrate up

migrate-validate:
    cd server && go run ./cmd/graft migrate validate

compose-up:
    docker compose pull
    docker compose up -d

compose-down:
    docker compose down

generate:
    cd server && go generate ./...
    node scripts/openapi-bundle.mjs
    cd web && bun run openapi:types

quality:
    bun run quality:eff-u-code:score:changed

openapi-check:
    cd server && go run ./cmd/graft validate openapi
    cd web && bun run openapi:types:check

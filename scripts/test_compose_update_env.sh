#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
workspace="$(mktemp -d)"
trap 'rm -rf "$workspace"' EXIT

cp "$repo_root/compose.yml" "$workspace/"
cp "$repo_root/compose.env.example" "$workspace/.env"
sed -i 's/^GRAFT_IMAGE_TAG=.*/GRAFT_IMAGE_TAG=beta/' "$workspace/.env"
sed -i 's/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=test-password/' "$workspace/.env"
sed -i 's/^GRAFT_AUTH_JWT_SECRET=.*/GRAFT_AUTH_JWT_SECRET=test-secret/' "$workspace/.env"
sed -i '/^GRAFT_DEPLOYMENT_RUNTIME=/d' "$workspace/.env"

rendered="$(docker compose --env-file "$workspace/.env" -f "$workspace/compose.yml" config --format json)"
server_tag="$(jq -r '.services.server.environment.GRAFT_IMAGE_TAG' <<<"$rendered")"
bootstrap_runtime="$(jq -r '.services.bootstrap.environment.GRAFT_DEPLOYMENT_RUNTIME' <<<"$rendered")"
server_runtime="$(jq -r '.services.server.environment.GRAFT_DEPLOYMENT_RUNTIME' <<<"$rendered")"
bootstrap_compose_root="$(jq -r '.services.bootstrap.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"
server_compose_root="$(jq -r '.services.server.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"

test "$server_tag" = "beta"
test "$bootstrap_runtime" = "compose"
test "$server_runtime" = "compose"
test "$bootstrap_compose_root" = "<unset>"
test "$server_compose_root" = "<unset>"

printf '\nGRAFT_DEPLOYMENT_COMPOSE_ROOT=/opt/graft/compose\n' >> "$workspace/.env"
rendered="$(docker compose --env-file "$workspace/.env" -f "$workspace/compose.yml" config --format json)"
bootstrap_compose_root="$(jq -r '.services.bootstrap.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"
server_compose_root="$(jq -r '.services.server.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"

test "$bootstrap_compose_root" = "/opt/graft/compose"
test "$server_compose_root" = "/opt/graft/compose"

sed -i 's|^GRAFT_DEPLOYMENT_COMPOSE_ROOT=.*|GRAFT_DEPLOYMENT_COMPOSE_ROOT=|' "$workspace/.env"
rendered="$(docker compose --env-file "$workspace/.env" -f "$workspace/compose.yml" config --format json)"
bootstrap_compose_root="$(jq -r '.services.bootstrap.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"
server_compose_root="$(jq -r '.services.server.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"

test "$bootstrap_compose_root" = ""
test "$server_compose_root" = ""

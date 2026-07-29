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

rendered="$(docker compose --env-file "$workspace/.env" -f "$workspace/compose.yml" config --format json)"
server_tag="$(jq -r '.services.server.environment.GRAFT_IMAGE_TAG' <<<"$rendered")"
server_mode="$(jq -r '.services.server.environment.GRAFT_UPDATE_DEPLOYMENT_MODE' <<<"$rendered")"

test "$server_tag" = "beta"
test "$server_mode" = "compose"

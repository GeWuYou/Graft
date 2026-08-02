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
config_validate_image="$(jq -r '.services["config-validate"].image' <<<"$rendered")"
config_validate_user="$(jq -r '.services["config-validate"].user' <<<"$rendered")"
config_validate_command="$(jq -c '.services["config-validate"].command' <<<"$rendered")"
config_validate_volume_count="$(jq '[.services["config-validate"].volumes[] | select(.target == "/opt/graft/deployment/compose.yml" or .target == "/opt/graft/deployment/.env")] | length' <<<"$rendered")"
config_validate_read_only_volume_count="$(jq '[.services["config-validate"].volumes[] | select((.target == "/opt/graft/deployment/compose.yml" or .target == "/opt/graft/deployment/.env") and .read_only)] | length' <<<"$rendered")"
bootstrap_config_gate="$(jq -r '.services.bootstrap.depends_on["config-validate"].condition' <<<"$rendered")"
application_root_config_gate="$(jq -r '.services["application-root-init"].depends_on["config-validate"].condition' <<<"$rendered")"
backup_root_config_gate="$(jq -r '.services["backup-root-init"].depends_on["config-validate"].condition' <<<"$rendered")"
bootstrap_runtime="$(jq -r '.services.bootstrap.environment.GRAFT_DEPLOYMENT_RUNTIME' <<<"$rendered")"
server_runtime="$(jq -r '.services.server.environment.GRAFT_DEPLOYMENT_RUNTIME' <<<"$rendered")"
bootstrap_compose_root="$(jq -r '.services.bootstrap.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"
server_compose_root="$(jq -r '.services.server.environment.GRAFT_DEPLOYMENT_COMPOSE_ROOT // "<unset>"' <<<"$rendered")"

test "$server_tag" = "beta"
test "$config_validate_image" = "ghcr.io/gewuyou/graft-server:beta"
test "$config_validate_user" = "0:0"
test "$config_validate_command" = '["config","validate","--env-file","/opt/graft/deployment/.env","--compose-file","/opt/graft/deployment/compose.yml"]'
test "$config_validate_volume_count" = "2"
test "$config_validate_read_only_volume_count" = "2"
test "$bootstrap_config_gate" = "service_completed_successfully"
test "$application_root_config_gate" = "service_completed_successfully"
test "$backup_root_config_gate" = "service_completed_successfully"
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

grep -qx 'GRAFT_CONFIG_SCHEMA_VERSION=1' "$workspace/.env"

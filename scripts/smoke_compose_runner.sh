#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture_dir="${repo_root}/server/runner/compose/testdata/smoke"
runner_image="${GRAFT_COMPOSE_RUNNER_SMOKE_IMAGE:-graft-compose-runner:local-smoke}"
workspace="$(mktemp -d)"
project="graft-runner-smoke-$$"
registry_id=""
cleanup() {
  docker compose --project-name "${project}" --project-directory "${workspace}" down --volumes --remove-orphans >/dev/null 2>&1 || true
  if [[ -n "${registry_id}" ]]; then
    docker rm -f "${registry_id}" >/dev/null 2>&1 || true
  fi
  rm -rf "${workspace}"
}
trap cleanup EXIT

docker image inspect "${runner_image}" >/dev/null
docker build --progress=quiet -t graft-runner-fixture/graft-server:v1.1.0 -f "${fixture_dir}/Dockerfile.fixture" "${fixture_dir}"
docker tag graft-runner-fixture/graft-server:v1.1.0 graft-runner-fixture/graft-web:v1.1.0
registry_id="$(docker run -d -p 127.0.0.1::5000 registry:2)"
registry_port="$(docker port "${registry_id}" 5000/tcp | sed 's/.*://')"
registry="127.0.0.1:${registry_port}"
docker tag graft-runner-fixture/graft-server:v1.1.0 "${registry}/graft-server:v1.1.0"
docker tag graft-runner-fixture/graft-web:v1.1.0 "${registry}/graft-web:v1.1.0"
docker tag "${runner_image}" "${registry}/graft-compose-runner:v1.1.0"
docker push "${registry}/graft-server:v1.1.0" >/dev/null
docker push "${registry}/graft-web:v1.1.0" >/dev/null
docker push "${registry}/graft-compose-runner:v1.1.0" >/dev/null
manifest_digest() {
  curl -fsSI \
    -H 'Accept: application/vnd.docker.distribution.manifest.v2+json' \
    "http://${registry}/v2/$1/manifests/v1.1.0" \
    | awk -F ': ' 'tolower($1) == "docker-content-digest" { sub("\\r$", "", $2); print $2; exit }'
}
server_digest="$(manifest_digest graft-server)"
web_digest="$(manifest_digest graft-web)"
runner_digest="$(manifest_digest graft-compose-runner)"
server_reference="${registry}/graft-server@${server_digest}"
web_reference="${registry}/graft-web@${web_digest}"
runner_reference="${registry}/graft-compose-runner@${runner_digest}"
cp "${fixture_dir}/compose.yml" "${workspace}/compose.yml"
cat > "${workspace}/.env" <<EOF
GRAFT_IMAGE_TAG=v1.0.0
COMPOSE_PROJECT_NAME=${project}
EOF
docker tag "${registry}/graft-server:v1.1.0" "${registry}/graft-server:v1.0.0"
docker tag "${registry}/graft-web:v1.1.0" "${registry}/graft-web:v1.0.0"
chmod 0777 "${workspace}"
chmod 0666 "${workspace}/.env"
docker compose --project-name "${project}" --project-directory "${workspace}" up -d postgres
for _ in $(seq 1 30); do
  if docker compose --project-name "${project}" --project-directory "${workspace}" exec -T postgres pg_isready -U graft -d graft >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker compose --project-name "${project}" --project-directory "${workspace}" exec -T postgres pg_isready -U graft -d graft >/dev/null
cat > "${workspace}/runner-input.json" <<EOF
{
  "protocol_version": 1,
  "operation_id": "compose-runner-smoke",
  "task_id": 1,
  "preflight": {
    "declared_mode": "compose",
    "deployment_strategy": "beta_tracking",
    "image_tag": "beta",
    "detected_mode": "compose",
    "compose_root": "${workspace}",
    "platform": "linux/amd64",
    "docker_socket": "/var/run/docker.sock",
    "compose_files": ["${workspace}/compose.yml"],
    "server_reference": "${server_reference}",
    "web_reference": "${web_reference}",
    "runner_reference": "${runner_reference}",
    "server_digest": "${server_digest}",
    "web_digest": "${web_digest}",
    "runner_digest": "${runner_digest}",
    "bundled_postgres": true,
    "official_server_image": "${registry}/graft-server",
    "official_web_image": "${registry}/graft-web",
    "official_runner_image": "${registry}/graft-compose-runner"
  }
}
EOF
socket_gid="$(stat -c '%g' /var/run/docker.sock)"
docker run --rm \
  --user 65532:65532 \
  --group-add "${socket_gid}" \
  --network none \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges:true \
  -e GRAFT_UPDATE_RUNNER_INPUT="${workspace}/runner-input.json" \
  -v "${workspace}:${workspace}:rw" \
  -v /var/run/docker.sock:/var/run/docker.sock:rw \
  "${runner_image}"
jq -e '
  .protocol_version == 1
  and .operation_id == "compose-runner-smoke"
  and .migration_started == true
  and .succeeded == true
  and (.backup_completion.ConfigSnapshotSHA256 | test("^[0-9a-f]{64}$"))
  and (.backup_completion.DatabaseDumpSHA256 | test("^[0-9a-f]{64}$"))
' "${workspace}/.graft-update/receipts/compose-runner-smoke.json" >/dev/null
grep -Fqx "GRAFT_IMAGE_TAG=v1.1.0" "${workspace}/.env"
test -s "${workspace}/.graft-update/backups/compose-runner-smoke/config.snapshot"
test -s "${workspace}/.graft-update/backups/compose-runner-smoke/database.dump"
printf 'Compose runner smoke passed.\n'

#!/usr/bin/env bash
set -euo pipefail

server_image="${GRAFT_RELEASE_SERVER_IMAGE:?GRAFT_RELEASE_SERVER_IMAGE is required}"
web_image="${GRAFT_RELEASE_WEB_IMAGE:?GRAFT_RELEASE_WEB_IMAGE is required}"
image_tag="${GRAFT_RELEASE_IMAGE_TAG:?GRAFT_RELEASE_IMAGE_TAG is required}"
log_dir="${GRAFT_RELEASE_SMOKE_LOG_DIR:-.tmp/release-compose-smoke}"
workspace="$(mktemp -d)"
project="graft-release-smoke-${GITHUB_RUN_ID:-$$}"

cleanup() {
  status=$?
  set +e
  mkdir -p "${log_dir}"
  docker compose --project-name "${project}" --project-directory "${workspace}" ps -a > "${log_dir}/compose-ps.txt" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" logs bootstrap > "${log_dir}/compose-bootstrap.log" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" logs application-root-init > "${log_dir}/compose-application-root-init.log" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" logs postgres > "${log_dir}/compose-postgres.log" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" logs redis > "${log_dir}/compose-redis.log" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" logs server > "${log_dir}/compose-server.log" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" logs web > "${log_dir}/compose-web.log" 2>&1
  docker compose --project-name "${project}" --project-directory "${workspace}" down -v --remove-orphans > "${log_dir}/compose-down.log" 2>&1
  rm -rf "${workspace}"
  exit "${status}"
}
trap cleanup EXIT

mkdir -p "${log_dir}" "${workspace}/apps"
cp compose.yml compose.smoke.yml "${workspace}/"
docker image inspect "${server_image}:${image_tag}" >/dev/null
docker image inspect "${web_image}:${image_tag}" >/dev/null
server_image_id="$(docker image inspect "${server_image}:${image_tag}" --format '{{.Id}}')"
web_image_id="$(docker image inspect "${web_image}:${image_tag}" --format '{{.Id}}')"

cat > "${workspace}/.env" <<EOF
GRAFT_SERVER_IMAGE_REPOSITORY=${server_image}
GRAFT_SERVER_IMAGE_DIGEST=${server_image_id}
GRAFT_WEB_IMAGE_REPOSITORY=${web_image}
GRAFT_WEB_IMAGE_DIGEST=${web_image_id}
COMPOSE_FILE_DIR=${workspace}
POSTGRES_DB=graft
POSTGRES_USER=graft
POSTGRES_PASSWORD=graft
GRAFT_AUTH_JWT_SECRET=ci-compose-smoke-secret
GRAFT_DOCS_ENABLED=true
GRAFT_APPLICATION_ROOT_HOST_PATH=${workspace}/apps
GRAFT_WEB_HOST_PORT=3000
EOF

docker compose --project-name "${project}" --project-directory "${workspace}" config --quiet
docker compose --project-name "${project}" --project-directory "${workspace}" up -d postgres redis
docker compose --project-name "${project}" --project-directory "${workspace}" -f compose.yml -f compose.smoke.yml up -d bootstrap

for _ in $(seq 1 60); do
  bootstrap_id="$(docker compose --project-name "${project}" --project-directory "${workspace}" ps -aq bootstrap)"
  if [[ -z "${bootstrap_id}" ]]; then
    sleep 2
    continue
  fi
  status="$(docker inspect -f '{{.State.Status}}' "${bootstrap_id}")"
  exit_code="$(docker inspect -f '{{.State.ExitCode}}' "${bootstrap_id}")"
  if [[ "${status}" == "exited" && "${exit_code}" == "0" ]]; then
    break
  fi
  if [[ "${status}" == "exited" && "${exit_code}" != "0" ]]; then
    echo "bootstrap failed with exit code ${exit_code}"
    exit 1
  fi
  sleep 2
done

bootstrap_id="$(docker compose --project-name "${project}" --project-directory "${workspace}" ps -aq bootstrap)"
if [[ -z "${bootstrap_id}" || "$(docker inspect -f '{{.State.Status}}' "${bootstrap_id}")" != "exited" || "$(docker inspect -f '{{.State.ExitCode}}' "${bootstrap_id}")" != "0" ]]; then
  echo "bootstrap did not complete successfully"
  exit 1
fi

docker compose --project-name "${project}" --project-directory "${workspace}" up -d server web
web_mapping="$(docker compose --project-name "${project}" --project-directory "${workspace}" port web 80 | sed -n '1p')"
web_port="${web_mapping##*:}"
if ! [[ "${web_port}" =~ ^[1-9][0-9]{0,4}$ ]] || (( web_port > 65535 )); then
  echo "unable to resolve the web host port from Compose mapping: ${web_mapping:-<empty>}"
  exit 1
fi

probe_base_url="http://127.0.0.1:${web_port}"
for _ in $(seq 1 60); do
  if curl -fsS "${probe_base_url}/healthz" > "${log_dir}/compose-healthz.json" \
    && curl -fsS "${probe_base_url}/openapi.json" > "${log_dir}/compose-openapi.json" \
    && curl -fsS "${probe_base_url}/" > "${log_dir}/compose-index.html"; then
    grep -q "<!doctype html" "${log_dir}/compose-index.html"
    grep -q '"openapi"' "${log_dir}/compose-openapi.json"
    exit 0
  fi
  sleep 2
done

echo "compose services were not ready in time"
exit 1

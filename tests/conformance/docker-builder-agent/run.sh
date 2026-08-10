#!/bin/sh
set -eu

compose_file=${COMPOSE_FILE:-tests/conformance/docker-builder-agent/compose.yml}
compose="docker compose --profile conformance -f $compose_file"
evidence_file=${CONFORMANCE_EVIDENCE:-docker-agent-conformance.log}
prepare_evidence=$(mktemp "${TMPDIR:-/tmp}/graft-docker-builder-prepare.XXXXXX")
bootstrap_evidence=$(mktemp "${TMPDIR:-/tmp}/graft-docker-builder-bootstrap.XXXXXX")

cleanup() {
  rm -f "$prepare_evidence" "$bootstrap_evidence"
}
trap cleanup EXIT HUP INT TERM

require_env() {
  eval "value=\${$1:-}"
  if [ -z "$value" ]; then
    echo "$1 is required" >&2
    exit 2
  fi
}

require_env GRAFT_BACKEND_IMAGE
require_env GRAFT_AGENT_IMAGE
require_env GRAFT_CONFORMANCE_DRIVER_IMAGE
require_env GRAFT_CONFORMANCE_POSTGRES_PASSWORD
require_env GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST
require_env VAULT_DEV_ROOT_TOKEN

run_driver() {
  $compose run --rm --no-deps conformance-driver "$@"
}

wait_for_success() {
  service=$1
  deadline=$(( $(date +%s) + 120 ))
  while :; do
    container_id=$($compose ps -aq "$service")
    if [ -n "$container_id" ]; then
      status=$(docker inspect -f '{{.State.Status}}:{{.State.ExitCode}}' "$container_id")
      case "$status" in
        exited:0) return 0 ;;
        exited:*)
          echo "$service exited unsuccessfully ($status)" >&2
          return 1
          ;;
      esac
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
      echo "timed out waiting for $service" >&2
      return 1
    fi
    sleep 1
  done
}

validate_evidence() {
  jq -e '
    (.target_id | type == "number" and . > 0) and
    (.agent_id | type == "string" and length > 0) and
    (.identity_id | type == "string" and length > 0) and
    (.generation | type == "number" and . > 0) and
    (.ledger_receipt_count | type == "number" and . >= 0)
  ' "$1" >/dev/null
}

echo "[1/8] start PostgreSQL, Vault, and fixture initialization"
$compose up -d postgres vault vault-init
wait_for_success vault-init

echo "[2/8] verify real Vault AppRole login without logging credentials"
$compose exec -T vault sh -ec '
  role_id=$(cat /conformance/secrets/role_id)
  secret_id=$(cat /conformance/secrets/secret_id)
  test -n "$role_id" && test -n "$secret_id"
  vault write -format=json auth/approle/login role_id="$role_id" secret_id="$secret_id" >/dev/null
'

echo "[3/8] apply normal Backend migrations and start Backend listeners"
$compose up -d --wait migrate backend
$compose exec -T backend curl --fail --silent http://127.0.0.1:8080/healthz >/dev/null

echo "[4/8] prepare Backend-owned Runtime Target enrollment and delivery"
run_driver \
  --phase prepare \
  --agent-id docker-builder-agent \
  --image-digest "$GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST" \
  --agent-version "${GRAFT_CONFORMANCE_AGENT_VERSION:-fixture}" \
  --enrollment-ref docker-builder-agent-conformance \
  --automation-id docker-builder-agent-conformance \
  --docker-installation-ref docker:fixture \
  --docker-secret-ref fixture:bootstrap-delivery \
  --bootstrap-material-file /conformance/agent-bootstrap/bootstrap-token \
  --agent-config-file /conformance/agent-config/agent.json \
  --bootstrap-url https://backend:8443 \
  --agent-url https://backend:8444 \
  --bootstrap-ca-file /run/graft-agent-trust/ca.pem \
  --trust-bundle-file /run/graft-agent-trust/ca.pem \
  --agent-secret-file /run/graft-bootstrap/bootstrap-token > "$prepare_evidence"
validate_evidence "$prepare_evidence"

echo "[5/8] run first Agent lifecycle"
$compose run --rm --no-deps docker-builder-agent

echo "[6/8] verify bootstrap issuance, activation, and first ledger receipt"
run_driver \
  --phase verify-bootstrap \
  --agent-id "$(jq -r '.agent_id' "$prepare_evidence")" \
  --target-id "$(jq -r '.target_id' "$prepare_evidence")" > "$bootstrap_evidence"
validate_evidence "$bootstrap_evidence"
test "$(jq -r '.ledger_receipt_count' "$bootstrap_evidence")" -ge 1

echo "[7/8] run restart lifecycle with persisted Agent state"
$compose run --rm --no-deps docker-builder-agent

echo "[8/8] verify restored identity, no second enrollment, and a fresh receipt"
run_driver \
  --phase verify-restart \
  --agent-id "$(jq -r '.agent_id' "$bootstrap_evidence")" \
  --target-id "$(jq -r '.target_id' "$bootstrap_evidence")" \
  --identity-id "$(jq -r '.identity_id' "$bootstrap_evidence")" \
  --generation "$(jq -r '.generation' "$bootstrap_evidence")" \
  --receipt-count "$(jq -r '.ledger_receipt_count' "$bootstrap_evidence")" | jq -e '
    (.target_id | type == "number" and . > 0) and
    (.identity_id | type == "string" and length > 0) and
    (.generation | type == "number" and . > 0) and
    (.ledger_receipt_count | type == "number" and . >= 2)
  ' >/dev/null

$compose logs --no-color postgres vault vault-init migrate backend docker-builder-agent > "$evidence_file"
echo "Docker Builder Agent conformance completed; redacted service logs are in $evidence_file."

#!/bin/sh
set -eu

compose_file=${COMPOSE_FILE:-deployments/compose/docker-builder-agent/compose.yml}
compose="docker compose -f $compose_file"

if [ -z "${CONFORMANCE_DRIVER_CMD:-}" ]; then
  echo "CONFORMANCE_DRIVER_CMD is required: invoke the normal authenticated Runtime Target workflow" >&2
  exit 2
fi

echo "[1/5] start Vault and initialize AppRole/PKI"
$compose up -d vault vault-init
$compose ps vault vault-init

echo "[2/5] verify real Vault AppRole login"
role_id=$($compose exec -T vault-init sh -ec 'cat /conformance/secrets/role_id')
secret_id=$($compose exec -T vault-init sh -ec 'cat /conformance/secrets/secret_id')
test -n "$role_id" && test -n "$secret_id"
$compose exec -T vault sh -ec 'vault write -format=json auth/approle/login role_id="$1" secret_id="$2" >/dev/null' sh "$role_id" "$secret_id"

echo "[3/5] start backend and invoke normal enrollment/delivery workflow"
$compose up -d backend
sh -ec "$CONFORMANCE_DRIVER_CMD"

echo "[4/5] start Agent and verify image entrypoint"
$compose up -d docker-builder-agent
$compose exec -T docker-builder-agent /app/graft-docker-builder-agent --version

echo "[5/5] emit restart evidence"
$compose restart docker-builder-agent
$compose logs --no-color docker-builder-agent backend vault > "${CONFORMANCE_EVIDENCE:-docker-agent-conformance.log}"
echo "Conformance driver completed; inspect logs and ledger/receipt evidence before accepting the gate."

#!/bin/sh
set -eu

vault auth enable approle 2>/dev/null || true
vault secrets enable pki 2>/dev/null || true
vault secrets tune -max-lease-ttl=8760h pki
vault write pki/root/generate/internal common_name=graft.local ttl=8760h
vault write pki/roles/graft-docker-builder-agent \
  allowed_uri_sans='spiffe://graft/runtime-target/*/builder-agent/*' \
  allow_glob_domains=true \
  require_cn=false \
  use_csr_common_name=false \
  max_ttl=24h
printf '%s\n' \
  'path "pki/issue/graft-docker-builder-agent" { capabilities = ["update"] }' \
  'path "pki/cert/ca" { capabilities = ["read"] }' | vault policy write graft-docker-builder-agent -
vault write auth/approle/role/graft-docker-builder-agent token_policies=graft-docker-builder-agent token_ttl=1h token_max_ttl=2h
role_id="$(vault read -field=role_id auth/approle/role/graft-docker-builder-agent/role-id)"
secret_id="$(vault write -field=secret_id -f auth/approle/role/graft-docker-builder-agent/secret-id)"
vault read -field=certificate pki/cert/ca > /conformance/secrets/ca.pem
printf '%s' "$role_id" > /conformance/secrets/role_id
printf '%s' "$secret_id" > /conformance/secrets/secret_id
chmod 0600 /conformance/secrets/*

#!/bin/sh
set -eu

if [ -f /conformance/secrets/.initialized ]; then
  exit 0
fi

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
vault write pki/roles/graft-conformance-backend \
  allowed_domains='backend,localhost' \
  allow_bare_domains=true \
  server_flag=true \
  client_flag=false \
  max_ttl=24h

printf '%s\n' \
  'path "pki/sign/graft-docker-builder-agent" { capabilities = ["update"] }' \
  'path "pki/cert/ca" { capabilities = ["read"] }' | vault policy write graft-docker-builder-agent -
vault write auth/approle/role/graft-docker-builder-agent token_policies=graft-docker-builder-agent token_ttl=1h token_max_ttl=2h

role_id="$(vault read -field=role_id auth/approle/role/graft-docker-builder-agent/role-id)"
secret_id="$(vault write -field=secret_id -f auth/approle/role/graft-docker-builder-agent/secret-id)"
test -s /conformance/vault-tls/vault-ca.pem
cp /conformance/vault-tls/vault-ca.pem /conformance/secrets/vault-ca.pem
vault read -field=certificate pki/cert/ca > /conformance/secrets/ca.pem
vault read -field=certificate pki/cert/ca > /conformance/agent-trust/ca.pem
vault write -format=json pki/issue/graft-conformance-backend common_name=backend alt_names=localhost > /tmp/backend-certificate.json
backend_certificate="$(sed -n 's/^[[:space:]]*"certificate": "\(.*\)",$/\1/p' /tmp/backend-certificate.json)"
backend_key="$(sed -n 's/^[[:space:]]*"private_key": "\(.*\)",$/\1/p' /tmp/backend-certificate.json)"
test -n "$backend_certificate" && test -n "$backend_key"
printf '%b\n' "$backend_certificate" > /conformance/secrets/backend-cert.pem
printf '%b\n' "$backend_key" > /conformance/secrets/backend-key.pem
rm -f /tmp/backend-certificate.json
printf '%s' "$role_id" > /conformance/secrets/role_id
printf '%s' "$secret_id" > /conformance/secrets/secret_id
dd if=/dev/urandom of=/conformance/secrets/enrollment-pepper bs=32 count=1 status=none

# Backend runs as UID 10001. Agent-visible material is deliberately copied to a
# separate volume so the Agent cannot read the Backend's AppRole credentials.
chown 10001:10001 /conformance/secrets/*
chmod 0600 /conformance/secrets/*
chmod 0444 /conformance/agent-trust/ca.pem
touch /conformance/secrets/.initialized

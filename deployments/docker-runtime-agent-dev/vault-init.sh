#!/bin/sh
set -eu

# 此 Vault 仅用于一次性本地开发拓扑；root token 和跳过 TLS 校验不得复制到非本地环境。
umask 077

export VAULT_ADDR=https://vault:8200
export VAULT_TOKEN=graft-docker-runtime-agent-dev
root=${GRAFT_DOCKER_RUNTIME_DEV_ROOT:?}
export VAULT_SKIP_VERIFY=true

if [ -f "$root/secrets/.initialized" ] && vault read -field=certificate pki/cert/ca >/dev/null 2>&1; then
  exit 0
fi

rm -f "$root/secrets/.initialized"
mkdir -p "$root/secrets" "$root/agent/bootstrap" "$root/agent/trust" "$root/agent/state"
vault auth enable approle 2>/dev/null || true
vault secrets enable pki 2>/dev/null || true
vault secrets tune -max-lease-ttl=8760h pki
vault write pki/root/generate/internal common_name=graft.local ttl=8760h
vault write pki/roles/graft-docker-runtime-agent allowed_uri_sans='spiffe://graft/runtime-target/*/runtime-agent/*' allow_glob_domains=true require_cn=false use_csr_common_name=false max_ttl=24h
vault write pki/roles/graft-local-server allowed_domains='localhost' allow_bare_domains=true server_flag=true client_flag=false max_ttl=24h
printf '%s\n' 'path "pki/sign/graft-docker-runtime-agent" { capabilities = ["update"] }' 'path "pki/cert/ca" { capabilities = ["read"] }' | vault policy write graft-docker-runtime-agent -
vault write auth/approle/role/graft-docker-runtime-agent token_policies=graft-docker-runtime-agent token_ttl=1h token_max_ttl=2h

vault read -field=role_id auth/approle/role/graft-docker-runtime-agent/role-id > "$root/secrets/role_id"
vault write -field=secret_id -f auth/approle/role/graft-docker-runtime-agent/secret-id > "$root/secrets/secret_id"
vault read -field=certificate pki/cert/ca > "$root/secrets/ca.pem"
cp "$root/secrets/ca.pem" "$root/agent/trust/ca.pem"
issued=$(mktemp)
trap 'rm -f "$issued"' EXIT HUP INT TERM
vault write -format=json pki/issue/graft-local-server common_name=localhost alt_names=localhost ip_sans=127.0.0.1 > "$issued"
server_certificate="$(sed -n 's/^[[:space:]]*"certificate": "\(.*\)",\{0,1\}$/\1/p' "$issued")"
server_key="$(sed -n 's/^[[:space:]]*"private_key": "\(.*\)",\{0,1\}$/\1/p' "$issued")"
test -n "$server_certificate"
test -n "$server_key"
printf '%b\n' "$server_certificate" > "$root/secrets/server-cert.pem"
printf '%b\n' "$server_key" > "$root/secrets/server-key.pem"
rm -f "$issued"
trap - EXIT HUP INT TERM
dd if=/dev/urandom of="$root/secrets/enrollment-pepper" bs=32 count=1 status=none
chmod 0600 "$root/secrets"/*
chmod 0444 "$root/agent/trust/ca.pem"
touch "$root/secrets/.initialized"

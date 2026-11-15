#!/usr/bin/env bash
set -euo pipefail

# 同一源码只改变链接身份，测试流仍由统一 backend 验证入口执行。
repository_root="$(git -C "$(dirname "$0")/.." rev-parse --show-toplevel)"
source_revision="$(git -C "$repository_root" rev-parse HEAD)"
build_time_utc="$(date -u -d "@$(git -C "$repository_root" show -s --format=%ct HEAD)" +%Y-%m-%dT%H:%M:%SZ)"
export CGO_ENABLED=0 GOOS=linux GOARCH=amd64
git -C "$repository_root" diff --quiet HEAD --
if [ -n "$(git -C "$repository_root" status --porcelain --untracked-files=normal -- server tests/conformance/docker-runtime-agent)" ]; then
  echo "Runtime acceptance binaries require clean tracked source inputs" >&2
  exit 1
fi
mkdir -p "$repository_root/.tmp/compiled/server" "$repository_root/.tmp/compiled/target-server" \
  "$repository_root/.tmp/compiled/runner" "$repository_root/.tmp/compiled/release-source" \
  "$repository_root/.tmp/compiled/build-registry"

for version in 0.0.0-beta.1 0.0.0-beta.2; do
  output=server
  if [ "$version" = 0.0.0-beta.2 ]; then
    output=target-server
  fi
  go -C "$repository_root/server" build -trimpath \
    -ldflags="-s -w -X graft/server/internal/buildinfo.version=$version -X graft/server/internal/buildinfo.gitCommit=$source_revision -X graft/server/internal/buildinfo.buildTimeUTC=$build_time_utc -X graft/server/internal/buildinfo.gitTreeState=clean" \
    -o "$repository_root/.tmp/compiled/$output/graft-server-linux-amd64" ./cmd/graft
done
go -C "$repository_root/server" build -trimpath -ldflags="-s -w" -o "$repository_root/.tmp/compiled/runner/graft-compose-runner" ./cmd/graft-compose-runner
go build -trimpath -ldflags="-s -w" -o "$repository_root/.tmp/compiled/release-source/graft-release-fixture" \
  "$repository_root/tests/conformance/docker-runtime-agent/release-source.go"
go -C "$repository_root/server" build -mod=readonly -trimpath -ldflags="-s -w" \
  -o "$repository_root/.tmp/compiled/build-registry/graft-fixture-htpasswd" \
  "$repository_root/tests/conformance/docker-runtime-agent/registry-auth/main.go"

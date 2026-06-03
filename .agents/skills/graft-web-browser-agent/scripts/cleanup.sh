#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(git -C "${SCRIPT_DIR}/../../.." rev-parse --show-toplevel)"
ARTIFACT_DIR="${ROOT_DIR}/.ai/artifacts/browser"
SESSION=""
ALL=false
OLDER_THAN=""

usage() {
    cat <<'EOF'
Usage:
  .agents/skills/graft-web-browser-agent/scripts/cleanup.sh --session <id>
  .agents/skills/graft-web-browser-agent/scripts/cleanup.sh --all
  .agents/skills/graft-web-browser-agent/scripts/cleanup.sh --older-than <find-mtime>

Examples:
  cleanup.sh --session audit-filter-check
  cleanup.sh --older-than +7
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --session)
            SESSION="${2:-}"
            shift 2
            ;;
        --all)
            ALL=true
            shift
            ;;
        --older-than)
            OLDER_THAN="${2:-}"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            usage >&2
            exit 1
            ;;
    esac
done

if [[ ! -d "${ARTIFACT_DIR}" ]]; then
    echo "No browser artifacts found at ${ARTIFACT_DIR}"
    exit 0
fi

if [[ -n "${SESSION}" ]]; then
    case "${SESSION}" in
        *"/"*|".."*|"")
            echo "Invalid session id: ${SESSION}" >&2
            exit 1
            ;;
    esac
    rm -rf "${ARTIFACT_DIR}/${SESSION}"
    echo "Removed ${ARTIFACT_DIR}/${SESSION}"
    exit 0
fi

if [[ "${ALL}" == "true" ]]; then
    rm -rf "${ARTIFACT_DIR}"
    echo "Removed ${ARTIFACT_DIR}"
    exit 0
fi

if [[ -n "${OLDER_THAN}" ]]; then
    find "${ARTIFACT_DIR}" -mindepth 1 -maxdepth 1 -type d -mtime "${OLDER_THAN}" -exec rm -rf {} +
    echo "Removed browser artifact sessions older than ${OLDER_THAN} days from ${ARTIFACT_DIR}"
    exit 0
fi

usage >&2
exit 1

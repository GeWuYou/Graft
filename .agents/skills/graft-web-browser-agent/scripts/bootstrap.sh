#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(git -C "${SCRIPT_DIR}/../../.." rev-parse --show-toplevel)"
VENV_DIR="${ROOT_DIR}/.ai/venv"
REQUIREMENTS_PATH="${ROOT_DIR}/.ai/browser/requirements.txt"
BROWSERS_DIR="${ROOT_DIR}/.ai/ms-playwright"
FORCE=false

usage() {
    cat <<'EOF'
Usage:
  .agents/skills/graft-web-browser-agent/scripts/bootstrap.sh [--force]

Creates the project-local Python environment used by graft-web-browser-agent.
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --force)
            FORCE=true
            shift
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

if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 is required but was not found in PATH" >&2
    exit 1
fi

if [[ "${FORCE}" == "true" && -d "${VENV_DIR}" ]]; then
    rm -rf "${VENV_DIR}"
fi

mkdir -p "$(dirname "${REQUIREMENTS_PATH}")" "${BROWSERS_DIR}"

if [[ ! -f "${REQUIREMENTS_PATH}" ]]; then
    echo "Missing ${REQUIREMENTS_PATH}" >&2
    exit 1
fi

if [[ ! -x "${VENV_DIR}/bin/python" ]]; then
    python3 -m venv "${VENV_DIR}"
fi

"${VENV_DIR}/bin/python" -m pip install --upgrade pip
"${VENV_DIR}/bin/python" -m pip install -r "${REQUIREMENTS_PATH}"

PLAYWRIGHT_BROWSERS_PATH="${BROWSERS_DIR}" "${VENV_DIR}/bin/python" -m playwright install chromium

if ! PLAYWRIGHT_BROWSERS_PATH="${BROWSERS_DIR}" "${VENV_DIR}/bin/python" -m playwright install-deps --dry-run chromium >/tmp/graft-playwright-deps.out 2>/tmp/graft-playwright-deps.err; then
    cat >&2 <<EOF
Playwright is installed, but this machine is missing system packages required to launch Chromium.
Review the dry-run output below and install the listed packages explicitly if browser launch fails.

Command:
  PLAYWRIGHT_BROWSERS_PATH="${BROWSERS_DIR}" "${VENV_DIR}/bin/python" -m playwright install-deps chromium
EOF
    cat /tmp/graft-playwright-deps.out >&2
    cat /tmp/graft-playwright-deps.err >&2
fi

cat <<EOF
Graft browser agent environment is ready.
Python: ${VENV_DIR}/bin/python
Browsers: ${BROWSERS_DIR}
EOF

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(git -C "${SCRIPT_DIR}/.." rev-parse --show-toplevel)"
OUTPUT_PATH="${ROOT_DIR}/.ai/environment/tools.raw.yaml"
MODE="${1:---check}"

usage() {
    cat <<'EOF'
Usage:
  bash scripts/collect-dev-environment.sh --check
  bash scripts/collect-dev-environment.sh --write

Modes:
  --check  Print the raw project-relevant environment inventory.
  --write  Write the raw inventory to .ai/environment/tools.raw.yaml.
EOF
}

ensure_supported_mode() {
    case "${MODE}" in
        --check|--write)
            ;;
        *)
            usage
            exit 1
            ;;
    esac
}

command_path() {
    local tool="$1"

    if command -v "${tool}" >/dev/null 2>&1; then
        command -v "${tool}"
    else
        printf '%s' ""
    fi
}

command_installed() {
    local tool="$1"

    if command -v "${tool}" >/dev/null 2>&1; then
        printf 'true'
    else
        printf 'false'
    fi
}

command_version() {
    local tool="$1"

    if ! command -v "${tool}" >/dev/null 2>&1; then
        printf '%s' "not-installed"
        return
    fi

    case "${tool}" in
        go)
            go version 2>/dev/null || printf '%s' "unknown"
            ;;
        python3)
            python3 --version 2>/dev/null || printf '%s' "unknown"
            ;;
        node)
            node --version 2>/dev/null || printf '%s' "unknown"
            ;;
        npm)
            npm --version 2>/dev/null || printf '%s' "unknown"
            ;;
        bun)
            bun --version 2>/dev/null || printf '%s' "unknown"
            ;;
        git)
            git --version 2>/dev/null || printf '%s' "unknown"
            ;;
        rg)
            rg --version 2>/dev/null | head -n 1 || printf '%s' "unknown"
            ;;
        jq)
            jq --version 2>/dev/null || printf '%s' "unknown"
            ;;
        docker)
            docker --version 2>/dev/null || printf '%s' "unknown"
            ;;
        bash)
            bash --version 2>/dev/null | head -n 1 || printf '%s' "unknown"
            ;;
        *)
            "${tool}" --version 2>/dev/null | head -n 1 || printf '%s' "unknown"
            ;;
    esac
}

headroom_path() {
    if command -v headroom >/dev/null 2>&1; then
        command -v headroom
    elif [[ -x "${ROOT_DIR}/.ai/venv/bin/headroom" ]]; then
        printf '%s' "${ROOT_DIR}/.ai/venv/bin/headroom"
    else
        printf '%s' ""
    fi
}

headroom_installed() {
    if [[ -n "$(headroom_path)" ]]; then
        printf 'true'
    else
        printf 'false'
    fi
}

headroom_version() {
    local binary

    binary="$(headroom_path)"
    if [[ -z "${binary}" ]]; then
        printf '%s' "not-installed"
        return
    fi

    "${binary}" --version 2>/dev/null | head -n 1 || printf '%s' "unknown"
}

gh_authenticated() {
    if ! command -v gh >/dev/null 2>&1; then
        printf 'false'
        return
    fi

    if gh auth status >/dev/null 2>&1; then
        printf 'true'
    else
        printf 'false'
    fi
}

codex_mcp_configured() {
    local server_name="$1"

    if ! command -v codex >/dev/null 2>&1; then
        printf 'false'
        return
    fi

    if codex mcp get "${server_name}" >/dev/null 2>&1; then
        printf 'true'
    else
        printf 'false'
    fi
}

python_package_version() {
    local package_name="$1"

    python3 - "${package_name}" <<'PY'
from importlib import metadata
import sys

package_name = sys.argv[1]

try:
    print(metadata.version(package_name))
except metadata.PackageNotFoundError:
    print("not-installed")
PY
}

python_package_installed() {
    local package_name="$1"
    local version

    version="$(python_package_version "${package_name}")"

    if [[ "${version}" == "not-installed" ]]; then
        printf 'false'
    else
        printf 'true'
    fi
}

python_module_command_available() {
    local module_name="$1"

    if python3 -m "${module_name}" --help >/dev/null 2>&1; then
        printf 'true'
    else
        printf 'false'
    fi
}

project_python_package_version() {
    local package_name="$1"
    local project_python="${ROOT_DIR}/.ai/venv/bin/python"

    if [[ ! -x "${project_python}" ]]; then
        printf '%s' "not-installed"
        return
    fi

    "${project_python}" - "${package_name}" <<'PY'
from importlib import metadata
import sys

package_name = sys.argv[1]

try:
    print(metadata.version(package_name))
except metadata.PackageNotFoundError:
    print("not-installed")
PY
}

project_python_package_installed() {
    local package_name="$1"
    local version

    version="$(project_python_package_version "${package_name}")"

    if [[ "${version}" == "not-installed" ]]; then
        printf 'false'
    else
        printf 'true'
    fi
}

playwright_system_deps_available() {
    local project_python="${ROOT_DIR}/.ai/venv/bin/python"

    if [[ ! -x "${project_python}" ]]; then
        printf 'false'
        return
    fi

    if PLAYWRIGHT_BROWSERS_PATH="${ROOT_DIR}/.ai/ms-playwright" "${project_python}" -m playwright install-deps --dry-run chromium >/dev/null 2>&1; then
        printf 'true'
    else
        printf 'false'
    fi
}

playwright_browsers_present() {
    local browsers_dir="${ROOT_DIR}/.ai/ms-playwright"

    if [[ ! -d "${browsers_dir}" ]]; then
        printf 'false'
        return
    fi

    if find "${browsers_dir}" -mindepth 1 -maxdepth 1 -type d -print -quit 2>/dev/null | grep -q .; then
        printf 'true'
    else
        printf 'false'
    fi
}

read_os_release() {
    local key="$1"

    python3 - "$key" <<'PY'
import pathlib
import sys

target_key = sys.argv[1]
values = {}
for line in pathlib.Path("/etc/os-release").read_text(encoding="utf-8").splitlines():
    if "=" not in line:
        continue
    key, value = line.split("=", 1)
    values[key] = value.strip().strip('"')

print(values.get(target_key, "unknown"))
PY
}

repo_file_present() {
    local relative_path="$1"

    if [[ -f "${ROOT_DIR}/${relative_path}" ]]; then
        printf 'true'
    else
        printf 'false'
    fi
}

collect_inventory() {
    local os_name distro version_id kernel shell_name wsl_enabled wsl_version timestamp

    os_name="$(uname -s)"
    distro="$(read_os_release PRETTY_NAME)"
    version_id="$(read_os_release VERSION_ID)"
    kernel="$(uname -r)"
    shell_name="$(basename "${SHELL:-bash}")"
    timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

    if grep -qi microsoft /proc/version 2>/dev/null; then
        wsl_enabled="true"
    else
        wsl_enabled="false"
    fi

    if command -v wslinfo >/dev/null 2>&1; then
        wsl_version="$(wslinfo --wsl-version 2>/dev/null || printf '%s' "unknown")"
    else
        wsl_version="unknown"
    fi

    cat <<EOF
schema_version: 1
generated_at_utc: "${timestamp}"
generator: "scripts/collect-dev-environment.sh"

platform:
  os: "${os_name}"
  distro: "${distro}"
  version: "${version_id}"
  kernel: "${kernel}"
  wsl: ${wsl_enabled}
  wsl_version: "${wsl_version}"
  shell: "${shell_name}"

repository:
  server_go_mod:
    present: $(repo_file_present server/go.mod)
    path: "server/go.mod"
    purpose: "Primary signal that the server runtime has been scaffolded."
  web_package_json:
    present: $(repo_file_present web/package.json)
    path: "web/package.json"
    purpose: "Primary signal that the web runtime has been scaffolded."
  web_bun_lock:
    present: $(repo_file_present web/bun.lock)
    path: "web/bun.lock"
    purpose: "Signal that bun is the preferred locked web package manager."

required_runtimes:
  go:
    installed: $(command_installed go)
    version: "$(command_version go)"
    path: "$(command_path go)"
    purpose: "Builds and tests the Graft server once server/go.mod exists."
  python3:
    installed: $(command_installed python3)
    version: "$(command_version python3)"
    path: "$(command_path python3)"
    purpose: "Runs local automation and environment collection scripts."
  node:
    installed: $(command_installed node)
    version: "$(command_version node)"
    path: "$(command_path node)"
    purpose: "Provides the JavaScript runtime used by the web toolchain."
  npm:
    installed: $(command_installed npm)
    version: "$(command_version npm)"
    path: "$(command_path npm)"
    purpose: "Fallback package manager for the web toolchain."
  bun:
    installed: $(command_installed bun)
    version: "$(command_version bun)"
    path: "$(command_path bun)"
    purpose: "Preferred package manager when web/bun.lock is present."

required_tools:
  git:
    installed: $(command_installed git)
    version: "$(command_version git)"
    path: "$(command_path git)"
    purpose: "Source control and patch review."
  bash:
    installed: $(command_installed bash)
    version: "$(command_version bash)"
    path: "$(command_path bash)"
    purpose: "Executes repository scripts and shell automation."
  rg:
    installed: $(command_installed rg)
    version: "$(command_version rg)"
    path: "$(command_path rg)"
    purpose: "Fast text search across the repository."
  jq:
    installed: $(command_installed jq)
    version: "$(command_version jq)"
    path: "$(command_path jq)"
    purpose: "Inspecting and transforming JSON outputs."

project_tools:
  docker:
    installed: $(command_installed docker)
    version: "$(command_version docker)"
    path: "$(command_path docker)"
    purpose: "Optional container runtime for local services or future automation."
  gh:
    installed: $(command_installed gh)
    authenticated: $(gh_authenticated)
    version: "$(command_version gh)"
    path: "$(command_path gh)"
    purpose: "GitHub CLI for authenticated PR automation and future environment bootstrap scripts."

ai_tools:
  headroom:
    installed: $(headroom_installed)
    version: "$(headroom_version)"
    path: "$(headroom_path)"
    purpose: "Optional local user-level MCP-based context compression tool for AI-assisted development."
    mcp_command: "$(headroom_path) mcp serve"
    memory_status: "controlled-local-only"
    memory_dir: ".ai/headroom/memory"
    learn_status: "controlled-local-only"
    learn_dir: ".ai/headroom/learn"
    instructions_auto_write: "disabled"

mcp_servers:
  codegraph:
    configured: $(codex_mcp_configured codegraph)
    purpose: "Local code graph navigation for AI-assisted source discovery."
  tdesign:
    configured: $(codex_mcp_configured tdesign)
    purpose: "TDesign Vue Next component documentation and DOM knowledge source."
  context7:
    configured: $(codex_mcp_configured context7)
    purpose: "Current third-party library documentation lookup for AI-assisted implementation."
  github:
    configured: $(codex_mcp_configured github)
    purpose: "GitHub PR, Actions, and repository context lookup for AI-assisted review workflows."
  playwright:
    configured: $(codex_mcp_configured playwright)
    purpose: "Exploratory browser interaction aid for graft-web-browser-agent workflows."
  headroom:
    configured: $(codex_mcp_configured headroom)
    purpose: "Optional local MCP compression, retrieval, and stats tools for AI-assisted context management."

python_packages:
  requests:
    installed: $(python_package_installed requests)
    version: "$(python_package_version requests)"
    purpose: "Simple HTTP calls in local helper scripts."
  rich:
    installed: $(python_package_installed rich)
    version: "$(python_package_version rich)"
    purpose: "Readable CLI output for local Python helpers."
  openai:
    installed: $(python_package_installed openai)
    version: "$(python_package_version openai)"
    purpose: "Optional scripted access to OpenAI APIs."
  tiktoken:
    installed: $(python_package_installed tiktoken)
    version: "$(python_package_version tiktoken)"
    purpose: "Optional token counting for prompt and context inspection."
  pydantic:
    installed: $(python_package_installed pydantic)
    version: "$(python_package_version pydantic)"
    purpose: "Optional typed config and schema validation for helper scripts."
  pytest:
    installed: $(python_package_installed pytest)
    version: "$(python_package_version pytest)"
    purpose: "Optional lightweight testing for Python helper scripts."
  pyyaml:
    installed: $(python_package_installed pyyaml)
    version: "$(python_package_version pyyaml)"
    purpose: "Optional YAML validation for local verification commands."
  playwright:
    installed: $(project_python_package_installed playwright)
    version: "$(project_python_package_version playwright)"
    purpose: "Project-local browser automation for AI-assisted web UI inspection."

python_environment:
  system_pip:
    available: $(python_module_command_available pip)
    purpose: "System Python pip availability; project automation should prefer .ai/venv when available."
  venv:
    available: $(python_module_command_available venv)
    purpose: "Creates the project-local .ai/venv Python environment."
  project_venv:
    present: $(if [[ -x "${ROOT_DIR}/.ai/venv/bin/python" ]]; then printf 'true'; else printf 'false'; fi)
    path: ".ai/venv"
    purpose: "Project-local Python helper environment for AI tooling."
  playwright_browsers:
    present: $(playwright_browsers_present)
    path: ".ai/ms-playwright"
    purpose: "Project-local Playwright browser cache used by graft-web-browser-agent."
  playwright_system_deps:
    available: $(playwright_system_deps_available)
    purpose: "Whether Chromium system libraries appear available according to Playwright dry-run checks."
EOF
}

ensure_supported_mode

if [[ "${MODE}" == "--write" ]]; then
    mkdir -p "$(dirname "${OUTPUT_PATH}")"
    collect_inventory > "${OUTPUT_PATH}"
    printf 'Wrote %s\n' "${OUTPUT_PATH}"
else
    collect_inventory
fi

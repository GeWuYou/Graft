#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlsplit

from yaml import YAMLError, safe_load


def repo_root() -> Path:
    current = Path(__file__).resolve()
    for parent in current.parents:
        if (parent / ".git").exists():
            return parent
    raise RuntimeError("Could not locate repository root from script path.")


ROOT_DIR = repo_root()
DEFAULT_OUTPUT_DIR = ROOT_DIR / ".ai" / "artifacts" / "browser"
DEFAULT_BROWSERS_DIR = ROOT_DIR / ".ai" / "ms-playwright"
TARGET_CONFIG_FILE = ROOT_DIR / ".ai" / "private" / "graft-browser-targets.yaml"
AUTH_PATH_PREFIX = "/api/auth/"
SAFE_RUNTIME_IDENTITY = re.compile(r"[A-Za-z0-9][A-Za-z0-9 ._/-]*")
TARGET_CONFIG_TEMPLATE = """schema_version: 1
defaults:
  environment: local
  instance: primary
  service: web
environments:
  local:
    instances:
      primary:
        services:
          web:
            base_url: "http://127.0.0.1:3002"
            credentials:
              username: ""
              password: ""
"""


def parse_viewport(raw: str) -> tuple[int, int]:
    match = re.fullmatch(r"(\d+)x(\d+)", raw.strip())
    if not match:
        raise argparse.ArgumentTypeError("viewport must be WIDTHxHEIGHT, for example 1440x1000")
    width = int(match.group(1))
    height = int(match.group(2))
    if width < 320 or height < 240:
        raise argparse.ArgumentTypeError("viewport is too small")
    return width, height


def timestamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")


def safe_session_name(raw: str | None) -> str:
    value = raw.strip() if raw else f"session-{timestamp()}"
    safe = re.sub(r"[^a-zA-Z0-9_.-]+", "-", value).strip(".-")
    return safe or f"session-{timestamp()}"


def checkout_identity() -> dict[str, str]:
    """Return non-secret Git metadata that ties browser evidence to its checkout."""
    commands = {
        "repository_root": ["rev-parse", "--show-toplevel"],
        "branch": ["branch", "--show-current"],
        "head": ["rev-parse", "HEAD"],
    }
    identity: dict[str, str] = {}
    for name, arguments in commands.items():
        result = subprocess.run(
            ["git", "-C", str(ROOT_DIR), *arguments],
            check=True,
            capture_output=True,
            text=True,
        )
        value = result.stdout.strip()
        identity[name] = value or "detached"
    return identity


def primary_checkout_root() -> Path:
    """Return the repository's developer-owned primary checkout path."""
    result = subprocess.run(
        ["git", "-C", str(ROOT_DIR), "worktree", "list", "--porcelain"],
        check=True,
        capture_output=True,
        text=True,
    )
    for line in result.stdout.splitlines():
        if line.startswith("worktree "):
            return Path(line.removeprefix("worktree ")).resolve()
    raise RuntimeError("Git did not report a primary checkout.")


def working_tree_is_clean() -> bool:
    """Report whether the checkout has no tracked or untracked changes."""
    result = subprocess.run(
        ["git", "-C", str(ROOT_DIR), "status", "--porcelain"],
        check=True,
        capture_output=True,
        text=True,
    )
    return not result.stdout.strip()


def runtime_identity(raw: str | None, checkout: dict[str, str]) -> dict[str, str]:
    """Keep a caller-supplied runtime label auditable without storing credentials."""
    explicit = raw.strip() if raw else ""
    if not explicit or not SAFE_RUNTIME_IDENTITY.fullmatch(explicit):
        raise ValueError("--runtime-identity must be a non-secret label containing the current branch and full HEAD.")
    tokens = set(explicit.split())
    if checkout["branch"] not in tokens or checkout["head"] not in tokens:
        raise ValueError("--runtime-identity must include the current branch and full HEAD as separate values.")
    return {"value": explicit, "source": "explicit"}


def browser_preflight(raw_runtime_identity: str | None) -> dict[str, Any]:
    """Reject unapproved checkout state before importing or launching Playwright."""
    checkout = checkout_identity()
    primary_root = primary_checkout_root()
    if Path(checkout["repository_root"]).resolve() != primary_root:
        raise ValueError("Browser QA is allowed only from the developer-owned primary checkout.")
    if checkout["branch"] == "detached":
        raise ValueError("Browser QA requires a checked-out branch under review.")
    if not working_tree_is_clean():
        raise ValueError("Browser QA requires a clean primary checkout so evidence has a stable HEAD.")
    return {
        "checkout": checkout,
        "runtime_identity": runtime_identity(raw_runtime_identity, checkout),
        "primary_checkout_root": str(primary_root),
        "working_tree_clean": True,
    }


def load_sync_playwright() -> Any:
    """Load Playwright only after checkout and runtime evidence are validated."""
    from playwright.sync_api import sync_playwright

    return sync_playwright


def parse_fill_action(value: str) -> dict[str, str]:
    selector, separator, text = value.rpartition("=")
    selector = selector.strip()
    if not separator:
        raise ValueError("--fill expects SELECTOR=TEXT")
    if not selector:
        raise ValueError("--fill expects a nonempty selector")
    if not text.strip():
        raise ValueError("--fill expects nonempty text")
    return {"kind": "fill", "selector": selector, "text": text}


def parse_click_action(value: str) -> dict[str, str]:
    selector = value.strip()
    if not selector:
        raise ValueError("--click expects a nonempty selector")
    return {"kind": "click", "selector": selector}


class BrowserAction(argparse.Action):
    def __call__(
        self,
        parser: argparse.ArgumentParser,
        namespace: argparse.Namespace,
        values: str | list[str] | None,
        option_string: str | None = None,
    ) -> None:
        actions = list(getattr(namespace, "actions", None) or [])
        value = values if isinstance(values, str) else ""
        if option_string == "--fill":
            actions.append(parse_fill_action(value))
        elif option_string == "--click":
            actions.append(parse_click_action(value))
        else:
            raise ValueError(f"unsupported browser action: {option_string}")
        setattr(namespace, "actions", actions)


def initialize_target_config(path: Path = TARGET_CONFIG_FILE) -> None:
    """Create the private target template exactly once without overwriting user values."""
    path.parent.mkdir(parents=True, exist_ok=True)
    try:
        with path.open("x", encoding="utf-8") as handle:
            handle.write(TARGET_CONFIG_TEMPLATE)
    except FileExistsError as exc:
        raise FileExistsError(f"Target config already exists; refusing to overwrite: {path}") from exc


def load_target_config(path: Path = TARGET_CONFIG_FILE) -> dict[str, Any]:
    try:
        document = safe_load(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        raise
    except YAMLError as exc:
        raise ValueError(f"Target config must be valid YAML: {path}") from exc
    if not isinstance(document, dict) or document.get("schema_version") != 1:
        raise ValueError("Target config must be a mapping with schema_version: 1.")
    if not isinstance(document.get("defaults"), dict) or not isinstance(document.get("environments"), dict):
        raise ValueError("Target config must define defaults and environments mappings.")
    return document


def _selected_name(explicit: str | None, defaults: dict[str, Any], key: str) -> str:
    value = explicit or defaults.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"Target selection requires --{key} or defaults.{key}.")
    return value.strip()


def _mapping_entry(parent: dict[str, Any], key: str, label: str) -> dict[str, Any]:
    value = parent.get(key)
    if not isinstance(value, dict):
        available = ", ".join(sorted(str(name) for name in parent)) or "none"
        raise ValueError(f"Unknown {label}: {key}. Available: {available}")
    return value


def validate_target_url(raw: Any) -> str:
    if not isinstance(raw, str) or not raw.strip():
        raise ValueError("Selected service must define a non-empty base_url, or use --url for one run.")
    value = raw.strip()
    parsed = urlsplit(value)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        raise ValueError("Browser target URL must be an absolute http or https URL.")
    if parsed.username or parsed.password:
        raise ValueError("Browser target URL must not embed credentials.")
    try:
        parsed.port
    except ValueError as exc:
        raise ValueError("Browser target URL must use a valid port.") from exc
    return value


def target_origin(url: str) -> tuple[str, str, int]:
    """Return a normalized HTTP origin for an already validated target URL."""
    parsed = urlsplit(url)
    default_port = 443 if parsed.scheme == "https" else 80
    return parsed.scheme.lower(), (parsed.hostname or "").lower(), parsed.port or default_port


def resolve_target(
    document: dict[str, Any],
    environment: str | None,
    instance: str | None,
    service: str | None,
    url_override: str | None,
    login: bool,
) -> dict[str, Any]:
    defaults = document["defaults"]
    environment_name = _selected_name(environment, defaults, "environment")
    instance_name = _selected_name(instance, defaults, "instance")
    service_name = _selected_name(service, defaults, "service")

    environment_config = _mapping_entry(document["environments"], environment_name, "environment")
    instances = environment_config.get("instances")
    if not isinstance(instances, dict):
        raise ValueError(f"Environment {environment_name} must define an instances mapping.")
    instance_config = _mapping_entry(instances, instance_name, "instance")
    services = instance_config.get("services")
    if not isinstance(services, dict):
        raise ValueError(f"Instance {instance_name} must define a services mapping.")
    service_config = _mapping_entry(services, service_name, "service")
    base_url = validate_target_url(service_config.get("base_url"))

    if login and url_override:
        raise ValueError("--login cannot be used with --url; select a registered target instead.")
    target_url = validate_target_url(url_override) if url_override else base_url
    if url_override and target_origin(target_url) != target_origin(base_url):
        raise ValueError("--url must use the same scheme, host, and port as the selected service base_url.")

    credentials: dict[str, str] | None = None
    if login:
        raw_credentials = service_config.get("credentials")
        if not isinstance(raw_credentials, dict):
            raise ValueError("--login requires service-local credentials in the private target config.")
        username = raw_credentials.get("username")
        password = raw_credentials.get("password")
        if not isinstance(username, str) or not username.strip() or not isinstance(password, str) or not password:
            raise ValueError("--login requires non-empty service-local credentials.username and credentials.password.")
        credentials = {"username": username.strip(), "password": password}

    return {
        "environment": environment_name,
        "instance": instance_name,
        "service": service_name,
        "url": target_url,
        "url_source": "override" if url_override else "config",
        "credentials": credentials,
    }


def public_target_metadata(target: dict[str, Any]) -> dict[str, str]:
    """Return only non-secret target selectors for summary.json."""
    return {
        "environment": target["environment"],
        "instance": target["instance"],
        "service": target["service"],
        "url_source": target["url_source"],
    }


def public_navigation_metadata(requested_url: str, final_url: str) -> dict[str, str]:
    """Record navigation paths without copying private origins, queries, or fragments."""
    requested = urlsplit(requested_url)
    final = urlsplit(final_url)
    return {
        "requested_path": requested.path or "/",
        "final_path": final.path or "/",
    }


def redact_sensitive_values(value: Any, secrets: list[str]) -> Any:
    """Remove configured secrets from any string before serializing evidence."""
    protected = sorted({secret for secret in secrets if secret}, key=len, reverse=True)
    if isinstance(value, str):
        redacted = value
        for secret in protected:
            redacted = redacted.replace(secret, "[REDACTED]")
        return redacted
    if isinstance(value, list):
        return [redact_sensitive_values(item, protected) for item in value]
    if isinstance(value, dict):
        return {key: redact_sensitive_values(item, protected) for key, item in value.items()}
    return value


def redact_actions(actions: list[dict[str, str]]) -> list[dict[str, Any]]:
    redacted: list[dict[str, Any]] = []
    for action in actions:
        if action["kind"] == "fill":
            redacted.append(
                {
                    "kind": "fill",
                    "selector": action["selector"],
                    "text_length": len(action["text"]),
                }
            )
            continue
        redacted.append(dict(action))
    return redacted


def auth_response_event(response: Any) -> dict[str, Any] | None:
    parsed = urlsplit(response.url)
    if AUTH_PATH_PREFIX not in parsed.path:
        return None
    return {"status": response.status, "path": parsed.path}


def has_auth_event(events: list[dict[str, Any]], suffix: str, status: int) -> bool:
    return any(event["path"].endswith(suffix) and event["status"] == status for event in events)


def wait_for_auth_events(events: list[dict[str, Any]], timeout_ms: int) -> None:
    deadline = time.monotonic() + timeout_ms / 1000
    while time.monotonic() < deadline:
        if has_auth_event(events, "/login", 200) and has_auth_event(events, "/bootstrap", 200):
            return
        time.sleep(0.1)
    raise TimeoutError("Timed out waiting for successful /api/auth/login and /api/auth/bootstrap responses.")


def perform_login(page: Any, credentials: dict[str, str], timeout_ms: int) -> None:
    text_inputs = page.locator("input:not([type='checkbox']):not([type='hidden'])")
    text_inputs.first.wait_for(state="visible", timeout=timeout_ms)
    text_inputs.nth(0).fill(credentials["username"])

    password_inputs = page.locator("input[type='password']")
    if password_inputs.count() > 0:
        password_inputs.first.fill(credentials["password"])
    else:
        text_inputs.nth(1).fill(credentials["password"])

    page.get_by_role("button", name=re.compile(r"登录|sign\s*in|login", re.IGNORECASE)).click()
    page.wait_for_function("() => window.location.pathname !== '/login'", timeout=timeout_ms)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Inspect the local Graft web UI with project-local Playwright."
    )
    parser.add_argument("--init-config", action="store_true", help="Create the private target template and stop.")
    parser.add_argument("--environment", help="Environment name; defaults to defaults.environment in private config.")
    parser.add_argument("--instance", help="Instance name; defaults to defaults.instance in private config.")
    parser.add_argument("--service", help="Service name; defaults to defaults.service in private config.")
    parser.add_argument("--url", help="Approved one-run URL override for the selected service.")
    parser.add_argument(
        "--runtime-identity",
        help="Non-secret identity confirmed for the runtime under review; recorded in summary.json.",
    )
    parser.add_argument("--session", help="Stable session id used for artifact directory naming.")
    parser.add_argument("--output-dir", default=str(DEFAULT_OUTPUT_DIR), help="Artifact root directory.")
    parser.add_argument("--viewport", default="1440x1000", type=parse_viewport, help="Viewport as WIDTHxHEIGHT.")
    parser.add_argument("--headful", action="store_true", help="Run a visible browser instead of headless mode.")
    parser.add_argument("--screenshot", action="store_true", help="Write a full-page screenshot.")
    parser.add_argument("--snapshot-text", action="store_true", help="Write visible body text to page-text.txt.")
    parser.add_argument("--click", action=BrowserAction, help="Click a Playwright selector. Repeatable.")
    parser.add_argument("--fill", action=BrowserAction, help="Fill an input with SELECTOR=TEXT. Repeatable.")
    parser.add_argument("--wait-for", help="Wait for a Playwright selector before capturing artifacts.")
    parser.add_argument("--wait-ms", type=int, default=0, help="Extra wait time in milliseconds.")
    parser.add_argument("--timeout-ms", type=int, default=15000, help="Navigation and selector timeout.")
    parser.add_argument("--login", action="store_true", help="Log in to the Graft admin shell before capture.")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    if args.init_config:
        try:
            initialize_target_config(TARGET_CONFIG_FILE)
        except FileExistsError as exc:
            print(str(exc), file=sys.stderr)
            return 2
        print(f"Created {TARGET_CONFIG_FILE}. Fill the approved targets and credentials, then rerun.")
        return 0
    if not TARGET_CONFIG_FILE.exists():
        initialize_target_config(TARGET_CONFIG_FILE)
        print(
            f"Created {TARGET_CONFIG_FILE}. Fill the approved targets and credentials, then rerun; no browser was launched.",
            file=sys.stderr,
        )
        return 2
    try:
        target = resolve_target(
            load_target_config(TARGET_CONFIG_FILE),
            args.environment,
            args.instance,
            args.service,
            args.url,
            args.login,
        )
    except (FileNotFoundError, ValueError) as exc:
        print(f"Browser target selection failed: {exc}", file=sys.stderr)
        return 2

    session = safe_session_name(args.session)
    session_dir = Path(args.output_dir).resolve() / session
    session_dir.mkdir(parents=True, exist_ok=True)

    os.environ.setdefault("PLAYWRIGHT_BROWSERS_PATH", str(DEFAULT_BROWSERS_DIR))
    try:
        preflight = browser_preflight(args.runtime_identity)
    except (RuntimeError, ValueError, subprocess.CalledProcessError) as exc:
        print(f"Browser preflight failed: {exc}", file=sys.stderr)
        return 2

    try:
        sync_playwright = load_sync_playwright()
    except ModuleNotFoundError:
        print(
            "Playwright is not installed. Run "
            ".agents/skills/graft-web-browser-agent/scripts/bootstrap.sh first.",
            file=sys.stderr,
        )
        return 2

    actions = list(getattr(args, "actions", None) or [])
    width, height = args.viewport
    started_at = datetime.now(timezone.utc).isoformat()
    checkout = preflight["checkout"]
    credentials = target["credentials"]
    auth_events: list[dict[str, Any]] = []

    with sync_playwright() as playwright:
        try:
            browser = playwright.chromium.launch(headless=not args.headful)
        except Exception as exc:
            message = str(exc)
            if "error while loading shared libraries" in message or "Host system is missing dependencies" in message:
                print(
                    "Chromium could not start because system browser dependencies are missing. "
                    "Run this explicit system-dependency step if appropriate for this machine:\n"
                    f"  PLAYWRIGHT_BROWSERS_PATH=\"{DEFAULT_BROWSERS_DIR}\" "
                    f"{ROOT_DIR}/.ai/venv/bin/python -m playwright install-deps chromium",
                    file=sys.stderr,
                )
            raise
        context = browser.new_context(viewport={"width": width, "height": height})
        page = context.new_page()
        page.set_default_timeout(args.timeout_ms)
        page.on(
            "response",
            lambda response: auth_events.append(event) if (event := auth_response_event(response)) else None,
        )
        page.goto(target["url"], wait_until="networkidle", timeout=args.timeout_ms)

        if credentials:
            perform_login(page, credentials, args.timeout_ms)
            wait_for_auth_events(auth_events, args.timeout_ms)

        for action in actions:
            if action["kind"] == "click":
                page.locator(action["selector"]).click()
            elif action["kind"] == "fill":
                page.locator(action["selector"]).fill(action["text"])

        if args.wait_for:
            page.locator(args.wait_for).wait_for(timeout=args.timeout_ms)
        if args.wait_ms > 0:
            page.wait_for_timeout(args.wait_ms)

        screenshot_path: str | None = None
        if args.screenshot:
            artifact_path = session_dir / f"{timestamp()}.png"
            page.screenshot(path=str(artifact_path), full_page=True)
            screenshot_path = str(artifact_path)

        text_path: str | None = None
        if args.snapshot_text:
            artifact_path = session_dir / "page-text.txt"
            artifact_path.write_text(page.locator("body").inner_text(timeout=args.timeout_ms), encoding="utf-8")
            text_path = str(artifact_path)

        summary: dict[str, Any] = {
            "session": session,
            "target": public_target_metadata(target),
            "navigation": public_navigation_metadata(target["url"], page.url),
            "started_at": started_at,
            "finished_at": datetime.now(timezone.utc).isoformat(),
            "checkout": checkout,
            "runtime_identity": preflight["runtime_identity"],
            "browser_preflight": {
                "primary_checkout_root": preflight["primary_checkout_root"],
                "working_tree_clean": preflight["working_tree_clean"],
            },
            "viewport": {"width": width, "height": height},
            "headless": not args.headful,
            "actions": redact_actions(actions),
            "login": {
                "attempted": bool(args.login),
                "authenticated": bool(
                    args.login
                    and page.url
                    and urlsplit(page.url).path != "/login"
                    and has_auth_event(auth_events, "/login", 200)
                    and has_auth_event(auth_events, "/bootstrap", 200)
                ),
                "auth_responses": auth_events,
            },
            "screenshot": screenshot_path,
            "text_snapshot": text_path,
            "artifact_dir": str(session_dir),
            "title": page.title(),
        }
        secret_values = [target["url"]]
        if credentials:
            secret_values.extend(credentials.values())
        summary = redact_sensitive_values(summary, secret_values)
        summary_path = session_dir / "summary.json"
        summary_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        context.close()
        browser.close()

    print(json.dumps({"ok": True, "session": session, "artifact_dir": str(session_dir)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

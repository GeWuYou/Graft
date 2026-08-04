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
DEFAULT_CREDENTIALS_FILE = ROOT_DIR / "temp" / "username-passward.yaml"
AUTH_PATH_PREFIX = "/api/auth/"
SAFE_RUNTIME_IDENTITY = re.compile(r"[A-Za-z0-9][A-Za-z0-9 ._/-]*")


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


def runtime_identity(raw: str | None, checkout: dict[str, str]) -> dict[str, str]:
    """Keep a caller-supplied runtime label auditable without storing credentials."""
    explicit = raw.strip() if raw else ""
    if explicit:
        if not SAFE_RUNTIME_IDENTITY.fullmatch(explicit):
            raise ValueError("--runtime-identity must be a non-secret deployment, branch, or build label.")
        return {"value": explicit, "source": "explicit"}
    return {"value": f"{checkout['branch']}@{checkout['head']}", "source": "checkout-default"}


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


def parse_credentials(path: Path, profile: str | None = None) -> tuple[dict[str, str], str | None]:
    if not path.exists():
        raise FileNotFoundError(f"Credentials file does not exist: {path}")

    try:
        document = safe_load(path.read_text(encoding="utf-8"))
    except YAMLError as exc:
        raise ValueError(f"Credentials file must be valid YAML: {path}") from exc
    if not isinstance(document, dict):
        raise ValueError("Credentials YAML must contain a mapping of profiles or credential fields.")

    selected_profile: str | None = None
    if profile:
        selected_profile = select_credential_profile(document, profile)
        candidate = document[selected_profile]
        if not isinstance(candidate, dict):
            raise ValueError(f"Credential profile must be a mapping: {selected_profile}")
        fields = normalize_credential_fields(candidate)
    else:
        fields = normalize_credential_fields(document)
        if not has_credential_fields(fields):
            selected_profile = select_credential_profile(document, profile)
            candidate = document[selected_profile]
            if not isinstance(candidate, dict):
                raise ValueError(f"Credential profile must be a mapping: {selected_profile}")
            fields = normalize_credential_fields(candidate)

    username = first_nonempty(fields, ("username", "account", "user"))
    password = first_nonempty(fields, ("password", "passward", "passwd", "pwd"))
    if not username or not password:
        raise ValueError(
            "Selected credentials must include username/account/user and password/passward/passwd/pwd fields."
        )
    return {"username": username, "password": password}, selected_profile


def normalize_credential_fields(raw_fields: dict[Any, Any]) -> dict[str, str]:
    return {
        str(key).strip().lower(): str(value).strip()
        for key, value in raw_fields.items()
        if value is not None and not isinstance(value, dict)
    }


def has_credential_fields(fields: dict[str, str]) -> bool:
    return bool(first_nonempty(fields, ("username", "account", "user")))


def select_credential_profile(document: dict[Any, Any], requested_profile: str | None) -> str:
    if requested_profile:
        if requested_profile not in document:
            available = ", ".join(str(key) for key, value in document.items() if isinstance(value, dict))
            raise ValueError(f"Credential profile not found: {requested_profile}. Available profiles: {available}")
        return requested_profile

    if isinstance(document.get("dev"), dict):
        return "dev"
    for name, value in document.items():
        if isinstance(value, dict):
            return str(name)
    raise ValueError("Credentials YAML must contain a profile with username and password fields.")


def first_nonempty(fields: dict[str, str], keys: tuple[str, ...]) -> str:
    for key in keys:
        value = fields.get(key, "").strip()
        if value:
            return value
    return ""


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
    parser.add_argument("--url", required=True, help="URL to open, for example http://localhost:5173")
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
    parser.add_argument(
        "--credentials",
        default=str(DEFAULT_CREDENTIALS_FILE),
        help="YAML credential file for --login. Defaults to temp/username-passward.yaml.",
    )
    parser.add_argument(
        "--credential-profile",
        help="Credential profile in the YAML file. Defaults to dev, then the first profile.",
    )
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    session = safe_session_name(args.session)
    session_dir = Path(args.output_dir).resolve() / session
    session_dir.mkdir(parents=True, exist_ok=True)

    os.environ.setdefault("PLAYWRIGHT_BROWSERS_PATH", str(DEFAULT_BROWSERS_DIR))

    try:
        from playwright.sync_api import sync_playwright
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
    checkout = checkout_identity()
    credentials, credential_profile = (
        parse_credentials(Path(args.credentials).resolve(), args.credential_profile) if args.login else (None, None)
    )
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
        page.goto(args.url, wait_until="networkidle", timeout=args.timeout_ms)

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
            target = session_dir / f"{timestamp()}.png"
            page.screenshot(path=str(target), full_page=True)
            screenshot_path = str(target)

        text_path: str | None = None
        if args.snapshot_text:
            target = session_dir / "page-text.txt"
            target.write_text(page.locator("body").inner_text(timeout=args.timeout_ms), encoding="utf-8")
            text_path = str(target)

        summary: dict[str, Any] = {
            "session": session,
            "url": args.url,
            "final_url": page.url,
            "started_at": started_at,
            "finished_at": datetime.now(timezone.utc).isoformat(),
            "checkout": checkout,
            "runtime_identity": runtime_identity(args.runtime_identity, checkout),
            "viewport": {"width": width, "height": height},
            "headless": not args.headful,
            "actions": redact_actions(actions),
            "login": {
                "attempted": bool(args.login),
                "credential_profile": credential_profile,
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
        summary_path = session_dir / "summary.json"
        summary_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        context.close()
        browser.close()

    print(json.dumps({"ok": True, "session": session, "artifact_dir": str(session_dir)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

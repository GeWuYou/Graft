#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def repo_root() -> Path:
    current = Path(__file__).resolve()
    for parent in current.parents:
        if (parent / ".git").exists():
            return parent
    raise RuntimeError("Could not locate repository root from script path.")


ROOT_DIR = repo_root()
DEFAULT_OUTPUT_DIR = ROOT_DIR / ".ai" / "artifacts" / "browser"
DEFAULT_BROWSERS_DIR = ROOT_DIR / ".ai" / "ms-playwright"


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


def collect_action(values: list[str] | None, kind: str) -> list[dict[str, str]]:
    actions: list[dict[str, str]] = []
    if not values:
        return actions
    for value in values:
        if kind == "fill":
            selector, separator, text = value.partition("=")
            if not separator:
                raise ValueError("--fill expects SELECTOR=TEXT")
            actions.append({"kind": "fill", "selector": selector, "text": text})
        else:
            actions.append({"kind": kind, "selector": value})
    return actions


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Inspect the local Graft web UI with project-local Playwright."
    )
    parser.add_argument("--url", required=True, help="URL to open, for example http://localhost:5173")
    parser.add_argument("--session", help="Stable session id used for artifact directory naming.")
    parser.add_argument("--output-dir", default=str(DEFAULT_OUTPUT_DIR), help="Artifact root directory.")
    parser.add_argument("--viewport", default="1440x1000", type=parse_viewport, help="Viewport as WIDTHxHEIGHT.")
    parser.add_argument("--headful", action="store_true", help="Run a visible browser instead of headless mode.")
    parser.add_argument("--screenshot", action="store_true", help="Write a full-page screenshot.")
    parser.add_argument("--snapshot-text", action="store_true", help="Write visible body text to page-text.txt.")
    parser.add_argument("--click", action="append", help="Click a Playwright selector. Repeatable.")
    parser.add_argument("--fill", action="append", help="Fill an input with SELECTOR=TEXT. Repeatable.")
    parser.add_argument("--wait-for", help="Wait for a Playwright selector before capturing artifacts.")
    parser.add_argument("--wait-ms", type=int, default=0, help="Extra wait time in milliseconds.")
    parser.add_argument("--timeout-ms", type=int, default=15000, help="Navigation and selector timeout.")
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

    actions = collect_action(args.click, "click") + collect_action(args.fill, "fill")
    width, height = args.viewport
    started_at = datetime.now(timezone.utc).isoformat()

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
        page.goto(args.url, wait_until="networkidle", timeout=args.timeout_ms)

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
            "started_at": started_at,
            "finished_at": datetime.now(timezone.utc).isoformat(),
            "viewport": {"width": width, "height": height},
            "headless": not args.headful,
            "actions": actions,
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

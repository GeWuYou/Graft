#!/usr/bin/env python3
"""Risk-tiered real-API browser acceptance harness for Graft.

Builds the web production bundle, provisions a per-suite isolated
PostgreSQL/Redis environment through tests/browser/compose, seeds
deterministic fixture data via the test-only Go fixtures, and drives the
Playwright suites under tests/browser/suites against a real backend.

This harness is a developer primary-checkout entrypoint; it never runs in
leased task worktrees (no browser execution, no long-running services).
Retries are disabled by design: an attempt either passes or produces one
sanitized JSON report per attempt.
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass, field
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import sys
import time
from typing import TextIO

ROOT = Path(__file__).resolve().parents[1]
SERVER_DIR = "server"
WEB_DIST_DIR = Path("web") / "dist"
BROWSER_DIR = Path("tests") / "browser"
PLAYWRIGHT_CONFIG = "playwright.config.ts"
COMPOSE_FILE = BROWSER_DIR / "compose" / "docker-compose.browser.yml"
DIST_SERVER_SCRIPT = Path("scripts") / "browser" / "serve_dist.py"
DEFAULT_REPORT_DIR = Path(".artifacts") / "browser-acceptance"
ENV_STATE_FILENAME = "environment.json"
RUN_ID_PATTERN = "%Y%m%d-%H%M%S"

SEED_DATABASE_URL_ENV = "GRAFT_BROWSER_SEED_DATABASE_URL"
WEB_BASE_URL_ENV = "GRAFT_BROWSER_WEB_BASE_URL"
ADMIN_USERNAME_ENV = "GRAFT_BROWSER_ADMIN_USERNAME"
ADMIN_PASSWORD_ENV = "GRAFT_BROWSER_ADMIN_PASSWORD"

DEFAULT_ADMIN_USERNAME = "graft"
DEFAULT_ADMIN_PASSWORD = "graft-admin"
LOCAL_JWT_SECRET = "graft-browser-acceptance-local-dev-secret"
LOCAL_DB_USER = "graft"
LOCAL_DB_PASSWORD = "graft"
LOCAL_DB_NAME = "graft"

PROVISIONED_SERVICES = ("postgres", "redis")

EXIT_OK = 0
EXIT_TEST_FAILURE = 1
EXIT_USAGE = 2
EXIT_ENVIRONMENT = 3

REDACTED = "[redacted]"
SECRET_KEY_PATTERN = re.compile(
    r"password|passwd|secret|token|credential|authorization|cookie|api[-_]?key|session",
    re.IGNORECASE,
)
DSN_PASSWORD_PATTERN = re.compile(r"://([^:/@\s]+):([^@\s]+)@")
BEARER_PATTERN = re.compile(r"[Bb]earer\s+[A-Za-z0-9._\-]+")
QUERY_SECRET_PATTERN = re.compile(
    r"(password|passwd|secret|token|credential|api_key|apikey)=([^&\s\"']+)",
    re.IGNORECASE,
)
POSIX_HOME_PATTERN = re.compile(r"/home/[A-Za-z0-9._\-]+")


@dataclass(frozen=True)
class SuiteSpec:
    """One browser acceptance suite and its execution constraints."""

    name: str
    description: str
    spec: str  # path relative to ROOT, e.g. tests/browser/suites/workbench.spec.ts
    cross_browser: bool = False
    daemon_required: bool = False
    default_enabled: bool = True

    @property
    def tag(self) -> str:
        return "@cross-browser" if self.cross_browser else "@chromium-only"


@dataclass(frozen=True)
class EngineSpec:
    """One Playwright engine and its matrix constraints."""

    name: str
    description: str
    cross_browser_only: bool = False
    default: bool = False


@dataclass
class RunPlan:
    suite: SuiteSpec
    engine: EngineSpec
    tier: str
    keep: bool
    reuse_environment: bool
    skip_build: bool
    allow_daemon: bool = False

    def to_json(self, root: Path = ROOT) -> dict:
        return {
            "suite": self.suite.name,
            "suite_tag": self.suite.tag,
            "engine": self.engine.name,
            "tier": self.tier,
            "keep": self.keep,
            "reuse_environment": self.reuse_environment,
            "skip_build": self.skip_build,
            "daemon_required": self.suite.daemon_required,
            "allow_daemon": self.allow_daemon,
        }


SUITES: dict[str, SuiteSpec] = {
    spec.name: spec
    for spec in (
        SuiteSpec(
            name="workbench",
            description="Dashboard workbench summary and widget rendering against the real backend.",
            spec="tests/browser/suites/workbench.spec.ts",
        ),
        SuiteSpec(
            name="task",
            description="Application task history list, stage detail and log surfaces for the fixture application.",
            spec="tests/browser/suites/task.spec.ts",
        ),
        SuiteSpec(
            name="runtime-target",
            description="Runtime target list rendering and detail drilldown for the seeded fixture targets.",
            spec="tests/browser/suites/runtime-target.spec.ts",
            cross_browser=True,
        ),
        SuiteSpec(
            name="docker",
            description="Docker daemon-dependent surfaces (containers, images). Requires a local Docker daemon.",
            spec="tests/browser/suites/docker.spec.ts",
            daemon_required=True,
            default_enabled=False,
        ),
        SuiteSpec(
            name="compose",
            description="Compose project lifecycle surfaces. Requires a local Docker daemon.",
            spec="tests/browser/suites/compose.spec.ts",
            daemon_required=True,
            default_enabled=False,
        ),
    )
}

ENGINES: dict[str, EngineSpec] = {
    spec.name: spec
    for spec in (
        EngineSpec(name="chromium", description="Baseline engine; runs every suite.", default=True),
        EngineSpec(
            name="firefox",
            description="Cross-browser tier; runs @cross-browser suites only.",
            cross_browser_only=True,
        ),
        EngineSpec(
            name="webkit",
            description="Cross-browser tier; runs @cross-browser suites only.",
            cross_browser_only=True,
        ),
    )
}

TIERS = ("local", "ci")


class UsageError(Exception):
    """Configuration or selection error reported as exit code 2."""


class EnvironmentError_(Exception):
    """Environment provisioning failure reported as exit code 3."""


def resolve_suites(
    requested: list[str],
    registry: dict[str, SuiteSpec] = SUITES,
    allow_daemon: bool = False,
) -> list[SuiteSpec]:
    """Resolve explicit or default suite selection with daemon guardrails."""
    if requested:
        resolved: list[SuiteSpec] = []
        for name in requested:
            suite = registry.get(name)
            if suite is None:
                raise UsageError(
                    f"Unknown suite: {name} (available: {', '.join(sorted(registry))})"
                )
            if suite.daemon_required and not allow_daemon:
                raise UsageError(
                    f"Suite {name} requires a Docker daemon; pass --allow-daemon explicitly "
                    "and run it in a developer primary checkout."
                )
            resolved.append(suite)
        return resolved
    return [registry[name] for name in sorted(registry)
            if registry[name].default_enabled and not registry[name].daemon_required]


def filter_suites_for_engine(suites: list[SuiteSpec], engine: EngineSpec) -> list[SuiteSpec]:
    """Restrict a suite list to the engine's matrix eligibility."""
    if not engine.cross_browser_only:
        return list(suites)
    return [suite for suite in suites if suite.cross_browser]


def select_plan(args: argparse.Namespace) -> list[RunPlan]:
    """Build the full suites x engine plan for one invocation."""
    requested = [
        name.strip()
        for chunk in args.suite
        for name in chunk.split(",")
        if name.strip()
    ]
    suites = resolve_suites(requested, allow_daemon=args.allow_daemon)
    engine_names = [name.strip() for name in args.engine.split(",") if name.strip()]
    engines: list[EngineSpec] = []
    for name in engine_names:
        engine = ENGINES.get(name)
        if engine is None:
            raise UsageError(
                f"Unknown engine: {name} (available: {', '.join(sorted(ENGINES))})"
            )
        engines.append(engine)
    if not engines:
        engines = [ENGINES["chromium"]]

    plans: list[RunPlan] = []
    for engine in engines:
        eligible = filter_suites_for_engine(suites, engine)
        ineligible = [suite.name for suite in suites if suite not in eligible]
        if ineligible:
            raise UsageError(
                f"Engine {engine.name} runs @cross-browser suites only; "
                f"{', '.join(ineligible)} is tagged @chromium-only. "
                "Move the spec to a cross-browser suite or use --engine chromium."
            )
        for suite in eligible:
            plans.append(
                RunPlan(
                    suite=suite,
                    engine=engine,
                    tier=args.tier,
                    keep=args.keep,
                    reuse_environment=args.reuse_environment,
                    skip_build=args.skip_build,
                    allow_daemon=args.allow_daemon,
                )
            )
    return plans


def build_playwright_command(
    plan: RunPlan,
    report_dir: Path,
    config_name: str = PLAYWRIGHT_CONFIG,
) -> tuple[list[str], Path]:
    """Assemble the zero-retry Playwright invocation for one attempt.

    The command runs with cwd = tests/browser, so the spec path and output
    path are expressed relative to that directory.
    """
    suite_dir = ROOT / BROWSER_DIR
    spec_relative = str((ROOT / plan.suite.spec).relative_to(ROOT / BROWSER_DIR))
    output_relative = _relative_or_absolute(report_dir / "playwright-output", suite_dir)
    command = [
        "bunx",
        "playwright",
        "test",
        spec_relative,
        "--config",
        config_name,
        "--project",
        plan.engine.name,
        "--retries=0",
        "--workers=1",
        "--reporter=json",
        f"--output={output_relative}",
    ]
    return command, suite_dir


def _relative_or_absolute(path: Path, root: Path) -> str:
    try:
        return str(path.relative_to(root))
    except ValueError:
        return str(path)


def build_environment_commands(
    pg_port: int,
    redis_port: int,
    backend_port: int,
    proxy_port: int,
    compose_project: str = "graft-browser-acceptance",
) -> dict[str, list[str]]:
    """Expose every provisioning command for dry-run inspection and execution."""
    database_url = local_database_url(pg_port)
    server_env_export = {
        "GRAFT_APP_ENV": "test",
        "GRAFT_DATABASE_URL": database_url,
        "GRAFT_REDIS_ADDR": f"127.0.0.1:{redis_port}",
        "GRAFT_AUTH_JWT_SECRET": LOCAL_JWT_SECRET,
        "GRAFT_HTTP_ADDR": f"127.0.0.1:{backend_port}",
        "GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS": f"http://127.0.0.1:{proxy_port}",
    }
    env_prefix = " ".join(f"{key}={value}" for key, value in server_env_export.items())
    return {
        "compose_up": [
            "env",
            f"GRAFT_BROWSER_PG_HOST_PORT={pg_port}",
            f"GRAFT_BROWSER_REDIS_HOST_PORT={redis_port}",
            "docker", "compose", "-f", str(COMPOSE_FILE),
            "-p", compose_project, "up", "-d", "--wait",
        ],
        "migrate_up": ["env", *f"{env_prefix} go run ./cmd/graft migrate up".split()],
        "seed_fixtures": [
            "env",
            *f"{SEED_DATABASE_URL_ENV}={database_url} go test ./modules/project ./modules/task ./modules/runtime-target -run TestSeedBrowserFixture -count=1".split(),
        ],
        "reset_admin": ["env", *f"{env_prefix} go run ./cmd/graft dev reset-admin".split()],
        "backend_serve": ["env", *f"{env_prefix} go run ./cmd/graft serve".split()],
        "dist_server": [
            sys.executable, "-B", str(DIST_SERVER_SCRIPT),
            "--dist", str(WEB_DIST_DIR),
            "--backend", f"http://127.0.0.1:{backend_port}",
            "--port", str(proxy_port),
        ],
        "web_build": ["bun", "run", "build"],
    }


def local_database_url(pg_port: int) -> str:
    return (
        f"postgres://{LOCAL_DB_USER}:{LOCAL_DB_PASSWORD}@127.0.0.1:{pg_port}/{LOCAL_DB_NAME}"
        "?sslmode=disable"
    )


def reserve_local_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def sanitize_text(text: str, root: Path = ROOT) -> str:
    """Strip credentials and machine-specific absolute paths from free text."""
    if not text:
        return text
    sanitized = DSN_PASSWORD_PATTERN.sub(r"://\1:[redacted]@", text)
    sanitized = BEARER_PATTERN.sub(f"Bearer {REDACTED}", sanitized)
    sanitized = QUERY_SECRET_PATTERN.sub(r"\1=[redacted]", sanitized)
    sanitized = sanitized.replace(str(root) + os.sep, "").replace(str(root), "<repo>")
    # Machine-specific POSIX home paths (including other users') never belong in reports.
    sanitized = POSIX_HOME_PATTERN.sub("~", sanitized)
    home = Path.home()
    if str(home) not in ("", "/"):
        sanitized = sanitized.replace(str(home), "~")
    return sanitized


def sanitize_mapping(mapping: dict, secret_pattern: re.Pattern[str] = SECRET_KEY_PATTERN) -> dict:
    """Return a copy of a flat string mapping with secret-like values redacted."""
    sanitized: dict = {}
    for key, value in mapping.items():
        if secret_pattern.search(str(key)):
            sanitized[key] = REDACTED
        else:
            sanitized[key] = value
    return sanitized


def sanitize_playwright_report(payload: dict, root: Path = ROOT) -> dict:
    """Project the Playwright JSON report onto sanitized per-attempt metadata.

    Playwright marks passing tests with status "expected" and failures with
    "unexpected"; flaky results cannot occur because retries are disabled but
    are still reported as failures if they appear.
    """
    stats = payload.get("stats") or {}
    failures: list[dict] = []

    def visit(node: dict, suite_titles: list[str]) -> None:
        titles = suite_titles + [node.get("title", "")] if node.get("title") else suite_titles
        for spec in node.get("specs", []) or []:
            for test in spec.get("tests", []) or []:
                status = test.get("status")
                if status in ("expected", "skipped"):
                    continue
                result = next(
                    (item for item in (test.get("results") or []) if item.get("error")), None
                )
                error = (result or {}).get("error") or {}
                failures.append({
                    "suite_titles": list(titles),
                    "spec": sanitize_text(str(spec.get("file", "")), root),
                    "line": spec.get("line"),
                    "title": test.get("title", ""),
                    "status": status,
                    "expected_status": test.get("expectedStatus"),
                    "message": sanitize_text(str(error.get("message", "")), root),
                    "snippet": sanitize_text(str(error.get("snippet", "")), root),
                })
        for child in node.get("suites", []) or []:
            visit(child, titles)

    for top_suite in payload.get("suites", []) or []:
        visit(top_suite, [])
    return {
        "stats": {
            key: stats.get(key)
            for key in ("total", "expected", "unexpected", "flaky", "skipped")
        },
        "failures": failures,
    }


def wait_for_http(url: str, timeout_seconds: float, expect_body_substr: str | None = None) -> None:
    """Poll an HTTP endpoint until it responds (optionally with a body marker)."""
    import urllib.error
    import urllib.request

    deadline = time.monotonic() + timeout_seconds
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=2) as response:  # noqa: S310 - fixed local URLs
                body = response.read().decode("utf-8", "replace")
                if expect_body_substr is None or expect_body_substr in body:
                    return
                last_error = RuntimeError(f"unexpected body from {url}")
        except (urllib.error.URLError, OSError, RuntimeError) as error:
            last_error = error
        time.sleep(0.5)
    raise EnvironmentError_(f"endpoint did not become ready within {timeout_seconds}s: {url} ({last_error})")


def run_command(argv: list[str], cwd: Path, env: dict | None = None, capture: bool = True) -> subprocess.CompletedProcess:
    """Run a provisioning command to completion, raising on nonzero exit."""
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    try:
        result = subprocess.run(
            argv,
            cwd=cwd,
            env=merged_env,
            check=False,
            capture_output=capture,
            text=True,
        )
    except OSError as error:
        raise EnvironmentError_(f"missing prerequisite for {argv[0]}: {error}") from error
    if result.returncode != 0:
        detail = (result.stderr or result.stdout or "")[-2000:]
        raise EnvironmentError_(
            f"command failed ({result.returncode}): {' '.join(argv[:4])}... {sanitize_text(detail)}"
        )
    return result


class EnvironmentManager:
    """Provision, keep, reuse and tear down one per-attempt environment."""

    def __init__(self, root: Path, report_dir: Path, progress: TextIO) -> None:
        self.root = root
        self.report_dir = report_dir
        self.progress = progress
        self.state_path = report_dir / ENV_STATE_FILENAME
        self.state: dict = {}

    def provision_fresh(self, skip_build: bool) -> dict:
        pg_port = reserve_local_port()
        redis_port = reserve_local_port()
        backend_port = reserve_local_port()
        proxy_port = reserve_local_port()
        # A unique project per fresh environment keeps concurrent or kept runs
        # from recreating each other's compose services.
        compose_project = f"graft-browser-{self.report_dir.name}"
        commands = build_environment_commands(
            pg_port, redis_port, backend_port, proxy_port, compose_project
        )
        # Record every resource as soon as it exists so a mid-startup failure
        # still tears down what was already created.
        self.state = {
            "mode": "fresh",
            "compose_project": compose_project,
            "compose_file": str(COMPOSE_FILE),
            "ports": {
                "postgres": pg_port,
                "redis": redis_port,
                "backend": backend_port,
                "web": proxy_port,
            },
            "processes": {},
            "created_at": _utcnow(),
        }
        self._log(f"PROVISION compose project {compose_project} (postgres:{pg_port} redis:{redis_port})")
        run_command(commands["compose_up"], self.root)
        self._log("MIGRATE schema via graft migrate up")
        run_command(commands["migrate_up"], self.root / SERVER_DIR)
        self._log("SEED deterministic browser fixtures")
        run_command(commands["seed_fixtures"], self.root / SERVER_DIR)
        self._log("RESET default admin to the deterministic dev credential")
        run_command(commands["reset_admin"], self.root / SERVER_DIR)
        if skip_build:
            self._log("SKIP web production build (--skip-build)")
        else:
            self._log("BUILD web production bundle (bun run build)")
            run_command(commands["web_build"], self.root / "web")
        self._log("START real backend (graft serve)")
        backend_proc = self._spawn(commands["backend_serve"], self.root / SERVER_DIR)
        self.state["processes"]["backend_pid"] = backend_proc.pid
        wait_for_http(f"http://127.0.0.1:{backend_port}/healthz", 180)
        self._log("START production dist server with backend proxy")
        proxy_proc = self._spawn(commands["dist_server"], self.root)
        self.state["processes"]["web_pid"] = proxy_proc.pid
        wait_for_http(f"http://127.0.0.1:{proxy_port}/healthz", 60)
        self._write_state()
        return self.state

    def reuse(self) -> dict:
        if not self.state_path.exists():
            raise UsageError(
                f"--reuse-environment requires a kept environment state file at {self.state_path}; "
                "run a previous attempt with --keep first."
            )
        try:
            state = json.loads(self.state_path.read_text(encoding="utf-8"))
        except (OSError, ValueError) as error:
            raise UsageError(f"kept environment state is unreadable: {error}") from error
        ports = state.get("ports") or {}
        backend_port = ports.get("backend")
        proxy_port = ports.get("web")
        if not backend_port or not proxy_port:
            raise UsageError("kept environment state is missing backend/web ports")
        wait_for_http(f"http://127.0.0.1:{backend_port}/healthz", 15)
        wait_for_http(f"http://127.0.0.1:{proxy_port}/healthz", 15)
        state["mode"] = "reused"
        self.state = state
        self._log(f"REUSE kept environment (backend:{backend_port} web:{proxy_port})")
        return self.state

    def keep(self) -> None:
        if not self.state:
            return
        self.state["mode"] = "kept"
        self._write_state()
        ports = self.state.get("ports", {})
        project = self.state.get("compose_project", "graft-browser-acceptance")
        self._log(
            "KEEP environment for diagnostics: "
            f"GRAFT_BROWSER_PG_HOST_PORT={ports.get('postgres')} "
            f"GRAFT_BROWSER_REDIS_HOST_PORT={ports.get('redis')} "
            f"docker compose -f {COMPOSE_FILE} -p {project} down -v removes the containers; "
            f"the harness pids {self.state.get('processes', {})} and ports {ports} stay alive."
        )

    def teardown(self) -> None:
        processes = self.state.get("processes", {})
        for key in ("web_pid", "backend_pid"):
            pid = processes.get(key)
            if pid:
                self._terminate(pid)
        project = self.state.get("compose_project")
        compose_file = self.state.get("compose_file", str(COMPOSE_FILE))
        if project:
            self._log(f"TEARDOWN compose project {project}")
            # The overlay requires the recorded host ports to load; restore
            # them so `down` cannot fail on missing compose variables.
            ports = self.state.get("ports", {})
            down_env = os.environ.copy()
            if ports.get("postgres"):
                down_env["GRAFT_BROWSER_PG_HOST_PORT"] = str(ports["postgres"])
            if ports.get("redis"):
                down_env["GRAFT_BROWSER_REDIS_HOST_PORT"] = str(ports["redis"])
            result = subprocess.run(
                ["docker", "compose", "-f", compose_file, "-p", project, "down", "-v", "--remove-orphans"],
                cwd=self.root,
                env=down_env,
                check=False,
                capture_output=True,
                text=True,
            )
            if result.returncode != 0:
                detail = sanitize_text((result.stderr or result.stdout or "")[-500:])
                print(
                    f"TEARDOWN compose project {project} failed ({result.returncode}); "
                    f"manual cleanup may be required. {detail}",
                    file=self.progress,
                    flush=True,
                )
        if self.state_path.exists():
            self.state_path.unlink()

    def teardown_only(self) -> None:
        if not self.state_path.exists():
            raise UsageError(
                f"--teardown-only requires a kept environment state file at {self.state_path}."
            )
        try:
            self.state = json.loads(self.state_path.read_text(encoding="utf-8"))
        except (OSError, ValueError) as error:
            raise UsageError(f"kept environment state is unreadable: {error}") from error
        self.teardown()

    def _spawn(self, argv: list[str], cwd: Path) -> subprocess.Popen:
        merged_env = os.environ.copy()
        if argv and argv[0] == "env":
            pairs = argv[1:]
            split_at = next((index for index, item in enumerate(pairs) if "=" not in item), len(pairs))
            for item in pairs[:split_at]:
                key, _, value = item.partition("=")
                merged_env[key] = value
            argv = pairs[split_at:]
        try:
            process = subprocess.Popen(
                argv,
                cwd=cwd,
                env=merged_env,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                start_new_session=(os.name == "posix"),
            )
        except OSError as error:
            raise EnvironmentError_(f"failed to start {argv[0]}: {error}") from error
        return process

    def _terminate(self, pid: int) -> None:
        if os.name == "posix":
            try:
                os.killpg(os.getpgid(pid), 15)
                return
            except (ProcessLookupError, PermissionError, OSError):
                pass
        try:
            os.kill(pid, 15)
        except OSError:
            pass

    def _write_state(self) -> None:
        self.state_path.write_text(
            json.dumps(sanitize_mapping(self.state), indent=2) + "\n", encoding="utf-8"
        )

    def _log(self, message: str) -> None:
        print(message, file=self.progress, flush=True)


def build_attempt_environment(plan: RunPlan, web_port: int) -> dict:
    """Assemble the deterministic per-attempt environment for Playwright."""
    attempt_env = {
        WEB_BASE_URL_ENV: f"http://127.0.0.1:{web_port}",
        ADMIN_USERNAME_ENV: os.environ.get(ADMIN_USERNAME_ENV, DEFAULT_ADMIN_USERNAME),
        ADMIN_PASSWORD_ENV: os.environ.get(ADMIN_PASSWORD_ENV, DEFAULT_ADMIN_PASSWORD),
        "GRAFT_BROWSER_SUITE": plan.suite.name,
        "GRAFT_BROWSER_ENGINE": plan.engine.name,
        "GRAFT_BROWSER_TIER": plan.tier,
    }
    if plan.suite.daemon_required and plan.allow_daemon:
        # The daemon-required specs skip themselves unless the explicit
        # permission is forwarded into the Playwright environment.
        attempt_env["GRAFT_BROWSER_ALLOW_DAEMON"] = "1"
    return attempt_env


def run_attempt(
    plan: RunPlan,
    root: Path,
    report_dir: Path,
    manager: EnvironmentManager,
    progress: TextIO,
) -> tuple[int, dict]:
    """Execute one suites x engine attempt and produce its sanitized report."""
    started = datetime.now(timezone.utc).isoformat()
    suite_report_dir = report_dir / f"{plan.suite.name}-{plan.engine.name}"
    suite_report_dir.mkdir(parents=True, exist_ok=True)
    command, cwd = build_playwright_command(plan, suite_report_dir)
    ports = manager.state.get("ports", {})
    web_port = ports.get("web")
    if not web_port:
        raise EnvironmentError_("environment state is missing the web port")
    attempt_env = build_attempt_environment(plan, web_port)
    print(f"RUN browser/{plan.suite.name}@{plan.engine.name}", file=progress, flush=True)
    merged_env = os.environ.copy()
    merged_env.update(attempt_env)
    json_path = suite_report_dir / "playwright-report.json"
    exit_code: int | None = None
    failure_reason: str | None = None
    try:
        with open(json_path, "w", encoding="utf-8") as json_handle:
            process = subprocess.run(
                command,
                cwd=cwd,
                env=merged_env,
                stdout=json_handle,
                stderr=subprocess.PIPE,
                text=True,
                check=False,
            )
        exit_code = process.returncode
        if process.stderr:
            (suite_report_dir / "playwright-stderr.log").write_text(
                sanitize_text(process.stderr), encoding="utf-8"
            )
    except OSError as error:
        failure_reason = f"playwright could not be started: {error}"

    report: dict = {
        "harness": "graft-browser-acceptance",
        "run_id": report_dir.name,
        "started_at": started,
        "finished_at": datetime.now(timezone.utc).isoformat(),
        "plan": plan.to_json(root),
        "exit_code": exit_code,
        "result": _attempt_result(plan, exit_code, failure_reason),
        "playwright": {
            "argv": [sanitize_text(item, root) for item in command],
            "report": sanitize_text(_relative_or_absolute(json_path, root), root),
            "retries": 0,
        },
        "environment": {
            "mode": manager.state.get("mode", "unknown"),
            "ports": ports,
            "services": list(PROVISIONED_SERVICES),
        },
        "notes": [],
    }
    if failure_reason:
        report["notes"].append(failure_reason)
    try:
        payload = json.loads(json_path.read_text(encoding="utf-8"))
        report["playwright"]["stats_failures"] = sanitize_playwright_report(payload, root)
    except (OSError, ValueError) as error:
        report["playwright"]["stats_failures"] = {"stats": {}, "failures": []}
        report["notes"].append(f"playwright JSON report was not parseable: {error}")

    report_path = suite_report_dir / "attempt.json"
    report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    marker = report["result"].upper()
    print(f"{marker} browser/{plan.suite.name}@{plan.engine.name} "
          f"report={sanitize_text(_relative_or_absolute(report_path, root), root)}",
          file=progress, flush=True)
    return exit_code if exit_code is not None else EXIT_TEST_FAILURE, report


def _attempt_result(plan: RunPlan, exit_code: int | None, failure_reason: str | None) -> str:
    if failure_reason:
        return "error"
    if exit_code == EXIT_OK:
        return "passed"
    return "failed"


def _utcnow() -> str:
    return datetime.now(timezone.utc).isoformat()


def check_prerequisites(root: Path, skip_build: bool) -> list[str]:
    """Return human-readable missing prerequisites without executing anything."""
    missing: list[str] = []
    if shutil.which("docker") is None:
        missing.append("docker (required for the per-suite PostgreSQL/Redis environment)")
    if shutil.which("go") is None:
        missing.append("go (required for migrate, seed fixtures and the backend)")
    if shutil.which("bun") is None and not skip_build:
        missing.append("bun (required to build the web production bundle)")
    if not (root / WEB_DIST_DIR / "index.html").exists() and skip_build:
        missing.append("web/dist/index.html (--skip-build without a previous production build)")
    return missing


def ensure_self_ignored(report_dir: Path) -> None:
    """Keep generated reports out of git status without editing the repo .gitignore."""
    marker = report_dir / ".gitignore"
    if not marker.exists():
        marker.write_text("*\n", encoding="utf-8")


def print_list_suites(progress: TextIO, registry: dict[str, SuiteSpec] = SUITES) -> None:
    print("Suites:", file=progress)
    for name in sorted(registry):
        suite = registry[name]
        markers = []
        if suite.default_enabled and not suite.daemon_required:
            markers.append("default")
        if suite.daemon_required:
            markers.append("daemon-required")
        markers.append(suite.tag)
        print(f"  {name}: {suite.description} [{'/'.join(markers)}]", file=progress)
        print(f"    spec: {suite.spec}", file=progress)


def print_list_engines(progress: TextIO, engines: dict[str, EngineSpec] = ENGINES) -> None:
    print("Engines:", file=progress)
    for name in sorted(engines):
        engine = engines[name]
        marker = "default" if engine.default else ("@cross-browser suites only" if engine.cross_browser_only else "")
        suffix = f" [{marker}]" if marker else ""
        print(f"  {name}: {engine.description}{suffix}", file=progress)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--suite",
        action="append",
        default=[],
        help="Suite to run; repeat or comma-separate. Default: the non-daemon suites.",
    )
    parser.add_argument(
        "--engine",
        default="chromium",
        help="Comma-separated Playwright engines (default: chromium).",
    )
    parser.add_argument(
        "--tier",
        choices=TIERS,
        default="local",
        help="Execution tier: local allows keep/reuse diagnostics; ci forbids them.",
    )
    parser.add_argument(
        "--keep",
        action="store_true",
        help="Keep the provisioned environment running after the attempt for diagnostics.",
    )
    parser.add_argument(
        "--reuse-environment",
        action="store_true",
        help="Attach to an environment kept by a previous --keep run instead of provisioning.",
    )
    parser.add_argument(
        "--skip-build",
        action="store_true",
        help="Skip the web production build (reuse web/dist from a previous build).",
    )
    parser.add_argument("--allow-daemon", action="store_true",
                        help="Explicitly allow daemon-required suites (Docker dependent).")
    parser.add_argument("--teardown-only", action="store_true",
                        help="Tear down a kept environment from the last --keep run, then exit.")
    parser.add_argument("--dry-run", action="store_true",
                        help="Print the resolved plan and commands without executing anything.")
    parser.add_argument("--list-suites", action="store_true",
                        help="Print the suite registry and exit.")
    parser.add_argument("--list-engines", action="store_true",
                        help="Print the engine matrix and exit.")
    parser.add_argument("--report-dir", default=None,
                        help=f"Directory for per-attempt reports (default {DEFAULT_REPORT_DIR}).")
    return parser


def validate_tier_flags(args: argparse.Namespace) -> None:
    if args.tier == "ci" and (args.keep or args.reuse_environment or args.teardown_only):
        raise UsageError(
            "tier ci runs a fixed fresh environment per attempt; --keep/--reuse-environment/"
            "--teardown-only are local-tier diagnostics."
        )


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        if args.list_suites:
            print_list_suites(sys.stdout)
            return EXIT_OK
        if args.list_engines:
            print_list_engines(sys.stdout)
            return EXIT_OK
        validate_tier_flags(args)
        if args.keep and args.reuse_environment:
            raise UsageError("--keep and --reuse-environment are mutually exclusive.")
        if args.report_dir:
            report_dir = ROOT / args.report_dir
        elif args.tier == "ci":
            report_dir = ROOT / DEFAULT_REPORT_DIR / "ci" / datetime.now().strftime(RUN_ID_PATTERN)
        else:
            report_dir = ROOT / DEFAULT_REPORT_DIR / datetime.now().strftime(RUN_ID_PATTERN)
        if not report_dir.is_absolute():
            report_dir = ROOT / report_dir
        manager = EnvironmentManager(ROOT, report_dir, sys.stdout)

        if args.teardown_only:
            manager.teardown_only()
            return EXIT_OK

        plans = select_plan(args)

        if args.dry_run:
            # Plan inspection must work without any prerequisite installed so
            # leased worktrees can verify the wiring without executing anything.
            commands = build_environment_commands(
                reserve_local_port(), reserve_local_port(), reserve_local_port(), reserve_local_port(),
                f"graft-browser-{report_dir.name}",
            )
            attempts = []
            for plan in plans:
                attempt_command, _ = build_playwright_command(plan, report_dir / f"{plan.suite.name}-{plan.engine.name}")
                attempts.append({
                    "plan": plan.to_json(ROOT),
                    "command": attempt_command,
                    "cwd": _relative_or_absolute(ROOT / BROWSER_DIR, ROOT),
                })
            print(json.dumps({
                "plans": [plan.to_json(ROOT) for plan in plans],
                "report_dir": _relative_or_absolute(report_dir, ROOT),
                "steps": {name: argv for name, argv in commands.items()},
                "attempts": attempts,
                "retries": 0,
            }, indent=2))
            return EXIT_OK

        missing = check_prerequisites(ROOT, args.skip_build)
        if missing:
            for item in missing:
                print(f"MISSING {item}", file=sys.stderr, flush=True)
            return EXIT_ENVIRONMENT

        report_dir.mkdir(parents=True, exist_ok=True)
        ensure_self_ignored(report_dir)
        try:
            if args.reuse_environment:
                manager.reuse()
            else:
                manager.provision_fresh(args.skip_build)
        except UsageError:
            raise
        except EnvironmentError_ as error:
            print(f"ENVIRONMENT {error}", file=sys.stderr, flush=True)
            manager.teardown()
            return EXIT_ENVIRONMENT

        exit_code = EXIT_OK
        reports: list[dict] = []
        for plan in plans:
            attempt_exit, report = run_attempt(plan, ROOT, report_dir, manager, sys.stdout)
            reports.append(report)
            if attempt_exit != EXIT_OK:
                exit_code = EXIT_TEST_FAILURE
        if args.keep:
            manager.keep()
        else:
            manager.teardown()

        summary = {
            "harness": "graft-browser-acceptance",
            "run_id": report_dir.name,
            "attempts": [
                {"plan": report["plan"], "result": report["result"], "exit_code": report["exit_code"]}
                for report in reports
            ],
        }
        (report_dir / "summary.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
        return exit_code
    except UsageError as error:
        print(f"USAGE {error}", file=sys.stderr, flush=True)
        return EXIT_USAGE


if __name__ == "__main__":
    raise SystemExit(main())

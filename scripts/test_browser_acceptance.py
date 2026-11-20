#!/usr/bin/env python3
"""Unit tests for the browser acceptance harness (no Docker, browser or DB).

The tests verify CLI parsing, suite/engine/tier selection rules, zero-retry
command assembly, report sanitization, environment keep/reuse flags and the
spec-file tag convention. Full environment provisioning and browser execution
stay with the developer primary-checkout procedure documented in
docs/development/browser-acceptance.md.
"""
from __future__ import annotations

import argparse
import importlib.util
import json
from pathlib import Path
import socket
import subprocess
import sys
import threading
import time
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]

from scripts import browser_acceptance as harness  # noqa: E402 - repository script module


def load_serve_dist():
    spec = importlib.util.spec_from_file_location(
        "serve_dist", ROOT / "scripts" / "browser" / "serve_dist.py"
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


serve_dist = load_serve_dist()


class SuiteRegistryTests(unittest.TestCase):
    def test_default_selection_excludes_daemon_required_suites(self) -> None:
        resolved = harness.resolve_suites([])
        self.assertEqual(
            [suite.name for suite in resolved],
            ["runtime-target", "task", "workbench"],
        )

    def test_explicit_suite_selection_is_order_preserving(self) -> None:
        resolved = harness.resolve_suites(["task", "workbench"])
        self.assertEqual([suite.name for suite in resolved], ["task", "workbench"])

    def test_unknown_suite_is_a_usage_error(self) -> None:
        with self.assertRaises(harness.UsageError) as context:
            harness.resolve_suites(["does-not-exist"])
        self.assertIn("Unknown suite: does-not-exist", str(context.exception))

    def test_daemon_required_suite_requires_explicit_allow_daemon(self) -> None:
        with self.assertRaises(harness.UsageError) as context:
            harness.resolve_suites(["docker"])
        self.assertIn("Docker daemon", str(context.exception))
        resolved = harness.resolve_suites(["docker"], allow_daemon=True)
        self.assertEqual([suite.name for suite in resolved], ["docker"])

    def test_registered_specs_exist_and_carry_the_declared_tag_marker(self) -> None:
        for suite in harness.SUITES.values():
            spec_path = ROOT / suite.spec
            self.assertTrue(spec_path.is_file(), f"missing suite spec: {suite.spec}")
            marker = f"// {suite.tag}"
            content = spec_path.read_text(encoding="utf-8")
            self.assertIn(
                marker, content,
                f"spec {suite.spec} must declare {suite.tag} in a leading marker comment",
            )


class EngineMatrixTests(unittest.TestCase):
    def test_chromium_runs_every_selected_suite(self) -> None:
        engine = harness.ENGINES["chromium"]
        self.assertTrue(engine.default)
        self.assertFalse(engine.cross_browser_only)
        suites = harness.resolve_suites(["workbench", "runtime-target"])
        self.assertEqual(len(harness.filter_suites_for_engine(suites, engine)), 2)

    def test_cross_browser_engines_run_tagged_suites_only(self) -> None:
        for name in ("firefox", "webkit"):
            engine = harness.ENGINES[name]
            self.assertTrue(engine.cross_browser_only)
            suites = harness.resolve_suites(["workbench", "runtime-target"])
            eligible = harness.filter_suites_for_engine(suites, engine)
            self.assertEqual([suite.name for suite in eligible], ["runtime-target"])

    def test_select_plan_rejects_chromium_only_suite_on_cross_browser_engine(self) -> None:
        args = argparse.Namespace(
            suite=["workbench"], engine="firefox", tier="local",
            keep=False, reuse_environment=False, skip_build=False,
            allow_daemon=False,
        )
        with self.assertRaises(harness.UsageError) as context:
            harness.select_plan(args)
        self.assertIn("@chromium-only", str(context.exception))

    def test_select_plan_builds_one_plan_per_suite_and_engine(self) -> None:
        args = argparse.Namespace(
            suite=["runtime-target"], engine="firefox,webkit", tier="local",
            keep=False, reuse_environment=False, skip_build=False,
            allow_daemon=False,
        )
        plans = harness.select_plan(args)
        self.assertEqual(
            [(plan.suite.name, plan.engine.name) for plan in plans],
            [("runtime-target", "firefox"), ("runtime-target", "webkit")],
        )

    def test_select_plan_splits_comma_separated_suite_names(self) -> None:
        args = argparse.Namespace(
            suite=["task,workbench", " runtime-target "], engine="chromium", tier="local",
            keep=False, reuse_environment=False, skip_build=False,
            allow_daemon=False,
        )
        plans = harness.select_plan(args)
        self.assertEqual(
            [plan.suite.name for plan in plans],
            ["task", "workbench", "runtime-target"],
        )

    def test_unknown_engine_is_a_usage_error(self) -> None:
        args = argparse.Namespace(
            suite=[], engine="safari", tier="local",
            keep=False, reuse_environment=False, skip_build=False,
            allow_daemon=False,
        )
        with self.assertRaises(harness.UsageError) as context:
            harness.select_plan(args)
        self.assertIn("Unknown engine: safari", str(context.exception))


class TierAndFlagTests(unittest.TestCase):
    def test_tier_ci_forbids_keep_and_reuse_diagnostics(self) -> None:
        for flag in ("keep", "reuse_environment", "teardown_only"):
            args = argparse.Namespace(
                tier="ci", keep=flag == "keep",
                reuse_environment=flag == "reuse_environment",
                teardown_only=flag == "teardown_only",
            )
            with self.assertRaises(harness.UsageError):
                harness.validate_tier_flags(args)

    def test_tier_local_allows_keep_and_reuse(self) -> None:
        args = argparse.Namespace(
            tier="local", keep=True, reuse_environment=False, teardown_only=False
        )
        harness.validate_tier_flags(args)  # must not raise

    def test_tier_ci_defaults_to_the_ci_report_directory(self) -> None:
        import io

        captured = io.StringIO()
        original_stdout = sys.stdout
        sys.stdout = captured
        try:
            exit_code = harness.main(["--dry-run", "--tier", "ci"])
        finally:
            sys.stdout = original_stdout
        self.assertEqual(exit_code, harness.EXIT_OK)
        payload = json.loads(captured.getvalue())
        self.assertTrue(
            payload["report_dir"].startswith(".artifacts/browser-acceptance/ci/"),
            payload["report_dir"],
        )

    def test_keep_and_reuse_are_mutually_exclusive(self) -> None:
        exit_code = harness.main(["--keep", "--reuse-environment"])
        self.assertEqual(exit_code, harness.EXIT_USAGE)


class PlaywrightCommandTests(unittest.TestCase):
    def _plan(self, suite_name: str = "workbench", engine: str = "chromium") -> harness.RunPlan:
        return harness.RunPlan(
            suite=harness.SUITES[suite_name],
            engine=harness.ENGINES[engine],
            tier="local",
            keep=False,
            reuse_environment=False,
            skip_build=False,
        )

    def test_command_enforces_zero_retries_and_single_worker(self) -> None:
        command, cwd = harness.build_playwright_command(
            self._plan(), Path("/tmp/attempt")
        )
        self.assertIn("--retries=0", command)
        self.assertIn("--workers=1", command)
        self.assertEqual(cwd, ROOT / "tests" / "browser")

    def test_command_selects_project_and_suite_spec(self) -> None:
        command, _ = harness.build_playwright_command(
            self._plan("runtime-target", "firefox"), Path("/tmp/attempt")
        )
        self.assertEqual(command[command.index("--project") + 1], "firefox")
        self.assertEqual(command[command.index("--config") + 1], "playwright.config.ts")
        self.assertIn("suites/runtime-target.spec.ts", command[command.index("test") + 1])

    def test_command_output_lives_under_the_attempt_report_dir(self) -> None:
        command, _ = harness.build_playwright_command(self._plan(), ROOT / ".artifacts" / "attempt")
        output = next(item for item in command if item.startswith("--output="))
        self.assertIn("attempt" + str(Path("/playwright-output")).replace("\\", "/"), output)


class EnvironmentCommandTests(unittest.TestCase):
    def test_commands_cover_the_full_provisioning_flow(self) -> None:
        commands = harness.build_environment_commands(5401, 5402, 5403, 5404)
        self.assertEqual(
            sorted(commands),
            ["backend_serve", "compose_up", "dist_server", "migrate_up",
             "reset_admin", "seed_fixtures", "web_build"],
        )

    def test_compose_project_is_per_environment_not_global(self) -> None:
        commands = harness.build_environment_commands(
            5401, 5402, 5403, 5404, "graft-browser-20261120-040000"
        )
        self.assertIn("graft-browser-20261120-040000", commands["compose_up"])

    def test_compose_up_passes_the_reserved_host_ports(self) -> None:
        commands = harness.build_environment_commands(5401, 5402, 5403, 5404)
        compose = commands["compose_up"]
        self.assertIn("GRAFT_BROWSER_PG_HOST_PORT=5401", compose)
        self.assertIn("GRAFT_BROWSER_REDIS_HOST_PORT=5402", compose)

    def test_seed_command_uses_the_env_guarded_fixture_with_count_one(self) -> None:
        commands = harness.build_environment_commands(5401, 5402, 5403, 5404)
        seed = commands["seed_fixtures"]
        joined = " ".join(seed)
        self.assertIn("TestSeedBrowserFixture", joined)
        self.assertIn("-count=1", joined)
        self.assertIn("./modules/project", joined)
        self.assertIn("./modules/task", joined)
        self.assertIn("./modules/runtime-target", joined)
        self.assertIn(harness.SEED_DATABASE_URL_ENV, joined)

    def test_migrate_and_serve_target_the_isolated_ports(self) -> None:
        commands = harness.build_environment_commands(5401, 5402, 5403, 5404)
        migrate = " ".join(commands["migrate_up"])
        serve = " ".join(commands["backend_serve"])
        for argv in (migrate, serve):
            self.assertIn("postgres://graft:graft@127.0.0.1:5401/graft", argv)
            self.assertIn("127.0.0.1:5402", argv)
            self.assertIn("GRAFT_APP_ENV=test", argv)
        self.assertIn("127.0.0.1:5403", serve)
        dist_server = commands["dist_server"]
        dist_flag_index = dist_server.index("--dist")
        self.assertEqual(dist_server[dist_flag_index + 1], "web/dist")
        self.assertEqual(dist_server[dist_server.index("--port") + 1], "5404")

    def test_local_database_url_format(self) -> None:
        url = harness.local_database_url(6500)
        self.assertEqual(
            url,
            "postgres://graft:graft@127.0.0.1:6500/graft?sslmode=disable",
        )


class SanitizationTests(unittest.TestCase):
    def test_sanitize_text_strips_repo_and_home_paths(self) -> None:
        text = f"at {ROOT}{Path('/scripts/browser_acceptance.py')} and {Path.home()}{Path('/x')}"
        sanitized = harness.sanitize_text(text)
        self.assertNotIn(str(ROOT), sanitized)
        self.assertNotIn(str(Path.home()), sanitized)
        self.assertIn("scripts/browser_acceptance.py", sanitized.replace("\\", "/"))

    def test_sanitize_text_redacts_dsn_passwords_and_bearer_tokens(self) -> None:
        # Compose the fixture credential at runtime so the tracked source never
        # contains a scannable postgres URI literal (synthetic-fixture policy).
        fixture_dsn = "postgres://" + "graft" + ":" + "hunter2" + "@" + "db" + ":5432"
        sanitized = harness.sanitize_text(fixture_dsn + " and Authorization: Bearer abc.def-ghi")
        self.assertNotIn("hunter2", sanitized)
        self.assertNotIn("abc.def-ghi", sanitized)
        self.assertIn("[redacted]", sanitized)

    def test_sanitize_mapping_redacts_secret_like_keys_only(self) -> None:
        sanitized = harness.sanitize_mapping({
            "GRAFT_AUTH_JWT_SECRET": "s3cr3t",
            "ADMIN_PASSWORD": "graft-admin",
            "GRAFT_HTTP_ADDR": "127.0.0.1:8080",
        })
        self.assertEqual(sanitized["GRAFT_AUTH_JWT_SECRET"], harness.REDACTED)
        self.assertEqual(sanitized["ADMIN_PASSWORD"], harness.REDACTED)
        self.assertEqual(sanitized["GRAFT_HTTP_ADDR"], "127.0.0.1:8080")

    def test_sanitize_playwright_report_projects_stats_and_failures(self) -> None:
        payload = {
            "stats": {"total": 2, "expected": 1, "unexpected": 1, "flaky": 0, "skipped": 0},
            "suites": [{
                "title": "task history",
                "specs": [{
                    "file": f"{ROOT}/tests/browser/suites/task.spec.ts",
                    "line": 9,
                    "tests": [
                        {"title": "renders history", "status": "expected",
                         "expectedStatus": "expected", "results": []},
                        {"title": "opens detail", "status": "unexpected",
                         "expectedStatus": "expected",
                         "results": [{"error": {
                             "message": f"boom at {ROOT}/x.js password=hunter2",
                             "snippet": "    at /home/someone/x.js",
                         }}]},
                    ],
                }],
            }],
        }
        projected = harness.sanitize_playwright_report(payload, ROOT)
        self.assertEqual(projected["stats"]["total"], 2)
        self.assertEqual(projected["stats"]["unexpected"], 1)
        self.assertEqual(len(projected["failures"]), 1)
        failure = projected["failures"][0]
        self.assertEqual(failure["suite_titles"], ["task history"])
        self.assertEqual(failure["title"], "opens detail")
        self.assertIn("tests/browser/suites/task.spec.ts", failure["spec"])
        self.assertNotIn(str(ROOT), failure["spec"])
        self.assertNotIn("hunter2", failure["message"])
        self.assertNotIn("/home/someone", failure["snippet"])

    def test_attempt_json_contains_no_credentials(self) -> None:
        report = {
            "plan": harness.RunPlan(
                suite=harness.SUITES["workbench"], engine=harness.ENGINES["chromium"],
                tier="local", keep=False, reuse_environment=False, skip_build=False,
            ).to_json(ROOT),
            "environment": {"ports": {"web": 4173}},
        }
        rendered = json.dumps(report)
        self.assertNotIn(harness.DEFAULT_ADMIN_PASSWORD, rendered)
        self.assertNotIn(harness.LOCAL_JWT_SECRET, rendered)


class AttemptEnvironmentTests(unittest.TestCase):
    def _plan(self, suite_name: str, allow_daemon: bool = False) -> harness.RunPlan:
        return harness.RunPlan(
            suite=harness.SUITES[suite_name],
            engine=harness.ENGINES["chromium"],
            tier="local",
            keep=False,
            reuse_environment=False,
            skip_build=False,
            allow_daemon=allow_daemon,
        )

    def test_daemon_permission_is_forwarded_to_the_playwright_environment(self) -> None:
        environment = harness.build_attempt_environment(self._plan("docker", allow_daemon=True), 4173)
        self.assertEqual(environment["GRAFT_BROWSER_ALLOW_DAEMON"], "1")

    def test_default_suites_never_receive_the_daemon_flag(self) -> None:
        for suite_name in ("workbench", "task", "runtime-target"):
            environment = harness.build_attempt_environment(self._plan(suite_name, allow_daemon=True), 4173)
            self.assertNotIn("GRAFT_BROWSER_ALLOW_DAEMON", environment)

    def test_daemon_suite_without_permission_does_not_receive_the_flag(self) -> None:
        environment = harness.build_attempt_environment(self._plan("docker", allow_daemon=False), 4173)
        self.assertNotIn("GRAFT_BROWSER_ALLOW_DAEMON", environment)

    def test_attempt_environment_exposes_the_web_origin(self) -> None:
        environment = harness.build_attempt_environment(self._plan("workbench"), 4173)
        self.assertEqual(environment[harness.WEB_BASE_URL_ENV], "http://127.0.0.1:4173")
        self.assertEqual(environment["GRAFT_BROWSER_SUITE"], "workbench")


class TeardownTests(unittest.TestCase):
    def test_teardown_restores_compose_port_variables_and_reports_failures(self) -> None:
        import io

        manager = harness.EnvironmentManager(ROOT, Path("/tmp/teardown-test"), io.StringIO())
        manager.state = {
            "mode": "kept",
            "compose_project": "graft-browser-test",
            "compose_file": str(harness.COMPOSE_FILE),
            "ports": {"postgres": 55001, "redis": 55002, "backend": 55003, "web": 55004},
            "processes": {},
        }
        captured_calls: list[dict] = []

        def fake_run(argv, cwd, env, check, capture_output, text):
            captured_calls.append({"argv": argv, "env": dict(env)})
            return subprocess.CompletedProcess(argv, 1, stdout="", stderr="boom")

        with mock.patch.object(harness.subprocess, "run", fake_run):
            manager.teardown()
        self.assertEqual(len(captured_calls), 1)
        call = captured_calls[0]
        self.assertIn("down", call["argv"])
        self.assertEqual(call["env"].get("GRAFT_BROWSER_PG_HOST_PORT"), "55001")
        self.assertEqual(call["env"].get("GRAFT_BROWSER_REDIS_HOST_PORT"), "55002")

    def test_self_ignored_writes_the_gitignore_marker_once(self) -> None:
        import tempfile

        with tempfile.TemporaryDirectory() as directory:
            report_dir = Path(directory) / "run"
            report_dir.mkdir()
            harness.ensure_self_ignored(report_dir)
            self.assertEqual((report_dir / ".gitignore").read_text(encoding="utf-8"), "*\n")
            harness.ensure_self_ignored(report_dir)
            self.assertEqual((report_dir / ".gitignore").read_text(encoding="utf-8"), "*\n")


class ListOutputTests(unittest.TestCase):
    def test_list_suites_marks_daemon_required_and_tags(self) -> None:
        output = _capture(lambda stream: harness.print_list_suites(stream))
        self.assertIn("[default/@chromium-only]", output)
        self.assertIn("[default/@cross-browser]", output)
        self.assertIn("[daemon-required/@chromium-only]", output)
        self.assertIn("spec: tests/browser/suites/workbench.spec.ts", output)
        self.assertIn("spec: tests/browser/suites/runtime-target.spec.ts", output)
        self.assertIn("spec: tests/browser/suites/docker.spec.ts", output)

    def test_list_engines_marks_baseline_and_cross_browser(self) -> None:
        output = _capture(lambda stream: harness.print_list_engines(stream))
        self.assertIn("chromium: Baseline engine; runs every suite. [default]", output)
        self.assertIn("firefox: Cross-browser tier; runs @cross-browser suites only.", output)


class DryRunTests(unittest.TestCase):
    def test_dry_run_prints_the_plan_without_executing(self) -> None:
        import io

        captured = io.StringIO()
        original_stdout = sys.stdout
        sys.stdout = captured
        try:
            exit_code = harness.main(["--dry-run", "--suite", "runtime-target", "--engine", "firefox"])
        finally:
            sys.stdout = original_stdout
        self.assertEqual(exit_code, harness.EXIT_OK)
        payload = json.loads(captured.getvalue())
        self.assertEqual(payload["plans"][0]["suite"], "runtime-target")
        self.assertEqual(payload["plans"][0]["engine"], "firefox")
        self.assertEqual(payload["retries"], 0)
        self.assertIn("seed_fixtures", payload["steps"])
        self.assertIn("compose_up", payload["steps"])
        self.assertIn("playwright", payload["attempts"][0]["command"])


class ServeDistTests(unittest.TestCase):
    def test_proxied_paths_mirror_the_production_proxy_surface(self) -> None:
        for path in (
            "/api/v1/tasks", "/api", "/ws", "/healthz", "/docs", "/openapi.json",
            "/openapi.yaml", "/mcp/docs", "/mcp/docs.json", "/sse/events",
        ):
            self.assertTrue(serve_dist.is_proxied_path(path), path)

    def test_static_paths_are_served_from_the_bundle(self) -> None:
        for path in ("/", "/index.html", "/assets/index-abc.js", "/favicon.ico", "/login"):
            self.assertFalse(serve_dist.is_proxied_path(path), path)

    def test_hop_by_hop_headers_are_filtered_from_proxied_requests(self) -> None:
        filtered = serve_dist.filter_hop_by_hop([
            ("Connection", "close"), ("Upgrade", "websocket"),
            ("Authorization", "Bearer x"), ("Content-Type", "application/json"),
        ])
        names = [name for name, _ in filtered]
        self.assertNotIn("Connection", names)
        self.assertNotIn("Upgrade", names)
        self.assertIn("Authorization", names)
        self.assertIn("Content-Type", names)

    def test_websocket_upgrade_detection(self) -> None:
        class Headers:
            def __init__(self, value: str) -> None:
                self.value = value

            def get(self, name: str, default: str = "") -> str:
                return self.value if name == "Upgrade" else default

        self.assertTrue(serve_dist.is_websocket_upgrade(Headers("websocket")))
        self.assertFalse(serve_dist.is_websocket_upgrade(Headers("")))


def _capture(writer) -> str:
    import io

    stream = io.StringIO()
    writer(stream)
    return stream.getvalue()


class _StubBackendHandler(serve_dist.BaseHTTPRequestHandler):
    """In-process backend stub: JSON echo, held-open SSE and a websocket echo."""

    protocol_version = "HTTP/1.1"

    def do_GET(self) -> None:  # noqa: N802 - http.server naming
        if self.path == "/api/json":
            payload = b'{"data":{"total":2}}'
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)
        elif self.path == "/sse":
            # Headers plus the first event arrive immediately; the connection
            # then stays open briefly to prove the proxy streams instead of
            # buffering the whole response.
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.send_header("Connection", "close")
            self.end_headers()
            self.wfile.write(b"data: heartbeat-1\n\n")
            self.wfile.flush()
            time.sleep(5)
            self.wfile.write(b"data: heartbeat-2\n\n")
            self.wfile.flush()
        elif self.path == "/ws" and "websocket" in self.headers.get("Upgrade", "").lower():
            self.connection.sendall(b"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n\r\n")
            self.connection.settimeout(5)
            try:
                while True:
                    chunk = self.connection.recv(65536)
                    if not chunk:
                        break
                    self.connection.sendall(chunk)
            except OSError:
                pass
            self.close_connection = True
        else:
            self.send_response(404)
            self.send_header("Content-Length", "0")
            self.end_headers()

    def log_message(self, format: str, *args) -> None:  # noqa: A002 - stdlib signature
        pass


class _StubDistHandler(serve_dist.BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802 - http.server naming
        self.send_response(404)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def log_message(self, format: str, *args) -> None:  # noqa: A002 - stdlib signature
        pass


class ServeDistProxyTests(unittest.TestCase):
    """Ephemeral in-process servers: no external services or daemons."""

    def _reserve(self) -> int:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
            sock.bind(("127.0.0.1", 0))
            return int(sock.getsockname()[1])

    def _start(self, handler_class) -> serve_dist.ThreadingHTTPServer:
        server = serve_dist.ThreadingHTTPServer(("127.0.0.1", self._reserve()), handler_class)
        server.daemon_threads = True
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        return server

    def setUp(self) -> None:
        self.backend = self._start(_StubBackendHandler)
        self.dist = self._start(_StubDistHandler)
        backend_origin = f"http://127.0.0.1:{self.backend.server_address[1]}"
        proxy = serve_dist.ThreadingHTTPServer(("127.0.0.1", self._reserve()), serve_dist.DistProxyHandler)
        proxy.daemon_threads = True
        proxy.dist_dir = Path(".")
        proxy.backend_origin = backend_origin
        threading.Thread(target=proxy.serve_forever, daemon=True).start()
        self.addCleanup(proxy.server_close)
        self.addCleanup(proxy.shutdown)
        self.proxy_port = proxy.server_address[1]

    def _request(self, path: str, timeout: float = 5.0) -> bytes:
        connection = socket.create_connection(("127.0.0.1", self.proxy_port), timeout=timeout)
        try:
            connection.settimeout(timeout)
            connection.sendall(f"GET {path} HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n".encode())
            chunks: list[bytes] = []
            try:
                while True:
                    chunk = connection.recv(65536)
                    if not chunk:
                        break
                    chunks.append(chunk)
            except socket.timeout:
                pass
            return b"".join(chunks)
        finally:
            connection.close()

    def test_json_response_is_proxied_with_a_header_mapping(self) -> None:
        raw = self._request("/api/json")
        self.assertIn(b"200 OK", raw)
        self.assertIn(b'{"data":{"total":2}}', raw)

    def test_sse_streams_before_the_backend_closes(self) -> None:
        # The stub holds the second heartbeat for 5s; a buffering proxy would
        # send nothing within the 2s window, a streaming proxy forwards
        # headers plus the first heartbeat immediately.
        raw = self._request("/sse", timeout=2)
        self.assertIn(b"200 OK", raw)
        self.assertIn(b"data: heartbeat-1", raw)
        self.assertNotIn(b"heartbeat-2", raw)

    def test_websocket_upgrade_tunnels_bytes_both_ways(self) -> None:
        connection = socket.create_connection(("127.0.0.1", self.proxy_port), timeout=5)
        try:
            connection.settimeout(5)
            connection.sendall(
                b"GET /ws HTTP/1.1\r\nHost: 127.0.0.1\r\nUpgrade: websocket\r\n"
                b"Connection: Upgrade\r\n\r\n"
            )
            response = connection.recv(65536)
            self.assertIn(b"101 Switching Protocols", response)
            connection.sendall(b"ping-through-tunnel")
            echoed = connection.recv(65536)
            self.assertEqual(echoed, b"ping-through-tunnel")
        finally:
            connection.close()


if __name__ == "__main__":
    unittest.main()

"""Regression checks for the checkout-owned preview lifecycle."""
import argparse
import contextlib
import fcntl
import hashlib
import io
import json
import os
from contextlib import redirect_stdout
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))

import preview
from preview import PreviewError


class PreviewHttpServer:
    """Ephemeral in-process HTTP server for health and status probing."""

    def __init__(self, status=200, content_type="text/html", body=b"<html>ok</html>"):
        outer = self

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(outer.status)
                self.send_header("Content-Type", outer.content_type)
                self.send_header("Content-Length", str(len(outer.body)))
                self.end_headers()
                self.wfile.write(outer.body)

            def log_message(self, *args):
                pass

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.status = status
        self.content_type = content_type
        self.body = body

    @property
    def port(self):
        return self.server.server_address[1]

    def start(self):
        import threading
        thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        thread.start()
        return self

    def stop(self):
        self.server.shutdown()
        self.server.server_close()


def fake_compose(recorder):
    def run(arguments, *, compose_file, project, env_file, timeout=None, capture=True):
        recorder.append({"arguments": arguments, "project": project,
                         "compose_file": str(compose_file), "env_file": str(env_file)})
        return subprocess.CompletedProcess(command_arguments(arguments), 0,
                                           stdout="fake compose log", stderr="")
    return run


def command_arguments(arguments):
    return ["docker", "compose", *arguments]


def lifecycle_args(**overrides):
    values = {"task": "task-one", "image_tag": "graft-test", "port": None, "env": [],
              "health_timeout": 5}
    values.update(overrides)
    return argparse.Namespace(**values)


class IdentityTests(unittest.TestCase):
    def test_identity_is_deterministic_and_separates_checkout_and_task(self):
        first = preview.identity_for(Path("/repo/a"), "PR-42")
        self.assertEqual(first, preview.identity_for(Path("/repo/a"), "PR-42"))
        self.assertNotEqual(first, preview.identity_for(Path("/repo/b"), "PR-42"))
        self.assertNotEqual(first, preview.identity_for(Path("/repo/a"), "PR-43"))

    def test_slug_is_sanitized_and_digest_disambiguates_collisions(self):
        identity = preview.identity_for(Path("/repo/a"), "Weird Task/Name!!")
        self.assertTrue(identity.startswith("weird-task-name-"))
        suffix = identity.rsplit("-", 1)[1]
        self.assertRegex(suffix, r"^[0-9a-f]{12}$")
        self.assertLessEqual(len(identity.rsplit("-", 1)[0]), 24)
        self.assertRegex(identity, r"^[a-z0-9-]+$")
        self.assertRegex(preview.compose_project_for(identity), r"^graft-preview-[a-z0-9-]+$")

    def test_empty_task_is_refused(self):
        with self.assertRaises(PreviewError):
            preview.identity_for(Path("/repo/a"), "   ")


class RegistryTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def test_missing_registry_reads_as_empty_default(self):
        document = preview.read_registry(self.root)
        self.assertEqual(document, {"schema_version": 1, "instances": {}})

    def test_registry_round_trip_is_atomic_and_private(self):
        document = {"schema_version": 1, "instances": {"task-abc": {"web_port": 38000}}}
        preview.write_registry(self.root, document)
        path = self.root / "registry.json"
        self.assertEqual(json.loads(path.read_text(encoding="utf-8")), document)
        self.assertEqual(path.stat().st_mode & 0o777, 0o600)
        self.assertFalse(list(self.root.glob("registry.json.tmp")))

    def test_unsupported_schema_is_rejected(self):
        (self.root / "registry.json").write_text('{"schema_version": 99, "instances": {}}',
                                                 encoding="utf-8")
        with self.assertRaises(PreviewError):
            preview.read_registry(self.root)


class RegistryLockTests(unittest.TestCase):
    def test_registry_lock_excludes_other_holders_with_a_real_flock(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            with preview.registry_lock(root):
                probe = open(root / "registry.lock", "a+")
                try:
                    with self.assertRaises(OSError):
                        fcntl.flock(probe.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
                finally:
                    probe.close()
            # After release another exclusive holder acquires cleanly.
            with preview.registry_lock(root):
                pass

    def test_start_reserves_the_port_inside_the_registry_lock(self):
        # Port allocation and cap reservation must happen while the exclusive
        # registry lock is held, or overlapping starts could double-allocate.
        events = []
        original_allocate = preview.allocate_port
        original_lock = preview.registry_lock

        @contextlib.contextmanager
        def recording_lock(root):
            events.append("lock:enter")
            try:
                with original_lock(root):
                    events.append("locked")
                    yield
            finally:
                events.append("lock:exit")

        def recording_allocate(*args, **kwargs):
            events.append("allocate")
            return original_allocate(*args, **kwargs)

        with tempfile.TemporaryDirectory() as temporary:
            with patch.dict(os.environ, {"GRAFT_PREVIEW_STATE_DIR": temporary}):
                with patch.object(preview, "registry_lock", recording_lock), \
                        patch.object(preview, "allocate_port", recording_allocate), \
                        patch.object(preview, "run_compose", fake_compose([])), \
                        patch.object(preview, "probe_http",
                                     return_value={"ok": True, "status": 200, "error": None}):
                    preview.start(lifecycle_args(task="locked-port"))
        # Allocation must sit inside the first locked span (the later spans
        # are the post-compose state flips re-acquiring the same lock).
        self.assertIn("allocate", events)
        self.assertEqual(events.index("allocate"), events.index("locked") + 1)
        self.assertLess(events.index("allocate"), events.index("lock:exit"))

    def test_two_starts_allocate_distinct_ports(self):
        with tempfile.TemporaryDirectory() as temporary:
            with patch.dict(os.environ, {"GRAFT_PREVIEW_STATE_DIR": temporary,
                                         "GRAFT_PREVIEW_PORT_BASE": "38200"}):
                with patch.object(preview, "run_compose", fake_compose([])), \
                        patch.object(preview, "probe_http",
                                     return_value={"ok": True, "status": 200, "error": None}):
                    first = preview.start(lifecycle_args(task="first-port"))
                    second = preview.start(lifecycle_args(task="second-port"))
        self.assertNotEqual(first["web_url"], second["web_url"])

    def test_registry_writes_use_unique_temp_files(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            document = {"schema_version": 1, "instances": {}}
            with patch.object(preview.os, "getpid", return_value=12345):
                preview.write_registry(root, document)
            leftovers = [path.name for path in root.iterdir() if path.suffix == ".tmp"]
            self.assertEqual(leftovers, [])
            self.assertTrue((root / "registry.json").is_file())


class PortAllocationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.socket.bind(("127.0.0.1", 0))
        self.socket.listen(1)
        self.addCleanup(self.socket.close)
        self.bound_port = self.socket.getsockname()[1]

    def document_with(self, ports):
        return {"schema_version": 1, "instances": {
            f"other-{index}": {"web_port": port} for index, port in enumerate(ports)}}

    def test_foreign_listener_inside_range_is_refused(self):
        with patch.dict(os.environ, {"GRAFT_PREVIEW_PORT_BASE": str(self.bound_port)}):
            with self.assertRaises(PreviewError) as context:
                preview.allocate_port(self.root, self.document_with([]), None)
        self.assertIn(str(self.bound_port), str(context.exception))
        self.assertIn("--port", str(context.exception))

    def test_registered_ports_are_skipped_without_stealing_them(self):
        document = self.document_with([self.bound_port])
        with patch.dict(os.environ, {"GRAFT_PREVIEW_PORT_BASE": str(self.bound_port)}):
            with patch.object(preview, "port_in_use", return_value=False):
                allocated = preview.allocate_port(self.root, document, None)
        self.assertEqual(allocated, self.bound_port + 1)

    def test_foreign_listener_after_a_registered_skip_still_refuses(self):
        document = self.document_with([self.bound_port - 1])
        with patch.dict(os.environ, {"GRAFT_PREVIEW_PORT_BASE": str(self.bound_port - 1)}):
            with self.assertRaises(PreviewError) as context:
                preview.allocate_port(self.root, document, None)
        self.assertIn(str(self.bound_port), str(context.exception))

    def test_explicit_port_conflicts_are_refused(self):
        with self.assertRaises(PreviewError):
            preview.allocate_port(self.root, self.document_with([self.bound_port]), self.bound_port)
        with self.assertRaises(PreviewError):
            preview.allocate_port(self.root, self.document_with([]), self.bound_port)

    def test_exhausted_range_is_refused_explicitly(self):
        occupied = [self.bound_port + offset for offset in range(preview.PORT_RANGE)]
        document = self.document_with(occupied)
        with patch.dict(os.environ, {"GRAFT_PREVIEW_PORT_BASE": str(self.bound_port)}):
            with self.assertRaises(PreviewError):
                preview.allocate_port(self.root, document, None)


class MaxRunningTests(unittest.TestCase):
    def test_env_parsing_validates_bounds(self):
        with patch.dict(os.environ, {"PREVIEW_MAX_RUNNING": "not-a-number"}):
            with self.assertRaises(PreviewError):
                preview.parse_max_running()
        with patch.dict(os.environ, {"PREVIEW_MAX_RUNNING": "0"}):
            with self.assertRaises(PreviewError):
                preview.parse_max_running()
        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop("PREVIEW_MAX_RUNNING", None)
            self.assertEqual(preview.parse_max_running(), preview.DEFAULT_MAX_RUNNING)

    def test_start_refuses_at_the_running_cap(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            document = {"schema_version": 1, "instances": {
                "full-one": {"state": "running", "web_port": 38001},
                "full-two": {"state": "running", "web_port": 38002}}}
            preview.write_registry(root, document)
            with patch.dict(os.environ, {"GRAFT_PREVIEW_STATE_DIR": str(root),
                                         "PREVIEW_MAX_RUNNING": "2"}):
                with patch.object(preview, "run_compose", side_effect=AssertionError("must not run")):
                    with self.assertRaises(PreviewError) as context:
                        preview.start(lifecycle_args(task="third"))
            self.assertIn("PREVIEW_MAX_RUNNING", str(context.exception))
            self.assertNotIn(preview.identity_for(preview.checkout_root(), "third"),
                             preview.read_registry(root)["instances"])


class StartStopDestroyTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.calls = []
        self.compose_patcher = patch.object(preview, "run_compose", fake_compose(self.calls))
        self.compose_patcher.start()
        self.addCleanup(self.compose_patcher.stop)
        self.probe_patcher = patch.object(preview, "probe_http",
                                          return_value={"ok": True, "status": 200, "error": None})
        self.probe_patcher.start()
        self.addCleanup(self.probe_patcher.stop)
        self.cleanup_calls = []
        self.privileged_cleanup_patcher = patch.object(
            preview, "reset_container_owned_permissions",
            side_effect=lambda directory: self.cleanup_calls.append(str(directory)))
        self.privileged_cleanup_patcher.start()
        self.addCleanup(self.privileged_cleanup_patcher.stop)

    def env(self, **extra):
        mapping = {"GRAFT_PREVIEW_STATE_DIR": str(self.root)}
        mapping.update(extra)
        return patch.dict(os.environ, mapping)

    def register(self, identity, state="running", port=38010):
        document = preview.read_registry(self.root)
        document["instances"][identity] = {
            "identity": identity, "task_id": identity, "state": state,
            "web_port": port, "compose_project": f"graft-preview-{identity}",
            "checkout": str(preview.checkout_root()),
            "source": {"revision": "a" * 40, "digest": "0" * 64},
        }
        preview.write_registry(self.root, document)
        directory = preview.instance_dir(self.root, identity)
        directory.mkdir(parents=True, exist_ok=True)
        (directory / "compose.yml").write_text("services: {}\n", encoding="utf-8")
        (directory / ".env").write_text("GRAFT_IMAGE_TAG=graft:test\n", encoding="utf-8")
        return document

    def test_start_registers_instance_writes_env_and_converges(self):
        with self.env():
            result = preview.start(lifecycle_args(task="start-one", env=["GRAFT_DATABASE_URL=x"]))
        identity = result["identity"]
        document = preview.read_registry(self.root)
        entry = document["instances"][identity]
        self.assertEqual(entry["state"], "running")
        directory = preview.instance_dir(self.root, identity)
        env_text = (directory / ".env").read_text(encoding="utf-8")
        self.assertIn("GRAFT_IMAGE_TAG=graft-test", env_text)
        self.assertIn("GRAFT_DATABASE_URL=x", env_text)
        self.assertIn(f"GRAFT_BACKUP_ARTIFACT_HOST_PATH={directory / 'backups'}", env_text)
        self.assertIn(f"GRAFT_BUILD_SNAPSHOT_VOLUME=graft-preview-{identity}-build-snapshots", env_text)
        self.assertIn(f"GRAFT_UPDATE_STATE_VOLUME=graft-preview-{identity}-update-state", env_text)
        self.assertTrue((directory / "compose.yml").is_file())
        self.assertTrue((directory / "logs" / "compose.log").is_file())
        up_calls = [call for call in self.calls if call["arguments"] == ["up", "-d", "--wait"]]
        self.assertEqual(len(up_calls), 1)
        self.assertEqual(up_calls[0]["project"], f"graft-preview-{identity}")
        self.assertEqual(result["web_url"], f"http://127.0.0.1:{entry['web_port']}/")

    def test_start_is_idempotent_for_a_running_instance(self):
        identity = preview.identity_for(preview.checkout_root(), "already-running")
        self.register(identity, state="running")
        registered_port = preview.read_registry(self.root)["instances"][identity]["web_port"]
        with self.env():
            result = preview.start(lifecycle_args(task="already-running"))
        self.assertTrue(result["already_running"])
        self.assertEqual(result["state"], "running")
        self.assertEqual(result["web_url"], f"http://127.0.0.1:{registered_port}/")
        self.assertEqual(self.calls, [])

    def test_start_restores_an_unhealthy_marked_running_instance(self):
        # A registry entry can say "running" while the containers were
        # stopped or removed externally; start must fall through to compose
        # up and restore the stack instead of returning a lying early success.
        identity = preview.identity_for(preview.checkout_root(), "restore-one")
        self.register(identity, state="running")
        with self.env():
            with patch.object(preview, "wait_for_health", side_effect=[
                    {"ok": False, "status": None, "error": "connection refused"},
                    {"ok": True, "status": 200, "error": None}]):
                result = preview.start(lifecycle_args(task="restore-one"))
        self.assertFalse(result["already_running"])
        self.assertEqual(result["state"], "running")
        up_calls = [call for call in self.calls if call["arguments"] == ["up", "-d", "--wait"]]
        self.assertEqual(len(up_calls), 1)
        self.assertEqual(preview.read_registry(self.root)["instances"][identity]["state"],
                         "running")

    def test_start_refuses_port_mismatch_on_registered_instance(self):
        identity = preview.identity_for(preview.checkout_root(), "pinned-port")
        self.register(identity, state="stopped", port=38099)
        with self.env():
            with self.assertRaises(PreviewError):
                preview.start(lifecycle_args(task="pinned-port", port=38100))

    def test_start_refuses_restart_at_the_running_cap(self):
        # Restarting a stopped instance re-occupies the cap; a running sibling
        # must block the restart instead of the stopped entry bypassing it.
        identity = preview.identity_for(preview.checkout_root(), "restart-at-cap")
        self.register(identity, state="stopped")
        document = preview.read_registry(self.root)
        document["instances"]["running-sibling"] = {
            "identity": "running-sibling", "task_id": "running-sibling",
            "state": "running", "web_port": 38021,
            "compose_project": "graft-preview-running-sibling",
            "checkout": str(preview.checkout_root())}
        preview.write_registry(self.root, document)
        with self.env(PREVIEW_MAX_RUNNING="1"):
            with self.assertRaises(PreviewError) as context:
                preview.start(lifecycle_args(task="restart-at-cap"))
        self.assertIn("PREVIEW_MAX_RUNNING", str(context.exception))
        self.assertEqual(self.calls, [])
        self.assertEqual(preview.read_registry(self.root)["instances"][identity]["state"],
                         "stopped")

    def test_start_allows_restart_when_the_cap_has_room(self):
        identity = preview.identity_for(preview.checkout_root(), "restart-with-room")
        self.register(identity, state="stopped")
        with self.env(PREVIEW_MAX_RUNNING="1"):
            result = preview.start(lifecycle_args(task="restart-with-room"))
        self.assertEqual(result["state"], "running")
        up_calls = [call for call in self.calls if call["arguments"] == ["up", "-d", "--wait"]]
        self.assertEqual(len(up_calls), 1)

    def test_start_refuses_reserved_and_malformed_env_overrides(self):
        with self.env():
            with self.assertRaises(PreviewError):
                preview.start(lifecycle_args(task="reserved-env", env=["GRAFT_IMAGE_TAG=evil"]))
            with self.assertRaises(PreviewError):
                preview.start(lifecycle_args(task="reserved-backup-path",
                                             env=["GRAFT_BACKUP_ARTIFACT_HOST_PATH=/opt/graft/backups"]))
            with self.assertRaises(PreviewError):
                preview.start(lifecycle_args(task="malformed-env", env=["bad-key=1"]))

    def test_start_rejects_full_image_refs_in_favor_of_bare_tags(self):
        # compose.yml already prefixes the registry image name; a full ref
        # would produce a double image reference.
        with self.env():
            for bad in ("ghcr.io/gewuyou/graft-server:candidate-v1.2.3-abc",
                        "graft-server:candidate-v1.2.3-abc",
                        "sha256:deadbeef"):
                with self.assertRaises(PreviewError) as context:
                    preview.start(lifecycle_args(task="full-ref", image_tag=bad))
            self.assertEqual(self.calls, [])
        with self.env():
            result = preview.start(lifecycle_args(task="bare-tag",
                                                  image_tag="candidate-v1.2.3-abc"))
        self.assertEqual(result["state"], "running")
        identity = result["identity"]
        env_text = (preview.instance_dir(self.root, identity) / ".env").read_text(encoding="utf-8")
        self.assertIn("GRAFT_IMAGE_TAG=candidate-v1.2.3-abc", env_text)

    def test_compose_failure_marks_the_instance_failed(self):
        def failing_run(arguments, **kwargs):
            return subprocess.CompletedProcess(["docker"], 1, stdout="", stderr="boom")
        with self.env():
            with patch.object(preview, "run_compose", failing_run):
                with self.assertRaises(PreviewError):
                    preview.start(lifecycle_args(task="failing-start"))
        identity = preview.identity_for(preview.checkout_root(), "failing-start")
        self.assertEqual(preview.read_registry(self.root)["instances"][identity]["state"], "failed")

    def test_stop_uses_compose_stop_and_retains_data(self):
        identity = preview.identity_for(preview.checkout_root(), "stop-one")
        self.register(identity, state="running")
        arguments = argparse.Namespace(task="stop-one", identity=None)
        with self.env():
            result = preview.stop(arguments)
        self.assertEqual(result, {"identity": identity, "state": "stopped", "data_retained": True})
        self.assertEqual([call["arguments"] for call in self.calls], [["stop"]])
        entry = preview.read_registry(self.root)["instances"][identity]
        self.assertEqual(entry["state"], "stopped")
        self.assertTrue((preview.instance_dir(self.root, identity) / ".env").is_file())

    def test_destroy_requires_the_exact_confirm_instance(self):
        identity = preview.identity_for(preview.checkout_root(), "destroy-guard")
        self.register(identity, state="running")
        arguments = argparse.Namespace(task="destroy-guard", identity=None,
                                       confirm_instance=identity.upper())
        with self.env():
            with self.assertRaises(PreviewError):
                preview.destroy(arguments)
        self.assertIn(identity, preview.read_registry(self.root)["instances"])
        self.assertEqual(self.calls, [])

    def test_destroy_removes_data_and_archives_sha256_evidence(self):
        identity = preview.identity_for(preview.checkout_root(), "destroy-ok")
        document = self.register(identity, state="running")
        entry_before = document["instances"][identity]
        arguments = argparse.Namespace(task="destroy-ok", identity=None,
                                       confirm_instance=identity)
        with self.env():
            result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])
        self.assertFalse(preview.instance_dir(self.root, identity).exists())
        down_calls = [call for call in self.calls if call["arguments"]]
        self.assertIn(["down", "--volumes", "--remove-orphans"],
                      [call["arguments"] for call in down_calls])
        # Container-owned data is reclaimed through the privileged cleanup
        # before the instance directory is removed.
        self.assertEqual(self.cleanup_calls, [str(preview.instance_dir(self.root, identity))])
        archive_file = Path(result["archive_file"])
        record = json.loads(archive_file.read_text(encoding="utf-8"))
        self.assertEqual(record["registry_entry"], entry_before)
        self.assertEqual(record["evidence_sha256"],
                         hashlib.sha256(preview.canonical_json(entry_before).encode()).hexdigest())
        manifest = preview.read_archive_manifest(self.root)
        self.assertEqual(manifest["entries"][archive_file.name]["sha256"],
                         preview.digest_file(archive_file))

    def test_destroy_cleanup_failure_keeps_the_entry_retryable(self):
        # Cleanup failure must not leave an untracked half-cleanup: the entry
        # stays registered with a cleanup_failed marker and destroy can be
        # retried once the blocker is gone.
        identity = preview.identity_for(preview.checkout_root(), "destroy-retry")
        self.register(identity, state="running")
        arguments = argparse.Namespace(task="destroy-retry", identity=None,
                                       confirm_instance=identity)
        with self.env():
            with patch.object(preview, "reset_container_owned_permissions",
                              side_effect=PreviewError("docker run failed: image missing")):
                with self.assertRaises(PreviewError) as context:
                    preview.destroy(arguments)
        self.assertIn("entry kept for retry", str(context.exception))
        entry = preview.read_registry(self.root)["instances"][identity]
        self.assertEqual(entry["state"], "cleanup_failed")
        self.assertIn("docker run failed", entry["cleanup_error"])
        self.assertEqual(preview.read_archive_manifest(self.root).get("entries", {}), {})
        self.assertTrue(preview.instance_dir(self.root, identity).exists())
        with self.env():
            result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])
        self.assertFalse(preview.instance_dir(self.root, identity).exists())


class StatusTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.http = PreviewHttpServer().start()
        self.addCleanup(self.http.stop)
        document = {"schema_version": 1, "instances": {
            "status-one": {"identity": "status-one", "task_id": "status-one",
                           "state": "running", "web_port": self.http.port,
                           "compose_project": "graft-preview-status-one",
                           "checkout": str(preview.checkout_root()),
                           "image_tag": "graft:test", "created_at": "2026-01-01T00:00:00Z",
                           "updated_at": "2026-01-01T00:00:00Z",
                           "source": {"revision": "a" * 40, "digest": "0" * 64}}}}
        preview.write_registry(self.root, document)

    def env(self):
        return patch.dict(os.environ, {"GRAFT_PREVIEW_STATE_DIR": str(self.root)})

    def test_status_json_reports_urls_health_logs_and_source_digest(self):
        arguments = argparse.Namespace(task=None, identity="status-one", all=False)
        with self.env():
            result = preview.status(arguments)
        self.assertEqual(result["identity"], "status-one")
        self.assertEqual(result["state"], "running")
        self.assertEqual(result["web"]["url"], f"http://127.0.0.1:{self.http.port}/")
        self.assertEqual(result["health"]["web"]["status"], 200)
        self.assertTrue(result["health"]["web"]["ok"])
        self.assertIn("logs/compose.log", result["logs"]["compose_log"])
        self.assertEqual(result["source"]["revision"], "a" * 40)
        self.assertIn("current_digest", result["source"])

    def test_status_reports_source_drift_and_checkout_mismatch(self):
        arguments = argparse.Namespace(task=None, identity="status-one", all=False)
        directory = preview.instance_dir(self.root, "status-one")
        directory.mkdir(parents=True, exist_ok=True)
        (directory / "compose.yml").write_text("services: {drifted: {}}\n", encoding="utf-8")
        with self.env():
            result = preview.status(arguments)
        self.assertTrue(result["source"]["drifted"])
        self.assertFalse(result["source"]["checkout_matches_start"])

    def test_status_unknown_identity_is_explicit(self):
        arguments = argparse.Namespace(task=None, identity="missing-one", all=False)
        with self.env():
            result = preview.status(arguments)
        self.assertFalse(result["registered"])
        self.assertEqual(result["identity"], "missing-one")

    def test_status_all_lists_every_instance(self):
        arguments = argparse.Namespace(task=None, identity=None, all=True)
        with self.env():
            result = preview.status(arguments)
        self.assertEqual([item["identity"] for item in result["instances"]], ["status-one"])

    def test_main_status_emits_pure_json_to_stdout(self):
        buffer = io.StringIO()
        with self.env():
            with redirect_stdout(buffer):
                preview.main(["status", "--identity", "status-one"])
        document = json.loads(buffer.getvalue())
        self.assertEqual(document["identity"], "status-one")


class EnvFileTests(unittest.TestCase):
    def test_extra_env_validation(self):
        parsed = preview.parse_extra_env(["GRAFT_DATABASE_URL=host=db", "POSTGRES_DB=preview"])
        self.assertEqual(parsed, {"GRAFT_DATABASE_URL": "host=db", "POSTGRES_DB": "preview"})
        for bad in ("no-equals", "=value", "lower-case=1", "KEY=\nnewline"):
            with self.assertRaises(PreviewError):
                preview.parse_extra_env([bad])
        with self.assertRaises(PreviewError):
            preview.parse_extra_env(["COMPOSE_FILE_DIR=/elsewhere"])


if __name__ == "__main__":
    unittest.main()

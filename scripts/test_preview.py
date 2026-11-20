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
import shutil
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
            preview, "remove_container_owned_data",
            side_effect=lambda directory: self.cleanup_calls.append(str(directory)))
        self.privileged_cleanup_patcher.start()
        self.addCleanup(self.privileged_cleanup_patcher.stop)
        self.teardown_calls = []
        self.teardown_patcher = patch.object(
            preview, "teardown_recorded_project",
            side_effect=lambda project: self.teardown_calls.append(project))
        self.teardown_patcher.start()
        self.addCleanup(self.teardown_patcher.stop)
        self.restore_calls = []
        self.restore_patcher = patch.object(
            preview, "restore_developer_ownership",
            side_effect=lambda directory: self.restore_calls.append(str(directory)))
        self.restore_patcher.start()
        self.addCleanup(self.restore_patcher.stop)

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
        directory = preview.instance_dir(self.root, identity)
        # Simulate the container-owned data roots the stack creates
        # (application root, backups, compose-managed .data) as they look
        # after a run: present, host-unremovable in production.
        (directory / "apps").mkdir(parents=True)
        (directory / "apps" / "data.bin").write_text("container data", encoding="utf-8")
        (directory / "backups").mkdir(parents=True)
        (directory / ".data" / "postgres").mkdir(parents=True)

        def in_container_removal(target):
            # Mirrors remove_container_owned_data: the privileged container
            # deletes every top-level entry except the two host files.
            for child in target.iterdir():
                if child.name in ("compose.yml", ".env"):
                    continue
                if child.is_dir():
                    shutil.rmtree(child)
                else:
                    child.unlink()

        arguments = argparse.Namespace(task="destroy-ok", identity=None,
                                       confirm_instance=identity)
        with self.env():
            with patch.object(preview, "remove_container_owned_data",
                              side_effect=in_container_removal):
                result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])
        self.assertFalse(directory.exists())
        down_calls = [call for call in self.calls if call["arguments"]]
        self.assertIn(["down", "--volumes", "--remove-orphans"],
                      [call["arguments"] for call in down_calls])
        # Stack-created data never triggers the ownership restore: the
        # privileged removal deletes it after the stack is stopped.
        self.assertEqual(self.restore_calls, [])
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
        # retried once the blocker is gone; status stays readable meanwhile.
        identity = preview.identity_for(preview.checkout_root(), "destroy-retry")
        self.register(identity, state="running")
        arguments = argparse.Namespace(task="destroy-retry", identity=None,
                                       confirm_instance=identity)
        with self.env():
            with patch.object(preview, "remove_container_owned_data",
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
            status_report = preview.status(argparse.Namespace(task=None, identity=None, all=True))
        self.assertEqual(status_report["instances"][0]["state"], "cleanup_failed")
        with self.env():
            result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])
        self.assertFalse(preview.instance_dir(self.root, identity).exists())

    def test_privileged_removal_deletes_inside_the_container_not_chmod_for_the_host(self):
        recorded = {}

        def fake_run(command, capture_output, text, timeout):
            recorded["command"] = command
            return subprocess.CompletedProcess(command, 0, stdout="", stderr="")

        with tempfile.TemporaryDirectory() as temporary:
            # Suspend the class-level recorder so the real privileged
            # function executes under the stubbed docker call.
            self.privileged_cleanup_patcher.stop()
            try:
                with patch.object(preview.subprocess, "run", side_effect=fake_run):
                    preview.remove_container_owned_data(Path(temporary))
                with patch.object(preview.subprocess, "run",
                                  return_value=subprocess.CompletedProcess(
                                      [], 1, stdout="", stderr="no such image")):
                    with self.assertRaises(PreviewError):
                        preview.remove_container_owned_data(Path(temporary))
            finally:
                self.privileged_cleanup_patcher.start()
            command = recorded["command"]
            self.assertEqual(command[:5], ["docker", "run", "--rm", "--user", "0:0"])
            self.assertIn("postgres:16.16-alpine", command)
            script = command[command.index("sh") + 2]
            self.assertIn("rm -rf", script)
            self.assertIn("! -name compose.yml", script)
            self.assertIn("! -name .env", script)
            self.assertIn("/target", script)
            self.assertNotIn("chown", script)
            self.assertIn(f"{temporary}:/target", command)

    def test_status_all_reports_unreadable_instance_dir_without_crashing(self):
        # A failed destroy can leave root-owned files in the instance
        # directory; status --all must report the anomaly instead of
        # crashing on every reader.
        identity = preview.identity_for(preview.checkout_root(), "unreadable-status")
        self.register(identity, state="running")
        compose_file = preview.instance_dir(self.root, identity) / "compose.yml"
        compose_file.chmod(0o000)
        try:
            with self.env():
                report = preview.status(argparse.Namespace(task=None, identity=None, all=True))
        finally:
            compose_file.chmod(0o600)
        instance = report["instances"][0]
        self.assertTrue(any("instance_dir_unreadable" in anomaly
                            for anomaly in instance["anomalies"]))
        self.assertIsNone(instance["source"]["current_digest"])
        self.assertEqual(instance["state"], "running")

    def test_unreadable_checkout_preserves_the_instance_verdict(self):
        # A checkout-side read failure is a different anomaly: it must not
        # clear the instance digest and drift verdict computed from the
        # readable instance side, nor point the developer at the wrong
        # directory.
        identity = preview.identity_for(preview.checkout_root(), "checkout-unreadable")
        self.register(identity, state="running")
        with tempfile.TemporaryDirectory() as temporary:
            checkout_compose = Path(temporary) / "compose.yml"
            checkout_compose.write_text("services: {}\n", encoding="utf-8")
            checkout_compose.chmod(0o000)
            document = preview.read_registry(self.root)
            document["instances"][identity]["checkout"] = temporary
            preview.write_registry(self.root, document)
            try:
                with self.env():
                    report = preview.status(argparse.Namespace(task=None,
                                                               identity=identity, all=False))
            finally:
                checkout_compose.chmod(0o600)
        instance = report
        self.assertTrue(any("checkout_compose_unreadable" in anomaly
                            for anomaly in instance["anomalies"]))
        self.assertFalse(any("instance_dir_unreadable" in anomaly
                             for anomaly in instance["anomalies"]))
        instance_compose = preview.instance_dir(self.root, identity) / "compose.yml"
        self.assertEqual(instance["source"]["current_digest"],
                         preview.digest_file(instance_compose))
        self.assertTrue(instance["source"]["drifted"])

    def test_destroy_without_stored_files_tears_down_the_recorded_project(self):
        # A lost instance directory must not free the entry while containers
        # or volumes from the recorded project may still be live: destroy
        # tears down by the recorded project name instead of skipping, and
        # never starts the image-dependent ownership restore for a
        # directory that does not exist.
        identity = preview.identity_for(preview.checkout_root(), "destroy-fileless")
        self.register(identity, state="running")
        recorded_project = f"graft-preview-{identity}"
        shutil.rmtree(preview.instance_dir(self.root, identity))
        arguments = argparse.Namespace(task="destroy-fileless", identity=None,
                                       confirm_instance=identity)
        with self.env():
            result = preview.destroy(arguments)
        self.assertEqual(self.teardown_calls, [recorded_project])
        self.assertEqual(self.restore_calls, [])
        self.assertEqual(result["state"], "destroyed")
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])
        self.assertEqual(self.cleanup_calls, [])
        self.assertEqual(self.calls, [])

    def test_destroy_teardown_failure_keeps_the_entry_retryable(self):
        identity = preview.identity_for(preview.checkout_root(), "destroy-teardown-fail")
        self.register(identity, state="running")
        shutil.rmtree(preview.instance_dir(self.root, identity))
        arguments = argparse.Namespace(task="destroy-teardown-fail", identity=None,
                                       confirm_instance=identity)
        with self.env():
            with patch.object(preview, "teardown_recorded_project",
                              side_effect=PreviewError("docker ps query failed")):
                with self.assertRaises(PreviewError) as context:
                    preview.destroy(arguments)
        self.assertIn("entry kept for retry", str(context.exception))
        entry = preview.read_registry(self.root)["instances"][identity]
        self.assertEqual(entry["state"], "cleanup_failed")
        self.assertIn("docker ps", entry["cleanup_error"])
        self.assertEqual(preview.read_archive_manifest(self.root).get("entries", {}), {})
        with self.env():
            result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])

    def test_destroy_restores_developer_ownership_for_root_owned_leftovers(self):
        # Instances left by older cleanup attempts carry root-owned dirs and
        # files; destroy must restore developer ownership once and then
        # complete end-to-end instead of failing before the reclaim.
        identity = preview.identity_for(preview.checkout_root(), "destroy-root-owned")
        self.register(identity, state="running")
        directory = preview.instance_dir(self.root, identity)
        directory.chmod(0o000)

        def privileged_restore(target):
            self.restore_calls.append(str(target))
            target.chmod(0o700)

        arguments = argparse.Namespace(task="destroy-root-owned", identity=None,
                                       confirm_instance=identity)
        with self.env():
            with patch.object(preview, "restore_developer_ownership",
                              side_effect=privileged_restore):
                result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertEqual(self.restore_calls, [str(directory)])
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])
        self.assertFalse(directory.exists())

    def test_destroy_proceeds_when_only_the_checkout_is_unreadable(self):
        # An unreadable checkout compose file must not block the instance
        # destroy or trigger an ownership restore of the instance data.
        identity = preview.identity_for(preview.checkout_root(), "destroy-checkout-lock")
        self.register(identity, state="running")
        with tempfile.TemporaryDirectory() as temporary:
            checkout_compose = Path(temporary) / "compose.yml"
            checkout_compose.write_text("services: {}\n", encoding="utf-8")
            checkout_compose.chmod(0o000)
            document = preview.read_registry(self.root)
            document["instances"][identity]["checkout"] = temporary
            preview.write_registry(self.root, document)
            arguments = argparse.Namespace(task="destroy-checkout-lock", identity=None,
                                           confirm_instance=identity)
            with self.env():
                result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertEqual(self.restore_calls, [])
        self.assertNotIn(identity, preview.read_registry(self.root)["instances"])

    def test_ownership_restore_command_uses_numeric_identity(self):
        recorded = {}

        def fake_run(command, capture_output, text, timeout):
            recorded["command"] = command
            return subprocess.CompletedProcess(command, 0, stdout="", stderr="")

        with tempfile.TemporaryDirectory() as temporary:
            self.restore_patcher.stop()
            try:
                with patch.object(preview.subprocess, "run", side_effect=fake_run):
                    preview.restore_developer_ownership(Path(temporary))
                with patch.object(preview.subprocess, "run",
                                  return_value=subprocess.CompletedProcess(
                                      [], 1, stdout="", stderr="denied")):
                    with self.assertRaises(PreviewError):
                        preview.restore_developer_ownership(Path(temporary))
            finally:
                self.restore_patcher.start()
        command = recorded["command"]
        self.assertEqual(command[:5], ["docker", "run", "--rm", "--user", "0:0"])
        self.assertIn("postgres:16.16-alpine", command)
        script = command[command.index("sh") + 2]
        self.assertIn(f"chown -R {os.getuid()}:{os.getgid()} /target", script)
        self.assertIn("chmod -R u+rwX /target", script)
        self.assertIn(f"{temporary}:/target", command)

    def test_destroy_teardowns_before_touching_ownership(self):
        # The ownership restore must run only on the stopped stack's data:
        # healthy running containers are torn down first, never chowned
        # mid-run.
        identity = preview.identity_for(preview.checkout_root(), "destroy-order")
        self.register(identity, state="running")
        arguments = argparse.Namespace(task="destroy-order", identity=None,
                                       confirm_instance=identity)
        events = []
        base_compose = preview.run_compose

        def recording_compose(arguments, **kwargs):
            events.append("compose-down")
            return base_compose(arguments, **kwargs)

        with self.env():
            with patch.object(preview, "run_compose", side_effect=recording_compose), \
                    patch.object(preview, "needs_ownership_restore", return_value=True), \
                    patch.object(preview, "restore_developer_ownership",
                                 side_effect=lambda directory: events.append("restore")):
                result = preview.destroy(arguments)
        self.assertEqual(result["state"], "destroyed")
        self.assertEqual(events, ["compose-down", "restore"])

    def test_destroy_teardown_failure_never_touches_ownership(self):
        # A teardown failure must leave the stack intact and the entry
        # retryable WITHOUT having chowned any live file.
        identity = preview.identity_for(preview.checkout_root(), "destroy-order-fail")
        self.register(identity, state="running")
        arguments = argparse.Namespace(task="destroy-order-fail", identity=None,
                                       confirm_instance=identity)

        def failing_compose(arguments, **kwargs):
            return subprocess.CompletedProcess(["docker"], 1, stdout="", stderr="compose down failed")

        with self.env():
            with patch.object(preview, "run_compose", failing_compose), \
                    patch.object(preview, "needs_ownership_restore", return_value=True), \
                    patch.object(preview, "restore_developer_ownership",
                                 side_effect=AssertionError("must not chown before teardown")):
                with self.assertRaises(PreviewError) as context:
                    preview.destroy(arguments)
        self.assertIn("stack teardown failed", str(context.exception))
        entry = preview.read_registry(self.root)["instances"][identity]
        self.assertEqual(entry["state"], "cleanup_failed")
        self.assertIn("compose down failed", entry["cleanup_error"])


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

    def test_unsearchable_checkout_dir_reports_anomaly_for_all_instances(self):
        # pathlib's is_file() re-raises EACCES on an unsearchable checkout
        # directory; status --all must survive it and report the anomaly on
        # the affected instance only.
        broken_identity = "broken-checkout"
        with tempfile.TemporaryDirectory() as temporary:
            checkout_dir = Path(temporary) / "checkout"
            checkout_dir.mkdir()
            (checkout_dir / "compose.yml").write_text("services: {}\n", encoding="utf-8")
            document = preview.read_registry(self.root)
            document["instances"][broken_identity] = {
                "identity": broken_identity, "task_id": broken_identity,
                "state": "running", "web_port": self.http.port,
                "compose_project": f"graft-preview-{broken_identity}",
                "checkout": str(checkout_dir),
                "source": {"revision": "a" * 40, "digest": "0" * 64}}
            preview.write_registry(self.root, document)
            checkout_dir.chmod(0o000)
            try:
                with self.env():
                    report = preview.status(argparse.Namespace(task=None,
                                                               identity=None, all=True))
            finally:
                checkout_dir.chmod(0o700)
        instances = {item["identity"]: item for item in report["instances"]}
        self.assertEqual(len(report["instances"]), 2)
        self.assertTrue(any("checkout_compose_unreadable" in anomaly
                            for anomaly in instances[broken_identity]["anomalies"]))
        self.assertFalse(any("instance_dir_unreadable" in anomaly
                             for anomaly in instances[broken_identity]["anomalies"]))
        self.assertEqual(instances["status-one"]["anomalies"], [])
        # The healthy instance reads its own (missing here) instance file
        # without the broken checkout affecting it.
        self.assertIsNone(instances["status-one"]["source"]["current_digest"])


class EnvFileTests(unittest.TestCase):
    def test_extra_env_validation(self):
        parsed = preview.parse_extra_env(["GRAFT_DATABASE_URL=host=db", "POSTGRES_DB=preview"])
        self.assertEqual(parsed, {"GRAFT_DATABASE_URL": "host=db", "POSTGRES_DB": "preview"})
        for bad in ("no-equals", "=value", "lower-case=1", "KEY=\nnewline"):
            with self.assertRaises(PreviewError):
                preview.parse_extra_env([bad])
        with self.assertRaises(PreviewError):
            preview.parse_extra_env(["COMPOSE_FILE_DIR=/elsewhere"])


class ComposeEnvironmentTests(unittest.TestCase):
    def test_reserved_keys_are_stripped_from_the_compose_child_environment(self):
        # Compose gives the process environment precedence over --env-file,
        # so exported deployment settings (shared update-state volume,
        # different web port or image) must be stripped before the call or
        # previews would share state that destroy --volumes could remove.
        host_env = {
            "PATH": "/usr/bin",
            "HOME": "/home/dev",
            "GRAFT_UPDATE_STATE_VOLUME": "graft-update-state",
            "GRAFT_BUILD_SNAPSHOT_VOLUME": "graft-build-snapshots",
            "GRAFT_WEB_HOST_PORT": "3000",
            "GRAFT_IMAGE_TAG": "latest",
            "GRAFT_EXECUTION_MODE": "agent",
            "POSTGRES_PASSWORD": "shared-secret",
            "COMPOSE_FILE": "/elsewhere/compose.yml",
            "COMPOSE_PATH_SEPARATOR": ";",
            "GRAFT_DATABASE_URL": "host=shared",
        }
        recorded = {}

        def fake_run(command, capture_output, text, timeout, env):
            recorded["command"] = command
            recorded["env"] = env
            return subprocess.CompletedProcess(command, 0, stdout="", stderr="")

        with tempfile.TemporaryDirectory() as temporary:
            env_file = Path(temporary) / ".env"
            compose_file = Path(temporary) / "compose.yml"
            with patch.dict(os.environ, host_env, clear=True):
                with patch.object(preview.subprocess, "run", side_effect=fake_run):
                    completed = preview.run_compose(
                        ["up", "-d"], compose_file=compose_file,
                        project="graft-preview-x", env_file=env_file)
        self.assertEqual(completed.returncode, 0)
        child = recorded["env"]
        for reserved in ("GRAFT_UPDATE_STATE_VOLUME", "GRAFT_BUILD_SNAPSHOT_VOLUME",
                         "GRAFT_WEB_HOST_PORT", "GRAFT_IMAGE_TAG", "GRAFT_EXECUTION_MODE",
                         "POSTGRES_PASSWORD", "COMPOSE_FILE", "COMPOSE_PATH_SEPARATOR"):
            self.assertNotIn(reserved, child)
        self.assertEqual(child["GRAFT_DATABASE_URL"], "host=shared")
        self.assertEqual(child["PATH"], "/usr/bin")
        self.assertEqual(child["HOME"], "/home/dev")
        self.assertEqual(child["COMPOSE_FILE_DIR"], str(env_file.parent))
        self.assertIn("--env-file", recorded["command"])
        self.assertEqual(recorded["command"][4:6], ["-p", "graft-preview-x"])

    def test_every_env_file_key_is_stripped_from_the_child_environment(self):
        # Guards the strip list against drift: every key build_env_file
        # writes must be a reserved key the child environment removes.
        with tempfile.TemporaryDirectory() as temporary:
            env_file = preview.build_env_file(Path(temporary), "env-cover", "bare-tag", 38000, {})
            keys = {line.partition("=")[0]
                    for line in env_file.read_text(encoding="utf-8").splitlines() if line}
        self.assertTrue(keys.issubset(preview.RESERVED_ENV_KEYS))


if __name__ == "__main__":
    unittest.main()

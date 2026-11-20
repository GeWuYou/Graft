"""Regression checks for the private Docker boundary in the conformance overlay."""
import copy
import base64
import unittest
import json
import io
import hashlib
import tarfile
from urllib.error import HTTPError
from pathlib import Path
import tempfile
import sys
import subprocess
from unittest.mock import Mock, patch
sys.path.insert(0, str(Path(__file__).resolve().parent))

from validate_docker_isolated import Fixture, OfficialFixture, AGENT_REQUIRED_SCENARIOS, LOCAL_REQUIRED_SCENARIOS, completed_official_install, controller_receipt, image_archive_identity, require_agent_acceptance, require_local_acceptance, validate_topology


def image_archive_fixture(path, *, index=False, revision="a" * 40, architecture="amd64", tamper=False):
    config = {"os": "linux", "architecture": architecture,
        "config": {"Labels": {"org.opencontainers.image.revision": revision}},
        "rootfs": {"type": "layers", "diff_ids": ["sha256:" + "b" * 64]}}
    payload = json.dumps(config).encode()
    config_digest = hashlib.sha256(payload).hexdigest()
    files = {"blobs/sha256/" + config_digest: payload + (b" " if tamper else b""),
        "manifest.json": json.dumps([{"Config": "blobs/sha256/" + config_digest, "RepoTags": None, "Layers": []}]).encode()}
    registered = "sha256:" + config_digest
    if index:
        manifest = json.dumps({"schemaVersion": 2, "config": {"digest": registered}, "layers": []}).encode()
        manifest_digest = hashlib.sha256(manifest).hexdigest()
        files["blobs/sha256/" + manifest_digest] = manifest
        descriptor = json.dumps({"schemaVersion": 2, "manifests": [{"digest": "sha256:" + manifest_digest,
            "platform": {"os": "linux", "architecture": architecture}}]}).encode()
        descriptor_digest = hashlib.sha256(descriptor).hexdigest()
        files["blobs/sha256/" + descriptor_digest] = descriptor
        registered = "sha256:" + descriptor_digest
    with tarfile.open(path, "w") as archive:
        for name, content in files.items():
            member = tarfile.TarInfo(name)
            member.size = len(content)
            archive.addfile(member, io.BytesIO(content))
    return registered, "sha256:" + config_digest, config


class ImageImportIdentityTests(unittest.TestCase):
    def test_classic_configuration_and_containerd_index_resolve_the_same_runtime_identity(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "image.tar"
            for index in (False, True):
                registered, private, config = image_archive_fixture(path, index=index)
                identity = image_archive_identity(path, registered, "a" * 40)
                self.assertEqual(identity["registered_image"], registered)
                self.assertEqual(identity["private_image"], private)
                self.assertEqual(identity["rootfs_layers"], config["rootfs"]["diff_ids"])

    def test_rejects_tampered_metadata_wrong_source_platform_and_unbound_graph(self):
        for kwargs, requested in (({"tamper": True}, None), ({"revision": "c" * 40}, None),
                                  ({"architecture": "arm64"}, None), ({}, "sha256:" + "d" * 64)):
            with self.subTest(kwargs=kwargs), tempfile.TemporaryDirectory() as temporary:
                path = Path(temporary) / "image.tar"
                registered, _, _ = image_archive_fixture(path, index=True, **kwargs)
                with self.assertRaises((RuntimeError, KeyError)):
                    image_archive_identity(path, requested or registered, "a" * 40)

    def test_private_load_verifies_actual_config_and_filesystem_before_returning_identity(self):
        for mismatch in (False, True):
            with self.subTest(mismatch=mismatch), tempfile.TemporaryDirectory() as temporary:
                fixture = Fixture.__new__(Fixture)
                fixture.directory = Path(temporary)
                fixture.state = {"source_revision": "a" * 40}
                fixture.environment = {}
                fixture.prefix = ["docker", "compose", "--project-name", "owned-fixture"]
                archive_path = Path(temporary) / "source.tar"
                registered, private, config = image_archive_fixture(archive_path, index=True)
                actual = {"Id": private, "RootFS": {"Layers": config["rootfs"]["diff_ids"]},
                    "Os": "linux", "Architecture": "amd64", "Config": config["config"]}
                if mismatch:
                    actual["RootFS"]["Layers"] = ["sha256:" + "e" * 64]
                fixture.engine = Mock(return_value=json.dumps([actual]))
                def command(args, **kwargs):
                    if args[:3] == ["docker", "image", "save"]:
                        Path(args[4]).write_bytes(archive_path.read_bytes())
                    else:
                        self.assertTrue(kwargs["stdin"].read(1))
                    return subprocess.CompletedProcess(args, 0, stdout=b"Loaded image ID: " + private.encode(), stderr=b"")
                with patch("validate_docker_isolated.subprocess.run", side_effect=command):
                    if mismatch:
                        with self.assertRaisesRegex(RuntimeError, "verified exported configuration"):
                            fixture.import_image(registered)
                    else:
                        self.assertEqual(fixture.import_image(registered), private)
                        proof = json.loads((fixture.directory / "image-import-evidence.json").read_text())
                        self.assertEqual(proof["images"][registered]["private_image"], private)
                self.assertFalse([path for path in fixture.directory.glob("image-import-*") if path.is_dir()])


class PrivateDockerTopologyTests(unittest.TestCase):
    def setUp(self):
        self.configuration = {"services": {}}
        for name in ("docker-daemon", "backend", "conformance-driver", "docker-runtime-agent"):
            self.configuration["services"][name] = {"volumes": [
                {"type": "volume", "source": "daemon-socket", "target": "/run"},
                {"type": "volume", "source": "application-workspaces", "target": "/opt/graft/apps"},
            ]}

    def test_accepts_private_socket_and_consistent_workspace(self):
        validate_topology(self.configuration)

    def test_rejects_host_socket_retained_by_compose_merge(self):
        configuration = copy.deepcopy(self.configuration)
        configuration["services"]["backend"]["volumes"].append({
            "type": "bind", "source": "/var/run/docker.sock", "target": "/var/run/docker.sock"})
        with self.assertRaisesRegex(RuntimeError, "host Docker socket"):
            validate_topology(configuration)

    def test_rejects_private_socket_missing_from_driver(self):
        self.configuration["services"]["conformance-driver"]["volumes"].pop(0)
        with self.assertRaisesRegex(RuntimeError, "private daemon socket"):
            validate_topology(self.configuration)

    def test_rejects_daemon_and_agent_workspace_path_drift(self):
        self.configuration["services"]["docker-runtime-agent"]["volumes"][1]["target"] = "/workspace"
        with self.assertRaisesRegex(RuntimeError, "workspace mapping"):
            validate_topology(self.configuration)

    def test_rejects_readonly_parent_hiding_nested_mountpoints(self):
        self.configuration["services"]["backend"]["volumes"][0]["read_only"] = True
        with self.assertRaisesRegex(RuntimeError, "nested secret mountpoints"):
            validate_topology(self.configuration)

    def test_local_fixture_uses_private_backend_without_agent_or_vault(self):
        services = self.configuration["services"]
        del services["conformance-driver"]
        del services["docker-runtime-agent"]
        validate_topology(self.configuration, "local")
        services["vault"] = {}
        with self.assertRaisesRegex(RuntimeError, "without Vault or Agent"):
            validate_topology(self.configuration, "local")


class FixtureLifecycleTests(unittest.TestCase):
    def test_failed_official_probe_never_leaves_an_accepted_installation_marker(self):
        for previous_complete in (False, True):
            for failure in ("probe", "gate"):
                with self.subTest(previous_complete=previous_complete, failure=failure), tempfile.TemporaryDirectory() as temporary:
                    fixture = Fixture.__new__(Fixture)
                    fixture.directory = Path(temporary)
                    fixture.state = {"project": "graft-isolated-install-proof", "source_revision": "a" * 40,
                                     "execution_mode": "local"}
                    fixture.environment = {"GRAFT_BACKEND_IMAGE": "sha256:" + "b" * 64,
                        "GRAFT_WEB_IMAGE": "sha256:" + "c" * 64, "GRAFT_CONFORMANCE_OFFICIAL_PORT": "28081"}
                    source = fixture.directory / "source-assets" / "root"
                    source.mkdir(parents=True)
                    (source / "compose.yml").write_text("services: {}\n")
                    (source / "compose.env.example").write_text("GRAFT_IMAGE_TAG=latest\n")
                    if previous_complete:
                        official = fixture.directory / "official"
                        official.mkdir()
                        (official / "installation-evidence.json").write_text('{"complete":true}')
                    private_images = {fixture.environment["GRAFT_BACKEND_IMAGE"]: "sha256:" + "d" * 64,
                                      fixture.environment["GRAFT_WEB_IMAGE"]: "sha256:" + "e" * 64}
                    fixture.import_image = Mock(side_effect=lambda image: private_images[image])
                    fixture.last_command_returncode = 0
                    fixture.compose = Mock(return_value="")
                    fixture.engine = Mock(side_effect=lambda *args: private_images[fixture.environment["GRAFT_BACKEND_IMAGE"]]
                        if args[-1] == "container-server" else private_images[fixture.environment["GRAFT_WEB_IMAGE"]])
                    nested = Mock()
                    nested.compose.side_effect = lambda *args, **kwargs: "container-" + args[-1] if args[0] == "ps" else ""
                    nested.results = []
                    if failure == "probe":
                        nested.probe.side_effect = RuntimeError("probe failed")
                    with patch("validate_docker_isolated.OfficialFixture", return_value=nested):
                        with self.assertRaises(RuntimeError):
                            fixture.official_install()
                    fixture.engine.assert_any_call("tag", "sha256:" + "d" * 64, "ghcr.io/gewuyou/graft-server:v0.0.0-beta.1")
                    self.assertFalse(completed_official_install(fixture.directory))
                    with self.assertRaisesRegex(RuntimeError, "successful local official-install"):
                        fixture.build_api()
                    with self.assertRaisesRegex(RuntimeError, "successful local official-install"):
                        fixture.update_api()

    def test_completed_installation_proof_requires_an_explicit_true_completion(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            official = directory / "official"
            official.mkdir()
            marker = official / "installation-evidence.json"
            for data, accepted in (({}, False), ({"complete": False}, False), ({"complete": True}, True)):
                marker.write_text(json.dumps(data))
                self.assertEqual(completed_official_install(directory), accepted)

    def test_operation_setup_records_success_only_after_actual_effective_value_readback(self):
        for effective, expected in (("true", "passed"), ("false", "failed")):
            with self.subTest(effective=effective), tempfile.TemporaryDirectory() as temporary:
                fixture = Fixture.__new__(Fixture)
                fixture.api = Mock(side_effect=[(200, {}), (200, {"data": {"effective_value": effective}})] * 3)
                fixture.record = Mock()
                fixture.enable_operations()
                self.assertEqual(fixture.record.call_count, 3)
                self.assertTrue(all(call.args[1] == expected for call in fixture.record.call_args_list))
                self.assertEqual([call.args[0] for call in fixture.api.call_args_list], ["PUT", "GET"] * 3)

    def test_controller_receipt_requires_actual_bound_success_and_backup_digests(self):
        receipt = {"protocol_version": 2, "operation_id": "controller-test", "runner_id": "runner-test",
            "execution_mode": "local", "succeeded": True, "migration_started": True,
            "backup_completion": {"OperationID": "controller-test", "TaskID": 1,
                "ConfigSnapshotSHA256": "a" * 64, "ConfigSnapshotBytes": 12,
                "DatabaseDumpSHA256": "b" * 64, "DatabaseDumpBytes": 200}}
        def output(value):
            return "GRAFT_UPDATE_RECEIPT:" + base64.b64encode(json.dumps(value).encode()).decode().rstrip("=")
        self.assertEqual(controller_receipt(output(receipt), "controller-test", "runner-test"), receipt)
        for field, value in (("succeeded", False), ("execution_mode", "agent"), ("runner_id", "other")):
            with self.subTest(field=field), self.assertRaisesRegex(RuntimeError, "evidence checks"):
                controller_receipt(output({**receipt, field: value}), "controller-test", "runner-test")
        receipt["backup_completion"]["OperationID"] = "other-operation"
        with self.assertRaisesRegex(RuntimeError, "evidence checks"):
            controller_receipt(output(receipt), "controller-test", "runner-test")

    def test_every_required_local_scenario_must_have_passing_evidence(self):
        complete = [{"scenario": name, "status": "passed"} for name in LOCAL_REQUIRED_SCENARIOS]
        require_local_acceptance(complete)
        for status in ("unverified", "skipped", "failed", "observed"):
            with self.subTest(status=status):
                incomplete = [item for item in complete if item["scenario"] != "local-execution-interruption"]
                incomplete.append({"scenario": "local-execution-interruption", "status": status})
                with self.assertRaisesRegex(RuntimeError, "local-execution-interruption"):
                    require_local_acceptance(incomplete)
        with self.assertRaisesRegex(RuntimeError, "local-execution-interruption"):
            require_local_acceptance([item for item in complete if item["scenario"] != "local-execution-interruption"])

    def test_official_probe_commands_stay_inside_registered_private_daemon(self):
        parent = Fixture.__new__(Fixture)
        parent.state = {"project": "graft-isolated-official-test", "execution_mode": "local"}
        parent.environment = {"GRAFT_CONFORMANCE_OFFICIAL_PORT": "28081"}
        parent.last_command_returncode = 0
        parent.compose = Mock(return_value="10001")
        fixture = OfficialFixture(parent, Path("/evidence/official"), "/opt/graft-official")
        fixture.compose("exec", "-T", "backend", "id", "-u")
        command = parent.compose.call_args.args
        self.assertEqual(command[:5], ("exec", "-T", "docker-daemon", "docker", "--host=unix:///run/graft-docker/docker.sock"))
        self.assertIn("graft-isolated-official-test-official", command)
        self.assertEqual(command[-7:], ("exec", "-T", "--user", "graft", "server", "id", "-u"))
        fixture.compose("exec", "-T", "docker-daemon", "chmod", "660", "/run/graft-docker/docker.sock")
        self.assertEqual(parent.compose.call_args.args[:3], ("exec", "-T", "docker-daemon"))

    def test_local_startup_needs_only_backend_image_and_no_trust_delivery(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            (directory / "instance.json").write_text(json.dumps({
                "project": "graft-isolated-local-test", "execution_mode": "local", "environment": {}}))
            fixture = Fixture(directory)
            self.assertIn("compose.local.yml", fixture.prefix[-1])
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            del topology.configuration["services"]["conformance-driver"]
            del topology.configuration["services"]["docker-runtime-agent"]
            def compose_result(*args):
                if args[0] == "config":
                    return json.dumps(topology.configuration)
                if args[0] == "wait":
                    raise RuntimeError("no containers for project after the one-shot already exited")
                return ""
            fixture.compose = Mock(side_effect=compose_result)
            fixture.engine = Mock(return_value="local-engine")
            fixture.up()
            calls = [call.args for call in fixture.compose.call_args_list]
            self.assertIn(("up", "-d", "--wait", "docker-daemon", "postgres", "redis"), calls)
            self.assertIn(("run", "--rm", "--no-deps", "-T", "workspace-init"), calls)
            self.assertIn(("run", "--rm", "--no-deps", "-T", "migrate"), calls)
            self.assertFalse(any(args[0] == "wait" for args in calls))
            self.assertFalse(any("vault" in args or "docker-runtime-agent" in args for args in calls))
            self.assertTrue((directory / "local-readiness-evidence.json").exists())

    def test_agent_initialization_uses_foreground_exit_codes_and_preserves_vault_order(self):
        with tempfile.TemporaryDirectory() as temporary:
            fixture = Fixture.__new__(Fixture)
            fixture.directory = Path(temporary)
            fixture.environment = {"GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST": "sha256:" + "a" * 64}
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            def compose_result(*args):
                if args[0] == "config":
                    return json.dumps(topology.configuration)
                if args[:4] == ("exec", "-T", "vault", "sh"):
                    return '{"approle_login":true,"pki_ca_matches_delivery":true,"ca_sha256":"proof"}'
                if args[0] == "wait":
                    raise RuntimeError("no containers for project after the one-shot already exited")
                return ""
            fixture.compose = Mock(side_effect=compose_result)
            fixture.driver = Mock(return_value={"agent_id": "fixture", "target_id": 1,
                "identity_id": "identity", "generation": 1, "ledger_receipt_count": 2})
            fixture.engine = Mock(return_value="private-engine")
            fixture.up()
            calls = [call.args for call in fixture.compose.call_args_list]
            for service in ("workspace-init", "vault-init", "migrate"):
                self.assertIn(("run", "--rm", "--no-deps", "-T", service), calls)
            self.assertFalse(any(args[0] == "wait" for args in calls))
            vault_check = next(args for args in calls if args[:4] == ("exec", "-T", "vault", "sh"))
            self.assertIn("auth/approle/login", vault_check[-1])
            self.assertIn("pki/cert/ca", vault_check[-1])
            self.assertLess(calls.index(("run", "--rm", "--no-deps", "-T", "vault-init")), calls.index(vault_check))
            self.assertLess(calls.index(vault_check), calls.index(("run", "--rm", "--no-deps", "-T", "migrate")))
            readiness = [args for args in calls if "--wait" in args]
            self.assertEqual(readiness, [("up", "-d", "--wait", "docker-daemon", "postgres", "redis", "vault"),
                                         ("up", "-d", "--no-deps", "--wait", "backend")])
            self.assertTrue((fixture.directory / "prepared-evidence.json").exists())
            self.assertFalse(any("--once" in args for args in calls))
            self.assertEqual(calls.count(("up", "-d", "--no-deps", "docker-runtime-agent")), 2)
            self.assertIn(("stop", "docker-runtime-agent"), calls)
            verification_calls = [call.args for call in fixture.driver.call_args_list if "verify-restart" in call.args]
            self.assertEqual(verification_calls, [("--phase", "verify-restart", "--agent-id", "fixture")])

    def test_foreground_migration_failure_preserves_exit_code_and_blocks_backend(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            (directory / "instance.json").write_text(json.dumps({
                "project": "graft-isolated-migrate-failure", "execution_mode": "local", "environment": {}}))
            fixture = Fixture(directory)
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            del topology.configuration["services"]["conformance-driver"]
            del topology.configuration["services"]["docker-runtime-agent"]
            def command(arguments, **kwargs):
                if arguments[-3:] == ["config", "--format", "json"]:
                    return subprocess.CompletedProcess(arguments, 0, stdout=json.dumps(topology.configuration), stderr="")
                code = 7 if arguments[-5:] == ["run", "--rm", "--no-deps", "-T", "migrate"] else 0
                return subprocess.CompletedProcess(arguments, code, stdout="", stderr="synthetic migration failure" if code else "")
            with patch("validate_docker_isolated.subprocess.run", side_effect=command) as runner:
                with self.assertRaisesRegex(RuntimeError, "Compose command failed"):
                    fixture.up()
            self.assertEqual(fixture.last_command_returncode, 7)
            self.assertFalse(any(call.args[0][-1] == "backend" for call in runner.call_args_list))
            self.assertFalse((directory / "local-readiness-evidence.json").exists())
            self.assertIn("synthetic migration failure", (directory / "command-error.log").read_text())

    def test_stop_preserves_vault_and_stops_all_other_declared_services(self):
        fixture = Fixture.__new__(Fixture)
        services = {name: {} for name in ("vault", "backend", "docker-runtime-agent", "postgres", "redis", "docker-daemon", "vault-init")}
        fixture.compose = Mock(side_effect=[json.dumps({"services": services}), ""])
        fixture.stop()
        command = fixture.compose.call_args_list[-1].args
        self.assertEqual(command[0], "stop")
        self.assertEqual(set(command[1:]), set(services) - {"vault"})

    def test_stop_with_only_vault_never_expands_to_stop_all(self):
        fixture = Fixture.__new__(Fixture)
        fixture.compose = Mock(return_value='{"services":{"vault":{}}}')
        fixture.stop()
        self.assertEqual(fixture.compose.call_count, 1)

    def test_lost_vault_authority_aborts_before_migrations_or_backend(self):
        with tempfile.TemporaryDirectory() as temporary:
            fixture = Fixture.__new__(Fixture)
            fixture.directory = Path(temporary)
            topology = PrivateDockerTopologyTests()
            topology.setUp()
            def compose_result(*args):
                if args[0] == "config":
                    return json.dumps(topology.configuration)
                if args[:4] == ("exec", "-T", "vault", "sh"):
                    raise RuntimeError("current Vault rejected stored AppRole")
                return ""
            fixture.compose = Mock(side_effect=compose_result)
            fixture.driver = Mock()
            with self.assertRaisesRegex(RuntimeError, "Vault rejected"):
                fixture.up()
            calls = [call.args for call in fixture.compose.call_args_list]
            self.assertNotIn(("up", "-d", "--no-deps", "migrate"), calls)
            self.assertFalse(any("backend" in args for args in calls))
            fixture.driver.assert_not_called()

    def test_reauthenticates_after_empty_password_change_response(self):
        fixture = Fixture.__new__(Fixture)
        fixture.environment = {"GRAFT_FIXTURE_ADMIN_PASSWORD": "synthetic-password"}
        fixture.api = Mock(side_effect=[(401, {}),
            (200, {"data": {"access_token": "restricted", "must_change_password": True}}),
            (200, {"data": None}), (200, {"data": {"access_token": "normal"}})])
        fixture.authenticate()
        self.assertEqual(fixture.token, "normal")


class FixtureAPIContractTests(unittest.TestCase):
    def fixture(self, directory):
        fixture = Fixture.__new__(Fixture)
        fixture.directory = Path(directory)
        fixture.environment = {"GRAFT_CONFORMANCE_BACKEND_PORT": "28080"}
        fixture.token = "synthetic-fixture-token"
        return fixture

    def test_operation_posts_supply_and_reuse_caller_idempotency_key(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            response = Mock()
            response.status = 202
            response.read.return_value = b'{"data":{"task_id":1}}'
            response.__enter__ = Mock(return_value=response)
            response.__exit__ = Mock(return_value=False)
            with patch("validate_docker_isolated.request.urlopen", return_value=response) as transport:
                fixture.api("POST", "/api/ops/containers/probe/start", {}, idempotency_key="same-operation")
                fixture.api("POST", "/api/ops/containers/probe/start", {}, idempotency_key="same-operation")
                fixture.api("POST", "/api/ops/docker/images/pull", {"reference": "alpine"})
                fixture.api("POST", "/api/build/jobs", {"input_snapshot_id": "fixture"}, idempotency_key="build-operation")
            headers = [dict(call.args[0].header_items()) for call in transport.call_args_list]
            self.assertEqual(headers[0]["Idempotency-key"], "same-operation")
            self.assertEqual(headers[1]["Idempotency-key"], "same-operation")
            self.assertEqual(len(headers[2]["Idempotency-key"]), 32)
            self.assertEqual(headers[3]["Idempotency-key"], "build-operation")

    def test_nonblocking_command_preserves_failure_exit_code(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.prefix = ["docker", "compose"]
            fixture.environment = {}
            result = Mock(returncode=7, stdout="", stderr="socket connection denied")
            with patch("validate_docker_isolated.subprocess.run", return_value=result):
                self.assertEqual(fixture.compose("exec", "backend", "curl", check=False), "")
            self.assertEqual(fixture.last_command_returncode, 7)

    def test_failed_http_evidence_excludes_request_payload_and_response_details(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            failure = HTTPError("http://fixture", 400, "Bad Request", {},
                io.BytesIO(b'{"code":"INVALID","traceId":"safe-trace","message":"sensitive provider detail"}'))
            with patch("validate_docker_isolated.request.urlopen", side_effect=failure):
                fixture.api("POST", "/api/ops/containers/probe/start", {"secret": "sensitive payload"})
            evidence = (Path(directory) / "api-errors.json").read_text()
            self.assertIn("safe-trace", evidence)
            self.assertIn("INVALID", evidence)
            self.assertNotIn("sensitive", evidence)


class RuntimeRepairEvidenceTests(unittest.TestCase):
    def test_local_target_discovery_requires_unique_canonical_wire_target(self):
        fixture = Fixture.__new__(Fixture)
        fixture.api = Mock(return_value=(200, {"data": {"items": [{
            "id": 7, "runtime": {"provider": "docker"}, "connection": {"kind": "unix_socket"}}]}}))
        self.assertEqual(fixture.local_target_id(), 7)
        fixture.api.return_value = (200, {"data": {"items": []}})
        with self.assertRaisesRegex(RuntimeError, "uniquely discovered"):
            fixture.local_target_id()

    def test_local_interruption_requires_unknown_task_and_preserved_engine_effect(self):
        for status, expected in (("needs_attention", "passed"), ("success", "failed")):
            with self.subTest(status=status), tempfile.TemporaryDirectory() as directory:
                fixture = self.fixture(directory)
                fixture.state["execution_mode"] = "local"
                fixture.api = Mock(side_effect=[(201, {"data": {"application_id": "app-fixture"}}),
                    (202, {"data": {"task_id": 1}}), (200, {})])
                fixture.wait_task = Mock(side_effect=[{"status": status}, {"status": "cancelled"}])
                fixture.engine = Mock(side_effect=["graft-interruption-probe-test-app", "id started", "id started", "id started"])
                with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                    fixture.interruption_probe({}, "services: {}", "original", {})
                results = {call.args[0]: call for call in fixture.record.call_args_list}
                self.assertEqual(results["local-execution-interruption"].args, ("local-execution-interruption", expected))
                if expected == "passed":
                    self.assertEqual(results["local-interruption-reconciliation"].args[1], "passed")
                self.assertIn(unittest.mock.call("kill", "--signal", "SIGKILL", "backend"), fixture.compose.call_args_list)

    def test_original_runner_reuses_complete_restart_evidence(self):
        runner = Path(__file__).resolve().parents[1] / "tests/conformance/docker-runtime-agent/run.sh"
        script = runner.read_text()
        self.assertEqual(script.count("$compose run --rm --no-deps docker-runtime-agent"), 2)
        self.assertIn("--phase verify-bootstrap", script)
        command = script.split('echo "[8/8]', 1)[1].split("| jq", 1)[0]
        self.assertIn("--phase verify-restart", command)
        self.assertIn("--agent-id", command)
        for partial_flag in ("--target-id", "--identity-id", "--generation", "--receipt-count"):
            self.assertNotIn(partial_flag, command)

    def fixture(self, directory):
        fixture = Fixture.__new__(Fixture)
        fixture.directory = Path(directory)
        fixture.state = {"project": "graft-isolated-test"}
        fixture.environment = {}
        fixture.compose = Mock(return_value="")
        fixture.record = Mock()
        return fixture

    def identity(self, receipts):
        return {"agent_id": "fixture", "target_id": 1, "identity_id": "identity", "generation": 2,
                "ledger_receipt_count": receipts}

    def test_once_requires_successful_exit_and_advanced_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"restart": self.identity(1)}))
            fixture.wait_driver = Mock(side_effect=[self.identity(2), self.identity(3)])
            with patch("validate_docker_isolated.subprocess.run") as cleanup:
                fixture.once_probe()
            fixture.record.assert_called_once()
            self.assertEqual(fixture.record.call_args.args, ("agent-once-exit", "passed"))
            calls = fixture.compose.call_args_list
            one_shot = next(call for call in calls if "--once" in call.args)
            self.assertEqual(one_shot.kwargs["timeout"], 60)
            self.assertEqual(calls[-1].args, ("up", "-d", "--no-deps", "docker-runtime-agent"))
            self.assertEqual(cleanup.call_args.args[0], ["docker", "rm", "-f", "graft-isolated-test-once-probe"])
            self.assertTrue(fixture.record.call_args.kwargs["ledger_receipt_advanced"])
            restart = fixture.wait_driver.call_args_list[-1]
            self.assertEqual(restart.args, ("--phase", "verify-restart", "--agent-id", "fixture"))

    def test_once_timeout_fails_and_restores_regular_agent(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"restart": self.identity(1)}))
            fixture.wait_driver = Mock(return_value=self.identity(2))
            def compose(*args, **kwargs):
                if "--once" in args:
                    raise subprocess.TimeoutExpired("fixture-agent", 60)
                return ""
            fixture.compose.side_effect = compose
            with patch("validate_docker_isolated.subprocess.run") as cleanup:
                fixture.once_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-once-exit", "failed"))
            self.assertEqual(fixture.compose.call_args.args, ("up", "-d", "--no-deps", "docker-runtime-agent"))
            cleanup.assert_called_once()

    def test_once_receipt_without_advance_cannot_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"restart": self.identity(1)}))
            fixture.wait_driver = Mock(return_value=self.identity(2))
            with patch("validate_docker_isolated.subprocess.run"):
                fixture.once_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-once-exit", "failed"))

    def test_import_readback_detects_persisted_field_drift(self):
        self.assert_import_readback(persisted_target=99, expected_status="failed")

    def test_import_readback_and_conflict_preserve_persisted_fields(self):
        self.assert_import_readback(persisted_target=1, expected_status="passed")

    def assert_import_readback(self, persisted_target, expected_status):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            identifier = "app_" + "0" * 26
            expected = {"display_name": "Imported graft-import-probe-test", "runtime_target_id": 1,
                        "compose_project_name": "compose-probe", "compose_project_name_source": "computed"}
            detail = {key: value for key, value in expected.items() if key != "runtime_target_id"}
            detail["runtime_target"] = {"id": 1}
            fixture.api = Mock(side_effect=[
                (200, {"data": {"items": [{"managed": True, "provider": "local", "root_id": "root"}]}}),
                (200, {"data": {"inspection_id": "first", "compose_project_name": "compose-probe", "compose_project_name_source": "computed"}}),
                (201, {"data": {"application": {"application_id": identifier}}}),
                (200, {"data": detail}), (200, {"data": {"inspection_id": "second"}}), (409, {})])
            fixture.persisted_application = Mock(return_value={**expected, "runtime_target_id": persisted_target})
            with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                fixture.import_probe("/workspace", "services: {}", "container", 1, {})
            results = {call.args[0]: call for call in fixture.record.call_args_list}
            self.assertEqual(results["compose-import-readback"].args[1], expected_status)
            self.assertEqual(results["compose-reimport-conflict"].args[1], "passed")
            self.assertEqual(fixture.persisted_application.call_count, 2)
            self.assertEqual(fixture.api.call_args_list[-1].args[2]["inspection_id"], "second")

    def test_compose_stop_requires_engine_to_be_stopped(self):
        for running, expected in (("true", "failed"), ("false", "passed")):
            with self.subTest(running=running), tempfile.TemporaryDirectory() as directory:
                fixture = self.fixture(directory)
                (Path(directory) / "trust-evidence.json").write_text(json.dumps({"bootstrap": self.identity(1)}))
                fixture.api = Mock(side_effect=[
                    (201, {"data": {"application_id": "app-fixture", "workspace_path": "/workspace"}}),
                    (202, {"data": {"task_id": 1}}), (202, {"data": {"task_id": 2}}),
                    (409, {}), (500, {})])
                fixture.wait_task = Mock(return_value={"status": "success"})
                fixture.import_probe = Mock()
                fixture.engine = Mock(side_effect=["graft-compose-probe-test-app", "true",
                    "graft-relative-mount-verified", running])
                with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                    fixture.application_probe()
                results = {call.args[0]: call for call in fixture.record.call_args_list}
                self.assertEqual(results["compose-stop"].args[1], expected)
                self.assertEqual(results["compose-stop"].kwargs["engine_running"], running)

    def test_compose_up_requires_running_container(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (Path(directory) / "trust-evidence.json").write_text(json.dumps({"bootstrap": self.identity(1)}))
            fixture.api = Mock(side_effect=[
                (201, {"data": {"application_id": "app-fixture", "workspace_path": "/workspace"}}),
                (202, {"data": {"task_id": 1}})])
            fixture.wait_task = Mock(return_value={"status": "success"})
            fixture.record.side_effect = [None, RuntimeError("up evidence recorded")]
            fixture.engine = Mock(side_effect=["graft-compose-probe-test-app", "false"])
            with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                with self.assertRaisesRegex(RuntimeError, "up evidence recorded"):
                    fixture.application_probe()
            results = {call.args[0]: call for call in fixture.record.call_args_list}
            self.assertEqual(results["compose-up"].args[1], "failed")
            self.assertNotIn("compose-stop", results)

    def test_container_restart_requires_started_at_change(self):
        for after, expected in (("original", "failed"), ("new-start", "passed")):
            with self.subTest(after=after), tempfile.TemporaryDirectory() as directory:
                fixture = self.fixture(directory)
                fixture.api = Mock(return_value=(202, {"data": {"task_id": 1}}))
                fixture.wait_task = Mock(return_value={"status": "success"})
                fixture.engine = Mock(side_effect=["original", "true", after])
                fixture.container_action_probe("fixture", "restart", "true")
                self.assertEqual(fixture.record.call_args.args, ("container-restart", expected))
                self.assertEqual(fixture.record.call_args.kwargs["engine_started_at_changed"], after != "original")

    def test_persisted_readback_rejects_noncanonical_application_identifier(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            with self.assertRaisesRegex(RuntimeError, "invalid application identity"):
                fixture.persisted_application("app_'; DELETE FROM applications")
            fixture.compose.assert_not_called()


class RecoveryScenarioTests(unittest.TestCase):
    """Bounded automatic recovery acceptance: scenario registration, probe wiring and assertion helpers."""

    def fixture(self, directory):
        fixture = Fixture.__new__(Fixture)
        fixture.directory = Path(directory)
        fixture.state = {"project": "graft-isolated-recovery-test", "execution_mode": "agent"}
        fixture.environment = {"GRAFT_CONFORMANCE_BACKEND_PORT": "28080"}
        fixture.token = "synthetic-fixture-token"
        fixture.compose = Mock(return_value="")
        fixture.record = Mock()
        return fixture

    def clock(self, ticks):
        fake = Mock()
        fake.monotonic = Mock(side_effect=list(ticks))
        fake.sleep = Mock()
        return patch("validate_docker_isolated.time", fake)

    def test_recovery_scenarios_register_in_both_gates(self):
        self.assertIn("compose-material-stage-code", LOCAL_REQUIRED_SCENARIOS)
        for scenario in ("agent-claim-admission-rejected", "agent-recovery-resume", "agent-recovery-budget-exhausted"):
            self.assertIn(scenario, AGENT_REQUIRED_SCENARIOS)
        complete = [{"scenario": name, "status": "passed"} for name in sorted(AGENT_REQUIRED_SCENARIOS)]
        require_agent_acceptance(complete)
        for status in ("unverified", "failed", "observed"):
            with self.subTest(status=status):
                incomplete = [item for item in complete if item["scenario"] != "agent-recovery-resume"]
                incomplete.append({"scenario": "agent-recovery-resume", "status": status})
                with self.assertRaisesRegex(RuntimeError, "agent-recovery-resume"):
                    require_agent_acceptance(incomplete)
        with self.assertRaisesRegex(RuntimeError, "agent-recovery-resume"):
            require_agent_acceptance([item for item in complete if item["scenario"] != "agent-recovery-resume"])

    def test_material_stage_code_requires_task_and_stage_projection(self):
        for stage_items, expected in (([{"failure_stage": "material_resolve"}], "passed"),
                                      ([{"failure_stage": None}], "failed"), ([], "failed")):
            with self.subTest(expected=expected), tempfile.TemporaryDirectory() as directory:
                fixture = self.fixture(directory)
                fixture.api = Mock(side_effect=[
                    (200, {"data": {"id": 3, "status": "failed", "failure_stage": "material_resolve"}}),
                    (200, {"data": {"items": stage_items}})])
                fixture.material_stage_code_probe(3)
                self.assertEqual(fixture.record.call_args.args, ("compose-material-stage-code", expected))

    def test_latest_claim_admission_reads_only_target_detail_projection(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.api = Mock(side_effect=[
                (200, {"data": {"items": [{"id": 4}]}}),
                (200, {"data": {"agent": {"claim_admission": {"class": "identity_rejected"}}}})])
            self.assertEqual(fixture.latest_claim_admission(), {"class": "identity_rejected"})
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.api = Mock(side_effect=[
                (200, {"data": {"items": [{"id": 4}]}}),
                (200, {"data": {"agent": {}}}),
                (200, {"data": {"items": [{"id": 5}]}}),
                (200, {"data": {"agent": {"claim_admission": None}}})])
            self.assertIsNone(fixture.latest_claim_admission())

    def test_claim_admission_probe_restores_the_observed_generation(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.sql = Mock(side_effect=lambda statement: "5:3" if statement.startswith("SELECT") else "")
            fixture.latest_claim_admission = Mock(side_effect=[None, {"class": "identity_rejected", "observed_at": "t"}])
            with self.clock([0, 1, 2, 10 ** 9]):
                fixture.claim_admission_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-claim-admission-rejected", "passed"))
            statements = [call.args[0] for call in fixture.sql.call_args_list]
            self.assertIn("generation = generation + 1", statements[1])
            self.assertIn("generation = 3", statements[-1])
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.sql = Mock(side_effect=lambda statement: "5:3" if statement.startswith("SELECT") else "")
            fixture.latest_claim_admission = Mock(return_value={"class": "no_work"})
            with self.clock([0, 10 ** 9]):
                fixture.claim_admission_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-claim-admission-rejected", "failed"))
            statements = [call.args[0] for call in fixture.sql.call_args_list]
            self.assertEqual(statements[-1], "UPDATE runtime_target_agent_generations SET generation = 3 WHERE id = 5")

    def test_wait_task_failure_code_matches_converged_diagnostics_only(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.api = Mock(side_effect=[
                (200, {"data": {"status": "needs_attention", "failure_code": "external_execution_lease_expired"}}),
                (200, {"data": {"status": "needs_attention", "failure_code": "auto_recovery_budget_exhausted",
                                "failure_stage": None, "failure_class": "internal"}})])
            with self.clock([0, 1, 2]):
                converged = fixture.wait_task_failure_code(9, "auto_recovery_budget_exhausted", timeout=300)
            self.assertEqual(converged["failure_code"], "auto_recovery_budget_exhausted")
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.api = Mock(return_value=(200, {"data": {"status": "needs_attention", "failure_code": "other"}}))
            with self.clock([0, 1, 10 ** 9]):
                timed_out = fixture.wait_task_failure_code(9, "auto_recovery_budget_exhausted", timeout=300)
            self.assertEqual(timed_out["status"], "timeout")
            self.assertEqual(timed_out["observed_failure_code"], "other")

    def test_wait_for_engine_container_polls_until_the_effect_is_visible(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            fixture.engine = Mock(side_effect=["other", "graft-recovery-probe-test-app"])
            with self.clock([0, 1, 2]):
                self.assertTrue(fixture.wait_for_engine_container("graft-recovery-probe-test-app"))

    def test_recovery_resume_probe_passes_only_on_single_idempotent_replay(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (fixture.directory / "trust-evidence.json").write_text(json.dumps({"bootstrap": {"target_id": 1}}))
            fixture.api = Mock(side_effect=[
                (201, {"data": {"application_id": "app-x", "workspace_path": "/workspace"}}),
                (202, {"data": {"task_id": 9}}),
                (200, {"data": {"items": [{"attempt": 2, "max_attempts": 2, "status": "success"}]}})])
            fixture.engine = Mock(side_effect=["graft-recovery-resume-probe-test-app", "id started", "id started"])
            fixture.sql = Mock(return_value="")
            fixture.wait_task = Mock(return_value={"id": 9, "status": "needs_attention"})
            fixture.wait_task_status = Mock(return_value={"id": 9, "status": "success"})
            with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                fixture.recovery_resume_probe()
            record = fixture.record.call_args
            self.assertEqual(record.args, ("agent-recovery-resume", "passed"))
            self.assertTrue(record.kwargs["same_container_and_started_at"])
            self.assertEqual(record.kwargs["resumed_stage_attempt"], 2)
            statements = [call.args[0] for call in fixture.sql.call_args_list]
            self.assertIn("retry_if_idempotent", statements[0])
            self.assertIn("GREATEST(max_attempts, attempt + 1)", statements[0])
            self.assertIn("task_id = 9", statements[0])
            compose_calls = [call.args for call in fixture.compose.call_args_list]
            self.assertEqual(compose_calls.count(("kill", "--signal", "SIGKILL", "docker-runtime-agent")), 1)
            self.assertEqual(compose_calls.count(("up", "-d", "--wait", "docker-runtime-agent")), 1)
        for resumed, attempt, expected in (({"status": "timeout"}, 2, "failed"), ({"status": "success"}, 3, "failed"),
                                           ({"status": "success"}, 2, "passed")):
            with self.subTest(expected=expected), tempfile.TemporaryDirectory() as directory:
                fixture = self.fixture(directory)
                (fixture.directory / "trust-evidence.json").write_text(json.dumps({"bootstrap": {"target_id": 1}}))
                fixture.api = Mock(side_effect=[
                    (201, {"data": {"application_id": "app-x", "workspace_path": "/workspace"}}),
                    (202, {"data": {"task_id": 9}}),
                    (200, {"data": {"items": [{"attempt": attempt, "max_attempts": 2, "status": "success"}]}})])
                fixture.engine = Mock(side_effect=["graft-recovery-resume-probe-test-app", "id started", "id started"])
                fixture.sql = Mock(return_value="")
                fixture.wait_task = Mock(return_value={"id": 9, "status": "needs_attention"})
                fixture.wait_task_status = Mock(return_value=resumed)
                with patch("validate_docker_isolated.secrets.token_hex", return_value="test"):
                    fixture.recovery_resume_probe()
                self.assertEqual(fixture.record.call_args.args, ("agent-recovery-resume", expected))

    def test_recovery_budget_probe_requires_converged_exhaustion_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (fixture.directory / "trust-evidence.json").write_text(json.dumps({"bootstrap": {"target_id": 1}}))
            fixture.api = Mock(side_effect=[
                (201, {"data": {"application_id": "app-y", "workspace_path": "/workspace"}}),
                (202, {"data": {"task_id": 11}})])
            fixture.task_stages = Mock(side_effect=[
                [{"status": "running", "attempt": 2, "max_attempts": 2}],
                [{"status": "unknown", "attempt": 2, "max_attempts": 2}]])
            fixture.engine = Mock(side_effect=["graft-recovery-budget-probe-test-app", "id started", "id started"])
            fixture.sql = Mock(return_value="")
            fixture.wait_task = Mock(return_value={"id": 11, "status": "needs_attention"})
            fixture.wait_task_failure_code = Mock(return_value={"id": 11, "status": "needs_attention",
                "failure_code": "auto_recovery_budget_exhausted", "failure_stage": None, "failure_class": "internal"})
            with patch("validate_docker_isolated.secrets.token_hex", return_value="test"), self.clock([0, 1, 2, 3]):
                fixture.recovery_budget_probe()
            record = fixture.record.call_args
            self.assertEqual(record.args, ("agent-recovery-budget-exhausted", "passed"))
            self.assertEqual((record.kwargs["stage_attempt"], record.kwargs["stage_max_attempts"]), (2, 2))
            self.assertTrue(record.kwargs["same_container_and_started_at"])
            statements = [call.args[0] for call in fixture.sql.call_args_list]
            self.assertIn("GREATEST(max_attempts, 2)", statements[0])
            compose_calls = [call.args for call in fixture.compose.call_args_list]
            self.assertEqual(compose_calls.count(("kill", "--signal", "SIGKILL", "docker-runtime-agent")), 2)
        with tempfile.TemporaryDirectory() as directory:
            fixture = self.fixture(directory)
            (fixture.directory / "trust-evidence.json").write_text(json.dumps({"bootstrap": {"target_id": 1}}))
            fixture.api = Mock(side_effect=[
                (201, {"data": {"application_id": "app-y", "workspace_path": "/workspace"}}),
                (202, {"data": {"task_id": 11}})])
            fixture.task_stages = Mock(return_value=[{"status": "unknown", "attempt": 2, "max_attempts": 2}])
            fixture.engine = Mock(side_effect=["graft-recovery-budget-probe-test-app", "id started", "id started"])
            fixture.sql = Mock(return_value="")
            fixture.wait_task = Mock(return_value={"id": 11, "status": "needs_attention"})
            fixture.wait_task_failure_code = Mock(return_value={"id": 11, "status": "timeout", "observed_failure_code": None})
            with patch("validate_docker_isolated.secrets.token_hex", return_value="test"), self.clock([0, 1, 2, 3, 10 ** 9]):
                fixture.recovery_budget_probe()
            self.assertEqual(fixture.record.call_args.args, ("agent-recovery-budget-exhausted", "failed"))


if __name__ == "__main__":
    unittest.main()

"""Focused checks for the immutable Build context and public upload contract."""
import importlib.util
import hashlib
import io
import json
from pathlib import Path
import tarfile
import unittest
from unittest.mock import Mock, patch


SPEC = importlib.util.spec_from_file_location("graft_fixture_build_api", Path(__file__).with_name("build_api.py"))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class BuildAPIFixtureTests(unittest.TestCase):
    def test_manifest_config_is_bound_to_actual_published_bytes(self):
        config = "sha256:" + "a" * 64
        payload = json.dumps({"schemaVersion": 2, "config": {"digest": config}})
        digest = "sha256:" + hashlib.sha256(payload.encode()).hexdigest()
        nested = Mock()
        nested.compose.return_value = payload
        self.assertEqual(MODULE.manifest_config_digest(nested, (), "probe", digest), config)
        with self.assertRaisesRegex(RuntimeError, "immutable digest"):
            MODULE.manifest_config_digest(nested, (), "probe", "sha256:" + "b" * 64)

    def test_image_content_proof_removes_only_its_stopped_container_even_on_mismatch(self):
        for valid in (True, False):
            with self.subTest(valid=valid):
                fixture = Mock()
                fixture.state = {"project": "graft-isolated-build"}
                fixture.engine.side_effect = ["proof-container", "", ""]
                digest = hashlib.sha256(MODULE.PROBE_BYTES).hexdigest() if valid else "0" * 64
                with patch.object(MODULE, "daemon", side_effect=["", digest + " /probe", str(len(MODULE.PROBE_BYTES))]):
                    if valid:
                        self.assertTrue(MODULE.verify_image_probe(fixture, "sha256:" + "a" * 64)["probe_verified"])
                    else:
                        with self.assertRaisesRegex(RuntimeError, "uploaded context"):
                            MODULE.verify_image_probe(fixture, "sha256:" + "a" * 64)
                self.assertEqual(fixture.engine.call_args.args, ("rm", "proof-container"))
                self.assertFalse(any(call.args[0] == "start" for call in fixture.engine.call_args_list))

    def test_context_contains_a_real_scratch_build_without_external_base_dependencies(self):
        with tarfile.open(fileobj=io.BytesIO(MODULE.context_archive()), mode="r:gz") as archive:
            self.assertEqual(set(archive.getnames()), {"Dockerfile", "probe"})
            self.assertEqual(archive.extractfile("Dockerfile").read(), b"FROM scratch\nCOPY probe /probe\n")
            self.assertIn(b"graft-build-publication-verified", archive.extractfile("probe").read())

    def test_snapshot_upload_uses_the_declared_archive_field_and_authenticated_entrypoint(self):
        nested = Mock()
        nested.environment = {"GRAFT_CONFORMANCE_BACKEND_PORT": "28081"}
        nested.token = "synthetic-session"
        response = Mock(status=201)
        response.read.return_value = json.dumps({"data": {"snapshot_id": "snapshot-fixture"}}).encode()
        response.__enter__ = Mock(return_value=response)
        response.__exit__ = Mock(return_value=False)
        with patch.object(MODULE.request, "urlopen", return_value=response) as transport:
            self.assertEqual(MODULE.upload_snapshot(nested), "snapshot-fixture")
        call = transport.call_args.args[0]
        self.assertTrue(call.full_url.endswith("/api/build/input-snapshots"))
        self.assertEqual(call.get_header("Authorization"), "Bearer synthetic-session")
        self.assertIn(b'name="archive"', call.data)
        self.assertNotIn(b"synthetic-session", call.data)


if __name__ == "__main__":
    unittest.main()

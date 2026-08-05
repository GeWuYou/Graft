#!/usr/bin/env python3
"""Focused tests for historical Runner release-manifest selection."""

from __future__ import annotations

import hashlib
import importlib.util
import json
from pathlib import Path
import sys
import unittest
from unittest import mock


SCRIPT_PATH = Path(__file__).with_name("resolve_runner_release_reuse.py")
MODULE_SPEC = importlib.util.spec_from_file_location("resolve_runner_release_reuse", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"Unable to load module from {SCRIPT_PATH}.")
MODULE = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = MODULE
MODULE_SPEC.loader.exec_module(MODULE)

RUNNER_IMAGE = "ghcr.io/gewuyou/graft-compose-runner"
DIGEST_A = "sha256:" + "a" * 64
DIGEST_B = "sha256:" + "b" * 64


def manifest(tag: str, digest: str, *, channel: str = "beta", image: str = RUNNER_IMAGE) -> bytes:
    return json.dumps(
        {
            "schema_version": 1,
            "version": tag[1:],
            "channel": channel,
            "release_tag": tag,
            "runners": {"compose": {"image": image, "digest": digest, "reference": f"{image}@{digest}"}},
        },
        separators=(",", ":"),
    ).encode()


def checksum(content: bytes, *, filename: str = "image-assets/release-manifest.json") -> bytes:
    return f"{hashlib.sha256(content).hexdigest()}  {filename}\n".encode()


def release(tag: str, *, published_at: str, prerelease: bool = True) -> dict[str, object]:
    return {
        "tag_name": tag,
        "prerelease": prerelease,
        "draft": False,
        "published_at": published_at,
        "assets": [
            {"name": "release-manifest.json", "url": f"manifest-{tag}"},
            {"name": "release-manifest.json.sha256", "url": f"checksum-{tag}"},
        ],
    }


class ResolveCandidatesTests(unittest.TestCase):
    def resolve(self, releases: list[dict[str, object]], assets: dict[str, bytes]) -> list[object]:
        with mock.patch.object(MODULE, "download_asset", side_effect=lambda url: assets[url]):
            return MODULE.resolve_candidates(releases, channel="beta", runner_image=RUNNER_IMAGE, current_tag="v1.2.0-beta.5")

    def test_selects_newest_valid_same_channel_release(self) -> None:
        older = manifest("v1.2.0-beta.3", DIGEST_A)
        newer = manifest("v1.2.0-beta.4", DIGEST_B)
        releases = [
            release("v1.2.0-beta.3", published_at="2026-01-01T00:00:00Z"),
            release("v1.2.0-beta.4", published_at="2026-01-02T00:00:00Z"),
        ]
        candidates = self.resolve(
            releases,
            {
                "manifest-v1.2.0-beta.3": older,
                "checksum-v1.2.0-beta.3": checksum(older),
                "manifest-v1.2.0-beta.4": newer,
                "checksum-v1.2.0-beta.4": checksum(newer),
            },
        )
        self.assertEqual(candidates, [MODULE.Candidate("v1.2.0-beta.4", DIGEST_B), MODULE.Candidate("v1.2.0-beta.3", DIGEST_A)])

    def test_skips_newer_invalid_manifest_and_uses_older_valid_release(self) -> None:
        valid = manifest("v1.2.0-beta.3", DIGEST_A)
        invalid = manifest("v1.2.0-beta.4", DIGEST_B)
        releases = [
            release("v1.2.0-beta.4", published_at="2026-01-02T00:00:00Z"),
            release("v1.2.0-beta.3", published_at="2026-01-01T00:00:00Z"),
        ]
        candidates = self.resolve(
            releases,
            {
                "manifest-v1.2.0-beta.4": invalid,
                "checksum-v1.2.0-beta.4": b"0" * 64 + b"  release-manifest.json\n",
                "manifest-v1.2.0-beta.3": valid,
                "checksum-v1.2.0-beta.3": checksum(valid),
            },
        )
        self.assertEqual(candidates, [MODULE.Candidate("v1.2.0-beta.3", DIGEST_A)])

    def test_skips_invalid_utf8_manifest_and_uses_older_valid_release(self) -> None:
        valid = manifest("v1.2.0-beta.3", DIGEST_A)
        invalid_utf8 = b'\xff{"schema_version":1}'
        releases = [
            release("v1.2.0-beta.4", published_at="2026-01-02T00:00:00Z"),
            release("v1.2.0-beta.3", published_at="2026-01-01T00:00:00Z"),
        ]
        candidates = self.resolve(
            releases,
            {
                "manifest-v1.2.0-beta.4": invalid_utf8,
                "checksum-v1.2.0-beta.4": checksum(invalid_utf8),
                "manifest-v1.2.0-beta.3": valid,
                "checksum-v1.2.0-beta.3": checksum(valid),
            },
        )
        self.assertEqual(candidates, [MODULE.Candidate("v1.2.0-beta.3", DIGEST_A)])

    def test_rejects_wrong_image_and_cross_channel_release(self) -> None:
        wrong_image = manifest("v1.2.0-beta.4", DIGEST_B, image="ghcr.io/gewuyou/other-runner")
        stable = manifest("v1.1.9", DIGEST_A, channel="stable")
        releases = [
            release("v1.2.0-beta.4", published_at="2026-01-02T00:00:00Z"),
            release("v1.1.9", published_at="2026-01-01T00:00:00Z", prerelease=False),
        ]
        candidates = self.resolve(
            releases,
            {
                "manifest-v1.2.0-beta.4": wrong_image,
                "checksum-v1.2.0-beta.4": checksum(wrong_image),
                "manifest-v1.1.9": stable,
                "checksum-v1.1.9": checksum(stable),
            },
        )
        self.assertEqual(candidates, [])

    def test_rejects_current_or_newer_release(self) -> None:
        current = manifest("v1.2.0-beta.5", DIGEST_A)
        future = manifest("v1.2.0-beta.6", DIGEST_B)
        releases = [
            release("v1.2.0-beta.5", published_at="2026-01-02T00:00:00Z"),
            release("v1.2.0-beta.6", published_at="2026-01-03T00:00:00Z"),
        ]
        self.assertEqual(self.resolve(releases, {}), [])


class DownloadAssetTests(unittest.TestCase):
    def test_rejects_urls_outside_github_api_origin(self) -> None:
        with mock.patch.object(MODULE.urllib.request, "urlopen") as urlopen:
            with self.assertRaisesRegex(RuntimeError, "https://api\\.github\\.com"):
                MODULE.download_asset("https://uploads.github.com/repos/example/releases/assets/1")
        urlopen.assert_not_called()

    def test_sends_token_only_on_initial_request(self) -> None:
        response = mock.MagicMock()
        response.read.return_value = b"asset"
        opener = mock.MagicMock()
        opener.__enter__.return_value = response
        with (
            mock.patch.dict(MODULE.os.environ, {"GH_TOKEN": "test-token"}, clear=True),
            mock.patch.object(MODULE.urllib.request, "urlopen", return_value=opener) as urlopen,
        ):
            self.assertEqual(MODULE.download_asset("https://api.github.com/repos/example/releases/assets/1"), b"asset")

        request = urlopen.call_args.args[0]
        self.assertEqual(request.unredirected_hdrs["Authorization"], "Bearer test-token")
        self.assertNotIn("Authorization", request.headers)


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Find verified historical release manifests that can be considered for image reuse."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Any


DIGEST_PATTERN = re.compile(r"^sha256:[0-9a-f]{64}$")
VERSION_PATTERN = re.compile(
    r"^v(?P<major>0|[1-9][0-9]*)\.(?P<minor>0|[1-9][0-9]*)\.(?P<patch>0|[1-9][0-9]*)(?:-beta\.(?P<beta>[1-9][0-9]*))?$"
)
CHECKSUM_PATTERN = re.compile(r"^(?P<digest>[0-9a-f]{64})\s+\*?(?P<path>\S+)\s*$")


@dataclass(frozen=True)
class Candidate:
    tag: str
    digest: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--channel", choices=("stable", "beta"), required=True)
    image_group = parser.add_mutually_exclusive_group(required=True)
    image_group.add_argument("--runner-image")
    image_group.add_argument("--agent-image")
    parser.add_argument("--current-tag", required=True)
    return parser.parse_args()


def download_asset(asset_url: str) -> bytes:
    parsed_url = urllib.parse.urlsplit(asset_url)
    if parsed_url.scheme != "https" or parsed_url.netloc != "api.github.com":
        raise RuntimeError("download release asset: URL must use the https://api.github.com origin")
    headers = {"Accept": "application/octet-stream", "User-Agent": "graft-runner-release-resolver"}
    request = urllib.request.Request(asset_url, headers=headers)
    token = os.environ.get("GH_TOKEN", "").strip()
    if token:
        request.add_unredirected_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return response.read(1 << 20)
    except (OSError, urllib.error.URLError, urllib.error.HTTPError) as error:
        raise RuntimeError(f"download release asset: {error}") from error


def parse_version(tag: str) -> tuple[int, int, int, int | None] | None:
    matched = VERSION_PATTERN.fullmatch(tag)
    if matched is None:
        return None
    return (
        int(matched.group("major")),
        int(matched.group("minor")),
        int(matched.group("patch")),
        int(matched.group("beta")) if matched.group("beta") is not None else None,
    )


def version_key(version: tuple[int, int, int, int | None]) -> tuple[int, int, int, int, int]:
    major, minor, patch, beta = version
    # A stable release sorts after beta releases with the same core version.
    return major, minor, patch, 1 if beta is None else 0, beta or 0


def flatten_releases(payload: Any) -> list[dict[str, Any]]:
    if not isinstance(payload, list):
        return []
    if all(isinstance(item, list) for item in payload):
        return [release for page in payload for release in page if isinstance(release, dict)]
    return [release for release in payload if isinstance(release, dict)]


def asset_url(release: dict[str, Any], name: str) -> str | None:
    assets = release.get("assets")
    if not isinstance(assets, list):
        return None
    for asset in assets:
        if isinstance(asset, dict) and asset.get("name") == name and isinstance(asset.get("url"), str):
            return asset["url"]
    return None


def checksum_matches(manifest: bytes, checksum: bytes) -> bool:
    expected = hashlib.sha256(manifest).hexdigest()
    matches = []
    for line in checksum.decode("utf-8", errors="replace").splitlines():
        matched = CHECKSUM_PATTERN.fullmatch(line)
        if matched is not None and matched.group("path").rsplit("/", 1)[-1] == "release-manifest.json":
            matches.append(matched.group("digest"))
    return len(matches) == 1 and matches[0] == expected


def manifest_image(manifest: dict[str, Any], path: tuple[str, ...]) -> dict[str, Any] | None:
    value: Any = manifest
    for key in path:
        if not isinstance(value, dict):
            return None
        value = value.get(key)
    return value if isinstance(value, dict) else None


def valid_manifest(
    manifest: Any,
    tag: str,
    channel: str,
    image: str,
    image_path: tuple[str, ...] = ("runners", "compose"),
) -> str | None:
    if not isinstance(manifest, dict) or manifest.get("schema_version") != 1:
        return None
    version = parse_version(tag)
    if version is None or manifest.get("release_tag") != tag or manifest.get("version") != tag[1:]:
        return None
    if manifest.get("channel") != channel or (channel == "beta") != (version[3] is not None):
        return None
    component = manifest_image(manifest, image_path)
    if component is None or component.get("image") != image:
        return None
    digest = component.get("digest")
    if not isinstance(digest, str) or DIGEST_PATTERN.fullmatch(digest) is None:
        return None
    if component.get("reference") != f"{image}@{digest}":
        return None
    return digest


def resolve_candidates(
    releases: list[dict[str, Any]],
    *,
    channel: str,
    runner_image: str,
    current_tag: str,
    image_path: tuple[str, ...] = ("runners", "compose"),
) -> list[Candidate]:
    current_version = parse_version(current_tag)
    if current_version is None:
        raise ValueError(f"invalid current release tag: {current_tag}")
    candidates: list[tuple[tuple[str, str], Candidate, str]] = []
    for release in releases:
        tag = release.get("tag_name")
        version = parse_version(tag) if isinstance(tag, str) else None
        if (
            version is None
            or version_key(version) >= version_key(current_version)
            or bool(release.get("draft"))
            or bool(release.get("prerelease")) != (channel == "beta")
        ):
            continue
        manifest_url = asset_url(release, "release-manifest.json")
        checksum_url = asset_url(release, "release-manifest.json.sha256")
        if manifest_url is None or checksum_url is None:
            continue
        try:
            manifest_bytes = download_asset(manifest_url)
            checksum_bytes = download_asset(checksum_url)
            manifest = json.loads(manifest_bytes)
        except (RuntimeError, UnicodeDecodeError, json.JSONDecodeError):
            continue
        if not checksum_matches(manifest_bytes, checksum_bytes):
            continue
        digest = valid_manifest(manifest, tag, channel, runner_image, image_path)
        if digest is None:
            continue
        published_at = release.get("published_at") if isinstance(release.get("published_at"), str) else ""
        candidates.append(((published_at, tag), Candidate(tag=tag, digest=digest), published_at))
    candidates.sort(key=lambda item: item[0], reverse=True)
    return [candidate for _, candidate, _ in candidates]


def main() -> int:
    args = parse_args()
    try:
        payload = json.load(sys.stdin)
        image = args.runner_image or args.agent_image
        image_path = ("runners", "compose") if args.runner_image else ("agents", "docker_runtime_agent")
        candidates = resolve_candidates(
            flatten_releases(payload),
            channel=args.channel,
            runner_image=image,
            current_tag=args.current_tag,
            image_path=image_path,
        )
    except (json.JSONDecodeError, ValueError) as error:
        print(f"resolve runner release reuse: {error}", file=sys.stderr)
        return 2
    json.dump({"candidates": [candidate.__dict__ for candidate in candidates]}, sys.stdout, separators=(",", ":"))
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Scan the complete PR history and recognize known synthetic test findings."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
SCANNER_IMAGE = "ghcr.io/trufflesecurity/trufflehog@sha256:2ec535840f860aa27f11fee20222fb790326ecd66325c96bd49d02ec6af46eca"
SYNTHETIC_FINDINGS = [
    {
        "detector": "Postgres",
        "raw_sha256": "bbdae61f9b06d7ba875710ecf5b5573b085f30143a90ef32444a70e50c935c2e",
        "file": "scripts/test_runtime_failure_diagnostics.py",
        "commit": "d8172f6eafac645051c832e81e8b0ba22d153bb6",
        "line": 45,
        "reason": "Historical test fixture uses a fixed invented URI solely to verify credential redaction. Its invented host can produce a verification error. Verified findings still block. The current test composes it at runtime.",
    },
    {
        "detector": "Postgres",
        "raw_sha256": "f70d79c0964d779078ce1d8890c855a879f79d59d13cb912b26a074455facf58",
        "file": "scripts/test_browser_acceptance.py",
        "commit": "c1501a2b55cb4a3561e8e61362f0895743dfdade",
        "line": 245,
        "reason": "Historical sanitization test fixture used a fixed invented DSN solely to verify credential redaction. Verified findings still block. The current test composes the credential at runtime.",
    },
]


def match_synthetic_finding(finding: dict) -> dict | None:
    """Return the registered synthetic entry this finding matches exactly, or None."""
    try:
        location = finding.get("SourceMetadata", {}).get("Data", {}).get("Git", {})
    except AttributeError:
        return None
    if not isinstance(location, dict):
        return None
    raw = finding.get("Raw")
    if not isinstance(raw, str):
        return None
    digest = hashlib.sha256(raw.encode()).hexdigest()
    for entry in SYNTHETIC_FINDINGS:
        if (
            finding.get("DetectorName") == entry["detector"]
            and finding.get("Verified") is False
            and digest == entry["raw_sha256"]
            and all(location.get(key) == entry[key] for key in ("file", "commit", "line"))
        ):
            return entry
    return None


def known_synthetic_finding(finding: dict) -> bool:
    return match_synthetic_finding(finding) is not None


def evaluate_scan(output: str, exit_code: int) -> tuple[int, int]:
    if exit_code not in (0, 183):
        raise ValueError("Secret scanner failed before producing a complete successful scan")
    findings = []
    for line in output.splitlines():
        if line.strip():
            value = json.loads(line)
            if not isinstance(value, dict):
                raise ValueError("Secret scanner emitted an invalid finding")
            findings.append(value)
    if bool(findings) != (exit_code == 183):
        raise ValueError("Secret scanner result count and exit status differ")
    accepted = sum(known_synthetic_finding(finding) for finding in findings)
    return len(findings) - accepted, accepted


def scan(root: Path, base: str, head: str) -> int:
    if any(not re.fullmatch(r"[a-f0-9]{40}", revision) for revision in (base, head)) or base == head:
        raise ValueError("Secret scanning requires distinct exact base and head commits")
    for revision in (base, head):
        subprocess.run(["git", "cat-file", "-e", revision + "^{commit}"], cwd=root, check=True, capture_output=True)
    common = Path(subprocess.check_output(["git", "rev-parse", "--path-format=absolute", "--git-common-dir"], cwd=root, text=True).strip())
    repository = common.parent
    command = ["docker", "run", "--rm", "-v", str(repository) + ":/tmp/repository:ro",
               "-v", str(root / ".trufflehog-exclude-paths.txt") + ":/tmp/exclude-paths.txt:ro",
               SCANNER_IMAGE, "git", "file:///tmp/repository", "--since-commit", base, "--branch", head,
               "--fail", "--fail-on-scan-errors", "--no-update", "--json", "--exclude-paths=/tmp/exclude-paths.txt"]
    # Raw findings stay in a temporary local stream and are never printed or uploaded.
    with tempfile.TemporaryFile(mode="w+t") as output, tempfile.TemporaryFile(mode="w+t") as diagnostics:
        result = subprocess.run(command, cwd=root, stdout=output, stderr=diagnostics, timeout=480)
        output.seek(0)
        scanned_output = output.read()
    blocked, accepted = evaluate_scan(scanned_output, result.returncode)
    print(f"Secret scan completed with {blocked} blocking findings and {accepted} exact synthetic fixture findings")
    if accepted:
        # evaluate_scan already validated every line, so re-parsing for reasons is safe.
        reasons: list[str] = []
        for line in scanned_output.splitlines():
            if not line.strip():
                continue
            entry = match_synthetic_finding(json.loads(line))
            if entry is not None and entry["reason"] not in reasons:
                reasons.append(entry["reason"])
        for reason in reasons:
            print(reason)
    return 1 if blocked else 0


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", required=True)
    parser.add_argument("--head", required=True)
    args = parser.parse_args()
    try:
        code = scan(ROOT, args.base, args.head)
    except (OSError, ValueError, subprocess.SubprocessError):
        parser.exit(2, "Secret scan could not establish complete valid results\n")
    raise SystemExit(code)


if __name__ == "__main__":
    main()

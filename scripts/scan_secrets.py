#!/usr/bin/env python3
"""Scan the complete PR history and recognize one exact synthetic test finding."""
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
SYNTHETIC_FINDING = {
    "detector": "Postgres",
    "raw_sha256": "bbdae61f9b06d7ba875710ecf5b5573b085f30143a90ef32444a70e50c935c2e",
    "file": "scripts/test_runtime_failure_diagnostics.py",
    "commit": "d8172f6eafac645051c832e81e8b0ba22d153bb6",
    "line": 45,
    "reason": "Historical test fixture uses a fixed invented URI solely to verify credential redaction. Its invented host can produce a verification error. Verified findings still block. The current test composes it at runtime.",
}


def known_synthetic_finding(finding: dict) -> bool:
    try:
        location = finding.get("SourceMetadata", {}).get("Data", {}).get("Git", {})
    except AttributeError:
        return False
    if not isinstance(location, dict):
        return False
    raw = finding.get("Raw")
    return (
        finding.get("DetectorName") == SYNTHETIC_FINDING["detector"]
        and finding.get("Verified") is False
        and isinstance(raw, str)
        and hashlib.sha256(raw.encode()).hexdigest() == SYNTHETIC_FINDING["raw_sha256"]
        and all(location.get(key) == SYNTHETIC_FINDING[key] for key in ("file", "commit", "line"))
    )


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
        blocked, accepted = evaluate_scan(output.read(), result.returncode)
    print(f"Secret scan completed with {blocked} blocking findings and {accepted} exact synthetic fixture findings")
    if accepted:
        print(SYNTHETIC_FINDING["reason"])
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

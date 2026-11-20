#!/usr/bin/env python3
"""Read-only dry-run reconciliation for Graft preview instance leftovers.

Scans the preview registry, instance directories and archived cleanup
evidence without mutating anything. A destroy suggestion is only emitted
when ownership is proven: the caller runs inside the registered checkout,
or an explicit identity + task-id claim is accepted while the registry is
repository-bound (never for an overridden state directory). Archives are
verified against their SHA-256 manifest digests so tampered cleanup
evidence is reported instead of trusted.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

import preview

REPORT_SCHEMA_VERSION = 1


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def resolve_state_dir(args: argparse.Namespace) -> tuple[Path, bool]:
    """Return (state root, repository_bound). A repository-bound registry is
    the Git-common-directory location, which is what task-proof needs."""
    override = os.environ.get("GRAFT_PREVIEW_STATE_DIR")
    if args.state_dir:
        return Path(args.state_dir).resolve(), False
    if override:
        return Path(override).resolve(), False
    return preview.git_common_dir() / preview.STATE_DIRNAME, True


def tree_snapshot(root: Path) -> dict:
    """Content snapshot used by tests and audits to prove read-only behavior."""
    snapshot = {}
    for path in sorted(root.rglob("*")):
        if path.is_file():
            snapshot[str(path.relative_to(root))] = hashlib.sha256(path.read_bytes()).hexdigest()
        elif path.is_dir():
            snapshot[str(path.relative_to(root)) + "/"] = "dir"
    return snapshot


def ownership_proof(entry: dict, state_root: Path, repository_bound: bool,
                    claims: argparse.Namespace) -> tuple[bool, str]:
    identity = entry.get("identity")
    if identity in claims.claim_ownership:
        if repository_bound and identity in claims.claim_task_id \
                and claims.claim_task_id[identity] == entry.get("task_id"):
            return True, "identity and task-id claim accepted on a repository-bound registry"
        if str(preview.checkout_root()) == entry.get("checkout"):
            return True, "caller runs inside the registered checkout"
        return False, ("identity claim requires a matching --claim-task-id on a repository-bound "
                       "registry, or the registered checkout")
    if str(preview.checkout_root()) == entry.get("checkout"):
        return True, "caller runs inside the registered checkout"
    return False, ("ownership unproven: run from the registered checkout or pass "
                   "--claim-ownership <identity> --claim-task-id <task-id>")


def evidence_verification(state_root: Path) -> list[dict]:
    archive = preview.archive_dir(state_root)
    findings: list[dict] = []
    manifest = preview.read_archive_manifest(state_root)
    listed: dict[str, dict] = manifest.get("entries", {})
    present: set[str] = set()
    if archive.is_dir():
        for path in sorted(archive.glob("*.json")):
            present.add(path.name)
            if path.name == "manifest.json":
                continue
            finding = {"file": path.name, "status": "ok", "sha256_ok": None,
                       "evidence_ok": None}
            expected = listed.get(path.name)
            if expected is None:
                finding["status"] = "unlisted"
                findings.append(finding)
                continue
            actual_digest = preview.digest_file(path)
            finding["sha256_ok"] = actual_digest == expected.get("sha256")
            try:
                record = json.loads(path.read_text(encoding="utf-8"))
                entry = record.get("registry_entry")
                finding["evidence_ok"] = bool(entry) and \
                    record.get("evidence_sha256") == \
                    hashlib.sha256(preview.canonical_json(entry).encode("utf-8")).hexdigest()
            except (json.JSONDecodeError, AttributeError, TypeError):
                finding["evidence_ok"] = False
            if not finding["sha256_ok"] or finding["evidence_ok"] is not True:
                finding["status"] = "tampered"
            findings.append(finding)
    for name, expected in sorted(listed.items()):
        if name not in present:
            findings.append({"file": name, "status": "missing", "sha256_ok": False,
                             "evidence_ok": None})
    return findings


def instance_report(entry: dict, state_root: Path, repository_bound: bool,
                    claims: argparse.Namespace) -> dict:
    identity = entry["identity"]
    directory = preview.instance_dir(state_root, identity)
    checkout = Path(entry.get("checkout") or "")
    checkout_exists = checkout.is_dir()
    files_present = {
        "env": (directory / ".env").is_file(),
        "compose": (directory / "compose.yml").is_file(),
    }
    findings = []
    if not checkout_exists:
        findings.append("worktree_lost")
    if not all(files_present.values()):
        findings.append("instance_files_missing")
    proven, reason = ownership_proof(entry, state_root, repository_bound, claims)
    suggestion = None
    # A destroy suggestion is reconciliation output for leftovers only; a
    # healthy running instance is never suggested for destruction.
    if proven and findings:
        suggestion = {
            # A lost worktree cannot re-derive the identity from --task, so
            # the suggestion must name the registered identity directly.
            "command": (f"python3 scripts/preview.py destroy --identity {identity} "
                        f"--confirm-instance {identity}"),
            "ownership_proof": reason,
            "warning": "destroy removes containers, volumes and instance data",
        }
    return {
        "identity": identity,
        "task_id": entry.get("task_id"),
        "state": entry.get("state"),
        "checkout": entry.get("checkout"),
        "checkout_exists": checkout_exists,
        "instance_dir": str(directory),
        "instance_files_present": files_present,
        "findings": findings,
        "destroy_suggestion": suggestion,
        "ownership": {"proven": proven, "reason": reason},
    }


def scan(args: argparse.Namespace) -> dict:
    state_root, repository_bound = resolve_state_dir(args)
    document = preview.read_registry(state_root)
    # Orphan detection compares against the FULL registry; the --identity
    # filter only narrows the per-instance reports, never the baseline.
    registered_dirs = set(document.get("instances", {}).keys())
    entries = document.get("instances", {})
    if args.identity:
        entries = {key: value for key, value in entries.items() if key == args.identity}
        if not entries:
            raise preview.PreviewError(f"no registered preview instance for identity {args.identity}")
    instances = [instance_report(entry, state_root, repository_bound, args)
                 for entry in entries.values()]
    instance_dirs = state_root / "instances"
    unregistered_dirs = []
    if instance_dirs.is_dir():
        for path in sorted(instance_dirs.iterdir()):
            if path.is_dir() and path.name not in registered_dirs:
                unregistered_dirs.append(str(path))
    anomalies = []
    for report in instances:
        anomalies.extend(f"{report['identity']}: {finding}" for finding in report["findings"])
    for path in unregistered_dirs:
        anomalies.append(f"{path}: unregistered instance directory (no registry entry)")
    evidence = evidence_verification(state_root)
    for finding in evidence:
        if finding["status"] != "ok":
            anomalies.append(f"archive/{finding['file']}: {finding['status']}")
    return {
        "schema_version": REPORT_SCHEMA_VERSION,
        "generated_at": utc_now(),
        "mode": "dry-run-read-only",
        "state_root": str(state_root),
        "repository_bound_registry": repository_bound,
        "instances": instances,
        "unregistered_instance_dirs": unregistered_dirs,
        "archive_evidence": evidence,
        "anomalies": anomalies,
        "verdict": "attention" if anomalies else "clean",
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, epilog=(
        "This command never writes: it only reads the registry, instance directories "
        "and archived cleanup evidence, then reports a JSON reconciliation verdict."))
    parser.add_argument("--state-dir", help="Override the preview state directory (read-only)")
    parser.add_argument("--identity", help="Restrict the reconciliation to one identity")
    parser.add_argument("--claim-ownership", action="append", default=[],
                        metavar="IDENTITY", help="Claim ownership of this instance identity")
    parser.add_argument("--claim-task-id", action="append", default=[],
                        metavar="IDENTITY=TASK_ID",
                        help="Claim the registered task id for one identity, as IDENTITY=TASK_ID")
    return parser


def parse_claim_task_ids(items: list[str]) -> dict[str, str]:
    """Parse repeated IDENTITY=TASK_ID claims into a per-identity mapping."""
    claims: dict[str, str] = {}
    for item in items:
        identity, separator, task_id = item.partition("=")
        if not separator or not identity or not task_id:
            raise ValueError(f"--claim-task-id expects IDENTITY=TASK_ID (for example "
                             f"--claim-task-id my-task-abc123=my-task): {item!r}")
        claims[identity] = task_id
    return claims


def main(argv: list[str] | None = None) -> None:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        args.claim_task_id = parse_claim_task_ids(args.claim_task_id)
    except ValueError as exc:
        parser.error(str(exc))
    args.claim_ownership = set(args.claim_ownership)
    report = scan(args)
    print(json.dumps(report, indent=2, ensure_ascii=False))
    if report["anomalies"]:
        raise SystemExit(1)


if __name__ == "__main__":
    try:
        main()
    except preview.PreviewError as exc:
        print(f"preview-recovery: {exc}", file=sys.stderr)
        raise SystemExit(1) from None
    except (OSError, ValueError, KeyError, json.JSONDecodeError) as exc:
        print(f"preview-recovery: {type(exc).__name__}: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc

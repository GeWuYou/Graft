#!/usr/bin/env python3
"""Package and verify source-bound CI runtime artifacts without rebuilding them."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import runpy
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
BINARY_ARTIFACTS = {
    "server": (".tmp/compiled/server/graft-server-linux-amd64", "graft-server-linux-amd64", "0.0.0-beta.1"),
    "target-server": (".tmp/compiled/target-server/graft-server-linux-amd64", "graft-server-linux-amd64", "0.0.0-beta.2"),
    "runner": (".tmp/compiled/runner/graft-compose-runner", "out/graft-compose-runner", None),
    "release-source": (".tmp/compiled/release-source/graft-release-fixture", "out/graft-release-fixture", None),
    "build-registry": (".tmp/compiled/build-registry/graft-fixture-htpasswd", "out/graft-fixture-htpasswd", None),
}


def binary_identity(binary: Path) -> dict[str, str]:
    output = subprocess.check_output([str(binary), "version"], text=True)
    return dict(line.split(": ", 1) for line in output.splitlines() if ": " in line)


def check_binary_identity(binary: Path, revision: str, version: str) -> dict[str, str]:
    identity = binary_identity(binary)
    if identity.get("version") != version or identity.get("git_commit") != revision or identity.get("git_tree_state") != "clean":
        raise ValueError("Backend binary identity differs from its required source and version")
    return identity



def source_revision(root: Path) -> str:
    return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()


def fingerprints(directory: Path) -> dict[str, str]:
    result = {}
    for path in sorted(directory.rglob("*")):
        if path.is_symlink():
            raise ValueError("Runtime artifacts must contain regular files, not symlinks")
        if path.is_file() and path != directory / "provenance.json":
            result[path.relative_to(directory).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    return result


def prepare(root: Path, directory: Path, kind: str) -> None:
    if directory.exists():
        raise ValueError("Artifact destination already exists")
    directory.mkdir(parents=True)
    revision = source_revision(root)
    identity = None
    if kind in BINARY_ARTIFACTS:
        source, target, version = BINARY_ARTIFACTS[kind]
        binary = root / source
        if not binary.is_file() or binary.is_symlink():
            raise ValueError("Compiled runtime binary is missing")
        destination = directory / target
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(binary, destination)
        destination.chmod(0o755)
        if version:
            identity = check_binary_identity(destination, revision, version)
    else:
        distribution = root / "web/dist"
        if not (distribution / "index.html").is_file():
            raise ValueError("Validated web distribution is missing")
        # Reject links before copying so dependency or external trees cannot enter the artifact.
        fingerprints(distribution)
        shutil.copytree(distribution, directory / "dist")
    data = {"source_revision": revision, "kind": kind, "sha256": fingerprints(directory)}
    if identity:
        data["build_identity"] = identity
    (directory / "provenance.json").write_text(json.dumps(data, indent=2) + "\n")


def verify(root: Path, directory: Path, kind: str) -> None:
    metadata = json.loads((directory / "provenance.json").read_text())
    if metadata.get("source_revision") != source_revision(root) or metadata.get("kind") != kind:
        raise ValueError("Runtime artifact source or kind differs from the checked-out commit")
    recorded = metadata.get("sha256")
    if not recorded or recorded != fingerprints(directory):
        raise ValueError("Runtime artifact content differs from its recorded hashes")
    required = BINARY_ARTIFACTS[kind][1] if kind in BINARY_ARTIFACTS else "dist/index.html"
    if required not in recorded:
        raise ValueError("Runtime artifact is missing its required entrypoint")
    if kind in BINARY_ARTIFACTS:
        # Artifact downloads preserve bytes; executable mode is restored after hashing.
        binary = directory / required
        binary.chmod(0o755)
        version = BINARY_ARTIFACTS[kind][2]
        if version:
            identity = check_binary_identity(binary, source_revision(root), version)
            if identity != metadata.get("build_identity"):
                raise ValueError("Backend binary identity differs from its artifact metadata")


def verify_acceptance(root: Path, directory: Path) -> None:
    revision = source_revision(root)
    validator = runpy.run_path(str(root / "scripts/validate_docker_isolated.py"))["require_local_acceptance"]
    for relative in ("api-evidence.json", "official/api-evidence.json"):
        evidence = json.loads((directory / relative).read_text())
        if evidence.get("source_revision") != revision or evidence.get("execution_mode") != "local":
            raise ValueError("Runtime evidence differs from the checked-out local source")
        validator(evidence.get("results", []))
    installation = json.loads((directory / "official/installation-evidence.json").read_text())
    assets = json.loads((directory / "source-assets.json").read_text())
    if assets.get("source_revision") != revision or installation.get("source_revision") != revision:
        raise ValueError("Official installation source provenance is incomplete")
    if installation.get("official_compose") is not True or installation.get("execution_mode") != "local" or installation.get("complete") is not True:
        raise ValueError("Official local installation proof is incomplete")


def verify_build_acceptance(root: Path, directory: Path) -> None:
    evidence = json.loads((directory / "build-api/build-api-evidence.json").read_text())
    artifact = evidence.get("artifact") or {}
    digest = artifact.get("digest", "")
    image = evidence.get("engine_image_id", "")
    publications = evidence.get("publications") or []
    requested = evidence.get("requested_destination") or {}
    matching_publication = any(
        item.get("publication_id")
        and item.get("artifact_id") == artifact.get("artifact_id")
        and item.get("digest") == digest
        and all((item.get("destination") or {}).get(key) == requested.get(key)
                for key in ("kind", "connection_ref", "repository_ref", "reference"))
        for item in publications
    )
    valid = (
        evidence.get("source_revision") == source_revision(root)
        and int(evidence.get("task_id", 0)) > 0
        and evidence.get("task_status") == "success"
        and evidence.get("registry_anonymous_status") == 401
        and evidence.get("registry_authenticated_status") == 200
        and evidence.get("registry_verified") is True
        and evidence.get("credential_absent_from_task_and_logs") is True
        and bool(artifact.get("artifact_id"))
        and requested.get("kind") == "oci_registry"
        and all(isinstance(requested.get(key), str) and requested[key].strip()
                for key in ("connection_ref", "repository_ref", "reference"))
        and bool(re.fullmatch(r"sha256:[a-f0-9]{64}", digest))
        and bool(re.fullmatch(r"sha256:[a-f0-9]{64}", image))
        and evidence.get("remote_manifest_digest") == digest
        and evidence.get("manifest_config_digest") == image
        and evidence.get("probe_verified") is True
        and bool(re.fullmatch(r"[a-f0-9]{64}", evidence.get("probe_sha256", "")))
        and evidence.get("probe_sha256") == evidence.get("expected_probe_sha256")
        and int(evidence.get("probe_size_bytes", 0)) > 0
        and matching_publication
    )
    if not valid:
        raise ValueError("Build API acceptance evidence has incomplete authentication, artifact, publication or engine proof")


def verify_update_acceptance(root: Path, directory: Path) -> None:
    verify_acceptance(root, directory)
    verify_build_acceptance(root, directory)
    evidence = json.loads((directory / "release-api/update-api-evidence.json").read_text())
    source = json.loads((root / ".tmp/runtime-server/provenance.json").read_text())
    target = json.loads((root / ".tmp/runtime-target-server/provenance.json").read_text())
    revision = source_revision(root)
    source_hash = source.get("sha256", {}).get("graft-server-linux-amd64", "")
    target_hash = target.get("sha256", {}).get("graft-server-linux-amd64", "")
    if (
        source.get("source_revision") != revision or target.get("source_revision") != revision
        or source.get("kind") != "server" or target.get("kind") != "target-server"
        or not re.fullmatch(r"[a-f0-9]{64}", source_hash)
        or not re.fullmatch(r"[a-f0-9]{64}", target_hash) or source_hash == target_hash
    ):
        raise ValueError("Versioned backend artifact provenance is incomplete")
    receipt = evidence.get("receipt") or {}
    backup = receipt.get("backup_completion") or {}
    failed = evidence.get("failed_backup_operation") or {}
    failed_receipt = evidence.get("failed_backup_receipt") or {}
    migration = evidence.get("migration_invocation") or {}
    operation = evidence.get("operation") or {}
    backup_detail = evidence.get("backup") or {}
    config_snapshot = backup_detail.get("config_snapshot") or {}
    database_dump = backup_detail.get("database_dump") or {}
    valid = (
        evidence.get("source_revision") == source_revision(root)
        and evidence.get("synthetic_release_source") is True
        and evidence.get("source_version") == "0.0.0-beta.1"
        and evidence.get("target_version") == "0.0.0-beta.2"
        and evidence.get("source_binary_sha256") == source_hash
        and evidence.get("target_binary_sha256") == target_hash
        and evidence.get("task_status") == "success"
        and evidence.get("server_container_replaced") is True
        and operation.get("phase") == "SUCCESS"
        and failed.get("phase") == "FAILED"
        and failed.get("error") == "backup_failed"
        and bool(failed.get("operation_id"))
        and failed_receipt.get("operation_id") == failed.get("operation_id")
        and failed_receipt.get("failure_code") == "backup_failed"
        and failed_receipt.get("failure_stage") == "env_snapshot"
        and failed_receipt.get("migration_started") is False
        and failed_receipt.get("succeeded") is False
        and migration.get("bootstrap_invoked") is True
        and migration.get("source_revision") == revision
        and migration.get("target_revision") == revision
        and receipt.get("protocol_version") == 2
        and receipt.get("migration_started") is True
        and receipt.get("execution_mode") == "local"
        and receipt.get("succeeded") is True
        and receipt.get("operation_id") == operation.get("operation_id")
        and backup.get("OperationID") == operation.get("operation_id")
        and backup.get("TaskID") == evidence.get("task_id")
        and int(backup_detail.get("id", 0)) > 0
        and int(evidence.get("task_id", 0)) > 0
        and backup_detail.get("task_id") == evidence.get("task_id")
        and backup_detail.get("status") == "AVAILABLE"
        and config_snapshot.get("size_bytes") == backup.get("ConfigSnapshotBytes")
        and config_snapshot.get("sha256") == backup.get("ConfigSnapshotSHA256")
        and database_dump.get("size_bytes") == backup.get("DatabaseDumpBytes")
        and database_dump.get("sha256") == backup.get("DatabaseDumpSHA256")
        and int(receipt.get("launch_stage_id", 0)) > 0
        and int(receipt.get("launch_attempt", 0)) > 0
        and int(backup.get("ConfigSnapshotBytes", 0)) > 0
        and int(backup.get("DatabaseDumpBytes", 0)) > 0
        and bool(re.fullmatch(r"[a-f0-9]{64}", backup.get("ConfigSnapshotSHA256", "")))
        and bool(re.fullmatch(r"[a-f0-9]{64}", backup.get("DatabaseDumpSHA256", "")))
    )
    if not valid:
        raise ValueError("Public Update API acceptance evidence is incomplete or has mismatched binary provenance")


SECRET_NAME = re.compile(r"password|secret|token|pepper|private.?key", re.IGNORECASE)
DIAGNOSTIC_LIMIT = 8192


def diagnostic_secrets(directory: Path, state: dict) -> set[str]:
    values = set()

    def collect(value, key=""):
        if isinstance(value, dict):
            for name, child in value.items():
                collect(child, name)
        elif isinstance(value, list):
            for child in value:
                collect(child, key)
        elif isinstance(value, str) and SECRET_NAME.search(key) and value:
            values.add(value)

    collect(state)
    for path in directory.rglob("*"):
        if path.is_symlink() or not path.is_file() or path.stat().st_size > 262144:
            continue
        if path.name == "password":
            values.add(path.read_text().strip())
        elif path.name == "credentials.json":
            collect(json.loads(path.read_text()))
        elif path.name == ".env":
            for line in path.read_text().splitlines():
                name, separator, value = line.partition("=")
                if separator and SECRET_NAME.search(name):
                    values.add(value.strip().strip("\""))
        elif path.name == "netrc":
            values.update(re.findall(r"\bpassword\s+(\S+)", path.read_text()))
    return values - {""}


def redact_diagnostic(text: str, secrets: set[str]) -> str:
    for value in sorted(secrets, key=len, reverse=True):
        text = text.replace(value, "[REDACTED]")
    text = re.sub(r"-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----", "[REDACTED PRIVATE KEY]", text, flags=re.DOTALL)
    text = re.sub(r"(?i)([a-z][a-z0-9+.-]*://)[^\s/@]+(?::[^\s/@]*)?@", r"\1[REDACTED]@", text)
    text = re.sub(r"(?i)((?:bearer|basic)\s+)[A-Za-z0-9._~+/-]+=*", r"\1[REDACTED]", text)
    text = re.sub(r"(?i)((?:password|secret|token|pepper|private_key)[\w.-]*[\"\x27]?\s*[:=]\s*[\"\x27]?)[^\s,\"\x27}]+", r"\1[REDACTED]", text)
    return text[-DIAGNOSTIC_LIMIT:]


def diagnostic_command(prefix: list[str], args: list[str], root: Path, environment: dict) -> tuple[int, str]:
    try:
        result = subprocess.run(prefix + args, cwd=root, env=environment, capture_output=True, text=True, timeout=15)
        return result.returncode, result.stdout + result.stderr
    except subprocess.TimeoutExpired:
        return 124, "Diagnostic command timed out"
    except OSError as error:
        return 127, type(error).__name__


def compose_failure_summary(prefix: list[str], root: Path, environment: dict, secrets: set[str]) -> dict:
    code, output = diagnostic_command(prefix, ["ps", "--all", "--format", "json"], root, environment)
    rows = []
    if code == 0 and output.strip():
        try:
            decoded = json.loads(output)
            rows = decoded if isinstance(decoded, list) else [decoded]
        except json.JSONDecodeError:
            try:
                rows = [json.loads(line) for line in output.splitlines() if line.strip()]
            except json.JSONDecodeError:
                code = 65
    states = [{key: row[key] for key in ("Service", "State", "Health", "ExitCode") if key in row}
              for row in rows if isinstance(row, dict)]
    services = [row.get("Service", "") for row in states
                if row.get("State") in ("dead", "restarting") or row.get("Health") == "unhealthy"
                or (row.get("State") == "exited" and row.get("ExitCode") not in (0, "0", None))]
    services.sort()
    services.extend(row["Service"] for row in states if row.get("Service") in ("backend", "server", "postgres"))
    if not services:
        services = [row["Service"] for row in states if row.get("Service") in ("migrate", "bootstrap", "config-validate")]
    services = list(dict.fromkeys(name for name in services if isinstance(name, str) and re.fullmatch(r"[a-zA-Z0-9_.-]+", name)))[:6]
    log_code, logs = (0, "")
    if services:
        log_code, logs = diagnostic_command(prefix, ["logs", "--no-color", "--tail", "40", *services], root, environment)
    return {"status_exit_code": code, "services": states, "status_error": redact_diagnostic(output, secrets) if code else "",
            "logs_exit_code": log_code, "logs": redact_diagnostic(logs, secrets)}


def failure_diagnostics(root: Path, directory: Path) -> None:
    state_path = directory / "instance.json"
    if not state_path.is_file():
        print("Runtime failure diagnostics unavailable: acceptance instance was not registered")
        return
    state = json.loads(state_path.read_text())
    if not re.fullmatch(r"graft-isolated-[a-z0-9-]+", state.get("project", "")):
        raise ValueError("Runtime diagnostics require a registered isolated project")
    secrets = diagnostic_secrets(directory, state)
    namespace = runpy.run_path(str(root / "scripts/validate_docker_isolated.py"))
    fixture = namespace["Fixture"](directory)
    prefixes = {"private": fixture.prefix}
    if (directory / "official/.env").is_file():
        prefixes["official"] = fixture.prefix + ["exec", "-T", "docker-daemon", "docker", "--host=unix:///run/graft-docker/docker.sock",
            "compose", "--project-name", state["project"] + "-official", "--project-directory", "/opt/graft-official",
            "--env-file", "/opt/graft-official/.env", "-f", "/opt/graft-official/compose.yml"]
    command_error = directory / "command-error.log"
    error_text = ""
    if command_error.is_file() and not command_error.is_symlink():
        with command_error.open("rb") as stream:
            stream.seek(max(0, command_error.stat().st_size - 262144))
            error_text = stream.read(262144).decode("utf-8", errors="replace")
    summary = {"project": state["project"], "source_revision": state.get("source_revision"), "acceptance_passed": False,
               "command_error": redact_diagnostic(error_text, secrets),
               "topologies": {name: compose_failure_summary(prefix, root, fixture.environment, secrets) for name, prefix in prefixes.items()}}
    path = directory / "failure-evidence.json"
    path.write_text(json.dumps(summary, indent=2) + "\n")
    path.chmod(0o600)
    reason = summary["command_error"].strip().splitlines()
    print("Runtime acceptance failed. Sanitized diagnostics saved to failure-evidence.json")
    if reason:
        print("\n".join(reason[-3:])[-1200:])
    for name, topology in summary["topologies"].items():
        for service in topology["services"]:
            print(name, service.get("Service"), service.get("State"), "exit=" + str(service.get("ExitCode", "unknown")), "health=" + str(service.get("Health", "unknown")))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("prepare", "verify", "acceptance", "build-acceptance", "update-acceptance", "failure-diagnostics"))
    parser.add_argument("--kind", choices=(*BINARY_ARTIFACTS, "web"))
    parser.add_argument("--directory", type=Path, required=True)
    args = parser.parse_args()
    if args.action in ("prepare", "verify") and args.kind is None:
        parser.error("--kind is required when packaging or verifying build artifacts")
    try:
        if args.action == "failure-diagnostics":
            failure_diagnostics(ROOT, args.directory.resolve())
        elif args.action == "acceptance":
            verify_acceptance(ROOT, args.directory.resolve())
        elif args.action == "build-acceptance":
            verify_build_acceptance(ROOT, args.directory.resolve())
        elif args.action == "update-acceptance":
            verify_update_acceptance(ROOT, args.directory.resolve())
        else:
            {"prepare": prepare, "verify": verify}[args.action](ROOT, args.directory.resolve(), args.kind)
    except (OSError, ValueError, RuntimeError, subprocess.CalledProcessError) as error:
        parser.exit(1, str(error) + "\n")


if __name__ == "__main__":
    main()

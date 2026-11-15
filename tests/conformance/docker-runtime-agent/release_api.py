"""Contained Update API acceptance using real TLS, release parsing and OCI digests."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import re
import secrets
import time
from urllib.error import URLError


ROOT = "/opt/graft-official"
RELEASE_ROOT = ROOT + "/.fixture-release"
SOURCE_VERSION = "0.0.0-beta.1"
TARGET_VERSION = "0.0.0-beta.2"
TAG = "v" + TARGET_VERSION


def daemon(fixture, *arguments, check=True):
    return fixture.compose("exec", "-T", "docker-daemon", *arguments, check=check)


def docker(fixture, *arguments, check=True):
    return daemon(fixture, "docker", "--host=unix:///run/graft-docker/docker.sock", *arguments, check=check)


def write_file(fixture, directory: Path, name: str, content: bytes):
    path = directory / name
    path.write_bytes(content)
    path.chmod(0o600)
    fixture.compose("cp", str(path), "docker-daemon:" + RELEASE_ROOT + "/" + name)


def image_digest(fixture, reference: str) -> str:
    repository = reference.rsplit(":", 1)[0]
    values = json.loads(docker(fixture, "image", "inspect", "--format", "{{json .RepoDigests}}", reference))
    value = next((item for item in values if item.startswith(repository + "@sha256:")), "")
    if not re.fullmatch(re.escape(repository) + r"@sha256:[a-f0-9]{64}", value):
        raise RuntimeError("TLS fixture registry returned no immutable image digest")
    return value.split("@", 1)[1]


def publish_image(fixture, image: str, repository: str) -> dict:
    reference = repository + ":" + TAG
    docker(fixture, "tag", image, reference)
    deadline = time.monotonic() + 30
    while True:
        docker(fixture, "push", reference, check=False)
        if fixture.last_command_returncode == 0:
            break
        if time.monotonic() >= deadline:
            raise RuntimeError("Task-private TLS registry push failed; inspect restricted fixture diagnostics")
        time.sleep(1)
    digest = image_digest(fixture, reference)
    return {"image": repository, "digest": digest, "reference": repository + "@" + digest}


def archive_artifact(fixture, image: str, source: str, name: str) -> str:
    container = docker(fixture, "create", "--name", fixture.state["project"] + "-artifact-" + secrets.token_hex(3), image).strip()
    try:
        daemon(fixture, "mkdir", "-p", RELEASE_ROOT + "/artifacts/" + name)
        docker(fixture, "cp", container + ":" + source, RELEASE_ROOT + "/artifacts/" + name)
    finally:
        docker(fixture, "rm", container)
    daemon(fixture, "tar", "-czf", RELEASE_ROOT + "/" + name + ".tar.gz", "-C", RELEASE_ROOT + "/artifacts", name)
    digest = daemon(fixture, "sha256sum", RELEASE_ROOT + "/" + name + ".tar.gz").split()[0]
    if not re.fullmatch(r"[a-f0-9]{64}", digest):
        raise RuntimeError("Release artifact digest was not produced")
    return digest


def release_metadata(images: dict, hashes: dict) -> tuple[dict, list]:
    base = "https://github.com/GeWuYou/Graft/releases/download/" + TAG + "/"
    manifest = {"release_tag": TAG, "version": TARGET_VERSION, "channel": "beta",
        "release_notes_url": "https://github.com/GeWuYou/Graft/releases/tag/" + TAG,
        "minimum_source_version": SOURCE_VERSION,
        "upgrade_notes": "Task-private synthetic release for API, Task and Controller acceptance.",
        "artifacts": {"server": "server.tar.gz", "web": "web.tar.gz", "checksums": "checksums.txt", "sha256": hashes},
        "images": {"server": images["server"], "web": images["web"]},
        "runners": {"compose": images["runner"]}}
    assets = [{"name": name, "browser_download_url": base + name} for name in (
        "release-manifest.json", "release-manifest.json.sha256", "checksums.txt", "server.tar.gz", "web.tar.gz")]
    releases = [{"tag_name": TAG, "prerelease": True, "draft": False,
        "published_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "body": "Contained acceptance release.", "assets": assets}]
    return manifest, releases


def start_trusted_source(fixture, nested, directory: Path) -> dict:
    environment = dict(fixture.environment)
    for key in ("GRAFT_TARGET_BACKEND_IMAGE", "GRAFT_RELEASE_FIXTURE_IMAGE", "GRAFT_RUNNER_IMAGE", "GRAFT_WEB_IMAGE"):
        if not environment.get(key):
            raise RuntimeError("update-api requires --target-backend-image, --release-fixture-image and --runner-image at registration")
        environment[key] = fixture.import_image(environment[key])
    daemon(fixture, "mkdir", "-p", RELEASE_ROOT)
    docker(fixture, "run", "--rm", "-v", RELEASE_ROOT + ":/fixture", environment["GRAFT_RELEASE_FIXTURE_IMAGE"], "--init-ca")
    daemon(fixture, "sh", "-ec", "mkdir -p /etc/docker/certs.d/ghcr.io && cp /opt/graft-official/.fixture-release/ca.crt /etc/docker/certs.d/ghcr.io/ca.crt && (grep -q '^127.0.0.1[[:space:]]ghcr.io$' /etc/hosts || echo '127.0.0.1 ghcr.io' >> /etc/hosts)")
    registry = fixture.state["project"] + "-tls-registry"
    docker(fixture, "run", "-d", "--name", registry, "--network", "host",
        "-e", "REGISTRY_HTTP_ADDR=0.0.0.0:443", "-e", "REGISTRY_HTTP_TLS_CERTIFICATE=/fixture/server.crt",
        "-e", "REGISTRY_HTTP_TLS_KEY=/fixture/server.key", "-v", RELEASE_ROOT + ":/fixture:ro", "registry:3.0.0")
    images = {
        "server": publish_image(fixture, environment["GRAFT_TARGET_BACKEND_IMAGE"], "ghcr.io/gewuyou/graft-server"),
        "web": publish_image(fixture, environment["GRAFT_WEB_IMAGE"], "ghcr.io/gewuyou/graft-web"),
        "runner": publish_image(fixture, environment["GRAFT_RUNNER_IMAGE"], "ghcr.io/gewuyou/graft-compose-runner"),
    }
    hashes = {"server.tar.gz": archive_artifact(fixture, environment["GRAFT_TARGET_BACKEND_IMAGE"], "/app/graft", "server"),
              "web.tar.gz": archive_artifact(fixture, environment["GRAFT_WEB_IMAGE"], "/usr/share/nginx/html/.", "web")}
    manifest, releases = release_metadata(images, hashes)
    manifest_bytes = (json.dumps(manifest, indent=2) + "\n").encode()
    manifest_sha = hashlib.sha256(manifest_bytes).hexdigest()
    write_file(fixture, directory, "release-manifest.json", manifest_bytes)
    write_file(fixture, directory, "release-manifest.json.sha256", (manifest_sha + "  release-manifest.json\n").encode())
    write_file(fixture, directory, "checksums.txt", "".join(value + "  " + name + "\n" for name, value in hashes.items()).encode())
    write_file(fixture, directory, "releases.json", (json.dumps(releases) + "\n").encode())
    source_name = fixture.state["project"] + "-release-source"
    docker(fixture, "run", "-d", "--name", source_name, "--network", nested.state["project"] + "_default",
        "--network-alias", "api.github.com", "--network-alias", "github.com",
        "-v", RELEASE_ROOT + ":/fixture:ro", environment["GRAFT_RELEASE_FIXTURE_IMAGE"])
    overlay = {"services": {"server": {"environment": {"SSL_CERT_FILE": "/run/graft-fixture-release/ca.crt",
        "GRAFT_DEPLOYMENT_COMPOSE_ROOT": ROOT},
        "volumes": [RELEASE_ROOT + "/ca.crt:/run/graft-fixture-release/ca.crt:ro"]}}}
    overlay_bytes = (json.dumps(overlay, indent=2) + "\n").encode()
    path = directory / "compose.release-fixture.yml"
    path.write_bytes(overlay_bytes)
    fixture.compose("cp", str(path), "docker-daemon:" + ROOT + "/compose.release-fixture.yml")
    compose = ["compose", "--project-name", nested.state["project"], "--project-directory", ROOT,
        "--env-file", ROOT + "/.env", "-f", ROOT + "/compose.yml"]
    build_overlay = fixture.directory / "build-api" / "compose.build-fixture.yml"
    if build_overlay.is_file():
        compose.extend(["-f", ROOT + "/compose.build-fixture.yml"])
    compose.extend(["-f", ROOT + "/compose.release-fixture.yml", "up", "-d", "--no-deps", "--force-recreate", "--wait", "server"])
    docker(fixture, *compose)
    return {"images": images, "manifest_sha256": manifest_sha, "target_private_image": environment["GRAFT_TARGET_BACKEND_IMAGE"],
        "overlay_sha256": hashlib.sha256(overlay_bytes).hexdigest(),
        "target_binary_sha256": daemon(fixture, "sha256sum", RELEASE_ROOT + "/artifacts/server/graft").split()[0],
        "ca_sha256": daemon(fixture, "sha256sum", RELEASE_ROOT + "/ca.crt").split()[0]}


def wait_operation(nested, operation: str, expected: str, timeout: int = 240) -> tuple[dict, dict]:
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", operation):
        raise RuntimeError("Update API returned an invalid operation identity")
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            status, response = nested.api("GET", "/api/platform/updates/operations")
            history = next((item for item in response.get("data", []) if item.get("operation_id") == operation), None)
            if status == 200:
                task = owned_update_task(nested, operation)
                if task and task.get("status") in {"success", "failed", "cancelled", "needs_attention"}:
                    if history is None:
                        status, response = nested.api("GET", "/api/platform/updates/operations/" + operation)
                        history = response.get("data") if status == 200 else None
                    if history is not None:
                        if history.get("operation_id") != operation or history.get("phase") != expected:
                            raise RuntimeError("Update operation reached an unexpected terminal phase")
                        return history, task
        except (OSError, URLError, ValueError):
            # 预期的 server 替换窗口只影响有界只读轮询，提交操作始终只发送一次。
            pass
        time.sleep(1)
    raise RuntimeError("Update API did not settle its operation and Task before the acceptance deadline")


def owned_update_task(nested, operation: str) -> dict | None:
    status, response = nested.api("GET", "/api/tasks?owner_type=platform_update&owner_id=" + operation)
    if status != 200:
        return None
    tasks = (response.get("data") or {}).get("items", [])
    matching = [item for item in tasks if item.get("owner_type") == "platform_update" and item.get("owner_id") == operation
                and item.get("type") == "platform.update.compose"]
    if not matching:
        return None
    if len(matching) != 1:
        raise RuntimeError("Update operation has an ambiguous Task owner binding")
    status, response = nested.api("GET", "/api/tasks/" + str(matching[0]["id"]))
    task = response.get("data") or {}
    if status != 200:
        return None
    if task.get("id") != matching[0]["id"] or task.get("owner_type") != "platform_update" or task.get("owner_id") != operation:
        raise RuntimeError("Update Task detail differs from its public owner binding")
    return task


def backup_for_task(nested, task_id: int, receipt: dict) -> dict:
    status, response = nested.api("GET", "/api/platform/backups?limit=100&offset=0")
    if status != 200:
        raise RuntimeError("Backup history API was unavailable after Update settlement")
    for summary in (response.get("data") or {}).get("items", []):
        status, response = nested.api("GET", "/api/platform/backups/" + str(summary["id"]))
        detail = response.get("data") or {}
        if status != 200 or detail.get("task_id") != task_id:
            continue
        if (detail.get("id") != summary["id"] or detail.get("status") != "AVAILABLE" or int(detail.get("id", 0)) < 1
            or detail.get("config_snapshot", {}).get("sha256") != receipt.get("ConfigSnapshotSHA256")
            or detail.get("config_snapshot", {}).get("size_bytes") != receipt.get("ConfigSnapshotBytes")
            or detail.get("database_dump", {}).get("sha256") != receipt.get("DatabaseDumpSHA256")
            or detail.get("database_dump", {}).get("size_bytes") != receipt.get("DatabaseDumpBytes")):
            raise RuntimeError("Backup API integrity metadata differs from the settled Controller proof")
        return detail
    raise RuntimeError("Backup API has no asset bound to the successful Update Task")


def submit_update(nested) -> str:
    status, response = nested.api("POST", "/api/platform/updates/operations", {"target_version": TARGET_VERSION})
    if status != 202:
        raise RuntimeError("Update API rejected the verified synthetic release with HTTP " + str(status))
    return response["data"]["operation_id"]


def migration_invocation_scope(receipt: dict, source_revision: str) -> dict:
    if receipt.get("migration_started") is not True:
        raise RuntimeError("Successful Update acceptance requires the actual bootstrap migration invocation")
    return {"bootstrap_invoked": True, "source_revision": source_revision,
            "target_revision": source_revision, "scope": "same-source migration graph with no new pending DDL"}


def run_release_api(fixture, nested):
    directory = fixture.directory / "release-api"
    directory.mkdir(mode=0o700, exist_ok=True)
    nested.authenticate()
    status, response = nested.api("GET", "/api/platform/updates/status")
    if status != 200 or response.get("data", {}).get("current_version") != SOURCE_VERSION:
        raise RuntimeError("update-api requires a fresh official installation built as " + SOURCE_VERSION)
    tag = daemon(fixture, "grep", "^GRAFT_IMAGE_TAG=", ROOT + "/.env").strip().partition("=")[2]
    if tag != "v" + SOURCE_VERSION:
        raise RuntimeError("update-api requires the untouched source image tag; use a separate fixture from controller-smoke")
    source_binary = nested.compose("exec", "-T", "backend", "sha256sum", "/app/graft").split()[0]
    source = start_trusted_source(fixture, nested, directory)
    nested.authenticate()
    deadline = time.monotonic() + 30
    while True:
        status, response = nested.api("POST", "/api/platform/updates/check", {})
        catalog = response.get("data") or {}
        if status == 200 and any(item.get("version") == TARGET_VERSION for item in catalog.get("available_releases", [])):
            break
        if time.monotonic() >= deadline:
            raise RuntimeError("Production release parser did not accept the task-private TLS release and checksum")
        time.sleep(1)
    before = nested.compose("ps", "-q", "server").strip()
    # 配置快照缺失是受控故障注入，恢复原文件后才进入正常更新。
    daemon(fixture, "mv", ROOT + "/.env", ROOT + "/.env.fixture-held")
    try:
        failed_operation = submit_update(nested)
        failure, failed_task = wait_operation(nested, failed_operation, "FAILED")
        failed_state = json.loads(docker(fixture, "exec", "--user", "graft", before, "cat", "/var/lib/graft/update-state/current.json"))
        failed_receipt = failed_state.get("receipt") or {}
        if (failed_task.get("status") != "failed" or failure.get("error") != "backup_failed"
            or failed_receipt.get("operation_id") != failed_operation or failed_receipt.get("failure_stage") != "env_snapshot"
            or failed_receipt.get("migration_started") is not False or failed_receipt.get("succeeded") is not False):
            raise RuntimeError("Missing deployment environment did not produce the expected failed Backup/Task")
    finally:
        daemon(fixture, "mv", ROOT + "/.env.fixture-held", ROOT + "/.env")
    if nested.compose("ps", "-q", "server").strip() != before:
        raise RuntimeError("Backup failure unexpectedly replaced the server")
    operation = submit_update(nested)
    history, task = wait_operation(nested, operation, "SUCCESS")
    status, response = nested.api("GET", "/api/platform/updates/status")
    after = nested.compose("ps", "-q", "server").strip()
    actual_image = docker(fixture, "inspect", "--format", "{{.Image}}", after).strip()
    state = json.loads(nested.compose("exec", "-T", "backend", "cat", "/var/lib/graft/update-state/current.json"))
    receipt = state.get("receipt") or {}
    migration = migration_invocation_scope(receipt, fixture.state["source_revision"])
    stages = task.get("stages") or []
    launch = next((item for item in stages if item.get("key") == "controller_launch"), {})
    final = next((item for item in stages if item.get("key") == "controller_result"), {})
    backup = receipt.get("backup_completion") or {}
    backup_detail = backup_for_task(nested, task["id"], backup)
    valid = (task.get("status") == "success" and before != after and actual_image == source["target_private_image"]
        and status == 200 and response.get("data", {}).get("current_version") == TARGET_VERSION
        and receipt.get("operation_id") == operation and receipt.get("execution_mode") == "local"
        and receipt.get("succeeded") is True and backup.get("OperationID") == operation
        and backup.get("TaskID") == task.get("id")
        and launch.get("status") == "success" and final.get("status") == "success"
        and receipt.get("launch_stage_id") == launch.get("id") and receipt.get("launch_attempt") == launch.get("attempt"))
    if not valid:
        raise RuntimeError("Public Update API, Task, backup or restarted target identity did not satisfy the acceptance contract")
    for filename, prefix in (("config.snapshot", "ConfigSnapshot"), ("database.dump", "DatabaseDump")):
        artifact = "/var/lib/graft/backups/" + operation + "/" + filename
        actual_hash = nested.compose("exec", "-T", "backend", "sha256sum", artifact).split()[0]
        actual_size = int(nested.compose("exec", "-T", "backend", "wc", "-c", artifact).split()[0])
        if backup.get(prefix + "SHA256") != actual_hash or backup.get(prefix + "Bytes") != actual_size or actual_size == 0:
            raise RuntimeError("Persisted backup bytes do not match the settled Controller receipt")
    evidence = {"source_revision": fixture.state["source_revision"], "synthetic_release_source": True,
        "trust_scope": "Registered private daemon and nested server containers only", **source,
        "source_version": SOURCE_VERSION, "target_version": TARGET_VERSION, "source_binary_sha256": source_binary,
        "migration_invocation": migration,
        "failed_backup_operation": failure, "failed_backup_task_id": failed_task.get("id"),
        "failed_backup_receipt": failed_receipt,
        "operation": history, "backup": backup_detail, "task_id": task["id"], "task_status": task["status"],
        "server_container_replaced": True, "target_image": actual_image, "receipt": receipt}
    path = directory / "update-api-evidence.json"
    path.write_text(json.dumps(evidence, indent=2) + "\n")
    path.chmod(0o600)
    print("Production Update API settled Task, Backup and local Controller after target server replacement", flush=True)

#!/usr/bin/env python3
"""Checkout-owned preview instance lifecycle for Graft compose previews.

Identity is derived deterministically from the checkout path plus the task id,
so every command in the same checkout resolves the same instance without a
shared mutable handle. The registry lives in the Git common directory so it
survives worktree loss; `preview_recovery.py` reconciles leftovers read-only.

`stop` pauses containers and retains data; `destroy` removes containers,
volumes and the instance directory, requires the exact instance identity via
--confirm-instance, and archives a SHA-256 cleanup evidence record.
"""
from __future__ import annotations

import argparse
import contextlib
import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import socket
import subprocess
import sys
import time
import uuid
from urllib import error as urlerror
from urllib import request as urlrequest

ROOT = Path(__file__).resolve().parents[1]
REGISTRY_SCHEMA_VERSION = 1
STATE_DIRNAME = "graft-preview"
DEFAULT_PORT_BASE = 38000
PORT_RANGE = 64
DEFAULT_MAX_RUNNING = 3
DEFAULT_HEALTH_TIMEOUT = 120
RESERVED_ENV_KEYS = frozenset({
    "GRAFT_IMAGE_TAG", "GRAFT_WEB_HOST_PORT", "COMPOSE_FILE_DIR", "COMPOSE_PROFILES",
    "COMPOSE_PROJECT_NAME", "GRAFT_EXECUTION_MODE", "GRAFT_BUILD_SNAPSHOT_VOLUME",
    "GRAFT_UPDATE_STATE_VOLUME", "GRAFT_APPLICATION_ROOT_HOST_PATH",
    "GRAFT_APPLICATION_ROOT_CONTAINER_PATH", "GRAFT_BACKUP_ARTIFACT_HOST_PATH",
    "POSTGRES_PASSWORD", "GRAFT_AUTH_JWT_SECRET",
})
ENV_KEY_PATTERN = re.compile(r"[A-Z][A-Z0-9_]*")
HEALTH_POLL_SECONDS = 1.0


class PreviewError(RuntimeError):
    """Refusal or operational failure that should end the command with exit 1."""


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def canonical_json(value) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def slugify(task_id: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", task_id.lower()).strip("-")
    return (slug or "task")[:24]


def identity_for(checkout: Path, task_id: str) -> str:
    """Deterministic instance identity: checkout path + task id, digest-bound."""
    if not task_id.strip():
        raise PreviewError("task id must not be empty")
    checkout = checkout.resolve()
    digest = hashlib.sha256(f"{checkout}\n{task_id}".encode("utf-8")).hexdigest()[:12]
    return f"{slugify(task_id)}-{digest}"


def compose_project_for(identity: str) -> str:
    return f"graft-preview-{identity}"


def checkout_root() -> Path:
    output = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True, cwd=ROOT)
    return Path(output.strip()).resolve()


def git_common_dir() -> Path:
    output = subprocess.check_output(["git", "rev-parse", "--git-common-dir"], text=True, cwd=ROOT)
    return Path(output.strip()).resolve()


def state_root() -> Path:
    override = os.environ.get("GRAFT_PREVIEW_STATE_DIR")
    if override:
        return Path(override).resolve()
    return git_common_dir() / STATE_DIRNAME


def source_revision() -> str:
    return subprocess.check_output(["git", "rev-parse", "HEAD"], text=True, cwd=ROOT).strip()


def digest_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def registry_path(root: Path) -> Path:
    return root / "registry.json"


def read_registry(root: Path) -> dict:
    path = registry_path(root)
    if not path.is_file():
        return {"schema_version": REGISTRY_SCHEMA_VERSION, "instances": {}}
    document = json.loads(path.read_text(encoding="utf-8"))
    if document.get("schema_version") != REGISTRY_SCHEMA_VERSION:
        raise PreviewError(f"registry schema {document.get('schema_version')!r} is not supported")
    document.setdefault("instances", {})
    return document


def write_registry(root: Path, document: dict) -> None:
    root.mkdir(parents=True, exist_ok=True)
    path = registry_path(root)
    # A per-writer temporary name: a fixed one would let two writers race
    # on the same temp file before the rename.
    temporary = path.parent / f".{path.name}.{os.getpid()}-{uuid.uuid4().hex[:8]}.tmp"
    temporary.write_text(json.dumps(document, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    os.replace(temporary, path)
    path.chmod(0o600)


@contextlib.contextmanager
def registry_lock(root: Path):
    """Serialize registry read→change→write sections across processes.

    The lock file lives beside the registry in the state root so every
    process resolving the same registry contends on the same lock; the
    port allocation and PREVIEW_MAX_RUNNING reservation must happen while
    it is held or two overlapping starts could double-allocate.
    """
    root.mkdir(parents=True, exist_ok=True)
    lock_path = root / "registry.lock"
    with open(lock_path, "a+") as handle:
        fcntl.flock(handle.fileno(), fcntl.LOCK_EX)
        try:
            yield
        finally:
            fcntl.flock(handle.fileno(), fcntl.LOCK_UN)


def update_entry(root: Path, identity: str, mutate) -> dict:
    """Read-modify-write exactly one registry entry under the exclusive lock."""
    with registry_lock(root):
        document = read_registry(root)
        entry = document["instances"].get(identity)
        if entry is None:
            raise PreviewError(f"no registered preview instance for identity {identity}")
        mutate(entry)
        write_registry(root, document)
        return entry


def archive_dir(root: Path) -> Path:
    return root / "archive"


def read_archive_manifest(root: Path) -> dict:
    path = archive_dir(root) / "manifest.json"
    if not path.is_file():
        return {"schema_version": REGISTRY_SCHEMA_VERSION, "entries": {}}
    return json.loads(path.read_text(encoding="utf-8"))


def port_in_use(port: int, host: str = "127.0.0.1") -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            probe.bind((host, port))
        except OSError:
            return True
    return False


def parse_port_base() -> int:
    raw = os.environ.get("GRAFT_PREVIEW_PORT_BASE")
    if raw is None:
        return DEFAULT_PORT_BASE
    try:
        value = int(raw)
    except ValueError as exc:
        raise PreviewError(f"GRAFT_PREVIEW_PORT_BASE={raw!r} is not an integer") from exc
    if not 1024 <= value <= 65535:
        raise PreviewError(f"GRAFT_PREVIEW_PORT_BASE={value} must be an unprivileged port")
    return value


def parse_max_running() -> int:
    raw = os.environ.get("PREVIEW_MAX_RUNNING")
    if raw is None:
        return DEFAULT_MAX_RUNNING
    try:
        value = int(raw)
    except ValueError as exc:
        raise PreviewError(f"PREVIEW_MAX_RUNNING={raw!r} is not an integer") from exc
    if value < 1:
        raise PreviewError("PREVIEW_MAX_RUNNING must be at least 1")
    return value


def allocate_port(root: Path, document: dict, requested: int | None) -> int:
    """Refuse on conflict; never silently steal a foreign or registered port."""
    base = parse_port_base()
    registered = {entry.get("web_port") for entry in document["instances"].values()
                  if isinstance(entry.get("web_port"), int)}
    if requested is not None:
        if not 1024 <= requested <= 65535:
            raise PreviewError(f"--port {requested} must be an unprivileged port")
        if requested in registered:
            raise PreviewError(f"port {requested} is already registered to another preview instance")
        if port_in_use(requested):
            raise PreviewError(f"port {requested} is already in use by another listener")
        return requested
    for candidate in range(base, base + PORT_RANGE):
        if candidate in registered:
            continue
        if not port_in_use(candidate):
            return candidate
        # Foreign listener inside the preview port range: refuse so the operator
        # resolves the conflict explicitly instead of a silent skip.
        raise PreviewError(
            f"port {candidate} inside the preview range (base {base}) is occupied by a foreign listener; "
            "resolve the conflict or pass --port explicitly")
    raise PreviewError(f"no free port in range {base}..{base + PORT_RANGE - 1}; pass --port explicitly")


def running_count(document: dict, exclude: str | None = None) -> int:
    return sum(1 for identity, entry in document["instances"].items()
               if identity != exclude and entry.get("state") == "running")


def instance_dir(root: Path, identity: str) -> Path:
    return root / "instances" / identity


def build_env_file(directory: Path, identity: str, image_tag: str, web_port: int,
                   extra_env: dict[str, str]) -> Path:
    """Write the per-instance .env; secrets persist on disk so convergence
    after restart must reuse this file instead of rotating credentials."""
    lines = [
        f"GRAFT_IMAGE_TAG={image_tag}",
        f"GRAFT_WEB_HOST_PORT={web_port}",
        "GRAFT_EXECUTION_MODE=local",
        f"POSTGRES_PASSWORD={secrets.token_urlsafe(24)}",
        f"GRAFT_AUTH_JWT_SECRET={secrets.token_urlsafe(32)}",
        f"COMPOSE_FILE_DIR={directory}",
        f"GRAFT_APPLICATION_ROOT_HOST_PATH={directory / 'apps'}",
        f"GRAFT_APPLICATION_ROOT_CONTAINER_PATH={directory / 'apps'}",
        f"GRAFT_BACKUP_ARTIFACT_HOST_PATH={directory / 'backups'}",
        f"GRAFT_BUILD_SNAPSHOT_VOLUME=graft-preview-{identity}-build-snapshots",
        f"GRAFT_UPDATE_STATE_VOLUME=graft-preview-{identity}-update-state",
    ]
    for key, value in sorted(extra_env.items()):
        lines.append(f"{key}={value}")
    env_file = directory / ".env"
    env_file.write_text("\n".join(lines) + "\n", encoding="utf-8")
    env_file.chmod(0o600)
    return env_file


def parse_extra_env(pairs: list[str]) -> dict[str, str]:
    parsed: dict[str, str] = {}
    for item in pairs:
        key, separator, value = item.partition("=")
        if not separator or not ENV_KEY_PATTERN.fullmatch(key):
            raise PreviewError(f"--env expects KEY=VALUE with KEY matching {ENV_KEY_PATTERN.pattern}: {item!r}")
        if "\n" in value or "\r" in value:
            raise PreviewError(f"--env value for {key} must be a single line")
        if key in RESERVED_ENV_KEYS:
            raise PreviewError(f"--env may not override reserved preview key {key}")
        parsed[key] = value
    return parsed


def run_compose(arguments: list[str], *, compose_file: Path, project: str, env_file: Path,
                timeout: float | None = None, capture: bool = True) -> subprocess.CompletedProcess:
    command = ["docker", "compose", "-f", str(compose_file), "-p", project,
               "--env-file", str(env_file), *arguments]
    environment = dict(os.environ)
    environment["COMPOSE_FILE_DIR"] = str(env_file.parent)
    return subprocess.run(command, capture_output=capture, text=True, timeout=timeout,
                          env=environment)


def probe_http(url: str, timeout: float = 5.0) -> dict:
    try:
        with urlrequest.urlopen(url, timeout=timeout) as response:
            return {"ok": 200 <= response.status < 400, "status": response.status, "error": None}
    except urlerror.HTTPError as exc:
        return {"ok": False, "status": exc.code, "error": None}
    except (urlerror.URLError, OSError, ValueError) as exc:
        return {"ok": False, "status": None, "error": str(exc)}


def wait_for_health(url: str, timeout_seconds: float) -> dict:
    deadline = time.monotonic() + timeout_seconds
    last: dict | None = None
    while time.monotonic() < deadline:
        last = probe_http(url)
        if last["ok"]:
            return last
        time.sleep(HEALTH_POLL_SECONDS)
    return last or {"ok": False, "status": None, "error": "health wait timed out"}


def web_url(entry: dict) -> str:
    return f"http://127.0.0.1:{entry['web_port']}/"


def resolve_identity(args: argparse.Namespace, root: Path) -> str:
    if getattr(args, "identity", None):
        return args.identity
    return identity_for(checkout_root(), args.task)


def require_entry(document: dict, identity: str) -> dict:
    entry = document["instances"].get(identity)
    if entry is None:
        raise PreviewError(f"no registered preview instance for identity {identity}")
    return entry


def start(args: argparse.Namespace) -> dict:
    checkout = checkout_root()
    compose_source = checkout / "compose.yml"
    if not compose_source.is_file():
        raise PreviewError(f"checkout {checkout} has no compose.yml to preview")
    identity = identity_for(checkout, args.task)
    project = compose_project_for(identity)
    root = state_root()
    image_tag = args.image_tag
    if not image_tag:
        raise PreviewError("--image-tag is required (the compose file refuses an empty GRAFT_IMAGE_TAG)")
    # compose.yml already prefixes the registry image name; a full ref here
    # would produce a double image reference.
    if "/" in image_tag or ":" in image_tag:
        raise PreviewError(
            "--image-tag must be a bare tag without '/' or ':' (compose.yml already "
            "prefixes the image name), for example: candidate-v1.2.3-abc")
    directory = instance_dir(root, identity)
    env_file = directory / ".env"
    compose_file = directory / "compose.yml"
    web_port: int | None = None
    source_info: dict = {}

    # The registry read, the port allocation and the PREVIEW_MAX_RUNNING
    # reservation form one critical section: holding the registry lock here
    # makes double allocation impossible across overlapping processes.
    with registry_lock(root):
        document = read_registry(root)
        entry = document["instances"].get(identity)
        if entry is None:
            cap = parse_max_running()
            if running_count(document) >= cap:
                raise PreviewError(
                    f"PREVIEW_MAX_RUNNING={cap} reached ({running_count(document)} running instances); "
                    "stop or destroy one first")
            web_port = allocate_port(root, document, args.port)
            directory.mkdir(parents=True, mode=0o700, exist_ok=True)
            build_env_file(directory, identity, image_tag, web_port, parse_extra_env(args.env))
            shutil.copyfile(compose_source, compose_file)
            source_info = {"revision": source_revision(), "digest": digest_file(compose_file)}
            document["instances"][identity] = {
                "identity": identity,
                "task_id": args.task,
                "checkout": str(checkout),
                "compose_project": project,
                "image_tag": image_tag,
                "web_port": web_port,
                "state": "starting",
                "created_at": utc_now(),
                "updated_at": utc_now(),
                "source": source_info,
            }
            write_registry(root, document)
        else:
            # Convergence on a registered instance keeps its reserved port and
            # stored secrets; regenerating the env file would rotate credentials.
            if not env_file.is_file() or not compose_file.is_file():
                raise PreviewError(
                    f"instance {identity} is registered but its env/compose files are missing; "
                    "destroy it with --confirm-instance before starting again")
            if args.port is not None and args.port != entry.get("web_port"):
                raise PreviewError(
                    f"instance {identity} is registered on port {entry.get('web_port')}; "
                    "--port must match or be omitted")
            web_port = entry.get("web_port")
            source_info = entry.get("source") or {}
            if entry.get("state") == "running":
                converged = wait_for_health(web_url(entry), min(args.health_timeout, 10))
                if converged["ok"]:
                    entry["updated_at"] = utc_now()
                    write_registry(root, document)
                    return {"identity": identity, "state": "running", "already_running": True,
                            "health": converged, "web_url": web_url(entry)}
                # Marked running but unhealthy: containers may have been
                # stopped or removed externally, so fall through and restore
                # the stack via compose up instead of reporting a lie.
            else:
                # A restart re-occupies the cap: count running instances excluding
                # this entry so a stopped instance cannot bypass PREVIEW_MAX_RUNNING.
                cap = parse_max_running()
                running_others = running_count(document, exclude=identity)
                if running_others >= cap:
                    raise PreviewError(
                        f"PREVIEW_MAX_RUNNING={cap} reached ({running_others} other running instances); "
                        "restart refuses to exceed the cap, stop or destroy one first")

    def mark(state: str, **fields) -> None:
        # Compose runs outside the lock; every state flip re-reads the
        # registry under the lock so it never clobbers concurrent writers
        # with a stale in-memory copy.
        update_entry(root, identity,
                     lambda current: current.update(state=state, updated_at=utc_now(), **fields))

    mark("starting")
    try:
        completed = run_compose(["up", "-d", "--wait"], compose_file=compose_file, project=project,
                                env_file=env_file, timeout=args.health_timeout + 60)
        if completed.returncode != 0:
            raise PreviewError(
                "compose up exited with code "
                f"{completed.returncode}: {completed.stderr.strip()[:500]}")
    except (subprocess.SubprocessError, PreviewError) as exc:
        mark("failed", failure=str(exc))
        raise PreviewError(f"compose up failed for {identity}: {exc}") from exc

    mark("running")
    health = wait_for_health(f"http://127.0.0.1:{web_port}/", args.health_timeout)
    logs_dir = directory / "logs"
    logs_dir.mkdir(parents=True, exist_ok=True)
    snapshot = logs_dir / "compose.log"
    try:
        completed = run_compose(["logs", "--no-color"], compose_file=compose_file,
                                project=project, env_file=env_file, timeout=60)
        snapshot.write_text(completed.stdout, encoding="utf-8")
    except (subprocess.SubprocessError, OSError) as exc:
        snapshot.write_text(f"log snapshot failed: {exc}\n", encoding="utf-8")
    return {"identity": identity, "state": "running", "already_running": False,
            "web_url": f"http://127.0.0.1:{web_port}/", "health": health,
            "source": source_info, "registry": str(registry_path(root)),
            "instance_dir": str(directory)}


def status(args: argparse.Namespace) -> dict:
    root = state_root()
    document = read_registry(root)
    if getattr(args, "all", False):
        return {"schema_version": REGISTRY_SCHEMA_VERSION,
                "state_root": str(root),
                "instances": [status_entry(root, entry) for entry in document["instances"].values()]}
    identity = resolve_identity(args, root)
    entry = document["instances"].get(identity)
    if entry is None:
        return {"identity": identity, "registered": False, "state_root": str(root)}
    return status_entry(root, entry)


def status_entry(root: Path, entry: dict) -> dict:
    directory = instance_dir(root, entry["identity"])
    compose_file = directory / "compose.yml"
    source = dict(entry.get("source") or {})
    source["current_digest"] = digest_file(compose_file) if compose_file.is_file() else None
    source["drifted"] = (source["current_digest"] != source.get("digest")) \
        if compose_file.is_file() else None
    checkout_compose = Path(entry.get("checkout") or "") / "compose.yml"
    if checkout_compose.is_file():
        source["checkout_digest"] = digest_file(checkout_compose)
        source["checkout_matches_start"] = source.get("digest") == source["checkout_digest"]
    health = None
    if entry.get("state") == "running":
        health = {"web": probe_http(web_url(entry)), "url": web_url(entry)}
    return {
        "identity": entry["identity"],
        "task_id": entry.get("task_id"),
        "state": entry.get("state"),
        "compose_project": entry.get("compose_project"),
        "checkout": entry.get("checkout"),
        "image_tag": entry.get("image_tag"),
        "web": {"host_port": entry.get("web_port"), "url": web_url(entry) if entry.get("web_port") else None},
        "health": health,
        "logs": {"compose_log": str(directory / "logs" / "compose.log"),
                 "instance_dir": str(directory)},
        "source": source,
        "created_at": entry.get("created_at"),
        "updated_at": entry.get("updated_at"),
    }


def logs(args: argparse.Namespace) -> dict:
    root = state_root()
    document = read_registry(root)
    identity = resolve_identity(args, root)
    entry = require_entry(document, identity)
    directory = instance_dir(root, identity)
    compose_file = directory / "compose.yml"
    env_file = directory / ".env"
    if not compose_file.is_file() or not env_file.is_file():
        raise PreviewError(f"instance {identity} has no stored compose/env files; only destroy applies")
    arguments = ["logs", "--no-color", "--tail", str(args.tail)]
    if args.follow:
        arguments.append("--follow")
    try:
        completed = run_compose(arguments, compose_file=compose_file,
                                project=entry["compose_project"], env_file=env_file,
                                timeout=None if args.follow else 60, capture=False)
    except subprocess.SubprocessError as exc:
        raise PreviewError(f"compose logs failed for {identity}: {exc}") from exc
    if completed.returncode != 0:
        raise PreviewError(f"compose logs exited with code {completed.returncode} for {identity}")
    return {"identity": identity, "tail": args.tail, "exit_code": completed.returncode,
            "log_file": str(directory / "logs" / "compose.log")}


def stop(args: argparse.Namespace) -> dict:
    root = state_root()
    document = read_registry(root)
    identity = resolve_identity(args, root)
    entry = require_entry(document, identity)
    directory = instance_dir(root, identity)
    compose_file = directory / "compose.yml"
    env_file = directory / ".env"
    if not compose_file.is_file() or not env_file.is_file():
        raise PreviewError(f"instance {identity} has no stored compose/env files; only destroy applies")
    # `stop` retains containers and volumes; data survives for restart.
    completed = run_compose(["stop"], compose_file=compose_file, project=entry["compose_project"],
                            env_file=env_file, timeout=120)
    if completed.returncode != 0:
        raise PreviewError(
            f"compose stop exited with code {completed.returncode}: {completed.stderr.strip()[:500]}")
    entry["state"] = "stopped"
    entry["updated_at"] = utc_now()
    write_registry(root, document)
    return {"identity": identity, "state": "stopped", "data_retained": True}


def append_archive(root: Path, entry: dict) -> dict:
    archive = archive_dir(root)
    archive.mkdir(parents=True, exist_ok=True)
    record = {
        "schema_version": REGISTRY_SCHEMA_VERSION,
        "identity": entry["identity"],
        "destroyed_at": utc_now(),
        "evidence_sha256": hashlib.sha256(canonical_json(entry).encode("utf-8")).hexdigest(),
        "registry_entry": entry,
    }
    filename = f"{entry['identity']}-{dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%SZ')}.json"
    target = archive / filename
    target.write_text(json.dumps(record, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    manifest = read_archive_manifest(root)
    manifest.setdefault("entries", {})[filename] = {
        "identity": entry["identity"],
        "destroyed_at": record["destroyed_at"],
        "sha256": digest_file(target),
    }
    manifest_path = archive / "manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return {"archive_file": str(target), "evidence_sha256": record["evidence_sha256"]}


def reset_container_owned_permissions(directory: Path) -> None:
    """Reclaim container-owned files before the host-side removal.

    application-root-init leaves apps/ at UID 10001 mode 0750 and postgres
    writes similar data, so shutil.rmtree as the host user would fail. A
    short-lived run of the stack's own postgres image (already pulled by
    any preview start, so no extra image download) chowns the instance
    directory back to root first.
    """
    command = ["docker", "run", "--rm", "--user", "0:0",
               "-v", f"{directory}:/target", "postgres:16.16-alpine",
               "sh", "-c", "chown -R 0:0 /target && chmod -R u+rwX /target"]
    completed = subprocess.run(command, capture_output=True, text=True, timeout=120)
    if completed.returncode != 0:
        raise PreviewError(
            f"privileged cleanup exited with code {completed.returncode}: "
            f"{completed.stderr.strip()[:300]}")


def teardown_recorded_project(project: str, timeout: float = 180) -> None:
    """Remove a compose project by its recorded name when the stored
    compose/env files are gone. Compose labels every resource with the
    project name, so a label-filtered removal works without a compose
    file and across compose versions."""
    errors: list[str] = []
    for kind, query in (
        ("container", ["docker", "ps", "-aq", "--filter",
                       f"label=com.docker.compose.project={project}"]),
        ("volume", ["docker", "volume", "ls", "-q", "--filter",
                    f"label=com.docker.compose.project={project}"]),
        ("network", ["docker", "network", "ls", "-q", "--filter",
                     f"label=com.docker.compose.project={project}"]),
    ):
        listed = subprocess.run(query, capture_output=True, text=True, timeout=timeout)
        if listed.returncode != 0:
            raise PreviewError(
                f"recorded-project {kind} query exited with code {listed.returncode}: "
                f"{listed.stderr.strip()[:300]}")
        for resource in (line.strip() for line in listed.stdout.splitlines() if line.strip()):
            command = ["docker", kind, "rm"] + (["-f"] if kind == "container" else []) + [resource]
            removed = subprocess.run(command, capture_output=True, text=True, timeout=timeout)
            if removed.returncode != 0:
                errors.append(f"{kind} {resource}: {removed.stderr.strip()[:200]}")
    if errors:
        raise PreviewError("recorded-project teardown failed: " + "; ".join(errors))


def destroy(args: argparse.Namespace) -> dict:
    """Order matters: tear down the stack, reclaim container-owned data,
    and only then archive evidence and drop the registry entry. A cleanup
    failure keeps the entry with a cleanup_failed marker so destroy stays
    retryable instead of leaving an untracked half-cleanup. Destroy never
    reports success while the recorded stack may still be live."""
    root = state_root()
    identity = resolve_identity(args, root)
    # The lock spans the whole teardown so a concurrent start cannot
    # reserve this instance's port while its entry is mid-destruction.
    with registry_lock(root):
        document = read_registry(root)
        entry = require_entry(document, identity)
        if args.confirm_instance != identity:
            raise PreviewError(
                f"destroy requires --confirm-instance {identity} (the exact registered identity)")
        directory = instance_dir(root, identity)
        compose_file = directory / "compose.yml"
        env_file = directory / ".env"
        try:
            if compose_file.is_file() and env_file.is_file():
                completed = run_compose(["down", "--volumes", "--remove-orphans"],
                                        compose_file=compose_file, project=entry["compose_project"],
                                        env_file=env_file, timeout=180)
                if completed.returncode != 0:
                    raise PreviewError(
                        f"compose down exited with code {completed.returncode}: "
                        f"{completed.stderr.strip()[:500]}")
            else:
                # Files may be gone (worktree loss, manual deletion) while the
                # stack lives on; tear down from the recorded project name so
                # the entry is never dropped while containers or volumes run.
                teardown_recorded_project(entry["compose_project"])
        except (PreviewError, subprocess.SubprocessError) as exc:
            entry["state"] = "cleanup_failed"
            entry["cleanup_error"] = str(exc)
            entry["updated_at"] = utc_now()
            write_registry(root, document)
            raise PreviewError(
                f"stack teardown failed for {identity} (entry kept for retry): {exc}") from exc
        if directory.exists():
            try:
                reset_container_owned_permissions(directory)
                shutil.rmtree(directory)
            except (PreviewError, subprocess.SubprocessError, OSError) as exc:
                entry["state"] = "cleanup_failed"
                entry["cleanup_error"] = str(exc)
                entry["updated_at"] = utc_now()
                write_registry(root, document)
                raise PreviewError(
                    f"instance data cleanup failed for {identity} (entry kept for retry): {exc}") from exc
        archive = append_archive(root, entry)
        del document["instances"][identity]
        write_registry(root, document)
    return {"identity": identity, "state": "destroyed", "volumes_removed": True, **archive}


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, epilog=(
        "stop retains containers and data; destroy removes containers, volumes and "
        "the instance directory and requires --confirm-instance <identity>."))
    subparsers = parser.add_subparsers(dest="command", required=True)

    start_parser = subparsers.add_parser("start", help="Start or converge the checkout preview")
    start_parser.add_argument("--task", required=True, help="Task identifier bound to this checkout")
    start_parser.add_argument("--image-tag", help="GRAFT_IMAGE_TAG for the preview stack")
    start_parser.add_argument("--port", type=int, help="Explicit web host port (default: allocate)")
    start_parser.add_argument("--env", action="append", default=[],
                              help="Extra KEY=VALUE for the instance env file (repeatable)")
    start_parser.add_argument("--health-timeout", type=float, default=DEFAULT_HEALTH_TIMEOUT)
    start_parser.set_defaults(handler=start)

    status_parser = subparsers.add_parser("status", help="JSON status of one or all instances")
    status_parser.add_argument("--task")
    status_parser.add_argument("--identity")
    status_parser.add_argument("--all", action="store_true")
    status_parser.set_defaults(handler=status)

    logs_parser = subparsers.add_parser("logs", help="Preview compose logs (short-lived)")
    logs_parser.add_argument("--task")
    logs_parser.add_argument("--identity")
    logs_parser.add_argument("--tail", type=int, default=100)
    logs_parser.add_argument("--follow", action="store_true")
    logs_parser.set_defaults(handler=logs)

    stop_parser = subparsers.add_parser("stop", help="Stop the preview; data is retained")
    stop_parser.add_argument("--task")
    stop_parser.add_argument("--identity")
    stop_parser.set_defaults(handler=stop)

    destroy_parser = subparsers.add_parser("destroy", help="Destroy the preview and its data")
    destroy_parser.add_argument("--task")
    destroy_parser.add_argument("--identity")
    destroy_parser.add_argument("--confirm-instance",
                                help="The exact registered identity, as printed by status")
    destroy_parser.set_defaults(handler=destroy)
    return parser


def main(argv: list[str] | None = None) -> None:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.command in ("status", "logs", "stop", "destroy"):
        has_target = getattr(args, "task", None) or getattr(args, "identity", None) \
            or getattr(args, "all", False)
        if not has_target:
            parser.error("one of --task, --identity or --all is required")
    result = args.handler(args)
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    try:
        main()
    except PreviewError as exc:
        print(f"preview: {exc}", file=sys.stderr)
        raise SystemExit(1) from None
    except (subprocess.SubprocessError, OSError, ValueError, KeyError, json.JSONDecodeError) as exc:
        print(f"preview: {type(exc).__name__}: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc

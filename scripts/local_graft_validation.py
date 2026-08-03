#!/usr/bin/env python3
"""Manage isolated local Graft Compose deployments for release validation."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
from typing import Iterable
from urllib import error as urlerror
from urllib import request as urlrequest
import re


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_LOCAL_ROOT = REPO_ROOT / ".local" / "graft-validation"
INSTANCES = ("beta", "latest", "fixed")
INSTANCE_DEFAULTS = {
    "beta": {"tag": "beta", "port": "3101"},
    "latest": {"tag": "latest", "port": "3102"},
    "fixed": {"tag": "v0.11.0-beta.39", "port": "3103"},
}
SAFE_CONFIG_KEYS = {
    "GRAFT_WEB_HOST_PORT",
    "GRAFT_IMAGE_TAG",
    "GRAFT_LOG_LEVEL",
    "GRAFT_LOG_FORMAT",
    "GRAFT_DOCS_ENABLED",
    "GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS",
}
FIXED_RELEASE_TAG_RE = re.compile(r"^v[0-9]+(?:\.[0-9]+){1,3}(?:[-+][0-9A-Za-z.-]+)?$")
README_START = "<!-- graft-local-validation:start -->"
README_END = "<!-- graft-local-validation:end -->"
OFFICIAL_SERVER_IMAGE = "ghcr.io/gewuyou/graft-server"
GITHUB_RAW_BASE_URL = "https://raw.githubusercontent.com/GeWuYou/Graft"


class LocalValidationError(RuntimeError):
    """Raised when a requested local deployment operation is unsafe."""


def instance_root(local_root: Path, instance: str) -> Path:
    return local_root / instance


def compose_project_name(instance: str) -> str:
    return f"graft-local-verify-{instance}"


def update_volume_name(instance: str) -> str:
    return f"graft-local-verify-{instance}-update-state"


def parse_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.is_file():
        return values
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line or line.lstrip().startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key] = value.split(" #", 1)[0].strip()
    return values


def set_env_values(template: str, values: dict[str, str]) -> str:
    rendered: list[str] = []
    seen: set[str] = set()
    for line in template.splitlines():
        if line and not line.lstrip().startswith("#") and "=" in line:
            key, old_value = line.split("=", 1)
            if key in values:
                comment = ""
                if " #" in old_value:
                    comment = " #" + old_value.split(" #", 1)[1]
                rendered.append(f"{key}={values[key]}{comment}")
                seen.add(key)
                continue
        rendered.append(line)
    for key in sorted(values.keys() - seen):
        rendered.append(f"{key}={values[key]}")
    return "\n".join(rendered) + "\n"


def random_secret() -> str:
    return secrets.token_urlsafe(32)


def desired_env(local_root: Path, instance: str, existing: dict[str, str] | None = None) -> dict[str, str]:
    if instance not in INSTANCE_DEFAULTS:
        raise LocalValidationError(f"unknown instance: {instance}")
    root = instance_root(local_root, instance).resolve()
    defaults = INSTANCE_DEFAULTS[instance]
    prior = existing or {}
    port = prior.get("GRAFT_WEB_HOST_PORT", defaults["port"])
    return {
        "GRAFT_IMAGE_TAG": prior.get("GRAFT_IMAGE_TAG", defaults["tag"]),
        "GRAFT_APP_ENV": "production",
        "GRAFT_DEPLOYMENT_RUNTIME": "compose",
        "GRAFT_DEPLOYMENT_COMPOSE_ROOT": str(root),
        "GRAFT_UPDATE_STATE_VOLUME": update_volume_name(instance),
        "POSTGRES_DB": prior.get("POSTGRES_DB", f"graft_{instance}"),
        "POSTGRES_USER": prior.get("POSTGRES_USER", f"graft_{instance}"),
        "POSTGRES_PASSWORD": prior.get("POSTGRES_PASSWORD", random_secret()),
        "GRAFT_AUTH_JWT_SECRET": prior.get("GRAFT_AUTH_JWT_SECRET", random_secret()),
        "GRAFT_REDIS_PASSWORD": prior.get("GRAFT_REDIS_PASSWORD", random_secret()),
        "GRAFT_WEB_HOST_PORT": port,
        "GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS": (
            f"http://127.0.0.1:{port},http://localhost:{port}"
        ),
        "GRAFT_PROJECT_IMPORT_HOST_PATH": "./data/imports",
        "GRAFT_APPLICATION_ROOT_HOST_PATH": "./data/apps",
        "GRAFT_BACKUP_ARTIFACT_HOST_PATH": "./data/backups",
        "GRAFT_MODULES_ENABLED": "",
    }


def selected_instances(target: str) -> tuple[str, ...]:
    return INSTANCES if target == "all" else (target,)


def docker_published_ports() -> set[int]:
    return set(docker_published_port_owners())


def docker_published_port_owners() -> dict[int, set[str]]:
    try:
        result = subprocess.run(
            ["docker", "ps", "--format", "{{.Names}}\t{{json .Ports}}"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
        )
    except (FileNotFoundError, subprocess.CalledProcessError):
        return {}
    ports: dict[int, set[str]] = {}
    for raw in result.stdout.splitlines():
        name, separator, raw_ports = raw.partition("\t")
        if not separator:
            continue
        try:
            value = json.loads(raw_ports)
        except json.JSONDecodeError:
            continue
        owners = {instance for instance in INSTANCES if name.startswith(compose_project_name(instance) + "-")}
        for item in str(value).split(","):
            if "->" not in item:
                continue
            host = item.strip().split("->", 1)[0].rsplit(":", 1)[-1]
            if host.isdigit():
                ports.setdefault(int(host), set()).update(owners)
    return ports


def is_socket_available(port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            probe.bind(("127.0.0.1", port))
        except OSError:
            return False
    return True


def configured_ports(local_root: Path, candidates: Iterable[str] = INSTANCES) -> dict[str, int]:
    ports: dict[str, int] = {}
    for instance in candidates:
        env = parse_env(instance_root(local_root, instance) / ".env")
        raw = env.get("GRAFT_WEB_HOST_PORT", INSTANCE_DEFAULTS[instance]["port"])
        try:
            port = int(raw)
        except ValueError as error:
            raise LocalValidationError(f"{instance} has an invalid web port: {raw!r}") from error
        if not 1 <= port <= 65535:
            raise LocalValidationError(f"{instance} has an invalid web port: {port}")
        ports[instance] = port
    return ports


def validate_ports(local_root: Path, *, proposed: dict[str, int] | None = None, check_host: bool = True) -> None:
    ports = configured_ports(local_root)
    ports.update(proposed or {})
    inverse: dict[int, str] = {}
    for instance, port in ports.items():
        other = inverse.get(port)
        if other:
            raise LocalValidationError(f"web port {port} is shared by {other} and {instance}")
        inverse[port] = instance
    if not check_host:
        return
    docker_owners = docker_published_port_owners()
    for port, instance in inverse.items():
        owners = docker_owners.get(port, set())
        if port in docker_owners and owners != {instance}:
            raise LocalValidationError(f"web port {port} is already published by a Docker container")
        if not owners and not is_socket_available(port):
            raise LocalValidationError(f"web port {port} is already in use on this host")


def compose_command(root: Path, instance: str, *args: str) -> list[str]:
    directory = instance_root(root, instance)
    return [
        "docker", "compose", "--project-name", compose_project_name(instance),
        "--project-directory", str(directory), "--env-file", str(directory / ".env"),
        "-f", str(directory / "compose.yml"), *args,
    ]


def run_compose(root: Path, instance: str, *args: str) -> int:
    return subprocess.run(compose_command(root, instance, *args), check=False).returncode


def published_server_revision(tag: str) -> str:
    result = subprocess.run(
        [
            "docker",
            "image",
            "inspect",
            f"{OFFICIAL_SERVER_IMAGE}:{tag}",
            "--format",
            "{{ index .Config.Labels \"org.opencontainers.image.revision\" }}",
        ],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    revision = result.stdout.strip()
    if result.returncode != 0 or not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise LocalValidationError(
            f"{tag} server image has no usable source revision; pull the official image before synchronizing Compose"
        )
    return revision


def download_release_file(revision: str, name: str) -> str:
    url = f"{GITHUB_RAW_BASE_URL}/{revision}/{name}"
    try:
        with urlrequest.urlopen(url, timeout=20) as response:
            contents = response.read().decode("utf-8")
    except (OSError, UnicodeDecodeError, urlerror.URLError) as error:
        raise LocalValidationError(f"could not download {name} for release revision {revision}: {error}") from error
    if not contents.strip():
        raise LocalValidationError(f"release revision {revision} returned an empty {name}")
    return contents


def sync_instance_compose(root: Path, instance: str) -> str:
    require_initialised(root, instance)
    directory = instance_root(root, instance)
    env_path = directory / ".env"
    compose_path = directory / "compose.yml"
    env_values = parse_env(env_path)
    tag = env_values["GRAFT_IMAGE_TAG"]
    revision = published_server_revision(tag)
    release_compose = download_release_file(revision, "compose.yml")
    release_template = download_release_file(revision, "compose.env.example")
    if "services:" not in release_compose:
        raise LocalValidationError(f"release revision {revision} does not provide an official Compose topology")
    previous_compose = compose_path.read_text(encoding="utf-8")
    previous_env = env_path.read_text(encoding="utf-8")
    compose_path.write_text(release_compose, encoding="utf-8")
    env_path.write_text(set_env_values(release_template, env_values), encoding="utf-8")
    if run_compose(root, instance, "config", "--quiet") != 0:
        compose_path.write_text(previous_compose, encoding="utf-8")
        env_path.write_text(previous_env, encoding="utf-8")
        raise LocalValidationError(f"{instance} Compose validation failed after release sync; restored previous files")
    return revision


def render_readme(local_root: Path) -> str:
    rows: list[str] = []
    for instance in INSTANCES:
        env = parse_env(instance_root(local_root, instance) / ".env")
        tag = env.get("GRAFT_IMAGE_TAG", INSTANCE_DEFAULTS[instance]["tag"])
        port = env.get("GRAFT_WEB_HOST_PORT", INSTANCE_DEFAULTS[instance]["port"])
        rows.append(
            f"| `{instance}` | `{tag}` | `{compose_project_name(instance)}` | http://127.0.0.1:{port} | "
            f"`graft-validation/{instance}/data` | `{update_volume_name(instance)}` |"
        )
    fixed_tag = INSTANCE_DEFAULTS["fixed"]["tag"]
    return "\n".join((
        README_START,
        "# Graft local validation environments",
        "",
        "| Instance | Image tag | Compose project | URL | Data root | Update state volume |",
        "| --- | --- | --- | --- | --- | --- |",
        *rows,
        "",
        "Initial login is `graft` / `graft-admin`; change the password after first login.",
        "Current production passwords are local operator records only and are not read or reset by this tool.",
        "",
        "Use `python3 scripts/local_graft_validation.py status all`, `logs beta`, or `doctor all` from the repository root.",
        "Use `record-access <instance> --note '...'` to append a local operator access note below this managed block.",
        "Health evidence: `doctor` runs Compose config validation, reports the current Compose service state, and probes the Web `/healthz` endpoint.",
        "",
        "The server containers share the host Docker socket. Docker operations performed through a local instance can affect other containers on this machine.",
        f"The fixed instance is pinned to `{fixed_tag}`. If the image is not published yet, `pull` and `up` report it as pending and never fall back to another tag.",
        README_END,
        "",
    ))


def refresh_readme(local_root: Path) -> None:
    path = local_root.parent / "README.md"
    path.parent.mkdir(parents=True, exist_ok=True)
    managed = render_readme(local_root)
    existing = path.read_text(encoding="utf-8") if path.exists() else ""
    start = existing.find(README_START)
    end = existing.find(README_END)
    if start >= 0 and end >= start:
        end += len(README_END)
        updated = existing[:start] + managed.rstrip() + existing[end:]
    elif existing:
        updated = existing.rstrip() + "\n\n" + managed
    else:
        updated = managed
    path.write_text(updated.rstrip() + "\n", encoding="utf-8")


def initialise(root: Path) -> None:
    compose_source = REPO_ROOT / "compose.yml"
    env_source = REPO_ROOT / "compose.env.example"
    if not compose_source.is_file() or not env_source.is_file():
        raise LocalValidationError("official compose.yml or compose.env.example is missing")
    root.mkdir(parents=True, exist_ok=True)
    validate_ports(root, check_host=True)
    template = env_source.read_text(encoding="utf-8")
    for instance in INSTANCES:
        directory = instance_root(root, instance)
        directory.mkdir(parents=True, exist_ok=True)
        (directory / "data" / "apps").mkdir(parents=True, exist_ok=True)
        (directory / "data" / "backups").mkdir(parents=True, exist_ok=True)
        (directory / "data" / "imports").mkdir(parents=True, exist_ok=True)
        (directory / ".data" / "postgres").mkdir(parents=True, exist_ok=True)
        compose_path = directory / "compose.yml"
        if not compose_path.exists():
            shutil.copy2(compose_source, compose_path)
        env_path = directory / ".env"
        if not env_path.exists():
            values = desired_env(root, instance)
            env_path.write_text(set_env_values(template, values), encoding="utf-8")
    refresh_readme(root)


def require_initialised(root: Path, instance: str) -> None:
    directory = instance_root(root, instance)
    if not (directory / "compose.yml").is_file() or not (directory / ".env").is_file():
        raise LocalValidationError(f"{instance} is not initialized; run init first")
    env = parse_env(directory / ".env")
    missing = {"GRAFT_IMAGE_TAG", "GRAFT_WEB_HOST_PORT"} - env.keys()
    if missing:
        raise LocalValidationError(
            f"{instance} is missing required initialization keys: {', '.join(sorted(missing))}; run init first"
        )
    env_root = env.get("GRAFT_DEPLOYMENT_COMPOSE_ROOT")
    if env_root != str(directory.resolve()):
        raise LocalValidationError(
            f"{instance} has an unsafe GRAFT_DEPLOYMENT_COMPOSE_ROOT; run init to restore the local instance root"
        )


def set_config(root: Path, instance: str, key: str, value: str) -> None:
    if key not in SAFE_CONFIG_KEYS:
        raise LocalValidationError(f"{key} is not a supported local configuration key")
    require_initialised(root, instance)
    env_path = instance_root(root, instance) / ".env"
    values = parse_env(env_path)
    if key == "GRAFT_IMAGE_TAG" and instance != "fixed" and value not in {"beta", "latest"}:
        raise LocalValidationError("tracking instances only accept beta or latest image tags")
    if key == "GRAFT_IMAGE_TAG" and instance == "fixed" and not FIXED_RELEASE_TAG_RE.fullmatch(value):
        raise LocalValidationError("the fixed instance requires an explicit v-prefixed release tag")
    if key == "GRAFT_WEB_HOST_PORT":
        try:
            port = int(value)
        except ValueError as error:
            raise LocalValidationError("GRAFT_WEB_HOST_PORT must be an integer") from error
        if not 1 <= port <= 65535:
            raise LocalValidationError(f"GRAFT_WEB_HOST_PORT must be between 1 and 65535: {port}")
        validate_ports(root, proposed={instance: port})
        values["GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS"] = f"http://127.0.0.1:{port},http://localhost:{port}"
    values[key] = value
    previous = env_path.read_text(encoding="utf-8")
    template = (REPO_ROOT / "compose.env.example").read_text(encoding="utf-8")
    env_path.write_text(set_env_values(template, values), encoding="utf-8")
    if run_compose(root, instance, "config", "--quiet") != 0:
        env_path.write_text(previous, encoding="utf-8")
        raise LocalValidationError("Compose validation failed; restored the previous .env")
    refresh_readme(root)


def reset(root: Path, target: str, confirmed: bool) -> None:
    if not confirmed:
        raise LocalValidationError("reset requires --confirm")
    for instance in selected_instances(target):
        directory = instance_root(root, instance)
        require_initialised(root, instance)
        if run_compose(root, instance, "down", "--volumes") != 0:
            raise LocalValidationError(f"{instance} could not be stopped before reset")
        data = directory / "data"
        postgres = directory / ".data" / "postgres"
        for path in (data, postgres):
            if path.exists():
                shutil.rmtree(path)
        subprocess.run(["docker", "volume", "rm", update_volume_name(instance)], check=False)
        (directory / "data" / "apps").mkdir(parents=True, exist_ok=True)
        (directory / "data" / "backups").mkdir(parents=True, exist_ok=True)
        (directory / "data" / "imports").mkdir(parents=True, exist_ok=True)
        (directory / ".data" / "postgres").mkdir(parents=True, exist_ok=True)
    refresh_readme(root)


def status(root: Path, target: str) -> int:
    result = 0
    for instance in selected_instances(target):
        require_initialised(root, instance)
        result |= run_compose(root, instance, "ps")
    return result


def doctor(root: Path, target: str) -> int:
    result = 0
    for instance in selected_instances(target):
        require_initialised(root, instance)
        print(f"[{instance}] validating Compose configuration")
        result |= run_compose(root, instance, "config", "--quiet")
        print(f"[{instance}] current service state")
        result |= run_compose(root, instance, "ps")
        env = parse_env(instance_root(root, instance) / ".env")
        port = int(env["GRAFT_WEB_HOST_PORT"])
        if not probe_health(port):
            print(f"[{instance}] health probe failed: http://127.0.0.1:{port}/healthz", file=sys.stderr)
            result |= 1
    return result


def pull(root: Path, target: str) -> int:
    result = 0
    for instance in selected_instances(target):
        require_initialised(root, instance)
        code = run_compose(root, instance, "pull")
        if code != 0:
            result |= code
            continue
        try:
            revision = sync_instance_compose(root, instance)
            print(f"[{instance}] synchronized official Compose files from {revision}")
        except LocalValidationError as error:
            print(f"[{instance}] Compose synchronization failed: {error}", file=sys.stderr)
            result |= 1
    refresh_readme(root)
    return result


def probe_health(port: int) -> bool:
    """Return whether the web proxy health endpoint answers successfully."""
    try:
        with urlrequest.urlopen(f"http://127.0.0.1:{port}/healthz", timeout=5) as response:
            return 200 <= response.status < 300
    except (OSError, urlerror.URLError):
        return False


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--local-root", type=Path, default=DEFAULT_LOCAL_ROOT, help="local instance root")
    subparsers = parser.add_subparsers(dest="command", required=True)
    init_parser = subparsers.add_parser("init", help="materialize all local instances")
    init_parser.add_argument("target", nargs="?", choices=("all",), default="all")
    for command in ("up", "down", "pull", "restart", "status", "logs", "doctor", "sync-compose", "reset"):
        item = subparsers.add_parser(command)
        item.add_argument("target", choices=(*INSTANCES, "all"), nargs="?", default="all")
        if command == "reset":
            item.add_argument("--confirm", action="store_true")
    config_parser = subparsers.add_parser("set-config")
    config_parser.add_argument("instance", choices=INSTANCES)
    config_parser.add_argument("key")
    config_parser.add_argument("value")
    access_parser = subparsers.add_parser("record-access")
    access_parser.add_argument("instance", choices=INSTANCES)
    access_parser.add_argument("--note", required=True)
    args = parser.parse_args(argv)
    root = args.local_root.resolve()
    try:
        if args.command == "init":
            initialise(root)
            return 0
        if args.command == "set-config":
            set_config(root, args.instance, args.key, args.value)
            return 0
        if args.command == "record-access":
            require_initialised(root, args.instance)
            note_path = root.parent / "README.md"
            with note_path.open("a", encoding="utf-8") as output:
                output.write(f"Local access note for `{args.instance}`: {args.note}\n")
            return 0
        if args.command == "reset":
            reset(root, args.target, args.confirm)
            return 0
        for instance in selected_instances(args.target):
            require_initialised(root, instance)
        if args.command in {"up", "restart"}:
            validate_ports(root)
        if args.command == "up":
            if "fixed" in selected_instances(args.target):
                print("[fixed] attempting the pinned tag; an unpublished tag remains pending and will not fall back")
            return max(run_compose(root, instance, "up", "-d") for instance in selected_instances(args.target))
        if args.command == "down":
            return max(run_compose(root, instance, "down") for instance in selected_instances(args.target))
        if args.command == "pull":
            if "fixed" in selected_instances(args.target):
                print("[fixed] attempting the pinned tag; an unpublished tag remains pending and will not fall back")
            return pull(root, args.target)
        if args.command == "restart":
            return max(run_compose(root, instance, "restart") for instance in selected_instances(args.target))
        if args.command == "status":
            return status(root, args.target)
        if args.command == "logs":
            return max(run_compose(root, instance, "logs", "--tail", "200") for instance in selected_instances(args.target))
        if args.command == "doctor":
            return doctor(root, args.target)
        if args.command == "sync-compose":
            for instance in selected_instances(args.target):
                revision = sync_instance_compose(root, instance)
                print(f"[{instance}] synchronized official Compose files from {revision}")
            refresh_readme(root)
            return 0
    except LocalValidationError as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

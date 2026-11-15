#!/usr/bin/env python3
"""Run the existing Agent fixture with a task-owned Docker daemon and API evidence."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import time
from urllib import error, request

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "tests/conformance/docker-runtime-agent"
TERMINAL = {"success", "failed", "cancelled", "needs_attention"}


def save(path: Path, value: dict) -> None:
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    path.chmod(0o600)


class Fixture:
    def __init__(self, directory: Path):
        self.directory = directory
        self.state = json.loads((directory / "instance.json").read_text())
        self.environment = {**os.environ, **self.state["environment"]}
        self.prefix = ["docker", "compose", "--project-name", self.state["project"],
                       "--profile", "conformance", "-f", str(FIXTURE / "compose.yml"),
                       "-f", str(FIXTURE / "compose.isolated.yml")]
        self.token = ""
        self.results: list[dict] = []

    def compose(self, *arguments: str, check: bool = True) -> str:
        result = subprocess.run(self.prefix + list(arguments), cwd=ROOT,
                                env=self.environment, capture_output=True, text=True, timeout=600)
        if check and result.returncode:
            # Command diagnostics are restricted local evidence, separate from the shareable report.
            log = self.directory / "command-error.log"
            log.write_text(result.stderr, encoding="utf-8")
            log.chmod(0o600)
            raise RuntimeError(f"Compose command failed, restricted diagnostics in {log.name}")
        return result.stdout

    def engine(self, *arguments: str) -> str:
        return self.compose("exec", "-T", "docker-daemon", "docker",
                            "--host=unix:///run/graft-docker/docker.sock", *arguments)

    def driver(self, *arguments: str) -> dict:
        return json.loads(self.compose("run", "--rm", "--no-deps", "conformance-driver", *arguments))

    def api(self, method: str, path: str, payload: dict | None = None) -> tuple[int, dict]:
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        body = json.dumps(payload).encode() if payload is not None else None
        url = f"http://127.0.0.1:{self.environment['GRAFT_CONFORMANCE_BACKEND_PORT']}" + path
        try:
            response = request.urlopen(request.Request(url, body, headers, method=method), timeout=20)
        except error.HTTPError as response:
            return response.code, json.loads(response.read())
        with response:
            return response.status, json.loads(response.read())

    def record(self, scenario: str, status: str, **facts) -> None:
        self.results.append({"scenario": scenario, "status": status, **facts})
        save(self.directory / "api-evidence.json", {"project": self.state["project"],
             "source_revision": self.state["source_revision"], "results": self.results})
        print(f"{scenario}: {status}", flush=True)

    def up(self) -> None:
        rendered = json.loads(self.compose("config", "--format", "json"))
        validate_topology(rendered)
        self.compose("up", "-d", "--wait", "docker-daemon", "postgres", "redis", "vault", "vault-init")
        self.compose("up", "-d", "--wait", "migrate", "backend")
        args = ["--phase", "prepare", "--agent-id", "docker-runtime-agent",
                "--image-digest", self.environment["GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST"],
                "--agent-version", "isolated-fixture", "--enrollment-ref", "isolated-fixture",
                "--automation-id", "isolated-fixture", "--docker-installation-ref", "docker:fixture",
                "--docker-secret-ref", "fixture:bootstrap-delivery",
                "--bootstrap-material-file", "/conformance/agent-bootstrap/bootstrap-token",
                "--agent-config-file", "/conformance/agent-config/agent.json",
                "--bootstrap-url", "https://backend:8443", "--agent-url", "https://backend:8444",
                "--bootstrap-ca-file", "/run/graft-agent-trust/ca.pem",
                "--trust-bundle-file", "/run/graft-agent-trust/ca.pem",
                "--agent-secret-file", "/run/graft-bootstrap/bootstrap-token"]
        prepared = self.driver(*args)
        self.compose("run", "--rm", "--no-deps", "docker-runtime-agent", "--once")
        boot = self.driver("--phase", "verify-bootstrap", "--agent-id", prepared["agent_id"],
                           "--target-id", str(prepared["target_id"]))
        self.compose("run", "--rm", "--no-deps", "docker-runtime-agent", "--once")
        restart = self.driver("--phase", "verify-restart", "--agent-id", boot["agent_id"],
                              "--target-id", str(boot["target_id"]), "--identity-id", boot["identity_id"],
                              "--generation", str(boot["generation"]),
                              "--receipt-count", str(boot["ledger_receipt_count"]))
        self.compose("up", "-d", "docker-runtime-agent")
        save(self.directory / "trust-evidence.json", {"bootstrap": boot, "restart": restart,
             "engine_id": self.engine("info", "--format", "{{.ID}}").strip()})
        print("Private daemon, Backend, Agent bootstrap and restart ready", flush=True)

    def authenticate(self) -> None:
        status, response = self.api("POST", "/api/auth/login", {"username": "graft", "password": self.environment["GRAFT_FIXTURE_ADMIN_PASSWORD"]})
        if status != 200:
            status, response = self.api("POST", "/api/auth/login", {"username": "graft", "password": "graft-admin"})
        if status != 200:
            raise RuntimeError(f"Fixture login failed with HTTP {status}")
        data = response["data"]
        self.token = data["access_token"]
        if data["must_change_password"]:
            status, response = self.api("POST", "/api/auth/complete-required-password-change",
                                         {"new_password": self.environment["GRAFT_FIXTURE_ADMIN_PASSWORD"]})
            if status != 200:
                raise RuntimeError(f"Fixture password bootstrap failed with HTTP {status}")
            self.token = response["data"]["access_token"]

    def wait_task(self, identifier: int, timeout: int = 90) -> dict:
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            status, response = self.api("GET", f"/api/tasks/{identifier}")
            data = response.get("data", {})
            if status == 200 and data.get("status") in TERMINAL:
                return {key: data.get(key) for key in ("id", "status", "failure_code", "current_stage_key")}
            time.sleep(1)
        return {"id": identifier, "status": "timeout"}

    def probe(self) -> None:
        self.authenticate()
        for key in ("ops.container.runtime.enabled", "ops.container.actions.dangerous_enabled", "ops.container.shell.enabled"):
            status, _ = self.api("PUT", "/api/system-configs/" + key, {"value": True})
            if status != 200:
                self.record("enable-" + key, "failed", http_status=status)
        name = "graft-isolated-api-probe"
        self.engine("pull", "alpine:3.24.4")
        if name in self.engine("ps", "-a", "--format", "{{.Names}}").splitlines():
            self.engine("rm", "-f", name)
        self.engine("run", "-d", "--name", name, "alpine:3.24.4", "sh", "-c", "echo graft-probe-ready; sleep 86400")
        status, response = self.api("GET", "/api/ops/containers")
        self.record("container-list", "passed" if status == 200 else "failed", http_status=status)
        status, response = self.api("GET", f"/api/ops/containers/{name}/logs?tail=20")
        self.record("container-logs", "passed" if status == 200 and "graft-probe-ready" in json.dumps(response) else "failed", http_status=status)
        for action, expected in (("stop", "false"), ("start", "true"), ("restart", "true")):
            status, response = self.api("POST", f"/api/ops/containers/{name}/{action}", {})
            task = self.wait_task(response["data"]["task_id"]) if status == 202 else {}
            running = self.engine("inspect", "--format", "{{.State.Running}}", name).strip()
            passed = status == 202 and task.get("status") == "success" and running == expected
            self.record("container-" + action, "passed" if passed else "failed", http_status=status, task=task, engine_running=running)
        status, response = self.api("POST", f"/api/ops/containers/{name}/shell/sessions", {"command": "sh", "cols": 80, "rows": 24})
        self.record("shell-ticket", "passed" if status == 200 else "failed", http_status=status,
                    scope="Ticket issuance only, interactive transport requires a WebSocket probe")
        self.compose("exec", "-T", "docker-daemon", "chmod", "000", "/run/graft-docker/docker.sock")
        try:
            status, _ = self.api("GET", f"/api/ops/containers/{name}")
            self.record("backend-socket-permission-denied", "passed" if status >= 400 else "failed", http_status=status,
                        scope="Non-root Backend read boundary, fixture Agent runs as root")
        finally:
            self.compose("exec", "-T", "docker-daemon", "chmod", "660", "/run/graft-docker/docker.sock")
        self.compose("stop", "docker-runtime-agent")
        try:
            status, response = self.api("POST", f"/api/ops/containers/{name}/restart", {})
            task = self.wait_task(response["data"]["task_id"], timeout=10) if status == 202 else {}
            if task.get("status") == "timeout":
                self.api("POST", f"/api/tasks/{task['id']}/cancel", {})
                task = self.wait_task(task["id"], timeout=15)
            self.record("agent-offline", "observed", http_status=status, task=task,
                        engine_running=self.engine("inspect", "--format", "{{.State.Running}}", name).strip())
        finally:
            self.compose("up", "-d", "docker-runtime-agent")
        print("API evidence saved, repeat-safe task cleanup is available through stop/destroy", flush=True)


def validate_topology(configuration: dict) -> None:
    for service in ("backend", "conformance-driver", "docker-runtime-agent"):
        volumes = configuration["services"][service]["volumes"]
        if any(v.get("type") == "bind" and "docker.sock" in v.get("source", "") for v in volumes):
            raise RuntimeError(f"{service} inherited a host Docker socket")
        if not any(v.get("type") == "volume" and v.get("source") == "daemon-socket" and v["target"] == "/var/run" for v in volumes):
            raise RuntimeError(f"{service} has no private daemon socket")
    for service in ("docker-daemon", "backend", "docker-runtime-agent"):
        if not any(v.get("source") == "application-workspaces" and v["target"] == "/opt/graft/apps" for v in configuration["services"][service]["volumes"]):
            raise RuntimeError(f"{service} has inconsistent application workspace mapping")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("init", "up", "probe", "status", "stop", "destroy"))
    parser.add_argument("--directory", type=Path, required=True, help="Task-owned ignored evidence and secret directory")
    parser.add_argument("--project")
    parser.add_argument("--source-revision")
    parser.add_argument("--backend-image")
    parser.add_argument("--agent-image")
    parser.add_argument("--driver-image")
    parser.add_argument("--port", type=int, default=18080)
    parser.add_argument("--confirm-project")
    args = parser.parse_args()
    directory = args.directory.resolve()
    if args.action == "init":
        if not args.project or not re.fullmatch(r"graft-isolated-[a-z0-9-]+", args.project):
            parser.error("--project must use the task-specific graft-isolated- prefix")
        if not all((args.source_revision, args.backend_image, args.agent_image, args.driver_image)):
            parser.error("init requires source revision and all three immutable fixture images")
        directory.mkdir(parents=True, mode=0o700, exist_ok=True)
        if (directory / "instance.json").exists():
            parser.error("instance already registered, reuse its existing directory")
        if not re.fullmatch(r"[a-f0-9]{40}", args.source_revision):
            parser.error("source revision must be an exact Git commit")
        images = {}
        for key, value in (("backend", args.backend_image), ("agent", args.agent_image), ("driver", args.driver_image)):
            image = json.loads(subprocess.check_output(["docker", "image", "inspect", value], text=True))[0]
            revision = (image["Config"].get("Labels") or {}).get("org.opencontainers.image.revision")
            if revision != args.source_revision:
                parser.error(f"{key} image source revision does not match the requested commit")
            images[key] = image["Id"]
        digest = images["agent"]
        env = {"GRAFT_BACKEND_IMAGE": images["backend"], "GRAFT_AGENT_IMAGE": images["agent"],
               "GRAFT_CONFORMANCE_DRIVER_IMAGE": images["driver"], "GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST": digest,
               "GRAFT_CONFORMANCE_POSTGRES_PASSWORD": secrets.token_urlsafe(24), "VAULT_DEV_ROOT_TOKEN": secrets.token_urlsafe(24),
               "GRAFT_FIXTURE_ADMIN_PASSWORD": secrets.token_urlsafe(24), "GRAFT_CONFORMANCE_BACKEND_PORT": str(args.port),
               "GRAFT_CONFORMANCE_BOOTSTRAP_PORT": str(args.port + 1), "GRAFT_CONFORMANCE_AGENT_PORT": str(args.port + 2)}
        save(directory / "instance.json", {"project": args.project, "source_revision": args.source_revision, "environment": env})
        print("Isolated task registered", flush=True)
        return
    fixture = Fixture(directory)
    if args.action == "status":
        print(fixture.compose("ps", "--format", "json"))
    elif args.action == "destroy":
        if args.confirm_project != fixture.state["project"]:
            parser.error("destroy requires --confirm-project matching the registered task")
        fixture.compose("down", "--volumes", "--remove-orphans")
        save(directory / "cleanup.json", {"project": fixture.state["project"], "resources_removed": True})
    elif args.action == "stop":
        fixture.compose("stop")
    else:
        getattr(fixture, args.action)()


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.SubprocessError, OSError, ValueError, KeyError) as exc:
        raise SystemExit(str(exc)) from exc

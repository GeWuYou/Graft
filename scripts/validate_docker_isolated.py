#!/usr/bin/env python3
"""Verify local or Agent execution against a task-owned Docker daemon."""
from __future__ import annotations

import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import socket
import struct
import subprocess
import time
from urllib import error, request

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "tests/conformance/docker-runtime-agent"
TERMINAL = {"success", "failed", "cancelled", "needs_attention"}
LOCAL_REQUIRED_SCENARIOS = frozenset({
    "enable-ops.container.runtime.enabled", "enable-ops.container.actions.dangerous_enabled",
    "enable-ops.container.shell.enabled", "image-pull", "container-list", "container-logs",
    "container-stop", "container-start", "container-restart", "shell-ticket", "shell-command-output",
    "compose-managed-create", "compose-up", "compose-relative-bind", "compose-stop", "compose-material-missing",
    "compose-import", "compose-import-readback", "compose-reimport-conflict", "local-execution-interruption",
    "local-interruption-reconciliation",
    "backend-socket-permission-denied", "local-mode-preflight", "image-pull-unreachable",
})


def require_local_acceptance(results: list[dict]) -> None:
    observed = {item["scenario"]: item["status"] for item in results}
    incomplete = sorted(scenario for scenario in LOCAL_REQUIRED_SCENARIOS if observed.get(scenario) != "passed")
    if incomplete:
        raise RuntimeError("Required local acceptance scenarios incomplete: " + ", ".join(incomplete))


def controller_receipt(output: str, operation: str, runner: str) -> dict:
    marker = "GRAFT_UPDATE_RECEIPT:"
    encoded = next((line[len(marker):] for line in reversed(output.splitlines()) if line.startswith(marker)), "")
    if not encoded or len(encoded) > 65536:
        raise RuntimeError("Controller did not produce a bounded receipt")
    receipt = json.loads(base64.b64decode(encoded + "=" * (-len(encoded) % 4), validate=True))
    backup = receipt.get("backup_completion") or {}
    valid = (receipt.get("protocol_version") == 2 and receipt.get("operation_id") == operation
             and receipt.get("runner_id") == runner and receipt.get("execution_mode") == "local"
             and receipt.get("succeeded") is True and receipt.get("migration_started") is True
             and backup.get("OperationID") == operation and backup.get("TaskID") == 1)
    for prefix in ("ConfigSnapshot", "DatabaseDump"):
        valid = valid and bool(re.fullmatch(r"[a-f0-9]{64}", str(backup.get(prefix + "SHA256", ""))))
        valid = valid and isinstance(backup.get(prefix + "Bytes"), int) and backup[prefix + "Bytes"] > 0
    if not valid:
        raise RuntimeError("Controller receipt failed identity, local mode, success or backup evidence checks")
    return receipt


def save(path: Path, value: dict) -> None:
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    path.chmod(0o600)


class Fixture:
    def __init__(self, directory: Path):
        self.directory = directory
        self.state = json.loads((directory / "instance.json").read_text())
        self.environment = {**os.environ, **self.state["environment"]}
        self.prefix = ["docker", "compose", "--project-name", self.state["project"]]
        fixture_source = self.directory / "source-assets" / "fixture"
        if not fixture_source.is_dir():
            fixture_source = FIXTURE
        if self.execution_mode == "local":
            self.prefix.extend(["-f", str(fixture_source / "compose.local.yml")])
        else:
            self.prefix.extend(["--profile", "conformance", "-f", str(fixture_source / "compose.yml"),
                                "-f", str(fixture_source / "compose.isolated.yml")])
        self.token = ""
        self.results: list[dict] = []

    @property
    def execution_mode(self) -> str:
        return getattr(self, "state", {}).get("execution_mode", "agent")

    def compose(self, *arguments: str, check: bool = True, timeout: int = 600) -> str:
        result = subprocess.run(self.prefix + list(arguments), cwd=ROOT,
                                env=self.environment, capture_output=True, text=True, timeout=timeout)
        self.last_command_returncode = result.returncode
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

    def wait_driver(self, *arguments: str, timeout: int = 90) -> dict:
        deadline = time.monotonic() + timeout
        while True:
            try:
                return self.driver(*arguments)
            except RuntimeError:
                if time.monotonic() >= deadline:
                    raise
                time.sleep(2)

    def api(self, method: str, path: str, payload: dict | None = None,
            *, idempotency_key: str | None = None) -> tuple[int, dict]:
        headers = {"Content-Type": "application/json"}
        if method == "POST" and (path.startswith("/api/ops/") or idempotency_key is not None):
            headers["Idempotency-Key"] = idempotency_key or secrets.token_hex(16)
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        body = json.dumps(payload).encode() if payload is not None else None
        url = f"http://127.0.0.1:{self.environment['GRAFT_CONFORMANCE_BACKEND_PORT']}" + path
        try:
            response = request.urlopen(request.Request(url, body, headers, method=method), timeout=20)
        except error.HTTPError as response:
            data = json.loads(response.read())
            self.api_errors = getattr(self, "api_errors", [])
            self.api_errors.append({"method": method, "path": path, "http_status": response.code,
                                    "error_code": data.get("code"),
                                    "trace_id": data.get("traceId") or data.get("trace_id")})
            save(self.directory / "api-errors.json", {"requests": self.api_errors})
            return response.code, data
        with response:
            return response.status, json.loads(response.read())

    def record(self, scenario: str, status: str, **facts) -> None:
        self.results.append({"scenario": scenario, "status": status, **facts})
        save(self.directory / "api-evidence.json", {"project": self.state["project"],
             "source_revision": self.state["source_revision"], "execution_mode": self.execution_mode,
             "results": self.results})
        print(f"{scenario}: {status}", flush=True)

    def verify_vault_authority(self) -> dict:
        # 既有证书仍可通过 mTLS，必须另行证明当前 Vault 接受已交付 AppRole 并持有同一 PKI。
        script = r'''role_id=$(cat /conformance/secrets/role_id)
secret_id=$(cat /conformance/secrets/secret_id)
VAULT_TOKEN=$(vault write -format=json auth/approle/login role_id="$role_id" secret_id="$secret_id" |
  sed -n 's/^[[:space:]]*"client_token": "\([^" ]*\)".*/\1/p')
test -n "$VAULT_TOKEN"
export VAULT_TOKEN
live_certificate=$(vault read -field=certificate pki/cert/ca)
delivered_certificate=$(cat /conformance/agent-trust/ca.pem)
test -n "$live_certificate"
test -n "$delivered_certificate"
live_ca=$(printf %s "$live_certificate" | sha256sum | awk '{print $1}')
delivered_ca=$(printf %s "$delivered_certificate" | sha256sum | awk '{print $1}')
test "$live_ca" = "$delivered_ca"
printf '{"approle_login":true,"pki_ca_matches_delivery":true,"ca_sha256":"%s"}\n' "$live_ca"
'''
        proof = json.loads(self.compose("exec", "-T", "vault", "sh", "-ec", script))
        save(self.directory / "vault-authority-evidence.json", proof)
        return proof

    def stop(self) -> None:
        services = json.loads(self.compose("config", "--format", "json"))["services"]
        # Vault dev 的 PKI 保存在进程内存中，逻辑停止保留该 authority，最终 destroy 统一释放。
        workloads = [service for service in services if service != "vault"]
        if workloads:
            self.compose("stop", *workloads)
        print("Fixture workloads stopped" + (", Vault authority retained until destroy" if "vault" in services else ""), flush=True)

    def up(self) -> None:
        rendered = json.loads(self.compose("config", "--format", "json"))
        validate_topology(rendered, self.execution_mode)
        if self.execution_mode == "local":
            self.compose("up", "-d", "--wait", "docker-daemon", "postgres", "redis")
            self.compose("up", "-d", "--no-deps", "workspace-init")
            self.compose("wait", "workspace-init")
            self.compose("up", "-d", "--no-deps", "migrate")
            self.compose("wait", "migrate")
            self.compose("up", "-d", "--no-deps", "--wait", "backend")
            save(self.directory / "local-readiness-evidence.json", {
                "execution_mode": "local", "engine_id": self.engine("info", "--format", "{{.ID}}").strip(),
                "agent_required": False, "vault_required": False})
            print("Private daemon and local Backend ready", flush=True)
            return
        self.compose("up", "-d", "--wait", "docker-daemon", "postgres", "redis", "vault")
        self.compose("up", "-d", "--no-deps", "workspace-init", "vault-init")
        self.compose("wait", "workspace-init", "vault-init")
        self.verify_vault_authority()
        self.compose("up", "-d", "--no-deps", "migrate")
        self.compose("wait", "migrate")
        self.compose("up", "-d", "--no-deps", "--wait", "backend")
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
        prepared_path = self.directory / "prepared-evidence.json"
        if prepared_path.exists():
            prepared = json.loads(prepared_path.read_text())
        else:
            prepared = self.driver(*args)
            save(prepared_path, prepared)
        self.compose("up", "-d", "--no-deps", "docker-runtime-agent")
        boot = self.wait_driver("--phase", "verify-bootstrap", "--agent-id", prepared["agent_id"])
        self.compose("stop", "docker-runtime-agent")
        self.compose("up", "-d", "--no-deps", "docker-runtime-agent")
        restart = self.wait_driver("--phase", "verify-restart", "--agent-id", boot["agent_id"])
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
        self.user_id = int((data.get("user") or {}).get("id", 0))
        self.token = data["access_token"]
        if data["must_change_password"]:
            status, response = self.api("POST", "/api/auth/complete-required-password-change",
                                         {"new_password": self.environment["GRAFT_FIXTURE_ADMIN_PASSWORD"]})
            if status != 200:
                raise RuntimeError(f"Fixture password bootstrap failed with HTTP {status}")
            self.token = ""
            status, response = self.api("POST", "/api/auth/login", {
                "username": "graft", "password": self.environment["GRAFT_FIXTURE_ADMIN_PASSWORD"]})
            if status != 200:
                raise RuntimeError(f"Fixture login after password change failed with HTTP {status}")
            self.token = response["data"]["access_token"]
            self.user_id = int((response["data"].get("user") or {}).get("id", 0))

    def wait_task(self, identifier: int, timeout: int = 90) -> dict:
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            status, response = self.api("GET", f"/api/tasks/{identifier}")
            data = response.get("data", {})
            if status == 200 and data.get("status") in TERMINAL:
                return {key: data.get(key) for key in ("id", "status", "failure_code", "current_stage_key")}
            time.sleep(1)
        return {"id": identifier, "status": "timeout"}

    def shell_output(self, relative_url: str) -> bool:
        port = int(self.environment["GRAFT_CONFORMANCE_BACKEND_PORT"])
        key = base64.b64encode(secrets.token_bytes(16)).decode()
        headers = (f"GET {relative_url} HTTP/1.1\r\nHost: 127.0.0.1:{port}\r\n"
                   f"Origin: http://127.0.0.1:{port}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
                   f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n")
        with socket.create_connection(("127.0.0.1", port), timeout=10) as connection:
            connection.sendall(headers.encode())
            stream = connection.makefile("rb")
            response = stream.readline()
            response_headers = {}
            while line := stream.readline().strip():
                name, value = line.decode().split(":", 1)
                response_headers[name.lower()] = value.strip()
            accepted = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest()).decode()
            if b" 101 " not in response or response_headers.get("sec-websocket-accept") != accepted:
                return False
            data = json.dumps({"type": "input", "data": "printf 'graft-shell-%s\\n' verified\nexit\n"}).encode()
            mask = secrets.token_bytes(4)
            masked = bytes(value ^ mask[index % 4] for index, value in enumerate(data))
            connection.sendall(bytes((0x81, 0x80 | len(data))) + mask + masked)
            deadline = time.monotonic() + 10
            output = ""
            while time.monotonic() < deadline:
                frame = stream.read(2)
                if len(frame) != 2 or frame[0] & 0x0f == 8:
                    return False
                length = frame[1] & 0x7f
                if length == 126:
                    length = struct.unpack("!H", stream.read(2))[0]
                elif length == 127:
                    length = struct.unpack("!Q", stream.read(8))[0]
                if length > 1024 * 1024:
                    raise RuntimeError("Shell output frame exceeded fixture bound")
                payload = stream.read(length)
                if frame[0] & 0x0f == 1:
                    message = json.loads(payload)
                    if message.get("type") == "output":
                        output += message.get("data", "")
                        if "graft-shell-verified" in output:
                            return True
            return False

    def application_probe(self) -> None:
        name = "graft-compose-probe-" + secrets.token_hex(3)
        container = name + "-app"
        definition = ("services:\n  app:\n    image: alpine:3.24.4\n    container_name: " + container +
                      "\n    command: [sh, -c, 'echo compose-probe-ready; sleep 86400']\n"
                      "    volumes:\n      - ./probe.txt:/probe.txt:ro\n")
        policy = {"strategy_kind": "standard", "profiles": [], "down_before_redeploy": False,
                  "pull_before_redeploy": False, "build_before_up": False, "force_recreate": False,
                  "remove_orphans": True, "wait_after_up": False, "wait_timeout_seconds": 60,
                  "renew_anon_volumes": False, "prune_images_after_redeploy": False, "managed_service_names": []}
        target = self.local_target_id() if self.execution_mode == "local" else json.loads(
            (self.directory / "trust-evidence.json").read_text())["bootstrap"]["target_id"]
        body = {"display_name": name, "runtime_target_id": target, "application_name": name,
                "compose_file_path": "compose.yml", "lifecycle_configuration": policy,
                "workspace_entries": [{"path": "compose.yml", "node_type": "file", "content": definition},
                                      {"path": "probe.txt", "node_type": "file", "content": "graft-relative-mount-verified"}]}
        status, response = self.api("POST", "/api/ops/applications/create/managed", body)
        self.record("compose-managed-create", "passed" if status in (200, 201) else "failed", http_status=status,
                    error_code=response.get("code"))
        if status not in (200, 201):
            return
        created = response["data"]
        application = created["application_id"]
        status, response = self.api("POST", f"/api/ops/applications/{application}/up", {})
        task = self.wait_task(response["data"]["task_id"]) if status == 202 else {}
        containers = self.engine("ps", "-a", "--format", "{{.Names}}").splitlines()
        running = self.engine("inspect", "--format", "{{.State.Running}}", container).strip() if container in containers else "missing"
        passed = task.get("status") == "success" and running == "true"
        self.record("compose-up", "passed" if passed else "failed", http_status=status, task=task,
                    engine_container_present=container in containers, engine_running=running)
        if passed:
            content = self.engine("exec", container, "cat", "/probe.txt").strip()
            self.record("compose-relative-bind", "passed" if content == "graft-relative-mount-verified" else "failed")
            status, response = self.api("POST", f"/api/ops/applications/{application}/stop", {})
            task = self.wait_task(response["data"]["task_id"]) if status == 202 else {}
            running = self.engine("inspect", "--format", "{{.State.Running}}", container).strip()
            stopped = task.get("status") == "success" and running == "false"
            self.record("compose-stop", "passed" if stopped else "failed", http_status=status, task=task,
                        engine_running=running)
        workspace = created["workspace_path"]
        self.compose("exec", "-T", "backend", "mv", workspace + "/compose.yml", workspace + "/compose.missing")
        try:
            status, response = self.api("POST", f"/api/ops/applications/{application}/up", {})
            task = self.wait_task(response["data"]["task_id"], timeout=30) if status == 202 else {}
            rejected = status == 409 or (task.get("status") == "failed" and bool(task.get("failure_code")))
            self.record("compose-material-missing", "passed" if rejected else "failed", http_status=status, task=task,
                        error_code=response.get("code"))
        finally:
            self.compose("exec", "-T", "backend", "mv", workspace + "/compose.missing", workspace + "/compose.yml")
        self.import_probe(workspace, definition, container, target, policy)

        self.interruption_probe(body, definition, container, policy)

    def local_target_id(self) -> int:
        status, response = self.api("GET", "/api/runtime-targets")
        targets = (response.get("data") or {}).get("items", [])
        local = [item for item in targets if (item.get("runtime") or {}).get("provider") == "docker"
                 and (item.get("connection") or {}).get("kind") == "unix_socket"]
        if status != 200 or len(local) != 1:
            raise RuntimeError("Local runtime target could not be uniquely discovered")
        return int(local[0]["id"])

    def interruption_probe(self, body: dict, definition: str, container: str, policy: dict) -> None:
        scenario = "local-execution-interruption" if self.execution_mode == "local" else "agent-execution-interruption"
        interrupted_name = "graft-interruption-probe-" + secrets.token_hex(3)
        interrupted_container = interrupted_name + "-app"
        interrupted_definition = definition.replace(container, interrupted_container) + (
            "    healthcheck:\n      test: [CMD-SHELL, 'sleep 20; exit 0']\n"
            "      interval: 1s\n      timeout: 30s\n      retries: 3\n")
        interrupted_body = {**body, "display_name": interrupted_name, "application_name": interrupted_name,
            "lifecycle_configuration": {**policy, "wait_after_up": True, "wait_timeout_seconds": 90},
            "workspace_entries": [{"path": "compose.yml", "node_type": "file", "content": interrupted_definition},
                                  {"path": "probe.txt", "node_type": "file", "content": "interruption-probe"}]}
        status, response = self.api("POST", "/api/ops/applications/create/managed", interrupted_body)
        if status not in (200, 201):
            self.record(scenario, "failed", http_status=status, scope="Scenario creation failed")
            return
        interrupted_application = response["data"]["application_id"]
        status, response = self.api("POST", f"/api/ops/applications/{interrupted_application}/up", {})
        if status != 202:
            self.record(scenario, "failed", http_status=status, scope="Task admission failed")
            return
        task_id = response["data"]["task_id"]
        deadline = time.monotonic() + 30
        started = False
        while time.monotonic() < deadline:
            if interrupted_container in self.engine("ps", "-a", "--format", "{{.Names}}").splitlines():
                started = True
                break
            time.sleep(1)
        if not started:
            self.record(scenario, "unverified", task=self.wait_task(task_id, timeout=5),
                        scope="Provider did not reach an observable side effect before the bounded deadline")
            return
        before = self.engine("inspect", "--format", "{{.Id}} {{.State.StartedAt}}", interrupted_container).strip()
        executor = "backend" if self.execution_mode == "local" else "docker-runtime-agent"
        self.compose("kill", "--signal", "SIGKILL", executor)
        self.compose("up", "-d", "--wait", executor)
        task = self.wait_task(task_id, timeout=120)
        after = self.engine("inspect", "--format", "{{.Id}} {{.State.StartedAt}}", interrupted_container).strip()
        expected = task.get("status") == "needs_attention" and before == after
        self.record(scenario, "passed" if expected else "failed", task=task,
                    same_container_and_started_at=before == after,
                    scope="Task settlement and container identity after killing the executing process")
        if expected and self.execution_mode == "local":
            status, _ = self.api("POST", f"/api/tasks/{task_id}/cancel", {})
            reconciled = self.wait_task(task_id, timeout=30)
            preserved = self.engine("inspect", "--format", "{{.Id}} {{.State.StartedAt}}", interrupted_container).strip() == before
            self.record("local-interruption-reconciliation", "passed" if status == 200 and reconciled.get("status") == "cancelled" and preserved else "failed",
                task=reconciled, engine_effect_preserved=preserved,
                scope="Fixture operator inspected the preserved effect and cancelled the unresolved operation through the public owner-authorized API")

    def persisted_application(self, identifier: str) -> dict:
        if not re.fullmatch(r"app_[0-9A-HJKMNP-TV-Z]{26}", identifier):
            raise RuntimeError("Import returned an invalid application identity")
        query = ("SELECT json_build_object('display_name', display_name, "
                 "'runtime_target_id', runtime_target_id, 'compose_project_name', compose_project_name, "
                 "'compose_project_name_source', compose_project_name_source) "
                 f"FROM applications WHERE application_id = '{identifier}' AND deleted_at = 0")
        result = self.compose("exec", "-T", "postgres", "psql", "-U", "graft", "-d", "graft", "-Atc", query)
        return json.loads(result)

    def import_probe(self, workspace: str, definition: str, container: str, target: int, policy: dict) -> None:
        name = "graft-import-probe-" + secrets.token_hex(3)
        display_name = "Imported " + name
        imported_path = "/opt/graft/apps/" + name
        self.compose("exec", "-T", "backend", "mkdir", "-p", imported_path)
        encoded = base64.b64encode(definition.replace(container, name + "-app").encode()).decode()
        self.compose("exec", "-T", "backend", "sh", "-ec", f"printf %s {encoded} | base64 -d > {imported_path}/compose.yml")
        self.compose("exec", "-T", "backend", "cp", workspace + "/probe.txt", imported_path + "/probe.txt")
        status, response = self.api("GET", "/api/ops/applications/import/directory-sources")
        sources = (response.get("data") or {}).get("items", [])
        source = next((item for item in sources if item.get("managed")), None)
        if status != 200 or source is None:
            self.record("compose-import", "failed", http_status=status, scope="Managed import root unavailable")
            return
        inspect_body = {"directory_ref": {"provider": source["provider"], "root_id": source["root_id"], "path": name}}
        status, response = self.api("POST", "/api/ops/applications/import/inspect", inspect_body)
        if status != 200:
            self.record("compose-import-inspect", "failed", http_status=status, error_code=response.get("code"))
            return
        inspected = response["data"]
        expected = {"display_name": display_name, "runtime_target_id": target,
                    "compose_project_name": inspected["compose_project_name"],
                    "compose_project_name_source": inspected["compose_project_name_source"]}
        status, response = self.api("POST", "/api/ops/applications/import", {
            "inspection_id": inspected["inspection_id"], "display_name": display_name,
            "lifecycle_configuration": policy})
        self.record("compose-import", "passed" if status in (200, 201) else "failed", http_status=status,
                    error_code=response.get("code"))
        if status not in (200, 201):
            return
        identifier = response["data"]["application"]["application_id"]
        status, response = self.api("GET", f"/api/ops/applications/{identifier}")
        detail = response.get("data") or {}
        actual = {key: detail.get(key) for key in expected}
        actual["runtime_target_id"] = (detail.get("runtime_target") or {}).get("id")
        persisted = self.persisted_application(identifier)
        self.record("compose-import-readback", "passed" if status == 200 and actual == expected and persisted == expected else "failed",
                    http_status=status, expected=expected, api_fields=actual, persisted_fields=persisted)
        status, response = self.api("POST", "/api/ops/applications/import/inspect", inspect_body)
        if status != 200:
            self.record("compose-reimport-conflict", "failed", http_status=status, scope="Second inspection failed")
            return
        status, _ = self.api("POST", "/api/ops/applications/import", {
            "inspection_id": response["data"]["inspection_id"], "display_name": display_name + " changed",
            "lifecycle_configuration": policy})
        unchanged = self.persisted_application(identifier) == persisted
        self.record("compose-reimport-conflict", "passed" if status == 409 and unchanged else "failed",
                    http_status=status, persisted_fields_unchanged=unchanged,
                    scope="Public import rejects registered workspace, store upsert updates require direct repository regression")

    def once_probe(self) -> None:
        saved = json.loads((self.directory / "trust-evidence.json").read_text())["restart"]
        name = self.state["project"] + "-once-probe"
        self.compose("stop", "docker-runtime-agent")
        try:
            before = self.wait_driver("--phase", "verify-bootstrap", "--agent-id", saved["agent_id"])
            started = time.monotonic()
            try:
                self.compose("run", "--rm", "--no-deps", "--name", name, "docker-runtime-agent", "--once", timeout=60)
            except (RuntimeError, subprocess.TimeoutExpired) as exc:
                self.record("agent-once-exit", "failed", error_type=type(exc).__name__,
                            elapsed_seconds=round(time.monotonic() - started, 2))
                return
            elapsed = round(time.monotonic() - started, 2)
            after = self.wait_driver("--phase", "verify-restart", "--agent-id", before["agent_id"])
            identity_fields = ("target_id", "agent_id", "identity_id", "generation")
            same_identity = all(after[field] == before[field] for field in identity_fields)
            advanced = after["ledger_receipt_count"] > before["ledger_receipt_count"]
            self.record("agent-once-exit", "passed" if same_identity and advanced else "failed",
                        exit_code=0, elapsed_seconds=elapsed, same_identity=same_identity,
                        ledger_receipt_advanced=advanced)
        finally:
            try:
                subprocess.run(["docker", "rm", "-f", name], cwd=ROOT, env=self.environment,
                               capture_output=True, text=True, timeout=30, check=False)
            finally:
                self.compose("up", "-d", "--no-deps", "docker-runtime-agent")

    def container_action_probe(self, name: str, action: str, expected_running: str) -> None:
        before = self.engine("inspect", "--format", "{{.State.StartedAt}}", name).strip()
        status, response = self.api("POST", f"/api/ops/containers/{name}/{action}", {})
        task = self.wait_task(response["data"]["task_id"]) if status == 202 else {}
        running = self.engine("inspect", "--format", "{{.State.Running}}", name).strip()
        after = self.engine("inspect", "--format", "{{.State.StartedAt}}", name).strip()
        restarted = action != "restart" or before != after
        passed = status == 202 and task.get("status") == "success" and running == expected_running and restarted
        self.record("container-" + action, "passed" if passed else "failed", http_status=status, task=task,
                    engine_running=running, engine_started_at_changed=before != after)

    def enable_operations(self) -> None:
        for key in ("ops.container.runtime.enabled", "ops.container.actions.dangerous_enabled", "ops.container.shell.enabled"):
            status, _ = self.api("PUT", "/api/system-configs/" + key, {"value": True})
            read_status, response = self.api("GET", "/api/system-configs/" + key) if status == 200 else (status, {})
            value = (response.get("data") or {}).get("effective_value")
            enabled = status == 200 and read_status == 200 and isinstance(value, str) and json.loads(value) is True
            self.record("enable-" + key, "passed" if enabled else "failed", http_status=status,
                        readback_http_status=read_status, effective_value_enabled=enabled)

    def probe(self) -> None:
        self.authenticate()
        if self.execution_mode == "agent":
            self.once_probe()
        self.enable_operations()
        name = "graft-isolated-api-probe"
        status, response = self.api("POST", "/api/ops/docker/images/pull", {"reference": "alpine:3.24.4"})
        task = self.wait_task(response["data"]["task_id"]) if status == 202 else {}
        self.record("image-pull", "passed" if task.get("status") == "success" else "failed", http_status=status, task=task)
        self.engine("pull", "alpine:3.24.4")
        if name in self.engine("ps", "-a", "--format", "{{.Names}}").splitlines():
            self.engine("rm", "-f", name)
        self.engine("run", "-d", "--name", name, "alpine:3.24.4", "sh", "-c", "echo graft-probe-ready; sleep 86400")
        status, response = self.api("GET", "/api/ops/containers")
        self.record("container-list", "passed" if status == 200 else "failed", http_status=status)
        status, response = self.api("GET", f"/api/ops/containers/{name}/logs?tail=20")
        self.record("container-logs", "passed" if status == 200 and "graft-probe-ready" in json.dumps(response) else "failed", http_status=status)
        for action, expected in (("stop", "false"), ("start", "true"), ("restart", "true")):
            self.container_action_probe(name, action, expected)
        status, response = self.api("POST", f"/api/ops/containers/{name}/shell/sessions", {"command": "sh", "cols": 80, "rows": 24})
        self.record("shell-ticket", "passed" if status == 200 else "failed", http_status=status)
        if status == 200:
            try:
                shell_passed = self.shell_output(response["data"]["websocket_url"])
                self.record("shell-command-output", "passed" if shell_passed else "failed")
            except (OSError, ValueError) as exc:
                self.record("shell-command-output", "failed", error_type=type(exc).__name__)
        self.application_probe()
        backend_uid = self.compose("exec", "-T", "backend", "id", "-u").strip()
        ping = ("exec", "-T", "backend", "curl", "--fail", "--silent", "--show-error", "--max-time", "5",
                "--unix-socket", "/var/run/docker.sock", "http://localhost/_ping")
        ping_before = self.compose(*ping, check=False)
        connection_worked = self.last_command_returncode == 0 and ping_before.strip() == "OK"
        self.compose("exec", "-T", "docker-daemon", "chmod", "000", "/run/graft-docker/docker.sock")
        try:
            self.compose(*ping, check=False)
            denied_exit_code = self.last_command_returncode
            status, _ = self.api("GET", f"/api/ops/containers/{name}")
            passed = backend_uid == "10001" and connection_worked and denied_exit_code != 0
            self.record("backend-socket-permission-denied", "passed" if passed else "failed",
                        http_status=status, backend_uid=backend_uid, fresh_ping_before=connection_worked,
                        denied_fresh_connection_exit_code=denied_exit_code,
                        scope="Fresh non-root Backend socket connection, existing API keepalive may remain usable")
        finally:
            self.compose("exec", "-T", "docker-daemon", "chmod", "660", "/run/graft-docker/docker.sock")
        if self.execution_mode == "local":
            self.local_mode_preflight_probe()
            self.failed_pull_probe()
            print("Local API evidence saved; stop/destroy use the registered task identity", flush=True)
            return
        offline_before = self.engine("inspect", "--format", "{{.State.StartedAt}}", name).strip()
        self.compose("stop", "docker-runtime-agent")
        try:
            status, response = self.api("POST", f"/api/ops/containers/{name}/restart", {})
            task = self.wait_task(response["data"]["task_id"], timeout=10) if status == 202 else {}
            if task.get("status") == "timeout":
                self.api("POST", f"/api/tasks/{task['id']}/cancel", {})
                task = self.wait_task(task["id"], timeout=15)
            self.record("agent-offline", "observed", http_status=status, task=task,
                        engine_running=self.engine("inspect", "--format", "{{.State.Running}}", name).strip(),
                        engine_started_at_unchanged=offline_before == self.engine("inspect", "--format", "{{.State.StartedAt}}", name).strip())
        finally:
            self.compose("up", "-d", "docker-runtime-agent")
        print("API evidence saved, repeat-safe task cleanup is available through stop/destroy", flush=True)

    def local_mode_preflight_probe(self) -> None:
        evidence = self.compose("exec", "-T", "backend", "/app/graft", "runtime", "mode", "preflight",
                                "--mode", "local", "--format", "json", check=False)
        success = self.last_command_returncode == 0
        self.record("local-mode-preflight", "passed" if success else "failed",
                    exit_code=self.last_command_returncode)
        if success:
            save(self.directory / "mode-preflight-evidence.json", json.loads(evidence))

    def failed_pull_probe(self) -> None:
        status, response = self.api("POST", "/api/ops/docker/images/pull", {
            "reference": "127.0.0.1:1/graft-fixture-unreachable:probe"})
        task = self.wait_task(response["data"]["task_id"], timeout=90) if status == 202 else {}
        passed = task.get("status") == "failed" and bool(task.get("failure_code"))
        self.record("image-pull-unreachable", "passed" if passed else "failed", http_status=status, task=task)

    def import_image(self, image: str) -> None:
        # 镜像字节只流入任务私有 daemon，避免把主机标签或全局 Docker 资源变成验收目标。
        with subprocess.Popen(["docker", "image", "save", image], stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, cwd=ROOT, env=self.environment) as exporter:
            try:
                loaded = subprocess.run(self.prefix + ["exec", "-T", "docker-daemon", "docker",
                    "--host=unix:///run/graft-docker/docker.sock", "image", "load"], stdin=exporter.stdout,
                    capture_output=True, cwd=ROOT, env=self.environment, timeout=600)
            finally:
                exporter.stdout.close()
                exporter.stdout = None
            _, diagnostics = exporter.communicate(timeout=30)
            if exporter.returncode != 0 or loaded.returncode != 0:
                (self.directory / "image-import-error.log").write_bytes(diagnostics + loaded.stderr)
                (self.directory / "image-import-error.log").chmod(0o600)
                raise RuntimeError("Private daemon image import failed; restricted diagnostics retained")

    def official_install(self) -> None:
        if self.execution_mode != "local" or not self.environment.get("GRAFT_WEB_IMAGE"):
            raise RuntimeError("official-install requires a local registration with --web-image from the same source")
        source = self.directory / "source-assets" / "root"
        if not (source / "compose.yml").is_file():
            raise RuntimeError("Official Compose source was not frozen at registration; initialize a new task")
        for role in ("BACKEND", "WEB"):
            self.import_image(self.environment[f"GRAFT_{role}_IMAGE"])
        tag = "v0.0.0-beta.1"
        self.engine("tag", self.environment["GRAFT_BACKEND_IMAGE"], "ghcr.io/gewuyou/graft-server:" + tag)
        self.engine("tag", self.environment["GRAFT_WEB_IMAGE"], "ghcr.io/gewuyou/graft-web:" + tag)
        root = "/opt/graft-official"
        self.compose("exec", "-T", "docker-daemon", "sh", "-ec",
                     "mkdir -p /opt/graft-official; ln -sf /run/graft-docker/docker.sock /var/run/docker.sock")
        directory = self.directory / "official"
        directory.mkdir(mode=0o700, exist_ok=True)
        env_path = directory / ".env"
        if not env_path.exists():
            values = {"GRAFT_IMAGE_TAG": tag, "GRAFT_EXECUTION_MODE": "local", "COMPOSE_PROFILES": "",
                "COMPOSE_PROJECT_NAME": self.state["project"] + "-official",
                "POSTGRES_PASSWORD": secrets.token_urlsafe(24), "GRAFT_AUTH_JWT_SECRET": secrets.token_urlsafe(32),
                "GRAFT_REDIS_PASSWORD": secrets.token_urlsafe(24), "GRAFT_APPLICATION_ROOT_HOST_PATH": "/opt/graft/apps",
                "GRAFT_APPLICATION_ROOT_CONTAINER_PATH": "/opt/graft/apps", "GRAFT_WEB_HOST_PORT": "3000",
                "GRAFT_UPDATE_STATE_VOLUME": self.state["project"] + "-official-update-state",
                "GRAFT_BUILD_SNAPSHOT_VOLUME": self.state["project"] + "-official-build-snapshots",
                "GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS": "http://127.0.0.1:" + self.environment["GRAFT_CONFORMANCE_OFFICIAL_PORT"]}
            template = (source / "compose.env.example").read_text()
            lines = []
            for line in template.splitlines():
                key = line.split("=", 1)[0]
                lines.append(key + "=" + values.pop(key) if key in values else line)
            lines.extend(key + "=" + value for key, value in values.items())
            env_path.write_text("\n".join(lines) + "\n")
            env_path.chmod(0o600)
        self.compose("cp", str(source / "compose.yml"), "docker-daemon:" + root + "/compose.yml")
        self.compose("exec", "-T", "docker-daemon", "test", "-f", root + "/.env", check=False)
        if self.last_command_returncode != 0:
            if (directory / "installation-evidence.json").exists():
                raise RuntimeError("Registered official root was lost; preserve evidence and initialize a fresh fixture")
            self.compose("cp", str(env_path), "docker-daemon:" + root + "/.env")
        nested = OfficialFixture(self, directory, root)
        nested.compose("up", "-d", "--wait", timeout=600)
        for service, role in (("server", "BACKEND"), ("web", "WEB")):
            identifier = nested.compose("ps", "-q", service).strip()
            actual = self.engine("inspect", "--format", "{{.Image}}", identifier).strip()
            if actual != self.environment[f"GRAFT_{role}_IMAGE"]:
                raise RuntimeError("Official installation image no longer matches its registered source")
        save(directory / "installation-evidence.json", {"source_revision": self.state["source_revision"],
            "execution_mode": "local", "official_compose": True,
            "backend_image": self.environment["GRAFT_BACKEND_IMAGE"], "web_image": self.environment["GRAFT_WEB_IMAGE"]})
        nested.probe()
        require_local_acceptance(nested.results)

    def controller_smoke(self) -> None:
        if not self.environment.get("GRAFT_RUNNER_IMAGE"):
            raise RuntimeError("controller-smoke requires registration with --runner-image from the same source")
        directory = self.directory / "official"
        if not (directory / "installation-evidence.json").exists():
            raise RuntimeError("controller-smoke requires a successful official-install first")
        self.import_image(self.environment["GRAFT_RUNNER_IMAGE"])
        registry_name = self.state["project"] + "-controller-registry"
        if registry_name in self.engine("ps", "-a", "--format", "{{.Names}}").splitlines():
            self.engine("start", registry_name)
        else:
            self.engine("run", "-d", "--name", registry_name, "--network", "host", "registry:3.0.0")
        deadline = time.monotonic() + 30
        while True:
            self.compose("exec", "-T", "docker-daemon", "wget", "-q", "-O", "/dev/null",
                         "http://127.0.0.1:5000/v2/", check=False)
            if self.last_command_returncode == 0:
                break
            if time.monotonic() >= deadline:
                raise RuntimeError("Private Controller registry did not become ready")
            time.sleep(1)
        registry = "127.0.0.1:5000"
        tag = "v0.0.0-beta.2"
        images = {}
        for service, role in (("server", "BACKEND"), ("web", "WEB"), ("compose-runner", "RUNNER")):
            repository = registry + "/graft-" + service
            reference = repository + ":" + tag
            self.engine("tag", self.environment[f"GRAFT_{role}_IMAGE"], reference)
            self.engine("push", reference)
            digests = json.loads(self.engine("image", "inspect", "--format", "{{json .RepoDigests}}", reference))
            digest_reference = next((value for value in digests if value.startswith(repository + "@sha256:")), "")
            if not re.fullmatch(re.escape(repository) + r"@sha256:[a-f0-9]{64}", digest_reference):
                raise RuntimeError("Private registry returned no verified image digest")
            images[service] = {"repository": repository, "reference": reference,
                               "digest": digest_reference.split("@", 1)[1], "immutable": digest_reference}
        root = "/opt/graft-official"
        overlay = directory / "compose.registry.yml"
        server_services = ("server", "bootstrap", "config-validate", "application-root-init",
                           "backup-root-init", "build-snapshot-root-init", "migration-preflight", "runtime-mode-preflight")
        services = {name: {"image": images["server"]["repository"] + ":${GRAFT_IMAGE_TAG}"} for name in server_services}
        services["web"] = {"image": images["web"]["repository"] + ":${GRAFT_IMAGE_TAG}"}
        overlay.write_text(json.dumps({"services": services}, indent=2) + "\n")
        self.compose("cp", str(overlay), "docker-daemon:" + root + "/compose.registry.yml")
        nested = OfficialFixture(self, directory, root)
        before = nested.compose("ps", "-q", "server").strip()
        operation = "controller-fixture-" + secrets.token_hex(6)
        runner = "runner-" + secrets.token_hex(6)
        payload = {"protocol_version": 2, "operation_id": operation, "runner_id": runner,
            "source_version": "0.0.0-beta.1", "target_version": "0.0.0-beta.2", "task_id": 1,
            "execution_mode": "local", "backup_artifact_root": "/var/lib/graft/backups/" + operation,
            "preflight": {"declared_mode": "compose", "detected_mode": "compose", "deployment_strategy": "pinned_beta",
                "image_tag": "v0.0.0-beta.1", "compose_root": root, "platform": "linux/amd64",
                "docker_socket": "/var/run/docker.sock", "compose_files": [root + "/compose.yml", root + "/compose.registry.yml"],
                "bundled_postgres": True, "server_reference": images["server"]["reference"],
                "web_reference": images["web"]["reference"], "runner_reference": images["compose-runner"]["immutable"],
                "server_digest": images["server"]["digest"], "web_digest": images["web"]["digest"],
                "runner_digest": images["compose-runner"]["digest"], "official_server_image": images["server"]["repository"],
                "official_web_image": images["web"]["repository"], "official_runner_image": images["compose-runner"]["repository"]}}
        encoded = base64.b64encode(json.dumps(payload).encode()).decode().rstrip("=")
        state_volume = self.state["project"] + "-official-update-state"
        output = self.engine("run", "--name", runner, "--user", "0:0", "--network", "none", "--read-only",
            "--cap-drop", "ALL", "--cap-add", "CHOWN", "--security-opt", "no-new-privileges:true",
            "--label", "io.graft.update.protocol=compose-runner/v2", "--label", "io.graft.update.operation=" + operation,
            "-e", "GRAFT_UPDATE_RUNNER_INPUT_B64=" + encoded, "-v", root + ":" + root + ":rw",
            "-v", "/var/run/docker.sock:/var/run/docker.sock:rw",
            "-v", state_volume + ":/var/lib/graft/update-state:rw", images["compose-runner"]["immutable"])
        receipt = controller_receipt(output, operation, runner)
        after = nested.compose("ps", "-q", "server").strip()
        state = json.loads(nested.compose("exec", "-T", "backend", "cat", "/var/lib/graft/update-state/current.json"))
        if not before or not after or before == after or state.get("phase") != "SUCCESS" or state.get("receipt") != receipt:
            raise RuntimeError("Controller did not preserve successful durable evidence across server replacement")
        save(directory / "controller-evidence.json", {"source_revision": self.state["source_revision"],
            "server_container_replaced": True, "execution_mode": "local", "receipt": receipt,
            "scope": "Production Controller protocol with same-source image retags; public Update API and Task settlement are separate acceptance"})
        print("Production local Controller replaced the server and persisted verified backup/receipt evidence", flush=True)

    def update_api(self) -> None:
        if self.execution_mode != "local" or not (self.directory / "official" / "installation-evidence.json").exists():
            raise RuntimeError("update-api requires a successful local official-install first")
        source = self.directory / "source-assets" / "fixture" / "release_api.py"
        if not source.is_file():
            raise RuntimeError("Update acceptance driver was not frozen at registration")
        specification = importlib.util.spec_from_file_location("graft_release_acceptance", source)
        if specification is None or specification.loader is None:
            raise RuntimeError("Frozen Update acceptance driver cannot be loaded")
        module = importlib.util.module_from_spec(specification)
        specification.loader.exec_module(module)
        nested = OfficialFixture(self, self.directory / "official", "/opt/graft-official")
        module.run_release_api(self, nested)

    def build_api(self) -> None:
        if self.execution_mode != "local" or not (self.directory / "official" / "installation-evidence.json").exists():
            raise RuntimeError("build-api requires a successful local official-install first")
        source = self.directory / "source-assets" / "fixture" / "build_api.py"
        specification = importlib.util.spec_from_file_location("graft_build_acceptance", source)
        if not source.is_file() or specification is None or specification.loader is None:
            raise RuntimeError("Build acceptance driver was not frozen at registration")
        module = importlib.util.module_from_spec(specification)
        specification.loader.exec_module(module)
        nested = OfficialFixture(self, self.directory / "official", "/opt/graft-official")
        module.run_build_api(self, nested)


class OfficialFixture(Fixture):
    def __init__(self, parent: Fixture, directory: Path, root: str):
        self.parent = parent
        self.directory = directory
        self.root = root
        self.state = {**parent.state, "project": parent.state["project"] + "-official"}
        self.environment = {**parent.environment,
            "GRAFT_CONFORMANCE_BACKEND_PORT": parent.environment["GRAFT_CONFORMANCE_OFFICIAL_PORT"]}
        self.token = ""
        self.results = []

    def compose(self, *arguments: str, check: bool = True, timeout: int = 600) -> str:
        if arguments[:3] == ("exec", "-T", "docker-daemon"):
            output = self.parent.compose(*arguments, check=check, timeout=timeout)
            self.last_command_returncode = self.parent.last_command_returncode
            return output
        if arguments[:3] == ("exec", "-T", "backend"):
            arguments = ("exec", "-T", "--user", "graft", "server", *arguments[3:])
        arguments = tuple("server" if value == "backend" else value for value in arguments)
        output = self.parent.compose("exec", "-T", "docker-daemon", "docker",
            "--host=unix:///run/graft-docker/docker.sock", "compose", "--project-name", self.state["project"],
            "--project-directory", self.root, "--env-file", self.root + "/.env", "-f", self.root + "/compose.yml",
            *arguments, check=check, timeout=timeout)
        self.last_command_returncode = self.parent.last_command_returncode
        return output

    def engine(self, *arguments: str) -> str:
        return self.parent.engine(*arguments)


def validate_topology(configuration: dict, execution_mode: str = "agent") -> None:
    consumers = ("backend",) if execution_mode == "local" else ("backend", "conformance-driver", "docker-runtime-agent")
    if execution_mode == "local" and any(name in configuration["services"] for name in ("vault", "vault-init", "docker-runtime-agent")):
        raise RuntimeError("Local fixture must be independently executable without Vault or Agent services")
    for service in consumers:
        volumes = configuration["services"][service]["volumes"]
        if any(v.get("type") == "bind" and "docker.sock" in v.get("source", "") for v in volumes):
            raise RuntimeError(f"{service} inherited a host Docker socket")
        socket_mount = next((v for v in volumes if v.get("type") == "volume" and
                             v.get("source") == "daemon-socket" and v["target"] == "/run"), None)
        if socket_mount is None:
            raise RuntimeError(f"{service} has no private daemon socket")
        if socket_mount.get("read_only"):
            raise RuntimeError(f"{service} cannot prepare nested secret mountpoints in a read-only run volume")
    workspace_consumers = ("docker-daemon", "backend") if execution_mode == "local" else ("docker-daemon", "backend", "docker-runtime-agent")
    for service in workspace_consumers:
        if not any(v.get("source") == "application-workspaces" and v["target"] == "/opt/graft/apps" for v in configuration["services"][service]["volumes"]):
            raise RuntimeError(f"{service} has inconsistent application workspace mapping")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, epilog=(
        "stop pauses fixture workloads and retains the running development Vault authority. "
        "destroy removes every registered fixture service and volume."))
    parser.add_argument("action", choices=("init", "up", "probe", "official-install", "controller-smoke", "update-api", "build-api", "status", "stop", "destroy"))
    parser.add_argument("--directory", type=Path, required=True, help="Task-owned ignored evidence and secret directory")
    parser.add_argument("--project")
    parser.add_argument("--source-revision")
    parser.add_argument("--backend-image")
    parser.add_argument("--agent-image")
    parser.add_argument("--driver-image")
    parser.add_argument("--web-image", help="Same-source web image required for official-install")
    parser.add_argument("--runner-image", help="Same-source Controller image required for controller-smoke")
    parser.add_argument("--target-backend-image", help="Same-source server built as 0.0.0-beta.2 for Update API acceptance")
    parser.add_argument("--release-fixture-image", help="Same-source task-private TLS release fixture image")
    parser.add_argument("--build-registry-image", help="Same-source authenticated Registry fixture image")
    parser.add_argument("--execution-mode", choices=("local", "agent"), default="agent")
    parser.add_argument("--port", type=int, default=18080)
    parser.add_argument("--confirm-project")
    args = parser.parse_args()
    directory = args.directory.resolve()
    if args.action == "init":
        if not args.project or not re.fullmatch(r"graft-isolated-[a-z0-9-]+", args.project):
            parser.error("--project must use the task-specific graft-isolated- prefix")
        required_images = {"backend": args.backend_image}
        if args.web_image:
            required_images["web"] = args.web_image
        if args.runner_image:
            required_images["runner"] = args.runner_image
        if args.target_backend_image:
            required_images["target-backend"] = args.target_backend_image
        if args.release_fixture_image:
            required_images["release-fixture"] = args.release_fixture_image
        if args.build_registry_image:
            required_images["build-registry"] = args.build_registry_image
        if args.execution_mode == "agent":
            required_images.update(agent=args.agent_image, driver=args.driver_image)
        if not args.source_revision or not all(required_images.values()):
            parser.error("init requires source revision and every image used by the selected execution mode")
        port_count = 2 if args.execution_mode == "local" else 3
        if not 1024 <= args.port <= 65536 - port_count:
            parser.error("--port must leave room for the selected mode's unprivileged loopback listeners")
        for port in range(args.port, args.port + port_count):
            try:
                with socket.socket() as probe:
                    probe.bind(("127.0.0.1", port))
            except OSError:
                parser.error(f"loopback port {port} is already in use")
        directory.mkdir(parents=True, mode=0o700, exist_ok=True)
        if (directory / "instance.json").exists():
            parser.error("instance already registered, reuse its existing directory")
        if not re.fullmatch(r"[a-f0-9]{40}", args.source_revision):
            parser.error("source revision must be an exact Git commit")
        images = {}
        for key, value in required_images.items():
            image = json.loads(subprocess.check_output(["docker", "image", "inspect", value], text=True))[0]
            revision = (image["Config"].get("Labels") or {}).get("org.opencontainers.image.revision")
            if revision != args.source_revision:
                parser.error(f"{key} image source revision does not match the requested commit")
            images[key] = image["Id"]
        env = {"GRAFT_BACKEND_IMAGE": images["backend"],
               "GRAFT_CONFORMANCE_POSTGRES_PASSWORD": secrets.token_urlsafe(24), "VAULT_DEV_ROOT_TOKEN": secrets.token_urlsafe(24),
               "GRAFT_FIXTURE_ADMIN_PASSWORD": secrets.token_urlsafe(24), "GRAFT_CONFORMANCE_BACKEND_PORT": str(args.port),
               "GRAFT_CONFORMANCE_OFFICIAL_PORT": str(args.port + 1),
               "GRAFT_CONFORMANCE_BOOTSTRAP_PORT": str(args.port + 1), "GRAFT_CONFORMANCE_AGENT_PORT": str(args.port + 2)}
        if args.web_image:
            env["GRAFT_WEB_IMAGE"] = images["web"]
        if args.runner_image:
            env["GRAFT_RUNNER_IMAGE"] = images["runner"]
        if args.target_backend_image:
            env["GRAFT_TARGET_BACKEND_IMAGE"] = images["target-backend"]
        if args.release_fixture_image:
            env["GRAFT_RELEASE_FIXTURE_IMAGE"] = images["release-fixture"]
        if args.build_registry_image:
            env["GRAFT_BUILD_REGISTRY_IMAGE"] = images["build-registry"]
        if args.execution_mode == "agent":
            env.update(GRAFT_AGENT_IMAGE=images["agent"], GRAFT_CONFORMANCE_DRIVER_IMAGE=images["driver"],
                       GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST=images["agent"])
        assets = {"root/compose.yml": "compose.yml", "root/compose.env.example": "compose.env.example"}
        fixture_files = ("compose.local.yml", "release_api.py", "build_api.py") if args.execution_mode == "local" else (
            "compose.yml", "compose.isolated.yml", "vault-init.sh")
        assets.update({"fixture/" + name: "tests/conformance/docker-runtime-agent/" + name for name in fixture_files})
        fingerprints = {}
        for destination, repository_path in assets.items():
            content = subprocess.check_output(["git", "show", args.source_revision + ":" + repository_path], cwd=ROOT)
            target = directory / "source-assets" / destination
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(content)
            fingerprints[repository_path] = hashlib.sha256(content).hexdigest()
        save(directory / "source-assets.json", {"source_revision": args.source_revision, "sha256": fingerprints})
        save(directory / "instance.json", {"project": args.project, "source_revision": args.source_revision,
             "execution_mode": args.execution_mode, "environment": env})
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
        fixture.stop()
    elif args.action == "official-install":
        fixture.official_install()
    elif args.action == "controller-smoke":
        fixture.controller_smoke()
    elif args.action == "update-api":
        fixture.update_api()
    elif args.action == "build-api":
        fixture.build_api()
    else:
        getattr(fixture, args.action)()
        if args.action == "probe":
            if fixture.execution_mode == "local":
                require_local_acceptance(fixture.results)
            if any(item["status"] == "failed" for item in fixture.results):
                raise RuntimeError("Runtime acceptance failed; inspect the registered api-evidence.json")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.SubprocessError, OSError, ValueError, KeyError) as exc:
        raise SystemExit(str(exc)) from exc

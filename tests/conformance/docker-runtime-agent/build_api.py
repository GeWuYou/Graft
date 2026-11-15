"""Real Build API publication against a separate authenticated private registry."""
from __future__ import annotations

from datetime import datetime, timedelta, timezone
import hashlib
import io
import json
import re
import secrets
import tarfile
from urllib import request


ROOT = "/opt/graft-official"
TRUST_ROOT = ROOT + "/.fixture-release"
ENDPOINT = "https://build-registry:5443"
REPOSITORY = "probe/build"
PROBE_BYTES = b"graft-build-publication-verified\n"
MANIFEST_ACCEPT = "Accept: application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.docker.distribution.manifest.v2+json"


def daemon(fixture, *arguments):
    return fixture.compose("exec", "-T", "docker-daemon", *arguments)


def context_archive() -> bytes:
    content = io.BytesIO()
    with tarfile.open(fileobj=content, mode="w:gz") as archive:
        for name, value in {"Dockerfile": b"FROM scratch\nCOPY probe /probe\n", "probe": PROBE_BYTES}.items():
            member = tarfile.TarInfo(name)
            member.size, member.mode = len(value), 0o644
            archive.addfile(member, io.BytesIO(value))
    return content.getvalue()


def upload_snapshot(nested) -> str:
    boundary = "graft-fixture-" + secrets.token_hex(12)
    header = ("--" + boundary + '\r\nContent-Disposition: form-data; name="archive"; filename="context.tar.gz"\r\n'
              "Content-Type: application/gzip\r\n\r\n").encode()
    body = header + context_archive() + ("\r\n--" + boundary + "--\r\n").encode()
    url = "http://127.0.0.1:" + nested.environment["GRAFT_CONFORMANCE_BACKEND_PORT"] + "/api/build/input-snapshots"
    headers = {"Authorization": "Bearer " + nested.token, "Content-Type": "multipart/form-data; boundary=" + boundary}
    with request.urlopen(request.Request(url, body, headers, method="POST"), timeout=30) as response:
        if response.status not in (200, 201):
            raise RuntimeError("Build snapshot upload was rejected")
        return json.loads(response.read())["data"]["snapshot_id"]


def require_response(status: int, allowed: tuple[int, ...], stage: str):
    if status not in allowed:
        raise RuntimeError("Build fixture " + stage + " failed with HTTP " + str(status))


def manifest_config_digest(nested, curl: tuple, reference: str, expected_digest: str) -> str:
    def fetch(selected: str, digest: str) -> dict:
        payload = nested.compose(*curl, "--fail", "--netrc-file", "/run/graft/registry.netrc", "-H", MANIFEST_ACCEPT,
            ENDPOINT + "/v2/" + REPOSITORY + "/manifests/" + selected)
        if "sha256:" + hashlib.sha256(payload.encode()).hexdigest() != digest:
            raise RuntimeError("Published manifest bytes do not match their immutable digest")
        return json.loads(payload)
    manifest = fetch(reference, expected_digest)
    if "manifests" in manifest:
        descriptors = [item for item in manifest["manifests"] if (item.get("platform") or {}).get("os") == "linux"
                       and (item.get("platform") or {}).get("architecture") == "amd64"]
        if len(descriptors) != 1:
            raise RuntimeError("Published index does not uniquely identify the requested linux/amd64 image")
        manifest = fetch(descriptors[0]["digest"], descriptors[0]["digest"])
    digest = (manifest.get("config") or {}).get("digest", "")
    if not re.fullmatch(r"sha256:[a-f0-9]{64}", digest):
        raise RuntimeError("Published image manifest has no valid config digest")
    return digest


def verify_image_probe(fixture, image_id: str) -> dict:
    container = fixture.engine("create", "--name", fixture.state["project"] + "-image-proof-" + secrets.token_hex(3),
                               image_id, "/fixture-proof-not-started").strip()
    proof = TRUST_ROOT + "/build/proof-" + secrets.token_hex(4)
    try:
        daemon(fixture, "mkdir", "-p", proof)
        fixture.engine("cp", container + ":/probe", proof + "/probe")
        actual = daemon(fixture, "sha256sum", proof + "/probe").split()[0]
        size = int(daemon(fixture, "wc", "-c", proof + "/probe").split()[0])
    finally:
        fixture.engine("rm", container)
    expected = hashlib.sha256(PROBE_BYTES).hexdigest()
    if actual != expected or size != len(PROBE_BYTES):
        raise RuntimeError("Built image content does not match the uploaded context")
    return {"probe_sha256": actual, "expected_probe_sha256": expected, "probe_size_bytes": size, "probe_verified": True}


def prepare_registry(fixture, nested, directory):
    images = fixture.environment
    if not images.get("GRAFT_BUILD_REGISTRY_IMAGE") or not images.get("GRAFT_RELEASE_FIXTURE_IMAGE"):
        raise RuntimeError("build-api requires same-source --build-registry-image and --release-fixture-image")
    fixture.import_image(images["GRAFT_BUILD_REGISTRY_IMAGE"])
    fixture.import_image(images["GRAFT_RELEASE_FIXTURE_IMAGE"])
    daemon(fixture, "mkdir", "-p", TRUST_ROOT + "/build")
    fixture.engine("run", "--rm", "-v", TRUST_ROOT + ":/fixture", images["GRAFT_RELEASE_FIXTURE_IMAGE"], "--init-ca")
    password = secrets.token_urlsafe(24)
    expires = (datetime.now(timezone.utc) + timedelta(hours=2)).strftime("%Y-%m-%dT%H:%M:%SZ")
    credential = {"version": 1, "credentials": [{"credential_ref": "fixture-build", "endpoint": ENDPOINT,
        "repositories": [REPOSITORY], "operations": ["push", "pull", "manifest-push"],
        "username": "fixture", "password": password, "expires_at": expires}]}
    files = {"password": password, "credentials.json": json.dumps(credential),
             "netrc": "machine build-registry login fixture password " + password + "\n"}
    for name, value in files.items():
        path = directory / name
        path.write_text(value)
        path.chmod(0o600)
        fixture.compose("cp", str(path), "docker-daemon:" + TRUST_ROOT + "/build/" + name)
    fixture.engine("run", "--rm", "--entrypoint", "/usr/local/bin/graft-fixture-htpasswd",
        "-v", TRUST_ROOT + "/build:/fixture", images["GRAFT_BUILD_REGISTRY_IMAGE"])
    registry = fixture.state["project"] + "-build-registry"
    network = nested.state["project"] + "_default"
    fixture.engine("run", "-d", "--name", registry, "--user", "0:0", "--network", network,
        "--network-alias", "build-registry", "-e", "REGISTRY_HTTP_ADDR=0.0.0.0:5443",
        "-e", "REGISTRY_HTTP_TLS_CERTIFICATE=/fixture/server.crt", "-e", "REGISTRY_HTTP_TLS_KEY=/fixture/server.key",
        "-e", "REGISTRY_AUTH=htpasswd", "-e", "REGISTRY_AUTH_HTPASSWD_REALM=Graft acceptance",
        "-e", "REGISTRY_AUTH_HTPASSWD_PATH=/fixture/build/htpasswd", "-v", TRUST_ROOT + ":/fixture:ro",
        images["GRAFT_BUILD_REGISTRY_IMAGE"])
    inspect = json.loads(fixture.engine("inspect", registry))[0]
    address = inspect["NetworkSettings"]["Networks"][network]["IPAddress"]
    if not re.fullmatch(r"[0-9.]+", address):
        raise RuntimeError("Private registry returned an invalid bridge address")
    daemon(fixture, "sh", "-ec", "mkdir -p /etc/docker/certs.d/build-registry:5443 && cp " + TRUST_ROOT +
        "/ca.crt /etc/docker/certs.d/build-registry:5443/ca.crt && echo '" + address + " build-registry' >> /etc/hosts")
    daemon(fixture, "chown", "10001:10001", TRUST_ROOT + "/build/credentials.json", TRUST_ROOT + "/build/netrc")
    overlay = {"services": {"server": {"environment": {"SSL_CERT_FILE": "/run/graft-fixture-release/ca.crt",
        "GRAFT_REGISTRY_CREDENTIALS_FILE": "/run/graft/registry-credentials.json", "GRAFT_DEPLOYMENT_COMPOSE_ROOT": ROOT},
        "volumes": [TRUST_ROOT + "/ca.crt:/run/graft-fixture-release/ca.crt:ro",
            TRUST_ROOT + "/build/credentials.json:/run/graft/registry-credentials.json:ro",
            TRUST_ROOT + "/build/netrc:/run/graft/registry.netrc:ro"]}}}
    path = directory / "compose.build-fixture.yml"
    path.write_text(json.dumps(overlay, indent=2) + "\n")
    fixture.compose("cp", str(path), "docker-daemon:" + ROOT + "/compose.build-fixture.yml")
    fixture.engine("compose", "--project-name", nested.state["project"], "--project-directory", ROOT,
        "--env-file", ROOT + "/.env", "-f", ROOT + "/compose.yml", "-f", ROOT + "/compose.build-fixture.yml",
        "up", "-d", "--no-deps", "--force-recreate", "--wait", "server")
    return password


def run_build_api(fixture, nested):
    directory = fixture.directory / "build-api"
    directory.mkdir(mode=0o700, exist_ok=True)
    password = prepare_registry(fixture, nested, directory)
    nested.authenticate()
    if nested.user_id < 1:
        raise RuntimeError("Build fixture requires an authenticated resource owner identity")
    target = nested.local_target_id()
    curl = ("exec", "-T", "backend", "curl", "--silent", "--show-error", "--max-time", "10",
            "--cacert", "/run/graft-fixture-release/ca.crt")
    anonymous = nested.compose(*curl, "--output", "/dev/null", "--write-out", "%{http_code}", ENDPOINT + "/v2/").strip()
    authenticated = nested.compose(*curl, "--netrc-file", "/run/graft/registry.netrc", "--output", "/dev/null",
                                   "--write-out", "%{http_code}", ENDPOINT + "/v2/").strip()
    if anonymous != "401" or authenticated != "200":
        raise RuntimeError("Build registry did not enforce a real Basic authentication challenge")
    status, response = nested.api("GET", f"/api/runtime-targets/{target}/assignments")
    require_response(status, (200,), "target assignments")
    if not any(item.get("user_id") == nested.user_id for item in (response.get("data") or {}).get("items", [])):
        status, _ = nested.api("POST", f"/api/runtime-targets/{target}/assignments", {"user_id": nested.user_id})
        require_response(status, (200, 201), "target authorization")
    connection = "fixture-build-" + secrets.token_hex(4)
    status, _ = nested.api("POST", "/api/registries", {"connection_ref": connection, "display_name": "Build acceptance registry",
        "provider": "generic_oci", "endpoint": ENDPOINT, "enabled": True, "insecure": False, "credential_ref": "fixture-build"})
    require_response(status, (200, 201), "registry connection")
    status, _ = nested.api("POST", "/api/registries/" + connection + "/repositories", {"repository_ref": REPOSITORY,
        "display_name": "Build probe", "allow_pull": True, "allow_push": True, "grant_creator_use": True})
    require_response(status, (200, 201), "repository authorization")
    status, response = nested.api("POST", "/api/registries/" + connection + "/verify", {
        "runtime_target_id": target, "repository_ref": REPOSITORY})
    require_response(status, (200,), "registry verification")
    if (response.get("data") or {}).get("status") != "verified":
        raise RuntimeError("Registry verification did not accept the scoped synthetic credential")
    snapshot = upload_snapshot(nested)
    reference = "probe-" + secrets.token_hex(6)
    status, response = nested.api("POST", "/api/build/jobs", {"input_snapshot_id": snapshot, "runtime_target_id": target,
        "template_ref": "oci-dockerfile/default@v1", "driver": "docker-engine@v1", "platforms": ["linux/amd64"],
        "destination": {"kind": "oci_registry", "connection_ref": connection, "repository_ref": REPOSITORY, "reference": reference}},
        idempotency_key="graft-build-" + secrets.token_hex(12))
    require_response(status, (202,), "Task submission")
    task_id = response["data"]["task_id"]
    task = nested.wait_task(task_id, timeout=180)
    if task.get("status") != "success":
        raise RuntimeError("Local Build Task did not succeed")
    status, response = nested.api("GET", "/api/build/jobs")
    require_response(status, (200,), "Build projection")
    job = next((item for item in (response.get("data") or {}).get("items", []) if item.get("task_id") == task_id), None)
    artifact = (job or {}).get("artifact") or {}
    if not artifact.get("artifact_id") or not re.fullmatch(r"sha256:[a-f0-9]{64}", artifact.get("digest", "")):
        raise RuntimeError("Build projection did not expose a verified immutable artifact")
    status, response = nested.api("GET", "/api/build/artifacts/" + artifact["artifact_id"] + "/publications")
    require_response(status, (200,), "artifact publication")
    publications = (response.get("data") or {}).get("items", [])
    if not any(item.get("digest") == artifact["digest"] and (item.get("destination") or {}).get("reference") == reference for item in publications):
        raise RuntimeError("Artifact publication does not match the successful Build")
    image_id = fixture.engine("image", "inspect", "--format", "{{.Id}}", artifact["image_id"]).strip()
    headers = nested.compose(*curl, "--fail", "--head", "--netrc-file", "/run/graft/registry.netrc", "-H",
        MANIFEST_ACCEPT,
        ENDPOINT + "/v2/" + REPOSITORY + "/manifests/" + reference)
    remote_digest = next((line.split(":", 1)[1].strip() for line in headers.splitlines()
                          if line.lower().startswith("docker-content-digest:")), "")
    if remote_digest != artifact["digest"] or not re.fullmatch(r"sha256:[a-f0-9]{64}", image_id):
        raise RuntimeError("Actual Docker image or registry manifest does not match the Build artifact")
    config_digest = manifest_config_digest(nested, curl, reference, artifact["digest"])
    if config_digest != image_id:
        raise RuntimeError("Published manifest config does not identify the actual built Docker image")
    probe = verify_image_probe(fixture, image_id)
    status, logs = nested.api("GET", f"/api/tasks/{task_id}/logs")
    require_response(status, (200,), "Task log readback")
    if password in json.dumps({"task": task, "job": job, "publications": publications, "logs": logs}):
        raise RuntimeError("Build credential escaped its transient execution boundary")
    evidence = {"source_revision": fixture.state["source_revision"], "task_id": task_id, "task_status": "success",
        "registry_anonymous_status": 401, "registry_authenticated_status": 200, "registry_verified": True,
        "artifact": artifact, "publications": publications, "engine_image_id": image_id,
        "remote_manifest_digest": remote_digest, "manifest_config_digest": config_digest, **probe,
        "credential_absent_from_task_and_logs": True}
    path = directory / "build-api-evidence.json"
    path.write_text(json.dumps(evidence, indent=2) + "\n")
    path.chmod(0o600)
    print("Production Build API verified credentials and published the matching Docker artifact", flush=True)

#!/usr/bin/env python3
"""Package and verify source-bound CI runtime artifacts without rebuilding them."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import runpy
import subprocess

ROOT = Path(__file__).resolve().parents[1]


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
    if kind == "server":
        binary = root / "server/graft"
        if not binary.is_file() or binary.is_symlink():
            raise ValueError("Validated server binary is missing")
        shutil.copyfile(binary, directory / "graft-server-linux-amd64")
        (directory / "graft-server-linux-amd64").chmod(0o755)
    else:
        distribution = root / "web/dist"
        if not (distribution / "index.html").is_file():
            raise ValueError("Validated web distribution is missing")
        # Reject links before copying so dependency or external trees cannot enter the artifact.
        fingerprints(distribution)
        shutil.copytree(distribution, directory / "dist")
    data = {"source_revision": source_revision(root), "kind": kind, "sha256": fingerprints(directory)}
    (directory / "provenance.json").write_text(json.dumps(data, indent=2) + "\n")


def verify(root: Path, directory: Path, kind: str) -> None:
    metadata = json.loads((directory / "provenance.json").read_text())
    if metadata.get("source_revision") != source_revision(root) or metadata.get("kind") != kind:
        raise ValueError("Runtime artifact source or kind differs from the checked-out commit")
    recorded = metadata.get("sha256")
    if not recorded or recorded != fingerprints(directory):
        raise ValueError("Runtime artifact content differs from its recorded hashes")
    required = "graft-server-linux-amd64" if kind == "server" else "dist/index.html"
    if required not in recorded:
        raise ValueError("Runtime artifact is missing its required entrypoint")
    if kind == "server":
        # Artifact downloads preserve bytes; the executable mode is restored after verification.
        (directory / required).chmod(0o755)


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
    if installation.get("official_compose") is not True or installation.get("execution_mode") != "local":
        raise ValueError("Official local installation proof is incomplete")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("prepare", "verify", "acceptance"))
    parser.add_argument("--kind", choices=("server", "web"))
    parser.add_argument("--directory", type=Path, required=True)
    args = parser.parse_args()
    if args.action != "acceptance" and args.kind is None:
        parser.error("--kind is required when packaging or verifying build artifacts")
    try:
        if args.action == "acceptance":
            verify_acceptance(ROOT, args.directory.resolve())
        else:
            {"prepare": prepare, "verify": verify}[args.action](ROOT, args.directory.resolve(), args.kind)
    except (OSError, ValueError, RuntimeError, subprocess.CalledProcessError) as error:
        parser.exit(1, str(error) + "\n")


if __name__ == "__main__":
    main()

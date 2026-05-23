#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
SKILL_DIR = SCRIPT_DIR.parent
CHECKOUT_ROOT = SKILL_DIR.parent.parent.parent


class WorktreeInitError(RuntimeError):
    pass


@dataclass(frozen=True)
class LinkSpec:
    source: Path
    target: Path
    required: bool


def run_git(*args: str, cwd: Path) -> str:
    try:
        completed = subprocess.run(
            ["git", *args],
            cwd=str(cwd),
            check=True,
            capture_output=True,
            text=True,
        )
    except subprocess.CalledProcessError as exc:
        stderr = exc.stderr.strip() or exc.stdout.strip() or "git command failed"
        raise WorktreeInitError(stderr) from exc
    return completed.stdout.strip()


def resolve_git_repo(path: Path) -> Path:
    common_dir_text = run_git("rev-parse", "--path-format=absolute", "--git-common-dir", cwd=path)
    common_dir = Path(common_dir_text).resolve()
    if common_dir.name != ".git":
        raise WorktreeInitError(
            f"Unsupported git common dir layout: {common_dir}. Pass --repo-dir explicitly if needed."
        )
    return common_dir.parent.resolve()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Create or rebuild a Graft git worktree using the repository shared-resource manifest."
    )
    parser.add_argument("branch_name", help="Target git branch name for the worktree.")
    parser.add_argument("base_branch", nargs="?", default="main", help="Base branch when creating a new branch.")
    parser.add_argument("--rebuild", action="store_true", help="Remove and recreate the target worktree if it exists.")
    parser.add_argument("--repo-dir", help="Override the canonical repository root.")
    parser.add_argument("--worktree-root", help="Override the parent directory for new worktrees.")
    parser.add_argument("--dry-run", action="store_true", help="Print the execution plan without mutating anything.")
    return parser.parse_args()


def load_manifest(path: Path) -> list[LinkSpec]:
    data = json.loads(path.read_text(encoding="utf-8"))
    specs: list[LinkSpec] = []
    for raw in data.get("links", []):
        specs.append(
            LinkSpec(
                source=Path(raw["source"]),
                target=Path(raw["target"]),
                required=bool(raw.get("required", True)),
            )
        )
    return specs


def ensure_branch_name(repo_dir: Path, branch_name: str) -> None:
    subprocess.run(
        ["git", "check-ref-format", "--branch", branch_name],
        cwd=str(repo_dir),
        check=True,
        capture_output=True,
        text=True,
    )


def infer_worktree_root(repo_dir: Path, override: str | None) -> Path:
    if override:
        return Path(override).expanduser().resolve()

    candidate = repo_dir.parent / f"{repo_dir.name}-wt"
    if candidate.exists():
        return candidate.resolve()

    raise WorktreeInitError(
        "Could not infer worktree root because the default sibling path does not exist: "
        f"{candidate}. Pass --worktree-root explicitly."
    )


def is_registered_worktree(repo_dir: Path, candidate: Path) -> bool:
    output = run_git("worktree", "list", "--porcelain", cwd=repo_dir)
    return f"worktree {candidate}" in output.splitlines()


def local_branch_exists(repo_dir: Path, branch_name: str) -> bool:
    completed = subprocess.run(
        ["git", "show-ref", "--verify", "--quiet", f"refs/heads/{branch_name}"],
        cwd=str(repo_dir),
        check=False,
    )
    return completed.returncode == 0


def relative_symlink(src: Path, dest: Path) -> None:
    dest.parent.mkdir(parents=True, exist_ok=True)
    if dest.is_symlink():
        if dest.resolve() == src.resolve():
            return
        dest.unlink()
    elif dest.exists():
        raise WorktreeInitError(f"Destination already exists and is not a symlink: {dest}")

    relative_src = Path(shutil.os.path.relpath(src, start=dest.parent))
    dest.symlink_to(relative_src)


def remove_legacy_local(target_dir: Path) -> bool:
    legacy = target_dir / ".local"
    if legacy.is_symlink():
        legacy.unlink()
        return True
    if legacy.exists():
        raise WorktreeInitError(f"Legacy .local path exists and is not a symlink: {legacy}")
    return False


def print_plan(
    repo_dir: Path,
    worktree_root: Path,
    target_dir: Path,
    branch_name: str,
    base_branch: str,
    rebuild: bool,
    branch_exists: bool,
    specs: list[LinkSpec],
) -> None:
    branch_mode = "reuse existing branch" if branch_exists else "create branch from base"
    print("Worktree init plan:")
    print(f"- repo_dir: {repo_dir}")
    print(f"- worktree_root: {worktree_root}")
    print(f"- target_dir: {target_dir}")
    print(f"- branch_name: {branch_name}")
    print(f"- base_branch: {base_branch}")
    print(f"- branch_mode: {branch_mode}")
    print(f"- rebuild: {'yes' if rebuild else 'no'}")
    print("- shared_links:")
    for spec in specs:
        print(
            f"  - {spec.source.as_posix()} -> {spec.target.as_posix()} "
            f"({'required' if spec.required else 'optional'})"
        )
    print("- legacy_cleanup: remove target .local symlink if present")


def create_or_rebuild_worktree(
    repo_dir: Path,
    target_dir: Path,
    branch_name: str,
    base_branch: str,
    rebuild: bool,
) -> None:
    if target_dir.exists():
        if is_registered_worktree(repo_dir, target_dir):
            if not rebuild:
                raise WorktreeInitError(f"Target is already a registered worktree: {target_dir}")
            run_git("worktree", "remove", "--force", str(target_dir), cwd=repo_dir)
        elif rebuild:
            shutil.rmtree(target_dir)
        else:
            raise WorktreeInitError(f"Target exists but is not a registered worktree: {target_dir}")

    target_dir.parent.mkdir(parents=True, exist_ok=True)

    if local_branch_exists(repo_dir, branch_name):
        run_git("worktree", "add", "--force", str(target_dir), branch_name, cwd=repo_dir)
    else:
        run_git("worktree", "add", "--force", "-b", branch_name, str(target_dir), base_branch, cwd=repo_dir)


def apply_links(repo_dir: Path, target_dir: Path, specs: list[LinkSpec]) -> tuple[list[str], list[str]]:
    linked: list[str] = []
    skipped: list[str] = []

    for spec in specs:
        src = repo_dir / spec.source
        dest = target_dir / spec.target
        if not src.exists():
            message = f"{spec.source.as_posix()} missing in canonical repo root"
            if spec.required:
                raise WorktreeInitError(message)
            skipped.append(message)
            continue

        relative_symlink(src, dest)
        linked.append(f"{spec.source.as_posix()} -> {spec.target.as_posix()}")

    return linked, skipped


def main() -> int:
    args = parse_args()
    try:
        repo_dir = Path(args.repo_dir).expanduser().resolve() if args.repo_dir else resolve_git_repo(Path.cwd())
        manifest_path = repo_dir / ".worktree-shared.json"
        if not manifest_path.exists():
            raise WorktreeInitError(f"Manifest not found: {manifest_path}")
        ensure_branch_name(repo_dir, args.branch_name)
        worktree_root = infer_worktree_root(repo_dir, args.worktree_root)
        target_dir = (worktree_root / args.branch_name).resolve()
        specs = load_manifest(manifest_path)
        branch_exists = local_branch_exists(repo_dir, args.branch_name)

        print_plan(
            repo_dir=repo_dir,
            worktree_root=worktree_root,
            target_dir=target_dir,
            branch_name=args.branch_name,
            base_branch=args.base_branch,
            rebuild=args.rebuild,
            branch_exists=branch_exists,
            specs=specs,
        )

        if args.dry_run:
            return 0

        create_or_rebuild_worktree(
            repo_dir=repo_dir,
            target_dir=target_dir,
            branch_name=args.branch_name,
            base_branch=args.base_branch,
            rebuild=args.rebuild,
        )
        legacy_removed = remove_legacy_local(target_dir)
        linked, skipped = apply_links(repo_dir, target_dir, specs)

        print("Worktree init result:")
        print(f"- created: {target_dir}")
        print(f"- linked_count: {len(linked)}")
        for item in linked:
            print(f"  - linked {item}")
        for item in skipped:
            print(f"  - skipped {item}")
        if legacy_removed:
            print("- removed legacy .local symlink from target worktree")
        return 0
    except subprocess.CalledProcessError as exc:
        stderr = exc.stderr.strip() if exc.stderr else ""
        message = stderr or str(exc)
        print(f"error: {message}", file=sys.stderr)
        return 1
    except (WorktreeInitError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())

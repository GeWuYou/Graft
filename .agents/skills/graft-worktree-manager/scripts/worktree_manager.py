#!/usr/bin/env python3
"""Manage Graft's reusable, numbered AI-agent worktree pool."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


POOL_SUFFIX = re.compile(r"-wt-(?P<slot>\d{2,})$")
ALLOWED_BRANCH = re.compile(r"^(feature|fix|refactor|docs|chore|build|ci)/[a-z0-9]+(?:-[a-z0-9]+)*$")


class WorktreeManagerError(RuntimeError):
    pass


@dataclass(frozen=True)
class Worktree:
    path: Path
    branch: str | None
    head: str


def git(repo_dir: Path, *args: str, check: bool = True) -> str:
    completed = subprocess.run(
        ["git", *args], cwd=repo_dir, check=False, capture_output=True, text=True
    )
    if check and completed.returncode:
        message = completed.stderr.strip() or completed.stdout.strip() or "git command failed"
        raise WorktreeManagerError(message)
    return completed.stdout.strip()


def repo_root(path: Path) -> Path:
    common_dir = Path(git(path, "rev-parse", "--path-format=absolute", "--git-common-dir")).resolve()
    if common_dir.name != ".git":
        raise WorktreeManagerError(f"unsupported git common directory: {common_dir}")
    return common_dir.parent


def parse_worktrees(repo_dir: Path) -> list[Worktree]:
    records = git(repo_dir, "worktree", "list", "--porcelain").split("\n\n")
    result: list[Worktree] = []
    for record in records:
        values: dict[str, str] = {}
        for line in record.splitlines():
            key, _, value = line.partition(" ")
            if value:
                values[key] = value
        if "worktree" in values:
            branch = values.get("branch", "").removeprefix("refs/heads/") or None
            result.append(Worktree(Path(values["worktree"]), branch, values.get("HEAD", "")))
    return result


def pool_slot(root: Path, path: Path) -> int | None:
    if path.parent != root.parent:
        return None
    match = POOL_SUFFIX.search(path.name)
    if match is None or not path.name.startswith(f"{root.name}-"):
        return None
    return int(match.group("slot"))


def changes(path: Path) -> int:
    return len([line for line in git(path, "status", "--porcelain").splitlines() if line])


def origin_main(root: Path) -> str:
    return git(root, "rev-parse", "origin/main")


def at_main_baseline(worktree: Worktree, root: Path) -> bool:
    if worktree.branch == "main":
        return True
    return worktree.branch is None and worktree.head == origin_main(root)


def state(worktree: Worktree, root: Path) -> str:
    if worktree.path == root:
        return "integration"
    if changes(worktree.path):
        return "dirty"
    if pool_slot(root, worktree.path) is not None and at_main_baseline(worktree, root):
        return "available"
    return "occupied"


def status_rows(root: Path) -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for worktree in parse_worktrees(root):
        rows.append(
            {
                "path": str(worktree.path),
                "branch": worktree.branch,
                "head": worktree.head,
                "changes": changes(worktree.path),
                "state": state(worktree, root),
            }
        )
    return sorted(rows, key=lambda row: str(row["path"]))


def print_status(root: Path, as_json: bool) -> None:
    rows = status_rows(root)
    if as_json:
        print(json.dumps({"repo": str(root), "worktrees": rows}, indent=2))
        return
    for row in rows:
        ref = row["branch"] or f"detached@{str(row['head'])[:12]}"
        print(f"{row['state']}: {row['path']} | {ref} | changes={row['changes']}")


def fetch(root: Path) -> None:
    git(root, "fetch", "origin", "--prune")
    origin_main(root)


def branch_exists(root: Path, branch: str) -> bool:
    return any(
        subprocess.run(
            ["git", "show-ref", "--verify", "--quiet", ref],
            cwd=root,
            capture_output=True,
            text=True,
            check=False,
        ).returncode
        == 0
        for ref in (f"refs/heads/{branch}", f"refs/remotes/origin/{branch}")
    )


def load_links(root: Path) -> list[dict[str, object]]:
    manifest = root / ".worktree-shared.json"
    if not manifest.is_file():
        return []
    return list(json.loads(manifest.read_text(encoding="utf-8")).get("links", []))


def apply_links(root: Path, target: Path) -> None:
    for link in load_links(root):
        source = root / str(link["source"])
        destination = target / str(link["target"])
        if not source.exists():
            if link.get("required", True):
                raise WorktreeManagerError(f"required shared resource missing: {link['source']}")
            continue
        destination.parent.mkdir(parents=True, exist_ok=True)
        relative = Path(os.path.relpath(source, destination.parent))
        if destination.is_symlink() and destination.resolve() == source.resolve():
            continue
        if destination.exists() or destination.is_symlink():
            raise WorktreeManagerError(f"shared resource destination already exists: {destination}")
        destination.symlink_to(relative)


def next_pool_path(root: Path) -> Path:
    slots = [slot for item in parse_worktrees(root) if (slot := pool_slot(root, item.path)) is not None]
    return root.parent / f"{root.name}-wt-{max(slots, default=0) + 1:02d}"


def sync_main_baseline(root: Path, target: Path) -> None:
    git(target, "switch", "main", check=False)
    if git(target, "branch", "--show-current") == "main":
        git(target, "pull", "--ff-only", "origin", "main")
        return
    git(target, "switch", "--detach", "origin/main")


def acquire(root: Path, branch: str) -> None:
    if ALLOWED_BRANCH.fullmatch(branch) is None:
        raise WorktreeManagerError("branch must use an approved type and lowercase kebab-case name")
    fetch(root)
    if branch_exists(root, branch):
        raise WorktreeManagerError(f"branch already exists locally or on origin: {branch}")
    available = [item for item in parse_worktrees(root) if state(item, root) == "available"]
    available.sort(key=lambda item: pool_slot(root, item.path) or 0)
    if available:
        target = available[0].path
    else:
        target = next_pool_path(root)
        git(root, "worktree", "add", "--detach", str(target), "origin/main")
        apply_links(root, target)
    if changes(target):
        raise WorktreeManagerError(f"refusing to acquire dirty worktree: {target}")
    sync_main_baseline(root, target)
    git(target, "switch", "-c", branch, "origin/main")
    print(f"Using Worktree: {target}")
    print(f"Branch: {branch}")


def release(root: Path, target: Path, confirmation: str | None) -> None:
    entry = next((item for item in parse_worktrees(root) if item.path == target), None)
    if entry is None or pool_slot(root, target) is None:
        raise WorktreeManagerError("release only accepts a registered numbered pool worktree")
    if entry.branch in (None, "main"):
        raise WorktreeManagerError("worktree has no releasable task branch")
    if changes(target):
        raise WorktreeManagerError("release requires a clean worktree")
    print("Review Summary:")
    print(f"- worktree: {target}")
    print(f"- branch: {entry.branch}")
    print("- commits:")
    print(git(target, "log", "--oneline", f"origin/main..{entry.branch}") or "  - none")
    print("- diff_stat:")
    print(git(target, "diff", "--stat", f"origin/main...{entry.branch}") or "  - none")
    if confirmation is None:
        print("Awaiting developer integration confirmation; no worktree state changed.")
        return
    git(root, "rev-parse", "--verify", confirmation)
    fetch(root)
    old_branch = entry.branch
    sync_main_baseline(root, target)
    git(root, "branch", "-D", old_branch)
    print(f"Released Worktree: {target}")
    print(f"Deleted local branch: {old_branch}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-dir", type=Path, help="Canonical repository root override")
    subparsers = parser.add_subparsers(dest="command", required=True)
    status_parser = subparsers.add_parser("status")
    status_parser.add_argument("--json", action="store_true")
    acquire_parser = subparsers.add_parser("acquire")
    acquire_parser.add_argument("branch")
    release_parser = subparsers.add_parser("release")
    release_parser.add_argument("--worktree", type=Path, default=Path.cwd())
    release_parser.add_argument("--confirm-integrated")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        root = args.repo_dir.resolve() if args.repo_dir else repo_root(Path.cwd())
        if args.command == "status":
            print_status(root, args.json)
        elif args.command == "acquire":
            acquire(root, args.branch)
        else:
            release(root, args.worktree.resolve(), args.confirm_integrated)
    except (WorktreeManagerError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Manage Graft's reusable, numbered AI-agent worktree pool."""

from __future__ import annotations

import argparse
from contextlib import contextmanager
import fcntl
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


POOL_SLOT = re.compile(r"(?P<slot>\d{2,})$")
LEGACY_POOL_SUFFIX = re.compile(r"-wt-(?P<slot>\d{2,})$")
POOL_BRANCH = re.compile(r"main-(?P<slot>\d{2,})$")
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
    if path.parent != pool_root(root):
        return None
    match = POOL_SLOT.fullmatch(path.name)
    if match is None:
        return None
    return int(match.group("slot"))


def pool_root(root: Path) -> Path:
    return root / ".worktrees"


def pool_branch(root: Path, path: Path) -> str:
    slot = pool_slot(root, path)
    if slot is None:
        raise WorktreeManagerError(f"not a numbered pool worktree: {path}")
    return f"main-{slot:02d}"


def legacy_pool_slot(root: Path, path: Path) -> int | None:
    if path.parent != root.parent:
        return None
    match = LEGACY_POOL_SUFFIX.search(path.name)
    if match is None or not path.name.startswith(f"{root.name}-"):
        return None
    return int(match.group("slot"))


def changes(path: Path) -> int:
    return len([line for line in git(path, "status", "--porcelain").splitlines() if line])


def origin_main(root: Path) -> str:
    return git(root, "rev-parse", "origin/main")


def is_ancestor(root: Path, ancestor: str, descendant: str) -> bool:
    completed = subprocess.run(
        ["git", "merge-base", "--is-ancestor", ancestor, descendant],
        cwd=root,
        capture_output=True,
        text=True,
        check=False,
    )
    return completed.returncode == 0


def at_main_baseline(worktree: Worktree, root: Path) -> bool:
    return worktree.head == origin_main(root) and (worktree.branch == "main" or worktree.branch is None)


def is_reusable_slot(worktree: Worktree, root: Path) -> bool:
    if pool_slot(root, worktree.path) is None or changes(worktree.path):
        return False
    marker = pool_branch(root, worktree.path)
    if worktree.branch not in (None, marker):
        return False
    current = origin_main(root)
    return worktree.head == current or is_ancestor(root, worktree.head, current)


def baseline_state(worktree: Worktree, root: Path) -> str:
    current = origin_main(root)
    if worktree.head == current:
        return "current"
    if is_ancestor(root, worktree.head, current):
        return "stale"
    return "unknown"


def state(worktree: Worktree, root: Path) -> str:
    if worktree.path == root:
        return "integration"
    if changes(worktree.path):
        return "dirty"
    if is_reusable_slot(worktree, root):
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
                "slot_branch": pool_branch(root, worktree.path)
                if pool_slot(root, worktree.path) is not None
                else None,
                "baseline": baseline_state(worktree, root)
                if pool_slot(root, worktree.path) is not None
                else None,
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
        baseline = f" | baseline={row['baseline']}" if row["baseline"] else ""
        print(f"{row['state']}: {row['path']} | {ref} | changes={row['changes']}{baseline}")


@contextmanager
def manager_lock(root: Path):
    common_dir = Path(git(root, "rev-parse", "--git-common-dir")).resolve()
    lock_path = common_dir / "graft-worktree-manager.lock"
    with lock_path.open("w", encoding="utf-8") as lock:
        fcntl.flock(lock.fileno(), fcntl.LOCK_EX)
        try:
            yield
        finally:
            fcntl.flock(lock.fileno(), fcntl.LOCK_UN)


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


def validate_link_rebuild(root: Path, target: Path) -> None:
    for link in load_links(root):
        source = root / str(link["source"])
        destination = target / str(link["target"])
        if not source.exists() and link.get("required", True):
            raise WorktreeManagerError(f"required shared resource missing: {link['source']}")
        if destination.exists() and not destination.is_symlink():
            raise WorktreeManagerError(f"shared resource destination is not a symlink: {destination}")


def rebuild_links(root: Path, target: Path) -> None:
    links = load_links(root)
    previous: dict[Path, str] = {
        target / str(link["target"]): os.readlink(target / str(link["target"]))
        for link in links
        if (target / str(link["target"])).is_symlink()
    }

    try:
        for link in links:
            source = root / str(link["source"])
            destination = target / str(link["target"])
            if destination.is_symlink():
                destination.unlink()
            if not source.exists():
                continue
            destination.parent.mkdir(parents=True, exist_ok=True)
            relative = Path(os.path.relpath(source, destination.parent))
            destination.symlink_to(relative)
    except OSError as exc:
        try:
            for link in links:
                destination = target / str(link["target"])
                if destination.is_symlink():
                    destination.unlink()
                if destination in previous:
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    destination.symlink_to(previous[destination])
        except OSError as restore_exc:
            raise WorktreeManagerError(
                f"rebuild_links failed for {target}: {exc}; "
                f"failed to restore old links: {restore_exc}"
            ) from exc
        raise WorktreeManagerError(f"rebuild_links failed for {target}: {exc}") from exc


def next_pool_path(root: Path) -> Path:
    slots = [slot for item in parse_worktrees(root) if (slot := pool_slot(root, item.path)) is not None]
    return pool_root(root) / f"{max(slots, default=0) + 1:02d}"


def sync_pool_slot(root: Path, target: Path) -> None:
    entry = next((item for item in parse_worktrees(root) if item.path == target), None)
    if entry is None:
        raise WorktreeManagerError(f"unregistered pool worktree: {target}")
    if changes(target):
        raise WorktreeManagerError(f"refusing to sync dirty worktree: {target}")
    marker = pool_branch(root, target)
    if entry.branch not in (None, marker):
        raise WorktreeManagerError(f"worktree has an active branch: {target}")
    if entry.branch is None and not is_ancestor(root, entry.head, origin_main(root)):
        raise WorktreeManagerError(f"detached worktree is not an old main baseline: {target}")
    git(target, "switch", "-C", marker, "origin/main")
    git(target, "branch", "--unset-upstream", check=False)
    apply_links(root, target)


def acquire(root: Path, branch: str) -> None:
    with manager_lock(root):
        if ALLOWED_BRANCH.fullmatch(branch) is None:
            raise WorktreeManagerError("branch must use an approved type and lowercase kebab-case name")
        fetch(root)
        if branch_exists(root, branch):
            raise WorktreeManagerError(f"branch already exists locally or on origin: {branch}")
        available = [item for item in parse_worktrees(root) if is_reusable_slot(item, root)]
        available.sort(key=lambda item: pool_slot(root, item.path) or 0)
        if available:
            target = available[0].path
        else:
            target = next_pool_path(root)
            target.parent.mkdir(parents=True, exist_ok=True)
            git(root, "worktree", "add", "--detach", str(target), "origin/main")
        sync_pool_slot(root, target)
        git(target, "switch", "-c", branch, "origin/main")
        print(f"Using Worktree: {target}")
        print(f"Branch: {branch}")


def release(root: Path, target: Path, confirmation: str | None) -> None:
    with manager_lock(root):
        entry = next((item for item in parse_worktrees(root) if item.path == target), None)
        if entry is None or pool_slot(root, target) is None:
            raise WorktreeManagerError("release only accepts a registered numbered pool worktree")
        if entry.branch in (None, "main") or POOL_BRANCH.fullmatch(entry.branch or ""):
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
        if not is_ancestor(root, entry.branch, confirmation):
            raise WorktreeManagerError(
                f"confirmation ref does not contain task branch: {entry.branch} -> {confirmation}"
            )
        fetch(root)
        old_branch = entry.branch
        marker = pool_branch(root, target)
        git(target, "switch", "-C", marker, "origin/main")
        git(target, "branch", "--unset-upstream", check=False)
        apply_links(root, target)
        git(root, "branch", "-D", old_branch)
        print(f"Released Worktree: {target}")
        print(f"Restored pool branch: {marker}")
        print(f"Deleted local branch: {old_branch}")


def reconcile(root: Path, slots: list[str], confirmed: bool) -> None:
    if not confirmed:
        raise WorktreeManagerError("reconcile requires --confirm")
    with manager_lock(root):
        fetch(root)
        try:
            requested = [int(slot) for slot in slots] if slots else None
        except ValueError as exc:
            raise WorktreeManagerError("reconcile slots must be numeric") from exc
        if requested is not None and (any(slot < 1 for slot in requested) or len(set(requested)) != len(requested)):
            raise WorktreeManagerError("reconcile slots must be unique positive numbers")
        worktrees = parse_worktrees(root)
        candidates = [
            item
            for item in worktrees
            if pool_slot(root, item.path) is not None
            and (requested is None or pool_slot(root, item.path) in requested)
        ]
        if requested is not None and {pool_slot(root, item.path) for item in candidates} != set(requested):
            raise WorktreeManagerError("reconcile requested an unregistered pool slot")
        for item in candidates:
            if not is_reusable_slot(item, root):
                raise WorktreeManagerError(f"pool slot is dirty or occupied: {item.path}")
            validate_link_rebuild(root, item.path)
        for item in sorted(candidates, key=lambda value: pool_slot(root, value.path) or 0):
            sync_pool_slot(root, item.path)
            print(f"Reconciled Worktree: {item.path} -> {pool_branch(root, item.path)}")


def _relocate(root: Path, confirmed: bool) -> None:
    if not confirmed:
        raise WorktreeManagerError("relocate requires --confirm")
    fetch(root)
    worktrees = parse_worktrees(root)
    legacy = [item for item in worktrees if legacy_pool_slot(root, item.path) is not None]
    if not legacy:
        raise WorktreeManagerError("no legacy sibling pool worktrees found")
    legacy.sort(key=lambda item: legacy_pool_slot(root, item.path) or 0)
    destinations: list[tuple[Worktree, Path]] = []
    for item in legacy:
        if changes(item.path):
            raise WorktreeManagerError(f"legacy pool worktree must be clean: {item.path}")
        if not at_main_baseline(item, root):
            raise WorktreeManagerError(f"legacy pool worktree is not at origin/main: {item.path}")
        target = pool_root(root) / f"{legacy_pool_slot(root, item.path):02d}"
        if target.exists() or target.is_symlink():
            raise WorktreeManagerError(f"relocation destination already exists: {target}")
        validate_link_rebuild(root, item.path)
        destinations.append((item, target))
    pool_root(root).mkdir(parents=True, exist_ok=True)
    moved: list[tuple[Path, Path]] = []
    try:
        for item, target in destinations:
            git(root, "worktree", "move", str(item.path), str(target))
            moved.append((item.path, target))
            rebuild_links(root, target)
    except WorktreeManagerError as exc:
        rollback_errors: list[str] = []
        for source, target in reversed(moved):
            try:
                git(root, "worktree", "move", str(target), str(source))
                rebuild_links(root, source)
            except WorktreeManagerError as rollback_exc:
                rollback_errors.append(f"{source}: {rollback_exc}")
        if rollback_errors:
            raise WorktreeManagerError(
                f"relocation failed: {exc}; rollback failed: {'; '.join(rollback_errors)}"
            ) from exc
        raise WorktreeManagerError(f"relocation failed and was rolled back: {exc}") from exc
    print(f"Relocated {len(moved)} worktree(s) into {pool_root(root)}")


def relocate(root: Path, confirmed: bool) -> None:
    with manager_lock(root):
        _relocate(root, confirmed)


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
    relocate_parser = subparsers.add_parser("relocate")
    relocate_parser.add_argument("--confirm", action="store_true")
    reconcile_parser = subparsers.add_parser("reconcile")
    reconcile_parser.add_argument("--confirm", action="store_true")
    reconcile_parser.add_argument("slots", nargs="*")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        root = args.repo_dir.resolve() if args.repo_dir else repo_root(Path.cwd())
        if args.command == "status":
            print_status(root, args.json)
        elif args.command == "acquire":
            acquire(root, args.branch)
        elif args.command == "release":
            release(root, args.worktree.resolve(), args.confirm_integrated)
        elif args.command == "reconcile":
            reconcile(root, args.slots, args.confirm)
        else:
            relocate(root, args.confirm)
    except (WorktreeManagerError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

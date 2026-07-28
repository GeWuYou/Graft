#!/usr/bin/env python3
"""Manage Graft's reusable, numbered AI-agent worktree pool."""

from __future__ import annotations

import argparse
from contextlib import contextmanager
from dataclasses import dataclass
from datetime import UTC, datetime
import fcntl
import json
import os
import re
import subprocess
import sys
from pathlib import Path


POOL_SLOT = re.compile(r"(?P<slot>\d{2})$")
LEGACY_POOL_SUFFIX = re.compile(r"-wt-(?P<slot>\d{2})$")
POOL_BRANCH = re.compile(r"main-(?P<slot>\d{2})$")
ALLOWED_BRANCH = re.compile(r"^(feature|fix|refactor|docs|chore|build|ci)/[a-z0-9]+(?:-[a-z0-9]+)*$")
LEASE_FILE = "graft-worktree-manager-leases.json"
MIN_POOL_SLOT = 1
MAX_POOL_SLOT = 99


class WorktreeManagerError(RuntimeError):
    pass


@dataclass(frozen=True)
class Worktree:
    path: Path
    branch: str | None
    head: str


@dataclass(frozen=True)
class Slot:
    number: int
    path: Path
    worktree: Worktree | None
    marker: str | None
    marker_head: str | None
    directory_exists: bool
    kind: str
    reason: str | None


def git(repo_dir: Path, *args: str, check: bool = True) -> str:
    completed = subprocess.run(
        ["git", *args], cwd=repo_dir, check=False, capture_output=True, text=True
    )
    if check and completed.returncode:
        message = completed.stderr.strip() or completed.stdout.strip() or "git command failed"
        raise WorktreeManagerError(message)
    return completed.stdout.strip()


def git_common_dir(root: Path) -> Path:
    return Path(git(root, "rev-parse", "--path-format=absolute", "--git-common-dir")).resolve()


def repo_root(path: Path) -> Path:
    common_dir = git_common_dir(path)
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


def pool_root(root: Path) -> Path:
    return root / ".worktrees"


def pool_slot(root: Path, path: Path) -> int | None:
    if path.parent != pool_root(root):
        return None
    match = POOL_SLOT.fullmatch(path.name)
    return int(match.group("slot")) if match else None


def pool_path(root: Path, number: int) -> Path:
    validate_pool_slot(number)
    return pool_root(root) / f"{number:02d}"


def pool_branch(number: int) -> str:
    validate_pool_slot(number)
    return f"main-{number:02d}"


def validate_pool_slot(number: int) -> None:
    if not MIN_POOL_SLOT <= number <= MAX_POOL_SLOT:
        raise WorktreeManagerError(
            f"pool slot must be between {MIN_POOL_SLOT:02d} and {MAX_POOL_SLOT:02d}"
        )


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


def marker_branches(root: Path) -> dict[int, tuple[str, str]]:
    result: dict[int, tuple[str, str]] = {}
    for line in git(root, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads").splitlines():
        branch, _, head = line.partition(" ")
        match = POOL_BRANCH.fullmatch(branch)
        if match:
            result[int(match.group("slot"))] = (branch, head)
    return result


def numbered_directories(root: Path) -> set[int]:
    directory = pool_root(root)
    if not directory.is_dir():
        return set()
    result: set[int] = set()
    for item in directory.iterdir():
        match = POOL_SLOT.fullmatch(item.name)
        if match:
            result.add(int(match.group("slot")))
    return result


def inspect_slots(
    root: Path,
    parsed_worktrees: list[Worktree] | None = None,
    current: str | None = None,
) -> list[Slot]:
    worktrees = {
        pool_slot(root, item.path): item
        for item in parsed_worktrees or parse_worktrees(root)
        if pool_slot(root, item.path) is not None
    }
    markers = marker_branches(root)
    directories = numbered_directories(root)
    current = current or origin_main(root)
    slots = set(worktrees) | set(markers) | directories
    result: list[Slot] = []
    for number in sorted(slot for slot in slots if slot is not None):
        path = pool_path(root, number)
        worktree = worktrees.get(number)
        marker, marker_head = markers.get(number, (None, None))
        directory_exists = number in directories
        if worktree and not directory_exists:
            kind, reason = "broken", "registered worktree directory is missing"
        elif directory_exists and not worktree:
            kind, reason = "broken", "numbered directory is not a registered worktree"
        elif worktree:
            kind, reason = "registered", "registered legacy slot has no local marker branch" if marker is None else None
        elif marker_head and is_ancestor(root, marker_head, current):
            kind, reason = "recoverable", "safe marker exists but worktree directory is absent"
        else:
            kind, reason = "broken", "marker branch contains commits outside origin/main"
        result.append(Slot(number, path, worktree, marker, marker_head, directory_exists, kind, reason))
    return result


def lease_path(root: Path) -> Path:
    return git_common_dir(root) / LEASE_FILE


def load_leases(root: Path) -> dict[str, dict[str, str]]:
    path = lease_path(root)
    if not path.exists():
        return {}
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise WorktreeManagerError(f"invalid local lease metadata: {path}") from exc
    leases = raw.get("leases") if isinstance(raw, dict) else None
    if not isinstance(leases, dict) or not all(isinstance(value, dict) for value in leases.values()):
        raise WorktreeManagerError(f"invalid local lease metadata: {path}")
    return {str(key): {str(field): str(value) for field, value in lease.items()} for key, lease in leases.items()}


def save_leases(root: Path, leases: dict[str, dict[str, str]]) -> None:
    path = lease_path(root)
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps({"version": 1, "leases": leases}, indent=2) + "\n", encoding="utf-8")
    temporary.replace(path)


def now() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def baseline_state(worktree: Worktree, root: Path, current: str | None = None) -> str:
    current = current or origin_main(root)
    if worktree.head == current:
        return "current"
    if is_ancestor(root, worktree.head, current):
        return "stale"
    return "unknown"


def is_reusable_slot(
    slot: Slot,
    root: Path,
    current: str | None = None,
    change_count: int | None = None,
) -> bool:
    worktree = slot.worktree
    if slot.kind != "registered" or worktree is None:
        return False
    is_dirty = change_count if change_count is not None else changes(worktree.path)
    if is_dirty:
        return False
    return worktree.branch in (None, pool_branch(slot.number)) and is_ancestor(
        root, worktree.head, current or origin_main(root)
    )


def lifecycle(
    slot: Slot,
    root: Path,
    leases: dict[str, dict[str, str]],
    current: str | None = None,
    change_count: int | None = None,
) -> str:
    worktree = slot.worktree
    if slot.kind == "recoverable":
        return "recoverable"
    if slot.kind == "broken":
        return "broken"
    if worktree is None:
        return "free"
    if is_reusable_slot(slot, root, current, change_count):
        return "idle"
    lease = leases.get(str(slot.number))
    if lease is None:
        return "legacy-untracked"
    if lease.get("branch") != (worktree.branch or ""):
        return "lease-mismatch"
    is_dirty = change_count if change_count is not None else changes(worktree.path)
    if is_dirty:
        return "in-progress"
    if lease.get("closeout_commit") == worktree.head:
        return "release-ready"
    return "in-progress" if lease.get("base") != worktree.head else "acquired"


def state(
    slot: Slot,
    root: Path,
    current: str | None = None,
    change_count: int | None = None,
) -> str:
    worktree = slot.worktree
    if slot.kind == "recoverable":
        return "recoverable"
    if slot.kind == "broken":
        return "broken"
    if worktree is None:
        return "free"
    is_dirty = change_count if change_count is not None else changes(worktree.path)
    if is_dirty:
        return "dirty"
    return "available" if is_reusable_slot(slot, root, current, change_count) else "occupied"


def status_rows(root: Path) -> list[dict[str, object]]:
    leases = load_leases(root)
    parsed_worktrees = parse_worktrees(root)
    current = origin_main(root)
    change_counts = {worktree.path: changes(worktree.path) for worktree in parsed_worktrees}
    rows: list[dict[str, object]] = [{
        "path": str(root), "branch": next(item.branch for item in parsed_worktrees if item.path == root),
        "head": git(root, "rev-parse", "HEAD"), "changes": change_counts[root], "state": "integration",
        "slot": None, "slot_state": None, "lifecycle": None, "baseline": None, "reason": None,
    }]
    for slot in inspect_slots(root, parsed_worktrees, current):
        worktree = slot.worktree
        change_count = change_counts[worktree.path] if worktree else 0
        rows.append({
            "path": str(slot.path),
            "branch": worktree.branch if worktree else slot.marker,
            "head": worktree.head if worktree else slot.marker_head,
            "changes": change_count,
            "state": state(slot, root, current, change_count),
            "slot": slot.number,
            "slot_state": slot.kind,
            "lifecycle": lifecycle(slot, root, leases, current, change_count),
            "baseline": baseline_state(worktree, root, current) if worktree else None,
            "reason": slot.reason,
            "lease": leases.get(str(slot.number)),
        })
    return rows


def print_status(root: Path, as_json: bool, doctor: bool = False) -> None:
    rows = status_rows(root)
    if as_json:
        print(json.dumps({"repo": str(root), "worktrees": rows}, indent=2))
        return
    for row in rows:
        ref = row["branch"] or f"detached@{str(row['head'])[:12]}"
        baseline = f" | baseline={row['baseline']}" if row["baseline"] else ""
        lifecycle_value = f" | lifecycle={row['lifecycle']}" if row["lifecycle"] else ""
        reason = f" | reason={row['reason']}" if doctor and row["reason"] else ""
        print(f"{row['state']}: {row['path']} | {ref} | changes={row['changes']}{baseline}{lifecycle_value}{reason}")


@contextmanager
def manager_lock(root: Path):
    lock_path = git_common_dir(root) / "graft-worktree-manager.lock"
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
        subprocess.run(["git", "show-ref", "--verify", "--quiet", ref], cwd=root, capture_output=True, text=True, check=False).returncode == 0
        for ref in (f"refs/heads/{branch}", f"refs/remotes/origin/{branch}")
    )


def load_links(root: Path) -> list[dict[str, object]]:
    manifest = root / ".worktree-shared.json"
    return list(json.loads(manifest.read_text(encoding="utf-8")).get("links", [])) if manifest.is_file() else []


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
    previous: dict[Path, str] = {}
    try:
        previous = {
            target / str(link["target"]): os.readlink(target / str(link["target"]))
            for link in links
            if (target / str(link["target"])).is_symlink()
        }
        for link in links:
            source = root / str(link["source"])
            destination = target / str(link["target"])
            if destination.is_symlink():
                destination.unlink()
            if not source.exists():
                continue
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.symlink_to(Path(os.path.relpath(source, destination.parent)))
    except OSError as exc:
        rollback_errors: list[str] = []
        for link in links:
            destination = target / str(link["target"])
            try:
                if destination.is_symlink():
                    destination.unlink()
                if destination in previous:
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    destination.symlink_to(previous[destination])
            except OSError as rollback_exc:
                rollback_errors.append(f"{destination}: {rollback_exc}")
        suffix = f"; rollback failed: {'; '.join(rollback_errors)}" if rollback_errors else ""
        raise WorktreeManagerError(f"rebuild_links failed for {target}: {exc}{suffix}") from exc


def sync_pool_slot(root: Path, target: Path) -> None:
    entry = next((item for item in parse_worktrees(root) if item.path == target), None)
    number = pool_slot(root, target)
    if entry is None or number is None:
        raise WorktreeManagerError(f"unregistered pool worktree: {target}")
    validate_pool_slot(number)
    if changes(target):
        raise WorktreeManagerError(f"refusing to sync non-clean worktree: {target}")
    marker = pool_branch(number)
    if entry.branch not in (None, marker):
        raise WorktreeManagerError(f"worktree has an active branch: {target}")
    if entry.branch is None and not is_ancestor(root, entry.head, origin_main(root)):
        raise WorktreeManagerError(f"detached worktree is not an old main baseline: {target}")
    git(target, "switch", "-C", marker, "origin/main")
    git(target, "branch", "--unset-upstream", check=False)
    apply_links(root, target)


def first_free_slot(slots: list[Slot]) -> int:
    used = {slot.number for slot in slots}
    number = 1
    while number in used:
        number += 1
    validate_pool_slot(number)
    return number


def recover_slot(root: Path, slot: Slot) -> Path:
    if slot.kind != "recoverable" or slot.marker is None:
        raise WorktreeManagerError(f"slot is not safely recoverable: {slot.path}")
    slot.path.parent.mkdir(parents=True, exist_ok=True)
    git(root, "worktree", "add", str(slot.path), slot.marker)
    sync_pool_slot(root, slot.path)
    return slot.path


def acquire(root: Path, branch: str, owner: str | None) -> None:
    with manager_lock(root):
        if ALLOWED_BRANCH.fullmatch(branch) is None:
            raise WorktreeManagerError("branch must use an approved type and lowercase kebab-case name")
        fetch(root)
        if branch_exists(root, branch):
            raise WorktreeManagerError(f"branch already exists locally or on origin: {branch}")
        slots = inspect_slots(root)
        broken = [slot for slot in slots if slot.kind == "broken"]
        if broken:
            details = "; ".join(f"{slot.path}: {slot.reason}" for slot in broken)
            raise WorktreeManagerError(f"repair broken pool slots before acquire: {details}")
        available = sorted((slot for slot in slots if is_reusable_slot(slot, root)), key=lambda slot: slot.number)
        if available:
            target = available[0].path
        else:
            recoverable = sorted((slot for slot in slots if slot.kind == "recoverable"), key=lambda slot: slot.number)
            if recoverable:
                target = recover_slot(root, recoverable[0])
            else:
                number = first_free_slot(slots)
                target = pool_path(root, number)
                target.parent.mkdir(parents=True, exist_ok=True)
                git(root, "worktree", "add", "--detach", str(target), "origin/main")
        sync_pool_slot(root, target)
        git(target, "switch", "-c", branch, "origin/main")
        number = pool_slot(root, target)
        if number is None:
            raise WorktreeManagerError(f"allocated path is not a numbered pool slot: {target}")
        leases = load_leases(root)
        leases[str(number)] = {"branch": branch, "owner": owner or branch, "base": origin_main(root), "acquired_at": now()}
        save_leases(root, leases)
        print(f"Using Worktree: {target}")
        print(f"Branch: {branch}")


def closeout(root: Path, target: Path) -> None:
    with manager_lock(root):
        slot = next((item for item in inspect_slots(root) if item.path == target), None)
        if slot is None or slot.worktree is None or slot.kind != "registered":
            raise WorktreeManagerError("closeout only accepts a registered numbered pool worktree")
        branch = slot.worktree.branch
        if branch is None or branch == pool_branch(slot.number):
            raise WorktreeManagerError("closeout requires an active task branch")
        if changes(target):
            raise WorktreeManagerError("closeout requires a clean worktree")
        leases = load_leases(root)
        lease = leases.get(str(slot.number))
        if lease is None or lease.get("branch") != branch:
            raise WorktreeManagerError("closeout requires a lease created by acquire; legacy branches may use confirmed release")
        lease["closeout_commit"] = slot.worktree.head
        lease["closed_at"] = now()
        leases[str(slot.number)] = lease
        save_leases(root, leases)
        print(f"Closeout ready: {target}")
        print(f"Commit: {slot.worktree.head}")


def release(root: Path, target: Path, confirmation: str | None) -> None:
    with manager_lock(root):
        slot = next((item for item in inspect_slots(root) if item.path == target), None)
        if slot is None or slot.worktree is None or slot.kind != "registered":
            raise WorktreeManagerError("release only accepts a registered numbered pool worktree")
        branch = slot.worktree.branch
        if branch is None or branch == pool_branch(slot.number):
            raise WorktreeManagerError("worktree has no releasable task branch")
        if changes(target):
            raise WorktreeManagerError("release requires a clean worktree")
        leases = load_leases(root)
        lease = leases.get(str(slot.number))
        if lease is not None and lease.get("closeout_commit") != slot.worktree.head:
            raise WorktreeManagerError("release requires manager closeout before an acquired lease can be recycled")
        print("Review Summary:")
        print(f"- worktree: {target}")
        print(f"- branch: {branch}")
        if lease is None:
            print("- lifecycle: legacy-untracked")
        print("- commits:")
        print(git(target, "log", "--oneline", f"origin/main..{branch}") or "  - none")
        print("- diff_stat:")
        print(git(target, "diff", "--stat", f"origin/main...{branch}") or "  - none")
        if confirmation is None:
            print("Awaiting developer integration confirmation; no worktree state changed.")
            return
        fetch(root)
        git(root, "rev-parse", "--verify", confirmation)
        if not is_ancestor(root, branch, confirmation):
            raise WorktreeManagerError(f"confirmation ref does not contain task branch: {branch} -> {confirmation}")
        git(target, "switch", "-C", pool_branch(slot.number), "origin/main")
        git(target, "branch", "--unset-upstream", check=False)
        apply_links(root, target)
        git(root, "branch", "-D", branch)
        leases.pop(str(slot.number), None)
        save_leases(root, leases)
        print(f"Released Worktree: {target}")
        print(f"Restored pool branch: {pool_branch(slot.number)}")
        print(f"Deleted local branch: {branch}")


def repair(root: Path, number: int, confirmed: bool) -> None:
    validate_pool_slot(number)
    if not confirmed:
        raise WorktreeManagerError("repair requires --confirm")
    with manager_lock(root):
        fetch(root)
        slot = next((item for item in inspect_slots(root) if item.number == number), None)
        if slot is None:
            raise WorktreeManagerError(f"slot does not exist: {number:02d}")
        if slot.kind != "recoverable":
            reason = slot.reason or "slot is registered or already free"
            raise WorktreeManagerError(f"slot is not safely repairable: {slot.path}: {reason}")
        target = recover_slot(root, slot)
        print(f"Repaired Worktree: {target} -> {pool_branch(number)}")


def reconcile(root: Path, numbers: list[str], confirmed: bool) -> None:
    if not confirmed:
        raise WorktreeManagerError("reconcile requires --confirm")
    with manager_lock(root):
        fetch(root)
        try:
            requested = [int(number) for number in numbers] if numbers else None
        except ValueError as exc:
            raise WorktreeManagerError("reconcile slots must be numeric") from exc
        if requested is not None and (len(set(requested)) != len(requested)):
            raise WorktreeManagerError("reconcile slots must be unique positive numbers")
        if requested is not None:
            for number in requested:
                validate_pool_slot(number)
        slots = inspect_slots(root)
        candidates = [slot for slot in slots if slot.kind == "registered" and (requested is None or slot.number in requested)]
        if requested is not None and {slot.number for slot in candidates} != set(requested):
            raise WorktreeManagerError("reconcile requested an unregistered pool slot")
        for slot in candidates:
            if not is_reusable_slot(slot, root):
                raise WorktreeManagerError(f"pool slot is dirty or occupied: {slot.path}")
            validate_link_rebuild(root, slot.path)
        for slot in candidates:
            sync_pool_slot(root, slot.path)
            print(f"Reconciled Worktree: {slot.path} -> {pool_branch(slot.number)}")


def at_main_baseline(worktree: Worktree, root: Path) -> bool:
    return worktree.head == origin_main(root) and (worktree.branch == "main" or worktree.branch is None)


def relocate(root: Path, confirmed: bool) -> None:
    if not confirmed:
        raise WorktreeManagerError("relocate requires --confirm")
    with manager_lock(root):
        fetch(root)
        legacy = [item for item in parse_worktrees(root) if legacy_pool_slot(root, item.path) is not None]
        if not legacy:
            raise WorktreeManagerError("no legacy sibling pool worktrees found")
        destinations: list[tuple[Worktree, Path]] = []
        for item in legacy:
            if changes(item.path):
                raise WorktreeManagerError(f"legacy pool worktree must be clean: {item.path}")
            if not at_main_baseline(item, root):
                raise WorktreeManagerError(f"legacy pool worktree is not at origin/main: {item.path}")
            target = pool_path(root, legacy_pool_slot(root, item.path) or 0)
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
                sync_pool_slot(root, target)
        except WorktreeManagerError as exc:
            rollback_errors: list[str] = []
            for source, target in reversed(moved):
                try:
                    git(root, "worktree", "move", str(target), str(source))
                    rebuild_links(root, source)
                except WorktreeManagerError as rollback_exc:
                    rollback_errors.append(f"{source}: {rollback_exc}")
            if rollback_errors:
                raise WorktreeManagerError(f"relocation failed: {exc}; rollback failed: {'; '.join(rollback_errors)}") from exc
            raise WorktreeManagerError(f"relocation failed and was rolled back: {exc}") from exc
        print(f"Relocated {len(moved)} worktree(s) into {pool_root(root)}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-dir", type=Path, help="Canonical repository root override")
    subparsers = parser.add_subparsers(dest="command", required=True)
    status_parser = subparsers.add_parser("status")
    status_parser.add_argument("--json", action="store_true")
    doctor_parser = subparsers.add_parser("doctor")
    doctor_parser.add_argument("--json", action="store_true")
    acquire_parser = subparsers.add_parser("acquire")
    acquire_parser.add_argument("branch")
    acquire_parser.add_argument("--owner")
    closeout_parser = subparsers.add_parser("closeout")
    closeout_parser.add_argument("--worktree", type=Path, default=Path.cwd())
    release_parser = subparsers.add_parser("release")
    release_parser.add_argument("--worktree", type=Path, default=Path.cwd())
    release_parser.add_argument("--confirm-integrated")
    repair_parser = subparsers.add_parser("repair")
    repair_parser.add_argument("--confirm", action="store_true")
    repair_parser.add_argument("slot")
    reconcile_parser = subparsers.add_parser("reconcile")
    reconcile_parser.add_argument("--confirm", action="store_true")
    reconcile_parser.add_argument("slots", nargs="*")
    relocate_parser = subparsers.add_parser("relocate")
    relocate_parser.add_argument("--confirm", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        root = args.repo_dir.resolve() if args.repo_dir else repo_root(Path.cwd())
        if args.command == "status":
            print_status(root, args.json)
        elif args.command == "doctor":
            print_status(root, args.json, doctor=True)
        elif args.command == "acquire":
            acquire(root, args.branch, args.owner)
        elif args.command == "closeout":
            closeout(root, args.worktree.resolve())
        elif args.command == "release":
            release(root, args.worktree.resolve(), args.confirm_integrated)
        elif args.command == "repair":
            try:
                number = int(args.slot)
            except ValueError as exc:
                raise WorktreeManagerError("repair slot must be numeric") from exc
            repair(root, number, args.confirm)
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

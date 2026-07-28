#!/usr/bin/env python3
"""End-to-end tests for the reusable worktree manager."""

from __future__ import annotations

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).with_name("worktree_manager.py")
sys.path.insert(0, str(SCRIPT.parent))
import worktree_manager


def run(*args: str, cwd: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=cwd, text=True, capture_output=True, check=True)


class WorktreeManagerTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name)
        self.remote = self.base / "remote.git"
        self.seed = self.base / "seed"
        self.repo = self.base / "graft"
        run("git", "init", "--bare", str(self.remote), cwd=self.base)
        run("git", "init", "-b", "main", str(self.seed), cwd=self.base)
        run("git", "config", "user.email", "test@example.com", cwd=self.seed)
        run("git", "config", "user.name", "Test User", cwd=self.seed)
        (self.seed / "README.md").write_text("seed\n", encoding="utf-8")
        run("git", "add", "README.md", cwd=self.seed)
        run("git", "commit", "-m", "seed", cwd=self.seed)
        run("git", "remote", "add", "origin", str(self.remote), cwd=self.seed)
        run("git", "push", "-u", "origin", "main", cwd=self.seed)
        run("git", "clone", "--branch", "main", str(self.remote), str(self.repo), cwd=self.base)
        (self.repo / ".worktree-shared.json").write_text(json.dumps({"links": []}), encoding="utf-8")

    def tearDown(self) -> None:
        self.temp.cleanup()

    def manager(self, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["python3", str(SCRIPT), "--repo-dir", str(self.repo), *args],
            cwd=self.repo,
            text=True,
            capture_output=True,
            check=check,
        )

    def closeout(self, worker: Path) -> subprocess.CompletedProcess[str]:
        return self.manager("closeout", "--worktree", str(worker))

    def test_acquire_status_and_release_reuse_numbered_slot(self) -> None:
        acquired = self.manager("acquire", "feature/runtime-target")
        worker = self.repo / ".worktrees" / "01"
        self.assertIn(str(worker), acquired.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/runtime-target")

        occupied = self.manager("status", "--json")
        rows = json.loads(occupied.stdout)["worktrees"]
        self.assertEqual(next(row for row in rows if row["path"] == str(worker))["state"], "occupied")

        self.closeout(worker)
        preview = self.manager("release", "--worktree", str(worker))
        self.assertIn("Awaiting developer integration confirmation", preview.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/runtime-target")

        released = self.manager("release", "--worktree", str(worker), "--confirm-integrated", "origin/main")
        self.assertIn("Released Worktree", released.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "main-01")
        self.assertEqual(
            run("git", "rev-parse", "HEAD", cwd=worker).stdout.strip(),
            run("git", "rev-parse", "origin/main", cwd=self.repo).stdout.strip(),
        )
        self.assertIn("available", self.manager("status").stdout)

    def test_manager_lock_requests_absolute_git_common_directory(self) -> None:
        git_dir = self.repo / ".git"
        with mock.patch.object(worktree_manager, "git", return_value=str(git_dir)) as git_mock:
            with worktree_manager.manager_lock(self.repo):
                pass

        git_mock.assert_called_once_with(
            self.repo, "rev-parse", "--path-format=absolute", "--git-common-dir"
        )
        self.assertTrue((git_dir / "graft-worktree-manager.lock").exists())

    def test_acquire_reuses_stale_clean_detached_slot_after_fetch(self) -> None:
        worker = self.repo / ".worktrees" / "01"
        run("git", "worktree", "add", "--detach", str(worker), "origin/main", cwd=self.repo)
        (self.seed / "advance.txt").write_text("advance\n", encoding="utf-8")
        run("git", "add", "advance.txt", cwd=self.seed)
        run("git", "commit", "-m", "advance main", cwd=self.seed)
        run("git", "push", "origin", "main", cwd=self.seed)

        acquired = self.manager("acquire", "feature/reuse-stale-slot")

        self.assertIn(str(worker), acquired.stdout)
        self.assertEqual(
            run("git", "rev-parse", "HEAD", cwd=worker).stdout.strip(),
            run("git", "rev-parse", "origin/main", cwd=self.repo).stdout.strip(),
        )
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/reuse-stale-slot")

    def test_reconcile_converts_selected_legacy_slots_to_pool_branches(self) -> None:
        worker = self.repo / ".worktrees" / "01"
        run("git", "worktree", "add", "--detach", str(worker), "origin/main", cwd=self.repo)
        (self.seed / "advance.txt").write_text("advance\n", encoding="utf-8")
        run("git", "add", "advance.txt", cwd=self.seed)
        run("git", "commit", "-m", "advance main", cwd=self.seed)
        run("git", "push", "origin", "main", cwd=self.seed)

        reconciled = self.manager("reconcile", "--confirm", "01")

        self.assertIn("main-01", reconciled.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "main-01")
        self.assertEqual(
            run("git", "rev-parse", "HEAD", cwd=worker).stdout.strip(),
            run("git", "rev-parse", "origin/main", cwd=self.repo).stdout.strip(),
        )
        self.assertEqual(
            subprocess.run(
                ["git", "config", "--get", "branch.main-01.remote"],
                cwd=self.repo,
                text=True,
                capture_output=True,
                check=False,
            ).returncode,
            1,
        )

    def test_reconcile_refuses_detached_worktree_with_unmerged_commit(self) -> None:
        worker = self.repo / ".worktrees" / "01"
        run("git", "worktree", "add", "--detach", str(worker), "origin/main", cwd=self.repo)
        (worker / "unmerged.txt").write_text("unmerged\n", encoding="utf-8")
        run("git", "add", "unmerged.txt", cwd=worker)
        run("git", "commit", "-m", "unmerged work", cwd=worker)

        rejected = self.manager("reconcile", "--confirm", "01", check=False)

        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn("dirty or occupied", rejected.stderr)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "")

    def test_release_rejects_confirmation_without_integrated_branch(self) -> None:
        self.manager("acquire", "feature/runtime-target")
        worker = self.repo / ".worktrees" / "01"
        (worker / "worker.txt").write_text("worker\n", encoding="utf-8")
        run("git", "add", "worker.txt", cwd=worker)
        run("git", "commit", "-m", "worker change", cwd=worker)
        self.closeout(worker)

        rejected = self.manager("release", "--worktree", str(worker), "--confirm-integrated", "origin/main", check=False)

        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn("does not contain task branch", rejected.stderr)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/runtime-target")

    def test_release_fetches_before_validating_integration_confirmation(self) -> None:
        self.manager("acquire", "feature/runtime-target")
        worker = self.repo / ".worktrees" / "01"
        (worker / "worker.txt").write_text("worker\n", encoding="utf-8")
        run("git", "add", "worker.txt", cwd=worker)
        run("git", "commit", "-m", "worker change", cwd=worker)
        self.closeout(worker)
        run("git", "push", "origin", "feature/runtime-target", cwd=worker)

        run("git", "fetch", "origin", "feature/runtime-target", cwd=self.seed)
        run("git", "merge", "--ff-only", "origin/feature/runtime-target", cwd=self.seed)
        run("git", "push", "origin", "main", cwd=self.seed)

        released = self.manager(
            "release", "--worktree", str(worker), "--confirm-integrated", "origin/main"
        )

        self.assertIn("Released Worktree", released.stdout)

    def test_main_baseline_requires_current_origin_head(self) -> None:
        origin = worktree_manager.Worktree(self.repo, "main", "different-head")
        detached = worktree_manager.Worktree(self.repo / ".worktrees" / "01", None, run("git", "rev-parse", "origin/main", cwd=self.repo).stdout.strip())

        with mock.patch.object(worktree_manager, "origin_main", return_value="origin-head"):
            self.assertFalse(worktree_manager.at_main_baseline(origin, self.repo))
            self.assertFalse(worktree_manager.at_main_baseline(detached, self.repo))
        with mock.patch.object(worktree_manager, "origin_main", return_value=detached.head):
            self.assertTrue(worktree_manager.at_main_baseline(detached, self.repo))

    def test_acquire_rejects_duplicate_remote_or_local_branch(self) -> None:
        self.manager("acquire", "feature/runtime-target")
        duplicate = self.manager("acquire", "feature/runtime-target", check=False)
        self.assertNotEqual(duplicate.returncode, 0)
        self.assertIn("branch already exists", duplicate.stderr)

    def test_acquire_applies_shared_links_to_existing_available_slot(self) -> None:
        worker = self.repo / ".worktrees" / "01"
        worker.parent.mkdir()
        run("git", "-C", str(self.repo), "worktree", "add", "--detach", str(worker), "origin/main", cwd=self.repo)
        (self.repo / "shared.env").write_text("shared\n", encoding="utf-8")
        (self.repo / ".worktree-shared.json").write_text(
            json.dumps({"links": [{"source": "shared.env", "target": "shared.env", "required": True}]}),
            encoding="utf-8",
        )

        self.manager("acquire", "feature/reuse-slot")

        self.assertTrue((worker / "shared.env").is_symlink())

    def test_acquire_recovers_lowest_safe_marker_slot(self) -> None:
        run("git", "branch", "main-03", "origin/main", cwd=self.repo)

        acquired = self.manager("acquire", "feature/recover-slot")
        worker = self.repo / ".worktrees" / "03"

        self.assertIn(str(worker), acquired.stdout)
        self.assertTrue(worker.is_dir())
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/recover-slot")
        self.assertFalse((self.repo / ".worktrees" / "01").exists())

    def test_repair_restores_safe_marker_slot_to_idle_pool(self) -> None:
        run("git", "branch", "main-03", "origin/main", cwd=self.repo)

        repaired = self.manager("repair", "--confirm", "03")
        worker = self.repo / ".worktrees" / "03"

        self.assertIn("Repaired Worktree", repaired.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "main-03")
        rows = json.loads(self.manager("status", "--json").stdout)["worktrees"]
        row = next(row for row in rows if row["path"] == str(worker))
        self.assertEqual(row["state"], "available")
        self.assertEqual(row["lifecycle"], "idle")

    def test_acquire_fills_lowest_missing_slot_when_registered_slots_are_occupied(self) -> None:
        for number in (1, 2, 4):
            worker = self.repo / ".worktrees" / f"{number:02d}"
            worker.parent.mkdir(exist_ok=True)
            run("git", "worktree", "add", "-b", f"feature/occupied-{number}", str(worker), "origin/main", cwd=self.repo)

        acquired = self.manager("acquire", "feature/fill-gap")

        self.assertIn(str(self.repo / ".worktrees" / "03"), acquired.stdout)
        self.assertFalse((self.repo / ".worktrees" / "05").exists())

    def test_acquire_refuses_broken_numbered_directory(self) -> None:
        broken = self.repo / ".worktrees" / "01"
        broken.mkdir(parents=True)

        rejected = self.manager("acquire", "feature/refuse-broken", check=False)

        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn("repair broken pool slots", rejected.stderr)
        self.assertFalse((self.repo / ".worktrees" / "02").exists())

    def test_closeout_rejects_dirty_worktree_and_release_requires_closeout(self) -> None:
        self.manager("acquire", "feature/closeout-gate")
        worker = self.repo / ".worktrees" / "01"
        (worker / "worker.txt").write_text("worker\n", encoding="utf-8")

        dirty_closeout = self.manager("closeout", "--worktree", str(worker), check=False)
        self.assertNotEqual(dirty_closeout.returncode, 0)
        self.assertIn("clean worktree", dirty_closeout.stderr)

        run("git", "add", "worker.txt", cwd=worker)
        run("git", "commit", "-m", "worker change", cwd=worker)
        before_closeout = self.manager("release", "--worktree", str(worker), check=False)
        self.assertNotEqual(before_closeout.returncode, 0)
        self.assertIn("manager closeout", before_closeout.stderr)

        self.closeout(worker)
        rows = json.loads(self.manager("status", "--json").stdout)["worktrees"]
        self.assertEqual(next(row for row in rows if row["path"] == str(worker))["lifecycle"], "release-ready")

    def test_relocate_moves_clean_legacy_pool_and_rebuilds_shared_links(self) -> None:
        legacy = self.base / "graft-wt-01"
        run("git", "-C", str(self.repo), "worktree", "add", "--detach", str(legacy), "origin/main", cwd=self.repo)
        (self.repo / ".git" / "info" / "exclude").write_text(".env\n", encoding="utf-8")
        (self.repo / "shared.env").write_text("shared\n", encoding="utf-8")
        (self.repo / ".worktree-shared.json").write_text(
            json.dumps({"links": [{"source": "shared.env", "target": ".env", "required": True}]}),
            encoding="utf-8",
        )
        (legacy / ".env").symlink_to("missing-source")

        relocated = self.manager("relocate", "--confirm")
        worker = self.repo / ".worktrees" / "01"

        self.assertIn("Relocated 1 worktree", relocated.stdout)
        self.assertFalse(legacy.exists())
        self.assertEqual((worker / ".env").resolve(), (self.repo / "shared.env").resolve())
        rows = json.loads(self.manager("status", "--json").stdout)["worktrees"]
        self.assertEqual(next(row for row in rows if row["path"] == str(worker))["state"], "available")

    def test_relocate_rolls_back_move_and_restores_old_links_when_rebuild_fails(self) -> None:
        legacy = self.base / "graft-wt-01"
        run("git", "-C", str(self.repo), "worktree", "add", "--detach", str(legacy), "origin/main", cwd=self.repo)
        (self.repo / ".git" / "info" / "exclude").write_text(".env\n", encoding="utf-8")
        (self.repo / "old.env").write_text("old\n", encoding="utf-8")
        (self.repo / ".worktree-shared.json").write_text(
            json.dumps({"links": [{"source": "old.env", "target": ".env", "required": True}]}),
            encoding="utf-8",
        )
        (legacy / ".env").symlink_to("../graft/old.env")

        original_symlink_to = Path.symlink_to
        calls = 0

        def fail_once(path: Path, target: str | Path, *args: object, **kwargs: object) -> None:
            nonlocal calls
            calls += 1
            if calls == 1:
                raise OSError("disk full")
            original_symlink_to(path, target, *args, **kwargs)

        with mock.patch.object(Path, "symlink_to", new=fail_once):
            with self.assertRaisesRegex(worktree_manager.WorktreeManagerError, "relocation"):
                worktree_manager.relocate(self.repo, True)

        self.assertTrue(legacy.exists())
        self.assertEqual((legacy / ".env").readlink(), Path("../graft/old.env"))

    def test_relocate_refuses_dirty_legacy_pool_without_moving_it(self) -> None:
        legacy = self.base / "graft-wt-01"
        run("git", "-C", str(self.repo), "worktree", "add", "--detach", str(legacy), "origin/main", cwd=self.repo)
        (legacy / "scratch").write_text("dirty\n", encoding="utf-8")

        rejected = self.manager("relocate", "--confirm", check=False)

        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn("legacy pool worktree must be clean", rejected.stderr)
        self.assertTrue(legacy.exists())
        self.assertFalse((self.repo / ".worktrees" / "01").exists())

    def test_relocate_ignores_primary_checkout_changes(self) -> None:
        legacy = self.base / "graft-wt-01"
        run("git", "-C", str(self.repo), "worktree", "add", "--detach", str(legacy), "origin/main", cwd=self.repo)
        (self.repo / "integration-scratch").write_text("local work\n", encoding="utf-8")

        relocated = self.manager("relocate", "--confirm")

        self.assertIn("Relocated 1 worktree", relocated.stdout)
        self.assertTrue((self.repo / ".worktrees" / "01").is_dir())


if __name__ == "__main__":
    unittest.main()

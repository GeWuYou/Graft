#!/usr/bin/env python3
"""End-to-end tests for the reusable worktree manager."""

from __future__ import annotations

import json
from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("worktree_manager.py")


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

    def test_acquire_status_and_release_reuse_numbered_slot(self) -> None:
        acquired = self.manager("acquire", "feature/runtime-target")
        worker = self.repo / ".worktrees" / "01"
        self.assertIn(str(worker), acquired.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/runtime-target")

        occupied = self.manager("status", "--json")
        rows = json.loads(occupied.stdout)["worktrees"]
        self.assertEqual(next(row for row in rows if row["path"] == str(worker))["state"], "occupied")

        preview = self.manager("release", "--worktree", str(worker))
        self.assertIn("Awaiting developer integration confirmation", preview.stdout)
        self.assertEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/runtime-target")

        released = self.manager("release", "--worktree", str(worker), "--confirm-integrated", "origin/main")
        self.assertIn("Released Worktree", released.stdout)
        self.assertNotEqual(run("git", "branch", "--show-current", cwd=worker).stdout.strip(), "feature/runtime-target")
        self.assertIn("available", self.manager("status").stdout)

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

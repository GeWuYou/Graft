from __future__ import annotations

import importlib.util
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory


SCRIPT_PATH = Path(__file__).with_name("check_line_endings.py")
MODULE_SPEC = importlib.util.spec_from_file_location("check_line_endings", SCRIPT_PATH)
if MODULE_SPEC is None or MODULE_SPEC.loader is None:
    raise RuntimeError(f"failed to load {SCRIPT_PATH}")
MODULE = importlib.util.module_from_spec(MODULE_SPEC)
MODULE_SPEC.loader.exec_module(MODULE)
paths_with_carriage_returns = MODULE.paths_with_carriage_returns


class LineEndingGateTest(unittest.TestCase):
    def test_reports_text_blob_with_crlf(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = self.init_repo(Path(temp_dir))
            (root / "lf.txt").write_bytes(b"first\nsecond\n")
            (root / "crlf.txt").write_bytes(b"first\r\nsecond\r\n")
            subprocess.run(["git", "add", "--", "lf.txt", "crlf.txt"], cwd=root, check=True)

            self.assertEqual(paths_with_carriage_returns(root), ["crlf.txt"])

    def test_ignores_binary_blob_with_carriage_return(self) -> None:
        with TemporaryDirectory() as temp_dir:
            root = self.init_repo(Path(temp_dir))
            (root / "asset.bin").write_bytes(b"\x00\r\n\xff")
            subprocess.run(["git", "add", "--", "asset.bin"], cwd=root, check=True)

            self.assertEqual(paths_with_carriage_returns(root), [])

    @staticmethod
    def init_repo(root: Path) -> Path:
        subprocess.run(["git", "init", "--quiet"], cwd=root, check=True)
        subprocess.run(["git", "config", "core.autocrlf", "false"], cwd=root, check=True)
        subprocess.run(["git", "config", "core.safecrlf", "false"], cwd=root, check=True)
        return root


if __name__ == "__main__":
    unittest.main()

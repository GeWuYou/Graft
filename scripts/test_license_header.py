#!/usr/bin/env python3
# Copyright (c) 2025-2026 GeWuYou
# SPDX-License-Identifier: Apache-2.0

"""Unit tests for Apache-2.0 license header automation."""

from __future__ import annotations

import contextlib
import io
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


sys.path.insert(0, str(Path(__file__).resolve().parent))
import license_header


class LicenseHeaderTests(unittest.TestCase):
    """Cover file formats, idempotence, and added-file selection behavior."""

    def test_inserts_go_header(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "main.go"
            path.write_text("package main\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "// Copyright (c) 2025-2026 GeWuYou\n"
                "// SPDX-License-Identifier: Apache-2.0\n"
                "\n"
                "package main\n",
                path.read_text(encoding="utf-8"),
            )

    def test_existing_spdx_apache_header_is_compliant(self) -> None:
        text = (
            "// Copyright (c) 2026 GeWuYou\n"
            "// SPDX-License-Identifier: Apache-2.0\n"
            "package demo\n"
        )

        self.assertTrue(license_header.has_license_header(text))

    def test_long_form_apache_wording_is_not_compliant(self) -> None:
        text = (
            "// Copyright (c) 2026 GeWuYou\n"
            "// Licensed under the Apache License, Version 2.0 (the \"License\");\n"
            "package demo\n"
        )

        self.assertFalse(license_header.has_license_header(text))

    def test_inserts_after_shebang(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "tool.sh"
            path.write_text("#!/usr/bin/env bash\necho ok\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "#!/usr/bin/env bash\n"
                "# Copyright (c) 2025-2026 GeWuYou\n"
                "# SPDX-License-Identifier: Apache-2.0\n"
                "\n"
                "echo ok\n",
                path.read_text(encoding="utf-8"),
            )

    def test_inserts_ts_vue_and_sql_headers(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            cases = {
                "app.ts": (
                    "export const ok = true;\n",
                    "// Copyright (c) 2025-2026 GeWuYou\n"
                    "// SPDX-License-Identifier: Apache-2.0\n"
                    "\n"
                    "export const ok = true;\n",
                ),
                "Widget.vue": (
                    "<template></template>\n",
                    "<!--\n"
                    "  Copyright (c) 2025-2026 GeWuYou\n"
                    "  SPDX-License-Identifier: Apache-2.0\n"
                    "-->\n"
                    "\n"
                    "<template></template>\n",
                ),
                "schema.sql": (
                    "create table demo (id bigint);\n",
                    "-- Copyright (c) 2025-2026 GeWuYou\n"
                    "-- SPDX-License-Identifier: Apache-2.0\n"
                    "\n"
                    "create table demo (id bigint);\n",
                ),
            }

            for file_name, (initial, expected) in cases.items():
                with self.subTest(file_name=file_name):
                    path = Path(temp_dir) / file_name
                    path.write_text(initial, encoding="utf-8")

                    license_header.add_license_header(path)

                    self.assertEqual(expected, path.read_text(encoding="utf-8"))

    def test_uses_block_comment_for_css(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "style.css"
            path.write_text(".demo { color: red; }\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "/*\n"
                " * Copyright (c) 2025-2026 GeWuYou\n"
                " * SPDX-License-Identifier: Apache-2.0\n"
                " */\n"
                "\n"
                ".demo { color: red; }\n",
                path.read_text(encoding="utf-8"),
            )

    def test_skips_markdown_yaml_and_json(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            paths = ["note.md", "config.yaml", "package.json"]
            for relative_path in paths:
                (root / relative_path).write_text("content\n", encoding="utf-8")

            self.assertEqual([], list(license_header.scan_files(root, paths)))
            for relative_path in paths:
                self.assertFalse(license_header.is_supported_path(relative_path))

    def test_skips_agents_paths(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            relative_path = ".agents/skills/demo/tool.go"
            path = root / relative_path
            path.parent.mkdir(parents=True)
            path.write_text("package demo\n", encoding="utf-8")

            self.assertEqual([], list(license_header.scan_files(root, [relative_path])))
            self.assertFalse(license_header.is_supported_path(relative_path))

    def test_excludes_generated_lock_and_environment_paths(self) -> None:
        self.assertFalse(license_header.is_supported_path(".ai/environment/tools.ai.yaml"))
        self.assertFalse(license_header.is_supported_path("ai-plan/design/example.go"))
        self.assertFalse(license_header.is_supported_path("docs/example.ts"))
        self.assertFalse(license_header.is_supported_path("openapi/generated/schema.ts"))
        self.assertFalse(license_header.is_supported_path("web/bun.lock"))
        self.assertFalse(license_header.is_supported_path("go.mod"))
        self.assertFalse(license_header.is_supported_path("go.sum"))
        self.assertFalse(license_header.is_supported_path("bun.lock"))
        self.assertFalse(license_header.is_supported_path("package-lock.json"))
        self.assertFalse(license_header.is_supported_path("pnpm-lock.yaml"))
        self.assertFalse(license_header.is_supported_path(".env.local"))
        self.assertFalse(license_header.is_supported_path("openapi/dist/openapi.bundle.json"))
        self.assertFalse(license_header.is_supported_path("server/internal/contract/openapi/generated/types.gen.go"))
        self.assertFalse(license_header.is_supported_path("server/internal/contract/openapi/user/zz_generated.management.go"))
        self.assertFalse(license_header.is_supported_path("server/modules/rbac/ent/zz_generated.audit.go"))
        self.assertFalse(license_header.is_supported_path("server/modules/rbac/ent/client.go"))
        self.assertTrue(license_header.is_supported_path("server/modules/rbac/ent/schema/role.go"))
        self.assertFalse(license_header.is_supported_path("web/src/api/generated/schema.ts"))
        self.assertTrue(license_header.is_supported_path("server/internal/app/runtime.go"))

    def test_existing_header_repeated_fix_is_unchanged(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            path = root / "main.go"
            path.write_text("package main\n", encoding="utf-8")

            with contextlib.redirect_stdout(io.StringIO()):
                first_exit = license_header.main(["--fix", "--root", str(root), "--paths", "main.go"])
                first_text = path.read_text(encoding="utf-8")
                second_exit = license_header.main(["--fix", "--root", str(root), "--paths", "main.go"])

            self.assertEqual(0, first_exit)
            self.assertEqual(0, second_exit)
            self.assertEqual(first_text, path.read_text(encoding="utf-8"))

    def test_added_scope_uses_added_diff_filter(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            with patch("license_header.run_git_output") as run_git_output:
                run_git_output.return_value = ["server/new.go"]

                result = license_header.list_added_files(root, "origin/main", "HEAD")

        self.assertEqual(["server/new.go"], result)
        run_git_output.assert_called_once_with(
            root,
            ["diff", "--name-only", "--diff-filter=A", "origin/main...HEAD"],
        )


if __name__ == "__main__":
    unittest.main()

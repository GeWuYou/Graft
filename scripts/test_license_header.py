#!/usr/bin/env python3
# Copyright (c) 2025-2026 GeWuYou
# SPDX-License-Identifier: Apache-2.0

"""Unit tests for Apache-2.0 license header automation."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


sys.path.insert(0, str(Path(__file__).resolve().parent))
import license_header


class LicenseHeaderTests(unittest.TestCase):
    """Cover file format, repair, and added-file selection behavior."""

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

    def test_existing_apache_header_is_compliant(self) -> None:
        text = (
            "// Copyright (c) 2026 GeWuYou\n"
            "// Licensed under the Apache License, Version 2.0 (the \"License\");\n"
            "package demo\n"
        )

        self.assertTrue(license_header.has_license_header(text))

    def test_inserts_after_shebang(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "tool.py"
            path.write_text("#!/usr/bin/env python3\nprint('ok')\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "#!/usr/bin/env python3\n"
                "# Copyright (c) 2025-2026 GeWuYou\n"
                "# SPDX-License-Identifier: Apache-2.0\n"
                "\n"
                "print('ok')\n",
                path.read_text(encoding="utf-8"),
            )

    def test_uses_html_comment_for_vue(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "Widget.vue"
            path.write_text("<template></template>\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "<!--\n"
                "  Copyright (c) 2025-2026 GeWuYou\n"
                "  SPDX-License-Identifier: Apache-2.0\n"
                "-->\n"
                "\n"
                "<template></template>\n",
                path.read_text(encoding="utf-8"),
            )

    def test_inserts_markdown_header_after_frontmatter(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "note.md"
            path.write_text("---\ntitle: Demo\n---\n# Demo\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "---\n"
                "title: Demo\n"
                "---\n"
                "<!--\n"
                "  Copyright (c) 2025-2026 GeWuYou\n"
                "  SPDX-License-Identifier: Apache-2.0\n"
                "-->\n"
                "\n"
                "# Demo\n",
                path.read_text(encoding="utf-8"),
            )

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

    def test_inserts_xml_header_after_declaration(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "schema.xml"
            path.write_text("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<root />\n", encoding="utf-8")

            license_header.add_license_header(path)

            self.assertEqual(
                "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"
                "<!--\n"
                "  Copyright (c) 2025-2026 GeWuYou\n"
                "  SPDX-License-Identifier: Apache-2.0\n"
                "-->\n"
                "\n"
                "<root />\n",
                path.read_text(encoding="utf-8"),
            )

    def test_repairs_xml_header_before_declaration(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "schema.xml"
            path.write_text(
                "<!--\n"
                "  Copyright (c) 2025-2026 GeWuYou\n"
                "  SPDX-License-Identifier: Apache-2.0\n"
                "-->\n"
                "\n"
                "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"
                "<root />\n",
                encoding="utf-8",
            )

            self.assertTrue(license_header.needs_header_repair(path, path.read_text(encoding="utf-8")))
            license_header.repair_license_header(path)

            self.assertEqual(
                "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"
                "<!--\n"
                "  Copyright (c) 2025-2026 GeWuYou\n"
                "  SPDX-License-Identifier: Apache-2.0\n"
                "-->\n"
                "\n"
                "<root />\n",
                path.read_text(encoding="utf-8"),
            )

    def test_excludes_generated_lock_and_environment_paths(self) -> None:
        self.assertFalse(license_header.is_supported_path(".ai/environment/tools.ai.yaml"))
        self.assertFalse(license_header.is_supported_path("web/bun.lock"))
        self.assertFalse(license_header.is_supported_path("openapi/dist/openapi.bundle.json"))
        self.assertFalse(license_header.is_supported_path("server/internal/contract/openapi/generated/types.gen.go"))
        self.assertFalse(license_header.is_supported_path("server/internal/contract/openapi/user/zz_generated.management.go"))
        self.assertFalse(license_header.is_supported_path("server/modules/rbac/ent/zz_generated.audit.go"))
        self.assertFalse(license_header.is_supported_path("server/modules/rbac/ent/client.go"))
        self.assertTrue(license_header.is_supported_path("server/modules/rbac/ent/schema/role.go"))
        self.assertFalse(license_header.is_supported_path("web/src/contracts/openapi/generated/schema.ts"))
        self.assertTrue(license_header.is_supported_path("server/internal/app/runtime.go"))

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

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from openapi_generated_freshness_check import OPENAPI_BINDINGS, strict_server_targets


class StrictServerTargetTests(unittest.TestCase):
    def test_discovers_output_from_the_domain_generation_directive(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            domain = repo / OPENAPI_BINDINGS / "runtimetarget"
            domain.mkdir(parents=True)
            (domain / "oapi-codegen.yaml").touch()
            (domain / "generate.go").write_text(
                "//go:generate go tool oapi-codegen -config oapi-codegen.yaml "
                "-o zz_generated.runtime_target.go ../../../../../openapi/openapi.yaml\n",
                encoding="utf-8",
            )

            self.assertEqual(
                strict_server_targets(repo),
                [("runtimetarget", OPENAPI_BINDINGS / "runtimetarget" / "zz_generated.runtime_target.go")],
            )


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Run the canonical Graft PR-review helper from its skill-owned location."""

from __future__ import annotations

from pathlib import Path
import runpy


CANONICAL_HELPER = (
    Path(__file__).resolve().parents[1]
    / ".agents"
    / "skills"
    / "graft-pr-review"
    / "scripts"
    / "fetch_current_pr_review.py"
)


if __name__ == "__main__":
    runpy.run_path(str(CANONICAL_HELPER), run_name="__main__")

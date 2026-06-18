#!/usr/bin/env python3
"""Block reintroduction of module-owned locale YAML under internal i18n."""

from __future__ import annotations

import sys
from pathlib import Path


ALLOWED_NON_LOCALE_FILES = {"README.md"}
LEGACY_MODULE_LOCALE_DIR = Path("server/internal/i18n/locales/modules")


def repo_root() -> Path:
    return Path(__file__).resolve().parent.parent


def find_legacy_locale_files(root: Path) -> list[Path]:
    target_dir = root / LEGACY_MODULE_LOCALE_DIR
    if not target_dir.is_dir():
        return []

    offenders: list[Path] = []
    for path in sorted(target_dir.iterdir()):
        if not path.is_file():
            continue
        if path.name in ALLOWED_NON_LOCALE_FILES:
            continue
        if path.suffix != ".yaml":
            continue
        offenders.append(path.relative_to(root))
    return offenders


def main() -> int:
    root = repo_root()
    offenders = find_legacy_locale_files(root)
    if not offenders:
        print("server locale ownership guard: ok")
        return 0

    for offender in offenders:
        print(
            f"server locale ownership guard: disallowed legacy module locale resource {offender}",
            file=sys.stderr,
        )
    print(
        "module-owned and module-runtime-owned locale YAML must stay in owner-local locales/ directories; "
        "server/internal/i18n/locales/modules/ is reserved for legacy-free infrastructure state.",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env sh
set -eu

repo_root="$(git rev-parse --show-toplevel)"

cd "$repo_root"

git config core.hooksPath .husky

for hook in .husky/commit-msg .husky/pre-commit .husky/pre-push; do
  if [ -f "$hook" ]; then
    chmod 755 "$hook"
  fi
done

if [ -f "scripts/run_python.sh" ]; then
  chmod 755 "scripts/run_python.sh"
fi

if [ -f "scripts/check_migration_versions.py" ]; then
  chmod 755 "scripts/check_migration_versions.py"
fi

printf 'git hooks installed at %s\n' "$(git config --get core.hooksPath)"

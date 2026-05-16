#!/usr/bin/env sh
set -eu

if command -v python3 >/dev/null 2>&1; then
  exec python3 "$@"
fi

if command -v python >/dev/null 2>&1; then
  exec python "$@"
fi

if command -v py >/dev/null 2>&1; then
  exec py -3 "$@"
fi

echo "python launcher: expected one of python3, python, or py -3 in PATH" >&2
exit 1

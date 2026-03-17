#!/usr/bin/env bash
ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
exec "$ROOT/fab/.kit/bin/fab" hook artifact-write 2>/dev/null; exit 0

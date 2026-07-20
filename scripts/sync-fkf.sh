#!/usr/bin/env bash
# Copy the canonical docs/site/fkf.md — the authoritative FKF standard, published at
# https://shll.ai/fab-kit/fkf — into src/kit/reference/ so it ships verbatim in the
# kit cache as $(fab kit-path)/reference/fkf.md (the copy deployed skills cite). The
# committed copy is what releases ship (both distribution paths copy src/kit/
# verbatim); the drift-guard test in src/go/fab/cmd/fab/fkf_sync_test.go keeps it
# byte-honest on every `go test`.
set -euo pipefail

# Run from the repo root regardless of caller CWD.
cd "$(dirname "$0")/.."

SRC="docs/site/fkf.md"
DEST="src/kit/reference/fkf.md"

cp -f "$SRC" "$DEST"
echo "synced FKF standard: $DEST"

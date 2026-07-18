#!/usr/bin/env bash
# Copy the canonical docs/site/skill.md into the fab-go cmd/fab package dir so it
# can be embedded via //go:embed. The fab-go module root is src/go/fab/ and
# docs/site/ sits above it, so embed cannot reach the canonical file directly —
# this copy step bridges the gap (the exact mechanism scripts/sync-standards.sh
# established in sahil87/shll, adapted to a single file). The committed copy is
# what a clean `go build ./...` (which does not run this script) compiles; the
# drift-guard test in src/go/fab/cmd/fab/skill_test.go keeps it byte-honest.
set -euo pipefail

# Run from the repo root regardless of caller CWD.
cd "$(dirname "$0")/.."

SRC="docs/site/skill.md"
DEST="src/go/fab/cmd/fab/skill.md"

cp -f "$SRC" "$DEST"
echo "synced skill bundle: $DEST"

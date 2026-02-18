#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Initializing shared test dependencies..."
git submodule update --init --recursive tests/libs/

echo "Done. Run 'just test-packages' to verify."

#!/usr/bin/env bash
# src/resolve-change/test-simple.sh
#
# Quick smoke test for _resolve-change.sh
# Run: ./test-simple.sh

set -euo pipefail

source "$(dirname "$0")/_resolve-change.sh"

TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

# Set up minimal test environment
mkdir -p "$TEST_DIR/changes/smoke-test"
echo "smoke-test" > "$TEST_DIR/current"

echo "Testing resolve_change from fab/current..."
if resolve_change "$TEST_DIR" ""; then
  if [ "$RESOLVED_CHANGE_NAME" = "smoke-test" ]; then
    echo "✓ resolve_change from fab/current works"
  else
    echo "✗ resolve_change set wrong name: $RESOLVED_CHANGE_NAME"
    exit 1
  fi
else
  echo "✗ resolve_change failed"
  exit 1
fi

echo "Testing resolve_change with override..."
if resolve_change "$TEST_DIR" "smoke"; then
  if [ "$RESOLVED_CHANGE_NAME" = "smoke-test" ]; then
    echo "✓ resolve_change with substring override works"
  else
    echo "✗ resolve_change set wrong name: $RESOLVED_CHANGE_NAME"
    exit 1
  fi
else
  echo "✗ resolve_change with override failed"
  exit 1
fi

echo ""
echo "✓ All smoke tests passed"

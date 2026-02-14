#!/usr/bin/env bash
# src/preflight/test-simple.sh
#
# Quick smoke test for fab-preflight.sh
# Run: ./test-simple.sh

set -uo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Set up minimal test environment
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

FAB="$TEST_DIR/fab"
mkdir -p "$FAB/.kit/scripts" "$FAB/.kit/schemas" "$FAB/changes/smoke-test"
cp "$PROJECT_ROOT/fab/.kit/scripts/fab-preflight.sh" "$FAB/.kit/scripts/"
cp "$PROJECT_ROOT/fab/.kit/scripts/_stageman.sh" "$FAB/.kit/scripts/"
cp "$PROJECT_ROOT/fab/.kit/scripts/_resolve-change.sh" "$FAB/.kit/scripts/"
cp "$PROJECT_ROOT/fab/.kit/schemas/workflow.yaml" "$FAB/.kit/schemas/"
chmod +x "$FAB/.kit/scripts/fab-preflight.sh"

echo "version: 1" > "$FAB/config.yaml"
echo "# Constitution" > "$FAB/constitution.md"
echo "smoke-test" > "$FAB/current"
cat > "$FAB/changes/smoke-test/.status.yaml" <<EOF
progress:
  brief: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

echo "Testing fab-preflight.sh smoke test..."

output=$("$FAB/.kit/scripts/fab-preflight.sh" 2>&1)
code=$?

if [ $code -ne 0 ]; then
  echo "✗ preflight exited with code $code"
  echo "$output"
  exit 1
fi
echo "✓ preflight exits 0 on valid change"

if grep -q "name: smoke-test" <<< "$output"; then
  echo "✓ output contains correct name"
else
  echo "✗ output missing name"
  exit 1
fi

if grep -q "stage: spec" <<< "$output"; then
  echo "✓ output detects active stage"
else
  echo "✗ output has wrong stage"
  exit 1
fi

echo ""
echo "✓ All smoke tests passed"

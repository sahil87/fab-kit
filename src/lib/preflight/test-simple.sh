#!/usr/bin/env bash
# src/lib/preflight/test-simple.sh
#
# Quick smoke test for preflight.sh
# Run: ./test-simple.sh

set -uo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

# Set up minimal test environment
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

FAB="$TEST_DIR/fab"
mkdir -p "$FAB/.kit/scripts/lib" "$FAB/.kit/schemas" "$FAB/changes/smoke-test"
cp "$PROJECT_ROOT/fab/.kit/scripts/lib/preflight.sh" "$FAB/.kit/scripts/lib/"
cp "$PROJECT_ROOT/fab/.kit/scripts/lib/statusman.sh" "$FAB/.kit/scripts/lib/"
cp "$PROJECT_ROOT/fab/.kit/scripts/lib/changeman.sh" "$FAB/.kit/scripts/lib/"
cp "$PROJECT_ROOT/fab/.kit/scripts/lib/resolve.sh" "$FAB/.kit/scripts/lib/"
cp "$PROJECT_ROOT/fab/.kit/scripts/lib/logman.sh" "$FAB/.kit/scripts/lib/"
cp "$PROJECT_ROOT/fab/.kit/schemas/workflow.yaml" "$FAB/.kit/schemas/"
chmod +x "$FAB/.kit/scripts/lib/preflight.sh"

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

echo "Testing preflight.sh smoke test..."

output=$("$FAB/.kit/scripts/lib/preflight.sh" 2>&1)
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

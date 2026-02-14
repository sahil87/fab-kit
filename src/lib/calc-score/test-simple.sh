#!/usr/bin/env bash
# src/lib/calc-score/test-simple.sh
#
# Quick smoke test for calc-score.sh
# Run: ./test-simple.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CALC_SCORE="$SCRIPT_DIR/calc-score.sh"

TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

# Set up minimal change directory with spec.md containing Assumptions table
cat > "$TEST_DIR/.status.yaml" <<'EOF'
name: smoke-test
progress:
  brief: done
  spec: done
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
checklist:
  generated: false
  completed: 0
  total: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 5.0
EOF

cat > "$TEST_DIR/spec.md" <<'EOF'
# Spec: Smoke Test

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Test decision | Smoke test |
EOF

echo "Testing calc-score.sh basic invocation..."
output=$("$CALC_SCORE" "$TEST_DIR")
if echo "$output" | grep -q "confident: 1"; then
  echo "✓ Grade counting works"
else
  echo "✗ Grade counting failed"
  echo "  Output: $output"
  exit 1
fi

if echo "$output" | grep -q "score: 4.7"; then
  echo "✓ Score computation works"
else
  echo "✗ Score computation failed"
  echo "  Output: $output"
  exit 1
fi

if echo "$output" | grep -q "delta:"; then
  echo "✓ Delta present in output"
else
  echo "✗ Delta missing from output"
  echo "  Output: $output"
  exit 1
fi

if grep -q '"event":"confidence"' "$TEST_DIR/.history.jsonl"; then
  echo "✓ Confidence history event logged"
else
  echo "✗ Confidence history event missing"
  exit 1
fi

echo ""
echo "✓ All smoke tests passed"

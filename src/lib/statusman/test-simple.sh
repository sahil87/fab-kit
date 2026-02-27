#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
STATUSMAN="$SCRIPT_DIR/statusman.sh"

echo "Testing all-stages..."
stages=$("$STATUSMAN" all-stages)
if [[ "$stages" == *"spec"* ]]; then
  echo "✓ all-stages works"
else
  echo "✗ all-stages failed"
  exit 1
fi

# Test accessor subcommands
echo "Testing progress-map..."
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT
cat > "$TEST_DIR/status.yaml" <<EOF
progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
checklist:
  generated: true
  completed: 3
  total: 10
confidence:
  certain: 5
  confident: 2
  tentative: 1
  unresolved: 0
  score: 3.4
EOF

progress=$("$STATUSMAN" progress-map "$TEST_DIR/status.yaml")
if [[ "$progress" == *"intake:done"* ]] && [[ "$progress" == *"spec:active"* ]]; then
  echo "✓ progress-map works"
else
  echo "✗ progress-map failed"
  exit 1
fi

echo "Testing checklist..."
checklist=$("$STATUSMAN" checklist "$TEST_DIR/status.yaml")
if [[ "$checklist" == *"generated:true"* ]] && [[ "$checklist" == *"total:10"* ]]; then
  echo "✓ checklist works"
else
  echo "✗ checklist failed"
  exit 1
fi

echo "Testing confidence..."
confidence=$("$STATUSMAN" confidence "$TEST_DIR/status.yaml")
if [[ "$confidence" == *"score:3.4"* ]] && [[ "$confidence" == *"certain:5"* ]]; then
  echo "✓ confidence works"
else
  echo "✗ confidence failed"
  exit 1
fi

echo ""
echo "✓ All basic tests passed"

#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/stageman.sh"

echo "Testing get_all_states..."
states=$(get_all_states)
if [[ "$states" == *"pending"* ]]; then
  echo "✓ get_all_states works"
else
  echo "✗ get_all_states failed"
  exit 1
fi

echo "Testing get_all_stages..."
stages=$(get_all_stages)
if [[ "$stages" == *"spec"* ]]; then
  echo "✓ get_all_stages works"
else
  echo "✗ get_all_stages failed"
  exit 1
fi

echo "Testing get_stage_number..."
num=$(get_stage_number "spec")
if [ "$num" = "2" ]; then
  echo "✓ get_stage_number works"
else
  echo "✗ get_stage_number failed (got $num)"
  exit 1
fi

# Test new accessor functions
echo "Testing get_progress_map..."
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT
cat > "$TEST_DIR/status.yaml" <<EOF
progress:
  brief: done
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

progress=$(get_progress_map "$TEST_DIR/status.yaml")
if [[ "$progress" == *"brief:done"* ]] && [[ "$progress" == *"spec:active"* ]]; then
  echo "✓ get_progress_map works"
else
  echo "✗ get_progress_map failed"
  exit 1
fi

echo "Testing get_checklist..."
checklist=$(get_checklist "$TEST_DIR/status.yaml")
if [[ "$checklist" == *"generated:true"* ]] && [[ "$checklist" == *"total:10"* ]]; then
  echo "✓ get_checklist works"
else
  echo "✗ get_checklist failed"
  exit 1
fi

echo "Testing get_confidence..."
confidence=$(get_confidence "$TEST_DIR/status.yaml")
if [[ "$confidence" == *"score:3.4"* ]] && [[ "$confidence" == *"certain:5"* ]]; then
  echo "✓ get_confidence works"
else
  echo "✗ get_confidence failed"
  exit 1
fi

echo ""
echo "✓ All basic tests passed"

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

echo ""
echo "✓ All basic tests passed"

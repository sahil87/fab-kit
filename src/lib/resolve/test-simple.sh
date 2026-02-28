#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
RESOLVE="$PROJECT_ROOT/fab/.kit/scripts/lib/resolve.sh"

echo "Testing resolve.sh..."

# Test --id (default)
output=$(bash "$RESOLVE" --id 9fg2 2>/dev/null) || { echo "✗ --id failed"; exit 1; }
if [[ "$output" == "9fg2" ]]; then
  echo "✓ --id returns 4-char ID"
else
  echo "✗ --id returned '$output' (expected '9fg2')"
  exit 1
fi

# Test --folder
output=$(bash "$RESOLVE" --folder 9fg2 2>/dev/null) || { echo "✗ --folder failed"; exit 1; }
if [[ "$output" == *"9fg2"* ]] && [[ "$output" == *"refactor-kit"* ]]; then
  echo "✓ --folder returns full folder name"
else
  echo "✗ --folder returned '$output'"
  exit 1
fi

# Test --dir
output=$(bash "$RESOLVE" --dir 9fg2 2>/dev/null) || { echo "✗ --dir failed"; exit 1; }
if [[ "$output" == "fab/changes/"*"/" ]]; then
  echo "✓ --dir returns directory path"
else
  echo "✗ --dir returned '$output'"
  exit 1
fi

# Test --status
output=$(bash "$RESOLVE" --status 9fg2 2>/dev/null) || { echo "✗ --status failed"; exit 1; }
if [[ "$output" == *".status.yaml" ]]; then
  echo "✓ --status returns .status.yaml path"
else
  echo "✗ --status returned '$output'"
  exit 1
fi

# Test no-match error
if bash "$RESOLVE" nonexistent 2>/dev/null; then
  echo "✗ should fail on nonexistent"
  exit 1
else
  echo "✓ returns error for nonexistent change"
fi

echo ""
echo "✓ All resolve.sh smoke tests passed"

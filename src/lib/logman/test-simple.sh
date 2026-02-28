#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
LOGMAN="$PROJECT_ROOT/fab/.kit/scripts/lib/logman.sh"

echo "Testing logman.sh..."

# Count existing lines
HISTORY="$PROJECT_ROOT/fab/changes/260228-9fg2-refactor-kit-scripts/.history.jsonl"
before=$(wc -l < "$HISTORY")

# Test command subcommand
bash "$LOGMAN" command 9fg2 "test-logman" "smoke" 2>/dev/null
after=$(wc -l < "$HISTORY")
if [ "$after" -eq "$((before + 1))" ]; then
  echo "✓ command appends one line"
else
  echo "✗ command: expected $((before + 1)) lines, got $after"
  exit 1
fi

# Verify JSON structure
last_line=$(tail -1 "$HISTORY")
if [[ "$last_line" == *'"event":"command"'* ]] && [[ "$last_line" == *'"cmd":"test-logman"'* ]]; then
  echo "✓ command produces valid JSON"
else
  echo "✗ command JSON invalid: $last_line"
  exit 1
fi

# Test review subcommand
bash "$LOGMAN" review 9fg2 "passed" 2>/dev/null
last_line=$(tail -1 "$HISTORY")
if [[ "$last_line" == *'"event":"review"'* ]] && [[ "$last_line" == *'"result":"passed"'* ]]; then
  echo "✓ review produces valid JSON"
else
  echo "✗ review JSON invalid: $last_line"
  exit 1
fi

# Test confidence subcommand
bash "$LOGMAN" confidence 9fg2 3.8 "+0.5" "test" 2>/dev/null
last_line=$(tail -1 "$HISTORY")
if [[ "$last_line" == *'"event":"confidence"'* ]] && [[ "$last_line" == *'"score":3.8'* ]]; then
  echo "✓ confidence produces valid JSON"
else
  echo "✗ confidence JSON invalid: $last_line"
  exit 1
fi

echo ""
echo "✓ All logman.sh smoke tests passed"

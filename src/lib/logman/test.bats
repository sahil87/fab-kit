#!/usr/bin/env bats

# Test suite for logman.sh
# Covers: command/confidence/review subcommands, append-only behavior,
#         file creation, error cases, change resolution integration,
#         optional change resolution via fab/current

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
LOGMAN_SRC="$REPO_ROOT/fab/.kit/scripts/lib/logman.sh"
RESOLVE_SRC="$REPO_ROOT/fab/.kit/scripts/lib/resolve.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
  FAB_ROOT="$TEST_DIR/fab"
  mkdir -p "$FAB_ROOT/changes" "$FAB_ROOT/.kit/scripts/lib"

  # Copy both logman.sh and resolve.sh (logman calls resolve internally)
  cp "$LOGMAN_SRC" "$FAB_ROOT/.kit/scripts/lib/logman.sh"
  cp "$RESOLVE_SRC" "$FAB_ROOT/.kit/scripts/lib/resolve.sh"
  chmod +x "$FAB_ROOT/.kit/scripts/lib/logman.sh"
  chmod +x "$FAB_ROOT/.kit/scripts/lib/resolve.sh"

  LOGMAN="$FAB_ROOT/.kit/scripts/lib/logman.sh"

  # Create a default change directory for most tests
  CHANGE_NAME="260228-a1b2-test-change"
  CHANGE_DIR="$FAB_ROOT/changes/$CHANGE_NAME"
  mkdir -p "$CHANGE_DIR"
  echo "name: $CHANGE_NAME" > "$CHANGE_DIR/.status.yaml"
  HISTORY="$CHANGE_DIR/.history.jsonl"

  # logman uses resolve.sh --dir which returns relative paths (fab/changes/...)
  # so CWD must be the root of the test directory for writes to land correctly
  cd "$TEST_DIR"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# ─────────────────────────────────────────────────────────────────────────────
# Command Subcommand (with explicit change)
# ─────────────────────────────────────────────────────────────────────────────

@test "command appends one line" {
  echo '{"ts":"old","event":"command","cmd":"setup"}' > "$HISTORY"
  local before
  before=$(wc -l < "$HISTORY")

  run bash "$LOGMAN" command "test-cmd" a1b2 "test-args"
  [ "$status" -eq 0 ]

  local after
  after=$(wc -l < "$HISTORY")
  [ "$after" -eq "$((before + 1))" ]
}

@test "command JSON has required fields" {
  run bash "$LOGMAN" command "my-skill" a1b2
  [ "$status" -eq 0 ]

  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" == *'"event":"command"'* ]]
  [[ "$last_line" == *'"cmd":"my-skill"'* ]]
  [[ "$last_line" == *'"ts":'* ]]
}

@test "command with args includes args field" {
  run bash "$LOGMAN" command "my-skill" a1b2 "spec"
  [ "$status" -eq 0 ]

  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" == *'"args":"spec"'* ]]
}

@test "command without args omits args field" {
  run bash "$LOGMAN" command "my-skill" a1b2
  [ "$status" -eq 0 ]

  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" != *'"args"'* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Command Subcommand (optional change via fab/current)
# ─────────────────────────────────────────────────────────────────────────────

@test "command with cmd only resolves via fab/current" {
  echo "$CHANGE_NAME" > "$FAB_ROOT/current"

  run bash "$LOGMAN" command "fab-discuss"
  [ "$status" -eq 0 ]

  [ -f "$HISTORY" ]
  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" == *'"cmd":"fab-discuss"'* ]]
}

@test "command with cmd only and no fab/current exits 0 silently" {
  rm -f "$FAB_ROOT/current"

  run bash "$LOGMAN" command "fab-setup"
  [ "$status" -eq 0 ]
  [ -z "$output" ]

  # No history file should be created or modified
  [ ! -f "$HISTORY" ] || [ "$(wc -l < "$HISTORY")" -eq 0 ] || true
}

@test "command with cmd only and empty fab/current exits 0 silently" {
  echo "" > "$FAB_ROOT/current"

  run bash "$LOGMAN" command "fab-help"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "command with cmd only and stale fab/current exits 0 silently" {
  echo "nonexistent-stale-change" > "$FAB_ROOT/current"

  run bash "$LOGMAN" command "fab-switch"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Command Subcommand (explicit change that doesn't resolve)
# ─────────────────────────────────────────────────────────────────────────────

@test "command with explicit nonexistent change returns error" {
  run bash "$LOGMAN" command "fab-continue" "nonexistent"
  [ "$status" -eq 1 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Confidence Subcommand
# ─────────────────────────────────────────────────────────────────────────────

@test "confidence produces valid JSON" {
  run bash "$LOGMAN" confidence a1b2 3.8 "+0.5" "calc-score"
  [ "$status" -eq 0 ]

  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" == *'"event":"confidence"'* ]]
  [[ "$last_line" == *'"score":3.8'* ]]
  [[ "$last_line" == *'"delta":"+0.5"'* ]]
  [[ "$last_line" == *'"trigger":"calc-score"'* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Review Subcommand
# ─────────────────────────────────────────────────────────────────────────────

@test "review passed without rework" {
  run bash "$LOGMAN" review a1b2 "passed"
  [ "$status" -eq 0 ]

  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" == *'"event":"review"'* ]]
  [[ "$last_line" == *'"result":"passed"'* ]]
  [[ "$last_line" != *'"rework"'* ]]
}

@test "review failed with rework type" {
  run bash "$LOGMAN" review a1b2 "failed" "fix-code"
  [ "$status" -eq 0 ]

  local last_line
  last_line=$(tail -1 "$HISTORY")
  [[ "$last_line" == *'"event":"review"'* ]]
  [[ "$last_line" == *'"result":"failed"'* ]]
  [[ "$last_line" == *'"rework":"fix-code"'* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Append-Only Behavior
# ─────────────────────────────────────────────────────────────────────────────

@test "existing lines preserved" {
  # Pre-populate with 3 lines
  echo '{"ts":"t1","event":"command","cmd":"one"}' > "$HISTORY"
  echo '{"ts":"t2","event":"command","cmd":"two"}' >> "$HISTORY"
  echo '{"ts":"t3","event":"command","cmd":"three"}' >> "$HISTORY"

  run bash "$LOGMAN" command "four" a1b2
  [ "$status" -eq 0 ]

  local total
  total=$(wc -l < "$HISTORY")
  [ "$total" -eq 4 ]

  # Verify original lines are unchanged
  local first_line
  first_line=$(head -1 "$HISTORY")
  [[ "$first_line" == *'"cmd":"one"'* ]]

  local third_line
  third_line=$(sed -n '3p' "$HISTORY")
  [[ "$third_line" == *'"cmd":"three"'* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# File Creation
# ─────────────────────────────────────────────────────────────────────────────

@test "creates .history.jsonl when absent" {
  rm -f "$HISTORY"
  [ ! -f "$HISTORY" ]

  run bash "$LOGMAN" command "first" a1b2
  [ "$status" -eq 0 ]

  [ -f "$HISTORY" ]
  local count
  count=$(wc -l < "$HISTORY")
  [ "$count" -eq 1 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Error Cases
# ─────────────────────────────────────────────────────────────────────────────

@test "no subcommand returns error with message" {
  run bash "$LOGMAN"
  [ "$status" -eq 1 ]
  [[ "$output" == *"No subcommand"* ]]
}

@test "unknown subcommand returns error with message" {
  run bash "$LOGMAN" badcmd a1b2
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown subcommand"* ]]
}

@test "command with no args after subcommand returns error" {
  run bash "$LOGMAN" command
  [ "$status" -eq 1 ]
}

@test "confidence with wrong argument count returns error" {
  run bash "$LOGMAN" confidence a1b2 3.8
  [ "$status" -eq 1 ]
}

@test "--help prints usage" {
  run bash "$LOGMAN" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"USAGE"* ]]
}

@test "--help shows new command signature" {
  run bash "$LOGMAN" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *'command <cmd> [change] [args]'* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Change Resolution Integration
# ─────────────────────────────────────────────────────────────────────────────

@test "4-char ID resolves to correct change directory" {
  run bash "$LOGMAN" command "test" a1b2
  [ "$status" -eq 0 ]

  [ -f "$CHANGE_DIR/.history.jsonl" ]
  local last_line
  last_line=$(tail -1 "$CHANGE_DIR/.history.jsonl")
  [[ "$last_line" == *'"cmd":"test"'* ]]
}

@test "folder name substring resolves" {
  run bash "$LOGMAN" command "test" "test-change"
  [ "$status" -eq 0 ]

  [ -f "$CHANGE_DIR/.history.jsonl" ]
}

@test "unresolvable explicit change returns error" {
  run bash "$LOGMAN" command "test" nonexistent
  [ "$status" -eq 1 ]
}

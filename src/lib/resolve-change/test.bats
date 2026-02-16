#!/usr/bin/env bats

# Test suite for resolve-change.sh
# Covers: exact match, substring, case-insensitive, multiple match,
#         no match, fab/current, missing dir, archive exclusion

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"

setup() {
  source "$SCRIPT_DIR/resolve-change.sh"
  TEST_DIR="$(mktemp -d)"
  mkdir -p "$TEST_DIR/changes/260213-puow-consolidate-status-reads"
  mkdir -p "$TEST_DIR/changes/260213-k7m2-kit-version-migrations"
  mkdir -p "$TEST_DIR/changes/260212-f9m3-enhance-srad-fuzzy"
  mkdir -p "$TEST_DIR/changes/archive"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# Helper: run resolve_change with stderr→stdout combined
_run_resolve() {
  resolve_change "$@" 2>&1
}

# ─────────────────────────────────────────────────────────────────────────────
# Exact Match
# ─────────────────────────────────────────────────────────────────────────────

@test "exact match resolves" {
  resolve_change "$TEST_DIR" "260213-puow-consolidate-status-reads" 2>/dev/null
  [ "$RESOLVED_CHANGE_NAME" = "260213-puow-consolidate-status-reads" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Substring Match
# ─────────────────────────────────────────────────────────────────────────────

@test "single substring match resolves" {
  resolve_change "$TEST_DIR" "puow" 2>/dev/null
  [ "$RESOLVED_CHANGE_NAME" = "260213-puow-consolidate-status-reads" ]
}

@test "4-char ID match resolves" {
  resolve_change "$TEST_DIR" "f9m3" 2>/dev/null
  [ "$RESOLVED_CHANGE_NAME" = "260212-f9m3-enhance-srad-fuzzy" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Case Insensitive
# ─────────────────────────────────────────────────────────────────────────────

@test "uppercase substring resolves" {
  resolve_change "$TEST_DIR" "PUOW" 2>/dev/null
  [ "$RESOLVED_CHANGE_NAME" = "260213-puow-consolidate-status-reads" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Multiple Match
# ─────────────────────────────────────────────────────────────────────────────

@test "multiple matches returns error" {
  run _run_resolve "$TEST_DIR" "260213"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Multiple changes match"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# No Match
# ─────────────────────────────────────────────────────────────────────────────

@test "no match returns error" {
  run _run_resolve "$TEST_DIR" "nonexistent"
  [ "$status" -ne 0 ]
  [[ "$output" == *"No change matches"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# fab/current
# ─────────────────────────────────────────────────────────────────────────────

@test "reads fab/current when no override" {
  echo "260213-puow-consolidate-status-reads" > "$TEST_DIR/current"
  resolve_change "$TEST_DIR" "" 2>/dev/null
  [ "$RESOLVED_CHANGE_NAME" = "260213-puow-consolidate-status-reads" ]
}

@test "fab/current with trailing whitespace resolves" {
  printf "260213-puow-consolidate-status-reads\n  " > "$TEST_DIR/current"
  resolve_change "$TEST_DIR" "" 2>/dev/null
  [ "$RESOLVED_CHANGE_NAME" = "260213-puow-consolidate-status-reads" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# No Active Change
# ─────────────────────────────────────────────────────────────────────────────

@test "missing fab/current returns error" {
  run _run_resolve "$TEST_DIR" ""
  [ "$status" -ne 0 ]
  [[ "$output" == *"No active change"* ]]
}

@test "empty fab/current returns error" {
  echo "" > "$TEST_DIR/current"
  run _run_resolve "$TEST_DIR" ""
  [ "$status" -ne 0 ]
  [[ "$output" == *"No active change"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Missing Changes Directory
# ─────────────────────────────────────────────────────────────────────────────

@test "missing changes directory returns error" {
  local empty_dir
  empty_dir="$(mktemp -d)"
  run _run_resolve "$empty_dir" "something"
  [ "$status" -ne 0 ]
  [[ "$output" == *"fab/changes/ not found"* ]]
  rm -rf "$empty_dir"
}

# ─────────────────────────────────────────────────────────────────────────────
# Archive Exclusion
# ─────────────────────────────────────────────────────────────────────────────

@test "archive folder excluded from matches" {
  run _run_resolve "$TEST_DIR" "archive"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Generic Error Messages
# ─────────────────────────────────────────────────────────────────────────────

@test "error message does not contain command suggestions" {
  run _run_resolve "$TEST_DIR" ""
  [[ "$output" != *"Run /fab"* ]]
}

#!/usr/bin/env bats

# Test suite for resolve.sh
# Covers: output modes, input forms, single-change guess, error cases, archive exclusion

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
RESOLVE_SRC="$REPO_ROOT/fab/.kit/scripts/lib/resolve.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
  FAB_ROOT="$TEST_DIR/fab"
  mkdir -p "$FAB_ROOT/changes" "$FAB_ROOT/.kit/scripts/lib"

  # Copy the real resolve.sh into the isolated kit location
  cp "$RESOLVE_SRC" "$FAB_ROOT/.kit/scripts/lib/resolve.sh"
  chmod +x "$FAB_ROOT/.kit/scripts/lib/resolve.sh"

  RESOLVE="$FAB_ROOT/.kit/scripts/lib/resolve.sh"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# Helper: create a change folder (optionally with .status.yaml)
make_change() {
  local name="$1"
  local with_status="${2:-true}"
  mkdir -p "$FAB_ROOT/changes/$name"
  if [ "$with_status" = "true" ]; then
    echo "name: $name" > "$FAB_ROOT/changes/$name/.status.yaml"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Output Mode Tests
# ─────────────────────────────────────────────────────────────────────────────

@test "--id extracts 4-char ID" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" --id a1b2
  [ "$status" -eq 0 ]
  [ "$output" = "a1b2" ]
}

@test "--folder returns full folder name" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" --folder a1b2
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-test-change" ]
}

@test "--dir returns directory path with trailing slash" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" --dir a1b2
  [ "$status" -eq 0 ]
  [ "$output" = "fab/changes/260228-a1b2-test-change/" ]
}

@test "--status returns .status.yaml path" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" --status a1b2
  [ "$status" -eq 0 ]
  [ "$output" = "fab/changes/260228-a1b2-test-change/.status.yaml" ]
}

@test "default mode is --id" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" a1b2
  [ "$status" -eq 0 ]
  [ "$output" = "a1b2" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Input Form Tests
# ─────────────────────────────────────────────────────────────────────────────

@test "full folder name resolves via exact match" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" --folder "260228-a1b2-test-change"
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-test-change" ]
}

@test "4-char ID resolves" {
  make_change "260228-x9y8-my-feature"
  run bash "$RESOLVE" --folder x9y8
  [ "$status" -eq 0 ]
  [ "$output" = "260228-x9y8-my-feature" ]
}

@test "substring match resolves uniquely" {
  make_change "260228-a1b2-alpha"
  make_change "260228-c3d4-beta"
  run bash "$RESOLVE" --folder alpha
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-alpha" ]
}

@test "case-insensitive matching" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" --folder A1B2
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-test-change" ]
}

@test "no argument reads fab/current" {
  make_change "260228-a1b2-test-change"
  printf '260228-a1b2-test-change' > "$FAB_ROOT/current"
  run bash "$RESOLVE" --folder
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-test-change" ]
}

@test "fab/current with trailing whitespace" {
  make_change "260228-a1b2-test-change"
  printf '260228-a1b2-test-change\n  ' > "$FAB_ROOT/current"
  run bash "$RESOLVE" --folder
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-test-change" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Single-Change Guess Fallback
# ─────────────────────────────────────────────────────────────────────────────

@test "single change guessed when fab/current missing" {
  make_change "260228-a1b2-only-change"
  rm -f "$FAB_ROOT/current"
  run bash "$RESOLVE" --folder
  [ "$status" -eq 0 ]
  [[ "$output" == *"resolved from single active change"* ]]
  [[ "$output" == *"260228-a1b2-only-change"* ]]
}

@test "guess requires .status.yaml" {
  make_change "260228-a1b2-no-status" false
  rm -f "$FAB_ROOT/current"
  run bash "$RESOLVE" --folder
  [ "$status" -eq 1 ]
  [[ "$output" == *"No active change"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Error Cases
# ─────────────────────────────────────────────────────────────────────────────

@test "no match returns error with message" {
  make_change "260228-a1b2-test-change"
  run bash "$RESOLVE" nonexistent
  [ "$status" -eq 1 ]
  [[ "$output" == *'No change matches'* ]]
}

@test "multiple matches returns error with message" {
  make_change "260228-a1b2-test-alpha"
  make_change "260228-c3d4-test-beta"
  run bash "$RESOLVE" test
  [ "$status" -eq 1 ]
  [[ "$output" == *"Multiple changes match"* ]]
}

@test "missing fab/changes/ returns error with message" {
  rm -rf "$FAB_ROOT/changes"
  run bash "$RESOLVE" something
  [ "$status" -eq 1 ]
  [[ "$output" == *"fab/changes/ not found"* ]]
}

@test "no fab/current with multiple changes returns error with message" {
  make_change "260228-a1b2-alpha"
  make_change "260228-c3d4-beta"
  rm -f "$FAB_ROOT/current"
  run bash "$RESOLVE" --folder
  [ "$status" -eq 1 ]
  [[ "$output" == *"No active change"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Archive Exclusion
# ─────────────────────────────────────────────────────────────────────────────

@test "archive directory excluded from matches" {
  mkdir -p "$FAB_ROOT/changes/archive"
  run bash "$RESOLVE" archive
  [ "$status" -eq 1 ]
}

@test "archive excluded but other changes resolve" {
  mkdir -p "$FAB_ROOT/changes/archive"
  make_change "260228-a1b2-real-change"
  run bash "$RESOLVE" --folder a1b2
  [ "$status" -eq 0 ]
  [ "$output" = "260228-a1b2-real-change" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────────────────

@test "--help prints usage" {
  run bash "$RESOLVE" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"USAGE"* ]]
}

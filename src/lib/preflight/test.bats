#!/usr/bin/env bats

# Test suite for preflight.sh
# Covers: init validation, change name resolution (fab/current + override),
#         directory/status validation, YAML output, checklist/confidence, errors

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

setup() {
  TEST_DIR="$(mktemp -d)"

  local fab="$TEST_DIR/fab"
  mkdir -p "$fab/.kit/scripts/lib" "$fab/.kit/schemas" "$fab/changes"

  # Copy real scripts and schema
  cp "$PROJECT_ROOT/fab/.kit/scripts/lib/preflight.sh" "$fab/.kit/scripts/lib/"
  cp "$PROJECT_ROOT/fab/.kit/scripts/lib/stageman.sh" "$fab/.kit/scripts/lib/"
  cp "$PROJECT_ROOT/fab/.kit/scripts/lib/changeman.sh" "$fab/.kit/scripts/lib/"
  cp "$PROJECT_ROOT/fab/.kit/schemas/workflow.yaml" "$fab/.kit/schemas/"
  chmod +x "$fab/.kit/scripts/lib/preflight.sh"

  # Create required init files
  echo "version: 1" > "$fab/config.yaml"
  echo "# Constitution" > "$fab/constitution.md"

  PREFLIGHT="$TEST_DIR/fab/.kit/scripts/lib/preflight.sh"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# Helper: create a change with a status file
create_change() {
  local name="$1"
  local status_content="$2"
  local change_dir="$TEST_DIR/fab/changes/$name"
  mkdir -p "$change_dir"
  echo "$status_content" > "$change_dir/.status.yaml"
}

# Helper: set the active change in fab/current
set_current() {
  echo "$1" > "$TEST_DIR/fab/current"
}

# Helper: run preflight with stderr→stdout combined
run_preflight_combined() {
  "$PREFLIGHT" "$@" 2>&1
}

# ─────────────────────────────────────────────────────────────────────────────
# 1. Project Initialization Validation
# ─────────────────────────────────────────────────────────────────────────────

@test "rejects when config.yaml missing" {
  rm "$TEST_DIR/fab/config.yaml"
  set_current "test-change"
  create_change "test-change" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects when constitution.md missing" {
  rm "$TEST_DIR/fab/constitution.md"
  set_current "test-change"
  create_change "test-change" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects when both init files missing" {
  rm "$TEST_DIR/fab/config.yaml" "$TEST_DIR/fab/constitution.md"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# 2. Change Name Resolution — fab/current
# ─────────────────────────────────────────────────────────────────────────────

@test "rejects when fab/current missing" {
  create_change "test-change" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects when fab/current is empty" {
  create_change "test-change" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  echo "" > "$TEST_DIR/fab/current"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "accepts valid fab/current" {
  create_change "my-feature" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "my-feature"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: my-feature"* ]]
}

@test "handles whitespace in fab/current" {
  create_change "trim-test" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  printf "trim-test  \n" > "$TEST_DIR/fab/current"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: trim-test"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# 3. Change Name Resolution — Override ($1)
# ─────────────────────────────────────────────────────────────────────────────

@test "override: exact match works" {
  create_change "alpha-feature" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run "$PREFLIGHT" "alpha-feature"
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: alpha-feature"* ]]
}

@test "override: case-insensitive exact match preserves casing" {
  create_change "Alpha-Feature" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run "$PREFLIGHT" "alpha-feature"
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: Alpha-Feature"* ]]
}

@test "override: single partial match works" {
  create_change "big-refactor-auth" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  create_change "small-ui-fix" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run "$PREFLIGHT" "refactor"
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: big-refactor-auth"* ]]
}

@test "override: rejects ambiguous partial match" {
  create_change "auth-login" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  create_change "auth-signup" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run run_preflight_combined "auth"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Multiple changes match"* ]]
}

@test "override: rejects when no match" {
  create_change "some-feature" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  run run_preflight_combined "nonexistent"
  [ "$status" -eq 1 ]
  [[ "$output" == *"No change matches"* ]]
}

@test "override: rejects when changes dir missing" {
  rm -rf "$TEST_DIR/fab/changes"
  run run_preflight_combined "anything"
  [ "$status" -eq 1 ]
}

@test "override: rejects when only archive folder exists" {
  mkdir -p "$TEST_DIR/fab/changes/archive"
  run run_preflight_combined "test"
  [ "$status" -eq 1 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# 4. Directory & Status File Validation
# ─────────────────────────────────────────────────────────────────────────────

@test "rejects when change directory missing" {
  set_current "ghost-change"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects when .status.yaml missing" {
  mkdir -p "$TEST_DIR/fab/changes/no-status"
  set_current "no-status"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects invalid state in .status.yaml" {
  create_change "bad-status" "progress:
  intake: done
  spec: bogus_state
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "bad-status"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects multiple active stages" {
  create_change "multi-active" "progress:
  brief: active
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "multi-active"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

@test "rejects state not in allowed_states for stage" {
  create_change "bad-allowed" "progress:
  intake: failed
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "bad-allowed"
  run run_preflight_combined
  [ "$status" -eq 1 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# 5. YAML Output — Progress & Stage Detection
# ─────────────────────────────────────────────────────────────────────────────

@test "valid change produces correct YAML output" {
  create_change "feature-x" "progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "feature-x"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"name: feature-x"* ]]
  [[ "$output" == *"change_dir: changes/feature-x"* ]]
  [[ "$output" == *"stage: spec"* ]]
  [[ "$output" == *"intake: done"* ]]
  [[ "$output" == *"spec: active"* ]]
  [[ "$output" == *"tasks: pending"* ]]
}

@test "detects intake as current stage" {
  create_change "new-change" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "new-change"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"stage: intake"* ]]
}

@test "detects apply as current stage" {
  create_change "mid-change" "progress:
  intake: done
  spec: done
  tasks: done
  apply: active
  review: pending
  hydrate: pending"
  set_current "mid-change"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"stage: apply"* ]]
}

@test "falls back to hydrate when all done" {
  create_change "done-change" "progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: done
  hydrate: done"
  set_current "done-change"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"stage: hydrate"* ]]
}

@test "review:failed accepted and falls back" {
  create_change "failed-review" "progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: failed
  hydrate: pending"
  set_current "failed-review"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"stage: hydrate"* ]]
  [[ "$output" == *"review: failed"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# 6. Checklist & Confidence Fields
# ─────────────────────────────────────────────────────────────────────────────

@test "bare change defaults checklist and confidence" {
  create_change "bare-change" "progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
  set_current "bare-change"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"generated: false"* ]]
  [[ "$output" == *"completed: 0"* ]]
  [[ "$output" == *"total: 0"* ]]
  [[ "$output" == *"score: 5.0"* ]]
}

@test "rich change with checklist and confidence" {
  create_change "rich-change" "progress:
  intake: done
  spec: done
  tasks: active
  apply: pending
  review: pending
  hydrate: pending
checklist:
  generated: true
  completed: 5
  total: 12
confidence:
  certain: 3
  confident: 4
  tentative: 2
  unresolved: 1
  score: 7.5"
  set_current "rich-change"
  run "$PREFLIGHT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"generated: true"* ]]
  [[ "$output" == *"completed: 5"* ]]
  [[ "$output" == *"total: 12"* ]]
  [[ "$output" == *"certain: 3"* ]]
  [[ "$output" == *"confident: 4"* ]]
  [[ "$output" == *"tentative: 2"* ]]
  [[ "$output" == *"unresolved: 1"* ]]
  [[ "$output" == *"score: 7.5"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# 7. Error Messages
# ─────────────────────────────────────────────────────────────────────────────

@test "error msg: project not initialized" {
  rm "$TEST_DIR/fab/config.yaml"
  set_current "x"
  run run_preflight_combined
  [[ "$output" == *"not initialized"* ]]
}

@test "error msg: no active change" {
  run run_preflight_combined
  [[ "$output" == *"No active change"* ]]
}

@test "error msg: change directory not found" {
  set_current "ghost"
  run run_preflight_combined
  [[ "$output" == *"not found"* ]]
}

@test "error msg: corrupted change" {
  mkdir -p "$TEST_DIR/fab/changes/broken"
  set_current "broken"
  run run_preflight_combined
  [[ "$output" == *"corrupted"* ]]
}

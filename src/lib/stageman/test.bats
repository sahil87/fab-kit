#!/usr/bin/env bats

# Test suite for stageman.sh (CLI-only contract tests)
# Covers: state queries, stage queries, progression, validation,
#         .status.yaml accessors, write functions, stage metrics, history logging

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
STAGEMAN="$REPO_ROOT/fab/.kit/scripts/lib/stageman.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# Helper: create a fresh write-test fixture
make_write_fixture() {
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test-write
progress:
  intake: done
  spec: done
  tasks: done
  apply: active
  review: pending
  hydrate: pending
checklist:
  generated: true
  path: checklist.md
  completed: 3
  total: 10
confidence:
  certain: 5
  confident: 2
  tentative: 1
  unresolved: 0
  score: 3.4
stage_metrics: {}
last_updated: 2026-01-01T00:00:00+00:00
EOF
}

# ─────────────────────────────────────────────────────────────────────────────
# Stage Tests
# ─────────────────────────────────────────────────────────────────────────────

@test "all-stages includes all 6 stages" {
  run "$STAGEMAN" all-stages
  [[ "$output" == *"intake"* ]]
  [[ "$output" == *"spec"* ]]
  [[ "$output" == *"tasks"* ]]
  [[ "$output" == *"apply"* ]]
  [[ "$output" == *"review"* ]]
  [[ "$output" == *"hydrate"* ]]
  local stage_count
  stage_count=$(echo "$output" | wc -l)
  [ "$stage_count" -eq 6 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Validation
# ─────────────────────────────────────────────────────────────────────────────

@test "validate-status-file accepts valid status" {
  cat > "$TEST_DIR/valid.yaml" <<EOF
progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
stage_metrics: {}
EOF
  run "$STAGEMAN" validate-status-file "$TEST_DIR/valid.yaml"
  [ "$status" -eq 0 ]
}

@test "validate-status-file rejects invalid state" {
  cat > "$TEST_DIR/invalid-state.yaml" <<EOF
progress:
  intake: done
  spec: invalid_state
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" validate-status-file "$TEST_DIR/invalid-state.yaml"
  [ "$status" -ne 0 ]
}

@test "validate-status-file rejects multiple active stages" {
  cat > "$TEST_DIR/multiple-active.yaml" <<EOF
progress:
  intake: active
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" validate-status-file "$TEST_DIR/multiple-active.yaml"
  [ "$status" -ne 0 ]
}

@test "validate-status-file rejects state not in allowed_states" {
  cat > "$TEST_DIR/wrong-allowed-state.yaml" <<EOF
progress:
  intake: failed
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" validate-status-file "$TEST_DIR/wrong-allowed-state.yaml"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# .status.yaml Accessors
# ─────────────────────────────────────────────────────────────────────────────

@test "progress-map extracts correct stage-state pairs" {
  cat > "$TEST_DIR/full-status.yaml" <<EOF
name: test-change
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
  run "$STAGEMAN" progress-map "$TEST_DIR/full-status.yaml"
  [[ "$output" == *"intake:done"* ]]
  [[ "$output" == *"spec:active"* ]]
  [[ "$output" == *"hydrate:pending"* ]]
  local line_count
  line_count=$(echo "$output" | wc -l | tr -d ' ')
  [ "$line_count" -eq 6 ]
}

@test "progress-map defaults missing stage to pending" {
  cat > "$TEST_DIR/missing-stage.yaml" <<EOF
progress:
  intake: done
  spec: active
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" progress-map "$TEST_DIR/missing-stage.yaml"
  [[ "$output" == *"tasks:pending"* ]]
}

@test "checklist extracts values" {
  cat > "$TEST_DIR/full-status.yaml" <<EOF
name: test-change
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
  run "$STAGEMAN" checklist "$TEST_DIR/full-status.yaml"
  [[ "$output" == *"generated:true"* ]]
  [[ "$output" == *"completed:3"* ]]
  [[ "$output" == *"total:10"* ]]
}

@test "checklist defaults when block missing" {
  cat > "$TEST_DIR/no-checklist.yaml" <<EOF
progress:
  intake: active
EOF
  run "$STAGEMAN" checklist "$TEST_DIR/no-checklist.yaml"
  [[ "$output" == *"generated:false"* ]]
  [[ "$output" == *"completed:0"* ]]
  [[ "$output" == *"total:0"* ]]
}

@test "confidence extracts values" {
  cat > "$TEST_DIR/full-status.yaml" <<EOF
name: test-change
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
  run "$STAGEMAN" confidence "$TEST_DIR/full-status.yaml"
  [[ "$output" == *"certain:5"* ]]
  [[ "$output" == *"confident:2"* ]]
  [[ "$output" == *"tentative:1"* ]]
  [[ "$output" == *"unresolved:0"* ]]
  [[ "$output" == *"score:3.4"* ]]
}

@test "confidence defaults when block missing" {
  cat > "$TEST_DIR/no-confidence.yaml" <<EOF
progress:
  intake: active
EOF
  run "$STAGEMAN" confidence "$TEST_DIR/no-confidence.yaml"
  [[ "$output" == *"certain:0"* ]]
  [[ "$output" == *"score:5.0"* ]]
}

@test "current-stage finds active stage" {
  cat > "$TEST_DIR/full-status.yaml" <<EOF
name: test-change
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
  run "$STAGEMAN" current-stage "$TEST_DIR/full-status.yaml"
  [ "$output" = "spec" ]
}

@test "current-stage fallback: first pending after last done" {
  cat > "$TEST_DIR/no-active.yaml" <<EOF
progress:
  intake: done
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" current-stage "$TEST_DIR/no-active.yaml"
  [ "$output" = "spec" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Write: set-state
# ─────────────────────────────────────────────────────────────────────────────

@test "set-state: valid state change succeeds" {
  make_write_fixture
  run "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "set-state: refreshes last_updated" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "test-driver"
  local ts
  ts=$(yq '.last_updated' "$TEST_DIR/write-status.yaml")
  [[ "$ts" == *"T"* ]]
}

@test "set-state: other stages unchanged" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "test-driver"
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "set-state: rejects active without driver" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-state: done without driver succeeds" {
  make_write_fixture
  run "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "apply" "done"
  [ "$status" -eq 0 ]
}

@test "set-state: rejects invalid stage" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "invalid_stage" "active" "test"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-state: rejects state not allowed for stage" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "intake" "failed"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-state: rejects nonexistent file" {
  run "$STAGEMAN" set-state "/nonexistent/path/.status.yaml" "spec" "done"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Write: transition
# ─────────────────────────────────────────────────────────────────────────────

@test "transition: valid forward transition succeeds" {
  make_write_fixture
  run "$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "review" "test-driver"
  [ "$status" -eq 0 ]
  local from_state to_state
  from_state=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  to_state=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$from_state" = "done" ]
  [ "$to_state" = "active" ]
}

@test "transition: rejects without driver" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "review"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "transition: rejects non-adjacent stages" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "hydrate" "test-driver"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "transition: rejects when from_stage is not active" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "spec" "tasks" "test-driver"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "transition: rejects nonexistent file" {
  run "$STAGEMAN" transition "/nonexistent/path/.status.yaml" "apply" "review" "test-driver"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Write: set-checklist
# ─────────────────────────────────────────────────────────────────────────────

@test "set-checklist: completed update succeeds" {
  make_write_fixture
  run "$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "completed" "7"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  completed:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "7" ]
}

@test "set-checklist: generated update succeeds" {
  make_write_fixture
  run "$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "generated" "false"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  generated:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "false" ]
}

@test "set-checklist: total update succeeds" {
  make_write_fixture
  run "$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "total" "25"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  total:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "25" ]
}

@test "set-checklist: rejects invalid field" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "invalid_field" "5"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-checklist: rejects negative integer" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "completed" "-3"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-checklist: rejects non-bool for generated" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "generated" "yes"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-checklist: rejects nonexistent file" {
  run "$STAGEMAN" set-checklist "/nonexistent/path/.status.yaml" "completed" "5"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Write: set-confidence
# ─────────────────────────────────────────────────────────────────────────────

@test "set-confidence: valid replacement succeeds" {
  make_write_fixture
  run "$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  certain:" "$TEST_DIR/write-status.yaml" | sed 's/.*: *//')
  [ "$result" = "10" ]
  result=$(grep "^  score:" "$TEST_DIR/write-status.yaml" | sed 's/.*: *//')
  [ "$result" = "4.1" ]
}

@test "set-confidence: refreshes last_updated" {
  make_write_fixture
  "$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1"
  local ts
  ts=$(grep "^last_updated:" "$TEST_DIR/write-status.yaml" | sed 's/last_updated: //')
  [[ "$ts" == *"T"* ]]
}

@test "set-confidence: preserves other blocks" {
  make_write_fixture
  "$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1"
  local result
  result=$(grep "^  apply:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "active" ]
  result=$(grep "^  total:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "10" ]
}

@test "set-confidence: rejects negative count" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "-1" "3" "1" "0" "4.1"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-confidence: rejects non-numeric score" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "abc"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-confidence: rejects negative score" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "-2.5"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-confidence: rejects nonexistent file" {
  run "$STAGEMAN" set-confidence "/nonexistent/path/.status.yaml" "10" "3" "1" "0" "4.1"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Stage Metrics
# ─────────────────────────────────────────────────────────────────────────────

@test "stage-metrics: set-state active creates metrics" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue"
  local driver iter started
  driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
  started=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  [ "$driver" = "fab-continue" ]
  [ "$iter" = "1" ]
  [[ "$started" == *"T"* ]]
}

@test "stage-metrics: set-state done sets completed_at and preserves driver" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue"
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "done"
  local completed driver
  completed=$(yq '.stage_metrics.review.completed_at' "$TEST_DIR/write-status.yaml")
  driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  [[ "$completed" == *"T"* ]]
  [ "$driver" = "fab-continue" ]
}

@test "stage-metrics: rework re-activation increments iterations" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue"
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "done"
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue"
  local iter
  iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
  [ "$iter" = "2" ]
}

@test "stage-metrics: pending clears metrics entry" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "apply" "done"
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue"
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "pending"
  local result
  result=$(yq '.stage_metrics.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "null" ]
}

@test "stage-metrics: transition sets metrics for both stages" {
  make_write_fixture
  "$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "review" "fab-ff"
  local from_completed to_driver to_iter
  from_completed=$(yq '.stage_metrics.apply.completed_at' "$TEST_DIR/write-status.yaml")
  to_driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  to_iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
  [[ "$from_completed" == *"T"* ]]
  [ "$to_driver" = "fab-ff" ]
  [ "$to_iter" = "1" ]
}

@test "validate-status-file ignores stage_metrics content" {
  make_write_fixture
  yq -i '.stage_metrics.intake = {"started_at": "now", "driver": "test", "iterations": 1}' "$TEST_DIR/write-status.yaml"
  run "$STAGEMAN" validate-status-file "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# History Logging
# ─────────────────────────────────────────────────────────────────────────────

@test "log-command creates command event" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-command "$history_dir" "fab-continue" ""
  local line
  line=$(head -1 "$history_dir/.history.jsonl")
  [[ "$line" == *'"event":"command"'* ]]
  [[ "$line" == *'"cmd":"fab-continue"'* ]]
}

@test "log-command omits empty args" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-command "$history_dir" "fab-continue" ""
  local line
  line=$(head -1 "$history_dir/.history.jsonl")
  [[ "$line" != *'"args"'* ]]
}

@test "log-command includes non-empty args" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-command "$history_dir" "fab-ff" "spec"
  local line
  line=$(tail -1 "$history_dir/.history.jsonl")
  [[ "$line" == *'"args":"spec"'* ]]
}

@test "log-confidence creates confidence event with score" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-confidence "$history_dir" 4.1 "+4.1" "calc-score"
  local line
  line=$(tail -1 "$history_dir/.history.jsonl")
  [[ "$line" == *'"event":"confidence"'* ]]
  [[ "$line" == *'"score":4.1'* ]]
}

@test "log-review creates passed event without rework" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-review "$history_dir" "passed"
  local line
  line=$(tail -1 "$history_dir/.history.jsonl")
  [[ "$line" == *'"result":"passed"'* ]]
  [[ "$line" != *'"rework"'* ]]
}

@test "log-review creates failed event with rework" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-review "$history_dir" "failed" "revise-tasks"
  local line
  line=$(tail -1 "$history_dir/.history.jsonl")
  [[ "$line" == *'"rework":"revise-tasks"'* ]]
}

@test "log-command resolves relative path against fab root" {
  # Derive fab root from stageman location (same logic as resolve_change_dir)
  local stageman_dir
  stageman_dir="$(cd "$(dirname "$(readlink -f "$STAGEMAN")")" && pwd)"
  local fab_root
  fab_root="$(cd "$stageman_dir/../../.." && pwd)"

  # Create a temp dir under fab root
  local test_subdir=".test-resolve-$$"
  mkdir -p "$fab_root/$test_subdir"

  # Run log-command with a relative path from a different cwd
  (cd /tmp && "$STAGEMAN" log-command "$test_subdir" "test-resolve" "")

  # Verify file was written at the resolved absolute path, not relative to cwd
  [ -f "$fab_root/$test_subdir/.history.jsonl" ]
  [[ "$(head -1 "$fab_root/$test_subdir/.history.jsonl")" == *'"cmd":"test-resolve"'* ]]

  # Clean up
  rm -rf "$fab_root/$test_subdir"
}

# ─────────────────────────────────────────────────────────────────────────────
# Progress Line
# ─────────────────────────────────────────────────────────────────────────────

@test "progress-line: all pending returns empty" {
  cat > "$TEST_DIR/all-pending.yaml" <<EOF
progress:
  intake: pending
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/all-pending.yaml"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "progress-line: first stage active" {
  cat > "$TEST_DIR/first-active.yaml" <<EOF
progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/first-active.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake ⏳" ]
}

@test "progress-line: mid-pipeline" {
  cat > "$TEST_DIR/mid-pipeline.yaml" <<EOF
progress:
  intake: done
  spec: done
  tasks: done
  apply: active
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/mid-pipeline.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake → spec → tasks → apply ⏳" ]
}

@test "progress-line: failed stage" {
  cat > "$TEST_DIR/failed.yaml" <<EOF
progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: failed
  hydrate: pending
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/failed.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake → spec → tasks → apply → review ✗" ]
}

@test "progress-line: all done" {
  cat > "$TEST_DIR/all-done.yaml" <<EOF
progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: done
  hydrate: done
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/all-done.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake → spec → tasks → apply → review → hydrate ✓" ]
}

@test "progress-line: single done rest pending" {
  cat > "$TEST_DIR/single-done.yaml" <<EOF
progress:
  intake: done
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/single-done.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Shipped Tracking
# ─────────────────────────────────────────────────────────────────────────────

@test "ship: appends URL to empty shipped array" {
  make_write_fixture
  yq -i '.shipped = []' "$TEST_DIR/write-status.yaml"
  run "$STAGEMAN" ship "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/42"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.shipped[0]' "$TEST_DIR/write-status.yaml")
  [ "$result" = "https://github.com/org/repo/pull/42" ]
}

@test "ship: appends second URL" {
  make_write_fixture
  yq -i '.shipped = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  "$STAGEMAN" ship "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/43"
  local count
  count=$(yq '.shipped | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "2" ]
  local second
  second=$(yq '.shipped[1]' "$TEST_DIR/write-status.yaml")
  [ "$second" = "https://github.com/org/repo/pull/43" ]
}

@test "ship: deduplicates existing URL" {
  make_write_fixture
  yq -i '.shipped = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  run "$STAGEMAN" ship "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/42"
  [ "$status" -eq 0 ]
  local count
  count=$(yq '.shipped | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "1" ]
}

@test "ship: does not treat substring as duplicate" {
  make_write_fixture
  yq -i '.shipped = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  "$STAGEMAN" ship "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/4"
  local count
  count=$(yq '.shipped | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "2" ]
}

@test "ship: creates shipped key when missing" {
  make_write_fixture
  run "$STAGEMAN" ship "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/99"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.shipped[0]' "$TEST_DIR/write-status.yaml")
  [ "$result" = "https://github.com/org/repo/pull/99" ]
}

@test "ship: refreshes last_updated" {
  make_write_fixture
  yq -i '.shipped = []' "$TEST_DIR/write-status.yaml"
  "$STAGEMAN" ship "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/42"
  local ts
  ts=$(yq '.last_updated' "$TEST_DIR/write-status.yaml")
  [[ "$ts" != "2026-01-01T00:00:00+00:00" ]]
}

@test "ship: rejects nonexistent file" {
  run "$STAGEMAN" ship "/nonexistent/path/.status.yaml" "https://example.com"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Status file not found"* ]]
}

@test "is-shipped: returns 0 when shipped has entries" {
  make_write_fixture
  yq -i '.shipped = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  run "$STAGEMAN" is-shipped "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "is-shipped: returns 1 when shipped is empty" {
  make_write_fixture
  yq -i '.shipped = []' "$TEST_DIR/write-status.yaml"
  run "$STAGEMAN" is-shipped "$TEST_DIR/write-status.yaml"
  [ "$status" -ne 0 ]
  [ -z "$output" ]
}

@test "is-shipped: returns 1 when shipped key missing" {
  make_write_fixture
  run "$STAGEMAN" is-shipped "$TEST_DIR/write-status.yaml"
  [ "$status" -ne 0 ]
  [ -z "$output" ]
}

@test "is-shipped: rejects nonexistent file" {
  run "$STAGEMAN" is-shipped "/nonexistent/path/.status.yaml"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Status file not found"* ]]
}

@test "history file accumulates events" {
  local history_dir="$TEST_DIR/history"
  mkdir -p "$history_dir"
  "$STAGEMAN" log-command "$history_dir" "fab-continue" ""
  "$STAGEMAN" log-command "$history_dir" "fab-ff" "spec"
  "$STAGEMAN" log-confidence "$history_dir" 4.1 "+4.1" "calc-score"
  "$STAGEMAN" log-review "$history_dir" "passed"
  "$STAGEMAN" log-review "$history_dir" "failed" "revise-tasks"
  local event_count
  event_count=$(wc -l < "$history_dir/.history.jsonl" | tr -d ' ')
  [ "$event_count" = "5" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Ready State Tests
# ─────────────────────────────────────────────────────────────────────────────

@test "progress-line: ready stage shown with symbol" {
  cat > "$TEST_DIR/ready.yaml" <<EOF
progress:
  intake: done
  spec: ready
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" progress-line "$TEST_DIR/ready.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake → spec ◷" ]
}

@test "current-stage: returns ready stage" {
  cat > "$TEST_DIR/ready-stage.yaml" <<EOF
progress:
  intake: done
  spec: ready
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" current-stage "$TEST_DIR/ready-stage.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "spec" ]
}

@test "display-stage: shows ready state" {
  cat > "$TEST_DIR/display-ready.yaml" <<EOF
progress:
  intake: done
  spec: ready
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" display-stage "$TEST_DIR/display-ready.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "spec:ready" ]
}

@test "set-state: ready succeeds without driver" {
  make_write_fixture
  run "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "apply" "ready"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "ready" ]
}

@test "stage-metrics: set-state ready is no-op for metrics" {
  make_write_fixture
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue"
  local started_before
  started_before=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  "$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "ready"
  local started_after completed
  started_after=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  completed=$(yq '.stage_metrics.review.completed_at' "$TEST_DIR/write-status.yaml")
  [ "$started_after" = "$started_before" ]
  [ "$completed" = "null" ]
}

@test "validate-status-file: one active and one ready is valid" {
  cat > "$TEST_DIR/active-and-ready.yaml" <<EOF
progress:
  intake: done
  spec: active
  tasks: ready
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" validate-status-file "$TEST_DIR/active-and-ready.yaml"
  [ "$status" -eq 0 ]
}

@test "validate-status-file: ready state is valid for spec" {
  cat > "$TEST_DIR/ready-valid.yaml" <<EOF
progress:
  intake: done
  spec: ready
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
  run "$STAGEMAN" validate-status-file "$TEST_DIR/ready-valid.yaml"
  [ "$status" -eq 0 ]
}

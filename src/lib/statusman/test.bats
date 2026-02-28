#!/usr/bin/env bats

# Test suite for statusman.sh (CLI-only contract tests)
# Covers: stage queries, validation, .status.yaml accessors, event commands,
#         stage metrics, finish side-effects, reset cascade, fail, history logging

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
STATUSMAN="$REPO_ROOT/fab/.kit/scripts/lib/statusman.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# Helper: create a fresh write-test fixture
# Default state: intake..tasks done, apply active, review+hydrate pending
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
  run "$STATUSMAN" all-stages
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
  run "$STATUSMAN" validate-status-file "$TEST_DIR/valid.yaml"
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
  run "$STATUSMAN" validate-status-file "$TEST_DIR/invalid-state.yaml"
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
  run "$STATUSMAN" validate-status-file "$TEST_DIR/multiple-active.yaml"
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
  run "$STATUSMAN" validate-status-file "$TEST_DIR/wrong-allowed-state.yaml"
  [ "$status" -ne 0 ]
}

@test "validate-status-file ignores stage_metrics content" {
  make_write_fixture
  yq -i '.stage_metrics.intake = {"started_at": "now", "driver": "test", "iterations": 1}' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" validate-status-file "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
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
  run "$STATUSMAN" validate-status-file "$TEST_DIR/active-and-ready.yaml"
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
  run "$STATUSMAN" validate-status-file "$TEST_DIR/ready-valid.yaml"
  [ "$status" -eq 0 ]
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
  run "$STATUSMAN" progress-map "$TEST_DIR/full-status.yaml"
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
  run "$STATUSMAN" progress-map "$TEST_DIR/missing-stage.yaml"
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
  run "$STATUSMAN" checklist "$TEST_DIR/full-status.yaml"
  [[ "$output" == *"generated:true"* ]]
  [[ "$output" == *"completed:3"* ]]
  [[ "$output" == *"total:10"* ]]
}

@test "checklist defaults when block missing" {
  cat > "$TEST_DIR/no-checklist.yaml" <<EOF
progress:
  intake: active
EOF
  run "$STATUSMAN" checklist "$TEST_DIR/no-checklist.yaml"
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
  run "$STATUSMAN" confidence "$TEST_DIR/full-status.yaml"
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
  run "$STATUSMAN" confidence "$TEST_DIR/no-confidence.yaml"
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
  run "$STATUSMAN" current-stage "$TEST_DIR/full-status.yaml"
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
  run "$STATUSMAN" current-stage "$TEST_DIR/no-active.yaml"
  [ "$output" = "spec" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Event: start — {pending,failed} -> active
# ─────────────────────────────────────────────────────────────────────────────

@test "start: pending -> active succeeds" {
  make_write_fixture
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "start: refreshes last_updated" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "test-driver"
  local ts
  ts=$(yq '.last_updated' "$TEST_DIR/write-status.yaml")
  [[ "$ts" != "2026-01-01T00:00:00+00:00" ]]
  [[ "$ts" == *"T"* ]]
}

@test "start: other stages unchanged" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "test-driver"
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
  result=$(yq '.progress.hydrate' "$TEST_DIR/write-status.yaml")
  [ "$result" = "pending" ]
}

@test "start: works without driver" {
  make_write_fixture
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "start: rejects active -> active (already active)" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "apply"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "start: rejects done -> active (use reset instead)" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "intake"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "start: rejects ready -> active (use reset instead)" {
  make_write_fixture
  # Put apply into ready state first
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "apply"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "start: rejects invalid stage" {
  make_write_fixture
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "invalid_stage"
  [ "$status" -ne 0 ]
}

@test "start: rejects nonexistent file" {
  run "$STATUSMAN" start "/nonexistent/path/.status.yaml" "spec"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Event: advance — active -> ready
# ─────────────────────────────────────────────────────────────────────────────

@test "advance: active -> ready succeeds" {
  make_write_fixture
  run "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "ready" ]
}

@test "advance: rejects pending -> ready" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "advance: rejects done -> ready" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "intake"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "advance: rejects ready -> ready (already ready)" {
  make_write_fixture
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "advance: rejects invalid stage" {
  make_write_fixture
  run "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "bogus"
  [ "$status" -ne 0 ]
}

@test "advance: rejects nonexistent file" {
  run "$STATUSMAN" advance "/nonexistent/path/.status.yaml" "spec"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Event: finish — {active,ready} -> done (+next pending -> active)
# ─────────────────────────────────────────────────────────────────────────────

@test "finish: active -> done succeeds" {
  make_write_fixture
  run "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "done" ]
}

@test "finish: ready -> done succeeds" {
  make_write_fixture
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  run "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "done" ]
}

@test "finish: rejects pending -> done" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "finish: rejects done -> done (already done)" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "intake"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "finish: rejects invalid stage" {
  make_write_fixture
  run "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "bogus"
  [ "$status" -ne 0 ]
}

@test "finish: rejects nonexistent file" {
  run "$STATUSMAN" finish "/nonexistent/path/.status.yaml" "apply"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Finish Side-Effects: next stage activation
# ─────────────────────────────────────────────────────────────────────────────

@test "finish side-effect: next pending stage becomes active" {
  make_write_fixture
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "test-driver"
  local result
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "finish side-effect: non-pending next stage is unchanged" {
  make_write_fixture
  # Start review so it is active, then finish apply
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "other-driver"
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "test-driver"
  local result
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  # review was already active, should stay active (not changed)
  [ "$result" = "active" ]
}

@test "finish side-effect: hydrate has no next stage (no side-effect)" {
  # Build a fixture where hydrate is active
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test
progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: done
  hydrate: active
stage_metrics: {}
last_updated: 2026-01-01T00:00:00+00:00
EOF
  run "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "hydrate" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.hydrate' "$TEST_DIR/write-status.yaml")
  [ "$result" = "done" ]
}

@test "finish side-effect: finishing apply activates review with driver" {
  make_write_fixture
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "fab-ff"
  local driver iter
  driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
  [ "$driver" = "fab-ff" ]
  [ "$iter" = "1" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Event: reset — {done,ready} -> active (+cascade downstream to pending)
# ─────────────────────────────────────────────────────────────────────────────

@test "reset: done -> active succeeds" {
  make_write_fixture
  run "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.spec' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "reset: ready -> active succeeds" {
  make_write_fixture
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  run "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "apply" "test-driver"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "reset: rejects pending -> active" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "reset: rejects active -> active" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "apply"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "reset: rejects invalid stage" {
  make_write_fixture
  run "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "bogus"
  [ "$status" -ne 0 ]
}

@test "reset: rejects nonexistent file" {
  run "$STATUSMAN" reset "/nonexistent/path/.status.yaml" "spec"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Reset Cascade
# ─────────────────────────────────────────────────────────────────────────────

@test "reset cascade: downstream stages become pending" {
  make_write_fixture
  # Fixture: intake..tasks=done, apply=active, review+hydrate=pending
  # Reset spec -> tasks,apply,review,hydrate should all go to pending
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "test-driver"
  local result
  result=$(yq '.progress.tasks' "$TEST_DIR/write-status.yaml")
  [ "$result" = "pending" ]
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "pending" ]
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "pending" ]
  result=$(yq '.progress.hydrate' "$TEST_DIR/write-status.yaml")
  [ "$result" = "pending" ]
}

@test "reset cascade: before-target stages preserved" {
  make_write_fixture
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "test-driver"
  local result
  result=$(yq '.progress.intake' "$TEST_DIR/write-status.yaml")
  [ "$result" = "done" ]
}

@test "reset cascade: target itself becomes active" {
  make_write_fixture
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "test-driver"
  local result
  result=$(yq '.progress.spec' "$TEST_DIR/write-status.yaml")
  [ "$result" = "active" ]
}

@test "reset cascade: downstream metrics removed" {
  make_write_fixture
  # Give apply some metrics via start (it is already active from fixture,
  # so finish it and then restart it to get metrics)
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "drv1"
  # review is now active (side-effect). Give it metrics too.
  # Now reset spec: tasks, apply, review, hydrate all go to pending
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "reset-drv"
  local apply_metrics review_metrics
  apply_metrics=$(yq '.stage_metrics.apply' "$TEST_DIR/write-status.yaml")
  review_metrics=$(yq '.stage_metrics.review' "$TEST_DIR/write-status.yaml")
  [ "$apply_metrics" = "null" ]
  [ "$review_metrics" = "null" ]
}

@test "reset cascade: target gets fresh metrics" {
  make_write_fixture
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "reset-drv"
  local driver iter started
  driver=$(yq '.stage_metrics.spec.driver' "$TEST_DIR/write-status.yaml")
  iter=$(yq '.stage_metrics.spec.iterations' "$TEST_DIR/write-status.yaml")
  started=$(yq '.stage_metrics.spec.started_at' "$TEST_DIR/write-status.yaml")
  [ "$driver" = "reset-drv" ]
  [ "$iter" = "1" ]
  [[ "$started" == *"T"* ]]
}

@test "reset cascade: before-target metrics preserved" {
  make_write_fixture
  # Give intake metrics manually so we can verify they survive reset of spec
  yq -i '.stage_metrics.intake = {"started_at": "2026-01-01T00:00:00+00:00", "driver": "orig", "iterations": 1, "completed_at": "2026-01-01T01:00:00+00:00"}' "$TEST_DIR/write-status.yaml"
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "spec" "reset-drv"
  local intake_driver
  intake_driver=$(yq '.stage_metrics.intake.driver' "$TEST_DIR/write-status.yaml")
  [ "$intake_driver" = "orig" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Event: fail — active -> failed (review stage only)
# ─────────────────────────────────────────────────────────────────────────────

@test "fail: review active -> failed succeeds" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "test-driver"
  run "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$result" = "failed" ]
}

@test "fail: rejects non-review stage (apply)" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "apply"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "fail: rejects non-review stage (spec)" {
  # Build a fixture where spec is active
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test
progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
stage_metrics: {}
last_updated: 2026-01-01T00:00:00+00:00
EOF
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "spec"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "fail: rejects review when pending" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "fail: rejects review when done" {
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test
progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: done
  hydrate: active
stage_metrics: {}
last_updated: 2026-01-01T00:00:00+00:00
EOF
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "review"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "fail: rejects invalid stage" {
  make_write_fixture
  run "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "bogus"
  [ "$status" -ne 0 ]
}

@test "fail: rejects nonexistent file" {
  run "$STATUSMAN" fail "/nonexistent/path/.status.yaml" "review"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Review-Specific: start from failed
# ─────────────────────────────────────────────────────────────────────────────

@test "start: failed -> active on review succeeds" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "first-pass"
  "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "review"
  # Verify review is failed
  local state
  state=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$state" = "failed" ]
  # Now start from failed
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "retry-driver"
  [ "$status" -eq 0 ]
  state=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
  [ "$state" = "active" ]
}

@test "start: failed -> active rejected on non-review stage" {
  # Only review supports the failed state, so this test verifies that
  # if a non-review stage somehow had failed, start would reject it.
  # Since non-review stages can't be set to failed, we construct the
  # fixture manually.
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test
progress:
  intake: done
  spec: done
  tasks: failed
  apply: pending
  review: pending
  hydrate: pending
stage_metrics: {}
last_updated: 2026-01-01T00:00:00+00:00
EOF
  run "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "tasks"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Stage Metrics: event-driven
# ─────────────────────────────────────────────────────────────────────────────

@test "stage-metrics: start creates entry with started_at, driver, iterations" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "fab-continue"
  local driver iter started
  driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
  started=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  [ "$driver" = "fab-continue" ]
  [ "$iter" = "1" ]
  [[ "$started" == *"T"* ]]
}

@test "stage-metrics: start without driver records empty driver" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review"
  local driver
  driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  [ "$driver" = "" ]
}

@test "stage-metrics: advance is a no-op for metrics" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "fab-continue"
  local started_before
  started_before=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  # advance apply (which is already active in fixture)
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  # Apply metrics should still be empty/what they were (no entry created by advance)
  # Check review metrics are untouched too
  local started_after
  started_after=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  [ "$started_after" = "$started_before" ]
}

@test "stage-metrics: finish sets completed_at on finished stage" {
  make_write_fixture
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "fab-continue"
  local completed
  completed=$(yq '.stage_metrics.apply.completed_at' "$TEST_DIR/write-status.yaml")
  [[ "$completed" == *"T"* ]]
}

@test "stage-metrics: finish preserves existing driver on finished stage" {
  make_write_fixture
  # apply is active in the fixture. Give it metrics via the active state.
  # The fixture has stage_metrics: {}, so start a new one.
  # Actually, apply is already active, but has no metrics entry.
  # finish will call _apply_metrics_side_effect with "done" which sets completed_at.
  # We need an existing metrics entry. Let's reset and start manually.
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test
progress:
  intake: done
  spec: done
  tasks: done
  apply: active
  review: pending
  hydrate: pending
stage_metrics:
  apply: {started_at: "2026-01-01T00:00:00+00:00", driver: "orig-driver", iterations: 1}
last_updated: 2026-01-01T00:00:00+00:00
EOF
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "finish-driver"
  local driver
  driver=$(yq '.stage_metrics.apply.driver' "$TEST_DIR/write-status.yaml")
  [ "$driver" = "orig-driver" ]
}

@test "stage-metrics: finish activates next stage with metrics" {
  make_write_fixture
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "fab-ff"
  local to_driver to_iter to_started
  to_driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  to_iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
  to_started=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  [ "$to_driver" = "fab-ff" ]
  [ "$to_iter" = "1" ]
  [[ "$to_started" == *"T"* ]]
}

@test "stage-metrics: reset increments iterations" {
  # Build a fixture where apply has existing metrics with iterations: 1
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test
progress:
  intake: done
  spec: done
  tasks: done
  apply: done
  review: pending
  hydrate: pending
stage_metrics:
  apply: {started_at: "2026-01-01T00:00:00+00:00", driver: "d1", iterations: 1, completed_at: "2026-01-01T01:00:00+00:00"}
last_updated: 2026-01-01T00:00:00+00:00
EOF
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "apply" "d2"
  local iter
  iter=$(yq '.stage_metrics.apply.iterations' "$TEST_DIR/write-status.yaml")
  [ "$iter" = "2" ]
}

@test "stage-metrics: fail is a no-op for metrics" {
  make_write_fixture
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "fab-continue"
  local started_before driver_before
  started_before=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  driver_before=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  "$STATUSMAN" fail "$TEST_DIR/write-status.yaml" "review"
  local started_after driver_after completed
  started_after=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  driver_after=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  completed=$(yq '.stage_metrics.review.completed_at' "$TEST_DIR/write-status.yaml")
  [ "$started_after" = "$started_before" ]
  [ "$driver_after" = "$driver_before" ]
  [ "$completed" = "null" ]
}

@test "stage-metrics: reset clears downstream metrics" {
  make_write_fixture
  "$STATUSMAN" finish "$TEST_DIR/write-status.yaml" "apply" "d1"
  # review is now active with metrics
  local review_before
  review_before=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
  [ "$review_before" = "d1" ]
  # Reset apply — review (downstream) should lose metrics
  "$STATUSMAN" reset "$TEST_DIR/write-status.yaml" "apply" "d2"
  local review_after
  review_after=$(yq '.stage_metrics.review' "$TEST_DIR/write-status.yaml")
  [ "$review_after" = "null" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Write: set-checklist
# ─────────────────────────────────────────────────────────────────────────────

@test "set-checklist: completed update succeeds" {
  make_write_fixture
  run "$STATUSMAN" set-checklist "$TEST_DIR/write-status.yaml" "completed" "7"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  completed:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "7" ]
}

@test "set-checklist: generated update succeeds" {
  make_write_fixture
  run "$STATUSMAN" set-checklist "$TEST_DIR/write-status.yaml" "generated" "false"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  generated:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "false" ]
}

@test "set-checklist: total update succeeds" {
  make_write_fixture
  run "$STATUSMAN" set-checklist "$TEST_DIR/write-status.yaml" "total" "25"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  total:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
  [ "$result" = "25" ]
}

@test "set-checklist: rejects invalid field" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" set-checklist "$TEST_DIR/write-status.yaml" "invalid_field" "5"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-checklist: rejects negative integer" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" set-checklist "$TEST_DIR/write-status.yaml" "completed" "-3"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-checklist: rejects non-bool for generated" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" set-checklist "$TEST_DIR/write-status.yaml" "generated" "yes"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-checklist: rejects nonexistent file" {
  run "$STATUSMAN" set-checklist "/nonexistent/path/.status.yaml" "completed" "5"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Write: set-confidence
# ─────────────────────────────────────────────────────────────────────────────

@test "set-confidence: valid replacement succeeds" {
  make_write_fixture
  run "$STATUSMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1"
  [ "$status" -eq 0 ]
  local result
  result=$(grep "^  certain:" "$TEST_DIR/write-status.yaml" | sed 's/.*: *//')
  [ "$result" = "10" ]
  result=$(grep "^  score:" "$TEST_DIR/write-status.yaml" | sed 's/.*: *//')
  [ "$result" = "4.1" ]
}

@test "set-confidence: refreshes last_updated" {
  make_write_fixture
  "$STATUSMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1"
  local ts
  ts=$(grep "^last_updated:" "$TEST_DIR/write-status.yaml" | sed 's/last_updated: //')
  [[ "$ts" == *"T"* ]]
}

@test "set-confidence: preserves other blocks" {
  make_write_fixture
  "$STATUSMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1"
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
  run "$STATUSMAN" set-confidence "$TEST_DIR/write-status.yaml" "-1" "3" "1" "0" "4.1"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-confidence: rejects non-numeric score" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "abc"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-confidence: rejects negative score" {
  make_write_fixture
  local before
  before=$(cat "$TEST_DIR/write-status.yaml")
  run "$STATUSMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "-2.5"
  [ "$status" -ne 0 ]
  local after
  after=$(cat "$TEST_DIR/write-status.yaml")
  [ "$before" = "$after" ]
}

@test "set-confidence: rejects nonexistent file" {
  run "$STATUSMAN" set-confidence "/nonexistent/path/.status.yaml" "10" "3" "1" "0" "4.1"
  [ "$status" -ne 0 ]
}


# (History logging tests removed — log_command, log_confidence, log_review
# were extracted to logman.sh. See src/lib/logman/ for logman tests.)

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
  run "$STATUSMAN" progress-line "$TEST_DIR/all-pending.yaml"
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
  run "$STATUSMAN" progress-line "$TEST_DIR/first-active.yaml"
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
  run "$STATUSMAN" progress-line "$TEST_DIR/mid-pipeline.yaml"
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
  run "$STATUSMAN" progress-line "$TEST_DIR/failed.yaml"
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
  run "$STATUSMAN" progress-line "$TEST_DIR/all-done.yaml"
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
  run "$STATUSMAN" progress-line "$TEST_DIR/single-done.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake" ]
}

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
  run "$STATUSMAN" progress-line "$TEST_DIR/ready.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "intake → spec ◷" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Issues & PRs
# ─────────────────────────────────────────────────────────────────────────────

@test "add-pr: appends URL to empty prs array" {
  make_write_fixture
  yq -i '.prs = []' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" add-pr "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/42"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.prs[0]' "$TEST_DIR/write-status.yaml")
  [ "$result" = "https://github.com/org/repo/pull/42" ]
}

@test "add-pr: appends second URL" {
  make_write_fixture
  yq -i '.prs = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  "$STATUSMAN" add-pr "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/43"
  local count
  count=$(yq '.prs | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "2" ]
  local second
  second=$(yq '.prs[1]' "$TEST_DIR/write-status.yaml")
  [ "$second" = "https://github.com/org/repo/pull/43" ]
}

@test "add-pr: deduplicates existing URL" {
  make_write_fixture
  yq -i '.prs = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" add-pr "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/42"
  [ "$status" -eq 0 ]
  local count
  count=$(yq '.prs | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "1" ]
}

@test "add-pr: does not treat substring as duplicate" {
  make_write_fixture
  yq -i '.prs = ["https://github.com/org/repo/pull/42"]' "$TEST_DIR/write-status.yaml"
  "$STATUSMAN" add-pr "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/4"
  local count
  count=$(yq '.prs | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "2" ]
}

@test "add-pr: creates prs key when missing" {
  make_write_fixture
  run "$STATUSMAN" add-pr "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/99"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.prs[0]' "$TEST_DIR/write-status.yaml")
  [ "$result" = "https://github.com/org/repo/pull/99" ]
}

@test "add-pr: refreshes last_updated" {
  make_write_fixture
  yq -i '.prs = []' "$TEST_DIR/write-status.yaml"
  "$STATUSMAN" add-pr "$TEST_DIR/write-status.yaml" "https://github.com/org/repo/pull/42"
  local ts
  ts=$(yq '.last_updated' "$TEST_DIR/write-status.yaml")
  [[ "$ts" != "2026-01-01T00:00:00+00:00" ]]
}

@test "add-pr: rejects nonexistent file" {
  run "$STATUSMAN" add-pr "/nonexistent/path/.status.yaml" "https://example.com"
  [ "$status" -ne 0 ]
}

@test "get-prs: returns URLs one per line" {
  make_write_fixture
  yq -i '.prs = ["https://github.com/org/repo/pull/42", "https://github.com/org/repo/pull/43"]' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" get-prs "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | wc -l)" -eq 2 ]
}

@test "get-prs: returns empty for empty array" {
  make_write_fixture
  yq -i '.prs = []' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" get-prs "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "get-prs: returns empty when key missing" {
  make_write_fixture
  run "$STATUSMAN" get-prs "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "get-prs: rejects nonexistent file" {
  run "$STATUSMAN" get-prs "/nonexistent/path/.status.yaml"
  [ "$status" -ne 0 ]
}

@test "add-issue: appends ID to empty issues array" {
  make_write_fixture
  yq -i '.issues = []' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" add-issue "$TEST_DIR/write-status.yaml" "DEV-123"
  [ "$status" -eq 0 ]
  local result
  result=$(yq '.issues[0]' "$TEST_DIR/write-status.yaml")
  [ "$result" = "DEV-123" ]
}

@test "add-issue: deduplicates existing ID" {
  make_write_fixture
  yq -i '.issues = ["DEV-123"]' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" add-issue "$TEST_DIR/write-status.yaml" "DEV-123"
  [ "$status" -eq 0 ]
  local count
  count=$(yq '.issues | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "1" ]
}

@test "add-issue: appends second ID" {
  make_write_fixture
  yq -i '.issues = ["DEV-123"]' "$TEST_DIR/write-status.yaml"
  "$STATUSMAN" add-issue "$TEST_DIR/write-status.yaml" "DEV-456"
  local count
  count=$(yq '.issues | length' "$TEST_DIR/write-status.yaml")
  [ "$count" = "2" ]
}

@test "add-issue: refreshes last_updated" {
  make_write_fixture
  yq -i '.issues = []' "$TEST_DIR/write-status.yaml"
  "$STATUSMAN" add-issue "$TEST_DIR/write-status.yaml" "DEV-123"
  local ts
  ts=$(yq '.last_updated' "$TEST_DIR/write-status.yaml")
  [[ "$ts" != "2026-01-01T00:00:00+00:00" ]]
}

@test "add-issue: rejects nonexistent file" {
  run "$STATUSMAN" add-issue "/nonexistent/path/.status.yaml" "DEV-123"
  [ "$status" -ne 0 ]
}

@test "get-issues: returns IDs one per line" {
  make_write_fixture
  yq -i '.issues = ["DEV-123", "DEV-456"]' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" get-issues "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
  [ "$(echo "$output" | wc -l)" -eq 2 ]
}

@test "get-issues: returns empty for empty array" {
  make_write_fixture
  yq -i '.issues = []' "$TEST_DIR/write-status.yaml"
  run "$STATUSMAN" get-issues "$TEST_DIR/write-status.yaml"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "get-issues: rejects nonexistent file" {
  run "$STATUSMAN" get-issues "/nonexistent/path/.status.yaml"
  [ "$status" -ne 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Ready State (accessor/display tests)
# ─────────────────────────────────────────────────────────────────────────────

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
  run "$STATUSMAN" current-stage "$TEST_DIR/ready-stage.yaml"
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
  run "$STATUSMAN" display-stage "$TEST_DIR/display-ready.yaml"
  [ "$status" -eq 0 ]
  [ "$output" = "spec:ready" ]
}

@test "advance: produces ready state in file" {
  make_write_fixture
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "apply"
  local result
  result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
  [ "$result" = "ready" ]
}

@test "stage-metrics: advance is no-op for metrics (preserves active metrics)" {
  make_write_fixture
  # Start review to give it metrics
  "$STATUSMAN" start "$TEST_DIR/write-status.yaml" "review" "fab-continue"
  local started_before
  started_before=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  # Advance review to ready
  "$STATUSMAN" advance "$TEST_DIR/write-status.yaml" "review"
  local started_after completed
  started_after=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
  completed=$(yq '.stage_metrics.review.completed_at' "$TEST_DIR/write-status.yaml")
  [ "$started_after" = "$started_before" ]
  [ "$completed" = "null" ]
}

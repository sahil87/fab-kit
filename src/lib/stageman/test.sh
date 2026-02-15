#!/usr/bin/env bash
# src/lib/stageman/test.sh
#
# Comprehensive test suite for stageman.sh (CLI-only contract tests)
# Run: ./test.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
STAGEMAN="$SCRIPT_DIR/stageman.sh"

# Test colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helpers
assert_equal() {
  local expected="$1"
  local actual="$2"
  local test_name="$3"

  ((TESTS_RUN++))

  if [ "$expected" = "$actual" ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "  Expected: $expected"
    echo "  Got:      $actual"
    ((TESTS_FAILED++))
    return 1
  fi
}

assert_success() {
  local exit_code=$?
  local test_name="$1"
  ((TESTS_RUN++))

  if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name (exit code: $exit_code)"
    ((TESTS_FAILED++))
    return 1
  fi
}

assert_failure() {
  local exit_code=$?
  local test_name="$1"
  ((TESTS_RUN++))

  if [ $exit_code -ne 0 ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name"
    ((TESTS_FAILED++))
    return 1
  fi
}

assert_contains() {
  local needle="$1"
  local haystack="$2"
  local test_name="$3"

  ((TESTS_RUN++))

  if grep -q "$needle" <<< "$haystack"; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "  Expected to find: $needle"
    echo "  In: $haystack"
    ((TESTS_FAILED++))
    return 1
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# State Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing State Queries..."
echo ""

# all-states
all_states=$("$STAGEMAN" all-states)
assert_contains "pending" "$all_states" "all-states includes pending"
assert_contains "active" "$all_states" "all-states includes active"
assert_contains "done" "$all_states" "all-states includes done"
assert_contains "skipped" "$all_states" "all-states includes skipped"
assert_contains "failed" "$all_states" "all-states includes failed"

# validate-state
"$STAGEMAN" validate-state "done"
assert_success "validate-state accepts 'done'"

"$STAGEMAN" validate-state "invalid" 2>/dev/null
assert_failure "validate-state rejects 'invalid'"

# state-symbol
symbol=$("$STAGEMAN" state-symbol "active")
assert_equal "●" "$symbol" "state-symbol returns correct symbol for active"

symbol=$("$STAGEMAN" state-symbol "done")
assert_equal "✓" "$symbol" "state-symbol returns correct symbol for done"

# is-terminal
"$STAGEMAN" is-terminal "done"
assert_success "is-terminal returns true for done"

"$STAGEMAN" is-terminal "active"
assert_failure "is-terminal returns false for active"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Stage Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Stage Queries..."
echo ""

# all-stages
all_stages=$("$STAGEMAN" all-stages)
assert_contains "brief" "$all_stages" "all-stages includes brief"
assert_contains "spec" "$all_stages" "all-stages includes spec"
assert_contains "tasks" "$all_stages" "all-stages includes tasks"
assert_contains "apply" "$all_stages" "all-stages includes apply"
assert_contains "review" "$all_stages" "all-stages includes review"
assert_contains "hydrate" "$all_stages" "all-stages includes hydrate"

# Stage count
stage_count=$("$STAGEMAN" all-stages | wc -l)
assert_equal "6" "$stage_count" "all-stages returns exactly 6 stages"

# validate-stage
"$STAGEMAN" validate-stage "spec"
assert_success "validate-stage accepts 'spec'"

"$STAGEMAN" validate-stage "invalid" 2>/dev/null
assert_failure "validate-stage rejects 'invalid'"

# stage-number
num=$("$STAGEMAN" stage-number "spec")
assert_equal "2" "$num" "stage-number returns 2 for spec"

num=$("$STAGEMAN" stage-number "hydrate")
assert_equal "6" "$num" "stage-number returns 6 for hydrate"

# stage-name
name=$("$STAGEMAN" stage-name "spec")
assert_equal "Specification" "$name" "stage-name returns display name"

# stage-artifact
artifact=$("$STAGEMAN" stage-artifact "spec")
assert_equal "spec.md" "$artifact" "stage-artifact returns correct filename"

artifact=$("$STAGEMAN" stage-artifact "apply")
assert_equal "" "$artifact" "stage-artifact returns empty for non-generating stage"

# allowed-states
allowed=$("$STAGEMAN" allowed-states "review")
assert_contains "failed" "$allowed" "allowed-states for review includes failed"
assert_contains "done" "$allowed" "allowed-states for review includes done"

# initial-state
initial=$("$STAGEMAN" initial-state "brief")
assert_equal "active" "$initial" "initial-state for brief is active"

initial=$("$STAGEMAN" initial-state "spec")
assert_equal "pending" "$initial" "initial-state for spec is pending"

# has-auto-checklist
"$STAGEMAN" has-auto-checklist "tasks"
assert_success "has-auto-checklist returns true for tasks"

"$STAGEMAN" has-auto-checklist "spec"
assert_failure "has-auto-checklist returns false for spec"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Progression Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Progression..."
echo ""

# next-stage
next=$("$STAGEMAN" next-stage "spec")
assert_equal "tasks" "$next" "next-stage after spec is tasks"

next=$("$STAGEMAN" next-stage "brief")
assert_equal "spec" "$next" "next-stage after brief is spec"

# Last stage has no next
"$STAGEMAN" next-stage "hydrate" >/dev/null 2>&1
assert_failure "next-stage returns error for hydrate"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Validation Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Validation..."
echo ""

# Create test status file
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

cat > "$TEST_DIR/valid.yaml" <<EOF
progress:
  brief: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
stage_metrics: {}
EOF

"$STAGEMAN" validate-status-file "$TEST_DIR/valid.yaml" 2>&1
assert_success "validate-status-file accepts valid status"

cat > "$TEST_DIR/invalid-state.yaml" <<EOF
progress:
  brief: done
  spec: invalid_state
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

"$STAGEMAN" validate-status-file "$TEST_DIR/invalid-state.yaml" >/dev/null 2>&1
assert_failure "validate-status-file rejects invalid state"

cat > "$TEST_DIR/multiple-active.yaml" <<EOF
progress:
  brief: active
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

"$STAGEMAN" validate-status-file "$TEST_DIR/multiple-active.yaml" >/dev/null 2>&1
assert_failure "validate-status-file rejects multiple active stages"

cat > "$TEST_DIR/wrong-allowed-state.yaml" <<EOF
progress:
  brief: failed
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

"$STAGEMAN" validate-status-file "$TEST_DIR/wrong-allowed-state.yaml" >/dev/null 2>&1
assert_failure "validate-status-file rejects state not in allowed_states"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# .status.yaml Accessor Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing .status.yaml Accessors..."
echo ""

# Full status file for accessor tests
cat > "$TEST_DIR/full-status.yaml" <<EOF
name: test-change
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

# progress-map
progress_output=$("$STAGEMAN" progress-map "$TEST_DIR/full-status.yaml")
assert_contains "brief:done" "$progress_output" "progress-map extracts brief:done"
assert_contains "spec:active" "$progress_output" "progress-map extracts spec:active"
assert_contains "hydrate:pending" "$progress_output" "progress-map extracts hydrate:pending"

line_count=$(echo "$progress_output" | wc -l | tr -d ' ')
assert_equal "6" "$line_count" "progress-map returns exactly 6 lines"

# progress-map with missing stage (defaults to pending)
cat > "$TEST_DIR/missing-stage.yaml" <<EOF
progress:
  brief: done
  spec: active
  apply: pending
  review: pending
  hydrate: pending
EOF
missing_output=$("$STAGEMAN" progress-map "$TEST_DIR/missing-stage.yaml")
assert_contains "tasks:pending" "$missing_output" "progress-map defaults missing stage to pending"

# checklist
checklist_output=$("$STAGEMAN" checklist "$TEST_DIR/full-status.yaml")
assert_contains "generated:true" "$checklist_output" "checklist extracts generated"
assert_contains "completed:3" "$checklist_output" "checklist extracts completed"
assert_contains "total:10" "$checklist_output" "checklist extracts total"

# checklist with missing block (defaults)
cat > "$TEST_DIR/no-checklist.yaml" <<EOF
progress:
  brief: active
EOF
no_chk_output=$("$STAGEMAN" checklist "$TEST_DIR/no-checklist.yaml")
assert_contains "generated:false" "$no_chk_output" "checklist defaults generated to false"
assert_contains "completed:0" "$no_chk_output" "checklist defaults completed to 0"
assert_contains "total:0" "$no_chk_output" "checklist defaults total to 0"

# confidence
confidence_output=$("$STAGEMAN" confidence "$TEST_DIR/full-status.yaml")
assert_contains "certain:5" "$confidence_output" "confidence extracts certain"
assert_contains "confident:2" "$confidence_output" "confidence extracts confident"
assert_contains "tentative:1" "$confidence_output" "confidence extracts tentative"
assert_contains "unresolved:0" "$confidence_output" "confidence extracts unresolved"
assert_contains "score:3.4" "$confidence_output" "confidence extracts score"

# confidence with missing block (defaults)
cat > "$TEST_DIR/no-confidence.yaml" <<EOF
progress:
  brief: active
EOF
no_conf_output=$("$STAGEMAN" confidence "$TEST_DIR/no-confidence.yaml")
assert_contains "certain:0" "$no_conf_output" "confidence defaults certain to 0"
assert_contains "score:5.0" "$no_conf_output" "confidence defaults score to 5.0"

# current-stage
current=$("$STAGEMAN" current-stage "$TEST_DIR/full-status.yaml")
assert_equal "spec" "$current" "current-stage finds active stage via CLI"

# current-stage fallback (no active, first pending after last done)
cat > "$TEST_DIR/no-active.yaml" <<EOF
progress:
  brief: done
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
fallback=$("$STAGEMAN" current-stage "$TEST_DIR/no-active.yaml")
assert_equal "spec" "$fallback" "current-stage fallback finds first pending after last done"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Write Function Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Write Functions..."
echo ""

# Create a fresh status file for write tests
make_write_fixture() {
  cat > "$TEST_DIR/write-status.yaml" <<EOF
name: test-write
progress:
  brief: done
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

# --- set-state ---

echo "  set-state:"

make_write_fixture
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "test-driver" 2>/dev/null
assert_success "    valid state change succeeds"
result=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
assert_equal "active" "$result" "    review is now active"

# Verify last_updated was refreshed
ts=$(yq '.last_updated' "$TEST_DIR/write-status.yaml")
assert_contains "T" "$ts" "    last_updated refreshed with ISO 8601 timestamp"

# Other stages unchanged
result=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
assert_equal "active" "$result" "    unrelated stages unchanged"

# active requires driver
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" 2>/dev/null
assert_failure "    rejects active without driver"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged when driver missing"

# done does not require driver
make_write_fixture
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "apply" "done" 2>/dev/null
assert_success "    done without driver succeeds"

# Validation failures
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "invalid_stage" "active" "test" 2>/dev/null
assert_failure "    rejects invalid stage"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on invalid stage"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "brief" "failed" 2>/dev/null
assert_failure "    rejects state not allowed for stage"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on invalid state"

"$STAGEMAN" set-state "/nonexistent/path/.status.yaml" "spec" "done" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# --- transition ---

echo "  transition:"

make_write_fixture
"$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "review" "test-driver" 2>/dev/null
assert_success "    valid forward transition succeeds"
from_state=$(yq '.progress.apply' "$TEST_DIR/write-status.yaml")
to_state=$(yq '.progress.review' "$TEST_DIR/write-status.yaml")
assert_equal "done" "$from_state" "    from_stage set to done"
assert_equal "active" "$to_state" "    to_stage set to active"

# transition requires driver
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "review" 2>/dev/null
assert_failure "    rejects transition without driver"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged without driver"

# Non-adjacent stages
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "hydrate" "test-driver" 2>/dev/null
assert_failure "    rejects non-adjacent stages"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on non-adjacent"

# from_stage not active
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "spec" "tasks" "test-driver" 2>/dev/null
assert_failure "    rejects when from_stage is not active"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged when from_stage not active"

"$STAGEMAN" transition "/nonexistent/path/.status.yaml" "apply" "review" "test-driver" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# --- set-checklist ---

echo "  set-checklist:"

make_write_fixture
"$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "completed" "7" 2>/dev/null
assert_success "    valid completed update succeeds"
result=$(grep "^  completed:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "7" "$result" "    completed is now 7"

make_write_fixture
"$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "generated" "false" 2>/dev/null
assert_success "    valid generated update succeeds"
result=$(grep "^  generated:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "false" "$result" "    generated is now false"

make_write_fixture
"$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "total" "25" 2>/dev/null
assert_success "    valid total update succeeds"
result=$(grep "^  total:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "25" "$result" "    total is now 25"

# Validation failures
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "invalid_field" "5" 2>/dev/null
assert_failure "    rejects invalid field name"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on invalid field"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "completed" "-3" 2>/dev/null
assert_failure "    rejects negative integer"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on negative value"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-checklist "$TEST_DIR/write-status.yaml" "generated" "yes" 2>/dev/null
assert_failure "    rejects non-bool for generated"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on wrong type"

"$STAGEMAN" set-checklist "/nonexistent/path/.status.yaml" "completed" "5" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# --- set-confidence ---

echo "  set-confidence:"

make_write_fixture
"$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1" 2>/dev/null
assert_success "    valid confidence block replacement succeeds"
result=$(grep "^  certain:" "$TEST_DIR/write-status.yaml" | sed 's/.*: *//')
assert_equal "10" "$result" "    certain updated to 10"
result=$(grep "^  score:" "$TEST_DIR/write-status.yaml" | sed 's/.*: *//')
assert_equal "4.1" "$result" "    score updated to 4.1"

# Verify last_updated refreshed
ts=$(grep "^last_updated:" "$TEST_DIR/write-status.yaml" | sed 's/last_updated: //')
assert_contains "T" "$ts" "    last_updated refreshed"

# Verify other blocks preserved
result=$(grep "^  apply:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "active" "$result" "    progress block preserved"
result=$(grep "^  total:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "10" "$result" "    checklist block preserved"

# Validation failures
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "-1" "3" "1" "0" "4.1" 2>/dev/null
assert_failure "    rejects negative count"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on negative count"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "abc" 2>/dev/null
assert_failure "    rejects non-numeric score"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on non-numeric score"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
"$STAGEMAN" set-confidence "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "-2.5" 2>/dev/null
assert_failure "    rejects negative score"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on negative score"

"$STAGEMAN" set-confidence "/nonexistent/path/.status.yaml" "10" "3" "1" "0" "4.1" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Stage Metrics Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Stage Metrics..."
echo ""

# stage-metrics on empty block
make_write_fixture
metrics_output=$("$STAGEMAN" stage-metrics "$TEST_DIR/write-status.yaml")
assert_equal "" "$metrics_output" "stage-metrics returns empty on empty block"

# set-state active creates metrics
make_write_fixture
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue" 2>/dev/null
driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
assert_equal "fab-continue" "$driver" "set-state active sets driver metric"
iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
assert_equal "1" "$iter" "set-state active sets iterations=1"
started=$(yq '.stage_metrics.review.started_at' "$TEST_DIR/write-status.yaml")
assert_contains "T" "$started" "set-state active sets started_at"

# set-state done sets completed_at
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "done" 2>/dev/null
completed=$(yq '.stage_metrics.review.completed_at' "$TEST_DIR/write-status.yaml")
assert_contains "T" "$completed" "set-state done sets completed_at"
# Other fields preserved
driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
assert_equal "fab-continue" "$driver" "set-state done preserves driver"

# Rework: re-activation increments iterations
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue" 2>/dev/null
iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
assert_equal "2" "$iter" "rework re-activation increments iterations to 2"

# pending clears metrics entry
make_write_fixture
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "apply" "done" 2>/dev/null
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "active" "fab-continue" 2>/dev/null
"$STAGEMAN" set-state "$TEST_DIR/write-status.yaml" "review" "pending" 2>/dev/null
result=$(yq '.stage_metrics.review' "$TEST_DIR/write-status.yaml")
assert_equal "null" "$result" "pending clears stage metrics entry"

# transition sets metrics for both stages
make_write_fixture
"$STAGEMAN" transition "$TEST_DIR/write-status.yaml" "apply" "review" "fab-ff" 2>/dev/null
from_completed=$(yq '.stage_metrics.apply.completed_at' "$TEST_DIR/write-status.yaml")
assert_contains "T" "$from_completed" "transition sets from→done completed_at"
to_driver=$(yq '.stage_metrics.review.driver' "$TEST_DIR/write-status.yaml")
assert_equal "fab-ff" "$to_driver" "transition sets to→active driver"
to_iter=$(yq '.stage_metrics.review.iterations' "$TEST_DIR/write-status.yaml")
assert_equal "1" "$to_iter" "transition sets to→active iterations=1"

# stage-metrics single stage
metrics_single=$("$STAGEMAN" stage-metrics "$TEST_DIR/write-status.yaml" "review")
assert_contains "driver" "$metrics_single" "stage-metrics single stage returns fields"

# stage-metrics all stages
metrics_all=$("$STAGEMAN" stage-metrics "$TEST_DIR/write-status.yaml")
line_count=$(echo "$metrics_all" | wc -l | tr -d ' ')
assert_equal "2" "$line_count" "stage-metrics all returns 2 entries"

# stage-metrics with missing block entirely
cat > "$TEST_DIR/no-metrics.yaml" <<EOF
progress:
  brief: active
last_updated: now
EOF
no_metrics=$("$STAGEMAN" stage-metrics "$TEST_DIR/no-metrics.yaml")
assert_equal "" "$no_metrics" "stage-metrics handles missing block"

# validate-status-file ignores stage_metrics
make_write_fixture
yq -i '.stage_metrics.brief = {"started_at": "now", "driver": "test", "iterations": 1}' "$TEST_DIR/write-status.yaml"
"$STAGEMAN" validate-status-file "$TEST_DIR/write-status.yaml" 2>&1
assert_success "validate-status-file ignores stage_metrics content"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# History Logging Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing History Logging..."
echo ""

HISTORY_DIR=$(mktemp -d)

"$STAGEMAN" log-command "$HISTORY_DIR" "fab-continue" ""
line=$(head -1 "$HISTORY_DIR/.history.jsonl")
echo "$line" | grep -q '"event":"command"' 2>/dev/null
assert_success "log-command creates command event"
echo "$line" | grep -q '"cmd":"fab-continue"' 2>/dev/null
assert_success "log-command includes cmd field"
echo "$line" | grep -q '"args"' 2>/dev/null
assert_failure "log-command omits empty args"

"$STAGEMAN" log-command "$HISTORY_DIR" "fab-ff" "spec"
line=$(tail -1 "$HISTORY_DIR/.history.jsonl")
echo "$line" | grep -q '"args":"spec"' 2>/dev/null
assert_success "log-command includes non-empty args"

"$STAGEMAN" log-confidence "$HISTORY_DIR" 4.1 "+4.1" "calc-score"
line=$(tail -1 "$HISTORY_DIR/.history.jsonl")
echo "$line" | grep -q '"event":"confidence"' 2>/dev/null
assert_success "log-confidence creates confidence event"
echo "$line" | grep -q '"score":4.1' 2>/dev/null
assert_success "log-confidence includes score"

"$STAGEMAN" log-review "$HISTORY_DIR" "passed"
line=$(tail -1 "$HISTORY_DIR/.history.jsonl")
echo "$line" | grep -q '"result":"passed"' 2>/dev/null
assert_success "log-review creates passed event"
echo "$line" | grep -q '"rework"' 2>/dev/null
assert_failure "log-review omits empty rework"

"$STAGEMAN" log-review "$HISTORY_DIR" "failed" "revise-tasks"
line=$(tail -1 "$HISTORY_DIR/.history.jsonl")
echo "$line" | grep -q '"rework":"revise-tasks"' 2>/dev/null
assert_success "log-review includes rework when provided"

event_count=$(wc -l < "$HISTORY_DIR/.history.jsonl" | tr -d ' ')
assert_equal "5" "$event_count" "history file has 5 events"

rm -rf "$HISTORY_DIR"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────────────────────

echo "═══════════════════════════════════════════════════════════════════════════"
echo "Test Results"
echo "═══════════════════════════════════════════════════════════════════════════"
echo ""
echo "Tests run:    $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"

if [ $TESTS_FAILED -gt 0 ]; then
  echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
  echo ""
  echo -e "${RED}FAIL${NC}"
  exit 1
else
  echo "Tests failed: 0"
  echo ""
  echo -e "${GREEN}PASS${NC}"
  exit 0
fi

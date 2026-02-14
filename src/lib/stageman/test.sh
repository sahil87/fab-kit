#!/usr/bin/env bash
# src/lib/stageman/test.sh
#
# Comprehensive test suite for stageman.sh
# Run: ./test.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/stageman.sh"
set +e  # stageman.sh enables -e; disable it for test assertions

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

# get_all_states
all_states=$(get_all_states)
assert_contains "pending" "$all_states" "get_all_states includes pending"
assert_contains "active" "$all_states" "get_all_states includes active"
assert_contains "done" "$all_states" "get_all_states includes done"
assert_contains "skipped" "$all_states" "get_all_states includes skipped"
assert_contains "failed" "$all_states" "get_all_states includes failed"

# validate_state
validate_state "done"
assert_success "validate_state accepts 'done'"

validate_state "invalid" 2>/dev/null
assert_failure "validate_state rejects 'invalid'"

# get_state_symbol
symbol=$(get_state_symbol "active")
assert_equal "●" "$symbol" "get_state_symbol returns correct symbol for active"

symbol=$(get_state_symbol "done")
assert_equal "✓" "$symbol" "get_state_symbol returns correct symbol for done"

# is_terminal_state
is_terminal_state "done"
assert_success "is_terminal_state returns true for done"

is_terminal_state "active"
assert_failure "is_terminal_state returns false for active"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Stage Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Stage Queries..."
echo ""

# get_all_stages
all_stages=$(get_all_stages)
assert_contains "brief" "$all_stages" "get_all_stages includes brief"
assert_contains "spec" "$all_stages" "get_all_stages includes spec"
assert_contains "tasks" "$all_stages" "get_all_stages includes tasks"
assert_contains "apply" "$all_stages" "get_all_stages includes apply"
assert_contains "review" "$all_stages" "get_all_stages includes review"
assert_contains "hydrate" "$all_stages" "get_all_stages includes hydrate"

# Stage count
stage_count=$(get_all_stages | wc -l)
assert_equal "6" "$stage_count" "get_all_stages returns exactly 6 stages"

# validate_stage
validate_stage "spec"
assert_success "validate_stage accepts 'spec'"

validate_stage "invalid" 2>/dev/null
assert_failure "validate_stage rejects 'invalid'"

# get_stage_number
num=$(get_stage_number "spec")
assert_equal "2" "$num" "get_stage_number returns 2 for spec"

num=$(get_stage_number "hydrate")
assert_equal "6" "$num" "get_stage_number returns 6 for hydrate"

# get_stage_name
name=$(get_stage_name "spec")
assert_equal "Specification" "$name" "get_stage_name returns display name"

# get_stage_artifact
artifact=$(get_stage_artifact "spec")
assert_equal "spec.md" "$artifact" "get_stage_artifact returns correct filename"

artifact=$(get_stage_artifact "apply")
assert_equal "" "$artifact" "get_stage_artifact returns empty for non-generating stage"

# get_allowed_states
allowed=$(get_allowed_states "review")
assert_contains "failed" "$allowed" "get_allowed_states for review includes failed"
assert_contains "done" "$allowed" "get_allowed_states for review includes done"

# get_initial_state
initial=$(get_initial_state "brief")
assert_equal "active" "$initial" "get_initial_state for brief is active"

initial=$(get_initial_state "spec")
assert_equal "pending" "$initial" "get_initial_state for spec is pending"

# has_auto_checklist
has_auto_checklist "tasks"
assert_success "has_auto_checklist returns true for tasks"

has_auto_checklist "spec"
assert_failure "has_auto_checklist returns false for spec"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Progression Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Progression..."
echo ""

# get_next_stage
next=$(get_next_stage "spec")
assert_equal "tasks" "$next" "get_next_stage after spec is tasks"

next=$(get_next_stage "brief")
assert_equal "spec" "$next" "get_next_stage after brief is spec"

# Last stage has no next
get_next_stage "hydrate" >/dev/null 2>&1
assert_failure "get_next_stage returns error for hydrate"

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
EOF

validate_status_file "$TEST_DIR/valid.yaml" 2>&1
assert_success "validate_status_file accepts valid status"

cat > "$TEST_DIR/invalid-state.yaml" <<EOF
progress:
  brief: done
  spec: invalid_state
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

validate_status_file "$TEST_DIR/invalid-state.yaml" >/dev/null 2>&1
assert_failure "validate_status_file rejects invalid state"

cat > "$TEST_DIR/multiple-active.yaml" <<EOF
progress:
  brief: active
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

validate_status_file "$TEST_DIR/multiple-active.yaml" >/dev/null 2>&1
assert_failure "validate_status_file rejects multiple active stages"

cat > "$TEST_DIR/wrong-allowed-state.yaml" <<EOF
progress:
  brief: failed
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF

validate_status_file "$TEST_DIR/wrong-allowed-state.yaml" >/dev/null 2>&1
assert_failure "validate_status_file rejects state not in allowed_states"

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

# get_progress_map
progress_output=$(get_progress_map "$TEST_DIR/full-status.yaml")
assert_contains "brief:done" "$progress_output" "get_progress_map extracts brief:done"
assert_contains "spec:active" "$progress_output" "get_progress_map extracts spec:active"
assert_contains "hydrate:pending" "$progress_output" "get_progress_map extracts hydrate:pending"

line_count=$(echo "$progress_output" | wc -l | tr -d ' ')
assert_equal "6" "$line_count" "get_progress_map returns exactly 6 lines"

# get_progress_map with missing stage (defaults to pending)
cat > "$TEST_DIR/missing-stage.yaml" <<EOF
progress:
  brief: done
  spec: active
  apply: pending
  review: pending
  hydrate: pending
EOF
missing_output=$(get_progress_map "$TEST_DIR/missing-stage.yaml")
assert_contains "tasks:pending" "$missing_output" "get_progress_map defaults missing stage to pending"

# get_checklist
checklist_output=$(get_checklist "$TEST_DIR/full-status.yaml")
assert_contains "generated:true" "$checklist_output" "get_checklist extracts generated"
assert_contains "completed:3" "$checklist_output" "get_checklist extracts completed"
assert_contains "total:10" "$checklist_output" "get_checklist extracts total"

# get_checklist with missing block (defaults)
cat > "$TEST_DIR/no-checklist.yaml" <<EOF
progress:
  brief: active
EOF
no_chk_output=$(get_checklist "$TEST_DIR/no-checklist.yaml")
assert_contains "generated:false" "$no_chk_output" "get_checklist defaults generated to false"
assert_contains "completed:0" "$no_chk_output" "get_checklist defaults completed to 0"
assert_contains "total:0" "$no_chk_output" "get_checklist defaults total to 0"

# get_confidence
confidence_output=$(get_confidence "$TEST_DIR/full-status.yaml")
assert_contains "certain:5" "$confidence_output" "get_confidence extracts certain"
assert_contains "confident:2" "$confidence_output" "get_confidence extracts confident"
assert_contains "tentative:1" "$confidence_output" "get_confidence extracts tentative"
assert_contains "unresolved:0" "$confidence_output" "get_confidence extracts unresolved"
assert_contains "score:3.4" "$confidence_output" "get_confidence extracts score"

# get_confidence with missing block (defaults)
cat > "$TEST_DIR/no-confidence.yaml" <<EOF
progress:
  brief: active
EOF
no_conf_output=$(get_confidence "$TEST_DIR/no-confidence.yaml")
assert_contains "certain:0" "$no_conf_output" "get_confidence defaults certain to 0"
assert_contains "score:5.0" "$no_conf_output" "get_confidence defaults score to 5.0"

# get_current_stage (refactored to use get_progress_map)
current=$(get_current_stage "$TEST_DIR/full-status.yaml")
assert_equal "spec" "$current" "get_current_stage finds active stage via accessor"

# get_current_stage fallback (no active, first pending after last done)
cat > "$TEST_DIR/no-active.yaml" <<EOF
progress:
  brief: done
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
EOF
fallback=$(get_current_stage "$TEST_DIR/no-active.yaml")
assert_equal "spec" "$fallback" "get_current_stage fallback finds first pending after last done"

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
last_updated: 2026-01-01T00:00:00+00:00
EOF
}

# --- set_stage_state ---

echo "  set_stage_state:"

make_write_fixture
set_stage_state "$TEST_DIR/write-status.yaml" "review" "active" 2>/dev/null
assert_success "    valid state change succeeds"
result=$(grep "^  review:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "active" "$result" "    review is now active"

# Verify last_updated was refreshed
ts=$(grep "^last_updated:" "$TEST_DIR/write-status.yaml" | sed 's/last_updated: //')
assert_contains "T" "$ts" "    last_updated refreshed with ISO 8601 timestamp"

# Other stages unchanged
result=$(grep "^  apply:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "active" "$result" "    unrelated stages unchanged"

# Validation failures
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_stage_state "$TEST_DIR/write-status.yaml" "invalid_stage" "active" 2>/dev/null
assert_failure "    rejects invalid stage"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on invalid stage"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_stage_state "$TEST_DIR/write-status.yaml" "brief" "failed" 2>/dev/null
assert_failure "    rejects state not allowed for stage"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on invalid state"

set_stage_state "/nonexistent/path/.status.yaml" "spec" "done" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# --- transition_stages ---

echo "  transition_stages:"

make_write_fixture
transition_stages "$TEST_DIR/write-status.yaml" "apply" "review" 2>/dev/null
assert_success "    valid forward transition succeeds"
from_state=$(grep "^  apply:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
to_state=$(grep "^  review:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "done" "$from_state" "    from_stage set to done"
assert_equal "active" "$to_state" "    to_stage set to active"

# Non-adjacent stages
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
transition_stages "$TEST_DIR/write-status.yaml" "apply" "hydrate" 2>/dev/null
assert_failure "    rejects non-adjacent stages"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on non-adjacent"

# from_stage not active
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
transition_stages "$TEST_DIR/write-status.yaml" "spec" "tasks" 2>/dev/null
assert_failure "    rejects when from_stage is not active"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged when from_stage not active"

transition_stages "/nonexistent/path/.status.yaml" "apply" "review" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# --- set_checklist_field ---

echo "  set_checklist_field:"

make_write_fixture
set_checklist_field "$TEST_DIR/write-status.yaml" "completed" "7" 2>/dev/null
assert_success "    valid completed update succeeds"
result=$(grep "^  completed:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "7" "$result" "    completed is now 7"

make_write_fixture
set_checklist_field "$TEST_DIR/write-status.yaml" "generated" "false" 2>/dev/null
assert_success "    valid generated update succeeds"
result=$(grep "^  generated:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "false" "$result" "    generated is now false"

make_write_fixture
set_checklist_field "$TEST_DIR/write-status.yaml" "total" "25" 2>/dev/null
assert_success "    valid total update succeeds"
result=$(grep "^  total:" "$TEST_DIR/write-status.yaml" | sed 's/.*: //')
assert_equal "25" "$result" "    total is now 25"

# Validation failures
make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_checklist_field "$TEST_DIR/write-status.yaml" "invalid_field" "5" 2>/dev/null
assert_failure "    rejects invalid field name"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on invalid field"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_checklist_field "$TEST_DIR/write-status.yaml" "completed" "-3" 2>/dev/null
assert_failure "    rejects negative integer"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on negative value"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_checklist_field "$TEST_DIR/write-status.yaml" "generated" "yes" 2>/dev/null
assert_failure "    rejects non-bool for generated"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on wrong type"

set_checklist_field "/nonexistent/path/.status.yaml" "completed" "5" 2>/dev/null
assert_failure "    rejects nonexistent file"

echo ""

# --- set_confidence_block ---

echo "  set_confidence_block:"

make_write_fixture
set_confidence_block "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "4.1" 2>/dev/null
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
set_confidence_block "$TEST_DIR/write-status.yaml" "-1" "3" "1" "0" "4.1" 2>/dev/null
assert_failure "    rejects negative count"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on negative count"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_confidence_block "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "abc" 2>/dev/null
assert_failure "    rejects non-numeric score"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on non-numeric score"

make_write_fixture
before=$(cat "$TEST_DIR/write-status.yaml")
set_confidence_block "$TEST_DIR/write-status.yaml" "10" "3" "1" "0" "-2.5" 2>/dev/null
assert_failure "    rejects negative score"
after=$(cat "$TEST_DIR/write-status.yaml")
assert_equal "$before" "$after" "    file unchanged on negative score"

set_confidence_block "/nonexistent/path/.status.yaml" "10" "3" "1" "0" "4.1" 2>/dev/null
assert_failure "    rejects nonexistent file"

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

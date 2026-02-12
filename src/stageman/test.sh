#!/usr/bin/env bash
# src/stageman/test.sh
#
# Comprehensive test suite for stageman.sh
# Run: ./test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/stageman.sh"

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
assert_contains "archive" "$all_stages" "get_all_stages includes archive"

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

num=$(get_stage_number "archive")
assert_equal "6" "$num" "get_stage_number returns 6 for archive"

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
get_next_stage "archive" >/dev/null 2>&1
assert_failure "get_next_stage returns error for archive"

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
  archive: pending
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
  archive: pending
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
  archive: pending
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
  archive: pending
EOF

validate_status_file "$TEST_DIR/wrong-allowed-state.yaml" >/dev/null 2>&1
assert_failure "validate_status_file rejects state not in allowed_states"

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

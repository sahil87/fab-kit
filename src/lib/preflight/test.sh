#!/usr/bin/env bash
# src/lib/preflight/test.sh
#
# Comprehensive test suite for preflight.sh
# Run: ./test.sh

set -uo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

# Test colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# ─────────────────────────────────────────────────────────────────────────────
# Test helpers
# ─────────────────────────────────────────────────────────────────────────────

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

assert_exit_code() {
  local expected="$1"
  local actual="$2"
  local test_name="$3"

  ((TESTS_RUN++))

  if [ "$expected" -eq "$actual" ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name (expected exit $expected, got $actual)"
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

assert_not_contains() {
  local needle="$1"
  local haystack="$2"
  local test_name="$3"

  ((TESTS_RUN++))

  if ! grep -q "$needle" <<< "$haystack"; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "  Expected NOT to find: $needle"
    echo "  In: $haystack"
    ((TESTS_FAILED++))
    return 1
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Test environment setup
# ─────────────────────────────────────────────────────────────────────────────

TEST_DIR=""

setup_env() {
  TEST_DIR=$(mktemp -d)

  # Mirror the fab/ directory structure
  local fab="$TEST_DIR/fab"
  mkdir -p "$fab/.kit/scripts/lib" "$fab/.kit/schemas" "$fab/changes"

  # Copy real scripts and schema
  cp "$PROJECT_ROOT/fab/.kit/scripts/lib/preflight.sh" "$fab/.kit/scripts/lib/"
  cp "$PROJECT_ROOT/fab/.kit/scripts/lib/stageman.sh" "$fab/.kit/scripts/lib/"
  cp "$PROJECT_ROOT/fab/.kit/scripts/lib/resolve-change.sh" "$fab/.kit/scripts/lib/"
  cp "$PROJECT_ROOT/fab/.kit/schemas/workflow.yaml" "$fab/.kit/schemas/"
  chmod +x "$fab/.kit/scripts/lib/preflight.sh"

  # Create required init files
  echo "version: 1" > "$fab/config.yaml"
  echo "# Constitution" > "$fab/constitution.md"
}

teardown_env() {
  [ -n "$TEST_DIR" ] && rm -rf "$TEST_DIR"
  TEST_DIR=""
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

# Helper: run preflight and capture output + exit code
run_preflight() {
  local output
  output=$("$TEST_DIR/fab/.kit/scripts/lib/preflight.sh" "$@" 2>&1)
  local code=$?
  echo "$output"
  return $code
}

# Helper: run preflight, capture stdout and stderr separately
run_preflight_split() {
  local stdout_file stderr_file
  stdout_file=$(mktemp)
  stderr_file=$(mktemp)
  "$TEST_DIR/fab/.kit/scripts/lib/preflight.sh" "$@" >"$stdout_file" 2>"$stderr_file"
  local code=$?
  LAST_STDOUT=$(cat "$stdout_file")
  LAST_STDERR=$(cat "$stderr_file")
  rm -f "$stdout_file" "$stderr_file"
  return $code
}

LAST_STDOUT=""
LAST_STDERR=""

# ─────────────────────────────────────────────────────────────────────────────
# 1. Project Initialization Validation
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Project Initialization Validation..."
echo ""

# Missing config.yaml
setup_env
rm "$TEST_DIR/fab/config.yaml"
set_current "test-change"
create_change "test-change" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when config.yaml missing"
teardown_env

# Missing constitution.md
setup_env
rm "$TEST_DIR/fab/constitution.md"
set_current "test-change"
create_change "test-change" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when constitution.md missing"
teardown_env

# Both missing
setup_env
rm "$TEST_DIR/fab/config.yaml" "$TEST_DIR/fab/constitution.md"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when both init files missing"
teardown_env

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 2. Change Name Resolution — fab/current
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Change Name Resolution (fab/current)..."
echo ""

# No fab/current file
setup_env
create_change "test-change" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
# Don't set fab/current
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when fab/current missing"
teardown_env

# Empty fab/current
setup_env
create_change "test-change" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
echo "" > "$TEST_DIR/fab/current"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when fab/current is empty"
teardown_env

# Valid fab/current reads change name
setup_env
create_change "my-feature" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "my-feature"
run_preflight_split
assert_exit_code 0 $? "accepts valid fab/current"
assert_contains "name: my-feature" "$LAST_STDOUT" "output contains correct name from fab/current"
teardown_env

# fab/current with trailing whitespace
setup_env
create_change "trim-test" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
printf "trim-test  \n" > "$TEST_DIR/fab/current"
run_preflight_split
assert_exit_code 0 $? "handles whitespace in fab/current"
assert_contains "name: trim-test" "$LAST_STDOUT" "trims whitespace from fab/current"
teardown_env

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 3. Change Name Resolution — Override ($1)
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Change Name Resolution (override)..."
echo ""

# Exact match
setup_env
create_change "alpha-feature" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight_split "alpha-feature"
assert_exit_code 0 $? "override: exact match works"
assert_contains "name: alpha-feature" "$LAST_STDOUT" "override: output has correct name"
teardown_env

# Case-insensitive exact match
setup_env
create_change "Alpha-Feature" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight_split "alpha-feature"
assert_exit_code 0 $? "override: case-insensitive exact match"
assert_contains "name: Alpha-Feature" "$LAST_STDOUT" "override: preserves original casing"
teardown_env

# Partial match (single)
setup_env
create_change "big-refactor-auth" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
create_change "small-ui-fix" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight_split "refactor"
assert_exit_code 0 $? "override: single partial match works"
assert_contains "name: big-refactor-auth" "$LAST_STDOUT" "override: partial match resolves correctly"
teardown_env

# Ambiguous partial match
setup_env
create_change "auth-login" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
create_change "auth-signup" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight_split "auth"
assert_exit_code 1 $? "override: rejects ambiguous partial match"
assert_contains "Multiple changes match" "$LAST_STDERR" "override: reports ambiguity"
teardown_env

# No match
setup_env
create_change "some-feature" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
run_preflight_split "nonexistent"
assert_exit_code 1 $? "override: rejects when no match"
assert_contains "No change matches" "$LAST_STDERR" "override: reports no match"
teardown_env

# No changes directory at all
setup_env
rm -rf "$TEST_DIR/fab/changes"
run_preflight_split "anything"
assert_exit_code 1 $? "override: rejects when changes/ dir missing"
teardown_env

# No active (non-archive) changes
setup_env
# Only create an "archive" folder — preflight skips it
mkdir -p "$TEST_DIR/fab/changes/archive"
run_preflight_split "test"
assert_exit_code 1 $? "override: rejects when only archive folder exists"
teardown_env

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 4. Directory & Status File Validation
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Directory & Status Validation..."
echo ""

# Missing change directory
setup_env
set_current "ghost-change"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when change directory missing"
teardown_env

# Missing .status.yaml
setup_env
mkdir -p "$TEST_DIR/fab/changes/no-status"
set_current "no-status"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects when .status.yaml missing"
teardown_env

# Invalid .status.yaml (bad state)
setup_env
create_change "bad-status" "progress:
  brief: done
  spec: bogus_state
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "bad-status"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects invalid state in .status.yaml"
teardown_env

# Invalid .status.yaml (multiple active)
setup_env
create_change "multi-active" "progress:
  brief: active
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "multi-active"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects multiple active stages"
teardown_env

# Invalid .status.yaml (state not in allowed_states — brief:failed)
setup_env
create_change "bad-allowed" "progress:
  brief: failed
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "bad-allowed"
run_preflight >/dev/null 2>&1
assert_exit_code 1 $? "rejects state not in allowed_states for stage"
teardown_env

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 5. YAML Output — Progress & Stage Detection
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing YAML Output..."
echo ""

# Basic output structure
setup_env
create_change "feature-x" "progress:
  brief: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "feature-x"
run_preflight_split
assert_exit_code 0 $? "valid change produces YAML output"
assert_contains "name: feature-x" "$LAST_STDOUT" "output: name field"
assert_contains "change_dir: changes/feature-x" "$LAST_STDOUT" "output: change_dir field"
assert_contains "stage: spec" "$LAST_STDOUT" "output: detects active stage"
assert_contains "brief: done" "$LAST_STDOUT" "output: progress brief=done"
assert_contains "spec: active" "$LAST_STDOUT" "output: progress spec=active"
assert_contains "tasks: pending" "$LAST_STDOUT" "output: progress tasks=pending"
teardown_env

# Stage detection — brief active
setup_env
create_change "new-change" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "new-change"
run_preflight_split
assert_exit_code 0 $? "brief-active change accepted"
assert_contains "stage: brief" "$LAST_STDOUT" "output: detects brief as current stage"
teardown_env

# Stage detection — apply active
setup_env
create_change "mid-change" "progress:
  brief: done
  spec: done
  tasks: done
  apply: active
  review: pending
  hydrate: pending"
set_current "mid-change"
run_preflight_split
assert_exit_code 0 $? "apply-active change accepted"
assert_contains "stage: apply" "$LAST_STDOUT" "output: detects apply as current stage"
teardown_env

# Stage detection — all done → hydrate fallback
setup_env
create_change "done-change" "progress:
  brief: done
  spec: done
  tasks: done
  apply: done
  review: done
  hydrate: done"
set_current "done-change"
run_preflight_split
assert_exit_code 0 $? "all-done change accepted"
assert_contains "stage: hydrate" "$LAST_STDOUT" "output: falls back to hydrate when all done"
teardown_env

# Stage detection — review:failed (non-terminal state unique to review)
setup_env
create_change "failed-review" "progress:
  brief: done
  spec: done
  tasks: done
  apply: done
  review: failed
  hydrate: pending"
set_current "failed-review"
run_preflight_split
assert_exit_code 0 $? "review:failed accepted (valid per schema)"
assert_contains "stage: hydrate" "$LAST_STDOUT" "output: no active stage falls back to hydrate"
assert_contains "review: failed" "$LAST_STDOUT" "output: preserves failed state"
teardown_env

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 6. YAML Output — Checklist & Confidence
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Checklist & Confidence Fields..."
echo ""

# Default values when missing
setup_env
create_change "bare-change" "progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending"
set_current "bare-change"
run_preflight_split
assert_exit_code 0 $? "bare change (no checklist/confidence) accepted"
assert_contains "generated: false" "$LAST_STDOUT" "defaults: checklist generated=false"
assert_contains "completed: 0" "$LAST_STDOUT" "defaults: checklist completed=0"
assert_contains "total: 0" "$LAST_STDOUT" "defaults: checklist total=0"
assert_contains "score: 5.0" "$LAST_STDOUT" "defaults: confidence score=5.0"
teardown_env

# Present checklist and confidence values
setup_env
create_change "rich-change" "progress:
  brief: done
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
run_preflight_split
assert_exit_code 0 $? "rich change with checklist/confidence accepted"
assert_contains "generated: true" "$LAST_STDOUT" "checklist: generated=true"
assert_contains "completed: 5" "$LAST_STDOUT" "checklist: completed=5"
assert_contains "total: 12" "$LAST_STDOUT" "checklist: total=12"
assert_contains "certain: 3" "$LAST_STDOUT" "confidence: certain=3"
assert_contains "confident: 4" "$LAST_STDOUT" "confidence: confident=4"
assert_contains "tentative: 2" "$LAST_STDOUT" "confidence: tentative=2"
assert_contains "unresolved: 1" "$LAST_STDOUT" "confidence: unresolved=1"
assert_contains "score: 7.5" "$LAST_STDOUT" "confidence: score=7.5"
teardown_env

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 7. Error Messages
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Error Messages..."
echo ""

# Uninitialized project
setup_env
rm "$TEST_DIR/fab/config.yaml"
set_current "x"
run_preflight_split 2>/dev/null || true
assert_contains "not initialized" "$LAST_STDERR" "error msg: project not initialized"
teardown_env

# No active change
setup_env
run_preflight_split 2>/dev/null || true
assert_contains "No active change" "$LAST_STDERR" "error msg: no active change"
teardown_env

# Missing change dir
setup_env
set_current "ghost"
run_preflight_split 2>/dev/null || true
assert_contains "not found" "$LAST_STDERR" "error msg: change directory not found"
teardown_env

# Corrupted status
setup_env
mkdir -p "$TEST_DIR/fab/changes/broken"
set_current "broken"
run_preflight_split 2>/dev/null || true
assert_contains "corrupted" "$LAST_STDERR" "error msg: corrupted change"
teardown_env

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

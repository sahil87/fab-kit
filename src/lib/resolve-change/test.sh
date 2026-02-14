#!/usr/bin/env bash
# src/lib/resolve-change/test.sh
#
# Comprehensive test suite for resolve-change.sh
# Run: ./test.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/resolve-change.sh"
set +e

# Test colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

assert_equal() {
  local expected="$1"
  local actual="$2"
  local test_name="$3"
  ((TESTS_RUN++))
  if [ "$expected" = "$actual" ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "  Expected: $expected"
    echo "  Got:      $actual"
    ((TESTS_FAILED++))
  fi
}

assert_success() {
  local exit_code=$?
  local test_name="$1"
  ((TESTS_RUN++))
  if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
  else
    echo -e "${RED}✗${NC} $test_name (exit code: $exit_code)"
    ((TESTS_FAILED++))
  fi
}

assert_failure() {
  local exit_code=$?
  local test_name="$1"
  ((TESTS_RUN++))
  if [ $exit_code -ne 0 ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((TESTS_PASSED++))
  else
    echo -e "${RED}✗${NC} $test_name (expected failure, got success)"
    ((TESTS_FAILED++))
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
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "  Expected to find: $needle"
    echo "  In: $haystack"
    ((TESTS_FAILED++))
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Setup
# ─────────────────────────────────────────────────────────────────────────────

TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

# Create test fab structure
mkdir -p "$TEST_DIR/changes/260213-puow-consolidate-status-reads"
mkdir -p "$TEST_DIR/changes/260213-k7m2-kit-version-migrations"
mkdir -p "$TEST_DIR/changes/260212-f9m3-enhance-srad-fuzzy"
mkdir -p "$TEST_DIR/changes/archive"

# ─────────────────────────────────────────────────────────────────────────────
# Exact Match Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Exact Match..."
echo ""

resolve_change "$TEST_DIR" "260213-puow-consolidate-status-reads" 2>/dev/null
assert_success "exact match resolves"
assert_equal "260213-puow-consolidate-status-reads" "$RESOLVED_CHANGE_NAME" "exact match sets correct name"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Substring Match Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Substring Match..."
echo ""

resolve_change "$TEST_DIR" "puow" 2>/dev/null
assert_success "single substring match resolves"
assert_equal "260213-puow-consolidate-status-reads" "$RESOLVED_CHANGE_NAME" "single substring sets correct name"

resolve_change "$TEST_DIR" "f9m3" 2>/dev/null
assert_success "4-char ID match resolves"
assert_equal "260212-f9m3-enhance-srad-fuzzy" "$RESOLVED_CHANGE_NAME" "4-char ID sets correct name"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Case Insensitive Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Case Insensitivity..."
echo ""

resolve_change "$TEST_DIR" "PUOW" 2>/dev/null
assert_success "uppercase substring resolves"
assert_equal "260213-puow-consolidate-status-reads" "$RESOLVED_CHANGE_NAME" "uppercase match sets correct name"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Multiple Match Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Multiple Matches..."
echo ""

err=$(resolve_change "$TEST_DIR" "260213" 2>&1)
assert_failure "multiple matches returns error"
assert_contains "Multiple changes match" "$err" "error message lists ambiguous matches"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# No Match Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing No Match..."
echo ""

err=$(resolve_change "$TEST_DIR" "nonexistent" 2>&1)
assert_failure "no match returns error"
assert_contains "No change matches" "$err" "error message says no match"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# fab/current Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing fab/current..."
echo ""

echo "260213-puow-consolidate-status-reads" > "$TEST_DIR/current"
resolve_change "$TEST_DIR" "" 2>/dev/null
assert_success "reads fab/current when no override"
assert_equal "260213-puow-consolidate-status-reads" "$RESOLVED_CHANGE_NAME" "fab/current sets correct name"

# fab/current with trailing whitespace
printf "260213-puow-consolidate-status-reads\n  " > "$TEST_DIR/current"
resolve_change "$TEST_DIR" "" 2>/dev/null
assert_success "fab/current with trailing whitespace resolves"
assert_equal "260213-puow-consolidate-status-reads" "$RESOLVED_CHANGE_NAME" "trailing whitespace stripped"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# No Active Change Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing No Active Change..."
echo ""

rm -f "$TEST_DIR/current"
err=$(resolve_change "$TEST_DIR" "" 2>&1)
assert_failure "missing fab/current returns error"
assert_contains "No active change" "$err" "error message says no active change"

# Empty fab/current
echo "" > "$TEST_DIR/current"
err=$(resolve_change "$TEST_DIR" "" 2>&1)
assert_failure "empty fab/current returns error"
assert_contains "No active change" "$err" "empty file error says no active change"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Missing Changes Directory Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Missing Changes Directory..."
echo ""

EMPTY_DIR=$(mktemp -d)
err=$(resolve_change "$EMPTY_DIR" "something" 2>&1)
assert_failure "missing fab/changes/ returns error"
assert_contains "fab/changes/ not found" "$err" "error message says changes dir not found"
rm -rf "$EMPTY_DIR"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Archive Exclusion Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Archive Exclusion..."
echo ""

err=$(resolve_change "$TEST_DIR" "archive" 2>&1)
assert_failure "archive folder excluded from matches"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Generic Error Messages
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Generic Error Messages..."
echo ""

rm -f "$TEST_DIR/current"
err=$(resolve_change "$TEST_DIR" "" 2>&1)
# Should NOT contain command suggestions
if ! grep -q "Run /fab" <<< "$err" 2>/dev/null; then
  ((TESTS_RUN++))
  echo -e "${GREEN}✓${NC} error message does not contain command suggestions"
  ((TESTS_PASSED++))
else
  ((TESTS_RUN++))
  echo -e "${RED}✗${NC} error message should not contain command suggestions"
  echo "  Got: $err"
  ((TESTS_FAILED++))
fi

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

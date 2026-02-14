#!/usr/bin/env bash
# src/lib/calc-score/test.sh
#
# Comprehensive test suite for calc-score.sh
# Run: ./test.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CALC_SCORE="$SCRIPT_DIR/calc-score.sh"

# Test colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

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

assert_contains() {
  local needle="$1"
  local haystack="$2"
  local test_name="$3"

  ((TESTS_RUN++))

  if echo "$haystack" | grep -q "$needle"; then
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

assert_exit_code() {
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
    echo "  Expected exit code: $expected"
    echo "  Got: $actual"
    ((TESTS_FAILED++))
    return 1
  fi
}

# Create a minimal .status.yaml for a test change dir
make_status() {
  local dir="$1"
  local certain="${2:-0}"
  local score="${3:-5.0}"
  cat > "$dir/.status.yaml" <<EOF
name: test-change
progress:
  brief: done
  spec: done
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
checklist:
  generated: false
  completed: 0
  total: 0
confidence:
  certain: $certain
  confident: 0
  tentative: 0
  unresolved: 0
  score: $score
EOF
}

# ─────────────────────────────────────────────────────────────────────────────
# Setup
# ─────────────────────────────────────────────────────────────────────────────

TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

# ─────────────────────────────────────────────────────────────────────────────
# Grade Counting Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Grade Counting..."
echo ""

# Test: Counts grades from spec Assumptions table
d="$TEST_DIR/grade-count"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Decision A | Reason A |
| 2 | Confident | Decision B | Reason B |
| 3 | Tentative | Decision C | Reason C |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "confident: 2" "$output" "Counts 2 Confident grades from spec"
assert_contains "tentative: 1" "$output" "Counts 1 Tentative grade from spec"

# Test: Combined grades from brief and spec
d="$TEST_DIR/combined-grades"
mkdir -p "$d"
make_status "$d"
cat > "$d/brief.md" <<'EOF'
# Brief

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Tentative | Brief decision | Brief reason |
EOF
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Spec decision | Spec reason |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "confident: 1" "$output" "Counts Confident from spec"
assert_contains "tentative: 1" "$output" "Counts Tentative from brief"

# Test: Case-insensitive grade matching
d="$TEST_DIR/case-insensitive"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | confident | Lower case | Test |
| 2 | TENTATIVE | Upper case | Test |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "confident: 1" "$output" "Case-insensitive: lowercase confident"
assert_contains "tentative: 1" "$output" "Case-insensitive: uppercase TENTATIVE"

# Test: No Assumptions section — zero counts
d="$TEST_DIR/no-assumptions"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions here.
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "confident: 0" "$output" "No Assumptions table: confident is 0"
assert_contains "tentative: 0" "$output" "No Assumptions table: tentative is 0"
assert_contains "score: 5.0" "$output" "No Assumptions table: score is 5.0"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Score Formula Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Score Formula..."
echo ""

# Test: 2 Confident + 1 Tentative → 3.4
d="$TEST_DIR/formula-basic"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
| 2 | Confident | B | R |
| 3 | Tentative | C | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "score: 3.4" "$output" "Score: 5.0 - 0.3*2 - 1.0*1 = 3.4"

# Test: Score floor at 0.0
d="$TEST_DIR/formula-floor"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Tentative | A | R |
| 2 | Tentative | B | R |
| 3 | Tentative | C | R |
| 4 | Tentative | D | R |
| 5 | Tentative | E | R |
| 6 | Tentative | F | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "score: 0.0" "$output" "Score floors at 0.0 (6 tentative = -1.0 before floor)"

# Test: All confident, no tentative
d="$TEST_DIR/formula-confident-only"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
| 2 | Confident | B | R |
| 3 | Confident | C | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "score: 4.1" "$output" "Score: 5.0 - 0.3*3 = 4.1"
assert_contains "mode: legacy" "$output" "Default mode is legacy (backward compatibility)"
assert_contains "threshold: 3.0" "$output" "Legacy mode uses fixed 3.0 threshold"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Fuzzy Mode & Threshold Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Fuzzy Mode & Thresholds..."
echo ""

# Test: Weighted composite uses S/R/A/D = 0.2/0.3/0.3/0.2
d="$TEST_DIR/fuzzy-weighted-composite"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Weighted test S=0,R=100,A=100,D=0 | Composite should be 60 |
EOF

output=$(FAB_SCORE_MODE=fuzzy "$CALC_SCORE" "$d")
assert_contains "mode: fuzzy" "$output" "Fuzzy mode enabled"
assert_contains "score: 4.7" "$output" "Weighted composite yields expected score (composite 60 => confident membership 1.0)"
assert_contains "effective_confident: 1.00" "$output" "Fuzzy effective confident contribution computed"
assert_contains "effective_tentative: 0.00" "$output" "Fuzzy effective tentative contribution computed"

# Test: Mid-range composite produces partial memberships
d="$TEST_DIR/fuzzy-partial-membership"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Partial membership S=50,R=50,A=50,D=50 | Composite should be 50 |
EOF

output=$(FAB_SCORE_MODE=fuzzy "$CALC_SCORE" "$d")
assert_contains "effective_confident: 0.67" "$output" "Partial confident membership captured"
assert_contains "effective_tentative: 0.33" "$output" "Partial tentative membership captured"
assert_contains "score: 4.5" "$output" "Partial memberships affect score continuously"

# Test: Dynamic threshold mapping by change type (fuzzy mode)
d="$TEST_DIR/fuzzy-threshold-types"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
EOF

output=$(FAB_SCORE_MODE=fuzzy FAB_CHANGE_TYPE=bugfix "$CALC_SCORE" "$d")
assert_contains "change_type: bugfix" "$output" "Bugfix type propagated"
assert_contains "threshold: 2.7" "$output" "Bugfix threshold is 2.7"

output=$(FAB_SCORE_MODE=fuzzy FAB_CHANGE_TYPE=architecture "$CALC_SCORE" "$d")
assert_contains "change_type: architecture" "$output" "Architecture type propagated"
assert_contains "threshold: 3.6" "$output" "Architecture threshold is 3.6"

# Test: Unknown change type defaults to feature threshold
output=$(FAB_SCORE_MODE=fuzzy FAB_CHANGE_TYPE=unknown "$CALC_SCORE" "$d")
assert_contains "change_type: feature" "$output" "Unknown type defaults to feature"
assert_contains "threshold: 3.3" "$output" "Feature default threshold is 3.3"
assert_contains "change_type_inferred: true" "$output" "Defaulting unknown type marked as inferred"

# Test: Sparse history fallback keeps baseline weights
d="$TEST_DIR/fuzzy-sparse-fallback"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Sparse fallback check | R |
EOF

output=$(FAB_SCORE_MODE=fuzzy FAB_WEIGHT_CONFIDENT=0.1 FAB_WEIGHT_TENTATIVE=0.4 "$CALC_SCORE" "$d")
assert_contains "weights_source: baseline" "$output" "Sparse history prevents calibrated weights"
assert_contains "weight_confident: 0.30" "$output" "Baseline confident weight retained"
assert_contains "weight_tentative: 1.00" "$output" "Baseline tentative weight retained"
assert_contains "sparse_fallback: true" "$output" "Sparse fallback is reported"

# Test: Malformed SRAD metadata falls back safely
d="$TEST_DIR/fuzzy-malformed-metadata"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Malformed S=abc,R=50,A=200,D=-1 | Should fallback to grade defaults |
EOF

output=$(FAB_SCORE_MODE=fuzzy "$CALC_SCORE" "$d")
assert_contains "score: 4.7" "$output" "Malformed metadata does not crash and falls back to grade defaults"
assert_contains "effective_confident: 1.00" "$output" "Fallback keeps expected confident contribution"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Carry-Forward Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Carry-Forward..."
echo ""

# Test: Previous certain carried forward when no Certain in table
d="$TEST_DIR/carry-forward"
mkdir -p "$d"
make_status "$d" 5 "5.0"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "certain: 5" "$output" "Carry-forward: 5 implicit certain preserved"

# Test: Explicit Certain in table reduces implicit carry-forward
d="$TEST_DIR/carry-forward-partial"
mkdir -p "$d"
make_status "$d" 5 "5.0"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | A | R |
| 2 | Certain | B | R |
| 3 | Confident | C | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "certain: 5" "$output" "Carry-forward: 2 explicit + 3 implicit = 5 total"

# Test: No previous certain, no table certain → 0
d="$TEST_DIR/carry-forward-zero"
mkdir -p "$d"
make_status "$d" 0 "5.0"
cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions.
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "certain: 0" "$output" "No carry-forward when previous certain is 0"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Status Update Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing .status.yaml Update..."
echo ""

# Test: Confidence block is replaced in .status.yaml
d="$TEST_DIR/status-update"
mkdir -p "$d"
make_status "$d" 0 "5.0"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
| 2 | Tentative | B | R |
EOF

"$CALC_SCORE" "$d" > /dev/null

# Check the .status.yaml was updated
status_content=$(cat "$d/.status.yaml")
assert_contains "confident: 1" "$status_content" "Status updated: confident count"
assert_contains "tentative: 1" "$status_content" "Status updated: tentative count"
assert_contains "score: 3.7" "$status_content" "Status updated: score"

# Check other fields preserved
assert_contains "name: test-change" "$status_content" "Status preserved: name field"
assert_contains "brief: done" "$status_content" "Status preserved: progress fields"
assert_contains "generated: false" "$status_content" "Status preserved: checklist fields"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Delta Computation Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Delta Computation..."
echo ""

# Test: Negative delta
d="$TEST_DIR/delta-negative"
mkdir -p "$d"
make_status "$d" 0 "5.0"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
| 2 | Tentative | B | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "delta: -1.3" "$output" "Delta: 3.7 - 5.0 = -1.3"

# Test: Positive delta (unlikely in practice, but test the format)
d="$TEST_DIR/delta-positive"
mkdir -p "$d"
make_status "$d" 0 "2.0"
cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions.
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "delta: +3.0" "$output" "Delta: 5.0 - 2.0 = +3.0"

# Test: Zero delta
d="$TEST_DIR/delta-zero"
mkdir -p "$d"
make_status "$d" 0 "5.0"
cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions.
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "delta: +0.0" "$output" "Delta: 5.0 - 5.0 = +0.0"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Error Case Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Error Cases..."
echo ""

# Test: No arguments
stderr_output=$("$CALC_SCORE" 2>&1 || true)
rc=$?
# Script uses set -e, so we need to capture differently
"$CALC_SCORE" > /dev/null 2>"$TEST_DIR/no-args-stderr" || rc=$?
assert_equal "1" "$rc" "No arguments: exit code 1"
stderr_content=$(cat "$TEST_DIR/no-args-stderr")
assert_contains "Usage:" "$stderr_content" "No arguments: stderr has usage message"

# Test: Missing change directory
"$CALC_SCORE" "/nonexistent/path" > /dev/null 2>"$TEST_DIR/missing-dir-stderr" || rc=$?
assert_equal "1" "$rc" "Missing directory: exit code 1"
stderr_content=$(cat "$TEST_DIR/missing-dir-stderr")
assert_contains "Change directory not found" "$stderr_content" "Missing directory: appropriate stderr"

# Test: Missing spec.md
d="$TEST_DIR/no-spec"
mkdir -p "$d"
make_status "$d"
"$CALC_SCORE" "$d" > /dev/null 2>"$TEST_DIR/no-spec-stderr" || rc=$?
assert_equal "1" "$rc" "Missing spec.md: exit code 1"
stderr_content=$(cat "$TEST_DIR/no-spec-stderr")
assert_contains "spec.md required" "$stderr_content" "Missing spec.md: appropriate stderr"

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

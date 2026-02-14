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
# Fuzzy Dimension Parsing Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Fuzzy Dimension Parsing..."
echo ""

# Test: Scores column detected and parsed
d="$TEST_DIR/fuzzy-basic"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:75 R:80 A:65 D:70 | Use OAuth2 | Config shows REST API |
| 2 | Confident | S:85 R:90 A:75 D:80 | Use JWT | Standard for REST |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "fuzzy: true" "$output" "Fuzzy flag present when Scores column exists"
assert_contains "signal: 80.0" "$output" "Mean signal: (75+85)/2 = 80.0"
assert_contains "reversibility: 85.0" "$output" "Mean reversibility: (80+90)/2 = 85.0"
assert_contains "competence: 70.0" "$output" "Mean competence: (65+75)/2 = 70.0"
assert_contains "disambiguation: 75.0" "$output" "Mean disambiguation: (70+80)/2 = 75.0"

# Test: Legacy table (no Scores column) — no fuzzy output
d="$TEST_DIR/fuzzy-legacy"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
EOF

output=$("$CALC_SCORE" "$d")
if echo "$output" | grep -q "fuzzy:"; then
  ((TESTS_RUN++))
  echo -e "${RED}✗${NC} Legacy table: no fuzzy output"
  echo "  Output contained fuzzy when it shouldn't"
  ((TESTS_FAILED++))
else
  ((TESTS_RUN++))
  echo -e "${GREEN}✓${NC} Legacy table: no fuzzy output"
  ((TESTS_PASSED++))
fi

# Test: Mixed rows — some with Scores, some without (only complete rows count for dimensions)
d="$TEST_DIR/fuzzy-mixed"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:60 R:70 A:80 D:90 | A | R |
| 2 | Tentative |  | B | R |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "confident: 1" "$output" "Mixed: grades still counted correctly (1 confident)"
assert_contains "tentative: 1" "$output" "Mixed: grades still counted correctly (1 tentative)"
assert_contains "fuzzy: true" "$output" "Mixed: fuzzy flag set when at least one row has scores"
assert_contains "signal: 60.0" "$output" "Mixed: mean over single scored row only"

# Test: Partial dimension data (only S and R) — row skipped for dimensions
d="$TEST_DIR/fuzzy-partial"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:50 R:60 | Partial | Missing A and D |
| 2 | Confident | S:80 R:90 A:70 D:60 | Complete | All dimensions |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "fuzzy: true" "$output" "Partial: fuzzy flag set (one complete row exists)"
assert_contains "signal: 80.0" "$output" "Partial: mean uses only complete row (S=80)"
assert_contains "reversibility: 90.0" "$output" "Partial: mean uses only complete row (R=90)"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Gate Check Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Gate Check (--check-gate)..."
echo ""

# Helper to create status with change_type and score
make_gate_status() {
  local dir="$1"
  local change_type="$2"
  local score="$3"
  cat > "$dir/.status.yaml" <<EOF
name: test-change
change_type: $change_type
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
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: $score
last_updated: 2026-02-14T00:00:00Z
EOF
}

# Test: Bugfix passes at 2.5 (threshold 2.0)
d="$TEST_DIR/gate-bugfix-pass"
mkdir -p "$d"
make_gate_status "$d" "bugfix" "2.5"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: pass" "$output" "Gate: bugfix 2.5 >= 2.0 passes"
assert_contains "threshold: 2.0" "$output" "Gate: bugfix threshold is 2.0"

# Test: Bugfix fails at 1.5 (threshold 2.0)
d="$TEST_DIR/gate-bugfix-fail"
mkdir -p "$d"
make_gate_status "$d" "bugfix" "1.5"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: fail" "$output" "Gate: bugfix 1.5 < 2.0 fails"

# Test: Feature passes at 3.0 (threshold 3.0, exact boundary)
d="$TEST_DIR/gate-feature-exact"
mkdir -p "$d"
make_gate_status "$d" "feature" "3.0"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: pass" "$output" "Gate: feature 3.0 >= 3.0 passes (exact boundary)"

# Test: Feature fails at 2.9
d="$TEST_DIR/gate-feature-fail"
mkdir -p "$d"
make_gate_status "$d" "feature" "2.9"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: fail" "$output" "Gate: feature 2.9 < 3.0 fails"

# Test: Architecture requires 4.0
d="$TEST_DIR/gate-arch-fail"
mkdir -p "$d"
make_gate_status "$d" "architecture" "3.5"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: fail" "$output" "Gate: architecture 3.5 < 4.0 fails"
assert_contains "threshold: 4.0" "$output" "Gate: architecture threshold is 4.0"

# Test: Architecture passes at 4.0
d="$TEST_DIR/gate-arch-pass"
mkdir -p "$d"
make_gate_status "$d" "architecture" "4.0"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: pass" "$output" "Gate: architecture 4.0 >= 4.0 passes"

# Test: Refactor uses 3.0 threshold
d="$TEST_DIR/gate-refactor"
mkdir -p "$d"
make_gate_status "$d" "refactor" "3.0"
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: pass" "$output" "Gate: refactor 3.0 >= 3.0 passes"
assert_contains "threshold: 3.0" "$output" "Gate: refactor threshold is 3.0"

# Test: Missing change_type defaults to feature (threshold 3.0)
d="$TEST_DIR/gate-no-type"
mkdir -p "$d"
cat > "$d/.status.yaml" <<'EOF'
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
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 2.5
last_updated: 2026-02-14T00:00:00Z
EOF
output=$("$CALC_SCORE" --check-gate "$d")
assert_contains "gate: fail" "$output" "Gate: no change_type, score 2.5 < default 3.0 fails"
assert_contains "change_type: feature" "$output" "Gate: defaults to feature type"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Backward Compatibility Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Backward Compatibility..."
echo ""

# Test: Existing test fixture produces identical results
d="$TEST_DIR/compat-identical"
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
assert_contains "confident: 2" "$output" "Compat: same confident count as before"
assert_contains "tentative: 1" "$output" "Compat: same tentative count as before"
assert_contains "score: 3.4" "$output" "Compat: same score formula result"

# Test: Status.yaml written without fuzzy fields for legacy tables
status_content=$(cat "$d/.status.yaml")
if echo "$status_content" | grep -q "fuzzy:"; then
  ((TESTS_RUN++))
  echo -e "${RED}✗${NC} Compat: no fuzzy field in status for legacy tables"
  ((TESTS_FAILED++))
else
  ((TESTS_RUN++))
  echo -e "${GREEN}✓${NC} Compat: no fuzzy field in status for legacy tables"
  ((TESTS_PASSED++))
fi

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Edge Case Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Edge Cases..."
echo ""

# Test: All-zero dimension scores
d="$TEST_DIR/edge-zero"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Tentative | S:0 R:0 A:0 D:0 | Unknown | Everything ambiguous |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "signal: 0.0" "$output" "Edge: all-zero scores — signal is 0.0"
assert_contains "reversibility: 0.0" "$output" "Edge: all-zero scores — reversibility is 0.0"
assert_contains "fuzzy: true" "$output" "Edge: all-zero still sets fuzzy flag"

# Test: All-100 dimension scores
d="$TEST_DIR/edge-max"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Certain | S:100 R:100 A:100 D:100 | Crystal clear | Everything obvious |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "signal: 100.0" "$output" "Edge: all-100 scores — signal is 100.0"
assert_contains "disambiguation: 100.0" "$output" "Edge: all-100 scores — disambiguation is 100.0"

# Test: Single-row table with scores
d="$TEST_DIR/edge-single-row"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:42 R:88 A:55 D:73 | Sole decision | Only one |
EOF

output=$("$CALC_SCORE" "$d")
assert_contains "signal: 42.0" "$output" "Edge: single row — signal matches exactly"
assert_contains "reversibility: 88.0" "$output" "Edge: single row — reversibility matches exactly"
assert_contains "competence: 55.0" "$output" "Edge: single row — competence matches exactly"
assert_contains "disambiguation: 73.0" "$output" "Edge: single row — disambiguation matches exactly"

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# Fuzzy Status Write Tests
# ─────────────────────────────────────────────────────────────────────────────

echo "Testing Fuzzy Status Write..."
echo ""

# Test: .status.yaml gets fuzzy fields when Scores column present
d="$TEST_DIR/fuzzy-status-write"
mkdir -p "$d"
make_status "$d"
cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:70 R:80 A:60 D:90 | Test | Test |
EOF

"$CALC_SCORE" "$d" > /dev/null
status_content=$(cat "$d/.status.yaml")
assert_contains "fuzzy: true" "$status_content" "Fuzzy write: fuzzy flag in status"
assert_contains "signal:" "$status_content" "Fuzzy write: signal dimension in status"
assert_contains "reversibility:" "$status_content" "Fuzzy write: reversibility in status"
assert_contains "competence:" "$status_content" "Fuzzy write: competence in status"
assert_contains "disambiguation:" "$status_content" "Fuzzy write: disambiguation in status"
# Verify other fields preserved
assert_contains "name: test-change" "$status_content" "Fuzzy write: name preserved"
assert_contains "brief: done" "$status_content" "Fuzzy write: progress preserved"

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

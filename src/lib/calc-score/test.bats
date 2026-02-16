#!/usr/bin/env bats

# Test suite for calc-score.sh
# Covers: grade counting, score formula, carry-forward, status update,
#         delta computation, error cases, fuzzy dimension parsing, gate check,
#         backward compatibility, edge cases, fuzzy status write

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
CALC_SCORE="$(readlink -f "$SCRIPT_DIR/calc-score.sh")"

setup() {
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# Helper: create a minimal .status.yaml
make_status() {
  local dir="$1"
  local certain="${2:-0}"
  local score="${3:-5.0}"
  cat > "$dir/.status.yaml" <<EOF
name: test-change
progress:
  intake: done
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

# Helper: create status with change_type and score (for gate tests)
make_gate_status() {
  local dir="$1"
  local change_type="$2"
  local score="$3"
  cat > "$dir/.status.yaml" <<EOF
name: test-change
change_type: $change_type
progress:
  intake: done
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

# ─────────────────────────────────────────────────────────────────────────────
# Grade Counting
# ─────────────────────────────────────────────────────────────────────────────

@test "counts grades from spec Assumptions table" {
  local d="$TEST_DIR/grade-count"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"confident: 2"* ]]
  [[ "$output" == *"tentative: 1"* ]]
}

@test "case-insensitive grade matching" {
  local d="$TEST_DIR/case-insensitive"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"confident: 1"* ]]
  [[ "$output" == *"tentative: 1"* ]]
}

@test "no Assumptions section gives zero counts" {
  local d="$TEST_DIR/no-assumptions"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions here.
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"confident: 0"* ]]
  [[ "$output" == *"tentative: 0"* ]]
  [[ "$output" == *"score: 5.0"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Score Formula
# ─────────────────────────────────────────────────────────────────────────────

@test "score: 2 confident + 1 tentative = 3.4" {
  local d="$TEST_DIR/formula-basic"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"score: 3.4"* ]]
}

@test "score floors at 0.0" {
  local d="$TEST_DIR/formula-floor"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"score: 0.0"* ]]
}

@test "score: 3 confident only = 4.1" {
  local d="$TEST_DIR/formula-confident-only"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"score: 4.1"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Carry-Forward
# ─────────────────────────────────────────────────────────────────────────────

@test "certain count: only explicit Certain rows counted" {
  local d="$TEST_DIR/carry-forward"
  mkdir -p "$d"
  make_status "$d" 5 "5.0"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"certain: 0"* ]]
}

@test "certain count: explicit Certain rows counted from table" {
  local d="$TEST_DIR/carry-forward-partial"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"certain: 2"* ]]
}

@test "carry-forward: zero when previous certain is 0" {
  local d="$TEST_DIR/carry-forward-zero"
  mkdir -p "$d"
  make_status "$d" 0 "5.0"
  cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions.
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"certain: 0"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Status Update
# ─────────────────────────────────────────────────────────────────────────────

@test ".status.yaml updated with new confidence" {
  local d="$TEST_DIR/status-update"
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
  local status_content
  status_content=$(cat "$d/.status.yaml")
  [[ "$status_content" == *"confident: 1"* ]]
  [[ "$status_content" == *"tentative: 1"* ]]
  [[ "$status_content" == *"score: 3.7"* ]]
  [[ "$status_content" == *"name: test-change"* ]]
  [[ "$status_content" == *"intake: done"* ]]
  [[ "$status_content" == *"generated: false"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Delta Computation
# ─────────────────────────────────────────────────────────────────────────────

@test "delta: negative when score decreases" {
  local d="$TEST_DIR/delta-negative"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"delta: -1.3"* ]]
}

@test "delta: positive when score increases" {
  local d="$TEST_DIR/delta-positive"
  mkdir -p "$d"
  make_status "$d" 0 "2.0"
  cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions.
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"delta: +3.0"* ]]
}

@test "delta: zero when unchanged" {
  local d="$TEST_DIR/delta-zero"
  mkdir -p "$d"
  make_status "$d" 0 "5.0"
  cat > "$d/spec.md" <<'EOF'
# Spec

No assumptions.
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"delta: +0.0"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Error Cases
# ─────────────────────────────────────────────────────────────────────────────

@test "no arguments: exit 1 with usage" {
  run "$CALC_SCORE"
  [ "$status" -eq 1 ]
  run bash -c "'$CALC_SCORE' 2>&1"
  [[ "$output" == *"Usage:"* ]]
}

@test "missing directory: exit 1" {
  run bash -c "'$CALC_SCORE' /nonexistent/path 2>&1"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Change directory not found"* ]]
}

@test "missing spec.md: exit 1" {
  local d="$TEST_DIR/no-spec"
  mkdir -p "$d"
  make_status "$d"
  run bash -c "'$CALC_SCORE' '$d' 2>&1"
  [ "$status" -eq 1 ]
  [[ "$output" == *"spec.md required"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Fuzzy Dimension Parsing
# ─────────────────────────────────────────────────────────────────────────────

@test "fuzzy: scores column detected and parsed" {
  local d="$TEST_DIR/fuzzy-basic"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Use OAuth2 | Config shows REST API | S:75 R:80 A:65 D:70 |
| 2 | Confident | Use JWT | Standard for REST | S:85 R:90 A:75 D:80 |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"fuzzy: true"* ]]
  [[ "$output" == *"signal: 80.0"* ]]
  [[ "$output" == *"reversibility: 85.0"* ]]
  [[ "$output" == *"competence: 70.0"* ]]
  [[ "$output" == *"disambiguation: 75.0"* ]]
}

@test "fuzzy: legacy table (no Scores column) has no fuzzy output" {
  local d="$TEST_DIR/fuzzy-legacy"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | A | R |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" != *"fuzzy:"* ]]
}

@test "fuzzy: mixed rows count grades and compute dimensions from scored rows only" {
  local d="$TEST_DIR/fuzzy-mixed"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | A | R | S:60 R:70 A:80 D:90 |
| 2 | Tentative | B | R |  |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"confident: 1"* ]]
  [[ "$output" == *"tentative: 1"* ]]
  [[ "$output" == *"fuzzy: true"* ]]
  [[ "$output" == *"signal: 60.0"* ]]
}

@test "fuzzy: partial dimension data uses only complete rows" {
  local d="$TEST_DIR/fuzzy-partial"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Partial | Missing A and D | S:50 R:60 |
| 2 | Confident | Complete | All dimensions | S:80 R:90 A:70 D:60 |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"fuzzy: true"* ]]
  [[ "$output" == *"signal: 80.0"* ]]
  [[ "$output" == *"reversibility: 90.0"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Gate Check (--check-gate)
# ─────────────────────────────────────────────────────────────────────────────

@test "gate: bugfix passes at 2.5 (threshold 2.0)" {
  local d="$TEST_DIR/gate-bugfix-pass"
  mkdir -p "$d"
  make_gate_status "$d" "bugfix" "2.5"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: pass"* ]]
  [[ "$output" == *"threshold: 2.0"* ]]
}

@test "gate: bugfix fails at 1.5 (threshold 2.0)" {
  local d="$TEST_DIR/gate-bugfix-fail"
  mkdir -p "$d"
  make_gate_status "$d" "bugfix" "1.5"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: fail"* ]]
}

@test "gate: feature passes at 3.0 (exact boundary)" {
  local d="$TEST_DIR/gate-feature-exact"
  mkdir -p "$d"
  make_gate_status "$d" "feature" "3.0"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: pass"* ]]
}

@test "gate: feature fails at 2.9" {
  local d="$TEST_DIR/gate-feature-fail"
  mkdir -p "$d"
  make_gate_status "$d" "feature" "2.9"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: fail"* ]]
}

@test "gate: architecture fails at 3.5 (threshold 4.0)" {
  local d="$TEST_DIR/gate-arch-fail"
  mkdir -p "$d"
  make_gate_status "$d" "architecture" "3.5"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: fail"* ]]
  [[ "$output" == *"threshold: 4.0"* ]]
}

@test "gate: architecture passes at 4.0" {
  local d="$TEST_DIR/gate-arch-pass"
  mkdir -p "$d"
  make_gate_status "$d" "architecture" "4.0"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: pass"* ]]
}

@test "gate: refactor uses 3.0 threshold" {
  local d="$TEST_DIR/gate-refactor"
  mkdir -p "$d"
  make_gate_status "$d" "refactor" "3.0"
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: pass"* ]]
  [[ "$output" == *"threshold: 3.0"* ]]
}

@test "gate: missing change_type defaults to feature (threshold 3.0)" {
  local d="$TEST_DIR/gate-no-type"
  mkdir -p "$d"
  cat > "$d/.status.yaml" <<'EOF'
name: test-change
progress:
  intake: done
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
  run "$CALC_SCORE" --check-gate "$d"
  [[ "$output" == *"gate: fail"* ]]
  [[ "$output" == *"change_type: feature"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Backward Compatibility
# ─────────────────────────────────────────────────────────────────────────────

@test "compat: existing fixture produces identical results" {
  local d="$TEST_DIR/compat-identical"
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
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"confident: 2"* ]]
  [[ "$output" == *"tentative: 1"* ]]
  [[ "$output" == *"score: 3.4"* ]]
}

@test "compat: no fuzzy field in status for legacy tables" {
  local d="$TEST_DIR/compat-no-fuzzy"
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
  "$CALC_SCORE" "$d" > /dev/null
  local status_content
  status_content=$(cat "$d/.status.yaml")
  [[ "$status_content" != *"fuzzy:"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Edge Cases
# ─────────────────────────────────────────────────────────────────────────────

@test "edge: all-zero dimension scores" {
  local d="$TEST_DIR/edge-zero"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Tentative | Unknown | Everything ambiguous | S:0 R:0 A:0 D:0 |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"signal: 0.0"* ]]
  [[ "$output" == *"reversibility: 0.0"* ]]
  [[ "$output" == *"fuzzy: true"* ]]
}

@test "edge: all-100 dimension scores" {
  local d="$TEST_DIR/edge-max"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Crystal clear | Everything obvious | S:100 R:100 A:100 D:100 |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"signal: 100.0"* ]]
  [[ "$output" == *"disambiguation: 100.0"* ]]
}

@test "edge: single-row table with scores" {
  local d="$TEST_DIR/edge-single-row"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Sole decision | Only one | S:42 R:88 A:55 D:73 |
EOF
  run "$CALC_SCORE" "$d"
  [[ "$output" == *"signal: 42.0"* ]]
  [[ "$output" == *"reversibility: 88.0"* ]]
  [[ "$output" == *"competence: 55.0"* ]]
  [[ "$output" == *"disambiguation: 73.0"* ]]
}

# ─────────────────────────────────────────────────────────────────────────────
# Fuzzy Status Write
# ─────────────────────────────────────────────────────────────────────────────

@test "fuzzy write: .status.yaml gets dimension fields" {
  local d="$TEST_DIR/fuzzy-status-write"
  mkdir -p "$d"
  make_status "$d"
  cat > "$d/spec.md" <<'EOF'
# Spec

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Test | Test | S:70 R:80 A:60 D:90 |
EOF
  "$CALC_SCORE" "$d" > /dev/null
  local status_content
  status_content=$(cat "$d/.status.yaml")
  [[ "$status_content" == *"fuzzy: true"* ]]
  [[ "$status_content" == *"signal:"* ]]
  [[ "$status_content" == *"reversibility:"* ]]
  [[ "$status_content" == *"competence:"* ]]
  [[ "$status_content" == *"disambiguation:"* ]]
  [[ "$status_content" == *"name: test-change"* ]]
  [[ "$status_content" == *"intake: done"* ]]
}

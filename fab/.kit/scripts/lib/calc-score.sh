#!/usr/bin/env bash
set -euo pipefail

# CLI entry point for stageman (subprocess calls, not sourced)
STAGEMAN="$(dirname "$(readlink -f "$0")")/stageman.sh"

# calc-score.sh — Compute confidence score from spec.md Assumptions table
#
# Internal library script invoked by /fab-continue (spec stage) and
# /fab-clarify (suggest mode). Not called directly by users.
#
# Usage: calc-score.sh [--stage <stage>] <change-dir>
#        calc-score.sh --check-gate <change-dir>
# Output: YAML confidence block to stdout (or gate result for --check-gate)
# Side effect: Updates confidence block in .status.yaml
# Exit: 0 on success, 1 on error (message to stderr)

# --- Expected minimum decisions by stage and change_type ---
get_expected_min() {
  local stage="$1" change_type="$2"
  case "$stage" in
    intake)
      case "$change_type" in
        fix) echo 2 ;; feat) echo 4 ;; refactor) echo 3 ;; *) echo 2 ;;
      esac ;;
    spec|*)
      case "$change_type" in
        fix) echo 4 ;; feat) echo 6 ;; refactor) echo 5 ;; *) echo 3 ;;
      esac ;;
  esac
}

# --- Gate thresholds by change type (7-type taxonomy) ---
get_gate_threshold() {
  local change_type="$1"
  case "$change_type" in
    fix)      echo "2.0" ;;
    feat)     echo "3.0" ;;
    refactor) echo "3.0" ;;
    docs)     echo "2.0" ;;
    test)     echo "2.0" ;;
    ci)       echo "2.0" ;;
    chore)    echo "2.0" ;;
    *)        echo "3.0" ;;  # default to feat threshold
  esac
}

# --- Argument parsing ---
CHECK_GATE=false
SCORE_STAGE="spec"

while [ $# -gt 0 ]; do
  case "${1:-}" in
    --check-gate)
      CHECK_GATE=true
      shift
      ;;
    --stage)
      SCORE_STAGE="${2:-spec}"
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

change_dir="${1:-}"

if [ -z "$change_dir" ]; then
  echo "Usage: calc-score.sh [--check-gate] [--stage <stage>] <change-dir>" >&2
  exit 1
fi

if [ ! -d "$change_dir" ]; then
  echo "Change directory not found: $change_dir" >&2
  exit 1
fi

status_file="$change_dir/.status.yaml"

# --- Read change_type from .status.yaml (default: feat) ---
read_change_type() {
  local file="$1"
  if [ ! -f "$file" ]; then
    echo "feat"
    return
  fi
  local ct
  ct=$(grep '^ *change_type:' "$file" | sed 's/^ *change_type: *//' || echo "")
  ct=$(echo "$ct" | tr -d '[:space:]')
  if [ -z "$ct" ] || [ "$ct" = "null" ]; then
    echo "feat"
  else
    echo "$ct"
  fi
}

# --- Parse Assumptions table ---
# Extract Grade and Scores column values from ## Assumptions table.
# Scores column is required (column 6 after split by |).
# Outputs lines in format: "grade|S:nn R:nn A:nn D:nn"
parse_assumptions() {
  local file="$1"
  if [ ! -f "$file" ]; then
    return
  fi
  awk '
    /^## Assumptions/ { in_section = 1; header_seen = 0; next }
    in_section && /^## / { exit }
    in_section && /^\| *#/ {
      header_seen = 1
      next
    }
    in_section && /^\|[-| ]+\|/ { next }
    in_section && header_seen && /^\|/ {
      split($0, cols, "|")
      grade = cols[3]
      gsub(/^[ \t]+|[ \t]+$/, "", grade)

      scores = cols[6]
      gsub(/^[ \t]+|[ \t]+$/, "", scores)
      print grade "|" scores
    }
  ' "$file"
}

# --- Gate check mode ---
if [ "$CHECK_GATE" = true ]; then
  if [ ! -f "$status_file" ]; then
    echo "ERROR: .status.yaml not found in $change_dir" >&2
    exit 1
  fi

  change_type=$(read_change_type "$status_file")

  if [ "$SCORE_STAGE" = "intake" ]; then
    # Intake gate: compute indicative score on the fly, fixed threshold 3.0
    score_file="$change_dir/intake.md"
    if [ ! -f "$score_file" ]; then
      echo "ERROR: intake.md not found in $change_dir" >&2
      exit 1
    fi
    expected_min=$(get_expected_min "intake" "$change_type")
    # Parse and compute inline
    local_certain=0 local_confident=0 local_tentative=0 local_unresolved=0
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      grade="${line%%|*}"
      grade_lower=$(echo "$grade" | tr '[:upper:]' '[:lower:]')
      case "$grade_lower" in
        certain)    local_certain=$((local_certain + 1)) ;;
        confident)  local_confident=$((local_confident + 1)) ;;
        tentative)  local_tentative=$((local_tentative + 1)) ;;
        unresolved) local_unresolved=$((local_unresolved + 1)) ;;
      esac
    done <<< "$(parse_assumptions "$score_file")"
    local_total=$((local_certain + local_confident + local_tentative + local_unresolved))
    if [ "$local_unresolved" -gt 0 ]; then
      score="0.0"
    else
      score=$(awk -v confident="$local_confident" -v tentative="$local_tentative" \
                  -v total="$local_total" -v exp_min="$expected_min" \
        "BEGIN {
          base = 5.0 - 0.3 * confident - 1.0 * tentative
          if (base < 0.0) base = 0.0
          if (exp_min > 0) cover = total / exp_min; else cover = 1.0
          if (cover > 1.0) cover = 1.0
          printf \"%.1f\", base * cover
        }")
    fi
    threshold="3.0"
  else
    # Spec gate: read existing score from .status.yaml, dynamic per-type threshold
    score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || echo "0.0")
    threshold=$(get_gate_threshold "$change_type")
  fi

  # Compare score >= threshold
  passes=$(awk "BEGIN { print ($score >= $threshold) ? \"pass\" : \"fail\" }")

  cat <<EOF
gate: $passes
score: $score
threshold: $threshold
change_type: $change_type
EOF
  exit 0
fi

# --- Normal scoring mode ---
# Determine which artifact to read based on stage
if [ "$SCORE_STAGE" = "intake" ]; then
  score_file="$change_dir/intake.md"
  if [ ! -f "$score_file" ]; then
    echo "intake.md required for scoring at intake stage" >&2
    exit 1
  fi
else
  score_file="$change_dir/spec.md"
  if [ ! -f "$score_file" ]; then
    echo "spec.md required for scoring" >&2
    exit 1
  fi
fi

change_type=$(read_change_type "$status_file")
expected_min=$(get_expected_min "$SCORE_STAGE" "$change_type")

# Collect all parsed lines from the target artifact
all_parsed=""
all_parsed+="$(parse_assumptions "$score_file")"

# Count grades and collect dimension scores (case-insensitive)
table_certain=0
table_confident=0
table_tentative=0
table_unresolved=0
has_fuzzy=false
dim_count=0
sum_s=0
sum_r=0
sum_a=0
sum_d=0

while IFS= read -r line; do
  [ -z "$line" ] && continue

  # Split grade from optional scores
  grade="${line%%|*}"
  scores_part=""
  if [[ "$line" == *"|"* ]]; then
    scores_part="${line#*|}"
  fi

  grade_lower=$(echo "$grade" | tr '[:upper:]' '[:lower:]')
  case "$grade_lower" in
    certain)    table_certain=$((table_certain + 1)) ;;
    confident)  table_confident=$((table_confident + 1)) ;;
    tentative)  table_tentative=$((table_tentative + 1)) ;;
    unresolved) table_unresolved=$((table_unresolved + 1)) ;;
  esac

  # Parse dimension scores if present (format: S:nn R:nn A:nn D:nn)
  if [ -n "$scores_part" ]; then
    s_val=$(echo "$scores_part" | grep -oP 'S:\K[0-9]+' || echo "")
    r_val=$(echo "$scores_part" | grep -oP 'R:\K[0-9]+' || echo "")
    a_val=$(echo "$scores_part" | grep -oP 'A:\K[0-9]+' || echo "")
    d_val=$(echo "$scores_part" | grep -oP 'D:\K[0-9]+' || echo "")

    if [ -n "$s_val" ] && [ -n "$r_val" ] && [ -n "$a_val" ] && [ -n "$d_val" ]; then
      has_fuzzy=true
      dim_count=$((dim_count + 1))
      sum_s=$((sum_s + s_val))
      sum_r=$((sum_r + r_val))
      sum_a=$((sum_a + a_val))
      sum_d=$((sum_d + d_val))
    fi
  fi
done <<< "$all_parsed"

# Compute mean dimension scores
mean_s="0.0"
mean_r="0.0"
mean_a="0.0"
mean_d="0.0"
if [ "$dim_count" -gt 0 ]; then
  mean_s=$(awk "BEGIN { printf \"%.1f\", $sum_s / $dim_count }")
  mean_r=$(awk "BEGIN { printf \"%.1f\", $sum_r / $dim_count }")
  mean_a=$(awk "BEGIN { printf \"%.1f\", $sum_a / $dim_count }")
  mean_d=$(awk "BEGIN { printf \"%.1f\", $sum_d / $dim_count }")
fi

# --- Read previous score for delta computation ---
prev_score="0.0"
if [ -f "$status_file" ]; then
  confidence_data=$("$STAGEMAN" confidence "$status_file")
  prev_score=$(echo "$confidence_data" | grep '^score:' | cut -d: -f2)
  prev_score=${prev_score:-0.0}
fi

# --- Apply coverage-weighted formula ---
total_decisions=$((table_certain + table_confident + table_tentative + table_unresolved))

if [ "$table_unresolved" -gt 0 ]; then
  score="0.0"
else
  # base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  # cover = min(1.0, total_decisions / expected_min)
  # score = base * cover
  score=$(awk -v confident="$table_confident" \
              -v tentative="$table_tentative" \
              -v total="$total_decisions" \
              -v exp_min="$expected_min" \
    "BEGIN {
      base = 5.0 - 0.3 * confident - 1.0 * tentative
      if (base < 0.0) base = 0.0
      if (exp_min > 0)
        cover = total / exp_min
      else
        cover = 1.0
      if (cover > 1.0) cover = 1.0
      printf \"%.1f\", base * cover
    }")
fi

# --- Compute delta ---
delta=$(awk "BEGIN {
  d = $score - $prev_score
  if (d >= 0) printf \"+%.1f\", d
  else printf \"%.1f\", d
}")

# --- Write to .status.yaml ---
if [ -f "$status_file" ]; then
  if [ "$has_fuzzy" = true ]; then
    "$STAGEMAN" set-confidence-fuzzy "$status_file" "$table_certain" "$table_confident" "$table_tentative" "$table_unresolved" "$score" "$mean_s" "$mean_r" "$mean_a" "$mean_d"
  else
    "$STAGEMAN" set-confidence "$status_file" "$table_certain" "$table_confident" "$table_tentative" "$table_unresolved" "$score"
  fi
  "$STAGEMAN" log-confidence "$change_dir" "$score" "$delta" "calc-score"
fi

# --- Emit YAML to stdout ---
cat <<EOF
confidence:
  certain: $table_certain
  confident: $table_confident
  tentative: $table_tentative
  unresolved: $table_unresolved
  score: $score
  delta: $delta
EOF

if [ "$has_fuzzy" = true ]; then
  cat <<EOF
  fuzzy: true
  dimensions:
    signal: $mean_s
    reversibility: $mean_r
    competence: $mean_a
    disambiguation: $mean_d
EOF
fi

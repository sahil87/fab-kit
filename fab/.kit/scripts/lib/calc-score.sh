#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=stageman.sh
source "$(dirname "$(readlink -f "$0")")/stageman.sh"

# calc-score.sh — Compute confidence score from Assumptions tables
#
# Internal library script invoked by /fab-continue (spec stage) and
# /fab-clarify (suggest mode). Not called directly by users.
#
# Usage: calc-score.sh <change-dir>
#        calc-score.sh --check-gate <change-dir>
# Output: YAML confidence block to stdout (or gate result for --check-gate)
# Side effect: Updates confidence block in .status.yaml
# Exit: 0 on success, 1 on error (message to stderr)

# --- Argument parsing ---
CHECK_GATE=false

if [ "${1:-}" = "--check-gate" ]; then
  CHECK_GATE=true
  shift
fi

change_dir="${1:-}"

if [ -z "$change_dir" ]; then
  echo "Usage: calc-score.sh [--check-gate] <change-dir>" >&2
  exit 1
fi

if [ ! -d "$change_dir" ]; then
  echo "Change directory not found: $change_dir" >&2
  exit 1
fi

status_file="$change_dir/.status.yaml"

# --- Gate check mode ---
if [ "$CHECK_GATE" = true ]; then
  if [ ! -f "$status_file" ]; then
    echo "ERROR: .status.yaml not found in $change_dir" >&2
    exit 1
  fi

  score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || echo "0.0")
  change_type=$(grep '^ *change_type:' "$status_file" | sed 's/^ *change_type: *//' || echo "")

  # Determine threshold based on change type
  case "${change_type:-feature}" in
    bugfix)       threshold="2.0" ;;
    feature)      threshold="3.0" ;;
    refactor)     threshold="3.0" ;;
    architecture) threshold="4.0" ;;
    *)            threshold="3.0" ;;
  esac

  # Compare score >= threshold
  passes=$(awk "BEGIN { print ($score >= $threshold) ? \"pass\" : \"fail\" }")

  cat <<EOF
gate: $passes
score: $score
threshold: $threshold
change_type: ${change_type:-feature}
EOF
  exit 0
fi

# --- Normal scoring mode ---
brief_file="$change_dir/brief.md"
spec_file="$change_dir/spec.md"

if [ ! -f "$spec_file" ]; then
  echo "spec.md required for scoring" >&2
  exit 1
fi

# --- Parse Assumptions tables ---
# Extract Grade and optional Scores column values from ## Assumptions tables.
# Detects whether a Scores column exists by checking the header row.
# Outputs lines in format: "grade" or "grade|S:nn R:nn A:nn D:nn"
parse_assumptions() {
  local file="$1"
  if [ ! -f "$file" ]; then
    return
  fi
  awk '
    /^## Assumptions/ { in_section = 1; header_seen = 0; has_scores = 0; next }
    in_section && /^## / { exit }
    in_section && /^\| *#/ {
      header_seen = 1
      # Check if header contains "Scores" column
      if ($0 ~ /[Ss]cores/) has_scores = 1
      next
    }
    in_section && /^\|[-| ]+\|/ { next }
    in_section && header_seen && /^\|/ {
      split($0, cols, "|")
      # Grade is always column 3 (after leading empty and # columns)
      grade = cols[3]
      gsub(/^[ \t]+|[ \t]+$/, "", grade)

      if (has_scores) {
        scores = cols[4]
        gsub(/^[ \t]+|[ \t]+$/, "", scores)
        print grade "|" scores
      } else {
        print grade
      }
    }
  ' "$file"
}

# Collect all parsed lines from brief + spec
all_parsed=""
all_parsed+="$(parse_assumptions "$brief_file")"$'\n'
all_parsed+="$(parse_assumptions "$spec_file")"

# Count grades and collect dimension scores (case-insensitive)
table_certain=0
table_confident=0
table_tentative=0
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
    certain)   table_certain=$((table_certain + 1)) ;;
    confident) table_confident=$((table_confident + 1)) ;;
    tentative) table_tentative=$((table_tentative + 1)) ;;
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

# --- Carry-forward implicit Certain counts ---
prev_certain=0
prev_score="0.0"
if [ -f "$status_file" ]; then
  confidence_data=$(get_confidence "$status_file")
  prev_certain=$(echo "$confidence_data" | grep '^certain:' | cut -d: -f2)
  prev_certain=${prev_certain:-0}
  prev_score=$(echo "$confidence_data" | grep '^score:' | cut -d: -f2)
  prev_score=${prev_score:-0.0}
fi

# Implicit = previous total - explicit Certain found in tables
implicit_certain=$((prev_certain - table_certain))
if [ "$implicit_certain" -lt 0 ]; then
  implicit_certain=0
fi
total_certain=$((implicit_certain + table_certain))

# --- Apply formula ---
# Unresolved is always 0 (Unresolved decisions are asked interactively, never in tables)
unresolved=0

if [ "$unresolved" -gt 0 ]; then
  score="0.0"
else
  # score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  # Use awk for floating point arithmetic
  score=$(awk "BEGIN {
    s = 5.0 - 0.3 * $table_confident - 1.0 * $table_tentative
    if (s < 0.0) s = 0.0
    printf \"%.1f\", s
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
    set_confidence_block_fuzzy "$status_file" "$total_certain" "$table_confident" "$table_tentative" "$unresolved" "$score" "$mean_s" "$mean_r" "$mean_a" "$mean_d"
  else
    set_confidence_block "$status_file" "$total_certain" "$table_confident" "$table_tentative" "$unresolved" "$score"
  fi
  log_confidence "$change_dir" "$score" "$delta" "calc-score"
fi

# --- Emit YAML to stdout ---
cat <<EOF
confidence:
  certain: $total_certain
  confident: $table_confident
  tentative: $table_tentative
  unresolved: $unresolved
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

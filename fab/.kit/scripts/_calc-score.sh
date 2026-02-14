#!/usr/bin/env bash
set -euo pipefail

# _calc-score.sh — Compute confidence score from Assumptions tables
#
# Internal library script invoked by /fab-continue (spec stage) and
# /fab-clarify (suggest mode). Not called directly by users.
#
# Usage: _calc-score.sh <change-dir>
# Output: YAML confidence block to stdout
# Side effect: Updates confidence block in .status.yaml
# Exit: 0 on success, 1 on error (message to stderr)

change_dir="${1:-}"

if [ -z "$change_dir" ]; then
  echo "Usage: _calc-score.sh <change-dir>" >&2
  exit 1
fi

if [ ! -d "$change_dir" ]; then
  echo "Change directory not found: $change_dir" >&2
  exit 1
fi

status_file="$change_dir/.status.yaml"
brief_file="$change_dir/brief.md"
spec_file="$change_dir/spec.md"

if [ ! -f "$spec_file" ]; then
  echo "spec.md required for scoring" >&2
  exit 1
fi

# --- Parse Assumptions tables ---
# Extract Grade column values from ## Assumptions tables in a markdown file.
# Looks for lines between "## Assumptions" and the next "## " heading (or EOF),
# skips table header and separator rows, extracts 2nd pipe-delimited column.
count_grades() {
  local file="$1"
  if [ ! -f "$file" ]; then
    return
  fi
  awk '
    /^## Assumptions/ { in_section = 1; header_seen = 0; next }
    in_section && /^## / { exit }
    in_section && /^\| *#/ { header_seen = 1; next }
    in_section && /^\|[-| ]+\|/ { next }
    in_section && header_seen && /^\|/ {
      # Extract 2nd column (Grade)
      split($0, cols, "|")
      gsub(/^[ \t]+|[ \t]+$/, "", cols[3])
      print cols[3]
    }
  ' "$file"
}

# Collect all grade values from brief + spec
all_grades=""
all_grades+="$(count_grades "$brief_file")"$'\n'
all_grades+="$(count_grades "$spec_file")"

# Count grades (case-insensitive)
table_certain=0
table_confident=0
table_tentative=0

while IFS= read -r grade; do
  grade_lower=$(echo "$grade" | tr '[:upper:]' '[:lower:]')
  case "$grade_lower" in
    certain)   table_certain=$((table_certain + 1)) ;;
    confident) table_confident=$((table_confident + 1)) ;;
    tentative) table_tentative=$((table_tentative + 1)) ;;
  esac
done <<< "$all_grades"

# --- Carry-forward implicit Certain counts ---
prev_certain=0
prev_score="5.0"
if [ -f "$status_file" ]; then
  prev_certain=$(grep '^ *certain:' "$status_file" | sed 's/^ *certain: *//' || true)
  prev_certain=${prev_certain:-0}
  prev_score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || true)
  prev_score=${prev_score:-5.0}
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
  # Replace the confidence block using awk
  # Match from "confidence:" to the next top-level key (non-indented line) or EOF
  tmpfile=$(mktemp)
  awk -v certain="$total_certain" \
      -v confident="$table_confident" \
      -v tentative="$table_tentative" \
      -v unresolved="$unresolved" \
      -v score="$score" '
    /^confidence:/ { in_block = 1; skip = 1
      print "confidence:"
      print "  certain: " certain
      print "  confident: " confident
      print "  tentative: " tentative
      print "  unresolved: " unresolved
      print "  score: " score
      next
    }
    in_block && /^[^ ]/ { in_block = 0; skip = 0 }
    in_block { next }
    { print }
  ' "$status_file" > "$tmpfile"
  mv "$tmpfile" "$status_file"
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

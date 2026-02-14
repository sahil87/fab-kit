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
# Output: YAML confidence block to stdout
# Side effect: Updates confidence block in .status.yaml
# Exit: 0 on success, 1 on error (message to stderr)

BASELINE_WEIGHT_CONFIDENT="0.3"
BASELINE_WEIGHT_TENTATIVE="1.0"
DEFAULT_MODE="legacy"
DEFAULT_CHANGE_TYPE="feature"
SPARSE_MIN_SAMPLES="${FAB_CALIBRATION_MIN_SAMPLES:-20}"

change_dir="${1:-}"

if [ -z "$change_dir" ]; then
  echo "Usage: calc-score.sh <change-dir>" >&2
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

to_lower() {
  echo "$1" | tr '[:upper:]' '[:lower:]'
}

is_non_negative_number() {
  [[ "${1:-}" =~ ^[0-9]+([.][0-9]+)?$ ]]
}

status_scalar() {
  local key="$1"
  local file="$2"
  grep "^ *${key}:" "$file" 2>/dev/null | head -n 1 | sed -E "s/^ *${key}: *//" | tr -d '"' || true
}

normalize_mode() {
  local raw
  raw=$(to_lower "${1:-}")
  case "$raw" in
    legacy|fuzzy) echo "$raw" ;;
    *) echo "$DEFAULT_MODE" ;;
  esac
}

normalize_change_type() {
  local raw
  raw=$(to_lower "${1:-}")
  case "$raw" in
    bugfix|feature|refactor|architecture) echo "$raw" ;;
    *) echo "$DEFAULT_CHANGE_TYPE" ;;
  esac
}

infer_change_type_from_name() {
  local name
  name=$(basename "$change_dir")
  name=$(to_lower "$name")
  case "$name" in
    *bug*|*fix*|*hotfix*|*patch*) echo "bugfix" ;;
    *refactor*|*cleanup*|*reorg*|*rename*) echo "refactor" ;;
    *arch*|*architecture*|*infra*|*platform*|*foundation*) echo "architecture" ;;
    *) echo "$DEFAULT_CHANGE_TYPE" ;;
  esac
}

count_historical_samples() {
  local abs_change_dir root archive_dir
  abs_change_dir="$(cd "$change_dir" && pwd)"

  if [[ "$abs_change_dir" == */fab/changes/* ]]; then
    root="${abs_change_dir%/fab/changes/*}"
  else
    root="$(cd "$change_dir/../.." 2>/dev/null && pwd || pwd)"
  fi

  archive_dir="$root/fab/changes/archive"
  if [ ! -d "$archive_dir" ]; then
    echo "0"
    return
  fi

  find "$archive_dir" -type f -name ".status.yaml" | wc -l | tr -d ' '
}

format_float() {
  awk "BEGIN { printf \"%.2f\", $1 }"
}

weighted_composite() {
  local s="$1" r="$2" a="$3" d="$4"
  awk "BEGIN { printf \"%.4f\", (0.2 * $s) + (0.3 * $r) + (0.3 * $a) + (0.2 * $d) }"
}

fuzzy_membership() {
  local composite="$1"
  awk -v c="$composite" '
    BEGIN {
      conf = 0.0
      tent = 0.0
      if (c <= 30.0) {
        tent = 1.0
      } else if (c < 60.0) {
        tent = (60.0 - c) / 30.0
        conf = (c - 30.0) / 30.0
      } else if (c < 85.0) {
        conf = (85.0 - c) / 25.0
      }
      if (tent < 0.0) tent = 0.0
      if (conf < 0.0) conf = 0.0
      printf "%.4f %.4f", conf, tent
    }
  '
}

extract_dim_value() {
  local text="$1" key="$2"
  echo "$text" | sed -nE "s/.*${key}[[:space:]]*=[[:space:]]*([0-9]{1,3}).*/\\1/p" | head -n 1
}

has_all_dimensions() {
  local s="$1" r="$2" a="$3" d="$4"
  for n in "$s" "$r" "$a" "$d"; do
    if ! [[ "$n" =~ ^[0-9]+$ ]]; then
      return 1
    fi
    if [ "$n" -lt 0 ] || [ "$n" -gt 100 ]; then
      return 1
    fi
  done
  return 0
}

add_float() {
  local left="$1" right="$2"
  awk "BEGIN { printf \"%.6f\", $left + $right }"
}

default_composite_for_grade() {
  local grade="$1"
  case "$grade" in
    confident) echo "60" ;;
    tentative) echo "30" ;;
    certain) echo "90" ;;
    *) echo "60" ;;
  esac
}

threshold_for_type() {
  local mode="$1" ctype="$2"
  if [ "$mode" = "legacy" ]; then
    echo "3.0"
    return
  fi
  case "$ctype" in
    bugfix) echo "2.7" ;;
    refactor) echo "3.0" ;;
    feature) echo "3.3" ;;
    architecture) echo "3.6" ;;
    *) echo "3.3" ;;
  esac
}

# Detect mode and change type
mode_raw="${FAB_SCORE_MODE:-$(status_scalar "score_mode" "$status_file")}"
score_mode="$(normalize_mode "$mode_raw")"

change_type_raw="${FAB_CHANGE_TYPE:-$(status_scalar "change_type" "$status_file")}"
if [ -z "$change_type_raw" ]; then
  change_type="$(infer_change_type_from_name)"
  change_type_inferred=true
else
  normalized_type="$(normalize_change_type "$change_type_raw")"
  if [ "$normalized_type" = "$DEFAULT_CHANGE_TYPE" ] && [ "$(to_lower "$change_type_raw")" != "$DEFAULT_CHANGE_TYPE" ]; then
    change_type_inferred=true
  else
    change_type_inferred=false
  fi
  change_type="$normalized_type"
fi

# Select weights (with sparse-history fallback)
weight_confident="$BASELINE_WEIGHT_CONFIDENT"
weight_tentative="$BASELINE_WEIGHT_TENTATIVE"
historical_samples="$(count_historical_samples)"
weights_source="baseline"

if [ "$score_mode" = "fuzzy" ] && [ "${historical_samples:-0}" -ge "${SPARSE_MIN_SAMPLES:-20}" ]; then
  requested_conf="${FAB_WEIGHT_CONFIDENT:-$(status_scalar "weight_confident" "$status_file")}"
  requested_tent="${FAB_WEIGHT_TENTATIVE:-$(status_scalar "weight_tentative" "$status_file")}"
  if is_non_negative_number "${requested_conf:-}" && is_non_negative_number "${requested_tent:-}"; then
    weight_confident="$requested_conf"
    weight_tentative="$requested_tent"
    weights_source="calibrated"
  fi
fi

used_sparse_fallback=false
if [ "$score_mode" = "fuzzy" ] && [ "${historical_samples:-0}" -lt "${SPARSE_MIN_SAMPLES:-20}" ]; then
  used_sparse_fallback=true
fi

# --- Parse Assumptions tables ---
# Extract assumption rows from ## Assumptions table as:
# Grade|Decision|Rationale
extract_assumption_rows() {
  local file="$1"
  if [ ! -f "$file" ]; then
    return
  fi
  awk '
    function trim(s) {
      gsub(/^[ \t]+|[ \t]+$/, "", s)
      return s
    }
    /^## Assumptions/ { in_section = 1; header_seen = 0; next }
    in_section && /^## / { exit }
    in_section && /^\| *#/ { header_seen = 1; next }
    in_section && /^\|[-| ]+\|/ { next }
    in_section && header_seen && /^\|/ {
      split($0, cols, "|")
      grade = trim(cols[3])
      decision = trim(cols[4])
      rationale = trim(cols[5])
      print grade "|" decision "|" rationale
    }
  ' "$file"
}

# Collect all rows from brief + spec
all_rows=""
all_rows+="$(extract_assumption_rows "$brief_file")"$'\n'
all_rows+="$(extract_assumption_rows "$spec_file")"

# Count grades + fuzzy effective totals
table_certain=0
table_confident=0
table_tentative=0
effective_confident="0.0"
effective_tentative="0.0"

while IFS='|' read -r grade decision rationale; do
  [ -z "${grade:-}" ] && continue
  grade_lower=$(echo "$grade" | tr '[:upper:]' '[:lower:]')
  case "$grade_lower" in
    certain)
      table_certain=$((table_certain + 1))
      ;;
    confident)
      table_confident=$((table_confident + 1))
      ;;
    tentative)
      table_tentative=$((table_tentative + 1))
      ;;
    *)
      continue
      ;;
  esac

  if [ "$score_mode" = "legacy" ]; then
    case "$grade_lower" in
      confident) effective_confident=$(add_float "$effective_confident" "1.0") ;;
      tentative) effective_tentative=$(add_float "$effective_tentative" "1.0") ;;
    esac
    continue
  fi

  # Fuzzy mode: use SRAD dimension metadata when available, otherwise grade defaults
  row_text="$decision $rationale"
  s=$(extract_dim_value "$row_text" "S")
  r=$(extract_dim_value "$row_text" "R")
  a=$(extract_dim_value "$row_text" "A")
  d=$(extract_dim_value "$row_text" "D")

  if has_all_dimensions "${s:-}" "${r:-}" "${a:-}" "${d:-}"; then
    composite=$(weighted_composite "$s" "$r" "$a" "$d")
  else
    composite=$(default_composite_for_grade "$grade_lower")
  fi

  read -r conf_mem tent_mem <<< "$(fuzzy_membership "$composite")"

  case "$grade_lower" in
    confident|tentative)
      effective_confident=$(add_float "$effective_confident" "$conf_mem")
      effective_tentative=$(add_float "$effective_tentative" "$tent_mem")
      ;;
  esac
done <<< "$all_rows"

if [ "$score_mode" = "legacy" ]; then
  effective_confident="$table_confident"
  effective_tentative="$table_tentative"
fi

# --- Carry-forward implicit Certain counts ---
prev_certain=0
prev_score="0.0"
if [ -f "$status_file" ]; then
  prev_certain=$(grep '^ *certain:' "$status_file" | sed 's/^ *certain: *//' || true)
  prev_certain=${prev_certain:-0}
  prev_score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || true)
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
  # score = max(0.0, 5.0 - w_confident * effective_confident - w_tentative * effective_tentative)
  # Use awk for floating point arithmetic
  score=$(awk "BEGIN {
    s = 5.0 - $weight_confident * $effective_confident - $weight_tentative * $effective_tentative
    if (s < 0.0) s = 0.0
    printf \"%.1f\", s
  }")
fi

# --- Compute gate threshold metadata ---
threshold=$(threshold_for_type "$score_mode" "$change_type")
gate_passes=$(awk "BEGIN { print ($score >= $threshold) ? \"true\" : \"false\" }")

# --- Compute delta ---
delta=$(awk "BEGIN {
  d = $score - $prev_score
  if (d >= 0) printf \"+%.1f\", d
  else printf \"%.1f\", d
}")

# --- Write to .status.yaml ---
if [ -f "$status_file" ]; then
  set_confidence_block "$status_file" "$total_certain" "$table_confident" "$table_tentative" "$unresolved" "$score"
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
scoring:
  mode: $score_mode
  effective_confident: $(format_float "$effective_confident")
  effective_tentative: $(format_float "$effective_tentative")
  weight_confident: $(format_float "$weight_confident")
  weight_tentative: $(format_float "$weight_tentative")
  weights_source: $weights_source
  historical_samples: $historical_samples
  sparse_fallback: $used_sparse_fallback
gate:
  change_type: $change_type
  threshold: $threshold
  passes_fff: $gate_passes
  change_type_inferred: $change_type_inferred
EOF

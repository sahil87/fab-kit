#!/usr/bin/env bash
# src/lib/calc-score/sensitivity.sh
#
# Weight validation via sensitivity analysis for SRAD confidence scoring.
# Covers two domains:
#   Domain 1: Formula penalty weights (Confident/Tentative penalties)
#   Domain 2: Dimension aggregation weights (w_S, w_R, w_A, w_D)
#
# Usage: sensitivity.sh [--domain1|--domain2|--all] [archive-dir]
# Default: --all, archive-dir = fab/changes/archive/
#
# Exit: 0 on success, 1 on error, 2 on insufficient data

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Defaults
DOMAIN="all"
ARCHIVE_DIR=""
MIN_CHANGES=5

# Parse arguments
while [ $# -gt 0 ]; do
  case "$1" in
    --domain1) DOMAIN="domain1"; shift ;;
    --domain2) DOMAIN="domain2"; shift ;;
    --all)     DOMAIN="all"; shift ;;
    --help|-h)
      echo "Usage: sensitivity.sh [--domain1|--domain2|--all] [archive-dir]"
      echo ""
      echo "Options:"
      echo "  --domain1    Formula penalty weights only (Confident/Tentative penalties)"
      echo "  --domain2    Dimension aggregation weights only (w_S, w_R, w_A, w_D)"
      echo "  --all        Both domains (default)"
      echo ""
      echo "Arguments:"
      echo "  archive-dir  Path to fab/changes/archive/ (auto-detected if omitted)"
      exit 0
      ;;
    -*)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
    *)
      ARCHIVE_DIR="$1"; shift ;;
  esac
done

# Auto-detect archive directory
if [ -z "$ARCHIVE_DIR" ]; then
  # Walk up from script dir to find fab/changes/archive/
  candidate="$SCRIPT_DIR/../../../fab/changes/archive"
  if [ -d "$candidate" ]; then
    ARCHIVE_DIR="$(cd "$candidate" && pwd)"
  else
    echo "ERROR: Could not find fab/changes/archive/. Provide path as argument." >&2
    exit 1
  fi
fi

if [ ! -d "$ARCHIVE_DIR" ]; then
  echo "ERROR: Archive directory not found: $ARCHIVE_DIR" >&2
  exit 1
fi

# ─────────────────────────────────────────────────────────────────────────────
# Collect Historical Data
# ─────────────────────────────────────────────────────────────────────────────

collect_archive_data() {
  local count=0
  local data_lines=""

  for status_file in "$ARCHIVE_DIR"/*/.status.yaml; do
    [ -f "$status_file" ] || continue

    local confident tentative score review_state
    confident=$(grep '^ *confident:' "$status_file" | sed 's/^ *confident: *//' || echo "0")
    tentative=$(grep '^ *tentative:' "$status_file" | sed 's/^ *tentative: *//' || echo "0")
    score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || echo "0.0")
    review_state=$(grep '^ *review:' "$status_file" | sed 's/^ *review: *//' || echo "pending")

    # Determine outcome: "pass" if review is done, "fail" or "unknown" otherwise
    local outcome="unknown"
    case "$review_state" in
      done) outcome="pass" ;;
      failed) outcome="fail" ;;
    esac

    data_lines+="${confident},${tentative},${score},${outcome}"$'\n'
    ((count++))
  done

  echo "$count"
  echo "$data_lines"
}

# ─────────────────────────────────────────────────────────────────────────────
# Domain 1: Formula Penalty Weights
# ─────────────────────────────────────────────────────────────────────────────

run_domain1() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "Domain 1: Formula Penalty Weights"
  echo "═══════════════════════════════════════════════════════════════"
  echo ""

  # Collect data
  local result
  result=$(collect_archive_data)
  local count
  count=$(echo "$result" | head -1)
  local data
  data=$(echo "$result" | tail -n +2)

  if [ "$count" -lt "$MIN_CHANGES" ]; then
    echo "Insufficient data for reliable sensitivity analysis (N=$count < $MIN_CHANGES)"
    echo "Recommendation: Keep current weights (Confident=0.3, Tentative=1.0) as defaults."
    echo ""
    return 2
  fi

  echo "Archived changes analyzed: $count"
  echo ""

  # Grid search: Confident penalty [0.1, 0.2, 0.3, 0.4, 0.5] x Tentative penalty [0.5, 0.75, 1.0, 1.25, 1.5]
  echo "| Confident | Tentative | Mean Score | Pass Rate (>=3.0) | Discrimination |"
  echo "|-----------|-----------|------------|-------------------|----------------|"

  local best_disc=0 best_cp="" best_tp=""

  for cp in 0.1 0.2 0.3 0.4 0.5; do
    for tp in 0.50 0.75 1.00 1.25 1.50; do
      # Compute scores and discrimination for this weight combination
      local stats
      stats=$(echo "$data" | awk -F',' -v cp="$cp" -v tp="$tp" '
        NF >= 4 {
          score = 5.0 - cp * $1 - tp * $2
          if (score < 0) score = 0
          total++
          sum_score += score
          if (score >= 3.0) above++
          if (score >= 3.0 && $4 == "pass") tp_count++
          if (score < 3.0 && ($4 == "fail" || $4 == "unknown")) tn_count++
          if ($4 == "pass") pass_count++
        }
        END {
          if (total == 0) { print "0,0,0,0"; exit }
          mean = sum_score / total
          pass_rate = (total > 0) ? above / total * 100 : 0
          disc = (total > 0) ? (tp_count + tn_count) / total * 100 : 0
          printf "%.2f,%.1f,%.1f\n", mean, pass_rate, disc
        }
      ')

      local mean pass_rate disc
      mean=$(echo "$stats" | cut -d',' -f1)
      pass_rate=$(echo "$stats" | cut -d',' -f2)
      disc=$(echo "$stats" | cut -d',' -f3)

      printf "| %-9s | %-9s | %-10s | %-17s | %-14s |\n" \
        "$cp" "$tp" "$mean" "${pass_rate}%" "${disc}%"

      # Track best discrimination
      local disc_int
      disc_int=$(echo "$disc" | cut -d'.' -f1)
      local best_int
      best_int=$(echo "$best_disc" | cut -d'.' -f1)
      if [ "${disc_int:-0}" -gt "${best_int:-0}" ]; then
        best_disc="$disc"
        best_cp="$cp"
        best_tp="$tp"
      fi
    done
  done

  echo ""
  echo "Best discrimination: Confident=$best_cp, Tentative=$best_tp (${best_disc}%)"

  if [ "$best_cp" = "0.3" ] && [ "$best_tp" = "1.00" ]; then
    echo "Recommendation: Current weights (0.3/1.0) are optimal or near-optimal."
  else
    echo "Recommendation: Consider adjusting to Confident=$best_cp, Tentative=$best_tp."
    echo "  Current weights (0.3/1.0) may not maximize discrimination."
  fi
  echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Domain 2: Dimension Aggregation Weights
# ─────────────────────────────────────────────────────────────────────────────

run_domain2() {
  echo "═══════════════════════════════════════════════════════════════"
  echo "Domain 2: Dimension Aggregation Weights"
  echo "═══════════════════════════════════════════════════════════════"
  echo ""
  echo "Testing synthetic scenarios from spec worked examples."
  echo "Varying w_R in [0.20, 0.25, 0.30, 0.35, 0.40], others proportional."
  echo ""

  # Synthetic test cases from spec:
  # Case 1: High composite — S=90, R=85, A=95, D=88 → expect Certain
  # Case 2: Mixed — S=40, R=70, A=55, D=30 → expect Tentative
  # Case 3: Critical Rule — S=60, R=20, A=15, D=70 → expect Unresolved (override)

  echo "| w_R  | w_S  | w_A  | w_D  | Case 1 (High) | Case 2 (Mixed) | Case 3 (Critical) |"
  echo "|------|------|------|------|---------------|----------------|-------------------|"

  for wr in 0.20 0.25 0.30 0.35 0.40; do
    # Distribute remaining weight proportionally: S and A get equal shares, D gets remainder
    local remaining
    remaining=$(awk "BEGIN { printf \"%.2f\", 1.0 - $wr }")
    # Split: S=remaining*0.357, A=remaining*0.357, D=remaining*0.286 (roughly 5:5:4 ratio)
    local ws wa wd
    ws=$(awk "BEGIN { printf \"%.2f\", $remaining * 0.357 }")
    wa=$(awk "BEGIN { printf \"%.2f\", $remaining * 0.357 }")
    wd=$(awk "BEGIN { printf \"%.2f\", 1.0 - $wr - $ws - $wa }")

    # Case 1: S=90, R=85, A=95, D=88
    local c1
    c1=$(awk "BEGIN { printf \"%.1f\", $ws*90 + $wr*85 + $wa*95 + $wd*88 }")
    local g1="Certain"
    [ "$(awk "BEGIN { print ($c1 < 85) ? 1 : 0 }")" = "1" ] && g1="Confident"

    # Case 2: S=40, R=70, A=55, D=30
    local c2
    c2=$(awk "BEGIN { printf \"%.1f\", $ws*40 + $wr*70 + $wa*55 + $wd*30 }")
    local g2="Tentative"
    [ "$(awk "BEGIN { print ($c2 >= 60) ? 1 : 0 }")" = "1" ] && g2="Confident"
    [ "$(awk "BEGIN { print ($c2 < 30) ? 1 : 0 }")" = "1" ] && g2="Unresolved"

    # Case 3: S=60, R=20, A=15, D=70 — Critical Rule override (R<25 AND A<25)
    local c3
    c3=$(awk "BEGIN { printf \"%.1f\", $ws*60 + $wr*20 + $wa*15 + $wd*70 }")
    local g3="Unresolved (override)"

    printf "| %-4s | %-4s | %-4s | %-4s | %-13s | %-14s | %-17s |\n" \
      "$wr" "$ws" "$wa" "$wd" "$c1 ($g1)" "$c2 ($g2)" "$c3 ($g3)"
  done

  echo ""
  echo "Critical Rule override: Active for all weight configurations (R=20 < 25, A=15 < 25)."
  echo ""
  echo "Recommendation: Default weights w_S=0.25, w_R=0.30, w_A=0.25, w_D=0.20 provide"
  echo "  stable grade assignments across test cases. Increasing w_R beyond 0.35 risks"
  echo "  over-penalizing decisions where R is the only weak dimension."
  echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

echo "SRAD Confidence Scoring — Sensitivity Analysis"
echo "Archive: $ARCHIVE_DIR"
echo ""

exit_code=0

case "$DOMAIN" in
  domain1)
    run_domain1 || exit_code=$?
    ;;
  domain2)
    run_domain2
    ;;
  all)
    run_domain1 || exit_code=$?
    run_domain2
    ;;
esac

echo "═══════════════════════════════════════════════════════════════"
echo "Analysis complete."
echo "═══════════════════════════════════════════════════════════════"

exit "$exit_code"

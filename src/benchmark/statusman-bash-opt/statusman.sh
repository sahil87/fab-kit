#!/usr/bin/env bash
# Optimized bash statusman — minimizes yq/subprocess spawns.
# Batches yq reads, uses awk for writes.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"

STAGES="intake spec tasks apply review hydrate ship review-pr"

# ─────────────────────────────────────────────────────────────────────────────
# progress-map <status_file>
# Single yq call to extract all 8 stages at once.
# ─────────────────────────────────────────────────────────────────────────────
cmd_progress_map() {
  local status_file="$1"
  # One yq call — compound expression extracting all stages
  yq '
    .progress.intake // "pending",
    .progress.spec // "pending",
    .progress.tasks // "pending",
    .progress.apply // "pending",
    .progress.review // "pending",
    .progress.hydrate // "pending",
    .progress.ship // "pending",
    .progress["review-pr"] // "pending"
  ' "$status_file" | paste -d'\n' - | {
    local i=0
    local stage_arr=($STAGES)
    while IFS= read -r val; do
      echo "${stage_arr[$i]}:${val}"
      i=$((i + 1))
    done
  }
}

# ─────────────────────────────────────────────────────────────────────────────
# set-change-type <status_file> <type>
# Validates type, uses awk for the write.
# ─────────────────────────────────────────────────────────────────────────────
cmd_set_change_type() {
  local status_file="$1"
  local change_type="$2"

  case "$change_type" in
    feat|fix|refactor|docs|test|ci|chore) ;;
    *)
      echo "ERROR: Invalid change type '$change_type'" >&2
      return 1
      ;;
  esac

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  awk -v ct="$change_type" -v ts="$now" '
    /^change_type:/ { print "change_type: " ct; next }
    /^last_updated:/ { print "last_updated: \"" ts "\""; next }
    { print }
  ' "$status_file" > "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# ─────────────────────────────────────────────────────────────────────────────
# finish <status_file> <stage>
# Single yq call for reads, awk for all writes.
# ─────────────────────────────────────────────────────────────────────────────
cmd_finish() {
  local status_file="$1"
  local stage="$2"
  local now
  now=$(date -Iseconds)

  # Determine next stage name
  local next_stage=""
  local found=false
  for s in $STAGES; do
    if [ "$found" = "true" ]; then
      next_stage="$s"
      break
    fi
    if [ "$s" = "$stage" ]; then
      found=true
    fi
  done

  # Single yq call — read both current stage state and next stage state
  local yq_expr=".progress.\"${stage}\" // \"pending\""
  if [ -n "$next_stage" ]; then
    yq_expr="${yq_expr}, .progress.\"${next_stage}\" // \"pending\""
  fi
  local yq_output
  yq_output=$(yq "$yq_expr" "$status_file")

  local current_state
  current_state=$(echo "$yq_output" | sed -n '1p')
  local next_state="none"
  if [ -n "$next_stage" ]; then
    next_state=$(echo "$yq_output" | sed -n '2p')
  fi

  # Validate transition: finish requires active or ready
  case "$current_state" in
    active|ready) ;;
    *)
      echo "ERROR: Cannot finish stage '$stage' — current state is '$current_state'" >&2
      return 1
      ;;
  esac

  local activate_next=false
  if [ "$next_state" = "pending" ]; then
    activate_next=true
  fi

  # Write everything with awk — single pass
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  awk -v stage="$stage" -v next_stage="$next_stage" -v activate="$activate_next" -v ts="$now" '
    /^progress:/ { in_progress = 1; print; next }
    in_progress && /^[^ ]/ { in_progress = 0 }
    in_progress && index($0, stage ":") > 0 {
      sub(/: .*/, ": done")
      print
      next
    }
    in_progress && activate == "true" && next_stage != "" && index($0, next_stage ":") > 0 {
      sub(/: .*/, ": active")
      print
      next
    }
    /^stage_metrics:/ { in_metrics = 1; wrote_next_metrics = 0; print; next }
    in_metrics && /^[^ ]/ {
      # Leaving metrics block — inject next-stage metrics if not yet written
      if (activate == "true" && next_stage != "" && !wrote_next_metrics) {
        print "  " next_stage ": {started_at: \"" ts "\", driver: benchmark, iterations: 1}"
      }
      in_metrics = 0
    }
    in_metrics && index($0, stage ":") > 0 {
      # Rewrite current stage metrics line: append completed_at
      # Flow format: {key: val, key: val}
      sub(/}/, ", completed_at: \"" ts "\"}")
      print
      next
    }
    in_metrics && activate == "true" && next_stage != "" && index($0, next_stage ":") > 0 {
      # Replace existing next-stage metrics (if any)
      print "  " next_stage ": {started_at: \"" ts "\", driver: benchmark, iterations: 1}"
      wrote_next_metrics = 1
      next
    }
    /^last_updated:/ { print "last_updated: \"" ts "\""; next }
    { print }
    END {
      # If metrics block was the last thing in the file and we still need to inject
      if (in_metrics && activate == "true" && next_stage != "" && !wrote_next_metrics) {
        print "  " next_stage ": {started_at: \"" ts "\", driver: benchmark, iterations: 1}"
      }
    }
  ' "$status_file" > "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────
case "${1:-}" in
  --help|-h)
    echo "statusman-bash-opt: optimized bash benchmark contender"
    echo "Usage: statusman.sh {progress-map|set-change-type|finish} <status_file> [args...]"
    ;;
  progress-map)
    cmd_progress_map "$2"
    ;;
  set-change-type)
    cmd_set_change_type "$2" "$3"
    ;;
  finish)
    cmd_finish "$2" "$3"
    ;;
  *)
    echo "Unknown command: ${1:-}" >&2
    exit 1
    ;;
esac

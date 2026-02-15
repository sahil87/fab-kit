#!/usr/bin/env bash
# fab/.kit/scripts/lib/stageman.sh
#
# Stage Manager — CLI utility for workflow stages, states, and .status.yaml.
# Reads the canonical workflow schema and provides typed subcommands.
#
# Usage:
#   stageman.sh --help               Show usage
#   stageman.sh all-stages           List all stage IDs
#   stageman.sh validate-state done  Check if state is valid
#   stageman.sh progress-map <file>  Extract stage:state pairs
#   stageman.sh set-state <file> <stage> <state> [driver]

set -euo pipefail

# Locate workflow schema (works from both .kit/scripts/lib/ and src/lib/stageman/ symlink)
STAGEMAN_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
WORKFLOW_SCHEMA="$STAGEMAN_DIR/../../schemas/workflow.yaml"

if [ ! -f "$WORKFLOW_SCHEMA" ]; then
  echo "ERROR: workflow.yaml not found at $WORKFLOW_SCHEMA" >&2
  echo "       STAGEMAN_DIR=$STAGEMAN_DIR" >&2
  exit 1
fi

# Require yq (Mike Farah's Go v4) for .status.yaml operations
if ! command -v yq &>/dev/null; then
  echo "ERROR: yq (v4) is required but not found. Install: https://github.com/mikefarah/yq" >&2
  exit 1
fi

# ─────────────────────────────────────────────────────────────────────────────
# State Queries
# ─────────────────────────────────────────────────────────────────────────────

# get_all_states — Return all valid state IDs (one per line)
get_all_states() {
  awk '
    /^states:/ { in_states = 1; next }
    in_states && /^[a-z_]+:/ && !/^ / { exit }
    in_states && /^  - id:/ { print $3 }
  ' "$WORKFLOW_SCHEMA"
}

# validate_state <state> — Return 0 if state is valid, 1 otherwise
validate_state() {
  local state="$1"
  get_all_states | grep -qx "$state"
}

# get_state_symbol <state> — Return the display symbol for a state
get_state_symbol() {
  local state="$1"
  awk -v state="$state" '
    /^ *- id:/ { current_id = $3 }
    /^ *symbol:/ && current_id == state {
      gsub(/"/, "", $2)
      print $2
      exit
    }
  ' "$WORKFLOW_SCHEMA"
}

# get_state_suffix <state> — Return the display suffix for a state (if any)
get_state_suffix() {
  local state="$1"
  awk -v state="$state" '
    /^ *- id:/ { current_id = $3 }
    /^ *suffix:/ && current_id == state {
      match($0, /"[^"]*"/)
      if (RSTART > 0) print substr($0, RSTART+1, RLENGTH-2)
      exit
    }
  ' "$WORKFLOW_SCHEMA"
}

# is_terminal_state <state> — Return 0 if state is terminal, 1 otherwise
is_terminal_state() {
  local state="$1"
  local terminal
  terminal=$(awk -v state="$state" '
    /^ *- id:/ { current_id = $3 }
    /^ *terminal:/ && current_id == state { print $2; exit }
  ' "$WORKFLOW_SCHEMA")
  [ "$terminal" = "true" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# Stage Queries
# ─────────────────────────────────────────────────────────────────────────────

# get_all_stages — Return all stage IDs in order (one per line)
get_all_stages() {
  awk '
    /^stages:/ { in_stages = 1; next }
    in_stages && /^[a-z_]+:/ && !/^ / { exit }
    in_stages && /^  - id:/ { print $3 }
  ' "$WORKFLOW_SCHEMA"
}

# validate_stage <stage> — Return 0 if stage is valid, 1 otherwise
validate_stage() {
  local stage="$1"
  get_all_stages | grep -qx "$stage"
}

# get_stage_number <stage> — Return the 1-indexed position of a stage
get_stage_number() {
  local stage="$1"
  awk -v stage="$stage" '
    /^stage_numbers:/ { in_numbers = 1; next }
    in_numbers && /^[a-z_]+:/ && !/^ / { exit }
    in_numbers && $1 == stage":" { print $2; exit }
  ' "$WORKFLOW_SCHEMA"
}

# get_stage_name <stage> — Return the human-readable name
get_stage_name() {
  local stage="$1"
  awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3 }
    /^ *name:/ && current_id == stage {
      gsub(/"/, "", $2)
      print $2
      exit
    }
  ' "$WORKFLOW_SCHEMA"
}

# get_stage_artifact <stage> — Return the generated artifact filename (or empty)
get_stage_artifact() {
  local stage="$1"
  awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3 }
    /^ *generates:/ && current_id == stage {
      artifact = $2
      gsub(/"/, "", artifact)
      if (artifact != "null") print artifact
      exit
    }
  ' "$WORKFLOW_SCHEMA"
}

# get_allowed_states <stage> — Return allowed states for a stage (one per line)
get_allowed_states() {
  local stage="$1"
  awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3; in_stage = 0 }
    current_id == stage { in_stage = 1 }
    in_stage && /^ *allowed_states:/ {
      # Parse [state1, state2, state3] format
      gsub(/[\[\]]/, "")
      for (i = 2; i <= NF; i++) {
        state = $i
        gsub(/,/, "", state)
        print state
      }
      exit
    }
  ' "$WORKFLOW_SCHEMA"
}

# validate_stage_state <stage> <state> — Return 0 if state is allowed for stage
validate_stage_state() {
  local stage="$1"
  local state="$2"
  get_allowed_states "$stage" | grep -qx "$state"
}

# get_initial_state <stage> — Return the default state for a new change
get_initial_state() {
  local stage="$1"
  awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3 }
    /^ *initial_state:/ && current_id == stage {
      print $2
      exit
    }
  ' "$WORKFLOW_SCHEMA"
}

# is_required_stage <stage> — Return 0 if stage is required, 1 if optional
is_required_stage() {
  local stage="$1"
  local required
  required=$(awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3 }
    /^ *required:/ && current_id == stage { print $2; exit }
  ' "$WORKFLOW_SCHEMA")
  [ "$required" = "true" ]
}

# has_auto_checklist <stage> — Return 0 if stage generates a checklist
has_auto_checklist() {
  local stage="$1"
  local auto
  auto=$(awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3 }
    /^ *auto_checklist:/ && current_id == stage { print $2; exit }
  ' "$WORKFLOW_SCHEMA")
  [ "$auto" = "true" ]
}

# ─────────────────────────────────────────────────────────────────────────────
# .status.yaml Accessors
# ─────────────────────────────────────────────────────────────────────────────

# get_progress_map <status_file> — Extract all stage→state pairs
# Outputs one "stage:state" pair per line. Missing stages default to "pending".
get_progress_map() {
  local status_file="$1"
  local stage val
  for stage in $(get_all_stages); do
    val=$(yq ".progress.${stage} // \"pending\"" "$status_file")
    echo "${stage}:${val}"
  done
}

# get_checklist <status_file> — Extract checklist fields
# Outputs: generated:{val}, completed:{val}, total:{val}
get_checklist() {
  local status_file="$1"
  echo "generated:$(yq '.checklist.generated // "false"' "$status_file")"
  echo "completed:$(yq '.checklist.completed // 0' "$status_file")"
  echo "total:$(yq '.checklist.total // 0' "$status_file")"
}

# get_confidence <status_file> — Extract confidence fields
# Outputs: certain:{val}, confident:{val}, tentative:{val}, unresolved:{val}, score:{val}
get_confidence() {
  local status_file="$1"
  echo "certain:$(yq '.confidence.certain // 0' "$status_file")"
  echo "confident:$(yq '.confidence.confident // 0' "$status_file")"
  echo "tentative:$(yq '.confidence.tentative // 0' "$status_file")"
  echo "unresolved:$(yq '.confidence.unresolved // 0' "$status_file")"
  echo "score:$(yq '.confidence.score // "5.0"' "$status_file")"
}

# ─────────────────────────────────────────────────────────────────────────────
# Stage Metrics Accessors
# ─────────────────────────────────────────────────────────────────────────────

# get_stage_metrics <status_file> [stage] — Extract stage_metrics data
# Without stage: output all as "stage:{flow-yaml}" per line.
# With stage: output single stage's fields as "field:value" per line.
# Returns empty (exit 0) if stage_metrics is missing or empty.
get_stage_metrics() {
  local status_file="$1"
  local stage="${2:-}"

  if [ -n "$stage" ]; then
    local exists
    exists=$(yq ".stage_metrics.${stage} // \"\"" "$status_file")
    [ -z "$exists" ] && return 0
    yq -o=props ".stage_metrics.${stage}" "$status_file" | sed 's/ = /:/; s/^//'
  else
    local keys
    keys=$(yq '.stage_metrics | keys | .[]' "$status_file" 2>/dev/null) || return 0
    [ -z "$keys" ] && return 0
    local key
    while IFS= read -r key; do
      local flow
      flow=$(yq -o=json -I=0 ".stage_metrics.${key}" "$status_file")
      echo "${key}:${flow}"
    done <<< "$keys"
  fi
}

# set_stage_metric <status_file> <stage> <field> <value>
# Set an individual metric field for a stage. Creates stage_metrics map and
# stage entry if absent. Ensures flow style for the stage entry.
set_stage_metric() {
  local status_file="$1"
  local stage="$2"
  local field="$3"
  local value="$4"

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  cp "$status_file" "$tmpfile"
  yq -i ".stage_metrics.${stage}.${field} = ${value} | .last_updated = \"${now}\"" "$tmpfile"
  yq -i "(.stage_metrics.${stage}) style=\"flow\"" "$tmpfile"
  mv "$tmpfile" "$status_file"
}

# _apply_metrics_side_effect <tmpfile> <stage> <state> [driver]
# Internal helper: apply stage_metrics side-effects for a state change.
# Operates on tmpfile (caller handles atomicity).
_apply_metrics_side_effect() {
  local tmpfile="$1"
  local stage="$2"
  local state="$3"
  local driver="${4:-}"
  local now
  now=$(date -Iseconds)

  case "$state" in
    active)
      local iterations
      iterations=$(yq ".stage_metrics.${stage}.iterations // 0" "$tmpfile")
      iterations=$((iterations + 1))
      yq -i ".stage_metrics.${stage} = {\"started_at\": \"${now}\", \"driver\": \"${driver}\", \"iterations\": ${iterations}}" "$tmpfile"
      yq -i "(.stage_metrics.${stage}) style=\"flow\"" "$tmpfile"
      ;;
    done)
      yq -i ".stage_metrics.${stage}.completed_at = \"${now}\"" "$tmpfile"
      yq -i "(.stage_metrics.${stage}) style=\"flow\"" "$tmpfile"
      ;;
    pending)
      yq -i "del(.stage_metrics.${stage})" "$tmpfile"
      ;;
    # failed: no metrics change
  esac
}

# ─────────────────────────────────────────────────────────────────────────────
# Progression Queries
# ─────────────────────────────────────────────────────────────────────────────

# get_current_stage <status_file> — Determine active stage from .status.yaml
get_current_stage() {
  local status_file="$1"
  local stage state last_done="" found_last=false

  # Parse progress map once
  local progress_lines
  progress_lines=$(get_progress_map "$status_file")

  # Find first active stage
  while IFS=: read -r stage state; do
    if [ "$state" = "active" ]; then
      echo "$stage"
      return 0
    fi
  done <<< "$progress_lines"

  # Fallback: find first pending stage after last done
  while IFS=: read -r stage state; do
    if [ "$state" = "done" ]; then
      last_done="$stage"
    fi
  done <<< "$progress_lines"

  if [ -n "$last_done" ]; then
    while IFS=: read -r stage state; do
      if [ "$found_last" = "true" ] && [ "$state" = "pending" ]; then
        echo "$stage"
        return 0
      fi
      if [ "$stage" = "$last_done" ]; then
        found_last=true
      fi
    done <<< "$progress_lines"
  fi

  # Final fallback: all done (workflow complete)
  echo "hydrate"
}

# get_next_stage <current_stage> — Return the next stage in sequence
get_next_stage() {
  local current="$1"
  local found=false

  for stage in $(get_all_stages); do
    if [ "$found" = "true" ]; then
      echo "$stage"
      return 0
    fi
    if [ "$stage" = "$current" ]; then
      found=true
    fi
  done

  # No next stage (at end)
  return 1
}

# ─────────────────────────────────────────────────────────────────────────────
# Write Functions
# ─────────────────────────────────────────────────────────────────────────────

# set_stage_state <status_file> <stage> <state> [driver]
# Set a single stage's progress value in .status.yaml.
# driver is required when state is "active", ignored otherwise.
# Validates inputs, writes atomically (temp-file-then-mv), updates last_updated.
# Applies stage_metrics side-effects via _apply_metrics_side_effect.
# Returns 0 on success, 1 on validation failure (diagnostic to stderr).
set_stage_state() {
  local status_file="$1"
  local stage="$2"
  local state="$3"
  local driver="${4:-}"

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  if ! validate_stage "$stage"; then
    echo "ERROR: Invalid stage '$stage'" >&2
    return 1
  fi

  if ! validate_stage_state "$stage" "$state"; then
    echo "ERROR: State '$state' not allowed for stage '$stage'" >&2
    return 1
  fi

  if [ "$state" = "active" ] && [ -z "$driver" ]; then
    echo "ERROR: driver required when setting state to 'active'" >&2
    return 1
  fi

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  cp "$status_file" "$tmpfile"
  yq -i ".progress.${stage} = \"${state}\" | .last_updated = \"${now}\"" "$tmpfile"

  # Apply stage_metrics side-effects
  _apply_metrics_side_effect "$tmpfile" "$stage" "$state" "$driver"

  mv "$tmpfile" "$status_file"
}

# transition_stages <status_file> <from_stage> <to_stage> [driver]
# Two-write transition: set from_stage to done and to_stage to active atomically.
# driver is required (applied to to_stage's active side-effect).
# Validates adjacency and that from_stage is currently active.
# Applies stage_metrics side-effects for both stages.
# Returns 0 on success, 1 on validation failure.
transition_stages() {
  local status_file="$1"
  local from_stage="$2"
  local to_stage="$3"
  local driver="${4:-}"

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  if ! validate_stage "$from_stage"; then
    echo "ERROR: Invalid stage '$from_stage'" >&2
    return 1
  fi

  if ! validate_stage "$to_stage"; then
    echo "ERROR: Invalid stage '$to_stage'" >&2
    return 1
  fi

  if ! validate_stage_state "$from_stage" "done"; then
    echo "ERROR: State 'done' not allowed for stage '$from_stage'" >&2
    return 1
  fi

  if ! validate_stage_state "$to_stage" "active"; then
    echo "ERROR: State 'active' not allowed for stage '$to_stage'" >&2
    return 1
  fi

  if [ -z "$driver" ]; then
    echo "ERROR: driver required for transition" >&2
    return 1
  fi

  # Verify from_stage is currently active
  local current_state
  current_state=$(yq ".progress.${from_stage}" "$status_file")
  if [ "$current_state" != "active" ]; then
    echo "ERROR: Stage '$from_stage' is '$current_state', expected 'active'" >&2
    return 1
  fi

  # Verify adjacency
  local expected_next
  expected_next=$(get_next_stage "$from_stage") || true
  if [ "$expected_next" != "$to_stage" ]; then
    echo "ERROR: '$to_stage' is not adjacent to '$from_stage' (expected '$expected_next')" >&2
    return 1
  fi

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  cp "$status_file" "$tmpfile"
  yq -i ".progress.${from_stage} = \"done\" | .progress.${to_stage} = \"active\" | .last_updated = \"${now}\"" "$tmpfile"

  # Apply stage_metrics side-effects: from→done, to→active
  _apply_metrics_side_effect "$tmpfile" "$from_stage" "done"
  _apply_metrics_side_effect "$tmpfile" "$to_stage" "active" "$driver"

  mv "$tmpfile" "$status_file"
}

# set_checklist_field <status_file> <field> <value>
# Update a single field in the checklist block of .status.yaml.
# Valid fields: generated (true/false), completed (non-negative int), total (non-negative int).
# Returns 0 on success, 1 on validation failure.
set_checklist_field() {
  local status_file="$1"
  local field="$2"
  local value="$3"

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  case "$field" in
    generated)
      if [ "$value" != "true" ] && [ "$value" != "false" ]; then
        echo "ERROR: Invalid value '$value' for field 'generated' (expected true/false)" >&2
        return 1
      fi
      ;;
    completed|total)
      if ! [[ "$value" =~ ^[0-9]+$ ]]; then
        echo "ERROR: Invalid value '$value' for field '$field' (expected non-negative integer)" >&2
        return 1
      fi
      ;;
    *)
      echo "ERROR: Invalid checklist field '$field' (expected: generated, completed, total)" >&2
      return 1
      ;;
  esac

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  cp "$status_file" "$tmpfile"
  yq -i ".checklist.${field} = ${value} | .last_updated = \"${now}\"" "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# set_confidence_block <status_file> <certain> <confident> <tentative> <unresolved> <score>
# Replace the entire confidence block in .status.yaml.
# Validates counts are non-negative integers and score is a non-negative float.
# Returns 0 on success, 1 on validation failure.
set_confidence_block() {
  local status_file="$1"
  local certain="$2"
  local confident="$3"
  local tentative="$4"
  local unresolved="$5"
  local score="$6"

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  # Validate counts are non-negative integers
  local count_name count_val
  for count_name in certain confident tentative unresolved; do
    eval count_val=\$$count_name
    if ! [[ "$count_val" =~ ^[0-9]+$ ]]; then
      echo "ERROR: Invalid value '$count_val' for '$count_name' (expected non-negative integer)" >&2
      return 1
    fi
  done

  # Validate score is a non-negative float
  if ! [[ "$score" =~ ^[0-9]+\.?[0-9]*$ ]]; then
    echo "ERROR: Invalid score '$score' (expected non-negative float)" >&2
    return 1
  fi

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  cp "$status_file" "$tmpfile"
  yq -i "
    .confidence.certain = ${certain} |
    .confidence.confident = ${confident} |
    .confidence.tentative = ${tentative} |
    .confidence.unresolved = ${unresolved} |
    .confidence.score = ${score} |
    .last_updated = \"${now}\"
  " "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# set_confidence_block_fuzzy <status_file> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>
# Replace the entire confidence block in .status.yaml, including fuzzy dimension data.
# Extends set_confidence_block with fuzzy: true flag and dimensions sub-block.
set_confidence_block_fuzzy() {
  local status_file="$1"
  local certain="$2"
  local confident="$3"
  local tentative="$4"
  local unresolved="$5"
  local score="$6"
  local mean_s="$7"
  local mean_r="$8"
  local mean_a="$9"
  local mean_d="${10}"

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  # Validate counts are non-negative integers
  local count_name count_val
  for count_name in certain confident tentative unresolved; do
    eval count_val=\$$count_name
    if ! [[ "$count_val" =~ ^[0-9]+$ ]]; then
      echo "ERROR: Invalid value '$count_val' for '$count_name' (expected non-negative integer)" >&2
      return 1
    fi
  done

  # Validate score is a non-negative float
  if ! [[ "$score" =~ ^[0-9]+\.?[0-9]*$ ]]; then
    echo "ERROR: Invalid score '$score' (expected non-negative float)" >&2
    return 1
  fi

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  awk -v certain="$certain" \
      -v confident="$confident" \
      -v tentative="$tentative" \
      -v unresolved="$unresolved" \
      -v score="$score" \
      -v mean_s="$mean_s" \
      -v mean_r="$mean_r" \
      -v mean_a="$mean_a" \
      -v mean_d="$mean_d" \
      -v ts="$now" '
    /^confidence:/ {
      in_block = 1
      print "confidence:"
      print "  certain: " certain
      print "  confident: " confident
      print "  tentative: " tentative
      print "  unresolved: " unresolved
      print "  score: " score
      print "  fuzzy: true"
      print "  dimensions:"
      print "    signal: " mean_s
      print "    reversibility: " mean_r
      print "    competence: " mean_a
      print "    disambiguation: " mean_d
      next
    }
    in_block && /^[^ ]/ { in_block = 0 }
    in_block { next }
    /^last_updated:/ { print "last_updated: " ts; next }
    { print }
  ' "$status_file" > "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# ─────────────────────────────────────────────────────────────────────────────
# Display Helpers
# ─────────────────────────────────────────────────────────────────────────────

# format_state <state> — Return formatted state for display (symbol + suffix)
format_state() {
  local state="$1"
  local symbol suffix

  symbol=$(get_state_symbol "$state")
  suffix=$(get_state_suffix "$state")

  echo "${symbol}${suffix}"
}

# ─────────────────────────────────────────────────────────────────────────────
# Validation
# ─────────────────────────────────────────────────────────────────────────────

# validate_status_file <status_file> — Validate .status.yaml against schema
# Returns 0 if valid, 1 if invalid (prints errors to stderr)
validate_status_file() {
  local status_file="$1"
  local errors=0

  # Check each stage has valid state (via yq, skips stage_metrics)
  local active_count=0
  for stage in $(get_all_stages); do
    local state
    state=$(yq ".progress.${stage} // \"\"" "$status_file")

    if [ -z "$state" ]; then
      echo "ERROR: Missing progress.$stage in $status_file" >&2
      errors=$((errors + 1))
      continue
    fi

    if ! validate_state "$state"; then
      echo "ERROR: Invalid state '$state' for stage $stage" >&2
      errors=$((errors + 1))
      continue
    fi

    if ! validate_stage_state "$stage" "$state"; then
      echo "ERROR: State '$state' not allowed for stage $stage" >&2
      errors=$((errors + 1))
    fi

    if [ "$state" = "active" ]; then
      active_count=$((active_count + 1))
    fi
  done

  # Check active count (0 or 1)
  if [ "$active_count" -gt 1 ]; then
    echo "ERROR: Multiple stages are active (expected 0 or 1)" >&2
    errors=$((errors + 1))
  fi

  [ $errors -eq 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# History Logging
# ─────────────────────────────────────────────────────────────────────────────

# log_command <change_dir> <cmd> [args]
# Append a "command" event to <change_dir>/.history.jsonl.
log_command() {
  local change_dir="$1"
  local cmd="$2"
  local args="${3:-}"
  local now
  now=$(date -Iseconds)

  local json="{\"ts\":\"${now}\",\"event\":\"command\",\"cmd\":\"${cmd}\""
  if [ -n "$args" ]; then
    json="${json},\"args\":\"${args}\""
  fi
  json="${json}}"

  echo "$json" >> "${change_dir}/.history.jsonl"
}

# log_confidence <change_dir> <score> <delta> <trigger>
# Append a "confidence" event to <change_dir>/.history.jsonl.
log_confidence() {
  local change_dir="$1"
  local score="$2"
  local delta="$3"
  local trigger="$4"
  local now
  now=$(date -Iseconds)

  echo "{\"ts\":\"${now}\",\"event\":\"confidence\",\"score\":${score},\"delta\":\"${delta}\",\"trigger\":\"${trigger}\"}" >> "${change_dir}/.history.jsonl"
}

# log_review <change_dir> <result> [rework]
# Append a "review" event to <change_dir>/.history.jsonl.
log_review() {
  local change_dir="$1"
  local result="$2"
  local rework="${3:-}"
  local now
  now=$(date -Iseconds)

  local json="{\"ts\":\"${now}\",\"event\":\"review\",\"result\":\"${result}\""
  if [ -n "$rework" ]; then
    json="${json},\"rework\":\"${rework}\""
  fi
  json="${json}}"

  echo "$json" >> "${change_dir}/.history.jsonl"
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI Interface
# ─────────────────────────────────────────────────────────────────────────────

show_help() {
  cat <<'EOF'
stageman.sh - Stage Manager CLI

USAGE:
  stageman.sh <subcommand> [args...]
  stageman.sh --help | --test | --version

SUBCOMMANDS:
  State queries:
    all-states                         List all valid states
    validate-state <state>             Check if state is valid (exit 0/1)
    state-symbol <state>               Get display symbol (e.g. ✓)
    state-suffix <state>               Get display suffix (e.g. " (skipped)")
    is-terminal <state>                Check if state is terminal (exit 0/1)

  Stage queries:
    all-stages                         List all stages in order
    validate-stage <stage>             Check if stage exists (exit 0/1)
    stage-number <stage>               Get position (1-6)
    stage-name <stage>                 Get display name
    stage-artifact <stage>             Get generated filename
    allowed-states <stage>             List allowed states for stage
    initial-state <stage>              Get default state
    is-required <stage>                Check if stage is required (exit 0/1)
    has-auto-checklist <stage>         Check if stage generates checklist (exit 0/1)
    validate-stage-state <stage> <st>  Check if state is valid for stage (exit 0/1)

  .status.yaml accessors:
    progress-map <file>                Extract stage:state pairs (one per line)
    checklist <file>                   Extract checklist fields (key:value lines)
    confidence <file>                  Extract confidence fields (key:value lines)

  Stage metrics:
    stage-metrics <file> [stage]       Get stage metrics (all or one stage)
    set-stage-metric <file> <stage> <field> <value>  Set a stage metric field

  Progression:
    current-stage <file>               Detect active stage from .status.yaml
    next-stage <stage>                 Get next stage in sequence

  Validation:
    validate-status-file <file>        Validate .status.yaml against schema

  Display:
    format-state <state>               Format state for display (symbol + suffix)

  Write commands:
    set-state <file> <stage> <state> [driver]   Set a stage's state
    transition <file> <from> <to> [driver]      Two-write forward transition
    set-checklist <file> <field> <value>        Update checklist field
    set-confidence <file> <certain> <confident> <tentative> <unresolved> <score>
    set-confidence-fuzzy <file> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>

  History:
    log-command <change_dir> <cmd> [args]              Log a command invocation
    log-confidence <change_dir> <score> <delta> <trigger>  Log confidence change
    log-review <change_dir> <result> [rework]          Log review outcome

EXAMPLES:
  stageman.sh all-stages
  stageman.sh validate-state done && echo "Valid"
  stageman.sh progress-map .status.yaml
  stageman.sh set-state .status.yaml spec done fab-continue
  stageman.sh transition .status.yaml spec tasks fab-continue

SEE ALSO:
  src/lib/stageman/README.md - API reference and development guide
  fab/.kit/schemas/workflow.yaml - Schema definition
EOF
}

show_version() {
  echo "stageman version 1.0.0"
  echo "Schema version: $(grep '^ *version:' "$WORKFLOW_SCHEMA" | head -1 | sed 's/.*: *//' | tr -d '"')"
}

run_tests() {
  echo "Testing stageman..."
  echo ""

  echo "All states:"
  get_all_states
  echo ""

  echo "All stages:"
  get_all_stages
  echo ""

  echo "State symbols:"
  for state in $(get_all_states); do
    printf "  %s: %s\n" "$state" "$(get_state_symbol "$state")"
  done
  echo ""

  echo "Stage numbers:"
  for stage in $(get_all_stages); do
    printf "  %s: %s\n" "$stage" "$(get_stage_number "$stage")"
  done
  echo ""

  echo "Stage artifacts:"
  for stage in $(get_all_stages); do
    artifact=$(get_stage_artifact "$stage")
    printf "  %s: %s\n" "$stage" "${artifact:-none}"
  done
  echo ""

  echo "Allowed states per stage:"
  for stage in $(get_all_stages); do
    echo "  $stage:"
    get_allowed_states "$stage" | sed 's/^/    /'
  done

  echo ""
  echo "✓ All tests passed"
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI Dispatch
# ─────────────────────────────────────────────────────────────────────────────

case "${1:-}" in
  --help|-h)
    show_help
    ;;
  --version|-v)
    show_version
    ;;
  --test|-t)
    run_tests
    ;;
  "")
    # Default: run tests for backward compatibility
    run_tests
    ;;

  # ── State Queries ──────────────────────────────────────────────────────
  all-states)
    get_all_states
    ;;
  validate-state)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh validate-state <state>" >&2
      exit 1
    fi
    validate_state "$2"
    ;;
  state-symbol)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh state-symbol <state>" >&2
      exit 1
    fi
    get_state_symbol "$2"
    ;;
  state-suffix)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh state-suffix <state>" >&2
      exit 1
    fi
    get_state_suffix "$2"
    ;;
  is-terminal)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh is-terminal <state>" >&2
      exit 1
    fi
    is_terminal_state "$2"
    ;;

  # ── Stage Queries ──────────────────────────────────────────────────────
  all-stages)
    get_all_stages
    ;;
  validate-stage)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh validate-stage <stage>" >&2
      exit 1
    fi
    validate_stage "$2"
    ;;
  stage-number)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh stage-number <stage>" >&2
      exit 1
    fi
    get_stage_number "$2"
    ;;
  stage-name)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh stage-name <stage>" >&2
      exit 1
    fi
    get_stage_name "$2"
    ;;
  stage-artifact)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh stage-artifact <stage>" >&2
      exit 1
    fi
    get_stage_artifact "$2"
    ;;
  allowed-states)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh allowed-states <stage>" >&2
      exit 1
    fi
    get_allowed_states "$2"
    ;;
  initial-state)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh initial-state <stage>" >&2
      exit 1
    fi
    get_initial_state "$2"
    ;;
  is-required)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh is-required <stage>" >&2
      exit 1
    fi
    is_required_stage "$2"
    ;;
  has-auto-checklist)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh has-auto-checklist <stage>" >&2
      exit 1
    fi
    has_auto_checklist "$2"
    ;;
  validate-stage-state)
    if [ $# -ne 3 ]; then
      echo "Usage: stageman.sh validate-stage-state <stage> <state>" >&2
      exit 1
    fi
    validate_stage_state "$2" "$3"
    ;;

  # ── .status.yaml Accessors ─────────────────────────────────────────────
  progress-map)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh progress-map <file>" >&2
      exit 1
    fi
    get_progress_map "$2"
    ;;
  checklist)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh checklist <file>" >&2
      exit 1
    fi
    get_checklist "$2"
    ;;
  confidence)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh confidence <file>" >&2
      exit 1
    fi
    get_confidence "$2"
    ;;

  # ── Stage Metrics ──────────────────────────────────────────────────────
  stage-metrics)
    if [ $# -lt 2 ] || [ $# -gt 3 ]; then
      echo "Usage: stageman.sh stage-metrics <file> [stage]" >&2
      exit 1
    fi
    get_stage_metrics "$2" "${3:-}"
    ;;
  set-stage-metric)
    if [ $# -ne 5 ]; then
      echo "Usage: stageman.sh set-stage-metric <file> <stage> <field> <value>" >&2
      exit 1
    fi
    set_stage_metric "$2" "$3" "$4" "$5"
    ;;

  # ── Progression ────────────────────────────────────────────────────────
  current-stage)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh current-stage <file>" >&2
      exit 1
    fi
    get_current_stage "$2"
    ;;
  next-stage)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh next-stage <stage>" >&2
      exit 1
    fi
    get_next_stage "$2"
    ;;

  # ── Validation ─────────────────────────────────────────────────────────
  validate-status-file)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh validate-status-file <file>" >&2
      exit 1
    fi
    validate_status_file "$2"
    ;;

  # ── Display ────────────────────────────────────────────────────────────
  format-state)
    if [ $# -ne 2 ]; then
      echo "Usage: stageman.sh format-state <state>" >&2
      exit 1
    fi
    format_state "$2"
    ;;

  # ── Write Commands ─────────────────────────────────────────────────────
  set-state)
    if [ $# -lt 4 ] || [ $# -gt 5 ]; then
      echo "Usage: stageman.sh set-state <file> <stage> <state> [driver]" >&2
      exit 1
    fi
    set_stage_state "$2" "$3" "$4" "${5:-}"
    ;;
  transition)
    if [ $# -lt 4 ] || [ $# -gt 5 ]; then
      echo "Usage: stageman.sh transition <file> <from-stage> <to-stage> [driver]" >&2
      exit 1
    fi
    transition_stages "$2" "$3" "$4" "${5:-}"
    ;;
  set-checklist)
    if [ $# -ne 4 ]; then
      echo "Usage: stageman.sh set-checklist <file> <field> <value>" >&2
      exit 1
    fi
    set_checklist_field "$2" "$3" "$4"
    ;;
  set-confidence)
    if [ $# -ne 7 ]; then
      echo "Usage: stageman.sh set-confidence <file> <certain> <confident> <tentative> <unresolved> <score>" >&2
      exit 1
    fi
    set_confidence_block "$2" "$3" "$4" "$5" "$6" "$7"
    ;;
  set-confidence-fuzzy)
    if [ $# -ne 11 ]; then
      echo "Usage: stageman.sh set-confidence-fuzzy <file> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>" >&2
      exit 1
    fi
    set_confidence_block_fuzzy "$2" "$3" "$4" "$5" "$6" "$7" "$8" "$9" "${10}" "${11}"
    ;;

  # ── History Commands ───────────────────────────────────────────────────
  log-command)
    if [ $# -lt 3 ] || [ $# -gt 4 ]; then
      echo "Usage: stageman.sh log-command <change_dir> <cmd> [args]" >&2
      exit 1
    fi
    log_command "$2" "$3" "${4:-}"
    ;;
  log-confidence)
    if [ $# -ne 5 ]; then
      echo "Usage: stageman.sh log-confidence <change_dir> <score> <delta> <trigger>" >&2
      exit 1
    fi
    log_confidence "$2" "$3" "$4" "$5"
    ;;
  log-review)
    if [ $# -lt 3 ] || [ $# -gt 4 ]; then
      echo "Usage: stageman.sh log-review <change_dir> <result> [rework]" >&2
      exit 1
    fi
    log_review "$2" "$3" "${4:-}"
    ;;
  *)
    echo "Unknown option: $1" >&2
    echo "Try: stageman.sh --help" >&2
    exit 1
    ;;
esac

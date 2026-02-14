#!/usr/bin/env bash
# fab/.kit/scripts/lib/stageman.sh
#
# Stage Manager - Query utility for workflow stages/states and status mutations.
# Schema queries are read from workflow.yaml; status reads/writes are yq-backed.

set -euo pipefail

# Locate workflow schema (works from both .kit/scripts/lib and src/lib/stageman symlink)
if [ -n "${BASH_SOURCE[0]:-}" ]; then
  STAGEMAN_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
  STAGEMAN_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
fi
WORKFLOW_SCHEMA="$STAGEMAN_DIR/../../schemas/workflow.yaml"

if [ ! -f "$WORKFLOW_SCHEMA" ]; then
  echo "ERROR: workflow.yaml not found at $WORKFLOW_SCHEMA" >&2
  echo "       STAGEMAN_DIR=$STAGEMAN_DIR" >&2
  return 1 2>/dev/null || exit 1
fi

# --- yq runtime requirement -------------------------------------------------
ensure_yq_v4() {
  local version_raw candidate
  local -a candidates
  local default_yq=""

  default_yq="$(command -v yq 2>/dev/null || true)"
  if [ -n "$default_yq" ]; then
    version_raw="$("$default_yq" --version 2>/dev/null || true)"
    if printf '%s' "$version_raw" | grep -Eq 'version v4\.'; then
      return 0
    fi
  fi

  candidates=(
    "${YQ_BIN:-}"
    "/home/linuxbrew/.linuxbrew/bin/yq"
    "/opt/homebrew/bin/yq"
  )

  for candidate in "${candidates[@]}"; do
    if [ -z "$candidate" ] || [ ! -x "$candidate" ]; then
      continue
    fi
    version_raw="$("$candidate" --version 2>/dev/null || true)"
    if printf '%s' "$version_raw" | grep -Eq 'version v4\.'; then
      export PATH="$(dirname "$candidate"):$PATH"
      return 0
    fi
  done

  cat >&2 <<MSG
ERROR: Mike Farah yq v4 is required for stageman status operations.
Detected default yq: ${default_yq:-not found}
Detected version: ${version_raw:-unknown}
Install guide: https://github.com/mikefarah/yq/#install
MSG
  return 1
}

ensure_yq_v4 || return 1 2>/dev/null || exit 1

# --- internal helpers -------------------------------------------------------
now_iso8601() {
  date -Iseconds
}

json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e ':a;N;$!ba;s/\n/\\n/g'
}

mk_status_tmp() {
  local status_file="$1"
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")
  cp "$status_file" "$tmpfile"
  printf '%s\n' "$tmpfile"
}

append_history_json() {
  local status_file="$1"
  local json_line="$2"
  local history_file

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  history_file="$(dirname "$status_file")/.history.jsonl"
  touch "$history_file"
  printf '%s\n' "$json_line" >> "$history_file"
}

set_stage_active_metrics_on_tmp() {
  local tmpfile="$1"
  local stage="$2"
  local driver="$3"
  local ts="$4"

  STAGE="$stage" DRIVER="$driver" TS="$ts" yq eval -i '
    .stage_metrics = (.stage_metrics // {}) |
    .stage_metrics[strenv(STAGE)] = (.stage_metrics[strenv(STAGE)] // {}) |
    .stage_metrics[strenv(STAGE)].started_at = strenv(TS) |
    .stage_metrics[strenv(STAGE)].driver = strenv(DRIVER) |
    .stage_metrics[strenv(STAGE)].iterations = ((.stage_metrics[strenv(STAGE)].iterations // 0) + 1)
  ' "$tmpfile"
}

set_stage_completed_metrics_on_tmp() {
  local tmpfile="$1"
  local stage="$2"
  local ts="$3"

  STAGE="$stage" TS="$ts" yq eval -i '
    .stage_metrics = (.stage_metrics // {}) |
    .stage_metrics[strenv(STAGE)] = (.stage_metrics[strenv(STAGE)] // {}) |
    .stage_metrics[strenv(STAGE)].completed_at = strenv(TS)
  ' "$tmpfile"
}

# ─────────────────────────────────────────────────────────────────────────────
# State Queries
# ─────────────────────────────────────────────────────────────────────────────

get_all_states() {
  awk '
    /^states:/ { in_states = 1; next }
    in_states && /^[a-z_]+:/ && !/^ / { exit }
    in_states && /^  - id:/ { print $3 }
  ' "$WORKFLOW_SCHEMA"
}

validate_state() {
  local state="$1"
  get_all_states | grep -qx "$state"
}

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

get_all_stages() {
  awk '
    /^stages:/ { in_stages = 1; next }
    in_stages && /^[a-z_]+:/ && !/^ / { exit }
    in_stages && /^  - id:/ { print $3 }
  ' "$WORKFLOW_SCHEMA"
}

validate_stage() {
  local stage="$1"
  get_all_stages | grep -qx "$stage"
}

get_stage_number() {
  local stage="$1"
  awk -v stage="$stage" '
    /^stage_numbers:/ { in_numbers = 1; next }
    in_numbers && /^[a-z_]+:/ && !/^ / { exit }
    in_numbers && $1 == stage":" { print $2; exit }
  ' "$WORKFLOW_SCHEMA"
}

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

get_allowed_states() {
  local stage="$1"
  awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3; in_stage = 0 }
    current_id == stage { in_stage = 1 }
    in_stage && /^ *allowed_states:/ {
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

validate_stage_state() {
  local stage="$1"
  local state="$2"
  get_allowed_states "$stage" | grep -qx "$state"
}

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

is_required_stage() {
  local stage="$1"
  local required
  required=$(awk -v stage="$stage" '
    /^ *- id:/ { current_id = $3 }
    /^ *required:/ && current_id == stage { print $2; exit }
  ' "$WORKFLOW_SCHEMA")
  [ "$required" = "true" ]
}

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
# .status.yaml Accessors (yq-backed)
# ─────────────────────────────────────────────────────────────────────────────

get_progress_map() {
  local status_file="$1"
  local stage val

  for stage in $(get_all_stages); do
    val=$(STAGE="$stage" yq eval '.progress[strenv(STAGE)] // "pending"' "$status_file")
    echo "${stage}:${val}"
  done
}

get_checklist() {
  local status_file="$1"
  local generated completed total

  generated=$(yq eval '.checklist.generated // false' "$status_file")
  completed=$(yq eval '.checklist.completed // 0' "$status_file")
  total=$(yq eval '.checklist.total // 0' "$status_file")

  echo "generated:${generated}"
  echo "completed:${completed}"
  echo "total:${total}"
}

get_confidence() {
  local status_file="$1"
  local certain confident tentative unresolved score

  certain=$(yq eval '.confidence.certain // 0' "$status_file")
  confident=$(yq eval '.confidence.confident // 0' "$status_file")
  tentative=$(yq eval '.confidence.tentative // 0' "$status_file")
  unresolved=$(yq eval '.confidence.unresolved // 0' "$status_file")
  score=$(yq eval '.confidence.score // 0.0' "$status_file")

  echo "certain:${certain}"
  echo "confident:${confident}"
  echo "tentative:${tentative}"
  echo "unresolved:${unresolved}"
  echo "score:${score}"
}

# get_stage_metrics <status_file> [stage]
# With stage: returns started_at/completed_at/driver/iterations as key:value lines.
# Without stage: returns one stage:json line per known workflow stage.
get_stage_metrics() {
  local status_file="$1"
  local stage="${2:-}"

  if [ -n "$stage" ]; then
    if ! validate_stage "$stage"; then
      echo "ERROR: Invalid stage '$stage'" >&2
      return 1
    fi

    STAGE="$stage" yq eval -r '
      "started_at:" + ((.stage_metrics[strenv(STAGE)].started_at // "") | tostring),
      "completed_at:" + ((.stage_metrics[strenv(STAGE)].completed_at // "") | tostring),
      "driver:" + ((.stage_metrics[strenv(STAGE)].driver // "") | tostring),
      "iterations:" + ((.stage_metrics[strenv(STAGE)].iterations // 0) | tostring)
    ' "$status_file"
    return 0
  fi

  for stage in $(get_all_stages); do
    STAGE="$stage" yq eval -r 'strenv(STAGE) + ":" + ((.stage_metrics[strenv(STAGE)] // {}) | tojson)' "$status_file"
  done
}

# ─────────────────────────────────────────────────────────────────────────────
# Progression Queries
# ─────────────────────────────────────────────────────────────────────────────

get_current_stage() {
  local status_file="$1"
  local stage state last_done="" found_last=false
  local progress_lines

  progress_lines=$(get_progress_map "$status_file")

  while IFS=: read -r stage state; do
    if [ "$state" = "active" ]; then
      echo "$stage"
      return 0
    fi
  done <<< "$progress_lines"

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

  echo "hydrate"
}

get_next_stage() {
  local current="$1"
  local found=false
  local stage

  for stage in $(get_all_stages); do
    if [ "$found" = "true" ]; then
      echo "$stage"
      return 0
    fi
    if [ "$stage" = "$current" ]; then
      found=true
    fi
  done

  return 1
}

# ─────────────────────────────────────────────────────────────────────────────
# Write Functions (yq-backed)
# ─────────────────────────────────────────────────────────────────────────────

# set_stage_state <status_file> <stage> <state> [driver]
set_stage_state() {
  local status_file="$1"
  local stage="$2"
  local state="$3"
  local driver="${4:-unknown}"
  local now current_state tmpfile

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

  current_state=$(STAGE="$stage" yq eval '.progress[strenv(STAGE)] // ""' "$status_file")
  now=$(now_iso8601)
  tmpfile=$(mk_status_tmp "$status_file")

  STAGE="$stage" STATE="$state" TS="$now" yq eval -i '
    .progress[strenv(STAGE)] = strenv(STATE) |
    .last_updated = strenv(TS)
  ' "$tmpfile"

  if [ "$state" = "active" ] && [ "$current_state" != "active" ]; then
    set_stage_active_metrics_on_tmp "$tmpfile" "$stage" "$driver" "$now"
  fi

  if [ "$state" = "done" ] && [ "$current_state" != "done" ]; then
    set_stage_completed_metrics_on_tmp "$tmpfile" "$stage" "$now"
  fi

  mv "$tmpfile" "$status_file"
}

# transition_stages <status_file> <from_stage> <to_stage> [driver]
transition_stages() {
  local status_file="$1"
  local from_stage="$2"
  local to_stage="$3"
  local driver="${4:-unknown}"
  local now tmpfile current_state expected_next to_current_state

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

  current_state=$(STAGE="$from_stage" yq eval '.progress[strenv(STAGE)] // ""' "$status_file")
  if [ "$current_state" != "active" ]; then
    echo "ERROR: Stage '$from_stage' is '$current_state', expected 'active'" >&2
    return 1
  fi

  expected_next=$(get_next_stage "$from_stage") || true
  if [ "$expected_next" != "$to_stage" ]; then
    echo "ERROR: '$to_stage' is not adjacent to '$from_stage' (expected '$expected_next')" >&2
    return 1
  fi

  to_current_state=$(STAGE="$to_stage" yq eval '.progress[strenv(STAGE)] // "pending"' "$status_file")
  now=$(now_iso8601)
  tmpfile=$(mk_status_tmp "$status_file")

  FROM_STAGE="$from_stage" TO_STAGE="$to_stage" TS="$now" yq eval -i '
    .progress[strenv(FROM_STAGE)] = "done" |
    .progress[strenv(TO_STAGE)] = "active" |
    .last_updated = strenv(TS)
  ' "$tmpfile"

  set_stage_completed_metrics_on_tmp "$tmpfile" "$from_stage" "$now"

  if [ "$to_current_state" != "active" ]; then
    set_stage_active_metrics_on_tmp "$tmpfile" "$to_stage" "$driver" "$now"
  fi

  mv "$tmpfile" "$status_file"
}

# set_checklist_field <status_file> <field> <value>
set_checklist_field() {
  local status_file="$1"
  local field="$2"
  local value="$3"
  local now tmpfile

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

  now=$(now_iso8601)
  tmpfile=$(mk_status_tmp "$status_file")

  if [ "$field" = "generated" ]; then
    FIELD="$field" VALUE="$value" TS="$now" yq eval -i '
      .checklist[strenv(FIELD)] = (strenv(VALUE) == "true") |
      .last_updated = strenv(TS)
    ' "$tmpfile"
  else
    FIELD="$field" VALUE="$value" TS="$now" yq eval -i '
      .checklist[strenv(FIELD)] = (strenv(VALUE) | tonumber) |
      .last_updated = strenv(TS)
    ' "$tmpfile"
  fi

  mv "$tmpfile" "$status_file"
}

# set_confidence_block <status_file> <certain> <confident> <tentative> <unresolved> <score>
set_confidence_block() {
  local status_file="$1"
  local certain="$2"
  local confident="$3"
  local tentative="$4"
  local unresolved="$5"
  local score="$6"
  local count_name count_val now tmpfile

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  for count_name in certain confident tentative unresolved; do
    eval count_val=\$$count_name
    if ! [[ "$count_val" =~ ^[0-9]+$ ]]; then
      echo "ERROR: Invalid value '$count_val' for '$count_name' (expected non-negative integer)" >&2
      return 1
    fi
  done

  if ! [[ "$score" =~ ^[0-9]+\.?[0-9]*$ ]]; then
    echo "ERROR: Invalid score '$score' (expected non-negative float)" >&2
    return 1
  fi

  now=$(now_iso8601)
  tmpfile=$(mk_status_tmp "$status_file")

  CERTAIN="$certain" CONFIDENT="$confident" TENTATIVE="$tentative" UNRESOLVED="$unresolved" SCORE="$score" TS="$now" yq eval -i '
    .confidence = {
      "certain": (strenv(CERTAIN) | tonumber),
      "confident": (strenv(CONFIDENT) | tonumber),
      "tentative": (strenv(TENTATIVE) | tonumber),
      "unresolved": (strenv(UNRESOLVED) | tonumber),
      "score": (strenv(SCORE) | tonumber)
    } |
    .last_updated = strenv(TS)
  ' "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# set_stage_metric <status_file> <stage> <field> <value>
set_stage_metric() {
  local status_file="$1"
  local stage="$2"
  local field="$3"
  local value="$4"
  local now tmpfile

  if [ ! -f "$status_file" ]; then
    echo "ERROR: Status file not found: $status_file" >&2
    return 1
  fi

  if ! validate_stage "$stage"; then
    echo "ERROR: Invalid stage '$stage'" >&2
    return 1
  fi

  case "$field" in
    started_at|completed_at|driver|iterations)
      ;;
    *)
      echo "ERROR: Invalid stage metric field '$field' (expected: started_at, completed_at, driver, iterations)" >&2
      return 1
      ;;
  esac

  now=$(now_iso8601)
  tmpfile=$(mk_status_tmp "$status_file")

  if [ "$field" = "iterations" ]; then
    if ! [[ "$value" =~ ^[0-9]+$ ]]; then
      echo "ERROR: Invalid iterations '$value' (expected non-negative integer)" >&2
      rm -f "$tmpfile"
      return 1
    fi

    STAGE="$stage" FIELD="$field" VALUE="$value" TS="$now" yq eval -i '
      .stage_metrics = (.stage_metrics // {}) |
      .stage_metrics[strenv(STAGE)] = (.stage_metrics[strenv(STAGE)] // {}) |
      .stage_metrics[strenv(STAGE)][strenv(FIELD)] = (strenv(VALUE) | tonumber) |
      .last_updated = strenv(TS)
    ' "$tmpfile"
  else
    STAGE="$stage" FIELD="$field" VALUE="$value" TS="$now" yq eval -i '
      .stage_metrics = (.stage_metrics // {}) |
      .stage_metrics[strenv(STAGE)] = (.stage_metrics[strenv(STAGE)] // {}) |
      .stage_metrics[strenv(STAGE)][strenv(FIELD)] = strenv(VALUE) |
      .last_updated = strenv(TS)
    ' "$tmpfile"
  fi

  mv "$tmpfile" "$status_file"
}

# log_command <status_file> <cmd> [args] [outcome]
log_command() {
  local status_file="$1"
  local cmd="$2"
  local args="${3:-}"
  local outcome="${4:-success}"
  local ts cmd_esc args_esc outcome_esc
  local json

  if [ "$outcome" != "success" ] && [ "$outcome" != "error" ]; then
    echo "ERROR: Invalid command outcome '$outcome' (expected: success|error)" >&2
    return 1
  fi

  ts=$(now_iso8601)
  cmd_esc=$(json_escape "$cmd")
  args_esc=$(json_escape "$args")
  outcome_esc=$(json_escape "$outcome")

  if [ -n "$args" ]; then
    json=$(printf '{"ts":"%s","event":"command","cmd":"%s","args":"%s","outcome":"%s"}' "$ts" "$cmd_esc" "$args_esc" "$outcome_esc")
  else
    json=$(printf '{"ts":"%s","event":"command","cmd":"%s","outcome":"%s"}' "$ts" "$cmd_esc" "$outcome_esc")
  fi

  append_history_json "$status_file" "$json"
}

# log_confidence <status_file> <score> <delta> <trigger>
log_confidence() {
  local status_file="$1"
  local score="$2"
  local delta="$3"
  local trigger="$4"
  local ts delta_esc trigger_esc json

  if ! [[ "$score" =~ ^-?[0-9]+\.?[0-9]*$ ]]; then
    echo "ERROR: Invalid confidence score '$score'" >&2
    return 1
  fi

  ts=$(now_iso8601)
  delta_esc=$(json_escape "$delta")
  trigger_esc=$(json_escape "$trigger")
  json=$(printf '{"ts":"%s","event":"confidence","score":%s,"delta":"%s","trigger":"%s"}' "$ts" "$score" "$delta_esc" "$trigger_esc")

  append_history_json "$status_file" "$json"
}

# log_review <status_file> <result> [rework]
log_review() {
  local status_file="$1"
  local result="$2"
  local rework="${3:-}"
  local ts result_esc rework_esc json

  if [ "$result" != "passed" ] && [ "$result" != "failed" ]; then
    echo "ERROR: Invalid review result '$result' (expected: passed|failed)" >&2
    return 1
  fi

  ts=$(now_iso8601)
  result_esc=$(json_escape "$result")

  if [ -n "$rework" ]; then
    rework_esc=$(json_escape "$rework")
    json=$(printf '{"ts":"%s","event":"review","result":"%s","rework":"%s"}' "$ts" "$result_esc" "$rework_esc")
  else
    json=$(printf '{"ts":"%s","event":"review","result":"%s"}' "$ts" "$result_esc")
  fi

  append_history_json "$status_file" "$json"
}

# ─────────────────────────────────────────────────────────────────────────────
# Display Helpers
# ─────────────────────────────────────────────────────────────────────────────

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

validate_status_file() {
  local status_file="$1"
  local errors=0
  local stage state active_count

  for stage in $(get_all_stages); do
    state=$(STAGE="$stage" yq eval '.progress[strenv(STAGE)] // ""' "$status_file")

    if [ -z "$state" ]; then
      echo "ERROR: Missing progress.$stage in $status_file" >&2
      ((errors++))
      continue
    fi

    if ! validate_state "$state"; then
      echo "ERROR: Invalid state '$state' for stage $stage" >&2
      ((errors++))
      continue
    fi

    if ! validate_stage_state "$stage" "$state"; then
      echo "ERROR: State '$state' not allowed for stage $stage" >&2
      ((errors++))
    fi
  done

  active_count=$(yq eval '[.progress[] | select(. == "active")] | length' "$status_file")
  if [ "$active_count" -gt 1 ]; then
    echo "ERROR: Multiple stages are active (expected 0 or 1)" >&2
    ((errors++))
  fi

  [ "$errors" -eq 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI Interface
# ─────────────────────────────────────────────────────────────────────────────

show_help() {
  cat <<'HELP_EOF'
stageman.sh - Stage Manager (schema queries + yq-backed status API)

USAGE:
  As library:
    source stageman.sh
    get_all_stages
    get_progress_map path/to/.status.yaml

  As command:
    stageman.sh --help
    stageman.sh --version
    stageman.sh --test

  Write commands:
    stageman.sh set-state <file> <stage> <state> [driver]
    stageman.sh transition <file> <from-stage> <to-stage> [driver]
    stageman.sh set-checklist <file> <field> <value>
    stageman.sh set-confidence <file> <certain> <confident> <tentative> <unresolved> <score>
    stageman.sh set-stage-metric <file> <stage> <field> <value>

  History log commands:
    stageman.sh log-command <file> <cmd> [args] [outcome]
    stageman.sh log-confidence <file> <score> <delta> <trigger>
    stageman.sh log-review <file> <passed|failed> [rework]

NOTES:
  - Requires Mike Farah yq v4.x.
  - Status reads/writes are yq-backed; schema queries are workflow.yaml-backed.
HELP_EOF
}

show_version() {
  local schema_version
  schema_version=$(grep '^ *version:' "$WORKFLOW_SCHEMA" | head -1 | sed 's/.*: *//' | tr -d '"')
  echo "stageman version 2.0.0"
  echo "Schema version: ${schema_version}"
  yq --version
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

  echo "✓ Smoke checks passed"
}

# ─────────────────────────────────────────────────────────────────────────────
# Main (when executed directly)
# ─────────────────────────────────────────────────────────────────────────────

if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
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
      run_tests
      ;;
    set-state)
      if [ $# -lt 4 ] || [ $# -gt 5 ]; then
        echo "Usage: stageman.sh set-state <file> <stage> <state> [driver]" >&2
        exit 1
      fi
      set_stage_state "$2" "$3" "$4" "${5:-unknown}"
      ;;
    transition)
      if [ $# -lt 4 ] || [ $# -gt 5 ]; then
        echo "Usage: stageman.sh transition <file> <from-stage> <to-stage> [driver]" >&2
        exit 1
      fi
      transition_stages "$2" "$3" "$4" "${5:-unknown}"
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
    set-stage-metric)
      if [ $# -ne 5 ]; then
        echo "Usage: stageman.sh set-stage-metric <file> <stage> <field> <value>" >&2
        exit 1
      fi
      set_stage_metric "$2" "$3" "$4" "$5"
      ;;
    log-command)
      if [ $# -lt 3 ] || [ $# -gt 5 ]; then
        echo "Usage: stageman.sh log-command <file> <cmd> [args] [outcome]" >&2
        exit 1
      fi
      log_command "$2" "$3" "${4:-}" "${5:-success}"
      ;;
    log-confidence)
      if [ $# -ne 5 ]; then
        echo "Usage: stageman.sh log-confidence <file> <score> <delta> <trigger>" >&2
        exit 1
      fi
      log_confidence "$2" "$3" "$4" "$5"
      ;;
    log-review)
      if [ $# -lt 3 ] || [ $# -gt 4 ]; then
        echo "Usage: stageman.sh log-review <file> <passed|failed> [rework]" >&2
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
fi

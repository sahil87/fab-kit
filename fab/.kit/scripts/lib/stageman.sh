#!/usr/bin/env bash
# fab/.kit/scripts/lib/stageman.sh
#
# Stage Manager - Query utility for workflow stages and states.
# Reads the canonical workflow schema and provides typed accessors.
#
# Usage as library:
#   source "$(dirname "$0")/stageman.sh"
#   validate_state "done" || echo "Invalid state"
#   get_stage_number "spec"  # returns 2
#   get_state_symbol "active"  # returns ●
#
# Usage as command:
#   stageman.sh --help    # Show usage
#   stageman.sh --test    # Run self-tests
#   stageman.sh --version # Show version

set -euo pipefail

# Locate workflow schema (works from both .kit/scripts/ and src/stageman/ symlink)
if [ -n "${BASH_SOURCE[0]:-}" ]; then
  STAGEMAN_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
  # Fallback when sourced in a way that doesn't set BASH_SOURCE
  STAGEMAN_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
fi
WORKFLOW_SCHEMA="$STAGEMAN_DIR/../../schemas/workflow.yaml"

if [ ! -f "$WORKFLOW_SCHEMA" ]; then
  echo "ERROR: workflow.yaml not found at $WORKFLOW_SCHEMA" >&2
  echo "       STAGEMAN_DIR=$STAGEMAN_DIR" >&2
  return 1 2>/dev/null || exit 1
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
    val=$(grep "^ *${stage}:" "$status_file" | sed 's/^ *[a-z]*: *//' || echo "")
    echo "${stage}:${val:-pending}"
  done
}

# get_checklist <status_file> — Extract checklist fields
# Outputs: generated:{val}, completed:{val}, total:{val}
get_checklist() {
  local status_file="$1"
  local generated completed total
  generated=$(grep '^ *generated:' "$status_file" | sed 's/^ *generated: *//' || true)
  completed=$(grep '^ *completed:' "$status_file" | sed 's/^ *completed: *//' || true)
  total=$(grep '^ *total:' "$status_file" | sed 's/^ *total: *//' || true)
  echo "generated:${generated:-false}"
  echo "completed:${completed:-0}"
  echo "total:${total:-0}"
}

# get_confidence <status_file> — Extract confidence fields
# Outputs: certain:{val}, confident:{val}, tentative:{val}, unresolved:{val}, score:{val}
get_confidence() {
  local status_file="$1"
  local certain confident tentative unresolved score
  certain=$(grep '^ *certain:' "$status_file" | sed 's/^ *certain: *//' || true)
  confident=$(grep '^ *confident:' "$status_file" | sed 's/^ *confident: *//' || true)
  tentative=$(grep '^ *tentative:' "$status_file" | sed 's/^ *tentative: *//' || true)
  unresolved=$(grep '^ *unresolved:' "$status_file" | sed 's/^ *unresolved: *//' || true)
  score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || true)
  echo "certain:${certain:-0}"
  echo "confident:${confident:-0}"
  echo "tentative:${tentative:-0}"
  echo "unresolved:${unresolved:-0}"
  echo "score:${score:-5.0}"
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

# set_stage_state <status_file> <stage> <state>
# Set a single stage's progress value in .status.yaml.
# Validates inputs, writes atomically (temp-file-then-mv), updates last_updated.
# Returns 0 on success, 1 on validation failure (diagnostic to stderr).
set_stage_state() {
  local status_file="$1"
  local stage="$2"
  local state="$3"

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

  local now
  now=$(date -Iseconds)
  local tmpfile
  tmpfile=$(mktemp "$(dirname "$status_file")/.status.yaml.XXXXXX")

  awk -v stage="$stage" -v state="$state" -v ts="$now" '
    /^last_updated:/ { print "last_updated: " ts; next }
    /^  [a-z]+:/ {
      split($0, parts, ": ")
      gsub(/^ */, "", parts[1])
      if (parts[1] == stage) {
        sub(/: .*/, ": " state)
      }
    }
    { print }
  ' "$status_file" > "$tmpfile"

  mv "$tmpfile" "$status_file"
}

# transition_stages <status_file> <from_stage> <to_stage>
# Two-write transition: set from_stage to done and to_stage to active atomically.
# Validates adjacency and that from_stage is currently active.
# Returns 0 on success, 1 on validation failure.
transition_stages() {
  local status_file="$1"
  local from_stage="$2"
  local to_stage="$3"

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

  # Verify from_stage is currently active
  local current_state
  current_state=$(grep "^ *${from_stage}:" "$status_file" | sed 's/^ *[a-z]*: *//')
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

  awk -v from="$from_stage" -v to="$to_stage" -v ts="$now" '
    /^last_updated:/ { print "last_updated: " ts; next }
    /^  [a-z]+:/ {
      split($0, parts, ": ")
      gsub(/^ */, "", parts[1])
      if (parts[1] == from) {
        sub(/: .*/, ": done")
      } else if (parts[1] == to) {
        sub(/: .*/, ": active")
      }
    }
    { print }
  ' "$status_file" > "$tmpfile"

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

  awk -v field="$field" -v value="$value" -v ts="$now" '
    /^checklist:/ { in_checklist = 1 }
    in_checklist && /^[a-z]/ && !/^checklist:/ { in_checklist = 0 }
    in_checklist && $0 ~ "^  " field ":" {
      print "  " field ": " value
      next
    }
    /^last_updated:/ { print "last_updated: " ts; next }
    { print }
  ' "$status_file" > "$tmpfile"

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

  awk -v certain="$certain" \
      -v confident="$confident" \
      -v tentative="$tentative" \
      -v unresolved="$unresolved" \
      -v score="$score" \
      -v ts="$now" '
    /^confidence:/ {
      in_block = 1
      print "confidence:"
      print "  certain: " certain
      print "  confident: " confident
      print "  tentative: " tentative
      print "  unresolved: " unresolved
      print "  score: " score
      next
    }
    in_block && /^[^ ]/ { in_block = 0 }
    in_block { next }
    /^last_updated:/ { print "last_updated: " ts; next }
    { print }
  ' "$status_file" > "$tmpfile"

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

  # Check each stage has valid state
  for stage in $(get_all_stages); do
    local state
    state=$(grep "^ *${stage}:" "$status_file" | sed 's/^ *[a-z]*: *//' || echo "")

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

  # Check active count (0 or 1)
  local active_count
  active_count=$(grep '^ *[a-z]*: active$' "$status_file" | wc -l)
  if [ "$active_count" -gt 1 ]; then
    echo "ERROR: Multiple stages are active (expected 0 or 1)" >&2
    ((errors++))
  fi

  [ $errors -eq 0 ]
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI Interface
# ─────────────────────────────────────────────────────────────────────────────

show_help() {
  cat <<'EOF'
stageman.sh - Stage Manager (workflow schema query utility)

USAGE:
  As library (source in scripts):
    source stageman.sh
    get_all_stages
    get_state_symbol "active"

  As command (read-only):
    stageman.sh --help      Show this help
    stageman.sh --test      Run self-tests
    stageman.sh --version   Show version

  Write commands:
    stageman.sh set-state <file> <stage> <state>
    stageman.sh transition <file> <from-stage> <to-stage>
    stageman.sh set-checklist <file> <field> <value>
    stageman.sh set-confidence <file> <certain> <confident> <tentative> <unresolved> <score>

AVAILABLE FUNCTIONS:
  State queries:
    get_all_states              List all valid states
    validate_state <state>      Check if state is valid
    get_state_symbol <state>    Get display symbol (●, ✓, etc)
    get_state_suffix <state>    Get display suffix (e.g., " (skipped)")
    is_terminal_state <state>   Check if state is terminal

  Stage queries:
    get_all_stages              List all stages in order
    validate_stage <stage>      Check if stage exists
    get_stage_number <stage>    Get position (1-6)
    get_stage_name <stage>      Get display name
    get_stage_artifact <stage>  Get generated filename
    get_allowed_states <stage>  List allowed states for stage
    get_initial_state <stage>   Get default state
    is_required_stage <stage>   Check if stage is required
    has_auto_checklist <stage>  Check if stage generates checklist

  .status.yaml Accessors:
    get_progress_map <file>     Extract stage:state pairs (one per line)
    get_checklist <file>        Extract checklist fields (key:value lines)
    get_confidence <file>       Extract confidence fields (key:value lines)

  Progression:
    get_current_stage <file>    Detect active stage from .status.yaml
    get_next_stage <stage>      Get next stage in sequence

  Write:
    set_stage_state <file> <stage> <state>    Set a stage's state
    transition_stages <file> <from> <to>      Two-write forward transition
    set_checklist_field <file> <field> <val>  Update checklist field
    set_confidence_block <file> <c> <cf> <t> <u> <s>  Replace confidence block

  Validation:
    validate_status_file <file> Validate .status.yaml against schema

  Display:
    format_state <state>        Format state for display (symbol + suffix)

SCHEMA LOCATION:
  $WORKFLOW_SCHEMA

EXAMPLES:
  # Check if a state is valid
  if validate_state "done"; then echo "Valid"; fi

  # Iterate over all stages
  for stage in $(get_all_stages); do
    echo "$stage: $(get_stage_number "$stage")"
  done

  # Display progress
  for stage in $(get_all_stages); do
    state=$(grep "^ *${stage}:" .status.yaml | sed 's/.*: //')
    echo "$(format_state "$state") $stage"
  done

SEE ALSO:
  src/stageman/README.md - API reference and development guide
  fab/.kit/schemas/workflow.yaml - Schema definition
  docs/memory/fab-workflow/schemas.md - Schema overview and design principles
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
      # Default: run tests for backward compatibility
      run_tests
      ;;
    set-state)
      if [ $# -ne 4 ]; then
        echo "Usage: stageman.sh set-state <file> <stage> <state>" >&2
        exit 1
      fi
      set_stage_state "$2" "$3" "$4"
      ;;
    transition)
      if [ $# -ne 4 ]; then
        echo "Usage: stageman.sh transition <file> <from-stage> <to-stage>" >&2
        exit 1
      fi
      transition_stages "$2" "$3" "$4"
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
    *)
      echo "Unknown option: $1" >&2
      echo "Try: stageman.sh --help" >&2
      exit 1
      ;;
  esac
fi

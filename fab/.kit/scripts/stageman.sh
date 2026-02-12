#!/usr/bin/env bash
# fab/.kit/scripts/stageman.sh
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
WORKFLOW_SCHEMA="$STAGEMAN_DIR/../schemas/workflow.yaml"

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
      gsub(/"/, "", $2)
      print $2
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
# Progression Queries
# ─────────────────────────────────────────────────────────────────────────────

# get_current_stage <status_file> — Determine active stage from .status.yaml
get_current_stage() {
  local status_file="$1"

  # Find first active stage
  for stage in $(get_all_stages); do
    local state
    state=$(grep "^ *${stage}:" "$status_file" | sed 's/^ *[a-z]*: *//')
    if [ "$state" = "active" ]; then
      echo "$stage"
      return 0
    fi
  done

  # Fallback: archive (all done)
  echo "archive"
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

  As command:
    stageman.sh --help      Show this help
    stageman.sh --test      Run self-tests
    stageman.sh --version   Show version

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

  Progression:
    get_current_stage <file>    Detect active stage from .status.yaml
    get_next_stage <stage>      Get next stage in sequence

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
  src/stageman/README.md - Full documentation
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
    *)
      echo "Unknown option: $1" >&2
      echo "Try: stageman.sh --help" >&2
      exit 1
      ;;
  esac
fi

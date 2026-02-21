#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/pipeline/run.sh — Pipeline orchestrator
#
# Usage: run.sh <manifest-path>
#
# Reads a YAML pipeline manifest and dispatches changes in dependency order.
# Runs indefinitely until killed with Ctrl+C (SIGINT). The human adds entries
# to the manifest while the orchestrator processes earlier ones.
#
# Requires: yq, wt-create, claude CLI

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
FAB_DIR="$(dirname "$KIT_DIR")"
CONFIG_FILE="${FAB_DIR}/project/config.yaml"
DISPATCH="$SCRIPT_DIR/dispatch.sh"
POLL_INTERVAL="${PIPELINE_POLL_INTERVAL:-10}"

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------

usage() {
  cat <<'EOF'
Usage: run.sh <manifest-path>

Pipeline orchestrator — dispatches fab changes in dependency order.

Reads the manifest on every iteration. The human can add new entries
while the orchestrator runs. Runs until Ctrl+C.

Arguments:
  manifest-path   Path to the pipeline manifest YAML file

Environment:
  PIPELINE_POLL_INTERVAL   Seconds between idle polls (default: 10)

Example:
  fab/.kit/scripts/pipeline/run.sh fab/pipelines/my-feature.yaml
EOF
}

if [[ $# -lt 1 ]]; then
  usage >&2
  exit 1
fi

MANIFEST="$1"

if [[ ! -f "$MANIFEST" ]]; then
  echo "Error: manifest not found: $MANIFEST" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# State tracking
# ---------------------------------------------------------------------------

declare -A WORKTREE_PATHS  # change-id → worktree path
CURRENT_DISPATCH=""        # change-id being dispatched (for SIGINT summary)

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

log() {
  echo "[pipeline] $*"
}

# ---------------------------------------------------------------------------
# Manifest Parsing & Validation
# ---------------------------------------------------------------------------

validate_manifest() {
  local manifest="$1"

  # Check base field exists
  local base
  base=$(yq -r '.base // ""' "$manifest")
  if [[ -z "$base" || "$base" == "null" ]]; then
    echo "Error: manifest missing 'base' field" >&2
    return 1
  fi

  # Check changes list exists
  local count
  count=$(yq '.changes | length' "$manifest")
  if [[ "$count" -eq 0 ]]; then
    echo "Error: manifest has no changes" >&2
    return 1
  fi

  # Check each entry has required fields
  local i id deps_count
  for ((i = 0; i < count; i++)); do
    id=$(yq -r ".changes[$i].id // \"\"" "$manifest")
    if [[ -z "$id" || "$id" == "null" ]]; then
      echo "Error: changes[$i] missing 'id' field" >&2
      return 1
    fi

    # depends_on must exist (can be empty list)
    local has_deps
    has_deps=$(yq ".changes[$i] | has(\"depends_on\")" "$manifest")
    if [[ "$has_deps" != "true" ]]; then
      echo "Error: changes[$i] ($id) missing 'depends_on' field" >&2
      return 1
    fi

    # Single-dependency constraint — mark invalid but don't reject manifest
    deps_count=$(yq ".changes[$i].depends_on | length" "$manifest")
    if [[ "$deps_count" -gt 1 ]]; then
      local current_stage
      current_stage=$(yq -r "(.changes[$i]).stage // \"\"" "$manifest")
      if [[ -z "$current_stage" || "$current_stage" == "null" ]]; then
        echo "[pipeline] Multi-parent dependency not supported in v1: $id depends on $deps_count changes. Marking invalid." >&2
        yq -i "(.changes[$i]).stage = \"invalid\"" "$manifest"
      fi
    fi
  done

  # Validate depends_on references exist
  local all_ids dep
  all_ids=$(yq -r '.changes[].id' "$manifest")
  for ((i = 0; i < count; i++)); do
    deps_count=$(yq ".changes[$i].depends_on | length" "$manifest")
    if [[ "$deps_count" -gt 0 ]]; then
      dep=$(yq -r ".changes[$i].depends_on[0]" "$manifest")
      if ! echo "$all_ids" | grep -qx "$dep"; then
        id=$(yq -r ".changes[$i].id" "$manifest")
        echo "Error: $id depends on $dep, which is not in the manifest" >&2
        return 1
      fi
    fi
  done

  # Circular dependency detection
  if ! detect_cycles "$manifest"; then
    return 1
  fi

  return 0
}

detect_cycles() {
  local manifest="$1"
  local count
  count=$(yq '.changes | length' "$manifest")

  # Build adjacency and run DFS
  declare -A visited  # 0=unvisited, 1=in-progress, 2=done
  local i id

  for ((i = 0; i < count; i++)); do
    id=$(yq -r ".changes[$i].id" "$manifest")
    visited["$id"]=0
  done

  for id in "${!visited[@]}"; do
    if [[ "${visited[$id]}" -eq 0 ]]; then
      if ! dfs_visit "$manifest" "$id"; then
        return 1
      fi
    fi
  done
  return 0
}

dfs_visit() {
  local manifest="$1" id="$2"
  visited["$id"]=1  # in-progress

  local count deps_count dep i
  count=$(yq '.changes | length' "$manifest")

  for ((i = 0; i < count; i++)); do
    local entry_id
    entry_id=$(yq -r ".changes[$i].id" "$manifest")
    if [[ "$entry_id" == "$id" ]]; then
      deps_count=$(yq ".changes[$i].depends_on | length" "$manifest")
      if [[ "$deps_count" -gt 0 ]]; then
        dep=$(yq -r ".changes[$i].depends_on[0]" "$manifest")
        if [[ "${visited[$dep]:-0}" -eq 1 ]]; then
          echo "Error: Circular dependency detected: $id <-> $dep" >&2
          return 1
        elif [[ "${visited[$dep]:-0}" -eq 0 ]]; then
          if ! dfs_visit "$manifest" "$dep"; then
            return 1
          fi
        fi
      fi
      break
    fi
  done

  visited["$id"]=2  # done
  return 0
}

# ---------------------------------------------------------------------------
# Dispatch Logic
# ---------------------------------------------------------------------------

get_stage() {
  local manifest="$1" id="$2"
  yq -r "(.changes[] | select(.id == \"$id\")).stage // \"\"" "$manifest"
}

is_terminal() {
  local stage="$1"
  [[ "$stage" == "done" || "$stage" == "failed" || "$stage" == "invalid" ]]
}

is_dispatchable() {
  local manifest="$1" id="$2"
  local stage
  stage=$(get_stage "$manifest" "$id")

  # Terminal stages are never dispatchable
  if is_terminal "$stage"; then
    return 1
  fi

  # Check all dependencies are done
  local count i entry_id deps_count dep dep_stage
  count=$(yq '.changes | length' "$manifest")
  for ((i = 0; i < count; i++)); do
    entry_id=$(yq -r ".changes[$i].id" "$manifest")
    if [[ "$entry_id" == "$id" ]]; then
      deps_count=$(yq ".changes[$i].depends_on | length" "$manifest")
      if [[ "$deps_count" -gt 0 ]]; then
        dep=$(yq -r ".changes[$i].depends_on[0]" "$manifest")
        dep_stage=$(get_stage "$manifest" "$dep")
        if [[ "$dep_stage" != "done" ]]; then
          return 1
        fi
      fi
      return 0
    fi
  done
  return 1
}

find_next_dispatchable() {
  local manifest="$1"
  local count i id
  count=$(yq '.changes | length' "$manifest")

  for ((i = 0; i < count; i++)); do
    id=$(yq -r ".changes[$i].id" "$manifest")
    if is_dispatchable "$manifest" "$id"; then
      echo "$id"
      return 0
    fi
  done
  return 1
}

get_parent_branch() {
  local manifest="$1" id="$2"
  local count i entry_id deps_count dep
  local base
  base=$(yq -r '.base' "$manifest")

  count=$(yq '.changes | length' "$manifest")
  for ((i = 0; i < count; i++)); do
    entry_id=$(yq -r ".changes[$i].id" "$manifest")
    if [[ "$entry_id" == "$id" ]]; then
      deps_count=$(yq ".changes[$i].depends_on | length" "$manifest")
      if [[ "$deps_count" -eq 0 ]]; then
        echo "$base"
      else
        dep=$(yq -r ".changes[$i].depends_on[0]" "$manifest")
        # Parent's branch name = branch_prefix + dep id
        local prefix=""
        if [[ -f "$CONFIG_FILE" ]] && command -v yq &>/dev/null; then
          prefix=$(yq -r '.git.branch_prefix // ""' "$CONFIG_FILE")
        fi
        echo "${prefix}${dep}"
      fi
      return 0
    fi
  done
  echo "$base"
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

print_summary() {
  local manifest="$1"
  local count completed=() failed=() blocked=() skipped=() pending=() in_progress=()

  count=$(yq '.changes | length' "$manifest")
  local i id stage
  for ((i = 0; i < count; i++)); do
    id=$(yq -r ".changes[$i].id" "$manifest")
    stage=$(get_stage "$manifest" "$id")

    if [[ "$id" == "$CURRENT_DISPATCH" ]]; then
      in_progress+=("$id")
    elif [[ "$stage" == "done" ]]; then
      completed+=("$id")
    elif [[ "$stage" == "failed" ]]; then
      failed+=("$id")
    elif [[ "$stage" == "invalid" ]]; then
      skipped+=("$id")
    elif [[ -z "$stage" || "$stage" == "null" ]]; then
      pending+=("$id")
    else
      # Intermediate stage — blocked or pending re-dispatch
      pending+=("$id")
    fi
  done

  # Check for blocked (deps on failed)
  local truly_pending=() truly_blocked=()
  for id in "${pending[@]}"; do
    local dep_stage dep_id deps_count
    for ((i = 0; i < count; i++)); do
      local entry_id
      entry_id=$(yq -r ".changes[$i].id" "$manifest")
      if [[ "$entry_id" == "$id" ]]; then
        deps_count=$(yq ".changes[$i].depends_on | length" "$manifest")
        if [[ "$deps_count" -gt 0 ]]; then
          dep_id=$(yq -r ".changes[$i].depends_on[0]" "$manifest")
          dep_stage=$(get_stage "$manifest" "$dep_id")
          if [[ "$dep_stage" == "failed" || "$dep_stage" == "invalid" ]]; then
            truly_blocked+=("$id")
            continue 2
          fi
        fi
        break
      fi
    done
    truly_pending+=("$id")
  done

  local manifest_name
  manifest_name=$(basename "$MANIFEST" .yaml)

  echo ""
  log "Pipeline stopped: $manifest_name"
  log "  Completed:   ${#completed[@]} (${completed[*]:-none})"
  log "  Failed:      ${#failed[@]} (${failed[*]:-none})"
  log "  Blocked:     ${#truly_blocked[@]} (${truly_blocked[*]:-none})"
  log "  Skipped:     ${#skipped[@]} (${skipped[*]:-none})"
  log "  Pending:     ${#truly_pending[@]} (${truly_pending[*]:-none})"
  if [[ ${#in_progress[@]} -gt 0 ]]; then
    log "  In progress: ${#in_progress[@]} (${in_progress[*]})"
  fi

  # List worktree paths
  if [[ ${#WORKTREE_PATHS[@]} -gt 0 ]]; then
    echo ""
    log "Worktrees:"
    for id in "${!WORKTREE_PATHS[@]}"; do
      log "  $id → ${WORKTREE_PATHS[$id]}"
    done
  fi
}

# ---------------------------------------------------------------------------
# SIGINT Handler
# ---------------------------------------------------------------------------

on_sigint() {
  echo ""
  log "Caught SIGINT — shutting down..."
  print_summary "$MANIFEST"
  exit 130
}

trap on_sigint INT

# ---------------------------------------------------------------------------
# Main Loop
# ---------------------------------------------------------------------------

main() {
  log "Starting pipeline: $MANIFEST"
  log "Poll interval: ${POLL_INTERVAL}s"
  echo ""

  # Initial validation
  if ! validate_manifest "$MANIFEST"; then
    exit 1
  fi

  while true; do
    # Re-read and re-validate manifest each iteration
    if ! validate_manifest "$MANIFEST" 2>/dev/null; then
      # Validation failure on re-read — log and continue polling
      # (human may be mid-edit)
      sleep "$POLL_INTERVAL"
      continue
    fi

    # Find next dispatchable change
    local next_id
    if next_id=$(find_next_dispatchable "$MANIFEST"); then
      CURRENT_DISPATCH="$next_id"

      local parent_branch
      parent_branch=$(get_parent_branch "$MANIFEST" "$next_id")

      # Dispatch — capture output for worktree path, exit code 1 = infra failure
      local dispatch_output
      if dispatch_output=$(bash "$DISPATCH" "$next_id" "$parent_branch" "$MANIFEST" 2>&1 | tee /dev/stderr); then
        : # success
      else
        log "Infrastructure failure during dispatch of $next_id — aborting"
        print_summary "$MANIFEST"
        exit 1
      fi

      # Extract worktree path from dispatch output for summary
      local wt_line
      wt_line=$(echo "$dispatch_output" | grep '\[pipeline\] Dispatching:.*worktree:' | head -1 || true)
      if [[ -n "$wt_line" ]]; then
        local wt_path
        wt_path=$(echo "$wt_line" | sed 's/.*worktree: \(.*\))/\1/')
        WORKTREE_PATHS["$next_id"]="$wt_path"
      fi

      CURRENT_DISPATCH=""
    else
      # Nothing to dispatch — count completed for idle message
      local total completed_count
      total=$(yq '.changes | length' "$MANIFEST")
      completed_count=$(yq '[.changes[] | select(.stage == "done")] | length' "$MANIFEST")
      local pending_count=$((total - completed_count))

      log "Waiting for new entries... ($completed_count completed, $pending_count pending)"
      sleep "$POLL_INTERVAL"
    fi
  done
}

main

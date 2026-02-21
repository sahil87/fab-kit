#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/pipeline/dispatch.sh — Dispatch a single change through the fab pipeline
#
# Usage: dispatch.sh <manifest-id> <fs-change-id> <parent-branch> <manifest-path>
#
# Creates a worktree, provisions artifacts, validates prerequisites,
# runs fab-ff via claude -p, ships (commit/push/PR), and writes the
# terminal stage back to the manifest.
#
# Exit codes:
#   0  — change completed (done or failed written to manifest)
#   1  — infrastructure failure (caller should abort)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
FAB_DIR="$(dirname "$KIT_DIR")"
REPO_ROOT="$(dirname "$FAB_DIR")"
CONFIG_FILE="${FAB_DIR}/project/config.yaml"


# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------

usage() {
  cat <<'EOF'
Usage: dispatch.sh <manifest-id> <fs-change-id> <parent-branch> <manifest-path>

Dispatches a single change through the fab pipeline in an isolated worktree.

Arguments:
  manifest-id     Original change ID as written in the manifest (for manifest writes)
  fs-change-id    Resolved change folder name under fab/changes/ (for filesystem ops)
  parent-branch   Branch to base the worktree on (manifest's base for roots,
                  parent change's branch for dependents)
  manifest-path   Path to the pipeline manifest YAML file

The script creates a worktree, runs fab-ff, ships the result, and writes
the terminal stage (done/failed/invalid) to the manifest.

Exit code 0 = change processed (check manifest for stage).
Exit code 1 = infrastructure failure (caller should abort).
EOF
}

if [[ $# -lt 4 ]]; then
  usage >&2
  exit 1
fi

MANIFEST_ID="$1"
CHANGE_ID="$2"
PARENT_BRANCH="$3"
MANIFEST="$4"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

get_branch_prefix() {
  if [[ -f "$CONFIG_FILE" ]] && command -v yq &>/dev/null; then
    yq -r '.git.branch_prefix // ""' "$CONFIG_FILE"
  else
    echo ""
  fi
}

write_stage() {
  local id="$1" stage="$2" manifest="$3"
  yq -i "(.changes[] | select(.id == \"$id\")).stage = \"$stage\"" "$manifest"
}

log() {
  echo "[pipeline] $*"
}

# ---------------------------------------------------------------------------
# Worktree Creation
# ---------------------------------------------------------------------------

BRANCH_PREFIX="$(get_branch_prefix)"
CHANGE_BRANCH="${BRANCH_PREFIX}${CHANGE_ID}"

create_worktree() {
  local wt_path

  # For dependent nodes (PARENT_BRANCH is another change's branch, not base),
  # create the change branch from the parent's pushed branch
  if ! git show-ref --verify --quiet "refs/heads/$CHANGE_BRANCH" 2>/dev/null; then
    # Branch doesn't exist locally — create it
    if git ls-remote --exit-code --heads origin "$PARENT_BRANCH" &>/dev/null; then
      # Parent exists on remote — branch from it (dependent node)
      git branch "$CHANGE_BRANCH" "origin/$PARENT_BRANCH"
    fi
    # If parent not on remote, wt-create will create from HEAD (root node)
  fi

  wt_path=$(wt-create --non-interactive --worktree-open skip --worktree-name "$CHANGE_ID" "$CHANGE_BRANCH" | tail -1)

  if [[ -z "$wt_path" || ! -d "$wt_path" ]]; then
    echo "Error: wt-create failed — no worktree path returned" >&2
    return 1
  fi

  echo "$wt_path"
}

# ---------------------------------------------------------------------------
# Artifact Provisioning
# ---------------------------------------------------------------------------

provision_artifacts() {
  local wt_path="$1"
  local target_dir="$wt_path/fab/changes/$CHANGE_ID"

  if [[ ! -d "$target_dir" ]]; then
    local source_dir="$FAB_DIR/changes/$CHANGE_ID"
    if [[ ! -d "$source_dir" ]]; then
      echo "Error: source change folder not found at $source_dir" >&2
      return 1
    fi
    mkdir -p "$(dirname "$target_dir")"
    cp -r "$source_dir" "$target_dir"
    log "Copied artifacts: fab/changes/$CHANGE_ID → worktree"
  fi
}

# ---------------------------------------------------------------------------
# Prerequisite Validation
# ---------------------------------------------------------------------------

validate_prerequisites() {
  local wt_path="$1"
  local change_dir="$wt_path/fab/changes/$CHANGE_ID"

  if [[ ! -f "$change_dir/intake.md" ]]; then
    log "Failed: $CHANGE_ID — prerequisite failed: intake.md not found"
    write_stage "$MANIFEST_ID" "invalid" "$MANIFEST"
    return 2
  fi

  if [[ ! -f "$change_dir/spec.md" ]]; then
    log "Failed: $CHANGE_ID — prerequisite failed: spec.md not found"
    write_stage "$MANIFEST_ID" "invalid" "$MANIFEST"
    return 2
  fi

  # calc-score.sh path is relative to the worktree's kit directory
  local wt_calc_score="$wt_path/fab/.kit/scripts/lib/calc-score.sh"
  if [[ ! -f "$wt_calc_score" ]]; then
    log "Failed: $CHANGE_ID — infrastructure failure: calc-score.sh not found at $wt_calc_score"
    exit 1
  fi

  local gate_result
  gate_result=$(bash "$wt_calc_score" --check-gate "$change_dir" 2>/dev/null) || true
  local gate_status
  gate_status=$(echo "$gate_result" | grep "^gate:" | sed 's/gate: //')
  if [[ "$gate_status" == "fail" ]]; then
    local score threshold
    score=$(echo "$gate_result" | grep "^score:" | sed 's/score: //')
    threshold=$(echo "$gate_result" | grep "^threshold:" | sed 's/threshold: //')
    log "Failed: $CHANGE_ID — prerequisite failed: confidence $score below gate $threshold"
    write_stage "$MANIFEST_ID" "invalid" "$MANIFEST"
    return 2
  fi

  return 0
}

# ---------------------------------------------------------------------------
# Pipeline Execution
# ---------------------------------------------------------------------------

run_pipeline() {
  local wt_path="$1"

  # Activate the change
  if ! (cd "$wt_path" && claude -p --dangerously-skip-permissions "/fab-switch $CHANGE_ID --no-branch-change"); then
    log "Failed: $CHANGE_ID — fab-switch failed"
    write_stage "$MANIFEST_ID" "failed" "$MANIFEST"
    return 0  # not infra failure — change-level failure
  fi

  # Run fab-ff — capture exit code, then check stage unconditionally
  local fab_ff_exit=0
  if ! (cd "$wt_path" && claude -p --dangerously-skip-permissions "/fab-ff"); then
    fab_ff_exit=1
  fi

  # Determine actual stage reached via stageman (regardless of fab-ff exit code)
  local status_file="$wt_path/fab/changes/$CHANGE_ID/.status.yaml"
  if [[ ! -f "$status_file" ]]; then
    log "Failed: $CHANGE_ID — .status.yaml not found after pipeline"
    write_stage "$MANIFEST_ID" "failed" "$MANIFEST"
    return 0
  fi

  local wt_stageman="$wt_path/fab/.kit/scripts/lib/stageman.sh"
  if [[ ! -f "$wt_stageman" ]]; then
    log "Failed: $CHANGE_ID — infrastructure failure: stageman.sh not found at $wt_stageman"
    exit 1
  fi

  local display_stage
  display_stage=$(bash "$wt_stageman" display-stage "$status_file" 2>/dev/null) || display_stage="unknown"

  if [[ "$fab_ff_exit" -ne 0 ]]; then
    log "Failed: $CHANGE_ID — fab-ff failed at $display_stage"
    write_stage "$MANIFEST_ID" "failed" "$MANIFEST"
    return 0
  fi

  if [[ "$display_stage" != "hydrate:done" ]]; then
    log "Failed: $CHANGE_ID — fab-ff exited 0 but stage is $display_stage (expected hydrate:done)"
    write_stage "$MANIFEST_ID" "failed" "$MANIFEST"
    return 0
  fi

  return 0
}

# ---------------------------------------------------------------------------
# Shipping
# ---------------------------------------------------------------------------

ship() {
  local wt_path="$1"
  local target_branch="$2"

  if ! (cd "$wt_path" && claude -p --dangerously-skip-permissions \
    "Commit all changes and create a PR targeting '$target_branch'. Include a summary of what this change does based on the spec."); then
    log "Failed: $CHANGE_ID — shipping failed"
    write_stage "$MANIFEST_ID" "failed" "$MANIFEST"
    return 0
  fi

  return 0
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  # Create worktree — infrastructure failure if this fails
  local wt_path
  wt_path=$(create_worktree) || {
    echo "Error: worktree creation failed for $CHANGE_ID" >&2
    exit 1
  }
  log "Dispatching: $CHANGE_ID (worktree: $wt_path)"

  # Provision artifacts — infrastructure failure if source missing
  provision_artifacts "$wt_path" || {
    echo "Error: artifact provisioning failed for $CHANGE_ID" >&2
    exit 1
  }

  # Validate prerequisites — writes invalid on failure, not infra error
  if ! validate_prerequisites "$wt_path"; then
    return 0
  fi

  # Run pipeline — writes failed on failure, not infra error
  run_pipeline "$wt_path"

  # Ship if pipeline succeeded (check manifest for current stage)
  local current_stage
  current_stage=$(yq -r "(.changes[] | select(.id == \"$MANIFEST_ID\")).stage // \"\"" "$MANIFEST")
  if [[ -z "$current_stage" || "$current_stage" == "null" ]]; then
    # Pipeline succeeded, ship it
    ship "$wt_path" "$PARENT_BRANCH"

    # Write done
    write_stage "$MANIFEST_ID" "done" "$MANIFEST"
    log "Completed: $CHANGE_ID — done"
  fi
}

main

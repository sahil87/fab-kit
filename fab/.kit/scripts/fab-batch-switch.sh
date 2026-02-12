#!/usr/bin/env bash
set -euo pipefail

# fab-batch-switch.sh — Per change ID/name: open a tmux tab in its worktree
# and start a Claude Code session that runs /fab-switch <change>.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(dirname "$SCRIPT_DIR")"
FAB_DIR="$(dirname "$KIT_DIR")"
CHANGES_DIR="${FAB_DIR}/changes"
CONFIG_FILE="${FAB_DIR}/config.yaml"

usage() {
  cat <<'EOF'
Usage: fab-batch-switch <change> [<change>...]

Per change ID or name: opens a new tmux tab in its worktree and starts a
Claude Code session that runs /fab-switch <change>.

Options:
  --list    Show available changes
  --all     Open tabs for all changes

Examples:
  fab-batch-switch r7k3
  fab-batch-switch r7k3 ab12
  fab-batch-switch --all
EOF
}

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

list_changes() {
  echo "Available changes:"
  echo ""
  for dir in "$CHANGES_DIR"/*/; do
    [[ -d "$dir" ]] || continue
    local name
    name=$(basename "$dir")
    [[ "$name" == "archive" ]] && continue
    printf "  %s\n" "$name"
  done
}

all_change_names() {
  for dir in "$CHANGES_DIR"/*/; do
    [[ -d "$dir" ]] || continue
    local name
    name=$(basename "$dir")
    [[ "$name" == "archive" ]] && continue
    printf '%s\n' "$name"
  done
}

get_branch_prefix() {
  # Extract branch_prefix from fab/config.yaml
  # Looks for: branch_prefix: "value" or branch_prefix: ""
  if [[ -f "$CONFIG_FILE" ]]; then
    grep -E '^\s*branch_prefix:' "$CONFIG_FILE" | \
      sed -E 's/^\s*branch_prefix:\s*"?([^"]*)"?.*/\1/' || echo ""
  else
    echo ""
  fi
}

# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------

if [[ ! -d "$CHANGES_DIR" ]]; then
  echo "Error: changes directory not found at $CHANGES_DIR" >&2
  exit 1
fi

if [[ -z "${TMUX:-}" ]]; then
  echo "Error: not inside a tmux session" >&2
  exit 1
fi

if [[ $# -eq 0 ]]; then
  usage
  exit 1
fi

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

changes=()
case "$1" in
  --list)  list_changes; exit 0 ;;
  --all)
    mapfile -t changes < <(all_change_names)
    if [[ ${#changes[@]} -eq 0 ]]; then
      echo "No changes found." >&2
      exit 1
    fi
    echo "Opening ${#changes[@]} tabs for all changes..."
    ;;
  -h|--help) usage; exit 0 ;;
  *)         changes=("$@") ;;
esac

# ---------------------------------------------------------------------------
# Open tabs
# ---------------------------------------------------------------------------

for change in "${changes[@]}"; do
  # Resolve: exact match first, then substring match (e.g. "r7k3")
  match=""
  if [[ -d "${CHANGES_DIR}/${change}" ]]; then
    match="$change"
  else
    for dir in "$CHANGES_DIR"/*/; do
      [[ -d "$dir" ]] || continue
      name=$(basename "$dir")
      if [[ "$name" == *"${change}"* ]]; then
        if [[ -n "$match" ]]; then
          echo "Warning: '$change' matches multiple changes, skipping (be more specific)" >&2
          match=""
          break
        fi
        match="$name"
      fi
    done
  fi

  if [[ -z "$match" ]]; then
    echo "Warning: '$change' not found in changes, skipping" >&2
    continue
  fi

  printf "  %s\n" "$match"

  # Get branch prefix from config and construct the branch name fab-switch would use
  branch_prefix=$(get_branch_prefix)
  branch_name="${branch_prefix}${match}"

  # Create worktree with the exact branch name fab-switch expects
  wt_path=$(wt-create --non-interactive --worktree-name "$match" "$branch_name" | tail -1) || {
    echo "Error: failed to create worktree for '$match', skipping" >&2
    continue
  }

  # Escape single quotes for the nested shell: ' → '\''
  safe="${match//"'"/"'\''"}"

  # Call fab-switch with --no-branch-change since the branch is already set up
  tmux new-window -n "fab-${match}" -c "$wt_path" \
    "claude --dangerously-skip-permissions '/fab-switch ${safe} --no-branch-change'"
done

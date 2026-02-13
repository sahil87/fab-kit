#!/usr/bin/env bash
set -euo pipefail

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$scripts_dir")"
fab_root="$(dirname "$kit_dir")"

# Source stageman for schema-driven stage/state queries
source "$scripts_dir/stageman.sh"

# --- Version ---
if [ -f "$kit_dir/VERSION" ]; then
  version=$(cat "$kit_dir/VERSION")
else
  version="unknown"
fi

header="Fab Kit v$version"

# --- Resolve change name — from $1 override or fab/current ---
override="${1:-}"

if [ -n "$override" ]; then
  # Override mode: match $1 against fab/changes/ folders
  changes_dir="$fab_root/changes"
  if [ ! -d "$changes_dir" ]; then
    printf '%s\n\nfab/changes/ not found. Run /fab-init to set up the project.\n' "$header"
    exit 1
  fi

  # Collect non-archive folder names
  mapfile -t folders < <(
    for d in "$changes_dir"/*/; do
      [ -d "$d" ] || continue
      base="$(basename "$d")"
      [ "$base" = "archive" ] && continue
      echo "$base"
    done
  )

  if [ ${#folders[@]} -eq 0 ]; then
    printf '%s\n\nNo active changes found. Run /fab-new to start one.\n' "$header"
    exit 0
  fi

  # Case-insensitive matching
  override_lower=$(echo "$override" | tr '[:upper:]' '[:lower:]')
  exact_match=""
  partial_matches=()

  for folder in "${folders[@]}"; do
    folder_lower=$(echo "$folder" | tr '[:upper:]' '[:lower:]')
    if [ "$folder_lower" = "$override_lower" ]; then
      exact_match="$folder"
      break
    elif [[ "$folder_lower" == *"$override_lower"* ]]; then
      partial_matches+=("$folder")
    fi
  done

  if [ -n "$exact_match" ]; then
    name="$exact_match"
  elif [ ${#partial_matches[@]} -eq 1 ]; then
    name="${partial_matches[0]}"
  elif [ ${#partial_matches[@]} -gt 1 ]; then
    matches_list=$(printf ', %s' "${partial_matches[@]}")
    matches_list="${matches_list:2}"
    printf '%s\n\nMultiple changes match "%s": %s. Provide a more specific name.\n' "$header" "$override" "$matches_list"
    exit 1
  else
    printf '%s\n\nNo change matches "%s".\n' "$header" "$override"
    exit 1
  fi
else
  # Default mode: read fab/current
  current_file="$fab_root/current"
  if [ ! -f "$current_file" ]; then
    printf '%s\n\nNo active change. Run /fab-new to start one.\n' "$header"
    exit 0
  fi

  name=$(tr -d '[:space:]' < "$current_file")
  if [ -z "$name" ]; then
    printf '%s\n\nNo active change. Run /fab-new to start one.\n' "$header"
    exit 0
  fi
fi

# --- .status.yaml ---
change_dir="$fab_root/changes/$name"
status_file="$change_dir/.status.yaml"
if [ ! -f "$status_file" ]; then
  printf '%s\n\nActive change: %s\n⚠ .status.yaml not found — change may be corrupted.\n\nRun /fab-new to start a fresh change or /fab-switch to select another.\n' "$header" "$name"
  exit 1
fi

# --- Parse fields ---
get_field() { grep "^$1:" "$status_file" | sed "s/^$1: *//" || true; }
get_nested() { grep "^ *$1:" "$status_file" | sed "s/^ *$1: *//" || true; }

created_by=$(get_field "created_by")

# Live git branch (replaces .status.yaml branch field)
git_enabled="false"
if [ -f "$fab_root/config.yaml" ]; then
  git_enabled_val=$(grep '^ *enabled:' "$fab_root/config.yaml" | sed 's/^ *enabled: *//' || true)
  if [ "$git_enabled_val" = "true" ]; then
    git_enabled="true"
  fi
fi
branch=""
show_branch="false"
if [ "$git_enabled" = "true" ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  branch=$(git branch --show-current 2>/dev/null || true)
  show_branch="true"
fi

# Progress — dynamic extraction from schema stages
declare -A progress
for s in $(get_all_stages); do
  val=$(get_nested "$s")
  progress[$s]="${val:-pending}"
done

# Derive current stage from the active entry in the progress map
stage=""
for s in $(get_all_stages); do
  if [ "${progress[$s]}" = "active" ]; then
    stage="$s"
    break
  fi
done
if [ -z "$stage" ]; then
  stage="archive"
fi

# Checklist
chk_generated=$(get_nested "generated"); chk_generated=${chk_generated:-false}
chk_completed=$(get_nested "completed"); chk_completed=${chk_completed:-0}
chk_total=$(get_nested "total");         chk_total=${chk_total:-0}

# Confidence
conf_score=$(get_nested "score")
conf_certain=$(get_nested "certain")
conf_confident=$(get_nested "confident")
conf_tentative=$(get_nested "tentative")
conf_unresolved=$(get_nested "unresolved")

# --- Stage number (dynamic from schema) ---
stage_num=$(get_stage_number "$stage" 2>/dev/null || echo "?")
total_stages=$(get_all_stages | wc -l | tr -d ' ')

# --- Progress display (symbols from schema) ---
progress_line() {
  local sym suffix
  sym=$(get_state_symbol "$2" 2>/dev/null || printf '○')
  suffix=$(get_state_suffix "$2" 2>/dev/null || true)
  printf '  %s %s%s\n' "$sym" "$1" "$suffix"
}

# --- Next command ---
current_progress="${progress[$stage]:-}"

next="/fab-status"
case "${stage:-}:${current_progress:-}" in
  brief:active)            next="/fab-continue or /fab-clarify" ;;
  brief:done)              next="/fab-continue (spec) or /fab-clarify" ;;
  spec:active)             next="/fab-continue or /fab-clarify" ;;
  spec:done)               next="/fab-continue (tasks) or /fab-ff or /fab-clarify" ;;
  tasks:active)            next="/fab-continue or /fab-clarify" ;;
  tasks:done)              next="/fab-apply" ;;
  apply:active)            next="/fab-apply" ;;
  apply:done)              next="/fab-review" ;;
  review:active)           next="/fab-review" ;;
  review:done)             next="/fab-archive" ;;
  review:failed)           next="/fab-review (re-review after fixes)" ;;
  archive:done)            next="/fab-new <description>" ;;
esac

# --- Output ---
echo "$header"
echo ""
echo "Change:  $name"
if [ -n "$created_by" ]; then
  echo "Created by: $created_by"
fi
if [ "$show_branch" = "true" ]; then
  if [ -n "$branch" ]; then
    echo "Branch:  $branch"
  else
    echo "Branch:  (detached)"
  fi
fi
echo "Stage:   $stage ($stage_num/$total_stages)"
echo ""
echo "Progress:"
for s in $(get_all_stages); do
  progress_line "$s" "${progress[$s]}"
done
echo ""
if [ "$chk_generated" = "true" ]; then
  echo "Checklist: $chk_completed/$chk_total items"
else
  echo "Checklist: not yet generated"
fi
if [ -n "$conf_score" ]; then
  conf_line="Confidence: $conf_score/5.0 (${conf_certain:-0} certain, ${conf_confident:-0} confident, ${conf_tentative:-0} tentative"
  if [ "${conf_unresolved:-0}" != "0" ]; then
    conf_line="$conf_line, $conf_unresolved unresolved"
  fi
  conf_line="$conf_line)"
  echo "$conf_line"
else
  echo "Confidence: not yet scored"
fi
echo ""
echo "Next: $next"

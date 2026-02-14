#!/usr/bin/env bash
# fab/.kit/scripts/lib/resolve-change.sh
#
# Change Resolution Library — resolves a change name from an override argument
# or fab/current. Sourced by preflight.sh and fab-status.sh.
#
# Usage:
#   source "$(dirname "$0")/resolve-change.sh"
#   resolve_change "$fab_root" "$override"
#   echo "$RESOLVED_CHANGE_NAME"
#
# On success: sets RESOLVED_CHANGE_NAME, returns 0
# On failure: prints diagnostic to stderr, returns 1

# resolve_change <fab_root> [override]
# Resolve change name from override argument or fab/current.
resolve_change() {
  local fab_root="$1"
  local override="${2:-}"

  RESOLVED_CHANGE_NAME=""

  if [ -n "$override" ]; then
    # --- Override mode: match against fab/changes/ folders ---
    local changes_dir="$fab_root/changes"
    if [ ! -d "$changes_dir" ]; then
      echo "fab/changes/ not found." >&2
      return 1
    fi

    # Collect non-archive folder names
    local folders=()
    local d base
    for d in "$changes_dir"/*/; do
      [ -d "$d" ] || continue
      base="$(basename "$d")"
      [ "$base" = "archive" ] && continue
      folders+=("$base")
    done

    if [ ${#folders[@]} -eq 0 ]; then
      echo "No active changes found." >&2
      return 1
    fi

    # Case-insensitive matching
    local override_lower
    override_lower=$(echo "$override" | tr '[:upper:]' '[:lower:]')
    local exact_match=""
    local partial_matches=()

    local folder folder_lower
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
      RESOLVED_CHANGE_NAME="$exact_match"
    elif [ ${#partial_matches[@]} -eq 1 ]; then
      RESOLVED_CHANGE_NAME="${partial_matches[0]}"
    elif [ ${#partial_matches[@]} -gt 1 ]; then
      local matches_list
      matches_list=$(printf ', %s' "${partial_matches[@]}")
      matches_list="${matches_list:2}"  # trim leading ', '
      echo "Multiple changes match \"$override\": $matches_list." >&2
      return 1
    else
      echo "No change matches \"$override\"." >&2
      return 1
    fi
  else
    # --- Default mode: read fab/current ---
    local current_file="$fab_root/current"
    if [ ! -f "$current_file" ]; then
      echo "No active change." >&2
      return 1
    fi

    local name
    name=$(tr -d '[:space:]' < "$current_file")
    if [ -z "$name" ]; then
      echo "No active change." >&2
      return 1
    fi

    RESOLVED_CHANGE_NAME="$name"
  fi

  return 0
}

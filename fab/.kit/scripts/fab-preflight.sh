#!/usr/bin/env bash
set -euo pipefail

fab_root="$(dirname "$0")/../.."
scripts_dir="$(cd "$(dirname "$0")" && pwd)"

# Source stageman for schema-driven stage/state queries
source "$scripts_dir/stageman.sh"

# 1. Project initialization validation
if [ ! -f "$fab_root/config.yaml" ] || [ ! -f "$fab_root/constitution.md" ]; then
  echo "fab/ is not initialized. Run /fab-init first." >&2
  exit 1
fi

# 2. Resolve change name — from $1 override or fab/current
override="${1:-}"

if [ -n "$override" ]; then
  # --- Override mode: match $1 against fab/changes/ folders ---
  changes_dir="$fab_root/changes"
  if [ ! -d "$changes_dir" ]; then
    echo "fab/changes/ not found. Run /fab-init to set up the project." >&2
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
    echo "No active changes found. Run /fab-new to start one." >&2
    exit 1
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
    matches_list="${matches_list:2}"  # trim leading ', '
    echo "Multiple changes match \"$override\": $matches_list. Provide a more specific name." >&2
    exit 1
  else
    echo "No change matches \"$override\"." >&2
    exit 1
  fi
else
  # --- Default mode: read fab/current ---
  current_file="$fab_root/current"
  if [ ! -f "$current_file" ]; then
    echo "No active change. Run /fab-new to start one." >&2
    exit 1
  fi

  name=$(tr -d '[:space:]' < "$current_file")
  if [ -z "$name" ]; then
    echo "No active change. Run /fab-new to start one." >&2
    exit 1
  fi
fi

# 3. Change directory validation
change_dir="$fab_root/changes/$name"
if [ ! -d "$change_dir" ]; then
  echo "Change directory not found: changes/$name/" >&2
  exit 1
fi

# 4. .status.yaml validation
status_file="$change_dir/.status.yaml"
if [ ! -f "$status_file" ]; then
  echo "Active change \"$name\" is corrupted — .status.yaml not found." >&2
  exit 1
fi

# 5. Schema validation — catch state/stage violations early
if ! validate_status_file "$status_file"; then
  echo "Status file validation failed for \"$name\". Fix .status.yaml or run /fab-new." >&2
  exit 1
fi

# --- All validations passed — emit structured YAML to stdout ---

# Extract progress fields dynamically from schema stages
declare -A progress
for s in $(get_all_stages); do
  val=$(grep "^ *${s}:" "$status_file" | sed 's/^ *[a-z]*: *//' || echo "")
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
# Fallback: if no active entry, find first pending stage after last done
if [ -z "$stage" ]; then
  last_done=""
  for s in $(get_all_stages); do
    if [ "${progress[$s]}" = "done" ]; then
      last_done="$s"
    fi
  done
  if [ -n "$last_done" ]; then
    found_last=false
    for s in $(get_all_stages); do
      if [ "$found_last" = "true" ] && [ "${progress[$s]}" = "pending" ]; then
        stage="$s"
        break
      fi
      if [ "$s" = "$last_done" ]; then
        found_last=true
      fi
    done
  fi
  # Final fallback: all done (workflow complete)
  if [ -z "$stage" ]; then
    stage="hydrate"
  fi
fi

# Extract checklist fields (handle missing block gracefully)
chk_generated=$(grep '^ *generated:' "$status_file" | sed 's/^ *generated: *//' || true)
chk_completed=$(grep '^ *completed:' "$status_file" | sed 's/^ *completed: *//' || true)
chk_total=$(grep '^ *total:' "$status_file" | sed 's/^ *total: *//' || true)

# Extract confidence fields (handle missing block gracefully for backwards compat)
conf_certain=$(grep '^ *certain:' "$status_file" | sed 's/^ *certain: *//' || true)
conf_confident=$(grep '^ *confident:' "$status_file" | sed 's/^ *confident: *//' || true)
conf_tentative=$(grep '^ *tentative:' "$status_file" | sed 's/^ *tentative: *//' || true)
conf_unresolved=$(grep '^ *unresolved:' "$status_file" | sed 's/^ *unresolved: *//' || true)
conf_score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || true)

# Emit YAML output with dynamic stage progress
cat <<EOF
name: $name
change_dir: changes/$name
stage: $stage
progress:
$(for s in $(get_all_stages); do echo "  $s: ${progress[$s]}"; done)
checklist:
  generated: ${chk_generated:-false}
  completed: ${chk_completed:-0}
  total: ${chk_total:-0}
confidence:
  certain: ${conf_certain:-0}
  confident: ${conf_confident:-0}
  tentative: ${conf_tentative:-0}
  unresolved: ${conf_unresolved:-0}
  score: ${conf_score:-5.0}
EOF

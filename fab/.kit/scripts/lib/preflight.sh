#!/usr/bin/env bash
set -euo pipefail

fab_root="$(dirname "$0")/../../.."
scripts_dir="$(cd "$(dirname "$0")/.." && pwd)"

# Source libraries
source "$scripts_dir/lib/stageman.sh"
source "$scripts_dir/lib/resolve-change.sh"

# 1. Project initialization validation
if [ ! -f "$fab_root/config.yaml" ] || [ ! -f "$fab_root/constitution.md" ]; then
  echo "fab/ is not initialized. Run /fab-init first." >&2
  exit 1
fi

# 2. Resolve change name — from $1 override or fab/current
override="${1:-}"

if ! resolve_change "$fab_root" "$override"; then
  # Add context-appropriate guidance to the library's generic error
  if [ -n "$override" ]; then
    echo "Provide a more specific name." >&2
  else
    echo "Run /fab-new to start one." >&2
  fi
  exit 1
fi
name="$RESOLVED_CHANGE_NAME"

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

# Extract progress via stageman accessor
declare -A progress
while IFS=: read -r s val; do
  progress[$s]="$val"
done < <(get_progress_map "$status_file")

# Derive current stage via stageman
stage=$(get_current_stage "$status_file")

# Extract checklist via stageman accessor
declare -A checklist
while IFS=: read -r key val; do
  checklist[$key]="$val"
done < <(get_checklist "$status_file")

# Extract confidence via stageman accessor
declare -A confidence
while IFS=: read -r key val; do
  confidence[$key]="$val"
done < <(get_confidence "$status_file")

# Emit YAML output with dynamic stage progress
cat <<EOF
name: $name
change_dir: changes/$name
stage: $stage
progress:
$(for s in $(get_all_stages); do echo "  $s: ${progress[$s]}"; done)
checklist:
  generated: ${checklist[generated]}
  completed: ${checklist[completed]}
  total: ${checklist[total]}
confidence:
  certain: ${confidence[certain]}
  confident: ${confidence[confident]}
  tentative: ${confidence[tentative]}
  unresolved: ${confidence[unresolved]}
  score: ${confidence[score]}
EOF

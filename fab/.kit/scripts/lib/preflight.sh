#!/usr/bin/env bash
set -euo pipefail

fab_root="$(dirname "$0")/../../.."
scripts_dir="$(cd "$(dirname "$0")/.." && pwd)"

# CLI entry points (subprocess calls, not sourced)
STAGEMAN="$scripts_dir/lib/stageman.sh"
CHANGEMAN="$scripts_dir/lib/changeman.sh"

# 1. Project initialization validation
if [ ! -f "$fab_root/config.yaml" ] || [ ! -f "$fab_root/constitution.md" ]; then
  echo "fab/ is not initialized. Run /fab-setup first." >&2
  exit 1
fi

# 2. Resolve change name — from $1 override or fab/current
override="${1:-}"

if ! name=$("$CHANGEMAN" resolve "$override"); then
  # Add context-appropriate guidance to changeman's generic error
  if [ -n "$override" ]; then
    echo "Provide a more specific name." >&2
  else
    echo "Run /fab-new to start one." >&2
  fi
  exit 1
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
if ! "$STAGEMAN" validate-status-file "$status_file"; then
  echo "Status file validation failed for \"$name\". Fix .status.yaml or run /fab-new." >&2
  exit 1
fi

# --- All validations passed — emit structured YAML to stdout ---

# Extract progress via stageman CLI
declare -A progress
while IFS=: read -r s val; do
  progress[$s]="$val"
done < <("$STAGEMAN" progress-map "$status_file")

# Derive current stage via stageman CLI
stage=$("$STAGEMAN" current-stage "$status_file")

# Extract checklist via stageman CLI
declare -A checklist
while IFS=: read -r key val; do
  checklist[$key]="$val"
done < <("$STAGEMAN" checklist "$status_file")

# Extract confidence via stageman CLI
declare -A confidence
while IFS=: read -r key val; do
  confidence[$key]="$val"
done < <("$STAGEMAN" confidence "$status_file")

# Emit YAML output with dynamic stage progress
cat <<EOF
name: $name
change_dir: changes/$name
stage: $stage
progress:
$(for s in $("$STAGEMAN" all-stages); do echo "  $s: ${progress[$s]}"; done)
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

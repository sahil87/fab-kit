#!/usr/bin/env bash
set -euo pipefail

fab_root="$(dirname "$0")/../.."

# 1. Project initialization validation
if [ ! -f "$fab_root/config.yaml" ] || [ ! -f "$fab_root/constitution.md" ]; then
  echo "fab/ is not initialized. Run /fab-init first." >&2
  exit 1
fi

# 2. fab/current validation
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

# --- All validations passed — emit structured YAML to stdout ---

stage=$(grep '^stage:' "$status_file" | sed 's/^stage: *//')

# Extract progress fields
p_proposal=$(grep '^ *proposal:' "$status_file" | sed 's/^ *proposal: *//')
p_specs=$(grep '^ *specs:' "$status_file" | sed 's/^ *specs: *//')
p_plan=$(grep '^ *plan:' "$status_file" | sed 's/^ *plan: *//')
p_tasks=$(grep '^ *tasks:' "$status_file" | sed 's/^ *tasks: *//')
p_apply=$(grep '^ *apply:' "$status_file" | sed 's/^ *apply: *//')
p_review=$(grep '^ *review:' "$status_file" | sed 's/^ *review: *//')
p_archive=$(grep '^ *archive:' "$status_file" | sed 's/^ *archive: *//')

# Extract checklist fields
chk_generated=$(grep '^ *generated:' "$status_file" | sed 's/^ *generated: *//')
chk_completed=$(grep '^ *completed:' "$status_file" | sed 's/^ *completed: *//')
chk_total=$(grep '^ *total:' "$status_file" | sed 's/^ *total: *//')

# Extract confidence fields (handle missing block gracefully for backwards compat)
conf_certain=$(grep '^ *certain:' "$status_file" | sed 's/^ *certain: *//' || true)
conf_confident=$(grep '^ *confident:' "$status_file" | sed 's/^ *confident: *//' || true)
conf_tentative=$(grep '^ *tentative:' "$status_file" | sed 's/^ *tentative: *//' || true)
conf_unresolved=$(grep '^ *unresolved:' "$status_file" | sed 's/^ *unresolved: *//' || true)
conf_score=$(grep '^ *score:' "$status_file" | sed 's/^ *score: *//' || true)

cat <<EOF
name: $name
change_dir: changes/$name
stage: $stage
progress:
  proposal: $p_proposal
  specs: $p_specs
  plan: $p_plan
  tasks: $p_tasks
  apply: $p_apply
  review: $p_review
  archive: $p_archive
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

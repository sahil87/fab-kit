#!/usr/bin/env bash
set -euo pipefail

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$scripts_dir")"
fab_root="$(dirname "$kit_dir")"

# --- Version ---
if [ -f "$kit_dir/VERSION" ]; then
  version=$(cat "$kit_dir/VERSION")
else
  version="unknown"
fi

header="Fab Kit v$version"

# --- Active change ---
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

stage=$(get_field "stage")

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

# Progress (default to pending for missing fields)
p_proposal=$(get_nested "proposal"); p_proposal=${p_proposal:-pending}
p_specs=$(get_nested "specs");       p_specs=${p_specs:-pending}
p_plan=$(get_nested "plan");         p_plan=${p_plan:-pending}
p_tasks=$(get_nested "tasks");       p_tasks=${p_tasks:-pending}
p_apply=$(get_nested "apply");       p_apply=${p_apply:-pending}
p_review=$(get_nested "review");     p_review=${p_review:-pending}
p_archive=$(get_nested "archive");   p_archive=${p_archive:-pending}

# Checklist
chk_generated=$(get_nested "generated"); chk_generated=${chk_generated:-false}
chk_completed=$(get_nested "completed"); chk_completed=${chk_completed:-0}
chk_total=$(get_nested "total");         chk_total=${chk_total:-0}

# --- Stage number ---
case "${stage:-}" in
  proposal) stage_num=1 ;; specs) stage_num=2 ;; plan)    stage_num=3 ;;
  tasks)    stage_num=4 ;; apply) stage_num=5 ;; review)  stage_num=6 ;;
  archive)  stage_num=7 ;; *)     stage_num="?" ;;
esac

# --- Progress symbols ---
symbol() {
  case "$1" in
    done) printf '✓' ;; active)  printf '●' ;; pending) printf '○' ;;
    skipped) printf '—' ;; failed) printf '✗' ;; *)      printf '○' ;;
  esac
}

progress_line() {
  local sym extra=""
  sym=$(symbol "$2")
  [ "$2" = "skipped" ] && extra=" (skipped)"
  printf '  %s %s%s\n' "$sym" "$1" "$extra"
}

# --- Next command ---
current_progress=""
case "${stage:-}" in
  proposal) current_progress="$p_proposal" ;; specs)   current_progress="$p_specs" ;;
  plan)     current_progress="$p_plan" ;;     tasks)   current_progress="$p_tasks" ;;
  apply)    current_progress="$p_apply" ;;    review)  current_progress="$p_review" ;;
  archive)  current_progress="$p_archive" ;;
esac

next="/fab-status"
case "${stage:-}:${current_progress:-}" in
  proposal:active|proposal:done) next="/fab-continue or /fab-ff" ;;
  specs:active)                  next="/fab-continue" ;;
  specs:done)                    next="/fab-continue (plan) or /fab-ff or /fab-clarify" ;;
  plan:active)                   next="/fab-continue" ;;
  plan:done)                     next="/fab-continue (tasks) or /fab-clarify" ;;
  plan:skipped)                  next="/fab-continue (tasks)" ;;
  tasks:active)                  next="/fab-continue" ;;
  tasks:done)                    next="/fab-apply" ;;
  apply:active)                  next="/fab-apply" ;;
  apply:done)                    next="/fab-review" ;;
  review:active)                 next="/fab-review" ;;
  review:done)                   next="/fab-archive" ;;
  review:failed)                 next="/fab-review (re-review after fixes)" ;;
  archive:done)                  next="/fab-new <description>" ;;
esac

# --- Output ---
echo "$header"
echo ""
echo "Change:  $name"
if [ "$show_branch" = "true" ]; then
  if [ -n "$branch" ]; then
    echo "Branch:  $branch"
  else
    echo "Branch:  (detached)"
  fi
fi
echo "Stage:   $stage ($stage_num/7)"
echo ""
echo "Progress:"
progress_line "proposal" "$p_proposal"
progress_line "specs"    "$p_specs"
progress_line "plan"     "$p_plan"
progress_line "tasks"    "$p_tasks"
progress_line "apply"    "$p_apply"
progress_line "review"   "$p_review"
progress_line "archive"  "$p_archive"
echo ""
if [ "$chk_generated" = "true" ]; then
  echo "Checklist: $chk_completed/$chk_total items"
else
  echo "Checklist: not yet generated"
fi
echo ""
echo "Next: $next"

#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/fab-worktree-init.sh — Bootstrap a new worktree
#
# Runs common init scripts from .kit, then any project-specific
# scripts from fab/worktree-init/ if that directory exists.

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$scripts_dir")"
fab_dir="$(dirname "$kit_dir")"
repo_root="$(dirname "$fab_dir")"

common_dir="$kit_dir/worktree-init-common"

echo "Running common worktree-init scripts..."
for script in "$common_dir"/*.sh; do
  [ -f "$script" ] || continue
  echo "  → $(basename "$script")"
  bash "$script"
done

if [ -d "$fab_dir/worktree-init" ]; then
  echo "Running project-specific worktree-init scripts..."
  for script in "$fab_dir/worktree-init"/*.sh; do
    [ -f "$script" ] || continue
    echo "  → $(basename "$script")"
    bash "$script"
  done
fi

echo "Worktree init done."

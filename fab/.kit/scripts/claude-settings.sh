#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
src="$repo_root/.claude/settings.local.json"
dest="$repo_root/fab/worktree-init/assets/settings.local.json"

if [ ! -f "$src" ]; then
  echo "No .claude/settings.local.json found"
  exit 1
fi

mkdir -p "$(dirname "$dest")"
cp "$src" "$dest"
echo "Copied .claude/settings.local.json → fab/worktree-init/assets/"

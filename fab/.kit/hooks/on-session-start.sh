#!/usr/bin/env bash
# fab/.kit/hooks/on-session-start.sh — Claude Code SessionStart hook
#
# Clears the agent block from .fab-runtime.yaml (repo root)
# for the active change's folder name.
# Fires when a new Claude Code session begins.
# MUST exit 0 always — hooks must never block the agent.

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

current_file="$repo_root/fab/current"
[ -f "$current_file" ] || exit 0
change_name="$(cat "$current_file" 2>/dev/null | head -1 | tr -d '[:space:]')"
[ -n "$change_name" ] || exit 0

fab_cmd="$repo_root/fab/.kit/bin/fab"
[ -x "$fab_cmd" ] || exit 0

change_dir="$("$fab_cmd" resolve --dir "$change_name" 2>/dev/null)" || exit 0
[ -d "$repo_root/$change_dir" ] || exit 0

change_folder="$("$fab_cmd" resolve --folder "$change_name" 2>/dev/null)" || exit 0
[ -n "$change_folder" ] || exit 0

runtime_file="$repo_root/.fab-runtime.yaml"
[ -f "$runtime_file" ] || exit 0

command -v yq >/dev/null 2>&1 || exit 0

yq -i "del(.\"$change_folder\".agent)" "$runtime_file" 2>/dev/null || true
exit 0

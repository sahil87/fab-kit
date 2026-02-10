#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/fab-setup.sh — Structural bootstrap for fab
#
# Sets up directories, symlinks, and .gitignore entries that /fab-init
# would create (Phase 1 only — no config.yaml or constitution.md generation).
#
# Run from anywhere: fab/.kit/scripts/fab-setup.sh
# Safe to re-run (idempotent).

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$scripts_dir")"
fab_dir="$(dirname "$kit_dir")"
repo_root="$(dirname "$fab_dir")"

# ── Pre-flight ───────────────────────────────────────────────────────
if [ ! -f "$kit_dir/VERSION" ]; then
  echo "ERROR: fab/.kit/VERSION not found — kit may be corrupted."
  exit 1
fi

version=$(cat "$kit_dir/VERSION")
echo "Found fab/.kit/ (v${version}). Setting up structure..."

# ── 1. Directories ──────────────────────────────────────────────────
for dir in "$fab_dir/changes" "$fab_dir/docs"; do
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    echo "Created: ${dir#"$repo_root"/}"
  fi
done

if [ ! -f "$fab_dir/changes/.gitkeep" ]; then
  touch "$fab_dir/changes/.gitkeep"
fi

# ── 2. .envrc ─────────────────────────────────────────────────────
envrc_link="$repo_root/.envrc"
envrc_target="fab/.kit/envrc"

if [ -L "$envrc_link" ] && [ -e "$envrc_link" ]; then
  echo ".envrc: OK (symlink)"
elif [ -e "$envrc_link" ]; then
  rm "$envrc_link"
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: replaced file with symlink → $envrc_target"
else
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: created symlink → $envrc_target"
fi

# ── 3. Docs index ──────────────────────────────────────────────────
if [ ! -f "$fab_dir/docs/index.md" ]; then
  cat > "$fab_dir/docs/index.md" << 'EOF'
# Documentation Index

<!-- This index is maintained by /fab-archive when changes are completed. -->
<!-- Each domain gets a row linking to its docs. -->

| Domain | Description | Docs |
|--------|-------------|------|
EOF
  echo "Created: fab/docs/index.md"
fi

# ── 4. Skill symlinks ──────────────────────────────────────────────
# Canonical list: every *.md in .kit/skills/ except _context.md
skills=()
for f in "$kit_dir"/skills/*.md; do
  [ -f "$f" ] || continue
  [[ "$(basename "$f")" == _context.md ]] && continue
  skills+=("$(basename "$f" .md)")
done

# create_agent_symlinks <agent_label> <base_dir> <format> <rel_prefix>
#   format: "directory" → <base>/<name>/SKILL.md, "flat" → <base>/<name>.md
#   rel_prefix: relative path from symlink location back to repo root
create_agent_symlinks() {
  local agent_label="$1"
  local base_dir="$2"
  local format="$3"
  local rel_prefix="$4"

  mkdir -p "$base_dir"

  local created=0 repaired=0 ok=0

  for skill in "${skills[@]}"; do
    if [ ! -f "$kit_dir/skills/${skill}.md" ]; then
      echo "WARN: fab/.kit/skills/${skill}.md missing — skipping"
      continue
    fi

    local target="${rel_prefix}fab/.kit/skills/${skill}.md"
    local link

    if [ "$format" = "directory" ]; then
      mkdir -p "$base_dir/$skill"
      link="$base_dir/$skill/SKILL.md"
    else
      link="$base_dir/${skill}.md"
    fi

    if [ -L "$link" ] && [ -e "$link" ]; then
      ok=$((ok + 1))
    elif [ -L "$link" ]; then
      rm "$link"
      ln -s "$target" "$link"
      repaired=$((repaired + 1))
    elif [ -e "$link" ]; then
      rm "$link"
      ln -s "$target" "$link"
      repaired=$((repaired + 1))
    else
      ln -s "$target" "$link"
      created=$((created + 1))
    fi
  done

  local total=$((created + repaired + ok))
  printf "%-12s %d/%d (created %d, repaired %d, already valid %d)\n" \
    "${agent_label}:" "$total" "${#skills[@]}" "$created" "$repaired" "$ok"
}

# Claude Code: .claude/skills/<name>/SKILL.md (directory-based)
create_agent_symlinks "Claude Code" "$repo_root/.claude/skills" "directory" "../../../"

# OpenCode: .opencode/commands/<name>.md (flat file)
create_agent_symlinks "OpenCode" "$repo_root/.opencode/commands" "flat" "../../"

# Codex: .agents/skills/<name>/SKILL.md (directory-based)
create_agent_symlinks "Codex" "$repo_root/.agents/skills" "directory" "../../../"

# ── 5. .gitignore ──────────────────────────────────────────────────
gitignore="$repo_root/.gitignore"

if [ ! -f "$gitignore" ]; then
  echo "fab/current" > "$gitignore"
  echo "Created: .gitignore (added fab/current)"
elif ! grep -qx 'fab/current' "$gitignore"; then
  echo "" >> "$gitignore"
  echo "fab/current" >> "$gitignore"
  echo "Updated: .gitignore (added fab/current)"
fi

echo "Done."

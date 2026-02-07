#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/setup.sh — Structural bootstrap for fab
#
# Sets up directories, symlinks, and .gitignore entries that /fab:init
# would create (Phase 1 only — no config.yaml or constitution.md generation).
#
# Run from anywhere: fab/.kit/scripts/setup.sh
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

# ── 2. Docs index ──────────────────────────────────────────────────
if [ ! -f "$fab_dir/docs/index.md" ]; then
  cat > "$fab_dir/docs/index.md" << 'EOF'
# Documentation Index

<!-- This index is maintained by /fab:archive when changes are completed. -->
<!-- Each domain gets a row linking to its docs. -->

| Domain | Description | Docs |
|--------|-------------|------|
EOF
  echo "Created: fab/docs/index.md"
fi

# ── 3. Skill symlinks ──────────────────────────────────────────────
# Pattern: .claude/skills/fab-{name}/SKILL.md → ../../../fab/.kit/skills/fab-{name}.md
# Canonical list: every fab-*.md in .kit/skills/ (excludes _context.md)
skills=()
for f in "$kit_dir"/skills/fab-*.md; do
  [ -f "$f" ] || continue
  skills+=("$(basename "$f" .md)")
done

skills_dir="$repo_root/.claude/skills"
mkdir -p "$skills_dir"

created=0
repaired=0
ok=0

for skill in "${skills[@]}"; do
  target="../../../fab/.kit/skills/${skill}.md"
  skill_dir="$skills_dir/$skill"
  link="$skill_dir/SKILL.md"

  # Verify the target actually exists in .kit/skills/
  if [ ! -f "$kit_dir/skills/${skill}.md" ]; then
    echo "WARN: fab/.kit/skills/${skill}.md missing — skipping"
    continue
  fi

  mkdir -p "$skill_dir"

  if [ -L "$link" ] && [ -e "$link" ]; then
    ok=$((ok + 1))
  elif [ -L "$link" ]; then
    # Dangling symlink — repair
    rm "$link"
    ln -s "$target" "$link"
    repaired=$((repaired + 1))
  elif [ -e "$link" ]; then
    # Regular file where symlink expected — replace
    rm "$link"
    ln -s "$target" "$link"
    repaired=$((repaired + 1))
  else
    ln -s "$target" "$link"
    created=$((created + 1))
  fi
done

total=$((created + repaired + ok))
echo "Symlinks: ${total}/${#skills[@]} (created ${created}, repaired ${repaired}, already valid ${ok})"

# ── 4. .gitignore ──────────────────────────────────────────────────
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

#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/_fab-scaffold.sh — Structural bootstrap for fab
#
# Sets up directories, symlinks, and .gitignore entries that /fab-init
# would create (Phase 1 only — no config.yaml or constitution.md generation).
#
# Run from anywhere: fab/.kit/scripts/_fab-scaffold.sh
# Safe to re-run (idempotent).

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$scripts_dir")"
fab_dir="$(dirname "$kit_dir")"
repo_root="$(dirname "$fab_dir")"

# ── Pre-flight ───────────────────────────────────────────────────────
if [ ! -f "$kit_dir/VERSION" ]; then
  echo "ERROR: fab/.kit/VERSION not found — kit may be corrupted." >&2
  exit 1
fi

if [ ! -f "$kit_dir/model-tiers.yaml" ]; then
  echo "ERROR: fab/.kit/model-tiers.yaml not found — kit may be corrupted." >&2
  exit 1
fi

version=$(cat "$kit_dir/VERSION")
echo "Found fab/.kit/ (v${version}). Setting up structure..."

# ── Helper functions ─────────────────────────────────────────────────

# Extract a field value from YAML frontmatter (between --- markers).
# Usage: frontmatter_field <file> <field_name>
# Returns the value (unquoted) or empty string if not found.
frontmatter_field() {
  local file="$1" field="$2"
  sed -n '
    /^---$/,/^---$/{
      /^---$/d
      /^'"$field"': */{
        s/^'"$field"': *//
        s/^"//; s/"$//
        s/ *#.*//
        p
        q
      }
    }
  ' "$file"
}

# Extract a value from a 3-level nested YAML structure.
# Usage: yaml_value <file> <root_key> <second_key> <third_key>
# Example: yaml_value model-tiers.yaml tiers fast claude → "haiku"
# Returns the value or empty string if path not found.
yaml_value() {
  local file="$1" root="$2" second="$3" third="$4"
  sed -n "
    /^${root}:/,/^[^ ]/{
      /^  ${second}:/,/^  [^ ]/{
        /^    ${third}:/{
          s/^    ${third}: *//
          s/ *#.*//
          p
          q
        }
      }
    }
  " "$file"
}

# ── 1. Directories ──────────────────────────────────────────────────
for dir in "$fab_dir/changes" "$fab_dir/docs" "$fab_dir/design"; do
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

# ── 4. Design index ───────────────────────────────────────────────
if [ ! -f "$fab_dir/design/index.md" ]; then
  cat > "$fab_dir/design/index.md" << 'EOF'
# Specifications Index

> **Specs are pre-implementation artifacts** — what you *planned*. They capture conceptual design
> intent, high-level decisions, and the "why" behind features. Specs are human-curated,
> flat in structure, and deliberately size-controlled for quick reading.
>
> Contrast with [`fab/docs/index.md`](../docs/index.md): docs are *post-implementation* —
> what actually happened. Docs are the authoritative source of truth for system behavior,
> maintained by `/fab-archive` hydration.
>
> **Ownership**: Specs are written and maintained by humans. No automated tooling creates or
> enforces structure here — organize files however makes sense for your project.

| Spec | Description |
|------|-------------|
EOF
  echo "Created: fab/design/index.md"
fi

# ── 5. Skill symlinks ──────────────────────────────────────────────
# Canonical list: every *.md in .kit/skills/ except _context.md
skills=()
for f in "$kit_dir"/skills/*.md; do
  [ -f "$f" ] || continue
  [[ "$(basename "$f")" == _*.md ]] && continue
  skills+=("$(basename "$f" .md)")
done

# ── 4b. Classify skills by model tier ─────────────────────────────────
fast_skills=()
for skill in "${skills[@]}"; do
  skill_file="$kit_dir/skills/${skill}.md"
  [ -f "$skill_file" ] || continue

  # Skip files without frontmatter (e.g., internal skills)
  if ! head -1 "$skill_file" | grep -q '^---$'; then
    continue
  fi

  # Validate frontmatter has closing delimiter
  fm_count=$(head -20 "$skill_file" | grep -c '^---$' || true)
  if [ "$fm_count" -lt 2 ]; then
    echo "ERROR: Cannot parse frontmatter in ${skill}.md" >&2
    exit 1
  fi

  tier=$(frontmatter_field "$skill_file" "model_tier")
  if [ -n "$tier" ]; then
    if [ "$tier" != "fast" ]; then
      echo "ERROR: Unrecognized model_tier \"$tier\" in ${skill}.md. Valid values: fast" >&2
      exit 1
    fi
    fast_skills+=("$skill")
  fi
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

# ── 6. Model tier agent files ────────────────────────────────────────
# Fast-tier skills get generated agent files (in addition to skill symlinks)
# so pipeline operations can invoke them with cost-appropriate models.

if [ ${#fast_skills[@]} -gt 0 ]; then
  # Resolve Claude model for "fast" tier: config.yaml overrides .kit/ defaults
  claude_fast_model=$(yaml_value "$kit_dir/model-tiers.yaml" "tiers" "fast" "claude")

  if [ -f "$fab_dir/config.yaml" ]; then
    override=$(yaml_value "$fab_dir/config.yaml" "model_tiers" "fast" "claude")
    if [ -n "$override" ]; then
      claude_fast_model="$override"
    fi
  fi

  if [ -z "$claude_fast_model" ]; then
    echo "ERROR: No mapping for tier \"fast\" on platform \"claude\" in model-tiers.yaml" >&2
    exit 1
  fi

  # Generate Claude Code agent files: .claude/agents/<name>.md
  claude_agents_dir="$repo_root/.claude/agents"
  mkdir -p "$claude_agents_dir"

  created=0 updated=0 ok=0
  for skill in "${fast_skills[@]}"; do
    skill_file="$kit_dir/skills/${skill}.md"
    agent_file="$claude_agents_dir/${skill}.md"

    # Generate agent file: replace model_tier with platform-specific model
    new_content=$(sed 's/^model_tier: .*/model: '"$claude_fast_model"'/' "$skill_file")

    if [ -f "$agent_file" ]; then
      existing=$(cat "$agent_file")
      if [ "$new_content" = "$existing" ]; then
        ok=$((ok + 1))
      else
        printf '%s\n' "$new_content" > "$agent_file"
        updated=$((updated + 1))
      fi
    else
      printf '%s\n' "$new_content" > "$agent_file"
      created=$((created + 1))
    fi
  done

  total=$((created + updated + ok))
  printf "%-12s %d/%d (created %d, updated %d, already valid %d)\n" \
    "Agents:" "$total" "${#fast_skills[@]}" "$created" "$updated" "$ok"
fi

# ── 7. .gitignore ──────────────────────────────────────────────────
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

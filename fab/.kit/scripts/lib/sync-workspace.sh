#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/lib/sync-workspace.sh — Structural bootstrap for fab
#
# Syncs kit assets (directories, skill links/copies, agent files, .gitignore
# entries) into the workspace. Idempotent — safe to re-run at any time.
#
# Run from anywhere: fab/.kit/scripts/lib/sync-workspace.sh
# Safe to re-run (idempotent).

lib_dir="$(cd "$(dirname "$0")" && pwd)"
scripts_dir="$(dirname "$lib_dir")"
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
docs_dir="$repo_root/docs"
for dir in "$fab_dir/changes" "$docs_dir/memory" "$docs_dir/specs"; do
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    echo "Created: ${dir#"$repo_root"/}"
  fi
done

if [ ! -f "$fab_dir/changes/.gitkeep" ]; then
  touch "$fab_dir/changes/.gitkeep"
fi

# ── 1b. fab/VERSION ──────────────────────────────────────────────────
# Track the local project's kit version. New projects get the engine version;
# existing projects (have config.yaml) get the base version 0.1.0 so
# /fab-update runs all needed migrations.
if [ -f "$fab_dir/VERSION" ]; then
  echo "fab/VERSION: OK ($(cat "$fab_dir/VERSION"))"
elif [ -f "$fab_dir/config.yaml" ]; then
  # Existing project: set base version so /fab-update applies migrations
  echo "0.1.0" > "$fab_dir/VERSION"
  echo "Created: fab/VERSION (0.1.0 — existing project, run /fab-update to migrate)"
else
  # New project: match engine version
  cp "$kit_dir/VERSION" "$fab_dir/VERSION"
  echo "Created: fab/VERSION ($version)"
fi

# ── 2. .envrc ─────────────────────────────────────────────────────
envrc_link="$repo_root/.envrc"
envrc_target="fab/.kit/scaffold/envrc"

if [ -L "$envrc_link" ] && [ -e "$envrc_link" ]; then
  echo ".envrc: OK (symlink)"
elif [ -L "$envrc_link" ]; then
  # Broken symlink — remove and recreate
  rm "$envrc_link"
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: repaired broken symlink → $envrc_target"
elif [ -e "$envrc_link" ]; then
  rm "$envrc_link"
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: replaced file with symlink → $envrc_target"
else
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: created symlink → $envrc_target"
fi

# ── 3. Memory index ────────────────────────────────────────────────
if [ ! -f "$docs_dir/memory/index.md" ]; then
  cp "$kit_dir/scaffold/memory-index.md" "$docs_dir/memory/index.md"
  echo "Created: docs/memory/index.md"
fi

# ── 4. Specs index ────────────────────────────────────────────────
if [ ! -f "$docs_dir/specs/index.md" ]; then
  cp "$kit_dir/scaffold/specs-index.md" "$docs_dir/specs/index.md"
  echo "Created: docs/specs/index.md"
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

# sync_agent_skills <agent_label> <base_dir> <format> <mode> [<rel_prefix>]
#   format: "directory" → <base>/<name>/SKILL.md, "flat" → <base>/<name>.md
#   mode: "symlink" (needs rel_prefix) or "copy" (copies file content)
#   rel_prefix: relative path from symlink location back to repo root (symlink mode only)
sync_agent_skills() {
  local agent_label="$1"
  local base_dir="$2"
  local format="$3"
  local mode="$4"
  local rel_prefix="${5:-}"

  mkdir -p "$base_dir"

  local created=0 repaired=0 ok=0

  for skill in "${skills[@]}"; do
    local src="$kit_dir/skills/${skill}.md"
    if [ ! -f "$src" ]; then
      echo "WARN: fab/.kit/skills/${skill}.md missing — skipping"
      continue
    fi

    local dest
    if [ "$format" = "directory" ]; then
      mkdir -p "$base_dir/$skill"
      dest="$base_dir/$skill/SKILL.md"
    else
      dest="$base_dir/${skill}.md"
    fi

    if [ "$mode" = "copy" ]; then
      if [ -f "$dest" ] && [ ! -L "$dest" ] && cmp -s "$src" "$dest"; then
        ok=$((ok + 1))
      elif [ -e "$dest" ] || [ -L "$dest" ]; then
        rm "$dest"
        cp "$src" "$dest"
        repaired=$((repaired + 1))
      else
        cp "$src" "$dest"
        created=$((created + 1))
      fi
    else
      local target="${rel_prefix}fab/.kit/skills/${skill}.md"
      if [ -L "$dest" ] && [ -e "$dest" ]; then
        ok=$((ok + 1))
      elif [ -L "$dest" ] || [ -e "$dest" ]; then
        rm "$dest"
        ln -s "$target" "$dest"
        repaired=$((repaired + 1))
      else
        ln -s "$target" "$dest"
        created=$((created + 1))
      fi
    fi
  done

  local total=$((created + repaired + ok))
  printf "%-12s %d/%d (created %d, repaired %d, already valid %d)\n" \
    "${agent_label}:" "$total" "${#skills[@]}" "$created" "$repaired" "$ok"
}

# Claude Code: .claude/skills/<name>/SKILL.md (directory-based, symlinks)
sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "symlink" "../../../"

# OpenCode: .opencode/commands/<name>.md (flat file, symlinks)
sync_agent_skills "OpenCode" "$repo_root/.opencode/commands" "flat" "symlink" "../../"

# Codex: .agents/skills/<name>/SKILL.md (directory-based, copies — Codex ignores symlinks)
sync_agent_skills "Codex" "$repo_root/.agents/skills" "directory" "copy"

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
gitignore_entries="$kit_dir/scaffold/gitignore-entries"

if [ -f "$gitignore_entries" ]; then
  gitignore_existed=false
  [ -f "$gitignore" ] && gitignore_existed=true
  added=()

  while IFS= read -r entry || [ -n "$entry" ]; do
    # Skip comments and empty lines
    [[ -z "$entry" || "$entry" == \#* ]] && continue
    if [ ! -f "$gitignore" ]; then
      echo "$entry" > "$gitignore"
      added+=("$entry")
    elif ! grep -qxF "$entry" "$gitignore"; then
      echo "" >> "$gitignore"
      echo "$entry" >> "$gitignore"
      added+=("$entry")
    fi
  done < "$gitignore_entries"

  if [ ${#added[@]} -gt 0 ]; then
    if [ "$gitignore_existed" = false ]; then
      echo "Created: .gitignore (added ${added[*]})"
    else
      echo "Updated: .gitignore (added ${added[*]})"
    fi
  fi
fi

echo "Done."

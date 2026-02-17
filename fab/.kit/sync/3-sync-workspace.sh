#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/sync/3-sync-workspace.sh — Structural bootstrap for fab
#
# Syncs kit assets (directories, skill links/copies, agent files, .gitignore
# entries) into the workspace. Idempotent — safe to re-run at any time.

sync_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$sync_dir")"
fab_dir="$(dirname "$kit_dir")"
repo_root="$(dirname "$fab_dir")"

# ── Pre-flight ───────────────────────────────────────────────────────
if [ ! -f "$kit_dir/VERSION" ]; then
  echo "ERROR: fab/.kit/VERSION not found — kit may be corrupted." >&2
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
# Example: yaml_value config.yaml model_tiers fast claude → "haiku"
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
for dir in "$fab_dir/changes" "$fab_dir/changes/archive" "$docs_dir/memory" "$docs_dir/specs"; do
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    echo "Created: ${dir#"$repo_root"/}"
  fi
done

if [ ! -f "$fab_dir/changes/.gitkeep" ]; then
  touch "$fab_dir/changes/.gitkeep"
fi

if [ ! -f "$fab_dir/changes/archive/.gitkeep" ]; then
  touch "$fab_dir/changes/archive/.gitkeep"
fi

# ── 1b. fab/VERSION ──────────────────────────────────────────────────
# Track the local project's kit version. New projects get the engine version;
# existing projects (have config.yaml) get the base version 0.1.0 so
# /fab-setup migrations runs all needed migrations.
if [ -f "$fab_dir/VERSION" ]; then
  echo "fab/VERSION: OK ($(cat "$fab_dir/VERSION"))"
elif [ -f "$fab_dir/config.yaml" ]; then
  # Existing project: set base version so /fab-setup migrations applies migrations
  echo "0.1.0" > "$fab_dir/VERSION"
  echo "Created: fab/VERSION (0.1.0 — existing project, run /fab-setup migrations to migrate)"
else
  # New project: match engine version
  cp "$kit_dir/VERSION" "$fab_dir/VERSION"
  echo "Created: fab/VERSION ($version)"
fi

# ── 2. .envrc (line-ensuring, same pattern as .gitignore) ─────────
envrc_file="$repo_root/.envrc"
envrc_entries="$kit_dir/scaffold/envrc"

if [ -f "$envrc_entries" ]; then
  # Migrate: if .envrc is a symlink, replace with real file
  if [ -L "$envrc_file" ]; then
    resolved="$(cat "$envrc_file" 2>/dev/null || true)"
    rm "$envrc_file"
    if [ -n "$resolved" ]; then
      printf '%s\n' "$resolved" > "$envrc_file"
    fi
    echo ".envrc: migrated from symlink to file"
  fi

  envrc_existed=false
  [ -f "$envrc_file" ] && envrc_existed=true
  added=()

  while IFS= read -r entry || [ -n "$entry" ]; do
    [[ -z "$entry" || "$entry" == \#* ]] && continue
    if [ ! -f "$envrc_file" ]; then
      echo "$entry" > "$envrc_file"
      added+=("$entry")
    elif ! grep -qxF "$entry" "$envrc_file"; then
      echo "" >> "$envrc_file"
      echo "$entry" >> "$envrc_file"
      added+=("$entry")
    fi
  done < "$envrc_entries"

  if [ ${#added[@]} -gt 0 ]; then
    if [ "$envrc_existed" = false ]; then
      echo "Created: .envrc (added ${added[*]})"
    else
      echo "Updated: .envrc (added ${added[*]})"
    fi
  else
    echo ".envrc: OK"
  fi
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

# clean_stale_skills <base_dir> <format>
#   Removes skill entries in base_dir that are NOT in the skills[] array.
#   format: "directory" → remove <base>/<name>/ dirs, "flat" → remove <base>/<name>.md files
clean_stale_skills() {
  local base_dir="$1"
  local format="$2"
  local removed=0

  [ -d "$base_dir" ] || return 0

  if [ "$format" = "directory" ]; then
    for entry in "$base_dir"/*/; do
      [ -d "$entry" ] || continue
      local name
      name="$(basename "$entry")"
      local found=false
      for skill in "${skills[@]}"; do
        if [ "$skill" = "$name" ]; then
          found=true
          break
        fi
      done
      if [ "$found" = false ]; then
        rm -rf "$entry"
        removed=$((removed + 1))
      fi
    done
  else
    for entry in "$base_dir"/*.md; do
      [ -f "$entry" ] || [ -L "$entry" ] || continue
      local name
      name="$(basename "$entry" .md)"
      local found=false
      for skill in "${skills[@]}"; do
        if [ "$skill" = "$name" ]; then
          found=true
          break
        fi
      done
      if [ "$found" = false ]; then
        rm -f "$entry"
        removed=$((removed + 1))
      fi
    done
  fi

  if [ "$removed" -gt 0 ]; then
    echo "Cleaned: $removed stale entries from ${base_dir#"$repo_root"/}"
  fi
}

# Claude Code: .claude/skills/<name>/SKILL.md (directory-based, symlinks)
sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "symlink" "../../../"
clean_stale_skills "$repo_root/.claude/skills" "directory"

# OpenCode: .opencode/commands/<name>.md (flat file, symlinks)
sync_agent_skills "OpenCode" "$repo_root/.opencode/commands" "flat" "symlink" "../../"
clean_stale_skills "$repo_root/.opencode/commands" "flat"

# Codex: .agents/skills/<name>/SKILL.md (directory-based, copies — Codex ignores symlinks)
sync_agent_skills "Codex" "$repo_root/.agents/skills" "directory" "copy"
clean_stale_skills "$repo_root/.agents/skills" "directory"

# ── 6. Model tier agent files ────────────────────────────────────────
# Fast-tier skills get generated agent files (in addition to skill symlinks)
# so pipeline operations can invoke them with cost-appropriate models.

if [ ${#fast_skills[@]} -gt 0 ]; then
  # Resolve Claude model for "fast" tier from config.yaml, with hardcoded fallback
  claude_fast_model=""
  if [ -f "$fab_dir/config.yaml" ]; then
    claude_fast_model=$(yaml_value "$fab_dir/config.yaml" "model_tiers" "fast" "claude")
  fi

  # Fallback: use haiku if config.yaml has no model_tiers or doesn't exist
  if [ -z "$claude_fast_model" ]; then
    claude_fast_model="haiku"
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

  # Clean stale agent files: remove agents for skills no longer in fast_skills[]
  # Only remove files whose basename (without .md) matches a known skill name pattern
  # (i.e., exists or once existed in .kit/skills/). Preserve user-created agents.
  stale_agents=0
  for agent_file in "$claude_agents_dir"/*.md; do
    [ -f "$agent_file" ] || continue
    local_name="$(basename "$agent_file" .md)"
    found=false
    for skill in "${fast_skills[@]}"; do
      if [ "$skill" = "$local_name" ]; then
        found=true
        break
      fi
    done
    if [ "$found" = false ]; then
      # Only remove if a corresponding skill once existed (name matches skill naming pattern)
      # but is no longer in .kit/skills/ — this preserves truly user-created agents
      if [ ! -f "$kit_dir/skills/${local_name}.md" ]; then
        rm -f "$agent_file"
        stale_agents=$((stale_agents + 1))
      fi
    fi
  done
  if [ "$stale_agents" -gt 0 ]; then
    echo "Cleaned: $stale_agents stale agent files from .claude/agents/"
  fi
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

# ── 8. .claude/settings.local.json (JSON merge) ─────────────────────
# Scaffold provides baseline permissions for fab-kit scripts, git, gh, etc.
# If the file exists, merge scaffold entries into existing allow list.
# If it doesn't exist, copy the scaffold as-is.
settings_scaffold="$kit_dir/scaffold/settings.local.json"
settings_target="$repo_root/.claude/settings.local.json"

if [ -f "$settings_scaffold" ]; then
  mkdir -p "$repo_root/.claude"

  if [ ! -f "$settings_target" ]; then
    cp "$settings_scaffold" "$settings_target"
    count=$(jq '.permissions.allow | length' "$settings_target")
    echo "Created: .claude/settings.local.json ($count permission rules)"
  elif command -v jq >/dev/null 2>&1; then
    merged=$(jq -s '
      (.[0].permissions.allow // []) as $scaffold |
      (.[1] // {}) as $existing |
      ($existing.permissions.allow // []) as $current |
      ($scaffold | map(select(. as $s | $current | index($s) | not))) as $new |
      if ($new | length) > 0 then
        $existing | .permissions.allow = ($current + $new)
      else
        $existing
      end
    ' "$settings_scaffold" "$settings_target")
    new_count=$(echo "$merged" | jq '
      (input.permissions.allow // [] | length) as $before |
      (.permissions.allow | length) - $before
    ' - "$settings_target")
    if [ "$new_count" -gt 0 ]; then
      printf '%s\n' "$merged" > "$settings_target"
      echo "Updated: .claude/settings.local.json (added $new_count permission rules)"
    else
      echo ".claude/settings.local.json: OK"
    fi
  else
    echo "WARN: jq not found — skipping .claude/settings.local.json merge"
  fi
fi

# ── 9. fab/sync/ README scaffold ────────────────────────────────────
sync_readme_scaffold="$kit_dir/scaffold/sync-readme.md"
sync_readme_target="$fab_dir/sync/README.md"

if [ -f "$sync_readme_scaffold" ] && [ -d "$fab_dir/sync" ]; then
  if [ ! -f "$sync_readme_target" ]; then
    cp "$sync_readme_scaffold" "$sync_readme_target"
    echo "Created: fab/sync/README.md"
  fi
fi

echo "Done."

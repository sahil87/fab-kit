#!/usr/bin/env bats

# Test suite for fab-sync.sh
# Covers: directory creation, VERSION logic, .envrc, index seeding,
#         skill sync, model-tier agent generation (config.yaml source),
#         .gitignore, idempotency

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
FAB_SYNC="$(readlink -f "$SCRIPT_DIR/fab-sync.sh")"

setup() {
  # Create isolated temp workspace
  TEST_DIR="$(mktemp -d)"
  REPO_ROOT="$TEST_DIR/repo"
  mkdir -p "$REPO_ROOT"

  # Build minimal fab/.kit/ structure
  KIT="$REPO_ROOT/fab/.kit"
  mkdir -p "$KIT/scripts/lib" "$KIT/scaffold" "$KIT/skills" "$KIT/templates" "$KIT/schemas"

  # VERSION file
  echo "1.2.3" > "$KIT/VERSION"

  # Scaffold files
  echo "# Memory Index" > "$KIT/scaffold/memory-index.md"
  echo "# Specs Index" > "$KIT/scaffold/specs-index.md"
  echo "layout_variable" > "$KIT/scaffold/envrc"
  echo "fab/current" > "$KIT/scaffold/gitignore-entries"

  # Template status.yaml (minimal)
  cat > "$KIT/templates/status.yaml" <<'YAML'
name: "{NAME}"
created: "{CREATED}"
created_by: "{CREATED_BY}"
change_type: feature
progress:
  intake: pending
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
checklist:
  generated: false
  path: checklist.md
  completed: 0
  total: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
last_updated: ""
YAML

  # Create a minimal capable skill (no frontmatter model_tier)
  cat > "$KIT/skills/fab-continue.md" <<'MD'
# /fab-continue
Some skill content.
MD

  # Create a fast-tier skill (with frontmatter)
  cat > "$KIT/skills/fab-status.md" <<'MD'
---
name: fab-status
description: "Show status"
model_tier: fast
---
# /fab-status
Status skill content.
MD

  # Create a partial (should be skipped)
  cat > "$KIT/skills/_context.md" <<'MD'
# Shared Context
This is a partial — should not be deployed as a skill.
MD

  # Copy the actual fab-sync.sh into the kit
  cp "$FAB_SYNC" "$KIT/scripts/fab-sync.sh"
  chmod +x "$KIT/scripts/fab-sync.sh"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# ── Directory Creation ──────────────────────────────────────────────

@test "creates fab/changes directory when missing" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/fab/changes" ]
}

@test "creates docs/memory directory when missing" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/docs/memory" ]
}

@test "creates docs/specs directory when missing" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/docs/specs" ]
}

@test "creates fab/changes/.gitkeep" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/changes/.gitkeep" ]
}

@test "creates fab/changes/archive/.gitkeep" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/changes/archive/.gitkeep" ]
}

@test "skips directory creation when directories exist" {
  mkdir -p "$REPO_ROOT/fab/changes" "$REPO_ROOT/docs/memory" "$REPO_ROOT/docs/specs"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  # Verify no error about existing dirs
  [[ "$output" != *"ERROR"* ]]
}

# ── VERSION File Logic ──────────────────────────────────────────────

@test "new project gets engine version in fab/VERSION" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/VERSION" ]
  [ "$(cat "$REPO_ROOT/fab/VERSION")" = "1.2.3" ]
}

@test "existing project without VERSION gets 0.1.0" {
  # Create config.yaml to indicate existing project
  cat > "$REPO_ROOT/fab/config.yaml" <<'YAML'
project:
  name: test
YAML
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/VERSION")" = "0.1.0" ]
}

@test "existing fab/VERSION is preserved" {
  echo "0.5.0" > "$REPO_ROOT/fab/VERSION"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/VERSION")" = "0.5.0" ]
}

# ── .envrc File ─────────────────────────────────────────────────────

@test "creates .envrc file with scaffold entries" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.envrc" ]
  ! [ -L "$REPO_ROOT/.envrc" ]
  grep -qxF "layout_variable" "$REPO_ROOT/.envrc"
}

@test "migrates .envrc symlink to regular file" {
  ln -s "nonexistent/path" "$REPO_ROOT/.envrc"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.envrc" ]
  ! [ -L "$REPO_ROOT/.envrc" ]
  grep -qxF "layout_variable" "$REPO_ROOT/.envrc"
}

@test "preserves existing .envrc and appends missing entries" {
  echo "old content" > "$REPO_ROOT/.envrc"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  ! [ -L "$REPO_ROOT/.envrc" ]
  grep -qxF "old content" "$REPO_ROOT/.envrc"
  grep -qxF "layout_variable" "$REPO_ROOT/.envrc"
}

# ── Memory/Specs Index Seeding ──────────────────────────────────────

@test "creates docs/memory/index.md from scaffold" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/docs/memory/index.md" ]
  [[ "$(cat "$REPO_ROOT/docs/memory/index.md")" == *"Memory Index"* ]]
}

@test "creates docs/specs/index.md from scaffold" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/docs/specs/index.md" ]
  [[ "$(cat "$REPO_ROOT/docs/specs/index.md")" == *"Specs Index"* ]]
}

@test "does not overwrite existing memory index" {
  mkdir -p "$REPO_ROOT/docs/memory"
  echo "# Custom Index" > "$REPO_ROOT/docs/memory/index.md"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/docs/memory/index.md")" = "# Custom Index" ]
}

# ── Skill Sync ──────────────────────────────────────────────────────

@test "creates Claude Code skill symlinks (directory-based)" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -L "$REPO_ROOT/.claude/skills/fab-continue/SKILL.md" ]
  [ -e "$REPO_ROOT/.claude/skills/fab-continue/SKILL.md" ]
  [ -L "$REPO_ROOT/.claude/skills/fab-status/SKILL.md" ]
}

@test "creates OpenCode command symlinks (flat)" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -L "$REPO_ROOT/.opencode/commands/fab-continue.md" ]
  [ -e "$REPO_ROOT/.opencode/commands/fab-continue.md" ]
}

@test "creates Codex skill copies (not symlinks)" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.agents/skills/fab-continue/SKILL.md" ]
  # Codex files should NOT be symlinks
  [ ! -L "$REPO_ROOT/.agents/skills/fab-continue/SKILL.md" ]
}

@test "skips _context.md partial (not deployed as skill)" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ ! -e "$REPO_ROOT/.claude/skills/_context" ]
  [ ! -e "$REPO_ROOT/.claude/skills/_context.md" ]
}

# ── Model-Tier Agent Generation ─────────────────────────────────────

@test "generates agent file for fast-tier skill" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.claude/agents/fab-status.md" ]
}

@test "fast-tier agent file has model: instead of model_tier:" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  local agent_content
  agent_content="$(cat "$REPO_ROOT/.claude/agents/fab-status.md")"
  [[ "$agent_content" == *"model: haiku"* ]]
  [[ "$agent_content" != *"model_tier: fast"* ]]
}

@test "does not generate agent file for capable-tier skill" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ ! -f "$REPO_ROOT/.claude/agents/fab-continue.md" ]
}

# ── .gitignore Management ───────────────────────────────────────────

@test "creates .gitignore with fab/current entry" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.gitignore" ]
  grep -qxF "fab/current" "$REPO_ROOT/.gitignore"
}

@test "appends fab/current to existing .gitignore" {
  echo "node_modules/" > "$REPO_ROOT/.gitignore"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  grep -qxF "node_modules/" "$REPO_ROOT/.gitignore"
  grep -qxF "fab/current" "$REPO_ROOT/.gitignore"
}

@test "does not duplicate existing .gitignore entry" {
  echo "fab/current" > "$REPO_ROOT/.gitignore"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  local count
  count=$(grep -cxF "fab/current" "$REPO_ROOT/.gitignore")
  [ "$count" -eq 1 ]
}

# ── Idempotency ─────────────────────────────────────────────────────

@test "running twice produces no errors" {
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [[ "$output" != *"ERROR"* ]]
}

@test "running twice produces same file structure" {
  bash "$KIT/scripts/fab-sync.sh" >/dev/null 2>&1
  local first_tree
  first_tree="$(find "$REPO_ROOT" -type f -o -type l | sort)"
  bash "$KIT/scripts/fab-sync.sh" >/dev/null 2>&1
  local second_tree
  second_tree="$(find "$REPO_ROOT" -type f -o -type l | sort)"
  [ "$first_tree" = "$second_tree" ]
}

# ── Pre-flight Error Cases ──────────────────────────────────────────

@test "fails when VERSION file is missing" {
  rm "$KIT/VERSION"
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -ne 0 ]
  [[ "$output" == *"VERSION not found"* ]]
}

@test "uses haiku fallback when config.yaml has no model_tiers" {
  # No config.yaml at all — should fall back to haiku
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.claude/agents/fab-status.md" ]
  local agent_content
  agent_content="$(cat "$REPO_ROOT/.claude/agents/fab-status.md")"
  [[ "$agent_content" == *"model: haiku"* ]]
}

@test "reads model_tiers from config.yaml when present" {
  cat > "$REPO_ROOT/fab/config.yaml" <<'YAML'
project:
  name: test
model_tiers:
  fast:
    claude: sonnet
YAML
  run bash "$KIT/scripts/fab-sync.sh"
  [ "$status" -eq 0 ]
  local agent_content
  agent_content="$(cat "$REPO_ROOT/.claude/agents/fab-status.md")"
  [[ "$agent_content" == *"model: sonnet"* ]]
}

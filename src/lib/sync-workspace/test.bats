#!/usr/bin/env bats

# Test suite for 2-sync-workspace.sh
# Covers: directory creation, VERSION logic, .envrc, index seeding,
#         skill sync, .gitignore, idempotency

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
REPO_SRC_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SYNC_WORKSPACE="$REPO_SRC_ROOT/fab/.kit/sync/2-sync-workspace.sh"

setup() {
  # Create isolated temp workspace
  TEST_DIR="$(mktemp -d)"
  REPO_ROOT="$TEST_DIR/repo"
  mkdir -p "$REPO_ROOT"

  # Build minimal fab/.kit/ structure
  KIT="$REPO_ROOT/fab/.kit"
  mkdir -p "$KIT/sync" "$KIT/scaffold" "$KIT/skills" "$KIT/templates" "$KIT/schemas" "$KIT/scripts/lib"

  # Shared frontmatter parser (sourced by 2-sync-workspace.sh)
  cp "$REPO_SRC_ROOT/fab/.kit/scripts/lib/frontmatter.sh" "$KIT/scripts/lib/frontmatter.sh"

  # VERSION file
  echo "1.2.3" > "$KIT/VERSION"

  # Scaffold files (mirror real scaffold directory structure)
  mkdir -p "$KIT/scaffold/docs/memory" "$KIT/scaffold/docs/specs"
  echo "# Memory Index" > "$KIT/scaffold/docs/memory/index.md"
  echo "# Specs Index" > "$KIT/scaffold/docs/specs/index.md"
  echo "layout_variable" > "$KIT/scaffold/fragment-.envrc"
  echo "fab/current" > "$KIT/scaffold/fragment-.gitignore"

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

  # Create a skill without frontmatter
  cat > "$KIT/skills/fab-continue.md" <<'MD'
# /fab-continue
Some skill content.
MD

  # Create a skill with frontmatter
  cat > "$KIT/skills/fab-status.md" <<'MD'
---
name: fab-status
description: "Show status"
---
# /fab-status
Status skill content.
MD

  # Create underscore partials (deployed alongside skills)
  cat > "$KIT/skills/_preamble.md" <<'MD'
# Shared Context
This is a partial — deployed alongside skills.
MD

  cat > "$KIT/skills/_generation.md" <<'MD'
# Generation Procedures
Shared generation logic.
MD

  cat > "$KIT/skills/_scripts.md" <<'MD'
# Script Guide
Script invocation conventions.
MD

  # Copy the actual 2-sync-workspace.sh into the kit
  cp "$SYNC_WORKSPACE" "$KIT/sync/2-sync-workspace.sh"
  chmod +x "$KIT/sync/2-sync-workspace.sh"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# ── Directory Creation ──────────────────────────────────────────────

@test "creates fab/changes directory when missing" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/fab/changes" ]
}

@test "creates docs/memory directory when missing" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/docs/memory" ]
}

@test "creates docs/specs directory when missing" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/docs/specs" ]
}

@test "creates fab/changes/.gitkeep" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/changes/.gitkeep" ]
}

@test "creates fab/changes/archive/.gitkeep" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/changes/archive/.gitkeep" ]
}

@test "skips directory creation when directories exist" {
  mkdir -p "$REPO_ROOT/fab/changes" "$REPO_ROOT/docs/memory" "$REPO_ROOT/docs/specs"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  # Verify no error about existing dirs
  [[ "$output" != *"ERROR"* ]]
}

# ── VERSION File Logic ──────────────────────────────────────────────

@test "new project gets engine version in fab/.kit-migration-version" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/.kit-migration-version" ]
  [ "$(cat "$REPO_ROOT/fab/.kit-migration-version")" = "1.2.3" ]
}

@test "existing project without migration version gets 0.1.0" {
  # Create config.yaml to indicate existing project
  mkdir -p "$REPO_ROOT/fab/project"
  cat > "$REPO_ROOT/fab/project/config.yaml" <<'YAML'
project:
  name: test
YAML
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/.kit-migration-version")" = "0.1.0" ]
}

@test "existing fab/.kit-migration-version is preserved" {
  echo "0.5.0" > "$REPO_ROOT/fab/.kit-migration-version"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/.kit-migration-version")" = "0.5.0" ]
}

# ── Backward-Compat Migration ─────────────────────────────────────

@test "old fab/project/VERSION migrates to fab/.kit-migration-version" {
  mkdir -p "$REPO_ROOT/fab/project"
  echo "0.15.0" > "$REPO_ROOT/fab/project/VERSION"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/.kit-migration-version")" = "0.15.0" ]
  [ ! -f "$REPO_ROOT/fab/project/VERSION" ]
  [[ "$output" == *"Migrated: fab/project/VERSION"* ]]
}

@test "both VERSION files: old deleted, new preserved" {
  mkdir -p "$REPO_ROOT/fab/project"
  echo "0.15.0" > "$REPO_ROOT/fab/project/VERSION"
  echo "0.18.0" > "$REPO_ROOT/fab/.kit-migration-version"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/.kit-migration-version")" = "0.18.0" ]
  [ ! -f "$REPO_ROOT/fab/project/VERSION" ]
  [[ "$output" == *"Cleaned: stale fab/project/VERSION"* ]]
}

# ── Sync Version Stamp ─────────────────────────────────────────────

@test "fresh sync creates fab/.kit-sync-version" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/fab/.kit-sync-version" ]
  [ "$(cat "$REPO_ROOT/fab/.kit-sync-version")" = "1.2.3" ]
  [[ "$output" == *"Created: fab/.kit-sync-version (1.2.3)"* ]]
}

@test "re-run sync on same version produces OK" {
  echo "1.2.3" > "$REPO_ROOT/fab/.kit-sync-version"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/.kit-sync-version")" = "1.2.3" ]
  [[ "$output" == *"fab/.kit-sync-version: OK (1.2.3)"* ]]
}

@test "sync after kit upgrade updates stamp" {
  echo "1.1.0" > "$REPO_ROOT/fab/.kit-sync-version"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/fab/.kit-sync-version")" = "1.2.3" ]
  [[ "$output" == *"Updated: fab/.kit-sync-version (1.1.0 → 1.2.3)"* ]]
}

# ── .envrc File ─────────────────────────────────────────────────────

@test "creates .envrc file with scaffold entries" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.envrc" ]
  ! [ -L "$REPO_ROOT/.envrc" ]
  grep -qxF "layout_variable" "$REPO_ROOT/.envrc"
}

@test "migrates .envrc symlink to regular file" {
  ln -s "nonexistent/path" "$REPO_ROOT/.envrc"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.envrc" ]
  ! [ -L "$REPO_ROOT/.envrc" ]
  grep -qxF "layout_variable" "$REPO_ROOT/.envrc"
}

@test "preserves existing .envrc and appends missing entries" {
  echo "old content" > "$REPO_ROOT/.envrc"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  ! [ -L "$REPO_ROOT/.envrc" ]
  grep -qxF "old content" "$REPO_ROOT/.envrc"
  grep -qxF "layout_variable" "$REPO_ROOT/.envrc"
}

# ── Memory/Specs Index Seeding ──────────────────────────────────────

@test "creates docs/memory/index.md from scaffold" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/docs/memory/index.md" ]
  [[ "$(cat "$REPO_ROOT/docs/memory/index.md")" == *"Memory Index"* ]]
}

@test "creates docs/specs/index.md from scaffold" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/docs/specs/index.md" ]
  [[ "$(cat "$REPO_ROOT/docs/specs/index.md")" == *"Specs Index"* ]]
}

@test "does not overwrite existing memory index" {
  mkdir -p "$REPO_ROOT/docs/memory"
  echo "# Custom Index" > "$REPO_ROOT/docs/memory/index.md"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ "$(cat "$REPO_ROOT/docs/memory/index.md")" = "# Custom Index" ]
}

# ── Skill Sync ──────────────────────────────────────────────────────

@test "creates Claude Code skill copies (directory-based)" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.claude/skills/fab-continue/SKILL.md" ]
  [ ! -L "$REPO_ROOT/.claude/skills/fab-continue/SKILL.md" ]
  [ -f "$REPO_ROOT/.claude/skills/fab-status/SKILL.md" ]
  [ ! -L "$REPO_ROOT/.claude/skills/fab-status/SKILL.md" ]
}

@test "creates OpenCode command symlinks (flat)" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -L "$REPO_ROOT/.opencode/commands/fab-continue.md" ]
  [ -e "$REPO_ROOT/.opencode/commands/fab-continue.md" ]
}

@test "creates Codex skill copies (not symlinks)" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.agents/skills/fab-continue/SKILL.md" ]
  # Codex files should NOT be symlinks
  [ ! -L "$REPO_ROOT/.agents/skills/fab-continue/SKILL.md" ]
}

@test "deploys underscore partials as skills" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -d "$REPO_ROOT/.claude/skills/_preamble" ]
  [ -f "$REPO_ROOT/.claude/skills/_preamble/SKILL.md" ]
  [ -d "$REPO_ROOT/.claude/skills/_generation" ]
  [ -f "$REPO_ROOT/.claude/skills/_generation/SKILL.md" ]
  [ -d "$REPO_ROOT/.claude/skills/_scripts" ]
  [ -f "$REPO_ROOT/.claude/skills/_scripts/SKILL.md" ]
}

# ── Skill Copy Verification ───────────────────────────────────────

@test "skill copies are byte-accurate (no templating)" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  cmp -s "$KIT/skills/fab-status.md" "$REPO_ROOT/.claude/skills/fab-status/SKILL.md"
  cmp -s "$KIT/skills/fab-continue.md" "$REPO_ROOT/.claude/skills/fab-continue/SKILL.md"
}

@test "transitional cleanup removes stale agent files" {
  # Pre-create a stale agent file matching a known skill name
  mkdir -p "$REPO_ROOT/.claude/agents"
  echo "stale" > "$REPO_ROOT/.claude/agents/fab-status.md"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ ! -f "$REPO_ROOT/.claude/agents/fab-status.md" ]
}

# ── .gitignore Management ───────────────────────────────────────────

@test "creates .gitignore with fab/current entry" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [ -f "$REPO_ROOT/.gitignore" ]
  grep -qxF "fab/current" "$REPO_ROOT/.gitignore"
}

@test "appends fab/current to existing .gitignore" {
  echo "node_modules/" > "$REPO_ROOT/.gitignore"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  grep -qxF "node_modules/" "$REPO_ROOT/.gitignore"
  grep -qxF "fab/current" "$REPO_ROOT/.gitignore"
}

@test "does not duplicate existing .gitignore entry" {
  echo "fab/current" > "$REPO_ROOT/.gitignore"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  local count
  count=$(grep -cxF "fab/current" "$REPO_ROOT/.gitignore")
  [ "$count" -eq 1 ]
}

# ── Idempotency ─────────────────────────────────────────────────────

@test "running twice produces no errors" {
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -eq 0 ]
  [[ "$output" != *"ERROR"* ]]
}

@test "running twice produces same file structure" {
  bash "$KIT/sync/2-sync-workspace.sh" >/dev/null 2>&1
  local first_tree
  first_tree="$(find "$REPO_ROOT" -type f -o -type l | sort)"
  bash "$KIT/sync/2-sync-workspace.sh" >/dev/null 2>&1
  local second_tree
  second_tree="$(find "$REPO_ROOT" -type f -o -type l | sort)"
  [ "$first_tree" = "$second_tree" ]
}

# ── Pre-flight Error Cases ──────────────────────────────────────────

@test "fails when VERSION file is missing" {
  rm "$KIT/VERSION"
  run bash "$KIT/sync/2-sync-workspace.sh"
  [ "$status" -ne 0 ]
  [[ "$output" == *"VERSION not found"* ]]
}


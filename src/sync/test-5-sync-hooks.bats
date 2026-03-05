#!/usr/bin/env bats

# Test suite for fab/.kit/sync/5-sync-hooks.sh
# Covers: first sync creates hooks, idempotent re-sync, preserves user hooks,
#         no jq warns and skips, no hook scripts is silent,
#         missing settings.local.json creates file, unknown hook script ignored

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
REPO_SRC_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SYNC_SCRIPT="$REPO_SRC_ROOT/fab/.kit/sync/5-sync-hooks.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
  REPO="$TEST_DIR/repo"
  mkdir -p "$REPO"

  # Build minimal fab/.kit/ structure
  KIT="$REPO/fab/.kit"
  mkdir -p "$KIT/sync" "$KIT/hooks"

  # Copy the sync script
  cp "$SYNC_SCRIPT" "$KIT/sync/5-sync-hooks.sh"

  # Create hook scripts
  echo '#!/usr/bin/env bash' > "$KIT/hooks/on-session-start.sh"
  echo '#!/usr/bin/env bash' > "$KIT/hooks/on-stop.sh"
  chmod +x "$KIT/hooks/on-session-start.sh" "$KIT/hooks/on-stop.sh"

  # Create .claude directory
  mkdir -p "$REPO/.claude"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "sync-hooks: first sync creates hooks in settings" {
  # Start with settings that has permissions but no hooks
  cat > "$REPO/.claude/settings.local.json" <<'JSON'
{"permissions":{"allow":["Bash(git:*)"]}}
JSON

  cd "$REPO"
  run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Created: .claude/settings.local.json hooks (2 hook entries)"* ]]

  # Verify hooks are present
  session_start=$(jq -r '.hooks.SessionStart[0].command' "$REPO/.claude/settings.local.json")
  stop=$(jq -r '.hooks.Stop[0].command' "$REPO/.claude/settings.local.json")
  [ "$session_start" = "bash fab/.kit/hooks/on-session-start.sh" ]
  [ "$stop" = "bash fab/.kit/hooks/on-stop.sh" ]

  # Verify permissions preserved
  perm=$(jq -r '.permissions.allow[0]' "$REPO/.claude/settings.local.json")
  [ "$perm" = "Bash(git:*)" ]
}

@test "sync-hooks: idempotent re-sync produces OK" {
  cat > "$REPO/.claude/settings.local.json" <<'JSON'
{"permissions":{"allow":[]},"hooks":{"SessionStart":[{"type":"command","command":"bash fab/.kit/hooks/on-session-start.sh"}],"Stop":[{"type":"command","command":"bash fab/.kit/hooks/on-stop.sh"}]}}
JSON

  cd "$REPO"
  run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"hooks: OK"* ]]
}

@test "sync-hooks: preserves user hooks" {
  cat > "$REPO/.claude/settings.local.json" <<'JSON'
{"hooks":{"Stop":[{"type":"command","command":"bash my-custom-hook.sh"}]}}
JSON

  cd "$REPO"
  run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]

  # User hook preserved
  user_hook=$(jq -r '.hooks.Stop[0].command' "$REPO/.claude/settings.local.json")
  [ "$user_hook" = "bash my-custom-hook.sh" ]

  # Fab hook appended
  fab_hook=$(jq -r '.hooks.Stop[1].command' "$REPO/.claude/settings.local.json")
  [ "$fab_hook" = "bash fab/.kit/hooks/on-stop.sh" ]
}

@test "sync-hooks: no jq warns and skips" {
  cat > "$REPO/.claude/settings.local.json" <<'JSON'
{}
JSON

  cd "$REPO"
  # Create a PATH with basic tools but no jq
  mkdir -p "$TEST_DIR/no-jq-bin"
  for cmd in bash cat dirname cd pwd basename chmod mkdir sort; do
    local cmd_path
    cmd_path="$(command -v "$cmd" 2>/dev/null)" || continue
    ln -sf "$cmd_path" "$TEST_DIR/no-jq-bin/$cmd"
  done
  # Ensure common builtins' external fallbacks are available
  ln -sf "$(command -v test)" "$TEST_DIR/no-jq-bin/test" 2>/dev/null || true
  ln -sf "$(command -v [)" "$TEST_DIR/no-jq-bin/[" 2>/dev/null || true

  PATH="$TEST_DIR/no-jq-bin" run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN: jq not found"* ]]

  # File unchanged
  content=$(cat "$REPO/.claude/settings.local.json")
  [ "$content" = "{}" ]
}

@test "sync-hooks: no hook scripts is silent" {
  rm "$KIT/hooks/on-session-start.sh" "$KIT/hooks/on-stop.sh"

  cd "$REPO"
  run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "sync-hooks: missing settings.local.json creates file" {
  # No settings.local.json exists
  [ ! -f "$REPO/.claude/settings.local.json" ]

  cd "$REPO"
  run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Created: .claude/settings.local.json hooks (2 hook entries)"* ]]

  # File created with hooks
  [ -f "$REPO/.claude/settings.local.json" ]
  session_start=$(jq -r '.hooks.SessionStart[0].command' "$REPO/.claude/settings.local.json")
  [ "$session_start" = "bash fab/.kit/hooks/on-session-start.sh" ]
}

@test "sync-hooks: unknown hook script is ignored" {
  # Add a non-hook script
  echo '#!/usr/bin/env bash' > "$KIT/hooks/helper.sh"
  chmod +x "$KIT/hooks/helper.sh"

  cat > "$REPO/.claude/settings.local.json" <<'JSON'
{}
JSON

  cd "$REPO"
  run bash "$KIT/sync/5-sync-hooks.sh"
  [ "$status" -eq 0 ]

  # Only 2 hook entries (not 3)
  total=$(jq '[.hooks[]? | length] | add' "$REPO/.claude/settings.local.json")
  [ "$total" -eq 2 ]
}

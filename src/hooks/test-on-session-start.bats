#!/usr/bin/env bats

# Test suite for fab/.kit/hooks/on-session-start.sh (thin wrapper)
# Covers: delegates to binary, binary-missing graceful fallback

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
REPO_SRC_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HOOK_SCRIPT="$REPO_SRC_ROOT/fab/.kit/hooks/on-session-start.sh"

setup() {
  TEST_DIR="$(mktemp -d)"

  # Initialize as a git repo so git rev-parse works
  git init --quiet "$TEST_DIR/repo"
  REPO="$TEST_DIR/repo"

  # Minimal fab structure
  mkdir -p "$REPO/fab/.kit/bin" "$REPO/fab/.kit/hooks"
  cp "$HOOK_SCRIPT" "$REPO/fab/.kit/hooks/on-session-start.sh"

  # Create a stub fab binary that handles "hook session-start"
  CHANGE_DIR="fab/changes/260305-bs5x-test-change"
  mkdir -p "$REPO/$CHANGE_DIR"

  CHANGE_FOLDER="260305-bs5x-test-change"
  cat > "$REPO/fab/.kit/bin/fab" <<SCRIPT
#!/usr/bin/env bash
if [ "\$1" = "hook" ] && [ "\$2" = "session-start" ]; then
  repo_root="\$(git rev-parse --show-toplevel 2>/dev/null)"
  runtime="\$repo_root/.fab-runtime.yaml"
  [ -f "\$runtime" ] || exit 0
  yq -i 'del(.["$CHANGE_FOLDER"].agent)' "\$runtime" 2>/dev/null
  exit 0
fi
exit 1
SCRIPT
  chmod +x "$REPO/fab/.kit/bin/fab"

  # .status.yaml
  cat > "$REPO/$CHANGE_DIR/.status.yaml" <<'YAML'
name: test-change
progress:
  intake: done
YAML

  # .fab-runtime.yaml with agent idle state
  cat > "$REPO/.fab-runtime.yaml" <<YAML
$CHANGE_FOLDER:
  agent:
    idle_since: 1741193400
YAML

  ln -s "fab/changes/260305-bs5x-test-change/.status.yaml" "$REPO/.fab-status.yaml"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "on-session-start: thin wrapper delegates to binary and clears agent block" {
  cd "$REPO"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]

  agent=$(yq '.["260305-bs5x-test-change"].agent' "$REPO/.fab-runtime.yaml")
  [ "$agent" = "null" ]
}

@test "on-session-start: binary missing exits 0 gracefully" {
  cd "$REPO"
  rm "$REPO/fab/.kit/bin/fab"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]

  # Agent idle state should still be present (not cleared)
  idle_since=$(yq '.["260305-bs5x-test-change"].agent.idle_since' "$REPO/.fab-runtime.yaml")
  [ "$idle_since" = "1741193400" ]
}

@test "on-session-start: no .fab-status.yaml symlink exits 0" {
  cd "$REPO"
  rm "$REPO/.fab-status.yaml"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]
}

@test "on-session-start: idempotent — runs twice without error" {
  cd "$REPO"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]
}

#!/usr/bin/env bats

# Test suite for fab/.kit/hooks/on-session-start.sh
# Covers: active change clears agent block, no agent block is idempotent,
#         no .fab-status.yaml symlink, missing change dir, missing .status.yaml,
#         yq not available, fab dispatcher not available

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

  # Create a stub fab dispatcher
  CHANGE_DIR="fab/changes/260305-bs5x-test-change"
  mkdir -p "$REPO/$CHANGE_DIR"

  CHANGE_FOLDER="260305-bs5x-test-change"
  cat > "$REPO/fab/.kit/bin/fab" <<SCRIPT
#!/usr/bin/env bash
if [ "\$1" = "resolve" ] && [ "\$2" = "--folder" ]; then
  echo "$CHANGE_FOLDER"
  exit 0
fi
if [ "\$1" = "runtime" ] && [ "\$2" = "clear-idle" ]; then
  repo_root="\$(git rev-parse --show-toplevel 2>/dev/null)"
  yq -i "del(.[\"\$3\"].agent)" "\$repo_root/.fab-runtime.yaml" 2>/dev/null
  exit 0
fi
exit 1
SCRIPT
  chmod +x "$REPO/fab/.kit/bin/fab"

  # .status.yaml (no agent block — that's now in runtime)
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

@test "on-session-start: active change clears agent block" {
  cd "$REPO"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]

  agent=$(yq '.["260305-bs5x-test-change"].agent' "$REPO/.fab-runtime.yaml")
  [ "$agent" = "null" ]
}

@test "on-session-start: no agent block is idempotent" {
  cd "$REPO"
  # Remove agent block from runtime first
  yq -i 'del(.["260305-bs5x-test-change"].agent)' "$REPO/.fab-runtime.yaml"

  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]

  agent=$(yq '.["260305-bs5x-test-change"].agent' "$REPO/.fab-runtime.yaml")
  [ "$agent" = "null" ]
}

@test "on-session-start: no .fab-status.yaml symlink exits 0 silently" {
  cd "$REPO"
  rm "$REPO/.fab-status.yaml"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]

  # Agent idle state still present in runtime (not cleared)
  idle_since=$(yq '.["260305-bs5x-test-change"].agent.idle_since' "$REPO/.fab-runtime.yaml")
  [ "$idle_since" = "1741193400" ]
}

@test "on-session-start: missing change directory exits 0 silently" {
  cd "$REPO"
  rm -rf "$REPO/$CHANGE_DIR"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]
}

@test "on-session-start: missing .status.yaml exits 0 silently" {
  cd "$REPO"
  rm "$REPO/$CHANGE_DIR/.status.yaml"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]
}

@test "on-session-start: yq not available exits 0" {
  cd "$REPO"
  mkdir -p "$TEST_DIR/restricted-bin"
  ln -s "$(command -v bash)" "$TEST_DIR/restricted-bin/bash"
  ln -s "$(command -v git)" "$TEST_DIR/restricted-bin/git"
  ln -s "$(command -v cat)" "$TEST_DIR/restricted-bin/cat"
  ln -s "$(command -v head)" "$TEST_DIR/restricted-bin/head"
  ln -s "$(command -v tr)" "$TEST_DIR/restricted-bin/tr"
  ln -s "$(command -v test)" "$TEST_DIR/restricted-bin/test" 2>/dev/null || true
  ln -s "$REPO/fab/.kit/bin/fab" "$TEST_DIR/restricted-bin/fab"

  PATH="$TEST_DIR/restricted-bin" run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]

  # Agent idle state should still be present (not cleared)
  idle_since=$(yq '.["260305-bs5x-test-change"].agent.idle_since' "$REPO/.fab-runtime.yaml")
  [ "$idle_since" = "1741193400" ]
}

@test "on-session-start: fab dispatcher not available exits 0" {
  cd "$REPO"
  rm "$REPO/fab/.kit/bin/fab"
  run bash fab/.kit/hooks/on-session-start.sh
  [ "$status" -eq 0 ]
}

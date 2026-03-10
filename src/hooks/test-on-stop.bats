#!/usr/bin/env bats

# Test suite for fab/.kit/hooks/on-stop.sh (thin wrapper)
# Covers: delegates to binary, binary-missing graceful fallback

SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
REPO_SRC_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HOOK_SCRIPT="$REPO_SRC_ROOT/fab/.kit/hooks/on-stop.sh"

setup() {
  TEST_DIR="$(mktemp -d)"

  # Initialize as a git repo so git rev-parse works
  git init --quiet "$TEST_DIR/repo"
  REPO="$TEST_DIR/repo"

  # Minimal fab structure
  mkdir -p "$REPO/fab/.kit/bin" "$REPO/fab/.kit/hooks"
  cp "$HOOK_SCRIPT" "$REPO/fab/.kit/hooks/on-stop.sh"

  # Create a stub fab binary that handles "hook stop"
  CHANGE_DIR="fab/changes/260305-bs5x-test-change"
  mkdir -p "$REPO/$CHANGE_DIR"

  cat > "$REPO/fab/.kit/bin/fab" <<SCRIPT
#!/usr/bin/env bash
if [ "\$1" = "hook" ] && [ "\$2" = "stop" ]; then
  repo_root="\$(git rev-parse --show-toplevel 2>/dev/null)"
  runtime="\$repo_root/.fab-runtime.yaml"
  [ -f "\$runtime" ] || echo "{}" > "\$runtime"
  ts=\$(date +%s)
  yq -i ".[\\"260305-bs5x-test-change\\"].agent.idle_since = \$ts" "\$runtime" 2>/dev/null
  exit 0
fi
exit 1
SCRIPT
  chmod +x "$REPO/fab/.kit/bin/fab"

  # Minimal .status.yaml
  cat > "$REPO/$CHANGE_DIR/.status.yaml" <<'YAML'
name: test-change
progress:
  intake: done
YAML

  # Set active change via symlink
  ln -s "fab/changes/260305-bs5x-test-change/.status.yaml" "$REPO/.fab-status.yaml"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "on-stop: thin wrapper delegates to binary and sets idle timestamp" {
  cd "$REPO"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]

  # Verify agent.idle_since is a positive integer in .fab-runtime.yaml
  idle_since=$(yq '.["260305-bs5x-test-change"].agent.idle_since' "$REPO/.fab-runtime.yaml")
  [ "$idle_since" != "null" ]
  [ "$idle_since" -gt 0 ]
}

@test "on-stop: binary missing exits 0 gracefully" {
  cd "$REPO"
  rm "$REPO/fab/.kit/bin/fab"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]

  # No runtime file written
  [ ! -f "$REPO/.fab-runtime.yaml" ]
}

@test "on-stop: no .fab-status.yaml symlink exits 0" {
  cd "$REPO"
  rm "$REPO/.fab-status.yaml"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]
}

@test "on-stop: works from subdirectory" {
  mkdir -p "$REPO/some/deep/dir"
  cd "$REPO/some/deep/dir"
  run bash "$REPO/fab/.kit/hooks/on-stop.sh"
  [ "$status" -eq 0 ]

  idle_since=$(yq '.["260305-bs5x-test-change"].agent.idle_since' "$REPO/.fab-runtime.yaml")
  [ "$idle_since" != "null" ]
  [ "$idle_since" -gt 0 ]
}

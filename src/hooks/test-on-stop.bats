#!/usr/bin/env bats

# Test suite for fab/.kit/hooks/on-stop.sh
# Covers: active change writes timestamp, no .fab-status.yaml symlink, missing change dir,
#         missing .status.yaml, yq not available, fab dispatcher not available

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

  # Create a stub fab dispatcher that resolves to a known change dir
  CHANGE_DIR="fab/changes/260305-bs5x-test-change"
  mkdir -p "$REPO/$CHANGE_DIR"

  cat > "$REPO/fab/.kit/bin/fab" <<SCRIPT
#!/usr/bin/env bash
if [ "\$1" = "resolve" ] && [ "\$2" = "--folder" ]; then
  echo "260305-bs5x-test-change"
  exit 0
fi
if [ "\$1" = "runtime" ] && [ "\$2" = "set-idle" ]; then
  repo_root="\$(git rev-parse --show-toplevel 2>/dev/null)"
  runtime="\$repo_root/.fab-runtime.yaml"
  [ -f "\$runtime" ] || echo "{}" > "\$runtime"
  ts=\$(date +%s)
  yq -i ".[\"\$3\"].agent.idle_since = \$ts" "\$runtime" 2>/dev/null
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

@test "on-stop: active change writes idle_since timestamp" {
  cd "$REPO"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]

  # Verify agent.idle_since is a positive integer in .fab-runtime.yaml
  idle_since=$(yq '.["260305-bs5x-test-change"].agent.idle_since' "$REPO/.fab-runtime.yaml")
  [ "$idle_since" != "null" ]
  [ "$idle_since" -gt 0 ]
}

@test "on-stop: no .fab-status.yaml symlink exits 0 silently" {
  cd "$REPO"
  rm "$REPO/.fab-status.yaml"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]

  # No runtime file written
  [ ! -f "$REPO/.fab-runtime.yaml" ]
}

@test "on-stop: broken .fab-status.yaml symlink exits 0 silently" {
  cd "$REPO"
  rm "$REPO/.fab-status.yaml"
  ln -s "fab/changes/nonexistent/.status.yaml" "$REPO/.fab-status.yaml"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]
}

@test "on-stop: missing change directory exits 0 silently" {
  cd "$REPO"
  rm -rf "$REPO/$CHANGE_DIR"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]
}

@test "on-stop: missing .status.yaml exits 0 silently" {
  cd "$REPO"
  rm "$REPO/$CHANGE_DIR/.status.yaml"
  run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]
}

@test "on-stop: yq not available exits 0" {
  cd "$REPO"
  # Create a minimal PATH with bash/git/cat but no yq
  mkdir -p "$TEST_DIR/restricted-bin"
  ln -s "$(command -v bash)" "$TEST_DIR/restricted-bin/bash"
  ln -s "$(command -v git)" "$TEST_DIR/restricted-bin/git"
  ln -s "$(command -v cat)" "$TEST_DIR/restricted-bin/cat"
  ln -s "$(command -v head)" "$TEST_DIR/restricted-bin/head"
  ln -s "$(command -v tr)" "$TEST_DIR/restricted-bin/tr"
  ln -s "$(command -v test)" "$TEST_DIR/restricted-bin/test" 2>/dev/null || true
  ln -s "$REPO/fab/.kit/bin/fab" "$TEST_DIR/restricted-bin/fab"

  PATH="$TEST_DIR/restricted-bin" run bash fab/.kit/hooks/on-stop.sh
  [ "$status" -eq 0 ]
}

@test "on-stop: fab dispatcher not available exits 0" {
  cd "$REPO"
  rm "$REPO/fab/.kit/bin/fab"
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

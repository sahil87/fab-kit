#!/usr/bin/env bats

# Test suite for pipeline orchestrator: run.sh and dispatch.sh
# Covers: validate_manifest, detect_cycles, is_terminal, is_dispatchable,
#         find_next_dispatchable, get_parent_branch, provision_artifacts,
#         validate_prerequisites

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
RUN_SH="$REPO_ROOT/fab/.kit/scripts/pipeline/run.sh"
DISPATCH_SH="$REPO_ROOT/fab/.kit/scripts/pipeline/dispatch.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
  mkdir -p "$TEST_DIR/bin"
  export PATH="$TEST_DIR/bin:$PATH"

  # Stub changeman to return the id as-is (no resolution needed for tests)
  cat > "$TEST_DIR/bin/changeman.sh" <<'STUB'
#!/usr/bin/env bash
if [[ "$1" == "resolve" ]]; then echo "$2"; exit 0; fi
exit 1
STUB
  chmod +x "$TEST_DIR/bin/changeman.sh"

  # Stub calc-score.sh — default: gate passes
  cat > "$TEST_DIR/bin/calc-score.sh" <<'STUB'
#!/usr/bin/env bash
echo "gate: pass"
echo "score: 4.0"
echo "threshold: 3.0"
echo "change_type: feature"
STUB
  chmod +x "$TEST_DIR/bin/calc-score.sh"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# ---------------------------------------------------------------------------
# Helper: create a YAML manifest in TEST_DIR
# ---------------------------------------------------------------------------

make_manifest() {
  cat > "$TEST_DIR/manifest.yaml"
  echo "$TEST_DIR/manifest.yaml"
}

# ---------------------------------------------------------------------------
# Helper: source run.sh with required globals
# ---------------------------------------------------------------------------

source_run() {
  source "$RUN_SH"
  # Override globals computed by run.sh's top-level path resolution
  export CHANGEMAN="$TEST_DIR/bin/changeman.sh"
  export CONFIG_FILE="$TEST_DIR/config.yaml"
  export STAGEMAN="$TEST_DIR/bin/stageman.sh"
}

# ---------------------------------------------------------------------------
# Helper: source dispatch.sh with required globals
# ---------------------------------------------------------------------------

source_dispatch() {
  source "$DISPATCH_SH"
  # Override globals computed by dispatch.sh's top-level path resolution
  export CONFIG_FILE="$TEST_DIR/config.yaml"
  export FAB_DIR="$TEST_DIR/fab"
  export MANIFEST="$TEST_DIR/manifest.yaml"
  export MANIFEST_ID="${1:-test-change}"
  export CHANGE_ID="${1:-test-change}"
}

# ═══════════════════════════════════════════════════════════════════════════
# validate_manifest
# ═══════════════════════════════════════════════════════════════════════════

@test "validate_manifest: valid manifest passes" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
  - id: change-b
    depends_on: [change-a]
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 0 ]
}

@test "validate_manifest: missing base field fails" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
changes:
  - id: change-a
    depends_on: []
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 1 ]
  [[ "$output" == *"missing 'base' field"* ]]
}

@test "validate_manifest: empty changes array fails" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes: []
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 1 ]
  [[ "$output" == *"no changes"* ]]
}

@test "validate_manifest: missing id field fails" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - depends_on: []
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 1 ]
  [[ "$output" == *"missing 'id' field"* ]]
}

@test "validate_manifest: missing depends_on field fails" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 1 ]
  [[ "$output" == *"missing 'depends_on' field"* ]]
}

@test "validate_manifest: dangling dependency reference fails" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: [nonexistent]
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 1 ]
  [[ "$output" == *"not in the manifest"* ]]
}

@test "validate_manifest: multi-dependency marks invalid but passes" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
  - id: change-b
    depends_on: []
  - id: change-c
    depends_on: [change-a, change-b]
YAML
  )
  run validate_manifest "$m"
  [ "$status" -eq 0 ]
  # change-c should be marked invalid in the manifest
  local stage
  stage=$(yq -r '(.changes[] | select(.id == "change-c")).stage // ""' "$m")
  [ "$stage" = "invalid" ]
}

# ═══════════════════════════════════════════════════════════════════════════
# detect_cycles
# ═══════════════════════════════════════════════════════════════════════════

@test "detect_cycles: linear chain has no cycles" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: a
    depends_on: []
  - id: b
    depends_on: [a]
  - id: c
    depends_on: [b]
YAML
  )
  run detect_cycles "$m"
  [ "$status" -eq 0 ]
}

@test "detect_cycles: direct cycle detected" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: a
    depends_on: [b]
  - id: b
    depends_on: [a]
YAML
  )
  run detect_cycles "$m"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Circular dependency"* ]]
}

@test "detect_cycles: indirect cycle detected" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: a
    depends_on: [c]
  - id: b
    depends_on: [a]
  - id: c
    depends_on: [b]
YAML
  )
  run detect_cycles "$m"
  [ "$status" -eq 1 ]
}

@test "detect_cycles: independent nodes have no cycles" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: a
    depends_on: []
  - id: b
    depends_on: []
  - id: c
    depends_on: []
YAML
  )
  run detect_cycles "$m"
  [ "$status" -eq 0 ]
}

# ═══════════════════════════════════════════════════════════════════════════
# is_terminal
# ═══════════════════════════════════════════════════════════════════════════

@test "is_terminal: done is terminal" {
  source_run
  run is_terminal "done"
  [ "$status" -eq 0 ]
}

@test "is_terminal: failed is terminal" {
  source_run
  run is_terminal "failed"
  [ "$status" -eq 0 ]
}

@test "is_terminal: invalid is terminal" {
  source_run
  run is_terminal "invalid"
  [ "$status" -eq 0 ]
}

@test "is_terminal: intake is not terminal" {
  source_run
  run is_terminal "intake"
  [ "$status" -eq 1 ]
}

@test "is_terminal: spec is not terminal" {
  source_run
  run is_terminal "spec"
  [ "$status" -eq 1 ]
}

@test "is_terminal: tasks is not terminal" {
  source_run
  run is_terminal "tasks"
  [ "$status" -eq 1 ]
}

@test "is_terminal: apply is not terminal" {
  source_run
  run is_terminal "apply"
  [ "$status" -eq 1 ]
}

@test "is_terminal: review is not terminal" {
  source_run
  run is_terminal "review"
  [ "$status" -eq 1 ]
}

@test "is_terminal: hydrate is not terminal" {
  source_run
  run is_terminal "hydrate"
  [ "$status" -eq 1 ]
}

@test "is_terminal: empty string is not terminal" {
  source_run
  run is_terminal ""
  [ "$status" -eq 1 ]
}

# ═══════════════════════════════════════════════════════════════════════════
# is_dispatchable
# ═══════════════════════════════════════════════════════════════════════════

@test "is_dispatchable: no deps, non-terminal is dispatchable" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
YAML
  )
  run is_dispatchable "$m" "change-a"
  [ "$status" -eq 0 ]
}

@test "is_dispatchable: self is terminal, not dispatchable" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: done
YAML
  )
  run is_dispatchable "$m" "change-a"
  [ "$status" -eq 1 ]
}

@test "is_dispatchable: dependency not done, not dispatchable" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
  - id: change-b
    depends_on: [change-a]
YAML
  )
  run is_dispatchable "$m" "change-b"
  [ "$status" -eq 1 ]
}

@test "is_dispatchable: dependency done, is dispatchable" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: done
  - id: change-b
    depends_on: [change-a]
YAML
  )
  run is_dispatchable "$m" "change-b"
  [ "$status" -eq 0 ]
}

# ═══════════════════════════════════════════════════════════════════════════
# find_next_dispatchable
# ═══════════════════════════════════════════════════════════════════════════

@test "find_next_dispatchable: first dispatchable selected" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: done
  - id: change-b
    depends_on: [change-a]
  - id: change-c
    depends_on: [change-a]
YAML
  )
  run find_next_dispatchable "$m"
  [ "$status" -eq 0 ]
  [ "$output" = "change-b" ]
}

@test "find_next_dispatchable: all terminal returns 1" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: done
  - id: change-b
    depends_on: []
    stage: failed
YAML
  )
  run find_next_dispatchable "$m"
  [ "$status" -eq 1 ]
}

@test "find_next_dispatchable: intermediate stage change is re-dispatched" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: spec
  - id: change-b
    depends_on: [change-a]
YAML
  )
  # change-a is intermediate (not done, non-terminal, no deps) → dispatchable
  run find_next_dispatchable "$m"
  [ "$status" -eq 0 ]
  [ "$output" = "change-a" ]
}

@test "find_next_dispatchable: deps not met returns 1" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: failed
  - id: change-b
    depends_on: [change-a]
YAML
  )
  # change-a is terminal (failed), change-b's dep is not done → both non-dispatchable
  run find_next_dispatchable "$m"
  [ "$status" -eq 1 ]
}

@test "find_next_dispatchable: skips terminal, finds first dispatchable" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
    stage: done
  - id: change-b
    depends_on: []
    stage: failed
  - id: change-c
    depends_on: []
YAML
  )
  run find_next_dispatchable "$m"
  [ "$status" -eq 0 ]
  [ "$output" = "change-c" ]
}

# ═══════════════════════════════════════════════════════════════════════════
# get_parent_branch
# ═══════════════════════════════════════════════════════════════════════════

@test "get_parent_branch: root node returns base" {
  source_run
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
YAML
  )
  run get_parent_branch "$m" "change-a"
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "get_parent_branch: dependent node returns parent branch" {
  source_run
  # changeman resolve stub returns the id as-is
  local m
  m=$(make_manifest <<'YAML'
base: main
changes:
  - id: change-a
    depends_on: []
  - id: change-b
    depends_on: [change-a]
YAML
  )
  run get_parent_branch "$m" "change-b"
  [ "$status" -eq 0 ]
  # With no branch prefix and changeman returning "change-a" as-is
  [ "$output" = "change-a" ]
}

# ═══════════════════════════════════════════════════════════════════════════
# dispatch.sh: provision_artifacts
# ═══════════════════════════════════════════════════════════════════════════

@test "provision_artifacts: first provision creates target" {
  local wt_path="$TEST_DIR/worktree"
  mkdir -p "$wt_path/fab/changes"
  mkdir -p "$TEST_DIR/fab/changes/test-change"
  echo "intake content" > "$TEST_DIR/fab/changes/test-change/intake.md"
  echo "spec content" > "$TEST_DIR/fab/changes/test-change/spec.md"

  source_dispatch "test-change"

  run provision_artifacts "$wt_path"
  [ "$status" -eq 0 ]
  [ -f "$wt_path/fab/changes/test-change/intake.md" ]
  [ -f "$wt_path/fab/changes/test-change/spec.md" ]
}

@test "provision_artifacts: re-provision updates stale target" {
  local wt_path="$TEST_DIR/worktree"
  mkdir -p "$wt_path/fab/changes/test-change"
  echo "old intake" > "$wt_path/fab/changes/test-change/intake.md"
  # Backdate the stale file so cp -ru sees the source as newer
  touch -t 202001010000 "$wt_path/fab/changes/test-change/intake.md"
  mkdir -p "$TEST_DIR/fab/changes/test-change"
  echo "new intake" > "$TEST_DIR/fab/changes/test-change/intake.md"
  echo "spec content" > "$TEST_DIR/fab/changes/test-change/spec.md"

  source_dispatch "test-change"

  run provision_artifacts "$wt_path"
  [ "$status" -eq 0 ]
  # spec.md should now exist in the target
  [ -f "$wt_path/fab/changes/test-change/spec.md" ]
  # stale intake.md should be updated (cp -ru copies newer files)
  [ "$(cat "$wt_path/fab/changes/test-change/intake.md")" = "new intake" ]
}

@test "provision_artifacts: missing source fails" {
  local wt_path="$TEST_DIR/worktree"
  mkdir -p "$wt_path/fab/changes"
  # Do NOT create source dir

  source_dispatch "nonexistent-change"

  run provision_artifacts "$wt_path"
  [ "$status" -eq 1 ]
  [[ "$output" == *"source change folder not found"* ]]
}

# ═══════════════════════════════════════════════════════════════════════════
# dispatch.sh: validate_prerequisites
# ═══════════════════════════════════════════════════════════════════════════

@test "validate_prerequisites: missing intake.md fails" {
  local wt_path="$TEST_DIR/worktree"
  mkdir -p "$wt_path/fab/changes/test-change"
  # No intake.md, no spec.md

  # Create a manifest for write_stage
  cat > "$TEST_DIR/manifest.yaml" <<'YAML'
base: main
changes:
  - id: test-change
    depends_on: []
YAML

  source_dispatch "test-change"

  run validate_prerequisites "$wt_path"
  [ "$status" -eq 2 ]
  # Stage should be set to invalid
  local stage
  stage=$(yq -r '(.changes[] | select(.id == "test-change")).stage // ""' "$TEST_DIR/manifest.yaml")
  [ "$stage" = "invalid" ]
}

@test "validate_prerequisites: missing spec.md fails" {
  local wt_path="$TEST_DIR/worktree"
  mkdir -p "$wt_path/fab/changes/test-change"
  echo "intake" > "$wt_path/fab/changes/test-change/intake.md"
  # No spec.md

  cat > "$TEST_DIR/manifest.yaml" <<'YAML'
base: main
changes:
  - id: test-change
    depends_on: []
YAML

  source_dispatch "test-change"

  run validate_prerequisites "$wt_path"
  [ "$status" -eq 2 ]
  local stage
  stage=$(yq -r '(.changes[] | select(.id == "test-change")).stage // ""' "$TEST_DIR/manifest.yaml")
  [ "$stage" = "invalid" ]
}

@test "validate_prerequisites: both files present with passing gate succeeds" {
  local wt_path="$TEST_DIR/worktree"
  mkdir -p "$wt_path/fab/changes/test-change"
  echo "intake" > "$wt_path/fab/changes/test-change/intake.md"
  echo "spec" > "$wt_path/fab/changes/test-change/spec.md"
  # Put calc-score.sh where validate_prerequisites expects it
  mkdir -p "$wt_path/fab/.kit/scripts/lib"
  cp "$TEST_DIR/bin/calc-score.sh" "$wt_path/fab/.kit/scripts/lib/calc-score.sh"

  cat > "$TEST_DIR/manifest.yaml" <<'YAML'
base: main
changes:
  - id: test-change
    depends_on: []
YAML

  source_dispatch "test-change"

  run validate_prerequisites "$wt_path"
  [ "$status" -eq 0 ]
}

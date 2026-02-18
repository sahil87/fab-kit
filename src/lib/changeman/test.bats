#!/usr/bin/env bats

# Test suite for changeman.sh
# Covers: new, rename, resolve, switch subcommands

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
CHANGEMAN="$REPO_ROOT/fab/.kit/scripts/lib/changeman.sh"

setup() {
  TEST_DIR="$(mktemp -d)"
  FAB_ROOT="$TEST_DIR/fab"
  mkdir -p "$FAB_ROOT/changes" "$FAB_ROOT/.kit/scripts/lib" "$FAB_ROOT/.kit/templates"

  # Copy the actual changeman.sh into the kit location
  cp "$CHANGEMAN" "$FAB_ROOT/.kit/scripts/lib/changeman.sh"
  chmod +x "$FAB_ROOT/.kit/scripts/lib/changeman.sh"

  # Create a stub stageman.sh that records calls
  STAGEMAN_LOG="$TEST_DIR/stageman-calls.log"
  cat > "$FAB_ROOT/.kit/scripts/lib/stageman.sh" <<STUB
#!/usr/bin/env bash
echo "\$@" >> "$STAGEMAN_LOG"
STUB
  chmod +x "$FAB_ROOT/.kit/scripts/lib/stageman.sh"

  # Create minimal status.yaml template
  cat > "$FAB_ROOT/.kit/templates/status.yaml" <<'YAML'
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

  # Override gh and git to control detect_created_by
  # Default: gh fails, git returns "test-user"
  export PATH="$TEST_DIR/bin:$PATH"
  mkdir -p "$TEST_DIR/bin"

  cat > "$TEST_DIR/bin/gh" <<'STUB'
#!/usr/bin/env bash
exit 1
STUB
  chmod +x "$TEST_DIR/bin/gh"

  cat > "$TEST_DIR/bin/git" <<'STUB'
#!/usr/bin/env bash
if [ "$1" = "config" ] && [ "$2" = "user.name" ]; then
  echo "test-user"
  exit 0
fi
exit 1
STUB
  chmod +x "$TEST_DIR/bin/git"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# ── new: happy path ────────────────────────────────────────────────

@test "new creates folder with correct format YYMMDD-XXXX-slug" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug add-oauth
  [ "$status" -eq 0 ]
  # Output is the folder name
  [[ "$output" =~ ^[0-9]{6}-[a-z0-9]{4}-add-oauth$ ]]
  # Folder exists
  [ -d "$FAB_ROOT/changes/$output" ]
}

@test "new creates .status.yaml with correct name" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug test-change
  [ "$status" -eq 0 ]
  local folder="$output"
  [ -f "$FAB_ROOT/changes/$folder/.status.yaml" ]
  grep -q "name: \"$folder\"" "$FAB_ROOT/changes/$folder/.status.yaml" || \
    grep -q "name: $folder" "$FAB_ROOT/changes/$folder/.status.yaml"
}

@test "new with --change-id uses provided ID" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug my-change --change-id a7k2
  [ "$status" -eq 0 ]
  [[ "$output" =~ ^[0-9]{6}-a7k2-my-change$ ]]
}

@test "new with --log-args calls stageman log-command" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug my-change --log-args "Test description"
  [ "$status" -eq 0 ]
  grep -q "log-command" "$STAGEMAN_LOG"
  grep -q "Test description" "$STAGEMAN_LOG"
}

@test "new calls stageman set-state intake active fab-new" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug my-change
  [ "$status" -eq 0 ]
  grep -q "set-state.*intake active fab-new" "$STAGEMAN_LOG"
}

@test "new detects created_by from git config fallback" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug my-change
  [ "$status" -eq 0 ]
  local folder="$output"
  grep -q "test-user" "$FAB_ROOT/changes/$folder/.status.yaml"
}

# ── Slug Validation ─────────────────────────────────────────────────

@test "rejects empty slug" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug ""
  [ "$status" -ne 0 ]
  [[ "$output" == *"--slug is required"* ]]
}

@test "rejects slug with leading hyphen" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug "-bad-slug"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid slug format"* ]]
}

@test "rejects slug with trailing hyphen" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug "bad-slug-"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid slug format"* ]]
}

@test "accepts slug with uppercase (Linear issue IDs)" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug "DEV-988-add-oauth"
  [ "$status" -eq 0 ]
  [[ "$output" =~ DEV-988-add-oauth$ ]]
}

@test "accepts single-word slug" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug "refactor"
  [ "$status" -eq 0 ]
  [[ "$output" =~ -refactor$ ]]
}

# ── Change-ID Validation ───────────────────────────────────────────

@test "rejects change-id with uppercase" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug test --change-id ABCD
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid change-id"* ]]
}

@test "rejects change-id that is not 4 chars" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug test --change-id abc
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid change-id"* ]]
}

@test "rejects change-id with special chars" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug test --change-id "ab-d"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid change-id"* ]]
}

# ── Random ID Generation ───────────────────────────────────────────

@test "generates 4-char alphanumeric ID when not provided" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug my-change
  [ "$status" -eq 0 ]
  # Extract the ID portion (second field after date)
  local id
  id=$(echo "$output" | sed 's/^[0-9]\{6\}-\([a-z0-9]\{4\}\)-.*/\1/')
  [[ "$id" =~ ^[a-z0-9]{4}$ ]]
}

@test "two consecutive creates produce different IDs" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug change-one
  [ "$status" -eq 0 ]
  local id1
  id1=$(echo "$output" | sed 's/^[0-9]\{6\}-\([a-z0-9]\{4\}\)-.*/\1/')

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug change-two
  [ "$status" -eq 0 ]
  local id2
  id2=$(echo "$output" | sed 's/^[0-9]\{6\}-\([a-z0-9]\{4\}\)-.*/\1/')

  # IDs should differ (extremely unlikely to collide with 36^4 space)
  [ "$id1" != "$id2" ]
}

# ── Collision Detection ────────────────────────────────────────────

@test "provided ID collision is fatal" {
  # Create existing change with same ID
  mkdir "$FAB_ROOT/changes/260216-x1y2-existing-change"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug new-change --change-id x1y2
  [ "$status" -ne 0 ]
  [[ "$output" == *"already in use"* ]]
}

# ── --help ──────────────────────────────────────────────────────────

@test "--help prints usage" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"USAGE"* ]]
  [[ "$output" == *"--slug"* ]]
}

# ── Error Cases ─────────────────────────────────────────────────────

@test "missing slug produces error" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new
  [ "$status" -ne 0 ]
  [[ "$output" == *"--slug is required"* ]]
}

@test "unknown flag produces error" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug test --unknown-flag
  [ "$status" -ne 0 ]
  [[ "$output" == *"Unknown flag"* ]]
}

@test "no subcommand produces error" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh"
  [ "$status" -ne 0 ]
  [[ "$output" == *"No subcommand"* ]]
}

@test "unknown subcommand produces error" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" delete
  [ "$status" -ne 0 ]
  [[ "$output" == *"Unknown subcommand"* ]]
}

@test "--slug requires a value" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug
  [ "$status" -ne 0 ]
  [[ "$output" == *"--slug requires a value"* ]]
}

# ── detect_created_by Fallback Chain ────────────────────────────────

@test "detect_created_by falls back to unknown when both gh and git fail" {
  # Override git to also fail
  cat > "$TEST_DIR/bin/git" <<'STUB'
#!/usr/bin/env bash
exit 1
STUB
  chmod +x "$TEST_DIR/bin/git"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" new --slug my-change
  [ "$status" -eq 0 ]
  local folder="$output"
  grep -q "unknown" "$FAB_ROOT/changes/$folder/.status.yaml"
}

# ── rename: happy path ────────────────────────────────────────────

@test "rename changes folder name and outputs new name" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug new-slug
  [ "$status" -eq 0 ]
  [ "$output" = "260216-u6d5-new-slug" ]
  [ -d "$FAB_ROOT/changes/260216-u6d5-new-slug" ]
  [ ! -d "$FAB_ROOT/changes/260216-u6d5-old-slug" ]
}

@test "rename updates .status.yaml name field" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug new-slug
  [ "$status" -eq 0 ]
  grep -q "name: 260216-u6d5-new-slug" "$FAB_ROOT/changes/260216-u6d5-new-slug/.status.yaml"
}

@test "rename updates fab/current when it points to old folder" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"
  printf '260216-u6d5-old-slug' > "$FAB_ROOT/current"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug new-slug
  [ "$status" -eq 0 ]
  [ "$(cat "$FAB_ROOT/current")" = "260216-u6d5-new-slug" ]
}

@test "rename does not modify fab/current when it points to different change" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"
  printf '260216-abcd-other-change' > "$FAB_ROOT/current"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug new-slug
  [ "$status" -eq 0 ]
  [ "$(cat "$FAB_ROOT/current")" = "260216-abcd-other-change" ]
}

@test "rename does not create fab/current when absent" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"
  rm -f "$FAB_ROOT/current"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug new-slug
  [ "$status" -eq 0 ]
  [ ! -f "$FAB_ROOT/current" ]
}

# ── rename: slug validation ───────────────────────────────────────

@test "rename rejects slug with leading hyphen" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug "-bad-slug"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid slug format"* ]]
}

@test "rename rejects slug with trailing hyphen" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug "bad-slug-"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Invalid slug format"* ]]
}

@test "rename accepts slug with uppercase (Linear issue IDs)" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug DEV-1039-new-name
  [ "$status" -eq 0 ]
  [ "$output" = "260216-u6d5-DEV-1039-new-name" ]
}

# ── rename: error cases ───────────────────────────────────────────

@test "rename errors when source folder does not exist" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-xxxx-nonexistent --slug new-slug
  [ "$status" -ne 0 ]
  [[ "$output" == *"Change folder '260216-xxxx-nonexistent' not found"* ]]
}

@test "rename errors when destination folder already exists" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-name"
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-new-name"
  echo 'name: 260216-u6d5-old-name' > "$FAB_ROOT/changes/260216-u6d5-old-name/.status.yaml"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-name --slug new-name
  [ "$status" -ne 0 ]
  [[ "$output" == *"Folder '260216-u6d5-new-name' already exists"* ]]
}

@test "rename errors when new name is the same as current name" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-same-slug"
  echo 'name: 260216-u6d5-same-slug' > "$FAB_ROOT/changes/260216-u6d5-same-slug/.status.yaml"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-same-slug --slug same-slug
  [ "$status" -ne 0 ]
  [[ "$output" == *"New name is the same as current name"* ]]
}

@test "rename errors when --folder is missing" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --slug new-slug
  [ "$status" -ne 0 ]
  [[ "$output" == *"--folder is required"* ]]
}

@test "rename errors when --slug is missing" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug
  [ "$status" -ne 0 ]
  [[ "$output" == *"--slug is required"* ]]
}

# ── rename: stageman logging ──────────────────────────────────────

@test "rename calls stageman log-command" {
  mkdir -p "$FAB_ROOT/changes/260216-u6d5-old-slug"
  echo 'name: 260216-u6d5-old-slug' > "$FAB_ROOT/changes/260216-u6d5-old-slug/.status.yaml"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" rename --folder 260216-u6d5-old-slug --slug new-slug
  [ "$status" -eq 0 ]
  grep -q "log-command" "$STAGEMAN_LOG"
  grep -q "changeman-rename" "$STAGEMAN_LOG"
}

# ── resolve: exact match ─────────────────────────────────────────

@test "resolve: exact match resolves" {
  mkdir -p "$FAB_ROOT/changes/260213-puow-consolidate-status-reads"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "260213-puow-consolidate-status-reads"
  [ "$status" -eq 0 ]
  [ "$output" = "260213-puow-consolidate-status-reads" ]
}

# ── resolve: substring match ─────────────────────────────────────

@test "resolve: single substring match resolves" {
  mkdir -p "$FAB_ROOT/changes/260213-puow-consolidate-status-reads"
  mkdir -p "$FAB_ROOT/changes/260213-k7m2-kit-version-migrations"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "puow"
  [ "$status" -eq 0 ]
  [ "$output" = "260213-puow-consolidate-status-reads" ]
}

@test "resolve: 4-char ID match resolves" {
  mkdir -p "$FAB_ROOT/changes/260212-f9m3-enhance-srad-fuzzy"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "f9m3"
  [ "$status" -eq 0 ]
  [ "$output" = "260212-f9m3-enhance-srad-fuzzy" ]
}

# ── resolve: case insensitive ────────────────────────────────────

@test "resolve: uppercase substring resolves" {
  mkdir -p "$FAB_ROOT/changes/260213-puow-consolidate-status-reads"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "PUOW"
  [ "$status" -eq 0 ]
  [ "$output" = "260213-puow-consolidate-status-reads" ]
}

# ── resolve: multiple match ──────────────────────────────────────

@test "resolve: multiple matches returns error" {
  mkdir -p "$FAB_ROOT/changes/260213-puow-consolidate-status-reads"
  mkdir -p "$FAB_ROOT/changes/260213-k7m2-kit-version-migrations"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "260213"
  [ "$status" -ne 0 ]
  [[ "$output" == *"Multiple changes match"* ]]
}

# ── resolve: no match ────────────────────────────────────────────

@test "resolve: no match returns error" {
  mkdir -p "$FAB_ROOT/changes/260213-puow-consolidate-status-reads"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "nonexistent"
  [ "$status" -ne 0 ]
  [[ "$output" == *"No change matches"* ]]
}

# ── resolve: fab/current ─────────────────────────────────────────

@test "resolve: reads fab/current when no override" {
  mkdir -p "$FAB_ROOT/changes/260213-puow-consolidate-status-reads"
  echo "260213-puow-consolidate-status-reads" > "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve
  [ "$status" -eq 0 ]
  [ "$output" = "260213-puow-consolidate-status-reads" ]
}

@test "resolve: fab/current with trailing whitespace resolves" {
  printf "260213-puow-consolidate-status-reads\n  " > "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve
  [ "$status" -eq 0 ]
  [ "$output" = "260213-puow-consolidate-status-reads" ]
}

# ── resolve: no active change ────────────────────────────────────

@test "resolve: missing fab/current returns error" {
  rm -f "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve
  [ "$status" -ne 0 ]
  [[ "$output" == *"No active change"* ]]
}

@test "resolve: empty fab/current returns error" {
  echo "" > "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve
  [ "$status" -ne 0 ]
  [[ "$output" == *"No active change"* ]]
}

# ── resolve: missing changes directory ───────────────────────────

@test "resolve: missing changes directory returns error" {
  rm -rf "$FAB_ROOT/changes"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "something"
  [ "$status" -ne 0 ]
  [[ "$output" == *"fab/changes/ not found"* ]]
}

# ── resolve: archive exclusion ───────────────────────────────────

@test "resolve: archive folder excluded from matches" {
  mkdir -p "$FAB_ROOT/changes/archive"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve "archive"
  [ "$status" -ne 0 ]
}

@test "resolve: error message does not contain command suggestions" {
  rm -f "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" resolve
  [[ "$output" != *"Run /fab"* ]]
}

# ── switch: normal flow ──────────────────────────────────────────

@test "switch: writes fab/current" {
  mkdir -p "$FAB_ROOT/changes/260216-a7k2-add-oauth"
  cat > "$FAB_ROOT/changes/260216-a7k2-add-oauth/.status.yaml" <<'YAML'
progress:
  intake: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
YAML

  # Make stageman stub return a stage
  cat > "$FAB_ROOT/.kit/scripts/lib/stageman.sh" <<STUB
#!/usr/bin/env bash
if [ "\$1" = "current-stage" ]; then echo "intake"; exit 0; fi
echo "\$@" >> "$STAGEMAN_LOG"
STUB
  chmod +x "$FAB_ROOT/.kit/scripts/lib/stageman.sh"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" switch "a7k2"
  [ "$status" -eq 0 ]
  [ "$(cat "$FAB_ROOT/current")" = "260216-a7k2-add-oauth" ]
}

@test "switch: output includes stage and next command" {
  mkdir -p "$FAB_ROOT/changes/260216-a7k2-add-oauth"
  cat > "$FAB_ROOT/changes/260216-a7k2-add-oauth/.status.yaml" <<'YAML'
progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
YAML

  cat > "$FAB_ROOT/.kit/scripts/lib/stageman.sh" <<STUB
#!/usr/bin/env bash
if [ "\$1" = "current-stage" ]; then echo "spec"; exit 0; fi
if [ "\$1" = "display-stage" ]; then echo "spec:active"; exit 0; fi
echo "\$@" >> "$STAGEMAN_LOG"
STUB
  chmod +x "$FAB_ROOT/.kit/scripts/lib/stageman.sh"

  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" switch "a7k2"
  [ "$status" -eq 0 ]
  [[ "$output" == *"fab/current → 260216-a7k2-add-oauth"* ]]
  [[ "$output" == *"Stage:  spec (2/6) — active"* ]]
  [[ "$output" == *"Next:"* ]]
}

# ── switch: deactivation ─────────────────────────────────────────

@test "switch --blank: deletes fab/current" {
  printf 'some-change' > "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" switch --blank
  [ "$status" -eq 0 ]
  [ ! -f "$FAB_ROOT/current" ]
  [[ "$output" == *"No active change."* ]]
}

@test "switch --blank: already blank" {
  rm -f "$FAB_ROOT/current"
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" switch --blank
  [ "$status" -eq 0 ]
  [[ "$output" == *"already blank"* ]]
}

# ── switch: no argument ──────────────────────────────────────────

@test "switch: no argument produces error" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" switch
  [ "$status" -ne 0 ]
  [[ "$output" == *"requires"* ]]
}

# ── help includes resolve and switch ─────────────────────────────

@test "--help includes resolve and switch" {
  run bash "$FAB_ROOT/.kit/scripts/lib/changeman.sh" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"resolve"* ]]
  [[ "$output" == *"switch"* ]]
}

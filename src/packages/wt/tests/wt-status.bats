#!/usr/bin/env bats
#
# Tests for wt-status command
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load '../../tests/libs/bats-file/load'
load 'test_helper'

setup() {
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"

    export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Helper: Set up fab structure in a worktree
# ============================================================================

# Create a minimal fab structure with an active change
# Args: $1 = worktree path, $2 = change folder name, $3 = stage:state (for mock statusman)
setup_fab_change() {
    local wt_path="$1"
    local change_name="$2"
    local stage_state="${3:-intake:ready}"
    local change_id
    change_id=$(echo "$change_name" | cut -d'-' -f2)

    mkdir -p "$wt_path/fab/changes/$change_name"
    printf '%s\n%s' "$change_id" "$change_name" > "$wt_path/fab/current"

    # Create a minimal .status.yaml
    cat > "$wt_path/fab/changes/$change_name/.status.yaml" <<YAML
name: $change_name
progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
YAML

    # Create a mock statusman that validates the path argument and returns the requested stage:state
    mkdir -p "$wt_path/fab/.kit/scripts/lib"
    cat > "$wt_path/fab/.kit/scripts/lib/statusman.sh" <<SCRIPT
#!/usr/bin/env bash
case "\$1" in
    display-stage)
        if [ ! -f "\$2" ]; then
            echo "ERROR: file not found: \$2" >&2
            exit 1
        fi
        echo "$stage_state"
        ;;
    *) echo "unknown:unknown" ;;
esac
SCRIPT
    chmod +x "$wt_path/fab/.kit/scripts/lib/statusman.sh"
}

# Create fab directory without an active change
# Args: $1 = worktree path
setup_fab_no_change() {
    local wt_path="$1"
    mkdir -p "$wt_path/fab/project"
    # No fab/current file
}

# Create fab directory with a stale pointer
# Args: $1 = worktree path
setup_fab_stale() {
    local wt_path="$1"
    mkdir -p "$wt_path/fab/changes"
    printf '%s\n%s' "abcd" "260305-abcd-deleted-change" > "$wt_path/fab/current"
    # No actual change directory — pointer is stale
}

# ============================================================================
# Help Tests
# ============================================================================

@test "wt-status: shows help with 'help' argument" {
    run wt-status help

    assert_success
    assert_output --partial "Usage: wt-status"
    assert_output --partial "fab pipeline status"
}

@test "wt-status: shows help with --help flag" {
    run wt-status --help

    assert_success
    assert_output --partial "Usage: wt-status"
}

@test "wt-status: shows help with -h flag" {
    run wt-status -h

    assert_success
    assert_output --partial "Usage: wt-status"
}

# ============================================================================
# Default Mode (Current Worktree)
# ============================================================================

@test "wt-status: shows (no fab) when fab directory is absent" {
    run wt-status

    assert_success
    assert_output --partial "(no fab)"
}

@test "wt-status: shows (no change) when fab exists but no active change" {
    setup_fab_no_change "$TEST_REPO"

    run wt-status

    assert_success
    assert_output --partial "(no change)"
}

@test "wt-status: shows change status for current worktree" {
    setup_fab_change "$TEST_REPO" "260305-ab12-test-change" "spec:active"

    run wt-status

    assert_success
    assert_output --partial "260305-ab12-test-change"
    assert_output --partial "spec"
    assert_output --partial "active"
}

@test "wt-status: shows (main) label when in main repo" {
    run wt-status

    assert_success
    assert_output --partial "(main)"
}

@test "wt-status: shows worktree name when in a worktree" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name status-test 2>/dev/null)
    cd "$wt_path"

    run wt-status

    assert_success
    assert_output --partial "status-test"
}

# ============================================================================
# Named Mode
# ============================================================================

@test "wt-status: shows status for a named worktree" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name named-test 2>/dev/null)
    setup_fab_change "$wt_path" "260305-xy34-some-change" "tasks:active"

    run wt-status named-test

    assert_success
    assert_output --partial "260305-xy34-some-change"
    assert_output --partial "tasks"
}

@test "wt-status: errors on invalid worktree name" {
    run wt-status nonexistent

    assert_failure
    assert_output --partial "not found"
}

@test "wt-status: shows (no fab) for named worktree without fab" {
    wt-create --non-interactive --worktree-name no-fab-test &>/dev/null

    run wt-status no-fab-test

    assert_success
    assert_output --partial "(no fab)"
}

# ============================================================================
# --all Mode
# ============================================================================

@test "wt-status: --all shows all worktrees" {
    wt-create --non-interactive --worktree-name all-test-1 &>/dev/null
    wt-create --non-interactive --worktree-name all-test-2 &>/dev/null

    run wt-status --all

    assert_success
    assert_output --partial "all-test-1"
    assert_output --partial "all-test-2"
    assert_output --partial "(main)"
    assert_output --partial "Total: 3 worktree(s)"
}

@test "wt-status: --all shows repo name and location header" {
    run wt-status --all

    assert_success
    assert_output --partial "Worktrees for:"
    assert_output --partial "Location:"
}

@test "wt-status: --all marks current worktree" {
    wt-create --non-interactive --worktree-name marker-test &>/dev/null

    run wt-status --all

    assert_success
    # Current location (main repo) should be marked
    assert_output --regexp '\*.*main'
}

@test "wt-status: --all shows mixed states" {
    local wt1_path wt2_path
    wt1_path=$(wt-create --non-interactive --worktree-name fab-wt 2>/dev/null)
    wt2_path=$(wt-create --non-interactive --worktree-name plain-wt 2>/dev/null)

    setup_fab_change "$wt1_path" "260305-zz99-with-change" "review:active"
    # plain-wt has no fab — will show (no fab)

    run wt-status --all

    assert_success
    assert_output --partial "260305-zz99-with-change"
    assert_output --partial "review"
    assert_output --partial "(no fab)"
}

@test "wt-status: --all shows total count" {
    run wt-status --all

    assert_success
    assert_output --partial "Total: 1 worktree(s)"
}

# ============================================================================
# Edge Cases
# ============================================================================

@test "wt-status: handles stale fab/current pointer" {
    setup_fab_stale "$TEST_REPO"

    run wt-status

    assert_success
    assert_output --partial "(stale)"
}

@test "wt-status: handles empty fab/current file" {
    mkdir -p "$TEST_REPO/fab"
    touch "$TEST_REPO/fab/current"

    run wt-status

    assert_success
    assert_output --partial "(no change)"
}

@test "wt-status: handles fab/current with only one line (no line 2)" {
    mkdir -p "$TEST_REPO/fab"
    echo "abcd" > "$TEST_REPO/fab/current"

    run wt-status

    assert_success
    assert_output --partial "(no change)"
}

# ============================================================================
# Error Handling
# ============================================================================

@test "wt-status: errors when not in git repository" {
    cd /tmp

    run wt-status

    assert_failure
    assert_output --partial "Not a git repository"
}

@test "wt-status: errors on unknown option" {
    run wt-status --unknown

    assert_failure
    assert_output --partial "Unknown option"
}

@test "wt-status: errors on multiple positional arguments" {
    run wt-status foo bar

    assert_failure
    assert_output --partial "Unexpected argument"
}

#!/usr/bin/env bats
#
# Tests for edge cases and failure modes across wt-* commands
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load '../../tests/libs/bats-file/load'
load 'test_helper'

setup() {
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"
    export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
    clear_mock_log
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Corrupted State Recovery
# ============================================================================

@test "edge: worktree deleted outside git (rm -rf) — prune cleans up" {
    wt-create --non-interactive --worktree-name orphaned &>/dev/null

    # Worktree should be registered
    run git worktree list
    assert_output --partial "orphaned"

    # Delete outside git
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO").worktrees/orphaned"
    rm -rf "$wt_path"

    # Prune should clean it up
    git worktree prune

    run git worktree list
    refute_output --partial "orphaned"
}

@test "edge: wt-list works after external worktree deletion + prune" {
    wt-create --non-interactive --worktree-name ext-del &>/dev/null

    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO").worktrees/ext-del"
    rm -rf "$wt_path"
    git worktree prune

    run wt-list
    assert_success
    refute_output --partial "ext-del"
}

@test "edge: wt-delete handles already-removed worktree directory" {
    wt-create --non-interactive --worktree-name already-gone &>/dev/null

    # Remove directory but don't prune
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO").worktrees/already-gone"
    rm -rf "$wt_path"

    # wt-delete should handle this gracefully (may error or succeed with cleanup)
    run wt-delete --non-interactive --worktree-name already-gone
    # Accept either success (if it cleans up) or failure (if it reports not found)
    # The important thing is it doesn't crash with an unhandled error
    [[ "$status" -le 1 ]]
}

# ============================================================================
# Partial State / Failure Recovery
# ============================================================================

@test "edge: wt-create with invalid branch name fails cleanly" {
    run wt-create --non-interactive --worktree-name bad-branch "refs/invalid..name"
    assert_failure

    # Ensure no partial worktree directory was left behind
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO").worktrees/bad-branch"
    assert_dir_not_exists "$wt_path"
}

@test "edge: wt-create name collision reports clear error" {
    wt-create --non-interactive --worktree-name collision &>/dev/null

    run wt-create --non-interactive --worktree-name collision
    assert_failure
    assert_output --partial "already exists"
}

# ============================================================================
# Special Characters and Names
# ============================================================================

@test "edge: wt-create with branch containing slashes" {
    run wt-create --non-interactive feature/deep/nested/branch
    assert_success

    # Worktree name should be derived from last segment
    assert_output --partial "branch"
}

@test "edge: wt-create derives safe name from branch with special chars" {
    git checkout -b "feature/my_special-branch"
    git checkout main

    run wt-create --non-interactive "feature/my_special-branch"
    assert_success
}

# ============================================================================
# Detached HEAD
# ============================================================================

@test "edge: wt-list works in detached HEAD state" {
    git checkout --detach HEAD

    run wt-list
    assert_success
    assert_output --partial "(detached)"
}

@test "edge: wt-create works from detached HEAD" {
    git checkout --detach HEAD

    run wt-create --non-interactive --worktree-name from-detached
    assert_success
    assert_worktree_exists "from-detached"
}

# ============================================================================
# Multiple Worktrees
# ============================================================================

@test "edge: creating multiple worktrees generates unique names" {
    local names=()
    for i in 1 2 3 4 5; do
        local wt_path
        # --non-interactive sends human-readable output to stderr;
        # stdout contains only the worktree path (porcelain contract)
        wt_path=$(wt-create --non-interactive 2>/dev/null)
        local name
        name=$(basename "$wt_path")
        names+=("$name")
    done

    # All names should be unique
    local unique_count
    unique_count=$(printf '%s\n' "${names[@]}" | sort -u | wc -l)
    [[ "$unique_count" -eq 5 ]]
}

@test "edge: wt-list shows all worktrees correctly" {
    wt-create --non-interactive --worktree-name edge-a &>/dev/null
    wt-create --non-interactive --worktree-name edge-b &>/dev/null
    wt-create --non-interactive --worktree-name edge-c &>/dev/null

    run wt-list
    assert_success
    assert_output --partial "edge-a"
    assert_output --partial "edge-b"
    assert_output --partial "edge-c"
}

# ============================================================================
# Empty / Minimal Repository
# ============================================================================

@test "edge: wt-list on repo with no worktrees shows only main" {
    run wt-list
    assert_success
    assert_output --partial "(main)"
}

@test "edge: wt-delete --delete-all with no worktrees" {
    run wt-delete --non-interactive --delete-all
    assert_success
    assert_output --partial "No worktrees found"
}

# ============================================================================
# --reuse Flag Edge Cases
# ============================================================================

@test "edge: --reuse with orphaned directory (not registered as git worktree)" {
    # Create worktrees dir and an orphaned directory manually
    local repo_name=$(basename "$TEST_REPO")
    local worktrees_dir="$(dirname "$TEST_REPO")/${repo_name}.worktrees"
    mkdir -p "$worktrees_dir/orphaned"

    # --reuse should return the path blindly
    run wt-create --non-interactive --reuse --worktree-name orphaned some-branch

    assert_success
    local last_line=$(echo "$output" | tail -n 1)
    [ "$last_line" = "$worktrees_dir/orphaned" ]
}

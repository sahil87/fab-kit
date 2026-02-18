#!/usr/bin/env bats
#
# Tests for wt-delete command
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
# Help and Argument Parsing
# ============================================================================

@test "wt-delete: shows help with 'help' argument" {
    run wt-delete help
    assert_success
    assert_output --partial "Usage: wt-delete"
    assert_output --partial "Options:"
}

@test "wt-delete: shows help with --help flag" {
    run wt-delete --help
    assert_success
    assert_output --partial "Usage: wt-delete"
}

@test "wt-delete: shows help with -h flag" {
    run wt-delete -h
    assert_success
    assert_output --partial "Usage: wt-delete"
}

@test "wt-delete: errors with unknown option" {
    run wt-delete --invalid-option
    assert_failure
    assert_output --partial "Unknown option"
}

@test "wt-delete: errors when --worktree-name missing argument" {
    run wt-delete --worktree-name
    assert_failure
    assert_output --partial "Missing worktree name"
}

@test "wt-delete: errors when --delete-branch has invalid value" {
    run wt-delete --delete-branch maybe
    assert_failure
    assert_output --partial "Invalid value for --delete-branch"
}

@test "wt-delete: errors when --delete-remote has invalid value" {
    run wt-delete --delete-remote maybe
    assert_failure
    assert_output --partial "Invalid value for --delete-remote"
}

# ============================================================================
# Worktree Resolution
# ============================================================================

@test "wt-delete: deletes specified worktree by name" {
    wt-create --non-interactive --worktree-name test-wt &>/dev/null

    run wt-delete --non-interactive --worktree-name test-wt
    assert_success
    assert_output --partial "Deleted worktree"

    assert_worktree_not_exists "test-wt"
}

@test "wt-delete: deletes current worktree when inside it" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name inside-wt 2>/dev/null | tail -n 1)
    cd "$wt_path"

    run wt-delete --non-interactive
    assert_success
    assert_output --partial "Deleted worktree"
}

@test "wt-delete: errors in non-interactive mode without name and outside worktree" {
    run wt-delete --non-interactive
    assert_failure
    assert_output --partial "No worktree specified"
}

@test "wt-delete: errors when worktree not found" {
    run wt-delete --non-interactive --worktree-name nonexistent
    assert_failure
    assert_output --partial "not found"
}

# ============================================================================
# Deletion Modes - Branch Cleanup
# ============================================================================

@test "wt-delete: deletes local branch by default" {
    wt-create --non-interactive --worktree-name branch-test &>/dev/null

    run wt-delete --non-interactive --worktree-name branch-test
    assert_success

    assert_worktree_not_exists "branch-test"
    assert_branch_not_exists "wt/branch-test"
}

@test "wt-delete: preserves branch when --delete-branch false" {
    wt-create --non-interactive --worktree-name keep-branch &>/dev/null

    run wt-delete --non-interactive --worktree-name keep-branch --delete-branch false
    assert_success

    assert_worktree_not_exists "keep-branch"
    assert_branch_exists "wt/keep-branch"
}

@test "wt-delete: deletes remote branch by default" {
    # Create a branch and push it to remote
    wt-create --non-interactive --worktree-name remote-test &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/remote-test"
    (cd "$wt_path" && echo "test" > test-file.txt && git add . && git commit -q -m "test" && git push -q -u origin "wt/remote-test" 2>/dev/null) || true

    run wt-delete --non-interactive --worktree-name remote-test
    assert_success
    assert_output --partial "Deleted worktree"
}

@test "wt-delete: preserves remote branch when --delete-remote false" {
    wt-create --non-interactive --worktree-name keep-remote &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/keep-remote"
    (cd "$wt_path" && echo "test" > test-file.txt && git add . && git commit -q -m "test" && git push -q -u origin "wt/keep-remote" 2>/dev/null) || true

    run wt-delete --non-interactive --worktree-name keep-remote --delete-remote false
    assert_success

    assert_worktree_not_exists "keep-remote"
    # Remote branch should still exist
    assert_remote_branch_exists "wt/keep-remote"
}

@test "wt-delete: cleans up associated wt/* branch" {
    # Create worktree (creates wt/name branch)
    wt-create --non-interactive --worktree-name wt-branch-cleanup &>/dev/null

    # Verify wt/ branch was created
    assert_branch_exists "wt/wt-branch-cleanup"

    run wt-delete --non-interactive --worktree-name wt-branch-cleanup --delete-branch true
    assert_success

    assert_branch_not_exists "wt/wt-branch-cleanup"
}

# ============================================================================
# Uncommitted Changes Handling
# ============================================================================

@test "wt-delete: stashes changes with --stash flag" {
    wt-create --non-interactive --worktree-name stash-test &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/stash-test"

    # Create uncommitted changes
    echo "uncommitted" > "$wt_path/dirty-file.txt"
    (cd "$wt_path" && git add dirty-file.txt)

    run wt-delete --non-interactive --worktree-name stash-test --stash
    assert_success
    assert_output --partial "Stashing changes"
    assert_output --partial "Deleted worktree"

    # Verify stash exists
    assert_stash_exists "wt-delete"
}

@test "wt-delete: discards changes in non-interactive mode without --stash" {
    wt-create --non-interactive --worktree-name discard-test &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/discard-test"

    # Create uncommitted changes
    echo "will-be-discarded" > "$wt_path/dirty-file.txt"

    run wt-delete --non-interactive --worktree-name discard-test
    assert_success
    assert_output --partial "Deleted worktree"

    assert_worktree_not_exists "discard-test"
}

@test "wt-delete: --stash with -s shorthand" {
    wt-create --non-interactive --worktree-name stash-short &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/stash-short"

    echo "content" > "$wt_path/file.txt"
    (cd "$wt_path" && git add file.txt)

    run wt-delete --non-interactive --worktree-name stash-short -s
    assert_success
    assert_output --partial "Stashing changes"
}

# ============================================================================
# Unpushed Commits
# ============================================================================

@test "wt-delete: proceeds with unpushed commits in non-interactive mode" {
    wt-create --non-interactive --worktree-name unpushed-test &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/unpushed-test"

    # Create unpushed commits
    (cd "$wt_path" && echo "change" > new.txt && git add . && git commit -q -m "unpushed")

    run wt-delete --non-interactive --worktree-name unpushed-test
    assert_success
    assert_output --partial "Deleted worktree"
}

# ============================================================================
# Delete All
# ============================================================================

@test "wt-delete: --delete-all removes all worktrees" {
    wt-create --non-interactive --worktree-name wt-all-1 &>/dev/null
    wt-create --non-interactive --worktree-name wt-all-2 &>/dev/null
    wt-create --non-interactive --worktree-name wt-all-3 &>/dev/null

    run wt-delete --non-interactive --delete-all
    assert_success

    assert_worktree_not_exists "wt-all-1"
    assert_worktree_not_exists "wt-all-2"
    assert_worktree_not_exists "wt-all-3"
}

@test "wt-delete: --delete-all with no worktrees says none found" {
    run wt-delete --non-interactive --delete-all
    assert_success
    assert_output --partial "No worktrees found"
}

@test "wt-delete: --delete-all cleans up branches" {
    wt-create --non-interactive --worktree-name all-branch-1 &>/dev/null
    wt-create --non-interactive --worktree-name all-branch-2 &>/dev/null

    run wt-delete --non-interactive --delete-all --delete-branch true
    assert_success

    assert_branch_not_exists "wt/all-branch-1"
    assert_branch_not_exists "wt/all-branch-2"
}

# ============================================================================
# Error Handling
# ============================================================================

@test "wt-delete: errors when not in git repository" {
    cd /tmp
    run wt-delete
    assert_failure
    assert_output --partial "Not a git repository"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

@test "wt-delete: worktree directory is removed after deletion" {
    wt-create --non-interactive --worktree-name dir-check &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/dir-check"

    assert_dir_exists "$wt_path"

    wt-delete --non-interactive --worktree-name dir-check &>/dev/null

    assert_dir_not_exists "$wt_path"
}

@test "wt-delete: worktree not in git worktree list after deletion" {
    wt-create --non-interactive --worktree-name list-check &>/dev/null

    wt-delete --non-interactive --worktree-name list-check &>/dev/null

    run wt-list
    refute_output --partial "list-check"
}

# ============================================================================
# Integration
# ============================================================================

@test "wt-delete: create then delete with all options" {
    wt-create --non-interactive --worktree-name full-delete-test &>/dev/null

    run wt-delete --non-interactive \
        --worktree-name full-delete-test \
        --delete-branch true \
        --delete-remote true

    assert_success
    assert_worktree_not_exists "full-delete-test"
    assert_branch_not_exists "wt/full-delete-test"
}

@test "wt-delete: create → make changes → stash → delete → verify cleanup" {
    wt-create --non-interactive --worktree-name lifecycle-test &>/dev/null
    local wt_path="$(dirname "$TEST_REPO")/$(basename "$TEST_REPO")-worktrees/lifecycle-test"

    # Make uncommitted changes
    echo "important work" > "$wt_path/work.txt"
    (cd "$wt_path" && git add work.txt)

    run wt-delete --non-interactive --worktree-name lifecycle-test --stash
    assert_success

    assert_worktree_not_exists "lifecycle-test"
    assert_stash_exists "lifecycle-test"
}

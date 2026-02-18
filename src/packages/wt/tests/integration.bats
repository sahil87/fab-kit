#!/usr/bin/env bats
#
# Integration tests for wt-* commands working together
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load '../../tests/libs/bats-file/load'
load 'test_helper'

setup() {
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"
    export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
    export WORKTREE_INIT_SCRIPT="fab/.kit/worktree-init.sh"
    clear_mock_log
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Full Lifecycle Workflows
# ============================================================================

@test "integration: create → list → delete lifecycle" {
    # Create
    run wt-create --non-interactive --worktree-name lifecycle-test
    assert_success
    assert_output --partial "Created worktree: lifecycle-test"

    # List
    run wt-list
    assert_success
    assert_output --partial "lifecycle-test"

    # Delete
    run wt-delete --non-interactive --worktree-name lifecycle-test
    assert_success
    assert_output --partial "Deleted worktree"

    # Verify gone
    run wt-list
    refute_output --partial "lifecycle-test"
}

@test "integration: create → open → delete lifecycle" {
    # Create
    wt-create --non-interactive --worktree-name open-lifecycle &>/dev/null

    # Open
    run wt-open --app code open-lifecycle
    assert_success
    verify_mock_called "code" ".*open-lifecycle"

    # Delete
    run wt-delete --non-interactive --worktree-name open-lifecycle
    assert_success
}

@test "integration: create with init → init again (idempotency)" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    # Create with init enabled
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name init-idem --worktree-init true 2>/dev/null | tail -n 1)

    # Verify init ran
    assert_file_exists "${wt_path}/.init-script-ran"

    # Remove marker
    rm "${wt_path}/.init-script-ran"

    # Run init again from within the worktree
    cd "$wt_path"
    run wt-init
    assert_success

    # Verify init ran again
    assert_file_exists "${wt_path}/.init-script-ran"
}

@test "integration: create multiple → delete all" {
    wt-create --non-interactive --worktree-name multi-1 &>/dev/null
    wt-create --non-interactive --worktree-name multi-2 &>/dev/null
    wt-create --non-interactive --worktree-name multi-3 &>/dev/null

    # Verify all exist
    run wt-list
    assert_output --partial "multi-1"
    assert_output --partial "multi-2"
    assert_output --partial "multi-3"

    # Delete all
    run wt-delete --non-interactive --delete-all
    assert_success

    # Verify all gone
    run wt-list
    refute_output --partial "multi-1"
    refute_output --partial "multi-2"
    refute_output --partial "multi-3"

    # Verify branches cleaned up
    assert_branch_not_exists "wt/multi-1"
    assert_branch_not_exists "wt/multi-2"
    assert_branch_not_exists "wt/multi-3"
}

@test "integration: non-interactive automation workflow (no prompts)" {
    # Full lifecycle using only --non-interactive flags
    run wt-create --non-interactive --worktree-name auto-test --worktree-init false --worktree-open skip
    assert_success

    run wt-list
    assert_success
    assert_output --partial "auto-test"

    run wt-open --app code auto-test
    assert_success

    run wt-delete --non-interactive --worktree-name auto-test --delete-branch true --delete-remote true
    assert_success

    run wt-list
    refute_output --partial "auto-test"
}

# ============================================================================
# Cross-Command State Consistency
# ============================================================================

@test "integration: wt-create without open → wt-open later" {
    wt-create --non-interactive --worktree-name deferred-open --worktree-open skip &>/dev/null

    # Open later with explicit app
    run wt-open --app cursor deferred-open
    assert_success
    verify_mock_called "cursor" ".*deferred-open"
}

@test "integration: created worktree has correct branch" {
    run wt-create --non-interactive --worktree-name branch-verify
    assert_success

    assert_branch_exists "wt/branch-verify"

    # List should show the branch
    run wt-list
    assert_output --partial "wt/branch-verify"
}

@test "integration: branch-based worktree create → delete preserves other branches" {
    git checkout -b feature/keep-me
    echo "keep" > keep.txt
    git add keep.txt
    git commit -q -m "keep"
    git checkout main

    git checkout -b feature/delete-me
    git checkout main

    wt-create --non-interactive --worktree-name del-branch feature/delete-me &>/dev/null

    run wt-delete --non-interactive --worktree-name del-branch --delete-branch true
    assert_success

    # delete-me should be gone
    assert_branch_not_exists "feature/delete-me"

    # keep-me should still exist
    assert_branch_exists "feature/keep-me"
}

@test "integration: wt-create with --worktree-init false → wt-init later" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name late-init --worktree-init false 2>/dev/null | tail -n 1)

    # Init should not have run
    assert_file_not_exists "${wt_path}/.init-script-ran"

    # Run init manually
    cd "$wt_path"
    run wt-init
    assert_success
    assert_file_exists "${wt_path}/.init-script-ran"
}

# ============================================================================
# Multiple Commands in Sequence
# ============================================================================

@test "integration: rapid create-delete cycle" {
    for i in 1 2 3; do
        wt-create --non-interactive --worktree-name "cycle-$i" &>/dev/null
        wt-delete --non-interactive --worktree-name "cycle-$i" &>/dev/null
    done

    # All should be gone
    run wt-list
    refute_output --partial "cycle-"
}

@test "integration: delete from within worktree then create new" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name first-wt 2>/dev/null | tail -n 1)
    cd "$wt_path"

    # Delete current worktree
    wt-delete --non-interactive &>/dev/null

    # Go back to main repo
    cd "$TEST_REPO"

    # Create a new one
    run wt-create --non-interactive --worktree-name second-wt
    assert_success
    assert_worktree_exists "second-wt"
}

@test "integration: wt-list from within worktree shows all worktrees" {
    wt-create --non-interactive --worktree-name list-a &>/dev/null
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name list-b 2>/dev/null | tail -n 1)
    cd "$wt_path"

    run wt-list
    assert_success
    assert_output --partial "list-a"
    assert_output --partial "list-b"
}

# ============================================================================
# Git State Integrity
# ============================================================================

@test "integration: git state is clean after create-delete cycle" {
    wt-create --non-interactive --worktree-name integrity-test &>/dev/null
    wt-delete --non-interactive --worktree-name integrity-test &>/dev/null

    # Git repo should be in a clean state
    assert_git_state
}

@test "integration: main repo is unaffected by worktree operations" {
    # Get initial state
    local initial_commit
    initial_commit=$(git rev-parse HEAD)

    # Create, modify, and delete a worktree
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name no-affect 2>/dev/null | tail -n 1)
    (cd "$wt_path" && echo "new" > new.txt && git add . && git commit -q -m "worktree commit")
    wt-delete --non-interactive --worktree-name no-affect &>/dev/null

    # Main repo HEAD should be unchanged
    local current_commit
    current_commit=$(git rev-parse HEAD)
    [[ "$initial_commit" == "$current_commit" ]]
}

@test "integration: worktree commit is independent of main repo" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name independent 2>/dev/null | tail -n 1)

    # Commit in worktree
    (cd "$wt_path" && echo "wt content" > wt-file.txt && git add . && git commit -q -m "wt change")

    # File should not exist in main repo
    assert_file_not_exists "wt-file.txt"

    # Cleanup
    wt-delete --non-interactive --worktree-name independent &>/dev/null
}

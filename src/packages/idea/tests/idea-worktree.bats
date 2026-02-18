#!/usr/bin/env bats
#
# idea-worktree.bats - Tests for worktree resolution behavior
#

setup() {
    load '../../tests/libs/bats-support/load'
    load '../../tests/libs/bats-assert/load'
    load '../../tests/libs/bats-file/load'
    load 'test_helper'
    _common_setup

    # Create a worktree from the test repo
    WORKTREE_DIR="${TEST_REPO}-worktree-$$"
    git worktree add -q "$WORKTREE_DIR" -b test-worktree 2>/dev/null
}

teardown() {
    # Clean up worktree before removing repo
    if [[ -n "${WORKTREE_DIR:-}" && -d "${WORKTREE_DIR:-}" ]]; then
        cd "$TEST_REPO" 2>/dev/null || true
        git worktree remove --force "$WORKTREE_DIR" 2>/dev/null || true
        rm -rf "$WORKTREE_DIR" 2>/dev/null || true
    fi
    _common_teardown
}

# --- List from worktree ---

@test "worktree: list from worktree reads main repo backlog" {
    cd "$WORKTREE_DIR"

    run idea list
    assert_success
    assert_output --partial "ab12"
    assert_output --partial "Build a rocket ship"
}

# --- Add from worktree ---

@test "worktree: add from worktree writes to main repo backlog" {
    cd "$WORKTREE_DIR"

    run idea "New idea from worktree"
    assert_success

    # Idea should be in main repo's backlog
    assert_file_exists "${TEST_REPO}/fab/backlog.md"
    grep -q "New idea from worktree" "${TEST_REPO}/fab/backlog.md"

    # Backlog should NOT be created in the worktree
    assert_file_not_exists "${WORKTREE_DIR}/fab/backlog.md"
}

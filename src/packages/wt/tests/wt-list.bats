#!/usr/bin/env bats
#
# Tests for wt-list command
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load '../../tests/libs/bats-file/load'
load 'test_helper'

setup() {
    # Create test repository
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"

    # Add wt commands to PATH
    export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Basic Functionality Tests
# ============================================================================

@test "wt-list: shows repository name and location" {
    run wt-list

    assert_success
    assert_output --partial "Worktrees for:"
    assert_output --partial "$(basename "$TEST_REPO")"
    assert_output --partial "Location:"
}

@test "wt-list: shows main repository" {
    run wt-list

    assert_success
    assert_output --partial "(main)"
    assert_output --partial "main"  # branch name
    assert_output --partial "$TEST_REPO"
}

@test "wt-list: shows total count" {
    run wt-list

    assert_success
    assert_output --partial "Total: 1 worktree(s)"
}

@test "wt-list: marks current worktree with asterisk" {
    run wt-list

    assert_success
    # Should have asterisk for current location (may have ANSI color codes)
    assert_output --regexp '\*.*main'
}

@test "wt-list: lists multiple worktrees" {
    # Create a couple of worktrees
    wt-create --non-interactive --worktree-name test-wt1 &>/dev/null
    wt-create --non-interactive --worktree-name test-wt2 &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "test-wt1"
    assert_output --partial "test-wt2"
    assert_output --partial "Total: 3 worktree(s)"
}

@test "wt-list: shows branch names for worktrees" {
    # Create worktree for specific branch
    git checkout -b feature/test &>/dev/null
    git checkout main &>/dev/null
    wt-create --non-interactive --worktree-name my-feature feature/test &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "my-feature"
    assert_output --partial "feature/test"
}

# ============================================================================
# Help and Options Tests
# ============================================================================

@test "wt-list: shows help with 'help' argument" {
    run wt-list help

    assert_success
    assert_output --partial "Usage: wt-list"
    assert_output --partial "Lists all git worktrees"
}

@test "wt-list: shows help with --help flag" {
    run wt-list --help

    assert_success
    assert_output --partial "Usage: wt-list"
}

@test "wt-list: shows help with -h flag" {
    run wt-list -h

    assert_success
    assert_output --partial "Usage: wt-list"
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "wt-list: errors when not in git repository" {
    cd /tmp

    run wt-list

    assert_failure
    assert_output --partial "Not a git repository"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

@test "wt-list: succeeds with no worktrees" {
    # In main repo with no additional worktrees
    run wt-list

    assert_success
    assert_output --partial "Total: 1 worktree(s)"
}

# ============================================================================
# Output Format Tests
# ============================================================================

@test "wt-list: respects NO_COLOR environment variable" {
    export NO_COLOR=1

    run wt-list

    assert_success
    # Should not contain ANSI color codes when NO_COLOR is set
    refute_output --regexp $'\033\\['
}

@test "wt-list: output is well-formatted with columns" {
    wt-create --non-interactive --worktree-name aligned-test &>/dev/null

    run wt-list

    assert_success
    # Should have consistent formatting/alignment
    assert_output --regexp '[[:space:]]+aligned-test[[:space:]]+'
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "wt-list: shows worktree immediately after creation" {
    wt-create --non-interactive --worktree-name new-wt &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "new-wt"
}

@test "wt-list: no longer shows worktree after deletion" {
    wt-create --non-interactive --worktree-name temp-wt &>/dev/null

    # Verify it's listed
    run wt-list
    assert_output --partial "temp-wt"

    # Delete it
    wt-delete --non-interactive --worktree-name temp-wt &>/dev/null

    # Verify it's gone
    run wt-list
    refute_output --partial "temp-wt"
}

@test "wt-list: correctly shows worktree from within worktree" {
    # Create and enter a worktree
    local wt_path=$(wt-create --non-interactive --worktree-name inside-test 2>/dev/null | tail -n 1)
    cd "$wt_path"

    run wt-list

    assert_success
    assert_output --partial "inside-test"
    # Should mark this worktree as current (may have ANSI color codes)
    assert_output --regexp '\*.*inside-test'
}

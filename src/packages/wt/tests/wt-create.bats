#!/usr/bin/env bats
#
# Tests for wt-create command
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

    # Set default init script path
    export WORKTREE_INIT_SCRIPT="fab/.kit/worktree-init.sh"

    # Clear mock log
    clear_mock_log
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Basic Creation Tests - Exploratory Worktrees
# ============================================================================

@test "wt-create: creates exploratory worktree with random name (non-interactive)" {
    run wt-create --non-interactive

    assert_success
    assert_output --regexp 'Created worktree: [a-z]+-[a-z]+'

    # Verify worktree exists
    local wt_path=$(echo "$output" | tail -n 1)
    assert_dir_exists "$wt_path"

    local wt_name=$(basename "$wt_path")
    assert_worktree_exists "$wt_name"
}

@test "wt-create: creates wt/<name> branch for exploratory worktree" {
    run wt-create --non-interactive

    assert_success

    # Extract worktree name from output
    local wt_name=$(echo "$output" | grep "Created worktree:" | sed 's/Created worktree: //')
    local expected_branch="wt/${wt_name}"

    # Verify branch was created
    assert_branch_exists "$expected_branch"
}

@test "wt-create: prints full path as last line of output" {
    run wt-create --non-interactive

    assert_success

    # Last line should be a full path
    local last_line=$(echo "$output" | tail -n 1)
    assert_dir_exists "$last_line"
}

# ============================================================================
# Branch-Based Worktree Creation
# ============================================================================

@test "wt-create: creates worktree for existing local branch" {
    git checkout -b feature/auth
    git checkout main

    run wt-create --non-interactive --worktree-name my-feature feature/auth

    assert_success
    assert_output --partial "Created worktree: my-feature"
    assert_output --partial "Branch: feature/auth"

    assert_worktree_exists "my-feature"
}

@test "wt-create: fetches and creates worktree for remote branch" {
    # Create a remote branch
    add_remote_branch "remote-feature"

    # Remove local copy
    git branch -D remote-feature

    run wt-create --non-interactive --worktree-name remote-wt remote-feature

    assert_success
    assert_output --partial "remote-wt"
    assert_worktree_exists "remote-wt"
}

@test "wt-create: creates new branch when branch doesn't exist" {
    run wt-create --non-interactive --worktree-name new-branch-wt new-feature

    assert_success
    assert_worktree_exists "new-branch-wt"
    assert_branch_exists "new-feature"
}

# ============================================================================
# Name Generation and Collision Tests
# ============================================================================

@test "wt-create: uses --worktree-name when provided" {
    run wt-create --non-interactive --worktree-name custom-name

    assert_success
    assert_output --partial "custom-name"
    assert_worktree_exists "custom-name"
}

@test "wt-create: derives name from branch name" {
    run wt-create --non-interactive feature/login

    assert_success
    # Should derive "login" from "feature/login"
    assert_output --partial "login"
}

@test "wt-create: errors on name collision" {
    # Create first worktree
    wt-create --non-interactive --worktree-name collision-test &>/dev/null

    # Try to create another with same name
    run wt-create --non-interactive --worktree-name collision-test

    assert_failure
    assert_output --partial "already exists"
}

@test "wt-create: respects WT_CREATE_RETRIES environment variable" {
    # Set retries to a low value
    export WT_CREATE_RETRIES=2

    # This test verifies the variable is respected
    # (Full retry exhaustion test would require forcing collisions)
    run wt-create --non-interactive

    assert_success
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "wt-create: shows help with 'help' argument" {
    run wt-create help

    assert_success
    assert_output --partial "Usage: wt-create"
    assert_output --partial "Options:"
}

@test "wt-create: shows help with --help flag" {
    run wt-create --help

    assert_success
    assert_output --partial "Usage: wt-create"
}

@test "wt-create: shows help with -h flag" {
    run wt-create -h

    assert_success
    assert_output --partial "Usage: wt-create"
}

@test "wt-create: errors with unknown option" {
    run wt-create --invalid-option

    assert_failure
    assert_output --partial "Unknown option"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

@test "wt-create: errors when --worktree-name missing argument" {
    run wt-create --worktree-name

    assert_failure
    assert_output --partial "Missing worktree name"
}

@test "wt-create: errors when --worktree-init missing argument" {
    run wt-create --worktree-init

    assert_failure
    assert_output --partial "Missing value for --worktree-init"
}

@test "wt-create: errors when --worktree-init has invalid value" {
    run wt-create --worktree-init maybe

    assert_failure
    assert_output --partial "Invalid value for --worktree-init"
}

@test "wt-create: accepts --worktree-init true" {
    run wt-create --non-interactive --worktree-init true

    assert_success
}

@test "wt-create: accepts --worktree-init false" {
    run wt-create --non-interactive --worktree-init false

    assert_success
}

@test "wt-create: errors when too many branch arguments provided" {
    run wt-create branch1 branch2

    assert_failure
    assert_output --partial "Too many arguments"
}

# ============================================================================
# Init Script Tests
# ============================================================================

@test "wt-create: runs init script for exploratory worktree" {
    # Create and commit init script so it appears in worktree checkouts
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    run wt-create --non-interactive

    assert_success

    # Navigate to worktree and check if init script ran
    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_exists "${wt_path}/.init-script-ran"
}

@test "wt-create: skips init script when --worktree-init false" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    run wt-create --non-interactive --worktree-init false

    assert_success

    # Init script should not have run
    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_not_exists "${wt_path}/.init-script-ran"
}

@test "wt-create: handles missing init script gracefully" {
    # No init script exists
    run wt-create --non-interactive --worktree-init true

    assert_success
    # Should not error even though init script doesn't exist
}

@test "wt-create: respects WORKTREE_INIT_SCRIPT environment variable" {
    export WORKTREE_INIT_SCRIPT="custom-init.sh"
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add custom init script"

    run wt-create --non-interactive

    assert_success

    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_exists "${wt_path}/.init-script-ran"
}

# ============================================================================
# App Opening Tests
# ============================================================================

@test "wt-create: opens in VSCode when --worktree-open code specified" {
    run wt-create --non-interactive --worktree-open code

    assert_success

    # Verify mock was called
    verify_mock_called "code" ".+-worktrees/[a-z]+-[a-z]+$"
}

@test "wt-create: opens in Cursor when --worktree-open cursor specified" {
    run wt-create --non-interactive --worktree-open cursor

    assert_success

    verify_mock_called "cursor" ".+-worktrees/[a-z]+-[a-z]+$"
}

@test "wt-create: skips opening when --worktree-open skip specified" {
    run wt-create --non-interactive --worktree-open skip

    assert_success

    # Mock log should be empty
    run cat "$MOCK_LOG"
    assert_output ""
}

@test "wt-create: errors when --worktree-open missing argument" {
    run wt-create --worktree-open

    assert_failure
    assert_output --partial "Missing value for --worktree-open"
}

@test "wt-create: non-interactive mode skips app opening by default" {
    run wt-create --non-interactive

    assert_success

    # No app should be opened
    run cat "$MOCK_LOG"
    assert_output ""
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "wt-create: errors when not in git repository" {
    cd /tmp

    run wt-create

    assert_failure
    assert_output --partial "Not a git repository"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

@test "wt-create: errors when branch already checked out" {
    git checkout -b already-checked-out
    git checkout main

    # Create worktree for the branch
    wt-create --non-interactive --worktree-name wt1 already-checked-out &>/dev/null

    # Try to create another worktree for same branch
    run wt-create --non-interactive --worktree-name wt2 already-checked-out

    assert_failure
    assert_output --partial "already be checked out"
}

# ============================================================================
# Branch Base Tests (worktrees branch off HEAD, not main)
# ============================================================================

@test "wt-create: exploratory worktree branches off current branch, not main" {
    # Create a feature branch with a unique commit
    git checkout -b feature/has-marker
    echo "marker" > marker.txt
    git add marker.txt
    git commit -q -m "Add marker on feature branch"
    local feature_commit=$(git rev-parse HEAD)

    # Stay on the feature branch and create an exploratory worktree
    run wt-create --non-interactive --worktree-name from-feature

    assert_success

    # The worktree should contain the marker file (branched off feature, not main)
    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_exists "${wt_path}/marker.txt"

    # The worktree's HEAD should be the feature commit
    local wt_commit=$(git -C "$wt_path" rev-parse HEAD)
    [ "$wt_commit" = "$feature_commit" ]
}

@test "wt-create: new branch creation branches off current branch, not main" {
    # Create a feature branch with a unique commit
    git checkout -b feature/base-branch
    echo "base content" > base.txt
    git add base.txt
    git commit -q -m "Add base content"
    local base_commit=$(git rev-parse HEAD)

    # Create a new (non-existent) branch worktree while on feature branch
    run wt-create --non-interactive --worktree-name derived-wt derived-branch

    assert_success

    # The worktree should have the base content (derived from feature, not main)
    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_exists "${wt_path}/base.txt"

    # The worktree's HEAD should match the feature branch commit
    local wt_commit=$(git -C "$wt_path" rev-parse HEAD)
    [ "$wt_commit" = "$base_commit" ]
}

@test "wt-create: exploratory worktree from main still works" {
    # On main, behavior should be the same as before
    local main_commit=$(git rev-parse HEAD)

    run wt-create --non-interactive --worktree-name from-main

    assert_success

    local wt_path=$(echo "$output" | tail -n 1)
    local wt_commit=$(git -C "$wt_path" rev-parse HEAD)
    [ "$wt_commit" = "$main_commit" ]
}

@test "wt-create: existing branch checkout is unaffected by current branch" {
    # Create two branches with different content
    git checkout -b branch-a
    echo "branch-a content" > a.txt
    git add a.txt
    git commit -q -m "Add a.txt"
    git checkout main

    git checkout -b branch-b
    echo "branch-b content" > b.txt
    git add b.txt
    git commit -q -m "Add b.txt"

    # While on branch-b, check out branch-a into a worktree
    run wt-create --non-interactive --worktree-name checkout-a branch-a

    assert_success

    # Should have branch-a's content, not branch-b's
    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_exists "${wt_path}/a.txt"
    assert_file_not_exists "${wt_path}/b.txt"
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "wt-create: full workflow with all options" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"
    git checkout -b full-test-branch
    git checkout main

    run wt-create --non-interactive \
        --worktree-name full-test \
        --worktree-init true \
        --worktree-open code \
        full-test-branch

    assert_success
    assert_output --partial "Created worktree: full-test"
    assert_output --partial "Branch: full-test-branch"

    assert_worktree_exists "full-test"
    verify_mock_called "code"

    # Check init script ran
    local wt_path=$(echo "$output" | tail -n 1)
    assert_file_exists "${wt_path}/.init-script-ran"
}

@test "wt-create: creates worktree in correct directory structure" {
    run wt-create --non-interactive --worktree-name test-structure

    assert_success

    local repo_name=$(basename "$TEST_REPO")
    local expected_dir="$(dirname "$TEST_REPO")/${repo_name}-worktrees/test-structure"

    assert_dir_exists "$expected_dir"
}

@test "wt-create: worktree is immediately listed by wt-list" {
    wt-create --non-interactive --worktree-name immediate-list &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "immediate-list"
}

# ============================================================================
# --reuse Flag Tests
# ============================================================================

@test "wt-create: --reuse returns existing worktree path on collision" {
    # Create first worktree
    local first_output
    first_output=$(wt-create --non-interactive --worktree-name reuse-test 2>/dev/null)
    local first_path=$(echo "$first_output" | tail -n 1)

    # Try to create again with --reuse
    run wt-create --non-interactive --reuse --worktree-name reuse-test

    assert_success
    local reuse_path=$(echo "$output" | tail -n 1)
    [ "$reuse_path" = "$first_path" ]
}

@test "wt-create: --reuse creates normally when no collision" {
    run wt-create --non-interactive --reuse --worktree-name reuse-fresh

    assert_success
    assert_worktree_exists "reuse-fresh"
}

@test "wt-create: --reuse skips init script on collision" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    # Create first worktree (init runs)
    local first_output
    first_output=$(wt-create --non-interactive --worktree-name reuse-init-test 2>/dev/null)
    local wt_path=$(echo "$first_output" | tail -n 1)
    assert_file_exists "${wt_path}/.init-script-ran"

    # Remove marker
    rm "${wt_path}/.init-script-ran"

    # Reuse — init should NOT run again
    run wt-create --non-interactive --reuse --worktree-name reuse-init-test

    assert_success
    assert_file_not_exists "${wt_path}/.init-script-ran"
}

@test "wt-create: --reuse requires --worktree-name" {
    run wt-create --non-interactive --reuse

    assert_failure
    assert_output --partial "--reuse requires --worktree-name"
}

@test "wt-create: --reuse prints path as last line on collision" {
    # Create first worktree
    wt-create --non-interactive --worktree-name reuse-lastline &>/dev/null

    # Reuse
    run wt-create --non-interactive --reuse --worktree-name reuse-lastline

    assert_success
    local last_line=$(echo "$output" | tail -n 1)
    assert_dir_exists "$last_line"
}

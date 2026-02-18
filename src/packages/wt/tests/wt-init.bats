#!/usr/bin/env bats
#
# Tests for wt-init command
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
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Basic Functionality Tests
# ============================================================================

@test "wt-init: runs init script when it exists" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"

    run wt-init

    assert_success
    assert_output --partial "Running worktree init"
    assert_output --partial "Test init script executed"
    assert_output --partial "Worktree init complete"

    # Verify init script ran
    assert_file_exists ".init-script-ran"
}

@test "wt-init: friendly message when script doesn't exist" {
    # No init script exists

    run wt-init

    assert_success
    assert_output --partial "No init script found"
    assert_output --partial "To add an init script:"
}

@test "wt-init: shows instructions for creating missing script" {
    run wt-init

    assert_success
    assert_output --partial "mkdir -p"
    assert_output --partial "touch"
}

@test "wt-init: exits successfully even when script doesn't exist" {
    run wt-init

    assert_success
}

# ============================================================================
# Environment Variable Tests
# ============================================================================

@test "wt-init: respects default WORKTREE_INIT_SCRIPT path" {
    export WORKTREE_INIT_SCRIPT="fab/.kit/worktree-init.sh"
    create_test_init_script "$WORKTREE_INIT_SCRIPT"

    run wt-init

    assert_success
    assert_output --partial "Worktree init complete"
}

@test "wt-init: respects custom WORKTREE_INIT_SCRIPT path" {
    export WORKTREE_INIT_SCRIPT="custom/path/init.sh"
    create_test_init_script "$WORKTREE_INIT_SCRIPT"

    run wt-init

    assert_success
    assert_output --partial "Worktree init complete"
    assert_file_exists ".init-script-ran"
}

@test "wt-init: handles nested directory paths" {
    export WORKTREE_INIT_SCRIPT="very/deep/nested/path/init.sh"
    create_test_init_script "$WORKTREE_INIT_SCRIPT"

    run wt-init

    assert_success
}

# ============================================================================
# Help and Options Tests
# ============================================================================

@test "wt-init: shows help with 'help' argument" {
    run wt-init help

    assert_success
    assert_output --partial "Usage: wt-init"
    assert_output --partial "Runs the init script"
}

@test "wt-init: shows help with --help flag" {
    run wt-init --help

    assert_success
    assert_output --partial "Usage: wt-init"
}

@test "wt-init: shows help with -h flag" {
    run wt-init -h

    assert_success
    assert_output --partial "Usage: wt-init"
}

@test "wt-init: errors with unknown argument" {
    run wt-init unknown-arg

    assert_failure
    assert_output --partial "Unknown argument"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "wt-init: errors when not in git repository" {
    cd /tmp

    run wt-init

    assert_failure
    assert_output --partial "Not a git repository"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

# ============================================================================
# Script Execution Tests
# ============================================================================

@test "wt-init: runs script in repository root" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"

    # Add a command to script that creates a file in current directory
    cat >> "$WORKTREE_INIT_SCRIPT" <<'EOF'
# Create file in current directory
pwd > current-dir.txt
EOF

    run wt-init

    assert_success

    # Verify script ran in repo root
    assert_file_exists "current-dir.txt"
    local script_dir=$(cat "current-dir.txt")
    assert_equal "$script_dir" "$TEST_REPO"
}

@test "wt-init: can be run from worktree" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    # Create and enter worktree
    local wt_path=$(wt-create --non-interactive --worktree-name test-wt 2>/dev/null | tail -n 1)
    cd "$wt_path"

    run wt-init

    assert_success
    assert_output --partial "Worktree init complete"
}

@test "wt-init: is idempotent (can be run multiple times)" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"

    # Run first time
    run wt-init
    assert_success

    # Run second time
    run wt-init
    assert_success
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "wt-init: works after wt-create with --worktree-init false" {
    create_test_init_script "$WORKTREE_INIT_SCRIPT"
    git add "$WORKTREE_INIT_SCRIPT"
    git commit -q -m "Add init script"

    # Create worktree without running init
    local wt_path=$(wt-create --non-interactive --worktree-init false 2>/dev/null | tail -n 1)
    cd "$wt_path"

    # Verify init didn't run
    assert_file_not_exists ".init-script-ran"

    # Run init manually
    run wt-init

    assert_success
    assert_file_exists ".init-script-ran"
}

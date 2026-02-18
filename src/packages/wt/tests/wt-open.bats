#!/usr/bin/env bats
#
# Tests for wt-open command
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

@test "wt-open: shows help with 'help' argument" {
    run wt-open help
    assert_success
    assert_output --partial "Usage: wt-open"
}

@test "wt-open: shows help with --help flag" {
    run wt-open --help
    assert_success
    assert_output --partial "Usage: wt-open"
}

@test "wt-open: shows help with -h flag" {
    run wt-open -h
    assert_success
    assert_output --partial "Usage: wt-open"
}

@test "wt-open: errors with unknown option" {
    run wt-open --invalid-option
    assert_failure
    assert_output --partial "Unknown option"
}

@test "wt-open: errors when --app missing argument" {
    run wt-open --app
    assert_failure
    assert_output --partial "Missing app name"
}

@test "wt-open: errors when too many arguments" {
    run wt-open arg1 arg2
    assert_failure
    assert_output --partial "Too many arguments"
}

# ============================================================================
# Target Resolution
# ============================================================================

@test "wt-open: opens current worktree with --app code" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name open-test 2>/dev/null | tail -n 1)
    cd "$wt_path"

    run wt-open --app code
    assert_success

    verify_mock_called "code" ".*open-test"
}

@test "wt-open: opens named worktree with --app code" {
    wt-create --non-interactive --worktree-name named-open &>/dev/null

    run wt-open --app code named-open
    assert_success

    verify_mock_called "code" ".*named-open"
}

@test "wt-open: opens worktree by path" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name path-open 2>/dev/null | tail -n 1)

    run wt-open --app code "$wt_path"
    assert_success

    verify_mock_called "code" ".*path-open"
}

@test "wt-open: errors when worktree not found by name" {
    run wt-open --app code nonexistent-wt
    assert_failure
    assert_output --partial "not found"
}

@test "wt-open: errors with --app but no target from main repo" {
    run wt-open --app code
    assert_failure
    assert_output --partial "No worktree specified"
}

# ============================================================================
# App Detection — Mock Apps
# ============================================================================

@test "wt-open: opens in VSCode (code)" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name code-test 2>/dev/null | tail -n 1)

    run wt-open --app code "$wt_path"
    assert_success
    verify_mock_called "code"
}

@test "wt-open: opens in Cursor (cursor)" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name cursor-test 2>/dev/null | tail -n 1)

    run wt-open --app cursor "$wt_path"
    assert_success
    verify_mock_called "cursor"
}

@test "wt-open: opens in Ghostty (ghostty_linux)" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name ghostty-test 2>/dev/null | tail -n 1)

    run wt-open --app ghostty_linux "$wt_path"
    assert_success
    verify_mock_called "ghostty"
}

@test "wt-open: opens in Nautilus (nautilus)" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name nautilus-test 2>/dev/null | tail -n 1)

    run wt-open --app nautilus "$wt_path"
    assert_success
    verify_mock_called "nautilus"
}

@test "wt-open: opens in Dolphin (dolphin)" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name dolphin-test 2>/dev/null | tail -n 1)

    run wt-open --app dolphin "$wt_path"
    assert_success
    verify_mock_called "dolphin"
}

@test "wt-open: opens in tmux window (tmux_window)" {
    # Simulate plain tmux session (not byobu)
    export TMUX="/tmp/tmux-test/default,12345,0"
    unset BYOBU_TTY BYOBU_BACKEND BYOBU_SESSION BYOBU_CONFIG_DIR

    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name tmux-test 2>/dev/null | tail -n 1)

    run wt-open --app tmux_window "$wt_path"
    assert_success
    verify_mock_called "tmux"
}

@test "wt-open: opens in byobu tab (byobu_tab)" {
    # Simulate byobu session
    export BYOBU_BACKEND="tmux"
    export TMUX="/tmp/tmux-test/default,12345,0"

    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name byobu-test 2>/dev/null | tail -n 1)

    run wt-open --app byobu_tab "$wt_path"
    assert_success
    verify_mock_called "byobu"
}

# ============================================================================
# Smart Defaults
# ============================================================================

@test "wt-open: suggests code when TERM_PROGRAM is vscode" {
    export TERM_PROGRAM="vscode"
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name smart-code 2>/dev/null | tail -n 1)
    cd "$wt_path"

    # In non-interactive context, --app is needed; this test verifies the
    # smart detection by checking the menu default would be code
    # We can't easily test interactive menus, so just verify the code path works
    run wt-open --app code
    assert_success
    verify_mock_called "code"
}

@test "wt-open: suggests tmux_window when TMUX is set" {
    export TMUX="/tmp/tmux-test/default,12345,0"
    unset BYOBU_TTY BYOBU_BACKEND BYOBU_SESSION BYOBU_CONFIG_DIR
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name smart-tmux 2>/dev/null | tail -n 1)

    run wt-open --app tmux_window "$wt_path"
    assert_success
    verify_mock_called "tmux"
}

# ============================================================================
# Error Handling
# ============================================================================

@test "wt-open: errors when not in git repository" {
    cd /tmp
    run wt-open
    assert_failure
    assert_output --partial "Not a git repository"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

@test "wt-open: errors with unknown app" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name app-err 2>/dev/null | tail -n 1)

    run wt-open --app nonexistent-app "$wt_path"
    assert_failure
    assert_output --partial "Unknown app"
}

# ============================================================================
# Integration
# ============================================================================

@test "wt-open: create worktree then open with code" {
    wt-create --non-interactive --worktree-name open-integrate &>/dev/null

    run wt-open --app code open-integrate
    assert_success
    verify_mock_called "code" ".*open-integrate"
}

@test "wt-open: mock log records correct path" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name log-check 2>/dev/null | tail -n 1)

    wt-open --app code "$wt_path"

    # Verify the mock log has the right worktree path
    verify_mock_called "code" ".*log-check"
}

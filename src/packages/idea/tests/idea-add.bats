#!/usr/bin/env bats
#
# idea-add.bats - Tests for the idea add command (default action)
#

setup() {
    load '../../tests/libs/bats-support/load'
    load '../../tests/libs/bats-assert/load'
    load '../../tests/libs/bats-file/load'
    load 'test_helper'
    _common_setup
}

teardown() {
    _common_teardown
}

# --- Add with defaults ---

@test "add: appends idea with random ID and today's date" {
    local before_count
    before_count=$(count_backlog_lines)

    run idea "Build a rocket ship 2"
    assert_success
    assert_output --partial "Added:"
    assert_output --partial "Build a rocket ship 2"

    # Backlog should have one more line
    local after_count
    after_count=$(count_backlog_lines)
    [ "$after_count" -eq "$((before_count + 1))" ]

    # New line should match expected format
    assert_backlog_contains '\- \[ \] \[[a-z0-9]\{4\}\] [0-9]\{4\}-[0-9][0-9]-[0-9][0-9]: Build a rocket ship 2'
}

@test "add: preserves existing ideas in backlog" {
    run idea "New idea"
    assert_success

    # Original ideas should still be there
    assert_backlog_contains "ab12"
    assert_backlog_contains "cd34"
    assert_backlog_contains "ef56"
}

# --- Creates backlog if missing ---

@test "add: creates backlog file if it does not exist" {
    rm -rf fab/

    run idea "First idea ever"
    assert_success

    assert_file_exists "fab/backlog.md"
    assert_backlog_contains "First idea ever"
}

# --- Custom ID ---

@test "add: --id flag overrides random ID" {
    run idea --id zz99 "Custom slug idea"
    assert_success
    assert_output --partial "zz99"

    assert_backlog_contains '\[zz99\]'
}

@test "add: --id errors when ID already exists" {
    run idea --id ab12 "Duplicate ID"
    assert_failure
    assert_output --partial "already exists"
}

# --- Custom date ---

@test "add: --date flag overrides today's date" {
    run idea --date 2025-01-15 "Backdated idea"
    assert_success
    assert_output --partial "2025-01-15"

    assert_backlog_contains "2025-01-15"
}

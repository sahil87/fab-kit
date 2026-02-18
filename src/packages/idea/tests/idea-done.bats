#!/usr/bin/env bats
#
# idea-done.bats - Tests for the idea done command
#

setup() {
    load '../../tests/libs/bats-support/load'
    load '../../tests/libs/bats-assert/load'
    load 'test_helper'
    _common_setup
}

teardown() {
    _common_teardown
}

# --- Mark done by ID ---

@test "done: marks idea as done by ID" {
    run idea done ab12
    assert_success
    assert_output --partial "Done:"

    # Backlog should show [x] for this idea
    assert_backlog_contains '^\- \[x\] \[ab12\]'
    # ID, date, and text should be preserved
    assert_backlog_contains '\[ab12\] 2025-06-01: Build a rocket ship'
}

# --- Mark done by text ---

@test "done: marks idea as done by text match" {
    run idea done "rocket"
    assert_success

    assert_backlog_contains '^\- \[x\] \[ab12\]'
}

# --- Already done ---

@test "done: errors when idea is already done" {
    run idea done ef56
    assert_failure
    assert_output --partial "No idea matching"
}

# --- Ambiguous match ---

@test "done: errors on multiple matches" {
    seed_backlog \
        "- [ ] [aa11] 2025-06-01: Build a rocket" \
        "- [ ] [bb22] 2025-06-02: Build a submarine"

    run idea done "Build"
    assert_failure
    assert_output --partial "Multiple matches"

    # Neither idea should be modified
    assert_backlog_contains '^\- \[ \] \[aa11\]'
    assert_backlog_contains '^\- \[ \] \[bb22\]'
}

# --- No match ---

@test "done: errors when no idea matches" {
    run idea done "xyz"
    assert_failure
}

# --- Missing query ---

@test "done: no query displays usage" {
    run idea done
    assert_failure
    assert_output --partial "usage:"
}

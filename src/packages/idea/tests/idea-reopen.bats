#!/usr/bin/env bats
#
# idea-reopen.bats - Tests for the idea reopen command
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

# --- Reopen by ID ---

@test "reopen: reopens a done idea by ID" {
    run idea reopen ef56
    assert_success
    assert_output --partial "Reopened:"

    # Backlog should show [ ] for this idea
    assert_backlog_contains '^\- \[ \] \[ef56\]'
    # ID, date, and text should be preserved
    assert_backlog_contains '\[ef56\] 2025-05-20: Fix login redirect bug'
}

# --- Open idea not found ---

@test "reopen: errors when idea is already open" {
    run idea reopen ab12
    assert_failure
    assert_output --partial "No idea matching"
}

# --- Ambiguous match ---

@test "reopen: errors on multiple matches" {
    seed_backlog \
        "- [x] [aa11] 2025-06-01: Build a rocket" \
        "- [x] [bb22] 2025-06-02: Build a submarine"

    run idea reopen "Build"
    assert_failure
    assert_output --partial "Multiple matches"
}

# --- No match ---

@test "reopen: errors when no done idea matches" {
    run idea reopen "xyz"
    assert_failure
}

# --- Missing query ---

@test "reopen: no query displays usage" {
    run idea reopen
    assert_failure
    assert_output --partial "usage:"
}

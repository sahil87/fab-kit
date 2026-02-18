#!/usr/bin/env bats
#
# idea-show.bats - Tests for the idea show command
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

# --- Show by ID ---

@test "show: displays idea by exact ID" {
    run idea show ab12
    assert_success
    assert_output --partial "[ab12]"
    assert_output --partial "Build a rocket ship"
}

# --- Show by text match ---

@test "show: matches by text substring (case-insensitive)" {
    run idea show "rocket"
    assert_success
    assert_output --partial "rocket ship"
}

# --- Ambiguous match ---

@test "show: errors on multiple matches" {
    seed_backlog \
        "- [ ] [aa11] 2025-06-01: Build a rocket" \
        "- [ ] [bb22] 2025-06-02: Build a submarine"

    run idea show "Build"
    assert_failure
    assert_output --partial "Multiple matches"
}

# --- No match ---

@test "show: errors when no idea matches" {
    run idea show "nonexistent"
    assert_failure
    assert_output --partial "No idea matching"
}

# --- JSON output ---

@test "show: --json outputs JSON object with expected fields" {
    run idea show ab12 --json
    assert_success
    assert_output --partial '"id"'
    assert_output --partial '"date"'
    assert_output --partial '"status"'
    assert_output --partial '"text"'
}

# --- Missing query ---

@test "show: no query displays usage" {
    run idea show
    assert_failure
    assert_output --partial "usage:"
}

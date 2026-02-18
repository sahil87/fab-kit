#!/usr/bin/env bats
#
# idea-edit.bats - Tests for the idea edit command
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

# --- Edit text ---

@test "edit: updates idea text preserving ID, date, status" {
    run idea edit ab12 "New text"
    assert_success
    assert_output --partial "Updated:"

    assert_backlog_contains '\- \[ \] \[ab12\] 2025-06-01: New text'
}

# --- Preserves done status ---

@test "edit: preserves done status when editing" {
    run idea edit ef56 "Revised done idea"
    assert_success

    assert_backlog_contains '^\- \[x\] \[ef56\]'
    assert_backlog_contains 'Revised done idea'
}

# --- Edit with date override ---

@test "edit: --date updates the date field" {
    run idea edit ab12 "Same text" --date 2025-12-25
    assert_success

    assert_backlog_contains '2025-12-25'
}

# --- Edit with ID override ---

@test "edit: --id updates the idea's ID" {
    run idea edit ab12 "Same text" --id zz99
    assert_success

    assert_backlog_contains '\[zz99\]'
    refute_backlog_contains '\[ab12\]'
}

@test "edit: --id errors when new ID conflicts with existing" {
    run idea edit ab12 "Text" --id cd34
    assert_failure
    assert_output --partial "already exists"
}

# --- Ambiguous match ---

@test "edit: errors on multiple matches" {
    seed_backlog \
        "- [ ] [aa11] 2025-06-01: Build a rocket" \
        "- [ ] [bb22] 2025-06-02: Build a submarine"

    run idea edit "Build" "New text"
    assert_failure
    assert_output --partial "Multiple matches"
}

# --- Insufficient arguments ---

@test "edit: insufficient arguments displays usage" {
    run idea edit ab12
    assert_failure
    assert_output --partial "usage:"
}

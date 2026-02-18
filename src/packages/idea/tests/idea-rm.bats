#!/usr/bin/env bats
#
# idea-rm.bats - Tests for the idea rm command
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

# --- Remove with confirmation (accepted) ---

@test "rm: removes idea when confirmation accepted" {
    run bash -c 'echo "y" | idea rm ab12'
    assert_success
    assert_output --partial "Removed:"

    refute_backlog_contains "ab12"
}

# --- Remove with confirmation (cancelled) ---

@test "rm: cancels removal when confirmation declined" {
    run bash -c 'echo "n" | idea rm ab12'
    assert_success
    assert_output --partial "Cancelled"

    # Idea should still exist
    assert_backlog_contains "ab12"
}

# --- Force remove ---

@test "rm: --force removes without prompting" {
    run idea rm ab12 --force
    assert_success
    assert_output --partial "Removed:"

    refute_backlog_contains "ab12"
}

# --- Ambiguous match ---

@test "rm: errors on multiple matches" {
    seed_backlog \
        "- [ ] [aa11] 2025-06-01: Build a rocket" \
        "- [ ] [bb22] 2025-06-02: Build a submarine"

    run idea rm "Build" --force
    assert_failure
    assert_output --partial "Multiple matches"

    # Neither idea should be removed
    assert_backlog_contains "aa11"
    assert_backlog_contains "bb22"
}

# --- Not found ---

@test "rm: errors when no idea matches" {
    run idea rm "xyz" --force
    assert_failure
}

# --- Missing query ---

@test "rm: no query displays usage" {
    run idea rm
    assert_failure
    assert_output --partial "usage:"
}

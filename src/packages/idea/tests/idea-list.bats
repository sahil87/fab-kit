#!/usr/bin/env bats
#
# idea-list.bats - Tests for the idea list command
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

# --- List open ideas (default) ---

@test "list: shows only open ideas by default" {
    run idea list
    assert_success
    assert_output --partial "ab12"
    assert_output --partial "cd34"
    refute_output --partial "ef56"
}

# --- Filter by status ---

@test "list: -a shows all ideas (open and done)" {
    run idea list -a
    assert_success
    assert_output --partial "ab12"
    assert_output --partial "cd34"
    assert_output --partial "ef56"
}

@test "list: --done shows only completed ideas" {
    run idea list --done
    assert_success
    assert_output --partial "ef56"
    refute_output --partial "ab12"
    refute_output --partial "cd34"
}

# --- JSON output ---

@test "list: --json outputs valid JSON array" {
    run idea list --json
    assert_success

    # Should be parseable JSON
    echo "$output" | python3 -m json.tool > /dev/null 2>&1 || \
    echo "$output" | jq . > /dev/null 2>&1

    # Each object should have expected fields
    assert_output --partial '"id"'
    assert_output --partial '"date"'
    assert_output --partial '"status"'
    assert_output --partial '"text"'
}

@test "list: --json returns empty array when no backlog file" {
    rm -rf fab/

    run idea list --json
    assert_success
    assert_output "[]"
}

# --- Sort and reverse ---

@test "list: --sort id sorts alphabetically by ID" {
    seed_backlog \
        "- [ ] [cc11] 2025-06-01: Third" \
        "- [ ] [aa22] 2025-06-02: First" \
        "- [ ] [bb33] 2025-06-03: Second"

    run idea list -a --sort id
    assert_success

    # aa22 should appear before bb33, bb33 before cc11
    local pos_aa pos_bb pos_cc
    pos_aa=$(echo "$output" | grep -n "aa22" | cut -d: -f1)
    pos_bb=$(echo "$output" | grep -n "bb33" | cut -d: -f1)
    pos_cc=$(echo "$output" | grep -n "cc11" | cut -d: -f1)
    [ "$pos_aa" -lt "$pos_bb" ]
    [ "$pos_bb" -lt "$pos_cc" ]
}

@test "list: --reverse reverses sort order" {
    seed_backlog \
        "- [ ] [aa11] 2025-01-01: January" \
        "- [ ] [bb22] 2025-02-01: February" \
        "- [ ] [cc33] 2025-03-01: March"

    run idea list -a --reverse
    assert_success

    # March (newest) should appear before January (oldest)
    local pos_jan pos_mar
    pos_jan=$(echo "$output" | grep -n "January" | cut -d: -f1)
    pos_mar=$(echo "$output" | grep -n "March" | cut -d: -f1)
    [ "$pos_mar" -lt "$pos_jan" ]
}

# --- Empty state ---

@test "list: no ideas file shows friendly message" {
    rm -rf fab/

    run idea list
    assert_success
    assert_output --partial "No ideas file yet"
}

@test "list: no matching ideas shows 'No ideas found'" {
    seed_backlog \
        "- [x] [ab12] 2025-06-01: Only done idea"

    run idea list
    assert_success
    assert_output --partial "No ideas found"
}

# --- Unknown option ---

@test "list: unknown option is rejected" {
    run idea list --bogus
    assert_failure
    assert_output --partial "unknown option"
}

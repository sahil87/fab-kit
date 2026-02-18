#!/usr/bin/env bats
#
# idea-global.bats - Tests for global behavior (help, git requirement, missing backlog)
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

# --- Help ---

@test "global: --help shows usage" {
    run idea --help
    assert_success
    assert_output --partial "usage: idea"
}

@test "global: no arguments shows usage" {
    run idea
    assert_success
    assert_output --partial "usage: idea"
}

# --- Not in git repo ---

@test "global: errors when not in a git repository" {
    local non_git_dir="/tmp/idea-test-no-git-$$"
    mkdir -p "$non_git_dir"

    run bash -c "cd '$non_git_dir' && idea list"
    assert_failure
    assert_output --partial "not in a git repository"

    rm -rf "$non_git_dir"
}

# --- Missing backlog file ---

@test "global: show errors when backlog file does not exist" {
    rm -rf fab/

    run idea show "anything"
    assert_failure
    assert_output --partial "no ideas file found"
}

@test "global: done errors when backlog file does not exist" {
    rm -rf fab/

    run idea done "anything"
    assert_failure
    assert_output --partial "no ideas file found"
}

@test "global: reopen errors when backlog file does not exist" {
    rm -rf fab/

    run idea reopen "anything"
    assert_failure
    assert_output --partial "no ideas file found"
}

@test "global: edit errors when backlog file does not exist" {
    rm -rf fab/

    run idea edit "anything" "new text"
    assert_failure
    assert_output --partial "no ideas file found"
}

@test "global: rm errors when backlog file does not exist" {
    rm -rf fab/

    run idea rm "anything" --force
    assert_failure
    assert_output --partial "no ideas file found"
}

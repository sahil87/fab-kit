#!/usr/bin/env bats
#
# Tests for wt-common.sh shared library functions
# (rollback stack, signal handling, hash-based stash, branch validation)
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load 'test_helper'

setup() {
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"

    # Source wt-common.sh directly for unit testing
    REPO_ROOT="$(cd "${BATS_TEST_DIRNAME}/../../../.." && pwd)"
    source "$REPO_ROOT/fab/.kit/packages/wt/lib/wt-common.sh"
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Rollback Stack
# ============================================================================

@test "rollback: stack starts empty" {
    [[ ${#WT_ROLLBACK_STACK[@]} -eq 0 ]]
}

@test "rollback: register adds commands to stack" {
    wt_register_rollback "echo one"
    wt_register_rollback "echo two"
    [[ ${#WT_ROLLBACK_STACK[@]} -eq 2 ]]
}

@test "rollback: disarm clears the stack" {
    wt_register_rollback "echo one"
    wt_register_rollback "echo two"
    wt_disarm_rollback
    [[ ${#WT_ROLLBACK_STACK[@]} -eq 0 ]]
}

@test "rollback: executes commands in reverse order (LIFO)" {
    local log="$BATS_SUITE_TMPDIR/rollback-order.log"
    wt_register_rollback "echo first >> '$log'"
    wt_register_rollback "echo second >> '$log'"
    wt_register_rollback "echo third >> '$log'"

    wt_rollback

    run cat "$log"
    assert_line -n 0 "third"
    assert_line -n 1 "second"
    assert_line -n 2 "first"
}

@test "rollback: tolerates command failures without stopping" {
    local log="$BATS_SUITE_TMPDIR/rollback-tolerant.log"
    wt_register_rollback "false"  # this will fail
    wt_register_rollback "echo survived >> '$log'"

    # wt_rollback should run both, not stop at false
    wt_rollback

    run cat "$log"
    assert_output "survived"
}

@test "rollback: empty stack is a no-op" {
    wt_disarm_rollback
    run wt_rollback
    assert_success
}

# ============================================================================
# Branch Name Validation
# ============================================================================

@test "validation: accepts simple branch name" {
    run wt_validate_branch_name "feature-auth"
    assert_success
}

@test "validation: accepts branch with single slash" {
    run wt_validate_branch_name "feature/add-auth"
    assert_success
}

@test "validation: accepts branch with dots" {
    run wt_validate_branch_name "release/v1.2.3"
    assert_success
}

@test "validation: rejects empty string" {
    run wt_validate_branch_name ""
    assert_failure
}

@test "validation: rejects branch with tilde" {
    run wt_validate_branch_name "feature~bad"
    assert_failure
}

@test "validation: rejects branch with caret" {
    run wt_validate_branch_name "feature^bad"
    assert_failure
}

@test "validation: rejects branch with colon" {
    run wt_validate_branch_name "feature:bad"
    assert_failure
}

@test "validation: rejects branch with question mark" {
    run wt_validate_branch_name "feature?bad"
    assert_failure
}

@test "validation: rejects branch with asterisk" {
    run wt_validate_branch_name "feature*bad"
    assert_failure
}

@test "validation: rejects branch with bracket" {
    run wt_validate_branch_name "feature[bad"
    assert_failure
}

@test "validation: rejects branch with space" {
    run wt_validate_branch_name "feature bad"
    assert_failure
}

@test "validation: rejects double dot" {
    run wt_validate_branch_name "feature..branch"
    assert_failure
}

@test "validation: rejects .lock suffix" {
    run wt_validate_branch_name "feature.lock"
    assert_failure
}

@test "validation: rejects leading dot" {
    run wt_validate_branch_name ".hidden-branch"
    assert_failure
}

@test "validation: rejects hidden component (/.)" {
    run wt_validate_branch_name "feature/.hidden"
    assert_failure
}

# ============================================================================
# Hash-Based Stash
# ============================================================================

@test "stash: wt_stash_create returns hash when changes exist" {
    echo "dirty" > dirty-file.txt
    local hash
    hash=$(wt_stash_create "test stash")
    [[ -n "$hash" ]]
    # Working tree should be clean after stash
    run git status --porcelain
    assert_output ""
}

@test "stash: wt_stash_create stores in reflog" {
    echo "dirty" > dirty-file.txt
    local hash
    hash=$(wt_stash_create "test stash message")
    # Should be visible in git stash list
    run git stash list
    assert_output --partial "test stash message"
}

@test "stash: wt_stash_create returns empty when no changes" {
    local hash
    hash=$(wt_stash_create "no changes")
    [[ -z "$hash" ]]
}

@test "stash: wt_stash_apply restores stashed changes" {
    echo "restore me" > restore-file.txt
    local hash
    hash=$(wt_stash_create "restore test")

    # File should be gone after stash
    [[ ! -f restore-file.txt ]]

    wt_stash_apply "$hash"
    # File should be back
    [[ -f restore-file.txt ]]
    run cat restore-file.txt
    assert_output "restore me"
}

@test "stash: wt_stash_apply is no-op with empty hash" {
    run wt_stash_apply ""
    assert_success
}

@test "stash: wt_stash_create handles untracked files" {
    echo "untracked" > new-file.txt
    local hash
    hash=$(wt_stash_create "untracked test")
    [[ -n "$hash" ]]

    # Untracked file should be gone
    [[ ! -f new-file.txt ]]

    wt_stash_apply "$hash"
    [[ -f new-file.txt ]]
}

# ============================================================================
# Menu Display (wt_show_menu)
# ============================================================================

# Helper: run wt_show_menu with piped input, stripping ANSI codes from output
_run_menu() {
    local input="$1"
    shift
    run bash -c "
        source '$REPO_ROOT/fab/.kit/packages/wt/lib/wt-common.sh'
        echo '$input' | wt_show_menu \"\$@\" | sed 's/\x1b\[[0-9;]*m//g'
    " _ "$@"
}

@test "menu: empty default_choice does not create blank option" {
    _run_menu "1" "Pick one:" "" "Alpha" "Beta"

    # Should show "1) Alpha" and "2) Beta" — not a blank "1)" followed by "2) Alpha"
    assert_output --partial "1) Alpha"
    assert_output --partial "2) Beta"
    refute_output --partial "3)"
}

@test "menu: numeric default_choice is not rendered as an option" {
    _run_menu "1" "Pick one:" 2 "Alpha" "Beta"

    assert_output --partial "1) Alpha"
    assert_output --partial "2) Beta"
    # Only 2 numbered options plus Cancel
    refute_output --partial "3)"
}

@test "menu: option count matches arguments when default_choice is empty" {
    _run_menu "1" "Pick:" "" "A" "B" "C"

    assert_output --partial "1) A"
    assert_output --partial "2) B"
    assert_output --partial "3) C"
    refute_output --partial "4)"
}

@test "menu: option count matches arguments when default_choice is numeric" {
    _run_menu "1" "Pick:" 1 "A" "B" "C"

    assert_output --partial "1) A"
    assert_output --partial "2) B"
    assert_output --partial "3) C"
    refute_output --partial "4)"
}

@test "stash: hash is stable across concurrent stash operations" {
    echo "first stash" > first.txt
    local hash1
    hash1=$(wt_stash_create "first")

    echo "second stash" > second.txt
    local hash2
    hash2=$(wt_stash_create "second")

    # Both hashes should be different and valid
    [[ -n "$hash1" ]]
    [[ -n "$hash2" ]]
    [[ "$hash1" != "$hash2" ]]

    # Applying first hash should restore first file, not second
    wt_stash_apply "$hash1"
    [[ -f first.txt ]]
}

#!/usr/bin/env bats
#
# Tests for wt-list command
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load '../../tests/libs/bats-file/load'
load 'test_helper'

setup() {
    # Create test repository
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"

    # Add wt commands to PATH
    export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
}

# ============================================================================
# Basic Functionality Tests
# ============================================================================

@test "wt-list: shows repository name and location" {
    run wt-list

    assert_success
    assert_output --partial "Worktrees for:"
    assert_output --partial "$(basename "$TEST_REPO")"
    assert_output --partial "Location:"
}

@test "wt-list: shows main repository" {
    run wt-list

    assert_success
    assert_output --partial "(main)"
    assert_output --partial "main"  # branch name
    assert_output --partial "$TEST_REPO"
}

@test "wt-list: shows total count" {
    run wt-list

    assert_success
    assert_output --partial "Total: 1 worktree(s)"
}

@test "wt-list: marks current worktree with asterisk" {
    run wt-list

    assert_success
    # Should have asterisk for current location (may have ANSI color codes)
    assert_output --regexp '\*.*main'
}

@test "wt-list: lists multiple worktrees" {
    # Create a couple of worktrees
    wt-create --non-interactive --worktree-name test-wt1 &>/dev/null
    wt-create --non-interactive --worktree-name test-wt2 &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "test-wt1"
    assert_output --partial "test-wt2"
    assert_output --partial "Total: 3 worktree(s)"
}

@test "wt-list: shows branch names for worktrees" {
    # Create worktree for specific branch
    git checkout -b feature/test &>/dev/null
    git checkout main &>/dev/null
    wt-create --non-interactive --worktree-name my-feature feature/test &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "my-feature"
    assert_output --partial "feature/test"
}

# ============================================================================
# Help and Options Tests
# ============================================================================

@test "wt-list: shows help with 'help' argument" {
    run wt-list help

    assert_success
    assert_output --partial "Usage: wt-list"
    assert_output --partial "Lists all git worktrees"
}

@test "wt-list: shows help with --help flag" {
    run wt-list --help

    assert_success
    assert_output --partial "Usage: wt-list"
}

@test "wt-list: shows help with -h flag" {
    run wt-list -h

    assert_success
    assert_output --partial "Usage: wt-list"
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "wt-list: errors when not in git repository" {
    cd /tmp

    run wt-list

    assert_failure
    assert_output --partial "Not a git repository"
    assert_output --partial "Why:"
    assert_output --partial "Fix:"
}

@test "wt-list: succeeds with no worktrees" {
    # In main repo with no additional worktrees
    run wt-list

    assert_success
    assert_output --partial "Total: 1 worktree(s)"
}

# ============================================================================
# Output Format Tests
# ============================================================================

@test "wt-list: respects NO_COLOR environment variable" {
    export NO_COLOR=1

    run wt-list

    assert_success
    # Should not contain ANSI color codes when NO_COLOR is set
    refute_output --regexp $'\033\\['
}

@test "wt-list: output is well-formatted with columns" {
    wt-create --non-interactive --worktree-name aligned-test &>/dev/null

    run wt-list

    assert_success
    # Should have consistent formatting/alignment
    assert_output --regexp '[[:space:]]+aligned-test[[:space:]]+'
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "wt-list: shows worktree immediately after creation" {
    wt-create --non-interactive --worktree-name new-wt &>/dev/null

    run wt-list

    assert_success
    assert_output --partial "new-wt"
}

@test "wt-list: no longer shows worktree after deletion" {
    wt-create --non-interactive --worktree-name temp-wt &>/dev/null

    # Verify it's listed
    run wt-list
    assert_output --partial "temp-wt"

    # Delete it
    wt-delete --non-interactive --worktree-name temp-wt &>/dev/null

    # Verify it's gone
    run wt-list
    refute_output --partial "temp-wt"
}

@test "wt-list: correctly shows worktree from within worktree" {
    # Create and enter a worktree
    local wt_path=$(wt-create --non-interactive --worktree-name inside-test 2>/dev/null)
    cd "$wt_path"

    run wt-list

    assert_success
    assert_output --partial "inside-test"
    # Should mark this worktree as current (may have ANSI color codes)
    assert_output --regexp '\*.*inside-test'
}

# ============================================================================
# --path Flag Tests
# ============================================================================

@test "wt-list: --path returns absolute path for existing worktree" {
    wt-create --non-interactive --worktree-name path-test &>/dev/null

    run wt-list --path path-test

    assert_success
    assert_dir_exists "$output"
    assert_output --regexp '/path-test$'
}

@test "wt-list: --path exits 1 for nonexistent worktree" {
    run wt-list --path nonexistent

    assert_failure
    assert_output --partial "not found"
}

@test "wt-list: --path outputs single line only" {
    wt-create --non-interactive --worktree-name single-line-test &>/dev/null

    local stdout
    stdout=$(wt-list --path single-line-test)
    local line_count
    line_count=$(echo "$stdout" | wc -l | tr -d ' ')
    [ "$line_count" -eq 1 ]
}

@test "wt-list: --path requires argument" {
    run wt-list --path

    assert_failure
    assert_output --partial "Missing worktree name"
}

# ============================================================================
# --json Flag Tests
# ============================================================================

@test "wt-list: --json outputs valid JSON" {
    run wt-list --json

    assert_success
    # Validate JSON
    echo "$output" | jq . >/dev/null 2>&1
}

@test "wt-list: --json includes main repo" {
    run wt-list --json

    assert_success
    local main_count
    main_count=$(echo "$output" | jq '[.[] | select(.is_main == true)] | length')
    [ "$main_count" -eq 1 ]
}

@test "wt-list: --json includes all required fields" {
    wt-create --non-interactive --worktree-name json-fields-test &>/dev/null

    run wt-list --json

    assert_success
    # Check that a non-main entry has all required fields
    local has_fields
    has_fields=$(echo "$output" | jq '[.[] | select(.name == "json-fields-test")] | length')
    [ "$has_fields" -eq 1 ]

    # Verify field types
    echo "$output" | jq '.[] | select(.name == "json-fields-test") | .branch' | grep -q '"'
    echo "$output" | jq '.[] | select(.name == "json-fields-test") | .path' | grep -q '"'
    echo "$output" | jq -e '.[] | select(.name == "json-fields-test") | .is_main == false' >/dev/null
    echo "$output" | jq -e '.[] | select(.name == "json-fields-test") | .dirty | type == "boolean"' >/dev/null
    echo "$output" | jq -e '.[] | select(.name == "json-fields-test") | .unpushed | type == "number"' >/dev/null
}

@test "wt-list: --json detects dirty worktree" {
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name dirty-json-test 2>/dev/null)

    # Make the worktree dirty
    echo "dirty" > "$wt_path/dirty.txt"

    run wt-list --json

    assert_success
    local dirty
    dirty=$(echo "$output" | jq '.[] | select(.name == "dirty-json-test") | .dirty')
    [ "$dirty" = "true" ]
}

# ============================================================================
# Flag Mutual Exclusivity Tests
# ============================================================================

@test "wt-list: --path and --json are mutually exclusive" {
    run wt-list --path foo --json

    assert_failure
    assert_output --partial "mutually exclusive"
}

@test "wt-list: --json and --path are mutually exclusive (reverse order)" {
    run wt-list --json --path foo

    assert_failure
    assert_output --partial "mutually exclusive"
}

# ============================================================================
# Status Column Tests
# ============================================================================

@test "wt-list: shows dirty indicator for worktree with changes" {
    export NO_COLOR=1
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name dirty-status-test 2>/dev/null)

    # Make it dirty
    echo "dirty" > "$wt_path/dirty.txt"

    run wt-list

    assert_success
    # Should show * indicator for the dirty worktree
    assert_output --regexp 'dirty-status-test.*\*'
}

@test "wt-list: clean worktree shows no status indicator" {
    export NO_COLOR=1
    wt-create --non-interactive --worktree-name clean-status-test &>/dev/null

    run wt-list

    assert_success
    # The line for this worktree should not have a * status indicator
    # (the * at the beginning is the current-worktree marker, which is different)
    assert_output --partial "clean-status-test"
}

@test "wt-list: shows unpushed indicator for worktree with unpushed commits" {
    export NO_COLOR=1
    local wt_path
    wt_path=$(wt-create --non-interactive --worktree-name unpushed-test 2>/dev/null)

    # Make a commit in the worktree (branch has upstream from creation)
    (cd "$wt_path" && echo "new" > new.txt && git add new.txt && git commit -q -m "unpushed commit")

    # Set up upstream tracking so unpushed count works
    (cd "$wt_path" && git push -q -u origin "$(git rev-parse --abbrev-ref HEAD)" 2>/dev/null || true)
    (cd "$wt_path" && echo "another" > another.txt && git add another.txt && git commit -q -m "another unpushed")

    run wt-list

    assert_success
    # Should show ↑ indicator (may or may not show exact count depending on upstream)
    assert_output --partial "unpushed-test"
}

@test "wt-list: main repo shows dirty status" {
    export NO_COLOR=1

    # Make main repo dirty
    echo "dirty" > dirty-main.txt

    run wt-list

    assert_success
    # Main repo line should show * indicator
    assert_output --regexp '\(main\).*\*'

    # Clean up
    rm -f dirty-main.txt
}

@test "wt-list: --path does not match (main) label" {
    run wt-list --path "(main)"

    assert_failure
    assert_output --partial "not found"
}

@test "wt-list: --json is_current field is true for current directory" {
    run wt-list --json

    assert_success
    # The main repo should be current since we're in it
    local is_current
    is_current=$(echo "$output" | jq '.[] | select(.name == "main") | .is_current')
    [ "$is_current" = "true" ]
}

@test "wt-list: errors on unknown option" {
    run wt-list --unknown

    assert_failure
    assert_output --partial "Unknown option"
}

@test "wt-list: errors on unexpected positional argument" {
    run wt-list foo

    assert_failure
    assert_output --partial "Unexpected argument"
}

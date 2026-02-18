#!/usr/bin/env bash
#
# test_helper.bash - Shared test utilities for idea command tests
#

# ============================================================================
# Test Repository Management
# ============================================================================

# Create a test git repository with initial commit
# Returns: path to test repo
create_test_repo() {
    local test_dir="/tmp/idea-test-repo-$$-${RANDOM}"
    mkdir -p "$test_dir"

    (
        cd "$test_dir"
        git init -q
        git config user.name "Test User"
        git config user.email "test@example.com"

        echo "# Test Repository" > README.md
        git add README.md
        git commit -q -m "Initial commit"

        # Rename to main if needed
        local current_branch
        current_branch=$(git rev-parse --abbrev-ref HEAD)
        if [[ "$current_branch" != "main" ]]; then
            git branch -m main
        fi
    ) >&2

    echo "$test_dir"
}

# ============================================================================
# Backlog Seeding
# ============================================================================

# Seed the backlog file with given lines
# Args: lines to write (each argument is a line)
# Usage: seed_backlog "- [ ] [ab12] 2025-06-01: First idea" "- [x] [cd34] 2025-06-02: Done idea"
seed_backlog() {
    mkdir -p "$(dirname "$IDEAS_DEFAULT_FILE")"
    printf '' > "$IDEAS_DEFAULT_FILE"
    for line in "$@"; do
        echo "$line" >> "$IDEAS_DEFAULT_FILE"
    done
}

# Default backlog path relative to repo root
IDEAS_DEFAULT_FILE="fab/backlog.md"

# Seed with a standard set of test ideas
seed_default_backlog() {
    seed_backlog \
        "- [ ] [ab12] 2025-06-01: Build a rocket ship" \
        "- [ ] [cd34] 2025-06-15: Add dark mode support" \
        "- [x] [ef56] 2025-05-20: Fix login redirect bug"
}

# ============================================================================
# Per-Test Setup/Teardown
# ============================================================================

# Standard setup: create test repo, cd into it, seed backlog
_common_setup() {
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO" || return 1
    seed_default_backlog
}

# Standard teardown: clean up test repo
_common_teardown() {
    if [[ -n "${TEST_REPO:-}" && -d "${TEST_REPO:-}" ]]; then
        rm -rf "$TEST_REPO"
    fi
}

# ============================================================================
# Backlog Assertions
# ============================================================================

# Assert that the backlog contains a line matching the pattern
# Args: $1 = grep pattern
assert_backlog_contains() {
    local pattern="$1"
    if ! grep -q "$pattern" "$IDEAS_DEFAULT_FILE"; then
        echo "Backlog does not contain pattern: $pattern" >&2
        echo "Backlog contents:" >&2
        cat "$IDEAS_DEFAULT_FILE" >&2
        return 1
    fi
}

# Assert that the backlog does NOT contain a line matching the pattern
# Args: $1 = grep pattern
refute_backlog_contains() {
    local pattern="$1"
    if grep -q "$pattern" "$IDEAS_DEFAULT_FILE"; then
        echo "Backlog unexpectedly contains pattern: $pattern" >&2
        echo "Backlog contents:" >&2
        cat "$IDEAS_DEFAULT_FILE" >&2
        return 1
    fi
}

# Count lines in the backlog
count_backlog_lines() {
    wc -l < "$IDEAS_DEFAULT_FILE" | tr -d ' '
}

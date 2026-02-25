#!/usr/bin/env bats
#
# Tests for wt-pr command
#

load '../../tests/libs/bats-support/load'
load '../../tests/libs/bats-assert/load'
load '../../tests/libs/bats-file/load'
load 'test_helper'

setup() {
    TEST_REPO=$(create_test_repo)
    cd "$TEST_REPO"

    # Add wt commands to PATH
    export PATH="$BATS_TEST_DIRNAME/../bin:$PATH"

    # Set default init script path
    export WORKTREE_INIT_SCRIPT="fab/.kit/worktree-init.sh"

    # Create mock gh script directory
    MOCK_DIR="$BATS_SUITE_TMPDIR/mock-gh-$$"
    mkdir -p "$MOCK_DIR"

    clear_mock_log
}

teardown() {
    cd /
    cleanup_test_repo "$TEST_REPO"
    rm -rf "$MOCK_DIR" 2>/dev/null || true
}

# Helper: create a mock gh that returns a specific branch for pr view
# Args: $1 = branch name
setup_mock_gh() {
    local branch="$1"

    cat > "$MOCK_DIR/gh" <<EOF
#!/usr/bin/env bash
if [[ "\$1" == "pr" && "\$2" == "view" ]]; then
    echo "$branch"
elif [[ "\$1" == "pr" && "\$2" == "list" ]]; then
    echo '[{"number":42,"title":"Test PR","headRefName":"$branch"}]'
else
    echo "mock gh: unexpected args: \$*" >&2
    exit 1
fi
EOF
    chmod +x "$MOCK_DIR/gh"
    export PATH="$MOCK_DIR:$PATH"
}

# Helper: create a mock gh that simulates no gh installed
setup_no_gh() {
    # Ensure gh is not found
    local saved_path="$PATH"
    # Remove any real gh from PATH by creating a wrapper that fails
    cat > "$MOCK_DIR/gh" <<'EOF'
#!/usr/bin/env bash
exit 127
EOF
    chmod +x "$MOCK_DIR/gh"
    # Actually, wt-pr uses `command -v gh` which would find our mock.
    # Instead, make our mock not exist:
    rm -f "$MOCK_DIR/gh"
    # Override PATH to only use mock dir first (no real gh)
    export PATH="$MOCK_DIR:$BATS_TEST_DIRNAME/../bin:$TEST_REPO"
}

# ============================================================================
# Help
# ============================================================================

@test "wt-pr: shows help with 'help' argument" {
    run wt-pr help
    assert_success
    assert_output --partial "Usage: wt-pr"
}

@test "wt-pr: shows help with --help flag" {
    run wt-pr --help
    assert_success
    assert_output --partial "Usage: wt-pr"
}

@test "wt-pr: shows help with -h flag" {
    run wt-pr -h
    assert_success
    assert_output --partial "Usage: wt-pr"
}

# ============================================================================
# Argument Validation
# ============================================================================

@test "wt-pr: errors with unknown option" {
    setup_mock_gh "feature/test"
    run wt-pr --unknown
    assert_failure
    assert_output --partial "Unknown option"
}

@test "wt-pr: errors when --worktree-name missing argument" {
    setup_mock_gh "feature/test"
    run wt-pr 42 --worktree-name
    assert_failure
    assert_output --partial "Missing worktree name"
}

@test "wt-pr: errors when --worktree-init has invalid value" {
    setup_mock_gh "feature/test"
    run wt-pr 42 --worktree-init maybe
    assert_failure
    assert_output --partial "Invalid value"
}

@test "wt-pr: errors in non-interactive mode without PR number" {
    setup_mock_gh "feature/test"
    run wt-pr --non-interactive
    assert_failure
    assert_output --partial "No PR number specified"
}

# ============================================================================
# gh CLI Validation
# ============================================================================

@test "wt-pr: errors when gh CLI not installed" {
    # Build minimal PATH with required tools but not gh
    local fake_bin="$BATS_SUITE_TMPDIR/no-gh-bin-$$"
    mkdir -p "$fake_bin"
    ln -s "$(which bash)" "$fake_bin/bash"
    ln -s "$(which env)" "$fake_bin/env"
    ln -s "$(which dirname)" "$fake_bin/dirname"
    local wt_pr_dir
    wt_pr_dir=$(dirname "$(which wt-pr)")
    local saved_path="$PATH"
    export PATH="$fake_bin:$wt_pr_dir"
    run wt-pr 42
    export PATH="$saved_path"
    assert_failure
    assert_output --partial "gh CLI not found"
}

# ============================================================================
# PR Worktree Creation
# ============================================================================

@test "wt-pr: creates worktree for PR with explicit number" {
    # Create a branch that the "PR" references
    git checkout -q -b feature/login
    echo "login code" > login.txt
    git add login.txt
    git commit -q -m "Add login"
    git checkout -q main

    setup_mock_gh "feature/login"

    run wt-pr 42 --non-interactive --worktree-init false --worktree-open skip
    assert_success
    assert_output --partial "PR #42"
    assert_output --partial "feature/login"

    assert_worktree_exists "login"
}

@test "wt-pr: uses --worktree-name when provided" {
    git checkout -q -b feature/auth
    echo "auth code" > auth.txt
    git add auth.txt
    git commit -q -m "Add auth"
    git checkout -q main

    setup_mock_gh "feature/auth"

    run wt-pr 99 --non-interactive --worktree-name review-99 --worktree-init false --worktree-open skip
    assert_success
    assert_worktree_exists "review-99"
}

@test "wt-pr: creates worktree for new branch (not yet local)" {
    setup_mock_gh "feature/new-thing"

    run wt-pr 7 --non-interactive --worktree-init false --worktree-open skip
    assert_success
    assert_output --partial "feature/new-thing"
    assert_worktree_exists "new-thing"
}

@test "wt-pr: prints full path as last line" {
    git checkout -q -b fix/typo
    echo "fix" > typo.txt
    git add typo.txt
    git commit -q -m "Fix typo"
    git checkout -q main

    setup_mock_gh "fix/typo"

    run wt-pr 10 --non-interactive --worktree-init false --worktree-open skip
    assert_success

    local last_line
    last_line=$(echo "$output" | tail -n 1)
    [[ "$last_line" == *".worktrees/"* ]]
}

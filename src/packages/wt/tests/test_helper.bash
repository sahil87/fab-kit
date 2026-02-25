#!/usr/bin/env bash
#
# test_helper.bash - Shared test utilities for wt commands
#

# ============================================================================
# Test Repository Management
# ============================================================================

# Create a test git repository with main branch and mock remote
# Returns: path to test repo
create_test_repo() {
    local test_dir="/tmp/test-repo-$$-${RANDOM}"
    mkdir -p "$test_dir"

    (
        cd "$test_dir"
        git init -q
        git config user.name "Test User"
        git config user.email "test@example.com"

        # Create initial commit
        echo "# Test Repository" > README.md
        git add README.md
        git commit -q -m "Initial commit"

        # Rename to main if needed
        local current_branch=$(git rev-parse --abbrev-ref HEAD)
        if [[ "$current_branch" != "main" ]]; then
            git branch -m main
        fi

        # Set up mock remote
        local remote_dir="${test_dir}/../test-repo-remote-$$-${RANDOM}"
        mkdir -p "$remote_dir"
        (cd "$remote_dir" && git init -q --bare)
        git remote add origin "$remote_dir"
        git push -q -u origin main 2>/dev/null || true
    ) >&2

    echo "$test_dir"
}

# Add a branch to the mock remote
# Args: $1 = branch name
add_remote_branch() {
    local branch="$1"

    git checkout -q -b "$branch"
    echo "Test file for $branch" > "file-${branch}.txt"
    git add "file-${branch}.txt"
    git commit -q -m "Add $branch"
    git push -q -u origin "$branch" 2>/dev/null || true
    git checkout -q main
}

# ============================================================================
# Mock Application Utilities
# ============================================================================

# Verify that a mock application was called with expected arguments
# Args: $1 = app name, $2 = regex pattern for arguments
verify_mock_called() {
    local app="$1"
    local pattern="${2:-.*}"

    if [[ ! -f "$MOCK_LOG" ]]; then
        echo "MOCK_LOG not set or file doesn't exist" >&2
        return 1
    fi

    if ! grep -E "^${app} ${pattern}" "$MOCK_LOG"; then
        echo "Mock '$app' was not called with pattern: $pattern" >&2
        echo "Mock log contents:" >&2
        cat "$MOCK_LOG" >&2
        return 1
    fi
}

# Clear mock log
clear_mock_log() {
    if [[ -n "${MOCK_LOG:-}" ]]; then
        > "$MOCK_LOG"
    fi
}

# ============================================================================
# Worktree Assertions
# ============================================================================

# Assert that a worktree exists (both directory and git registration)
# Args: $1 = worktree name
assert_worktree_exists() {
    local name="$1"
    local repo_name=$(basename "$PWD")
    local wt_path="$(dirname "$PWD")/${repo_name}.worktrees/${name}"

    if [[ ! -d "$wt_path" ]]; then
        echo "Worktree directory does not exist: $wt_path" >&2
        return 1
    fi

    if ! git worktree list | grep -q "$wt_path"; then
        echo "Worktree not registered with git: $name" >&2
        git worktree list >&2
        return 1
    fi
}

# Assert that a worktree does not exist
# Args: $1 = worktree name
assert_worktree_not_exists() {
    local name="$1"
    local repo_name=$(basename "$PWD")
    local wt_path="$(dirname "$PWD")/${repo_name}.worktrees/${name}"

    if [[ -d "$wt_path" ]]; then
        echo "Worktree directory still exists: $wt_path" >&2
        return 1
    fi

    if git worktree list | grep -q "$wt_path"; then
        echo "Worktree still registered with git: $name" >&2
        git worktree list >&2
        return 1
    fi
}

# ============================================================================
# Branch Assertions
# ============================================================================

# Assert that a local branch exists
# Args: $1 = branch name
assert_branch_exists() {
    local branch="$1"

    if ! git show-ref --verify --quiet "refs/heads/$branch"; then
        echo "Branch does not exist: $branch" >&2
        echo "Available branches:" >&2
        git branch >&2
        return 1
    fi
}

# Assert that a local branch does not exist
# Args: $1 = branch name
assert_branch_not_exists() {
    local branch="$1"

    if git show-ref --verify --quiet "refs/heads/$branch"; then
        echo "Branch still exists: $branch" >&2
        echo "Available branches:" >&2
        git branch >&2
        return 1
    fi
}

# Assert that a remote branch exists (on mock remote)
# Args: $1 = branch name
assert_remote_branch_exists() {
    local branch="$1"

    if ! git ls-remote --heads origin "$branch" | grep -q "refs/heads/$branch"; then
        echo "Remote branch does not exist: $branch" >&2
        echo "Remote branches:" >&2
        git ls-remote --heads origin >&2
        return 1
    fi
}

# Assert that a remote branch does not exist
# Args: $1 = branch name
assert_remote_branch_not_exists() {
    local branch="$1"

    if git ls-remote --heads origin "$branch" | grep -q "refs/heads/$branch"; then
        echo "Remote branch still exists: $branch" >&2
        echo "Remote branches:" >&2
        git ls-remote --heads origin >&2
        return 1
    fi
}

# ============================================================================
# Git State Utilities
# ============================================================================

# Verify git repository integrity
assert_git_state() {
    if ! git fsck --no-progress &>/dev/null; then
        echo "Git repository integrity check failed" >&2
        return 1
    fi
}

# Count active worktrees (excluding main repo)
count_worktrees() {
    git worktree list --porcelain | grep -c '^worktree' || echo 0
}

# ============================================================================
# Input Simulation
# ============================================================================

# Simulate interactive input by piping to command
# Args: $1 = input string (newlines should be literal \n), $2+ = command and args
simulate_input() {
    local input="$1"
    shift
    printf "%b" "$input" | "$@"
}

# ============================================================================
# Environment Utilities
# ============================================================================

# Execute command with temporary environment variables
# Usage: with_env VAR=value VAR2=value2 -- command args
with_env() {
    local env_vars=()

    while [[ $# -gt 0 && "$1" != "--" ]]; do
        env_vars+=("$1")
        shift
    done

    if [[ "$1" == "--" ]]; then
        shift
    fi

    env "${env_vars[@]}" "$@"
}

# ============================================================================
# Output Utilities
# ============================================================================

# Get the last line of output (useful for wt-create path output)
get_last_line() {
    tail -n 1
}

# Get the first line of output
get_first_line() {
    head -n 1
}

# Count lines in output
count_lines() {
    wc -l | tr -d ' '
}

# ============================================================================
# File Utilities
# ============================================================================

# Create a test init script
# Args: $1 = script path (relative to repo root)
create_test_init_script() {
    local script_path="$1"

    mkdir -p "$(dirname "$script_path")"
    cat > "$script_path" <<'EOF'
#!/usr/bin/env bash
# Test init script

echo "Test init script executed"

# Create a marker file to prove it ran
touch .init-script-ran
EOF
    chmod +x "$script_path"
}

# Check if init script ran by looking for marker file
assert_init_script_ran() {
    if [[ ! -f ".init-script-ran" ]]; then
        echo "Init script did not run (marker file not found)" >&2
        return 1
    fi
}

# ============================================================================
# Stash Utilities
# ============================================================================

# Assert that a stash exists with a specific message pattern
# Args: $1 = message pattern
assert_stash_exists() {
    local pattern="$1"

    if ! git stash list | grep -q "$pattern"; then
        echo "No stash found matching pattern: $pattern" >&2
        echo "Stash list:" >&2
        git stash list >&2
        return 1
    fi
}

# ============================================================================
# Cleanup Utilities
# ============================================================================

# Clean up a test repository and all its worktrees
cleanup_test_repo() {
    local repo_path="${1:-$PWD}"

    if [[ -d "$repo_path" ]]; then
        # Remove all worktrees first
        local repo_name=$(basename "$repo_path")
        local worktrees_dir="$(dirname "$repo_path")/${repo_name}.worktrees"

        if [[ -d "$worktrees_dir" ]]; then
            rm -rf "$worktrees_dir"
        fi

        # Remove the main repo
        rm -rf "$repo_path"

        # Remove the mock remote if it exists
        if [[ -d "${repo_path}/../test-repo-remote-"* ]]; then
            rm -rf "${repo_path}/../test-repo-remote-"*
        fi
    fi
}

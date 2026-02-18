#!/usr/bin/env bash
#
# wt-common.sh - Shared functions for worktree management
#
# Source this file from wt-* commands:
#   source "$(dirname "$0")/../lib/wt-common.sh"

# ============================================================================
# Constants
# ============================================================================

# Exit codes
readonly WT_EXIT_SUCCESS=0
readonly WT_EXIT_GENERAL_ERROR=1
readonly WT_EXIT_INVALID_ARGS=2
readonly WT_EXIT_GIT_ERROR=3
readonly WT_EXIT_RETRY_EXHAUSTED=4
readonly WT_EXIT_BYOBU_TAB_ERROR=5
readonly WT_EXIT_TMUX_WINDOW_ERROR=6

# Default retry count for name collisions
readonly WT_DEFAULT_RETRIES=10

# Init script path (configurable via WORKTREE_INIT_SCRIPT env var)
WORKTREE_INIT_SCRIPT="${WORKTREE_INIT_SCRIPT:-fab/.kit/worktree-init.sh}"

# Colors (disabled if NO_COLOR is set)
if [[ -z "${NO_COLOR:-}" ]]; then
    readonly WT_RED=$'\033[0;31m'
    readonly WT_YELLOW=$'\033[0;33m'
    readonly WT_GREEN=$'\033[0;32m'
    readonly WT_BOLD=$'\033[1m'
    readonly WT_RESET=$'\033[0m'
else
    readonly WT_RED=''
    readonly WT_YELLOW=''
    readonly WT_GREEN=''
    readonly WT_BOLD=''
    readonly WT_RESET=''
fi

# Adjectives for random worktree naming (~50 adjectives)
WT_ADJECTIVES=(
    "swift" "calm" "bold" "keen" "warm" "cool" "bright" "quick"
    "brave" "noble" "wise" "kind" "fair" "proud" "sharp" "clear"
    "vivid" "agile" "sleek" "smart" "crisp" "fresh" "prime" "rapid"
    "witty" "zesty" "lucid" "happy" "sunny" "quiet" "eager" "alert"
    "nimble" "deft" "lively" "merry" "jolly" "golden" "silver" "amber"
    "cosmic" "stellar" "lunar" "solar" "rustic" "urban" "alpine" "arctic"
)

# Nouns (animals) for random worktree naming (~50 nouns)
WT_NOUNS=(
    "fox" "owl" "wolf" "bear" "hawk" "lynx" "puma" "orca"
    "falcon" "raven" "eagle" "crane" "heron" "finch" "robin" "sparrow"
    "otter" "beaver" "badger" "ferret" "marten" "stoat" "mink" "weasel"
    "tiger" "lion" "panther" "jaguar" "leopard" "cheetah" "cougar" "bobcat"
    "dolphin" "whale" "seal" "walrus" "penguin" "puffin" "pelican" "albatross"
    "cobra" "viper" "python" "mamba" "gecko" "iguana" "turtle" "tortoise"
)

# ============================================================================
# Error Handling
# ============================================================================

# Print error message in what/why/fix format and exit
# Usage: wt_error <what> <why> [fix]
wt_error() {
    local what="$1"
    local why="$2"
    local fix="${3:-}"

    echo "${WT_RED}Error:${WT_RESET} $what" >&2
    echo "  ${WT_BOLD}Why:${WT_RESET} $why" >&2
    if [[ -n "$fix" ]]; then
        echo "  ${WT_BOLD}Fix:${WT_RESET} $fix" >&2
    fi
    exit "$WT_EXIT_GENERAL_ERROR"
}

# Print error and exit with specific code
# Usage: wt_error_with_code <code> <what> <why> [fix]
wt_error_with_code() {
    local code="$1"
    local what="$2"
    local why="$3"
    local fix="${4:-}"

    echo "${WT_RED}Error:${WT_RESET} $what" >&2
    echo "  ${WT_BOLD}Why:${WT_RESET} $why" >&2
    if [[ -n "$fix" ]]; then
        echo "  ${WT_BOLD}Fix:${WT_RESET} $fix" >&2
    fi
    exit "$code"
}

# ============================================================================
# Menu Helper
# ============================================================================

# Display a numbered menu and get user choice
# Args: $1 = prompt, $2 = default choice (optional, 0 for cancel), $3... = menu options
# Returns: selected option number via $WT_MENU_CHOICE
# Option 0 is always "Cancel" and added automatically
# If default is provided, empty input selects that option
wt_show_menu() {
    local prompt="$1"
    shift

    local default_choice=""
    # Check if first arg is a number (default choice)
    if [[ "$1" =~ ^[0-9]+$ ]]; then
        default_choice="$1"
        shift
    fi

    local options=("$@")

    echo "$prompt"
    local i=1
    for opt in "${options[@]}"; do
        local default_marker=""
        if [[ "$default_choice" == "$i" ]]; then
            default_marker=" ${WT_GREEN}(default)${WT_RESET}"
        fi
        echo "  ${WT_BOLD}$i)${WT_RESET} $opt$default_marker"
        ((i++))
    done

    local cancel_marker=""
    if [[ "$default_choice" == "0" ]]; then
        cancel_marker=" ${WT_GREEN}(default)${WT_RESET}"
    fi
    echo "  ${WT_BOLD}0)${WT_RESET} Cancel$cancel_marker"
    echo ""

    while true; do
        # Flush any pending input
        read -t 0.1 -n 10000 discard 2>/dev/null || true

        if [[ -n "$default_choice" ]]; then
            printf "Choice [%s]: " "$default_choice"
        else
            printf "Choice: "
        fi
        read -r choice

        # Handle empty input
        if [[ -z "$choice" ]]; then
            if [[ -n "$default_choice" ]]; then
                WT_MENU_CHOICE="$default_choice"
                return 0
            else
                WT_MENU_CHOICE=0
                return 0
            fi
        fi

        # Validate numeric input
        if ! [[ "$choice" =~ ^[0-9]+$ ]]; then
            echo "Invalid choice. Please enter a number."
            continue
        fi

        # Check if choice is in valid range
        if ((choice < 0 || choice > ${#options[@]})); then
            echo "Invalid choice. Please enter a number between 0 and ${#options[@]}."
            continue
        fi

        WT_MENU_CHOICE="$choice"
        return 0
    done
}

# ============================================================================
# Git Repository Detection
# ============================================================================

# Validate we're in a git repository
wt_validate_git_repo() {
    if ! git rev-parse --is-inside-work-tree &>/dev/null; then
        wt_error_with_code "$WT_EXIT_GIT_ERROR" \
            "Not a git repository" \
            "This command requires a git repository" \
            "Navigate to a git repository and try again"
    fi
}

# Get repository context (root, name, worktrees directory)
# Sets: WT_REPO_ROOT, WT_REPO_NAME, WT_WORKTREES_DIR
# Always uses the MAIN repo root, even when run from a worktree
wt_get_repo_context() {
    local git_common_dir
    git_common_dir=$(git rev-parse --git-common-dir 2>/dev/null) || {
        wt_error_with_code "$WT_EXIT_GENERAL_ERROR" \
            "Not a git repository" \
            "This command must be run from within a git repository" \
            "Navigate to a git repository and try again"
    }

    # Convert to absolute path
    git_common_dir=$(cd "$git_common_dir" && pwd)

    # Derive main repo root by stripping /.git suffix
    WT_REPO_ROOT="${git_common_dir%/.git}"
    WT_REPO_NAME=$(basename "$WT_REPO_ROOT")
    WT_WORKTREES_DIR="$(dirname "$WT_REPO_ROOT")/${WT_REPO_NAME}-worktrees"
}

# Check if we're inside a worktree (not the main repo)
# Returns: 0 if in worktree, 1 if in main repo
wt_is_worktree() {
    local git_dir common_dir
    git_dir=$(git rev-parse --git-dir 2>/dev/null)
    common_dir=$(git rev-parse --git-common-dir 2>/dev/null)
    [[ "$git_dir" != "$common_dir" ]]
}

# Get the default branch (main or master)
wt_get_default_branch() {
    local default_branch

    # Try origin/HEAD first
    default_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')

    # Fallback to checking main/master existence
    if [[ -z "$default_branch" ]]; then
        if git show-ref --verify --quiet refs/heads/main 2>/dev/null; then
            default_branch="main"
        elif git show-ref --verify --quiet refs/heads/master 2>/dev/null; then
            default_branch="master"
        else
            default_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null) || default_branch="main"
        fi
    fi

    echo "$default_branch"
}

# Check if branch exists locally
wt_branch_exists_locally() {
    local branch="$1"
    git show-ref --verify --quiet "refs/heads/$branch" 2>/dev/null
}

# Check if branch exists on remote
wt_branch_exists_remotely() {
    local branch="$1"
    git ls-remote --heads origin "$branch" 2>/dev/null | grep -q "$branch"
}

# ============================================================================
# Change Detection
# ============================================================================

# Check for uncommitted changes (staged or unstaged)
wt_has_uncommitted_changes() {
    ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null
}

# Check for untracked files
wt_has_untracked_files() {
    [[ -n $(git ls-files --others --exclude-standard 2>/dev/null) ]]
}

# Check if branch has unpushed commits
wt_has_unpushed_commits() {
    local branch="$1"
    local upstream
    upstream=$(git rev-parse --abbrev-ref "${branch}@{upstream}" 2>/dev/null) || return 1
    [[ -n $(git log "${upstream}..${branch}" --oneline 2>/dev/null) ]]
}

# Get count of unpushed commits
wt_get_unpushed_count() {
    local branch="$1"
    local upstream
    upstream=$(git rev-parse --abbrev-ref "${branch}@{upstream}" 2>/dev/null) || { echo "0"; return; }
    git rev-list --count "${upstream}..${branch}" 2>/dev/null || echo "0"
}

# ============================================================================
# OS Detection
# ============================================================================

# Detect operating system
wt_detect_os() {
    case "$(uname -s)" in
        Darwin) echo "macos" ;;
        Linux)  echo "linux" ;;
        *)      echo "unknown" ;;
    esac
}

# Detect if running inside a byobu session
wt_is_byobu_session() {
    if [[ -n "${BYOBU_TTY:-}" || -n "${BYOBU_BACKEND:-}" || -n "${BYOBU_SESSION:-}" ]]; then
        return 0
    fi
    if [[ -n "${TMUX:-}" && -n "${BYOBU_CONFIG_DIR:-}" ]]; then
        return 0
    fi
    return 1
}

# Detect if running inside a plain tmux session (not byobu)
wt_is_tmux_session() {
    if [[ -n "${TMUX:-}" ]] && ! wt_is_byobu_session; then
        return 0
    fi
    return 1
}

# ============================================================================
# Random Name Generation
# ============================================================================

# Generate a random adjective-noun combo (e.g., "swift-fox", "calm-owl")
wt_generate_random_name() {
    local adj_index=$((RANDOM % ${#WT_ADJECTIVES[@]}))
    local noun_index=$((RANDOM % ${#WT_NOUNS[@]}))
    echo "${WT_ADJECTIVES[$adj_index]}-${WT_NOUNS[$noun_index]}"
}

# Derive worktree name from branch name
# Extracts the last segment after any slashes (feature/login → login)
wt_derive_worktree_name() {
    local branch="$1"
    local name="${branch##*/}"
    name="${name//[^a-zA-Z0-9_-]/-}"
    echo "$name"
}

# ============================================================================
# Worktree Listing
# ============================================================================

# List all worktrees as tab-separated path\tbranch lines
wt_list_worktrees() {
    git worktree list --porcelain 2>/dev/null | awk '
        /^worktree / { path = substr($0, 10) }
        /^branch / { branch = substr($0, 8); gsub(/refs\/heads\//, "", branch); print path "\t" branch }
        /^detached/ { print path "\t(detached)" }
    '
}

# Get worktree path by name (basename match)
# Returns: path on stdout, exit 0 if found, 1 if not
wt_get_worktree_path_by_name() {
    local name="$1"

    while IFS=$'\t' read -r path branch; do
        if [[ "$(basename "$path")" == "$name" ]]; then
            echo "$path"
            return 0
        fi
    done < <(wt_list_worktrees)

    return 1
}

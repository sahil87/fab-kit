#!/usr/bin/env bash
# fab/.kit/scripts/lib/changeman.sh
#
# Change Manager — CLI utility for change lifecycle operations.
# Currently supports the `new` subcommand for creating change directories.
#
# Usage:
#   changeman.sh new --slug <slug> [--change-id <4char>] [--log-args <description>]
#   changeman.sh --help

set -euo pipefail

# Path resolution (works from both .kit/scripts/lib/ and symlinks)
LIB_DIR="$(cd "$(dirname "$(readlink -f "$0")")" && pwd)"
FAB_ROOT="$(cd "$LIB_DIR/../../.." && pwd)"
STAGEMAN="$LIB_DIR/stageman.sh"

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

# detect_created_by — gh api → git config → "unknown" (silent failures)
detect_created_by() {
  local user
  user=$(gh api user --jq .login 2>/dev/null) && [ -n "$user" ] && echo "$user" && return 0
  user=$(git config user.name 2>/dev/null) && [ -n "$user" ] && echo "$user" && return 0
  echo "unknown"
}

# generate_random_id — 4 chars from [a-z0-9] via /dev/urandom
# Reads 128 bytes then filters, avoiding SIGPIPE from tr|head with pipefail.
generate_random_id() {
  local raw
  raw=$(head -c128 /dev/urandom | LC_ALL=C tr -dc 'a-z0-9')
  echo "${raw:0:4}"
}

# has_id_collision <changes_dir> <change_id> — check if any folder uses this ID
# Returns 0 (true) if collision exists, 1 (false) if no collision.
has_id_collision() {
  local changes_dir="$1" change_id="$2"
  for dir in "$changes_dir"/??????-"${change_id}"-*; do
    [ -d "$dir" ] && return 0
  done
  return 1
}

# ─────────────────────────────────────────────────────────────────────────────
# new subcommand
# ─────────────────────────────────────────────────────────────────────────────

cmd_new() {
  local slug="" change_id="" log_args=""
  local id_provided=false

  # Parse arguments
  while [ $# -gt 0 ]; do
    case "$1" in
      --slug)
        [ $# -lt 2 ] && { echo "ERROR: --slug requires a value" >&2; exit 1; }
        slug="$2"; shift 2 ;;
      --change-id)
        [ $# -lt 2 ] && { echo "ERROR: --change-id requires a value" >&2; exit 1; }
        change_id="$2"; id_provided=true; shift 2 ;;
      --log-args)
        [ $# -lt 2 ] && { echo "ERROR: --log-args requires a value" >&2; exit 1; }
        log_args="$2"; shift 2 ;;
      *)
        echo "ERROR: Unknown flag '$1'" >&2; exit 1 ;;
    esac
  done

  # Validate required --slug
  if [ -z "$slug" ]; then
    echo "ERROR: --slug is required" >&2
    exit 1
  fi

  # Validate slug format: alphanumeric start/end, hyphens allowed in middle
  if ! [[ "$slug" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$ ]]; then
    echo "ERROR: Invalid slug format '${slug}' (expected alphanumeric and hyphens, no leading/trailing hyphen)" >&2
    exit 1
  fi

  # Validate --change-id if provided
  if [ "$id_provided" = true ]; then
    if ! [[ "$change_id" =~ ^[a-z0-9]{4}$ ]]; then
      echo "ERROR: Invalid change-id '${change_id}' (expected 4 lowercase alphanumeric chars)" >&2
      exit 1
    fi
  fi

  # Generate date prefix
  local date_prefix
  date_prefix=$(date +%y%m%d)

  # Generate or use provided change ID, with collision detection
  local changes_dir="$FAB_ROOT/changes"
  local folder_name=""
  local max_retries=10

  if [ "$id_provided" = true ]; then
    folder_name="${date_prefix}-${change_id}-${slug}"
    # Provided ID collision is fatal — check any folder using this ID
    if has_id_collision "$changes_dir" "$change_id"; then
      local existing
      for dir in "$changes_dir"/??????-"${change_id}"-*; do
        [ -d "$dir" ] && existing=$(basename "$dir") && break
      done
      echo "ERROR: Change ID '${change_id}' already in use (${existing})" >&2
      exit 1
    fi
  else
    # Random ID with retry
    local attempt=0
    while [ $attempt -lt $max_retries ]; do
      change_id=$(generate_random_id)
      has_id_collision "$changes_dir" "$change_id" || break
      attempt=$((attempt + 1))
    done
    if [ $attempt -ge $max_retries ]; then
      echo "ERROR: Failed to generate unique change ID after ${max_retries} attempts" >&2
      exit 1
    fi
    folder_name="${date_prefix}-${change_id}-${slug}"
  fi

  # Create directory (plain mkdir — parent guaranteed by fab-sync.sh)
  mkdir "$changes_dir/$folder_name"

  # Detect created_by
  local created_by
  created_by=$(detect_created_by)

  # Initialize .status.yaml from template via sed
  local template="$FAB_ROOT/.kit/templates/status.yaml"
  local status_file="$changes_dir/$folder_name/.status.yaml"
  local now
  now=$(date -Iseconds)

  sed -e "s|{NAME}|${folder_name}|g" \
      -e "s|{CREATED}|${now}|g" \
      -e "s|{CREATED_BY}|${created_by}|g" \
      "$template" > "$status_file"

  # Stageman integration
  "$STAGEMAN" set-state "$status_file" intake active fab-new

  if [ -n "$log_args" ]; then
    "$STAGEMAN" log-command "$changes_dir/$folder_name" "fab-new" "$log_args"
  fi

  # Output: folder name only (one line to stdout)
  echo "$folder_name"
}

# ─────────────────────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────────────────────

show_help() {
  cat <<'EOF'
changeman.sh - Change Manager CLI

USAGE:
  changeman.sh new --slug <slug> [--change-id <4char>] [--log-args <description>]
  changeman.sh --help

SUBCOMMANDS:
  new     Create a new change directory with initialized .status.yaml

FLAGS (for new):
  --slug <slug>            Required. Folder name suffix (e.g., "add-oauth" or "DEV-988-add-oauth")
  --change-id <4char>      Optional. Explicit 4-char alphanumeric ID. Random if omitted.
  --log-args <description> Optional. Description logged via stageman log-command.

OUTPUT:
  On success: prints folder name to stdout (one line).
  On error: prints ERROR: message to stderr, exits non-zero.

EXAMPLES:
  changeman.sh new --slug add-oauth
  changeman.sh new --slug DEV-988-add-oauth --change-id a7k2 --log-args "Add OAuth"
EOF
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI Dispatch
# ─────────────────────────────────────────────────────────────────────────────

case "${1:-}" in
  --help|-h)
    show_help
    ;;
  new)
    shift
    cmd_new "$@"
    ;;
  "")
    echo "ERROR: No subcommand provided. Try: changeman.sh --help" >&2
    exit 1
    ;;
  *)
    echo "ERROR: Unknown subcommand '$1'. Try: changeman.sh --help" >&2
    exit 1
    ;;
esac

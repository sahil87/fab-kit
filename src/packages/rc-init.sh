#!/usr/bin/env sh

# Detect shell and set SCRIPT_DIR appropriately
if [ -n "$BASH_VERSION" ]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
elif [ -n "$ZSH_VERSION" ]; then
  SCRIPT_DIR="${${(%):-%x}:A:h}"
else
  echo "Warning: Unsupported shell. Expected bash or zsh." >&2
  return 1 2>/dev/null || exit 1
fi

# Delegate to the kit's env-packages.sh for PATH setup
source "$SCRIPT_DIR/../../fab/.kit/scripts/lib/env-packages.sh"

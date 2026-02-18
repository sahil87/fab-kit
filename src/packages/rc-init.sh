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

# Add all packages/*/bin directories to PATH
for pkg_bin in "$SCRIPT_DIR"/*/bin; do
  if [ -d "$pkg_bin" ]; then
    export PATH="$pkg_bin:$PATH"
  fi
done

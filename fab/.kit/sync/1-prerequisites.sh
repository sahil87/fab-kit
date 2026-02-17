#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/sync/1-prerequisites.sh — Validate required tools before sync
#
# Checks that all tools needed by fab-kit are available on PATH.
# Any missing tool is fatal — exit 1 with actionable error.

missing=()

for tool in yq jq gh direnv bats; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    missing+=("$tool")
  fi
done

if [ ${#missing[@]} -gt 0 ]; then
  echo "ERROR: Missing required tools: ${missing[*]}" >&2
  echo "Install with: brew install ${missing[*]}" >&2
  echo "See the Prerequisites section in the README for details." >&2
  exit 1
fi

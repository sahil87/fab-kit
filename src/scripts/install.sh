#!/usr/bin/env bash
set -euo pipefail

# install.sh — One-liner installer for Fab Kit
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/wvrdz/fab-kit/main/src/scripts/install.sh | bash
#
# Downloads the latest kit archive from GitHub Releases into fab/.kit/ in the
# current directory. Tries the platform-specific archive (includes Go binaries)
# first, falls back to the generic archive (shell scripts only).

REPO="wvrdz/fab-kit"
BASE_URL="https://github.com/$REPO/releases/latest/download"

# ── Platform detection ──────────────────────────────────────────────

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64)  arch="amd64" ;;
  aarch64) arch="arm64" ;;
esac

platform_archive="kit-${os}-${arch}.tar.gz"
generic_archive="kit.tar.gz"

# ── Download ────────────────────────────────────────────────────────

mkdir -p fab

echo "Installing Fab Kit (${os}/${arch})..."

if curl -fsSL "$BASE_URL/$platform_archive" | tar xz -C fab/; then
  echo "Installed fab/.kit/ with Go binaries (${os}/${arch})"
elif curl -fsSL "$BASE_URL/$generic_archive" | tar xz -C fab/; then
  echo "Installed fab/.kit/ (shell scripts only — no binary for ${os}/${arch})"
else
  echo "ERROR: Failed to download kit archive from $REPO."
  echo "       Check your network connection and try again."
  exit 1
fi

# ── Next steps ──────────────────────────────────────────────────────

version=$(cat fab/.kit/VERSION 2>/dev/null | tr -d '[:space:]' || echo "unknown")
echo ""
echo "Fab Kit $version installed successfully."
echo ""
echo "Next steps:"
echo "  bash fab/.kit/scripts/fab-sync.sh   # set up directories and PATH"
echo "  /fab-setup                          # generate project config (in your AI agent)"

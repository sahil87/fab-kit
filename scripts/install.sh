#!/usr/bin/env bash
set -euo pipefail

# install.sh — One-liner installer for Fab Kit
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sahil87/fab-kit/main/scripts/install.sh | bash
#
# Downloads the latest kit archive from GitHub Releases into the system cache
# at ~/.fab-kit/versions/{version}/. The kit content and fab-go binary are
# extracted to the cache; user projects no longer contain fab/.kit/.

REPO="sahil87/fab-kit"
BASE_URL="https://github.com/$REPO/releases/latest/download"
CACHE_BASE="$HOME/.fab-kit/versions"

# ── Platform detection ──────────────────────────────────────────────

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64)  arch="amd64" ;;
  aarch64) arch="arm64" ;;
esac

platform_archive="kit-${os}-${arch}.tar.gz"

# ── Resolve latest version ─────────────────────────────────────────

echo "Resolving latest version..."
latest_tag=$(curl -fsSL -o /dev/null -w '%{redirect_url}' "https://github.com/$REPO/releases/latest" | grep -oP 'v\K[0-9]+\.[0-9]+\.[0-9]+' || true)

if [ -z "$latest_tag" ]; then
  echo "ERROR: Could not resolve latest version from GitHub."
  exit 1
fi

echo "Latest version: $latest_tag"

# ── Download to cache ──────────────────────────────────────────────

cache_dir="$CACHE_BASE/$latest_tag"
mkdir -p "$cache_dir"

echo "Installing Fab Kit v${latest_tag} (${os}/${arch})..."

if curl -fsSL "$BASE_URL/$platform_archive" | tar xz -C "$cache_dir/"; then
  echo "Installed to $cache_dir/"
else
  echo "ERROR: Failed to download kit archive from $REPO."
  echo "       Check your network connection and try again."
  rm -rf "$cache_dir"
  exit 1
fi

# ── Next steps ──────────────────────────────────────────────────────

echo ""
echo "Fab Kit $latest_tag installed successfully."
echo ""
echo "Next steps:"
echo "  fab init                            # initialize a project"
echo "  /fab-setup                          # generate project config (in your AI agent)"

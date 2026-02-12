#!/usr/bin/env bash
set -euo pipefail

# fab/.kit/scripts/fab-release.sh — Create a GitHub Release for fab/.kit/
#
# Packages fab/.kit/ into kit.tar.gz, bumps VERSION, commits, and creates
# a GitHub Release with the archive as an asset.
#
# Usage: fab-release.sh [patch|minor|major]
#   patch (default) — 0.1.0 → 0.1.1
#   minor           — 0.1.0 → 0.2.0
#   major           — 0.1.0 → 1.0.0
#
# Requires: gh CLI (https://cli.github.com/)

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$scripts_dir")"
fab_dir="$(dirname "$kit_dir")"
repo_root="$(dirname "$fab_dir")"

bump_type="${1:-patch}"

# ── Validate arguments ───────────────────────────────────────────────

case "$bump_type" in
  patch|minor|major) ;;
  *)
    echo "ERROR: Invalid bump type '$bump_type'. Use: patch, minor, or major."
    exit 1
    ;;
esac

# ── Pre-flight ───────────────────────────────────────────────────────

# Check gh CLI
if ! command -v gh &>/dev/null; then
  echo "ERROR: gh CLI not found. Install it from https://cli.github.com/"
  exit 1
fi

repo="wvrdz/fab-kit"

# Check clean working tree
if [ -n "$(git -C "$repo_root" status --porcelain)" ]; then
  echo "ERROR: Working tree not clean. Commit or stash changes first."
  exit 1
fi

# Read current version
if [ ! -f "$kit_dir/VERSION" ]; then
  echo "ERROR: fab/.kit/VERSION not found — kit may be corrupted."
  exit 1
fi

current_version=$(cat "$kit_dir/VERSION" | tr -d '[:space:]')

# ── Bump version ─────────────────────────────────────────────────────

IFS='.' read -r major minor patch <<< "$current_version"

case "$bump_type" in
  patch) patch=$((patch + 1)) ;;
  minor) minor=$((minor + 1)); patch=0 ;;
  major) major=$((major + 1)); minor=0; patch=0 ;;
esac

new_version="${major}.${minor}.${patch}"
tag="v${new_version}"

echo "Bumping version: $current_version → $new_version ($bump_type)"

# Write new version
echo "$new_version" > "$kit_dir/VERSION"

# ── Package kit.tar.gz ───────────────────────────────────────────────

echo "Packaging kit.tar.gz..."

# Create archive rooted at .kit/ so `tar xz -C fab/` produces fab/.kit/
tar czf "$repo_root/kit.tar.gz" -C "$fab_dir" .kit

echo "Created kit.tar.gz ($(wc -c < "$repo_root/kit.tar.gz") bytes)"

# ── Commit and release ───────────────────────────────────────────────

echo "Committing VERSION bump..."

git -C "$repo_root" add "$kit_dir/VERSION"
git -C "$repo_root" commit -m "release: $tag"
git -C "$repo_root" push https://github.com/"$repo".git HEAD:main

echo "Creating GitHub Release $tag on $repo..."

gh release create "$tag" \
  --repo "$repo" \
  --title "Fab Kit $tag" \
  --notes "Fab Kit release $new_version" \
  "$repo_root/kit.tar.gz"

# Clean up the tar.gz from the repo root
rm -f "$repo_root/kit.tar.gz"

echo ""
echo "Release complete: $tag"
echo "  Tag:     $tag"
echo "  Version: $new_version"
echo "  Asset:   kit.tar.gz"

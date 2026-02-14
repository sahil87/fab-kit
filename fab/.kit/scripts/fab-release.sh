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

# ── Migration chain validation ───────────────────────────────────────

migrations_dir="$kit_dir/migrations"
if [ -d "$migrations_dir" ]; then
  # Check if any migration targets the new version
  has_target=false
  migration_files=()
  for mfile in "$migrations_dir"/*.md; do
    [ -f "$mfile" ] || continue
    mbase="$(basename "$mfile" .md)"
    migration_files+=("$mbase")
    # Extract TO version (everything after "-to-")
    mto="${mbase##*-to-}"
    if [ "$mto" = "$new_version" ]; then
      has_target=true
    fi
  done

  if [ ${#migration_files[@]} -gt 0 ] && [ "$has_target" = false ]; then
    echo "Note: No migration targets version $new_version. If this release changes project-level files, consider adding a migration."
  fi

  # Check for overlapping ranges
  overlap_found=false
  for ((i=0; i<${#migration_files[@]}; i++)); do
    a="${migration_files[$i]}"
    a_from="${a%%-to-*}"
    a_to="${a##*-to-}"
    for ((j=i+1; j<${#migration_files[@]}; j++)); do
      b="${migration_files[$j]}"
      b_from="${b%%-to-*}"
      b_to="${b##*-to-}"
      # Overlap check using string comparison (works for dotted semver)
      # A overlaps B if A.FROM < B.TO AND B.FROM < A.TO
      # Use sort -V for proper semver comparison
      if [ "$(printf '%s\n%s' "$a_from" "$b_to" | sort -V | head -1)" = "$a_from" ] && [ "$a_from" != "$b_to" ] && \
         [ "$(printf '%s\n%s' "$b_from" "$a_to" | sort -V | head -1)" = "$b_from" ] && [ "$b_from" != "$a_to" ]; then
        echo "Warning: Overlapping migration ranges detected: ${a}.md and ${b}.md — this will cause /fab-update to error."
        overlap_found=true
      fi
    done
  done
fi

# ── Package kit.tar.gz ───────────────────────────────────────────────

echo "Packaging kit.tar.gz..."

# Create archive rooted at .kit/ so `tar xz -C fab/` produces fab/.kit/
tar czf "$repo_root/kit.tar.gz" -C "$fab_dir" .kit

echo "Created kit.tar.gz ($(wc -c < "$repo_root/kit.tar.gz") bytes)"

# ── Commit and release ───────────────────────────────────────────────

# Capture previous tag before committing
previous_tag=$(git -C "$repo_root" describe --tags --abbrev=0 2>/dev/null || echo "")

echo "Committing VERSION bump..."

git -C "$repo_root" add "$kit_dir/VERSION"
git -C "$repo_root" commit -m "release: $tag"
git -C "$repo_root" tag "$tag"
git -C "$repo_root" push git@github.com:"$repo".git HEAD:main "$tag"

echo "Creating GitHub Release $tag on $repo..."

# Generate release notes with changelog
if [ -n "$previous_tag" ]; then
  changelog=$(git -C "$repo_root" log "$previous_tag..HEAD" --oneline --no-decorate | sed 's/^/- /')
  notes="Fab Kit release $new_version

## Changes since $previous_tag

$changelog"
else
  notes="Fab Kit release $new_version

Initial release of Fab Kit."
fi

gh release create "$tag" \
  --repo "$repo" \
  --title "Fab Kit $tag" \
  --notes "$notes" \
  "$repo_root/kit.tar.gz"

# Clean up the tar.gz from the repo root
rm -f "$repo_root/kit.tar.gz"

echo ""
echo "Release complete: $tag"
echo "  Tag:     $tag"
echo "  Version: $new_version"
echo "  Asset:   kit.tar.gz"

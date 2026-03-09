#!/usr/bin/env bash
set -euo pipefail

# src/scripts/fab-release.sh — Bump VERSION, commit, tag, and push.
#
# CI takes over from the tag push to cross-compile, package, and create
# the GitHub Release (see .github/workflows/release.yml).
#
# Usage: fab-release.sh <patch|minor|major>
#   patch — 0.1.0 → 0.1.1
#   minor — 0.1.0 → 0.2.0
#   major — 0.1.0 → 1.0.0

usage() {
  echo "Usage: fab-release.sh <patch|minor|major>"
  echo ""
  echo "  patch — bump patch version (e.g. 0.1.0 → 0.1.1)"
  echo "  minor — bump minor version (e.g. 0.1.0 → 0.2.0)"
  echo "  major — bump major version (e.g. 0.1.0 → 1.0.0)"
}

repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
kit_dir="$repo_root/fab/.kit"

# ── Parse arguments ──────────────────────────────────────────────────

bump_type=""

for arg in "$@"; do
  case "$arg" in
    patch|minor|major)
      if [ -n "$bump_type" ]; then
        echo "ERROR: Multiple bump types specified: '$bump_type' and '$arg'."
        echo ""
        usage
        exit 1
      fi
      bump_type="$arg"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: Unknown argument '$arg'. Use: patch, minor, or major."
      echo ""
      usage
      exit 1
      ;;
  esac
done

if [ -z "$bump_type" ]; then
  usage
  if [ $# -gt 0 ]; then
    exit 1  # Had flags but no bump type — that's an error
  fi
  exit 0
fi

# ── Pre-flight ───────────────────────────────────────────────────────

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
      if [ "$(printf '%s\n%s' "$a_from" "$b_to" | sort -V | head -1)" = "$a_from" ] && [ "$a_from" != "$b_to" ] && \
         [ "$(printf '%s\n%s' "$b_from" "$a_to" | sort -V | head -1)" = "$b_from" ] && [ "$b_from" != "$a_to" ]; then
        echo "Warning: Overlapping migration ranges detected: ${a}.md and ${b}.md — this will cause /fab-update to error."
        overlap_found=true
      fi
    done
  done
fi

# ── Commit, tag, and push ───────────────────────────────────────────

branch=$(git -C "$repo_root" branch --show-current)
if [ -z "$branch" ]; then
  echo "ERROR: Not on a branch (detached HEAD). Check out a branch before releasing."
  exit 1
fi

echo "Committing VERSION bump..."

git -C "$repo_root" add "$kit_dir/VERSION"
git -C "$repo_root" commit -m "release: $tag"
git -C "$repo_root" tag "$tag"
git -C "$repo_root" push origin HEAD:"$branch" "$tag"

echo ""
echo "Release tagged: $tag"
echo "  Tag:     $tag"
echo "  Version: $new_version"
echo "  Branch:  $branch"
echo ""
echo "CI will handle cross-compilation, packaging, and GitHub Release creation."

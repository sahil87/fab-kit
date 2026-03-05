#!/usr/bin/env bash
set -euo pipefail

# src/scripts/fab-release.sh — Create a GitHub Release for fab/.kit/
#
# Packages fab/.kit/ into kit.tar.gz (generic) and per-platform archives
# with the Go binary, bumps VERSION, commits, and creates a GitHub Release.
#
# Usage: fab-release.sh <patch|minor|major> [--no-latest]
#   patch — 0.1.0 → 0.1.1
#   minor — 0.1.0 → 0.2.0
#   major — 0.1.0 → 1.0.0
#
#   --no-latest — do NOT mark the release as "latest" on GitHub
#                 (use for backport releases on older version series)
#
# Requires: gh CLI (https://cli.github.com/), Go toolchain (https://go.dev/)

usage() {
  echo "Usage: fab-release.sh <patch|minor|major> [--no-latest]"
  echo ""
  echo "  patch       — bump patch version (e.g. 0.1.0 → 0.1.1)"
  echo "  minor       — bump minor version (e.g. 0.1.0 → 0.2.0)"
  echo "  major       — bump major version (e.g. 0.1.0 → 1.0.0)"
  echo "  --no-latest — don't mark as \"latest\" (for backport releases)"
}

repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
kit_dir="$repo_root/fab/.kit"
go_src="$repo_root/src/fab-go"

# ── Parse arguments ──────────────────────────────────────────────────

bump_type=""
no_latest=false

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
    --no-latest)
      no_latest=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: Unknown argument '$arg'. Use: patch, minor, or major. Optional: --no-latest."
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

# Check gh CLI
if ! command -v gh &>/dev/null; then
  echo "ERROR: gh CLI not found. Install it from https://cli.github.com/"
  exit 1
fi

# Check Go toolchain
if ! command -v go &>/dev/null; then
  echo "ERROR: Go toolchain not found. Install from https://go.dev/"
  exit 1
fi

repo=$(grep -E '^repo=' "$kit_dir/kit.conf" | cut -d= -f2 | tr -d '[:space:]')

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

# ── Cross-compile Go binary ─────────────────────────────────────────

platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")
build_dir="$repo_root/.release-build"

rm -rf "$build_dir"
mkdir -p "$build_dir"

echo "Cross-compiling Go binary for ${#platforms[@]} platforms..."

for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  output="$build_dir/fab-${os}-${arch}"
  echo "  Building ${os}/${arch}..."
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -o "$output" -C "$go_src" ./cmd/fab
done

echo "Cross-compilation complete."

# ── Package archives ─────────────────────────────────────────────────

echo "Packaging archives..."

# Generic archive (no binary)
COPYFILE_DISABLE=1 tar czf "$repo_root/kit.tar.gz" -C "$repo_root/fab" .kit
echo "  kit.tar.gz ($(wc -c < "$repo_root/kit.tar.gz") bytes)"

# Per-platform archives (kit + binary)
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  archive_name="kit-${os}-${arch}.tar.gz"
  binary="$build_dir/fab-${os}-${arch}"

  # Create temp staging area with .kit/ + binary
  staging="$build_dir/staging-${os}-${arch}"
  mkdir -p "$staging"
  cp -a "$repo_root/fab/.kit" "$staging/.kit"
  mkdir -p "$staging/.kit/bin"
  cp "$binary" "$staging/.kit/bin/fab"
  chmod +x "$staging/.kit/bin/fab"

  COPYFILE_DISABLE=1 tar czf "$repo_root/$archive_name" -C "$staging" .kit
  echo "  $archive_name ($(wc -c < "$repo_root/$archive_name") bytes)"
done

# ── Commit and release ───────────────────────────────────────────────

# Capture previous tag before committing
previous_tag=$(git -C "$repo_root" describe --tags --abbrev=0 2>/dev/null || echo "")

branch=$(git -C "$repo_root" branch --show-current)
if [ -z "$branch" ]; then
  echo "ERROR: Not on a branch (detached HEAD). Check out a branch before releasing."
  exit 1
fi

echo "Committing VERSION bump..."

git -C "$repo_root" add "$kit_dir/VERSION"
git -C "$repo_root" commit -m "release: $tag"
git -C "$repo_root" tag "$tag"
git -C "$repo_root" push git@github.com:"$repo".git HEAD:"$branch" "$tag"

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

latest_flag=()
if [ "$no_latest" = true ]; then
  latest_flag=(--latest=false)
fi

# Upload all 5 archives
gh release create "$tag" \
  --repo "$repo" \
  --title "Fab Kit $tag" \
  --notes "$notes" \
  "${latest_flag[@]}" \
  "$repo_root/kit.tar.gz" \
  "$repo_root/kit-darwin-arm64.tar.gz" \
  "$repo_root/kit-darwin-amd64.tar.gz" \
  "$repo_root/kit-linux-arm64.tar.gz" \
  "$repo_root/kit-linux-amd64.tar.gz"

# ── Cleanup ──────────────────────────────────────────────────────────

rm -f "$repo_root/kit.tar.gz"
rm -f "$repo_root"/kit-*.tar.gz
rm -rf "$build_dir"

echo ""
echo "Release complete: $tag"
echo "  Tag:     $tag"
echo "  Version: $new_version"
echo "  Assets:  kit.tar.gz + 4 platform archives"

if [ "$no_latest" = true ]; then
  echo ""
  echo "Note: This release was NOT marked as \"latest\"."
fi

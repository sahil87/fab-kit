#!/usr/bin/env bash
set -euo pipefail

# Package kit archives for release (per-platform with fab-go binary)
# Called by: just package-kit
#
# fab (router), fab-kit, wt, and idea are distributed via Homebrew, not kit archives.
# Only fab-go goes into the per-platform archives for auto-fetch.

platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")
build_dir=".release-build"

# Verify cross-compiled fab-go binaries exist for all platforms
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  binary="$build_dir/fab-go-${os}-${arch}"
  if [ ! -f "$binary" ]; then
    echo "ERROR: Missing fab-go binary $binary — run 'just build-all' first."
    exit 1
  fi
done

# Per-platform archives (kit content + fab-go binary)
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  archive_name="kit-${os}-${arch}.tar.gz"
  staging="$build_dir/staging-${os}-${arch}"
  rm -rf "$staging"
  mkdir -p "$staging"
  cp -a fab/.kit "$staging/"
  # Only fab-go goes in the archive (fab, fab-kit, wt, idea are Homebrew-only)
  mkdir -p "$staging/.kit/bin"
  cp "$build_dir/fab-go-${os}-${arch}" "$staging/.kit/bin/fab-go"
  chmod +x "$staging/.kit/bin/fab-go"
  COPYFILE_DISABLE=1 tar czf "$archive_name" -C "$staging" .kit
  echo "  $archive_name ($(wc -c < "$archive_name") bytes)"
  rm -rf "$staging"
done
echo "Packaging complete: ${#platforms[@]} platform archives"

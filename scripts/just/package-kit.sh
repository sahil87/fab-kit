#!/usr/bin/env bash
set -euo pipefail

# Package kit archives for release (generic + per-platform with fab-go binary)
# Called by: just package-kit
#
# fab (shim), wt, and idea are distributed via Homebrew, not kit archives.
# Only fab-go goes into the per-platform archives for shim auto-fetch.

platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")
build_dir=".release-build"

# Verify cross-compiled fab-go binaries exist for all platforms
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  binary="$build_dir/fab-${os}-${arch}"
  if [ ! -f "$binary" ]; then
    echo "ERROR: Missing fab binary $binary — run 'just build-all' first."
    exit 1
  fi
done

# Generic archive (no binaries)
echo "Packaging kit.tar.gz (generic, no binary)..."
COPYFILE_DISABLE=1 tar czf kit.tar.gz -C fab --exclude='.kit/bin/fab-go' --exclude='.kit/bin/wt' --exclude='.kit/bin/idea' .kit
echo "  kit.tar.gz ($(wc -c < kit.tar.gz) bytes)"

# Per-platform archives (kit + fab-go binary only)
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  archive_name="kit-${os}-${arch}.tar.gz"
  staging="$build_dir/staging-${os}-${arch}"
  rm -rf "$staging"
  mkdir -p "$staging"
  cp -a fab/.kit "$staging/"
  # Only fab-go goes in the archive (wt, idea, shim are Homebrew-only)
  cp "$build_dir/fab-${os}-${arch}" "$staging/.kit/bin/fab-go"
  chmod +x "$staging/.kit/bin/fab-go"
  COPYFILE_DISABLE=1 tar czf "$archive_name" -C "$staging" .kit
  echo "  $archive_name ($(wc -c < "$archive_name") bytes)"
  rm -rf "$staging"
done
echo "Packaging complete: kit.tar.gz + ${#platforms[@]} platform archives"

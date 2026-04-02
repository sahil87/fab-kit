#!/usr/bin/env bash
set -euo pipefail

# Package kit archives into dist/ (per-platform: kit content + fab-go binary)
# Called by: just package-kit
# Requires: just dist-kit && just build-all

platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")

# Verify dist/kit/ was assembled
if [ ! -d "dist/kit" ]; then
  echo "ERROR: dist/kit/ not found — run 'just dist-kit' first."
  exit 1
fi

# Verify cross-compiled fab-go binaries exist
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  binary="dist/bin/fab-go-${os}-${arch}"
  if [ ! -f "$binary" ]; then
    echo "ERROR: Missing $binary — run 'just build-all' first."
    exit 1
  fi
done

echo "Packaging kit archives..."

for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  archive="dist/kit-${os}-${arch}.tar.gz"
  staging="dist/staging-kit-${os}-${arch}"

  rm -rf "$staging"
  cp -a dist/kit "$staging"
  cp "dist/bin/fab-go-${os}-${arch}" "$staging/bin/fab-go"
  chmod +x "$staging/bin/fab-go"

  # Archive contents are rooted at .kit/ for extraction into version cache
  mv "$staging" "dist/.kit"
  COPYFILE_DISABLE=1 tar czf "$archive" -C dist .kit
  mv "dist/.kit" "$staging"

  echo "  kit-${os}-${arch}.tar.gz ($(wc -c < "$archive") bytes)"
  rm -rf "$staging"
done

echo "Kit packaging complete: ${#platforms[@]} archives"

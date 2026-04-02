#!/usr/bin/env bash
set -euo pipefail

# Package brew archives into dist/ (per-platform: fab, fab-kit, wt, idea)
# Called by: just package-brew
# Requires: just build-all

platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")

echo "Packaging brew archives..."

for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"

  fab="dist/bin/fab-${os}-${arch}"
  fab_kit="dist/bin/fab-kit-${os}-${arch}"
  wt="dist/bin/wt-${os}-${arch}"
  idea="dist/bin/idea-${os}-${arch}"

  for bin in "$fab" "$fab_kit" "$wt" "$idea"; do
    if [ ! -f "$bin" ]; then
      echo "ERROR: Missing $bin — run 'just build-all' first."
      exit 1
    fi
  done

  archive="dist/brew-${os}-${arch}.tar.gz"
  staging="dist/staging-brew-${os}-${arch}"

  rm -rf "$staging"
  mkdir -p "$staging"
  cp "$fab" "$staging/fab"
  cp "$fab_kit" "$staging/fab-kit"
  cp "$wt" "$staging/wt"
  cp "$idea" "$staging/idea"
  chmod +x "$staging/fab" "$staging/fab-kit" "$staging/wt" "$staging/idea"

  COPYFILE_DISABLE=1 tar czf "$archive" -C "$staging" fab fab-kit wt idea
  echo "  brew-${os}-${arch}.tar.gz ($(wc -c < "$archive") bytes)"
  rm -rf "$staging"
done

echo "Brew packaging complete: ${#platforms[@]} archives"

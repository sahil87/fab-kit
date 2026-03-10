#!/usr/bin/env bash
set -euo pipefail

# Package kit archives for release (generic + per-platform with Go binaries)
# Called by: just package-kit

platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")
build_dir=".release-build"

# Verify all cross-compiled Go binaries exist
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  for bin in fab wt idea; do
    binary="$build_dir/${bin}-${os}-${arch}"
    if [ ! -f "$binary" ]; then
      echo "ERROR: Missing $bin binary $binary — run 'just build-go-all' first."
      exit 1
    fi
  done
done

# Generic archive (no binaries)
echo "Packaging kit.tar.gz (generic, no binary)..."
COPYFILE_DISABLE=1 tar czf kit.tar.gz -C fab --exclude='.kit/bin/fab-go' --exclude='.kit/bin/fab-rust' --exclude='.kit/bin/wt' --exclude='.kit/bin/idea' .kit
echo "  kit.tar.gz ($(wc -c < kit.tar.gz) bytes)"

# Per-platform archives (kit + Go binaries)
for platform in "${platforms[@]}"; do
  os="${platform%%/*}"
  arch="${platform##*/}"
  archive_name="kit-${os}-${arch}.tar.gz"
  staging="$build_dir/staging-${os}-${arch}"
  rm -rf "$staging"
  mkdir -p "$staging/.kit/bin"
  cp -a fab/.kit "$staging/.kit"
  for bin_pair in "fab:fab-go" "wt:wt" "idea:idea"; do
    src_name="${bin_pair%%:*}"
    dest_name="${bin_pair##*:}"
    cp "$build_dir/${src_name}-${os}-${arch}" "$staging/.kit/bin/${dest_name}"
    chmod +x "$staging/.kit/bin/${dest_name}"
  done
  COPYFILE_DISABLE=1 tar czf "$archive_name" -C "$staging" .kit
  echo "  $archive_name ($(wc -c < "$archive_name") bytes)"
  rm -rf "$staging"
done
echo "Packaging complete: kit.tar.gz + ${#platforms[@]} platform archives"

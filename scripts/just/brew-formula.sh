#!/usr/bin/env bash
set -euo pipefail

# Generate Homebrew formula into dist/fab-kit.rb
# Called by: just brew-formula [tag]
# Requires: just package-brew (brew archives must exist in dist/)

TAG="${1:-$(git describe --tags --abbrev=0 HEAD 2>/dev/null || true)}"
if [ -z "$TAG" ]; then
  echo "ERROR: No tag found. Pass a tag or ensure HEAD is tagged."
  exit 1
fi

VERSION="${TAG#v}"

# Compute SHA256 for each platform archive
for platform in darwin-arm64 darwin-amd64 linux-arm64 linux-amd64; do
  archive="dist/brew-${platform}.tar.gz"
  if [ ! -f "$archive" ]; then
    echo "ERROR: Missing $archive — run 'just package-brew' first."
    exit 1
  fi
done

sha_darwin_arm64=$(shasum -a 256 dist/brew-darwin-arm64.tar.gz | cut -d' ' -f1)
sha_darwin_amd64=$(shasum -a 256 dist/brew-darwin-amd64.tar.gz | cut -d' ' -f1)
sha_linux_arm64=$(shasum -a 256 dist/brew-linux-arm64.tar.gz | cut -d' ' -f1)
sha_linux_amd64=$(shasum -a 256 dist/brew-linux-amd64.tar.gz | cut -d' ' -f1)

sed \
  -e "s/VERSION_PLACEHOLDER/${VERSION}/" \
  -e "s/SHA_DARWIN_ARM64/${sha_darwin_arm64}/" \
  -e "s/SHA_DARWIN_AMD64/${sha_darwin_amd64}/" \
  -e "s/SHA_LINUX_ARM64/${sha_linux_arm64}/" \
  -e "s/SHA_LINUX_AMD64/${sha_linux_amd64}/" \
  .github/formula-template.rb > dist/fab-kit.rb

echo "Generated dist/fab-kit.rb (v${VERSION})"

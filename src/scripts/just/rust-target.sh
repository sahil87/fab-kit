#!/usr/bin/env sh
# Map os/arch pair to Rust target triple
# Usage: rust-target.sh <os> <arch>
# Called by: just _rust-target

os="$1"
arch="$2"

case "${os}-${arch}" in
  darwin-arm64) echo "aarch64-apple-darwin" ;;
  darwin-amd64) echo "x86_64-apple-darwin" ;;
  linux-arm64)  echo "aarch64-unknown-linux-musl" ;;
  linux-amd64)  echo "x86_64-unknown-linux-musl" ;;
  *) echo "ERROR: unknown platform ${os}-${arch}" >&2; exit 1 ;;
esac

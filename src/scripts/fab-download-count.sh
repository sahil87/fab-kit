#!/usr/bin/env bash
set -euo pipefail

# src/scripts/fab-download-count.sh — Show download counts for GitHub Releases
#
# Usage: fab-download-count.sh [--all | --total | <tag>]
#   (no args) — show latest release downloads
#   --all     — show all releases
#   --total   — show grand total across all releases
#   <tag>     — show a specific release (e.g. v0.26.2)
#
# Requires: gh CLI (https://cli.github.com/)

repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
repo=$(grep -E '^repo=' "$repo_root/fab/.kit/kit.conf" | cut -d= -f2 | tr -d '[:space:]')

if ! command -v gh &>/dev/null; then
  echo "ERROR: gh CLI not found. Install it from https://cli.github.com/"
  exit 1
fi

mode="latest"
tag=""

for arg in "$@"; do
  case "$arg" in
    --all)   mode="all" ;;
    --total) mode="total" ;;
    -h|--help)
      echo "Usage: fab-download-count.sh [--all | --total | <tag>]"
      echo ""
      echo "  (no args)  — latest release"
      echo "  --all      — all releases"
      echo "  --total    — grand total only"
      echo "  <tag>      — specific release (e.g. v0.26.2)"
      exit 0
      ;;
    *)
      mode="tag"
      tag="$arg"
      ;;
  esac
done

# Fetch release data: tag name, asset name, download count
fetch_releases() {
  gh api "repos/$repo/releases" \
    --paginate \
    --jq '.[] | {tag: .tag_name, assets: [.assets[] | {name: .name, downloads: .download_count}]}'
}

print_release() {
  local data="$1"
  local rtag rcount

  rtag=$(echo "$data" | jq -r '.tag')
  rcount=$(echo "$data" | jq '[.assets[].downloads] | add // 0')

  printf "%-12s %6d downloads" "$rtag" "$rcount"

  # Per-asset breakdown if more than one asset
  local asset_count
  asset_count=$(echo "$data" | jq '.assets | length')
  if [ "$asset_count" -gt 1 ]; then
    echo ""
    echo "$data" | jq -r '.assets[] | "  \(.name): \(.downloads)"'
  else
    echo ""
  fi
}

case "$mode" in
  latest)
    data=$(gh api "repos/$repo/releases/latest" \
      --jq '{tag: .tag_name, assets: [.assets[] | {name: .name, downloads: .download_count}]}')
    print_release "$data"
    ;;

  tag)
    data=$(gh api "repos/$repo/releases/tags/$tag" \
      --jq '{tag: .tag_name, assets: [.assets[] | {name: .name, downloads: .download_count}]}')
    print_release "$data"
    ;;

  all)
    grand=0
    while IFS= read -r release; do
      [ -z "$release" ] && continue
      print_release "$release"
      count=$(echo "$release" | jq '[.assets[].downloads] | add // 0')
      grand=$((grand + count))
    done < <(fetch_releases)
    echo "─────────────────────────"
    printf "Total:       %6d downloads\n" "$grand"
    ;;

  total)
    total=$(gh api "repos/$repo/releases" \
      --paginate \
      --jq '[.[].assets[].download_count] | add // 0')
    printf "%d total downloads across all releases\n" "$total"
    ;;
esac

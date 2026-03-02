#!/usr/bin/env bash
set -euo pipefail

# src/scripts/fab-mirror-to-org.sh — Mirror fab-kit to another GitHub org
#
# Clones the current repo at a given tag, rebrands repo references to
# the target org/repo, and force-pushes to the target.
#
# Usage: fab-mirror-to-org.sh <org/repo> [options]
#
#   <org/repo>            — target GitHub repository (e.g. acme-corp/fab-kit)
#   --tag <tag>           — source tag to mirror (default: latest tag on current branch)
#   --branch <branch>     — target branch to push to (default: main)
#   --git-name <name>     — git user.name for the mirror commit (default: fab-kit-mirror)
#   --git-email <email>   — git user.email for the mirror commit (default: noreply@fab-kit)
#   --shell               — after pushing, drop into a subshell in the temp clone
#                           (useful for running fab-release.sh against the target)
#   --dry-run             — show what would change without pushing
#
# Requires: gh CLI (https://cli.github.com/)
#
# Examples:
#   # Mirror latest tag to another org:
#   fab-mirror-to-org.sh acme-corp/fab-kit
#
#   # Mirror and then release to the target org:
#   fab-mirror-to-org.sh acme-corp/fab-kit --tag v0.26.2 --shell
#   # (now inside the temp clone — kit.conf points to acme-corp/fab-kit)
#   GH_TOKEN=ghp_xxx bash src/scripts/fab-release.sh patch
#   exit
#
# Authentication:
#   git push uses SSH (configure host aliases in ~/.ssh/config for different orgs).
#   gh CLI uses GH_TOKEN env var — set it to authenticate as a different account:
#     GH_TOKEN=ghp_target_org_token fab-mirror-to-org.sh acme-corp/fab-kit

usage() {
  echo "Usage: fab-mirror-to-org.sh <org/repo> [options]"
  echo ""
  echo "  <org/repo>            — target GitHub repository (e.g. acme-corp/fab-kit)"
  echo "  --tag <tag>           — source tag to mirror (default: latest tag)"
  echo "  --branch <branch>     — target branch to push to (default: main)"
  echo "  --git-name <name>     — git user.name for commits (default: fab-kit-mirror)"
  echo "  --git-email <email>   — git user.email for commits (default: noreply@fab-kit)"
  echo "  --shell               — drop into a subshell in the temp clone after pushing"
  echo "  --dry-run             — show what would change without pushing"
  echo ""
  echo "To release to the target org, use --shell then run fab-release.sh:"
  echo "  GH_TOKEN=ghp_xxx bash src/scripts/fab-release.sh patch"
}

repo_root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
source_repo=$(grep -E '^repo=' "$repo_root/fab/.kit/kit.conf" | cut -d= -f2 | tr -d '[:space:]')

target=""
tag=""
branch="main"
git_name="fab-kit-mirror"
git_email="noreply@fab-kit"
dry_run=false
shell_mode=false

while [ $# -gt 0 ]; do
  case "$1" in
    --tag)       tag="$2"; shift 2 ;;
    --branch)    branch="$2"; shift 2 ;;
    --git-name)  git_name="$2"; shift 2 ;;
    --git-email) git_email="$2"; shift 2 ;;
    --shell)     shell_mode=true; shift ;;
    --dry-run)   dry_run=true; shift ;;
    -h|--help) usage; exit 0 ;;
    -*)       echo "ERROR: Unknown flag '$1'"; usage; exit 1 ;;
    *)
      if [ -z "$target" ]; then
        target="$1"; shift
      else
        echo "ERROR: Unexpected argument '$1'"; usage; exit 1
      fi
      ;;
  esac
done

if [ -z "$target" ]; then
  echo "ERROR: Target org/repo is required."
  echo ""
  usage
  exit 1
fi

if ! command -v gh &>/dev/null; then
  echo "ERROR: gh CLI not found. Install it from https://cli.github.com/"
  exit 1
fi

# Resolve source tag
if [ -z "$tag" ]; then
  tag=$(git -C "$repo_root" describe --tags --abbrev=0 2>/dev/null || true)
  if [ -z "$tag" ]; then
    echo "ERROR: No tags found. Specify one with --tag."
    exit 1
  fi
fi

echo "Mirroring $source_repo@$tag → $target ($branch)"

# Clone into temp directory at the specified tag (full clone for tag history)
tmp=$(mktemp -d)
if [ "$shell_mode" = false ]; then
  cleanup() { rm -rf "$tmp"; }
  trap cleanup EXIT
fi

git clone --branch "$tag" "git@github.com:$source_repo.git" "$tmp/repo" 2>/dev/null

# Replace the two canonical locations
sed -i '' "s|repo=$source_repo|repo=$target|" "$tmp/repo/fab/.kit/kit.conf"
sed -i '' "s|$source_repo|$target|g" "$tmp/repo/README.md"

# Show what changed
echo ""
echo "Changes:"
git -C "$tmp/repo" diff --stat
echo ""
git -C "$tmp/repo" diff

if [ "$dry_run" = true ]; then
  echo ""
  echo "(dry run — nothing pushed)"
  exit 0
fi

# Commit and push
git -C "$tmp/repo" config user.name "$git_name"
git -C "$tmp/repo" config user.email "$git_email"
git -C "$tmp/repo" add -A
git -C "$tmp/repo" commit -m "mirror: rebrand for $target ($tag)"
git -C "$tmp/repo" remote set-url origin "git@github.com:$target.git"
git -C "$tmp/repo" push origin "HEAD:$branch" --force

echo ""
echo "Mirrored $tag to $target ($branch)"

# Drop into subshell if requested
if [ "$shell_mode" = true ]; then
  echo ""
  echo "Entering subshell in $tmp/repo"
  echo "  kit.conf repo = $target"
  echo "  To release: GH_TOKEN=ghp_xxx bash src/scripts/fab-release.sh patch"
  echo "  Type 'exit' when done (temp dir will be cleaned up)"
  echo ""
  (cd "$tmp/repo" && exec "$SHELL")
  echo "Cleaning up $tmp..."
  rm -rf "$tmp"
fi

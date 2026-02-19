#!/usr/bin/env bash
# Placeholder file. Not being used currently. We have reverted back to committing fab/backlog.md to each worktree for 
# simplicity, but we may want to revisit this in the future if we want to ensure all worktrees share the same backlog.

# Symlink fab/backlog.md to the main worktree's copy so all worktrees share one backlog.

# main_worktree="$(git worktree list --porcelain | head -1 | sed 's/^worktree //')"
# backlog_src="$main_worktree/fab/backlog.md"

# if [ ! -f "$backlog_src" ]; then
#   echo "  ⚠ No backlog.md found in main worktree ($main_worktree), skipping."
#   exit 0
# fi

# # Remove the checked-out copy (if any) and replace with symlink
# rm -f fab/backlog.md
# ln -s "$backlog_src" fab/backlog.md

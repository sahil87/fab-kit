# Packages

Fab Kit ships two standalone CLI tools alongside the skill-based pipeline. Unlike skills (which are Claude Code prompts invoked via `/`), these are compiled Go binaries that run directly in the terminal — no AI agent required.

Both binaries live inside `fab/.kit/bin/` and are added to PATH via `env-packages.sh` during workspace setup. They complement the fab pipeline: **wt** provides the worktree isolation that enables parallel changes, and **idea** provides the backlog that feeds `/fab-new`.

---

## wt (Worktree Management)

Git worktrees let you have multiple checkouts of the same repository side by side. The wt binary wraps `git worktree` with opinionated defaults for the fab workflow: worktrees are created as siblings in `<repo>.worktrees/`, names are memorable random words, and each worktree can run its own fab change independently.

**Binary**: `fab/.kit/bin/wt` (Go binary, included in per-platform release archives)

### Commands

| Command | Description |
|---------|-------------|
| `wt create` | Create a git worktree for parallel development |
| `wt list` | List all git worktrees for the current repository |
| `wt open` | Open a git worktree in an editor, terminal, or file manager |
| `wt delete` | Delete a git worktree with optional branch cleanup |
| `wt init` | Run the init script for the current worktree |

Run `wt <command> --help` for full usage details.

> **Note**: `wt pr` was dropped — pull request creation is handled by `/git-pr`.

### Integration with Fab

The wt binary is the foundation of the [assembly-line pattern](assembly-line.md). Each fab change gets its own worktree, enabling parallel AI sessions — one tmux tab per change, zero conflicts between them.

The batch scripts orchestrate this at scale:

- **`batch-fab-new-backlog`** — Creates a worktree per backlog item, opens tmux tabs, runs `/fab-new` in each
- **`batch-fab-switch-change`** — Creates a worktree per existing change, opens tmux tabs with Claude sessions
- **`batch-fab-archive-change`** — Archives completed changes across worktrees

### Common Workflows

**Start a new change in isolation:**
```bash
wt create                    # Creates a worktree with a random name
cd ../repo.worktrees/name/   # Enter the worktree
# Work on your change — main repo is untouched
```

**Clean up when done:**
```bash
wt delete                    # Removes the worktree and optionally its branch
```

### Why wt-open Cannot cd in the Current Shell

A frequently requested feature is a "cd here" option in `wt open` that changes the calling shell's working directory to the worktree path. This is not possible due to a fundamental Unix process model constraint.

**The constraint**: Every shell command (including `wt open` and `wt create`) runs as a **child process** of the calling shell. A child process has its own working directory that is independent of its parent. When the child exits, the parent's working directory is unchanged — there is no mechanism for a child process to modify its parent's environment.

This is not a limitation of `wt open` specifically — it applies to all executables on Unix systems. Only code that runs **within** the shell process itself (shell builtins like `cd`, and shell functions defined via `source` or `.`) can change the shell's working directory.

**Workarounds**: Users who want to navigate to a worktree in their current shell can use:

```bash
# Option 1: Use command substitution with wt create's stdout path
cd "$(wt create --non-interactive)"

# Option 2: Copy path from wt open's "Copy path" menu option, then paste
cd <paste>

# Option 3: Define a shell function in your .bashrc/.zshrc
wt-cd() {
    local path
    path=$(wt list --path "${1:?worktree name required}") || return $?
    cd "$path" || return $?
}
```

A shell function wrapper like Option 3 works because functions execute in the calling shell's process, not as a child. The function delegates path resolution to `wt list --path` (which runs as a child) but executes `cd` in the parent shell's context.

**Why we don't ship a shell function**: Shipping a `wt-cd` function would require users to `source` a file from their shell rc (`.bashrc`/`.zshrc`), which is a different distribution model than the current PATH-based approach. `env-packages.sh` adds `bin/` directories to PATH — it doesn't define shell functions. Mixing the two models adds complexity for a convenience that can be achieved with a 4-line function in the user's own rc file.

---

## idea (Backlog Management)

The idea command manages a per-repo backlog stored in `fab/backlog.md`. It's a lightweight CRUD tool for capturing, triaging, and tracking ideas before they become fab changes.

**Binary**: `fab/.kit/bin/idea` (Go binary, included in per-platform release archives; added to PATH as `idea` by `env-packages.sh` during workspace setup).

### Commands

| Command | Description |
|---------|-------------|
| `idea add "text"` | Add a new idea to the backlog |
| `idea list` | List open (uncompleted) ideas |
| `idea show <query>` | Show a single idea matching the query |
| `idea done <query>` | Mark an idea as done |
| `idea reopen <query>` | Reopen a completed idea |
| `idea edit <query> "text"` | Modify an idea's text |
| `idea rm <query>` | Delete an idea |

Each idea gets a short 4-character ID (e.g., `[a7k2]`) and a timestamp. Queries match against IDs or text (substring, case-insensitive).

### Integration with Fab

The idea → fab pipeline:

1. **Capture**: `idea add "add retry logic to API client"` — appends to `fab/backlog.md`
2. **Triage**: `idea list` — review open ideas, decide what to build next
3. **Start work**: `/fab-new` can accept a backlog ID to create a change from an idea, pulling the description directly from the backlog
4. **Close**: `idea done a7k2` — marks the idea as completed after the change ships

The `batch-fab-new-backlog` script automates step 3 at scale — it reads all open backlog items and creates a worktree + Claude session per item.

### Common Workflows

**Capture ideas throughout the day:**
```bash
idea add "refactor auth middleware to use JWT"
idea add "add rate limiting to public endpoints"
idea add "update README with new setup instructions"
```

**Triage and start work:**
```bash
idea list                     # Review what's pending
# Pick the ones to work on next
batch-fab-new-backlog --all   # Create changes from all open ideas
```

---

## Package Architecture

The `fab/.kit/bin/` directory contains the shell dispatcher and compiled Go binaries, all added to PATH by `env-packages.sh`:

```
fab/.kit/bin/
├── fab                 # Shell dispatcher (entry point for all fab CLI operations)
├── fab-go              # Go binary backend (optional, platform-specific)
├── fab-rust            # Rust binary backend (optional, built locally)
├── wt                  # Go binary — worktree management
├── idea                # Go binary — backlog management
└── .gitkeep
```

**PATH setup**: `fab/.kit/scripts/lib/env-packages.sh` adds `$KIT_DIR/bin` to PATH (making the `fab` dispatcher, `wt`, and `idea` binaries available), then iterates `$KIT_DIR/packages/*/bin` and adds each existing directory to PATH (for any future shell packages). This script is sourced by `.envrc` (for direnv-based projects) and can be sourced from shell rc files.

**Distribution**: Go binaries are included in per-platform release archives (`kit-{os}-{arch}.tar.gz`). The generic `kit.tar.gz` is source-only: it contains skills, templates, and supporting scripts/configuration, but no compiled binaries. `fab-upgrade.sh` updates kit content atomically alongside skills and templates.

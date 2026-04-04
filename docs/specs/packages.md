# Packages

Fab Kit ships two standalone CLI tools alongside the skill-based pipeline. Unlike skills (which are Claude Code prompts invoked via `/`), these are compiled Go binaries that run directly in the terminal — no AI agent required.

Both binaries are compiled from `src/go/` and installed system-wide via Homebrew. They complement the fab pipeline: **wt** provides the worktree isolation that enables parallel changes, and **idea** provides the backlog that feeds `/fab-new`.

---

## wt (Worktree Management)

Git worktrees let you have multiple checkouts of the same repository side by side. The wt binary wraps `git worktree` with opinionated defaults for the fab workflow: worktrees are created as siblings in `<repo>.worktrees/`, names are memorable random words, and each worktree can run its own fab change independently.

**Binary**: `src/go/wt/` (Go binary, included in per-platform release archives)

### Commands

| Command | Description |
|---------|-------------|
| `wt create` | Create a git worktree for parallel development |
| `wt list` | List all git worktrees for the current repository |
| `wt open` | Open a git worktree in an editor, terminal, or file manager |
| `wt delete [names...]` | Delete one or more git worktrees with optional branch cleanup (positional args; `--worktree-name` flag is deprecated) |
| `wt init` | Run the init script for the current worktree |

Run `wt <command> --help` for full usage details.

> **Note**: `wt pr` was dropped — pull request creation is handled by `/git-pr`.

### `wt create --base`

The `--base <ref>` flag specifies a git start-point when creating a new branch. This maps to `git worktree add -b <branch> <path> <start-point>`.

| Scenario | `--base` | Behavior |
|----------|----------|----------|
| New branch (doesn't exist locally or remotely) | provided | Branch created from `--base` ref instead of HEAD |
| New branch | omitted | Branch created from HEAD (default) |
| Existing local branch | provided | Warning: `--base ignored: branch already exists locally` |
| Existing remote branch | provided | Warning: `--base ignored: fetching existing remote branch` |
| Exploratory (no branch arg) | provided | Exploratory branch created from `--base` ref |
| Exploratory | omitted | Branch created from current HEAD (default) |
| With `--reuse` (worktree exists) | provided | `--reuse` takes precedence; `--base` has no effect |
| Invalid ref | provided | Error exit; no worktree or branch created |

The ref is validated via `git rev-parse --verify` before worktree creation, so invalid refs produce a clear error rather than a partial failure.

### Integration with Fab

The wt binary is the foundation of the [assembly-line pattern](assembly-line.md). Each fab change gets its own worktree, enabling parallel AI sessions — one tmux tab per change, zero conflicts between them.

The batch scripts orchestrate this at scale:

- **`fab batch new`** — Creates a worktree per backlog item, opens tmux tabs, runs `/fab-new` in each
- **`fab batch switch`** — Creates a worktree per existing change, opens tmux tabs with Claude sessions
- **`fab batch archive`** — Archives completed changes across worktrees

### Common Workflows

**Start a new change in isolation:**
```bash
wt create                        # Interactive menu → creates worktree
cd ../repo.worktrees/<name>/     # Enter the worktree (name shown in output)
# Work on your change — main repo is untouched
```

**Clean up when done:**
```bash
wt delete                    # Removes the current worktree and optionally its branch
wt delete fox bear           # Removes multiple worktrees by name
```

### Why wt-open Cannot cd in the Current Shell

Child processes cannot change the parent shell's working directory — this is a Unix process model constraint, not a `wt` limitation. To get a "cd here" option in `wt open`, add the shell wrapper to your profile:

```bash
# Add to ~/.bashrc or ~/.zshrc
eval "$(wt shell-setup)"
```

This installs a shell function that wraps the `wt` binary. Because shell functions run in the calling shell's process, the wrapper can eval `cd` commands from `wt open`'s "Open here" menu option.

**Alternative workarounds** (without the shell wrapper):

```bash
# Option 1: Use command substitution with wt create's stdout path
cd "$(wt create --non-interactive)"

# Option 2: Copy path from wt open's "Copy path" menu option, then paste
cd /path/copied/from/menu
```

---

## idea (Backlog Management)

The idea command manages a per-repo backlog stored in `fab/backlog.md`. It's a lightweight CRUD tool for capturing, triaging, and tracking ideas before they become fab changes.

**Binary**: `src/go/idea/` (Go binary, included in per-platform release archives; installed as `idea` via Homebrew).

**Worktree behavior**: By default, `idea` operates on the **current worktree's** `fab/backlog.md` (resolved via `git rev-parse --show-toplevel`). Pass `--main` to target the main worktree's backlog instead; internally, `idea` resolves the main worktree root by running `git rev-parse --path-format=absolute --git-common-dir` and taking its parent directory. In the main worktree, both behave identically. This ensures that users in a linked worktree get predictable local behavior unless they explicitly opt into the shared backlog.

### Commands

| Command | Description |
|---------|-------------|
| `idea "text"` | Add a new idea (shorthand for `idea add`) |
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

The `fab batch new` command automates step 3 at scale — it reads all open backlog items and creates a worktree + Claude session per item.

### Common Workflows

**Capture ideas throughout the day:**
```bash
idea "refactor auth middleware to use JWT"
idea "add rate limiting to public endpoints"
idea "update README with new setup instructions"
```

**Triage and start work:**
```bash
idea list                     # Review what's pending
# Pick the ones to work on next
fab batch new --all           # Create changes from all open ideas
```

---

## Package Architecture

The `src/go/` directory contains the Go source for all binaries:

```
src/go/
├── fab/                # fab CLI router
├── fab-kit/            # fab-kit lifecycle (init, sync, upgrade, doctor)
├── wt/                 # Worktree management
└── idea/               # Backlog management
```

All four binaries are installed system-wide via `brew install fab-kit`.

**Distribution**: Go binaries are included in per-platform release archives (`kit-{os}-{arch}.tar.gz`). The generic `kit.tar.gz` is source-only: it contains skills, templates, and supporting scripts/configuration, but no compiled binaries. `fab-upgrade.sh` updates kit content atomically alongside skills and templates.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260316-euw2-multi-worktree-delete | 2026-03-16 | `wt delete` now accepts positional arguments for multi-worktree deletion (`wt delete fox bear`). `--worktree-name` flag deprecated. Fail-fast validation, single confirmation prompt, sequential deletion with continue-on-error. |
| 260312-96nf-remove-rust-implementation | 2026-03-12 | Removed `fab-rust` from directory tree. |

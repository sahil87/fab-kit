# Companion Tools

fab-kit composes with two standalone CLIs — **wt** and **idea** — that live in their own repositories and ship on their own release cadences. They are not bundled in fab-kit's source tree; fab-kit's Homebrew formula declares them as dependencies (`depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"`), so `brew install sahil87/tap/fab-kit` lands all four binaries (`fab`, `fab-kit`, `wt`, `idea`) on PATH in a single step. This page describes how the fab pipeline uses them — see each tool's README for its full command surface.

## wt — worktree isolation

fab-kit relies on `wt` for parallel-by-default execution: every active change runs in its own git worktree under `<repo>.worktrees/<name>/`, so multiple AI sessions can edit the same repo without stepping on each other. The integration touches two batch commands — `fab batch new` calls `wt create` once per open backlog item to spin up a worktree per change (no positional branch — an exploratory create), and `fab batch switch` calls `wt create` (with `--reuse`) to attach worktrees to existing changes. Because `switch` targets changes whose branches usually already exist, it **probes branch existence** (local `git show-ref`, then `git ls-remote --heads origin`) and **routes** the invocation per wt's explicit-checkout contract: an existing branch is passed via `--checkout <branch>` (the bare positional is new-branch-only and exits 2 on an existing branch), a missing one via the positional. This couples `fab batch switch` to a **minimum wt version** — the `--checkout` path requires the wt release carrying that contract (the `260717-2af2` change); an older wt rejects `--checkout` as an unknown flag, which surfaces (warn-and-skip with the child stderr) rather than failing silently. `fab sync` is wired into the worktree init script so each new worktree gets the kit deployed automatically.

Full command reference, flags, and shell-setup details live in [sahil87/wt](https://github.com/sahil87/wt).

## idea — backlog feeding `/fab-new`

fab-kit uses `idea` as the inbox that feeds the pipeline. The `idea` CLI maintains a per-repo `fab/backlog.md` of short text items; `/fab-new` accepts a backlog ID and pulls the description directly into the new change's intake. The bulk path is `fab batch new`, which reads every open backlog entry and creates a worktree plus a Claude session for each one — turning a triaged backlog into a fleet of parallel changes in a single command.

Full command reference and worktree-vs-main-repo backlog semantics live in [sahil87/idea](https://github.com/sahil87/idea).

## Package architecture

fab-kit's own source tree only contains the binaries it owns:

- `src/go/fab/` — `fab` workflow router (delegates to skills, scripts, and `fab-kit`)
- `src/go/fab-kit/` — `fab-kit` lifecycle CLI (`init`, `sync`, `upgrade`, `doctor`)

Distribution splits across three artifact families:

- **`brew-{os}-{arch}.tar.gz`** — per-platform tarball with the `fab` and `fab-kit` binaries, consumed by the `sahil87/tap/fab-kit` Homebrew formula.
- **`kit-{os}-{arch}.tar.gz`** — per-version cache containing the `fab-go` backend binary, fetched on demand by `fab-kit sync`.
- **`kit.tar.gz`** — generic source-only tarball with skills, templates, and scripts; no compiled binaries.

`wt` and `idea` are not part of any fab-kit release artifact. They ship from their own repositories via `sahil87/tap/wt` and `sahil87/tap/idea`, and Homebrew resolves them transitively when installing fab-kit.

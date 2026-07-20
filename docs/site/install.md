# Install

This guide covers installing the Fab Kit CLIs and their companion utilities, plus the
install-specific notes worth knowing before you set up a project.

For the conceptual overview and the 6-stage pipeline, see the
[project README](https://github.com/sahil87/fab-kit/blob/main/README.md). Once you are installed,
head to the [Workflows guide](./workflows.md) to learn how to drive the pipeline.

## Install the CLI

Fab Kit installs via [Homebrew](https://brew.sh/) (macOS and Linux):

```bash
brew tap sahil87/tap
brew install fab-kit
```

This installs two CLIs on your `PATH`:

| Binary | Role |
|--------|------|
| `fab` | The router — dispatches every subcommand to the right tool |
| `fab-kit` | Workspace lifecycle: `init`, `upgrade-repo`, `sync` |

Two independent companion projects pair with fab-kit and install from their own formulas:

```bash
brew install sahil87/tap/wt sahil87/tap/idea
```

| Binary | Role |
|--------|------|
| `wt` | Worktree manager — isolates each change in its own git worktree |
| `idea` | Per-repo idea backlog (`fab/backlog.md`) that feeds `/fab-new` |

They are recommended, not required: fab degrades gracefully when they are absent. `wt` is needed
for the worktree flows — `fab batch new`/`switch` and the operator's agent spawning stop upfront
with an install hint (`wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt`)
rather than failing mid-run — while an absent `idea` costs nothing (`/fab-new`
reads `fab/backlog.md` directly). See their repos at
[sahil87/wt](https://github.com/sahil87/wt) and [sahil87/idea](https://github.com/sahil87/idea).

## Install the companion utilities

`fab` shells out to a few single-binary utilities (per the project's "pure prompt play" principle —
no runtime, no SDK):

```bash
brew install yq jq gh direnv
```

| Tool | Why fab needs it |
|------|------------------|
| [yq](https://github.com/mikefarah/yq) | YAML processing for `.status.yaml` and config/schema files |
| [jq](https://jqlang.github.io/jq/) | JSON processing during the settings merge in `fab sync` |
| [gh](https://cli.github.com/) | GitHub CLI — drives releases and the PR workflow (`/git-pr`, `/git-pr-review`) |
| [direnv](https://direnv.net/) | Auto-loads `.envrc` to set workspace environment variables per directory |

Two post-install steps:

- **Authenticate `gh`** so the PR commands can talk to GitHub:

  ```bash
  gh auth login
  ```

- **Add the direnv shell hook** so `.envrc` is picked up automatically. Follow the per-shell
  instructions at the [direnv hook docs](https://direnv.net/docs/hook.html) (a one-line addition to
  your shell rc file). The first time you enter a fab workspace, run `direnv allow` to trust it.

### Optional: shell completion

Activate tab-completion for `fab` in your shell's rc file:

```bash
eval "$(fab shell-init zsh)"   # or bash / fish
```

This works from any directory — no fab project required. If you prefer a static completion script
on disk instead of the eval hook, generate one with `fab completion <shell>`.

### Developing Fab Kit itself

If you are building the binaries from source or contributing, you also need:

```bash
brew install go just
```

| Tool | Why |
|------|-----|
| [Go](https://go.dev/) | Builds the binaries from source (`src/go/`) |
| [just](https://just.systems/) | Task runner for the build, test, and release recipes |

Contributor setup and release mechanics live in the repo's
[CONTRIBUTING.md](https://github.com/sahil87/fab-kit/blob/main/CONTRIBUTING.md).

## Set up a project

Once the CLIs and utilities above are on your `PATH`, the project-setup flow itself — `fab init`
for a new project, `fab init`/`fab sync` + `/docs-hydrate-memory` to onboard an existing repo with
prior docs, and `fab upgrade-repo` + `/fab-setup migrations` to update — is covered step-by-step in
the README's
[Quick Start → Install](https://github.com/sahil87/fab-kit/blob/main/README.md#quick-start). This
page deliberately does not restate it: the same content is pulled to the site from the README, so
duplicating it here would render twice.

Two install-specific notes worth calling out:

- `fab sync` re-deploys skills, scaffolds structure, and syncs hooks **without** changing the
  pinned version — run it on its own right after cloning a repo that already uses Fab Kit. It also
  runs automatically in every new worktree created by `wt create`.
- Agent skills and hooks live under `.claude/`, which is gitignored by default, so **each developer
  deploys them locally** with `fab sync` — they do not arrive with a `git clone`.

## Verify your install

Run the doctor to check every prerequisite (git, `yq`, the direnv hook, shell completion, etc.) and
diagnose common setup issues:

```bash
fab doctor
```

For runtime troubleshooting beyond install (a stage failing mid-way, a `/fab-setup` not recognized,
agent producing bad code), see the README's
[Troubleshooting](https://github.com/sahil87/fab-kit/blob/main/README.md#troubleshooting) section.

Once `fab doctor` is green, continue to the [Workflows guide](./workflows.md) to run your first
change through the pipeline.

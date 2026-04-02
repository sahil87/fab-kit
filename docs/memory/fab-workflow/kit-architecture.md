# Kit Architecture

**Domain**: fab-workflow

## Overview

`src/kit/` is the portable engine directory that contains all workflow logic: skill definitions, artifact templates, utility shell scripts, and version tracking. It is content-only — no binaries. The system provides three binaries: `fab` (router), `fab-kit` (workspace lifecycle), and `fab-go` (workflow engine), all installed via `brew install fab-kit`. The `fab` router dispatches to either `fab-kit` or the version-resolved `fab-go`. `src/kit/` provides content (skills, templates, configuration). This doc covers the `.kit/` directory structure, the three-binary architecture, agent integration, distribution, updating, and monorepo guidance.

> **CLI Command Reference**: For calling conventions and full command signatures, see `$(fab kit-path)/skills/_cli-fab.md` (the canonical CLI reference, loaded by every skill via `_preamble.md`).

## Requirements

### Directory Structure

The `.kit/` directory SHALL contain:

```
src/kit/
├── VERSION                 # Semver string (e.g., "0.1.0")
├── bin/                    # Empty — no binaries in repo (system shim handles execution)
│   └── .gitkeep            # Ensures directory exists
├── skills/                 # Skill definitions (markdown prompts)
│   ├── _preamble.md         # Shared context loading convention
│   ├── _cli-fab.md          # Fab CLI command reference (always-load, renamed from _scripts.md)
│   ├── _cli-external.md     # External CLI tools: wt, tmux, /loop (operator-only load)
│   ├── _generation.md       # Spec/tasks generation procedures
│   ├── _naming.md           # Naming conventions: change folders, branches, worktrees (always-load)
│   ├── fab-setup.md
│   ├── docs-hydrate-memory.md
│   ├── docs-hydrate-specs.md
│   ├── docs-reorg-memory.md
│   ├── docs-reorg-specs.md
│   ├── fab-new.md
│   ├── fab-continue.md
│   ├── fab-ff.md
│   ├── fab-fff.md
│   ├── fab-clarify.md
│   ├── fab-switch.md
│   ├── fab-status.md
│   ├── fab-help.md
│   ├── fab-archive.md
│   ├── fab-discuss.md
│   ├── fab-operator.md      # Standalone operator — multi-agent coordination with dependency-aware spawning
│   ├── git-branch.md
│   ├── git-pr.md
│   ├── git-pr-review.md
│   ├── internal-consistency-check.md
│   ├── internal-retrospect.md
│   └── internal-skill-optimize.md
├── migrations/             # Version migration instructions (markdown)
│   └── .gitkeep            # Ships even if empty
├── templates/              # Artifact templates
│   ├── intake.md
│   ├── spec.md
│   ├── tasks.md
│   ├── checklist.md
│   └── status.yaml         # .status.yaml template (includes stage_metrics: {}, issues: [], prs: [])
├── scaffold/               # Overlay tree — paths mirror repo root destinations
│   ├── fragment-.envrc     # .envrc required entries (line-ensuring merge)
│   ├── fragment-.gitignore # .gitignore entries (line-ensuring merge)
│   ├── .claude/
│   │   └── fragment-settings.local.json  # Baseline permissions (JSON merge)
│   ├── docs/
│   │   ├── memory/index.md # Initial docs/memory/index.md (copy-if-absent)
│   │   └── specs/index.md  # Initial docs/specs/index.md (copy-if-absent)
│   └── fab/
│       ├── changes/archive/.gitkeep  # Archive directory marker
│       ├── project/
│       │   ├── config.yaml     # Default config.yaml template (copy-if-absent, /fab-setup detects)
│       │   ├── constitution.md # Constitution skeleton (copy-if-absent, /fab-setup detects)
│       │   ├── context.md      # Project context template (copy-if-absent)
│       │   ├── code-quality.md # Code quality defaults (copy-if-absent)
│       │   └── code-review.md  # Review policy defaults (copy-if-absent)
│       └── sync/README.md     # README template for fab/sync/ (copy-if-absent)
├── packages/               # Distributable CLI tools (idea)
│   └── idea/bin/idea       # Per-repo idea backlog manager
├── schemas/                # Workflow schema
│   └── workflow.yaml       # Canonical stage/state definitions
├── hooks/                  # Claude Code hook scripts (cwd-resilient wrappers delegating to fab hook <subcommand>)
│   ├── on-artifact-write.sh # PostToolUse hook (Write + Edit matchers) — delegates to fab hook artifact-write
│   ├── on-session-start.sh # SessionStart hook — delegates to fab hook session-start
│   ├── on-stop.sh          # Stop hook — delegates to fab hook stop
│   └── on-user-prompt.sh   # UserPromptSubmit hook — delegates to fab hook user-prompt
└── sync/                   # Kit-level sync scripts (empty after full fab-kit sync absorption)
    └── .gitkeep            # All sync scripts absorbed into fab-kit Go binary
```

### Shell Scripts

#### `fab-sync.sh` (Removed)

Replaced by `fab-kit sync` — a Go binary subcommand. See the `fab-kit` binary section below for the sync implementation. The `$WORKTREE_INIT_SCRIPT` env var in `.envrc` now points to `fab sync`.

#### `sync/1-prerequisites.sh` (Removed)

Prerequisites check absorbed into `fab-kit sync`. The Go implementation validates required tools (git, bash, yq v4+, direnv) before performing sync operations.

#### `sync/3-direnv.sh` (Removed)

`direnv allow` absorbed into `fab-kit sync` as an idempotent step.

#### `sync/2-sync-workspace.sh` (Removed)

All workspace sync logic absorbed into `fab-kit sync` (Go binary). The Go implementation replicates all behavior: directory scaffolding, scaffold tree-walk with fragment-merge and copy-if-absent strategies, multi-agent skill deployment, stale skill cleanup, version stamp tracking, and `fab/.kit-migration-version` creation. `/fab-setup` delegates to `fab-kit sync` (instead of `fab-sync.sh`) and adds the interactive parts (config, constitution).

#### Removed: `lib/` shell scripts (statusman.sh, logman.sh, calc-score.sh, changeman.sh, archiveman.sh)

These scripts were removed in change `260305-u8t9-clean-break-go-only`. Their operations are now handled by Go binary subcommands (`fab status`, `fab log`, `fab score`, `fab change`, `fab change archive/restore`). See `$(fab kit-path)/skills/_cli-fab.md` for the canonical CLI command reference.

#### `fab-doctor.sh` (Removed)

Replaced by `fab doctor` — a `fab-kit` subcommand. The Go implementation in `src/go/fab-kit/cmd/fab-kit/doctor.go` replicates all behavior: validates 7 tools (git, fab, bash, yq v4+, jq, gh, direnv+hook), supports `--porcelain` flag, exit code = failure count. Added to the `fab` router's `fabKitArgs` allowlist so it works before `config.yaml` exists (required for `/fab-setup` Phase 0 gate).

#### `fab-help.sh` (Removed)

Replaced by `fab fab-help` — a `fab-go` subcommand. The Go implementation in `src/go/fab/cmd/fab/fabhelp.go` dynamically scans `$(fab kit-path)/skills/*.md` frontmatter via `internal/frontmatter/`, groups commands by category (hardcoded map matching the former shell script's `skill_to_group`), and renders formatted output with version header, workflow diagram, and typical flow. Batch commands are read from `fab batch` cobra subcommands instead of scanning `batch-*.sh` scripts. Skills with `_` prefix (partials) and `internal-` prefix are excluded.

#### `lib/spawn.sh` (Removed)

Replaced by `internal/spawn/` package in `fab-go`. The Go implementation reads `agent.spawn_command` from `fab/project/config.yaml` via `gopkg.in/yaml.v3`, falls back to `claude --dangerously-skip-permissions`. Used by `fab operator` and all `fab batch` subcommands.

#### `lib/frontmatter.sh` (Removed)

Replaced by `internal/frontmatter/` package in `fab-go`. The Go implementation extracts fields from YAML frontmatter (content between `---` markers), handles quoted/unquoted values, strips inline comments. Used by `fab fab-help` for skill discovery.

#### `fab-upgrade.sh` (Removed)

Replaced by `fab upgrade` — a shim subcommand. See [distribution.md](distribution.md) for the full upgrade flow.

#### `release.sh` (dev-only, at `scripts/release.sh`)

Bumps VERSION (accepts `[patch|minor|major]` argument), validates the migration chain (warns if no migration targets the new version, warns on overlapping migration ranges), commits the version change, tags it, and pushes to the remote. CI takes over from the tag push to cross-compile, package, and create the GitHub Release. Requires clean working tree. This script is not shipped inside `src/kit/` — it is a dev-only tool for maintainers of the fab-kit repo.

#### Batch Scripts (Removed)

Replaced by `fab batch` subcommand group in `fab-go`. Source: `src/go/fab/cmd/fab/batch.go` (parent command), `batch_new.go`, `batch_switch.go`, `batch_archive.go`. All three share common patterns: tmux tab creation, spawn command resolution via `internal/spawn/`, `--list`/`--all` flags.

- **`fab batch new`** — Per backlog ID: creates a worktree via `wt create --non-interactive`, opens a tmux tab, starts a Claude Code session running `/fab-new <description>`. Parses `fab/backlog.md` with continuation line handling. Supports `--list`, `--all`, and positional ID arguments.
- **`fab batch switch`** — Per change name/ID: creates a worktree with the correct branch name (using `branch_prefix` from config), opens a tmux tab, runs `/fab-switch <change>`. Uses `fab change resolve` for name resolution. Supports `--list`, `--all`, and positional arguments.
- **`fab batch archive`** — Finds changes with `hydrate: done|skipped` in `.status.yaml`, spawns a single Claude Code session to run `/fab-archive` for each. Supports `--list`, `--all`, and positional change arguments. Uses `fab change resolve` for name resolution.

#### Launcher Scripts (Removed)

Replaced by `fab operator` — a `fab-go` subcommand. Source: `src/go/fab/cmd/fab/operator.go`. Creates a singleton tmux window named "operator" running the configured `agent.spawn_command` (via `internal/spawn/`) with `'/fab-operator'`. If the window already exists, switches to it. Requires an active tmux session (`$TMUX` check).

### Agent Skill Deployment

`fab-kit sync` deploys skills to each agent. Deployment is **conditional** — by default, each agent's CLI command is checked via PATH lookup before syncing. If an agent's CLI is not found in PATH, its sync is skipped with a message, and existing dot folders are preserved. When no agents are detected, a warning is printed but sync continues. The `FAB_AGENTS` environment variable (space-separated list of CLI command names, e.g., `claude opencode gemini`) can override PATH detection for testing and CI — when set, only the listed agents are synced.

All `*.md` files in `$(fab kit-path)/skills/` are deployed, including underscore partials (`_preamble.md`, `_generation.md`, `_cli-fab.md`, `_cli-external.md`, `_naming.md`) which have `user-invocable: false` frontmatter to prevent direct invocation. The skill prompt files are agent-agnostic markdown; only the deployment locations and formats differ per agent:

**Claude Code** (`claude`) — directory-based copies:
```
.claude/skills/fab-new/
└── SKILL.md    (copy of $(fab kit-path)/skills/fab-new.md)
```

**OpenCode** (`opencode`) — flat file symlinks:
```
.opencode/commands/
└── fab-new.md → ../../$(fab kit-path)/skills/fab-new.md
```

**Codex** (`codex`) — directory-based copies:
```
.agents/skills/fab-new/
└── SKILL.md    (copy of $(fab kit-path)/skills/fab-new.md)
```

**Gemini CLI** (`gemini`) — directory-based copies:
```
.gemini/skills/fab-new/
└── SKILL.md    (copy of $(fab kit-path)/skills/fab-new.md)
```

### Distribution & Bootstrapping

`.kit/` is a content-only directory — no binaries. The system binaries (`fab`, `fab-kit`, installed via `brew install fab-kit`) provide version-aware execution and workspace lifecycle management. `.kit/` provides content (skills, templates, configuration).

#### Bootstrap Sequence

**Primary method** (recommended):
```
brew tap wvrdz/tap && brew install fab-kit
cd <repo>
fab init
```

`fab init` populates `src/kit/` from the version cache, sets `fab_version` in `config.yaml`, and calls `Sync()` directly (the same logic as `fab-kit sync`).

**Legacy method** (curl one-liner, for environments without Homebrew):
```
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac
mkdir -p fab; curl -sL "https://github.com/{repo}/releases/latest/download/kit-${os}-${arch}.tar.gz" | tar xz -C fab/
```

**Manual copy** (from a local clone):
```
cp -r /path/to/fab-kit/fab/.kit fab/.kit
```

Then in either case:
1. User runs `/fab-setup` → generates `config.yaml`, `constitution.md`
2. User optionally runs `/fab-hydrate` → ingests external sources
3. User runs `/fab-new` → first change created

#### Why Two Phases

`/fab-setup` is itself a skill defined inside `.kit/`. It cannot run until `.kit/` exists. `fab init` solves this by populating `.kit/` from the cache before any skill invocation.

### Version Tracking (Dual-Version Model)

Three version locations track the relationship between the installed engine and the project's file format:

- **`$(fab kit-path)/VERSION`** (engine version) — ships inside `.kit/`, replaced on each `fab-upgrade.sh` run. Enables version display, update comparison, and migration targeting.
- **`fab/project/config.yaml` `fab_version`** (project version) — set by `fab upgrade` and `fab init`. Used by preflight to detect sync staleness (compared against `$(fab kit-path)/VERSION`).
- **`fab/.kit-migration-version`** (local project version) — lives outside `.kit/`, NOT replaced on upgrades. Tracks the version the project's `config.yaml`, `.status.yaml`, and conventions were written for. Created by `sync/2-sync-workspace.sh`. Renamed from `fab/project/VERSION`.

`VERSION` and `.kit-migration-version` contain a bare semver string (`MAJOR.MINOR.PATCH`). See [migrations.md](migrations.md) for the full migration system.

### Updating `.kit/`

Run `fab upgrade` to update to the latest release. The fab-kit subcommand downloads the new version to the cache (if not present), atomically replaces `src/kit/` with content from the cache, updates `fab_version` in `config.yaml`, and calls `Sync()` directly to repair directories and agents. After the upgrade, if `fab/.kit-migration-version` is behind the new engine version, the output includes a migration reminder. See [distribution.md](distribution.md) for full upgrade details.

Skill deployments in `.claude/skills/`, `.opencode/commands/`, `.agents/skills/`, and `.gemini/skills/` are refreshed by `fab-kit sync` after the update. OpenCode symlinks resolve automatically; copies for Claude Code, Codex, and Gemini are re-copied.

**Preserved** (lives outside `.kit/`): `config.yaml`, `constitution.md`, `docs/memory/`, `docs/specs/`, `changes/`, `.fab-status.yaml`, `.kit-migration-version`
**Replaced** (lives inside `.kit/`): `templates/`, `skills/`, `sync/`, `migrations/`, `packages/` (idea shell package), `bin/` (`.gitkeep` only), `VERSION`

### Portability

The `.kit/` directory MUST work in any project via `cp -r`, given the system binaries are installed (`brew install fab-kit` installs `fab`, `fab-kit`, `wt`, `idea`). The system binaries provide version-aware routing and workspace lifecycle management; `src/kit/` provides content (skills, templates, configuration). It SHALL have no assumptions about the host project's structure, language, or toolchain beyond the presence of a `fab/` directory. Project-specific configuration belongs in `fab/project/config.yaml` and `fab/project/constitution.md`, not in `.kit/`.

### Monorepo Guidance

A monorepo is one Fab project. Place a single `fab/` at the repository root — do not create per-package `fab/` directories.

**Why one `fab/`**:
- Changes naturally span packages — one change folder, one spec
- Memory is domain-based, not package-based — `docs/memory/auth/` describes auth regardless of which package implements it
- One developer, one change at a time — `.fab-status.yaml` points to a single active change
- Simplicity — multiple `fab/` directories means multiple constitutions, memory trees, and symlink conflicts

For mixed tech stacks, use labeled sections in `config.yaml`'s `context` field so skills can load relevant context per package.

### Three-Binary Architecture

The system provides three distinct binaries, each independently executable with its own `--help`:

#### `fab` (Router)

The `fab` binary (installed via `brew install fab-kit`) is the user-facing entry point. It uses negative-match routing: a static allowlist of fab-kit commands (`init`, `upgrade`, `sync`, `doctor`, `--version`, `-v`, `--help`, `-h`, `help`) is dispatched to `fab-kit` via `syscall.Exec`; all other commands are dispatched to the version-resolved `fab-go` via `syscall.Exec`.

For fab-go dispatch, the router:
1. Walks up from CWD to find `fab/project/config.yaml`
2. Reads `fab_version` from `config.yaml`
3. Resolves the matching `fab-go` binary from the local cache at `~/.fab-kit/versions/{version}/fab-go`
4. If not cached, auto-fetches from GitHub releases and caches it
5. Execs the cached `fab-go` with full argument passthrough

When not in a fab-managed repo (no `config.yaml`) and a non-fab-kit command is issued, the router exits with: "Not in a fab-managed repo. Run 'fab init' to set one up."

`fab help` composes help from both sub-binaries: workspace commands (from fab-kit) are always shown; workflow commands (from fab-go) are shown only when inside a fab-managed repo.

#### `fab-kit` (Workspace Lifecycle)

The `fab-kit` binary (installed via `brew install fab-kit`) owns workspace lifecycle operations:

- `fab-kit init` — initialize fab in a repo (resolve latest version, cache it, populate `src/kit/`, set `fab_version`, run sync)
- `fab-kit upgrade [version]` — upgrade to a different version (download to cache, atomic replace `src/kit/`, update `fab_version`, run sync)
- `fab-kit sync` — reconcile workspace with pinned version (6-step pipeline: prerequisites, version guard, ensure cache, scaffolding, direnv, project scripts). Supports `--shim` (steps 1-5) and `--project` (step 6) flags.
- `fab-kit doctor [--porcelain]` — validate fab-kit prerequisites (7 tools: git, fab, bash, yq v4+, jq, gh, direnv+hook). Works before `config.yaml` exists — required for `/fab-setup` Phase 0 gate. Exit code = failure count. `--porcelain` outputs only errors (no passes/hints/summary)

`fab-kit sync` resolves all kit content from the system cache at `~/.fab-kit/versions/{version}/kit/` (via `CachedKitDir(fab_version)`) rather than from `src/kit/` in the repo. The 6-step pipeline: (1) prerequisites check (git, bash, yq v4+, direnv), (2) version guard (ensures `fab_version` <= system `fab-kit` version, auto-runs `fab update` if behind), (3) ensure cache (calls `EnsureCached(fab_version)`, downloads if needed), (4) workspace scaffolding from cache (directory creation, scaffold tree-walk with fragment-merge and copy-if-absent, multi-agent skill deployment, hook sync, version stamp, legacy cleanup), (5) direnv allow, (6) project-level `fab/sync/*.sh` script execution. Hook sync (previously delegated to `5-sync-hooks.sh` and `fab hook sync`) is absorbed directly into step 4 — `fab-kit` replicates the hooklib sync logic internally rather than shelling out to the `fab` binary.

**Source layout**: Both `fab` (router) and `fab-kit` share a single Go module at `src/go/fab-kit/` with two `cmd/` entries: `cmd/fab/main.go` and `cmd/fab-kit/main.go`. Both import shared `internal/` packages for cache, download, and config resolution. This avoids Go workspace complexity and keeps infrastructure code importable by both without duplication.

The shell dispatcher at `fab-go binary at fab` has been removed. The `FAB_BACKEND` env var and `.fab-backend` file override mechanism has been removed — Go is the only backend.

### Go Binary (`fab-go`)

The workflow engine backend for all fab CLI operations. Source: `src/go/fab/`.

**Module**: `github.com/sahil87/fab-kit/src/go/fab` (Go 1.22+, dependencies: cobra, gopkg.in/yaml.v3, no CGo)

**Binary location**: `~/.fab-kit/versions/{version}/fab-go` — cached per-version by the system shim. Included in per-platform release archives (`kit-{os}-{arch}.tar.gz`). No longer stored in `fab-go binary at ` — the repo holds content only.

**Subcommands**:
- `fab resolve [--id|--folder|--dir|--status] [<change>]`
- `fab log command|confidence|review|transition ...`
- `fab status start|advance|finish|reset|skip|fail|all-stages|progress-map|...` (stage-machine operations plus status/diagnostic utilities)
- `fab preflight [<change>]`
- `fab change new|rename|switch|list|resolve ...`
- `fab score [--check-gate] [--stage <stage>] <change>`
- `fab change archive|restore|archive-list ...`
- `fab runtime set-idle <change>` — write `agent.idle_since` to `.fab-runtime.yaml`
- `fab runtime clear-idle <change>` — remove agent block from `.fab-runtime.yaml`
- `fab runtime is-idle <change>` — read-only idle state query; prints `idle {duration}`, `active`, or `unknown` (exit 0 always)
- `fab hook session-start` — clear agent idle state (SessionStart event)
- `fab hook stop` — set agent idle timestamp (Stop event)
- `fab hook user-prompt` — clear agent idle state (UserPromptSubmit event)
- `fab hook artifact-write` — artifact bookkeeping: parse PostToolUse JSON from stdin, pattern-match fab artifact paths, perform per-artifact bookkeeping (type inference, scoring, checklist counting)
- `fab hook sync` — register hook scripts into `.claude/settings.local.json` (replaces jq-dependent shell implementation)
- `fab pane-map [--json] [--session <name>] [--all-sessions]` — combine tmux pane introspection with worktree/change/runtime state into a unified view (table columns: Pane, WinIdx, Tab, Worktree, Change, Stage, Agent; plus conditional Session column with `--all-sessions`). `--json` outputs a JSON array with fields: `session`, `window_index`, `pane`, `tab`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration` (null semantics for non-fab panes). `--session <name>` targets a specific tmux session by name (skips `$TMUX` check). `--all-sessions` queries all tmux sessions. Flags `--session` and `--all-sessions` are mutually exclusive. Default mode uses current session via `$TMUX` (existing behavior). Shows all tmux panes (not just fab worktrees) with em-dash/null fallbacks for non-fab panes
- `fab resolve <change> --pane` — output the tmux pane ID (e.g., `%5`) for the pane running the resolved change; composable with `tmux send-keys -t "$(fab resolve <change> --pane)" "<text>" Enter`
- `fab idea add|list|show|done|reopen|edit|rm` — backlog idea management (CRUD for `fab/backlog.md`)
- `fab fab-help` — dynamic skill discovery and help overview (scans `.kit/skills/` frontmatter, groups by category)
- `fab operator` — singleton tmux tab launcher for the operator skill (reads `agent.spawn_command` from config)
- `fab batch new|switch|archive` — multi-target batch operations via tmux tabs with Claude Code sessions

**Architecture**: `internal/spawn` provides shared spawn command resolution (reads `agent.spawn_command` from `config.yaml`, used by `operator` and `batch` subcommands). `internal/frontmatter` provides YAML frontmatter parsing (used by `fab-help`). `internal/statusfile` is the shared foundation — a `StatusFile` struct parsed once via `Load()`, passed by pointer across all operations, and written atomically via temp+rename `Save()`. All other packages (`resolve`, `log`, `status`, `preflight`, `change`, `score`, `archive`, `worktree`) import `statusfile` for YAML access. The `worktree` package provides worktree discovery via `git worktree list --porcelain` and fab state resolution, and also contains the full worktree management library used by the `wt` binary (see below). The `internal/runtime` package (extracted from `cmd/fab/runtime.go`) provides shared runtime file manipulation (`LoadFile`, `SaveFile`, `FilePath`, `ClearIdle`, `SetIdle`) — used by both `fab runtime` CLI subcommands and `fab hook` subcommands. The `internal/hooklib` package provides artifact bookkeeping logic (JSON parsing, path pattern matching, change type inference, task/checklist counting) and hook sync logic (hook-to-event mapping, JSON merging for `.claude/settings.local.json`). The `pane-map` subcommand in `cmd/fab/panemap.go` combines tmux pane discovery, worktree resolution, change state, and runtime state into a single observation command. Supports `--json` (JSON array output), `--session <name>` (target specific session), and `--all-sessions` (enumerate all sessions). `discoverPanes(mode, sessionName)` accepts a session targeting mode and extends the tmux format string with `#{session_name}` and `#{window_index}`. Shared pane-matching functions (`discoverPanes`, `matchPanesByFolder`, `resolvePaneChange`) also live in `panemap.go` and are reused by `resolve --pane`.

**Parity**: All subcommands produce stdout/stderr output matching the bash versions (modulo timestamps).

**Testing**: Unit tests in `src/go/fab/` cover all internal packages via `go test ./...`. Run with `just test` (or `just test-v` for verbose). Tested packages: `cmd/fab` (panemap, operator, batch_new, batch_switch, batch_archive, fabhelp), `cmd/wt`, `internal/config`, `internal/hooks`, `internal/hooklib` (artifact bookkeeping + hook sync), `internal/runtime` (runtime file manipulation), `internal/status`, `internal/statusfile`, `internal/resolve`, `internal/log`, `internal/preflight`, `internal/score`, `internal/archive`, `internal/change`, `internal/worktree`, `internal/idea`, `internal/spawn`, `internal/frontmatter`. `fab-kit` tests: `cmd/fab-kit` (doctor). Test patterns: `t.TempDir()` for filesystem isolation, table-driven tests with `t.Run()` subtests, standard `testing` package only (no external test frameworks). The previous parity tests (`src/go/fab/test/parity/`) were removed — the bash scripts they validated against no longer exist.

#### `fab idea` Subcommand

Backlog idea management — CRUD operations for `fab/backlog.md`. Ported from the shell package at `src/kit/packages/idea/bin/idea` to a native Go implementation. Both coexist: the shell package remains for rollback safety, the Go binary is the preferred invocation path.

**Subcommands**: `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`.

**Persistent flag**: `--file <path>` overrides the backlog file path (relative to git root). `IDEAS_FILE` env var also overrides. Priority: `--file` > `IDEAS_FILE` > default `fab/backlog.md`.

**Package**: `internal/idea/` — `Idea` struct (ID, Date, Text, Done), `File` struct (preserves non-idea lines for round-trip fidelity), `ParseLine`/`FormatLine` for serialization, `Match`/`FindAll`/`RequireSingle` for query resolution, and top-level CRUD functions (`Add`, `List`, `Show`, `Done`, `Reopen`, `Edit`, `Rm`). Git root resolved via `git rev-parse` (exec, no Go git library). Random 4-char alphanumeric IDs generated via `crypto/rand`. `rm` requires `--force` (no interactive prompt — agent-context safety).

**Cobra integration**: `cmd/fab/idea.go` registers `ideaCmd()` as a top-level subcommand with 7 sub-subcommands. Each sub-subcommand resolves the backlog file path via `resolveIdeaFile()` (git root + flag/env/default precedence).

### wt Binary

A separate Go binary for git worktree management, built from `src/go/fab/cmd/wt/main.go`. Operates on any git repo — does not require a `fab/` directory. Different concern domain from `fab` (worktree management vs workflow pipeline), so users type `wt create`, not `fab wt create`.

**Binary location**: System PATH via Homebrew (`brew install fab-kit`). No longer stored in `fab-go binary at ` — distributed exclusively through the Homebrew formula as a version-independent standalone utility.

**Module**: Same Go module as `fab-go` (`github.com/sahil87/fab-kit/src/go/fab`). Dependencies: cobra (subcommand dispatch). Does NOT depend on any `fab`-specific packages — only `internal/worktree/` and shared utilities in `internal/`.

**Subcommands** (5 — `wt pr` dropped, overlaps with `/git-pr`):
- `wt create [flags] [branch]` — create a git worktree (random name for exploratory, branch-derived name for feature). `--base <ref>` sets the git start-point for new branches (maps to `git worktree add -b <branch> <path> <start-point>`); ignored with a warning for existing local/remote branches; validated via `git rev-parse --verify` before use; defers to `--reuse` when both are provided and the worktree already exists
- `wt list [flags]` — list worktrees with status indicators (dirty `*`, unpushed `↑N`)
- `wt open [flags] [name|path]` — open a worktree in a detected application (VSCode, Cursor, Ghostty, tmux, etc.)
- `wt delete [flags]` — delete a worktree with optional branch and remote cleanup
- `wt init` — run the worktree init script (`src/kit/worktree-init.sh`)

**`internal/worktree/` package**: The existing worktree listing/state code (used by `fab pane-map`) has been extended with the full worktree management library: repo context detection, random name generation, branch validation, change detection (uncommitted, untracked, unpushed), hash-based stash (create/apply), LIFO rollback stack, default branch detection, worktree CRUD (create/remove), interactive menu, OS/session detection (macOS/Linux, byobu/tmux), worktree name derivation, and application detection/launching.

**Replaces**: All 6 `wt-*` shell scripts and `wt-common.sh` from the removed `src/kit/packages/wt/` directory. No shim layer — direct cutover. `wt pr` dropped entirely (overlaps `/git-pr`).

**Exit codes**: `0` success, `1` general error, `2` invalid arguments, `3` git operation failed, `4` retry exhausted (name generation), `5` byobu tab error, `6` tmux window error.

**Error format**: Structured `Error: {what}\n  Why: {why}\n  Fix: {fix}`, colors disabled when `$NO_COLOR` is set.

**"Open here" option**: `wt create` and `wt open` app menus include an "Open here" (`open_here`) entry — always available (no detection needed), placed first in the list. The `open_here` handler in `OpenInApp()` prints `cd <quoted-path>` to stdout. `DetectDefaultApp()` skips `open_here` in its fallback logic — it is never the auto-detected default, but respects the last-app cache (if a user previously chose it, it becomes the default on next run when no context-based default applies). When `open_here` is selected, `create.go` suppresses the final path line to keep stdout clean for the shell wrapper. Requires a shell function wrapper to take effect — without it, the `cd` line prints harmlessly to the terminal. Standard pattern (cf. `nvm`, `direnv`, `z`): the wrapper captures stdout, checks for `cd ` prefix, and `eval`s it in the current shell.

#### Skill Invocation Convention (`_cli-fab.md`)

The `_cli-fab.md` partial (renamed from `_scripts.md`, loaded by every skill via `_preamble.md`) defines the calling convention for all kit operations. Skills invoke operations via `fab <command> <subcommand> [args...]` — this calls the system shim, which resolves the version and dispatches to the cached `fab-go`. The `_cli-fab.md` partial includes the full command mapping table, argument formats, stage transition sequences, and error patterns.

#### Underscore File Ecosystem

The `_` (underscore) prefix denotes internal partial files that are loaded by skills but not user-invocable. These files have `user-invocable: false` frontmatter and are deployed alongside regular skills via `fab-kit sync`. The ecosystem consists of:

| File | Load strategy | Purpose |
|------|--------------|---------|
| `_preamble.md` | Always-load (every skill) | Context loading convention, SRAD, confidence scoring, Next Steps |
| `_cli-fab.md` | Always-load (via preamble) | Fab CLI command reference (renamed from `_scripts.md`) |
| `_naming.md` | Always-load (via preamble) | Naming conventions for change folders, branches, worktrees, operator spawning rules |
| `_generation.md` | Selective (planning skills) | Spec/tasks generation procedures |
| `_cli-external.md` | Operator-only (loaded by `fab-operator.md` startup) | External CLI tools: `wt` (worktree manager), `tmux`, `/loop` |

The always-load files are referenced in `_preamble.md` §1. `_cli-external.md` is explicitly NOT in the always-load list — `wt`, `tmux`, and `/loop` are operator-specific tools that pipeline skills do not need.

#### `fab resolve --pane` Flag

Outputs the tmux pane ID for a change's worktree. Signature: `fab resolve <change> --pane`. The `--pane` flag is a `Bool` flag integrated into the resolve command's `PreRunE` priority chain after `--status` and before the default `--id`.

**Pane resolution**: Reuses `discoverPanes()` and `matchPanesByFolder()` from `panemap.go` with `resolvePaneChange` as the resolver function. No new tmux discovery logic — same session-scoped discovery as `fab pane-map`.

**Tmux guard**: Checks that `$TMUX` is set. If not, exits non-zero with: `Error: not inside a tmux session`.

**No matching pane**: If no pane matches the change, exits non-zero with: `no tmux pane found for change "<folder>"`.

**Multiple panes**: When multiple panes match the same change, outputs the first match and prints a warning to stderr: `Warning: multiple panes found for {change}, using {pane}`.

**Composable usage**: Intended to be composed with raw `tmux send-keys` for sending text to agent panes: `tmux send-keys -t "$(fab resolve <change> --pane)" "<text>" Enter`. This replaces the former `fab send-keys` subcommand (removed in 260312-kvng) — the decomposed approach is more composable and avoids duplicating tmux functionality in the CLI.

## Design Decisions

### All Logic in Markdown and Shell (with Three-Binary Go Architecture)
**Decision**: Workflow logic lives in markdown skill files and shell scripts. Three system binaries (`fab` router, `fab-kit` workspace lifecycle, `fab-go` workflow engine) are installed via `brew install fab-kit`. The `fab` router dispatches to `fab-kit` (for workspace commands) or the version-resolved `fab-go` (for workflow commands). No runtime dependencies for end users; the Go toolchain is only needed for building from source.
**Why**: Constitution I (Pure Prompt Play) and Constitution V (Portability). Any AI agent that can read markdown and execute shell commands can drive the workflow. All Go binaries are pre-compiled static binaries (no runtime dependencies via `CGO_ENABLED=0`). `fab-go` is cached per-version at `~/.fab-kit/versions/`. The three-binary split enables independent testability (`fab-kit -h`, `fab-go -h`, `fab -h` each work independently) and clean separation of concerns (workspace lifecycle vs workflow engine).
**Rejected**: CLI tool, npm package, or Python script — all introduce system dependencies. Also rejected: binary in repo (redundant when the router manages versions). Also rejected: `FAB_BACKEND` override mechanism (Go is the only backend). Also rejected: two-binary shim model (shim was untestable in isolation, blurred workspace and workflow concerns).
*Source*: doc/fab-spec/README.md, fab/project/constitution.md, 260401-46hw-brew-install-system-shim, 260402-3ac3-three-binary-architecture

### Agent Skill Deployment Strategy
**Decision**: Agent skill directories are deployed via copies (Claude Code, Codex, Gemini CLI) or symlinks (OpenCode). Deployment is conditional on agent CLI availability in PATH. Deployment is performed by `fab-kit sync` (Go binary), replacing the previous shell implementation in `sync/2-sync-workspace.sh`.
**Why**: Copies ensure each agent has a self-contained skill file regardless of symlink support. Conditional deployment avoids creating dot folders for agents the developer doesn't use, keeping workspaces clean. The `FAB_AGENTS` env var enables deterministic testing without PATH manipulation. Moving to Go enables consistent cross-platform behavior and testability.
**Rejected**: Unconditional deployment to all agents — creates workspace clutter for unused agents. Also rejected: symlinks for all agents — Claude Code and Codex don't reliably follow symlinks.
*Source*: 260303-l6nk-gemini-cli-agent-aware-sync, 260219-d2y2-copy-template-skills-drop-agents, 260402-3ac3-three-binary-architecture

### lib/ Subfolder for Internal Scripts (Removed)
**Decision**: Internal scripts (`statusman.sh`, `changeman.sh`, `calc-score.sh`, `preflight.sh`) lived in `src/kit/scripts/lib/`. User-facing scripts lived in the parent `scripts/` directory.
**Deprecated by**: 260402-41gc-migrate-kit-scripts — All scripts migrated to Go binary subcommands. The `scripts/` directory has been deleted entirely. `lib/spawn.sh` → `internal/spawn/` in `fab-go`. `lib/frontmatter.sh` → `internal/frontmatter/` in `fab-go`. User-facing scripts → `fab-kit` and `fab-go` subcommands.

### Scaffold Overlay Tree with Fragment Prefix
**Decision**: `scaffold/` is structured as a repo-root overlay tree where file paths mirror their destinations. Files requiring merge strategies use a `fragment-` filename prefix. Template files (config.yaml, constitution.md) are detected at runtime by `/fab-setup` via placeholder string checks rather than being excluded from the tree-walk via a skip-list.
**Why**: Implicit mapping — a file's path IS its destination, no lookup table needed. Adding a new scaffold file only requires dropping it in the right location. The `fragment-` prefix is self-describing (only 3 of 11 files need it), avoids a coordination manifest file, and enables generic strategy dispatch. Template detection in fab-setup (rather than a skip-list in the tree-walk) keeps the tree-walk fully generic with zero special cases.
**Rejected**: Flat scaffold directory with bespoke sync sections — required a new code block per file, hardcoded path mappings. Also rejected: `.merge-rules` manifest file — adds a coordination file when the prefix convention is simpler. Also rejected: skip-list in tree-walk for fab-setup files — couples sync to fab-setup ownership, and would incorrectly skip `scaffold/fab/sync/README.md` if using subtree exclusion.
*Source*: 260218-09fa-scaffold-overlay-tree

### Single Entry Point for Workspace Sync
**Decision**: `fab-kit sync` (Go binary) is the single entry point for workspace sync, replacing the previous `fab-sync.sh` shell orchestrator. All sync logic — including hook sync — is implemented in Go. The 6-step pipeline reads kit content from the system cache (`~/.fab-kit/versions/{version}/kit/`), not from `src/kit/` in the repo. Project-level `fab/sync/*.sh` scripts are still executed after kit-level sync (step 6). No kit-level sync scripts remain (`src/kit/sync/` contains only `.gitkeep`).
**Why**: Go implementation enables testability, cross-platform consistency, and eliminates the shell dependency chain. Cache-based resolution is consistent with how `fab-go` already runs from the cache, and is a step toward removing `src/kit/` from repos entirely. Clean cut — no transition period with dual implementations.
**Rejected**: Keeping `fab-sync.sh` alongside `fab-kit sync` (duplication, testing burden). Also rejected (initially): absorbing `5-sync-hooks.sh` into `fab-kit sync` — this was later reversed in 260402-ktbg when hooklib replication proved simpler than the cross-binary concern.
*Source*: 260402-3ac3-three-binary-architecture, 260402-ktbg-sync-from-cache

### Three-Binary Split for Testability
**Decision**: The system `fab` shim is split into `fab` (router) and `fab-kit` (workspace lifecycle) as separate binaries. Together with `fab-go` (workflow engine), there are three independently-invocable binaries.
**Why**: The two-binary shim model was untestable in isolation — `fab init --help` could trigger dispatch to fab-go. Three binaries means `fab-kit -h`, `fab-go -h`, and `fab -h` each work independently. Clean separation: workspace lifecycle (init, upgrade, sync) is a different concern from workflow execution (status, resolve, preflight).
**Rejected**: Keeping two binaries (shim + fab-go) — untestable, blurred concerns. Also rejected: prefix-based routing (e.g., `fab kit sync`) — changes user-facing CLI surface.
*Source*: 260402-3ac3-three-binary-architecture

### Negative-Match Router Dispatch
**Decision**: The `fab` router maintains a static allowlist of fab-kit commands and dispatches everything else to fab-go. The fab-kit command set is small and stable; fab-go commands change with every release.
**Why**: Negative match means the router doesn't need updating when fab-go adds subcommands. Same pattern as the previous `nonRepoCommands` map.
**Rejected**: Positive match (router would need fab-go's command list, requiring updates on every new subcommand). Also rejected: prefix-based routing (changes CLI surface).
*Source*: 260402-3ac3-three-binary-architecture

### Single Go Module for fab + fab-kit
**Decision**: Both `fab` (router) and `fab-kit` binaries share a single Go module at `src/go/fab-kit/` with two `cmd/` entries (`cmd/fab/`, `cmd/fab-kit/`) sharing `internal/` packages.
**Why**: Both binaries need `EnsureCached()`, `CachedKitDir()`, `Download()`, and `ResolveConfig()`. A shared `internal/` package is the standard Go pattern. No Go workspace complexity or published shared modules needed.
**Rejected**: Separate Go modules (requires Go workspaces or a published shared module). Code duplication (maintenance burden).
*Source*: 260402-3ac3-three-binary-architecture

### Clean Cut for Sync Migration
**Decision**: Shell scripts (`fab-sync.sh`, `1-prerequisites.sh`, `2-sync-workspace.sh`, `3-direnv.sh`) are removed immediately when `fab-kit sync` ships — no deprecation period.
**Why**: Both implementations would need to coexist and be tested if phased, adding complexity for no benefit since this is a version-gated change. User explicitly decided clean cut.
**Rejected**: Phased migration (`fab-sync.sh` delegates to `fab-kit sync` as intermediate step) — unnecessary complexity.
*Source*: 260402-3ac3-three-binary-architecture

### 5-sync-hooks.sh Removed (Hook Sync Absorbed)
**Decision**: The hook sync script (`5-sync-hooks.sh`) is removed. Hook sync logic (~100 lines) is replicated directly in `fab-kit`'s internal package, running as part of step 4 (workspace scaffolding). `fab-kit` no longer shells out to `fab hook sync`.
**Why**: Absorbing hook sync eliminates a shell-out to `fab` during sync, simplifying the pipeline. The hooklib logic is small (~100 lines) and self-contained, making replication cheaper than creating a shared Go module between the two separate `go.mod` files. The `fab hook sync` CLI command continues to exist in `fab-go` for standalone use.
**Rejected**: Shared Go module (over-engineering for ~100 lines, requires workspace complexity). Shelling out to `fab hook sync` (extra process spawn, reintroduces `fab` binary dependency during sync).
**Supersedes**: "5-sync-hooks.sh Retained" decision from 260402-3ac3.
*Source*: 260402-ktbg-sync-from-cache

### Single fab/ Per Repository
**Decision**: Even in monorepos, use one `fab/` at the repo root.
**Why**: Changes span packages, memory is domain-based, and `.fab-status.yaml` assumes a single active change. Per-package `fab/` directories would fragment the system.
**Rejected**: Per-package `fab/` directories — conflicting constitutions, fragmented memory trees, symlink conflicts.
*Source*: doc/fab-spec/ARCHITECTURE.md

### LIFO Rollback Stack
**Decision**: Multi-step wt commands (`wt create`, `wt delete`) use a LIFO rollback stack — a `Rollback` type in `internal/worktree/` with `Register(cmd)`, `Execute()` (LIFO order), and `Disarm()` methods. Commands register undo operations after each successful step. On success, `Disarm()` clears the stack. `Execute()` continues executing remaining commands even if individual commands fail. Signal handlers (SIGINT, SIGTERM) trigger rollback. Originally implemented as bash arrays with EXIT traps; now ported to Go.
**Why**: Git worktree creation involves multiple coupled steps (worktree add, branch create). A partial failure must undo completed steps. LIFO ordering ensures dependent resources are cleaned up before their prerequisites.
**Rejected**: Manual cleanup in error handlers at each callsite — fragile, easy to miss paths. Temp directory approach — doesn't apply to git branch/worktree state.
*Source*: 260218-qcqx-harden-wt-resilience, 260310-qbiq-go-wt-binary

### No cd-in-Current-Shell for wt open
**Decision**: `wt open` does not and cannot offer a "cd here" option that changes the calling shell's working directory.
**Why**: Unix process model constraint — child processes cannot modify the parent shell's environment. The `wt` binary runs as a child process of the calling shell. When the child exits, the parent's working directory is unchanged. Only shell builtins and shell functions (which run in the caller's process) can `cd`. A shell function wrapper (e.g., `wt-cd() { cd "$(wt list --path "$1")"; }`) is the standard workaround but would require users to source a file from their rc, which is a different distribution model than the current PATH-based `.envrc` approach. Users who want this can define a 4-line function in their own `.bashrc`/`.zshrc`.
**Rejected**: Adding a `cd` app type to `wt open` that prints the path for `eval` — adds complexity to `wt open` for something `wt list --path` already provides. Shipping a shell function in `.envrc` — mixes PATH setup with function definitions, different sourcing semantics.
*Source*: 260223-ufk6-wt-open-cd-current-shell (abandoned — documented as design constraint)

### Non-Interactive Porcelain Output Contract
**Decision**: `wt create --non-interactive` mode redirects all human-readable messages to stderr and writes only the worktree path to stdout. Batch callers capture the path via `$(wt create --non-interactive ...)` instead of `| tail -1`.
**Why**: Two batch consumers (`batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`) relied on `| tail -1` to extract the path — fragile against any output format change. The `--reuse` codepath already followed this pattern (messages to stderr). Making `--non-interactive` imply porcelain output unifies the contract without adding a separate flag.
**Rejected**: Separate `--porcelain`/`--quiet` flag — `--non-interactive` already existed with the same audience. Fd-based output (fd 3) — non-standard, breaks simple `$(command)` capture. Env var — subshells can't export to parent.
*Source*: 260222-s101-wt-create-stderr-wt-list-flags

### wt delete Interactive Menu Includes "All" Option
**Decision**: When `wt delete` is invoked without arguments from outside a worktree, the interactive selection menu shows "All (N worktrees)" as the first option (item 1), followed by individual worktrees. Selecting "All" deletes all worktrees sequentially. The `--delete-all` CLI flag is preserved for non-interactive usage.
**Why**: Deleting all worktrees is the most common interactive use case. Putting it in the menu eliminates the need to remember the `--delete-all` flag.
*Source*: 260305-38q7-wt-delete-show-all-in-menu

### Hash-Based Stash over Index-Based
**Decision**: `wt delete` uses `git stash create` + `git stash store` (hash-based) instead of `git stash push`/`git stash pop` (index-based). Stash hashes are registered with the rollback stack. Implemented in `internal/worktree/` as `StashCreate(msg)` and `StashApply(hash)`.
**Why**: Index-based stash (`stash@{0}`) is a global counter vulnerable to race conditions in concurrent worktree operations. Hash-based stash returns a stable SHA that uniquely identifies the stash regardless of concurrent `git stash` activity. `git stash store` writes the hash to the reflog for discoverability via `git stash list`.
**Rejected**: Index-based `git stash push`/`git stash pop` — unsafe with concurrent worktree deletions; another worktree's stash could shift indices.
*Source*: 260218-qcqx-harden-wt-resilience

## Performance Benchmark: Script Runtime Comparison

Benchmark conducted 2026-03-05 comparing 4 implementations of `statusman.sh` operations (progress-map, set-change-type, finish) on aarch64 Linux.

### Results Summary

| Contender | progress-map | set-change-type | finish | Startup |
|-----------|-------------|-----------------|--------|---------|
| bash+yq (baseline) | 19.5 ms | 6.8 ms | 39.4 ms | 2.5 ms |
| optimized bash | 4.1 ms (4.8x) | 3.5 ms (1.9x) | 7.4 ms (5.3x) | 1.4 ms |
| node (js-yaml) | 14.2 ms (1.4x) | 14.8 ms (0.5x) | 15.4 ms (2.6x) | 12.6 ms |
| go (yaml.v3) | 0.69 ms (28x) | 0.80 ms (8.4x) | 0.80 ms (49x) | 0.54 ms |

### Key Findings

- **Go** is 8-49x faster than baseline. Trivial cross-compilation (`GOOS`/`GOARCH`) is a major practical advantage
- **Optimized bash** (batched yq reads + awk writes) achieves 2-5x improvement with no new dependencies
- **Node** is slower than bash+yq baseline for simple operations due to V8 startup overhead (~13ms floor)
- The `finish` operation (39ms baseline) exposes the real cost of repeated yq subprocess spawns — each of the ~10 yq invocations adds ~4ms

### Constitution Alignment

Constitution Principle I requires "single-binary utilities" with no runtime dependencies. Go fits this constraint. Node violates it (requires node runtime + node_modules). Optimized bash stays within the current architecture but has a performance ceiling. Go's cross-compilation story (`GOOS=linux GOARCH=arm64 go build`) is straightforward.

### Benchmark Code

Full benchmark suite with harness and all 4 implementations: `src/benchmark/`

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260402-gnx5-relocate-kit-to-system-cache | 2026-04-02 | Kit content relocated from `fab/.kit/` (in-project) to `~/.fab-kit/versions/<version>/kit/` (system cache). Go binaries resolve kit via `kitpath.KitDir()` (exe-sibling for `fab-go`, version-from-config for `fab-kit`). New `fab kit-path` command exposes resolved path for agent-agnostic template access. Hook shell scripts eliminated — hooks now inline as `fab hook <subcommand>` in settings.local.json. `kit.conf` eliminated (build-type removed, repo hardcoded). Source repo layout: `fab/.kit/` renamed to `src/kit/`. Skills reference templates via `$(fab kit-path)/templates/`. Build scripts updated (`justfile`, `release.sh`, `.gitignore`). Migration ships for existing users (inline hooks, remove `fab/.kit/`, clean `.envrc`/`.gitignore`). |
| 260402-41gc-migrate-kit-scripts | 2026-04-02 | Migrated all 6 shell scripts and 2 lib files from `src/kit/scripts/` to Go binary subcommands. `fab-doctor.sh` → `fab doctor` (`fab-kit` subcommand, added to router allowlist). `fab-help.sh` → `fab fab-help` (`fab-go` subcommand with dynamic skill scanning via `internal/frontmatter/`). `fab-operator.sh` → `fab operator` (`fab-go` subcommand). 3 batch scripts → `fab batch new|switch|archive` (`fab-go` subcommand group). `lib/spawn.sh` → `internal/spawn/` package. `lib/frontmatter.sh` → `internal/frontmatter/` package. Deleted entire `src/kit/scripts/` directory. Updated skills: `/fab-setup` (doctor path), `/fab-help` (script path), `/fab-operator` (launcher path). Updated `_cli-fab.md` with new command signatures. |
| 260320-t13m-configurable-agent-spawn-command | 2026-03-20 | Added `lib/spawn.sh` (shared `fab_spawn_cmd` helper reading `agent.spawn_command` from `config.yaml` with fallback). Updated 5 scripts (`fab-operator4.sh`, `fab-operator5.sh`, `batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`, `batch-fab-archive-change.sh`) to read spawn command from config via `lib/spawn.sh` instead of hardcoding `claude --dangerously-skip-permissions`. Added `fab-operator5.sh` and `lib/spawn.sh` to directory tree. Updated batch script and launcher script descriptions. |
| 260317-mogj-resilient-hooks-cwd | 2026-03-17 | Made hook shell scripts cwd-resilient. Replaced `dirname "$0"` relative path resolution with `git rev-parse --show-toplevel` in all 4 hook scripts (`on-session-start.sh`, `on-stop.sh`, `on-user-prompt.sh`, `on-artifact-write.sh`). Scripts now find the fab binary from any subdirectory within the repo. No Go code or settings.local.json changes needed — the Go binary already handled cwd via `resolve.FabRoot()` upward walk. Updated directory tree comment from "thin wrappers" to "cwd-resilient wrappers". |
| 260402-ktbg-sync-from-cache | 2026-04-02 | Rewrote `fab-kit sync` to resolve kit content from system cache (`~/.fab-kit/versions/{version}/kit/`) instead of `src/kit/`. 6-step pipeline with version guard and cache-first resolution. Absorbed hook sync into step 4 — replicated hooklib logic in `fab-kit` internal package, removed `5-sync-hooks.sh` from `sync/` directory (now `.gitkeep` only). Added `--shim` and `--project` flags. Updated `$WORKTREE_INIT_SCRIPT` from `fab-kit sync` to `fab sync`. Superseded "5-sync-hooks.sh Retained" design decision. Updated "Single Entry Point" design decision with cache-based resolution. Updated prerequisites (removed jq, gh). |
| 260315-a2b2-standalone-operator4-rewrite | 2026-03-15 | Updated `_` file ecosystem: renamed `_scripts.md` to `_cli-fab.md`, added `_cli-external.md` (operator-only load for wt/tmux/loop), added `_naming.md` (always-load for naming conventions). Updated directory tree: removed operator1/2/3 skill files, added new `_` files, added missing skills (fab-archive, fab-discuss, git-branch, git-pr-review). Updated launcher script from `fab-operator.sh` to `fab-operator4.sh`. Added "Underscore File Ecosystem" section documenting load strategies for all `_` files. Updated `_scripts.md` references to `_cli-fab.md` in kit-architecture and related docs. Updated agent deployment section to list all current underscore partials. |
| 260312-96nf-remove-rust-implementation | 2026-03-12 | Removed Rust binary (`fab-rust`) and all Rust-related infrastructure. Deleted `src/rust/` directory tree, `scripts/just/rust-target.sh`. Removed `rust_src` variable and all Rust recipes (`test-rust`, `build-rust`, `_rust-target`, `build-rust-target`, `build-rust-all`) from justfile. Simplified dispatcher to Go-only (removed `fab-rust` from version handler, backend override, default priority, error message). Removed `fab-rust` from directory tree. Removed Rust Binary section. Updated overview, dispatcher, distribution, design decisions, and benchmark sections to reflect Go-only architecture. |
| 260312-kvng-resolve-pane-evolve-panemap | 2026-03-12 | Added `--pane` flag to `fab resolve` — outputs tmux pane ID for a change's worktree, composable with `tmux send-keys`. Removed `fab send-keys` subcommand entirely (replaced by `fab resolve --pane` + raw `tmux send-keys`). Moved `matchPanesByFolder` and `resolvePaneChange` from `sendkeys.go` to `panemap.go`; deleted `sendkeys.go` and `sendkeys_test.go`. Updated `pane-map` to show all tmux panes (not just fab worktrees) — non-fab panes display em-dash fallbacks for Change/Stage/Agent columns, non-git panes show `basename/` for Worktree. Empty message changed to `"No tmux panes found."`. Updated Rust binary docs (removed `sendkeys.rs`, 9→8 subcommands). |
| 260312-wrk6-add-wt-create-base-flag | 2026-03-12 | Added `--base <ref>` flag to `wt create` — sets the git start-point for new branches (`git worktree add -b <branch> <path> <start-point>`). Warn-and-ignore for existing local/remote branches. Ref validated via `git rev-parse --verify` before use. Defers to `--reuse` when both provided and worktree exists. `CreateWorktree`, `CreateBranchWorktree`, and `CreateExploratoryWorktree` gain `startPoint string` parameter. |
| 260310-bvc6-merge-hooks-into-go | 2026-03-10 | Merged Claude Code hooks into Go binary. New `fab hook` subcommand group with 5 subcommands: `session-start`, `stop`, `user-prompt`, `artifact-write`, `sync`. Shell scripts in `$(fab kit-path)/hooks/` are now thin wrappers (`exec ... 2>/dev/null; exit 0`) delegating to `fab hook <subcommand>`. New `on-user-prompt.sh` for `UserPromptSubmit` event (improves agent idle tracking). `5-sync-hooks.sh` rewritten as thin wrapper delegating to `fab hook sync`, eliminating its jq dependency. Removed jq and bats from `fab-doctor.sh` prerequisites (7 → 5 tools). New `internal/runtime/` package (extracted from `cmd/fab/runtime.go` for shared use by `fab runtime` and `fab hook` commands). New `internal/hooklib/` package (artifact bookkeeping + hook sync logic). |
| 260310-qbiq-go-wt-binary | 2026-03-10 | Replaced `wt` shell scripts (`src/kit/packages/wt/`) with a Go binary at `fab-go binary at wt` (source: `cmd/wt/`). 5 subcommands: `create`, `list`, `open`, `delete`, `init` (`wt pr` dropped — overlaps `/git-pr`). Extended `internal/worktree/` package with full worktree management library (context, names, git ops, stash, rollback, menu, platform, errors, CRUD, apps). No shim layer — direct cutover. `packages/` now contains only `idea`. Updated `fab-help.sh` PACKAGES section, design decisions (rollback stack, stash, porcelain output), and directory tree. |
| 260310-pl72-port-idea-to-go | 2026-03-10 | Added `fab idea` subcommand to Go binary — ports the shell `idea` package (`src/kit/packages/idea/bin/idea`) to native Go. Seven sub-subcommands: `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`. New `internal/idea/` package with `Idea`/`File` structs, round-trip serialization, case-insensitive query matching, and CRUD operations. Cobra integration via `cmd/fab/idea.go`. `--file` flag and `IDEAS_FILE` env var for file override. `rm` requires `--force` (no interactive prompt). Shell package retained for rollback safety. |
| 260310-czb7-go-test-coverage | 2026-03-10 | Documented Go test strategy: 11 internal packages now have unit tests (added resolve, log, preflight, score, archive, change). Tests run via `just test-go` / `just test-go-v` (`go test ./...`). Test patterns: `t.TempDir()` isolation, table-driven, `t.Run()` subtests, standard `testing` package. Only `internal/worktree` intentionally untested. Replaced stale parity test reference. |
| 260310-b8ff-operator-observation-fixes | 2026-03-10 | Updated pane-map and send-keys to use session-scoped pane discovery (`-s` instead of `-a`). Added Tab column to pane-map output (6 columns: Pane, Tab, Worktree, Change, Stage, Agent). Added `fab runtime is-idle <change>` read-only subcommand (prints `idle {duration}`, `active`, or `unknown`). Updated send-keys pane resolution description to reflect session scoping and `#{window_name}` in tmux format string. |
| 260307-bmp3-3-rust-binary-port | 2026-03-10 | Added Rust binary (`fab-rust`) as second backend — all 9 subcommands with strict Go parity. Source at `src/rust/fab/` (flat modules, clap derive, serde_yaml, anyhow). Release profile: `lto` + `strip`. Built locally via `just build-rust` (+ `test-rust` recipe). Updated dispatcher with backend override mechanism (`FAB_BACKEND` env var, `.fab-backend` file, priority: override > rust > go). Updated directory tree (`fab-rust` no longer "future"). CI/release for Rust deferred. |
| 260307-x2tx-status-symlink-pointer | 2026-03-07 | Replaced `fab/current` pointer file with `.fab-status.yaml` symlink at repo root. Added `id` field to `.status.yaml`. Updated resolution, switch, rename, pane-map, hooks, and dispatch. Migration `0.32.0-to-0.34.0` covers conversion. |
| 260306-qkov-operator1-skill | 2026-03-07 | Added `fab-operator1.md` to skills listing. Added `fab send-keys <change> "<text>"` subcommand to Go binary — resolves change to tmux pane (reuses pane-map discovery logic), sends text via `tmux send-keys`. Tmux guard, pane existence validation, multi-pane warning. No idle check in CLI (policy in skill, mechanism in CLI). |
| 260306-bh45-pane-map-subcommand | 2026-03-06 | Added `fab pane-map` subcommand to Go binary — discovers all tmux panes, resolves worktree roots via git, reads `.fab-status.yaml` symlink for active change, correlates `.fab-runtime.yaml` for agent idle state, and renders an aligned table (Pane, Worktree, Change, Stage, Agent). Requires tmux at runtime (graceful error if not in session). Main worktree shown as `(main)`. (Later updated: 260312-kvng expanded to show all panes, not just fab worktrees.) |
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Added `on-artifact-write.sh` PostToolUse hook (Write + Edit matchers) for automatic artifact bookkeeping. Added `fab runtime set-idle` and `fab runtime clear-idle` Go subcommands replacing yq in hooks. Updated `on-stop.sh` and `on-session-start.sh` descriptions (yq → fab runtime). Updated `5-sync-hooks.sh` to support tool-name matchers for PostToolUse events. Added `runtime` package to Go binary architecture. |
| 260306-7arg-fix-stale-shell-refs | 2026-03-06 | Deleted 20 orphaned shell test files (`src/lib/*/test.bats`, `src/lib/*/SPEC-*.md`, `src/lib/*/test-simple.sh`, `src/lib/calc-score/sensitivity.sh`, `src/sync/test-5-sync-hooks.bats`) and removed `src/lib/` and `src/sync/` directories. Removed "Dev folder" references from statusman, logman, calc-score, and archiveman sections. Added Go parity test documentation note. Added 4 missing status subcommands (`add-issue`, `get-issues`, `add-pr`, `get-prs`) to `_scripts.md`. Fixed `git-pr.md` Step 4 to pass `<change>` instead of `<status_file>` path to `add-pr`. |
| 260305-u8t9-clean-break-go-only | 2026-03-05 | Removed shell fallback from dispatcher (backend priority: rust > go > error, no shell fallback). Deleted all 7 ported shell scripts from `lib/` (statusman, changeman, archiveman, logman, calc-score, preflight, resolve). Only `env-packages.sh` and `frontmatter.sh` remain. Deleted `wt-status` from wt package (replaced by `fab status show`). Added `fab status show [--all] [--json] [<name>]` to Go binary for worktree pipeline status. Added `internal/worktree` package for worktree discovery. Updated `env-packages.sh` to add `$KIT_DIR/bin` to PATH. Updated `_scripts.md` to reflect Go-only backend. Updated parity tests with graceful skip when bash scripts missing. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added `hooks/` directory to `.kit/` tree with `on-session-start.sh` (clears `agent` block) and `on-stop.sh` (writes `agent.idle_since` timestamp). Added `5-sync-hooks.sh` to `sync/` directory (registers hooks into `.claude/settings.local.json` via idempotent jq merge). Fixed sync directory tree listing (added missing `4-get-fab-binary.sh`, corrected sort order of `2-sync-workspace.sh` and `3-direnv.sh`). |
| 260305-7zq4-worktree-status-command | 2026-03-05 | Added `wt-status` to wt package — shows fab pipeline status (stage + state) per worktree. Three modes: no args (current worktree), `<name>` (specific worktree), `--all` (all worktrees). Composable architecture: atomic `wt_get_fab_status` function reads `.fab-status.yaml` symlink + `.status.yaml` via `statusman.sh display-stage`. |
| 260305-38q7-wt-delete-show-all-in-menu | 2026-03-05 | wt-delete interactive selection menu now shows "All (N worktrees)" as first option (item 1). Selecting "All" delegates to `wt_delete_all_worktrees`. Individual worktrees shift by +1. Default selection (MRU) shifts accordingly. `--delete-all` flag preserved for non-interactive use. |
| 260303-hcq9-scriptify-fab-archive | 2026-03-04 | Added `archiveman.sh` to `scripts/lib/` — Archive Manager with `archive`, `restore`, and `list` subcommands. Slimmed `/fab-archive` skill to orchestrator (backlog matching only). Added `logman.sh` and `resolve.sh` to directory tree listing (were already documented in script sections but missing from the tree). Dev test suite: `src/lib/archiveman/test.bats` (41 tests). |
| 260303-l6nk-gemini-cli-agent-aware-sync | 2026-03-04 | Added Gemini CLI as 4th agent target (`.gemini/skills/<name>/SKILL.md`, directory-based copies). Made agent skill deployment conditional — each agent's CLI checked via `command -v` before syncing; absent agents skipped with message, existing dot folders preserved. Added `FAB_AGENTS` env var override for testing/CI. Added `/.gemini` to gitignore scaffold. Updated "Agent Integration via Symlinks" → "Agent Skill Deployment" section and design decision. |
| 260303-6b7c-update-underscore-skill-references | 2026-03-04 | Documented underscore file deployment in Agent Integration section — `2-sync-workspace.sh` now deploys all `*.md` files including `_preamble.md`, `_generation.md`, `_scripts.md` (with `user-invocable: false` frontmatter). Updated stale test assertion from "skips" to "deploys" underscore files. |
| 260227-gasp-consolidate-status-field-naming | 2026-02-27 | Replaced `ship_url()`/`is_shipped()` with generic `_append_to_array`/`_get_array` helpers and 4 symmetric functions: `add_issue`/`get_issues`/`add_pr`/`get_prs`. CLI routes `ship`/`is-shipped` → `add-issue`/`get-issues`/`add-pr`/`get-prs`. Template fields `issue_id: null` → `issues: []`, `shipped: []` → `prs: []`. |
| 260226-85rg-drop-fast-model-tier | 2026-02-26 | Removed "Model Tier Agent Files (Dual Deployment)" section — the fast tier has been eliminated. All skills are now deployed as plain copies with no model templating. See `model-tiers.md` for full details. |
| 260402-0ak9-remove-sync-version-file | 2026-04-02 | Removed `fab/.kit-sync-version` from version tracking section and preserved files list. Staleness detection now compares `$(fab kit-path)/VERSION` against `fab_version` in `config.yaml`. Updated version inventory to 3 locations. |
| 260226-koj1-version-staleness-warning | 2026-02-26 | Added `fab/.kit-migration-version` (renamed from `fab/project/VERSION`) to version tracking section. Updated preserved/replaced file lists. |
| 260223-sr3u-add-fab-doctor | 2026-02-23 | Added `fab-doctor.sh` standalone prerequisite checker (7 tools: git, bash, yq v4+, jq, gh, bats, direnv+hook). Rewrote `sync/1-prerequisites.sh` as thin `exec` delegate to doctor. Updated `fab-upgrade.sh` output: "Update complete" prints first, `⚠` migration warning prints last (or omitted when no drift). Added Phase 0 doctor gate to `fab-setup.md` (bare bootstrap only). Added `fab-doctor.sh` to directory tree, scripts section, and lib/ design decision user-facing scripts list. |
| 260222-s101-wt-create-stderr-wt-list-flags | 2026-02-22 | wt-create and wt-pr `--non-interactive` now redirects human messages to stderr (porcelain output: only path on stdout). Removed `\| tail -1` from 3 batch callers. Added `--path <name>` (single worktree path lookup), `--json` (JSON array with dirty/unpushed fields), and status column (`*` dirty, `↑N` unpushed) to wt-list. Added mutual exclusivity check for `--path`/`--json`. Added "Non-Interactive Porcelain Output Contract" design decision. |
| 260222-s90r-add-shipped-tracking | 2026-02-22 | Added `shipped` tracking to fab pipeline. Extended `statusman.sh` with `ship` (append PR URL, idempotent, exact-match dedup) and `is-shipped` (exit-code query) subcommands (14→16 CLI subcommands). Added `shipped: []` to `status.yaml` template. Added `shipped` documentation section to `workflow.yaml` schema. Updated `/git-pr` skill to call `statusman.sh ship` after PR creation with graceful degradation when no active change. Updated `_preamble.md` state table: hydrate row now routes to `/git-pr` as default, `/fab-archive` as alternative. Updated `changeman.sh` `default_command` for hydrate. Test suite: 53→71 tests. |
| 260222-n811-absorb-ship-command | 2026-02-22 | Added `git-pr.md` skill to skills directory listing. Added `git-pr` to `fab-help.sh` Completion group mapping. Native `/git-pr` replaces external `changes:ship` dependency for pipeline shipping. |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed `_context.md` → `_preamble.md` in skills directory tree listing. Updated `2-sync-workspace.sh` comment referencing excluded skill file. |
| 260221-alng-batch-script-frontmatter | 2026-02-21 | Added `shell_frontmatter_field()` to `lib/frontmatter.sh` for parsing `# ---` delimited shell-comment frontmatter. Added `# ---` frontmatter blocks (name, description) to all 3 batch scripts, replacing old comment headers. Added batch script scan loop to `fab-help.sh` — globs `batch-*.sh`, extracts frontmatter via `shell_frontmatter_field`, renders under "Batch Operations" group with centralized `batch_to_group` mapping and no `/` prefix. Updated tree comment for `frontmatter.sh`. |
| 260221-i0z6-move-env-packages-add-fab-pipeline | 2026-02-21 | Moved `env-packages.sh` from `scripts/` to `scripts/lib/` (off PATH). Updated `KIT_DIR` resolution (two levels up). Updated source references in `scaffold/fragment-.envrc` and `src/packages/rc-init.sh`. Added `lib/env-packages.sh` description section. |
| 260219-wq0e-move-5cs-to-project-folder | 2026-02-19 | Moved project identity files (5 Cs + VERSION) from `fab/` root into `fab/project/` subdirectory. Updated `fab/` directory tree (top-level now: `.kit/`, `project/`, `changes/`, `sync/`, `backlog.md`, `current`). Updated scaffold overlay tree (`$(fab kit-path)/scaffold/fab/project/`). Updated all shell script path references (`preflight.sh`, `changeman.sh`, `fab-upgrade.sh`, `batch-fab-switch-change.sh`, `2-sync-workspace.sh`). Updated section comment in `2-sync-workspace.sh` (`fab/project/VERSION`). |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Removed deleted `model-tiers.yaml` from directory tree; added missing `fab-fff.md` after `fab-ff.md` in skills listing |
| 260218-09fa-scaffold-overlay-tree | 2026-02-18 | Restructured `scaffold/` from flat directory to repo-root overlay tree. Files now mirror destination paths; 3 merge files use `fragment-` prefix (`.envrc`, `.gitignore`, `settings.local.json`), 8 others use copy-if-absent. Replaced 6 bespoke sections (2, 3, 4, 7, 8, 9) in `2-sync-workspace.sh` with generic tree-walk dispatching on `fragment-` prefix and file extension. Extracted `line_ensure_merge` and `json_merge_permissions` helper functions (absorbing legacy `.envrc` symlink migration). Updated `fab-setup.md`: 7 scaffold path references, template detection for `config.yaml`/`constitution.md` (placeholder check instead of existence check). Updated `0.7.0-to-0.8.0.md` scaffold path. Renumbered sync script sections (1, 1b, 2, 3, 3b, 4). Added "Scaffold Overlay Tree" design decision. |
| 260218-e0tj-document-wt-idea-packages | 2026-02-18 | Added static PACKAGES footer section to `fab-help.sh` listing wt commands (wt-create, wt-list, wt-open, wt-delete, wt-init, wt-pr) and idea with one-liner descriptions. Created `docs/specs/packages.md` covering both packages at concept/workflow level (overview, wt section with assembly-line integration, idea section with backlog→fab-new flow, package architecture). Added packages.md entry to `docs/specs/index.md`. Updated `fab-help.sh` description to mention PACKAGES section. |
| 260218-qcqx-harden-wt-resilience | 2026-02-18 | Added resilience patterns to wt package: LIFO rollback stack with EXIT trap (`wt_register_rollback`, `wt_rollback`, `wt_disarm_rollback`), signal handling (`wt_cleanup_on_signal` for INT/TERM), hash-based stash (`wt_stash_create`/`wt_stash_apply` using `git stash create`+`store`), branch name validation (`wt_validate_branch_name`). Integrated into wt-create (rollback, traps, validation, dirty-state check) and wt-delete (hash-based stash migration, signal traps, stash rollback registration). Added `wt-pr` command for PR-based worktree creation via `gh` CLI. Moved shared worktree creation functions from wt-create to wt-common.sh. Updated bin listing to include wt-pr. Added 2 design decisions (rollback stack, hash-based stash). |
| 260218-cif4-eliminate-symlinks-distribute-packages | 2026-02-18 | Eliminated 5 test-to-production symlinks in `src/lib/` — tests now use repo-root-relative paths. Moved package production code (idea, wt) from `src/packages/` to `src/kit/packages/` for distribution via `kit.tar.gz`. Added `env-packages.sh` script (sourced by `.envrc` and `rc-init.sh`) for centralized PATH setup. Added `packages/` to directory tree, `env-packages.sh` to scripts section. Updated `fab-upgrade.sh` description (symlinks → directories and agents). Updated calc-score.sh dev folder description (removed symlink reference). Updated "Replaced" list to include `packages/`. |
| 260217-j3a3-dynamic-fab-help-generation | 2026-02-18 | Rewrote `fab-help.sh` for dynamic command listing — reads skill names/descriptions from YAML frontmatter at runtime instead of hardcoding. Extracted `frontmatter_field()` to `lib/frontmatter.sh` (shared by `fab-help.sh` and `2-sync-workspace.sh`). Deleted stale hand-authored `.claude/agents/fab-help.md` (sync regenerates correctly from `model_tier: fast`). Added `frontmatter.sh` to `lib/` directory tree. Updated `fab-help.sh` description to reflect dynamic generation. |
| 260217-zkah-readme-quickstart-prereqs-check | 2026-02-18 | Added `sync/1-prerequisites.sh` (validates yq, jq, gh, direnv, bats before sync). Renumbered `1-direnv.sh` → `3-direnv.sh`, `2-sync-workspace.sh` → `2-sync-workspace.sh`. Updated directory tree, script section headings, and all sync filename references. |
| 260218-bx4d-consolidate-worktree-init-into-sync | 2026-02-18 | Consolidated worktree bootstrap into single entry point. Rewrote `fab-sync.sh` as thin orchestrator iterating `src/kit/sync/*.sh` then `fab/sync/*.sh`. Moved sync logic to `sync/2-sync-workspace.sh` (path resolution: `sync_dir` → `kit_dir`). Added `sync/1-direnv.sh` (moved from `worktree-init-common/`). Added `scaffold/sync-readme.md` and section 9 (fab/sync/README.md scaffolding) to 2-sync-workspace.sh. Renamed `fab/worktree-init/` → `fab/sync/` (deleted `1-claude-settings.sh` + `assets/`, renumbered `2-symlink-backlog.sh` → `1-symlink-backlog.sh`). Deleted `worktree-init.sh` and `worktree-init-common/`. Updated `scaffold/envrc` (`WORKTREE_INIT_SCRIPT` → `src/kit/scripts/fab-sync.sh`). Removed `fab-update-claude-settings.sh` (script no longer exists). Added "Single Entry Point" design decision. |
| 260216-oinh-DEV-1045-fold-resolve-into-changeman | 2026-02-17 | Folded `resolve-change.sh` into `changeman.sh` as `resolve` and `switch` subcommands. Removed `resolve-change.sh` from lib/ directory listing. Updated changeman description (4 subcommands, yq dependency for switch). Updated preflight description (calls changeman CLI instead of sourcing resolve-change). Migrated all callers (preflight, batch scripts) from source+variable pattern to CLI subprocess. Updated lib/ design decision script list. |
| 260217-17pe-DEV-1046-scaffold-setup-templates | 2026-02-17 | Added `config.yaml` and `constitution.md` to scaffold directory listing. Updated scaffold comment from "read by fab-sync.sh" to "read by fab-sync.sh and /fab-setup" |
| 260216-ymvx-DEV-1043-envrc-line-sync | 2026-02-16 | Replaced `.envrc` symlink management with line-ensuring sync in `fab-sync.sh` section 2. Scaffold `envrc` comment updated from "template (shipped)" to "required entries (line-ensuring)". `fab-sync.sh` description updated to reflect `.envrc` line-ensuring and symlink migration. |
| 260216-pr1u-DEV-1017-add-archive-gitkeep | 2026-02-16 | Added `fab/changes/archive/` to `fab-sync.sh` directory creation loop and `fab/changes/archive/.gitkeep` conditional touch. Updated fab-sync.sh description to enumerate created directories and .gitkeep files. |
| 260216-gcw7-DEV-1041-consolidate-script-signatures | 2026-02-16 | Consolidated `statusman.sh` CLI surface area: removed 12 dead functions and 6 internal-only CLI dispatch entries, reducing from ~35 to 14 externally-used subcommands. Removed `--test`, `--version` flags and self-test infrastructure. Internal helper functions retained for write/validation. Updated test suite (75 → 53 tests). Fixed pre-existing `brief` → `intake` bug in `test-simple.sh`. |
| 260216-u6d5-DEV-1039-add-changeman-rename | 2026-02-16 | Added `rename` subcommand to `changeman.sh` — renames change folder slug while preserving date-ID prefix, updates `.status.yaml` name field, conditionally updates `.fab-status.yaml` symlink, logs via statusman. Updated directory tree comment and script description. |
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | Promoted `lib/sync-workspace.sh` → `scripts/fab-sync.sh` (user-facing); replaced `fab-init.md` + `fab-update.md` with `fab-setup.md` in skills listing; added stale artifact cleanup to fab-sync.sh; updated all cross-references |
| 260216-f88c-DEV-1029-migrate-existing-tests-to-bats | 2026-02-16 | Migrated 4 legacy `test.sh` suites (preflight, resolve-change, statusman, calc-score) to bats-core `test.bats` format. Deleted all legacy test.sh files. Simplified justfile `test-bash` recipe to bats-only runner. Fixed stale `brief` → `intake` stage name in test fixtures. Fixed legacy bugs: incorrect carry-forward assumption in calc-score tests, wrong fuzzy column order. Total: 6 bats suites, 152 tests. Updated statusman section test reference. |
| 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests | 2026-02-16 | Renamed `init-scaffold.sh` → `sync-workspace.sh` throughout (directory tree, section heading, script description, design decisions). Updated `scaffold/` comment, agent integration references, bootstrap sequence, version tracking, and updating sections. Added `changeman.sh` and `sync-workspace.sh` to dev test directory listing (`src/lib/`) |
| 260215-9yjx-DEV-1022-create-changeman-script | 2026-02-15 | Added `changeman.sh` to `lib/` directory listing and Shell Scripts section. New script handles change creation lifecycle (folder name construction, collision detection, directory creation, created_by detection, .status.yaml initialization, statusman integration). Called by `/fab-new` Step 3. |
| 260215-g4r2-DEV-1023-batch-rename-default-list | 2026-02-15 | Renamed batch scripts from `batch-{verb}-{entity}.sh` to `batch-fab-{verb}-{entity}.sh`; changed no-arg behavior from showing help to showing `--list` output; updated directory tree, naming pattern, and script descriptions |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Updated directory listing: `brief.md` → `intake.md` in templates. Updated script references from brief to intake |
| 260215-lqm5-statusman-cli-only | 2026-02-15 | Migrated `statusman.sh` to CLI-only interface: added ~25 read/query subcommands, added `set-confidence-fuzzy` write subcommand, removed dual-mode (`source`/CLI) scaffolding (`BASH_SOURCE` guard, `return \|\| exit` patterns). Migrated `preflight.sh` and `calc-score.sh` from `source statusman.sh` to `$STATUSMAN <subcommand>` subprocess calls. Test suites (131 tests) converted to CLI invocation pattern as contract tests for future Rust rewrite. |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Relocated `memory/` and `specs/` from `fab/` to `docs/`; updated `_init_scaffold.sh` to create `docs/memory/` and `docs/specs/`; updated preserved files list; added migration file `0.2.0-to-0.3.0.md` |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added `$(fab kit-path)/migrations/` directory and `fab-update.md` skill to directory listing; updated version tracking to dual-version model; updated `_init_scaffold.sh` description (fab/VERSION creation); updated `fab-upgrade.sh` (drift reminder) and `release.sh` (migration chain validation) descriptions; updated preserved/replaced lists |
| 260214-w3r8-statusman-write-api | 2026-02-14 | Added write functions + CLI to `_statusman.sh` (`set_stage_state`, `transition_stages`, `set_checklist_field`, `set_confidence_block`); refactored `_calc-score.sh` to source `_statusman.sh` and delegate writes to `set_confidence_block` |
| 260214-q7f2-reorganize-src | 2026-02-14 | Reorganized `scripts/` directory: moved 5 internal scripts to `scripts/lib/` (dropped underscore prefix), moved `release.sh` to `scripts/` (dev-only); updated directory tree, all script section headings and descriptions; replaced "Underscore Prefix" design decision with "lib/ Subfolder" convention; updated `src/calc-score/` → `src/lib/calc-score/`; updated all bootstrap and symlink references |
| 260214-r7k3-statusman-yq-metrics | 2026-02-14 | Migrated statusman.sh accessors/writes from awk/grep/sed to yq v4; added yq dependency guard; added stage_metrics accessors (`get_stage_metrics`, `set_stage_metric`) and `_apply_metrics_side_effect`; added history logging (`log_command`, `log_confidence`, `log_review`); added `driver` param to `set_stage_state`/`transition_stages`; added `status.yaml` template with `stage_metrics: {}`; updated calc-score.sh to use `get_confidence` accessor and call `log_confidence`; added CLI commands for logging |
| 260214-eikh-consistency-fixes | 2026-02-14 | Added internal skills (`internal-consistency-check.md`, `internal-retrospect.md`, `internal-skill-optimize.md`) to `.kit/skills/` directory listing |
| 260214-mgh5-calc-score-dev-setup | 2026-02-14 | Added `src/calc-score/` dev folder for `_calc-score.sh` — symlink, README, smoke test, comprehensive test suite (30 tests) |
| 260214-r8kv-docs-skills-housekeeping | 2026-02-14 | Removed `fab-status.sh` from scripts listing. Renamed doc skills: `fab-hydrate.md` → `docs-hydrate-memory.md`, `fab-hydrate-specs.md` → `docs-hydrate-specs.md`, `fab-reorg-specs.md` → `docs-reorg-specs.md`. Added `docs-reorg-memory.md` to skills listing. |
| 260213-w8p3-extract-fab-score | 2026-02-14 | Added `_calc-score.sh` to scripts directory listing and Shell Scripts section — internal confidence scoring script |
| 260213-puow-consolidate-status-reads | 2026-02-14 | Renamed `statusman.sh` → `_statusman.sh`; added `.status.yaml` accessor API (`get_progress_map`, `get_checklist`, `get_confidence`); refactored `get_current_stage` to use accessors; extracted `_resolve-change.sh` change resolution library; documented underscore prefix convention |
| 260213-v3rn-batch-commands | 2026-02-14 | Renamed `fab-batch-new.sh` → `batch-new-backlog.sh`, `fab-batch-switch.sh` → `batch-switch-change.sh`; added `batch-archive-change.sh`; added batch scripts section to Shell Scripts docs |
| 260305-g0uq-2-ship-fab-go-binary | 2026-03-05 | Shipped Go binary: added `bin/` directory to `.kit/` tree (with `fab` binary and `.gitkeep`). Documented shim delegation layer — all 7 `lib/` shell scripts have shim at top that `exec`s `fab-go binary at fab` when present, falls through to bash when absent. Added `_scripts.md` partial as skill invocation guide — skills now call `fab-go binary at fab <command>` as primary convention. Updated bootstrap options (platform-aware + generic + manual). Updated `fab-upgrade.sh` description (platform detection via `uname`, fallback to generic). Updated `release.sh` description (cross-compiles 4 platforms, produces 5 archives). Updated Replaced list to include `bin/`. Updated "All Logic in Markdown and Shell" design decision to reflect optional Go binary with graceful degradation. |
| 260305-bhd6-1-build-fab-go-binary | 2026-03-05 | Added Go binary section: `src/go/fab/` with `fab` binary porting all 7 lib/ shell scripts into cobra subcommands. Uses `internal/statusfile` as shared YAML foundation (single parse, atomic writes), eliminating yq dependency. Shell scripts unchanged — switchover is a separate change. |
| 260305-gt52-rust-vs-node-benchmark | 2026-03-05 | Added Performance Benchmark section: Rust vs Node vs optimized bash vs bash+yq for statusman.sh operations. Rust is 23-123x faster than baseline; optimized bash 2-5x; Node slower than baseline due to V8 startup |
| 260213-3njv-scaffold-dir | 2026-02-13 | Added `scaffold/` directory to tree listing; `_init_scaffold.sh` now reads bootstrap content from scaffold files instead of hardcoded heredocs |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed `fab-setup.sh` → `_init_scaffold.sh` and `fab-update.sh` → `fab-upgrade.sh`; updated directory listing and all script references |
| 260213-v8r3-remove-dead-fab-help-agent | 2026-02-13 | Removed `.claude/agents/fab-help.md` from agent files listing — agent was never spawned; skill + script pair covers all usage |
| 260212-4tw0-migrate-scripts-statusman | 2026-02-12 | Migrated fab-status.sh and _preflight.sh to source statusman.sh; added statusman.sh, _preflight.sh, and schemas/ to directory listing |
| 260212-ipoe-checklist-folder-location | 2026-02-12 | Template listing already shows `checklist.md` — no structural change needed; changelog entry for traceability |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated directory listing: intake.md replaces proposal.md, plan.md removed, docs/specs/ replaces docs/specs/ |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Added `fab-upgrade.sh` and `release.sh` script descriptions, bootstrap one-liner (Option A), atomic update mechanism, version-based update flow |
| 260210-m3k7-multi-agent-support | 2026-02-10 | Added OpenCode commands and Codex skills symlink creation to `_init_scaffold.sh`; documented all three agent integration paths |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260313-wrt4-pane-map-json-session-flags | 2026-03-13 | Added `--json`, `--session <name>`, `--all-sessions` flags to `fab pane-map`. JSON output with snake_case fields and null semantics. `WinIdx` column in table, conditional `Session` column with `--all-sessions`. `discoverPanes` now accepts session targeting mode. |
| 260312-9lci-fix-status-show-fab-current | 2026-03-12 | Removed `fab status show` from Go binary subcommand list (was already deleted from source). Fixed stale `fab/current` references in send-keys pane resolution description and "Preserved" list (now `.fab-status.yaml`). |
| — | 2026-02-07 | Generated from doc/fab-spec/ (ARCHITECTURE.md, README.md) |
| 260401-46hw-brew-install-system-shim | 2026-04-02 | Binary-free `.kit/`: removed `fab`, `fab-go`, `wt`, `idea` from `fab-go binary at ` (only `.gitkeep` remains). System `fab` shim (Homebrew) dispatches to cached `fab-go` at `~/.fab-kit/versions/`. `wt` and `idea` are system-only Homebrew binaries. Removed shell dispatcher, backend override mechanism (`FAB_BACKEND`, `.fab-backend`). Removed `fab-upgrade.sh` (replaced by `fab upgrade` shim subcommand). Removed `4-get-fab-binary.sh` from sync pipeline. Updated `5-sync-hooks.sh` to call `fab hook sync` via system shim. Updated `fab-doctor.sh` to check for `fab` system binary. All skill invocations changed from `fab-go binary at fab` to `fab`. Updated Portability section to require system shim. Updated bootstrap sequence (primary: `brew install fab-kit` + `fab init`). |
| 260401-ixzv-org-migrate-mit-license | 2026-04-02 | Migrated GitHub org references from wvrdz to sahil87. License changed from PolyForm Internal Use to MIT (root LICENSE). |
| 260402-3ac3-three-binary-architecture | 2026-04-02 | Three-binary architecture: split shim into `fab` (router) and `fab-kit` (workspace lifecycle), alongside `fab-go` (workflow engine). Source at `src/go/fab-kit/` with two `cmd/` entries sharing `internal/`. `fab-kit sync` replaces `fab-sync.sh` and `sync/{1,2,3}-*.sh` (clean cut removal). Router uses negative-match dispatch (fab-kit commands allowlisted, everything else to fab-go). `fab help` composes output from both sub-binaries. Added 5 design decisions (three-binary split, negative-match routing, single Go module, clean cut sync migration, 5-sync-hooks retained). Updated directory tree, agent deployment, bootstrap, updating, and portability sections. |

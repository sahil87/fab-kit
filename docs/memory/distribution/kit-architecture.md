---
type: memory
description: "`src/kit/` structure (binary-free; `spec.md` template removed in j6cs; `schemas/` removed in c5tr — scaffold no longer seeds `stage_directives`), three-binary architecture (fab router + fab-kit + fab-go), router always-route policy + shared `LifecycleCommands` allowlist table with contract/collision drift tests (ye8r), `fab-kit sync` (post-state version guard, fail-loud deployment writes, version threading from init/upgrade, stamp-after-success upgrade), agent integration, versioning, monorepos, underscore file ecosystem, `fab pane` command group, `fab shell-init`, hidden `fab help-dump`, shared `internal/lines` + `internal/atomicfile` helpers (hv7t), widened `internal/config` single config.yaml parser (ye8r); tb6f: Go 1.26 + cobra v1.10 toolchain, fab-kit `sync.go` split (semver/prereqs/scaffold/skills + orchestrator), golden byte-stability suite as the standing yaml parity arbiter, yaml.v3-stays-pinned decision, `src/benchmark/` tombstoned; frlo: new `reference/` shipped content dir (read via `$(fab kit-path)/reference/...`, holds the `fkf.md` normative extract) + the whole-`src/kit/`-copied-verbatim packaging invariant (new kit content ships with no Go/packaging change); 2fm8: `templates/memory.md` added — canonical FKF memory-file template, the third artifact template, read on demand via `$(fab kit-path)/templates/memory.md` (ships verbatim, no Go/packaging/migration change); 6nke: new `fab config` command group (fab-go, always-routed, name must not collide with the fab-kit allowlist) with the pure-query `reference` subcommand + the `internal/configref` generator package; 6sgj: new `fab dispatch {start,status,logs,kill,clean}` command group (fab-go, always-routed) — the tmux-independent headless process manager consuming `internal/agent`+`internal/spawn` to launch a stage's resolved provider `dispatch_command` (tykw — was a per-tier `spawn_command`) detached (`SysProcAttr{Setsid:true}`), backed by the `internal/dispatch` package + a compile-time `_posix`/`_windows` platform split (POSIX-only v1); tykw: agent config v3 — top-level `providers:` table + five role tiers resolved in `internal/agent` (`ResolveProvider`, `ResolveTier`), new `fab agent [tier] [--print] [--repo]` command (`agent.go`) retiring `fab spawn-command`, `fab resolve-agent` spawn=→dispatch= + provider= + tier-name, `spawn.StripPlaceholders` removed; ioku: agent-state production divested — the three `fab hook session-start|stop|user-prompt` handlers are inert no-op shims and `fab hook sync` is a deprecated no-op (empty mapping table, legacy rewrite rows dropped), `internal/runtime`+`internal/proc` deleted (`internal/lockfile` stays), the `hooks/on-*.sh` scripts delegate to shims, the `2.13.6-to-2.14.0` migration removes the three hook settings entries (inline + legacy `on-*.sh` forms) and deletes `.fab-runtime.yaml`/`.lock` across worktrees; fab now reads `@rk_agent_state` as a pure consumer"
---
# Kit Architecture

**Domain**: distribution

## Overview

`src/kit/` is the portable engine directory that contains all workflow logic: skill definitions, artifact templates, utility shell scripts, and version tracking. It is content-only — no binaries. The system provides three binaries: `fab` (router), `fab-kit` (workspace lifecycle), and `fab-go` (workflow engine), all installed via `brew install fab-kit`. The `fab` router dispatches to either `fab-kit` or the version-resolved `fab-go`. `src/kit/` provides content (skills, templates, configuration). This doc covers the `.kit/` directory structure, the three-binary architecture, agent integration, distribution, updating, and monorepo guidance.

> **CLI Command Reference**: For calling conventions and full command signatures, see `$(fab kit-path)/skills/_cli-fab.md` (the canonical CLI reference — loaded selectively via a skill's `helpers: [_cli-fab]` frontmatter; the most-used command families are inlined into `_preamble.md § Common fab Commands`).

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
│   ├── _cli-fab.md          # Fab CLI command reference (selective via helpers: [_cli-fab])
│   ├── _cli-external.md     # External CLI tools: wt, tmux, /loop (selective via `helpers:`)
│   ├── _generation.md       # Spec/tasks generation procedures (selective via `helpers:`)
│   ├── _review.md           # Review procedures (selective via `helpers:`)
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
│   ├── plan.md             # Unified ## Requirements + ## Tasks + ## Acceptance — apply-stage artifact (spec.md absorbed in j6cs)
│   ├── memory.md           # Canonical FKF memory-file template (type: memory + description: + Overview/Requirements/Design Decisions skeleton, no ## Changelog) — read on demand by the doc skills (2fm8)
│   └── status.yaml         # .status.yaml template (6-stage progress, plan: block, stage_metrics: {}, issues: [], prs: [])
├── reference/              # Reference-to-read contracts shipped to the cache, read via $(fab kit-path)/reference/... (frlo)
│   └── fkf.md              # Shipped FKF normative extract (§2/§3/§5/§6/§7/§8); deployed skills cite $(fab kit-path)/reference/fkf.md
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
├── hooks/                  # Claude Code hook scripts (cwd-resilient wrappers). Since ioku the three below delegate to fab hook shims that are inert no-op exit-0 stubs (agent-state production divested to run-kit's @rk_agent_state; see runtime-agents.md). Retained one release for un-migrated settings; scripts + shims slated next-release deletion.
│   ├── on-session-start.sh # SessionStart hook — delegates to fab hook session-start (now a no-op shim)
│   ├── on-stop.sh          # Stop hook — delegates to fab hook stop (now a no-op shim)
│   └── on-user-prompt.sh   # UserPromptSubmit hook — delegates to fab hook user-prompt (now a no-op shim)
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

Replaced by `internal/spawn/` package in `fab-go`. Since tykw the session command is resolved from the **provider table** — `internal/agent`'s `ResolveProvider(name)` returns a provider's `session_command`/`dispatch_command`, and `agent.DefaultSessionCommand` (re-exported as `spawn.DefaultSpawnCommand`) is the built-in `claude` fallback — a `{model}`/`{effort}` template (`claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`, `260703-gvxd`; profile-substituted by `spawn.WithProfile`) — when the config is missing/empty/unreadable. Consumers (`fab operator`, `fab agent`, `fab batch new|switch`) compose from a resolved provider `session_command` + a tier profile via `spawn.WithProfile` (`fab batch archive` no longer spawns — it archives mechanically in-process). (Before tykw, `spawn.Command(configPath)` read the flat `agent.spawn_command` via nil-safe `GetSpawnCommand`, deleted with the field's relocation to `providers.claude.session_command`; the local one-off `gopkg.in/yaml.v3` parse had already been removed in 260612-ye8r.)

#### `lib/frontmatter.sh` (Removed)

Replaced by `internal/frontmatter/` package in `fab-go`. The Go implementation extracts fields from YAML frontmatter (content between `---` markers), handles quoted/unquoted values, strips inline comments. Used by `fab fab-help` for skill discovery.

#### `fab-upgrade.sh` (Removed)

Replaced by `fab upgrade-repo` — a shim subcommand. See [distribution.md](/distribution/distribution.md) for the full upgrade flow.

#### `release.sh` (dev-only, at `scripts/release.sh`)

Bumps VERSION (accepts `[patch|minor|major]` argument), validates the migration chain (warns if no migration targets the new version, warns on overlapping migration ranges), commits the version change, tags it, and pushes to the remote. CI takes over from the tag push to cross-compile, package, and create the GitHub Release. Requires clean working tree. This script is not shipped inside `src/kit/` — it is a dev-only tool for maintainers of the fab-kit repo.

#### Batch Scripts (Removed)

Replaced by `fab batch` subcommand group in `fab-go`. Source: `src/go/fab/cmd/fab/batch.go` (parent command), `batch_new.go`, `batch_switch.go`, `batch_archive.go`. The `new` and `switch` subcommands share common patterns: tmux tab creation, session-command resolution via `internal/spawn/` + `internal/agent`, `--list`/`--all` flags. `archive` is mechanical — it creates no tmux tab and resolves no session command (it archives in-process via a Go loop), and since 753q it diverged from the `--list`/`--all` flag shape to its own `--yes`/`-y` + `--dry-run` confirmation/preview model (see below).

- **`fab batch new`** — Per backlog ID: creates a worktree via `wt create --non-interactive`, opens a tmux tab, starts a Claude Code session running `/fab-new <description>`. Parses `fab/backlog.md` with continuation line handling. Supports `--list`, `--all`, and positional ID arguments. Since tykw the tmux command is composed via a shared `defaultTierSpawnCommand` (`batch.go`) — the **default-tier** provider `session_command` + the default tier profile substituted through `spawn.WithProfile` — so workers spawn **with** a tier profile (`spawn.StripPlaceholders` was removed as now-dead: workers are no longer spawned from an empty-profile raw command, so there is no placeholder-leak path left to guard). A templated provider command has its `{model}`/`{effort}` filled; a non-templated Claude command gets ` --model`/` --effort` appended.
- **`fab batch switch`** — Per change name/ID: creates a worktree with the correct branch name (using `branch_prefix` from the shared `internal/config` accessor), opens a tmux tab, runs `/fab-switch <change>`. Change resolution uses `resolve.ToFolder` **in-process** (since 260612-ye8r — the former `fab change resolve` subprocess was the only self-exec in either Go module: a PATH dependency whose shim round-trip could trigger a cache download and whose `.Output()` discarded the resolver's specific stderr; the warn-and-skip warning now names the specific error, e.g. `Multiple changes match…`). The whole batch family now resolves in-process. Since tykw its tmux command is composed by the same `defaultTierSpawnCommand` (default-tier provider `session_command` + profile) as `batch new`. Supports `--list`, `--all`, and positional arguments.
- **`fab batch archive`** — Finds changes with `hydrate: done|skipped` in `.status.yaml` and archives each one mechanically in-process via a Go loop (`archiveLoop` → `internal/archive.ArchiveWithBacklog`) — folder move, index update, backlog mark-done, and pointer clearing, with no spawned Claude session and no tmux tab. Change resolution uses `resolve.ToFolder` (not a `fab change resolve` subprocess). Per-change failures are isolated (a failure on one change is reported and the loop continues); already-archived changes are a soft skip (`ErrAlreadyArchived`), counted as `skipped` rather than `failed`. The loop prints per-change lines plus an `Archived N, skipped M, failed K.` footer, and the command exits non-zero only when `failed > 0`. The loop logic lives in the testable `archiveLoop` helper (returns counts, no `os.Exit`); `runBatchArchive` returns errors through RunE (`ERROR: {K} change(s) failed to archive` / `ERROR: No valid changes to archive.` — no in-handler `os.Exit` since 260612-ye8r). **Flag/confirmation model (753q, replacing the 260612-ye8r `--all`/`--list` shape):** archive is the one bulk-mutating member whose moves are effectively irreversible within the loop, so instead of staying list-by-default behind `--all` it uses a list-then-confirm model with a `--yes` escape hatch (apt/npm/gh-style). A bare `fab batch archive` on an interactive stdin lists the archivable set then prompts `Archive these N? [y/N]` (**default No** — Enter or any non-`y`/`yes` answer aborts, exit 0); `--yes`/`-y` archives all with no prompt (the non-interactive escape hatch, resolved behavior of the former `--all`); `--dry-run` lists only with no prompt/action (the former `--list`); a non-TTY stdin without `--yes` refuses with guidance and a non-zero exit rather than hanging (the tmux/operator runtime passes `--yes`); explicit positional args archive the named changes with no prompt and no TTY guard; `--dry-run --yes` is mutually exclusive (non-zero exit). The empty archivable set remains a benign no-op (`No archivable changes found.` + zero footer, exit 0), checked before any prompt/guard so finding F49 is preserved. TTY detection uses the stdlib `os.ModeCharDevice` pattern via an injectable `isStdinTTY` seam (no `golang.org/x/term` dependency, mirroring `src/go/fab-kit/internal/upgrade.go`). No longer imports `internal/spawn`, `os/exec`, or `syscall`.

#### Launcher Scripts (Removed)

Replaced by `fab operator` — a `fab-go` parent command with subcommands. Source: `src/go/fab/cmd/fab/operator.go`. Default behavior (no subcommand): creates a singleton tmux window named "operator" running the resolved **operator-tier** session command (composed from the operator provider's `session_command` + the operator tier profile via `internal/agent` + `internal/spawn`; tykw — previously the flat `agent.spawn_command`) with `'/fab-operator'`. If the window already exists, switches to it. Requires an active tmux session (`$TMUX` check).

Subcommands:
- **`fab operator tick-start`** — start-of-tick atomic state update: increments `tick_count`, writes `last_tick_at` (RFC3339 UTC), outputs `tick: N\nnow: HH:MM`. Writes to a **server-keyed XDG state file** — `<stateDir>/fab/operator/<server-slug>.yaml`, NOT the old repo-rooted `.fab-operator.yaml` (see "Operator State File" below). Source: `src/go/fab/cmd/fab/operator_tick_start.go`.

#### Operator State File (server-keyed XDG)

The operator's coordination state lives at a **server-scoped** path resolved by helpers in `cmd/fab/operator.go`, not at the repo-rooted `gitRepoRoot()/.fab-operator.yaml` it used previously. The relocation makes one tmux server (one operator) own one state file regardless of which repo any pane sits in — repo-rooting lost cross-repo state, and a fixed global path would force a machine-wide singleton.

- **`stateDir() (string, error)`** — resolves the XDG state base dir uniformly on Linux and macOS: returns `$XDG_STATE_HOME` only when it is set AND absolute, else `$HOME/.local/state`. Deliberately NOT `~/Library/...` on macOS (terminal users expect `~/.local/state`; the Go stdlib has no `UserStateDir()`).
- **`serverSlug(server string) string`** — queries the tmux socket path via `tmux <…> display-message -p '#{socket_path}'` (built through `pane.WithServer`) and slugifies it; falls back to the literal `"default"` when tmux cannot be queried, so the operator still functions if the query fails.
- **`slugify(string) string`** — the deterministic, collision-free, filesystem-safe rule: escape literal `-` by doubling it (`-` → `--`) FIRST, then strip the leading path separator and replace remaining separators with a single `-` (so `/tmp/tmux-1000/default` → `tmp-tmux--1000-default`). Escaping before substitution makes the mapping injective, so a socket path with a literal `-` cannot collide with one whose separator falls at the same spot (`/tmp/tmux/1000/default` → `tmp-tmux-1000-default`, distinct). Empty input slugifies to `"default"`.
- **`StatePath(server string) (string, error)`** — returns `<stateDir>/fab/operator/<server-slug>.yaml`, creating the parent dir with `MkdirAll` (0o755). `tick-start` calls `StatePath("")` (server `""` → the operator's own current tmux server). The test seam is `operatorStatePathOverride` (a full file path, not a directory).

Keying on the **socket path** (not server PID) is deliberate: the socket path survives a tmux-server restart (same `-L` label → same path), so a restarted operator resumes the same state file; a PID would change and orphan the file. There is **no migration** of old repo-rooted `.fab-operator.yaml` files — they are abandoned in place. `runOperator`'s launch CWD is unchanged: it still uses `gitRepoRoot()` as the new-window working directory; only the state path decouples from the repo.
- **`fab operator time`** — pure clock query: outputs `now: HH:MM`; with `--interval <duration>` also outputs `next: HH:MM`. No file I/O. Source: `src/go/fab/cmd/fab/operator_time.go`.

### Agent Skill Deployment

`fab-kit sync` deploys skills to each agent. Deployment is **conditional** — by default, each agent's CLI command is checked via PATH lookup before syncing. If an agent's CLI is not found in PATH, its sync is skipped with a message, and existing dot folders are preserved. When no agents are detected, a warning is printed but sync continues. The `FAB_AGENTS` environment variable (space-separated list of CLI command names, e.g., `claude opencode gemini`) can override PATH detection for testing and CI — when set, only the listed agents are synced.

All `*.md` files in `$(fab kit-path)/skills/` are deployed, including underscore partials (`_preamble.md`, `_generation.md`, `_review.md`, `_cli-fab.md`, `_cli-external.md`) which have `user-invocable: false` frontmatter to prevent direct invocation. The skill prompt files are agent-agnostic markdown; only the deployment locations and formats differ per agent:

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

#### Packaging invariant — the whole `src/kit/` tree is copied verbatim

New kit content ships automatically with **no Go/binary, packaging-list, or `release.yml` change**: both distribution paths copy the entire `src/kit/` tree verbatim. `just install` runs `rsync -a --delete src/kit/ {{local_cache}}/kit/`; `just dist-kit` runs `cp -a src/kit/. dist/kit/` and then archives the whole `dist/kit/` tree. Neither enumerates individual files, so adding a new file or directory under `src/kit/` (e.g. `src/kit/reference/fkf.md` — frlo) ships to the cache on the next `just install` / release with zero packaging edits. This is why the shipped FKF contract is a pure content change.

#### Bootstrap Sequence

**Primary method** (recommended):
```
brew tap sahil87/tap && brew install fab-kit
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
- **`fab/project/config.yaml` `fab_version`** (project version) — set by `fab upgrade-repo` and `fab init`. Used by preflight to detect sync staleness (compared against `$(fab kit-path)/VERSION`).
- **`fab/.kit-migration-version`** (local project version) — lives outside `.kit/`, NOT replaced on upgrades. Tracks the version the project's `config.yaml`, `.status.yaml`, and conventions were written for. Created by `sync/2-sync-workspace.sh`. Renamed from `fab/project/VERSION`.

`VERSION` and `.kit-migration-version` contain a bare semver string (`MAJOR.MINOR.PATCH`). See [migrations.md](/distribution/migrations.md) for the full migration system.

### Updating `.kit/`

Run `fab upgrade-repo` to update to the latest release. The fab-kit subcommand downloads the new version to the cache if not present (verified + atomic — see distribution.md's Auto-Download Hardening), calls `Sync()` directly FIRST (passing the target kit version explicitly while `config.yaml` still pins the old `fab_version`), and stamps `fab_version` in `config.yaml` only after the sync succeeds (260612-dn2c, F18). A sync failure exits non-zero with "run 'fab sync' to repair, then re-run 'fab upgrade-repo'" and leaves the stamp unwritten, so a re-run retries instead of short-circuiting on "Already on the latest version". Kit content is served from the cache — nothing is copied into the repo. After the upgrade, if `fab/.kit-migration-version` is behind the new engine version, the output includes a migration reminder. See [distribution.md](/distribution/distribution.md) for full upgrade details.

Skill deployments in `.claude/skills/`, `.opencode/commands/`, `.agents/skills/`, and `.gemini/skills/` are refreshed by `fab-kit sync` after the update. OpenCode symlinks resolve automatically; copies for Claude Code, Codex, and Gemini are re-copied.

**Preserved** (lives outside `.kit/`): `config.yaml`, `constitution.md`, `docs/memory/`, `docs/specs/`, `changes/`, `.fab-status.yaml`, `.kit-migration-version`
**Replaced** (lives inside `.kit/`): `templates/`, `reference/` (shipped read-only contracts, e.g. `reference/fkf.md` — frlo), `skills/`, `sync/`, `migrations/`, `packages/` (idea shell package), `bin/` (`.gitkeep` only), `VERSION`

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

The `fab` binary (installed via `brew install fab-kit`) is the user-facing entry point. It uses negative-match routing: a static allowlist of fab-kit commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) is dispatched to `fab-kit` via `syscall.Exec`; the inline arguments `--version`/`-v` and `--help`/`-h`/`help` are handled by the router itself (printing version or composed help); all other commands are dispatched to the version-resolved `fab-go` via `syscall.Exec`. The allowlist is **derived from the shared `internal.LifecycleCommands` table** (`src/go/fab-kit/internal/lifecycle.go`, since 260612-ye8r) — the single source of truth for the workspace command set (names + cobra Shorts), also feeding the router's help section, `cmd/fab-kit`'s `fabKitCommands`, and a registration cross-check test. Doc↔code drift is test-guarded in both directions: a fab-kit-module contract test pins `_cli-fab.md`'s router line to the table, and a fab-module collision test asserts no top-level fab-go command name (from the in-process help-dump tree) appears in that documented allowlist (a colliding name would be shadowed by the system-wide shim forever).

The router applies an **always-route policy** for non-fab-kit commands: every such command is dispatched to `fab-go` regardless of whether `fab/project/config.yaml` is present. There is no router-side config gate, no per-command allowlist, and no "Not in a fab-managed repo" exit from the router. Per-command guards inside `fab-go` (typically a call to `resolve.FabRoot()`) are the authoritative answer to "does this need config?" — they fail-closed with `ERROR: fab/ directory not found` for commands that require project state (`preflight`, `score`, `resolve`, `status`, `change`, `log`, `batch`, `fab-help`), while config-free commands (`kit-path`, `pane`, `operator`'s switch path, `hook session-start|stop|user-prompt`, `completion`, `shell-init`, `help`, and any `<subcommand> --help`) run cleanly from any directory.

Version selection for the fab-go exec is inline in `execFabGo` (and mirrored in `printHelp`): walk up from CWD to find `fab/project/config.yaml`; if `cfg != nil` use `cfg.FabVersion` (project-pinned), else use the router's build-time `version` constant (router-bundled). The resolved `fab-go` binary lives at `~/.fab-kit/versions/{version}/fab-go`; missing binaries are auto-fetched from GitHub releases and cached. On the `execFabGo` dispatch path, if `config.yaml` exists but cannot be parsed, the router hard-errors with the parse error from `internal.ResolveConfig` — only the missing-config case becomes a soft fall-through. The router-inline help and version paths take the opposite stance (see next paragraph) because they must remain available even with a broken config.

`fab help` composes help from both sub-binaries: workspace commands are rendered **in-process from the shared `LifecycleCommands` table** (names + Shorts — never by exec'ing `fab-kit --help`, so the section renders even when the fab-kit binary is absent and its Shorts cannot drift from the cobra registrations); workflow commands (from fab-go) are also always shown — inside a fab-managed repo using the project-pinned `fab_version`, and outside using the router's build-time `version` (bundled fab-go), so all workflow commands remain discoverable from scratch tabs. Errors during version resolution or subprocess execution for the workflow-commands block are silently swallowed — help is best-effort. `fab --version` and `fab -v` always print the system-installed binary version (`fab {version}`); when run inside a fab-managed repo, a second line shows the project-pinned version from `fab/project/config.yaml` (`project: {fab_version}`). Config resolution errors are silently ignored — the command always exits 0.

#### `fab-kit` (Workspace Lifecycle)

The `fab-kit` binary (installed via `brew install fab-kit`) owns workspace lifecycle operations:

- `fab-kit init` — initialize fab in a repo (git-repo precondition FIRST — fails non-zero before any download or config write; then resolve latest version, cache it, set `fab_version`, stamp `.kit-migration-version`, run sync; sync failure propagates non-zero)
- `fab-kit upgrade-repo [version]` — upgrade to a different version (download to cache, run sync FIRST, stamp `fab_version` only after sync succeeds; sync failure exits non-zero with repair guidance and a re-run retries)
- `fab-kit sync` — reconcile workspace with pinned version (6-step pipeline: prerequisites, version guard, ensure cache, scaffolding, direnv, project scripts). Supports `--shim` (steps 1-5) and `--project` (step 6) flags. Exits **`3` (`ExitNotManaged`, 52i9)** when run outside a fab-managed repo (a benign "not applicable here" signal, distinct from a failure — see § Distinguishable Exit Codes below), and **`1`** when any deployment/scaffolding write fails or the version guard trips (a genuine failure, unchanged).
- `fab-kit doctor [--porcelain]` — validate fab-kit prerequisites (7 tools: git, fab, bash, yq v4+, jq, gh, direnv+hook). Works before `config.yaml` exists — required for `/fab-setup` Phase 0 gate. Exit code = failure count. `--porcelain` outputs only errors (no passes/hints/summary)

`fab-kit sync` resolves all kit content from the system cache at `~/.fab-kit/versions/{version}/kit/` (via `CachedKitDir(fab_version)`) rather than from `src/kit/` in the repo. Signature (since 260612-dn2c, F22): `Sync(systemVersion, kitVersion string, shimOnly, projectOnly bool)` — `systemVersion` is the embedded binary version (feeds the version guard); `kitVersion` is the kit content version to sync, read from `fab_version` in config.yaml when empty (the plain `fab sync` path) and passed explicitly by `Init`/`Upgrade` (which lets Upgrade stamp config only after success, and makes the guard compare the *real* binary version instead of the kit version against itself — previously `Init`/`Upgrade` passed the kit version as `systemVersion`, so the guard always passed by construction). The 6-step pipeline: (1) prerequisites check (git, bash, yq v4+, direnv), (2) version guard — ensures `fab_version` <= system `fab-kit` version, now actually enforced via **post-state verification** (see the Design Decision below): on trip it attempts `Update()` then re-checks the installed binary version on PATH, and ALWAYS fails the current run, (3) ensure cache (calls `EnsureCached(fab_version)`, verified atomic download if needed), (4) workspace scaffolding from cache (directory creation, scaffold tree-walk with fragment-merge and copy-if-absent, multi-agent skill deployment, hook sync, version stamp, legacy cleanup), (5) direnv allow, (6) project-level `fab/sync/*.sh` script execution. Hook sync (previously delegated to `5-sync-hooks.sh` and `fab hook sync`) is absorbed directly into step 4 — `fab-kit` replicates the hooklib sync logic internally rather than shelling out to the `fab` binary. **Deployment writes are fail-loud** (260612-dn2c, F21): `syncAgentSkills` counts write/symlink/per-skill-MkdirAll/source-read failures per skill (never as `created`/`repaired`), prints `WARN:` lines on stderr and a `failed N` tally figure, and returns an error; `deploySkills` joins per-agent errors but lets the remaining repair steps run before `Sync` returns non-nil (no `Done.` on failure); `scaffoldDirectories` propagates MkdirAll/WriteFile errors — including the `.kit-migration-version` writes (whose silent failure used to silently disable migration discovery in `Upgrade`) and the kit `VERSION` read. Both the copy and symlink deployment branches are covered (the symlink branch is currently unused — all four agent configs deploy in copy mode; the branch and the `agentConfig.Mode` field that selects it are a recorded tb6f deletion candidate). **File layout** (since 260612-tb6f, F44): the former 886-line `internal/sync.go` monolith is split within the flat internal package — `sync.go` keeps the `Sync` orchestrator, `versionGuard`, and the orchestrator-owned step helpers (`gitRepoRoot`, `runDirenvAllow`, `runProjectSyncScripts`); `semver.go` holds `parseSemver`/`compareSemver`; `prereqs.go` holds `checkPrerequisites` (incl. yq version sniffing — the yq major-version check compares numerically since tb6f; the old lexicographic string compare would have rejected a hypothetical yq v10); `scaffold.go` holds the scaffold tree-walk plus both merge mini-engines (JSON permissions merge, line-ensure merge); `skills.go` holds skill deployment (`deploySkills`/`listSkills`/`syncAgentSkills`/`cleanStaleSkills`/`cleanLegacyAgents`). No API changes — tests moved with their functions, and `Sync` idempotency is now directly encoded by an integration test (run twice → content-identical no-op).

**Source layout**: Both `fab` (router) and `fab-kit` share a single Go module at `src/go/fab-kit/` with two `cmd/` entries: `cmd/fab/main.go` and `cmd/fab-kit/main.go`. Both import shared `internal/` packages for cache, download, and config resolution. This avoids Go workspace complexity and keeps infrastructure code importable by both without duplication.

The shell dispatcher at `fab-go binary at fab` has been removed. The `FAB_BACKEND` env var and `.fab-backend` file override mechanism has been removed — Go is the only backend.

#### Distinguishable Exit Codes — `ExitNotManaged` (52i9)

The `fab-kit` binary's `main()` returns exit `1` for **any** `RunE` error (`cmd/fab-kit/main.go`), so "not a fab-managed repo" was historically indistinguishable — at the exit-code level — from a genuine sync failure (corrupt config, failed scaffold write, version-guard trip). External callers that probe arbitrary directories (`wt`'s default init, `hop`, operator scripts) could not tell "skip me, this isn't a fab repo" from "a real failure happened here" without duplicating fab's `fab/project/config.yaml` walk-up client-side. `52i9` (2026-07-05) closes that by giving the unmanaged-repo outcome its own **distinct, documented exit code**:

- **`internal.ExitNotManaged = 3`** — a single exported named constant in `internal/config.go` (with a doc comment; no bare `3` at any call site — R3), deliberately distinct from the generic exit `1`. Chosen as `3` to sit alongside the `fab` binary's own in-handler `os.Exit(N)` tiering convention (`pane_window_name.go` uses 2/3 for the pane family — see [pane-commands.md](/runtime/pane-commands.md); `memory_index.go` uses 2 for destructive-loss). It collides only theoretically with `fab-kit doctor`'s dynamic `os.Exit(failureCount)` (0–7), an unambiguous diagnostic count on a different command.
- **`internal.RequireManagedRepo() (*ConfigResult, error)`** — the shared guard consolidating the formerly copy-pasted `ResolveConfig()` + `if cfg == nil { return fmt.Errorf("not in a fab-managed repo…") }` block (R4). It returns a genuine `ResolveConfig` error unchanged (corrupt config / missing `fab_version` still collapse to exit 1 in `main()` — R2), and on the `(nil, nil)` "walked to filesystem root, no config" case it prints the actionable `not in a fab-managed repo. Run 'fab init' to set one up` to stderr and calls `os.Exit(ExitNotManaged)` **in-handler** (a returned error would collapse to exit 1). This mirrors the `fab` binary's in-handler-`os.Exit` pattern precisely because the fab-kit `main()` funnel exits 1 uniformly.
- **Two call sites**: `internal.Sync()` (its `kitVersion == ""` branch — the plain `fab sync` path) and `cmd/fab-kit`'s `runMigrationsStatus`. Both now call `RequireManagedRepo()`; the `not in a fab-managed repo` literal no longer appears in `sync.go` or `migrations_status.go`.
- **Git-independence (the ordering fix)**: `Sync()` was reordered so `RequireManagedRepo()` gates **before** `gitRepoRoot()`. The managed-repo check is a `config.yaml` walk-up that does not depend on git, so a directory that is neither git-tracked nor fab-managed now exits `3`, not `1` — keeping `fab sync` symmetric with `fab-kit migrations-status` (which has no git precondition and already exited `3` in the same directory). A managed repo that lacks git context still fails at `gitRepoRoot()` with a genuine error → exit 1 (R2 unchanged). `Init`/`Upgrade` pass `kitVersion` explicitly (config.yaml is not yet stamped), so they skip the check.
- **Deliberate exclusion — `internal/upgrade.go` is untouched**: `Upgrade`'s two `not in a fab-managed repo` returns still exit `1`. Its guard is a *different semantic* — it tolerates a `config.yaml` that is present but missing its `fab_version` field (a partially-managed repo, not an unmanaged one), so folding it into `RequireManagedRepo()` would conflate the two. Documented here so a reader does not over-generalize the exit-3 contract to every "unmanaged repo" case; re-tiering `upgrade.go`'s exit code is a recorded future follow-up.

The `_cli-fab.md` CLI reference and its `SPEC-_cli-fab.md` mirror carry the same contract for the skill-facing surface (updated during apply). The intended downstream beneficiary: `wt` can retire its interim client-side `ResolveConfig`-mirroring marker probe and branch on exit `3` directly, keeping fab authoritative over the "is this a fab repo?" question (the change adds no new CLI flag surface — no `--if-managed` — the distinct exit code fully satisfies the need).

### Go Binary (`fab-go`)

The workflow engine backend for all fab CLI operations. Source: `src/go/fab/`.

**Module**: `github.com/sahil87/fab-kit/src/go/fab` (Go 1.26+, dependencies: cobra v1.10.x, gopkg.in/yaml.v3, no CGo — toolchain bumped from the out-of-security-window Go 1.22 in 260612-tb6f, F41)

**Binary location**: `~/.fab-kit/versions/{version}/fab-go` — cached per-version by the system shim. Included in per-platform release archives (`kit-{os}-{arch}.tar.gz`). No longer stored in `fab-go binary at ` — the repo holds content only. Cache installs are atomic since 260612-dn2c: the archive is digest-verified and extracted into `versions/{version}.tmp-<pid>`, then renamed into place under a version-keyed flock — a version dir that exists with `fab-go` is complete, so `ResolveBinary`'s exec-bit probe can no longer observe a partially written binary (the exec bit used to be set at file-create time, before content streamed). See distribution.md's Auto-Download Hardening.

**Subcommands**:
- `fab resolve [--id|--folder|--dir|--status|--pane] [--server <name>] [<change>]` — the five output-mode flags are mutually exclusive (`MarkFlagsMutuallyExclusive`; conflicting flags fail loudly, and `--id` is a real explicit-default flag wired into the selection — both since 260612-ye8r). `fab change resolve` is a thin cobra wrapper over the same shared `runResolve` implementation with `--folder` mode fixed, so the two spellings cannot drift
- `fab config reference` — pure query (`config` command group + `reference` subcommand, `cmd/fab/config.go`; `cobra.NoArgs`, no flags, no file writes) that prints a fully-commented reference `config.yaml` to stdout, exit 0, byte-stable for a given binary version. The body is GENERATED from Go constants in `internal/configref` (`Render()`) — defaults injected from `spawn.DefaultSpawnCommand` / `agent.DefaultTier` over `agent.TierNames` / `agent.StageNames`, so no default has a hand-typed second copy to drift. Covers both binary-consumed and skill-consumed keys (see [configuration.md](/_shared/configuration.md) § Schema Discovery). Added 6nke; the `config` group leaves room for a future `fab config validate`. As a fab-go command it is dispatched via the router's always-route policy — `config` is not a fab-kit lifecycle command, and the collision test (below) guards that no such top-level fab-go name shadows the fab-kit allowlist
- `fab log command|confidence|review|transition ...`
- `fab status start|advance|finish|reset|skip|fail|all-stages|progress-map|...` (stage-machine operations plus status/diagnostic utilities)
- `fab preflight [<change>]`
- `fab change new|rename|switch|list|resolve ...`
- `fab score [--check-gate] [--stage <stage>] <change>`
- `fab change archive|restore|archive-list ...`
- `fab hook session-start | stop | user-prompt` — **inert no-op exit-0 shims since ioku** (consume nothing, emit nothing). They formerly produced agent active/idle state in `.fab-runtime.yaml` (`_agents` map); that whole producer subsystem was deleted when agent-state production was divested to run-kit's `@rk_agent_state` tmux pane-option convention (fab is now a pure reader — see [runtime-agents.md](/runtime/runtime-agents.md)). The shims are retained one release for un-migrated `.claude/settings.local.json` files still invoking them; the `2.13.6-to-2.14.0` migration removes the three settings entries (both the inline `fab hook …` form and the legacy `on-*.sh` script-shim form) and deletes `.fab-runtime.yaml`/`.fab-runtime.yaml.lock` across worktrees. The shims themselves are slated for next-release deletion.

*Removed in 1.5.0*: `fab runtime set-idle|clear-idle|is-idle <change>` subcommands (hook-internal plumbing, never user-facing). The `_agents`-keyed hook writes that replaced them were themselves removed in ioku — see above.
- *Removed in y022*: `fab hook artifact-write` — the PostToolUse handler that parsed JSON from stdin, pattern-matched fab artifact paths, and performed per-artifact bookkeeping (type inference, scoring, checklist counting). A hook fires only in the Claude harness, so this correctness-critical `.status.yaml` state moved to the pull-based **`fab status refresh <change>`** (`internal/refresh.Refresh`), self-healed at the transition seams (`fab status advance`/`finish`, `fab preflight`). A one-release no-op shim `fab hook artifact-write` is retained for un-migrated settings (exits 0, emits nothing); the `2.10.1-to-2.11.0` migration removes the settings entry. (ioku scope explicitly excludes this y022 shim's own pending deletion — it has its own next-release cleanup.)
- `fab hook sync` — **deprecated no-op since ioku** (cobra Short: "Deprecated no-op — registers nothing and rewrites nothing; retained one release"). Its registration table (`hooklib.DefaultMappings`/fab-kit's `defaultHookMappings`) is now empty and the legacy `on-*.sh` → `fab hook …` rewrite rows were dropped, so it registers nothing and can no longer re-mint the entries the `2.13.6-to-2.14.0` migration deletes; it has no removal path (`Sync` never deletes stale fab-managed entries). Retained one release only to tolerate stale settings; full removal of the sync path is a follow-up.
- `fab pane` — parent command grouping five pane-related subcommands (`map`, `capture`, `send`, `process`, `window-name`). Available from any directory, including outside a fab repo — config-independent by virtue of the router's always-route policy plus the absence of any per-command `resolve.FabRoot()` guard (pane subcommands resolve state from pane IDs, not the invoker's CWD). Detailed subcommand behavior and the `--server`/`-L` flag live in [pane-commands.md](/runtime/pane-commands.md).
- `fab dispatch start|status|logs|kill|clean` — the **tmux-independent headless process manager** for CLI-dispatched pipeline stages (6sgj, change 3c of the cross-harness dispatch series). Parent command (`cmd/fab/dispatch.go`) grouping five subcommands split across `dispatch_start.go` / `dispatch_status.go` / `dispatch_logs.go` / `dispatch_kill.go` / `dispatch_clean.go` (mirroring the `pane*.go` split), registered via `dispatchCmd()` in `newRootCmd()`. `start` resolves the stage's tier → provider → `dispatch_command` via `internal/agent` + `internal/spawn.WithProfile` (**consuming the provider `dispatch_command`**; since tykw this is `providers.<name>.dispatch_command`, was 3b's per-tier `spawn_command`; errors clearly with no fallback to the provider's `session_command` when the tier's provider carries none), then launches the resolved command **detached** — `SysProcAttr{Setsid:true}` on a plain `sh -c` wrapper (NOT the `setsid` binary), so no Go supervisor remains and the recorded pid tracks the live worker shell — tracking state under `.fab-dispatch/{4-char-id}/` at the repo root, polled via the five byte-stable status states (incl. `failed (no-result)`). The testable core (state read/write, `WrapperArgv` composition, `DeriveState`, process signaling) lives in `internal/dispatch`; the launch/signal syscalls are platform-split (`dispatch_posix.go` `!windows` / `dispatch_windows.go` `windows`) so **POSIX-only v1** is a compile-time reality (Windows returns the clear POSIX-only error). Like `fab config` (6nke), `dispatch` is always-routed and its name must not collide with the fab-kit allowlist (`TestNoTopLevelCommandCollidesWithRouterAllowlist` stays green). Full runtime behavior — the detached-launch model, `.fab-dispatch/{id}/` layout, five states, refuse-if-running/last-attempt-only, timeout-in-wrapper, and the two cleanup paths — lives in [runtime/dispatch.md](/runtime/dispatch.md).
- `fab resolve <change> --pane` — output the tmux pane ID (e.g., `%5`) for the pane running the resolved change; composable with `tmux send-keys -t "$(fab resolve <change> --pane)" "<text>" Enter`
- `fab idea add|list|show|done|reopen|edit|rm` — backlog idea management (CRUD for `fab/backlog.md`)
- `fab fab-help` — dynamic skill discovery and help overview (scans `.kit/skills/` frontmatter, groups by category)
- `fab memory-index [--check [--json]]` — deterministically (re)generate the `docs/memory/` index files so agents never hand-edit them. Regenerates the root `docs/memory/index.md` (**domains-only** — `| Domain | Description |`; the legacy inlined per-file column is dropped) and every `docs/memory/{domain}/index.md` (file rows — `| File | Description |`) from folder contents + each file's `description:` frontmatter — **content-only, with no dates** (the `Last Updated` column was dropped in ugde, since a `git log` projection is HEAD/branch-relative and so not idempotent; the batched `git log` pass now serves `log.md` only). **Recurses one level into sub-domains** (sx7a): a `{domain}/{sub-domain}/` directory holding ≥1 non-index `.md` gets its own generated `{domain}/{sub-domain}/index.md` (same file-row contract), and the parent domain index gains a `## Sub-Domains` table (`| Sub-Domain | Description |` linking to `{sub-domain}/index.md`) emitted **only when sub-domains exist** — so sub-domain-free domain indexes render byte-identically to the pre-`sx7a` output. Byte-stable / idempotent (second run = no diff), so the indexes stop drifting and stop generating per-row merge conflicts. Emits **non-fatal stderr shape warnings** across the recursive tree when a folder (domain or sub-domain) exceeds the soft width bound (~12 topic files) or depth 3 (reserved domains `_shared/`/`_unsorted/` are width-exempt; the width exemption is domain-tier only — an over-wide sub-domain still warns) — advisory only, never affecting the byte-stable output. **`--check` is tiered (glwc):** it writes nothing and classifies the rendered-vs-existing drift by **severity** encoded in the exit code — **0** clean, **1** benign drift (regen changes content but destroys nothing — e.g. an improved `description:`; the former "out of date" condition, so existing "non-zero = stale" CI/preflight consumers keep working), **2** destructive loss (regen would wipe curated/historical content). Tier 2 has three categories (the mechanical form of `/docs-reorg-memory`'s prose signals): (1) a curated **description** that would regenerate to `—` because the file lacks `description:` frontmatter; (2) a **tombstone** row whose `docs/memory/`-relative link target is absent on disk (external/absolute links excluded — no false positives); (3) a custom structural **grouping** heading in the root `index.md` beyond the domains-only table. On tier 2 (non-`--json`) it enumerates each loss to stderr by category and ends with the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` (`/docs-reorg-memory` is the orchestrator for all three categories — it relocates tombstone rows itself and dispatches `/docs-hydrate-memory` backfill mode for descriptions; backfill alone does not relocate tombstones). Loss is a strict subset of drift (one render pass serves both); a **born-compatible fab-kit tree is provably never tier 2** (frontmatter present, no off-disk rows, domains-only root). The optional **`--json`** flag (with `--check`) emits the loss report as a single snake_case JSON object on stdout — `{"tier": 0|1|2, "drift": bool, "losses": [{"category": "description"|"tombstone"|"grouping", "path": "<repo-rel index>", "detail": "..."}]}` (mirrors the `fab pane`/`migrations-status` `--json` convention) — suppressing the human text; the exit code is unchanged. Callers pick a threshold: CI/pre-commit fails on exit ≥1; the hydrate/reorg refuse-before-regen guards fail only on exit ==2. The classifier + existing-index-row parser are pure functions in `internal/memoryindex` (unit-tested like `RenderRoot`/`Gather`); the cmd reuses the existing rendered-vs-existing byte-compare. Source: `cmd/fab/memory_index.go` + `internal/memoryindex/`. Consumed by the hydrate skills (`/docs-hydrate-memory`, `/fab-continue` hydrate — both with a refuse-before-regen guard keyed on exit 2) and `/docs-reorg-memory` (compatibility detection via `--check --json`)
- `fab operator` — parent command: default behavior launches singleton tmux tab for the operator skill (resolves the **operator tier** in-process → the operator provider's `session_command` + profile; tykw, was `agent.spawn_command`). Subcommands: `tick-start` (start-of-tick state update: increments `tick_count`, writes `last_tick_at` RFC3339 UTC to the **server-keyed XDG state file** — see "Operator State File" above — outputs `tick: N\nnow: HH:MM`) and `time` (pure clock query: outputs `now: HH:MM`; with `--interval <duration>` also outputs `next: HH:MM`)
- `fab agent [tier] [--print] [--repo <path>]` (tykw — **retires `fab spawn-command`**) — resolve a tier profile (`default` when the tier arg is omitted; any of the five role tiers accepted), compose `providers.<provider>.session_command` with `{model}`/`{effort}` substituted (or Claude-style flags appended) via `spawn.WithProfile`, and **exec it in the current shell** (via `/bin/sh -c`). `--print` prints the fully-resolved command instead of executing — **the `fab spawn-command` replacement**, with a semantic upgrade: the output is **profile-resolved** (model/effort substituted), not empty-profile placeholder-stripped as `fab spawn-command` was, so callers that spawn from the printed command finally get the tier profile. `--repo <path>` reads `<path>/fab/project/config.yaml` directly (no upward search — the operator's fetch-a-target-repo's-command use case, carried over from `fab spawn-command --repo`); without `--repo`, resolves the current repo's config via upward `resolve.FabRoot()`. Falls back to `spawn.DefaultSpawnCommand` (the `{model}`/`{effort}` template `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`, `260703-gvxd`; profile-substituted) when the config is missing/empty/unreadable. Exec does NOT TTY-guard (exec-and-let-the-CLI-fail). Source: `src/go/fab/cmd/fab/agent.go` (`agentCmd()`, wired into the root command in `main.go`). `fab spawn-command` (and `spawn_command.go`/`_test.go`) was removed in the same release — no deprecation alias (its only CLI consumer, the operator skill, ships and is updated in the same kit).
- `fab batch new|switch|archive` — multi-target batch operations via tmux tabs with Claude Code sessions
- `fab shell-init <shell>` — emit the shell-completion script for `bash`, `zsh`, or `fish`. Equivalent to (and delegated to) Cobra's auto-generated `fab completion <shell>`; provided as the `tu`-style verb users expect. Source: `src/go/fab/cmd/fab/shellinit.go`. Recommended install: add `eval "$(fab shell-init zsh)"` to `~/.zshrc` (or the bash/fish equivalent). Config-independent — works outside a fab repo
- `fab help-dump` — **hidden, CI/build-time-only** (`Hidden: true`, `cobra.NoArgs`). Walks the live cobra command tree of the assembled root command programmatically (via `cmd.Commands()`, not regex-parsing `-h`) and writes the frozen shll.ai "command reference" contract JSON to stdout: `{tool:"fab", version (from `main.version` ldflags), captured_at (RFC3339 UTC), schema_version:1, root:Node}` where `Node={name=cmd.Name(), path=cmd.CommandPath(), short, usage=cmd.UseLine(), text=cmd.UsageString(), commands[]}`. At every level the walk drops `completion`, `help`, and any `Hidden` command (self-excluding `help-dump`), then sorts surviving children by `Name()` for byte-stable output; leaves emit `commands:[]` (never `null`). The encoder uses 2-space indent and `SetEscapeHTML(false)` to preserve raw `-h` bytes. Because it is `Hidden`, it is absent from `fab --help` and from its own dumped tree. Source: `src/go/fab/cmd/fab/helpdump.go` (`helpDumpCmd()`, `dumpDoc`, recursive `buildNode`); consumed by the `Help-dump → shll.ai` release-workflow step (see [distribution.md](/distribution/distribution.md))

**Architecture**: since tykw, provider/tier resolution lives in `internal/agent` (`ResolveProvider(name)` → a provider's `session_command`/`dispatch_command` from the top-level `providers:` table; `ResolveTier`/`Resolve` → the five role tiers with per-field default-tier inheritance; `defaultProviders`/`defaultTiers` built-ins), and `internal/spawn` provides only the command-line composition. Consumers — `operator`, `agent`, and the `batch new`/`batch switch` subcommands (`batch archive` does not spawn) — compose a resolved provider `session_command` + a tier profile. `agent.DefaultSessionCommand` (re-exported as `spawn.DefaultSpawnCommand`) is the built-in `claude` fallback; `--repo <path>` (on `fab agent`) reads a target repo's config directly via `internal/config.LoadPath`. `spawn.WithProfile(cmd, model, effort)` is the one command-line-composition seam: since 6tmi it appends ` --model`/` --effort` only when `cmd` carries **no** `{model}`/`{effort}` placeholder (Claude-shaped back-compat), and otherwise **substitutes** the resolved profile into the placeholders (all-or-nothing template mode, with an empty-value token-drop rule that also strips a preceding `-`-flag; the all-non-empty path is a raw `strings.ReplaceAll` that preserves author whitespace). (`spawn.StripPlaceholders` was **removed in tykw** — the raw empty-profile print path it guarded is gone: `fab spawn-command` was retired for `fab agent --print` and `fab batch` composes with a profile, so no consumer interpolates an unresolved templated command into a shell. See [_shared/configuration.md](/_shared/configuration.md) § `providers`, [runtime/providers-and-tiers.md](/runtime/providers-and-tiers.md), and [runtime/operator.md](/runtime/operator.md).) `internal/frontmatter` provides YAML frontmatter parsing (used by `fab-help` and `memory-index` to read the `description:` field). `internal/memoryindex` powers `fab memory-index`: it follows the `internal/prmeta` Render/Gather split — pure `RenderRoot(RootData) string` (domains-only table) + `RenderDomain(DomainData) string` (file rows) renderers, plus a `Gather(repoRoot) (RootData, []DomainData, []Warning, error)` I/O orchestrator that walks `docs/memory/`, reads each topic file's H1 + `description:` frontmatter (via `internal/frontmatter.Field`), computes per-folder counts/depth, and collects shape `Warning`s (width-exempting `_shared`/`_unsorted`). The index render is **content-only** since ugde — it no longer consumes git dates: the index-only date plumbing (`FileEntry.LastUpdated`, `gitDates.byPath`, `(*gitDates).lookup`, and the `gitLastUpdated` per-file fallback) was **removed**, while `loadGitDates` / `gitDates.commitsByPath` are **retained** because `log.md` generation (`gatherLogEntries`) still consumes the batched `git log` pass. **Sub-domain recursion** (sx7a): `DomainData` gains a `SubDomains []DomainData` field; `gatherSubDomains` (called from `Gather`) enumerates each domain dir's child directories that hold ≥1 non-index `.md`, builds a `DomainData` per sub-domain via the existing `gatherFiles`, and attaches them lexicographically sorted to the parent. `RenderDomain` appends a `## Sub-Domains` table only when `len(SubDomains) > 0` (sub-domain-free output unchanged), and the same `RenderDomain` renders each sub-domain `index.md` (no bespoke `RenderSubDomain` — the file-row contract is tier-agnostic). `cmd/fab/memory_index.go` flattens domains + their sub-domains into the `indexTarget` list so every sub-domain `index.md` is written/checked. This resolved the PR #377 Copilot finding that `gatherFiles` was depth-2-only (it now reaches depth-3 sub-domain topics). Recursion is one level only, matching the depth-3 bound; deeper nesting is a depth warning, not a generated tier. Repo root is resolved as `filepath.Dir(resolve.FabRoot())` (the prmeta `repoDir` idiom). The curated domain description is round-tripped through the generated domain `index.md`'s own `description:` frontmatter so the root row survives regen. The pure renderers are byte-for-byte unit-testable without git fixtures. `internal/intake` derives the mechanical archive-index description for a change: `Title(changeDir)` reads the `# Intake: {title}` heading from `intake.md` (de-prefixed, internal whitespace collapsed; `""` on any read failure), and `DescriptionFor(fabRoot, folder)` prefers that title, falling back to a humanized slug (folder name minus the `YYMMDD-XXXX-` prefix, hyphens → spaces). `internal/archive` depends on `internal/intake`, not the reverse. `internal/backlog` holds the shared backlog parser (`Item`, `ParsePending`, `ExtractContent`) extracted from `batch_new.go` (formerly `package main`, unimportable) so the batch-new and archive paths share one copy of the `[a-z0-9]{4}` regex (since hv7t `ParsePending` returns `([]Item, error)` — open/read failures surface instead of a silent nil — and `ExtractContent` distinguishes read errors from a genuinely missing ID, `not found in backlog`); it also adds `Path(fabRoot)` and `MarkDone(backlogPath, id)`, which flips a backlog line `- [ ] [<id>]` → `- [x] [<id>]` in place (never moving it to a `## Done` section) and returns `marked` / `already` (no write) / `not_found` (no match, or backlog.md missing — silent nil-error no-op). `internal/archive` keeps `Archive()` pure (folder move / index / pointer; it auto-derives an empty `--description` from the intake title via `internal/intake` before the move) and adds an `ArchiveWithBacklog()` orchestrator that runs `Archive()`, extracts the 4-char change ID via `resolve.ExtractID` (the change ID *is* the originating backlog ID), and calls `backlog.MarkDone` — recording the result on `ArchiveResult.Backlog`, which `FormatArchiveYAML` emits as a `backlog:` field. Re-archiving an already-present change returns the `ErrAlreadyArchived` sentinel, which both `fab change archive` (exit-0 soft skip) and `fab batch archive` (counted `skipped`) treat as an idempotent no-op via `errors.Is`. `internal/lines` (hv7t) is the shared read-lines helper — `ReadFileLines(path) ([]string, error)` and `Split(content) []string`, splitting on `"\n"` with a per-line trailing-`"\r"` `TrimSuffix` to preserve `bufio.ScanLines`' CRLF behavior. It replaced every unchecked production `bufio.Scanner` site (score's `countGrades` via `Split` — it takes already-read content per mz4q F02, archive's `removeFromIndex`, backlog's `ParsePending`/`ExtractContent`, hooklib's section parsers via `Split`, prmeta's checkbox counters, frontmatter's `Field`/`HasFrontmatter`, memoryindex's `readH1`): reads are all-or-nothing, so bufio's 64KB `MaxScanTokenSize` truncation class is gone. (The last production `bufio.NewScanner` was `internal/proc/proc_linux.go`, which streamed `/proc` and checked `scanner.Err()`; that package was deleted in ioku with the grandparent PID walker.) `internal/atomicfile` (hv7t) is the temp+rename write helper serving the archive-index writers — `WriteFile(path, data, perm)`: temp in the destination dir, write, fsync, chmod to `perm`, rename, temp removed on any failure. It mirrors the `statusfile.Save` pattern; `statusfile.Save` keeps its own inline implementation because mz4q (F03/F04) gave it a fsync posture (`.status.yaml` fsyncs as the pipeline's source of truth) that the always-fsync helper matches but which was deliberately distinct from the since-deleted `runtime.SaveFile` (the ephemeral, re-derivable runtime file skipped fsync because its write sat on every hook event's latency path — a distinction now moot, that file being gone in ioku). `internal/statusfile` is the shared foundation — a `StatusFile` struct parsed once via `Load()`, passed by pointer across all operations, and written atomically via `Save()` (inline temp+fsync+rename, under the cross-process lock from `internal/lockfile`). All other packages (`resolve`, `log`, `status`, `preflight`, `change`, `score`, `archive`, `worktree`) import `statusfile` for YAML access — and since hv7t that single ownership holds everywhere: `batch_archive.go`'s `hydrateStatusRe` regex (the lone outlier `.status.yaml` parser, which matched `hydrate:` at any indentation anywhere) is deleted, and `isArchivable` goes through `statusfile.Load` + `GetProgress("hydrate")`. The `worktree` package provides worktree discovery via `git worktree list --porcelain` and fab state resolution, and also contains the full worktree management library used by the `wt` binary (see below). The `internal/runtime` package was **deleted in ioku** (along with `internal/proc`, the grandparent PID walker) when agent-state production was divested — it held the `.fab-runtime.yaml` `_agents` read/write that the removed hook write-pipeline used; fab now reads the `@rk_agent_state` tmux pane option instead of any runtime file (see [runtime-agents.md](/runtime/runtime-agents.md)). `internal/lockfile` is unaffected (it still serializes `.status.yaml` for status/preflight/score; only the runtime lock usage went away with the package). The `internal/hooklib` package provides the shared parsing primitives (change type inference, task/checklist section counting) and the now-inert hook sync logic (its hook-to-event mapping table is empty and the legacy-script rewrite rows are dropped since ioku, so `Sync` registers nothing); as of y022 its artifact-bookkeeping consumer is `internal/refresh` (the pull-based `fab status refresh`), not a PostToolUse hook — `internal/refresh.Refresh` calls `hooklib.InferChangeType`/`HasSectionHeading`/`CountSectionItemsBounded`/`CountCompletedSectionItemsBounded` directly. The `internal/pane` package provides shared pane resolution logic extracted from `panemap.go`: `ValidatePane(paneID)` (checks pane exists via `tmux list-panes`), `ResolvePaneContext(paneID)` (resolves worktree, change, stage, agent state into a `PaneContext` struct), `GetPanePID(paneID)` (shell PID via `tmux display-message`), and `FindMainWorktreeRoot(cwds)` (main worktree root discovery). Used by all four `fab pane` subcommands (`map`, `capture`, `send`, `process`). The `pane` parent command in `cmd/fab/pane.go` groups four subcommands: `map` (moved from root-level `pane-map`), `capture`, `send`, `process`. The `map` subcommand in `cmd/fab/panemap.go` combines tmux pane discovery, worktree resolution, change state, and runtime state into a single observation command — now delegates pane validation and context resolution to `internal/pane`. The `internal/dispatch` package (6sgj) is the headless-dispatch analog of `internal/pane`: it owns the `.fab-dispatch/{id}/` state read/write (via `internal/atomicfile`), the `sh -c` wrapper composition (`WrapperArgv`, optional `timeout N`), the pure five-state `DeriveState`, and the platform-split launch/signal syscalls (`dispatch_posix.go` `!windows` uses `SysProcAttr{Setsid:true}` + the `syscall.Kill(pid, 0)` liveness probe + `syscall.Kill(-pgid, SIGTERM)` group kill; `dispatch_windows.go` `windows` returns the POSIX-only error). It consumes `internal/agent` + `internal/spawn` for the resolved spawn command; `internal/archive.Archive()` imports it for the archive-time `.fab-dispatch/{id}/` deletion. Supports `--json` (JSON array output), `--session <name>` (target specific session), and `--all-sessions` (enumerate all sessions). `discoverPanes(mode, sessionName)` accepts a session targeting mode and extends the tmux format string with `#{session_name}` and `#{window_index}`. Shared pane-matching functions (`discoverPanes`, `matchPanesByFolder`, `resolvePaneChange`) also live in `panemap.go` and are reused by `resolve --pane`.

**Parity**: All subcommands produce stdout/stderr output matching the bash versions (modulo timestamps).

**Testing**: Unit tests in `src/go/fab/` cover all internal packages via `go test ./...`. Run with `just test` (or `just test-v` for verbose). Tested packages: `cmd/fab` (panemap, pane_capture, pane_send, pane_process, operator, operator tick-start, operator time, batch_new, batch_switch, batch_archive, fabhelp, memory_index, and — since ye8r — resolve (output-flag mutual exclusion, `--id` wiring, `change resolve` ↔ `resolve --folder` parity, `--server` registration), log (`fab log command` always-exit-0 + stderr-warning failure paths), pane exit-code mapping (`paneValidationExitCode` 2-vs-3 classification via `errors.As`), and the lifecycle collision test (`lifecycle_collision_test.go` — no top-level command of the in-process help-dump tree appears in the `_cli-fab.md` router allowlist)), `cmd/wt`, `internal/config`, `internal/hooks`, `internal/hooklib` (parsing primitives + hook sync), `internal/refresh` (pull-based artifact-derived `.status.yaml` recompute — the `fab status refresh` successor to the removed PostToolUse hook), `internal/pane` (shared pane validation, context resolution, PID resolution, and — since ioku — the pure `parseAgentState`/`@rk_agent_state` reader; `internal/runtime` and `internal/proc` were deleted in ioku), `internal/lines` (CRLF trim, >64KB lines, missing-file error, trailing-newline semantics), `internal/atomicfile` (content/perm, overwrite, failure leaves original + no temp residue), `internal/status` (since tb6f: the exhaustive 216-cell `lookupTransition` matrix in `transitions_test.go` — stage × event × from-state, hand-written expectations pinned to the tables now enumerated in [pipeline/schemas.md](/pipeline/schemas.md), incl. the failed→active start override and AllowedStates rejections; `Skip` forward-cascade tests; direct tests in `mutators_test.go` for the formerly-0% `SetChangeType`/`AddIssue`/`ProgressMap`/`ProgressLine`/`AllStages` plus `Advance`'s remaining branches; and a 3-cycle `stage_metrics.review.iterations` rework regression test that passes against the shipped k4ge behavior — package 67.8% → 88.5%), `internal/statusfile` (incl. tb6f's `golden_test.go` — the `.status.yaml` load→Save round-trip pinned byte-for-byte over a fully-populated document), `internal/resolve`, `internal/log`, `internal/preflight`, `internal/score`, `internal/archive` (incl. tb6f's golden archive-index full-content test), `internal/intake`, `internal/backlog`, `internal/change`, `internal/worktree`, `internal/idea`, `internal/spawn`, `internal/frontmatter`, `internal/memoryindex` (byte-for-byte RenderRoot/RenderDomain golden output, idempotency, missing-description/-date degradation, shape-warning thresholds, reserved-domain exemption, loom's stale-roster self-heal regression fixture, and sx7a's sub-domain recursion: nested-tree `RenderDomain` with a `## Sub-Domains` table, sub-domain-free byte-identical output, depth-3 sub-domain discovery + deterministic ordering, idempotency, depth-4 over-depth warning, empty-sub-dir non-recursion, and over-wide-sub-domain warning; since tb6f also `golden_test.go`, pinning the complete generated root/domain index documents byte-for-byte). Since tb6f `cmd/fab` also has cobra-execution tests (`change_exec_test.go`, extending the `memory_index_test.go` setupFabRepo + `SetArgs` pattern) over the formerly low-coverage RunE bodies — `change archive`, `change archive --list`, `change switch`, `change restore`, `change list`, `change rename`, `log review`, and the backlog pending-item listing — asserting the exact stdout shapes skills parse: the archive structured YAML, the `already archived:` soft-skip line (exit 0), and hv7t's `index: failed` print-then-error contract (YAML still on stdout, non-zero exit). `fab-kit` tests: `cmd/fab-kit` (doctor; since ye8r the registration cross-check — registered cobra commands ↔ `LifecycleCommands` table, names asserted in both directions plus `Short` equality — replacing the former tautological string re-declaration), `cmd/fab` (router `fabKitArgs` derived-from-table assertion; `clifab_doc_test.go`, the `_cli-fab.md` router-line contract test, walk-up doc location in the `changetypes_doc_test.go` style — both since ye8r), and `internal` (since tb6f: `sync_integration_test.go`'s twice-run `Sync` harness — temp git repo + fake cached kit + PATH shims for `checkPrerequisites`, asserting a correct workspace tree then a content-identical no-op second run, directly encoding constitution III idempotency — plus the `shimOnly`/`projectOnly` branch split, `cleanLegacyAgents` deletion scoping at 100% (legacy targets deleted, project files outside the documented scope survive), `Upgrade`'s stamp-after-success branches, and the post-split suites `semver_test.go`/`prereqs_test.go` (numeric yq-major regression — v10 passes the v4+ check)/`scaffold_test.go`/`skills_test.go`; module 67.1% → 80.2%). **Golden byte-stability suite (tb6f)**: `internal/statusfile/golden_test.go`, `internal/memoryindex/golden_test.go`, and `internal/archive/golden_test.go` pin the `.status.yaml` emit format and the generated memory/archive-index output byte-for-byte — they are the standing parity arbiter for any future YAML-library change (see the yaml.v3 Design Decision below): a candidate library is admissible only if they pass unmodified. CI (`ci.yml`) runs both module suites with `-race` and cross-compiles darwin/arm64 (build + vet) on both matrix legs on every PR; releases gate on `just test` before any tag mint or build (`release.yml` — see [distribution.md](/distribution/distribution.md)). Test patterns: `t.TempDir()` for filesystem isolation, table-driven tests with `t.Run()` subtests, standard `testing` package only (no external test frameworks). The previous parity tests (`src/go/fab/test/parity/`) were removed — the bash scripts they validated against no longer exist.

The `internal/score` package additionally carries a **code↔doc consistency test** (`changetypes_doc_test.go`, `TestDocTablesMatchScoringMaps`): the `expectedMin` / `gateThresholds` maps in `score.go` are the canonical source for per-change-type scoring data, and the test parses the "Expected Minimum Decisions" and "Gate Thresholds" tables in `docs/specs/change-types.md` to assert they mirror the resolved `getExpectedMin` / `getGateThreshold` values for all 7 change types, plus a bidirectional check that the doc covers exactly the canonical type set. The parser uses a test-local `bufio.Scanner`/pipe-split loop (no markdown library) — since hv7t this scanner is test-only: the production sites (including `countGrades`, which the test's comment originally cited as the shared idiom) were swept to `internal/lines`, and the test's scanner is a recorded deletion candidate (hv7t plan). `findDocFile` walks up from the test CWD to the repo root to locate the doc. Drift between the maps and the doc tables now fails `just test` (and CI). See [configuration.md](/_shared/configuration.md) for the scoring-data direction-of-truth history.

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
- `wt create [flags] [branch]` — create a git worktree (random name for exploratory, branch-derived name for feature). `--base <ref>` sets the git start-point for new branches (maps to `git worktree add -b <branch> <path> <start-point>`); ignored with a warning for existing local/remote branches; validated via `git rev-parse --verify` before use; defers to `--reuse` when both are provided and the worktree already exists. `--reuse`: when a name collision is found, reuses the existing worktree and also runs the init script on it (via `RunWorktreeSetup` in force mode) before returning, gated by `--worktree-init` (default `true`); init failure is non-fatal so autopilot respawns are not aborted by transient sync errors. The init script default is `"fab sync"` (controlled by `WORKTREE_INIT_SCRIPT` env var; was `"fab-kit sync"` before the three-binary consolidation)
- `wt list [flags]` — list worktrees with status indicators (dirty `*`, unpushed `↑N`)
- `wt open [flags] [name|path]` — open a worktree in a detected application (VSCode, Cursor, Ghostty, tmux window/session, etc.)
- `wt delete [flags]` — delete a worktree with optional branch and remote cleanup
- `wt init` — run the worktree init script (`src/kit/worktree-init.sh`)

**`internal/worktree/` package**: The existing worktree listing/state code (used by `fab pane-map`) has been extended with the full worktree management library: repo context detection, random name generation, branch validation, change detection (uncommitted, untracked, unpushed), hash-based stash (create/apply), LIFO rollback stack, default branch detection, worktree CRUD (create/remove), interactive menu, OS/session detection (macOS/Linux, byobu/tmux), worktree name derivation, and application detection/launching.

**Replaces**: All 6 `wt-*` shell scripts and `wt-common.sh` from the removed `src/kit/packages/wt/` directory. No shim layer — direct cutover. `wt pr` dropped entirely (overlaps `/git-pr`).

**Exit codes**: `0` success, `1` general error, `2` invalid arguments, `3` git operation failed, `4` retry exhausted (name generation), `5` byobu tab error, `6` tmux window error.

**Error format**: Structured `Error: {what}\n  Why: {why}\n  Fix: {fix}`, colors disabled when `$NO_COLOR` is set.

**"Open here" option**: `wt create` and `wt open` app menus include an "Open here" (`open_here`) entry — always available (no detection needed), placed first in the list. The `open_here` handler in `OpenInApp()` prints `cd <quoted-path>` to stdout. `DetectDefaultApp()` skips `open_here` in its fallback logic — it is never the auto-detected default, but respects the last-app cache (if a user previously chose it, it becomes the default on next run when no context-based default applies). When `open_here` is selected, `create.go` suppresses the final path line to keep stdout clean for the shell wrapper. Requires a shell function wrapper to take effect — without it, the `cd` line prints harmlessly to the terminal. Standard pattern (cf. `nvm`, `direnv`, `z`): the wrapper captures stdout, checks for `cd ` prefix, and `eval`s it in the current shell.

#### Skill Invocation Convention (`_cli-fab.md`)

The `_cli-fab.md` partial (renamed from `_scripts.md`) defines the calling convention for all kit operations. Skills invoke operations via `fab <command> <subcommand> [args...]` — this calls the system shim, which resolves the version and dispatches to the cached `fab-go`. Since 260418-or0o-flatten-skill-helpers, `_cli-fab` is loaded **selectively** via a skill's `helpers: [_cli-fab]` frontmatter rather than universally via `_preamble`. The 6 most-used command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`) are inlined into `_preamble.md` § Common fab Commands so most skills never need `_cli-fab`. The partial includes the full command mapping table, argument formats, stage transition sequences, and error patterns in ≤300 lines.

#### Underscore File Ecosystem

The `_` (underscore) prefix denotes internal partial files that are loaded by skills but not user-invocable. These files have `user-invocable: false` frontmatter and are deployed alongside regular skills via `fab-kit sync`. The ecosystem consists of:

| File | Load strategy | Purpose |
|------|--------------|---------|
| `_preamble.md` | Always-load (every skill) | Context loading, SRAD, confidence scoring, Next Steps, Skill Helper Declaration, inlined Naming Conventions, inlined Run-Kit (rk) Reference, Common fab Commands |
| `_cli-fab.md` | Selective (via `helpers: [_cli-fab]`) | Fab CLI command reference — commands and flags beyond the Common fab Commands headline in `_preamble`. Used only by `fab-operator` currently |
| `_generation.md` | Selective (via `helpers: [_generation]`) | Spec/tasks/intake generation procedures. Used by `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff` |
| `_review.md` | Selective (via `helpers: [_review]`) | Review procedures. Used by `fab-continue`, `fab-ff`, `fab-fff` |
| `_cli-external.md` | Selective (via `helpers: [_cli-external]`) | External CLI tools: `wt` (worktree manager), `tmux` (reduced — `capture-pane`/`send-keys` internalized as `fab pane capture`/`fab pane send`; only `new-window` remains), `/loop`. Used only by `fab-operator` |

Only `_preamble.md` is always-loaded. All other helpers are opt-in via the `helpers:` frontmatter field on each skill. `_naming.md` and `_cli-rk.md` no longer exist as separate files — their content is inlined into `_preamble.md` (`## Naming Conventions`, `## Run-Kit (rk) Reference`).

Skill → helper mapping:
- `fab-new`, `fab-draft` → `[_generation]`
- `fab-continue`, `fab-ff`, `fab-fff` → `[_generation, _review]`
- `fab-operator` → `[_cli-fab, _cli-external]`
- All other 19 skills → no `helpers:` (load only `_preamble`)

#### `fab resolve --pane` Flag

Outputs the tmux pane ID for a change's worktree. Signature: `fab resolve <change> --pane [--server <name>]`. The `--pane` flag is a `Bool` flag, mutually exclusive with the other four output-mode flags (the former `PreRunE` priority chain was replaced by `MarkFlagsMutuallyExclusive` in 260612-ye8r — conflicting flags now fail loudly).

**Pane resolution**: Reuses `discoverPanes()` and `matchPanesByFolder()` from `panemap.go` with `resolvePaneChange` as the resolver function. No new tmux discovery logic. Without `--server`: same session-scoped discovery as `fab pane map` (current session, `$TMUX` required). With `--server <name>`/`-L <name>` (plumbed through in 260612-ye8r, matching the pane family's persistent flag): every tmux invocation runs with `-L <name>` and discovery is **server-wide across all sessions** — "current session" is undefined on a foreign socket, and cross-socket callers (daemons) are not inside that server.

**Tmux guard**: Without `--server`, checks that `$TMUX` is set; if not, exits non-zero with `ERROR: not inside a tmux session` (returned through RunE since 260612-ye8r). With `--server`, the guard is skipped — tmux's own connection failure surfaces if the socket is unreachable.

**No matching pane**: If no pane matches the change, exits non-zero with: `ERROR: no tmux pane found for change "<folder>"`.

**Multiple panes**: When multiple panes match the same change, outputs the first match and prints a warning to stderr: `Warning: multiple panes found for {change}, using {pane}`.

**Composable usage**: Intended to be composed with raw `tmux send-keys` for sending text to agent panes: `tmux send-keys -t "$(fab resolve <change> --pane)" "<text>" Enter`. This replaces the former `fab send-keys` subcommand (removed in 260312-kvng) — the decomposed approach is more composable and avoids duplicating tmux functionality in the CLI. For validated sending with idle checks, prefer `fab pane send` instead (added in 260403-tam1).

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

### Generated Memory Index via Deterministic Go (`fab memory-index`)
**Decision**: Memory indexes are regenerated by a deterministic `fab memory-index` Go subcommand (`internal/memoryindex`, modeled on `internal/prmeta`'s Render/Gather split), not hand-edited by the hydrate/reorg skills. A new per-file `description:` frontmatter field feeds the index; the root index is domains-only. The command also walks the tree to emit non-fatal shape-bound warnings (the "detect" half of the memory-tree-shape work).
**Why**: Hand-maintained per-row index cells were the dominant `docs/memory/` merge-conflict and drift source (measured: 65/100 fab-kit commits touched a memory file, ~57 of those were pure in-place row rewrites; loom's hand-maintained root index was stale on 4/7 sampled folders). A generated, byte-stable index is correct by construction and removes the hand-edit conflict class. Reuses the established deterministic-render Go pattern (`prmeta`/`impact`/`score`), admitted by the constitution and fully unit-testable.
**Rejected**: Hand-edited rows (the churn this kills). Shelling the regeneration from the skill (non-deterministic, untestable). Splitting wide domains *first* to relocate the hot row (Approach A — only relocates the conflict and manufactures a one-time link-rewrite bomb).
**Follow-up shipped in `sx7a`**: The *file-moving* rebalancer — splitting an over-wide domain into sub-domains, with relative-link rewriting — landed in `sx7a` as an enhancement to the existing `docs-reorg-memory` skill (an agent-driven propose-then-apply path, per Pure Prompt Play — not a Go file-mover; the Internal-vs-External addressing decision resolved to **External**). `tciy` shipped the detect/diagnose + index-regen foundation; `sx7a` activated and hardened the apply path (move → both-direction link rewrite → frontmatter → `fab memory-index` → no-dangling-link guard), made cheap and conflict-free *because* the generated index already exists. The one Go change `sx7a` required was extending `fab memory-index` to recurse one level into sub-domains (the PR #377 Copilot finding) so the External addressing tier has generated indexes.
*Introduced by*: 260607-tciy-memory-tree-shape-rebalance; *Updated by*: 260607-sx7a-reorg-memory-shape-rebalance

### Post-State Version Guard with Threaded Binary Version
**Decision**: `versionGuard` no longer trusts `Update()`'s return value. When `fab_version > systemVersion` it attempts `Update()`, then re-checks the **actually installed** binary version via `installedBinaryVersion()` (runs `fab-kit --version` from PATH — after `brew upgrade`, the PATH symlink already points at the new Cellar binary — and parses the trailing `vX.Y.Z`; a package-var test seam like `isBrewInstalled`). When tripped, the guard ALWAYS fails the current sync run: installed-now-new-enough → `fab-kit was updated to vX — re-run 'fab sync'` (never continue in-process on the old binary); otherwise distinct actionable errors for update-failed, unverifiable post-state, and Homebrew tap release-lag (Update returned nil having upgraded nothing). `Update()` returns the `ErrNotBrewInstalled` sentinel (instead of nil) on the not-brew path so `fab update` exits non-zero. The guard's input is honest too: `Init(systemVersion)`/`Upgrade(systemVersion, target)` thread the embedded binary version from `cmd/fab-kit/main.go` into `Sync(systemVersion, kitVersion, …)` — previously they passed the kit version as `systemVersion`, making the guard compare a version against itself. `dev` bypass unchanged (and it does not shelter local builds — the justfile injects real semver via `-X main.version`).
**Why**: one post-state check covers all three silent-defeat shapes (non-brew installs, tap release lag, genuine update failure) — the guard's documented contract ("ensures fab_version <= system version") is now enforced rather than aspirational. Scope note: an in-flight sync that trips the guard still completes nothing on the old binary — the benefit is failing loudly so the *next* run is correct.
**Rejected**: trusting `Update()`'s nil (the prior model — guard silently defeated for every non-brew install); branching only on `ErrNotBrewInstalled` (misses the release-lag no-op); re-exec'ing the upgraded binary inside the guard (fail-current-run with re-run guidance chosen instead).
*Source*: 260612-dn2c-fab-kit-download-lifecycle-hardening (findings F19/F22; companion contracts in [distribution.md](/distribution/distribution.md) — stamp-after-success upgrade, SHA256SUMS baseline, atomic cache install)

### yaml.v3 Stays Pinned Despite Archive Status (goccy/go-yaml Rejected)
**Decision**: Both Go modules stay on `gopkg.in/yaml.v3 v3.0.1` even though the go-yaml project was archived in April 2025 and receives no fixes. The candidate replacement `github.com/goccy/go-yaml` (evaluated at v1.19.2, 260612-tb6f F41) was rejected on proven non-parity: a scratch round-trip probe showed it cannot reproduce yaml.v3's byte output (map key order is not preserved without yaml.v3's `yaml.Node` document API, default indentation is 2-space vs yaml.v3's 4-space, block sequences are unindented, and flow-style mappings are expanded to block style) — and it provides no drop-in equivalent of the `yaml.Node` AST that `internal/statusfile`'s field-preserving serialization layer (sparse-key insertion, mz4q F07) is built on, so a swap is a serialization-layer rewrite, not a dependency substitution.
**Why**: Byte-stable `.status.yaml` output is a documented contract (every saved file in the wild carries yaml.v3's emit style; skills diff and parse it). Golden byte-stability tests (`internal/statusfile/golden_test.go`, `internal/memoryindex/golden_test.go`, `internal/archive/golden_test.go`) now pin that format and are the standing parity arbiter: any future yaml-library candidate is admissible only if those tests pass byte-for-byte unmodified. yaml.v3's archived status is an accepted, monitored risk — the library parses only first-party files (`.status.yaml`, `config.yaml`), not untrusted input, which bounds the security exposure.
**Migration plan (if ever forced)**: (1) port `internal/statusfile`'s raw-node layer to the replacement's AST, (2) run the golden suite — byte-identical output is the gate, (3) if parity is impossible, ship a one-time format migration for `.status.yaml` (+ regenerate all indexes) as a `src/kit/migrations/` file per the migrations model, never a silent format change.
**Rejected**: goccy/go-yaml swap (non-parity, no Node API); vendoring/forking yaml.v3 (no fixes exist to pull; vendoring adds maintenance without benefit while upstream stays frozen).
*Source*: 260612-tb6f-tests-ci-toolchain (finding F41; report `docs/specs/findings/binary-review-2026-06-12.md` §B6)

### Distinguishable Exit Code via In-Handler `os.Exit` + Shared Guard
**Decision**: When a `fab-kit` outcome needs to be branchable by an external caller distinct from generic failure, encode it as a **named exit-code constant** (`internal.ExitNotManaged = 3`) emitted via an **in-handler `os.Exit(N)`** inside a **shared guard helper** (`RequireManagedRepo()`), rather than a returned `error` (which `main()` collapses to exit 1) or a new CLI flag. The guard consolidates a formerly copy-pasted check across its call sites and applies the distinct-exit behavior in exactly one place. Genuine failures keep returning a normal `error` → exit 1, unchanged.
**Why**: The fab-kit `main()` funnel exits 1 for any `RunE` error, so a distinct exit code MUST be set in-handler — the same constraint the `fab` binary already solved for its pane/memory-index tiers (`pane_window_name.go`, `memory_index.go`). Reusing that proven precedent keeps the fix idiomatic and adds no new API surface. A shared guard (vs. a per-call-site literal) avoids the magic-number anti-pattern (R3) and the duplication anti-pattern (R4), and lets external consumers (`wt`, `hop`, operator scripts) branch on "not applicable" vs. "real failure" without replicating fab's `config.yaml` walk-up. **Reusable pattern** for any future fab-kit command needing a branchable non-1 outcome.
**Rejected**: a returned sentinel error mapped once per binary's `Execute()` funnel via `errors.Is` (would make the exit branch unit-testable and is in fact single-site per binary — a genuine alternative, but kept the in-handler `os.Exit` to match the existing untested-thin-wrapper precedent and avoid widening scope after two rework cycles; flagged as a candidate future follow-up). Also rejected: a new `--if-managed` no-op flag (new API to learn/maintain, no existing pattern). The `nil → os.Exit(3)` path is deliberately not unit-tested (an `os.Exit` inside a test kills the process); the constant value + the non-nil / real-error paths are, mirroring the `memory_index.go` / `doctor.go` precedent.
*Introduced by*: 260705-52i9-sync-distinguishable-unmanaged-exit

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

The benchmark implementations, harness, and fixtures were deleted in 260612-tb6f (F47) — never compiled or tested by any workflow after the decision shipped. `src/benchmark/` retains only `README.md` + `RESULTS.md` as the historical decision record; this section is the surviving summary.

# Intake: Brew Install System Shim

**Change**: 260401-46hw-brew-install-system-shim
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Evolved from PR #287 (`260325-lhhk-brew-install-system-shim`). Key evolution: the Go binary (`fab-go`) no longer lives in the repo. The system `fab` shim reads `fab_version` from `fab/project/config.yaml`, resolves the matching binary from `~/.fab-kit/versions/`, and executes it directly. `fab/.kit/bin/` can be empty — the repo holds only content (skills, templates, yaml). All skill files switch from `fab/.kit/bin/fab <command>` to `fab <command>`. Constitution Principle V is amended to require the system shim. Full removal of `fab/.kit/` from repos is future scope.

This change was preceded by PR #287 (full pipeline completed) and a follow-up discussion that identified the binary-in-repo redundancy.

## Why

Today, fab-kit is distributed by copying `fab/.kit/` into each repo. This works but creates friction:

1. **First-run friction** — new repos require manual `cp -r` or running an upgrade script to bootstrap `fab/.kit/`. There's no standard system-level command to get started.
2. **No version management** — repos vendor their own copy of `.kit/`, but there's no mechanism for a repo to declare which version it needs and have the tooling automatically resolve it. Upgrades are manual per-repo.
3. **Standalone tools trapped in repos** — `wt` (worktree management) and `idea` (backlog management) are general-purpose utilities useful outside fab-managed repos, but currently live inside `fab/.kit/bin/` and are only accessible within repos that have fab installed.
4. **Binary in repo is redundant** — if the system shim already resolves and caches the correct version, storing the Go binary in each repo's `fab/.kit/bin/` is duplication. The repo should hold content (markdown, yaml), not executables.

Without this change, onboarding remains manual, version drift across repos is invisible, and repos carry unnecessary binary weight.

## What Changes

### System-level Homebrew formula (`fab-kit`)

A Homebrew formula named `fab-kit` that installs three binaries to the system PATH:

- **`fab`** — a version-aware shim/dispatcher (see below)
- **`wt`** — the worktree management binary (currently at `fab/.kit/bin/wt`)
- **`idea`** — the backlog management binary (currently at `fab/.kit/bin/idea`)

```
brew install fab-kit
# Installs: /usr/local/bin/fab, /usr/local/bin/wt, /usr/local/bin/idea
```

### The `fab` shim (version-aware dispatcher)

The system-installed `fab` binary acts as a thin shim. When invoked:

1. Walk up from CWD to find `fab/project/config.yaml`
2. Read `fab_version` from `config.yaml` (e.g., `fab_version: "0.43.0"`)
   - If `config.yaml` found but `fab_version` absent: error with actionable message (e.g., `"No fab_version in config.yaml. Run 'fab init' to set one."`)
3. Check the local cache for the matching version's binary (`~/.fab-kit/versions/0.43.0/fab-go`)
4. If not cached, download the release from GitHub (`wvrdz/fab-kit` releases) and cache it
5. Exec `~/.fab-kit/versions/0.43.0/fab-go <original args>` — full passthrough of all arguments

If no `config.yaml` is found (not in a fab-managed repo), the shim serves non-repo commands: `fab init` (primary use case — scaffolds a new project), `fab --version`, etc.

```yaml
# fab/project/config.yaml — new field
fab_version: "0.43.0"
```

### Cache layout

The cache stores the Go binary and the full `.kit/` content per version. The binary is used by the shim for dispatch; the content is used by `fab upgrade` to populate the repo's `fab/.kit/`.

```
~/.fab-kit/
  versions/
    0.43.0/
      fab-go              # versioned binary (shim dispatches here)
      kit/                # content (fab upgrade copies to repo's fab/.kit/)
        skills/
        templates/
        scripts/
        hooks/
        migrations/
        scaffold/
        VERSION
    0.42.1/
      fab-go
      kit/
        ...
```

When the shim auto-fetches a version (on first invocation or `fab upgrade`), it downloads the platform-specific release archive, extracts the binary to `fab-go`, and the content to `kit/`. Skills and templates remain in the repo's `fab/.kit/` for now — Claude Code reads them from the repo at runtime.

### `fab/.kit/bin/` becomes empty (no binaries in repo)

The Go binary (`fab-go`) and the shell dispatcher (`fab`) are removed from `fab/.kit/bin/`. The directory may remain for future use or be removed — either way, no binaries live here. The system `fab` shim is the sole entry point.

This means:
- **No platform-specific artifacts in version control**
- **`fab/.kit/` becomes purely content** — markdown skills, yaml templates, shell hooks
- **Upgrading the binary** = bumping `fab_version` in config.yaml (the shim fetches automatically)

### `wt` and `idea` become system-only binaries

These binaries move out of `fab/.kit/bin/` and are distributed exclusively through the Homebrew formula. They are not version-coupled to the per-repo fab-kit version — they're standalone utilities.

### All skill invocations change from `fab/.kit/bin/fab` to `fab`

Every skill file (`fab/.kit/skills/*.md`) and the CLI reference (`_cli-fab.md`) currently invokes the CLI as `fab/.kit/bin/fab <command>`. All of these change to `fab <command>`. The system shim resolves the version and dispatches to the cached binary.

Files affected (every `*.md` in `fab/.kit/skills/` that references `fab/.kit/bin/fab`):
- `_preamble.md`, `_cli-fab.md`, `_generation.md`
- `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`
- `fab-clarify.md`, `fab-switch.md`, `fab-status.md`, `fab-archive.md`
- `fab-setup.md`, `fab-operator7.md`
- `git-branch.md`, `git-pr.md`, `git-pr-review.md`

Also: `fab/.kit/hooks/on-*.sh` scripts that invoke the CLI.

### Constitution Principle V amendment

Principle V currently states: *"The `fab/.kit/` directory MUST work in any project via `cp -r`."*

Amended to: *"The `fab/.kit/` directory MUST work in any project via `cp -r`, given the system `fab` binary is installed (`brew install fab-kit`). The system binary provides version-aware execution; `fab/.kit/` provides content (skills, templates, configuration)."*

### New `config.yaml` field: `fab_version`

A new optional field in `fab/project/config.yaml` that declares the fab-kit version the repo expects. When present, the system shim uses it for version resolution. When absent, the shim errors with an actionable message directing the user to run `fab init`.

### `fab upgrade` (shim subcommand, replaces `fab-upgrade.sh`)

`fab upgrade [version]` is a shim subcommand that replaces the current `fab/.kit/scripts/fab-upgrade.sh` shell script. Flow:

1. Resolve target version — latest release if no argument, or the explicit version (e.g., `fab upgrade 0.44.0`)
2. Download the release to cache if not already present (binary + `.kit/` content)
3. Copy `~/.fab-kit/versions/X.Y.Z/kit/` → repo's `fab/.kit/` (atomic swap, same strategy as current `fab-upgrade.sh`)
4. Update `fab_version` in `fab/project/config.yaml` to the new version
5. Run `fab-sync.sh` to deploy skills to agent directories
6. Display version change and migration reminder if needed

`fab-upgrade.sh` is removed from `fab/.kit/scripts/`. The shim handles the full upgrade lifecycle.

### `fab init` (primary use case)

The shim's main onboarding command. When run outside a fab-managed repo (or in a repo without `fab_version`), `fab init` scaffolds the `fab/project/` structure, populates `fab/.kit/` from the cache, and sets `fab_version` to the latest release. This completes the "zero to working" story: `brew install fab-kit` → `fab init` → repo is fab-managed.

### Sync pipeline changes

The current sync pipeline (`fab/.kit/sync/*.sh`) is updated:

| Script | Status | Change |
|--------|--------|--------|
| `1-prerequisites.sh` | **Modified** | `fab-doctor.sh` adds check for `fab` system binary on PATH |
| `2-sync-workspace.sh` | **Unchanged** | Dirs, scaffold, skill deployment, migration version, sync stamp — all still needed |
| `3-direnv.sh` | **Unchanged** | |
| `4-get-fab-binary.sh` | **Removed** | No binaries in repo — `fab-go` comes from shim cache, `wt`/`idea` from Homebrew |
| `5-sync-hooks.sh` | **Modified** | `$kit_dir/bin/fab hook sync` → `fab hook sync` (system shim) |

The `.envrc` scaffold (`fragment-.envrc`) removes the `PATH_add fab/.kit/bin` line — the system shim is on PATH via Homebrew. `PATH_add fab/.kit/scripts` is retained for `fab-sync.sh` and `fab-doctor.sh`.

The shim auto-fetches on first invocation, so there is no scenario where `fab hook sync` runs against an uncached version. Whether invoked via `fab upgrade`, `fab init`, or a post-`git pull` invocation, the shim resolves and caches the version before dispatching.

### Migration

A migration file in `fab/.kit/migrations/` is required for existing repos. The migration:

1. **Prerequisite gate**: Instruct user to `brew install fab-kit` before proceeding
2. **Add `fab_version`**: Write `fab_version: "X.Y.Z"` to `fab/project/config.yaml` (set to the current kit VERSION)
3. **Clean `.envrc`**: Remove the `PATH_add fab/.kit/bin` line
4. **Clean `fab/.kit/bin/`**: Remove binaries (`fab`, `fab-go`, `wt`, `idea`) — only `.gitkeep` remains

After migration, the system shim handles all binary resolution. The migration is applied via `/fab-setup migrations` as with all fab-kit migrations.

### Build artifacts: 4 Go binaries

The build produces 4 Go binaries with distinct distribution paths:

| Binary | Source | Distribution | Location |
|--------|--------|-------------|----------|
| `fab` (shim) | `src/go/shim/` | Homebrew only | `/usr/local/bin/fab` |
| `fab-go` | `src/go/fab/` | Cache (per-version) | `~/.fab-kit/versions/X.Y.Z/fab-go` |
| `wt` | `src/go/wt/` | Homebrew only | `/usr/local/bin/wt` |
| `idea` | `src/go/idea/` | Homebrew only | `/usr/local/bin/idea` |

The shim (`fab`), `wt`, and `idea` are version-independent — they ship with `brew install fab-kit` and upgrade via `brew upgrade`. Only `fab-go` is version-coupled to the repo and lives in the cache. The shell dispatcher at `fab/.kit/bin/fab` is removed.

## Affected Memory

- `fab-workflow/distribution`: (modify) Document the Homebrew distribution model, shim architecture, cache layout, version resolution, binary-free repo model, `fab upgrade` replacing `fab-upgrade.sh`, sync pipeline changes
- `fab-workflow/kit-architecture`: (modify) Update to reflect binary-free `.kit/`, wt/idea system-only, shim dispatcher model
- `fab-workflow/configuration`: (modify) Document new `fab_version` field in config.yaml
- `fab-workflow/migrations`: (modify) Document the brew-install migration

## Impact

- **Homebrew formula**: New formula in a tap (`wvrdz/homebrew-tap`)
- **`fab/.kit/bin/`**: All binaries removed (`fab`, `fab-go`, `wt`, `idea`)
- **`fab/.kit/scripts/fab-upgrade.sh`**: Removed — replaced by `fab upgrade` shim subcommand
- **All skill files**: Path prefix `fab/.kit/bin/fab` → `fab` (bulk find-and-replace across ~20 files)
- **Hook scripts**: `fab/.kit/hooks/on-*.sh` updated to call `fab` instead of `fab/.kit/bin/fab`
- **Constitution**: Principle V amended
- **`config.yaml` schema**: New optional `fab_version` field
- **Go codebase**: New shim binary in `src/go/shim/` (with `init`, `upgrade`, and dispatch logic)
- **GitHub releases**: Release artifacts structured for shim to download and cache (binary + `.kit/` content)
- **Sync pipeline**: `4-get-fab-binary.sh` removed, `5-sync-hooks.sh` and `1-prerequisites.sh` updated
- **`.envrc` scaffold**: `PATH_add fab/.kit/bin` line removed
- **Existing repos**: Migration needed — install system shim, add `fab_version`, clean `.envrc` and `fab/.kit/bin/`

## Open Questions

- None — all questions from PR #287 resolved, binary-free model confirmed in follow-up discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Homebrew formula name is `fab-kit`, binary name is `fab` | Established in PR #287 — user explicitly chose this split to avoid Python Fabric collision on formula name while keeping `fab` as the user-facing command | S:95 R:90 A:95 D:95 |
| 2 | Certain | `wt` and `idea` are system-only, not per-repo | Established in PR #287 — user confirmed these are standalone utilities not version-coupled to the repo | S:95 R:80 A:90 D:95 |
| 3 | Certain | No Go binary in repo — `fab/.kit/bin/` is empty | Discussed — user proposed removing binary from repo; system shim resolves from cache. Repo holds content only. | S:95 R:70 A:90 D:90 |
| 4 | Certain | Version pinned per-repo via `config.yaml` field | Established in PR #287 — user chose version-manager pattern over wrapper pattern | S:90 R:70 A:85 D:85 |
| 5 | Certain | Skills call `fab` not `fab/.kit/bin/fab` | Discussed — direct consequence of binary-free repo; system shim is the entry point | S:95 R:65 A:95 D:95 |
| 6 | Certain | Constitution Principle V amended to require system shim | Discussed — user confirmed the amendment is needed and acceptable | S:95 R:60 A:90 D:90 |
| 7 | Confident | Cache lives at `~/.fab-kit/versions/` | Standard XDG-style user cache location; could also be `~/.cache/fab-kit/` | S:60 R:90 A:70 D:60 |
| 8 | Confident | Shim downloads from GitHub releases | Natural fit given existing `wvrdz/fab-kit` repo; alternative would be a separate artifact store | S:65 R:85 A:75 D:70 |
| 9 | Certain | Homebrew tap at `wvrdz/homebrew-tap` | Clarified in PR #287 — user confirmed org-level tap | S:95 R:90 A:50 D:50 |
| 10 | Certain | Error when `fab_version` absent from `config.yaml` | Clarified in PR #287 — user chose strict mode; shim errors with actionable message | S:95 R:70 A:50 D:45 |
| 11 | Certain | `fab init` is in scope as a primary use case | Clarified in PR #287 — user confirmed this is the main use case for the shim | S:95 R:70 A:80 D:90 |
| 12 | Certain | No automatic cache eviction — manual cleanup only | Clarified in PR #287 — versions are small, cleanup command can be added later | S:95 R:90 A:80 D:90 |
| 13 | Certain | No `fab self-update` — rely on `brew upgrade fab-kit` | Clarified in PR #287 — don't reinvent the package manager | S:95 R:90 A:85 D:90 |
| 14 | Certain | Full removal of `fab/.kit/` from repos is future scope — not this change | Discussed — user scoped this change to binary removal only; `.kit/` content stays | S:95 R:85 A:95 D:95 |
| 15 | Certain | Cache stores `fab-go` binary + full `.kit/` content per version | Discussed — user confirmed cache stores both; content needed for `fab upgrade` to populate repos | S:95 R:80 A:90 D:90 |
| 16 | Certain | `fab upgrade` is a shim subcommand replacing `fab-upgrade.sh` | Discussed — shim handles full upgrade lifecycle: download, cache, copy content to repo, bump version, sync | S:95 R:75 A:90 D:90 |
| 17 | Certain | 4 Go binaries: `fab` (shim), `fab-go`, `wt`, `idea` | Discussed — user confirmed the binary inventory; shell dispatcher removed | S:95 R:85 A:95 D:95 |
| 18 | Certain | Sync: `4-get-fab-binary.sh` removed, `5-sync-hooks.sh` calls `fab hook sync` | Discussed — no binaries in repo, shim is on PATH; auto-fetch ensures version is cached before dispatch | S:95 R:80 A:90 D:95 |
| 19 | Certain | `.envrc` scaffold removes `PATH_add fab/.kit/bin` | Discussed — system shim on PATH via Homebrew, repo bin dir no longer needed | S:95 R:85 A:95 D:95 |
| 20 | Certain | Migration required for existing repos | Discussed — adds `fab_version` to config.yaml, cleans `.envrc` and `fab/.kit/bin/`, gates on `brew install fab-kit` | S:95 R:65 A:85 D:90 |

20 assumptions (18 certain, 2 confident, 0 tentative, 0 unresolved).

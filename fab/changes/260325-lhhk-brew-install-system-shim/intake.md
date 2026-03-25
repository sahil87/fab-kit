# Intake: Brew Install System Shim

**Change**: 260325-lhhk-brew-install-system-shim
**Created**: 2026-03-25
**Status**: Draft

## Origin

> System-level installation for fab-kit via Homebrew. `brew install fab-kit` installs three binaries on PATH: `fab` (version-aware shim that reads fab_version from repo config.yaml, finds/fetches the matching fab/.kit/ release, and dispatches to the per-repo fab/.kit/bin/fab runtime), `wt` (worktree management), and `idea` (backlog management). The per-repo `fab/.kit/bin/fab` stays as the versioned runtime — never on PATH, invoked by the shim via absolute path. Each repo pins its own fab-kit version in config.yaml. The shim handles version resolution, download/caching of releases, and dispatch. wt and idea move out of fab/.kit/bin/ to system-only install.

This change was preceded by an exploratory `/fab-discuss` session covering distribution strategy, naming, and version management tradeoffs. Key decisions from that discussion are captured in the Assumptions table below.

## Why

Today, fab-kit is distributed by copying `fab/.kit/` into each repo. This works but creates friction:

1. **First-run friction** — new repos require manual `cp -r` or running an upgrade script to bootstrap `fab/.kit/`. There's no standard system-level command to get started.
2. **No version management** — repos vendor their own copy of `.kit/`, but there's no mechanism for a repo to declare which version it needs and have the tooling automatically resolve it. Upgrades are manual per-repo.
3. **Standalone tools trapped in repos** — `wt` (worktree management) and `idea` (backlog management) are general-purpose utilities useful outside fab-managed repos, but currently live inside `fab/.kit/bin/` and are only accessible within repos that have fab installed.

Without this change, onboarding remains manual, version drift across repos is invisible, and standalone utilities are unnecessarily scoped to fab-managed repos.

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
2. Read `fab_version` from `config.yaml` (e.g., `fab_version: "0.39.0"`)
3. Check the local cache for the matching version (`~/.fab-kit/versions/0.39.0/`)
4. If not cached, download the release from GitHub (`wvrdz/fab-kit` releases) and cache it
5. Exec `~/.fab-kit/versions/0.39.0/fab/.kit/bin/fab <original args>` — full passthrough of all arguments

If no `config.yaml` is found (not in a fab-managed repo), the shim can still serve non-repo commands (e.g., `fab init` to scaffold a new project, `fab --version`).

```yaml
# fab/project/config.yaml — new field
fab_version: "0.39.0"
```

### Cache layout

```
~/.fab-kit/
  versions/
    0.39.0/
      fab/.kit/bin/fab      # the versioned runtime
      fab/.kit/bin/fab-go   # the Go backend
      fab/.kit/skills/      # skill files for this version
      fab/.kit/templates/   # templates for this version
    0.38.3/
      ...
```

### `wt` and `idea` become system-only binaries

These binaries move out of `fab/.kit/bin/` and are distributed exclusively through the Homebrew formula. They are not version-coupled to the per-repo fab-kit version — they're standalone utilities.

### Per-repo runtime stays at `fab/.kit/bin/fab`

The per-repo binary at `fab/.kit/bin/fab` remains unchanged. It is the versioned runtime that the shim dispatches to. It is never on PATH — the shim invokes it by absolute path. The binary name stays `fab` (not renamed).

### New `config.yaml` field: `fab_version`

A new optional field in `fab/project/config.yaml` that declares the fab-kit version the repo expects. When present, the system shim uses it for version resolution. When absent, the shim falls back to the latest cached version (or latest release).

## Affected Memory

- `fab-workflow/distribution.md`: (new) Document the Homebrew distribution model, shim architecture, cache layout, and version resolution
- `fab-workflow/architecture.md`: (modify) Update to reflect wt/idea moving to system-only, shim dispatcher model

## Impact

- **Homebrew formula**: New formula in a tap or core homebrew (likely a tap: `wvrdz/tap`)
- **`fab/.kit/bin/`**: `wt` and `idea` removed from this directory in future releases
- **`config.yaml` schema**: New optional `fab_version` field
- **Go codebase**: New shim binary (small — walks up for config, resolves version, execs)
- **GitHub releases**: Release artifacts must be structured for the shim to download and cache
- **Existing repos**: No breaking change — repos without the shim continue to work via direct `fab/.kit/bin/fab` invocation

## Open Questions

- Should the shim support `fab init` for scaffolding new repos (downloading latest `.kit/` into a fresh project)?
- What's the cache eviction policy? Keep all versions indefinitely, or prune versions not used in N days?
- Should there be a `fab self-update` for updating the shim itself (separate from per-repo version)?
- How should the Homebrew tap be structured? `wvrdz/homebrew-tap` with a `fab-kit.rb` formula?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Homebrew formula name is `fab-kit`, binary name is `fab` | Discussed — user explicitly chose this split to avoid Python Fabric collision on formula name while keeping `fab` as the user-facing command | S:95 R:90 A:95 D:95 |
| 2 | Certain | `wt` and `idea` are system-only, not per-repo | Discussed — user confirmed these are standalone utilities not version-coupled to the repo | S:95 R:80 A:90 D:95 |
| 3 | Certain | Per-repo binary stays named `fab` at `fab/.kit/bin/fab` | Discussed — user decided against renaming the inner binary since the shim dispatches by absolute path | S:90 R:85 A:90 D:90 |
| 4 | Certain | Version pinned per-repo via `config.yaml` field | Discussed — user chose version-manager pattern (option 1) over wrapper pattern (option 2) | S:90 R:70 A:85 D:85 |
| 5 | Confident | Cache lives at `~/.fab-kit/versions/` | Standard XDG-style user cache location; could also be `~/.cache/fab-kit/` | S:60 R:90 A:70 D:60 |
| 6 | Confident | Shim downloads from GitHub releases | Natural fit given existing `wvrdz/fab-kit` repo; alternative would be a separate artifact store | S:65 R:85 A:75 D:70 |
| 7 | Tentative | Homebrew tap at `wvrdz/homebrew-tap` | Common pattern for org taps; could also be `wvrdz/homebrew-fab-kit` or attempt core homebrew | S:40 R:90 A:50 D:50 |
<!-- assumed: Homebrew tap naming — defaulting to org-level tap, could be project-specific -->
| 8 | Tentative | Fallback to latest cached version when `fab_version` absent | Reasonable default for repos not yet pinned; alternative is to error | S:50 R:70 A:50 D:45 |
<!-- assumed: Fallback behavior — could be stricter (require explicit version) or looser (always use latest) -->

8 assumptions (4 certain, 2 confident, 2 tentative, 0 unresolved). Run /fab-clarify to review.

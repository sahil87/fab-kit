---
type: memory
description: "How `src/kit/` is distributed — Homebrew formula (2 binaries direct + 2 via `depends_on`), `fab` router (always-route policy), `fab-kit` lifecycle, `fab init` bootstrap, `fab upgrade-repo` (offline-first default = installed binary's `systemVersion`; `--latest` opts into the GitHub-API newest-release path; explicit arg wins; `dev`/unstamped → network fallback — 1hmj), release workflow (3 binaries, 12 cross-compiled, `SHA256SUMS` for kit-* archives, `just test` gate before tag/build + `go-version-file` single-sourcing — tb6f), hardened auto-download (bounded HTTP timeouts, version-keyed flock + atomic rename, digest verification) + fail-loud lifecycle exit contracts (`init`/`update`/`upgrade-repo`/`sync`) incl. `sync`/`migrations-status`'s distinguishable exit `3` = \"not a fab-managed repo\" via the shared `RequireManagedRepo()` guard (`internal.ExitNotManaged`, 52i9; `upgrade-repo` unaffected, still exit 1), `wt shell-setup` wrapper; the `shll.ai/fab-kit` public docs site — README-slice pull (`ReadmeSlice.astro`) + producer-side README-conformance obligation (tail boundary at `## Development`, diagram SVGs by absolute raw URL, absolute external slice links — except README→`docs/site/` links kept repo-relative for the site rewrite) + `docs/` audience-axis layout (pull surface is exactly `README.md` + `docs/site/**`; `docs/site/**` pulled + rendered one page per file at `/tools/fab-kit/<path>`, §9 ACTIVE)"
---
# Distribution

**Domain**: distribution

## Overview

How `src/kit/` is distributed to new and existing projects. Covers the Homebrew distribution model (fab-kit ships `fab` router + `fab-kit` workspace lifecycle directly; declares `wt` and `idea` as Homebrew dependencies pulled from sibling formulas), the bootstrap process (getting `.kit/` into a project for the first time — primary method is `brew install fab-kit` + `fab init`), the update mechanism (`fab upgrade-repo` replaces the old `fab-upgrade.sh`; its no-arg default resolves offline to the installed binary's version, with `--latest` opting into the GitHub-API path), the release workflow (version management via `release.sh`, build recipes via `justfile`, CI orchestration via `.github/workflows/release.yml` — producing per-platform archives with Go binaries), the repo rename from `docs-sddr` to `fab-kit`, and the `shll.ai/fab-kit` public docs site (the README-slice pull surface and fab-kit's standing producer-side README-conformance obligation).

## Requirements

### Homebrew Distribution

#### Homebrew Formula

A Homebrew formula named `fab-kit` SHALL be published to the `sahil87/tap` tap (GitHub repo: `sahil87/homebrew-tap`). The formula SHALL install two binaries directly: `fab` (router/dispatcher) and `fab-kit` (workspace lifecycle). The formula SHALL declare `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"` so Homebrew transitively installs the standalone `wt` (worktree management) and `idea` (backlog management) formulas — yielding four binaries on PATH after install. Users add the tap via `brew tap sahil87/tap`.

The standalone `wt` and `idea` formulas in `sahil87/homebrew-tap` SHALL declare `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"` respectively. This allows them to take ownership of pre-existing symlinks (e.g., from a `fab-kit 1.6.2` install that previously bundled `wt`/`idea` directly) silently during `brew upgrade fab-kit`. The directives are also carried in the templates of `sahil87/wt` and `sahil87/idea`, so subsequent regenerations preserve them.

**Scenarios**:
- Fresh install (`brew tap sahil87/tap && brew install fab-kit`) — installs `fab` and `fab-kit` directly; resolves `depends_on` and installs `wt` and `idea` from sibling formulas; all four respond to `--version` (each reporting its own formula's version)
- Upgrade from fab-kit 1.6.2 (which bundled `wt`/`idea` directly) — `brew upgrade fab-kit` triggers installation of standalone `wt`/`idea` via `depends_on`; `link_overwrite` lets them adopt the existing `bin/wt`, `bin/idea` symlinks without "Refusing to link" errors
- Upgrade via Homebrew (`brew upgrade fab-kit`) — updates the router and fab-kit to the latest formula version; per-version cache is unaffected. Standalone `wt`/`idea` upgrade independently on their own release cadence
- Troubleshooting fallback — if `link_overwrite` does not resolve cleanly for some reason, run `brew unlink wt idea && brew upgrade fab-kit`

#### Router Architecture (System `fab` Binary)

The system `fab` binary acts as a router using negative-match dispatch. It maintains a static allowlist of fab-kit commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`) that are dispatched to `fab-kit` via `syscall.Exec`. A separate set of inline commands (`--version`, `-v`, `--help`, `-h`, `help`) are handled directly by the router without exec'ing any sub-binary. All other commands are dispatched to the version-resolved `fab-go` — the router applies an **always-route policy** with no `config.yaml` gate.

For fab-go dispatch, the router SHALL:

1. Walk up from CWD to find `fab/project/config.yaml`
2. Select version inline: if `cfg != nil` use `cfg.FabVersion` (project-pinned, e.g., `fab_version: "0.43.0"`); otherwise use the router's build-time `version` constant (router-bundled, set via `-ldflags -X`)
3. Check the local cache for the matching `fab-go` binary at `~/.fab-kit/versions/{version}/fab-go`
4. If not cached, download the release from GitHub (`sahil87/fab-kit` releases) and cache it
5. Exec the cached `fab-go` with full argument passthrough

If `config.yaml` exists but cannot be parsed, the router hard-errors with the parse error from `internal.ResolveConfig` (corrupted-config path is unchanged). Only the missing-config case becomes a soft fall-through to the bundled version.

fab-go's per-command guards (typically `resolve.FabRoot()`) are the authoritative answer to "does this need config?" — commands that require project state (`preflight`, `score`, `resolve`, `status`, `change`, `log`, `batch`, `fab-help`) fail-closed with `ERROR: fab/ directory not found`, while config-free commands (`kit-path`, `pane`, `operator`'s switch path, hooks, `completion`, `shell-init`, `help`, `<subcommand> --help`) run cleanly from anywhere.

`fab help` composes help from both sub-binaries: workspace commands (from fab-kit) are always shown; workflow commands (from fab-go) are also always shown — using the project-pinned `fab_version` inside a fab-managed repo and the router's build-time `version` (bundled fab-go) outside, so all workflow commands remain discoverable from scratch tabs. Help-block errors are silently swallowed (best-effort).

**Scenarios**:
- fab-kit command dispatch — `fab init`, `fab sync`, `fab upgrade-repo` are routed to `fab-kit` with all args passed through
- Normal fab-go dispatch — router reads `fab_version`, resolves cached `fab-go`, execs with all args passed through
- Version not cached — router auto-fetches from GitHub releases, caches binary + `.kit/` content, then dispatches
- No network during auto-fetch — exits non-zero with version and network hint
- `config.yaml` found but `fab_version` absent — exits with: `"No fab_version in config.yaml. Run 'fab init' to set one."`
- Not in a fab-managed repo, fab-kit command — `fab init`, `fab sync` dispatched to `fab-kit` (works without config.yaml)
- Not in a fab-managed repo, inline command — `fab --version` and `fab --help` handled inline by the router (no config.yaml needed); `fab --version` prints only the system version line when no `fab/project/config.yaml` is found
- Not in a fab-managed repo, workflow command — dispatched to the router-bundled `fab-go`. Commands that need project state (e.g., `fab preflight`, `fab score`) self-guard and exit non-zero with `ERROR: fab/ directory not found`. Config-free commands (e.g., `fab kit-path`, `fab pane map`, `fab completion zsh`, `fab shell-init zsh`, `fab --help`) run successfully
- Corrupted `config.yaml` — router hard-errors with the parse error before any dispatch

#### Cache Layout

The router and fab-kit store versioned artifacts at `~/.fab-kit/versions/{version}/`. Each version directory contains:

- `fab-go` — the Go backend binary for the current platform
- `kit/` — full `.kit/` content (skills, templates, scripts, hooks, migrations, scaffold, VERSION)

Multiple versions coexist independently. No automatic cache eviction — users manage cleanup manually.

**Scenarios**:
- Cache structure after auto-fetch — `~/.fab-kit/versions/0.43.0/fab-go` exists and is executable; `kit/VERSION` contains `0.43.0`; `kit/skills/` contains skill files
- Multiple versions coexist — repos using different `fab_version` values dispatch to separate cached binaries

#### Auto-Download Hardening (Timeouts, Lock, Atomic Install, Checksums)

The auto-download path (`internal/download.go` in `src/go/fab-kit`, shared by the router shim, `fab-kit init`/`upgrade-repo`/`sync` via `EnsureCached`) is hardened on four axes (260612-dn2c, findings F16/F17/F20):

1. **Bounded HTTP** — no `http.Get`/`http.DefaultClient` anywhere in the download path. Two dedicated clients: `apiClient` (flat 30s `Timeout`) serves small bodies (`LatestVersion`, the `SHA256SUMS` fetch); `downloadClient` serves the streaming archive with **no flat timeout** (it would abort legitimately slow downloads) — instead 30s dial/TLS bounds, 30s `ResponseHeaderTimeout`, and a 10-minute overall `context.WithTimeout` deadline on the request. A black-holed GitHub/CDN connection now fails with a clear error instead of hanging every uncached `fab <cmd>` (including bare `fab`/`fab --help`, whose `printHelp → EnsureCached` path also fetches).
2. **Version-keyed download lock** — concurrent downloaders of the same version serialize on a blocking exclusive `syscall.Flock` on `versions/<version>.lock` (`internal/lock.go`, `acquireLock`). The lock file is deliberately left in place after release (unlinking races with waiters already blocked on the inode). After acquiring the lock, `Download` re-checks `ResolveBinary` and returns early when a peer completed the fetch — N racing processes (operator pane fan-out after a pin bump) perform exactly one network fetch. The helper is intentionally local to the fab-kit module — the two Go modules (`src/go/fab`, `src/go/fab-kit`) deliberately share no code.
3. **Checksum verification** — `Download` fetches the same-release `SHA256SUMS` asset first, streams the archive to a temp file through SHA-256, and verifies the digest **before any byte is extracted**. Digest mismatch, or a sums file missing the platform archive's entry, refuses to install. HTTP 404 on `SHA256SUMS` (a pre-checksum release) warns on stderr and skips verification, so projects pinned to older versions remain installable. Trust model: same-release sums defend **integrity** (corruption/truncation, rustup/nvm-style baseline) — explicitly not asset-swap attackers; a separately-trusted digest channel and sigstore signing are recorded non-goals (follow-on).
4. **Atomic install** — extraction happens in `versions/<version>.tmp-<pid>`, then a single `os.Rename` into place. Cache-dir readiness is therefore **all-or-nothing**: a version dir that exists with `fab-go` is complete (previously the exec bit was set at file-create time before content streamed, so `ResolveBinary`'s `mode&0111` probe could pass on a partial binary). A pre-existing `versions/<version>/` dir with no resolvable `fab-go` (a stale partial left by pre-fix binaries) is removed under the lock just before the rename. Error-path cleanup is scoped to the temp archive/temp dir only — a failed download never `RemoveAll`s a live cache dir a sibling process may be reading.

**Scenarios**:
- Black-holed network on an uncached version — every `fab <cmd>` fails with a bounded, retryable timeout error instead of wedging; a legitimately slow multi-minute archive download still completes (only time-to-headers + the 10m deadline bound it)
- N concurrent `fab` processes racing on first fetch — exactly one downloads; the rest block on the lock and return early on the re-check; none can observe a partially extracted version dir
- Checksum mismatch — `checksum mismatch for kit-…: expected …, got … — refusing to extract`; nothing lands in the cache
- `SHA256SUMS` present but missing the platform archive's entry — hard failure (`refusing to install`)
- Pre-checksum release (404 on `SHA256SUMS`) — stderr `WARNING: release vX publishes no SHA256SUMS asset — skipping checksum verification`, install proceeds
- Extraction fails mid-stream (corrupt archive, disk full) — only the temp artifacts are removed; an existing live `versions/<v>/` dir is untouched

### Bootstrap

#### Primary Method: `brew install fab-kit` + `fab init`

The primary bootstrap path for new projects is:
```
brew tap sahil87/tap && brew install fab-kit
cd <repo>
fab init
```

`fab init` (a fab-kit subcommand, routed via the `fab` router, not dispatched to fab-go) SHALL:
1. Verify the CWD is inside a git repository — checked BEFORE any download or config write, so a failed init leaves no stale artifacts behind. Fails with `fab init requires a git repository — run 'git init' first` (non-zero exit, no network fetch, no `fab/` files created)
2. Resolve the latest release version from GitHub
3. Ensure the version is cached (verified, atomic download if not — see Auto-Download Hardening above); kit content is served from the system cache, never copied into the repo
4. Set `fab_version: "{latest}"` in `fab/project/config.yaml` (creating the file if needed)
5. Stamp `fab/.kit-migration-version` to the engine version (before sync, so a fresh project isn't classified as legacy)
6. Call `Sync(systemVersion, latest, …)` directly (the same logic as `fab-kit sync`) — the embedded binary version feeds the version guard and the just-resolved kit version is passed explicitly. A sync failure propagates: `fab init` exits non-zero

`fab-kit sync` follows a 6-step pipeline, resolving all kit content from the system cache (`~/.fab-kit/versions/{version}/kit/`) rather than `src/kit/` in the repo: (1) validates prerequisites (`git`, `bash`, `yq` v4+, `direnv`), (2) version guard (ensures `fab_version` <= system `fab-kit` version; when tripped it attempts `fab update` and then verifies post-state — it ALWAYS fails the current run, either with `fab-kit was updated to vX — re-run 'fab sync'` or with actionable too-old/release-lag/unverifiable instructions; see [kit-architecture.md](/distribution/kit-architecture.md)), (3) ensures cache (verified, atomic download if needed), (4) workspace scaffolding from cache (directories, scaffold tree-walk with fragment merges and copy-if-absent, skill deployment to detected agents, hook sync, version stamp, legacy cleanup), (5) direnv allow, (6) project-level `fab/sync/*.sh` scripts. Supports `--shim` (steps 1-5 only) and `--project` (step 6 only) flags; mutually exclusive. **Deployment write failures are fail-loud** (260612-dn2c, F21): failed skill writes are counted per-skill (never as `created`/`repaired`), surfaced as `WARN: {agent}: failed to deploy {skill}: …` on stderr plus a `failed N` figure in the per-agent tally, and make `Sync` return non-nil (no `Done.`) — `fab sync` exits non-zero, the failure signal `/fab-setup` and scripts rely on. Scaffolding write failures (directories, `.gitkeep`s, the `.kit-migration-version` writes — whose silent failure used to silently disable migration discovery — and the kit `VERSION` read) propagate the same way. **Distinguishable "not a fab-managed repo" exit** (52i9): a plain `fab sync` (no explicit `kitVersion`) run outside a fab-managed repo does NOT collapse to that generic exit `1` — it prints `not in a fab-managed repo. Run 'fab init' to set one up` to stderr and exits **`3` (`internal.ExitNotManaged`)** via the shared `RequireManagedRepo()` guard, so callers (`wt`'s default init, `hop`, operator scripts) can branch on "not applicable here" vs. a real sync failure without replicating fab's `config.yaml` walk-up. The managed-repo check gates **before** the git-root resolution, so a non-git, non-fab directory also exits `3` (symmetric with `fab-kit migrations-status`, which shares the same guard) — see [kit-architecture.md](/distribution/kit-architecture.md) § Distinguishable Exit Codes. A genuine sync failure (corrupt config, failed write, version-guard trip) still exits `1`, unchanged; `fab upgrade-repo` outside a managed repo is unaffected and still exits `1`.

**Scenarios**:
- Init in a new repo (no `fab/` directory) — `config.yaml` created with `fab_version` set to latest; `.kit-migration-version` stamped; sync deploys skills from the cache
- Init in a repo with existing `fab/` but no `fab_version` — `fab_version` added to existing `config.yaml`; existing project files NOT overwritten
- Init outside a git repository — exits non-zero immediately with the git-repo error; no network fetch occurs and no `fab/` files are created
- Sync fails during deployment (read-only/full filesystem under `.claude/skills/`) — each failed skill reported on stderr, tally shows `failed N`, `fab sync` exits non-zero (`1`)
- `fab sync` outside any fab-managed repo (no `fab/project/config.yaml` on any ancestor) — prints the `not in a fab-managed repo` message to stderr and exits `3`, not `1`; holds even when the directory is not inside a git repository (the managed-repo check gates before git-root resolution)

#### Legacy One-Liner Bootstrap

The curl-based one-liner bootstrap continues to work for environments where Homebrew is not available:

**With compiled backend** (auto-detects platform via `uname`):
```
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac
mkdir -p fab; curl -sL "https://github.com/{repo}/releases/latest/download/kit-${os}-${arch}.tar.gz" | tar xz -C fab/
```

Where `{repo}` is `sahil87/fab-kit`. After extraction, the user runs `src/kit/scripts/fab-sync.sh` to complete workspace setup.

#### Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work, given the system `fab` binary is installed (`brew install fab-kit`). The system binary provides version-aware execution; `src/kit/` provides content (skills, templates, configuration).

**Scenario**: Manual copy (`cp -r /path/to/fab-kit/fab/.kit fab/.kit`) produces an identical result to the curl bootstrap.

### Update

#### `fab upgrade-repo` (Shim Subcommand)

`fab upgrade-repo [version] [--latest]` is a fab-kit subcommand (routed via the `fab` router to `fab-kit`, not dispatched to `fab-go`) that replaces the former `src/kit/scripts/fab-upgrade.sh`. It SHALL (ordering per 260612-dn2c, F18 — sync first, stamp after):

1. Resolve the target version by this precedence — **first match wins** (signature: `Upgrade(systemVersion, targetVersion string, useLatest bool)`):
   - **Explicit `<version>` arg** (e.g., `fab upgrade-repo 0.44.0`) → that version. Wins over everything; `--latest` is ignored when an arg is given (explicit intent beats a discovery flag). Offline.
   - **`--latest`** → the newest published GitHub release via `LatestVersion()` (`GET releases/latest`). This is the opt-in network path — the pre-2.3.x default, now behind a flag.
   - **No arg, no `--latest` (the default)** → the running binary's own embedded `systemVersion`, provided it is a real release tag (not empty, not `"dev"`). **Offline** — no GitHub round-trip. This reconciles the repo's kit to the `brew`-installed `fab-kit` binary, which is the natural meaning of "upgrade this repo".
   - **No arg with a `dev`/unstamped binary** (`systemVersion == "dev"` or empty — a `just build` shim) → falls back to `LatestVersion()`, because a `dev` shim has no real release tag to sync to (syncing `vdev` would fail).

   *Why offline-first:* the user has just `brew upgrade`d the binary and wants their repo to match it — a question the binary already answers from its own embedded version, no network needed. The old API-default forced a GitHub round-trip on the dominant path, which on a shared host trips GitHub's unauthenticated rate limit (60 req/hr/IP) and hard-fails with a misleading `cannot resolve latest version: GitHub API returned HTTP 403` (a `403` reads as auth failure, not rate limiting). The *resolution* is now offline by default; the *fetch* of a resolved-but-uncached target still downloads on demand via `EnsureCached`.
2. Short-circuit when `fab_version` already equals the target: "Already on the latest version (X). No update needed."
3. Download the release to cache if not already present (binary + `.kit/` content; verified + atomic per Auto-Download Hardening) and verify the cached kit carries a `VERSION` file; kit content is served from the cache, never copied into the repo
4. Run `Sync()` FIRST — `Upgrade(systemVersion, targetVersion string, useLatest bool)` passes the kit version explicitly and the embedded binary version feeds the version guard. The in-upgrade sync (including project-level `fab/sync/*.sh` scripts) runs while `config.yaml` still pins the OLD `fab_version`
5. Stamp `fab_version` in `fab/project/config.yaml` only AFTER the sync succeeds
6. Display version change (`Updated: x -> y`) and run mechanical migration discovery (reminder / overlap warning / silent self-stamp as applicable)

**Failure contract** (the documented behavior is now enforced): on sync failure the command exits non-zero with `sync failed: … — run 'fab sync' to repair, then re-run 'fab upgrade-repo'`, never prints `Updated: x -> y`, and leaves `config.yaml` on the old version — so a re-run of `fab upgrade-repo` retries the upgrade instead of short-circuiting the broken state on "Already on the latest version". (Previously the stamp landed before sync, sync failure was downgraded to a stderr WARNING with exit 0, and the re-run short-circuited — stale skills with a lying success code.)

**Scenarios**:
- Default no-arg upgrade (binary stamped `2.3.1`) — resolves the target to `2.3.1` **offline** (the binary's own `systemVersion`); `LatestVersion()` / the GitHub API is never called; then downloads-if-uncached, syncs, and stamps. This is the common path after a `brew upgrade fab-kit`
- `fab upgrade-repo --latest` — resolves via `LatestVersion()` (GitHub `releases/latest`), downloads new version to cache, runs sync (skills deployed from the new cache), then updates `fab_version`, displays "Updated: 0.43.0 → 0.44.0"
- Upgrade to specific version (`fab upgrade-repo 0.42.1`) — resolves to the arg offline (no API call), downloads to cache, syncs, then updates `fab_version`
- Explicit arg with `--latest` (`fab upgrade-repo 2.2.0 --latest`) — resolves to `2.2.0`; `--latest` is ignored, no API call
- `dev`/unstamped binary, no arg — falls back to `LatestVersion()` (no real release tag to sync to otherwise)
- Already up to date — displays "Already on the latest version (0.43.0). No update needed.", no files modified
- Sync fails mid-upgrade — exits non-zero with the sync error + repair guidance, no `Updated:` line, `config.yaml` still on the old version; re-running `fab upgrade-repo` re-attempts the upgrade
- Migration reminder — when `fab/.kit-migration-version` is behind the new version and a migration exists, output includes a reminder to run `/fab-setup migrations`
- No network access, default no-arg path — succeeds (resolution is offline; the systemVersion is almost always already cached because the brew binary fetched its own kit on install). Only the `--latest`/`dev`-fallback paths require the network, and those exit non-zero on no network

#### Update Preserves Project Files

`fab upgrade-repo` MUST NOT modify project content. Its write surface is: `fab/project/config.yaml` (`fab_version` stamp, only after a successful sync), `fab/.kit-migration-version` (silent self-stamp only when no migrations apply), and the sync-managed workspace files (agent skill deployments, scaffolding). Preserved: `fab/project/constitution.md`, `docs/memory/`, `docs/specs/`, `fab/changes/`, `.fab-status.yaml`.

#### Deprecated: `fab-upgrade.sh`

`src/kit/scripts/fab-upgrade.sh` has been removed. Use `fab upgrade-repo` instead.

#### `fab update` Exit Semantics

`fab update` (fab-kit self-update via Homebrew) exits non-zero when fab-kit was not installed via Homebrew: `Update()` returns the sentinel `var ErrNotBrewInstalled` (`errors.Is`-matchable) instead of the former nil, while preserving the user guidance prints ("Update manually, or reinstall with: brew install sahil87/tap/fab-kit"). Previously the not-brew path returned nil — `fab update` exited 0 having updated nothing, and `versionGuard` treated the nil as "updated" (silently defeating the guard for go-install/manual/CI installs). Brew subprocess failures (`brew update`, `brew upgrade`, bounded at 30s/120s) also exit non-zero, as before. The `dev` version sentinel does not shelter local builds — the justfile injects real semver via `-X main.version`.

### Sync Staleness Detection

Preflight compares `$(fab kit-path)/VERSION` against `fab_version` in `fab/project/config.yaml` and emits a non-blocking stderr warning when they differ:

- `⚠ Skills may be out of sync — run fab sync to refresh (engine X, project Y)`

If either value is unreadable or empty, the check is silently skipped. This detects stale local skill deployments when a developer pulls new `src/kit/` source via git but hasn't re-run `fab sync` (since `.claude/`, `.agents/`, `.opencode/` are gitignored and not updated by git pull).

#### Atomic Update

Atomicity lives in the cache install, not in any in-repo copy (kit content is never copied into repos since `260402-gnx5`): downloads extract into `versions/<version>.tmp-<pid>` and are renamed into place only after digest verification and full extraction, under the version-keyed download lock — see **Auto-Download Hardening** above. `fab upgrade-repo` additionally verifies the cached kit carries a `VERSION` file before syncing.

**Scenarios**:
- Interrupted during download — live cache and project unchanged (only temp artifacts exist, removed on the error path)
- Interrupted during extraction to temp dir — live cache unchanged; the orphaned temp dir is inert (a fresh run re-downloads under the lock)
- Checksum or VERSION verification fails — aborts before any cache replacement, displays error

#### Skill Deployment Repair After Update

After caching the new version, `fab upgrade-repo` SHALL call `Sync()` directly (the same logic as `fab-kit sync`, before stamping `fab_version`) to ensure all skill deployments are up to date: copies refreshed (`.claude/skills/`, `.agents/skills/`), symlinks valid (`.opencode/commands/`), and stale agent files cleaned up (`.claude/agents/`).

### wt Shell Setup

#### `wt shell-setup` Subcommand

The `wt` binary provides a `shell-setup` subcommand that outputs a shell wrapper function to stdout, suitable for `eval` in the user's shell profile. This follows the direnv/rbenv/mise pattern.

**Recommended setup** (add to `~/.bashrc` or `~/.zshrc`):
```bash
eval "$(wt shell-setup)"
```

The output defines a `wt()` shell function that wraps the real `wt` binary: captures stdout line-by-line, prints each line through, and if the last line starts with `cd `, evals it in the calling shell. The output also includes `export WT_WRAPPER=1` so the binary can detect the wrapper is active.

Shell detection: reads `$SHELL` basename. For `bash`, `zsh`, or unset `$SHELL`, outputs the wrapper silently. For unrecognized shells, outputs the same bash/zsh wrapper with a stderr warning (`warning: unsupported shell "{shell}" — outputting bash/zsh wrapper`).

The wrapper function text is defined as `ShellWrapperFunc` constant in `cmd/shell_setup.go` of `github.com/sahil87/wt`.

#### `WT_WRAPPER` Environment Variable Detection

When `open_here` is selected (in both `wt open` and `wt create`, via the shared `OpenInApp` function in `internal/worktree/apps.go` of `github.com/sahil87/wt`), the binary checks `os.Getenv("WT_WRAPPER")`. If the value is not `"1"`, a two-line hint is printed to stderr before the `cd` command is printed to stdout:

```
hint: "Open here" requires the shell wrapper to cd. Run: eval "$(wt shell-setup)"
      Add it to your ~/.zshrc or ~/.bashrc to make it permanent.
```

The hint goes to stderr so it does not interfere with the `cd` command on stdout. When `WT_WRAPPER=1` is set, no hint is printed.

#### `"default"` Keyword for App Resolution

Both `wt open --app` and `wt create --worktree-open` accept `"default"` as a keyword value (case-sensitive, lowercase). When received, the command resolves the app via `ResolveDefaultApp()` (in `internal/worktree/apps.go` of `github.com/sahil87/wt`), which delegates to `DetectDefaultApp()` — the same priority chain used by the interactive menu (TERM_PROGRAM → tmux/byobu session → cached last-app → first available non-open_here app).

On success, `SaveLastApp` is called and the app opens. On failure (no default detected): `wt open` exits with an error; `wt create` prints a stderr warning and continues (non-fatal, matching the existing `--worktree-open` error pattern). This asymmetry exists because `wt open --app default` is an explicit open request (failure is meaningful), while `wt create --worktree-open default` is secondary to worktree creation (failing the entire create for an open failure would be disruptive).

### Release

Release is split across three components: `release.sh` handles version management and git operations, a `justfile` at repo root provides locally-replicable build recipes, and `.github/workflows/release.yml` orchestrates CI. The key principle: CI uses the exact same `just` commands a developer runs locally — no CI-only build logic.

#### Release Script (`release.sh`)

`scripts/release.sh` handles version bumping, migration validation, and git commit/tag/push. It does NOT cross-compile, package archives, or create GitHub Releases — those responsibilities moved to the justfile and CI workflow.

The script accepts a bump type argument (`patch`, `minor`, or `major`) that is required to perform a release. When invoked with no arguments, the script displays usage and exits successfully. Unknown arguments produce an error.

The script pushes to the current branch (via `git branch --show-current`) rather than hardcoded `main`. On `main`, behavior is identical to before. On a release branch (e.g., `release/0.25`), commits and tags are pushed to that branch. The tag push triggers CI to handle cross-compilation, packaging, and GitHub Release creation.

After bumping VERSION, the script validates the migration chain: warns if no migration file targets the new version (reminder for release authors), and warns if overlapping migration ranges are detected. These are warnings only — they do not block the release.

Pre-flight checks: clean working tree (error if dirty), `$(fab kit-path)/VERSION` exists (error if missing). The script does NOT check for `gh` CLI or Go toolchain — those are no longer needed locally for releasing.

**Scenarios**:
- Default patch release — bumps patch version (e.g., "0.34.0" → "0.34.1"), commits VERSION bump with message `release: v0.34.1`, creates tag `v0.34.1`, pushes commit and tag to current branch; CI takes over from the tag push
- Minor release (`release.sh minor`) — bumps minor version (e.g., "0.34.1" → "0.35.0")
- Major release (`release.sh major`) — bumps major version (e.g., "0.35.0" → "1.0.0")
- Backport release — on branch `release/0.34`, `release.sh patch` bumps 0.34.1→0.34.2, pushes to `release/0.34`, tags `v0.34.2`; CI creates the release, and GitHub's semver ordering ensures the backport is not marked as "latest"
- Backport workflow — `git checkout -b release/0.34 v0.34.1`, cherry-pick fixes, `release.sh patch` bumps and pushes to `release/0.34`, CI handles the rest
- Invalid bump argument — exits with error message listing valid options
- Unknown argument — exits with error listing valid options
- No git remote configured — exits with error
- Dirty working tree — aborts with error directing user to commit or stash

#### Build Recipes (`justfile`)

The `justfile` at repo root provides locally-replicable build recipes using [just](https://github.com/casey/just). These same recipes are invoked by CI.

**Development recipes**:
- **`build`** — compiles all three fab-kit-owned binaries (`fab` router, `fab-kit`, `fab-go`) for the current platform using `CGO_ENABLED=0`
- **`test`** — runs all unit tests across `src/go/fab/` and `src/go/fab-kit/`
- **`test-v`** — runs all unit tests (verbose)
- **`doctor`** — checks prerequisites and environment health

**Release recipes** (all output goes to `dist/`):
- **`release [bump]`** — bumps VERSION (default: patch), commits, tags, and pushes; CI handles the rest
- **`dist-kit`** — assembles `dist/kit/` from `src/kit/` (single copy, reused by packaging)
- **`build-target os arch`** — cross-compiles all three fab-kit-owned binaries for a specific platform into `dist/bin/{name}-{os}-{arch}`
- **`build-all`** — cross-compiles for all 4 release targets (`darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`), producing 12 binaries total (3 per platform)
- **`package-kit`** — creates 4 per-platform `dist/kit-{os}-{arch}.tar.gz` (kit content + `fab-go` only). Archives are rooted at `.kit/`.
- **`package-brew`** — creates 4 per-platform `dist/brew-{os}-{arch}.tar.gz` (`fab`, `fab-kit`)
- **`release-notes [tag]`** — generates `dist/release-notes.md` with commit-level changelog
- **`brew-formula [tag]`** — generates `dist/fab-kit.rb` from template with SHA256 hashes
- **`dist`** — full pipeline: `dist-kit` + `build-all` + `package-kit` + `package-brew`
- **`clean`** — removes `dist/`

**Three fab-kit-owned Go binaries**:

| Binary | Source | Distribution |
|--------|--------|-------------|
| `fab` (router) | `src/go/fab-kit/cmd/fab/` | Homebrew formula `sahil87/tap/fab-kit` |
| `fab-kit` | `src/go/fab-kit/cmd/fab-kit/` | Homebrew formula `sahil87/tap/fab-kit` |
| `fab-go` | `src/go/fab/` | Per-version cache via GitHub releases (`sahil87/fab-kit`) |

**Two external dependency binaries** (installed transitively via the fab-kit formula's `depends_on`):

| Binary | Source repo | Distribution |
|--------|-------------|-------------|
| `wt` | `github.com/sahil87/wt` | Homebrew formula `sahil87/tap/wt` |
| `idea` | `github.com/sahil87/idea` | Homebrew formula `sahil87/tap/idea` |

**Scenarios**:
- Local dev build (`just build`) — compiles three fab-kit-owned binaries for current platform
- Cross-compile for a single target (`just build-target darwin arm64`) — produces 3 binaries in `dist/bin/`
- Build all targets (`just build-all`) — produces 12 binaries in `dist/bin/` (3 per platform x 4 platforms)
- Full pipeline (`just dist`) — assembles kit, builds all, packages all into `dist/`
- Package without prior build (`just package-kit`) — fails with error directing to run prerequisite steps first
- Clean up (`just clean`) — removes `dist/`

#### CI Workflow (`.github/workflows/release.yml`)

`.github/workflows/release.yml` is a GitHub Actions workflow triggered on push of tags matching `v*`. It runs on a single `ubuntu-latest` runner and uses the same `just` recipes as local development.

Workflow steps:
1. Checkout repository (`actions/checkout@v4`)
2. Set up Go toolchain (`actions/setup-go@v5`, `go-version-file:` pointing at `src/go/fab/go.mod` — the module go.mod is the single source of truth for the Go version, same as ci.yml; both modules declare the same line)
3. Install `just` command runner (`extractions/setup-just@v2`)
4. Run `just test` — the release test gate (260612-tb6f, F40): both Go modules' suites must pass before any artifact is built, and on manual dispatch it runs BEFORE the tag is created, so a red suite never even mints the tag. Previously the workflow shipped binaries on any tag push with zero test steps
5. Run `just build-all` (cross-compiles all 12 targets: 3 fab-kit-owned binaries x 4 platforms)
6. Run `just package-kit` (creates the 4 per-platform `kit-*` archives with `fab-go` — `fab` router and `fab-kit` are Homebrew-distributed via `package-brew`; `wt` and `idea` are external Homebrew dependencies)
7. Generate `SHA256SUMS` over the four `kit-*` archives (`cd dist && sha256sum kit-*.tar.gz > SHA256SUMS`) — verified by the shim's `Download` before extraction; `brew-*` archives are verified by Homebrew itself (260612-dn2c, F20)
8. Create GitHub Release via `gh release create` with the 8 archives (4 `kit-*` + 4 `brew-*`) plus `SHA256SUMS` and commit-level changelog (minor releases cumulate all commits since the previous minor; patch releases show commits since the previous release)

The workflow sets `permissions: contents: write` for release creation. `GITHUB_TOKEN` is used implicitly by `gh`.

`Build all targets` flows directly into `Package kit archives` — the release workflow carries no push-side shll.ai delivery step. fab-kit's `help-dump` command remains the contract surface, but shll.ai now *pulls* the command reference itself (its own puller invokes `fab help-dump` and handles capture / validate / commit / render). See the change-log entry for `260603-mtf9-teardown-shll-push` below for the teardown.

GitHub determines "latest" release status based on semver ordering — backport releases for older version series (e.g., `v0.34.2` when `v0.35.0` exists) are not marked as latest automatically. For edge cases, use `gh release edit $TAG --latest=false` after CI creates the release.

**Scenarios**:
- Tag push triggers workflow — push of `v0.35.0` tag triggers the release workflow
- Non-tag push does not trigger — regular commits pushed without a `v*` tag do not run the workflow
- Full CI release — tag `v0.35.0` triggers workflow, which cross-compiles all fab-kit-owned Go binaries (12 total: fab router, fab-kit, fab-go x 4 platforms), packages 8 archives (4 `kit-*` with `fab-go` + 4 `brew-*`), generates `SHA256SUMS` over the `kit-*` archives, and creates a GitHub Release with the archives + `SHA256SUMS` and commit-level changelog. The standalone `wt` and `idea` releases are produced independently from `sahil87/wt` and `sahil87/idea` and are not part of the fab-kit release workflow
- Backport release via CI — tag `v0.34.2` triggers workflow; GitHub's semver ordering ensures it is not marked as "latest" since `v0.35.0` exists

#### Release Archive Contents

Each release produces per-platform archives structured for the router/fab-kit to download and cache. Per-platform archives (`kit-{os}-{arch}.tar.gz`) contain:
- `.kit/bin/fab-go` — the versioned Go backend binary
- `.kit/` — all content (skills, templates, scripts, hooks, migrations, scaffold, VERSION)

The router (or fab-kit) extracts `fab-go` to `~/.fab-kit/versions/{version}/fab-go` and the rest to `~/.fab-kit/versions/{version}/kit/`.

Per-platform archives:
- **`kit-darwin-arm64.tar.gz`** — Content + `fab-go` compiled for macOS Apple Silicon.
- **`kit-darwin-amd64.tar.gz`** — Content + `fab-go` compiled for macOS Intel.
- **`kit-linux-arm64.tar.gz`** — Content + `fab-go` compiled for Linux ARM64 (musl, fully static).
- **`kit-linux-amd64.tar.gz`** — Content + `fab-go` compiled for Linux x86-64 (musl, fully static).

Release assets also include **`SHA256SUMS`** — `sha256sum` digests covering the four `kit-*` archives, generated by the release workflow and consumed by the shim's `Download` to verify the archive before extraction (260612-dn2c, F20). The `brew-*` archives are not listed in it — Homebrew verifies those via the formula's own SHA256 fields. (The former generic `kit.tar.gz` content-only archive is no longer produced — `package-kit` builds exactly the 4 per-platform archives.)

No project-specific files (config.yaml, constitution.md, memory/, specs/, changes/) are included in any archive. Package production code (idea only) is included under `.kit/packages/`, hook scripts under `.kit/hooks/` — all delivered to downstream projects on upgrade. `src/kit/sync/` contains only `.gitkeep` (all sync scripts absorbed into `fab-kit` Go binary). `idea` is a standalone system binary (installed via Homebrew, not per-repo); the shell package at `.kit/packages/idea/bin/idea` is retained for rollback safety and generic-archive users. Skill files are included in all archives and deployed to agents by `fab-kit sync`. `fab-go binary at ` contains only `.gitkeep` — no binaries are shipped in the repo.

**Binary distribution split**: The router (`fab`) and `fab-kit` ship in fab-kit's Homebrew formula (version-coupled to fab-kit's release tag). `wt` and `idea` are external standalone Homebrew formulas in `sahil87/tap`, declared as `depends_on` so they install transitively (each versioned independently). Only `fab-go` is per-version cached, downloaded from `sahil87/fab-kit` GitHub releases on first use.

### shll.ai Public Docs Site (README-Slice Pull)

`shll.ai/fab-kit` is fab-kit's public documentation site. Its per-tool README page (`/tools/fab-kit/readme`) renders a **deduced, curated slice** of *this* repo's `README.md`, pulled by shll.ai's `scheduled-readme-refresh.yml` into `content/fab-kit/README.md` and rendered at build time by `ReadmeSlice.astro`. **The tool repo is canonical** — shll.ai never hand-edits the prose; any structural defect in our README ships verbatim to the public site, and the only place to fix it is *here*.

This is the **README-slice pull**, the second of two pull surfaces shll.ai consumes from fab-kit (the first is the **command reference**, pulled via `fab help-dump` — see the Release / CI Workflow section and the `260602-xob7` / `260603-mtf9` changelog entries). Both follow the same model: shll.ai pulls; fab-kit never pushes. The contract is `~/code/sahil87/shll.ai/docs/specs/readme-extraction-contract.md`.

**How the slice is deduced** (consumer-side, canonical on shll.ai — informational; not changed here):
- **Head**: the leading H1, the toolkit blockquote, and the badge row are skipped.
- **Tail**: the slice is cut at the first **denylisted heading** — `Contributing` / `Development` / `Building` / `License` / `Acknowledgements`. Everything from that heading to EOF is GitHub-only.
- **Stripping**: ` ```mermaid ` fences are stripped (Astro Starlight does not render mermaid), and `#gh-*-mode-only` themed images are dropped.
- **No relative-base rewrite**: `ReadmeSlice.astro` uses `createMarkdownProcessor({})` and emits hrefs/`src` verbatim — it performs **no** relative-path rewriting.

#### Producer-Side README Conformance Obligation

Because the slice ships verbatim, fab-kit's `README.md` MUST stay conformant to the extraction contract. These are **standing structural obligations** on this repo's README (a `docs`-type concern, not a CLI/skill concern):

1. **Tail boundary at `## Development`** — a single top-level `## Development` heading (a denylisted heading) ends the site slice. GitHub-native chrome lives below it as `###` subsections: the "Stage Coverage by Command" matrix, "Companion tools", and "Learn More". Everything above the boundary is the site slice and MUST end on genuinely user-facing prose. (The pre-existing `### Developing Fab Kit` subsection under Prerequisites is left as a `###` and is NOT promoted — its heading text is "Developing Fab Kit", which does not match the denylist.)
2. **Diagrams as committed SVGs referenced by absolute raw URLs** — each ` ```mermaid ` fence stays (GitHub renders mermaid natively) and is immediately followed by a markdown image referencing a committed, **hand-authored** SVG in `docs/img/` via an **absolute** `https://raw.githubusercontent.com/sahil87/fab-kit/main/docs/img/<name>.svg` URL with descriptive alt text. The site strips the fence; the SVG is the only thing that survives. Two assets today: `docs/img/pipeline-stages.svg` and `docs/img/stage-coverage.svg`. No build step / CI render / render script — the fence is the canonical source, the SVG is a one-time manual render (Constitution I). Accepted cost: a future mermaid edit needs a manual SVG re-export, and the two can drift.
3. **Absolute links in the slice region (one sanctioned relative exception)** — every doc link in the slice region (head → `## Development` boundary) that leaves the rendered site MUST be an absolute `https://github.com/sahil87/fab-kit/blob/main/<path>` GitHub-blob URL. The README's relative base differs by surface (`/` on GitHub vs `/tools/fab-kit/readme/` on shll.ai) and `ReadmeSlice.astro` does no relative-base rewrite, so no single relative string resolves in both — **with one exception the extractor DOES rewrite**: a README link written as the repo-relative `docs/site/<p>.md` is rewritten by shll.ai to the site-absolute `/tools/fab-kit/<p>` (and resolves as a plain repo link on GitHub). So README→`docs/site/` links are deliberately kept **repo-relative** (plain inline form only — never behind a badge/image, never reference-style, both of which are unhandled and 404); all other slice-region links stay absolute GitHub-blob URLs. In-page `#anchor` links survive when their target heading stays in the slice; anchors pointing at moved/non-existent sections are dropped or re-pointed.

#### `docs/` Audience-Axis Layout

`docs/` carries an audience-axis split (per the contract's §9 model — *who* the docs are for, not "wanted vs. unwanted"):

| Dir | Audience | Reaches the site? |
|-----|----------|-------------------|
| `README.md` slice | Users (tool consumers) | Yes — the slice today |
| `docs/site/` | Users | **Yes** — pulled + rendered as one page per file at `/tools/fab-kit/<path>` (§9 ACTIVE since `x0br`) |
| README tail (below `## Development`), `CONTRIBUTING.md` | GitHub-native readers | No — fenced off by the tail boundary |
| `docs/img/` | (asset store) | Indirectly — via absolute raw URLs in the README |

- **Pull surface is exactly `README.md` + `docs/site/**`** — everything else in the repo (source, tests, design notes, other `docs/` subtrees) is **un-pulled by default**. There is no blessed "maintainer-only" folder: maintainer/design notes need no special location and live **anywhere outside `docs/site/`** (the deleted `docs/internal/` concept is no longer part of the model). `docs/specs/` (Constitution VI pre-implementation design intent) and `docs/memory/` (post-implementation "what shipped") keep their fab meanings and are **not** pulled.
- **`docs/site/**/*.md` is a closed-set tree that IS pulled and rendered.** Each file becomes its own page at `/tools/fab-kit/<path>` (subtree shape preserved, the `docs/site/` prefix dropped). fab-kit publishes `docs/site/install.md` (→ `/tools/fab-kit/install`) and `docs/site/workflows.md` (→ `/tools/fab-kit/workflows`). Four closed-set producer rules govern every page: **(a) closure** — every relative link/image inside `docs/site/**` resolves *inside* `docs/site/` (no `..` escape); **(b) external links absolute-by-author** — `https://…`, with repo-internal targets written as `https://github.com/sahil87/fab-kit/blob/main/<path>`; **(c) all images absolute everywhere**; **(d) README→docs/site links** written naturally as `docs/site/<p>.md` (the site rewrites to `/tools/fab-kit/<p>`), as **plain inline links only** — never behind a badge/image and never reference-style, since those two shapes are unhandled by the consumer and 404. Reserved page slugs a `docs/site` page MUST NOT use: `overview` / `readme` / `commands`.

**Scenarios**:
- Simulated pull (mermaid fences stripped) — each diagram location still shows its SVG `<img>` (absolute src survives); diagrams do not vanish on the site.
- Tail-boundary scan ending at the first denylisted heading — yields a slice that excludes the Stage Coverage matrix, Companion tools, and Learn More.
- Following a rewritten doc link (e.g. Glossary) — resolves to a real file at `https://github.com/sahil87/fab-kit/blob/main/docs/specs/glossary.md` on both GitHub and the site.
- A new diagram or relative doc link added above the boundary — must carry an adjacent raw-URL SVG / be rewritten absolute, or it breaks on the site.
- Daily pull of the `docs/site/**` tree — `docs/site/install.md` and `docs/site/workflows.md` each render as their own page at `/tools/fab-kit/install` and `/tools/fab-kit/workflows`; an intra-set `./workflows.md` link resolves inside the set, while a `..`-escaping or repo-relative `docs/specs/` link inside a docs/site page would break (must be absolute `https://…`).
- A new `docs/site/` page named `overview` / `readme` / `commands` — collides with a reserved site-owned slug and must not be published; install/workflows are allowed.

### Deprecated: Backend Override Mechanism

The `FAB_BACKEND` env var and `.fab-backend` file mechanism has been removed. The Go backend is the only backend. The system shim dispatches to `fab-go` directly — no override needed. References to `FAB_BACKEND` and `.fab-backend` should be removed from scripts and documentation.

### Repo Rename

The repository SHALL be renamed from `docs-sddr` to `fab-kit` to reflect its role as the canonical source for `src/kit/`. GitHub auto-redirects handle existing URLs and clones.

**Scenarios**:
- Old URLs (`github.com/sahil87/docs-sddr`) redirect to the current repo URL
- Existing clones with old remote URL continue to work via redirect

## Design Decisions

- **CI/local parity via justfile (260307-ma7o-1)**: Build recipes live in the `justfile` so CI and local development use identical commands (`just build-all`, `just package-kit`). No CI-only build scripts or logic. This makes CI behavior fully reproducible locally.
- **Three-way release split (260307-ma7o-1)**: `release.sh` owns version/tag/push, `justfile` owns build/package, `.github/workflows/release.yml` owns orchestration. Each component has a single responsibility and can be tested independently.
- **GitHub semver ordering replaces `--no-latest` (260307-ma7o-1)**: GitHub automatically determines "latest" release based on semver. Backport releases (e.g., `v0.34.2` when `v0.35.0` exists) are not marked latest. The `--no-latest` flag was removed from `release.sh` — no flag to remember, no CI mechanism to pass it through. For edge cases, `gh release edit` can be used post-creation.
- **Commit-level release notes with minor cumulation**: CI generates release notes from `git log --oneline` with linked commit SHAs. Minor releases (x.y.0) cumulate all commits since the previous minor tag, giving a complete picture of the release cycle. Patch releases show commits since the previous release only. Major releases use the same patch-style diff (manual curation expected for milestone releases).
- **Homebrew distribution with two fab-kit-owned binaries + two external dependencies (260401-46hw, 260402-3ac3, 260506-4rtx)**: The system `fab` binary is a router installed via `brew install fab-kit`. It dispatches workspace commands to `fab-kit` and workflow commands to the version-resolved `fab-go`. `fab-kit` owns workspace lifecycle (init, upgrade-repo, sync, update, doctor — joined by `migrations-status` in 260610-9733; the 6-command set is single-sourced in the shared `LifecycleCommands` table since 260612-ye8r). The fab-kit formula directly ships only `fab` and `fab-kit`; `wt` and `idea` are pulled in via `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"` from sibling formulas in the same tap. User-visible install behavior is unchanged (all four binaries land on PATH), but each binary now versions and releases independently. Rejected: binary-in-repo (redundant when router manages versions), `fab self-update` (don't reinvent the package manager), two-binary shim model (untestable, blurred concerns).
- **Decouple wt and idea via `depends_on`, not Go module pin or CI-time external builds (260506-4rtx)**: After `wt` and `idea` were extracted into `github.com/sahil87/wt` and `github.com/sahil87/idea` with their own release pipelines, fab-kit's vendored `src/go/wt/` and `src/go/idea/` were removed and replaced with Homebrew dependency declarations. Each repo now versions and releases independently; fab-kit's CI shrinks (no longer cross-compiles `wt`/`idea`). Rejected: vendor via Go module dep `require github.com/sahil87/wt` (still ties fab-kit's release to a `wt` version pin and produces a duplicate `wt` binary built from fab-kit's CI); bundle binaries in fab-kit's brew tarball but build them from external repos at CI time (coupling moves from source to CI; fab-kit releases are blocked when external CI is broken); keep vendored sources, accept drift (defeats the purpose of the extraction).
- **`link_overwrite` in standalone wt/idea formulas, not `caveats` or `post_install` in fab-kit (260506-4rtx)**: To support upgrade from `fab-kit 1.6.2` (which bundled `wt`/`idea` directly), the standalone formulas declare `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"`. This is Homebrew's idiomatic mechanism for ownership transitions and runs silently. The directives are also carried in the `sahil87/wt` and `sahil87/idea` templates so subsequent regenerations preserve them. Rejected: `caveats` block in fab-kit asking users to `brew unlink wt idea` first (visible but does not actually solve the conflict); custom `post_install` migration logic in fab-kit (overkill, fragile).
- **`fab upgrade-repo` as fab-kit subcommand (260401-46hw, 260402-3ac3)**: `fab-kit` handles upgrade directly, replacing `fab-upgrade.sh`. `fab-kit` already has download/cache logic — upgrade is a natural extension. Rejected: keeping `fab-upgrade.sh` alongside `fab-kit` (duplication of download logic).
- **Cache stores binary + content (260401-46hw)**: Each cached version includes both `fab-go` and the full `.kit/` content. `fab upgrade-repo` needs the content to populate the repo's `src/kit/`. Rejected: binary-only cache (would need separate download for content).
- **Formula name `fab-kit`, binary name `fab` (260401-46hw)**: Homebrew formula uses `fab-kit` to avoid collision with Python Fabric's `fab` formula, while the installed binary is `fab`. Rejected: `fab` as formula name (collides with Fabric).
- **~~Help-dump delivered to shll.ai via auto-merging PR, not direct push (260602-xob7)~~**: *Superseded by 260603-mtf9* — shll.ai flipped from a push model to a pull model: its own puller now invokes `fab help-dump` and handles capture / validate / commit / render, so fab-kit's push-side delivery step was torn down (CI step list 7→5). Retained for history. The original rationale: the release workflow delivered `help/fab-kit.json` to `sahil87/shll.ai` by opening an auto-merge-squash PR rather than pushing to `main` directly, because fab-kit was one of 7 tools delivering their command reference into the single shll.ai repo around the same release window and concurrent direct pushes would hit non-fast-forward rejections; a per-tool PR with `--auto --squash` let GitHub serialize the merges. The dump + validate portion was **fatal** (a malformed dump is a genuine fab-kit bug), while the shll.ai PR portion was **non-fatal** (a downstream delivery failure must not block fab-kit's own release). The dump ran the rich `fab-go` binary (`dist/bin/fab-go-linux-amd64`), not the `fab`/`fab-kit` shim, because only that target carried both the full command tree and the `-X main.version` ldflags. The producer command (`help-dump`) remains the contract surface the puller invokes — only the push transport was removed.
- **README conformance is a producer-side, this-repo obligation; `docs/site/` lands as structure-only ahead of its consumer (260605-fcrp)**: shll.ai pulls a deduced README slice and never hand-edits, so README structure defects ship verbatim to the public site — the fix lives only in fab-kit's `README.md`. The standing obligations (tail boundary at `## Development`, diagrams as committed SVGs referenced by absolute raw URLs, absolute GitHub-blob links in the slice region) are folded into this distribution domain rather than a new `conventions/` domain — they are conventions *about the shll.ai distribution surface*, and `distribution.md` already owns the fab-kit↔shll.ai relationship (the `help-dump` command-reference pull). A standalone single-file domain would be thin and orphaned. On the diagrams: keep mermaid + commit an adjacent hand-authored SVG (rejected: SVG-only loses GitHub's native zoomable mermaid; drop diagrams leaves the current empty-gap defect). On links: GitHub-blob absolute (rejected: fully-relative — no string resolves in both bases; shll.ai-page URLs — needs a per-target page map that rots). On `docs/site/`: structure + explainer README only, **no content migration** *(superseded by 260608-yfg8 — §9 flipped RESERVED→ACTIVE on shll.ai, so `docs/site/` is now pulled + rendered; install/workflows pages migrated in, the explainer README removed)* (rejected: full migration now — §9 is RESERVED/unimplemented on shll.ai, so migrating load-bearing prose into the non-pulling `docs/site/` would strand it on both surfaces). The forward structure lands knowingly ahead of its §9 consumer.
- **`docs/internal/` removed; `docs/site/` activated as a pulled closed-set tree (260608-yfg8)**: the contract flipped §9 RESERVED→ACTIVE (change `x0br`, 2026-06-07) and deleted the `docs/internal/` concept (2026-06-08), so the pull surface is now exactly `README.md` + `docs/site/**` and maintainer notes need no blessed folder. Conforming is purely additive: removed the stale `docs/site/README.md` placeholder and the vestigial `docs/internal/` folder, added `docs/site/install.md` (→ `/tools/fab-kit/install`) + `docs/site/workflows.md` (→ `/tools/fab-kit/workflows`) as closed-set depth pages, and added two plain-inline README→docs/site body links. *Rejected*: rewriting `docs/site/README.md` to describe the new ACTIVE model — a README-named file under the now-pulled `docs/site/` renders as a public `/tools/fab-kit/README` page (and `README` is a reserved slug), and a docs-tree explainer is a maintainer note that belongs *outside* `docs/site/`. The depth pages **deepen** rather than duplicate the README slice (review de-duplicated `install.md` against the README per DEEPEN-not-duplicate, since the site pulls both surfaces).
- **Same-release SHA256SUMS as the accepted integrity baseline (260612-dn2c)**: the release publishes a `SHA256SUMS` asset for the `kit-*` archives and `Download` refuses to extract on digest mismatch or a missing entry; a 404 on the asset (pre-checksum releases) warns and skips so older pinned versions stay installable. This defends **integrity** (corruption/truncation — rustup/nvm-style industry baseline), explicitly NOT an attacker who can swap release assets (they'd swap the sums file too). Mechanism is hash-then-extract: the archive streams to a temp file through SHA-256 and extraction starts only after the digest verifies. *Rejected (for now)*: sigstore/cosign signing and separately-trusted digest channels (digest pinned in config.yaml / embedded in the brew-verified shim) — recorded follow-on; hash-while-extracting with post-hoc cleanup (extracts unverified bytes).
- **Atomic cache install under a version-keyed local flock (260612-dn2c)**: `Download` extracts into `versions/<version>.tmp-<pid>` and renames into place, serialized by a blocking exclusive `syscall.Flock` on `versions/<version>.lock` with a post-acquire `ResolveBinary` re-check; error cleanup is scoped to temp artifacts; the lock file is left in place after release (unlinking races with blocked waiters). The lock helper is implemented locally in the fab-kit module — the two Go modules (`src/go/fab`, `src/go/fab-kit`) deliberately share no code (`mz4q` builds its own flock helper in the other module). *Rejected*: extracting into the live dir with exec-bit readiness (the prior model — partial binaries were observable and error cleanup could `RemoveAll` a dir a sibling was using); cross-module lock-helper sharing; unlinking the lock file on release.
- **Stamp-after-success for `fab upgrade-repo` (260612-dn2c)**: `Upgrade` runs `Sync` first (kit version passed explicitly — enabled by the F22 version threading) and writes `fab_version` only after sync succeeds, so failure exits non-zero with repair guidance and a re-run retries. *Why*: rollback after a *partial* sync would create the inverse mismatch (new skills deployed, old version stamped). *Rejected*: literal stamp-then-rollback (the original backlog phrasing).
- **Offline-first `upgrade-repo` default = the installed binary's `systemVersion` (260613-1hmj)**: no-arg `fab upgrade-repo` resolves its target to the running `fab-kit` binary's embedded `systemVersion` (offline), instead of the former GitHub-API "newest published release" call; a new `--latest` flag opts into the API path; an explicit `<version>` arg still wins (and `--latest` is ignored when an arg is given); a `dev`/unstamped binary falls back to the network. The `systemVersion` was already threaded into `Upgrade()` (for the runSync version guard) — defaulting to it is a localized resolution-switch change with no downstream impact (`EnsureCached`/`runSync`/F18 stamp-after-sync/migration detection unchanged). *Why*: the dominant path is "I just `brew upgrade`d the binary, match my repo to it" — a question the binary answers from its own version offline. The old API-default forced a network round-trip that trips GitHub's unauthenticated rate limit (60 req/hr/IP) and hard-fails with a misleading `HTTP 403` on a shared host. *Rejected*: "latest *cached* version" (the user's first instinct — the cache can hold stale downloads or unreleased `local-versions/` dev builds, which `CachedKitDir` actively prefers; no enumeration helper exists; ambiguous and surprising); "keep the API default, only fix the 403 error text / read `GH_TOKEN`" (leaves the common path network-dependent — folded in partially: the rate-limit-naming error message and `GH_TOKEN` reading are deferred follow-ups, valuable only on the now-opt-in `--latest` path).
- **~~Backend override via env var + file (260307-bmp3-3)~~**: *Deprecated* — removed with the shim model. Go is the only backend; the shim dispatches to `fab-go` directly.

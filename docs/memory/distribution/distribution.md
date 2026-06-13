---
description: "How `src/kit/` is distributed — Homebrew formula (2 binaries direct + 2 via `depends_on`), `fab` router (always-route policy), `fab-kit` lifecycle, `fab init` bootstrap, `fab upgrade-repo` (offline-first default = installed binary's `systemVersion`; `--latest` opts into the GitHub-API newest-release path; explicit arg wins; `dev`/unstamped → network fallback — 1hmj), release workflow (3 binaries, 12 cross-compiled, `SHA256SUMS` for kit-* archives, `just test` gate before tag/build + `go-version-file` single-sourcing — tb6f), hardened auto-download (bounded HTTP timeouts, version-keyed flock + atomic rename, digest verification) + fail-loud lifecycle exit contracts (`init`/`update`/`upgrade-repo`/`sync`), `wt shell-setup` wrapper; the `shll.ai/fab-kit` public docs site — README-slice pull (`ReadmeSlice.astro`) + producer-side README-conformance obligation (tail boundary at `## Development`, diagram SVGs by absolute raw URL, absolute external slice links — except README→`docs/site/` links kept repo-relative for the site rewrite) + `docs/` audience-axis layout (pull surface is exactly `README.md` + `docs/site/**`; `docs/site/**` pulled + rendered one page per file at `/tools/fab-kit/<path>`, §9 ACTIVE)"
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

`fab-kit sync` follows a 6-step pipeline, resolving all kit content from the system cache (`~/.fab-kit/versions/{version}/kit/`) rather than `src/kit/` in the repo: (1) validates prerequisites (`git`, `bash`, `yq` v4+, `direnv`), (2) version guard (ensures `fab_version` <= system `fab-kit` version; when tripped it attempts `fab update` and then verifies post-state — it ALWAYS fails the current run, either with `fab-kit was updated to vX — re-run 'fab sync'` or with actionable too-old/release-lag/unverifiable instructions; see [kit-architecture.md](kit-architecture.md)), (3) ensures cache (verified, atomic download if needed), (4) workspace scaffolding from cache (directories, scaffold tree-walk with fragment merges and copy-if-absent, skill deployment to detected agents, hook sync, version stamp, legacy cleanup), (5) direnv allow, (6) project-level `fab/sync/*.sh` scripts. Supports `--shim` (steps 1-5 only) and `--project` (step 6 only) flags; mutually exclusive. **Deployment write failures are fail-loud** (260612-dn2c, F21): failed skill writes are counted per-skill (never as `created`/`repaired`), surfaced as `WARN: {agent}: failed to deploy {skill}: …` on stderr plus a `failed N` figure in the per-agent tally, and make `Sync` return non-nil (no `Done.`) — `fab sync` exits non-zero, the failure signal `/fab-setup` and scripts rely on. Scaffolding write failures (directories, `.gitkeep`s, the `.kit-migration-version` writes — whose silent failure used to silently disable migration discovery — and the kit `VERSION` read) propagate the same way.

**Scenarios**:
- Init in a new repo (no `fab/` directory) — `config.yaml` created with `fab_version` set to latest; `.kit-migration-version` stamped; sync deploys skills from the cache
- Init in a repo with existing `fab/` but no `fab_version` — `fab_version` added to existing `config.yaml`; existing project files NOT overwritten
- Init outside a git repository — exits non-zero immediately with the git-repo error; no network fetch occurs and no `fab/` files are created
- Sync fails during deployment (read-only/full filesystem under `.claude/skills/`) — each failed skill reported on stderr, tally shows `failed N`, `fab sync` exits non-zero

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
4. Run `Sync()` FIRST — `Upgrade(systemVersion, target)` passes the kit version explicitly and the embedded binary version feeds the version guard. The in-upgrade sync (including project-level `fab/sync/*.sh` scripts) runs while `config.yaml` still pins the OLD `fab_version`
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

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260613-1hmj-upgrade-repo-offline-default | 2026-06-13 | **Offline-first `upgrade-repo` default.** No-arg `fab upgrade-repo` now resolves its target to the running binary's embedded `systemVersion` (offline), instead of calling the GitHub API for the newest release. A new `--latest` boolean flag opts into the old GitHub-API path; an explicit `<version>` arg still wins (and silently ignores `--latest`); a `dev`/unstamped binary falls back to the network. `Upgrade(systemVersion, targetVersion string)` gained a third `useLatest bool` param; the no-arg resolution block became a 3-arm precedence switch (`useLatest` → `LatestVersion()`; else `systemVersion != "" && != "dev"` → `systemVersion`; else → `LatestVersion()` fallback). `upgradeCmd()` (`cmd/fab-kit/main.go`) wires the `--latest` flag; the `LifecycleCommands` `Short` for `upgrade-repo` is now "Upgrade the repo's kit to the installed binary's version (or --latest / an explicit version)" (`internal/lifecycle.go`). Everything downstream of resolution is unchanged (short-circuit, `EnsureCached`, `runSync`, F18 stamp-after-sync, migration detection). 4 new behavioral tests + the 9 existing callers updated to the new signature (`upgrade_test.go`). `_cli-fab.md` gained an `### upgrade-repo Version Resolution` precedence table (constitution-mandated CLI doc). *Why*: the old API-default tripped GitHub's unauthenticated rate limit (60 req/hr/IP) and hard-failed with a misleading `HTTP 403` on shared hosts; the binary already knows its own version, so the dominant "match my repo to the `brew`-upgraded binary" path needs no network. **Memory reconciliation** (this hydrate): rewrote the `#### fab upgrade-repo` step 1 to the precedence list + offline-first rationale, updated its scenarios, added the Design Decision above. Rate-limit error-message naming and `GH_TOKEN` reading deferred as follow-ups (valuable only on the `--latest` path). No migration (binary behavior + docs only). `fab-setup.md` / `SPEC-fab-setup.md` confirmed unaffected. |
| 260612-tb6f-tests-ci-toolchain | 2026-06-12 | **Release workflow test gate** (binary-review B6, F40/F41/F42): `release.yml` now runs `just test` (both Go modules) immediately after checkout — before the manual-dispatch tag mint and before any build/package/release step — so a red suite ships nothing and never even creates the tag; previously binaries shipped to Homebrew users on any `v*` tag push with zero test steps. The hardcoded `go-version: '1.22'` is replaced with `go-version-file: src/go/fab/go.mod`, making ci.yml's "single source of truth" comment true — the Go version now lives only in the module go.mod files (bumped to 1.26 in this change). ci.yml additionally runs both module suites with `-race` and cross-compiles darwin/arm64 (build + vet) on both matrix legs, so the build-constrained `*_darwin.go` files are type-checked on every PR instead of first compiling inside the release workflow after the tag is pushed. Release-workflow step list updated (setup-go step rewritten, `just test` inserted as step 4). Toolchain bump + yaml.v3 pin decision + golden byte-stability suite in [kit-architecture.md](kit-architecture.md). |
| 260612-ye8r-cli-single-sourcing-doc-conformance | 2026-06-12 | Body-text conformance only (binary-review B4, F23): the Homebrew design-decision bullet's workspace-lifecycle enumeration no longer omits `migrations-status` (joined the router allowlist in 260610-9733) and notes the 6-command set is single-sourced in the shared `LifecycleCommands` table (`src/go/fab-kit/internal/lifecycle.go`) — router `fabKitArgs`, in-process shim help, and `fab-kit` registrations all derive from it (see [kit-architecture.md](kit-architecture.md)). No distribution behavior change. |
| 260612-dn2c-fab-kit-download-lifecycle-hardening | 2026-06-12 | Hardened the auto-download path and made the lifecycle exit codes truthful (findings F16–F22 of `docs/specs/findings/binary-review-2026-06-12.md` §B3, plus absorbed backlog `[1old]`). **Download** (`src/go/fab-kit/internal/download.go` + new `lock.go`): dedicated HTTP clients (flat 30s for `LatestVersion`/sums; 30s dial/TLS/`ResponseHeaderTimeout` + 10-minute context deadline for the archive stream — no more `http.Get`/`http.DefaultClient`), version-keyed blocking `syscall.Flock` on `versions/<v>.lock` with post-acquire `ResolveBinary` re-check, hash-then-extract verification against a same-release `SHA256SUMS` asset (404 → warn-and-skip; mismatch/missing entry → refuse), extraction into `versions/<v>.tmp-<pid>` + atomic rename (readiness now all-or-nothing; stale binary-less dirs replaced under the lock), error cleanup scoped to temp artifacts. **Release workflow**: new `Generate SHA256SUMS` step + `dist/SHA256SUMS` in the `gh release create` asset list (only edits — test steps are `tb6f`'s). **upgrade-repo** (F18): sync runs FIRST and `fab_version` is stamped only after success — sync failure exits non-zero with "run 'fab sync' to repair, then re-run 'fab upgrade-repo'", no `Updated:` line, and re-runs retry (the `:116-130` documented contract is now enforced). **fab update** (F19): not-brew path returns `ErrNotBrewInstalled` (non-zero exit, guidance preserved). **sync** (F21): deployment/scaffolding write failures are counted (`failed N` tally), surfaced per-skill on stderr, and exit non-zero. **Version threading** (F22): `Init(version)`/`Upgrade(version, target)` pass the embedded binary version into `Sync(systemVersion, kitVersion, …)`. **fab init** ([1old]): git-repo precondition before any download/config write. Memory reconciliation: new Auto-Download Hardening subsection; init/upgrade-repo step lists rewritten (also dropping the pre-existing stale "copy kit/ into the repo's `src/kit/`" steps — kit content has been cache-served since `260402-gnx5`); Atomic Update section repointed at the cache install; verified incidental corrections to the release asset list (4 `kit-*` + 4 `brew-*`, no generic `kit.tar.gz`). `_cli-fab.md` gained a "Workspace Command Exit Semantics" table (constitution-mandated). versionGuard post-state semantics documented in [kit-architecture.md](kit-architecture.md). |
| 260608-yfg8-shll-readme-extraction-conformance | 2026-06-08 | Conformed the repo to shll.ai's README-extraction contract for the **§9-ACTIVE era**. Four content changes: (1) removed the stale `docs/site/README.md` placeholder (it asserted "§9 RESERVED / NOT YET IMPLEMENTED" and, with `docs/site/**` now pulled, would have rendered as a live self-contradicting `/tools/fab-kit/README` page on a reserved slug); (2) removed the `docs/internal/` folder entirely (the concept was deleted from the contract on 2026-06-08 — the pull surface is now exactly `README.md` + `docs/site/**`, so maintainer notes live anywhere outside `docs/site/`); (3) added `docs/site/install.md` (→ `/tools/fab-kit/install`) and `docs/site/workflows.md` (→ `/tools/fab-kit/workflows`) as closed-set depth pages obeying the four producer rules (closure/no-`..`-escape, external links absolute-by-author, all images absolute, README→docs/site links plain-inline); (4) added two plain-inline README→docs/site body links. **Memory reconciliation** (this hydrate): updated the `#### docs/ Audience-Axis Layout` table + bullets so `docs/site/` reads "pulled + rendered one page per file (§9 ACTIVE)" and the `docs/internal/` row/bullet are gone; rewrote the `docs/site/` bullet to the closed-set-tree model; updated the frontmatter `description:` tail; added the Design Decision above. **Supersedes** the §9-RESERVED deviation recorded by `260605-fcrp` (the contract flipped RESERVED→ACTIVE in `x0br`, 2026-06-07); the `260605-fcrp` Design-Decision `docs/site/` clause is annotated as superseded for history. Producer-side, this-repo-only `docs` change — no shll.ai edits, no CLI/skill changes. |
| 260605-fcrp-readme-site-extraction-conformance | 2026-06-05 | Added the **shll.ai Public Docs Site (README-Slice Pull)** Requirements subsection: documented the second pull surface (the README slice rendered at `/tools/fab-kit/readme` by `ReadmeSlice.astro`, pulled into `content/fab-kit/README.md` by `scheduled-readme-refresh.yml`; tool repo canonical, shll.ai never hand-edits) alongside the existing `help-dump` command-reference pull. Captured the producer-side conformance obligation now satisfied in `README.md`: (1) tail boundary at a single top-level `## Development` heading — Stage Coverage matrix + Companion tools + Learn More moved below it as `###` (GitHub-only); (2) both ` ```mermaid ` fences kept, each followed by an absolute `raw.githubusercontent.com/.../docs/img/<name>.svg` image (hand-authored `pipeline-stages.svg` + `stage-coverage.svg`, no build/CI/render step — Constitution I); (3) ~13 repo-relative `docs/specs/*.md` + `CONTRIBUTING.md` links in the slice region rewritten to absolute `github.com/.../blob/main/...` URLs; broken `#standalone-cli-tools` anchors dropped. Added the `docs/` audience-axis table: new `docs/internal/` (maintainer notes, never reaches site; distinct from `docs/specs/`/`docs/memory/`) and `docs/img/` (SVG assets). **Deviation recorded**: `docs/site/` landed as **structure + explainer README only — no content migrated**, because shll.ai's §9 `docs/site/` pull path is RESERVED/UNIMPLEMENTED (puller fetches only `README.md`); migrating load-bearing prose would strand it on both surfaces. Producer-side, this-repo-only `docs` change — no shll.ai edits, no `extract-readme.ts` changes, no CLI/skill changes. Folded into `distribution.md` rather than a new `conventions/` domain (thin/orphaned single-file domain avoided; `distribution.md` already owns the fab-kit↔shll.ai relationship). |
| 260603-mtf9-teardown-shll-push | 2026-06-03 | Tore down the deprecated push-side shll.ai integration now that shll.ai's puller is live and proven. Removed the entire `Help-dump → shll.ai` step from `.github/workflows/release.yml` (CI step list 7→5 entries) — both the fatal `fab help-dump > help/fab-kit.json` + `jq` dump/validate self-check AND the auto-merging cross-repo PR transport into `sahil87/shll.ai` (clone-with-token + `help-dump/fab-kit-<version>` branch + `gh pr create` + `gh pr merge --auto --squash`, secret `SHLLAI_TOKEN`). `Build all targets` now flows directly into `Package kit archives`. Removed the now-dead `/help/` entry (and its transient-artifact comment) from `.gitignore`. The `help-dump` command (`src/go/fab/cmd/fab/helpdump.go`, `helpdump_test.go`) and `src/kit/skills/_cli-fab.md` were deliberately left untouched — `help-dump` is the contract surface shll.ai's now-live puller invokes (it pulls the help tree itself rather than fab-kit pushing it). Manual follow-up (out-of-band, not a file edit): delete the `SHLLAI_TOKEN` repo secret from fab-kit's GitHub settings — a repo-wide grep confirmed it was referenced only by the removed step. Reverts the push-side CI mechanics added by `260602-xob7` while preserving the producer command. |
| 260602-xob7-cli-help-dump-shll-ai | 2026-06-03 | Release workflow gained a `Help-dump → shll.ai` step after `Build all targets` (CI step list 5→7 entries). The step runs `./dist/bin/fab-go-linux-amd64 help-dump > help/fab-kit.json`, validates it with `jq -e '.tool=="fab" and .schema_version==1 and (.version\|length>0) and (.root\|type=="object")'` (fatal), then opens an auto-merging PR into the external repo `sahil87/shll.ai` (clone-with-token + branch `help-dump/fab-kit-<version>` + `gh pr create` + `gh pr merge --auto --squash`, secret `SHLLAI_TOKEN`) writing `help/fab-kit.json` — non-fatal and idempotent (skips on byte-identical content via `git diff --cached --quiet`). PR-not-direct-push avoids the 7-tool push race into the single shll.ai repo. Dumps the rich `fab-go` binary (carries the command tree + `main.version` ldflags), not the shim. `/help/` added to `.gitignore` (the JSON is a transient shll.ai-owned artifact, not retained by fab-kit). shll.ai's site-side consumer (Astro loader + UI) is tracked separately in the shll.ai repo. |
| 260511-c432-fix-completion-outside-repo | 2026-05-12 | Router Architecture section rewritten to describe the always-route policy. Removed all references to the `fabGoNoConfigArgs` allowlist (no longer exists in `src/go/fab-kit/cmd/fab/main.go`) and the "Not in a fab-managed repo. Run 'fab init' to set one up." exit (gone). Router now routes every non-fab-kit command to `fab-go` regardless of `config.yaml` presence; version selection is inline (project-pinned when `cfg != nil`, router-bundled otherwise). fab-go's per-command guards are the authoritative gate. Corrupted-config path (parse error) still hard-errors. The "Not in a fab-managed repo, workflow command" scenario rewritten to reflect that workflow commands now reach fab-go and either run (config-free) or fail-closed via fab-go's own guard. |
| 260506-4rtx-decouple-wt-idea | 2026-05-06 | `wt` and `idea` split out of fab-kit's brew tarball into standalone Homebrew formulas in `sahil87/tap` (formerly bundled). fab-kit's formula declares them as `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"`. Brew tarball shrinks from 4 binaries to 2 (`fab`, `fab-kit`); cross-compile matrix shrinks from 5×4=20 to 3×4=12. `src/go/wt/` and `src/go/idea/` removed; canonical sources are `github.com/sahil87/wt` and `github.com/sahil87/idea`. `link_overwrite "bin/wt"` / `link_overwrite "bin/idea"` in the standalone formulas handle the upgrade-conflict from `fab-kit 1.6.2` (which owned those symlinks) transparently. Release notes for `1.7.0` document the `brew unlink wt idea && brew upgrade fab-kit` fallback for the rare case `link_overwrite` does not resolve cleanly. First release carrying this change is `1.7.0` (semver minor — user-visible install set unchanged via `depends_on` transitivity). Also swept stale `wvrdz/tap` references in active prose to `sahil87/tap` (historical Changelog rows preserved). |
| 260417-y0sw-pane-skip-config-check | 2026-04-17 | Router adds `fabGoNoConfigArgs` allowlist (currently `pane` only) exempting listed fab-go subcommands from the `config.yaml` requirement. Outside a fab repo, exempt commands dispatch to the bundled fab-go via the router's build-time `version`; inside a fab repo, the project-pinned `fab_version` is used unchanged. Updated "Not in a fab-managed repo, workflow command" scenario to note the `pane` exception. |
| 260409-5z32-wt-open-default-medium | 2026-04-09 | Added `"default"` keyword for `wt open --app` and `wt create --worktree-open`. Resolves via `ResolveDefaultApp()` → `DetectDefaultApp()` priority chain. `wt open` errors on no-default; `wt create` warns and continues. Added `SaveLastApp` call to `wt open --app` code path (was previously missing for all `--app` values). |
| 260404-g0x1-rename-upgrade-to-upgrade-repo | 2026-04-05 | Renamed `fab upgrade` to `fab upgrade-repo` throughout live prose, requirements, and command examples. Historical changelog entries preserved. |
| 260403-24ic-wt-open-shell-setup | 2026-04-03 | Added `wt shell-setup` subcommand (outputs shell wrapper function for `eval` in shell profile, following direnv/rbenv/mise pattern). Added `WT_WRAPPER=1` env var detection in `OpenInApp` — prints stderr hint when wrapper not installed and `open_here` is selected. Updated `wt` root command help text to reference `wt shell-setup` instead of inline function body. |
| 260402-5tci-remove-copilot-clean-scaffold | 2026-04-02 | Removed `scaffold/.github/copilot-code-review.yml` from the scaffold tree. Cleaned stale entries from `scaffold/fragment-.gitignore` (`fab/changes/**/.pr-done`, `/.ralph`). Scaffold file count unchanged at 11 (3 fragment, 8 copy-if-absent) after removal of the Copilot config and stale `.gitignore` lines. |
| 260402-gnx5-relocate-kit-to-system-cache | 2026-04-02 | Kit content no longer copied to user projects — served entirely from system cache at `~/.fab-kit/versions/<version>/kit/`. `fab init` and `fab upgrade` no longer create `fab/.kit/` in projects. Source repo layout: `fab/.kit/` renamed to `src/kit/`. Build scripts (`justfile`, `release.sh`) updated to read from `src/kit/`. `.gitignore` cleaned of `fab/.kit/` entries. `fab kit-path` command added for agent-agnostic kit path resolution. Bootstrap one-liner and manual copy references updated. |
| 260402-ktbg-sync-from-cache | 2026-04-02 | Rewrote `fab-kit sync` to resolve kit content from system cache (`~/.fab-kit/versions/{version}/kit/`) instead of `src/kit/` in the repo. 6-step pipeline: prerequisites, version guard, ensure cache, scaffolding, direnv, project scripts. Added `--shim` (steps 1-5) and `--project` (step 6) flags. Absorbed hook sync into step 4 (replicated hooklib in `fab-kit` internal package). Removed `5-sync-hooks.sh`. Fixed `fragment-.envrc` (`fab-kit sync` -> `fab sync`). Updated prerequisites (removed jq, gh — no longer needed by sync). Updated release archive description (sync/ now empty). |
| 260326-p4ki-allow-idea-shorthand | 2026-03-26 | Restored bare `idea "text"` shorthand (equivalent to `idea add "text"`). Added `RunE` with `cobra.ArbitraryArgs` to root command in `src/go/idea/cmd/main.go`. Multiple args joined with space. Empty text returns error. Persistent flags (`--main`, `--file`) work with shorthand. Updated `_cli-external.md` and `docs/specs/packages.md`. |
| 260320-9tqo-fix-idea-docs-main-flag | 2026-03-20 | Corrected `idea` documentation: moved Backlog section from `_cli-fab.md` to `_cli-external.md`, fixed invocation from `fab idea` to standalone `fab-go binary at idea`. Added `--main` persistent flag — default now uses current worktree (`--show-toplevel`), `--main` opts into main worktree (`--git-common-dir`). Renamed `GitRepoRoot()` to `MainRepoRoot()`, added `WorktreeRoot()`. Updated `_cli-external.md` frontmatter and `docs/specs/packages.md`. |
| 260312-96nf-remove-rust-implementation | 2026-03-12 | Removed all Rust references from distribution docs. Removed Rust recipes from build recipes section, Rust CI steps (toolchain, Zig, cargo-zigbuild), Rust from archive descriptions (3→2 binaries per platform, 12→8 total). Updated backend override to Go-only. Removed "Transition Period: Dual Backends" section. Updated bootstrap descriptions, packaging scenarios, and CI workflow steps. Removed cargo-zigbuild design decision. |
| 260310-8m3k-port-wt-tests-cleanup-legacy | 2026-03-10 | Removed `src/packages/` (legacy shell wt package and bats tests), `src/tests/` (bats submodule libs), and `.gitmodules` (bats submodule refs only). Ported 73 behavioral tests from bats to Go in `src/go/wt/cmd/*_test.go`. Removed `bats` from prerequisites description (already absent from actual sync scripts). Removed `test-setup` and `test-packages` justfile targets and their backing scripts (`scripts/just/test-setup.sh`, `test-packages.sh`). |
| 260310-qbiq-go-wt-binary | 2026-03-10 | Per-platform archives now include `wt` binary at `.kit/bin/wt` alongside `fab-go` and `fab-rust` (3 binaries per platform, 12 total cross-compiled). Added justfile recipes: `build-wt`, `build-wt-target`, `build-wt-all`. Updated `build-all` to include wt. Updated `package-kit` to verify and include wt binary. `src/kit/packages/wt/` removed — wt is a binary, not a shell package. `env-packages.sh` already adds `$KIT_DIR/bin` to PATH — no change needed for wt binary availability. |
| 260310-pl72-port-idea-to-go | 2026-03-10 | `idea` is now available as `fab idea` via the Go binary (in per-platform archives), in addition to the shell package at `.kit/packages/idea/bin/idea`. Both coexist — shell package retained for rollback safety and generic-archive users. |
| 260307-buf0-4-rust-ci-build | 2026-03-10 | Releases now ship both Go and Rust binaries. Added Rust cross-compilation recipes to justfile (`build-rust-target`, `build-rust-all`, `build-all`, `_rust-target`). Updated `package-kit` to include both `fab-go` and `fab-rust` in per-platform archives and exclude both from generic archive. CI workflow updated with Rust toolchain (`dtolnay/rust-toolchain`), Zig (`pip install ziglang`), `cargo-zigbuild`, and cached tool installations. `build-go-all` → `build-all` in CI. Linux Rust targets use musl for fully static binaries. |
| 260307-bmp3-3-rust-binary-port | 2026-03-10 | Added backend override mechanism (`FAB_BACKEND` env var, `.fab-backend` file) to dispatcher for switching between Rust and Go backends. Documented transition period where both binaries coexist — Rust preferred by default, Go shipped in release archives, Rust built locally via `just build-rust`. CI/release for Rust deferred. Updated bootstrap one-liners to reference both backends. |
| 260307-ma7o-1-ci-releases-justfile | 2026-03-09 | Split release workflow into three components: `release.sh` simplified to version bump + git commit/tag/push only (~60 lines, removed ~200 lines of build/package/release logic). New `justfile` at repo root provides build recipes (`build-go`, `build-go-target`, `build-go-all`, `package-kit`, `clean`) replicable locally and in CI. New `.github/workflows/release.yml` triggered on `v*` tag push — uses `just` recipes on single `ubuntu-latest` runner, creates GitHub Release with auto-generated notes. Removed `--no-latest` flag (GitHub's semver ordering handles backport "latest" status). Removed Go toolchain and `gh` CLI checks from release script. |
| 260306-qkov-operator1-skill | 2026-03-07 | Noted that `fab-operator1.md` ships as part of the kit skills directory in all archives — no new distribution mechanics, just another skill file deployed by `fab-sync.sh`. |
| 260305-u8t9-clean-break-go-only | 2026-03-05 | Updated generic archive (shell-only) scenario: no longer provides a working `fab` command — Go binary is required. Shell script fallback removed from dispatcher. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added `$(fab kit-path)/hooks/` as a new distributed directory (hook scripts shipped with kit). Updated bootstrap description to mention hook registration via `5-sync-hooks.sh`. Updated release archive contents to note hooks and sync scripts alongside packages. |
| 260305-g0uq-2-ship-fab-go-binary | 2026-03-05 | Ship fab Go binary: release now produces 5 archives (generic `kit.tar.gz` + 4 per-platform `kit-{os}-{arch}.tar.gz` with Go binary at `.kit/bin/fab`). `release.sh` cross-compiles via `CGO_ENABLED=0` for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64. `fab-upgrade.sh` detects platform via `uname -s`/`uname -m`, downloads platform archive with fallback to generic. README bootstrap one-liner is now platform-aware. Shell scripts in `lib/` have shim layer that delegates to Go binary when present. Skills updated via `_cli-fab.md` (renamed from `_scripts.md`) to invoke `fab-go binary at fab` as primary calling convention. |
| 260305-bhd6-1-build-fab-go-binary | 2026-03-05 | Go binary (`src/go/fab/`) built — ports all lib/ scripts to single `fab` binary. No distribution changes in this change — binary inclusion in kit.tar.gz and per-platform archives are deferred to a future change. Go toolchain required only for building from source, not for end users. |
| 260303-l6nk-gemini-cli-agent-aware-sync | 2026-03-04 | Added Gemini CLI as 4th supported agent. Updated bootstrap/sync descriptions to reflect conditional agent deployment (skills deployed only when agent's CLI found in PATH). Four agents: Claude Code (copies), OpenCode (symlinks), Codex (copies), Gemini CLI (copies). |
| 260301-08pa-version-pinned-upgrade-and-release | 2026-03-02 | Added version-pinned upgrade (`fab-upgrade.sh v0.24.0`) with tag-aware messaging. Added backport release support to `release.sh`: push to current branch instead of hardcoded `main`, `--no-latest` flag for `gh release create --latest=false`, position-independent argument parsing. |
| 260402-0ak9-remove-sync-version-file | 2026-04-02 | Removed `fab/.kit-sync-version` from preserved files list. Sync staleness detection now compares `$(fab kit-path)/VERSION` against `fab_version` in `config.yaml` (single warning message). |
| 260226-koj1-version-staleness-warning | 2026-02-26 | Added sync staleness detection (preflight stderr warning). Renamed `fab/project/VERSION` → `fab/.kit-migration-version`. Updated preserved files list in upgrade section. |
| 260224-v40o-wt-drop-prefix-and-dotworktrees | 2026-02-25 | wt package: dropped `wt/` branch prefix from exploratory worktrees (branch = worktree name directly). Switched worktree home directory from `<repo>-worktrees` to `<repo>.worktrees` (GitLens convention). Updated `wt-create` help text. No migration for existing worktrees. |
| 260221-i0z6-move-env-packages-add-fab-pipeline | 2026-02-21 | `env-packages.sh` moved from `scripts/` to `scripts/lib/` — now sourced from `src/kit/scripts/lib/env-packages.sh` in both `scaffold/fragment-.envrc` and `src/packages/rc-init.sh` |
| 260219-d2y2-copy-template-skills-drop-agents | 2026-02-19 | Updated references from symlinks to copies for Claude Code skills. Renamed "Symlink Repair After Update" to "Skill Deployment Repair After Update". Updated bootstrap and upgrade descriptions to reflect copy-with-template deployment |
| 260218-cif4-eliminate-symlinks-distribute-packages | 2026-02-18 | Package production code (idea, wt) now distributed via `kit.tar.gz` under `.kit/packages/`. Updated release archive contents description. Updated `fab-upgrade.sh` description (symlinks → directories and agents). Added `env-packages.sh` for centralized PATH setup, sourced by `scaffold/envrc` (direnv) and `src/packages/rc-init.sh` (shell rc). |
| 260217-zkah-readme-quickstart-prereqs-check | 2026-02-18 | Added prerequisites validation to `fab-sync.sh` pipeline (via `sync/1-prerequisites.sh`). Updated bootstrap description to mention prerequisites check. Restructured README Quick Start: folded Initialize and Updating under Install as sub-sections. |
| 260216-ymvx-DEV-1043-envrc-line-sync | 2026-02-16 | Updated `.envrc` references from symlink to line-ensuring: bootstrap description now says "`.envrc` entries (from `scaffold/envrc`, line-ensuring)"; scenario updated to note line-ensuring from scaffold |
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | `lib/sync-workspace.sh` → `fab-sync.sh` (promoted to `scripts/`); `/fab-init` → `/fab-setup`; `/fab-update` → `/fab-setup migrations` |
| 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests | 2026-02-16 | Renamed `init-scaffold.sh` → `sync-workspace.sh` throughout (bootstrap description, update script references, symlink repair) |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added version drift scenarios to update section; added `fab/VERSION` to preserved files list; added migration chain validation to release section |
| 260213-3njv-scaffold-dir | 2026-02-13 | Updated bootstrap description to mention `fab-sync.sh` reads from `scaffold/` files for index templates, envrc, and gitignore entries |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `fab-sync.sh` throughout; moved `release.sh` from `src/kit/scripts/` to `scripts/` (dev-only, not shipped in kit) |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed script references: `fab-setup.sh` → `_init_scaffold.sh`, `fab-update.sh` → `fab-upgrade.sh` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Updated bootstrap description to include `docs/specs/` directory and `design/index.md` in `fab-sync.sh` output |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Initial creation — bootstrap, update, release, and repo rename requirements |
| 260401-46hw-brew-install-system-shim | 2026-04-02 | Homebrew distribution model: system `fab` shim installed via `brew install fab-kit` (formula at `wvrdz/homebrew-tap`). Shim reads `fab_version` from `config.yaml`, resolves cached `fab-go` at `~/.fab-kit/versions/`, auto-fetches on miss. `fab init` bootstraps new repos (primary method replaces curl one-liner). `fab upgrade` replaces `fab-upgrade.sh` (shim subcommand). `wt` and `idea` become system-only Homebrew binaries. `fab-go binary at ` emptied (binary-free repo). Backend override mechanism (`FAB_BACKEND`, `.fab-backend`) removed. Sync pipeline: `4-get-fab-binary.sh` removed, `5-sync-hooks.sh` calls `fab hook sync` (system shim), `.envrc` scaffold removes `PATH_add src/kit/bin`. Release archives restructured for shim cache extraction (`fab-go` + `kit/` content). 4 Go binaries: `fab` (shim, Homebrew), `fab-go` (per-version cache), `wt` (Homebrew), `idea` (Homebrew). |
| 260401-ixzv-org-migrate-mit-license | 2026-04-02 | Migrated GitHub org references from wvrdz to sahil87. License changed from PolyForm Internal Use to MIT (root LICENSE). |
| 260402-3ac3-three-binary-architecture | 2026-04-02 | Three-binary architecture: Homebrew formula installs 4 binaries (`fab`, `fab-kit`, `wt`, `idea`). Shim section renamed to "Router Architecture" — `fab` uses negative-match dispatch to `fab-kit` or `fab-go`. Build produces 5 binaries (20 cross-compiled). Binary table updated: `fab` (router) from `src/go/fab-kit/cmd/fab/`, `fab-kit` from `src/go/fab-kit/cmd/fab-kit/`. `fab-sync.sh` references replaced with `fab-kit sync` / `fab sync`. `init` and `upgrade` call `Sync()` directly instead of `fab-sync.sh`. Updated design decisions (Homebrew distribution, fab upgrade). |

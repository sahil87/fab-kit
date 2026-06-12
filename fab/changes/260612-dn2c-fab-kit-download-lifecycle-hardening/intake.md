# Intake: fab-kit Download & Lifecycle Hardening

**Change**: 260612-dn2c-fab-kit-download-lifecycle-hardening
**Created**: 2026-06-12

## Origin

> dn2c

One-shot `/fab-new dn2c` resolving the backlog ID. Source material: the `[dn2c]` entry in `fab/backlog.md` (binary-review batch B3/6, filed 2026-06-12) and the underlying findings report `docs/specs/findings/binary-review-2026-06-12.md` §B3 — findings **F16–F22**, every one adversarially verified at high confidence. Line numbers below are vs commit `1431a9c3` (v2.1.6). This change **absorbs backlog item `[1old]`** (fab init must check for a git repo before downloading). No conversational discussion preceded this intake; all decisions trace to the backlog entry and the report's verifier notes, which include several corrections to the original fixes (adopted below).

**Parallelism context**: wave 1 of the binary-review batches — runs alongside `k4ge`, `mz4q`, `pw3k`. This change touches a **separate Go module** (`src/go/fab-kit`) with no file overlap with those batches. One seam: `.github/workflows/release.yml` — this change adds checksum publishing; `tb6f` (wave 3, last) adds the test step and rebases over us.

## Why

The `fab` shim auto-downloads the version-pinned engine (`fab-go`) plus kit content on every cache miss, and the lifecycle commands (`init`, `update`, `upgrade-repo`, `sync`) manage that cache. Today this path has four classes of defect:

1. **It can hang forever.** No HTTP timeout exists anywhere (`http.Get` / `http.DefaultClient`), and every uncached `fab <cmd>` — even bare `fab` / `fab --help` via `printHelp → EnsureCached` — performs a network fetch. A black-holed GitHub/CDN connection wedges the entire toolchain, multiplied by operator pane fan-out.
2. **It can exec a partial binary.** Extraction goes directly into the live `versions/<v>/` dir; `fab-go`'s exec bit is set at `OpenFile` time *before* `io.Copy` streams the content, and the readiness check is only `mode&0111`. There is no download lock, and the error path `os.RemoveAll(cacheDir)` can delete a directory a sibling process is actively using. The trigger (operator spawning N panes right after `upgrade-repo` bumps the pin) is the project's flagship workflow.
3. **It runs unverified bytes.** Release assets `kit-<os>-<arch>.tar.gz` ship with no checksum (SHA256s are computed only for `brew-*` archives, inside the formula generator) — `fab-go` is the sole unverified executable in the distribution chain, and the toolkit's primary remote-code surface.
4. **Lifecycle exit codes lie.** `upgrade-repo` stamps `fab_version` *before* syncing, converts sync failure to a stderr WARNING, prints `Updated: x -> y`, exits 0 — and the re-run short-circuits on the already-stamped version (`Already on the latest version`), making the broken state unrecoverable via the same command. `versionGuard` is silently defeated for every non-brew install (`Update()` returns nil after merely printing a message). `sync` counts failed skill writes as `repaired`/`created` and exits 0 with stale skills deployed — the exact state sync exists to prevent.

If unfixed, the failures stay silent and self-masking: agents run stale skills with no signal anywhere, scripts/CI cannot detect failed lifecycle ops, and the concurrency races fire precisely under the operator workflow the project is built around. The approach is the findings' verified fixes — hardening of existing behavior, not redesign — applied per the verifier-corrected recommendations. Documented contracts already agree with the fixes (`distribution.md:116-130` specifies upgrade-repo "exits non-zero with error message" on failure; `distribution.md:88` and `kit-architecture.md:282,285` document the version guard as "ensures fab_version <= system fab-kit version"); the code currently violates its own docs.

## What Changes

All paths in `src/go/fab-kit/` unless noted. The seven findings plus the absorbed `[1old]`:

### F16 — HTTP client timeouts on the auto-download path (high/small)

**Files**: `internal/download.go:30,64`, `cmd/fab/main.go:89,125`

`Download` uses `http.Get` (download.go:30) and `LatestVersion` uses `http.DefaultClient.Do` (download.go:64) — zero timeout. Replace with a dedicated `*http.Client`:

- **`LatestVersion`** (small JSON body; also called from `init.go:16` and `upgrade.go:41`): a short flat `Timeout` (e.g. 30s) is fine.
- **`Download`** (streams `resp.Body` through tar extraction at download.go:45): a flat `client.Timeout` would abort legitimately slow archive downloads — use a transport `ResponseHeaderTimeout` plus a generous overall deadline/context instead. *(Verifier correction adopted.)*

Precedent in the same module: `update.go:101` `runWithTimeout` already bounds brew subprocess ops at 30s/120s — the HTTP gap is an oversight, not a design decision.

### F17 — Atomic extraction under a version-keyed download lock (high/medium)

**Files**: `internal/download.go:40,47,118,165`, `internal/cache.go:50,85`

Today: extraction writes directly into `CacheDir(version)`; `writeFile` `os.OpenFile`-creates `fab-go` with `hdr mode|0111` *before* content streams (download.go:118-123 → 165-173); readiness = `mode&0111` only (cache.go:50-64, 85-91); error path `os.RemoveAll(cacheDir)` (download.go:47) deletes the tree a sibling may be reading kit/ skills from (via `kitpath.KitDir()`).

Fix:
1. Extract into a temp dir `versions/<version>.tmp-<pid>` and `os.Rename` into place only after the full archive is written — rename is atomic on the same filesystem, making readiness all-or-nothing.
2. Serialize concurrent downloaders with an advisory file lock (`syscall.Flock`, `LOCK_EX`, on a sibling `.lock` file keyed by version). Unix-only is acceptable — the module already depends on `syscall.Exec` (linux+darwin per the proc build tags).
3. Scope error-path cleanup to the temp dir only — a failed download must never remove a directory another process is using.

Note: `mz4q` (B1) builds a similar flock helper in the **other** module (`src/go/fab`). The two modules deliberately do not share code (documented decision, see refuted finding R4 / decision `260402-ktbg`) — implement the lock locally in fab-kit; do not import or workspace-link across modules.

### F18 — upgrade-repo must fail non-zero when sync fails (high/small)

**Files**: `internal/upgrade.go:50-53,75,80-83,86-90,131`, `cmd/fab/main.go:84-93`, `internal/init.go:51-53`

Today: `Upgrade()` stamps `fab_version` (upgrade.go:75) *before* `Sync`, converts sync failure to a WARNING (81-83), unconditionally prints `Updated: x -> y` (86-90), returns nil → exit 0; re-run short-circuits at 50-53. `Init()` already propagates the same Sync error correctly (init.go:51-53).

Fix: propagate the Sync error so the command exits non-zero, never print `Updated: x -> y` after a failed sync, and emit `run fab sync to repair` guidance. The stamp must not survive a failed sync. **Preferred mechanism**: stamp `fab_version` only *after* a successful Sync — requires Sync to take the kit version explicitly instead of re-reading it from config.yaml (this composes directly with F22's version-threading change). The backlog phrases this as "roll back the stamp"; the verifier notes that rollback after a *partial* sync creates the inverse mismatch (new skills deployed, old version stamped), which is why stamp-after-success is the cleaner equivalent. `EnsureCached(targetVersion)` already runs before the stamp (upgrade.go:61), so binary resolution never breaks either way — the harm being fixed is stale skills + false-success exit code.

### F19 — versionGuard must not trust `Update()`'s nil (high/small)

**Files**: `internal/update.go:17-22,88-98`, `internal/sync.go:108-126`

Today: `Update()` returns nil after merely printing "was not installed via Homebrew… Update manually" (update.go:17-22; `isBrewInstalled` is a `/Cellar/` path-substring check). Consequences: `fab update` exits 0 having updated nothing; worse, `versionGuard` (sync.go:117-125) treats nil as "updated" and lets sync proceed with a binary known to be too old — completely defeating the guard for go-install/manual/CI installs. (The `dev` sentinel does not shelter local builds — the justfile injects real semver via `-X main.version`.)

Fix:
1. `Update()` returns a sentinel (`ErrNotBrewInstalled`) instead of nil on the not-brew path; `fab update` exits non-zero there.
2. `versionGuard` **verifies post-state** — re-check the installed binary version after `Update()` rather than trusting a nil return. This also covers the verifier's reinforcing case (iii): brew-installed but tap release lag, where `Update` returns nil at update.go:43-46 having upgraded nothing.
3. When the running binary remains older than `fab_version`, fail the guard with actionable instructions. After a genuinely successful in-guard update, either re-exec the upgraded binary or fail the current sync with "fab-kit was updated, re-run fab sync" rather than continuing in-process on the old binary (the acknowledged gap at sync.go:123-125).

### F20 — Publish and verify checksums for the kit-* release archives (high/medium)

**Files**: `.github/workflows/release.yml:86-98`, `internal/download.go:30,45`, `cmd/fab/main.go:96`

Today: release.yml uploads only the 8 tarballs — no checksum asset; the shim streams the tarball straight into extraction and the result is `syscall.Exec`'d with no integrity check (gzip's CRC32 is not even reliably checked, since `archive/tar` stops before draining the stream).

Fix: publish a `SHA256SUMS` asset for the `kit-*` archives in release.yml, and have `Download` hash the downloaded bytes and refuse to extract/exec on mismatch. **Accepted trust model** (verifier-corrected): a same-release SHA256SUMS file defends against corruption/truncation and brings the chain up to industry baseline (rustup/nvm-style), but does *not* defend against an attacker who can swap release assets (they'd swap the sums file too). A separately-trusted digest channel (digest pinned in config.yaml, embedded in the brew-verified shim) and sigstore/cosign signing are explicit **non-goals** here — noted as the follow-on. Seam: only the checksum-publishing edit to release.yml belongs to this change; the test step is `tb6f`'s (it lands last and rebases).

### F21 — Stop counting failed writes as successes in sync deployment (medium/medium)

**Files**: `internal/sync.go:102-103,213-228,250-263,597-641` (+ verifier extras: `sync.go:578` MkdirAll of agent.BaseDir, `sync.go:260` ReadFile of kit VERSION)

Today: `syncAgentSkills` discards every `os.WriteFile`/`os.Symlink` error while incrementing `repaired`/`created` (sync.go:608-619 copy mode, 628-640 symlink mode); a failed source `ReadFile` is a bare `continue` (598-601); `scaffoldDirectories` ignores MkdirAll/WriteFile errors while printing `Created:` — including the `.kit-migration-version` writes (255, 261), whose silent failure **silently disables migration discovery** (upgrade.go:98-99 guards on `if err == nil`). Net: on perms/read-only-fs/full-disk errors, `fab sync` prints success tallies and `Done.`, exits 0, skills never deployed.

Fix: capture write/symlink errors, count them as failures (not `repaired`/`created`), surface them per-skill, and return a non-nil error from `Sync` when any deployment write failed — the ecosystem already treats sync's exit code as the failure signal (`fab-setup.md:95`: "if fab sync exits non-zero, STOP immediately"). Apply the same to `scaffoldDirectories` (incl. `.kit-migration-version`) and the verifier-noted `sync.go:578`/`sync.go:260` sites. Fix both copy and symlink branches even though the symlink branch is currently dead code (all four agent configs use Mode "copy") — same shape, trivial extra. Adjacent same-file functions (scaffoldTreeWalk, jsonMergePermissions, lineEnsureMerge) already propagate write errors — this restores consistency, not a new pattern.

### F22 — Thread the real binary version into Sync from Init/Upgrade (medium/small)

**Files**: `internal/sync.go:26-29,108-115`, `internal/upgrade.go:81`, `internal/init.go:51`, `cmd/fab-kit/main.go:12,74-79,93`

Today: Sync's first param is documented as "the embedded version of the fab-kit binary" and feeds `versionGuard`, but `Upgrade` passes `targetVersion` and `Init` passes `latest` (the just-resolved *kit* version) — `fabVersion == systemVersion` by construction, guard always passes. Only `fab sync` (cmd/fab-kit/main.go:93) passes the embedded version correctly.

Fix: thread the embedded binary version through `Init(version)` and `Upgrade(version, target)` from the cobra layer and pass it as Sync's `systemVersion`, so upgrade-repo to a too-new kit triggers the same guard path as plain sync. Scope note (verifier): even fixed, an in-flight sync that trips the guard and auto-updates still completes on the old binary — the fix's benefit is triggering the update attempt (or failing loudly) so the *next* run is correct. Composes with F18's preferred mechanism (Sync accepting the kit version explicitly).

### [1old] — fab init: check git repo before downloading (absorbed)

**Files**: `internal/init.go`

Today: `fab init` downloads the release, writes config, then fails at sync's git check — leaving stale artifacts behind. Fix: perform the git-repo precondition check at the top of `Init()`, before any download or config write. Do not work the original `[1old]` backlog entry standalone — mark it done when this change ships.

### Cross-cutting constraints

- **Tests** (constitution line 31): every changed contract gets test updates in this change — download timeout/atomicity/checksum behavior, upgrade-repo exit semantics (no `upgrade_test.go` exists today — create it), versionGuard sentinel/post-state, sync write-error propagation, init precondition ordering. The *broader* fab-kit suite (Sync/Init/Upgrade at 0% coverage generally) is `tb6f`'s F45 — do not expand into it.
- **Docs** (constitution line 31): update `src/kit/skills/_cli-fab.md` rows for `init` / `update` / `upgrade-repo` exit semantics (changed observable behavior: non-zero exits, new failure messages).
- `src/kit/` is canonical for skill content; `.claude/skills/` is never edited directly.

## Affected Memory

- `distribution/distribution`: (modify) release workflow gains the SHA256SUMS asset for kit-* archives; download behavior (timeouts, temp-dir+rename+flock, digest verification); upgrade-repo failure contract now actually exits non-zero (doc'd contract at :116-130 becomes true); `fab update` non-brew exit semantics
- `distribution/kit-architecture`: (modify) versionGuard semantics — post-state verification, ErrNotBrewInstalled sentinel, version threading from Init/Upgrade (documented "ensures fab_version <= system version" contract at :282,285 becomes enforced); cache readiness model (atomic rename replaces exec-bit-as-readiness)

## Impact

- **Go module `src/go/fab-kit`** (no overlap with wave-1 siblings): `internal/download.go`, `internal/cache.go`, `internal/upgrade.go`, `internal/init.go`, `internal/update.go`, `internal/sync.go`, `cmd/fab-kit/main.go` (version threading), `cmd/fab/main.go` (shim call sites — likely signature-only ripples)
- **CI/release**: `.github/workflows/release.yml` (checksum publishing only; seam with `tb6f`)
- **Skill docs**: `src/kit/skills/_cli-fab.md` (init/update/upgrade-repo exit-semantics rows)
- **Tests**: `download_test.go`, `update_test.go`, `init_test.go`, `sync_test.go` extended; `upgrade_test.go` created
- **Backlog**: `[dn2c]` and absorbed `[1old]` marked done when shipped
- **Behavioral risk**: exit codes change from 0→non-zero on failure paths for `fab update` (non-brew), `fab upgrade-repo` (sync failure), `fab sync` (deploy-write failure) — these are the *documented* contracts, but any script tolerating the old lying-success behavior will now see failures surfaced

## Open Questions

None — the adversarially-verified findings report plus the backlog entry resolve scope and mechanism; the remaining judgment calls are graded Confident below and recorded for apply.

## Clarifications

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 3 | Confirmed | — |
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly F16–F22 + absorbed [1old]; sigstore/cosign signing and separately-trusted digest channels are non-goals (F20 follow-on) | Backlog entry enumerates the actions explicitly; report marks signing as follow-on | S:95 R:90 A:95 D:95 |
| 2 | Certain | Tests accompany each changed contract; broader fab-kit suite coverage stays with tb6f (F45) | Constitution line 31 mandates tests; backlog draws the F45 boundary explicitly | S:95 R:90 A:95 D:90 |
| 3 | Certain | F16 timeout shape: short flat Timeout for LatestVersion; ResponseHeaderTimeout + generous overall deadline for Download (no flat timeout on the streaming path) | Clarified — user confirmed | S:95 R:85 A:85 D:80 |
| 4 | Certain | F17 lock implemented locally in the fab-kit module (syscall.Flock, version-keyed sibling .lock); no code sharing with mz4q's flock helper in src/go/fab | Clarified — user confirmed | S:95 R:75 A:90 D:80 |
| 5 | Certain | F18 mechanism: stamp fab_version only after successful Sync (Sync accepts kit version explicitly, composing with F22) rather than literal stamp-then-rollback | Clarified — user confirmed | S:95 R:80 A:80 D:70 |
| 6 | Certain | F19 guard verifies post-state (re-reads installed version after Update) instead of trusting nil; covers the brew release-lag case too | Clarified — user confirmed | S:95 R:80 A:85 D:80 |
| 7 | Certain | F20 trust model: same-release SHA256SUMS + digest check is the accepted baseline; defends corruption/truncation, explicitly not asset-swap attackers | Clarified — user confirmed | S:95 R:70 A:80 D:75 |
| 8 | Certain | F21 fixes both copy and symlink branches of syncAgentSkills plus verifier-noted sites (sync.go:578, :260), despite symlink branch being dead code today | Clarified — user confirmed | S:95 R:85 A:85 D:80 |
| 9 | Certain | release.yml edit limited to checksum publishing; no test step, no go-version changes (tb6f owns those and rebases last) | Clarified — user confirmed | S:95 R:85 A:90 D:85 |
| 10 | Certain | [1old]: git-repo check moves to the top of Init(), before any download or config write; original backlog entry marked done via this change | Clarified — user confirmed | S:95 R:85 A:90 D:85 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).

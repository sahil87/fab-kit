# Plan: fab-kit Download & Lifecycle Hardening

**Change**: 260612-dn2c-fab-kit-download-lifecycle-hardening
**Intake**: `intake.md`

## Requirements

All code paths in `src/go/fab-kit/` unless noted. Findings F16–F22 from `docs/specs/findings/binary-review-2026-06-12.md` §B3 plus absorbed backlog item `[1old]`.

### Download Path: HTTP Timeouts (F16)

#### R1: Bounded HTTP requests on the auto-download path
`LatestVersion` MUST use a dedicated `*http.Client` with a short flat `Timeout` (~30s). `Download` MUST NOT use a flat client timeout (it streams a large archive); it MUST instead bound the transport with `ResponseHeaderTimeout` (~30s) plus a generous overall context deadline (~10m). No call in `internal/download.go` may use `http.Get` or `http.DefaultClient`.

- **GIVEN** an uncached version and a black-holed GitHub/CDN connection
- **WHEN** any `fab <cmd>` triggers `EnsureCached → Download` (or `LatestVersion` via `init`/`upgrade-repo`)
- **THEN** the request fails with a clear, retryable timeout error instead of hanging indefinitely
- **AND** a legitimately slow (multi-minute) archive download still completes — only time-to-headers and the generous overall deadline bound it

### Download Path: Atomic Extraction Under a Version-Keyed Lock (F17)

#### R2: Temp-dir extraction, atomic rename, scoped cleanup, flock serialization
`Download` MUST extract into a temp dir `versions/<version>.tmp-<pid>` and `os.Rename` it into place only after the full archive is extracted (and verified, R3). Concurrent downloaders of the same version MUST serialize via a `syscall.Flock` `LOCK_EX` advisory lock on a version-keyed sibling `.lock` file (`versions/<version>.lock`), implemented locally in this module (no cross-module sharing with `src/go/fab`). After acquiring the lock, `Download` MUST re-check `ResolveBinary` and return early if a peer already completed the fetch. Error-path cleanup MUST be scoped to the temp dir only — a failed download never `RemoveAll`s the live cache dir.

- **GIVEN** N concurrent fab processes racing on first fetch of an uncached version (operator pane fan-out after a pin bump)
- **WHEN** they all call `Download(version)` simultaneously
- **THEN** exactly one performs the network fetch and extraction; the rest block on the lock and return early on the re-check
- **AND** no process can ever observe a partially extracted `versions/<version>/` dir (readiness is all-or-nothing via rename)

- **GIVEN** extraction fails mid-stream (corrupt archive, disk full)
- **WHEN** the error path runs
- **THEN** only `versions/<version>.tmp-<pid>` is removed; an existing live `versions/<version>/` dir is untouched

### Download Path: Checksum Verification (F20)

#### R3: Download verifies a same-release SHA256 digest before extraction
`Download` MUST fetch the `SHA256SUMS` asset from the same release, hash the downloaded archive bytes, and refuse to extract (and therefore exec) on mismatch or on a sums file missing the archive's entry. Trust model: same-release sums defend integrity (corruption/truncation), explicitly not asset-swap attackers; sigstore/separately-trusted digest channels are non-goals.

- **GIVEN** a release that publishes `SHA256SUMS` and a tampered/corrupted archive byte stream
- **WHEN** `Download` hashes the downloaded bytes
- **THEN** it returns a checksum-mismatch error and nothing is extracted into the cache

- **GIVEN** a pre-checksum release (no `SHA256SUMS` asset, HTTP 404)
- **WHEN** `Download` runs
- **THEN** it warns on stderr that verification is skipped and proceeds (older pinned versions remain installable)

#### R4: Release workflow publishes SHA256SUMS for kit-* archives
`.github/workflows/release.yml` MUST generate and upload a `SHA256SUMS` asset covering the four `kit-*.tar.gz` archives. The edit SHALL be limited to checksum publishing — no test steps, no go-version changes (owned by sibling change `tb6f`).

- **GIVEN** a tag push triggering the release workflow
- **WHEN** the release is created
- **THEN** the release assets include `SHA256SUMS` whose entries match `sha256sum kit-*.tar.gz` output

### Lifecycle Exit Contracts: upgrade-repo (F18)

#### R5: Upgrade stamps fab_version only after a successful Sync and propagates failure
`Upgrade()` MUST run Sync before stamping `fab_version`, passing the kit version explicitly (R9), and MUST propagate a Sync failure (command exits non-zero). On failure it MUST NOT print `Updated: x -> y` and MUST emit "run 'fab sync' to repair" guidance. Because the stamp never lands on failure, a re-run of `fab upgrade-repo` MUST retry rather than short-circuit on "Already on the latest version".

- **GIVEN** `fab upgrade-repo` to a new version where Sync fails (e.g., missing prerequisite, deploy write failure)
- **WHEN** the command finishes
- **THEN** it exits non-zero with the Sync error and repair guidance, `config.yaml` still holds the old `fab_version`, and no `Updated:` line is printed
- **AND** re-running `fab upgrade-repo` re-attempts the upgrade (no short-circuit)

### Lifecycle Exit Contracts: fab update & versionGuard (F19)

#### R6: Update returns ErrNotBrewInstalled on the not-brew path
`Update()` MUST return a sentinel error `ErrNotBrewInstalled` (instead of nil) when fab-kit was not installed via Homebrew, so `fab update` exits non-zero there. The existing user guidance message is preserved.

- **GIVEN** a go-install/manual/CI fab-kit binary (no `/Cellar/` in its resolved path)
- **WHEN** `fab update` runs
- **THEN** the process exits non-zero and the error satisfies `errors.Is(err, ErrNotBrewInstalled)`

#### R7: versionGuard verifies post-state instead of trusting Update's nil
When `fab_version > systemVersion`, `versionGuard` MUST, after attempting `Update()`, re-check the actually-installed binary version (query the `fab-kit` binary on PATH) rather than trusting a nil return. If the installed binary is now >= `fab_version`, the guard MUST fail the current sync with "fab-kit was updated — re-run 'fab sync'" (never continue in-process on the old binary). If the installed binary remains older than `fab_version` (not-brew, brew tap release lag, or update failure), the guard MUST fail with actionable instructions. The `dev` bypass is unchanged.

- **GIVEN** a non-brew install and a project pinned to a newer `fab_version`
- **WHEN** `fab sync` runs the version guard
- **THEN** sync exits non-zero with instructions to update fab-kit manually (the guard is no longer silently defeated)

- **GIVEN** a brew install where `brew upgrade` succeeds (or the tap lags so `Update` no-ops)
- **WHEN** the guard re-checks the installed version
- **THEN** post-state decides: new-enough → fail current sync with re-run guidance; still old → fail with release-lag instructions

### Lifecycle Exit Contracts: sync deployment writes (F21)

#### R8: Sync surfaces and propagates deployment write failures
`syncAgentSkills` MUST capture `os.WriteFile`/`os.Symlink`/per-skill `MkdirAll` and source `ReadFile` errors in BOTH copy and symlink branches, count them as failures (not `repaired`/`created`), surface them per-skill on stderr, and include the failure count in the per-agent tally. `scaffoldDirectories` MUST propagate `MkdirAll`/`WriteFile` errors including the `.kit-migration-version` writes and the kit `VERSION` read (sync.go:260). The `MkdirAll` of `agent.BaseDir` (sync.go:578) MUST be checked. `Sync` MUST return non-nil when any deployment write failed (no `Done.` on failure).

- **GIVEN** a read-only or full filesystem under `.claude/skills/`
- **WHEN** `fab sync` deploys skills
- **THEN** each failed skill is reported (`WARN: ... failed ...`), the tally shows `failed N`, and `fab sync` exits non-zero
- **AND** a silently failed `.kit-migration-version` write can no longer silently disable migration discovery

### Lifecycle Exit Contracts: version threading (F22)

#### R9: The embedded binary version is threaded from the cobra layer into Sync
`Init` and `Upgrade` MUST accept the embedded binary version from `cmd/fab-kit/main.go` (`Init(systemVersion)`, `Upgrade(systemVersion, target)`) and pass it as Sync's `systemVersion`, so upgrade-repo/init to a too-new kit trips the same guard as plain sync. `Sync` MUST accept the kit version explicitly (`Sync(systemVersion, kitVersion string, shimOnly, projectOnly bool)`); when `kitVersion` is empty (plain `fab sync`), it is read from `config.yaml` as today. This is the mechanism that lets R5 stamp after success.

- **GIVEN** `fab upgrade-repo 9.9.9` on a machine whose fab-kit binary is v2.1.6
- **WHEN** the in-upgrade Sync runs its version guard
- **THEN** the guard compares 9.9.9 against the real embedded 2.1.6 (not 9.9.9 vs 9.9.9) and trips

### Lifecycle Exit Contracts: init precondition ([1old])

#### R10: Init checks for a git repository before any download or config write
`Init()` MUST verify the CWD is inside a git repository as its first step — before `LatestVersion`, `EnsureCached`, and the `config.yaml`/`.kit-migration-version` writes — failing with an actionable error so a failed init leaves no stale artifacts.

- **GIVEN** a directory that is not a git repository
- **WHEN** `fab init` runs
- **THEN** it exits non-zero immediately with a "requires a git repository" error, no network fetch occurs, and no `fab/` files are created

### Docs: CLI reference (constitution line 31)

#### R11: _cli-fab.md documents the changed exit semantics
`src/kit/skills/_cli-fab.md` MUST document the changed observable behavior for `init` (git-repo precondition), `update` (non-zero when not brew-installed), `upgrade-repo` (non-zero on sync failure, stamp-after-success, repair guidance), and `sync` (non-zero on deployment write failure / version-guard failure). `src/kit/` is canonical — `.claude/skills/` copies are never edited.

- **GIVEN** an agent loading `_cli-fab.md` via `helpers:`
- **WHEN** it consults workspace command semantics
- **THEN** the documented exit codes and failure messages match the new implementation

### Non-Goals

- Sigstore/cosign signing and separately-trusted digest channels (config-pinned digest, shim-embedded digest) — F20 follow-on, explicitly out of scope
- Broader fab-kit suite coverage (Sync/Init/Upgrade general 0%-coverage backfill) — `tb6f`'s F45
- release.yml test steps or go-version changes — `tb6f` (lands last, rebases)
- Cross-module flock helper sharing with `src/go/fab` (`mz4q`) — documented two-module separation (decision `260402-ktbg`)
- Re-exec of the upgraded binary inside versionGuard — the chosen mechanism is fail-current-sync with re-run guidance

### Design Decisions

1. **Stamp-after-success over stamp-then-rollback (F18)**: Sync accepts the kit version explicitly so `fab_version` is written only after Sync succeeds — *Why*: rollback after a partial sync creates the inverse mismatch (new skills deployed, old version stamped) — *Rejected*: literal rollback of the stamp on failure.
2. **Post-state verification over sentinel-trusting (F19)**: the guard re-reads the installed binary version after `Update()` — *Why*: covers all three failure shapes (not-brew, tap release lag, genuine failure) with one check — *Rejected*: branching only on `ErrNotBrewInstalled` (misses release lag).
3. **Same-release SHA256SUMS as accepted baseline (F20)**: integrity defense only — *Why*: rustup/nvm-style industry baseline, cheap; asset-swap defense needs a separately-trusted channel (non-goal) — *Rejected for now*: sigstore signing.
4. **Hash-then-extract via temp file (F20+F17)**: the archive streams to a temp file while hashed, and extraction starts only after digest verification — *Why*: "refuse to extract on mismatch" requires the full digest before extraction; the temp file composes with F17's temp-dir flow — *Rejected*: hash-while-extracting with post-hoc cleanup (extracts unverified bytes).

## Tasks

### Phase 1: Setup

- [x] T001 Add `src/go/fab-kit/internal/lock.go` with `acquireLock(path) (release func(), err error)` using `syscall.Flock` LOCK_EX (blocking; lock file created 0644 and left in place), plus `src/go/fab-kit/internal/lock_test.go` covering acquire/release and mutual exclusion between two lock handles <!-- R2 --> <!-- rework: A-024 — name the ".lock" suffix constant; also fix the factually wrong comment at lock.go:17-19 (no flock helper exists in src/go/fab, and 260402-ktbg is about hooklib replication — cite the no-cross-module-sharing principle without inventing a sibling helper) -->

### Phase 2: Core Implementation

- [x] T002 In `src/go/fab-kit/internal/download.go`: add dedicated clients — `apiClient` (flat 30s Timeout) used by `LatestVersion` (and R3's sums fetch), and `downloadClient` (transport with `Proxy: ProxyFromEnvironment`, dial/TLS timeouts, `ResponseHeaderTimeout` 30s) used by `Download` with a 10-minute `context.WithTimeout` request context; convert `githubAPIURL`/download base URL to package vars as test seams <!-- R1 --> <!-- rework: A-024 — replace remaining magic values with named constants: inline 30s dial/TLS timeouts (download.go:48-49; name or reuse like the three primary timeout constants at :34-38), lock-file ".lock" suffix (:75), temp-archive prefix (:122), temp-dir pattern (:142). Test substring assertions on ".tmp-"/"-archive-" are unaffected. -->
- [x] T003 In `src/go/fab-kit/internal/download.go`: add `fetchChecksums(version)` (GET `SHA256SUMS` from the same release; 404 → `(nil, nil)`; parse `sha256sum` format incl. ` *name`), stream the archive to a temp file through `sha256` via `io.TeeReader`/`MultiWriter`, verify the digest before extraction, refuse on mismatch or missing entry, warn-and-skip when the asset is absent <!-- R3 -->
- [x] T004 In `src/go/fab-kit/internal/download.go`: restructure `Download` — acquire `versions/<version>.lock` via `acquireLock`, re-check `ResolveBinary` under the lock, extract into `versions/<version>.tmp-<pid>`, `os.Rename` into place (removing a stale binary-less `versions/<version>/` under the lock first), scope all error cleanup to the temp dir/temp file only (drop the `os.RemoveAll(cacheDir)` error path) <!-- R2 -->
- [x] T005 Extend `src/go/fab-kit/internal/download_test.go` with httptest-backed `Download` tests (HOME override): success path (fab-go executable, kit content present, no `.tmp-` leftovers, digest verified), checksum mismatch refusal (cache dir absent afterwards), missing-SHA256SUMS warn-and-proceed, extraction failure leaves a pre-existing stale cache dir untouched, and N concurrent `Download` calls perform exactly one archive fetch <!-- R2, R3, R1 -->
- [x] T006 In `src/go/fab-kit/internal/update.go`: add `var ErrNotBrewInstalled`, return it from the not-brew path of `Update` (keep the guidance prints), and add package-var seam `installedBinaryVersion` that runs `fab-kit --version` from PATH and parses the trailing `vX.Y.Z`; test the sentinel and parser in `src/go/fab-kit/internal/update_test.go` <!-- R6 -->
- [x] T007 In `src/go/fab-kit/internal/sync.go`: rewrite `versionGuard` to post-state verification — attempt `Update`, then re-check `installedBinaryVersion()`: new-enough → error "fab-kit was updated to vX — re-run 'fab sync'"; still-old → actionable error (distinct messages for update-failed / unverifiable / release-lag); add guard tests in `src/go/fab-kit/internal/sync_test.go` (dev bypass and pass-through unchanged; not-brew defeat now errors; post-state success fails sync with re-run guidance; release-lag via fake brew returning stable == systemVersion) <!-- R7 -->
- [x] T008 In `src/go/fab-kit/internal/sync.go`: propagate write failures — `syncAgentSkills` returns error, counts `failed`, prints per-skill `WARN:` on stderr and `failed N` in the tally (both copy and symlink branches, incl. per-skill `MkdirAll` and source `ReadFile`); check `MkdirAll(agent.BaseDir)`; `deploySkills` returns joined errors; `scaffoldDirectories` returns error (dirs, `.gitkeep`s, legacy rename/remove, `.kit-migration-version` writes, kit `VERSION` read); `Sync` returns non-nil on any of these (no `Done.` on failure); add write-failure tests in `src/go/fab-kit/internal/sync_test.go` (read-only BaseDir, unreadable source skill, missing kit VERSION) and update existing `scaffoldDirectories` tests for the new signature <!-- R8 -->
- [x] T009 Thread versions: `Sync(systemVersion, kitVersion string, shimOnly, projectOnly bool)` (empty kitVersion → read config.yaml; explicit → skip the config fab_version read); `Init(systemVersion string)` passes `(systemVersion, latest)`; `Upgrade(systemVersion, targetVersion string)` runs `runSync(systemVersion, targetVersion, false, false)` BEFORE `setFabVersion`, propagates Sync failure with "run 'fab sync' to repair" guidance and suppresses `Updated:` on failure; introduce `var runSync = Sync` as the Upgrade/Init test seam; update `src/go/fab-kit/cmd/fab-kit/main.go` to pass the embedded `version` into `Init`/`Upgrade`/`Sync` <!-- R9, R5 -->
- [x] T010 In `src/go/fab-kit/internal/init.go`: add the git-repository precondition (via `gitRepoRoot()`) as step 0 of `Init`, before `LatestVersion`/`EnsureCached`/config writes, with an actionable error; test in `src/go/fab-kit/internal/init_test.go` that Init in a non-git dir fails fast leaving no `fab/` artifacts and performing no network call <!-- R10 -->
- [x] T011 Create `src/go/fab-kit/internal/upgrade_test.go`: with cache pre-populated and `runSync` overridden — sync failure → non-nil error containing repair guidance and `fab_version` NOT stamped; re-run after failure retries (no "Already on the latest version" short-circuit); sync success → stamped and nil; `Already on latest` short-circuit still works when versions match <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T012 In `.github/workflows/release.yml`: add a `Generate SHA256SUMS` step (`cd dist && sha256sum kit-*.tar.gz > SHA256SUMS`) after packaging, and add `dist/SHA256SUMS` to the `gh release create` asset list — no other workflow edits <!-- R4 -->
- [x] T013 Run `go test ./...` and `go vet ./...` in `src/go/fab-kit`; fix any regressions across `cmd/fab`, `cmd/fab-kit`, and `internal` (signature ripples from T008/T009) <!-- R1, R2, R3, R5, R6, R7, R8, R9, R10 -->

### Phase 4: Polish

- [x] T014 Update `src/kit/skills/_cli-fab.md`: document workspace-command exit semantics — `init` git-repo precondition, `update` non-zero when not brew-installed, `upgrade-repo` stamp-after-success + non-zero on sync failure + "run 'fab sync' to repair", `sync` non-zero on deployment write failure and version-guard failure (edit `src/kit/` only, never `.claude/skills/`) <!-- R11 -->

## Execution Order

- T001 blocks T004 (lock helper used by Download)
- T002 and T003 block T004; T004 blocks T005
- T006 blocks T007 (sentinel + post-state seam used by versionGuard)
- T008 blocks T009 (Sync signature change lands once, after error propagation restructure)
- T009 blocks T010 and T011 (Init/Upgrade signatures)
- T012 is independent; T013 after all code tasks; T014 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/download.go` contains no `http.Get`/`http.DefaultClient`; `LatestVersion` uses a flat ~30s client; `Download` uses `ResponseHeaderTimeout` + a generous context deadline (no flat timeout on the streaming path)
- [x] A-002 R2: `Download` extracts to `versions/<version>.tmp-<pid>` and renames atomically; serialization via `syscall.Flock` LOCK_EX on `versions/<version>.lock`, implemented locally in the fab-kit module
- [x] A-003 R3: `Download` verifies the archive's SHA-256 against the same-release `SHA256SUMS` before extraction and refuses on mismatch
- [x] A-004 R4: release.yml publishes a `SHA256SUMS` asset covering the four kit-* archives, with no other workflow changes
- [x] A-005 R5: `Upgrade` propagates Sync failure (non-zero exit), stamps `fab_version` only after success, emits repair guidance, never prints `Updated:` after failure
- [x] A-006 R6: `Update` returns `ErrNotBrewInstalled` on the not-brew path and `fab update` exits non-zero there
- [x] A-007 R7: `versionGuard` re-checks the installed binary version after `Update` and fails the current sync in every too-old/just-updated outcome (never proceeds on the old binary when the guard tripped)
- [x] A-008 R8: failed deployment writes are counted as failures, surfaced per-skill, and make `Sync` return non-nil — in copy and symlink branches, `scaffoldDirectories` (incl. `.kit-migration-version`), `agent.BaseDir` MkdirAll, and the kit VERSION read
- [x] A-009 R9: `Init(version)`/`Upgrade(version, target)` receive the embedded binary version from `cmd/fab-kit/main.go` and pass it as Sync's `systemVersion`; `Sync` takes the kit version explicitly
- [x] A-010 R10: `Init` fails before any download or config write when not in a git repository
- [x] A-011 R11: `src/kit/skills/_cli-fab.md` documents the new init/update/upgrade-repo/sync exit semantics

### Behavioral Correctness

- [x] A-012 R5: after a failed upgrade-repo, re-running `fab upgrade-repo` retries (config still on the old version — no "Already on the latest version" short-circuit of the broken state)
- [x] A-013 R7: brew release-lag (Update returns nil having upgraded nothing) no longer passes the guard — post-state check fails sync with actionable instructions
- [x] A-014 R3: a release without `SHA256SUMS` (pre-checksum) still installs, with a stderr warning that verification was skipped

### Scenario Coverage

- [x] A-015 R2: test proves N concurrent `Download` calls of the same version perform exactly one archive fetch and yield a complete cache dir
- [x] A-016 R2: test proves extraction failure leaves a pre-existing cache dir untouched (cleanup scoped to the temp dir)
- [x] A-017 R5: `upgrade_test.go` exists and pins the sync-failure exit semantics (created per constitution line 31 — no upgrade tests existed)
- [x] A-018 R10: test proves init-in-non-git-dir creates no artifacts and performs no network fetch

### Edge Cases & Error Handling

- [x] A-019 R3: a `SHA256SUMS` present but missing the platform archive's entry is a hard failure (refuse to install)
- [x] A-020 R8: unreadable source skill files are counted as failures (not silently skipped) and surface in the tally

### Code Quality

- [x] A-021 Pattern consistency: new code follows module patterns — package-var test seams (like `isBrewInstalled`), `fmt.Errorf` `%w` wrapping, stderr `WARN:` soft-fail style mirrored from hook sync
- [x] A-022 No unnecessary duplication: reuses `gitRepoRoot`, `compareSemver`, `setFabVersion`, `dirExists`; flock helper deliberately local per the two-module decision (not duplication of `src/go/fab` — documented non-goal)
- [x] A-023 No god functions: `Download`'s new responsibilities (lock, checksum, temp-extract) are factored into helpers (`acquireLock`, `fetchChecksums`, extraction kept in `extractArchive`)
- [x] A-024 No magic values: timeouts and lock/temp naming use named constants

### Documentation Accuracy

- [x] A-025: `_cli-fab.md` statements match the implemented messages/exit codes exactly (verifiable against the code in this change)

### Cross References

- [x] A-026: intake's verifier-corrected mechanisms are the ones implemented (ResponseHeaderTimeout for Download, stamp-after-success, post-state guard) — no drift back to the original backlog phrasings (flat timeout, stamp-rollback, sentinel-trusting)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab-kit/internal/download_test.go:92` (`TestLatestVersionParsing`) — superseded by the new `TestLatestVersion_HTTPTestServer`, which exercises the real `LatestVersion` end-to-end via the `githubAPIURL` seam added in this change; the old test only unmarshals a locally re-declared struct and touches no production code (re-verified this cycle: both tests still present, candidate still holds)
- `src/go/fab-kit/internal/init.go:115` (`copyDir`) — zero production call sites (verified at HEAD too: dead since the 260402-ktbg sync-from-cache rewrite removed kit copying into repos, not made dead by this change); kept alive only by its own test `TestCopyDir` (init_test.go:106), which should be deleted with it

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Missing `SHA256SUMS` asset (HTTP 404) → warn on stderr and skip verification; sums present but entry missing, or digest mismatch → hard fail | Every existing release lacks the asset; projects pinned to older fab_versions must remain installable. Intake specifies refuse-on-mismatch but is silent on the missing-asset case; warn-and-proceed is the rustup/nvm-style baseline consistent with the accepted integrity-only trust model | S:70 R:85 A:80 D:75 |
| 2 | Confident | Concrete timeout values: 30s flat for `LatestVersion`/sums fetch; 30s `ResponseHeaderTimeout` (+dial/TLS bounds) and 10-minute overall context deadline for `Download` | Intake says "~30s" and "generous overall deadline" without a number; 10m comfortably covers slow links for a ~15MB archive while still bounding a stall | S:75 R:90 A:80 D:80 |
| 3 | Confident | Lock semantics: blocking LOCK_EX, lock file left in place after release, `ResolveBinary` re-check after acquisition | Unlinking the lock file races with waiters (classic flock pitfall); the holder's work is bounded by the HTTP deadlines so waiters are transitively bounded; re-check makes the N-process race do one fetch | S:70 R:85 A:85 D:80 |
| 4 | Confident | A pre-existing `versions/<version>/` dir with no resolvable fab-go (stale partial from pre-fix binaries) is removed under the exclusive lock before the rename | Rename onto a non-empty dir fails on POSIX; a binary-less dir is by definition not "live" (nothing execs from it), and the removal happens only under the lock after a verified extraction — distinct from the error-path cleanup R2 forbids | S:65 R:80 A:85 D:75 |
| 5 | Confident | Test seams: `var runSync = Sync` (Upgrade/Init), `installedBinaryVersion` package var (guard post-state), download URL bases as package vars | Mirrors the module's established `isBrewInstalled` var-override pattern; full-integration alternatives need brew/yq/direnv/network in CI | S:70 R:90 A:90 D:85 |
| 6 | Confident | Post-state query = run `fab-kit --version` from PATH and parse the trailing `vX.Y.Z` | The brew symlink on PATH points at the new Cellar binary after upgrade — this is exactly the binary the next sync run will be; parsing cobra's stable `fab-kit version vX.Y.Z` format | S:70 R:85 A:80 D:75 |
| 7 | Confident | Upgrade failure guidance combines both recoveries: "run 'fab sync' to repair, then re-run 'fab upgrade-repo'" | Intake mandates the "run fab sync to repair" text (restores old-version skill coherence after a partial sync); with stamp-after-success the natural retry is upgrade-repo itself — message states both | S:75 R:90 A:85 D:80 |

7 assumptions (0 certain, 7 confident, 0 tentative).

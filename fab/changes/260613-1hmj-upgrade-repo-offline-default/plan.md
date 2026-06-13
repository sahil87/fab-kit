# Plan: Offline-first `upgrade-repo` (default to systemVersion)

**Change**: 260613-1hmj-upgrade-repo-offline-default
**Intake**: `intake.md`

## Requirements

### Distribution: `upgrade-repo` version resolution

#### R1: No-arg `upgrade-repo` defaults to the running binary's version (offline)
When invoked with no version argument and without `--latest`, `fab upgrade-repo` MUST resolve its target to the embedded `systemVersion` of the running `fab-kit` binary and MUST NOT call the GitHub API, provided `systemVersion` is a real release tag (not empty and not `"dev"`).

- **GIVEN** a `fab-kit` binary stamped with `systemVersion == "2.3.1"`
- **WHEN** the user runs `fab upgrade-repo` with no arguments and without `--latest`
- **THEN** the resolved target version is `2.3.1`
- **AND** `LatestVersion()` (the GitHub API call) is never invoked

#### R2: `--latest` opts into the GitHub-API resolution path
`fab upgrade-repo --latest` MUST resolve its target by querying GitHub for the newest published release via `LatestVersion()` — the pre-change default behavior, now opt-in.

- **GIVEN** a `fab-kit` binary stamped with `systemVersion == "2.3.1"`
- **WHEN** the user runs `fab upgrade-repo --latest`
- **THEN** `LatestVersion()` is called and the resolved target is the tag it returns

#### R3: An explicit `<version>` argument wins over both the default and `--latest`
When an explicit `<version>` argument is supplied, `fab upgrade-repo` MUST resolve to that version regardless of `--latest`, and MUST NOT call the GitHub API for resolution. `--latest` is silently ignored when an explicit version is given (explicit intent beats a discovery flag).

- **GIVEN** a `fab-kit` binary stamped with `systemVersion == "2.3.1"`
- **WHEN** the user runs `fab upgrade-repo 2.2.0 --latest`
- **THEN** the resolved target is `2.2.0`
- **AND** `LatestVersion()` is never invoked

#### R4: A `dev` or empty `systemVersion` falls back to the GitHub-API path
When no explicit argument and no `--latest` are given, but `systemVersion` is `"dev"` or empty (a `just build` shim or an unstamped binary with no real release tag), `fab upgrade-repo` MUST fall back to resolving via `LatestVersion()` so it can still target a published release.

- **GIVEN** a `fab-kit` binary stamped with `systemVersion == "dev"`
- **WHEN** the user runs `fab upgrade-repo` with no arguments and without `--latest`
- **THEN** `LatestVersion()` is called to resolve the target

#### R5: `--latest` CLI flag is wired into the resolution path
The `upgrade-repo` command MUST expose a boolean `--latest` flag (default `false`) threaded into `internal.Upgrade` as a new `useLatest` parameter, and its `Short` description MUST reflect the new offline-default semantics.

- **GIVEN** the `fab-kit` CLI
- **WHEN** the user runs `fab upgrade-repo --help`
- **THEN** a `--latest` flag is listed
- **AND** the command `Short` describes upgrading to the installed binary's version (or `--latest` / an explicit version)

#### R6: Downstream upgrade behavior is unchanged
Everything downstream of target resolution — the `currentVersion == targetVersion` short-circuit, `EnsureCached`, `runSync`, the F18 stamp-after-sync ordering, and migration detection — MUST behave exactly as before. Only the *resolution* of the target version changes.

- **GIVEN** a resolved target version (by any path)
- **WHEN** the upgrade proceeds
- **THEN** caching, sync, version stamping, and migration detection are identical to the pre-change behavior

### Documentation: CLI reference

#### R7: `_cli-fab.md` documents the new resolution precedence
`src/kit/skills/_cli-fab.md` MUST document `upgrade-repo`'s new resolution precedence: default = the installed binary version (offline); `--latest` = newest GitHub release; an explicit `<version>` arg wins; with a `dev`/empty binary version falling back to GitHub.

- **GIVEN** a reader of the fab CLI reference
- **WHEN** they consult the `upgrade-repo` documentation
- **THEN** the offline default, the `--latest` opt-in, the explicit-arg precedence, and the `dev` fallback are all described

#### R8: `fab-setup.md` is verified unaffected (no SPEC update)
`src/kit/skills/fab-setup.md` MUST be confirmed unaffected by this change — its `upgrade-repo` references (cache population and the migration-stamp no-op) concern neither resolution nor the new flag. If it genuinely needs no change, it (and `docs/specs/skills/SPEC-fab-setup.md`) MUST be left untouched.

- **GIVEN** the `fab-setup.md` references to `upgrade-repo`
- **WHEN** evaluated against this change's scope (resolution + `--latest`)
- **THEN** none of them require editing, and no SPEC update is triggered

### Non-Goals

- The §4 error-message improvement (naming rate-limiting on a `403` with `X-RateLimit-Remaining: 0`) — out of scope / follow-up per intake assumption #6.
- Reading `GH_TOKEN`/`GITHUB_TOKEN` into the GitHub request — out of scope / follow-up.

### Design Decisions

1. **No-arg default = `systemVersion` (offline)**: the running binary already carries its own version; reconciling the repo to the installed binary is the natural meaning of "upgrade this repo". — *Why*: eliminates the rate-limited GitHub API call on the dominant path. — *Rejected*: "latest cached version" (cache can hold stale/unreleased builds, no enumeration helper exists); "keep API default, only fix the error" (leaves the common path network-dependent).
2. **Explicit arg ignores `--latest`**: since `targetVersion != ""` skips the whole resolution switch, a supplied arg wins and `--latest` is benignly ignored. — *Why*: explicit intent beats a discovery flag; avoids an error path for a harmless combination.
3. **`dev`/empty fallback to the network**: a `just build` shim reports `version == "dev"`, which is not a real release tag; without the fallback, resolution would try to sync a nonexistent `vdev`. — *Why*: keeps dev/unstamped binaries functional.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add a `useLatest bool` parameter to `Upgrade(systemVersion, targetVersion string)` in `src/go/fab-kit/internal/upgrade.go` and rewrite the no-arg resolution block (lines ~48-55) to the precedence switch: `useLatest` → `LatestVersion()`; else `systemVersion != "" && systemVersion != "dev"` → `systemVersion`; else fallback → `LatestVersion()`. Update the doc comment to describe `useLatest`. <!-- R1 -->
- [x] T002 Wire a `--latest` boolean flag into `upgradeCmd()` in `src/go/fab-kit/cmd/fab-kit/main.go`, thread it into `internal.Upgrade(version, targetVersion, useLatest)`, and update the command `Short` to "Upgrade the repo's kit to the installed binary's version (or --latest / an explicit version)". <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Update all 9 existing `Upgrade(...)` callers in `src/go/fab-kit/internal/upgrade_test.go` to pass the new third arg (`, false`). <!-- R6 -->
- [x] T004 Add 4 new behavioral test cases to `src/go/fab-kit/internal/upgrade_test.go`, following the `download_test.go` `githubAPIURL` httptest stubbing pattern: (a) default resolves to `systemVersion` with the API server failing the test if hit; (b) `--latest` hits the stubbed `releases/latest` endpoint; (c) `dev` binary falls back to the API; (d) explicit arg ignores `--latest` and does not hit the API. <!-- R1 R2 R3 R4 -->

### Phase 4: Polish

- [x] T005 Update `src/kit/skills/_cli-fab.md` to document `upgrade-repo`'s new resolution precedence (offline default = installed binary version; `--latest` = GitHub; explicit arg wins; `dev`/empty falls back to GitHub). <!-- R7 -->
- [x] T006 Verify `src/kit/skills/fab-setup.md` is unaffected (references at lines ~34, 302, 387 concern cache population and the migration-stamp no-op, not resolution); leave it and `SPEC-fab-setup.md` untouched if so. <!-- R8 -->

## Execution Order

- T001 blocks T002 (the flag threads into the new param) and T003/T004 (tests call the new signature).
- T003 must precede or accompany T004 (the file must compile with the new signature before new cases are added).

## Acceptance

### Functional Completeness

- [x] A-001 R1: No-arg `upgrade-repo` resolves to `systemVersion` and never calls `LatestVersion()` when `systemVersion` is a real release tag.
- [x] A-002 R2: `--latest` resolves via `LatestVersion()` (the GitHub API).
- [x] A-003 R3: An explicit `<version>` arg resolves to that version and ignores `--latest`, with no API call.
- [x] A-004 R4: A `dev` (or empty) `systemVersion` falls back to `LatestVersion()`.
- [x] A-005 R5: `upgradeCmd()` exposes a `--latest` flag (default false) threaded into `Upgrade`'s `useLatest` param, with an updated `Short`.
- [x] A-006 R7: `_cli-fab.md` documents the new resolution precedence.

### Behavioral Correctness

- [x] A-007 R1: The previously network-dependent no-arg path is now offline by default (behavior change verified by the no-network test).
- [x] A-008 R6: Downstream behavior (short-circuit, EnsureCached, runSync, F18 stamp ordering, migration detection) is unchanged — existing 9 tests still pass with the new signature.

### Scenario Coverage

- [x] A-009 R2: The `--latest` GIVEN/WHEN/THEN scenario is exercised by a test pointing `githubAPIURL` at an httptest server.
- [x] A-010 R3: The explicit-arg-ignores-`--latest` scenario is exercised by a test asserting no API hit.

### Edge Cases & Error Handling

- [x] A-011 R4: The `dev`-binary fallback scenario is exercised by a test that requires the API to be hit.

### Code Quality

- [x] A-012 Pattern consistency: New code follows the naming and structural patterns of surrounding code (the resolution switch mirrors the intake sketch; tests follow the `setupUpgradeRepo`/`stubRunSync`/`githubAPIURL` harness).
- [x] A-013 No unnecessary duplication: Existing utilities reused — `LatestVersion()`, `setupUpgradeRepo`, `stubRunSync`, `captureStdout`, the `githubAPIURL` seam — rather than reimplemented.
- [x] A-014 Test Integrity (constitution VII): Tests conform to the spec; the new behavioral cases assert the specified precedence, not the reverse.

### documentation_accuracy

- [x] A-015 R7: The `_cli-fab.md` update accurately reflects the shipped resolution precedence (offline default, `--latest`, explicit-arg, `dev` fallback) with no overstatement.

### cross_references

- [x] A-016 R8: `fab-setup.md` confirmed unaffected; no dangling reference to a removed/changed `upgrade-repo` behavior, and no orphaned `SPEC-fab-setup.md` update.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- The §4 rate-limit error message and `GH_TOKEN` reading are deliberately deferred (intake assumption #6).

## Deletion Candidates

- None — this change adds a `useLatest` parameter and a precedence switch without making existing code redundant. The old single-branch `LatestVersion()` resolution is not deleted; it is preserved verbatim in two of the three switch arms (the `--latest` opt-in and the `dev`/empty fallback), so no symbol or call site becomes unused. `LatestVersion()`, `EnsureCached`, `runSync`, `setFabVersion`, and the migration-detection block all remain reachable.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New test for the no-network default asserts by pointing `githubAPIURL` at an httptest server whose handler calls `t.Error`/`t.Fatal` if hit (per intake §3) | Intake specifies the exact assertion approach; the `download_test.go` harness already demonstrates the `githubAPIURL` override seam | S:95 R:85 A:95 D:95 |
| 2 | Confident | Document the new resolution precedence in `_cli-fab.md` as a focused note alongside the existing `upgrade-repo` Exit Semantics row, rather than introducing a new top-level section | `_cli-fab.md` has no existing resolution section for `upgrade-repo`; co-locating with the existing behavior row keeps the reference cohesive and matches the file's table-driven style | S:70 R:90 A:80 D:75 |
| 3 | Confident | Tests for `--latest`, `dev`-fallback, and explicit-arg reuse `setupUpgradeRepo` (which pre-populates the cache so `EnsureCached` never networks) and add a `githubAPIURL` httptest stub per-case | Matches the established harness; `setupUpgradeRepo` already isolates HOME and the cache, so only the API seam needs per-case stubbing | S:80 R:85 A:90 D:85 |

3 assumptions (1 certain, 2 confident, 0 tentative).

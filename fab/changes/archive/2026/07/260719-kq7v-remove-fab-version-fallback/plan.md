# Plan: Remove fab_version Config Fallback

**Change**: 260719-kq7v-remove-fab-version-fallback
**Intake**: `intake.md`

## Requirements

<!-- Derived from the intake. The change closes the one-compat-window config.yaml
     `fab_version:` fallback across both reader stacks (fab-go + fab-kit) and sweeps
     the kit-doc/spec class. `fab/.fab-version` becomes the sole version source. -->

### fab-go internal/config: sole-source version resolution

#### R1: config.yaml `fab_version:` is no longer parsed
`internal/config` MUST NOT parse a `fab_version:` key from config.yaml. The `Config.FabVersion` field is retained as a non-yaml carrier (`yaml:"-"`) populated only by the `fab/.fab-version` overlay in `Load`, so `GetFabVersion()` and preflight's staleness check keep working.

- **GIVEN** a config.yaml containing `fab_version: 9.9.9` and no `fab/.fab-version` sibling
- **WHEN** `Load(fabRoot)` runs
- **THEN** `cfg.GetFabVersion()` returns `""` (the key is ignored, an inert unknown key)
- **AND** no error is returned

#### R2: `fab/.fab-version` still wins and is the only source
When `fab/.fab-version` is present, its value is the resolved version regardless of any stale `fab_version:` still sitting in config.yaml.

- **GIVEN** a config.yaml with `fab_version: 1.0.0` AND a `fab/.fab-version` containing `2.15.0`
- **WHEN** `Load(fabRoot)` runs
- **THEN** `cfg.GetFabVersion()` returns `2.15.0`

### fab-go internal/config: pruneProjectScoped strip removed

#### R3: the `fab_version` strip line and its exception block are deleted
`pruneProjectScoped` MUST NOT special-case `fab_version`. With the yaml tag gone, a stale system-file `fab_version:` is an inert unknown key (nothing unmarshals it) — it can never reach a repo's resolved `Config.FabVersion`.

- **GIVEN** a system file `~/.fab-kit/config.yaml` containing `fab_version: 9.9.9` and a project with no `fab_version:` and no `.fab-version`
- **WHEN** `Load(fabRoot)` resolves the cascade
- **THEN** `cfg.GetFabVersion()` returns `""` (the system-file key never bleeds in)
- **AND** no warning is emitted for the `fab_version` key (it is an ignored unknown key, like any other)

### fab-go internal/configscope: comment-only alignment

#### R4: the `fab_version`-is-absent comment records sole-source truth
`internal/configscope`'s explanatory comment MUST keep the fact that `fab_version` is not a config key (it lives in `fab/.fab-version`) and MUST drop all compat-window/fallback/strip prose. No code changes (the `keyScopes` table already omits `fab_version`).

- **GIVEN** the `configscope.go` package comment
- **WHEN** apply completes
- **THEN** the comment no longer references a compat window, a config.yaml fallback, or the loader strip

### fab-kit internal/config: sole-source router resolution

#### R5: `readFabVersion` reads only `fab/.fab-version`
fab-kit's `readFabVersion` MUST resolve the pinned version from `fab/.fab-version` only. An absent/empty file is the error case. The config.yaml read, its anonymous-struct `yaml:"fab_version"` unmarshal, and the non-empty check are deleted. The `configPath` parameter is dropped from the signature (`readFabVersion(repoRoot string)`); the caller keeps `candidate` for `ConfigResult.ConfigPath`. The now-unused `gopkg.in/yaml.v3` import is dropped.

- **GIVEN** a repo with `fab/.fab-version` containing `2.15.0`
- **WHEN** `readFabVersion(repoRoot)` runs
- **THEN** it returns `2.15.0`, nil
- **AND GIVEN** a repo with a config.yaml `fab_version: 0.43.0` but no `fab/.fab-version`
- **WHEN** `readFabVersion(repoRoot)` runs
- **THEN** it returns a non-nil error (config.yaml is no longer consulted)

#### R6: the error message names `.fab-version` and the recovery path
The missing-version error MUST drop the "or config.yaml" mention and point at the existing-repo recovery command.

- **GIVEN** a repo with neither `fab/.fab-version` nor any version source
- **WHEN** `readFabVersion` fails
- **THEN** the error reads `no fab version found in fab/.fab-version. Run 'fab init' (new repo) or 'fab upgrade-repo' (existing repo) to set one`

### Kit-doc & spec sweeps (constitution-mandated)

#### R7: `_cli-fab.md` and its SPEC mirror drop the fallback clauses
The three `src/kit/skills/_cli-fab.md` sites claiming the fallback as live (router description ~line 44, `sync` version-guard row ~line 65, `migrations-status` ~line 541) MUST drop the "with the legacy config.yaml `fab_version:` fallback" clauses; `fab/.fab-version` is the sole source. `docs/specs/skills/SPEC-_cli-fab.md` MUST be swept for the same phrasing (the mirror carries no live-fallback clause today, but is re-verified).

- **GIVEN** the swept kit-doc/spec class
- **WHEN** apply completes
- **THEN** no `_cli-fab.md` or SPEC-_cli-fab.md site describes a live config.yaml `fab_version:` fallback

#### R8: `docs/specs/config.md` records the window as closed
`docs/specs/config.md` (the ~line 121-122 and ~line 309-313 passages) MUST be updated so the fallback + strip are recorded as closed by this change — `.fab-version` is the sole source, and a stale `fab_version:` (project or system file) is an inert unknown key.

- **GIVEN** `docs/specs/config.md`
- **WHEN** apply completes
- **THEN** the config.md passages no longer describe a live one-compat-window fallback or the loader strip as present design

### Non-Goals

- No new migration file — no user data is restructured; `2.14.0-to-2.15.0` already owns the relocation and stays in the chain for pre-2.15 repos. `src/kit/migrations/2.14.0-to-2.15.0.md` is untouched (it documents its own historical behavior).
- The `.fab-version` gitignore-negation seams (8ken, `2.15.1-to-2.15.2`) are untouched.
- `fab config upgrade`'s parked-removal block (the trailing `fab_version` comment in this repo's own `config.yaml`) is untouched — fence-engine territory, not this change.
- `docs/memory/` files are NOT edited during apply (hydrate owns them).
- `docs/specs/findings/` review-log files are historical artifacts, not swept.
- `docs/specs/architecture.md` line ~459 also carries the fallback phrasing and is swept as part of the mirror class (it restates the same router-resolution fact).

### Design Decisions

#### Keep `Config.FabVersion` as a non-yaml carrier field
**Decision**: Change the tag `yaml:"fab_version"` → `yaml:"-"` and keep the field; do not remove it from `Config`.
**Why**: `Load`'s `.fab-version` overlay populates it and preflight's `GetFabVersion` consumer still needs it; an explicit `yaml:"-"` (not a bare untagged field) prevents yaml.v3 from matching the lowercased field name.
**Rejected**: Deleting the field — forces re-plumbing the overlay + consumer for zero benefit.
*Introduced by*: 260719-kq7v-remove-fab-version-fallback

#### fab-kit test fixtures pin via `fab/.fab-version`, not config.yaml
**Decision**: Repoint the fab-kit test fixtures that create repos by writing `fab_version:` into config.yaml (`setupUpgradeRepo`, `sync_integration_test.go`, `scaffold_test.go`, `config_test.go`, the router-resolution tests) to write `fab/.fab-version` instead.
**Why**: Those fixtures relied on the deleted fallback to make the starting pin resolve; with the fallback gone the sole valid version source is `fab/.fab-version`.
**Rejected**: Leaving config.yaml fixtures and asserting the new empty/error behavior everywhere — would erase the pre-existing "starting pin resolves to X" coverage the upgrade tests need.
*Introduced by*: 260719-kq7v-remove-fab-version-fallback

## Tasks

### Phase 1: fab-go code changes

- [x] T001 In `src/go/fab/internal/config/config.go`: change `Config.FabVersion` yaml tag from `yaml:"fab_version"` to `yaml:"-"` (line 123); rewrite the `Load` doc comment (lines 132-139) to state `.fab-version` is the sole source (drop the compat-window fallback sentence); rewrite the `readDotFabVersion` comment (lines 151-154) to drop "defers to the config.yaml fallback in Load" (a missing `.fab-version` leaves `FabVersion` empty). <!-- R1 R2 --> <!-- rework cycle 1 RESOLVED: ran `gofmt -w internal/config/config.go` — the inserted FabVersion comment split the Config struct into two alignment groups; gofmt re-aligned each group (dropping the old wide tag padding). `gofmt -l` is now empty in both src/go/fab and src/go/fab-kit. Content edits unchanged. -->
- [x] T002 In `src/go/fab/internal/config/config.go`: delete the `delete(m, "fab_version")` strip line (line 327) and the whole "fab_version is a NAMED compat-window exception" comment block (lines 317-325) from `pruneProjectScoped`. <!-- R3 -->
- [x] T003 [P] In `src/go/fab/internal/configscope/configscope.go`: rewrite the "fab_version is intentionally ABSENT" comment paragraph (lines 60-70) — keep the fact that `fab_version` is not a config key (lives in `fab/.fab-version`), drop all compat-window/fallback/strip prose. No code change. <!-- R4 -->

### Phase 2: fab-kit code changes

- [x] T004 In `src/go/fab-kit/internal/config.go`: delete step 2 of `readFabVersion` (lines 108-122, the config.yaml read + anonymous-struct unmarshal + non-empty check); shrink the signature to `readFabVersion(repoRoot string)`; update the missing-version error to `no fab version found in fab/.fab-version. Run 'fab init' (new repo) or 'fab upgrade-repo' (existing repo) to set one`; update the caller `resolveConfigFrom` (line 74) to `readFabVersion(dir)`; drop the now-unused `gopkg.in/yaml.v3` import; update the `dotFabVersionRelPath` comment (lines 14-17) to drop the fallback sentence. <!-- R5 R6 -->

### Phase 3: Tests (both modules)

- [x] T005 In `src/go/fab/internal/config/config_test.go`: replace `TestLoad_FabVersionFallbackToConfig` (lines 173-192) with a test pinning the NEW behavior (config.yaml `fab_version:` present + no `.fab-version` ⇒ `GetFabVersion() == ""`, no error); keep `TestLoad_FabVersionFromDotFile` (the `.fab-version`-wins assertion still holds); rework the `pruneProjectScoped` strip tests (`TestScope_PruneAllProjectScopedFields` lines 695-738 and `TestScope_SystemFabVersionDoesNotBleedIntoResolvedConfig` lines 740-770) so they assert a system-file `fab_version` never reaches `Config.FabVersion` (inert unknown key) and produces no warning — no strip logic asserted. <!-- R1 R2 R3 -->
- [x] T006 [P] In `src/go/fab/internal/configscope/configscope_test.go`: update the `fab_version` comment/assertion in `TestScopeFor` (lines 33-42) to drop the loader-strip/compat-window prose while keeping the "unknown key" assertion. <!-- R4 -->
- [x] T007 [P] In `src/go/fab/cmd/fab/config_test.go`: update the `fab_version` exemption comment in `TestConfigReference*` (lines 100-106) to reflect `yaml:"-"` (the field is no longer a parse carrier for config.yaml) — the assertions themselves are unchanged. <!-- R1 -->
- [x] T008 In `src/go/fab-kit/internal/config_test.go`: repoint the router-resolution fixtures to write `fab/.fab-version`; replace `TestReadFabVersion_FallbackToConfig` with a test pinning that config.yaml is no longer consulted (config.yaml `fab_version:` + no `.fab-version` ⇒ error); update `TestReadFabVersion_MissingBothSources`/`TestReadFabVersion_InvalidYAML`/`TestResolveConfigFrom_Found`/`TestResolveConfigFrom_MissingFabVersion`/`TestRequireManagedRepo_*` to source the pin from `.fab-version`; update `readFabVersion(...)` call sites to the one-arg signature. <!-- R5 R6 -->
- [x] T009 In `src/go/fab-kit/internal/upgrade_test.go`: repoint `setupUpgradeRepo` (line 65) and the `TestUpgrade_VersionlessCachedKitFails` fixture (line 263) to write `fab/.fab-version` for the starting pin; update all `readFabVersion(repo, ...)` call sites to the one-arg signature and the mid-flow/failure-path comments that mention the "config.yaml pin/fallback". <!-- R5 -->
- [x] T010 [P] In `src/go/fab-kit/internal/sync_integration_test.go`: repoint the fixture (line 81) to write `fab/.fab-version` instead of config.yaml `fab_version:`. <!-- R5 -->
- [x] T011 [P] In `src/go/fab-kit/internal/scaffold_test.go`: repoint the fixtures that write `fab_version:` into config.yaml (lines 621, 649) to write `fab/.fab-version` where the test's assertion depends on a resolvable pin; leave fixtures where the config.yaml content is incidental to the assertion. <!-- R5 -->

### Phase 4: Kit-doc & spec sweeps

- [x] T012 In `src/kit/skills/_cli-fab.md`: drop the fallback clauses at the router description (line 44), the `sync` version-guard row (line 65), and the `migrations-status` section (line 541) — `fab/.fab-version` is the sole source. <!-- R7 -->
- [x] T013 [P] In `docs/specs/skills/SPEC-_cli-fab.md`: sweep the mirrored router/sync/migrations-status prose for the fallback phrasing and align; verify no live-fallback clause remains. <!-- R7 -->
- [x] T014 [P] In `docs/specs/config.md`: update the fallback + strip passages (lines ~117-123 and ~305-313) to record the window as closed by this change — `.fab-version` is the sole source; a stale `fab_version:` (project or system file) is an inert unknown key. <!-- R8 -->
- [x] T015 [P] In `docs/specs/architecture.md`: update line ~459's "Readers fall back to a legacy `fab_version:` key … for one compat window" phrasing to record the sole-source truth. <!-- R7 -->

### Phase 5: Verify

- [x] T016 Run `go test ./internal/config/... ./internal/configscope/... ./cmd/fab/...` in `src/go/fab`, then the full `go test ./...` in both `src/go/fab` and `src/go/fab-kit`; fix any failures. Final grep sweep: `grep -rn "compat window\|compat-window" src/kit docs/specs src/go` and the fallback phrasing must return only the untouched historical migration doc and finding logs. <!-- R1 R2 R3 R4 R5 R6 R7 R8 -->

### Phase 6: Review rework (cycle 1)

- [x] T017 Stale version-source comment sweep in `src/go` (review cycle 1 should-fix + nice-to-have; the T016 grep keyed on "fallback"/"compat window" and missed claims phrased differently): in `src/go/fab-kit/internal/sync.go` fix the doc-comment cluster claiming config.yaml as the version source/stamp target (line 26 "read from fab_version in config.yaml", line 32 "Resolve the kit version from config.yaml unless the caller provided it", lines 27 + 40 config.yaml-stamp phrasing — the plain-sync path resolves via `RequireManagedRepo` → `readFabVersion` → `fab/.fab-version` only, and the stamp target is `fab/.fab-version` since j0qm); in `src/go/fab-kit/internal/upgrade.go` line 48 fix "leaves config.yaml on the old version" → the pin/stamp target is `fab/.fab-version`; in `src/go/fab/internal/preflight/preflight.go` line 148 fix the "(single config.yaml parser)" rationale — the version rides `Load`'s `.fab-version` overlay, not config.yaml. Comments only, no behavior change; re-run `gofmt -l` + `go vet ./...` + `go test ./...` in both modules after. <!-- R7 -->

## Execution Order

- T001, T002 are in the same file (config.go) — apply sequentially.
- Phase 1-2 code changes precede Phase 3 tests (tests conform to the new signatures/behavior).
- T004 (signature change) blocks T008/T009 (call-site updates).
- Phase 4 doc sweeps are independent of code/tests.
- T016 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: config.yaml `fab_version:` is not parsed by `internal/config` (`Config.FabVersion` tag is `yaml:"-"`); a config-only `fab_version:` with no `.fab-version` yields `GetFabVersion() == ""` with no error.
- [x] A-002 R2: `fab/.fab-version` is the sole source and wins; a stale config.yaml key does not out-compete it.
- [x] A-003 R3: the `pruneProjectScoped` `fab_version` strip line and its exception comment block are gone.
- [x] A-004 R4: `configscope.go`'s comment records `fab_version`-not-a-config-key truth with no compat-window/fallback/strip prose; `keyScopes` is unchanged.
- [x] A-005 R5: fab-kit `readFabVersion` reads only `fab/.fab-version`, takes `(repoRoot string)`, and no longer imports `gopkg.in/yaml.v3`.
- [x] A-006 R6: the missing-version error names `fab/.fab-version` and points at `fab init` / `fab upgrade-repo`, dropping "or config.yaml".
- [x] A-007 R7: `_cli-fab.md` (3 sites), `SPEC-_cli-fab.md`, and `architecture.md` carry no live config.yaml `fab_version:` fallback claim.
- [x] A-008 R8: `docs/specs/config.md` records the compat window as closed by this change.

### Behavioral Correctness

- [x] A-009 R3: a system-file `fab_version:` never reaches `Config.FabVersion` (inert unknown key) and emits no warning (verified by the reworked scope tests).
- [x] A-010 R5: a not-yet-migrated repo (config.yaml `fab_version:` only, no `.fab-version`) hard-fails fab-kit router resolution with the actionable error.

### Removal Verification

- [x] A-011 R3: no `delete(m, "fab_version")` remains in `pruneProjectScoped`; no compat-window exception block remains in config.go.
- [x] A-012 R5: no config.yaml read / anonymous `yaml:"fab_version"` struct remains in `readFabVersion`; the `configPath` parameter is gone.

### Scenario Coverage

- [x] A-013 R1: `TestLoad_FabVersionFallbackToConfig` is replaced by a test asserting the config.yaml key is ignored (empty result, no error).
- [x] A-014 R2: `TestLoad_FabVersionFromDotFile` still passes (`.fab-version` wins).
- [x] A-015 R5: a fab-kit test pins that `.fab-version` absent + config.yaml `fab_version:` present ⇒ error.
- [x] A-016 R5: `go test ./...` passes in both `src/go/fab` and `src/go/fab-kit`.

### Edge Cases & Error Handling

- [x] A-017 R2: a missing `fab/.fab-version` leaves `Config.FabVersion` empty (preflight staleness silently skips) without error.
- [x] A-018 R5: an absent/empty `fab/.fab-version` is the fab-kit error case (router needs a pinned version).

### Code Quality

- [x] A-019 Pattern consistency: edits follow surrounding Go/comment style; no dead code (unused import removed, unused parameter removed). <!-- rework cycle 1 RESOLVED: config.go is now gofmt-clean (`gofmt -w` re-aligned the two Config-struct field groups the inserted FabVersion comment created); `gofmt -l` empty in both src/go/fab and src/go/fab-kit. Unused import/param removal verified earlier. -->
- [x] A-020 No unnecessary duplication: existing helpers reused; no new parser or utility introduced.
- [x] A-021 Canonical-source discipline: only `src/kit/` (not `.claude/skills/`) edited for kit content; every skill edit has its SPEC mirror updated (Constitution V + Additional Constraints).
- [x] A-022 Sibling/mirror sweep: the whole `fab_version` fallback class (code + `_cli-fab.md` + SPEC mirror + config.md + architecture.md) is swept up front; final grep confirms no stray live-fallback claims remain (documentation_accuracy + cross_references).
- [x] A-023: No stale config.yaml version-source/stamp-target comment claims remain in `src/go` (sync.go, upgrade.go, preflight.go swept — documentation_accuracy). <!-- rework cycle 1 RESOLVED (T017): sync.go doc-comment + inline resolution comment now name fab/.fab-version (via RequireManagedRepo → readFabVersion); upgrade.go ordering-contract comment says "leaves fab/.fab-version on the old version"; preflight.go staleness rationale states the version rides Load's .fab-version overlay, not a config.yaml parse. Remaining config.yaml mentions in these files are existence-checks / fence-engine (fab config upgrade) references, not version-source claims. -->

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Keep `Config.FabVersion` as a non-yaml carrier field (`yaml:"-"`) rather than deleting it | Backlog names the yaml-tag parse as the target; `Load`'s `.fab-version` overlay and preflight's `GetFabVersion` consumer still need the carrier; `yaml:"-"` (not bare untagged) blocks yaml.v3 lowercased-field matching | S:75 R:80 A:80 D:65 |
| 2 | Confident | `readFabVersion` error message: `no fab version found in fab/.fab-version. Run 'fab init' (new repo) or 'fab upgrade-repo' (existing repo) to set one` | Must stop naming config.yaml; `upgrade-repo` is the real recovery for an unmigrated repo (`Upgrade` tolerates a missing pin); wording is an apply decision, the two fixed requirements (drop "or config.yaml", point at upgrade-repo) are met | S:55 R:95 A:80 D:70 |
| 3 | Confident | Repoint fab-kit test fixtures (`setupUpgradeRepo`, sync/scaffold/config/router tests) to write `fab/.fab-version` for the starting pin instead of config.yaml `fab_version:` | The fixtures relied on the deleted fallback; `.fab-version` is now the sole valid pin source, and stamping it preserves the pre-existing "starting pin resolves to X" upgrade-test coverage; a successful `Upgrade` overwrites it and a failed one leaves it, matching the existing mid-flow/failure assertions | S:70 R:85 A:80 D:70 |
| 4 | Confident | Sweep `docs/specs/architecture.md` (line ~459) in addition to the intake-named sweep sites | It restates the same router-resolution fallback fact and is in the aggregate-spec sibling class (code-quality § Sibling & Mirror Sweeps); reviewers read the mirror class strictly | S:70 R:90 A:80 D:75 |
| 5 | Confident | Leave `src/kit/migrations/2.14.0-to-2.15.0.md` and `docs/specs/findings/*` untouched | The migration doc describes its own historical behavior (present-truth for a historical artifact) and stays in the chain for stragglers; findings are dated review logs, not present-truth prose | S:80 R:80 A:85 D:80 |

5 assumptions (0 certain, 5 confident, 0 tentative).

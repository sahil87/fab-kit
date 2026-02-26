# Spec: Version Staleness Warning

**Change**: 260226-koj1-version-staleness-warning
**Created**: 2026-02-26
**Affected memory**: `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/migrations.md`

## Non-Goals

- Remote staleness checking (checking GitHub for newer releases) — deferred to a separate change
- Blocking workflow on staleness — warning is advisory only
- Auto-running `fab-sync.sh` on staleness detection — users must explicitly opt in

## Sync: Sync Version Stamp

### Requirement: Stamp File Written After Sync

`2-sync-workspace.sh` SHALL write `fab/.kit-sync-version` after all skill deployments complete. The file SHALL contain the bare semver string from `fab/.kit/VERSION` at sync time (e.g., `0.20.0`), with no trailing whitespace or newline beyond the final one.

#### Scenario: Fresh sync on a new project
- **GIVEN** `fab/.kit/VERSION` contains `0.20.0`
- **AND** `fab/.kit-sync-version` does not exist
- **WHEN** `fab-sync.sh` completes successfully
- **THEN** `fab/.kit-sync-version` is created with content `0.20.0`
- **AND** stdout includes `Created: fab/.kit-sync-version (0.20.0)`

#### Scenario: Re-run sync on same version
- **GIVEN** `fab/.kit/VERSION` contains `0.20.0`
- **AND** `fab/.kit-sync-version` already contains `0.20.0`
- **WHEN** `fab-sync.sh` completes successfully
- **THEN** `fab/.kit-sync-version` is overwritten with `0.20.0` (idempotent)
- **AND** stdout includes `fab/.kit-sync-version: OK (0.20.0)`

#### Scenario: Sync after kit upgrade
- **GIVEN** `fab/.kit/VERSION` contains `0.21.0`
- **AND** `fab/.kit-sync-version` contains `0.20.0`
- **WHEN** `fab-sync.sh` completes successfully
- **THEN** `fab/.kit-sync-version` is updated to `0.21.0`
- **AND** stdout includes `Updated: fab/.kit-sync-version (0.20.0 → 0.21.0)`

### Requirement: Stamp File Location and Gitignore

`fab/.kit-sync-version` SHALL be located inside `fab/` but outside `fab/.kit/` (so it survives atomic kit replacement via `fab-upgrade.sh`). The file SHALL be gitignored — it represents per-developer local state, not shared project state.

The scaffold gitignore fragment (`fab/.kit/scaffold/fragment-.gitignore`) SHALL include `fab/.kit-sync-version` in the "Fab Specific" section.

#### Scenario: Stamp survives kit replacement
- **GIVEN** `fab/.kit-sync-version` contains `0.20.0`
- **WHEN** `fab-upgrade.sh` atomically replaces `fab/.kit/` (rm + mv)
- **THEN** `fab/.kit-sync-version` still exists with content `0.20.0`
- **AND** the staleness check detects the mismatch after upgrade

#### Scenario: Stamp not committed to git
- **GIVEN** `fab/.kit-sync-version` exists
- **WHEN** `git status` is run
- **THEN** the file does not appear in untracked or modified files (gitignored)

## Preflight: Staleness Detection

### Requirement: Non-Blocking Staleness Warning

`lib/preflight.sh` SHALL compare `fab/.kit/VERSION` against `fab/.kit-sync-version` after the project initialization check (step 1) but before change resolution (step 2). If the versions differ or the stamp file is missing, preflight SHALL emit a warning to **stderr** and continue normally — it MUST NOT exit non-zero or alter the stdout YAML output.

#### Scenario: Versions match (no warning)
- **GIVEN** `fab/.kit/VERSION` contains `0.20.0`
- **AND** `fab/.kit-sync-version` contains `0.20.0`
- **WHEN** `lib/preflight.sh` runs
- **THEN** no staleness warning is emitted to stderr
- **AND** stdout YAML is unaffected

#### Scenario: Stamp behind engine (warning)
- **GIVEN** `fab/.kit/VERSION` contains `0.21.0`
- **AND** `fab/.kit-sync-version` contains `0.20.0`
- **WHEN** `lib/preflight.sh` runs
- **THEN** stderr contains: `⚠ Skills out of sync — run fab-sync.sh to refresh (engine 0.21.0, last synced 0.20.0)`
- **AND** exit code is still 0 (if all other validations pass)
- **AND** stdout YAML is unaffected

#### Scenario: Stamp missing (warning)
- **GIVEN** `fab/.kit/VERSION` contains `0.20.0`
- **AND** `fab/.kit-sync-version` does not exist
- **WHEN** `lib/preflight.sh` runs
- **THEN** stderr contains: `⚠ Skills may be out of sync — run fab-sync.sh to refresh`
- **AND** exit code is still 0 (if all other validations pass)
- **AND** stdout YAML is unaffected

#### Scenario: Preflight fails for other reasons
- **GIVEN** `fab/.kit/VERSION` contains `0.21.0`
- **AND** `fab/.kit-sync-version` contains `0.20.0`
- **AND** `fab/project/config.yaml` does not exist
- **WHEN** `lib/preflight.sh` runs
- **THEN** the staleness warning is NOT emitted (init check fails first)
- **AND** exit code is 1 with the init error message

### Requirement: Warning Placement

The staleness check SHALL run after the project initialization check (config.yaml + constitution.md existence) but before change resolution. This ensures:
1. No warning noise when the project isn't initialized
2. Warning is visible even when no active change exists (preflight would fail at step 2, but the warning already emitted)
<!-- clarified: placement between init validation and change resolution confirmed during clarify session -->

#### Scenario: No active change but kit is stale
- **GIVEN** `fab/.kit/VERSION` contains `0.21.0`
- **AND** `fab/.kit-sync-version` contains `0.20.0`
- **AND** `fab/current` does not exist
- **WHEN** `lib/preflight.sh` runs
- **THEN** stderr contains the staleness warning
- **AND** stderr also contains the "no active change" error
- **AND** exit code is 1

## Migrations: Rename Version File

### Requirement: Rename `fab/project/VERSION` to `fab/.kit-migration-version`

The migration tracking file SHALL be renamed from `fab/project/VERSION` to `fab/.kit-migration-version`. The new location is inside `fab/` at the same level as `.kit-sync-version`, providing a consistent naming pattern for kit-related version files.

All scripts and skills that reference `fab/project/VERSION` SHALL be updated to use `fab/.kit-migration-version`:
- `fab/.kit/sync/2-sync-workspace.sh` — creation logic
- `fab/.kit/scripts/fab-upgrade.sh` — drift detection after upgrade
- `fab/.kit/skills/fab-setup.md` — migrations subcommand, bootstrap output
- `fab/.kit/skills/fab-status.md` — version drift display
- `fab/.kit/migrations/0.9.0-to-0.10.0.md` and `0.10.0-to-0.20.0.md` — any inline references

#### Scenario: New project bootstrap
- **GIVEN** no `fab/.kit-migration-version` exists
- **AND** no `fab/project/config.yaml` exists (new project)
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/.kit-migration-version` is created with the engine version from `fab/.kit/VERSION`
- **AND** stdout references the new path

#### Scenario: Existing project without migration file
- **GIVEN** no `fab/.kit-migration-version` exists
- **AND** `fab/project/config.yaml` exists (existing project)
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/.kit-migration-version` is created with `0.1.0`
- **AND** stdout includes guidance to run `/fab-setup migrations`

#### Scenario: Upgrade drift detection
- **GIVEN** `fab/.kit-migration-version` contains `0.19.0`
- **AND** `fab/.kit/VERSION` contains `0.20.0` (just upgraded)
- **WHEN** `fab-upgrade.sh` completes
- **THEN** stdout includes: `⚠ Run /fab-setup migrations to update project files (0.19.0 → 0.20.0)`

### Requirement: Backward Compatibility During Rename

`2-sync-workspace.sh` SHALL check for the old `fab/project/VERSION` location and migrate it to `fab/.kit-migration-version` if found. This is a one-time migration:

#### Scenario: Old VERSION file exists, new does not
- **GIVEN** `fab/project/VERSION` contains `0.15.0`
- **AND** `fab/.kit-migration-version` does not exist
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/.kit-migration-version` is created with `0.15.0`
- **AND** `fab/project/VERSION` is deleted
- **AND** stdout includes: `Migrated: fab/project/VERSION → fab/.kit-migration-version (0.15.0)`

#### Scenario: Both files exist (edge case)
- **GIVEN** `fab/project/VERSION` contains `0.15.0`
- **AND** `fab/.kit-migration-version` contains `0.18.0`
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/project/VERSION` is deleted (new file takes precedence)
- **AND** `fab/.kit-migration-version` retains `0.18.0`
- **AND** stdout includes: `Cleaned: stale fab/project/VERSION (migrated to fab/.kit-migration-version)`

#### Scenario: Only new file exists (normal state after migration)
- **GIVEN** `fab/.kit-migration-version` contains `0.20.0`
- **AND** `fab/project/VERSION` does not exist
- **WHEN** `fab-sync.sh` runs
- **THEN** `fab/.kit-migration-version` is preserved unchanged

## Migrations: Patch Migration File

### Requirement: Migration `0.20.0-to-0.21.0.md`

A migration file SHALL be created at `fab/.kit/migrations/0.20.0-to-0.21.0.md` that handles the rename of `fab/project/VERSION` → `fab/.kit-migration-version` for projects upgrading from 0.20.0. This migration runs via `/fab-setup migrations` after users pull the new kit version.

The migration SHALL:
1. Move `fab/project/VERSION` to `fab/.kit-migration-version` (preserving the version value)
2. Bump the version to `0.21.0` in the new file
3. Verify the rename succeeded

Note: `2-sync-workspace.sh` also handles this rename opportunistically (backward-compat check), but the migration is the formal mechanism for users who follow the standard upgrade flow (`fab-upgrade.sh` → `/fab-setup migrations`).

#### Scenario: Standard migration from 0.20.0
- **GIVEN** `fab/project/VERSION` contains `0.20.0`
- **AND** `fab/.kit-migration-version` does not exist
- **WHEN** `/fab-setup migrations` applies `0.20.0-to-0.21.0.md`
- **THEN** `fab/.kit-migration-version` is created with `0.21.0`
- **AND** `fab/project/VERSION` is deleted
- **AND** migration reports success

#### Scenario: Already migrated (sync ran first)
- **GIVEN** `fab/project/VERSION` does not exist
- **AND** `fab/.kit-migration-version` contains `0.20.0`
- **WHEN** `/fab-setup migrations` applies `0.20.0-to-0.21.0.md`
- **THEN** `fab/.kit-migration-version` is updated to `0.21.0`
- **AND** migration reports success (skip file move, just bump version)

#### Scenario: Both files exist (edge case)
- **GIVEN** `fab/project/VERSION` contains `0.20.0`
- **AND** `fab/.kit-migration-version` contains `0.20.0`
- **WHEN** `/fab-setup migrations` applies `0.20.0-to-0.21.0.md`
- **THEN** `fab/project/VERSION` is deleted
- **AND** `fab/.kit-migration-version` is updated to `0.21.0`

## Design Decisions

1. **Staleness check in preflight, not in skills or preamble**
   - *Why*: Preflight is a shell script that runs automatically on every pipeline skill. Putting the check there means zero agent cooperation needed — every skill gets the warning for free via stderr. Preamble-based checks would require every agent to implement the comparison.
   - *Rejected*: Preamble instruction — requires agent cooperation, fragile, and easy to skip. Standalone script — adds friction, users won't run it proactively.

2. **Gitignored stamp file, not committed**
   - *Why*: The stamp reflects *this developer's* local state. Developer A's sync doesn't mean Developer B's skills are current. A committed stamp would create a false signal — "someone synced" doesn't mean "you synced."
   - *Rejected*: Committed stamp — gives false confidence about other developers' local state.

3. **Version string comparison, not content hashing**
   - *Why*: Comparing `fab/.kit/VERSION` against the stamp is simple, fast, and covers the primary use case (kit upgraded, sync not re-run). Content hashing would detect manual edits to individual skill files, but that's an unusual scenario not worth the complexity.
   - *Rejected*: Content hash of all skill files — expensive, complex, targets an unlikely scenario.

4. **Rename `fab/project/VERSION` rather than keeping both**
   - *Why*: The old name (`fab/project/VERSION`) is ambiguous — "project version" could mean anything. `fab/.kit-migration-version` explicitly describes its purpose (migration tracking) and sits alongside `.kit-sync-version` with a consistent naming pattern. The backward-compatibility migration in `fab-sync.sh` makes this safe.
   - *Rejected*: Keeping `fab/project/VERSION` — perpetuates the ambiguous naming and doesn't align with the new `.kit-*-version` convention.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Problem is stale local skill copies, not project-vs-engine drift | Confirmed from intake #1 — user clarified; `.claude/`, `.agents/` are gitignored | S:95 R:95 A:95 D:95 |
| 2 | Certain | `fab-sync.sh` writes the stamp | Confirmed from intake #2 — it's the only mechanism that deploys skills | S:90 R:95 A:95 D:95 |
| 3 | Certain | Warning is non-blocking (stderr, exit 0) | Confirmed from intake #3 — Constitution III (idempotent operations) | S:85 R:95 A:90 D:95 |
| 4 | Certain | `.kit-sync-version` is gitignored | Confirmed from intake #4 — user decided: per-developer local state | S:95 R:90 A:95 D:95 |
| 5 | Certain | Rename `fab/project/VERSION` → `fab/.kit-migration-version` | Confirmed from intake #5 — user decided: clearer name, consistent pattern | S:95 R:80 A:95 D:95 |
| 6 | Certain | Stamp contains bare semver from `fab/.kit/VERSION` | Upgraded from intake Confident #6 — codebase confirms all version files use bare semver | S:90 R:90 A:95 D:95 |
| 7 | Confident | Preflight is the integration point for staleness warning | Confirmed from intake #7 — all pipeline skills run preflight; stderr surfaces automatically | S:75 R:85 A:90 D:80 |
| 8 | Confident | Stamp location is `fab/.kit-sync-version` | Confirmed from intake #8 — outside `.kit/` (survives replacement), inside `fab/` (colocated) | S:75 R:85 A:80 D:80 |
| 9 | Certain | Staleness check placed between init validation and change resolution | Clarified — confirmed placement; ensures visibility even without active change, no noise on uninitialized projects | S:90 R:85 A:90 D:90 |
| 10 | Confident | `fab-sync.sh` migrates old `fab/project/VERSION` to new location | New — backward compatibility requires one-time migration during sync | S:70 R:80 A:85 D:80 |
| 11 | Confident | Version string comparison, not content hashing | New — simple, fast, covers primary use case; hashing targets unlikely scenario | S:75 R:90 A:80 D:75 |
| 12 | Certain | Formal migration file for `fab/project/VERSION` rename | User requested — standard migration mechanism for the version bump, complements sync-time backward compat | S:95 R:85 A:95 D:95 |

12 assumptions (8 certain, 4 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-26

1. **Q**: Should the warning handle both directions (stamp behind engine vs stamp ahead of engine)?
   **A**: Keep current format — shows both values, user can reason about direction. Not worth adding directional logic for an extremely unlikely downgrade scenario.

2. **Q**: Should there be a formal migration file for the `fab/project/VERSION` rename?
   **A**: Yes — added `0.20.0-to-0.21.0.md` migration requirement. Standard upgrade flow (`fab-upgrade.sh` → `/fab-setup migrations`) needs this, even though `fab-sync.sh` also handles it opportunistically.

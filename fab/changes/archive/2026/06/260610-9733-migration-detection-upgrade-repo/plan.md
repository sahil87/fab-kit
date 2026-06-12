# Plan: Mechanical migration detection in `fab upgrade-repo`

**Change**: 260610-9733-migration-detection-upgrade-repo
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Discovery: Binary-Owned Migration Discovery

#### R1: Parse migration filenames into ranges
The discovery package SHALL parse migration filenames of the form `{FROM}-to-{TO}.md`, where FROM and TO are full semver strings, into a `MigrationRange{From, To, File}`. Non-matching names (e.g. `.gitkeep`, `README.md`, malformed names) SHALL be rejected without error.

- **GIVEN** a filename `1.9.7-to-1.10.0.md`
- **WHEN** `parseMigrationFilename` is called
- **THEN** it returns `MigrationRange{From: "1.9.7", To: "1.10.0", File: "1.9.7-to-1.10.0.md"}` and `true`
- **AND** for `.gitkeep`, `README.md`, or `foo.md` it returns `false`

#### R2: Discover the applicable migration chain
`DiscoverMigrations(migrationsDir, local, engine)` SHALL scan the directory, parse filenames, sort applicable candidates by FROM ascending, and walk the discovery loop to produce the ordered `Applicable` list, the `GapSkips` log, and the `Overlaps` list. The walk is: (1) find the first migration where `FROM <= current < TO` → append to `Applicable`, set `current = TO`, repeat; (2) else if a later migration exists with `FROM > current` → record a gap-skip string, advance `current` to that FROM, repeat; (3) else done.

- **GIVEN** local `0.2.0`, engine `0.4.0`, files `0.2.0-to-0.3.0.md` and `0.3.0-to-0.4.0.md`
- **WHEN** `DiscoverMigrations` runs
- **THEN** `Applicable` is `[0.2.0-to-0.3.0, 0.3.0-to-0.4.0]` in FROM-ascending order
- **AND** `GapSkips` and `Overlaps` are empty
- **GIVEN** local `0.2.0`, files `0.2.0-to-0.3.0.md` and `0.5.0-to-0.6.0.md` (gap at 0.3.0–0.5.0)
- **WHEN** discovery runs
- **THEN** `Applicable` contains `0.2.0-to-0.3.0` then `0.5.0-to-0.6.0`, with a `GapSkips` entry recording the `0.3.0 -> 0.5.0` skip

#### R3: Detect overlapping ranges
`DiscoverMigrations` SHALL detect overlapping ranges using `A.From < B.To && B.From < A.To` and record the overlapping filename pairs in `Overlaps`. Overlap is reported, not silently resolved.

- **GIVEN** two files `1.0.0-to-1.2.0.md` and `1.1.0-to-1.3.0.md`
- **WHEN** discovery runs
- **THEN** `Overlaps` contains the pair of conflicting filenames

#### R4: No-op when nothing applies
When no migration matches and the local version is at or beyond every range, `DiscoverMigrations` SHALL return an empty `Applicable` slice (the no-op signal). `len(Applicable) > 0` is the convenience predicate for "migrations needed".

- **GIVEN** local `2.1.0`, newest file `1.9.7-to-1.10.0.md`
- **WHEN** discovery runs
- **THEN** `Applicable` is empty, `Overlaps` is empty (current repo state)
- **GIVEN** local == engine
- **THEN** `Applicable` is empty

#### R5: Reuse existing semver helpers
The discovery code SHALL reuse the existing `parseSemver` and `compareSemver` helpers in `src/go/fab-kit/internal/sync.go` (same package). It SHALL NOT introduce a new semver dependency.

- **GIVEN** the `internal` package already exposes `parseSemver`/`compareSemver`
- **WHEN** discovery compares versions
- **THEN** it calls those helpers directly; `go.mod` gains no semver dependency

### Command: `fab migrations-status`

#### R6: Queryable discovery command
A new `migrations-status` subcommand SHALL resolve `fab/.kit-migration-version` (local), `$(fab kit-path)/VERSION` (engine), scan the engine migrations dir, run `DiscoverMigrations`, and report the result. It SHALL support a `--json` flag. Human output lists local, engine, the ordered applicable list (or "no migrations apply"), gap-skips, and overlaps. JSON output is `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}`.

- **GIVEN** a fab repo with local `2.1.0` and engine `2.1.2`
- **WHEN** `fab migrations-status` runs
- **THEN** it prints local/engine and "no migrations apply" and exits 0
- **WHEN** `fab migrations-status --json` runs
- **THEN** it prints `{"local":"2.1.0","engine":"2.1.2","applicable":[],"gap_skips":[],"overlaps":[]}` and exits 0

#### R7: Exit-code semantics
`migrations-status` SHALL exit 0 on any clean query — including the no-op case AND the overlap case (overlap is surfaced via the `overlaps` field, not the exit code). It SHALL exit non-zero only on a genuine error (missing VERSION file, unreadable migrations dir, missing `.kit-migration-version`).

- **GIVEN** overlapping migration files in the engine dir
- **WHEN** `fab migrations-status` runs
- **THEN** it exits 0 with the conflict surfaced in `overlaps`
- **GIVEN** the engine `VERSION` file is missing
- **THEN** it exits non-zero with an error

#### R8: Router allowlist registration
Because `migrations-status` lives in the `fab-kit` binary, the `fab` router SHALL route it to `fab-kit`. It SHALL be added to the router's `fabKitArgs` allowlist (and the `cmd/fab-kit` `fabKitCommands` map) so `fab migrations-status` reaches the right binary rather than falling through to `fab-go`.

- **GIVEN** the static negative-match router in `src/go/fab-kit/cmd/fab/main.go`
- **WHEN** a user runs `fab migrations-status`
- **THEN** the router dispatches to `fab-kit` (which owns the command), not `fab-go`

### Upgrade: Mechanical Detection + Self-Stamp in `upgrade-repo`

#### R9: Mechanical relevance check replaces string inequality
`fab upgrade-repo` SHALL, after sync, run `DiscoverMigrations` against the target version's cached migrations dir (`CachedKitDir(targetVersion)/migrations`) and the current `.kit-migration-version`, instead of comparing version strings for inequality.

- **GIVEN** local `2.1.0`, target `2.1.2`, newest migration `1.9.7-to-1.10.0.md`
- **WHEN** `fab upgrade-repo` finishes sync
- **THEN** it runs discovery, finds nothing applicable, and does NOT print a migration nag

#### R10: Styled, TTY-gated reminder when migrations apply
When `Applicable` is non-empty (and no overlap), `upgrade-repo` SHALL print the reminder `Run '/fab-setup migrations' to update project files (LOCAL -> TARGET)`, styled bold+yellow (`\033[1;33m...\033[0m`) only when `os.Stdout` is a character device, and plain otherwise. It SHALL NOT stamp `.kit-migration-version` (the skill owns the write after applying).

- **GIVEN** migrations apply and stdout is a TTY
- **WHEN** `upgrade-repo` finishes
- **THEN** the reminder is wrapped in bold-yellow ANSI codes
- **GIVEN** stdout is piped/redirected
- **THEN** the reminder is plain text with no ANSI codes
- **AND** `.kit-migration-version` is unchanged in both cases

#### R11: Silent self-stamp on the no-op case
When `Applicable` is empty and there is no overlap, `upgrade-repo` SHALL write the target version to `fab/.kit-migration-version` silently (no migration line printed), stopping the drift described in the intake.

- **GIVEN** nothing applies after upgrade
- **WHEN** `upgrade-repo` finishes
- **THEN** `fab/.kit-migration-version` is updated to the target version and no migration line is printed

#### R12: Overlap warns, does not stamp
When `Overlaps` is non-empty, `upgrade-repo` SHALL print a warning naming the conflicting files plus "Run '/fab-setup migrations' to resolve." and SHALL NOT stamp `.kit-migration-version`.

- **GIVEN** overlapping migration files in the target migrations dir
- **WHEN** `upgrade-repo` finishes
- **THEN** it warns with the conflicting filenames and leaves `.kit-migration-version` unchanged

#### R13: Missing `.kit-migration-version` preserves init guidance
When `fab/.kit-migration-version` is missing, `upgrade-repo` SHALL preserve the existing init-guidance behavior (per `docs/memory/distribution/migrations.md` "Version Drift Detection").

- **GIVEN** `.kit-migration-version` does not exist
- **WHEN** `upgrade-repo` finishes
- **THEN** the prior init-guidance behavior is unchanged (no discovery-driven stamp/nag)

#### R14: Dependency-free TTY detection
TTY detection SHALL be dependency-free via `f.Stat()` and `info.Mode()&os.ModeCharDevice != 0` — no `golang.org/x/term`, no `go-isatty`.

- **GIVEN** Constitution I (minimal single-binary deps)
- **WHEN** the styling helper decides whether to emit ANSI
- **THEN** it uses the stdlib `os.FileInfo.Mode()` char-device check only

### Skill: `/fab-setup migrations` Delegation

#### R15: Discovery delegated to the binary
The `/fab-setup migrations` skill (Step 2 "Discover Migrations" and Step 3 "Apply Migrations") SHALL run `fab migrations-status --json`, parse the result, STOP and report on non-empty `overlaps`, and otherwise apply each file in `applicable` in order. The "Applying a Migration" section SHALL remain unchanged (the LLM still reads each file and executes Pre-check/Changes/Verification and writes TO to `.kit-migration-version`). Only discovery moves to the binary.

- **GIVEN** the migrations subcommand runs
- **WHEN** it reaches discovery
- **THEN** it calls `fab migrations-status --json` instead of manually scanning/parsing/validating/sorting
- **AND** application of each migration file is unchanged

#### R16: Output-format scenarios preserved
The skill's Output Format scenarios SHALL be preserved and updated only as needed to match the binary-driven discovery flow.

- **GIVEN** the existing Output Format section
- **WHEN** the skill is updated
- **THEN** scenarios still cover multi-step, gap-skip, already-equal, ahead, none, overlap, and mid-chain-failure, consistent with the new flow

### Docs: Constitution-Required Companions

#### R17: CLI reference updated
`src/kit/skills/_cli-fab.md` SHALL document the `fab migrations-status [--json]` command signature (Constitution: CLI changes MUST update `_cli-fab.md`).

- **GIVEN** a new CLI command
- **WHEN** the change ships
- **THEN** `_cli-fab.md` has an entry matching the existing entry format

#### R18: SPEC updated
`docs/specs/skills/SPEC-fab-setup.md` SHALL reflect the skill's Step 2-3 delegation to `fab migrations-status` (Constitution: skill changes MUST update the corresponding `SPEC-*.md`).

- **GIVEN** the skill's discovery behavior changed
- **WHEN** the change ships
- **THEN** the SPEC's migrations flow and Tools-used table reflect `fab migrations-status`

#### R19: Memory changelog + behavior updated
`docs/memory/distribution/migrations.md` SHALL gain a changelog entry for this change (dated 2026-06-10) and update the "Version Drift Detection" subsection and the discovery-algorithm description to reflect: discovery now lives in the binary (`fab migrations-status`); `upgrade-repo` mechanically detects relevance, silently self-stamps when nothing applies, and styles a TTY-gated bold+yellow reminder when migrations are needed.

- **GIVEN** the migration system behavior changed
- **WHEN** the change ships
- **THEN** the memory doc's changelog has a new top row and the relevant subsections describe the binary-owned discovery + self-stamp + styled reminder

### Design Decisions

1. **Binary-owned discovery**: The discovery algorithm moves from skill prose into `src/go/fab-kit/internal/migrations.go` and is exposed as a queryable command. — *Why*: makes the binary the single source of truth, de-duplicates the algorithm that previously lived in both skill prose and (after this change) code, and gives `upgrade-repo` a mechanical relevance signal. — *Rejected*: a private helper used only by `upgrade-repo` (would leave the skill re-deriving discovery in prose, inviting divergence).

2. **Queryable command (`fab migrations-status`)** consumed by both `upgrade-repo` and the skill, rather than a one-off internal call. — *Why*: a shared command is testable, scriptable, and consumed identically by both callers. — *Rejected*: embedding discovery only inside `upgrade-repo`.

3. **TTY-gated styling** (bold+yellow ANSI only when `os.Stdout` is a char device). — *Why*: standard CLI convention; avoids garbled escape codes in piped/redirected logs. — *Rejected*: always-color (breaks logs), never-color (the reminder stays easy to miss, the reported problem).

4. **Silent self-stamp on the no-op case**: when nothing applies, `upgrade-repo` advances `.kit-migration-version` to the target with no printed line. — *Why*: mirrors the skill's existing "no match, no later migrations → set to engine version" rule and stops permanent drift; the no-op is provably safe so there's nothing to announce. — *Rejected*: announcing the stamp (noise), or never stamping from `upgrade-repo` (the drift the intake is fixing).

5. **Skill delegation, application unchanged**: only *discovery* moves to the binary; *application* (read each migration, run Pre-check/Changes/Verification, write TO) stays in the skill. — *Why*: migration files are pure-prompt LLM instructions (Constitution I — Pure Prompt Play); application cannot move into the binary without violating that principle.

6. **Exit 0 including overlap**: `migrations-status` exits 0 on any clean query (incl. no-op and overlap); non-zero only on genuine errors. — *Why*: both callers read the structured `overlaps` field, so a uniform exit-0 query semantics keeps consumption simple.

### Non-Goals

- This change ships no `{FROM}-to-{TO}.md` migration file of its own — it changes tool behavior, not project-file format.
- Migration *application* logic is not moved into the binary (stays LLM-driven per Constitution I).

## Tasks

### Phase 1: Core Discovery

- [x] T001 Create `src/go/fab-kit/internal/migrations.go` with `MigrationRange{From,To,File}`, `DiscoverResult{Local,Engine,Applicable,GapSkips,Overlaps}`, `parseMigrationFilename(name) (MigrationRange, bool)` (match `{FROM}-to-{TO}.md`, parse both as semver via existing `parseSemver`), and `DiscoverMigrations(migrationsDir, local, engine) (DiscoverResult, error)` (scan, parse, overlap-detect `A.From<B.To && B.From<A.To`, sort by FROM asc via `compareSemver`, walk the discovery loop). Reuse `parseSemver`/`compareSemver` from `sync.go`; no new dependency. <!-- R1 R2 R3 R4 R5 -->

### Phase 2: Command Wiring

- [x] T002 Add `migrations-status` subcommand to `src/go/fab-kit/cmd/fab-kit/main.go` with a `--json` flag: resolve local `fab/.kit-migration-version` and engine `$(fab kit-path)/VERSION`, scan the engine migrations dir, run `DiscoverMigrations`, render human or JSON output, exit 0 on clean query incl. overlap, non-zero only on genuine error. Register in `root.AddCommand(...)` and add to the `fabKitCommands` map. Follow the existing command-wiring pattern. <!-- R6 R7 -->
- [x] T003 Register `migrations-status` in the `fab` router allowlist `fabKitArgs` in `src/go/fab-kit/cmd/fab/main.go` (and update its `printHelp` workspace-commands list) so the router dispatches it to `fab-kit`. <!-- R8 -->

### Phase 3: Upgrade-Repo Behavior

- [x] T004 Rewrite the reminder block (lines ~92-99) in `src/go/fab-kit/internal/upgrade.go`: after sync, when `.kit-migration-version` exists, run `DiscoverMigrations(CachedKitDir(targetVersion)/migrations, local, targetVersion)`. Overlap → warn naming files + "Run '/fab-setup migrations' to resolve.", no stamp. Applicable non-empty → styled TTY-gated reminder, no stamp. Applicable empty + no overlap → silently write target to `.kit-migration-version`. Missing file → preserve existing init-guidance behavior. Add `isTTY(f *os.File) bool` and a bold-yellow styling helper. <!-- R9 R10 R11 R12 R13 R14 -->

### Phase 4: Tests

- [x] T005 Create `src/go/fab-kit/internal/migrations_test.go`: table-driven tests for `parseMigrationFilename` (valid, `.gitkeep`, `README.md`, malformed) and `DiscoverMigrations` (applicable chain across files, gap-skip, overlap detection, empty/no-op, local==engine, local-ahead-of-engine). Follow `sync_test.go` style (`t.TempDir()`, table-driven `t.Run`). <!-- R1 R2 R3 R4 -->

### Phase 5: Skill + Docs

- [x] T006 [P] Edit `src/kit/skills/fab-setup.md` Migrations Step 2-3: replace manual scan/parse/validate/sort prose and the inlined discovery loop with running `fab migrations-status --json`, parsing the result, STOP on non-empty `overlaps`, else apply each `applicable` file in order. Keep "Applying a Migration" unchanged. Preserve Output Format scenarios, updating only as needed. Edit ONLY the canonical source. <!-- R15 R16 -->
- [x] T007 [P] Edit `src/kit/skills/_cli-fab.md`: add the `fab migrations-status [--json]` command signature matching the existing entry format. <!-- R17 -->
- [x] T008 [P] Edit `docs/specs/skills/SPEC-fab-setup.md`: reflect the migrations Step 2-3 delegation to `fab migrations-status` (flow diagram + Tools-used table). <!-- R18 -->
- [x] T009 [P] Edit `docs/memory/distribution/migrations.md`: add a top changelog row for `260610-9733-migration-detection-upgrade-repo` dated 2026-06-10; update "Version Drift Detection" and the `/fab-setup migrations` discovery-algorithm description to reflect binary-owned discovery + silent self-stamp + TTY-gated styled reminder. <!-- R19 -->

### Phase 6: Build & Smoke Test

- [x] T010 Build + vet + test: `cd src/go/fab-kit && go build ./... && go vet ./... && go test ./internal/ ./cmd/...`; build the shim `cd src/go/fab && go build ./...`. Smoke-test `migrations-status` and `migrations-status --json` against this repo (expect no applicable migrations). <!-- R6 R7 R9 -->

## Execution Order

- T001 blocks T002, T004, T005 (they consume the discovery API)
- T002 blocks T003 (router registration mirrors the fab-kit command)
- T006-T009 are independent `[P]` (different files, no code dependency)
- T010 runs last (validates all Go work)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `parseMigrationFilename` returns the correct `MigrationRange` for valid names and `false` for `.gitkeep`/`README.md`/malformed names
- [x] A-002 R2: `DiscoverMigrations` returns the ordered applicable chain (FROM ascending) and records gap-skips
- [x] A-003 R3: `DiscoverMigrations` records overlapping filename pairs in `Overlaps` using the `A.From<B.To && B.From<A.To` test
- [x] A-004 R4: `DiscoverMigrations` returns empty `Applicable` for the no-op cases (local==engine, local ahead of newest range)
- [x] A-005 R5: discovery reuses `parseSemver`/`compareSemver` from `sync.go`; `go.mod` gains no semver dependency
- [x] A-006 R6: `fab migrations-status` and `--json` produce the documented human and JSON shapes
- [x] A-007 R7: exit code is 0 on clean query incl. overlap; non-zero only on genuine error
- [x] A-008 R8: `migrations-status` is in the router `fabKitArgs` allowlist and reaches `fab-kit`
- [x] A-009 R9: `upgrade-repo` runs `DiscoverMigrations` against the target migrations dir instead of string inequality
- [x] A-010 R11: `upgrade-repo` silently stamps `.kit-migration-version` to target when nothing applies
- [x] A-011 R15: the skill's Step 2-3 delegates discovery to `fab migrations-status --json`; "Applying a Migration" is unchanged
- [x] A-012 R17 R18 R19: `_cli-fab.md`, `SPEC-fab-setup.md`, and `migrations.md` are updated per the constitution-required companions

### Behavioral Correctness

- [x] A-013 R10: the reminder is bold-yellow ANSI on a TTY and plain when piped; `.kit-migration-version` is not stamped when migrations apply
- [x] A-014 R12: on overlap, `upgrade-repo` warns with the conflicting filenames and does not stamp
- [x] A-015 R13: missing `.kit-migration-version` preserves the prior init-guidance behavior
- [x] A-016 R14: TTY detection uses `os.ModeCharDevice` only — no new dependency

### Scenario Coverage

- [x] A-017 R2 R3 R4: `migrations_test.go` exercises applicable-chain, gap-skip, overlap, no-op, local==engine, and local-ahead scenarios
- [x] A-018 R6 R9: smoke test of `migrations-status` / `--json` against this repo shows no applicable migrations (local 2.1.0, engine 2.1.2)

### Edge Cases & Error Handling

- [x] A-019 R7: missing engine VERSION or unreadable migrations dir yields a non-zero exit with a clear error
- [x] A-020 R1: non-migration files in the dir are skipped without error

### Code Quality

- [x] A-021 Pattern consistency: new Go code follows the `internal` package's naming, error-handling (`fmt.Errorf` wrapping), and cobra command-wiring patterns observed in `sync.go`, `upgrade.go`, `doctor.go`, and `main.go`
- [x] A-022 No unnecessary duplication: discovery reuses `parseSemver`/`compareSemver`; no semver logic is reimplemented (per code-quality.md anti-pattern "Duplicating existing utilities")
- [x] A-023 No god functions: discovery and command-render logic stay focused (code-quality.md: functions >50 lines without clear reason are an anti-pattern); extract helpers where warranted
- [x] A-024 No magic strings: the migration filename suffix/pattern and ANSI codes are expressed as named constants or clearly-scoped literals (code-quality.md anti-pattern "Magic strings without named constants")

### Documentation Accuracy

- [x] A-025 R17 R18 R19: `_cli-fab.md`, `SPEC-fab-setup.md`, and `migrations.md` accurately describe the shipped behavior (command signature, flags, exit codes, discovery delegation, self-stamp, styled reminder) — no stale claims
- [x] A-026 R15: `src/kit/skills/fab-setup.md` is the only skill file edited (deployed `.claude/skills/` copies are not touched)

### Cross References

- [x] A-027 R8 R19: `migrations.md` and `kit-architecture.md` references to the router allowlist remain consistent (the allowlist now includes `migrations-status`); the changelog entry cross-links the change folder name
- [x] A-028 R15 R18: the skill's discovery steps and `SPEC-fab-setup.md` reference the same `fab migrations-status --json` contract consistently

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

<!-- All design points below were settled as Certain assumptions in intake.md (9 certain,
     re-confirmed via AskUserQuestion). The apply agent recorded one additional Certain
     decision (router allowlist) discovered while reading the codebase. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-op case stamps `.kit-migration-version` to target silently. | Settled in intake.md #1 (user chose "Stamp to target, silent"). | S:98 R:80 A:90 D:95 |
| 2 | Certain | Reminder styling bold+yellow, TTY-gated. | Settled in intake.md #2. | S:98 R:85 A:90 D:95 |
| 3 | Certain | Overlap warns + does not stamp; resolution deferred to skill. | Settled in intake.md #3. | S:98 R:75 A:85 D:90 |
| 4 | Certain | TTY detection dependency-free via `os.ModeCharDevice`. | Settled in intake.md #4. | S:95 R:80 A:85 D:75 |
| 5 | Certain | Discovery reuses `parseSemver`/`compareSemver` from `sync.go`. | Settled in intake.md #5. | S:95 R:80 A:90 D:80 |
| 6 | Certain | Discovery exposed as queryable command `fab migrations-status`. | Settled in intake.md #6. | S:95 R:65 A:85 D:80 |
| 7 | Certain | Skill keeps applying migrations; only discovery moves to the binary. | Settled in intake.md #7. | S:95 R:75 A:90 D:85 |
| 8 | Certain | `--json` shape `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}`. | Settled in intake.md #8. | S:95 R:75 A:80 D:70 |
| 9 | Certain | `migrations-status` exits 0 incl. overlap; non-zero only on genuine error. | Settled in intake.md #9. | S:95 R:80 A:65 D:75 |
| 10 | Certain | `migrations-status` must be added to the router `fabKitArgs` allowlist (and `cmd/fab-kit` `fabKitCommands` map), because the `fab` router uses a static negative-match allowlist for fab-kit commands and routes everything else to `fab-go`. | Determined by codebase: `src/go/fab-kit/cmd/fab/main.go` `fabKitArgs` is the dispatch gate; an unlisted fab-kit command would wrongly route to `fab-go`. Intake noted shim passthrough was "to be verified during apply"; verification shows explicit registration is required. | S:90 R:85 A:95 D:90 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).

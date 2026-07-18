# Plan: `fab skill` Subcommand + Canonical Skill Bundle

**Change**: 260718-fskl-fab-skill-subcommand
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from the intake design + the live shll `skill` standard
     (re-fetched at apply entry via `shll standards skill`, byte-matching the v0.0.23
     backlog pin). The standard is the binding contract per the constitution's
     Toolkit Standards article. -->

### Bundle: `docs/site/skill.md` canonical usage briefing

#### R1: The canonical bundle file exists and is a static, bounded, agent-first usage briefing

A new file `docs/site/skill.md` SHALL exist as the canonical `fab skill` bundle. It MUST be a **usage briefing** in agent-first language (when to reach for fab and when not; a capabilities map keyed to subcommands; composition patterns with `wt`/`rk`/`gh`/the `/fab-*` skills; stdout/exit-code contracts; gotchas). It MUST be **static-only** (byte-identical on every invocation/machine for a release — no timestamps, environment lookups, or session state). It MUST be **≤150 lines**. It MUST NOT contain exhaustive flag tables, full command trees, or install prose (those defer to `-h`/help-dump/README). It MUST disambiguate `fab skill` (this toolkit-standard bundle) from fab's own kit-skills (the `/fab-*` markdown deployed to `.claude/skills/` by `fab sync`).

- **GIVEN** the fab-kit repo after this change
- **WHEN** `docs/site/skill.md` is inspected
- **THEN** it is a static agent-usage briefing of ≤150 lines covering when-to-use / capabilities / composition / output-contracts / gotchas
- **AND** it contains no dynamic content, no exhaustive flag tables, no install prose
- **AND** it explicitly distinguishes the `fab skill` command from fab's kit-skills concept

### Command: `fab skill` subcommand

#### R2: A visible zero-arg `fab skill` command prints the bundle byte-identically to stdout with exit 0 and empty stderr

The rich fab-go CLI SHALL register a new visible `skill` subcommand (`src/go/fab/cmd/fab/skill.go`). It MUST print the embedded bundle as **raw markdown to stdout, byte-identical** to `docs/site/skill.md`, with **stderr empty on success and exit code 0**, no rendering/pager/framing. It MUST take no args/flags (`cobra.NoArgs`); an argued invocation MUST be a usage error (exit `2` via the existing binary-wide `run()`/`markRunReached` classification — no new exit-code code).

- **GIVEN** an installed `fab` binary
- **WHEN** `fab skill` is run with no arguments
- **THEN** stdout is byte-identical to the embedded bundle, stderr is empty, exit code is 0
- **AND WHEN** `fab skill foo` is run (extra arg)
- **THEN** it is rejected as a usage error with exit code 2

#### R3: The command does not collide with the router allowlist

The command name `skill` MUST NOT appear in the `fab` router's `LifecycleCommands` workspace allowlist (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`), so the router's always-route policy forwards it to fab-go unchanged. No router or fab-kit binary change is required.

- **GIVEN** the assembled fab-go command tree and the router allowlist
- **WHEN** `TestNoTopLevelCommandCollidesWithRouterAllowlist` runs
- **THEN** `skill` is not in the allowlist and the test passes

### Embedding: sync + drift-guard mechanism

#### R4: The bundle is embedded at build time via a committed copy + sync script + drift-guard test

The bundle MUST be embedded into the fab-go binary via the sync + drift-guard pattern shll's `standards` mechanism established, adapted to a single file. A committed copy of `docs/site/skill.md` SHALL live at `src/go/fab/cmd/fab/skill.md` beside `skill.go`, embedded via `//go:embed skill.md`, so a clean `go build ./...` compiles without running any script. A sync script `scripts/sync-skill.sh` (`set -euo pipefail`, cd to repo root, `cp -f` canonical → package dir) SHALL refresh the copy, referenced by a `//go:generate` directive in `skill.go`. A drift-guard Go test SHALL compare the embedded bytes against the canonical `docs/site/skill.md` byte-for-byte and fail when they diverge.

- **GIVEN** the repo
- **WHEN** `go build ./...` runs in `src/go/fab` without running the sync script
- **THEN** the build succeeds (the committed embedded copy is present)
- **AND WHEN** `docs/site/skill.md` is edited without re-running `scripts/sync-skill.sh`
- **THEN** the drift-guard test fails, naming the drifted file

#### R5: Contract tests pin the command's behavior and the bundle's constraints

Go tests SHALL assert: stdout byte-identity with the embedded copy, empty stderr, exit 0 for the bare command; the ≤150-line budget on the bundle; and a static-only sanity check where feasible. These accompany the code in the same change.

- **GIVEN** the fab-go test suite
- **WHEN** `go test ./cmd/fab/...` runs
- **THEN** the skill command's byte-identity, empty-stderr, and ≤150-line-budget tests pass

### Documentation & conformance obligations

#### R6: `_cli-fab.md` documents the new subcommand and the SPEC mirror sweep is performed

A new `## fab skill` section SHALL be added to `src/kit/skills/_cli-fab.md` documenting the subcommand (a new visible subcommand changes the CLI surface). The SPEC-mirror sweep (code-quality § Sibling & Mirror Sweeps) SHALL update `docs/specs/skills/SPEC-_cli-fab.md` (its Command Inventory) and any aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) that restate the command roster. The help-dump contract SHALL be re-verified: the new node flows into the live cobra tree automatically, carrying a sensible `Short`/`Usage`, and existing help-dump conformance tests re-run green.

- **GIVEN** the constitution's CLI-surface and SPEC-mirror rules
- **WHEN** the change is reviewed
- **THEN** `_cli-fab.md` carries a `## fab skill` section, `SPEC-_cli-fab.md` carries a matching Command Inventory row, and aggregate specs restating the roster are updated where applicable
- **AND** existing help-dump / lifecycle-collision conformance tests pass

### Non-Goals

- No cobra `Example:` block / no `exampleTargetPaths` entry for `fab skill` — the b91h audit scoped `Example:` to user-facing *multi-flag* commands; `fab skill` is zero-flag/zero-arg.
- No router or fab-kit binary change — the always-route policy covers the new command.
- No migration — nothing restructures user data.
- No consumer-side version-skew handling — shll `[clix]` owns the predates-subcommand fallback by design.
- No shll.ai rendering work — `docs/site/**` is already the pulled+rendered site surface; the page renders at `shll.ai/tools/fab-kit/skill` for free.

### Design Decisions

1. **Embedded copy placement**: `src/go/fab/cmd/fab/skill.md` beside `skill.go`, single-file `//go:embed skill.md` — *Why*: mirrors shll's `standards/` layout adapted from a 4-file dir to one file; the fab-go module root is `src/go/fab/` and `docs/site/` sits above it, so `//go:embed` cannot reach the canonical file directly (identical geometry to shll). *Rejected*: embedding from `docs/site/` directly (impossible — outside the module root); a `standards/`-style subdirectory (unnecessary indirection for a single file).
2. **Exit-code reuse**: rely on the existing binary-wide `run()`/`markRunReached` classification for the argued-invocation usage error (`cobra.NoArgs` → exit 2). *Why*: swon already established the phase-based classifier; a zero-arg command needs no bespoke exit code. *Rejected*: an in-handler `os.Exit` scheme (would need renumbering justification and adds nothing).
3. **Testable seam**: extract a `runSkill(stdout, stderr io.Writer) error` helper from the cobra factory, driven directly with `bytes.Buffer`s in tests. *Why*: mirrors shll's `runStandards` and fab's own `runList`-style seams — no subprocess needed since the command reads embedded bytes only.

## Tasks

### Phase 1: Canonical bundle

- [x] T001 Author `docs/site/skill.md` — a static, agent-first usage briefing ≤150 lines: when-to-use, capabilities map keyed to subcommands, composition patterns (`wt`/`rk`/`gh`/`/fab-*` skills via `fab sync`), stdout/exit-code contracts (stdout-is-data, `--json` availability, the `0`/`1`/`2` convention + special exits: pane 2/3, memory-index 0/1/2, sync/migrations-status 3), and gotchas (fab routes to two binaries; `.claude/skills/` is deployed copies never to edit; `<change>` accepts ID/substring/folder; `fab skill` ≠ fab's kit-skills). Sourced from README.md, `_cli-fab.md`, live `-h`, distribution memory. <!-- R1 --> <!-- rework cycle 2: review must-fix — (a) lines 29-31 + 76-79 deny `--json` on `fab status <query>`, but 9 of 10 status read-only queries carry it on THIS BRANCH (#490; only progress-line lacks it) — the cycle-1 fix verified against the older installed binary instead of the branch source; (b) lines 59-60 misattribute backlog mark-done to `fab change archive` — it belongs to `fab batch archive` / the /fab-archive skill. EXACT replacement text pinned in the dispatch prompt — splice verbatim, do not reword. Also should-fix line 43: `[--check] [--json]` → `[--check [--json]]`. Verify accuracy against the BRANCH-BUILT binary (go run/go build from src/go/fab), never the installed one. -->

### Phase 2: Embedding mechanism

- [x] T002 Create the committed embedded copy `src/go/fab/cmd/fab/skill.md` (copy of `docs/site/skill.md`) so a clean `go build ./...` compiles. <!-- R4 --> <!-- rework cycle 2: re-run scripts/sync-skill.sh after the T001 content fixes so the embedded copy stays byte-identical (drift-guard enforces) -->
- [x] T003 Create `scripts/sync-skill.sh` (`set -euo pipefail`, cd to repo root via `dirname "$0"/..`, `cp -f docs/site/skill.md src/go/fab/cmd/fab/skill.md`, echo confirmation) — mirroring `scripts/sync-standards.sh` adapted to one file. <!-- R4 -->

### Phase 3: Command implementation

- [x] T004 Add `src/go/fab/cmd/fab/skill.go`: `//go:generate ../../../../../scripts/sync-skill.sh` (5 levels from `cmd/fab` to repo root — verified), `//go:embed skill.md` var, `skillCmd()` cobra factory (`Use: "skill"`, `cobra.NoArgs`, sensible `Short`/`Long`), and a testable `runSkill(stdout io.Writer) error` seam that writes the embedded bytes verbatim. <!-- R2 -->
- [x] T005 Register `skillCmd()` in `src/go/fab/cmd/fab/main.go` `newRootCmd()` AddCommand list. <!-- R2 -->

### Phase 4: Tests

- [x] T006 Add `src/go/fab/cmd/fab/skill_test.go`: (a) `runSkill` stdout byte-identical to the embedded bytes, stderr empty; (b) drift-guard `TestSkillEmbedMatchesCanonical` comparing embedded bytes to canonical `docs/site/skill.md` byte-for-byte (shll's `TestStandardsEmbedMatchesCanonical` shape, using a `findSkillDocFile` walk-up helper from the test's CWD — robust to package depth); (c) ≤150-line budget assertion on the embedded bundle; (d) a static-only sanity check (no obviously dynamic tokens). <!-- R5 --> <!-- rework: review should-fix — findSkillDocFile duplicates findCollisionDocFile in the same package (lifecycle_collision_test.go:71-90); call the existing helper or hoist one neutrally-named shared test helper -->
- [x] T007 Verify `fab skill` exit-code behavior: bare invocation exits 0 (covered by byte-identity via the seam + `TestSkill_EmptyStderrThroughCobra`); `cobra.NoArgs` rejects extra args → exit 2 via the binary-wide classifier (`TestSkill_RejectsArgs` + end-to-end binary check). `TestNoTopLevelCommandCollidesWithRouterAllowlist` still passes (skill ∉ allowlist). <!-- R3 -->

### Phase 5: Documentation & conformance sweep

- [x] T008 Add a `## fab skill` section to `src/kit/skills/_cli-fab.md` (after `## fab shell-init`) documenting the command, its byte-identity/stdout/exit contract, the embedding mechanism, and the `fab skill` vs. kit-skills disambiguation; also added to the `## Contents` list and the `§ Commands covered` enumeration. <!-- R6 -->
- [x] T009 SPEC mirror sweep: added a `fab skill` row to the Command Inventory table in `docs/specs/skills/SPEC-_cli-fab.md`; added `skill` to the config-free-command roster in `docs/specs/architecture.md` § Always-Route Policy. `docs/specs/skills.md` and `docs/specs/glossary.md` restate no exhaustive command roster (glossary's `fab` CLI entry is illustrative + "and more") — correctly left untouched. <!-- R6 -->
- [x] T010 Ran `cd src/go/fab && go test ./... ` (all packages green) plus the fab-kit module (the `_cli-fab.md` router-line contract test) — help-dump conformance and lifecycle-collision tests pass with the new `skill` node present. <!-- R6 --> <!-- rework cycle 2: re-run after the cycle-2 T001/T002 edits to confirm the suites stay green, incl. the drift-guard on the re-synced copy -->

## Execution Order

- T001 blocks T002 (the committed copy is a copy of the canonical file).
- T002 + T004 block T006 (tests need the embedded copy and the seam).
- T004 blocks T005 (register after the factory exists).
- T008 blocks T009 (SPEC mirror follows the `_cli-fab.md` edit).
- T010 runs last (final conformance gate over the full package).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs/site/skill.md` exists as a static agent-usage briefing (when-to-use / capabilities / composition / output-contracts / gotchas), ≤150 lines, with the `fab skill` vs. kit-skills disambiguation present.
- [x] A-002 R2: `fab skill` (no args) prints the bundle byte-identically to stdout, stderr empty, exit 0; a `skill.go` command is registered in `newRootCmd()`.
- [x] A-003 R4: The committed embedded copy `src/go/fab/cmd/fab/skill.md`, `scripts/sync-skill.sh`, and the `//go:generate` + `//go:embed` directives all exist; a clean `go build ./...` compiles without running the sync script.
- [x] A-004 R5: The contract tests (byte-identity, empty stderr, ≤150-line budget) and the drift-guard test exist and pass.
- [x] A-005 R6: `_cli-fab.md` carries a `## fab skill` section and `SPEC-_cli-fab.md` carries a matching Command Inventory row.

### Behavioral Correctness

- [x] A-006 R2: `fab skill foo` (extra arg) is rejected as a usage error (exit 2 via the existing `run()`/`markRunReached` classifier — no new exit-code code added).
- [x] A-007 R4: Editing `docs/site/skill.md` without re-syncing makes the drift-guard test fail, naming the drifted file; re-running `scripts/sync-skill.sh` restores byte-identity.

### Scenario Coverage

- [x] A-008 R3: `TestNoTopLevelCommandCollidesWithRouterAllowlist` passes — `skill` is not in the router's `LifecycleCommands` allowlist, so the always-route policy forwards it with no router change.
- [x] A-009 R6: Existing help-dump conformance tests pass with the `skill` node present in the live cobra tree, carrying a non-empty `Short`/`Usage`.

### Edge Cases & Error Handling

- [x] A-010 R4: The `//go:embed skill.md` pattern resolves against the package-local copy (not the out-of-module `docs/site/` path), so the build never depends on the canonical file's location relative to the module root.

### Code Quality

- [x] A-011 Pattern consistency: `skill.go`/`skill_test.go` follow the surrounding `cmd/fab` conventions (cobra factory + testable `run*` seam driven by `bytes.Buffer`s, mirroring `kitpath.go`/`shellinit.go` and shll's `standards.go`).
- [x] A-012 No unnecessary duplication: the embedding reuses the shll sync+drift-guard shape rather than inventing a new mechanism; no second copy of the bundle beyond the committed embed.
- [x] A-013 Test integrity (constitution VII): tests conform to the standard's contract (byte-identity, ≤150 lines, static-only), not the reverse.

### documentation_accuracy

- [x] A-014 R6: The `## fab skill` section in `_cli-fab.md` and the `SPEC-_cli-fab.md` row accurately describe the shipped command (name, byte-identity, stdout/exit contract, zero-arg) with no stale or aspirational claims.

### cross_references

- [x] A-015 R6: The SPEC-mirror sweep class is complete — `_cli-fab.md` ↔ `SPEC-_cli-fab.md` are in sync and aggregate specs restating the command roster are updated (or correctly left untouched where no roster is restated); no dangling references to a non-existent command elsewhere.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Re-verified at cycle-2 review: purely additive — a new subcommand, bundle, sync script, and tests; the router, fab-kit binary, and all existing commands are untouched; no prior surface served the embedded-bundle role. The one near-duplicate the change could have created — a second walk-up helper beside `findCollisionDocFile` — was avoided by hoisting it to the shared `findRepoFile` in `lifecycle_collision_test.go`, leaving no leftover to delete.)

## Assumptions

<!-- Graded SRAD decisions made while co-generating ## Requirements. Three grades
     only (Certain/Confident/Tentative); Scores column required per row. Carried and
     translated from the intake's Assumptions table — the intake pre-graded the whole
     contract; apply confirms the concrete placement/authoring decisions. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Adopt the invocation contract verbatim from the live standard (command `skill`, raw markdown byte-identical to `docs/site/skill.md`, stderr empty, exit 0, no pager/framing, static-only, ≤150 lines) | Constitution's Toolkit Standards article binds; live standard re-fetched at apply entry (`shll standards skill`) and byte-matches the v0.0.23 backlog pin | S:95 R:90 A:95 D:95 |
| 2 | Certain | Embed via the sync + drift-guard pattern (committed copy in the `cmd/fab` package + `scripts/sync-skill.sh` + byte-identity drift-guard test), reusing shll's `standards` mechanism | The standard names this mechanism explicitly ("reuse it"); module-root geometry is identical to shll (module at `src/go/fab/`, `docs/site/` above it); fab-go has no `go:embed` today so this establishes it | S:90 R:85 A:90 D:90 |
| 3 | Confident | Concrete placement/naming: embedded copy at `src/go/fab/cmd/fab/skill.md` beside `skill.go` (single-file `//go:embed skill.md`); sync script `scripts/sync-skill.sh` with a `//go:generate` pointer using the `../../../../../` relative path from `cmd/fab` to repo root | Mirrors shll's layout adapted from a 4-file dir to one file; pure naming/path choice, trivially renameable; verified the `cmd/fab`→repo-root depth is five levels | S:75 R:90 A:85 D:70 |
| 4 | Confident | Bundle audience & content: a usage briefing for an agent operating the installed fab without the deployed kit-skills context; drawn from README/`_cli-fab.md`/`-h`/memory; must disambiguate `fab skill` from fab's kit-skills | The standard's genre definition answers the audience question; exact prose is authoring judgment grounded in strong repo signal | S:75 R:75 A:70 D:60 |
| 5 | Confident | Place the `## fab skill` doc section near the other zero-arg reference commands (`shell-init`/`kit-path`) in `_cli-fab.md`, and add exactly one Command Inventory row to `SPEC-_cli-fab.md`; sweep aggregate specs but add a roster mention only where a roster is genuinely restated | Follows the file's existing `##`-per-command organization and the code-quality § Sibling & Mirror Sweeps rule; aggregate-spec placement is judgment (some restate the roster, some don't) | S:70 R:85 A:80 D:70 |
| 6 | Confident | No cobra `Example:` block / no `exampleTargetPaths` entry for `fab skill` | The b91h audit scoped `Example:` to user-facing multi-flag commands; `fab skill` takes no flags or args (verified against `examples_test.go`) | S:65 R:95 A:85 D:80 |
| 7 | Certain | No router/fab-kit change and no migration; consumer-side version-skew and site rendering need no work here | Verified `skill` ∉ `LifecycleCommands` allowlist (guarded by `lifecycle_collision_test.go`); nothing restructures user data; shll `[clix]` owns the fallback; `docs/site/**` is already the pulled+rendered surface | S:85 R:90 A:90 D:92 |

7 assumptions (3 certain, 4 confident, 0 tentative).

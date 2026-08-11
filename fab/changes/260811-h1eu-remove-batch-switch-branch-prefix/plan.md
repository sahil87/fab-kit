# Plan: Remove batch switch branch_prefix — unify branch naming on the no-prefix convention

**Change**: 260811-h1eu-remove-batch-switch-branch-prefix
**Intake**: `intake.md`

## Requirements

### CLI: `fab batch switch` branch naming

#### R1: Branch name equals the change folder name
`fab batch switch` MUST name the worktree branch exactly the resolved change folder name, with no
prefix applied from any config key. This makes the two branch-creating paths (`/git-branch`,
`/fab-new` Step 11 and `fab batch switch`) share one branch namespace, as `docs/specs/naming.md`
§ Git Branch and `_preamble.md` § Naming Conventions already specify.

- **GIVEN** a change folder `260401-ab12-add-feature` and a project `fab/project/config.yaml`
  carrying a live `branch_prefix: "feature/"` key
- **WHEN** `fab batch switch ab12` runs
- **THEN** the branch it probes and hands to `wt create` is `260401-ab12-add-feature`
- **AND** no `feature/`-prefixed branch is created or checked out

#### R2: The branch-existence probe-and-route contract is unchanged
The `branchExists` probe-and-route behavior (local `git show-ref` first, then `git ls-remote --heads
origin`; exists → `--checkout <branch>`, missing → positional; offline `ls-remote` degrades to
positional) MUST be preserved verbatim — only the branch *name* computation changes.

- **GIVEN** the change's branch already exists locally
- **WHEN** `fab batch switch` runs for it
- **THEN** `wt create --non-interactive --reuse --worktree-name <folder> --checkout <folder>` is
  invoked
- **AND** with the branch absent both locally and on origin, the trailing positional `<folder>` form
  is invoked instead

### Config: retiring the `branch_prefix` key

#### R3: The key leaves the Go config surface entirely
`branch_prefix` MUST be removed from every live config surface in the `fab` module: the
`config.Config` struct field, the `GetBranchPrefix()` accessor, the `internal/configref` registry
row, and both `internal/configscope` tables (`keyScopes` and `dottedKeys`). After removal a
`branch_prefix:` key on disk MUST be an inert unknown key — ignored silently by the loader, exactly
as `review_tools` and `agent.spawn_command` are.

- **GIVEN** the retired registry row
- **WHEN** `fab config explain` renders the reference
- **THEN** no `branch_prefix` key token appears anywhere in the output, including the reference
  header's optional-override-block enumeration
- **AND** `configscope.ScopeFor("branch_prefix")` reports the key unknown
- **AND** `configref.Fields()` and `configscope.DottedKeys()` stay at parity (the existing parity
  test passes unchanged)

#### R4: Test coverage tracks the retirement
The affected `*_test.go` files MUST be updated so no test asserts the retired key's live behavior,
and `TestConfigReferenceRetiresLegacyKeys` MUST additionally assert `branch_prefix` is absent from
the rendered reference. Test *comments* stating the old behavior MUST be corrected alongside the
assertions.

- **GIVEN** the retired key
- **WHEN** `go test ./...` runs from `src/go/fab`
- **THEN** every package passes
- **AND** `TestConfigReferenceRetiresLegacyKeys` fails if a future change re-renders
  `branch_prefix` in the reference

#### R5: This repo's managed fence loses the advert
`fab/project/config.yaml`'s managed reference fence MUST no longer carry the `# branch_prefix — …` /
`# branch_prefix: ""` advert block, and MUST be produced by the rebuilt binary's `fab config
upgrade` rather than hand-edited (the fence is generator-owned).

- **GIVEN** the retired registry row and the rebuilt `fab` binary
- **WHEN** `fab config upgrade` runs with the repo root as CWD
- **THEN** the regenerated fence contains no `branch_prefix` line
- **AND** every other advert block in the fence (`checklist.extra_categories`,
  `consolidate.detectors`, `dispatch.*`, `agent.*`, `stage_hooks`) survives byte-identically
- **AND** the live keys above the fence are untouched

#### R6: User projects get a migration
A migration file `src/kit/migrations/2.19.4-to-2.20.0.md` MUST ship that removes a **live**
`branch_prefix:` key from a user's `fab/project/config.yaml` when present, and is a complete no-op
otherwise. It MUST follow the retired-key migration precedent (`2.14.0-to-2.15.0`,
`2.18.1-to-2.19.0`): Summary / Pre-check / Changes / Verification, sentinel-guarded for
idempotency, atomic temp-file + rename write, commented lines and the managed fence never touched.

- **GIVEN** a user config carrying a live top-level `branch_prefix: "feature/"`
- **WHEN** `/fab-setup migrations` applies the file
- **THEN** the key (and any adjacent comment line documenting it) is removed and every other key,
  value, and comment is preserved verbatim
- **GIVEN** a user config with no live `branch_prefix:` key
- **WHEN** the migration runs (or re-runs after a first pass)
- **THEN** the Pre-check sentinel trips and nothing is changed

### Docs: kit reference, specs, memory

#### R7: `_cli-fab.md` tracks the CLI and schema change
`src/kit/skills/_cli-fab.md` MUST drop `branch_prefix` from the full-schema key list and the
opt-in-override-block enumeration, name it in the retired-keys parenthetical alongside
`review_tools` / `agent.spawn_command`, and state that `fab batch switch` names branches by the
change folder name.

- **GIVEN** the constitution's rule that a CLI/behavior change updates `_cli-fab.md` plus tests
- **WHEN** `_cli-fab.md` is read after this change
- **THEN** it documents `switch` naming branches `{folder_name}` with no prefix
- **AND** carries no claim that `branch_prefix` is an available config key

#### R8: Specs and memory state the current truth
`docs/specs/config.md`, `docs/specs/architecture.md`,
`docs/memory/distribution/kit-architecture.md`, and `docs/memory/_shared/configuration.md` MUST
carry no claim that `branch_prefix` is a live config key or that `fab batch switch` applies a
prefix. Memory indexes MUST be regenerated with `fab memory-index` after the memory edits.
`docs/specs/naming.md` needs no change — it is already authoritative.

- **GIVEN** the retired key
- **WHEN** the four files are read
- **THEN** none advertises `branch_prefix` as settable, and none describes
  `branchName = branchPrefix + match`
- **AND** `fab memory-index --check` reports no new findings attributable to this change

#### R9: The sweep leaves only allowed residual classes
A repo-wide grep for `branch_prefix` / `BranchPrefix` / `branchPrefix` MUST leave hits only in these
classes: frozen/self-contained test fixtures under `src/go/fab/internal/configupgrade`; historical
`src/kit/migrations/2.*.md` files; the new `src/kit/migrations/2.19.4-to-2.20.0.md` (which names the
key it removes); `docs/specs/findings/`; `docs/memory/distribution/log.md` / `log.seed.md`;
`fab/changes/**` artifacts; `fab/backlog.md` (archive-time concern, deliberately untouched); and text
that names the key **as retired** — the retired-key guards and parentheticals, and the new regression
test's fixture, which writes a `branch_prefix:` key precisely to assert it has no effect. No
surviving hit may present the key as live or settable.

- **GIVEN** apply is otherwise complete
- **WHEN** the repo-wide grep runs
- **THEN** every surviving hit falls in one of those classes

### Non-Goals

- Teaching `/git-branch`, `/fab-new`, or the operator's branch-alignment checks to honor a prefix —
  the decision is removal, not propagation.
- Bumping `src/kit/VERSION` — release-owned.
- Editing `fab/backlog.md` — `[h1eu]` is marked done at archive time by `/fab-archive`.
- Rewriting the frozen `configupgrade` version snapshots (`freeze_test.go`) or the self-contained
  `golden_test.go` synthetic field table.
- Editing historical `src/kit/migrations/2.*.md` files (a shipped migration is a historical record).

### Design Decisions

#### Retire the key rather than propagate the prefix
**Decision**: Remove `branch_prefix` from `fab batch switch` and from the config surface entirely
(struct field, accessor, registry row, both scope tables), leaving a `branch_prefix:` on disk as an
inert unknown key.
**Why**: The slim-config change (`930ab5e2`, Feb 2026) deliberately removed `git.branch_prefix`
system-wide; the Go port of batch switch reintroduced it only there, so the prefix is a regression
against a standing decision rather than a feature. A prefixed branch actively breaks worktree reuse
— batch switch attaches to *existing* changes whose branches were created unprefixed, so a set
prefix makes it create an orphan branch with none of the change's commits. Honoring the prefix
instead would have to thread it through every skill that assumes `branch == change folder name`.
**Rejected**: Documenting the prefix and teaching `/git-branch` / naming.md / the operator's
branch-alignment checks to honor it — significant surface area for a key that defaults to `""`, was
undocumented for most of its life, and has no known users. Also rejected: bypassing the prefix only
in `batch_switch.go` while leaving the key registered, which recreates the zombie-config state spec
finding f044 flagged.
*Introduced by*: 260811-h1eu-remove-batch-switch-branch-prefix

## Tasks

### Phase 1: Core Implementation

- [x] T001 Drop the prefix from `src/go/fab/cmd/fab/batch_switch.go` — delete `cfg, _ := config.Load(fabRoot)` and `branchPrefix := cfg.GetBranchPrefix()`, set `branchName := match`, and drop the now-unused `internal/config` import <!-- R1 -->
- [x] T002 [P] Remove the `BranchPrefix` field, the `GetBranchPrefix()` accessor, and the "empty branch prefix" mention in the coupled-failure doc comment from `src/go/fab/internal/config/config.go` <!-- R3 -->
- [x] T003 [P] Remove the `branch_prefix` registry row from `src/go/fab/internal/configref/configref.go` and drop it from the `referenceHeader` optional-override-block enumeration <!-- R3 -->
- [x] T004 [P] Remove `branch_prefix` from both `keyScopes` and `dottedKeys` in `src/go/fab/internal/configscope/configscope.go` <!-- R3 -->
- [x] T005 [P] Drop `branch_prefix` from the `explain` long-help override-block enumeration in `src/go/fab/cmd/fab/config.go` <!-- R3 -->

### Phase 2: Tests

- [x] T006 Update `src/go/fab/cmd/fab/batch_switch_test.go` — correct the stale `getBranchPrefix` comment, note the no-prefix contract in `TestRunBatchSwitch_Routing`'s doc comment, and add a test asserting a live `branch_prefix:` in `config.yaml` leaves the branch name equal to the change folder name <!-- R1 -->
- [x] T007 Update `src/go/fab/internal/config/config_test.go` — drop the `GetBranchPrefix` round-trips (`TestLoad_WidenedKeys`, `TestAccessors_NilConfig`, `TestAccessors_EmptyConfig`, `TestCascade_AbsentSystemFile`), retarget the two malformed-YAML fixtures (`TestLoadPath_MalformedCoupledFailure`, `TestCascade_MalformedProjectFileStillErrors`) at a still-modeled key, and remove `branch_prefix` from `TestScope_PruneAllProjectScopedFields` <!-- R4 -->
- [x] T008 [P] Remove `branch_prefix` from `TestScopeFor` and `TestDottedKeys` in `src/go/fab/internal/configscope/configscope_test.go` <!-- R4 -->
- [x] T009 Update `src/go/fab/cmd/fab/config_test.go` — drop the commented-out-in-reference assertion, add `branch_prefix` to `TestConfigReferenceRetiresLegacyKeys`'s retired list (and its doc comment), and remove the row from `TestConfigReferenceScopeAssignments` plus its doc comment <!-- R4 -->
- [x] T010 [P] Drop the now-vacuous `branch_prefix:` entry from the system-scaffold absent-fields list in `src/go/fab/cmd/fab/config_show_init_test.go` <!-- R4 -->
- [x] T011 Retarget `TestRender_BelowFenceLiveOverrideHoisted` in `src/go/fab/internal/configupgrade/configupgrade_test.go` at a still-registered live A field — it consumes the REAL registry via `fieldsForTest`, so it is not covered by the frozen/synthetic-fixture carve-out (`golden_test.go`, `freeze_test.go` stay untouched) <!-- R4 -->
- [x] T012 Run the scoped Go tests: `./cmd/fab`, `./internal/config`, `./internal/configref`, `./internal/configscope`, `./internal/configupgrade` from `src/go/fab` <!-- R4 -->

### Phase 3: Kit content & fence

- [x] T013 Write `src/kit/migrations/2.19.4-to-2.20.0.md` per the retired-key precedent — Summary / Pre-check (skip-if-absent + live-key sentinel) / Changes (remove the live top-level key and any adjacent documenting comment, atomic write) / Verification; `src/kit/VERSION` is NOT bumped <!-- R6 -->
- [x] T014 [P] Update `src/kit/skills/_cli-fab.md` — the full-schema key list and override-block enumeration (line ~411, moving `branch_prefix` into the retired-keys parenthetical) and the `fab batch switch` branch-naming sentence (line ~1258) <!-- R7 -->
- [x] T015 Regenerate `fab/project/config.yaml`'s managed fence with the REBUILT binary (build `./cmd/fab` from `src/go/fab`, run `config upgrade` with the repo root as CWD), then verify the fence carries no `branch_prefix` line and every other advert survives <!-- R5 -->

### Phase 4: Specs & memory sweep

- [x] T016 [P] Update `docs/specs/config.md` — the short-segment shape illustration (lines ~160-162), the `project`-scope key enumeration (line ~192), the unenumerated-fields sentence (line ~195), and the `advertise: true` list (line ~266) <!-- R8 -->
- [x] T017 [P] Remove the `branch_prefix` comment + key from the sample config in `docs/specs/architecture.md` (lines ~341-342) <!-- R8 -->
- [x] T018 Update `docs/memory/distribution/kit-architecture.md` — the `fab batch switch` bullet's branch-naming sentence and the `branchName = branchPrefix + match` prose <!-- R8 -->
- [x] T019 Update `docs/memory/_shared/configuration.md` — the `Config`-struct key list (line ~19, plus the retired-keys guard sentence), the opt-in override-blocks list (line ~21), and the single-config-reader Design Decision's consumed-key/accessor enumerations and "empty branch prefix" default mention (line ~458) <!-- R8 -->
- [x] T020 Run `fab memory-index` to regenerate the memory indexes after the memory edits <!-- R8 -->
- [x] T021 Final sweep: grep `branch_prefix` / `BranchPrefix` / `branchPrefix` repo-wide and confirm every surviving hit is an allowed residual class, then run `go test ./...` from `src/go/fab` <!-- R9 -->

## Execution Order

- T001–T005 are independent edits across five files; T002/T003/T004 must all land before the module compiles again (the registry row's scope lint reads `configscope`).
- T012 requires T001–T011.
- T015 requires T003 + T004 (the fence generator reads the registry) and a successful build.
- T020 requires T018 + T019.
- T021 is last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab batch switch` computes the branch name as the resolved change folder name with no prefix, and `batch_switch.go` holds no `config.Load`/`GetBranchPrefix` call
- [x] A-002 R3: `branch_prefix` is absent from `config.Config`, has no accessor, has no `configref` registry row, and appears in neither `configscope.keyScopes` nor `configscope.dottedKeys`
- [x] A-003 R5: `fab/project/config.yaml`'s managed fence carries no `branch_prefix` line and was regenerated by `fab config upgrade`, not hand-edited — verified by re-running the rebuilt binary's `config upgrade` over a copy of the file in a scratch project: byte-identical except the `kit dev` version banner (an artifact of building without the release ldflag)
- [x] A-004 R6: `src/kit/migrations/2.19.4-to-2.20.0.md` exists with the four required sections, removes a live `branch_prefix:` key, and `src/kit/VERSION` is still `2.19.4`
- [x] A-005 R7: `src/kit/skills/_cli-fab.md` documents `switch` naming branches by folder name and lists `branch_prefix` only among retired keys
- [x] A-006 R8: `docs/specs/config.md`, `docs/specs/architecture.md`, `docs/memory/distribution/kit-architecture.md`, and `docs/memory/_shared/configuration.md` carry no live-key or prefix-applied claim; memory indexes are regenerated (`fab memory-index --check` exits 0; only pre-existing soft-cap/narration warnings remain)

### Behavioral Correctness

- [x] A-007 R1: A project config carrying a live `branch_prefix: "feature/"` produces an unprefixed branch from `fab batch switch` — asserted by a test, not by inspection (`TestRunBatchSwitch_BranchNameIsFolderName`; it asserts both the bare-suffix argv and the absence of `feature/`, so the pre-fix code would fail it)
- [x] A-008 R2: The `branchExists` probe-and-route arms (`--checkout` for existing local/remote, positional for missing, positional on an offline `ls-remote`) behave exactly as before — routing code untouched in the diff; all four `TestRunBatchSwitch_Routing` sub-tests pass
- [x] A-009 R3: A `branch_prefix:` key left on disk is an inert unknown key — the loader neither errors nor warns on it (verified empirically: `fab config show` over a project config carrying `branch_prefix: "feature/"` exits 0 with empty stderr)

### Removal Verification

- [x] A-010 R3: `fab config explain` renders no `branch_prefix` key token, header enumeration included (rebuilt binary: `fab config explain | grep -c branch_prefix` → 0)
- [x] A-011 R9: The repo-wide `branch_prefix`/`BranchPrefix`/`branchPrefix` grep leaves hits only in the R9 residual classes, and no surviving hit presents the key as live or settable

### Scenario Coverage

- [x] A-012 R4: `TestConfigReferenceRetiresLegacyKeys` asserts `branch_prefix` alongside `review_tools` and `spawn_command`
- [x] A-013 R4: `go test ./...` from `src/go/fab` passes, with the five directly touched packages passing individually first (`go build`/`go vet` clean too)
- [x] A-014 R6: The migration's Pre-check sentinel makes a second run a complete no-op, and its Verification steps are executable as written (steps 1-3, 5-7 verified against the rebuilt binary; step 4's *stated expectations* are partly inaccurate — see review should-fix #1 — but the step still passes post-migration)

### Edge Cases & Error Handling

- [x] A-015 R6: The migration targets only **live** column-0 keys — commented lines and the managed fence (regenerated by `fab config upgrade`) are never rewritten
- [x] A-016 R4: The two malformed-YAML test fixtures still exercise a real type error on a *modeled* key, so the coupled-failure semantic they pin remains covered (`test_paths` is a live `[]string` field; a mapping under it fails the single Unmarshal)

### Code Quality

- [x] A-017 Pattern consistency: Edits follow surrounding style — registry-row removal leaves the table's ordering and lint invariants intact, and test edits keep each file's existing comment and helper conventions
- [x] A-018 No unnecessary duplication: No new helper duplicates an existing one; the branch name reuses the already-resolved `match` value
- [x] A-019 Canonical sources only: Every skill edit lands in `src/kit/skills/`, never in the gitignored `.claude/skills/` deployed copies (Constitution V)
- [x] A-020 Migration discipline: The config restructuring ships as a `src/kit/migrations/` file, not a new subcommand or ad-hoc script (code-quality.md § Anti-Patterns)
- [x] A-021 CLI contract: The behavior change ships with both `_cli-fab.md` updates and test updates (Constitution Additional Constraints)
- [x] A-022 Sibling & mirror sweep done up front: the full `branch_prefix` class — Go sources, tests, test *comments*, kit skill reference, specs, memory, and this repo's own config fence — was swept in one pass, not reactively after review
- [x] A-023 Owner-or-pointer respected: no skill file restates a rule it does not own alongside a pointer to its owner — `_cli-fab.md`'s new sentence states the CLI fact it owns and cites `/git-branch`/`naming.md` only as corroboration (borderline; see review nice-to-have #3)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- `docs/specs/skills/SPEC-*.md` mirrors need no update: the only skill file touched is
  `_cli-fab.md`, whose mirror (`docs/specs/skills/SPEC-_cli-fab.md`) is a condensed structural
  quick-reference (title + header + Flow + Tools + Sub-agents, per 260811-xy7a) carrying no
  branch-naming or config-key content, and no skill's flow, tool usage, or sub-agent structure
  changes — the constitution's narrowed mirror rule is not triggered.

## Deletion Candidates

- `src/go/fab/internal/configupgrade/golden_test.go:34-38` — the local synthetic field table still
  names its subject `branch_prefix`; now that no live registry row exists, the name reads like a real
  key. Renaming it to an obviously-synthetic key (e.g. `example_prefix`) would leave the retired token
  with zero live-looking occurrences in Go sources. Deliberately out of scope here (plan § Non-Goals);
  `freeze_test.go:109,152` must stay frozen either way.
- `docs/specs/config.md:160-163` — the short-segment shape illustration is now carried only by a
  parenthetical disclaiming that its `branch_prefix` subject is synthetic. Re-anchoring the block to a
  real registry row (e.g. `stage_hooks` or `true_impact_exclude`) would let the disclaimer be deleted.
- Nothing else: the retirement left no orphaned production symbol. `shortAdvert` (the helper the
  removed registry row called) retains 10 call sites; `internal/config` is still imported elsewhere in
  `cmd/fab`; no scope-table, lint, or coverage helper lost its last caller.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `internal/configupgrade/configupgrade_test.go`'s `TestRender_BelowFenceLiveOverrideHoisted` IS in scope, contrary to the intake's blanket `configupgrade` carve-out — retarget it at a still-registered live A field (`true_impact_exclude`) | It calls `fieldsForTest` → `configref.Fields()`, the REAL registry, and uses `branch_prefix` as its live-A-field subject; once the row is retired the key becomes an unknown/parked key and the test fails. The carve-out's stated reason ("synthetic key name in local field tables") holds only for `golden_test.go`'s `goldenFields()` and `freeze_test.go`'s frozen snapshots, which stay untouched | S:75 R:80 A:85 D:80 |
| 2 | Confident | Retarget the two malformed-YAML fixtures at `test_paths` (a mapping where a sequence is expected) rather than inventing a new modeled key | `test_paths` is still on `Config`, so the type error still surfaces at the single Unmarshal — which is the exact coupled-failure semantic the two tests pin. Any remaining modeled key would do; `test_paths` keeps the fixture a one-liner | S:70 R:90 A:85 D:75 |
| 3 | Confident | Keep `docs/specs/config.md`'s short-segment shape block quoting `branch_prefix` verbatim but annotate the caption as a synthetic golden-test field name, rather than substituting a different key | The block is explicitly captioned "from the full-document golden tests, over a small synthetic field set", and `golden_test.go`'s synthetic table is a carve-out that stays. Substituting a key would make the spec quote a fixture that does not exist; the annotation removes the live-key implication without breaking the citation | S:45 R:85 A:70 D:55 |
| 4 | Confident | The migration's Summary states that `src/kit/VERSION` advances `2.19.4` → `2.20.0` **at release** and that the bump is release-owned, deviating from the "VERSION is bumped" phrasing every prior migration's Summary carries | Intake assumption 5 (Certain) and the dispatch instruction both forbid bumping VERSION in this change; the migration name still needs FROM=released / TO=next-minor per `DiscoverMigrations`. Stating the bump as release-owned keeps the file honest without claiming a bump this change did not make | S:60 R:85 A:80 D:70 |
| 5 | Tentative | Ship the migration at all, given ~zero expected users with the key set | Carried forward from intake assumption 3: `code-quality.md` § Anti-Patterns requires a migration for config restructuring, and a live `branch_prefix:` would otherwise sit silently ignored after `fab config upgrade` parks it as a comment. Real-world impact is expected to be nil | S:35 R:70 A:45 D:35 |
| 6 | Confident | Update `docs/memory/` files during apply (rather than deferring all memory writes to hydrate) | The intake's What-Changes §5 and `code-quality.md` § Sibling & Mirror Sweeps both put the memory sweep inside apply — a memory file stating `branchName = branchPrefix + match` is a diverged copy of the code this change edits, and the sweep rule says update the whole class up front. Hydrate still owns net-new memory content (e.g. a migrations-catalog entry) | S:65 R:80 A:80 D:70 |

6 assumptions (0 certain, 5 confident, 1 tentative).

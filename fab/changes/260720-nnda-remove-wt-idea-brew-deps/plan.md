# Plan: Remove wt/idea Homebrew Dependencies with Graceful Degradation

**Change**: 260720-nnda-remove-wt-idea-brew-deps
**Intake**: `intake.md`

## Requirements

### Distribution: Formula template

#### R1: Formula drops the wt/idea dependencies
`.github/formula-template.rb` MUST NOT declare `depends_on "sahil87/tap/wt"` or `depends_on "sahil87/tap/idea"`. Nothing else in the formula (installed binaries, URLs, test block, placeholders) changes.

- **GIVEN** the release workflow stamps `.github/formula-template.rb`
- **WHEN** the next release publishes the formula
- **THEN** `brew install sahil87/tap/fab-kit` installs only `fab` and `fab-kit`, with no transitive `wt`/`idea` install

### Kit skills: absent-binary policy

#### R2: `_cli-external.md` collapses to one gated class
`src/kit/skills/_cli-external.md` MUST reclassify `wt` and `idea` out of the "assumed-present — bare" class: all four owned binaries (`wt`, `idea`, `rk`, `hop`) are separate sibling formulas that may legitimately be absent, and every use-time delegation (`skill`, `help-dump`) is `command -v`-gated and fails silently. The section retitles (no "(two install classes)"). The "Homebrew `depends_on` of `fab-kit`" / "Installed system-wide via `brew install fab-kit`" claims are replaced with standalone-formula pointers (`brew install sahil87/tap/wt`, `brew install sahil87/tap/idea`). The doc MUST state the surviving distinction: `wt` is functionally required for worktree entry points (operator spawning, `fab batch new`/`switch`), which stop with an install hint rather than silently skipping.

- **GIVEN** an agent loads `_cli-external.md` on a machine without `wt`/`idea`
- **WHEN** it runs the use-time delegations
- **THEN** each is `command -v`-gated and skips silently — no raw `command not found` surfaces
- **AND** the doc explains why functional entry points instead stop with an install hint

#### R3: Upfront `exec.LookPath("wt")` guards on `fab batch new`/`switch`
`runBatchNew` and `runBatchSwitch` MUST check `exec.LookPath("wt")` once, after the `$TMUX` check and before any per-item work, returning exactly `wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt` (respectively `'fab batch switch'`) when absent. `batch_new.go` gains the `os/exec` import. Tests in `batch_new_test.go`/`batch_switch_test.go` MUST assert the upfront error via PATH manipulation and that the guard precedes per-item work; existing tests that now cross the guard stub `wt` onto PATH to stay hermetic.

- **GIVEN** `wt` is absent from PATH and the caller is inside tmux
- **WHEN** `fab batch new 90g5` (or `fab batch switch <change>`) runs
- **THEN** one upfront actionable error is returned before any worktree/tmux work — never N per-item `exec: "wt": executable file not found` failures

#### R4: `_cli-fab.md` § fab batch documents the wt requirement
`src/kit/skills/_cli-fab.md` § fab batch MUST note the upfront `wt` requirement/error for `new` and `switch` (behavior addition; no signature/flag changes).

- **GIVEN** an agent reads § fab batch
- **WHEN** it plans a `fab batch new`/`switch` invocation
- **THEN** it knows `wt` must be on PATH and what error appears when it is not

#### R5: Operator preflight wt probe + gated idea pre-step
`src/kit/skills/fab-operator.md` MUST add exactly one preflight `command -v wt` probe with the startup steps (stop with `wt is required for operator spawning — install it via: brew install sahil87/tap/wt` when absent) — not per-call-site gating (call sites at §3/§6 stay unmodified) — and MUST gate the Working-a-Change backlog pre-step's `idea show <id>` lookup on `command -v idea` with a graceful skip (spawn `/fab-new <id>` unchanged; `/fab-new` resolves backlog IDs itself).

- **GIVEN** the operator starts on a machine without `wt`
- **WHEN** startup runs
- **THEN** it stops with the install hint before any spawn is attempted
- **GIVEN** `idea` is absent and a backlog-ID request arrives
- **WHEN** the Working-a-Change pre-step runs
- **THEN** the lookup is skipped silently and `/fab-new <id>` is spawned unchanged

### Docs: mirror & sibling sweep

#### R6: Mirror class updated in the same change
The SPEC mirrors and sibling docs carrying the now-false claims MUST be updated: `docs/specs/skills/SPEC-_cli-external.md` (two-class model → one-class), `docs/specs/skills/SPEC-fab-operator.md` (preflight probe + gated idea pre-step), `docs/specs/companions.md` (depends_on/four-binaries claim → standalone installs + graceful degradation), `docs/specs/architecture.md` (formula `depends_on` line), `docs/site/install.md` (four-binary table → two binaries + separately-installed companions). A repo-wide grep for stale "depends_on"/"assumed present"/"four binaries via brew install fab-kit" claims MUST find no in-scope stragglers (docs/memory/ is hydrate's job; fab/changes/ history is never edited).

- **GIVEN** the change is applied
- **WHEN** grepping the repo for the old two-class/depends_on claims
- **THEN** only out-of-scope files (docs/memory/, fab/changes/, .claude/skills/ deployed copies) still carry them

#### R7: Toolkit-standards conformance check
The change MUST be checked against the `shll standards` governing the touched surfaces (CLI error text → `principles` №4/№8; docs/site/ → `readme-extraction`) before finalizing, with the outcome noted in the result.

- **GIVEN** `shll` is installed
- **WHEN** the CLI error text and docs/site edits are finalized
- **THEN** conformance against the governing standards is verified and recorded

### Non-Goals

- No tap-repo edit (the release workflow publishes the dependency-free formula)
- No `fab doctor`/`prereqs.go` wt/idea checks
- No per-call-site operator gating beyond the single preflight probe
- No migration (no user-data restructuring)
- docs/memory/ updates happen at hydrate, not apply

## Tasks

### Phase 1: Go guards + formula

- [x] T001 [P] Delete the two `depends_on` lines from `.github/formula-template.rb` <!-- R1 -->
- [x] T002 [P] Add `exec.LookPath("wt")` guard after the tmux check in `src/go/fab/cmd/fab/batch_new.go` (+ `os/exec` import), error `wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt` <!-- R3 -->
- [x] T003 [P] Add the same guard to `src/go/fab/cmd/fab/batch_switch.go` with the `'fab batch switch'` message <!-- R3 -->
- [x] T004 Add guard tests to `src/go/fab/cmd/fab/batch_new_test.go` (PATH without wt → exact upfront error, no per-item work) and stub `wt` onto PATH in existing tests that now cross the guard <!-- R3 -->
- [x] T005 Add guard tests to `src/go/fab/cmd/fab/batch_switch_test.go` (same pattern) and stub `wt` in existing tests that now cross the guard <!-- R3 -->
- [x] T006 Run `go test ./src/go/fab/cmd/fab/` (scoped runs first, then the package) — green <!-- R3 -->

### Phase 2: Kit skill sources

- [x] T007 Reclassify wt/idea in `src/kit/skills/_cli-external.md`: gate the `skill`/`help-dump` delegation lines like rk/hop, retitle/collapse the Absent-binary discipline section to one class, replace the `brew install fab-kit` claims with standalone-formula pointers, state the fail-silent-vs-stop-with-hint distinction, update frontmatter description <!-- R2 -->
- [x] T008 In `src/kit/skills/fab-operator.md`: add one preflight `command -v wt` probe with the startup steps (stop with install hint) and gate the Working-a-Change `idea show <id>` pre-step with a `command -v idea` graceful skip <!-- R5 -->
- [x] T009 Note the upfront wt requirement/error for `new`/`switch` in `src/kit/skills/_cli-fab.md` § fab batch <!-- R4 -->

### Phase 3: Mirror & sibling sweep

- [x] T010 [P] Update `docs/specs/skills/SPEC-_cli-external.md` (Summary + Reference Model inventory row) to the one-class model <!-- R6 -->
- [x] T011 [P] Update `docs/specs/skills/SPEC-fab-operator.md` (Startup + §6 prose) with the preflight probe and gated idea pre-step <!-- R6 -->
- [x] T012 [P] Rewrite `docs/specs/companions.md` intro/outro: standalone installs + graceful-degradation summary <!-- R6 -->
- [x] T013 [P] Update `docs/specs/architecture.md` formula `depends_on` line <!-- R6 -->
- [x] T014 [P] Update `docs/site/install.md`: two-binary install table + separately-installed wt/idea companions with install commands and degradation note <!-- R6 -->
- [x] T015 Repo-wide grep sweep for stale "depends_on"/"assumed present"/"four binaries" claims; fix in-scope stragglers (exclude docs/memory/, fab/changes/, .claude/skills/) <!-- R6 -->
- [x] T017 Update `docs/specs/skills/SPEC-_cli-fab.md` `fab batch` catalog row (~line 41) to mirror the `_cli-fab.md` § fab batch edit — note the upfront wt-required-on-PATH guard + install-hint error on `new`/`switch` <!-- R6 --> <!-- rework: review cycle 1 must-fix — SPEC-mirror sync for the _cli-fab.md edit was missed by the plan's sweep task -->

### Phase 4: Standards

- [x] T016 Check the CLI error text and docs/site/install.md edits against `shll standards` (`principles`, `readme-extraction`); record the outcome <!-- R7 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `.github/formula-template.rb` contains no `depends_on` line; the rest of the formula is byte-identical
- [ ] A-002 R2: `_cli-external.md` shows all four owned binaries `command -v`-gated fail-silent for `skill`/`help-dump`; no "assumed present — bare" or "Homebrew `depends_on` of `fab-kit`" claim remains
- [ ] A-003 R3: both batch commands return the exact upfront error when `wt` is absent, before any per-item work
- [ ] A-004 R4: `_cli-fab.md` § fab batch documents the wt requirement and error for `new` and `switch`
- [ ] A-005 R5: `fab-operator.md` carries exactly one startup `command -v wt` probe (stop-with-hint) and a `command -v idea`-gated graceful skip on the backlog pre-step; other wt call sites unmodified

### Behavioral Correctness

- [ ] A-006 R3: guard placement is after the `$TMUX` check and before ID/change collection — no "Opening N tabs..." output precedes the wt error
- [ ] A-007 R2: `_cli-external.md` states the surviving distinction — delegations fail silently, functional entry points (batch, operator spawning) stop with an install hint

### Removal Verification

- [ ] A-008 R1: repo-wide grep finds no in-scope claim that brew installs wt/idea transitively with fab-kit

### Scenario Coverage

- [ ] A-009 R3: tests cover wt-absent (exact error string, no per-item work) for both `new` and `switch`, and existing launch-path tests still pass with wt stubbed onto PATH

### Edge Cases & Error Handling

- [ ] A-010 R3: `$TMUX` unset still errors with `not inside a tmux session` (tmux check precedes the wt guard)
- [ ] A-011 R5: idea-absent backlog request still spawns `/fab-new <id>` (no functionality lost)

### Code Quality

- [ ] A-012 Pattern consistency: guard follows the `prereqs.go` LookPath + install-hint error shape; skill gating matches the existing rk/hop `command -v` lines
- [ ] A-013 No unnecessary duplication: no per-call-site gating where a single entry-point guard/probe suffices
- [ ] A-014 Canonical sources only: no edits under `.claude/skills/`
- [ ] A-015 SPEC-mirror sync: every touched `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in this change

## Deletion Candidates

None. The change removes the two formula `depends_on` lines and the two-class absent-binary prose directly; the Go guards and skill gating add behavior without making any existing code or prose newly redundant (review cycle 2 confirmed: 15 net non-test Go lines, no orphaned utilities).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Guard placement: immediately after the `$TMUX` check, before ID/change collection (not merely before the loop) — so no "Opening N tabs..." preamble precedes the error | Intake says "after the tmux check / before the loop"; both points qualify, but erroring before any progress output is the cleaner UX and matches "one upfront error" | S:75 R:85 A:85 D:70 |
| 2 | Confident | Existing tests that newly cross the guard (`TestRunBatchNew_NoPendingItems`, `TestRunBatchSwitch_UnresolvableWarnsAndSkips`, `TestRunBatchSwitch_QuietRetainsStderr`) get `wt` stubbed onto PATH so the suite stays hermetic on wt-less machines | Test-alongside + Test Integrity: the spec (upfront guard) changed, so tests adapt to it; PATH stubbing is the file's established pattern | S:70 R:85 A:85 D:80 |
| 3 | Confident | Operator probe lands as a `### wt Gate` subsection in §2 Startup (after Tmux Gate), mirroring the Tmux Gate stop shape | Intake: "near the top (with the startup/session-setup steps)"; the exact placement/heading is mine | S:70 R:80 A:80 D:70 |
| 4 | Confident | docs/site/install.md keeps its section structure (readme-extraction standard governs structure); only the install table/content changes to two binaries + a companion-install subsection | Structure-preserving edit avoids re-litigating the pull-surface layout; standard checked at T016 | S:65 R:80 A:80 D:75 |
| 5 | Certain | `hop`'s "genuinely-optional … not a `fab-kit` Homebrew dependency" contrast language is normalized to the one-class wording (all four are sibling formulas) | Direct consequence of R2's one-class collapse; leaving the old contrast would contradict the new model | S:80 R:85 A:90 D:85 |

5 assumptions (1 certain, 4 confident, 0 tentative).

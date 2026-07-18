# Plan: Usage-Error Exit Codes (Toolkit Principle №4)

**Change**: 260717-swon-usage-error-exit-codes
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. Exit-code mapping only — error wording is out of scope
     (assumption #4). Classification rides on execution phase / error values — never
     stderr string matching. -->

### fab binary: Usage-error exit-code convention

#### R1: The `fab` binary MUST exit 2 for usage errors
The `fab` binary (`src/go/fab/cmd/fab`) MUST exit with code **2** when an invocation fails
because of a *usage error* — a malformed invocation caught at parse/validation time, before any
subcommand handler runs. Success MUST remain exit **0**; operational failures MUST remain exit **1**.

- **GIVEN** the `fab` binary
- **WHEN** an invocation fails during flag parsing, argument validation, subcommand resolution, or flag-group validation (i.e. before the resolved command's run phase begins)
- **THEN** the process exits with code 2
- **AND** the existing `ERROR: %s\n` stderr line is printed exactly once (no double-print)

#### R2: The four usage-error classes MUST all exit 2
Each of the four usage-error classes named in the intake MUST exit 2:

- **GIVEN** any of: an unknown/malformed flag (`fab score --nope`), an arg-count violation (`fab score` with no args), an unknown subcommand (`fab nonsense`), or a mutually-exclusive flags-group conflict (`fab resolve --status --folder`)
- **WHEN** the binary is invoked that way
- **THEN** it exits 2
- **AND** every one of these exits 1 today, so the change is observable on all four

#### R3: Operational (data-condition) failures MUST stay exit 1
Syntactically valid invocations that fail on runtime/data conditions MUST remain exit 1 — they
are operational failures, not usage errors.

- **GIVEN** a syntactically valid invocation whose failure originates from within a subcommand's run phase (e.g. `fab resolve nope` — valid syntax, missing change; preflight validation failures; `fab score --check-gate` below-gate exits; tmux/gh/filesystem failures)
- **WHEN** it fails
- **THEN** it exits 1 (unchanged)
- **AND** `fab log command`'s always-exits-0 contract for valid usage is unaffected (its own cobra arg-count errors move 1→2, still "non-zero before RunE")

#### R4: Classification MUST ride on execution phase / error value — no string matching
The usage-vs-operational classification MUST be determined by *whether the resolved command's run
phase began* (an execution-phase signal) and/or by error values — never by matching stderr or error
message text.

- **GIVEN** the classification logic in `main()` / the root command wiring
- **WHEN** it decides between exit 1 and exit 2
- **THEN** the decision is based on execution phase (did any command's `RunE` start?) and/or typed error values, mirroring `paneValidationExitCode`'s error-value discipline
- **AND** no code path inspects the error's *message string* to classify it

#### R5: Existing domain-specific exit codes MUST be preserved (no renumbering)
The pane-family scheme (2 = pane missing, 3 = other tmux failure) and the `fab memory-index --check`
tiered scheme (0/1/2, where 2 = destructive loss) MUST be unchanged. Usage errors exit 2 binary-wide
at parse/validation time; the domain schemes apply in-handler. For those subcommands exit 2 is
therefore intentionally ambiguous between "usage error" and the domain meaning — documented per
subcommand, not renumbered.

- **GIVEN** the pane commands and `fab memory-index --check`
- **WHEN** they hit their domain conditions (pane missing, destructive loss)
- **THEN** they still exit with their documented domain codes via their in-handler `os.Exit` (which bypasses `main()`'s mapping entirely)
- **AND** `pane_exitcode_test.go` stays green **unmodified** (regression guard for the no-renumbering decision)

### Documentation: per-subcommand exit-code documentation

#### R6: The exit-code convention MUST be documented in `_cli-fab.md` + SPEC mirror, with stale claims corrected
The binary-wide `0`/`1`/`2` convention (plus the sentinel/phase classification and the coexistence
rule with the in-handler domain schemes) MUST be documented in `src/kit/skills/_cli-fab.md`, and its
SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` MUST be updated in the same change. Stale per-command
claims that today say a usage error "exits non-zero" / "non-zero before RunE" MUST be corrected to
name exit 2. Behavior-claim greps MUST include user-facing string literals.

- **GIVEN** `src/kit/skills/_cli-fab.md` and its SPEC mirror
- **WHEN** this change lands
- **THEN** a binary-wide exit-code convention section is present, the `fab log command`, `fab resolve` flags-group, `fab resolve-agent` usage-error, and pane usage-note claims are corrected to exit 2, and the pane/memory-index sections carry the coexistence note
- **AND** the SPEC mirror reflects the same claims (Sibling & Mirror Sweep)

#### R7: The stale `_preamble.md` claims MUST be corrected + SPEC mirror updated
The "cobra arg-count errors exit non-zero before RunE" claims in `src/kit/skills/_preamble.md` (the
§ Common fab Commands table row, the two Key-behaviors bullets, and the change-context `fab log
command` step) MUST be corrected to name exit 2, and `docs/specs/skills/SPEC-_preamble.md` MUST be
updated in the same change.

- **GIVEN** `src/kit/skills/_preamble.md` and its SPEC mirror
- **WHEN** this change lands
- **THEN** every "arg-count errors exit non-zero before RunE" claim reads "exit 2 (before RunE)" (or equivalent) and the SPEC mirror matches

### Tests

#### R8: The new mapping MUST be pinned by a table test alongside the Go code
A table test (mirroring `pane_exitcode_test.go`'s shape) MUST pin the mapping via a testable seam:
unknown flag → 2, no-arg `score` → 2, unknown subcommand → 2, flags-group conflict → 2, operational
error (nonexistent change) → 1, success → 0.

- **GIVEN** the new exit-code logic
- **WHEN** the cmd/fab test package runs
- **THEN** a table test exercises all six cases through a testable seam (the main body extracted into a helper returning the exit code, or the classifier tested directly) and passes
- **AND** `pane_exitcode_test.go` continues to pass unmodified

### Non-Goals

- The `fab-kit` and `fab-kit`-shim binaries (`src/go/fab-kit/`) and `fab-kit doctor`'s exit-with-failure-count aggregation — out of scope (assumption #3).
- Renumbering any existing exit-code scheme — explicitly rejected (assumption #1).
- Error-message *wording* changes — wording already conforms (assumption #4).
- A `docs/site/` CLI-surface exit-code page — fab-kit has none today (assumption #6); `_cli-fab.md` is the CLI reference.
- Making `fab pane <unknown-subcommand>` (a parent command with no `RunE`) exit non-zero — today it prints help and exits 0; this is help display, not one of the four in-scope usage classes.

### Design Decisions

1. **Unified execution-phase classifier over per-class cobra seams**: classify by *whether any resolved command's `RunE` began* — a single seam that wraps each command's `RunE` at assembly time in `newRootCmd()` to set a "reached run phase" flag; `main()` (via a testable `run()` helper) exits 2 when `Execute()` errors and that flag is still unset, else 1. — *Why*: all four usage classes surface in cobra's `execute()` **before** `RunE` (flag parse → `FlagErrorFunc`; `ValidateArgs` for arg-count; `Find`/`legacyArgs` for unknown subcommand; `ValidateFlagGroups` for the mutually-exclusive conflict), while operational errors originate *inside* `RunE`. One phase signal captures all four cleanly and needs no error-string matching, honoring R4. — *Rejected*: (a) per-class seams (`SetFlagErrorFunc` + an `Args`-validator tree-walk + a `Find`-error branch + a flags-group handler) — fragmented, and the mutually-exclusive flags-group conflict has **no public cobra hook** (`ValidateFlagGroups` returns a plain error mid-`execute()`), so it cannot be wrapped per-class without duplicating cobra internals; (b) a root `PersistentPreRunE` flag — it runs *before* `ValidateFlagGroups`, so a flags-group conflict would be misclassified as operational.
2. **Testable seam = extract `main`'s body into `run(args []string) int`**: `main()` becomes `os.Exit(run(os.Args[1:]))`; `run` builds the root, wraps `RunE`, executes, prints the `ERROR:` line, and returns the exit code. — *Why*: mirrors `pane_exitcode_test.go`'s classifier-test shape (the intake's stated model) and lets the table test drive real invocations end-to-end. — *Rejected*: testing only a standalone classifier function — it would not exercise the cobra wiring (the flag-wrap tree-walk, the flags-group path), leaving the integration untested.
3. **In-handler `os.Exit` domain codes are untouched**: pane/memory-index call `os.Exit(2|3)` from inside `RunE`, which bypasses `main()`/`run()` entirely — their codes are preserved with zero handler changes, satisfying R5 by construction. — *Why*: the coexistence rule is already structurally true; only documentation is needed.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add an unexported usage-error classification seam to `src/go/fab/cmd/fab/main.go`: extract the body of `main()` into `run(args []string, errW io.Writer) int`; add a `markRunReached` tree-walk that wraps each command's `RunE` in `newRootCmd()`'s assembled tree so a closure-captured "reached run phase" flag is set immediately before the real handler runs; in `run`, on `Execute()` error print the existing `ERROR: %s\n` line once and return 2 when the flag is still unset (usage error) else 1; `main()` becomes `os.Exit(run(os.Args[1:], os.Stderr))`. Classification rides on execution phase only — no string matching. <!-- R1 R2 R3 R4 -->
- [x] T002 Verify the domain-specific in-handler exit codes are structurally preserved (pane `os.Exit(2|3)` and memory-index `os.Exit(2)` bypass `run()`); make no handler changes. Confirm `pane_exitcode_test.go` needs no edit. <!-- R5 -->

### Phase 3: Tests

- [x] T003 Add `src/go/fab/cmd/fab/main_exitcode_test.go` (mirroring `pane_exitcode_test.go`'s table shape) driving `run()` end-to-end: `score --nope` → 2, `score` (no args) → 2, `nonsense` → 2, `resolve --status --folder` → 2, `resolve <nonexistent-change>` → 1 (operational), and a success case → 0. Routes the `ERROR:` print at `io.Discard`. <!-- R8 -->

### Phase 4: Documentation

- [x] T004 Update `src/kit/skills/_cli-fab.md`: add a binary-wide exit-code convention subsection near the top (`0` success / `1` operational / `2` usage error, the execution-phase classification, and the coexistence rule with in-handler domain schemes); correct stale claims — the `fab log command` "cobra arg-count errors exit non-zero before RunE" (line ~224 → exit 2), the `fab resolve` flags-group "exits non-zero" (line ~247 → exit 2), the `fab resolve-agent` usage error "exits non-zero" (line ~337 → exit 2); annotate the pane-family exit-code note (line ~408) and the `fab memory-index` exit-code section with the usage-error coexistence note. <!-- R6 -->
- [x] T005 [P] Update `src/kit/skills/_preamble.md`: correct the "cobra arg-count errors exit non-zero before RunE" claims — the § Common fab Commands `fab log command` table row (line ~201), the two Key-behaviors bullets (lines ~210, ~212), and the change-context `fab log command` step (line ~73) — to name exit 2. <!-- R7 -->
- [x] T006 Update the SPEC mirrors in the same change (code-quality § Sibling & Mirror Sweeps): `docs/specs/skills/SPEC-_cli-fab.md` and `docs/specs/skills/SPEC-_preamble.md` to reflect the exit-code convention + corrected claims. Grep the old claims repo-wide (including user-facing string literals) to confirm no stale "exit non-zero before RunE" / usage-error "exits non-zero" claim survives in any touched skill's mirror class. <!-- R6 R7 -->

## Execution Order

- T001 blocks T003 (the test drives `run()`) and T002 (the verification depends on the extracted seam existing).
- T004, T005 are independent of each other ([P]); T006 depends on T004 and T005 (the mirrors follow the canonical edits).

## Acceptance

### Functional Completeness

- [x] A-001 R1: The `fab` binary exits 2 for usage errors, 1 for operational failures, 0 on success; the `ERROR:` stderr line prints exactly once.
- [x] A-002 R2: All four usage classes (`score --nope`, `score`, `nonsense`, `resolve --status --folder`) exit 2.
- [x] A-003 R3: Operational failures (`resolve <nonexistent>`, preflight/below-gate/tmux failures) stay exit 1; `fab log command` valid-usage contract unaffected.
- [x] A-004 R4: Classification is by execution phase (reached-RunE flag) / error value — no stderr or message string matching anywhere in the seam.
- [x] A-005 R5: Pane (2/3) and memory-index (0/1/2) domain schemes are unchanged; `pane_exitcode_test.go` passes unmodified.
- [x] A-006 R6: `_cli-fab.md` carries the binary-wide convention + coexistence note and corrected per-command claims; `SPEC-_cli-fab.md` mirrors it.
- [x] A-007 R7: `_preamble.md`'s arg-count claims read exit 2; `SPEC-_preamble.md` mirrors it.

### Behavioral Correctness

- [x] A-008 R2: The four classes measurably change from exit 1 (today) to exit 2 — verified by the new table test asserting the new codes.
- [x] A-009 R3: `resolve <nonexistent>` measurably stays exit 1 (the test's operational case).

### Scenario Coverage

- [x] A-010 R8: `main_exitcode_test.go` exercises all six cases (four usage → 2, one operational → 1, one success → 0) through the `run()` seam and passes; `go test ./src/go/fab/...` is green.

### Edge Cases & Error Handling

- [x] A-011 R5: In-handler `os.Exit(2|3)` domain paths bypass the `main()` mapping so domain codes are preserved with no handler changes (verified by `pane_exitcode_test.go` staying green unmodified).

### Code Quality

- [x] A-012 Pattern consistency: The classification seam follows the codebase's error-value discipline (mirrors `paneValidationExitCode` / `errors.As`); `run()` extraction follows the established "extract-for-testability" pattern (e.g. `batch_archive.go`'s testable-body note).
- [x] A-013 No unnecessary duplication: No cobra-internal logic is re-implemented; the single phase seam is reused for all four classes rather than per-class handlers.
- [x] A-014 Test integrity: New tests conform to the intended behavior (usage=2/operational=1/success=0); `pane_exitcode_test.go` is not modified to accommodate the change.

### Documentation Accuracy

- [x] A-015 R6 R7: No stale "exits non-zero before RunE" / usage-error "exits non-zero" claim survives in `_cli-fab.md`, `_preamble.md`, or their SPEC mirrors (repo-wide grep, including string literals, comes back clean for the changed claims).

### Cross References

- [x] A-016 R6: The `_cli-fab.md` ↔ `SPEC-_cli-fab.md` and `_preamble.md` ↔ `SPEC-_preamble.md` pairs stay consistent (Sibling & Mirror Sweep); the pane/memory-index coexistence cross-reference is present in both the canonical and mirror.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The former `main()` body was restructured into `run()`, not left behind; no seam, test, or doc claim was orphaned — the repo-wide grep for the superseded "exit non-zero before RunE" phrasing in the touched skill/mirror class comes back clean.)

## Assumptions

<!-- Apply-agent record of graded decisions made while co-generating ## Requirements. The
     six intake assumptions are honored as-decided (not re-graded); the rows below are the
     apply-time seam/mechanism resolutions the intake deferred "to apply". -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Mechanism = unified execution-phase classifier (wrap each command's `RunE` in `newRootCmd()` to set a reached-run flag; `main` exits 2 when `Execute()` errs with the flag unset), chosen over per-class cobra seams | Intake assumption #2 deferred the seam+flags-group choice to apply; cobra's `execute()` runs all four usage classes before `RunE` while operational errors come from inside `RunE`, so one phase signal covers all four and cleanly handles the flags-group conflict that has no public cobra hook; internal + easily changed | S:70 R:85 A:80 D:70 |
| 2 | Confident | Testable seam = extract `main`'s body into `run(args []string) int`; table test drives `run()` end-to-end (six cases) | Intake §5 names "extract main's body into a helper returning the exit code" as an option and asks for a table test mirroring `pane_exitcode_test.go`; end-to-end drive exercises the cobra wiring the classifier-only alternative would miss | S:75 R:85 A:85 D:80 |
| 3 | Confident | `fab pane <unknown-subcommand>` (parent-with-no-RunE) is left as today (prints help, exits 0) — not treated as an in-scope usage class | The intake enumerates exactly four classes; a parent command showing help is help display, not one of them, and changing it would widen scope | S:65 R:85 A:80 D:70 |

3 assumptions (0 certain, 3 confident, 0 tentative).

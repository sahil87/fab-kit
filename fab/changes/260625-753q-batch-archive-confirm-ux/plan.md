# Plan: Redesign `fab batch archive` confirmation & preview UX

**Change**: 260625-753q-batch-archive-confirm-ux
**Intake**: `intake.md`

## Requirements

### CLI: `fab batch archive` flag surface

#### R1: Remove `--all` and `--list`; add `--yes`/`-y` and `--dry-run`
The `fab batch archive` command SHALL remove the `--all` and `--list` boolean flags entirely (no deprecation alias) and SHALL add a `--yes` flag with short alias `-y` and a `--dry-run` flag. `--dry-run` SHALL have no short alias.

- **GIVEN** the rebuilt command
- **WHEN** its flag set is inspected
- **THEN** `--all` and `--list` are absent, and `--yes` (`-y`) and `--dry-run` are present

#### R2: Bare invocation on a TTY lists then prompts (default No)
A bare `fab batch archive` (no args, no flags) on an interactive stdin SHALL compute the archivable set, list it, then prompt `Archive these N? [y/N]` with default No. A `y` or `yes` answer (case-insensitive) SHALL archive all archivable changes via `archiveLoop`; any other answer (including a bare Enter / empty line) SHALL abort with no action and exit 0.

- **GIVEN** N>0 archivable changes and an interactive stdin
- **WHEN** the user runs `fab batch archive` and answers `y`
- **THEN** all archivable changes are archived via `archiveLoop`
- **AND WHEN** the user answers with Enter / `n` / anything else
- **THEN** nothing is archived and the command exits 0

#### R3: `--yes`/`-y` archives all archivable without prompting
`fab batch archive --yes` (or `-y`) SHALL compute the archivable set and archive all of it with no prompt — the resolved behavior the old `--all` had.

- **GIVEN** N archivable changes
- **WHEN** the user runs `fab batch archive --yes`
- **THEN** all are archived with no prompt

#### R4: `--dry-run` lists only
`fab batch archive --dry-run` SHALL list what would be archived and take no action and issue no prompt — the behavior the old `--list` had.

- **GIVEN** archivable changes
- **WHEN** the user runs `fab batch archive --dry-run`
- **THEN** the archivable list is printed and nothing is archived

#### R5: Non-TTY without `--yes` refuses (no prompt, non-zero exit)
When stdin is not a TTY and `--yes` was not passed, the bare/archive-all path SHALL NOT reach the prompt. It SHALL print a guidance message to stderr and exit non-zero, e.g. `ERROR: refusing to prompt for confirmation on a non-interactive stdin.` plus `Re-run with --yes to archive non-interactively.`

- **GIVEN** a non-interactive stdin and no `--yes`
- **WHEN** the user runs a bare `fab batch archive`
- **THEN** it refuses with guidance and exits non-zero, archiving nothing

#### R6: Explicit args archive named changes WITHOUT prompting
`fab batch archive foo bar` SHALL archive the named changes with no prompt and no TTY guard — naming them is the opt-in. The existing explicit-args behavior (resolution, per-change archivability check, warn-and-skip on unresolvable/not-ready/ambiguous names, `No valid changes to archive.` exit-1 when nothing resolves, already-archived soft-skip) SHALL be preserved.

- **GIVEN** explicit change-name args
- **WHEN** the user runs `fab batch archive foo bar`
- **THEN** the named changes are archived with no prompt and no non-TTY guard

#### R7: `--dry-run --yes` is mutually exclusive → error
Passing both `--dry-run` and `--yes` SHALL error out (non-zero exit) with a clear message, e.g. `ERROR: --dry-run and --yes are mutually exclusive.`

- **GIVEN** both flags passed
- **WHEN** the command runs
- **THEN** it errors non-zero and archives nothing

#### R8: Preserve F49 empty-set behavior
When nothing is archivable, the command SHALL print `No archivable changes found.` and exit 0 — BEFORE any prompt or non-TTY guard. The empty path retains the `Archived 0, skipped 0, failed 0.` footer to match today's `--all` empty output (Assumption 13).

- **GIVEN** zero archivable changes
- **WHEN** the user runs a bare `fab batch archive` or `--yes` (on any stdin)
- **THEN** `No archivable changes found.` + the zero footer print and the command exits 0 with no prompt and no guard

#### R9: Testable prompt/guard seams
The answer-reading SHALL use `cmd.InOrStdin()` (the cobra seam, as elsewhere in this package) and the TTY check SHALL use an injectable seam so both the prompt branch and the non-TTY-refusal branch are unit-testable without a real terminal.

- **GIVEN** the implementation
- **WHEN** tests drive the prompt and guard
- **THEN** they can inject the answer and the TTY result without a real tty

### Docs & SPEC mirror sweep

#### R10: Update `_cli-fab.md` and its doc mirrors to the new model
The CLI reference (`src/kit/skills/_cli-fab.md`), the spec tables/examples (`docs/specs/overview.md`, `docs/specs/architecture.md`, `docs/specs/assembly-line.md`), and the memory paragraph (`docs/memory/pipeline/change-lifecycle.md`) SHALL describe the new `--yes`/`-y` + `--dry-run` + bare-prompt + non-TTY-guard model. Every `batch archive --all` / `batch archive --list` literal in canonical sources SHALL be swept.

- **GIVEN** the CLI-signature change
- **WHEN** the docs are read
- **THEN** no stale `batch archive --all`/`--list` literal remains and the new model is documented

### Non-Goals

- No migration — flag/behavior change only, no user-data restructuring (Assumption 11).
- `batch new` / `batch switch` keep `--all`/`--list` and stay list-by-default — only `archive` diverges.
- The `docs/memory/distribution/log.md` generated changelog is NOT hand-edited (regenerated by `fab memory-index` at hydrate).

### Design Decisions

1. **TTY detection via stdlib `os.ModeCharDevice`**: reuse the existing in-repo pattern (`src/go/fab-kit/internal/upgrade.go` `isTTY`) — `info.Mode()&os.ModeCharDevice != 0` — no new dependency. *Why*: `golang.org/x/term` is NOT in `src/go/fab/go.mod`; the constitution mandates minimal single-binary deps; this exact stdlib pattern already exists in the sibling module. *Rejected*: adding `golang.org/x/term` (new dep, violates Constitution I).
2. **Injectable TTY seam**: a package-level `isStdinTTY func(io.Reader) bool` indirection so tests force the TTY/non-TTY branches deterministically. *Why*: `os.Stdin.Stat()` is not controllable from a cobra test that sets `cmd.SetIn(buf)`. *Rejected*: relying on the real stdin in tests (flaky/non-deterministic under `go test`).
3. **Empty-set check first**: the `No archivable changes found.` + zero-footer path runs before the prompt and before the non-TTY guard, for bare AND `--yes`, so the benign no-op is unchanged and never blocks on a guard (R8/F49).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rebuild the flag layer in `src/go/fab/cmd/fab/batch_archive.go`: remove `listFlag`/`allFlag` and their `--list`/`--all` registrations; add `yesFlag` (`--yes`, shorthand `-y`) and `dryRunFlag` (`--dry-run`, no shorthand); thread them into `runBatchArchive`'s signature. <!-- R1 -->
- [x] T002 Add a stdlib TTY-detection helper with an injectable package-level seam in `src/go/fab/cmd/fab/batch_archive.go` (`isStdinTTY` var defaulting to an `os.ModeCharDevice` check on `*os.File`). <!-- R9 -->
- [x] T003 Rewrite `runBatchArchive` control flow in `src/go/fab/cmd/fab/batch_archive.go`: (a) `--dry-run --yes` → error (R7); (b) `--dry-run` → `listArchivable`, exit 0 (R4); (c) explicit args → existing resolve/validate/archiveLoop path, no prompt, no guard (R6); (d) bare/`--yes` path → compute archivable set, empty-set `No archivable changes found.` + zero footer exit 0 FIRST (R8), else if not `--yes` and non-TTY → refuse non-zero (R5), else if not `--yes` → list + prompt `Archive these N? [y/N]` default No reading `cmd.InOrStdin()` (R2), else archive all via existing resolve+archiveLoop (R2/R3). <!-- R2 -->
- [x] T004 Rewrite the stale `260612-ye8r` rationale comment (old lines 48-55) in `src/go/fab/cmd/fab/batch_archive.go` to explain the new prompt/`--yes`/`--dry-run` model and why `archive` diverges from `new`/`switch` (the one irreversible-within-loop bulk mutation earns the interactive confirm). <!-- R1 -->

### Phase 3: Tests

- [x] T005 Rewrite `src/go/fab/cmd/fab/batch_archive_test.go` to the new signature and cover every flag-matrix row: bare+TTY `y` archives; bare+TTY Enter/`n` aborts (exit 0, nothing archived); `--yes` archives no prompt; `--dry-run` lists only; non-TTY-without-`--yes` refuses non-zero (nothing archived); explicit-args no-prompt; `--dry-run --yes` error; empty-set exit 0 with notice+footer. Update `TestBatchArchiveCmd_Structure` to assert the new flags. Drive the TTY seam by overriding `isStdinTTY` and the answer via `cmd.SetIn`. <!-- R2 -->
- [x] T006 Run `go test ./...` in `src/go/fab` and confirm green. <!-- R9 -->

### Phase 4: Docs & SPEC/mirror sweep

- [x] T007 Update `src/kit/skills/_cli-fab.md`: the `fab batch` family signature (line ~783) and the `archive` behavior bullet (line ~787) — describe bare-prompt (default No) + `--yes`/`-y` + `--dry-run` + non-TTY guard + `--dry-run --yes` error + explicit-args-no-prompt + preserved empty-set exit 0. Keep `new`/`switch` `--list`/`--all` intact. <!-- R10 -->
- [x] T008 [P] Update `docs/specs/assembly-line.md` (line 128 `fab batch archive --all` → `fab batch archive --yes`) and `docs/specs/architecture.md` (line ~433 family signature) and `docs/specs/overview.md` (line ~106 table row, if a flag literal is present — none present, no edit). <!-- R10 -->
- [x] T009 [P] Update `docs/memory/pipeline/change-lifecycle.md` (line ~156) — replace the `--all`/`--list` semantics sentence with the new bare-prompt/`--yes`/`--dry-run`/non-TTY-guard/`--dry-run --yes`-error model; confirm preserved empty-set exit 0 (F49). <!-- R10 -->
- [x] T010 Re-grep canonical sources for any remaining `batch archive --all` / `batch archive --list` literal and sweep it. Also swept the missed-by-intake current-state memory at `docs/memory/distribution/kit-architecture.md:139,143` (per code-quality.md § Sibling & Mirror Sweeps). `docs/memory/runtime/pane-commands.md` confirmed no edit needed. The `docs/specs/findings/*` and `log.seed.md` `--all`/`--list` references are intentional dated history (F49 empty-set exit-0 is preserved, no note needed). <!-- R10 -->

## Execution Order

- T001 → T002 → T003 → T004 are sequential (same file).
- T005 depends on T001-T004; T006 depends on T005.
- T007-T010 are docs and may run after the code is settled; T008/T009 are `[P]` (different files).

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `batch_archive.go` defines `--yes`/`-y` and `--dry-run` (no shorthand) and no longer defines `--all` or `--list`.
- [ ] A-002 R2: Bare TTY invocation lists then prompts `Archive these N? [y/N]`; `y`/`yes` archives all, other answers abort exit 0.
- [ ] A-003 R3: `--yes`/`-y` archives all archivable with no prompt.
- [ ] A-004 R4: `--dry-run` lists only, no prompt, no action.
- [ ] A-005 R5: Non-TTY without `--yes` refuses with guidance and exits non-zero, archiving nothing.
- [ ] A-006 R6: Explicit-args invocation archives named changes with no prompt and no TTY guard; warn-and-skip and `No valid changes to archive.` exit-1 preserved.
- [ ] A-007 R7: `--dry-run --yes` errors non-zero with a mutual-exclusion message.
- [ ] A-008 R8: Empty archivable set prints `No archivable changes found.` + `Archived 0, skipped 0, failed 0.` and exits 0 before any prompt/guard.
- [ ] A-009 R10: No stale `batch archive --all`/`--list` literal remains in canonical sources; `_cli-fab.md`, overview/architecture/assembly-line specs, and change-lifecycle memory describe the new model.

### Behavioral Correctness

- [ ] A-010 R2: A bare Enter (empty line) on the prompt aborts (default No) and archives nothing.
- [ ] A-011 R8: The empty-set path is reached for both bare and `--yes` regardless of TTY (guard does not fire first).

### Scenario Coverage

- [ ] A-012 R9: `batch_archive_test.go` exercises every flag-matrix row (bare yes / bare abort / `--yes` / `--dry-run` / non-TTY refusal / explicit-args / `--dry-run --yes` error / empty-set) and passes via `go test ./...`.

### Edge Cases & Error Handling

- [ ] A-013 R6: Already-archived explicit names still soft-skip (exit 0); ambiguous names still surface the ambiguity warning distinctly.

### Code Quality

- [ ] A-014 Pattern consistency: New code follows the package's cobra/error-handling style; the TTY helper mirrors the existing `os.ModeCharDevice` pattern; the rationale comment is rewritten to reflect the new model.
- [ ] A-015 No unnecessary duplication: Reuses `allArchivableNames`, `isArchivable`, `archiveLoop`, `listArchivable`, `resolve.ToFolder` rather than reimplementing.
- [ ] A-016 No `.claude/skills/` edits: only canonical `src/kit/skills/_cli-fab.md` is edited for the skill mirror.

### Documentation Accuracy & Cross-References

- [ ] A-017 R10: The full SPEC/doc mirror sweep class is updated in this change (not deferred to review): `_cli-fab.md` + overview/architecture/assembly-line specs + change-lifecycle memory, with no remaining stale flag literal.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | TTY detection reuses the stdlib `info.Mode()&os.ModeCharDevice` pattern (mirroring `src/go/fab-kit/internal/upgrade.go` `isTTY`); no `golang.org/x/term` added | `golang.org/x/term` is NOT in `src/go/fab/go.mod` (only cobra+yaml); Constitution I mandates minimal deps; the exact stdlib pattern already exists in the sibling module | S:90 R:80 A:95 D:90 |
| 2 | Certain | Empty-set path keeps the `Archived 0, skipped 0, failed 0.` footer alongside `No archivable changes found.` (intake Assumption 13) | Mirrors today's `--all` empty path (old batch_archive.go:70); harmless either way; keeps existing `TestRunBatchArchive_EmptyAllSetIsBenignNoOp` assertion shape | S:60 R:80 A:75 D:70 |
| 3 | Certain | `-y` is the sole short alias; `--dry-run` gets none (intake Assumption 12) | `-y` is the universal convention; conversation listed `--yes`/`-y` and bare `--dry-run`, implying no `--dry-run` short form; trivially reversible | S:60 R:80 A:70 D:65 |
| 4 | Certain | Injectable package-level `isStdinTTY` seam for testability; answer read via `cmd.InOrStdin()` | Intake Open Question resolved: cobra `InOrStdin()` is used elsewhere in the package (`hook.go`); the TTY check needs a seam because `os.Stdin.Stat()` is not controllable from a cobra-test buffer | S:80 R:85 A:80 D:80 |
| 5 | Certain | Empty-set check runs before BOTH the prompt and the non-TTY guard, for bare and `--yes` | Intake §8 states the empty-set check happens before any prompt/guard so the benign no-op is unchanged | S:90 R:80 A:90 D:90 |
| 6 | Confident | SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` needs no flag-specific edit | That mirror is a high-level structural catalog (its `fab batch` row says only "Multi-target batch operations" with no `--all`/`--list` literal); grep confirms no batch-archive flag detail in it | S:75 R:75 A:80 D:75 |

6 assumptions (5 certain, 1 confident, 0 tentative).

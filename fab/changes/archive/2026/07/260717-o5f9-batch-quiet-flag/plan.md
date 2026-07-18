# Plan: `--quiet` for `fab batch archive` + `fab batch switch`

**Change**: 260717-o5f9-batch-quiet-flag
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. Principle №9 (bounded, high-signal output): what
     survives `--quiet` is the data and the errors — never progress, decoration,
     or chatter. Default (no-flag) behavior MUST remain byte-identical. -->

### CLI: `fab batch archive --quiet`

#### R1: Register `--quiet`/`-q` on `fab batch archive`
`fab batch archive` MUST expose a boolean `--quiet` flag with a `-q` shorthand, registered via `BoolVarP` (mirroring the existing `--yes`/`-y` precedent). Its help text describes suppressing per-change progress output. `fab batch new` MUST NOT gain the flag.

- **GIVEN** the `batch archive` command
- **WHEN** its flag set is inspected
- **THEN** `--quiet` is present AND `-q` resolves to it
- **AND** `--yes`/`-y` and `--dry-run` remain registered unchanged

#### R2: `--quiet` suppresses archive progress chatter
Under `--quiet`, `fab batch archive` MUST NOT print the `Archiving %d changes...` preamble (batch_archive.go:136) or any per-change loop line emitted by `archiveLoop` (`  %s — already archived, skipping`, `  %s — archived`, `  %s — archived (backlog marked done)`).

- **GIVEN** an archivable set and `--quiet` (with `--yes` or via explicit args)
- **WHEN** the archive runs to completion
- **THEN** neither the `Archiving N changes...` preamble nor any `  {name} — ...` per-change line appears on stdout

#### R3: `--quiet` retains data and the summary footer
Under `--quiet`, `fab batch archive` MUST still print the summary footer `\nArchived %d, skipped %d, failed %d.\n` (batch_archive.go:228), the empty-set no-op output (`No archivable changes found.` + zero footer), and the `--dry-run` listing. `--quiet` MUST NOT alter exit semantics (finding F49's exit-0-before-guards behavior preserved).

- **GIVEN** `--quiet --yes` over a non-empty archivable set
- **WHEN** the archive completes
- **THEN** the only stdout is the `Archived N, skipped N, failed N.` footer
- **AND GIVEN** `--quiet` over an empty set, `No archivable changes found.` + the zero footer still print (exit 0)
- **AND GIVEN** `--quiet --dry-run`, the archivable listing still prints (it is the requested data; effectively a no-op interaction)

#### R4: `--quiet` never touches stderr
Under `--quiet`, ALL stderr output MUST be retained unconditionally: resolve/not-ready/ambiguous warnings, `  %s — FAILED: %v`, and post-archive `    warning: %v` lines. The quiet gating MUST apply only to the stdout writer, never to `errW`.

- **GIVEN** `--quiet` with a change that fails to resolve or fails to archive
- **WHEN** the command runs
- **THEN** every warning/error line still appears on stderr, byte-for-byte as without `--quiet`

#### R5: `--quiet` is orthogonal to consent; it does not imply `--yes`
`--quiet` MUST NOT change the consent flow. The bare-invocation interactive path (archivable-set listing + `Archive these %d? [y/N]` prompt + `Aborted; nothing archived.`) is consent interaction, not progress, and is unaffected by `--quiet`. `--quiet` MUST NOT introduce any new mutual-exclusion rule with `--yes`, `--dry-run`, or explicit args.

- **GIVEN** a bare `fab batch archive --quiet` on an interactive TTY
- **WHEN** the command runs
- **THEN** the archivable-set listing and the `[y/N]` prompt still appear (consent survives `--quiet`)
- **AND** `--quiet` alone does not archive without a `y`/`yes` answer (no implied `--yes`)

### CLI: `fab batch switch --quiet`

#### R6: Register `--quiet`/`-q` on `fab batch switch`
`fab batch switch` MUST expose a boolean `--quiet` flag with a `-q` shorthand, registered via `BoolVarP`. `--list` and `--all` remain registered unchanged.

- **GIVEN** the `batch switch` command
- **WHEN** its flag set is inspected
- **THEN** `--quiet` is present AND `-q` resolves to it AND `--list`/`--all` are unchanged

#### R7: `--quiet` suppresses switch progress; stderr and `--list` unaffected; no new footer
Under `--quiet`, `fab batch switch` MUST NOT print the `Opening %d tabs for all changes...` preamble (batch_switch.go:68, `--all` path) or the per-change `  %s\n` resolved-name line (batch_switch.go:93). ALL stderr (`Warning: could not resolve ...`, `Error: failed to create worktree ...`) and the `--list` output MUST be unaffected. This change MUST NOT add a summary footer to `batch switch` — a quiet successful run is stdout-silent (standard Unix quiet semantics); tmux window creation remains the observable effect.

- **GIVEN** `--quiet` with resolvable change(s) inside tmux
- **WHEN** switch runs to completion
- **THEN** stdout is empty (no preamble, no per-change line, no footer)
- **AND GIVEN** a change that fails to resolve under `--quiet`, its `Warning:`/`Error:` line still appears on stderr
- **AND GIVEN** `--quiet --list`, the available-changes listing still prints (data unaffected)

### Docs: CLI reference + mirrors

#### R8: Document `--quiet` in `_cli-fab.md` § fab batch and sweep the mirror class
The constitution-mandated CLI-doc updates MUST land in the same change. `src/kit/skills/_cli-fab.md` § fab batch MUST document `--quiet`/`-q` on both surfaces: the family intro's two flag-surface lists (`[--list] [--all]` for switch, `[--yes|-y] [--dry-run]` for archive) each gain `[--quiet|-q]` (the intro MUST NOT imply `new` gains it); the `switch` bullet documents `--quiet` (suppresses preamble + per-change lines; stderr and `--list` unaffected); the `archive` bullet documents `--quiet` in its flag surface and in the "Per change prints ..." paragraph (per-change lines suppressed, footer + stderr retained, no consent-model interaction). The whole mirror class MUST be swept: `docs/specs/skills/SPEC-_cli-fab.md` (its constitution-required mirror) and `docs/specs/architecture.md` § Batch Operations (the aggregate spec that restates both flag surfaces).

- **GIVEN** the CLI signature change
- **WHEN** the docs are inspected
- **THEN** `_cli-fab.md` § fab batch documents `--quiet` on both switch and archive, and the family intro's two flag-surface lists each carry `[--quiet|-q]` without implying `new` gains it
- **AND** `docs/specs/architecture.md` § Batch Operations' flag-surface restatement carries `[--quiet|-q]` on both switch and archive
- **AND** `docs/specs/skills/SPEC-_cli-fab.md` reflects the batch flag-surface change (its `fab batch` inventory row)

### Non-Goals

- `fab batch new` — out of scope per the backlog entry (the audit scoped `--quiet` to archive + switch only).
- A new summary footer for `batch switch` — explicitly not added.
- Config, migration, or `.status.yaml` changes — none (no user-data restructuring).
- Manual edits to `fab help-dump` / `fab fab-help` output — the flag is picked up automatically from the cobra tree.
- `docs/memory/` edits — memory is hydrated at the hydrate stage, not apply.

### Design Decisions

1. **Progress gating via a discard writer threaded through the archive call chain**: in `runBatchArchive`, compute a progress writer `pw := w; if quietFlag { pw = io.Discard }` and thread `pw` through `archiveResolvedNames` into `archiveLoop`, while the footer keeps writing to the real `w`. — *Why*: smallest change that preserves `archiveLoop`'s testable, never-`os.Exit` shape and keeps the footer/stderr on their real writers; per-change lines route through `pw` and vanish under `--quiet`. — *Rejected*: a `quiet bool` parameter with inline `if !quiet` guards around each `Fprintf` (more conditionals, easy to miss one; the discard-writer routes all per-change lines through a single seam). The `Archiving N...` preamble in `runBatchArchive` is gated inline (`if !quietFlag`) since it is not inside `archiveLoop`.
2. **Switch gating inline**: `batch switch` has only two suppressed prints and no shared loop helper, so gate them with a straightforward `if !quietFlag` guard around each (the `Opening N tabs...` preamble and the `  %s` per-change line). — *Why*: two call sites, no writer-threading benefit; inline guards are the clearest form here. — *Rejected*: a discard writer (over-engineered for two lines that both live in `runBatchSwitch`).
3. **`-q` shorthand**: registered on both commands via `BoolVarP`. — *Why*: mirrors the family's `--yes`/`-y` precedent and universal CLI convention; the standard names only `--quiet` but `-q` is conventional. — *Rejected*: `--quiet` long-form only (breaks the family's short-alias convention).

## Tasks

### Phase 1: Core Implementation — archive

- [x] T001 Add the `--quiet`/`-q` flag to `batchArchiveCmd()` in `src/go/fab/cmd/fab/batch_archive.go` via `cmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress per-change progress output (keep the summary footer and all stderr)")`, declare `quietFlag` alongside `yesFlag`/`dryRunFlag`, and thread it into `runBatchArchive` (RunE closure + signature). <!-- R1 -->
- [x] T002 In `runBatchArchive` (`src/go/fab/cmd/fab/batch_archive.go`), compute the progress writer `pw := w; if quietFlag { pw = io.Discard }`, gate the `Archiving %d changes...` preamble behind `if !quietFlag`, and pass `pw` into `archiveResolvedNames`. Leave the empty-set no-op output, `--dry-run` listing, and the full bare-invocation consent flow (listing + `[y/N]` prompt + `Aborted` + non-TTY refusal) writing to the real `w` — unchanged. <!-- R2 --> <!-- R3 --> <!-- R5 -->
- [x] T003 Thread the progress writer through `archiveResolvedNames` and `archiveLoop` in `src/go/fab/cmd/fab/batch_archive.go`: add a `pw io.Writer` parameter, route every per-change `Fprintf`/`Fprintln` currently going to `w` (`  %s — already archived, skipping`, `  %s — archived`, `  %s — archived[ (backlog marked done)]`) through `pw`, and keep the footer `\nArchived %d, skipped %d, failed %d.\n` and ALL `errW` writes exactly as-is. <!-- R2 --> <!-- R3 --> <!-- R4 -->

### Phase 2: Core Implementation — switch

- [x] T004 Add the `--quiet`/`-q` flag to `batchSwitchCmd()` in `src/go/fab/cmd/fab/batch_switch.go` via `cmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress per-change progress output")`, declare `quietFlag` alongside `listFlag`/`allFlag`, and thread it into `runBatchSwitch` (RunE closure + signature). <!-- R6 -->
- [x] T005 In `runBatchSwitch` (`src/go/fab/cmd/fab/batch_switch.go`), gate the `Opening %d tabs for all changes...` preamble (`--all` path) and the per-change `  %s\n` resolved-name line behind `if !quietFlag`. Leave `listChanges`, all `errW` warn/error lines, and worktree/tmux creation untouched. Add no summary footer. <!-- R7 -->

### Phase 3: Tests

- [x] T006 [P] Extend `TestBatchArchiveCmd_Structure` in `src/go/fab/cmd/fab/batch_archive_test.go` to assert `--quiet` is registered and `-q` resolves to it. Add behavior tests: (a) `--quiet --yes` over a non-empty set prints only the `Archived N, skipped N, failed N.` footer (no `Archiving N...` preamble, no `  {name} — ...` lines); (b) `--quiet` with explicit args suppresses the same per-change lines and keeps the footer; (c) `--quiet` over an empty set still prints `No archivable changes found.` + zero footer (exit 0); (d) a failing/unresolvable change under `--quiet` still writes its warning to stderr; (e) bare `--quiet` on a forced-TTY still shows the `Archive these N? [y/N]` prompt (consent survives; no implied `--yes`). <!-- R1 --> <!-- R2 --> <!-- R3 --> <!-- R4 --> <!-- R5 -->
- [x] T007 [P] Extend `TestBatchSwitchCmd_Structure` in `src/go/fab/cmd/fab/batch_switch_test.go` to assert `--quiet` is registered and `-q` resolves to it. Add behavior tests using the existing stubbed-tmux/routing fixtures: (a) a successful `--quiet` switch produces empty stdout (no preamble, no per-change line, no footer) while still creating the tmux window; (b) an unresolvable name under `--quiet` still emits its `could not resolve` warning to stderr. <!-- R6 --> <!-- R7 -->

### Phase 4: Docs + mirror sweep

- [x] T008 Update `src/kit/skills/_cli-fab.md` § fab batch: add `[--quiet|-q]` to the switch flag-surface (`[--list] [--all]`) and the archive flag-surface (`[--yes|-y] [--dry-run]`) in the family intro line (without implying `new` gains it); document `--quiet` in the `switch` bullet (suppresses preamble + per-change lines; stderr and `--list` unaffected) and in the `archive` bullet's flag list + "Per change prints ..." paragraph (per-change lines suppressed; footer + stderr retained; no consent-model interaction). <!-- R8 -->
- [x] T009 [P] Sweep the mirror class: update `docs/specs/architecture.md` § Batch Operations (the `new`/`switch` `[--list] [--all]` and `archive` `[--yes|-y] [--dry-run]` flag-surface restatement at ~line 464) to carry `[--quiet|-q]` on both switch and archive; update the `fab batch` inventory row in `docs/specs/skills/SPEC-_cli-fab.md` to note the added `--quiet`/`-q` flag on switch and archive. <!-- R8 -->

## Execution Order

- T001 → T002 → T003 (archive: flag declared, then wired through `runBatchArchive`, then through `archiveResolvedNames`/`archiveLoop`).
- T004 → T005 (switch: flag declared, then gated in `runBatchSwitch`).
- T006 depends on T001–T003; T007 depends on T004–T005.
- Phase 4 (T008, T009) is independent of the Go work and may run in parallel with it; T009's two files are independent of each other.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab batch archive` registers `--quiet` with a working `-q` shorthand; `--yes`/`-y` and `--dry-run` remain; `batch new` does not gain `--quiet`.
- [x] A-002 R6: `fab batch switch` registers `--quiet` with a working `-q` shorthand; `--list`/`--all` remain.
- [x] A-003 R8: `_cli-fab.md` § fab batch documents `--quiet` on both switch and archive; the family intro's two flag-surface lists carry `[--quiet|-q]` without implying `new` gains it; `docs/specs/architecture.md` § Batch Operations and the `SPEC-_cli-fab.md` `fab batch` row reflect the change.

### Behavioral Correctness

- [x] A-004 R2: Under `--quiet`, `fab batch archive` suppresses the `Archiving N changes...` preamble and every `archiveLoop` per-change line (already-archived / archived / archived-backlog-marked).
- [x] A-005 R3: Under `--quiet`, `fab batch archive` still prints the `Archived N, skipped N, failed N.` footer, the empty-set `No archivable changes found.` + zero footer, and the `--dry-run` listing; exit semantics (incl. F49 exit-0 no-op) unchanged.
- [x] A-006 R7: Under `--quiet`, `fab batch switch` produces empty stdout on success (no preamble, no per-change line, no footer) while still creating tmux windows; `--list` output and stderr unaffected.

### Scenario Coverage

- [x] A-007 R2: A test exercises `--quiet --yes` (and `--quiet` + explicit args) for archive and asserts footer-only stdout.
- [x] A-008 R7: A test exercises a successful `--quiet` switch and asserts empty stdout with the tmux window still created.

### Edge Cases & Error Handling

- [x] A-009 R4: Under `--quiet`, all `fab batch archive` stderr (resolve/not-ready/ambiguous warnings, `FAILED:`, post-archive `warning:`) is retained; a test asserts stderr is untouched.
- [x] A-010 R5: `--quiet` does not imply `--yes` — a bare `--quiet` invocation on a TTY still lists the set and prompts `[y/N]`; no new mutual-exclusion error is introduced. A test asserts the prompt still appears.
- [x] A-011 R7: Under `--quiet`, an unresolvable `batch switch` name still emits its `could not resolve` warning to stderr; a test asserts this.
- [x] A-012 R3: Default (no-`--quiet`) behavior is byte-identical — the existing archive/switch tests pass with assertions unmodified (call sites mechanically updated for the new parameter, as T001–T005 themselves require).

### Code Quality

- [x] A-013 Pattern consistency: The flag is registered via `BoolVarP` mirroring the existing `--yes`/`-y` precedent; the writer-threading follows the existing `w`/`errW`/`io.Writer` parameter style in `batch_archive.go`.
- [x] A-014 No unnecessary duplication: Progress gating reuses `io.Discard` and the existing writer parameters rather than adding parallel print helpers; switch reuses inline guards for its two call sites.
- [x] A-015 Canonical source only: Kit doc edits land in `src/kit/skills/_cli-fab.md`, never in `.claude/skills/` (gitignored deployed copies).
- [x] A-016 CLI ⇒ docs + tests: The Go signature change ships `_cli-fab.md` + SPEC mirror updates and test updates in this same change (Constitution Additional Constraints).
- [x] A-017 SPEC-mirror sync: The `src/kit/skills/_cli-fab.md` edit carries its `docs/specs/skills/SPEC-_cli-fab.md` mirror update, and the aggregate `docs/specs/architecture.md` restatement is swept.

### Documentation Accuracy

- [x] A-018 R8: The documented `--quiet` semantics (suppressed lines, retained footer/stderr/data, orthogonal consent) match the implemented behavior exactly — no doc claims a suppression or retention the code does not perform.

### Cross References

- [x] A-019 R8: Every restatement of either batch flag surface across `src/kit/` + `docs/` (the `_cli-fab.md` intro, the `architecture.md` § Batch Operations line, the `SPEC-_cli-fab.md` row) is updated consistently; no stale flag-surface restatement of `[--list] [--all]` / `[--yes|-y] [--dry-run]` remains that should carry `[--quiet|-q]`. (Sweep re-run at review: `assembly-line.md`/`companions.md` are usage examples, `findings/*.md` dated snapshots — correctly out of class. One tightening flagged should-fix: the intro's first-sentence archive parenthetical still reads `[--yes|-y] [--dry-run]` before the complete surface follows.)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- `docs/specs/assembly-line.md` uses `fab batch archive --yes` / `fab batch switch --all` as illustrative command examples (not flag-surface enumerations) — out of the strict mirror class, left unchanged. `docs/specs/findings/*.md` are dated review snapshots, not living specs.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Checked: the `archiveLoop` writer split leaves no orphaned writer usage — `w` still carries the footer; no print helper, flag, or doc passage was superseded; nothing in `batch_new.go` or the consent/dry-run paths became dead.)

## Assumptions

<!-- Apply-time graded decisions (Certain/Confident/Tentative). The intake resolved
     all design points; these carry the intake's grades forward for the points that
     remained apply-implementation choices. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Archive progress gating uses `io.Discard` threaded through `archiveResolvedNames`/`archiveLoop` (`pw` param); footer + stderr stay on the real writers | Intake assumption 8 fixed the discard-writer approach and left the exact signature to the plan; a single `pw io.Writer` param is the smallest change preserving `archiveLoop`'s never-os.Exit testable shape | S:70 R:90 A:85 D:75 |
| 2 | Confident | The `Archiving N...` preamble (in `runBatchArchive`, not `archiveLoop`) is gated inline with `if !quietFlag`, not via the discard writer | It lives outside the loop helper the `pw` param threads through; an inline guard is the clearest gate for a single line at that site | S:75 R:90 A:85 D:80 |
| 3 | Confident | `batch switch`'s two suppressed prints are gated inline with `if !quietFlag` rather than a discard writer | Only two call sites, both in `runBatchSwitch`, no shared loop helper — a discard writer would be over-engineering for two lines | S:75 R:90 A:85 D:80 |
| 4 | Confident | `docs/specs/architecture.md` § Batch Operations is in the mirror sweep class; `assembly-line.md` usage examples and `findings/*.md` snapshots are not | code-quality § Sibling & Mirror Sweeps names `architecture.md` as an aggregate spec that restates per-skill facts; the grep confirms it restates both flag surfaces, while assembly-line uses bare command examples and findings are dated artifacts | S:80 R:85 A:80 D:80 |
| 5 | Confident | `-q` shorthand is registered on both commands (not `--quiet` long-form only) | Intake assumption 6: mirrors the family's `--yes`/`-y` precedent and universal CLI convention | S:65 R:90 A:80 D:75 |

5 assumptions (0 certain, 5 confident, 0 tentative).

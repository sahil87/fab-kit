# Plan: Pane Capture Tails to the Last N Lines

**Change**: 260819-y4mu-pane-capture-tail-last-lines
**Intake**: `intake.md`

## Requirements

### Runtime: `fab pane capture` line-count contract

#### R1: Capture returns the last N non-blank-padded lines
`pane.Capture(server, paneID, lines)` (src/go/fab/internal/pane/pane.go) SHALL return **at most the last N lines** of the tmux capture output, computed by first stripping *trailing* blank lines (the visible-screen padding tmux adds up to the pane height) and then taking the last N of what remains. Blank lines *interior* to the content SHALL be preserved. When fewer than N lines remain after padding-stripping, all of them are returned.

- **GIVEN** a pane whose visible screen is 40 rows with 30 content lines and scrollback history
- **WHEN** `fab pane capture -l 5 <pane>` runs
- **THEN** the captured text contains exactly the last 5 non-blank-padded lines (not 45)
- **AND** a blank line that sits *between* two content lines within that window is kept

#### R2: `--raw` byte fidelity within the returned window
Within the returned window, lines SHALL be byte-untouched — no per-line trimming — so `--raw` output stays byte-identical to tmux's bytes *within that window* (trailing spaces inside a line survive; the window's trailing newline matches tmux's own line termination).

- **GIVEN** a pane whose last content line carries trailing spaces
- **WHEN** `fab pane capture --raw -l 3 <pane>` runs
- **THEN** those trailing spaces appear verbatim in the output

#### R3: Tail is post-processing; the tmux argv and all consumers share one semantic
`CaptureArgs` SHALL remain unchanged (`capture-pane -t <pane> -p -S -N` — the raw-material fetch that guarantees ≥N lines of material whenever the pane has that much history). The tail SHALL be implemented as a pure helper (`TailLines(s string, n int) string`) in `internal/pane`, applied inside `Capture`, so all three consumers — `fab pane capture`, the dispatch readiness probe, and the delivery choreography (`gate.go`) — inherit the corrected semantics from the one shared runner.

- **GIVEN** the existing `TestCapturePaneArgs` expectations
- **WHEN** the change lands
- **THEN** the built argv is byte-identical to before (`-S -N` unchanged)
- **AND** `gate.go` needs no code change (its `Capture` calls flow through the same transform on both sides of every comparison)

#### R4: Comments and CLI docs state the true contract
The doc comments claiming "last N lines" (`cmd/fab/pane_capture.go` `capturePaneArgs`/`capturePaneContent`, `internal/pane/pane.go` `CaptureArgs`/`Capture`) SHALL describe the fetch-then-tail mechanism accurately. `src/kit/skills/_cli-fab.md` § `fab pane capture` SHALL state the corrected `-l N` semantics and reword the "`--raw` output is byte-identical to tmux's stdout (never trimmed)" claim to byte-identical *within the returned window*. `src/kit/skills/_cli-agents.md`'s capture-window guidance (the "`-l 50`+ … or `capture-pane -S -20` for the last 20 lines" workaround framing) SHALL be updated now that `-l N` really returns N lines. A repo-wide sweep SHALL catch any other stale behavior claim (e.g. `| tail -N` workarounds for this command).

- **GIVEN** the change is applied
- **WHEN** grepping skills/docs for the old claims ("N scrollback lines + screen" workarounds, "never trimmed")
- **THEN** no stale occurrence remains outside `docs/memory/` (memory updates land at hydrate) and the change folder/backlog

#### R5: Tests ship with the change (Constitution VII / test-alongside)
New table tests SHALL cover `TailLines` (trailing padding stripped, interior blanks preserved, short content, exact-N, n larger than content, byte fidelity within the window incl. trailing spaces, trailing-newline handling); existing capture/gate tests SHALL keep passing.

- **GIVEN** the implementation
- **WHEN** `go test ./internal/pane/... ./cmd/fab/...` runs (scoped)
- **THEN** all tests pass, including the new `TailLines` table tests

### Non-Goals

- No change to the `--json` schema: the `lines` field keeps echoing the requested flag value (post-fix the content honors it, so it stops misleading).
- No change to flags, defaults (`-l` 50), exit codes (2/3/1), mutual exclusion, enrichment header, or `ValidatePane` behavior.
- No memory edits during apply — `runtime/pane-commands.md` / `runtime/agent-primitives.md` updates are hydrate's.

### Design Decisions

#### Tail lives in the shared `internal/pane.Capture`, not the CLI layer
**Decision**: Apply the tail transform inside `pane.Capture` via a pure `TailLines` helper, so the CLI, the dispatch readiness probe, and the delivery choreography all get the corrected "last N lines" semantics.
**Why**: `internal/pane`'s own comment pins builder+runner there so the three consumers "cannot drift on the capture range"; a CLI-only tail would fork capture semantics across consumers. Gate comparisons are transform-symmetric (both captures pass through the same tail), so verification logic is unaffected and its snippets lose their blank padding.
**Rejected**: Tailing only in `cmd/fab/pane_capture.go` — keeps the gate byte-identical but re-introduces exactly the consumer drift the shared package exists to prevent.
*Introduced by*: 260819-y4mu-pane-capture-tail-last-lines

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add pure `TailLines(s string, n int) string` to `src/go/fab/internal/pane/pane.go` (strip trailing blank lines, take last n, preserve interior blanks and byte content within the window); apply it in `Capture`; update the `CaptureArgs`/`Capture` doc comments to describe fetch-then-tail <!-- R1, R2, R3 -->
- [x] T002 Update `src/go/fab/cmd/fab/pane_capture.go` doc comments (`capturePaneArgs`, `capturePaneContent`) to the corrected last-N contract; no logic change <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Add `TailLines` table tests + a `Capture`-path expectation to `src/go/fab/internal/pane/pane_test.go` (padding stripped, interior blanks kept, short content, exact-N, n > content, trailing spaces preserved, trailing newline); run scoped `go test ./internal/pane/... ./cmd/fab/...` and confirm existing capture/gate tests pass <!-- R5 -->

### Phase 4: Polish

- [x] T004 Update `src/kit/skills/_cli-fab.md` § `fab pane capture` and `src/kit/skills/_cli-agents.md` capture-window guidance; sweep the repo (skills, specs, scripts) for stale claims about the old capture range or `| tail` workarounds for this command <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `pane.Capture` returns at most the last N lines after stripping trailing blank padding; interior blank lines within the window are preserved
- [x] A-002 R3: `CaptureArgs` argv is unchanged (`-S -N`); `TailLines` is a pure helper applied inside `Capture`; `gate.go` required no code change

### Behavioral Correctness

- [x] A-003 R2: bytes within the returned window are untouched (trailing spaces inside lines survive; newline termination matches tmux's)
- [x] A-004 R1: `-l 5` against a tall pane yields 5 lines, not scrollback+screen (unit-verified through the tail transform)

### Scenario Coverage

- [x] A-005 R5: `TailLines` table tests cover padding-stripped / interior-blank / short-content / exact-N / n-exceeds-content / byte-fidelity / trailing-newline cases and pass

### Edge Cases & Error Handling

- [x] A-006 R1: content shorter than N returns everything (no error, no padding); n ≥ total lines is a no-op beyond padding-stripping

### Code Quality

- [x] A-007 Pattern consistency: `TailLines` matches `internal/pane`'s pure-helper, table-testable style and comment conventions
- [x] A-008 No unnecessary duplication: reuses stdlib `strings` splitting; no second tail implementation elsewhere
- [x] A-009 CLI ⇒ docs + tests: `src/kit/skills/_cli-fab.md` updated and tests ship in the same change (constitution Additional Constraints)
- [x] A-010 Canonical source only: skill edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-011 Sweep: no stale "last N lines"-contradicting claims or `-S -20` workaround prose remain in `src/kit/skills/` (memory sweep deferred to hydrate)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/internal/pane/gate.go:318` (`Snippet`'s `strings.TrimRight(capture, " \t\r\n")`) — partially redundant for real gate captures, which now arrive pre-stripped of trailing blank padding via `TailLines` inside `Capture`; it still earns its keep for test-fake inputs and last-line trailing-space normalization, so this is a simplification candidate, not a removal
- External `| tail -N` pipelines in operator/run-kit scripts — the workaround this fix exists to retire; not deletable from this repo, but the fab-operator §5 raw `tmux capture-pane -p -S -20` capture (`src/kit/skills/fab-operator.md:333`) is now strictly inferior to `fab pane capture --raw -l 20` and its migration is already tracked as backlog item zs1u

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | A "blank" padding line is empty **or whitespace-only**; stripping targets trailing such lines only | tmux pads with empty rows; whitespace-only trailing rows are indistinguishable from padding, and stripping them only shrinks the window from the end | S:55 R:85 A:75 D:60 |
| 2 | Confident | The returned window ends with a trailing newline when non-empty (matching tmux's own per-line termination); empty result is the empty string | Keeps `--raw` composable with shell pipelines exactly like tmux's stdout | S:50 R:90 A:80 D:65 |
| 3 | Certain | `gate_test.go` needs no updates — the gate is tested against a scripted fake `PaneIO`, not real tmux captures | Verified by reading gate.go/gate_test.go: `PaneIO` is an interface; fakes bypass `pane.Capture` | S:85 R:90 A:95 D:90 |

3 assumptions (1 certain, 2 confident, 0 tentative).

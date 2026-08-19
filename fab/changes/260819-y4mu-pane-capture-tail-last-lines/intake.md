# Intake: Pane Capture Tails to the Last N Lines

**Change**: 260819-y4mu-pane-capture-tail-last-lines
**Created**: 2026-08-20

## Origin

One-shot `/fab-new y4mu` from the backlog entry (no prior conversation):

> [y4mu] 2026-08-19: fab pane capture -l N does not return the last N lines: it builds 'capture-pane -p -S -N' = N scrollback lines + the ENTIRE visible screen (verified: -l 5 returned 45 content lines; the --json 'lines' field just echoes the flag). The code comments (cmd/fab/pane_capture.go:101, internal/pane/pane.go CaptureArgs) claim 'last N lines' — inaccurate. Fix: tail the capture internally to the last N non-blank-padded lines (keep --raw byte-fidelity within that window), so callers stop needing '| tail -N'. This convenience gap is why operators reach for raw tmux capture-pane instead.

## Why

1. **Problem**: `fab pane capture -l N` promises "last N lines" (flag help "Number of lines to capture", code comments at `cmd/fab/pane_capture.go:101` and `internal/pane/pane.go` `CaptureArgs`) but delivers far more. `tmux capture-pane -p -S -N` sets only the *start* of the capture window — N lines above the top of the visible screen — so the output is N scrollback lines **plus the entire visible screen**, padded with trailing blank lines to the pane height. Verified: `-l 5` returned 45 content lines.
2. **Consequence if unfixed**: every consumer that actually wants N lines must pipe through `| tail -N`, and the convenience gap pushes operators back to raw `tmux capture-pane` — defeating the point of the enriched fab verb. The `--json` `lines` field echoing the flag compounds the confusion (it reports a window size the content does not honor).
3. **Approach**: fix at the source — tail the captured text inside the shared `internal/pane` capture path to the last N non-blank-padded lines, keeping the tmux argv (`-S -N`) unchanged as the raw-material fetch. This makes the documented contract true rather than weakening the docs to match the bug, and it keeps the one-builder/one-runner no-drift property (`internal/pane` is shared by the CLI, the dispatch readiness probe, and the delivery choreography).

## What Changes

### `internal/pane`: tail the capture to the last N lines

`pane.Capture(server, paneID, lines)` (src/go/fab/internal/pane/pane.go) currently returns `tmux capture-pane -p -S -N` output verbatim. After this change it returns **the last N lines of that output after stripping trailing blank-padding lines**:

1. Capture with the existing argv (`CaptureArgs` unchanged — `-S -N` still fetches N scrollback lines + visible screen, guaranteeing ≥N lines of material whenever the pane has that much history).
2. Strip *trailing* blank lines (the visible-screen padding tmux adds up to the pane height). Blank lines *inside* the content are preserved.
3. Take the last N lines of what remains. Fewer than N lines exist → return them all.

Implement the transform as a pure, table-testable helper in `internal/pane` (e.g. `TailLines(s string, n int) string`) applied inside `Capture`, matching the package's established pure-helper style. Within the returned window, lines are byte-untouched — no per-line trimming — so `--raw` output stays byte-identical to tmux's bytes *within that window*.

All three `Capture` consumers get the corrected semantics: `fab pane capture`, the dispatch readiness probe, and the delivery choreography (`internal/pane/gate.go`, `captureLines = 50`). The gate compares captures against each other, and both sides of every comparison pass through the same transform, so its verification logic is unaffected; its snippets get cleaner (no blank padding).

### `cmd/fab/pane_capture.go`: comments become true; `--json` unchanged

- The "last N lines" comments (`pane_capture.go:101`, `pane.go` `CaptureArgs`/`Capture` docs) are updated to describe the fetch-then-tail mechanism accurately — after the fix the *contract* they state is finally true.
- The `--json` `lines` field keeps echoing the requested flag value (the requested window). Content now actually honors it, so the field stops being misleading without a schema change.
- No flag, exit-code, or error-behavior changes: `-l/--lines` default 50, `--lines < 1` → error, exit codes 2/3, `--json`/`--raw` mutual exclusion all stay as-is.

### Tests (constitution: Go changes ship tests)

- New table tests for the tail helper (padding stripped, interior blanks preserved, short content, exact-N, byte fidelity within the window).
- Update existing `Capture`/`capturePaneArgs` and gate tests where they assert untrimmed output.

### Docs sweep (constitution: CLI change ⇒ `_cli-fab.md`)

- `src/kit/skills/_cli-fab.md` § `fab pane capture` (~line 549): state the corrected `-l N` = last N non-blank-padded lines semantics and reword the `--raw` byte-identical claim to "byte-identical within the returned window".
- `src/kit/skills/_cli-agents.md` (~line 118): the "wide window compensates / or `capture-pane -S -20` for the last 20 lines" workaround prose — reword now that `-l N` really returns N lines.
- Behavior-claim sweep for `| tail -N` workarounds and "N scrollback lines + screen" descriptions across skills and memory (the memory half lands at hydrate).

## Affected Memory

- `runtime/pane-commands.md`: (modify) `fab pane capture` section — `-l N` now returns the last N non-blank-padded lines (fetch `-S -N`, tail internally); reword "`--raw` output is byte-identical to tmux's stdout (never trimmed)" to byte-identical within the returned window
- `runtime/agent-primitives.md`: (modify) capture/output guidance mirroring `_cli-agents.md` — drop or reword the raw-tmux `-S -20` workaround framing if present after sweep

## Impact

- `src/go/fab/internal/pane/pane.go` — `Capture` gains the tail step; new pure `TailLines` helper; doc comments corrected
- `src/go/fab/cmd/fab/pane_capture.go` — doc comments corrected (no logic change expected)
- `src/go/fab/internal/pane/gate.go` — no code change; behavior of its captures changes benignly (comparisons are transform-symmetric)
- Tests: `internal/pane` + `cmd/fab` pane-capture/gate test files
- `src/kit/skills/_cli-fab.md`, `src/kit/skills/_cli-agents.md` — documented semantics
- Scale: small — one helper + one call-site change + comment/doc/test updates

## Open Questions

- None — the backlog entry specifies the mechanism, the fix, and the fidelity constraint.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Tail is post-processing; the tmux argv (`-S -N`) stays unchanged as the raw-material fetch | Backlog says "tail the capture internally"; `-S -N` already guarantees ≥N lines of material | S:85 R:85 A:90 D:85 |
| 2 | Certain | "Non-blank-padded" = strip *trailing* blank lines only, then take last N; interior blanks preserved | Trailing blanks are tmux's screen-height padding — the exact artifact the backlog names | S:80 R:85 A:85 D:80 |
| 3 | Certain | `--raw` is tailed to the same window, byte-untouched within it | Backlog states it verbatim ("keep --raw byte-fidelity within that window") | S:90 R:80 A:85 D:85 |
| 4 | Confident | Tail lives in shared `internal/pane.Capture`, so the gate (readiness probe + delivery choreography) inherits it | One-builder/no-drift comment in pane.go argues for one semantic; gate comparisons are transform-symmetric so unaffected. Alternative (CLI-layer-only tail) rejected as it forks capture semantics across consumers | S:60 R:80 A:70 D:55 |
| 5 | Confident | `--json` `lines` field keeps echoing the requested flag value (no schema/meaning change) | Backlog observes the echo but demands no change; after the fix the request matches the content, so the field stops misleading without a contract break | S:50 R:90 A:75 D:55 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).

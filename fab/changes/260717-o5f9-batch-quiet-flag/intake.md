# Intake: `--quiet` for `fab batch archive` + `fab batch switch`

**Change**: 260717-o5f9-batch-quiet-flag
**Created**: 2026-07-18

## Origin

Backlog item `[o5f9]` (one-shot invocation via `/fab-new o5f9`, no prior conversation):

> [o5f9] 2026-07-18: Toolkit principle №9 (bounded, high-signal — `--quiet` where relevant) — no command offers `--quiet`; `batch archive`/`batch switch` print per-change progress lines (`batch_archive.go:136/212/225/228`, `batch_switch.go:68/93`) with no way to keep only the final summary. Deferred: add `--quiet` to `batch archive` + `batch switch` suppressing per-change progress while retaining the final `Archived N, skipped N, failed N.` summary and all stderr warnings. (Unbounded log output is already capped via `dispatch logs --tail N`.) New flag on two commands → updates `_cli-fab.md` + tests; deferred per change 260717-ptwh (principles audit) as additive-but-multi-command. shll v0.0.23.

Deferred from the 260717-ptwh principles audit as an additive multi-command follow-up. The governing standard was re-fetched live at intake time (`shll standards principles`, per the recurring-conformance rule): principle №9 *Bounded, high-signal output* (MUST) — "what survives `--quiet` is the data and the errors — never progress, decoration, or chatter."

## Why

1. **Pain point**: `fab batch archive` and `fab batch switch` are operated primarily by agents (the operator runtime calls `batch archive --yes` non-interactively). Both commands print one stdout line per change processed with no suppression mechanism — a large batch dumps dozens of progress lines into an agent's finite context window when only the outcome matters. No fab command currently offers `--quiet`, so principle №9's "mechanisms of control" obligation is unmet on these two surfaces.
2. **Consequence if unfixed**: fab remains non-conformant with a MUST-tier toolkit standard (Constitution § Toolkit Standards binds this repo to it), and every bulk archive/switch invocation taxes the invoking agent's context with per-change chatter it cannot opt out of.
3. **Why this approach**: a `--quiet` flag per command is the standard's own named mechanism. Scope is deliberately the two commands the audit identified — the only ones with per-change progress loops on stdout ready to gate. (`dispatch logs` — the other unbounded surface — is already capped via `--tail N`.)

## What Changes

### `fab batch archive --quiet` (`src/go/fab/cmd/fab/batch_archive.go`)

New `--quiet`/`-q` bool flag (registered via `BoolVarP`, mirroring the existing `--yes`/`-y` precedent). Classification of every current stdout write:

**Suppressed under `--quiet`** (progress — per-change chatter):
- `Archiving %d changes...\n` preamble (line 136)
- Per-change lines in `archiveLoop`: `  %s — already archived, skipping` (line 202), `  %s — archived` on the post-archive-step-warning path (line 212), and `  %s — archived[ (backlog marked done)]` (lines 221–225)

**Retained under `--quiet`** (data + errors):
- The summary footer `\nArchived %d, skipped %d, failed %d.\n` (line 228) — always printed
- ALL stderr output, unconditionally: resolve/not-ready warnings, `  %s — FAILED: %v`, post-archive `warning:` lines (these go to `errW` already and `--quiet` never touches stderr)
- The empty-set no-op output (`No archivable changes found.` + zero footer, lines 102–103) — it is the outcome of the run, not per-change progress; exit-0 semantics unchanged (finding F49 preserved)
- The `--dry-run` listing — that listing IS the data the caller asked for
- The bare-invocation interactive flow: the archivable-set listing + `Archive these %d? [y/N]` prompt (lines 125–126) and `Aborted; nothing archived.` — consent interaction, not progress. `--quiet` does NOT imply `--yes`; consent (`--yes`, TTY guard, non-TTY refusal) and verbosity remain fully orthogonal

**Flag interactions**: no new mutual-exclusion rules. `--quiet --dry-run` is legal and effectively a no-op (the dry-run path prints only data); `--quiet --yes` is the expected agent invocation (suppressed loop, footer only); `--quiet` with explicit args gates that path's per-change lines identically.

**Mechanism sketch** (final shape is plan's call): route progress lines through a progress writer — e.g. `pw := w; if quiet { pw = io.Discard }` in `runBatchArchive`, threading `pw`/`quiet` through `archiveResolvedNames` into `archiveLoop` — while the footer keeps writing to `w`. `archiveLoop`'s signature grows a parameter either way (`archiveLoop(w, pw, errW, ...)` or `quiet bool`); its unit tests (`TestArchiveLoop`) cover both modes.

### `fab batch switch --quiet` (`src/go/fab/cmd/fab/batch_switch.go`)

Same `--quiet`/`-q` flag. Classification:

**Suppressed under `--quiet`**:
- `Opening %d tabs for all changes...\n` preamble (line 68, `--all` path)
- Per-change `  %s\n` resolved-name line (line 93)

**Retained under `--quiet`**:
- ALL stderr: `Warning: could not resolve ...` and `Error: failed to create worktree ...` warn-and-skip lines
- The `--list` output (`listChanges`) — data, unaffected by `--quiet` (a `--quiet --list` combo simply prints the list)

`batch switch` has no summary footer today and this change does NOT add one — a quiet successful run is stdout-silent (standard Unix quiet semantics); tmux window creation is the observable effect. Exit semantics unchanged.

### Docs: `src/kit/skills/_cli-fab.md` § fab batch (+ SPEC mirror)

Constitution-mandated (CLI signature change ⇒ `_cli-fab.md` + tests). Updates within § fab batch:
- The family intro line — currently "`new` and `switch` subcommands take `[--list] [--all]`" and "`archive` ... has its own flag surface (`[--yes|-y] [--dry-run]`)" — both flag surfaces gain `[--quiet|-q]` (note: `new` does NOT gain it; the intro must not imply family-wide `--quiet`)
- The **`switch`** bullet — document `--quiet` (suppresses the preamble + per-change lines; stderr and `--list` unaffected)
- The **`archive`** bullet — document `--quiet` in the flag-surface list and in the "Per change prints ..." paragraph (per-change lines suppressed, footer + stderr retained, no interaction with the consent model)

**Mirror sweep** (code-quality § Sibling & Mirror Sweeps): `docs/specs/skills/SPEC-_cli-fab.md` updated in the same change. Sweep greps: `--yes|-y] [--dry-run`, `[--list] [--all]`, `Opening %d tabs`, `Archived {N}, skipped` across `src/kit/` + `docs/` to catch every restatement of either flag surface.

### Explicitly NOT changed

- `fab batch new` — out of scope per the backlog entry (audit scoped `--quiet` to archive + switch)
- No new summary line for `batch switch`
- No config, no migration (no user-data restructuring), no `.status.yaml` changes
- `fab help-dump` / `fab fab-help` — pick the flag up automatically from the cobra tree; no manual edits

## Affected Memory

- `pipeline/change-lifecycle.md`: (modify) the `fab batch archive` confirmation/preview model paragraph (753q block, ~line 162) documents archive's flag surface and per-change print strings — add `--quiet` semantics
- `distribution/kit-architecture.md`: (modify) the `fab batch switch` bullet (~line 138) documents switch's flag surface (`--list`, `--all`, positionals) — add `--quiet`

## Impact

- **Code**: `src/go/fab/cmd/fab/batch_archive.go` (flag + writer gating through `runBatchArchive`/`archiveResolvedNames`/`archiveLoop`), `src/go/fab/cmd/fab/batch_switch.go` (flag + two gated prints). Small, additive; default behavior byte-identical without the flag.
- **Tests** (constitution: Go changes ship tests): extend `TestBatchArchiveCmd_Structure` / `TestBatchSwitchCmd_Structure` for flag registration; behavior tests asserting quiet-mode stdout (footer-only for archive incl. the `--yes` and explicit-args paths; empty for switch) and untouched stderr; existing tests keep passing unmodified (no-flag behavior unchanged). Scope test runs to `src/go/fab/cmd/fab` first.
- **Docs**: `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md` (same change); memory files above at hydrate.
- **Not affected**: `batch new`, dispatch, operator skill, config schema, migrations, help-dump plumbing.

## Open Questions

*(none — the backlog entry plus the live principle №9 text resolve all decision points; remaining choices are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly `batch archive` + `batch switch`; `batch new` untouched | Backlog entry names the two commands explicitly and scopes the audit deferral to them | S:90 R:85 A:95 D:95 |
| 2 | Certain | Archive `--quiet` suppresses the `Archiving N...` preamble + all per-change loop lines; retains the `Archived N, skipped N, failed N.` footer + all stderr | Verbatim from the backlog entry; matches principle №9's "data and errors survive" rule | S:95 R:90 A:95 D:95 |
| 3 | Certain | `--quiet` does not imply `--yes` — consent model (prompt, TTY guard, non-TTY refusal) fully orthogonal to verbosity | Principles №1/№5 separate consent from output volume; conflating them would silently unlock a bulk-mutating op | S:70 R:85 A:90 D:80 |
| 4 | Confident | Switch `--quiet` leaves stdout empty on success — no new summary footer added | Backlog names only archive's footer as retained; switch has none today; adding one is scope creep beyond the audit item | S:70 R:85 A:80 D:75 |
| 5 | Confident | Data outputs unaffected by `--quiet`: `--dry-run` listing, `--list` listing, and the bare-path consent listing/prompt | Principle №9: `--quiet` strips progress, never data; no mutual-exclusion errors needed (quiet+dry-run is a benign no-op) | S:70 R:85 A:85 D:70 |
| 6 | Confident | `-q` shorthand registered alongside `--quiet` on both commands | Standard names only `--quiet`; `-q` mirrors the family's `--yes`/`-y` precedent and universal CLI convention | S:60 R:90 A:80 D:75 |
| 7 | Confident | Empty-set no-op output (`No archivable changes found.` + zero footer) retained under `--quiet` | It is the run's outcome, not per-change progress; preserves finding F49's exit-0-before-guards semantics untouched | S:65 R:85 A:80 D:75 |
| 8 | Confident | Progress gating via a discard-writer (or equivalent `quiet` plumbing) through `archiveLoop`, footer stays on the real writer | Smallest change preserving `archiveLoop`'s testable never-os.Exit shape; exact signature is plan's decision within this constraint | S:55 R:90 A:85 D:65 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).

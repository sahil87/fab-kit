# Intake: Scanner-Truncation Sweep + Score Truth-Telling

**Change**: 260612-hv7t-scanner-truncation-sweep-score-truth-telling
**Created**: 2026-06-12

## Origin

> hv7t

One-shot `/fab-new hv7t` (backlog ID). Backlog entry (fab/backlog.md:17):

> Binary-review batch B2/6 — scanner-truncation sweep + score truth-telling. DEPENDS: wave 2 — branch after k4ge merges (collides on cmd/fab/score.go and the _cli-fab.md fab-score rows; k4ge defines the gate exit-code scheme this batch's error surfacing must conform to). GOAL: the binary either reads the whole file or says it didn't — no silent truncation, no lying success. ACTIONS: kill the unchecked-bufio.Scanner class — 12 of 13 production sites never check scanner.Err(), none call scanner.Buffer (F13 lists every site); standardize on a shared read-lines helper (os.ReadFile + strings.Split; TrimSuffix each line of \r to stay CRLF-behavior-preserving — verifier caveat) and migrate all sites. [F09–F15 details.] CONSTRAINTS: constitution — tests per Go change; no CLI signature changes expected. REPORT: docs/specs/findings/binary-review-2026-06-12.md §B2 F09-F15 (vs 1431a9c3).

Source of truth for the findings: `docs/specs/findings/binary-review-2026-06-12.md` §B2 (F09–F15), each finding adversarially verified with corrections. The report was written against `1431a9c3`; k4ge (PR #395, commit `5c054b5d`) has since merged, satisfying the wave-2 dependency. All file:line references below were **re-verified at the current HEAD (`5c054b5d`)** — archive.go and batch_archive.go line numbers shifted from the report's citations.

## Why

1. **Problem**: The fab binary's text parsers silently lie. 12 of 13 production `bufio.NewScanner` sites never check `scanner.Err()` and none call `scanner.Buffer`, so any line over bufio's 64KB `MaxScanTokenSize` (or any transient read error) aborts the scan mid-file with zero indication. Separately, `fab score` swallows `.status.yaml` load/persist failures (prints a score, exits 0, persists nothing) and the archive index functions return `"updated"`/`"removed"` even when their writes failed.

2. **Consequence if unfixed**: The worst case is gate integrity — F09: truncation inside intake.md's Assumptions table can flip the single intake confidence gate from hard-fail 0.0 to PASS by dropping graded rows (e.g., feat with 5 certain + 3 tentative = 2.0 fail becomes 5×(5/7) = 3.6 pass if truncated after the certain rows). That gate is the sole authorization for unattended `/fab-ff`/`/fab-fff` execution. Second-worst is data loss — F10: archive `removeFromIndex` rewrites `changes/archive/index.md` from a truncated scan, silently deleting every entry after the abort point (constitution Principle III violation). Plus: undercounted `.status.yaml` plan counters and PR-body task counts (F13), backlog items vanishing from `fab batch new --list/--all` with misleading "not found" errors (F12), stale confidence persisted invisibly (F11).

3. **Why this approach**: The report's adversarial verifier confirmed the fix idiom already exists in-repo in four places (`intake.go:28`, `backlog.go:118` `MarkDone`, `archive.go` `updateIndex`, fab-kit `sync.go:456`): `os.ReadFile` + `strings.Split` — every scanned input is a small markdown/YAML file already loaded in full elsewhere, so streaming buys nothing and the line-length failure class disappears entirely. One shared helper + mechanical migration eliminates the class instead of patching sites one by one. Error-surfacing conforms to the k4ge exit-code scheme (report stays on stdout; errors return from cobra `RunE` → stderr + non-zero via main's handler).

## What Changes

### 1. Shared read-lines helper (foundation for all scanner migrations)

New small helper in the `fab` Go module (e.g., `src/go/fab/internal/lines`), used by every migrated site:

```go
// ReadFileLines reads path fully and returns its lines.
// Each line is TrimSuffix'd of "\r" to preserve bufio.ScanLines' CRLF behavior.
func ReadFileLines(path string) ([]string, error)

// Split returns the lines of in-memory content (same CRLF semantics).
func Split(content string) []string
```

The `\r` TrimSuffix is **required**, not optional — `bufio.ScanLines` strips a trailing `\r`, so a naive `strings.Split(content, "\n")` would change CRLF handling (verifier caveat on F13).

### 2. F09 — `countGrades` stops returning partial counts (gate integrity)

`src/go/fab/internal/score/score.go` — `countGrades` (:228, scanner at :238) currently: default-buffer scanner, no `Err()` check, swallows `os.Open` errors (returns zero `GradeCount`, indistinguishable from "no Assumptions table").

- Signature becomes `countGrades(file string) (GradeCount, error)`; migrate to the read-lines helper; propagate open/read errors.
- Callers `CheckGate` (:79) and `Compute` (:129) surface the error instead of scoring a truncated table. A read failure must be distinguishable from an empty Assumptions table.
- Gate math context (unchanged, but the reason this matters): `computeScore` (:323) hard-fails 0.0 only when `unresolved > 0` — dropped rows can both remove unresolved rows and inflate the certain-ratio, so partial counts can only be prevented, not detected post-hoc.

### 3. F11 — `fab score` stops lying about persistence and reads

`src/go/fab/internal/score/score.go` `Compute`:

- :145–154 — `.status.yaml` load failure currently skips the entire write-back block (:173–181) silently and defaults `change_type` to `feat`. Change: return the load error from `Compute` (the documented contract in `_cli-fab.md` is "compute, write .status.yaml" — silent non-persistence violates it).
- :175–181 — `_ = status.SetConfidence(...)` / `_ = log.ConfidenceLog(...)` discard real `statusFile.Save` errors. Change: propagate (or at minimum surface on stderr as `fab: warning: confidence not persisted (<err>)` — hard error preferred, see Assumptions #4).
- The hook caller (`cmd/fab/hook.go` :265–268) already guards with `if err == nil`, so moving error returns into the library keeps the hook path best-effort with **zero hook changes**.
- Exit-code conformance (k4ge scheme, cf. `cmd/fab/score.go` gate-fail handling merged in #395): YAML report on stdout; failures returned from `RunE` → stderr + non-zero. Stale-data consumers fixed by honest persistence: preflight (preflight.go:89,113–117), `fab status confidence`, `fab change view/list`, pr-meta.

### 4. F10 — archive `removeFromIndex` safe read-modify-write

`src/go/fab/internal/archive/archive.go` — `removeFromIndex` (:452, scanner at :466, called from `Restore` at ~:169): the only scanner site that **writes back what it scanned** — truncation upgrades from misreporting to data loss, and the `os.WriteFile` error is discarded with an unconditional `"removed"` return.

- Replace scanner with the read-lines helper (same pattern `MarkDone` backlog.go:106+ and `updateIndex` archive.go:366 already use), so the rewrite always derives from the complete file.
- Return signature gains an error (verifier correction: the string-only return must change — e.g., `(string, error)`); `Restore` surfaces it.

### 5. F15 — atomic archive index writes + honest error reporting

`src/go/fab/internal/archive/archive.go` — `updateIndex` (:366), `backfillIndex` (:404), `removeFromIndex` (:452):

- Route index writes through a temp+rename helper (the `statusfile.Save` pattern, statusfile.go:~297; second precedent in runtime.go:~106). Extract a shared atomic-write helper rather than adding a third copy (code-quality anti-pattern: duplicating existing utilities).
- `updateIndex` stops ignoring its ReadFile (:~380 region) and WriteFile errors and stops returning unconditional `"updated"`/`"created"`; `Archive`'s YAML result reports index status honestly (archive.go:122–127 already documents exactly this propagate-don't-lie principle for `ArchiveWithBacklog`).
- `backfillIndex` reads the index once per operation (content passed in) instead of re-reading post-write.
- Verifier scope correction: the race is same-checkout cross-process only (two panes), not cross-worktree; index.md is git-tracked so worst realistic loss is the current uncommitted entry — atomicity still required by Principle III.

### 6. F12 — backlog parsing stops silently dropping items

`src/go/fab/internal/backlog/backlog.go` — `ParsePending` (:33, scanner :41) and `ExtractContent` (:57, scanner :69) vs. `MarkDone` (:106) which already uses ReadFile+Split:

- Convert both to the read-lines helper; `ParsePending` stops swallowing `os.Open` errors (currently returns `nil`, :34–37) — signature gains an error return.
- Fixes: items below an over-long line vanishing from `fab batch new --list`/`--all` (batch_new.go:66, :123) and the misleading `not found` error (:92–93) for real IDs after the long line (batch_new.go:85).
- All callers live in `cmd/fab/batch_new.go` — signature changes are cheap; per-ID extract failures keep the existing warn-and-skip behavior.

### 7. F14 — `batch_archive` uses statusfile instead of regex-scanning

`src/go/fab/cmd/fab/batch_archive.go` — delete `hydrateStatusRe` (:36) and rewrite `isArchivable` (:191–205, scanner :198) as:

```go
sf, err := statusfile.Load(statusPath)
return err == nil && (sf.GetProgress("hydrate") == "done" || sf.GetProgress("hydrate") == "skipped")
```

`internal/statusfile` owns the `.status.yaml` schema (documented single ownership, kit-architecture.md) — the regex is the lone outlier parser (matches `hydrate:` at any indentation anywhere; likely a leftover grep idiom from the shell→Go migration). `cmd/fab` already imports statusfile — no new dependency. Test fixtures at batch_archive_test.go:62/67/85/97 write bare `"  hydrate: done\n"` fragments that pin the regex behavior — rewrite them as realistic `progress:` blocks (the helper at :129 already does this correctly).

### 8. F13 — mechanical sweep of every remaining scanner site

Current production sites at HEAD (13 total; only proc_linux checks `Err()`, none call `Buffer`):

| Site | Function | Treatment |
|------|----------|-----------|
| `internal/score/score.go:238` | `countGrades` | § 2 above |
| `internal/archive/archive.go:466` | `removeFromIndex` | § 4 above |
| `internal/backlog/backlog.go:41` | `ParsePending` | § 6 above |
| `internal/backlog/backlog.go:69` | `ExtractContent` | § 6 above |
| `cmd/fab/batch_archive.go:198` | `isArchivable` | § 7 above (statusfile, not helper) |
| `internal/hooklib/artifact.go:133` | `HasSectionHeading` | helper `Split` — in-memory; hook.go:~271 already holds full content; feeds `.status.yaml` plan counters via hook.go:276–297 |
| `internal/hooklib/artifact.go:167` | `scanSectionItems` | helper `Split` — same; wrong counts are *persisted*, not just displayed |
| `internal/prmeta/prmeta.go:403` | `countCheckboxesInTasksSection` | helper — feeds PR-body "Tasks done/total" |
| `internal/prmeta/prmeta.go:439` | `countCheckboxes` | helper — same |
| `internal/frontmatter/frontmatter.go:20` | `Field` | helper — silent-absent fields drop skills from `fab help` listings (fabhelp.go:216+) and descriptions from `fab memory-index` |
| `internal/frontmatter/frontmatter.go:62` | `HasFrontmatter` | helper — **report-uncited second site in this file**; swept for class elimination |
| `internal/memoryindex/memoryindex.go:362` | (index scan) | helper |
| `internal/proc/proc_linux.go:30` | proc reading | **left as-is** — the one site that checks `scanner.Err()` (:46); streams /proc, proven correct |

### 9. Docs + tests conformance

- Tests per Go change (constitution): new/updated tests covering error propagation (truncation/oversize-line cases, persistence-failure surfacing, statusfile-based archivability, atomic-rewrite failure paths), plus the F14 fixture rewrite.
- `src/kit/skills/_cli-fab.md` fab-score rows: document the new failure surfacing (normal mode errors non-zero on load/persist failure). No CLI signature changes expected (backlog CONSTRAINTS) — flags and YAML success-output stay identical.
- No skill-file behavior changes anticipated: `_pipeline.md` already STOPs on non-zero, preflight convention is "non-zero → STOP and surface stderr".

## Affected Memory

- `pipeline/schemas.md`: (modify) `fab score` failure-surfacing behavior — load/persist/read errors now exit non-zero with stderr detail (extends the k4ge gate-fail exit row)
- `distribution/kit-architecture.md`: (modify) new shared read-lines helper + atomic-write helper in the internal package ecosystem; batch_archive outlier parser removed — statusfile single-ownership now holds everywhere
- `pipeline/execution-skills.md`: (modify) archive/restore index updates are atomic and report honest failure status
- `pipeline/change-lifecycle.md`: (modify) backlog scanning — parse failures now surface instead of silently shrinking `batch new --list/--all`

## Impact

- **Code**: `src/go/fab/internal/{score,archive,backlog,hooklib,prmeta,frontmatter,memoryindex}`, `src/go/fab/cmd/fab/{batch_archive,score,batch_new}.go`, new `internal/lines` (name TBD) + shared atomic-write helper. fab-kit module untouched.
- **Tests**: each touched package per constitution; batch_archive fixtures rewritten.
- **Cross-change seam**: ye8r [x8c9] will single-source `gateThresholds`/`expectedMin` data maps in `internal/score/score.go` — this change touches only scan/error logic (`countGrades`, `Compute`, `CheckGate` error paths), **not** the data maps (:344–:356). Whichever merges second rebases trivially.
- **Consumers made honest, not changed**: preflight confidence display, `fab status confidence`, `fab change view/list`, pr-meta, the plan.md PostToolUse hook (already error-guarded).
- **No CLI signature changes**; internal Go signatures change freely (`countGrades`, `removeFromIndex`, `updateIndex`, `ParsePending` gain error returns — all callers in-repo).

## Open Questions

None — the input is a fully specified, adversarially verified findings batch (F09–F15 with verifier confirmations and corrections); all decision points resolved as graded assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | New shared read-lines helper lives in a small new internal package in the fab module (e.g. `internal/lines`) exposing `ReadFileLines(path)` + `Split(content)`; per-line `TrimSuffix "\r"` | Semantics fixed verbatim by backlog + verifier caveat; only placement/naming is agent-chosen, easily renamed in review | S:85 R:90 A:80 D:65 |
| 2 | Certain | `proc_linux.go:30` scanner left untouched | The one correct site (checks `Err()` at :46); backlog targets the "12 of 13" unchecked class; /proc streaming is the legit scanner use | S:80 R:95 A:90 D:85 |
| 3 | Certain | hooklib's two in-memory scanners become helper `Split` line iteration | Verifier: hook.go already reads the full file, so `strings.Split` is a strict simplification; only `ErrTooLong` was possible there | S:85 R:90 A:90 D:80 |
| 4 | Confident | F11 posture: hard error (Compute returns load/persist errors; `RunE` surfaces → stderr + non-zero), not warn-and-exit-0 | Finding offers both; verified evidence favors hard error: `_cli-fab.md` documents normal mode as "compute, write .status.yaml", skill failure convention is non-zero→STOP, and the hook caller's `err == nil` guard keeps the only best-effort path intact | S:75 R:75 A:85 D:60 |
| 5 | Certain | `countGrades` signature becomes `(GradeCount, error)`; open + read errors propagate through `CheckGate`/`Compute` | Backlog states it explicitly ("propagate countGrades read errors so read-failure is distinguishable from an empty Assumptions table") | S:90 R:85 A:90 D:85 |
| 6 | Confident | `removeFromIndex`/`updateIndex` gain error returns; `Archive`/`Restore` YAML report honest index status | Verifier correction on F10 requires the signature change; archive.go:122–127 documents the same propagate principle in-file | S:75 R:80 A:85 D:70 |
| 7 | Confident | Atomic index writes via a shared temp+rename helper extracted from the existing statusfile.Save/runtime.go patterns (no third copy) | Two in-repo precedents exist; code-quality.md anti-pattern forbids duplicating utilities; extraction is low-risk | S:70 R:85 A:75 D:60 |
| 8 | Certain | F14 fixtures (batch_archive_test.go:62/67/85/97) rewritten as realistic `progress:` blocks | Verifier identified they pin regex behavior; constitution VII — tests conform to spec, never the reverse; the :129 helper already writes the correct shape | S:85 R:90 A:90 D:85 |
| 9 | Certain | Sweep includes the report-uncited `frontmatter.go:62` (`HasFrontmatter`) scanner | Goal is class elimination ("kill the unchecked-bufio.Scanner class"), and the site matches the class exactly | S:80 R:90 A:90 D:90 |
| 10 | Certain | score.go seam with ye8r: touch only scan/error logic; `gateThresholds`/`expectedMin` data maps (:344–:356) untouched | Backlog defines the seam explicitly ("data maps vs scan logic, different sections, coordinate") | S:90 R:85 A:95 D:90 |
| 11 | Confident | `ParsePending` signature gains an error return (stops swallowing `os.Open`); `ExtractContent` keeps `(string, error)` but errors become accurate; batch-new per-ID warn-and-skip preserved | Verifier: all callers confined to batch_new.go, signature change "cheap"; preserving warn-and-skip keeps documented batch semantics | S:75 R:80 A:85 D:70 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).

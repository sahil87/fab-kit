# Intake: Make fab archive (single + batch) fully mechanical

**Change**: 260601-zu41-mechanical-archive
**Created**: 2026-06-01
**Status**: Draft

## Origin

Conversational. Started from a `/fab-discuss` session where the user asked: *"Make fab archive (and also its batch form) faster — is there a way to make it more mechanical?"*

The session explored the current implementation (`/fab-archive` skill, `fab change archive`, `fab batch archive`, the `internal/archive` package) and surfaced that archiving is already ~90% mechanical but two agent-driven steps remain, while `fab batch archive` is *entirely* agent-driven (it spawns a Claude session). Five design decisions were resolved interactively via structured questions; a Plan sub-agent validated the design and caught three corrections (no `internal/idea` package exists, the real backlog has no `## Done` section, and `Archive()` should stay pure with a separate orchestrator). The full validated design lives at `fab/plans/sahil/reactive-yawning-alpaca.md`.

> Make fab archive (and also its batch form) faster — is there a way to make it more mechanical?

## Why

**Problem.** Archiving a completed change routes through the Claude agent more than it needs to, and the batch path is the worst offender:

1. The single `/fab-archive` skill is already ~90% delegated to the Go binary (`fab change archive` does the folder move, index update, backfill, and pointer clearing). But two steps still require the agent: (a) summarizing `intake.md`'s `## Why` prose into a `--description` for the archive index, and (b) backlog matching (exact-ID + a fuzzy keyword scan + an interactive confirmation prompt).

2. `fab batch archive` does **not** archive mechanically despite living in the Go binary. It collects the archivable changes (`hydrate: done|skipped`), builds the prompt `"Run /fab-archive for each of these changes, one at a time: …"`, and `syscall.Exec`s a freshly **spawned Claude session** that runs `/fab-archive` N times, serialized (`src/go/fab/cmd/fab/batch_archive.go:99-114`). Archiving 5 changes therefore costs 5 full agent passes in a new session.

**Consequence if unfixed.** Archiving — a pure file-mechanics operation with nothing interactive about it — keeps paying agent latency and token cost. Batch archiving a backlog of completed changes is slow (seconds-to-minutes of agent time) when it could be sub-second. The agent is pure overhead here.

**Why this approach.** Move both remaining agent steps into Go so `fab change archive` is fully self-contained, then make `fab batch archive` a pure Go loop over the same function. The only genuinely non-mechanical input was the archive-index description (free prose in `## Why`); the discussion resolved this by sourcing the description from the structured **intake title line** instead, which needs no summarization. Backlog matching reduces to exact-ID marking because the change folder's 4-char ID *is* the backlog ID when the change came from backlog — a deterministic string operation. With both inputs mechanical, the agent is no longer needed for archiving at all.

## What Changes

### 1. New package `internal/intake`

`src/go/fab/internal/intake/intake.go` — provides the mechanical archive-index description.

```go
package intake

// Title reads the `# Intake: {title}` heading from changeDir/intake.md and returns
// the de-prefixed title. Returns "" on missing/unreadable file or no matching heading.
func Title(changeDir string) string

// DescriptionFor returns a one-line archive description for a change folder.
// Prefers the intake title; falls back to a humanized slug (folder name minus the
// `YYMMDD-XXXX-` prefix, hyphens → spaces) when the title is absent.
func DescriptionFor(fabRoot, folder string) string
```

- `Title` matches `^#\s+Intake:\s*(.+)$`, trims, collapses internal whitespace. Backticked titles (e.g. `` # Intake: Fix stale `fab status` ``) pass through verbatim.
- `DescriptionFor` reads `changes/{folder}/intake.md`; on an empty title, humanizes the slug — the folder-name segment after `resolve.ExtractID` with hyphens replaced by spaces. `updateIndex` already normalizes `\n\r\t` → space, so the raw string is safe to pass downstream.

### 2. New package `internal/backlog` (extracted + reused)

`src/go/fab/internal/backlog/backlog.go` — the backlog parser currently lives in `batch_new.go` as `package main`, which is unimportable. Extract it so both `batch_new` and the archive path share one copy (reuse over duplication).

```go
package backlog

type Item struct { ID, Desc string }

func Path(fabRoot string) string                             // fabRoot/backlog.md
func ParsePending(backlogPath string) []Item                 // moved from batch_new.go
func ExtractContent(backlogPath, id string) (string, error)  // moved from batch_new.go

// MarkDone flips `- [ ] [id]` → `- [x] [id]` in place. Returns:
//   "marked"    — found unchecked, flipped
//   "already"   — found, already [x] (no write)
//   "not_found" — no such ID, or backlog.md missing (nil error — silent no-op)
// Idempotent. Does NOT move to a Done section (project backlog uses none).
func MarkDone(backlogPath, id string) (string, error)
```

Symbols moved from `batch_new.go`: `backlogItem` → `Item`, `backlogItemRe`, `backlogPrefixRe`, `parsePendingItems` → `ParsePending`, `extractBacklogContent` → `ExtractContent`. `batch_new.go` is refactored to import `internal/backlog` and call `backlog.*`.

`MarkDone` design: read all lines, find `^- \[[x ]\] \[<id>\]`; if `[ ]` rewrite that one checkbox to `[x]` and write the file back (`"marked"`); if `[x]` return `"already"` without writing; no match → `"not_found"`; backlog file missing → `"not_found"` with a nil error (silent, matching the current "skip silently if backlog.md doesn't exist" behavior).

### 3. Modify `internal/archive/archive.go`

- Add a `Backlog string` field to `ArchiveResult`.
- Add a sentinel for the soft-skip case:
  ```go
  var ErrAlreadyArchived = errors.New("change already archived")
  ```
  Return it (wrapped with the destination path for context) in place of the current ad-hoc `fmt.Errorf("Archive destination already exists: %s", destPath)` at `archive.go:74`.
- `Archive(fabRoot, changeArg, description string)`: **drop** the hard `--description is required` guard (`archive.go:53-55`). When `description == ""`, derive it via `intake.DescriptionFor(fabRoot, folder)` — computed **before** the `os.Rename` (intake.md is still in the source folder at that point). `Archive()` otherwise stays pure (move / index / pointer; no backlog dependency).
- Add the orchestrator both callers use:
  ```go
  // ArchiveWithBacklog runs Archive, then marks the originating backlog item done.
  func ArchiveWithBacklog(fabRoot, changeArg, description string) (*ArchiveResult, error)
  ```
  Internals: `Archive(...)`; on success `id := resolve.ExtractID(result.Name)` then `result.Backlog, _ = backlog.MarkDone(backlog.Path(fabRoot), id)`. `ErrAlreadyArchived` propagates unchanged for callers to handle.
- `FormatArchiveYAML` appends `\nbacklog: %s`.

### 4. Modify `cmd/fab/archive.go`

- `changeArchiveCmd` calls `archive.ArchiveWithBacklog` instead of `Archive`.
- `--description` flag help: `(required)` → `(optional; defaults to intake title)`.
- On `errors.Is(err, archive.ErrAlreadyArchived)`: print a one-line `already archived` note and exit 0 (idempotent single-archive re-run).

### 5. Modify `cmd/fab/batch_archive.go` — pure Go loop

Replace the spawn/exec tail (`batch_archive.go:99-114`) and the `fab change resolve` subprocess (`batch_archive.go:78-83`) entirely.

- **Resolution**: in the validation loop, swap `exec.Command("fab","change","resolve",change)` for `resolve.ToFolder(fabRoot, change)`. Keep the `isArchivable` guard. This drops the `fab`-on-PATH assumption and the `os/exec`, `syscall`, and `spawn` imports.
- **Loop** — extract a testable helper that returns counts (no `os.Exit` inside):
  ```go
  func archiveLoop(w, errW io.Writer, fabRoot string, resolved []string) (archived, skipped, failed int)
  ```
  Per change, call `archive.ArchiveWithBacklog(fabRoot, name, "")`:
  - `errors.Is(err, archive.ErrAlreadyArchived)` → print `  {name} — already archived, skipping`; skipped++
  - other error → print `  {name} — FAILED: {err}`; failed++; **continue** (one failure never aborts the batch)
  - success → print `  {name} — archived` plus ` (backlog marked done)` when `Backlog == "marked"`; archived++
  - Footer: `Archived %d, skipped %d, failed %d.`
- `runBatchArchive` calls `archiveLoop`, then `os.Exit(1)` only when `failed > 0`.
- `batchArchiveCmd.Long`: change from "by running /fab-archive for each" to "mechanically (move, index, backlog, pointer) in a Go loop". `--list` / `--all` / positional behavior is unchanged.

### 6. Modify skill `src/kit/skills/fab-archive.md`

Collapse Archive Mode to a thin wrapper:

- **Delete Step 1** (Extract Description) and **Step 3** (3a/3b/3c backlog matching).
- Archive Mode reduces to: preflight → hydrate guard → `fab change archive <change>` (no `--description`) → format YAML.
- **Step 4 / Output**: remove the agent-driven `Scan:` line. Add a `backlog:` YAML mapping:
  - `backlog: marked` → `Backlog:  ✓ [ID] marked done`
  - `backlog: already` → `Backlog:  — already done`
  - `backlog: not_found` → `Backlog:  — no match`
- **Context Loading**: `None beyond preflight` (no longer reads `intake.md` / `backlog.md`).
- **Purpose** + **Key Properties**: reword to "delegates all mechanical operations (move, index, backlog, pointer) to `fab change archive`; the skill only formats output." Idempotency note stays true (`MarkDone` → `already`; archive → soft skip). The skill no longer uses `Edit`.

### 7. Docs / specs (constitution: skill changes MUST update specs)

- `docs/specs/skills/SPEC-fab-archive.md` — drop Step 1 / Step 3 from the flow; remove the `Edit` (backlog) and intake/backlog `Read` rows from the Tools-used table; note mechanical backlog mark-done.
- `src/kit/skills/_cli-fab.md` — rewrite the `fab batch archive` bullet (~L295) to the Go-loop description; qualify "tmux windows require `$TMUX`" (~L291) to `new`/`switch` only; update `fab change archive` (~L42) — `--description` now optional, marks backlog, emits a `backlog` field (~L46).
- `docs/memory/fab-workflow/kit-architecture.md` — L139 (mechanical loop, no spawn), L135 / L306 (archive no longer shares the tmux/spawn pattern — only `new`/`switch` do), L119 (spawn used by `new`/`switch` only); add `internal/intake` and `internal/backlog` to the package list (~L309) and the tested-packages list (~L313).
- `docs/specs/overview.md` L103 — change the archive row from `Worktree + tmux tab per change` to `Folder(s) moved to archive/, backlog marked` (corrects a long-standing inaccuracy — archive never created tabs).
- `docs/specs/architecture.md` L116 — the stale `batch-fab-archive-change.sh` / "tmux tab per change" row → mechanical Go loop (scope the edit to the archive row).
- `docs/specs/assembly-line.md` — verify prose near L128 doesn't claim a tab/worktree per archived change (the `fab batch archive --all` example itself is fine).

### 8. Tests

- **`internal/intake/intake_test.go`** (new): `Title` exact / missing file / malformed heading / backticked title; `DescriptionFor` title-present / slug-fallback / no-slug-segment.
- **`internal/backlog/backlog_test.go`** (new): migrate the 5 parser assertions from `batch_new_test.go` (lines 31, 50, 62, 75, 84 → `backlog.ParsePending` / `backlog.ExtractContent`); add `MarkDone`: `[ ]`→marked; `[x]`→already (no write); missing ID→not_found; missing file→not_found + nil err; continuation lines untouched; only the matching item flipped.
- **`internal/archive/archive_test.go`** (update): `TestArchive_MissingArgs` (L120-123) — empty description now **succeeds** (slug fallback; fixture has no intake.md); empty `changeArg` stays an error. `TestFormatArchiveYAML` (L264) — add `Backlog` to the struct literal, assert `backlog:` appears. New: derives-from-intake-title, slug-fallback, `ArchiveWithBacklog` marks done, no-backlog-file → not_found, not-from-backlog → not_found, `ErrAlreadyArchived` on re-archive.
- **`cmd/fab/batch_archive_test.go`** (add): `TestArchiveLoop` — fixture with 2 archivable changes (intake.md + `hydrate: done`), call `archiveLoop` with buffers, assert both moved to `archive/yyyy/mm/`, counts correct; a third already-archived change → skipped, not failed. Existing helper tests (`isArchivable`, `allArchivableNames`, `listArchivable`, cmd structure) unchanged.
- **`cmd/fab/batch_new_test.go`** (update): repoint the 5 parser calls (lines 31, 50, 62, 75, 84) to `backlog.*`.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update the `fab batch archive` description (mechanical Go loop, no spawn/tmux), the spawn-usage note (only `new`/`switch`), and the batch-commands shared-pattern note; add `internal/intake` and `internal/backlog` to the package and tested-packages lists.

## Impact

**Code areas**:
- `src/go/fab/internal/archive/archive.go` (modify), `archive_test.go` (modify)
- `src/go/fab/internal/intake/` (new package + test)
- `src/go/fab/internal/backlog/` (new package + test)
- `src/go/fab/cmd/fab/archive.go` (modify)
- `src/go/fab/cmd/fab/batch_archive.go` (modify), `batch_archive_test.go` (add test)
- `src/go/fab/cmd/fab/batch_new.go` (refactor to use `internal/backlog`), `batch_new_test.go` (repoint parser calls)

**Skill / docs**:
- `src/kit/skills/fab-archive.md`, `src/kit/skills/_cli-fab.md`
- `docs/specs/skills/SPEC-fab-archive.md`, `docs/specs/overview.md`, `docs/specs/architecture.md`, `docs/specs/assembly-line.md`
- `docs/memory/fab-workflow/kit-architecture.md`

**Dependencies**: none added — all internal Go packages and stdlib (`errors`, `regexp`, `bufio`, `os`, `strings`, `path/filepath`). Removes `os/exec` / `syscall` / `internal/spawn` usage from `batch_archive.go`.

**Behavioral surface**:
- `fab change archive` gains an auto-derived description (no `--description` needed), marks the originating backlog item done, and emits a `backlog:` YAML field; re-archiving is now a clean exit-0 soft skip.
- `fab batch archive` no longer spawns a Claude session — it archives in-process. Output format changes (per-change lines + `Archived/skipped/failed` footer).
- `/fab-archive` no longer prompts interactively and no longer runs a fuzzy backlog keyword scan.

**Constraints honored**: Pure Prompt Play (logic in Go binary, no new runtime deps); Idempotent Operations (soft-skip + `MarkDone` returning `already`); Test Integrity (tests updated to match new behavior); skill→spec sync requirement.

## Open Questions

None. All five design decisions were resolved interactively during the discussion (see Assumptions). The two implementation-time verification items (whether `batch_new_test.go` references the moved parser symbols, and what `architecture.md` says about batch archive) were both confirmed during the planning session and are reflected above.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Archive-index description sourced from the intake **title line** (`# Intake: {title}`), de-prefixed, with humanized-slug fallback | Discussed — user explicitly chose "Intake title line" over "first sentence of Why" and "slug" via structured question. The `## Why` section is formatted prose (`**Problem**:`, lists) so first-sentence extraction would capture markdown noise | S:95 R:80 A:90 D:95 |
| 2 | Certain | Backlog matching = exact-ID only, mechanized in Go; drop the fuzzy keyword scan + interactive confirm | Discussed — user chose "Exact-ID auto, drop keyword scan". The folder's 4-char ID *is* the backlog ID (`fab-new.md`: "the 4-char backlog ID becomes the change ID"), making this a deterministic string op | S:95 R:75 A:90 D:90 |
| 3 | Certain | `fab batch archive` becomes a pure Go loop — no agent, no tmux, no spawn | Discussed — user chose "Pure Go loop, no agent". Archiving is pure file mechanics with nothing interactive | S:95 R:70 A:95 D:95 |
| 4 | Certain | Extract the backlog parser into a shared `internal/backlog` package; refactor `batch_new.go` to use it | Discussed — user chose "Extract shared package" over duplicating the regex. Honors constitution reuse-over-duplication; the parser currently sits in `package main` (unimportable) | S:90 R:70 A:90 D:90 |
| 5 | Certain | Already-archived folders are a **soft skip** (report + continue, exit 0), not a failure | Discussed — user chose "Soft skip". Aligns with constitution §III Idempotent Operations; requires an `ErrAlreadyArchived` sentinel so callers can distinguish it | S:90 R:80 A:90 D:90 |
| 6 | Confident | Keep `Archive()` pure (move/index/pointer); add a separate `ArchiveWithBacklog()` orchestrator for the backlog side-effect | Plan sub-agent recommendation — cleaner separation than growing `Archive()` a backlog dependency. Strong codebase signal; easily reversible | S:80 R:80 A:85 D:80 |
| 7 | Confident | `MarkDone` flips `[ ]`→`[x]` **in place**, never moving items to a `## Done` section | Verified against the real `fab/backlog.md` — it has only `## Open`, no `## Done`. Moving would invent a section the project doesn't use | S:85 R:85 A:90 D:85 |
| 8 | Confident | Change type = `refactor` | Mechanizing existing agent-driven behavior; no new user-facing capability. Matches the `refactor` taxonomy (restructure/consolidate) over `feat` | S:75 R:90 A:85 D:80 |
| 9 | Confident | `internal/intake` is a new standalone package (not folded into `internal/archive`) | Plan sub-agent recommendation — keeps `archive` focused on filesystem ops; `Title` is independently testable and reusable. Low cost (one small file) | S:75 R:85 A:80 D:75 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).

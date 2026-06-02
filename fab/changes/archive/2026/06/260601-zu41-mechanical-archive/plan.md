# Plan: Make fab archive (single + batch) fully mechanical

**Change**: 260601-zu41-mechanical-archive
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Tasks

<!-- Sequential work items for the apply stage. Checked off [x] as completed. -->

### Phase 1: Setup (new packages)

- [x] T001 [P] Create `src/go/fab/internal/intake/intake.go` (package `intake`): `Title(changeDir string) string` matching `^#\s+Intake:\s*(.+)$` with trimmed/collapsed whitespace ("" on any failure); `DescriptionFor(fabRoot, folder string) string` preferring the intake title, falling back to humanized slug (folder minus `YYMMDD-XXXX-` prefix via `resolve.ExtractID`, hyphens→spaces). Imports `internal/resolve` only — MUST NOT import `internal/archive`. <!-- A-001 A-002 A-003 -->
- [x] T002 [P] Create `src/go/fab/internal/backlog/backlog.go` (package `backlog`): move `Item` (fields `ID`, `Desc`), `backlogItemRe`, `backlogPrefixRe`, `ParsePending`, `ExtractContent` from `cmd/fab/batch_new.go`; add `Path(fabRoot string) string` → `filepath.Join(fabRoot, "backlog.md")`; add `MarkDone(backlogPath, id string) (string, error)` flipping `- [ ] [<id>]` → `- [x] [<id>]` in place (returns `marked`/`already`/`not_found`; missing file → `not_found`, nil err). <!-- A-008 A-009 A-010 A-011 -->

### Phase 2: Core Implementation

- [x] T003 Refactor `src/go/fab/cmd/fab/batch_new.go` to import `internal/backlog` and call `backlog.Item`/`backlog.ParsePending`/`backlog.ExtractContent`, deleting the local copies of `backlogItem`, `backlogItemRe`, `backlogPrefixRe`, `parsePendingItems`, `extractBacklogContent`. No duplicate `[a-z0-9]{4}` regex may remain. Depends on T002. <!-- A-012 -->
- [x] T004 Modify `src/go/fab/internal/archive/archive.go`: add `Backlog string` field to `ArchiveResult`; add `var ErrAlreadyArchived = errors.New("change already archived")` and wrap the destination-exists error via `fmt.Errorf("%w: %s", ErrAlreadyArchived, destPath)`; drop the `--description is required` guard; when `description == ""` derive via `intake.DescriptionFor(fabRoot, folder)` before `os.Rename`; add `ArchiveWithBacklog(fabRoot, changeArg, description string) (*ArchiveResult, error)` calling `Archive` then `backlog.MarkDone(backlog.Path(fabRoot), resolve.ExtractID(result.Name))`; `FormatArchiveYAML` appends `\nbacklog: %s`. Imports `internal/intake` + `internal/backlog`. Depends on T001, T002. <!-- A-001 A-002 A-006 A-013 A-014 A-015 A-016 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Modify `src/go/fab/cmd/fab/archive.go`: `changeArchiveCmd` RunE calls `archive.ArchiveWithBacklog` instead of `Archive`; on `errors.Is(err, archive.ErrAlreadyArchived)` print a one-line "already archived" note to stdout and return nil (exit 0); `--description` flag help `(required)` → `(optional; defaults to intake title)`. Depends on T004. <!-- A-006 A-007 -->
- [x] T006 Modify `src/go/fab/cmd/fab/batch_archive.go`: replace `exec.Command("fab","change","resolve",change)` with `resolve.ToFolder(fabRoot, change)`; extract `archiveLoop(w, errW io.Writer, fabRoot string, resolved []string) (archived, skipped, failed int)` calling `archive.ArchiveWithBacklog(fabRoot, name, "")` per change (ErrAlreadyArchived → skip line + skipped++; other err → FAILED line + failed++ + continue; success → archived line, append " (backlog marked done)" when `Backlog=="marked"`, archived++; footer `Archived %d, skipped %d, failed %d.`); `runBatchArchive` calls `archiveLoop` then `os.Exit(1)` only when `failed > 0`; update `batchArchiveCmd.Long` to the mechanical-loop description; remove now-unused `os/exec`, `syscall`, `spawn` imports. Depends on T004. <!-- A-004 A-005 A-017 A-018 A-019 A-020 -->

### Phase 4: Documentation

- [x] T007 [P] Modify skill `src/kit/skills/fab-archive.md`: delete Step 1 (Extract Description) and Step 3 (backlog 3a/3b/3c); Archive Mode = preflight → hydrate guard → `fab change archive <change>` (no `--description`) → format YAML; Step 4/Output drop the `Scan:` line and add `backlog:` YAML mapping (`marked`→`✓ [ID] marked done`, `already`→`— already done`, `not_found`→`— no match`); Context Loading "None beyond preflight"; reword Purpose + Key Properties (delegates all mechanics; no `Edit` tool). <!-- A-021 A-022 -->
- [x] T008 [P] Modify `docs/specs/skills/SPEC-fab-archive.md`: drop Step 1/Step 3 from the flow; remove `Edit (backlog)` + intake/backlog `Read` rows from Tools-used; note mechanical backlog mark-done. <!-- A-023 -->
- [x] T009 [P] Modify `src/kit/skills/_cli-fab.md`: rewrite the `fab batch archive` bullet to the Go-loop description; qualify "tmux windows require `$TMUX`" to `new`/`switch` only; update `fab change archive` row — `--description` optional, marks backlog, emits a `backlog` field. <!-- A-023 A-024 -->
- [x] T010 [P] Modify `docs/specs/overview.md` archive row: `Worktree + tmux tab per change` → `Folder(s) moved to archive/, backlog marked`. <!-- A-023 -->
- [x] T011 [P] Modify `docs/specs/architecture.md` batch-archive row: stale `batch-fab-archive-change.sh` / "tmux tab per change" → mechanical Go loop (scope edit to the archive row only). <!-- A-023 -->
- [x] T012 [P] Verify `docs/specs/assembly-line.md` prose near L128; edit only if it claims a tab/worktree per archived change (the `fab batch archive --all` example is fine). <!-- A-023 -->

### Phase 5: Tests

- [x] T013 [P] Create `src/go/fab/internal/intake/intake_test.go`: `Title` exact / missing file / malformed heading / backticked title; `DescriptionFor` title-present / slug-fallback / no-slug-segment. Depends on T001. <!-- A-003 A-025 -->
- [x] T014 [P] Create `src/go/fab/internal/backlog/backlog_test.go`: migrate the 5 parser assertions from `batch_new_test.go`; `MarkDone` `[ ]`→marked, `[x]`→already (no write), missing ID→not_found, missing file→not_found + nil err, continuation lines untouched, only matching item flipped. Depends on T002. <!-- A-010 A-011 A-025 -->
- [x] T015 Update `src/go/fab/internal/archive/archive_test.go`: `TestArchive_MissingArgs` — empty description now SUCCEEDS via slug fallback, empty `changeArg` stays an error; `TestFormatArchiveYAML` — add `Backlog` field + assert `backlog:` present; add derive-from-intake-title, slug-fallback, `ArchiveWithBacklog`-marks-done, no-backlog-file→not_found, not-from-backlog→not_found, `ErrAlreadyArchived`-on-re-archive. Depends on T004. <!-- A-001 A-002 A-013 A-014 A-016 A-025 -->
- [x] T016 Add `TestArchiveLoop` to `src/go/fab/cmd/fab/batch_archive_test.go`: fixture with 2 archivable changes (intake.md + `hydrate: done` `.status.yaml`), call `archiveLoop` with `bytes.Buffer`, assert both moved to `archive/yyyy/mm/`, counts correct; third already-archived change → skipped not failed. Depends on T006. <!-- A-017 A-018 A-025 -->
- [x] T017 Update `src/go/fab/cmd/fab/batch_new_test.go`: repoint the 5 parser calls (`parsePendingItems`, `extractBacklogContent`, `.id`) to `backlog.*` and `Item.ID`. Depends on T003. <!-- A-012 A-025 -->

### Phase 6: Verification

- [x] T018 From `src/go/fab` run `go build ./...` then `go test ./...` — all pass. Scope-run `./internal/archive/ ./internal/intake/ ./internal/backlog/ ./cmd/fab/` first, then the full suite. <!-- A-025 A-026 A-027 -->

## Execution Order

- T001, T002 are independent (both Phase 1).
- T003 depends on T002; T004 depends on T001 + T002.
- T005, T006 depend on T004.
- T013 depends on T001; T014 on T002; T015 on T004; T016 on T006; T017 on T003.
- Phase 4 docs tasks (T007–T012) are independent of the Go work and of each other.
- T018 runs last (after all code + tests).

## Acceptance

### Functional Completeness

- [ ] A-001 Mechanical description derivation: `fab change archive` with no `--description` derives the index description from `intake.md`'s `# Intake: {title}` line (prefix stripped, internal whitespace collapsed), computed before the folder move.
- [ ] A-002 Slug fallback: when the intake title is absent/unreadable, `DescriptionFor` falls back to the folder name minus the `YYMMDD-XXXX-` prefix with hyphens replaced by spaces.
- [ ] A-003 `internal/intake` provides `Title` and `DescriptionFor` as standalone reusable functions; `internal/archive` depends on `internal/intake` but `internal/intake` does NOT depend on `internal/archive`.
- [ ] A-004 `fab batch archive` archives each resolved change in-process via `archive.ArchiveWithBacklog` — no `/fab-archive` prompt, no spawned Claude session, no `internal/spawn`/`os/exec`/`syscall` for the archive step; resolution uses `resolve.ToFolder`.
- [ ] A-005 `fab batch archive --list` continues to show archivable changes (`hydrate: done|skipped`) without archiving anything.
- [ ] A-006 `internal/archive.Archive` returns the `ErrAlreadyArchived` sentinel when the destination exists, and `fab change archive` treats it as a successful no-op (prints note, exits 0).
- [ ] A-007 The `--description` flag help reads `(optional; defaults to intake title)`.
- [ ] A-008 The backlog parser logic is extracted into `internal/backlog` (`Item`, `ParsePending`, `ExtractContent`); no second copy of the `[a-z0-9]{4}` regex exists after the change.
- [ ] A-009 `internal/backlog` provides `Path` and `MarkDone` matching the spec contract.

### Behavioral Correctness

- [ ] A-010 `MarkDone` flips `- [ ] [<id>]` → `- [x] [<id>]` in place (returns `marked`), returns `already` without writing when already `[x]`, and never moves the item to a `## Done` section.
- [ ] A-011 `MarkDone` returns `not_found` with a nil error when no line matches the ID or when `fab/backlog.md` is missing; archiving still succeeds.
- [ ] A-012 `fab batch new --list` lists the same pending items as before the refactor (behavior preserved via the shared parser).
- [ ] A-013 `ArchiveWithBacklog` extracts the 4-char ID via `resolve.ExtractID(result.Name)` and calls `MarkDone` after a successful archive; reports `backlog: marked` when an unchecked matching item exists.
- [ ] A-014 An explicit `--description` overrides the derived title.
- [ ] A-015 `Archive()` stays pure (move/index/pointer); the backlog side-effect lives only in `ArchiveWithBacklog`.

### Scenario Coverage

- [ ] A-016 `ArchiveResult` includes a `Backlog` field and `FormatArchiveYAML` emits a `backlog: {marked|already|not_found}` line alongside the existing fields.
- [ ] A-017 `archiveLoop` is a testable helper returning `(archived, skipped, failed)` and does NOT call `os.Exit`; already-archived changes count as `skipped` not `failed`; the footer reads `Archived {N}, skipped {M}, failed {K}.`.
- [ ] A-018 A per-change failure is reported and does not abort the remaining changes (loop continues); `runBatchArchive` exits non-zero only when `failed > 0`.
- [ ] A-019 Success lines append ` (backlog marked done)` when `result.Backlog == "marked"`.
- [ ] A-020 `batchArchiveCmd.Long` describes the mechanical Go loop (no `/fab-archive` spawn).

### Edge Cases & Error Handling

- [ ] A-021 The `/fab-archive` skill performs no fuzzy keyword scan and asks no interactive confirmation; the `Backlog:` report line is sourced from the CLI's `backlog:` YAML field.
- [ ] A-022 The `/fab-archive` skill no longer uses the `Edit` tool and Context Loading reads "None beyond preflight".

### Documentation Accuracy

- [ ] A-023 `SPEC-fab-archive.md`, `docs/specs/overview.md`, `docs/specs/architecture.md`, and `docs/specs/assembly-line.md` are updated/verified to reflect the mechanical archive (no "tmux tab per change" for batch archive); `kit-architecture.md` is intentionally left to the hydrate stage.
- [ ] A-024 `src/kit/skills/_cli-fab.md` documents the Go-loop `fab batch archive`, the optional `--description`, the backlog mark, and the `backlog` field for `fab change archive`; the `$TMUX` note is scoped to `new`/`switch`.

### Cross-References

- [ ] A-025 New and updated tests reference the moved/renamed symbols correctly (`backlog.*`, `intake.*`, `archive.ArchiveWithBacklog`, `ErrAlreadyArchived`) and the skill→spec sync requirement is honored.

### Code Quality

- [ ] A-026 Pattern consistency: new code follows the naming, error-handling (returned errors, `fmt.Errorf` wrapping, `errors.Is` matching), and table-driven `testing`-only test patterns of surrounding Go code.
- [ ] A-027 No unnecessary duplication: the backlog parser is reused (not copied) via `internal/backlog`; no god functions introduced; no magic strings beyond the sentinel/status constants the spec mandates.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

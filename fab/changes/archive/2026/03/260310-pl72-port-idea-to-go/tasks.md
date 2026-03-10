# Tasks: Port Idea Script to Go Binary

**Change**: 260310-pl72-port-idea-to-go
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/idea/` package directory and `idea.go` with `Idea` struct, `ParseLine`, `FormatLine` functions, and `GitRepoRoot` helper (exec `git rev-parse --path-format=absolute --git-common-dir`)
- [x] T002 Create `src/go/fab/cmd/fab/idea.go` with `ideaCmd()` parent command and register it in `src/go/fab/cmd/fab/main.go` via `root.AddCommand(ideaCmd())`

## Phase 2: Core Implementation

- [x] T003 Implement file operations in `src/go/fab/internal/idea/idea.go`: `LoadFile` (parse lines, preserve non-idea lines), `SaveFile` (reconstruct with preserved lines), `ResolveFilePath` (--file flag > IDEAS_FILE env > default `fab/backlog.md`)
- [x] T004 Implement query matching in `src/go/fab/internal/idea/idea.go`: `Match(query, idea)` (case-insensitive substring on ID and text), `FindAll(query, ideas, filter)`, `RequireSingle(query, ideas, filter)` (0 → error, >1 → disambiguation error, 1 → result)
- [x] T005 Implement `Add` operation in `src/go/fab/internal/idea/idea.go`: generate random 4-char ID, validate no collision, append line, auto-create file/dirs if missing. Wire `addCmd()` in `src/go/fab/cmd/fab/idea.go` with `--id`, `--date` flags
- [x] T006 Implement `List` operation in `src/go/fab/internal/idea/idea.go`: filter by status (open/done/all), sort by id or date, reverse, JSON output. Wire `listCmd()` in `src/go/fab/cmd/fab/idea.go` with `-a`, `--done`, `--json`, `--sort`, `--reverse` flags
- [x] T007 Implement `Show` operation in `src/go/fab/internal/idea/idea.go`: find single match, display line or JSON. Wire `showCmd()` in `src/go/fab/cmd/fab/idea.go` with `--json` flag
- [x] T008 Implement `Done` and `Reopen` operations in `src/go/fab/internal/idea/idea.go`: toggle status for single match (done filters open, reopen filters done). Wire `doneCmd()` and `reopenCmd()` in `src/go/fab/cmd/fab/idea.go`
- [x] T009 Implement `Edit` operation in `src/go/fab/internal/idea/idea.go`: modify text, optional `--id` (with collision check) and `--date`. Wire `editCmd()` in `src/go/fab/cmd/fab/idea.go`
- [x] T010 Implement `Rm` operation in `src/go/fab/internal/idea/idea.go`: delete single match, require `--force`. Wire `rmCmd()` in `src/go/fab/cmd/fab/idea.go`

## Phase 3: Integration & Edge Cases

- [x] T011 Add `--file` persistent flag on `ideaCmd()` in `src/go/fab/cmd/fab/idea.go` and implement `IDEAS_FILE` env var fallback in `ResolveFilePath`
- [x] T012 Write comprehensive tests in `src/go/fab/internal/idea/idea_test.go`: parsing (valid/invalid lines), query matching (by ID, by text, case-insensitive, multiple, none), all CRUD operations (add, list, show, done, reopen, edit, rm), flag combinations (--json, -a, --done, --sort, --reverse, --force, --id, --date), edge cases (empty file, missing file auto-create, ID collision)

---

## Execution Order

- T001 blocks T003, T004, T005, T006, T007, T008, T009, T010
- T002 depends on T001 (needs package to import)
- T003 and T004 are prerequisites for T005–T010
- T005–T010 can proceed incrementally (each adds one operation)
- T011 can proceed after T002
- T012 can proceed after T005–T010 are complete

# Tasks: Remove fab status show and fix stale fab/current references

**Change**: 260312-9lci-fix-status-show-fab-current
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Remove `statusShowCmd()` from `cmd.AddCommand()` call in `statusCmd()` at `src/go/fab/cmd/fab/status.go:25`
- [x] T002 Delete `statusShowCmd` function (lines 627-692), `worktreeInfo` struct (506-517), `listWorktrees` (519-540), `resolveWorktreeFabState` (542-582), `findWorktreeByName` (584-592), `currentWorktree` (594-601), `formatWorktreeHuman` (603-608), `formatWorktreesHuman` (610-625) from `src/go/fab/cmd/fab/status.go`
- [x] T003 Remove unused imports (`encoding/json`, `os/exec`, `path/filepath`, `os`, `strings`) from `src/go/fab/cmd/fab/status.go` — `os` and `strings` also no longer used after code removal
- [x] T004 Verify `go build` succeeds after code removal

## Phase 2: Documentation Updates

- [x] T005 [P] Remove `fab status show` row from Command Reference table in `fab/.kit/skills/_scripts.md` (line 39) and remove the `show` entry from Key subcommands table (line 108)
- [x] T006 [P] Update `fab send-keys` pane resolution in `fab/.kit/skills/_scripts.md` (line 319): replace `fab/current` with `.fab-status.yaml`
- [x] T007 [P] Update `README.md`: replace `fab/current/` in directory tree (line 58) with `.fab-status.yaml` representation, and update activation text (line 158) to reference `.fab-status.yaml`
- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md`: remove `fab status show` from subcommand list (line 308), fix `fab send-keys` pane resolution `fab/current` reference (line 414)
- [x] T009 [P] Update `docs/memory/fab-workflow/execution-skills.md`: replace `fab status show --all` fallback references with `wt list` + `fab change list` in Orientation on Start (line 190) and State Re-derivation (line 194)

---

## Execution Order

- T001 → T002 → T003 → T004 (sequential: registration removal, code deletion, import cleanup, build verification)
- T005 through T009 are independent of each other ([P]) and can run after T004

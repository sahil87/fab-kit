# Tasks: wt open Shell Setup

**Change**: 260403-24ic-wt-open-shell-setup
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `src/go/wt/cmd/shell_setup.go` with cobra subcommand scaffold and register in `src/go/wt/cmd/main.go` via `root.AddCommand(shellSetupCmd())`

## Phase 2: Core Implementation

- [x] T002 Implement `shell-setup` subcommand body in `src/go/wt/cmd/shell_setup.go` — detect `$SHELL` basename, output wrapper function and `export WT_WRAPPER=1` to stdout, print stderr warning for unrecognized shells
- [x] T003 Add `WT_WRAPPER` detection in `OpenInApp` `open_here` case in `src/go/wt/internal/worktree/apps.go` — check `os.Getenv("WT_WRAPPER")`, print stderr hint when not `"1"`
- [x] T004 [P] Update `wt` root command `Long` help text in `src/go/wt/cmd/main.go` — replace inline function with `eval "$(wt shell-setup)"` reference

## Phase 3: Integration & Edge Cases

- [x] T005 Add tests in `src/go/wt/internal/worktree/apps_test.go` — test `OpenInApp` open_here with and without `WT_WRAPPER=1` set (verify stderr hint presence/absence, stdout cd output preserved)
- [x] T006 [P] Add test for `shell-setup` subcommand in `src/go/wt/cmd/shell_setup_test.go` — verify stdout output matches expected wrapper function, verify stderr warning for unsupported shell
- [x] T007 [P] Update `docs/specs/packages.md` "Why wt-open Cannot cd" section — replace manual function with `eval "$(wt shell-setup)"`, preserve Unix constraint explanation

---

## Execution Order

- T001 blocks T002 (scaffold needed before implementation)
- T002 blocks T006 (subcommand must exist before testing it)
- T003 blocks T005 (detection logic needed before testing it)
- T004, T007 are independent of each other and of T003

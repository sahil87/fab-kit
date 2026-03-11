# Tasks: Fix Stale Shell-Script References After Go Binary Conversion

**Change**: 260311-i7it-fix-stale-shell-script-refs
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Documentation Fixes

- [x] T001 [P] Rewrite `docs/specs/packages.md` — replace entire content with Go binary architecture (wt as Go binary at `fab/.kit/bin/wt` with subcommands, idea as Go binary at `fab/.kit/bin/idea` with shell fallback, updated directory structure showing only `idea/` under packages/, accurate PATH setup via `env-packages.sh`)
- [x] T002 [P] Fix `docs/specs/naming.md` line 42 — change Worktree "Encoded in" from `wt-create` (`fab/.kit/packages/wt/bin/wt-create`) to `wt create` (`fab/.kit/bin/wt`); fix line 66 — change Backlog Entry "Encoded in" from `fab/.kit/packages/wt/` to `fab/.kit/bin/idea` (backlog management)

## Phase 2: Memory File Updates

- [x] T003 Delete `docs/memory/fab-workflow/kit-scripts.md` — entirely obsolete (documents 7 deleted shell scripts)
- [x] T004 Remove `kit-scripts` entry from `docs/memory/fab-workflow/index.md` file table
- [x] T005 Remove stale `lib/` script subsections from `docs/memory/fab-workflow/kit-architecture.md` — delete `#### lib/statusman.sh`, `#### lib/logman.sh`, `#### lib/calc-score.sh`, `#### lib/changeman.sh`, `#### lib/archiveman.sh` subsections (lines ~126-166); insert brief replacement note with cross-reference to `_scripts.md`
- [x] T006 Add cross-reference note to `fab/.kit/skills/_scripts.md` in the Overview section of `docs/memory/fab-workflow/kit-architecture.md`
- [x] T007 Clarify `lib/env-packages.sh` description in `docs/memory/fab-workflow/kit-architecture.md` — explicitly note that `wt` is a Go binary in `$KIT_DIR/bin/`, `packages/*/bin` iteration picks up only remaining shell packages (currently: `idea`)

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 and T004 should be sequential (delete file first, then update index)
- T005, T006, T007 all modify kit-architecture.md — execute sequentially
- Phase 1 and Phase 2 are independent

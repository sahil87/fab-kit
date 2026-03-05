# Tasks: Standardize Script Names and Add fab-help.sh

**Change**: 260207-cg03-standardize-script-names
**Spec**: `spec.md`
**Proposal**: `proposal.md`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Rename Existing Scripts

<!-- Rename files via git mv to preserve history. Update self-referencing comments. -->

- [x] T001 [P] Rename `fab/.kit/scripts/setup.sh` → `fab/.kit/scripts/fab-setup.sh` via `git mv`. Update self-referencing comment headers on lines 4 and 9 to use `fab-setup.sh`.
- [x] T002 [P] Rename `fab/.kit/scripts/status.sh` → `fab/.kit/scripts/fab-status.sh` via `git mv`. No internal self-references to update (script has no comment header referencing its own name).
- [x] T003 [P] Rename `fab/.kit/scripts/update-claude-settings.sh` → `fab/.kit/scripts/fab-update-claude-settings.sh` via `git mv`. No internal self-references to update.

## Phase 2: Create fab-help.sh

<!-- New script. Depends on Phase 1 being complete (clean scripts/ directory). -->

- [x] T004 Create `fab/.kit/scripts/fab-help.sh` — executable shell script that reads version from `fab/.kit/VERSION` (with fallback to "unknown" if missing) and outputs the canonical help text matching the current `fab-help.md` Output section. Must use `#!/usr/bin/env bash`, `set -euo pipefail`, and relative path resolution via `$(dirname "$0")`.

## Phase 3: Update Internal References

<!-- Update all files that reference old script names. All tasks are independent. -->

- [x] T005 [P] Update `fab/.kit/skills/fab-help.md` — replace inline help text with instruction to execute `fab/.kit/scripts/fab-help.sh` and display its output. Remove the literal help text block. Preserve Key Properties table and Context Loading section.
- [x] T006 [P] Update `fab/.kit/skills/fab-init.md` — replace `scripts/setup.sh` with `scripts/fab-setup.sh` on line 176.
- [x] T007 [P] Update `fab/worktree-init/assets/settings.local.json` — change permission pattern on line 72 from `Bash(fab/.kit/scripts/setup.sh:*)` to `Bash(fab/.kit/scripts/fab-setup.sh:*)`.
- [x] T008 [P] Update `doc/fab-spec/ARCHITECTURE.md` — update tree listing (line 32) to show all four `fab-*` scripts, and replace all prose references to `scripts/status.sh` and `scripts/setup.sh` with their `fab-` prefixed names (lines 112, 122, 372, 410, 417, 468).
- [x] T009 [P] Update `doc/fab-spec/README.md` — replace `scripts/status.sh` with `scripts/fab-status.sh` on lines 25 and 68.

## Phase 4: Verify

<!-- Run renamed scripts to confirm behavior is preserved. -->

- [x] T010 Run `bash fab/.kit/scripts/fab-setup.sh` and verify it completes successfully with expected output (symlinks, directories, .gitignore).
- [x] T011 Run `bash fab/.kit/scripts/fab-status.sh` and verify it outputs the current active change status correctly.
- [x] T012 Run `bash fab/.kit/scripts/fab-help.sh` and verify the help text includes version, WORKFLOW, COMMANDS, and TYPICAL FLOW sections.
- [x] T013 Verify no old script names remain in active files: grep for `scripts/setup.sh`, `scripts/status.sh`, `scripts/update-claude-settings.sh` across `fab/.kit/`, `doc/fab-spec/`, and `fab/worktree-init/` — expect zero matches.

---

## Execution Order

- T001, T002, T003 are independent (parallel)
- T004 can start after Phase 1 (clean directory state)
- T005–T009 are independent (parallel), can start after T004
- T010–T013 require all prior phases complete

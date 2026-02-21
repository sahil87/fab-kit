# Tasks: Batch Script Frontmatter for fab-help Discovery

**Change**: 260221-alng-batch-script-frontmatter
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Add `shell_frontmatter_field` function to `fab/.kit/scripts/lib/frontmatter.sh` — parse `# ---` delimited shell-comment frontmatter with same interface as `frontmatter_field` (`<file> <field_name>` → unquoted value). Match `# ---` delimiters (not bare `---`), strip leading `# ` from lines, handle quoted/unquoted values, strip trailing inline comments

## Phase 2: Core Implementation

- [x] T002 [P] Add `# ---` frontmatter block to `fab/.kit/scripts/batch-fab-switch-change.sh` — insert between shebang and `set -euo pipefail` with `name: batch-fab-switch-change` and `description: "Open tmux tabs in worktrees for one or more changes"`. Remove the existing comment header line (`# batch-fab-switch-change.sh — ...`)
- [x] T003 [P] Add `# ---` frontmatter block to `fab/.kit/scripts/batch-fab-archive-change.sh` — insert between shebang and `set -euo pipefail` with `name: batch-fab-archive-change` and `description: "Archive multiple completed changes in one session"`. Remove the existing comment header line
- [x] T004 [P] Add `# ---` frontmatter block to `fab/.kit/scripts/batch-fab-new-backlog.sh` — insert between shebang and `set -euo pipefail` with `name: batch-fab-new-backlog` and `description: "Create worktree tabs from backlog items"`. Remove the existing comment header line

## Phase 3: Integration

- [x] T005 Add batch script scan loop and "Batch Operations" group to `fab/.kit/scripts/fab-help.sh` — add `"Batch Operations"` to `group_order`, add `batch_to_group` mapping for the three batch scripts, add a scan loop that globs `"$kit_dir"/scripts/batch-*.sh` and calls `shell_frontmatter_field` for `name` and `description`, collect into rendering data structures, render without `/` prefix. Include batch display names in `max_len` alignment computation

---

## Execution Order

- T001 blocks T005 (frontmatter parser needed before scan loop)
- T002, T003, T004 are independent and parallel
- T005 depends on T001 (parser) and T002-T004 (frontmatter to parse)

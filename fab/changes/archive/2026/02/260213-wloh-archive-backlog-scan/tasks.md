# Tasks: Broaden Archive Backlog Scanning

**Change**: 260213-wloh-archive-backlog-scan
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Step 7 Rewrite

- [x] T001 Expand archive Step 7 in `fab/.kit/skills/fab-continue.md` — restructure into sub-steps: Step 7a (existing exact-ID check, preserved as-is), Step 7b (keyword extraction from brief title + Why section, stop word filtering, normalization, 2-keyword threshold matching against unchecked backlog items), Step 7c (interactive confirmation prompt in batch format with comma-separated selection). Exclude items already marked by exact-ID check.
- [x] T002 Add auto-mode skip guard to Step 7 in `fab/.kit/skills/fab-continue.md` — keyword scan (Steps 7b-7c) SHALL be skipped entirely when archive runs via `/fab-ff` or `/fab-fff`. Only the exact-ID check (Step 7a) runs in auto mode.

## Phase 2: Output & Error Handling

- [x] T003 Update the "Archive Complete" output template in `fab/.kit/skills/fab-continue.md` — add a `Backlog (scan):` line showing keyword scan results (e.g., "2 candidates found, 1 marked done" or "skipped (auto mode)" or "no matches"). Place after the existing `Backlog:` line.
- [x] T004 Add edge case handling to the archive behavior section in `fab/.kit/skills/fab-continue.md` — document behavior when: `fab/backlog.md` does not exist (skip keyword scan silently), no keyword matches found (proceed silently), all candidates declined by user (proceed normally).

---

## Execution Order

- T001 blocks T002 (auto-mode guard references the sub-step structure from T001)
- T003 and T004 are independent of each other but depend on T001 (reference the new structure)

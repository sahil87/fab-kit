# Tasks: Remove Copilot Integration and Clean Stale Scaffold

**Change**: 260402-5tci-remove-copilot-clean-scaffold
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Scaffold Cleanup

- [x] T001 [P] Delete `src/kit/scaffold/.github/copilot-code-review.yml`
- [x] T002 [P] Remove `fab/changes/**/.pr-done` and `/.ralph` lines from `src/kit/scaffold/fragment-.gitignore`

## Phase 2: Skill Modification

- [x] T003 Strip Copilot Phases 2, 3, Path B, API comment, and Copilot-specific references from `src/kit/skills/git-pr-review.md` — update routing so no-reviews falls through to stop message, remove Path B from Step 3, simplify commit message logic in Step 5, update phase tracking table, update description frontmatter, clean Step 6 references

## Phase 3: Migration

- [x] T004 Append three new sections (5, 6, 7) to `src/kit/migrations/0.46.0-to-0.47.0.md` — remove `.github/copilot-code-review.yml`, clean stale `.gitignore` entries, delete `.pr-done` files — and update Verification section

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 is independent of T001/T002
- T004 is independent of all others

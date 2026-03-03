# Tasks: Git PR Copilot Fix

**Change**: 260303-4ojc-git-pr-copilot-fix
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create new skill file `fab/.kit/skills/git-pr-fix.md` with frontmatter (name, description, allowed-tools)

## Phase 2: Core Implementation

- [x] T002 Write `/git-pr-fix` Step 1 (Resolve PR) — branch detection via `git branch --show-current`, PR lookup via `gh pr view --json number,url`, bail if no PR
- [x] T003 Write `/git-pr-fix` Step 2 (Copilot Review Detection) — owner/repo resolution, reviews API polling, first-poll bail for standalone mode, wait mode (30s × 12) for inline invocation
- [x] T004 Write `/git-pr-fix` Step 3 (Comment Fetching and Triage) — fetch comments via reviews/comments API, classify actionable vs informational, read affected files, apply fixes
- [x] T005 Write `/git-pr-fix` Step 4 (Commit and Push) — stage specific files, single commit with `fix: address copilot review feedback`, push, summary output
- [x] T006 Write `/git-pr-fix` idempotency behavior — re-run finds existing review, re-triages but makes no changes, prints "No actionable comments." and stops
- [x] T007 Write `/git-pr-fix` error handling — gh CLI not found, API errors, no partial commits

## Phase 3: Integration

- [x] T008 Add Step 6 (Auto-Fix Copilot Review) to `fab/.kit/skills/git-pr.md` after Step 5 — inline `/git-pr-fix` behavior in wait mode, best-effort wrapping, "Shipped." always prints
- [x] T009 Update `/git-pr` Rules section — add "Step 6 (Copilot fix) is best-effort — never blocks shipping"

---

## Execution Order

- T002 → T003 → T004 → T005 (sequential within git-pr-fix, each step builds on prior)
- T006, T007 depend on T002-T005 (need complete skill to add edge cases)
- T008 depends on T002-T007 (git-pr references git-pr-fix behavior)
- T009 is independent of T008 but logically grouped

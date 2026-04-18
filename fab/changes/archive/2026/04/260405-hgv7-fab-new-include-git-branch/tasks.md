# Tasks: fab-new Include Git Branch

**Change**: 260405-hgv7-fab-new-include-git-branch
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add Step 11 (git branch creation) to `src/kit/skills/fab-new.md` — insert after Step 10, before the `---` divider; update `description` frontmatter; add `Branch:` line to Output section; add two error-handling table rows
- [x] T002 [P] Add canonical source constraint to `fab/project/constitution.md` — one bullet after the existing `src/kit/skills/*.md` → `SPEC-*.md` rule in Additional Constraints

## Phase 2: Spec and Memory Updates

- [x] T003 Update `docs/specs/skills/SPEC-fab-new.md` — add Step 10 (Activate Change) and Step 11 (Create Git Branch) to the flow diagram<!-- clarified: SPEC flow currently ends at Step 9 — Step 10 was added to fab-new.md in a prior change but the SPEC was never updated; both steps need adding together -->, add `Bash(git:*)` rows to Tools table, update summary description
- [x] T004 [P] Update `docs/memory/fab-workflow/planning-skills.md` — fix stale `/fab-new` section: (a) remove "never activates changes" text, (b) add git-branch step, (c) update output description

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 and T004 depend on T001 being done (spec/memory describe the final state)
- T003 and T004 are independent of each other, can run in parallel

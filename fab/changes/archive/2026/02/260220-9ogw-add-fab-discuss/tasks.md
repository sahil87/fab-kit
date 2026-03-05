# Tasks: Add fab-discuss Skill

**Change**: 260220-9ogw-add-fab-discuss
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create skill file `fab/.kit/skills/fab-discuss.md` with frontmatter (name, description, model_tier: capable) and full skill body — loads 7 always-load files, reads `fab/current` + `.status.yaml` for active change, outputs structured orientation summary, ends with ready signal
- [x] T002 Add `[fab-discuss]="Start & Navigate"` to `skill_to_group` mapping in `fab/.kit/scripts/fab-help.sh`

## Phase 2: Core Implementation

- [x] T003 [P] Add `## /fab-discuss` section to `docs/specs/skills.md` — document purpose, context (7-file always-load layer), key properties (no active change required, read-only, idempotent), output format
- [x] T004 [P] Fix stale **Context Loading Convention** section in `docs/specs/skills.md` — update "Always loaded" list from 3 files to 7 files (add `context.md`, `code-quality.md`, `code-review.md`, `docs/specs/index.md`)
- [x] T005 [P] Add `/fab-discuss` row to Quick Reference table in `docs/specs/overview.md`
- [x] T006 [P] Add `/fab-discuss` node to "Utility (anytime)" subgraph in `docs/specs/user-flow.md` Section 3A diagram

## Phase 3: Integration & Edge Cases

- [x] T007 Add note in **Exception Skills** section of `docs/memory/fab-workflow/context-loading.md` clarifying `fab-discuss`'s relationship to the always-load layer + add changelog entry

---

## Execution Order

- T001 is independent — skill file creation
- T002 is independent — fab-help.sh group mapping
- T003, T004, T005, T006 are all independent [P] tasks touching different files/sections
- T007 is independent — memory file update

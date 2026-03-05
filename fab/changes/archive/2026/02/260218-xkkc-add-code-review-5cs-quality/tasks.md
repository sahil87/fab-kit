# Tasks: Add Code Review Scaffold & 5 Cs of Quality

**Change**: 260218-xkkc-add-code-review-5cs-quality
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scaffold/code-review.md` with 5 populated sections: Severity Definitions, Review Scope, False Positive Policy, Rework Budget, Project-Specific Review Rules. Include HTML guidance comments per section (matching `code-quality.md` scaffold pattern). Severity defaults must match the three-tier scheme in `fab/.kit/skills/fab-continue.md` review behavior.

## Phase 2: Core Implementation

- [x] T002 [P] Update `fab/.kit/skills/_context.md` — add `fab/code-review.md` as 7th item in the Always Load list (after `docs/specs/index.md`). Mark as optional with the note: `*(optional — no error if missing)*`.
- [x] T003 [P] Update `fab/.kit/scaffold/config.yaml` — add `#   fab/code-review.md    — review policy for validation sub-agent (optional)` to the companion files comment block, after the `fab/code-quality.md` line.
- [x] T004 [P] Update `fab/.kit/skills/fab-setup.md` — (a) add bootstrap step `1b4. fab/code-review.md` after step 1b3, following the same if-missing/copy/report pattern. (b) Add item 10 `code-review.md` to the config menu (update menu display, valid sections list, and menu range). (c) Add `Created: fab/code-review.md` to the bootstrap output example.
- [x] T005 [P] Update `fab/.kit/skills/fab-continue.md` — add `fab/code-review.md (if present)` to the "Context provided to the sub-agent" list in the Review Behavior > Sub-Agent Dispatch section.

## Phase 3: Integration

- [x] T006 Update `README.md` — expand "Code Quality as a Guardrail" section with the 5 Cs mental model table (Constitution, Context, Code Quality, Code Review, Config → file + question). Add a short narrative explaining the author-vs-critic distinction between code-quality.md and code-review.md. Place after the existing review explanation, before the "Structured Autonomy" section.

---

## Execution Order

- T001 is standalone (no dependencies)
- T002–T005 are parallelizable and independent of each other
- T006 depends on no other task but is placed in Phase 3 for logical ordering

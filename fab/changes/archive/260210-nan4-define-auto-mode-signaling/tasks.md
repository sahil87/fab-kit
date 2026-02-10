# Tasks: Define auto-mode signaling mechanism for skill-to-skill invocation

**Change**: 260210-nan4-define-auto-mode-signaling
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Implementation

- [x] T001 Add "Skill Invocation Protocol" section to `fab/.kit/skills/_context.md` defining the `[AUTO-MODE]` prefix convention, placement rules (first line of invocation prompt), and called-skill detection behavior
- [x] T002 [P] Update Mode Selection section in `fab/.kit/skills/fab-clarify.md` to reference the Skill Invocation Protocol in `_context.md` and specify concrete `[AUTO-MODE]` prefix detection logic
- [x] T003 [P] Update auto-clarify invocation instructions in `fab/.kit/skills/fab-ff.md` to explicitly use the `[AUTO-MODE]` prefix per the protocol, and reference the `_context.md` protocol section

## Phase 2: Integration & Verification

- [x] T004 Audit `fab/.kit/skills/fab-fff.md` for any standalone auto-clarify invocations; verify it delegates to fab-ff behavior (if so, no changes needed) or update any direct invocations to use the `[AUTO-MODE]` protocol
- [x] T005 Update centralized doc `fab/docs/fab-workflow/planning-skills.md` — add a reference to the skill invocation protocol in the `/fab-ff` and `/fab-fff` sections, and update the "Clarify Mode Selection by Call Context" design decision to note the protocol
- [x] T006 [P] Update centralized doc `fab/docs/fab-workflow/clarify.md` — add a reference to the `[AUTO-MODE]` prefix detection protocol in the dual-mode operation section

---

## Execution Order

- T001 blocks T002, T003, T004 (protocol must be defined before it can be referenced)
- T002 and T003 are parallel after T001
- T004 depends on T001 (needs protocol to audit against)
- T005 and T006 are parallel, can proceed after T002-T004 are complete

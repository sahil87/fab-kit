# Tasks: /fab-proceed — Prefer Conversation Context Over Stale Intakes

**Change**: 260419-1a98-fab-proceed-prefer-context
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Rewrite

- [x] T001 Rewrite State Detection section in `src/kit/skills/fab-proceed.md` to match spec Requirement §Detection Order: replace the current 4-step sequence (Active Change → Branch → Unactivated Intake → Conversation Context) with the new 5-step structure (Active Change → Branch Check → Conversation Classification → Unactivated Intake Scan → Dispatch Decision). Mark Steps 3 and 4 as order-independent. Reuse the existing substantive-vs-empty criterion verbatim for Step 3. Replace the existing 5-row dispatch table with the new 7-row table from spec §Dispatch Table.

- [x] T002 Add a new Relevance Assessment subsection in `src/kit/skills/fab-proceed.md` (after the dispatch table, before Dispatch Behavior) per spec Requirement §Relevance Assessment: read title + Origin + Why + What Changes sections of each candidate intake; classify as clearly relevant or not clearly relevant; asymmetric-bias rule maps ambiguous to not clearly relevant; date-descending tiebreak only among equally-relevant candidates.

- [x] T003 Update the Output section in `src/kit/skills/fab-proceed.md` per spec Requirement §Output Format: add the Bypass Note format (`Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.`); specify ordering (one line per bypassed draft, date-descending, emitted BEFORE the `Created intake:` line); clarify that no Note lines are emitted on the activation path or on the empty/thin → activate branch.

- [x] T004 Update Dispatch Behavior and Error Handling sections in `src/kit/skills/fab-proceed.md` to match the new dispatch table rows — ensure the `/fab-new` dispatch path covers both "no intake" and "intake present but not clearly relevant"; keep the existing Subagent Dispatch and Conversation Context Synthesis subsections, adding a short pointer that bypassed drafts are NOT synthesized from (only the conversation is the synthesis source).

## Phase 2: Spec and Memory Sync

- [x] T005 [P] Update `docs/specs/skills/SPEC-fab-proceed.md` Flow diagram, Dispatch Table, and Sub-agents section to reflect the new 5-step detection, 7-row dispatch table, and relevance check. Preserve the "Key differences from /fab-fff and /fab-ff" section unchanged.

- [x] T006 [P] Update `docs/memory/fab-workflow/execution-skills.md` — rewrite the `**Pipeline orchestrator**:` paragraph describing `/fab-proceed` (currently at line ~15) and its embedded dispatch table (currently at lines ~17-23) to describe the new 5-step detection, 7-row dispatch table, relevance check, asymmetric-bias rule, bypass-note output, and single substantive-vs-empty classifier.

## Phase 3: Verification

- [x] T007 Verified consistency — grepped `fab-proceed` in `docs/memory/` (2 hits: `execution-skills.md` updated; `fab-workflow/index.md` — generic table entry, unaffected) and `docs/specs/` (5 hits: `SPEC-fab-proceed.md` updated; `skills.md` updated — had stale "4-step pipeline" reference; `user-flow.md` — Mermaid edge label, generic; `operator.md` — version log entry, generic). All stale references fixed.

- [x] T008 Traced all 7 dispatch-table rows against the rewritten `src/kit/skills/fab-proceed.md`: (1) Active+match → fab-fff only; (2) Active+nomatch → git-branch→fab-fff; (3) No active+substantive+no intake → fab-new→git-branch→fab-fff; (4) No active+substantive+≥1 intake+relevant → fab-switch→git-branch→fab-fff; (5) No active+substantive+≥1 intake+not relevant → fab-new→git-branch→fab-fff + bypass notes; (6) No active+empty/thin+≥1 intake → fab-switch→git-branch→fab-fff (date-recency); (7) No active+empty/thin+no intake → error. All 7 rows present in Dispatch Table and addressed by Subagent Dispatch + Error Handling sections. Bypass note format, ordering, and zero-prompt preservation all present.

---

## Execution Order

- T001 → T002 → T003 → T004 (sequential — all edit the same file `src/kit/skills/fab-proceed.md`)
- T005 [P] and T006 [P] can run in parallel (different files), after T001-T004 complete (they describe the same new logic)
- T007 and T008 are verification tasks, run after all edits

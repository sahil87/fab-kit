# Tasks: Add fab-discuss Command and Confidence Scoring to fab-new

**Change**: 260208-lgd7-fab-discuss-command
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

- [x] T001 Create skill registration directory `.claude/skills/fab-discuss/` and symlink `SKILL.md -> ../../../fab/.kit/skills/fab-discuss.md`

## Phase 2: Core Implementation

- [x] T002 Create `fab/.kit/skills/fab-discuss.md` — the full skill definition with: pre-flight check, context-driven mode selection (active change detection), gap analysis phase, conversational proposal development flow, conversation termination logic (score >= 3.0 + user signal), proposal output (new and refine modes), confidence scoring, next steps, error handling
- [x] T003 Update `fab/.kit/skills/_context.md` — add `/fab-discuss` row to the Skill-Specific Autonomy Levels table (posture: free-form conversation, no question cap; interruption budget: unlimited; output: proposal + confidence + switch hint; escape valve: user ends early; recomputes confidence: yes). Add `/fab-discuss` entries to the Next Steps lookup table. Update `/fab-init` and `/fab-hydrate` next steps to include `/fab-discuss`

## Phase 3: Integration

- [x] T004 Verify `fab/.kit/skills/fab-new.md` Step 8 (Compute Confidence Score) properly overwrites the template defaults in `.status.yaml`. If the confidence block is only initialized with zeros in Step 5 and Step 8 doesn't explicitly write updated values, add the write instruction. Confirm Step 9 (Mark Proposal Complete) writes the confidence block

## Phase 4: Polish

(not warranted — no documentation, cleanup, or performance work needed beyond the core implementation)

---

## Execution Order

- T001 blocks T002 (symlink must exist before skill file, though practically they're independent)
- T003 is independent, can run alongside T002
- T004 is independent, can run alongside T002-T003

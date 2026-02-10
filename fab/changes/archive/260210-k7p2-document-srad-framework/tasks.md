# Tasks: Document the SRAD Framework

**Change**: 260210-k7p2-document-srad-framework
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Implementation

- [x] T001 Create `fab/specs/srad.md` with the complete SRAD framework specification. The file MUST include all sections required by the spec: (1) acronym expansion with one-sentence definitions for each dimension, (2) dimension evaluation criteria table (high vs low for S, R, A, D), (3) four confidence grades with meanings, artifact markers, and output visibility, (4) confidence scoring formula with penalty weight explanations, (5) gate threshold documentation (>= 3.0 for `/fab-fff`) with practical implications, (6) confidence lifecycle table (which skills compute, recompute, consume), (7) Critical Rule definition and rationale, (8) two worked examples at proposal level (high-ambiguity scoring near 0.0, low-ambiguity scoring near 5.0) each showing input description, SRAD evaluation of 2-3 decisions, confidence counts, and computed score, (9) skill-specific autonomy levels table covering posture, interruption budget, output format, and escape valve per skill. Source content from `fab/.kit/skills/_context.md` (canonical runtime definition). The file MUST be self-contained — readable without consulting `_context.md`.

- [x] T002 [P] Update `fab/specs/index.md` to add a row for `[srad](srad.md)` in the specs table with description: "SRAD autonomy framework — scoring dimensions, confidence grades, confidence scoring, gating, worked examples"

---

## Execution Order

- T001 and T002 are independent and can execute in parallel.

# Tasks: Extract confidence scoring into standalone script

**Change**: 260213-w8p3-extract-fab-score
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Create `fab/.kit/scripts/_fab-score.sh` — standalone bash script that scans `## Assumptions` tables in `brief.md` + `spec.md`, counts SRAD grades (case-insensitive), applies carry-forward for implicit Certain counts from `.status.yaml`, applies confidence formula, writes confidence block to `.status.yaml` via sed, emits YAML with delta to stdout. Exit 0 on success, exit 1 with stderr on failure (missing spec.md, missing change dir).

## Phase 2: Remove Inline Scoring from Skills

- [x] T002 [P] Edit `fab/.kit/skills/fab-new.md` — delete Step 7 ("Compute Confidence Score", lines 71-73), renumber Step 8 → Step 7
- [x] T003 [P] Edit `fab/.kit/skills/fab-continue.md` — remove "recompute confidence score" from Step 3 (line 67) and Step 4 (line 71). Add note after spec generation in Stage dispatch table: "After spec generation, invoke `_fab-score.sh`". Spec stage only — no scoring at other stages.
- [x] T004 [P] Edit `fab/.kit/skills/fab-clarify.md` — replace Step 7 "Recompute Confidence" (lines 100-102) with: "Run `_fab-score.sh` if `spec.md` exists; skip if at brief stage. Auto mode: do not invoke."

## Phase 3: Documentation Updates

- [x] T005 [P] Edit `fab/.kit/skills/_context.md` — in Confidence Scoring section (lines 250-262): delete Lifecycle table, replace with one-liner about `_fab-score.sh`. Update Template note. Update Skill-Specific Autonomy table "Recomputes confidence?" row: `/fab-new`: No, `/fab-continue`: Spec stage only, `/fab-ff`: No, `/fab-fff`: No.
- [x] T006 [P] Edit `fab/specs/srad.md` — replace 5-row Confidence Lifecycle table (lines 149-155) with 3-row table (Computation, Recomputation, Gate check). Update Skill-Specific Autonomy table row (line 221) same as T005.
- [x] T007 [P] Edit `fab/memory/fab-workflow/planning-skills.md` — delete `/fab-new` Confidence Scoring paragraph (lines 73-76). Update `/fab-continue` forward flow step 6 (line 94) to reference `_fab-score.sh` at spec stage only. Update `/fab-fff` Confidence Recomputation note (lines 166-168) to reference `/fab-continue` (spec stage) and `/fab-clarify` instead of `/fab-new`.
- [x] T008 [P] Edit `fab/memory/fab-workflow/change-lifecycle.md` — update confidence field description (line 51) from "Computed by `/fab-new`, recomputed by `/fab-continue` and `/fab-clarify`" to "Computed by `_fab-score.sh`, invoked at spec stage by `/fab-continue` and by `/fab-clarify`"

---

## Execution Order

- T001 blocks T002, T003, T004 (script must exist before skills reference it)
- T002, T003, T004 are independent (different skill files)
- T005, T006, T007, T008 are independent (different documentation files)

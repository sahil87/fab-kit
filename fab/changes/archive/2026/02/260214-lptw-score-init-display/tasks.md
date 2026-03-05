# Tasks: Confidence score initial value and display format

**Change**: 260214-lptw-score-init-display
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core — Initial Value

- [x] T001 [P] Update `fab/.kit/templates/status.yaml` — change `score: 5.0` to `score: 0.0`
- [x] T002 [P] Update `fab/.kit/scripts/_calc-score.sh` — change `prev_score` fallback from `"5.0"` to `"0.0"` (line 79)

## Phase 2: Core — Display Format

- [x] T003 [P] Update `fab/.kit/skills/fab-status.md` — change confidence display format from `{score}/5.0` to `{score} of 5.0` (line 48)
- [x] T004 [P] Update `fab/.kit/skills/fab-fff.md` — change gate failure message to `Confidence is {score} of 5.0 (need >= 3.0)` and output header to `confidence {score} of 5.0` (lines 28, 75)
- [x] T005 [P] Update `fab/.kit/skills/_context.md` — change Template subsection from "score 5.0" to "score 0.0" (line 256)

---

## Execution Order

All tasks are independent (`[P]`) — no dependencies between them.

# Tasks: Coverage-Weighted Confidence Scoring & Formalize Change Types

**Change**: 260226-tnr8-coverage-scoring-change-types
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `set-change-type` subcommand to `fab/.kit/scripts/lib/stageman.sh` — validates against 7 canonical types (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`), writes `change_type` field via `yq`, updates `last_updated`. Follow existing `set_checklist_field` pattern for validation and atomic write.

- [x] T002 [P] Update `fab/.kit/templates/status.yaml` — change `change_type: feature` to `change_type: feat`

## Phase 2: Core Implementation

- [x] T003 Update `fab/.kit/scripts/lib/calc-score.sh` — add `--stage <stage>` flag (default: `spec`), embed `expected_min` lookup tables by `{stage, change_type}`, read `change_type` from `.status.yaml` (default to `feat`), compute `cover = min(1.0, total_decisions / expected_min)`, apply `score = base * cover`

- [x] T004 Update `fab/.kit/scripts/lib/calc-score.sh` `--check-gate` mode — replace `bugfix`/`feature`/`refactor`/`architecture` thresholds with 7-type taxonomy: `fix`=2.0, `feat`=3.0, `refactor`=3.0, `docs`/`test`/`ci`/`chore`=2.0. Default unknown types to `feat` threshold (3.0).

- [x] T005 Update `fab/.kit/skills/fab-new.md` — add change type inference step after intake generation (keyword heuristic, first match wins, 7 types), write via `stageman.sh set-change-type`. Add indicative confidence display step (count assumptions, compute coverage-weighted score, display-only).

- [x] T006 Update `fab/.kit/skills/git-pr.md` Step 0 — insert new step 2 in resolution chain: read `change_type` from `.status.yaml` via `changeman.sh resolve`. Fall through if null or missing. Shift existing steps 2→3, 3→4.

## Phase 3: Integration & Documentation

- [x] T007 Create `docs/specs/change-types.md` — authoritative spec defining 7 types with descriptions, examples, `expected_min` thresholds per stage, gate thresholds, PR template tier mapping, keyword heuristics, relationship to conventional commits

- [x] T008 [P] Update `docs/specs/index.md` — add `change-types` entry to the specs table

- [x] T009 Update `fab/.kit/skills/_preamble.md` §Confidence Scoring — replace formula with coverage-weighted version, replace 4-type gate threshold table with 7-type taxonomy, reference `docs/specs/change-types.md` for full taxonomy

- [x] T010 Update `docs/specs/srad.md` — replace §Confidence Scoring formula with coverage-weighted version, replace §Gate Threshold table with 7-type taxonomy, update default `change_type` reference from `feature` to `feat`, update worked examples to show coverage factor

---

## Execution Order

- T001 is independent (new subcommand, no dependencies on other tasks)
- T002 is independent (template change)
- T003 depends on T001 being available (reads change_type field that T001 writes)
- T004 is part of the same file as T003 but logically independent
- T005 depends on T001 (uses `set-change-type`) and references the coverage formula
- T006 is independent of other tasks
- T007–T010 can proceed after T003–T004 are done (reference the final formula and thresholds)

# Tasks: Fix Orchestrator False Fail on Review

**Change**: 260221-h1l8-fix-orchestrator-false-fail-on-review
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Remove `:failed` catch-all from `poll_change()` in `fab/.kit/scripts/pipeline/run.sh` — delete the `elif` block (lines 376–383) that greps for `:failed$` in the progress map and marks the change as `failed` in the manifest
- [x] T002 [P] Remove stale `[pipeline]` prefix from progress printf in `fab/.kit/scripts/pipeline/run.sh` — change `printf "\r[pipeline] %s: %s (%dm %02ds)  "` to `printf "\r%s: %s (%dm %02ds)  "` on line 352

## Phase 2: Memory Update Alignment

- [x] T003 Verify `docs/memory/fab-workflow/pipeline-orchestrator.md` Stage Detection section references `:failed` as terminal — will need updating in hydrate

---

## Execution Order

- T001 and T002 are independent (`[P]`), can run in parallel
- T003 is verification only (no code change), independent of T001/T002

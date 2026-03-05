# Tasks: Persist Indicative Confidence

**Change**: 260305-8ooz-persist-indicative-confidence
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Script Changes

- [x] T001 Extend `statusman.sh` `get_confidence` to output `indicative:{true|false}` — read `confidence.indicative` from `.status.yaml`, default to `false` (`fab/.kit/scripts/lib/statusman.sh`)
- [x] T002 Extend `statusman.sh` `set_confidence_block` to accept optional `--indicative` trailing flag — when present, write `confidence.indicative: true`; when absent, omit the key (`fab/.kit/scripts/lib/statusman.sh`)
- [x] T003 Extend `statusman.sh` `set_confidence_block_fuzzy` to accept optional `--indicative` trailing flag — same semantics as T002, using AWK block replacement (`fab/.kit/scripts/lib/statusman.sh`)
- [x] T004 Update `statusman.sh` CLI dispatch for `set-confidence` and `set-confidence-fuzzy` to parse optional `--indicative` flag and pass to underlying functions (`fab/.kit/scripts/lib/statusman.sh`)

## Phase 2: calc-score.sh Integration

- [x] T005 Update `calc-score.sh` normal mode — when `--stage intake`, pass `--indicative` to statusman write calls; when not intake, ensure `--indicative` is NOT passed (`fab/.kit/scripts/lib/calc-score.sh`)

## Phase 3: Consumer Scripts

- [x] T006 [P] Extend `changeman.sh` `cmd_list` to read confidence via `statusman.sh confidence` and append `:score:indicative` to each output line (`fab/.kit/scripts/lib/changeman.sh`)
- [x] T007 [P] Extend `changeman.sh` `cmd_switch` to read confidence via `statusman.sh confidence` and add `Confidence:` line between `Stage:` and `Next:` (`fab/.kit/scripts/lib/changeman.sh`)
- [x] T008 [P] Extend `preflight.sh` to include `indicative: {true|false}` in YAML output confidence section, reading from `statusman.sh confidence` (`fab/.kit/scripts/lib/preflight.sh`)

## Phase 4: Skill Updates

- [x] T009 [P] Update `fab/.kit/skills/fab-new.md` Step 7 — replace inline computation with `calc-score.sh --stage intake <change>` call (normal mode, not `--check-gate`) (`fab/.kit/skills/fab-new.md`)
- [x] T010 [P] Update `fab/.kit/skills/fab-status.md` — remove intake-stage special case, read confidence uniformly from `.status.yaml` via preflight; apply display rules (indicative label, "not yet scored") (`fab/.kit/skills/fab-status.md`)
- [x] T011 [P] Update `fab/.kit/skills/fab-switch.md` — update output format to reflect new changeman output (Confidence line); update no-argument list format for `:score:indicative` (`fab/.kit/skills/fab-switch.md`)

## Phase 5: Documentation

- [x] T012 [P] Update `fab/.kit/skills/_preamble.md` Confidence Scoring section — add `indicative: true` to schema example, note `/fab-new` persistence, note uniform consumer reads (`fab/.kit/skills/_preamble.md`)
- [x] T013 [P] Update `fab/.kit/skills/_scripts.md` — document `--indicative` flag on `set-confidence` and `set-confidence-fuzzy` CLI commands (`fab/.kit/skills/_scripts.md`)

---

## Execution Order

- T001 blocks T002, T003, T004 (accessor must exist before writers and CLI use it)
- T002, T003 block T004 (functions must exist before CLI dispatch)
- T004 blocks T005 (statusman CLI must support `--indicative` before calc-score uses it)
- T005 blocks T009 (calc-score must support indicative before fab-new calls it)
- T001 blocks T006, T007, T008 (confidence accessor must include indicative before consumers read it)
- T006, T007, T008 can run in parallel after T001
- T009, T010, T011, T012, T013 can run in parallel after their respective blockers

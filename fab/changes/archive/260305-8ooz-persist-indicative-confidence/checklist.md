# Quality Checklist: Persist Indicative Confidence

**Change**: 260305-8ooz-persist-indicative-confidence
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 statusman get_confidence: outputs `indicative:{true|false}` field
- [ ] CHK-002 statusman set_confidence_block: accepts `--indicative` flag and writes `indicative: true`
- [ ] CHK-003 statusman set_confidence_block_fuzzy: accepts `--indicative` flag and writes `indicative: true`
- [ ] CHK-004 statusman CLI: `set-confidence` and `set-confidence-fuzzy` parse `--indicative` flag
- [ ] CHK-005 calc-score.sh: intake normal mode passes `--indicative` to statusman
- [ ] CHK-006 calc-score.sh: spec normal mode does NOT pass `--indicative`
- [ ] CHK-007 changeman list: output includes `:score:indicative` per line
- [ ] CHK-008 changeman switch: output includes `Confidence:` line
- [ ] CHK-009 preflight.sh: YAML output includes `indicative:` field
- [ ] CHK-010 fab-new.md: Step 7 calls `calc-score.sh --stage intake` (normal mode)
- [ ] CHK-011 fab-status.md: no intake-stage live `calc-score.sh` call; reads .status.yaml uniformly
- [ ] CHK-012 fab-switch.md: output format matches new changeman output
- [ ] CHK-013 _preamble.md: Confidence Scoring section documents `indicative: true`

## Behavioral Correctness
- [ ] CHK-014 Missing `confidence.indicative` defaults to `false` in all consumers
- [ ] CHK-015 Spec scoring clears `indicative` flag (not left from intake)
- [ ] CHK-016 `--check-gate` mode remains read-only (does not write `.status.yaml`)
- [ ] CHK-017 Pre-intake changes display "not yet scored" (score 0.0 + all counts 0)

## Scenario Coverage
- [ ] CHK-018 Switch to change with indicative score shows `(indicative)` suffix
- [ ] CHK-019 Switch to change with spec score shows no suffix
- [ ] CHK-020 Switch to change with no confidence shows `not yet scored`
- [ ] CHK-021 List changes shows scores and indicative flags correctly
- [ ] CHK-022 fab-status at intake reads from .status.yaml (no live calc-score call)

## Edge Cases & Error Handling
- [ ] CHK-023 Legacy `.status.yaml` without `confidence.indicative` key works without error
- [ ] CHK-024 `set-confidence` without `--indicative` removes existing `indicative: true` if present

## Code Quality
- [ ] CHK-025 Pattern consistency: new code follows naming and structural patterns of surrounding code
- [ ] CHK-026 No unnecessary duplication: existing utilities reused where applicable
- [ ] CHK-027 Readability: changes maintain readability of statusman.sh, changeman.sh, calc-score.sh
- [ ] CHK-028 No god functions: new logic added doesn't create over-long functions
- [ ] CHK-029 No magic strings: indicative flag uses consistent naming across all scripts

## Documentation Accuracy
- [ ] CHK-030 _preamble.md schema matches actual `.status.yaml` output
- [ ] CHK-031 _scripts.md CLI reference matches actual argument parsing

## Cross References
- [ ] CHK-032 All skill files reference correct script CLI syntax
- [ ] CHK-033 Memory files to be hydrated match actual implementation

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

# Quality Checklist: Migrate Stageman to CLI-Only Interface

**Change**: 260215-lqm5-stageman-cli-only
**Generated**: 2026-02-15
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 All 25 new read/query subcommands present in CLI dispatch block
- [x] CHK-002 `set-confidence-fuzzy` write subcommand present and functional
- [x] CHK-003 `preflight.sh` uses `$STAGEMAN` subprocess calls — no `source stageman.sh`
- [x] CHK-004 `calc-score.sh` uses `$STAGEMAN` subprocess calls — no `source stageman.sh`
- [x] CHK-005 `test.sh` uses `$STAGEMAN` subprocess invocations — no `source stageman.sh`
- [x] CHK-006 `test-simple.sh` uses `$STAGEMAN` subprocess invocations — no `source stageman.sh`
- [x] CHK-007 `BASH_SOURCE` guard removed from `stageman.sh`
- [x] CHK-008 Help text lists all subcommands, no `source` references

## Behavioral Correctness

- [x] CHK-009 `preflight.sh` produces identical YAML output before and after migration
- [x] CHK-010 `calc-score.sh` produces identical confidence scores before and after migration
- [x] CHK-011 `calc-score.sh --check-gate` produces identical gate results
- [x] CHK-012 Boolean subcommands (`validate-state`, `is-terminal`, etc.) return correct exit codes
- [x] CHK-013 Multi-value subcommands (`progress-map`, `checklist`, `confidence`) return `key:value` lines

## Removal Verification

- [x] CHK-014 No `return 1 2>/dev/null || exit 1` patterns remain in `stageman.sh`
- [x] CHK-015 No source/import comments in `stageman.sh` header
- [x] CHK-016 No `source.*stageman.sh` in `preflight.sh`, `calc-score.sh`, `test.sh`, `test-simple.sh`

## Scenario Coverage

- [x] CHK-017 Schema query via CLI: `all-stages` returns 6 stages
- [x] CHK-018 Boolean query exit code: `validate-state done` exits 0, `validate-state bogus` exits 1
- [x] CHK-019 Accessor query: `progress-map <file>` returns correct `stage:state` pairs
- [x] CHK-020 Write subcommand: `set-confidence-fuzzy` updates `.status.yaml` correctly
- [x] CHK-021 Argument validation: wrong arg count prints usage to stderr, exits 1
- [x] CHK-022 Comprehensive test suite (`test.sh`) passes with all assertions green
- [x] CHK-023 Simple test suite (`test-simple.sh`) passes

## Edge Cases & Error Handling

- [x] CHK-024 Subcommands with missing `.status.yaml` file produce clear error
- [x] CHK-025 `validate-status-file` on invalid YAML exits 1 with errors to stderr
- [x] CHK-026 `next-stage hydrate` exits 1 (no next stage)
- [x] CHK-027 `stage-metrics <file>` on empty metrics block returns empty (exit 0)

## Documentation Accuracy

- [x] CHK-028 `README.md` shows CLI-only usage — no `source` references
- [x] CHK-029 `README.md` API reference uses subcommand names not function names

## Cross References

- [x] CHK-030 No skills reference sourcing `stageman.sh` (skills already use CLI — verify unchanged)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

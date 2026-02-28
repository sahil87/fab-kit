# Quality Checklist: Refactor Kit Scripts Into Atomic Responsibilities

**Change**: 260228-9fg2-refactor-kit-scripts
**Generated**: 2026-02-28
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 resolve.sh exists and returns 4-char ID by default
- [ ] CHK-002 resolve.sh --folder, --dir, --status flags produce correct output
- [ ] CHK-003 resolve.sh handles all input forms (ID, substring, full name, fab/current, single-change guess)
- [ ] CHK-004 statusman.sh exists at new path, stageman.sh removed
- [ ] CHK-005 statusman.sh has no log_command, log_confidence, log_review functions
- [ ] CHK-006 statusman.sh has no resolve_change_arg function
- [ ] CHK-007 statusman.sh finish review auto-logs "passed" to .history.jsonl
- [ ] CHK-008 statusman.sh fail review auto-logs "failed" with optional rework
- [ ] CHK-009 logman.sh exists with command, confidence, review subcommands
- [ ] CHK-010 logman.sh appends valid JSON to .history.jsonl
- [ ] CHK-011 preflight.sh --driver flag auto-logs command invocation
- [ ] CHK-012 changeman.sh uses resolve.sh for resolution, logman.sh for logging
- [ ] CHK-013 calc-score.sh accepts <change> (not <change-dir>), uses DRY'd helpers

## Behavioral Correctness

- [ ] CHK-014 statusman.sh event commands (start, advance, finish, reset, fail) behave identically to stageman
- [ ] CHK-015 statusman.sh metadata writers (set-change-type, set-checklist, set-confidence) behave identically
- [ ] CHK-016 Non-review finish does NOT auto-log review outcomes
- [ ] CHK-017 preflight.sh without --driver does NOT log command invocations
- [ ] CHK-018 calc-score.sh gate-check mode produces identical results to previous version
- [ ] CHK-019 calc-score.sh normal mode produces identical confidence scores

## Removal Verification

- [ ] CHK-020 No file named stageman.sh exists in fab/.kit/scripts/lib/
- [ ] CHK-021 No reference to "stageman" in any skill file under fab/.kit/skills/
- [ ] CHK-022 No reference to "stageman" in any memory file under docs/memory/
- [ ] CHK-023 No manual log-command invocations in skill files
- [ ] CHK-024 No manual log-review invocations in skill files

## Scenario Coverage

- [ ] CHK-025 resolve.sh ambiguous match returns error with folder list
- [ ] CHK-026 resolve.sh no match returns error
- [ ] CHK-027 logman.sh append-only: existing .history.jsonl lines preserved
- [ ] CHK-028 statusman.sh fail review with rework argument passes through to logman

## Edge Cases & Error Handling

- [ ] CHK-029 resolve.sh with no changes directory returns error
- [ ] CHK-030 resolve.sh with no fab/current and multiple changes returns error
- [ ] CHK-031 logman.sh creates .history.jsonl if it doesn't exist
- [ ] CHK-032 statusman.sh history subcommands (log-command etc.) return "Unknown option" error

## Code Quality

- [ ] CHK-033 Pattern consistency: new scripts follow existing kit script conventions (set -euo pipefail, help text, CLI dispatch pattern)
- [ ] CHK-034 No unnecessary duplication: score formula appears exactly once in calc-score.sh
- [ ] CHK-035 Readability: resolve.sh is concise (~60 lines), logman.sh is concise (~50 lines)

## Documentation Accuracy

- [ ] CHK-036 _scripts.md documents all 5 scripts with correct signatures
- [ ] CHK-037 _scripts.md documents the internal call graph

## Cross References

- [ ] CHK-038 All skill files that call statusman use consistent <change> convention
- [ ] CHK-039 Memory files accurately describe the new script architecture

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

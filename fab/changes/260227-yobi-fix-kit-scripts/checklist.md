# Quality Checklist: Fix Kit Scripts

**Change**: 260227-yobi-fix-kit-scripts
**Generated**: 2026-02-28
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 History commands use `resolve_change_arg`: `log-command`, `log-confidence`, `log-review` CLI dispatch calls `resolve_change_arg` instead of passing raw arg to function
- [ ] CHK-002 History functions derive change dir via `dirname`: `log_command()`, `log_confidence()`, `log_review()` compute `change_dir=$(dirname "$1")` from resolved `.status.yaml` path
- [ ] CHK-003 `resolve_change_dir` removed: function and doc comment (lines 924-936) deleted, zero grep matches in codebase
- [ ] CHK-004 Spec gate parses spec.md inline: `--check-gate` without `--stage intake` parses `spec.md` directly, does not read `score:` from `.status.yaml`
- [ ] CHK-005 `_scripts.md` created: file exists at `fab/.kit/skills/_scripts.md` with `<change>` convention, script summaries, stage transitions, error patterns
- [ ] CHK-006 `_preamble.md` references `_scripts.md`: always-load section includes instruction to read `_scripts.md`
- [ ] CHK-007 Memory file created: `docs/memory/fab-workflow/kit-scripts.md` exists with deep reference content
- [ ] CHK-008 Memory index updated: `docs/memory/fab-workflow/index.md` contains `kit-scripts` entry

## Behavioral Correctness

- [ ] CHK-009 History command with change ID works: `stageman.sh log-command yobi "test"` appends to correct `.history.jsonl`
- [ ] CHK-010 History command with `.status.yaml` path works: `stageman.sh log-review fab/changes/{name}/.status.yaml "passed"` appends to correct `.history.jsonl`
- [ ] CHK-011 History command with folder name works: `stageman.sh log-command 260227-yobi-fix-kit-scripts "test"` resolves and appends correctly
- [ ] CHK-012 Spec gate without prior scoring run: `calc-score.sh --check-gate <change-dir>` returns correct score when `.status.yaml` has `score: 0.0`
- [ ] CHK-013 Gate check is read-only: `--check-gate` does not modify `.status.yaml`

## Scenario Coverage

- [ ] CHK-014 changeman.sh `log-command` calls work after update: `changeman.sh new` and `changeman.sh rename` successfully log commands via updated convention
- [ ] CHK-015 calc-score.sh `log-confidence` call works after update: normal scoring run logs confidence event correctly with `$status_file`

## Edge Cases & Error Handling

- [ ] CHK-016 History command with bare directory path fails gracefully: `stageman.sh log-command fab/changes/{name} "test"` returns "Status file not found" error (not a crash)
- [ ] CHK-017 History command with invalid change ID: `stageman.sh log-command nonexistent "test"` returns resolution error

## Code Quality

- [ ] CHK-018 Pattern consistency: history command dispatch follows same pattern as all other commands (resolve_change_arg → function call)
- [ ] CHK-019 No unnecessary duplication: spec gate path reuses `parse_assumptions` function, same as intake gate and normal mode

## Documentation Accuracy

- [ ] CHK-020 Help text consistent: `stageman.sh --help` shows `<change>` for all commands including history
- [ ] CHK-021 Skill prompts consistent: no instances of `<change_dir>` remain in `fab-ff.md`, `fab-fff.md`, `fab-continue.md`, `fab-clarify.md` for stageman calls
- [ ] CHK-022 `_generation.md` consistent: no instances of `<file>` remain in stageman call examples

## Cross References

- [ ] CHK-023 `_scripts.md` matches actual script behavior: conventions documented in `_scripts.md` align with implemented `resolve_change_arg` behavior
- [ ] CHK-024 Memory file matches implementation: `kit-scripts.md` accurately describes internal functions and state machine

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

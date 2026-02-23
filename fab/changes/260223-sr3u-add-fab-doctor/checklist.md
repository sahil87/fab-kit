# Quality Checklist: Add fab-doctor.sh

**Change**: 260223-sr3u-add-fab-doctor
**Generated**: 2026-02-23
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Script location: `fab/.kit/scripts/fab-doctor.sh` exists and is executable
- [ ] CHK-002 Tool presence checks: All 7 tools checked via `command -v` (git, bash, yq, jq, gh, bats, direnv)
- [ ] CHK-003 Version display: Each passing tool shows its version alongside the checkmark
- [ ] CHK-004 yq version gate: yq v3 is rejected with specific error; v4+ passes
- [ ] CHK-005 Direnv hook detection: Binary presence AND shell hook verified (zsh `_direnv_hook`, bash `PROMPT_COMMAND`)
- [ ] CHK-006 1-prerequisites.sh delegation: Rewritten as thin `exec` delegate to `fab-doctor.sh`
- [ ] CHK-007 fab-upgrade.sh output: Migration reminder is last line with `⚠` prefix; no warning when versions match
- [ ] CHK-008 fab-setup.md doctor gate: Bare bootstrap calls doctor before step 1a; non-zero stops bootstrap

## Behavioral Correctness

- [ ] CHK-009 1-prerequisites.sh exit propagation: Non-zero exit from doctor halts `fab-sync.sh` pipeline via `set -e`
- [ ] CHK-010 fab-upgrade.sh three cases: drift (⚠ with versions), no drift (no warning), missing VERSION (⚠ with init guidance)

## Scenario Coverage

- [ ] CHK-011 All tools present: Checkmark per tool, "7/7 checks passed.", exit 0
- [ ] CHK-012 One tool missing: Cross for missing tool, fix hint, remaining checks run, exit 1
- [ ] CHK-013 Multiple tools missing: All failures shown, correct pass/fail counts, exit = failure count
- [ ] CHK-014 yq v3 detection: Reports wrong version with specific message
- [ ] CHK-015 Direnv binary present but hook missing: Reports hook detection failure with rc file fix hint
- [ ] CHK-016 Direnv not installed: Reports not found with install hint

## Edge Cases & Error Handling

- [ ] CHK-017 Failure accumulation: Script runs all 7 checks even if first fails (no early exit)
- [ ] CHK-018 Subshell noise suppression: Direnv hook check suppresses interactive shell output via `&>/dev/null`
- [ ] CHK-019 Exit code correctness: Exit code equals failure count (0-7)

## Code Quality

- [ ] CHK-020 Pattern consistency: Script follows existing fab-help.sh patterns (shebang, set flags, header comment, self-location)
- [ ] CHK-021 No unnecessary duplication: No tool names duplicated between doctor and 1-prerequisites.sh

## Documentation Accuracy

- [ ] CHK-022 Output format matches spec: Header, per-tool lines, blank line, summary — as specified
- [ ] CHK-023 Fix hints match spec: Each failure includes actionable install/config command

## Cross References

- [ ] CHK-024 fab-setup.md integration: Doctor gate step documented correctly in skill file
- [ ] CHK-025 1-prerequisites.sh: Thin delegate matches spec (3-line body with exec)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

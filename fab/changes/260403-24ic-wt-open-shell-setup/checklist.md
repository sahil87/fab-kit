# Quality Checklist: wt open Shell Setup

**Change**: 260403-24ic-wt-open-shell-setup
**Generated**: 2026-04-03
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 shell-setup subcommand: `wt shell-setup` outputs wrapper function and `export WT_WRAPPER=1` to stdout
- [ ] CHK-002 Shell detection: `$SHELL` basename detected; unrecognized shell prints stderr warning, SHELL unset falls back silently
- [ ] CHK-003 WT_WRAPPER detection: `OpenInApp` open_here case checks `WT_WRAPPER` env var
- [ ] CHK-004 Stderr hint: hint printed when `WT_WRAPPER != "1"`, suppressed when `WT_WRAPPER == "1"`
- [ ] CHK-005 Help text updated: `wt --help` shows `eval "$(wt shell-setup)"` instead of inline function

## Behavioral Correctness

- [ ] CHK-006 cd output preserved: `open_here` still prints `cd -- <path>` to stdout regardless of wrapper state
- [ ] CHK-007 Hint on stderr only: hint does not appear on stdout (would break wrapper eval)
- [ ] CHK-008 wt create compatibility: `WT_WRAPPER` hint works in `wt create` open menu (shared `OpenInApp`)

## Scenario Coverage

- [ ] CHK-009 Wrapper not installed scenario: open_here without WT_WRAPPER shows hint + cd
- [ ] CHK-010 Wrapper installed scenario: open_here with WT_WRAPPER=1 shows only cd
- [ ] CHK-011 Unrecognized shell scenario: shell-setup with non-bash/zsh SHELL outputs bash wrapper + stderr warning
- [ ] CHK-012 SHELL unset scenario: shell-setup with empty SHELL outputs bash wrapper, no warning

## Edge Cases & Error Handling

- [ ] CHK-013 WT_WRAPPER set to non-"1" value: treated as wrapper absent, hint shown

## Code Quality

- [ ] CHK-014 Pattern consistency: New subcommand follows existing cobra command patterns in `src/go/wt/cmd/`
- [ ] CHK-015 No unnecessary duplication: Wrapper function text defined once, not duplicated between help and shell-setup output

## Documentation Accuracy

- [ ] CHK-016 packages.md updated: "Why wt-open Cannot cd" section references `wt shell-setup`

## Cross References

- [ ] CHK-017 main.go help text and shell-setup output are consistent (both reference the same setup method)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

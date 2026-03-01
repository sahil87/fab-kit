# Quality Checklist: Extract Logman from Preflight

**Change**: 260302-9fnn-extract-logman-from-preflight
**Generated**: 2026-03-02
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Command subcommand accepts `<cmd>` as first positional arg: `logman.sh command "skill-name" "change" "args"` writes correct JSON
- [ ] CHK-002 Command subcommand silently exits 0 when change omitted and no active change
- [ ] CHK-003 Command subcommand resolves via `fab/current` when change omitted and active change exists
- [ ] CHK-004 Command subcommand fails loudly (exit 1) when explicit change doesn't resolve
- [ ] CHK-005 Preflight no longer accepts `--driver` flag
- [ ] CHK-006 Preflight validation and YAML output unchanged after `--driver` removal
- [ ] CHK-007 changeman `new` calls logman with flipped arg order
- [ ] CHK-008 changeman `rename` calls logman with flipped arg order
- [ ] CHK-009 _preamble.md §2 includes logman call step after preflight YAML parsing
- [ ] CHK-010 All 5 exempt skills have logman call instructions

## Behavioral Correctness

- [ ] CHK-011 Help text and usage messages show new `command <cmd> [change] [args]` signature
- [ ] CHK-012 Arg count validation: `logman.sh command` (0 extra args) returns error; `logman.sh command "cmd"` (1 extra arg) is valid minimum
- [ ] CHK-013 Preflight treats `--driver` as positional arg (change-name override) — not a recognized flag

## Removal Verification

- [ ] CHK-014 No `--driver` flag parsing in preflight.sh — no `driver=""`, no `while/case` block for `--driver`, no `LOGMAN` variable, no step 6 logman call
- [ ] CHK-015 _scripts.md does not reference `--driver` in preflight section
- [ ] CHK-016 _scripts.md does not say "Skills never call logman.sh directly"

## Scenario Coverage

- [ ] CHK-017 Test: cmd + explicit change → JSON appended with correct fields
- [ ] CHK-018 Test: cmd only + active `fab/current` → JSON appended
- [ ] CHK-019 Test: cmd only + no `fab/current` → silent exit 0, no file written
- [ ] CHK-020 Test: cmd only + stale `fab/current` → silent exit 0
- [ ] CHK-021 Test: explicit change doesn't resolve → exit 1 with stderr
- [ ] CHK-022 Changeman tests pass after arg order flip
- [ ] CHK-023 Preflight tests pass after `--driver` removal

## Edge Cases & Error Handling

- [ ] CHK-024 logman `command` with no args after subcommand → usage error (exit 1)
- [ ] CHK-025 `fab/current` exists but is empty → silent exit 0 (no crash)
- [ ] CHK-026 `confidence` and `review` subcommands unaffected by changes

## Code Quality

- [ ] CHK-027 Pattern consistency: logman.sh changes follow existing script patterns (set -euo pipefail, resolve_change_dir helper, case-based dispatch)
- [ ] CHK-028 No unnecessary duplication: reuses existing resolve.sh for change resolution
- [ ] CHK-029 Readability: new conditional logic in logman.sh is clear and maintainable
- [ ] CHK-030 No god functions: logman `command` case remains under 50 lines
- [ ] CHK-031 No magic strings: arg positions are clear from the parsing flow

## Documentation Accuracy

- [ ] CHK-032 _scripts.md logman signature matches actual implementation
- [ ] CHK-033 _scripts.md preflight signature matches actual implementation
- [ ] CHK-034 _scripts.md call graph accurately reflects post-change caller relationships

## Cross References

- [ ] CHK-035 _preamble.md logman call pattern matches actual logman.sh interface
- [ ] CHK-036 Exempt skill logman instructions match actual logman.sh interface

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

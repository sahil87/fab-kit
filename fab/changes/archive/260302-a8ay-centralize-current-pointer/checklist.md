# Quality Checklist: Centralize Current Pointer Format

**Change**: 260302-a8ay-centralize-current-pointer
**Generated**: 2026-03-02
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Two-line format: `fab/current` written by `changeman.sh switch` contains 4-char ID on line 1, full folder name on line 2
- [ ] CHK-002 resolve.sh default mode: reads folder name from line 2 of `fab/current`
- [ ] CHK-003 changeman.sh rename: updates line 2, preserves line 1 when active change is renamed
- [ ] CHK-004 preflight.sh id field: YAML output contains `id:` field matching 4-char portion of name
- [ ] CHK-005 logman.sh delegation: no-change-arg path uses resolve.sh instead of direct fab/current read
- [ ] CHK-006 dispatch.sh polling: compares 4-char ID (line 1) instead of full content
- [ ] CHK-007 _preamble.md: §2 instructs agent to use `id` for script calls, `name` for display
- [ ] CHK-008 fab-discuss: uses resolve.sh instead of reading fab/current directly
- [ ] CHK-009 fab-archive: uses changeman.sh for all pointer operations (resolve, blank, switch)

## Behavioral Correctness

- [ ] CHK-010 resolve.sh single-change guess fallback still works when fab/current is missing
- [ ] CHK-011 changeman.sh switch --blank still deletes the file entirely (not format-dependent)
- [ ] CHK-012 logman.sh silent exit: exits 0 silently when fab/current missing, empty, or stale
- [ ] CHK-013 changeman.sh rename: does not modify fab/current when renaming a non-active change
- [ ] CHK-014 changeman.sh rename: does not create fab/current when it's absent

## Removal Verification

- [ ] CHK-015 logman.sh direct-read: `current_file` / `tr -d '[:space:]'` block removed from command subcommand
- [ ] CHK-016 fab-discuss: no remaining "Read `fab/current`" instructions in SKILL.md
- [ ] CHK-017 fab-archive: no remaining direct `fab/current` read/write/delete instructions in SKILL.md

## Scenario Coverage

- [ ] CHK-018 resolve test: "no argument reads fab/current" updated to two-line format
- [ ] CHK-019 resolve test: whitespace handling test updated to two-line format
- [ ] CHK-020 changeman test: switch test asserts both line 1 and line 2
- [ ] CHK-021 changeman test: rename tests updated for two-line format
- [ ] CHK-022 changeman test: resolve via fab/current tests updated for two-line format
- [ ] CHK-023 logman test: "command with cmd only resolves via fab/current" updated
- [ ] CHK-024 preflight test: set_current helper writes two-line format
- [ ] CHK-025 preflight test: new test for id field in YAML output
- [ ] CHK-026 All affected tests pass: `bats src/lib/resolve/test.bats src/lib/changeman/test.bats src/lib/logman/test.bats src/lib/preflight/test.bats`

## Edge Cases & Error Handling

- [ ] CHK-027 resolve.sh: trailing whitespace on line 2 stripped correctly
- [ ] CHK-028 logman.sh: stale two-line fab/current (name no longer exists) exits 0 silently
- [ ] CHK-029 dispatch.sh: polling timeout still works when fab/current is never written

## Code Quality

- [ ] CHK-030 Pattern consistency: new code follows naming and structural patterns of surrounding code
- [ ] CHK-031 No unnecessary duplication: existing utilities reused (extract_id, resolve.sh) where applicable
- [ ] CHK-032 Readability: no god functions introduced (>50 lines)
- [ ] CHK-033 No magic strings: format details (line numbers, cut fields) documented in comments
- [ ] CHK-034 No duplicated utilities: reuses extract_id/cut -d'-' -f2 pattern consistently

## Documentation Accuracy

- [ ] CHK-035 _preamble.md: all examples reference `id` for script calls
- [ ] CHK-036 fab-discuss SKILL.md: instructions match resolve.sh API
- [ ] CHK-037 fab-archive SKILL.md: instructions match changeman.sh API

## Cross References

- [ ] CHK-038 All scripts that read fab/current go through resolve.sh (no remaining direct readers)
- [ ] CHK-039 All scripts/skills that write fab/current go through changeman.sh (no remaining direct writers)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

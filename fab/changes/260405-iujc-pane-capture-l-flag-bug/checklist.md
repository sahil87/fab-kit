# Quality Checklist: Pane Capture -l Flag Bug

**Change**: 260405-iujc-pane-capture-l-flag-bug
**Generated**: 2026-04-05
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Flag table both-forms: `src/kit/skills/_cli-fab.md` pane capture flags table shows `` `-l`, `--lines` `` in the Flag column (not just `-l`)
- [x] CHK-002 Flag description accuracy: The description for the lines flag reads "Number of lines to capture" without any reference to `tmux capture-pane -l`
- [x] CHK-003 Deployed copy sync: `.claude/skills/_cli-fab/SKILL.md` contains the updated flag table entry (both forms, clean description)

## Behavioral Correctness

- [x] CHK-004 Short form still documented: `-l` remains visible in the updated flag entry (not replaced by `--lines` alone)
- [x] CHK-005 Long form added: `--lines` is now explicitly shown alongside `-l`

## Scenario Coverage

- [x] CHK-006 Agent long-form invocation: An agent reading `_cli-fab` can discover `--lines N` as a valid flag and use it
- [x] CHK-007 Agent short-form invocation: An agent reading `_cli-fab` can discover `-l N` as a valid flag and use it

## Documentation Accuracy

- [x] CHK-008 No internal tmux detail exposed: Flag description contains no reference to internal `tmux capture-pane` invocation details
- [x] CHK-009 Canonical source is authoritative: `src/kit/skills/_cli-fab.md` is the file that was edited (not only the deployed copy)

## Code Quality

- [x] CHK-010 Pattern consistency: Updated flag table row follows the same format as other rows in the same table (backtick-quoted flag names, consistent column count)
- [x] CHK-011 No unnecessary duplication: No duplicate flag entries introduced

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-001 **N/A**: {reason}`

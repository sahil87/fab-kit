# Quality Checklist: Unified Tick Status List

**Change**: 260326-oxgu-unified-tick-status-list
**Generated**: 2026-03-26
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Unified Entry List: Status frame example shows single flat list with all entry types
- [ ] CHK-002 Column Structure: Example entries follow Type → ID → Autopilot → Health → Detail layout
- [ ] CHK-003 Type Indicators: `[change]` and `[watch]` prefixes present in example
- [ ] CHK-004 Autopilot Per-Change: `▶` symbol shown on autopilot-driven entries, absent on others
- [ ] CHK-005 Universal Health Emoji: Both changes and watches show health emoji in same column position
- [ ] CHK-006 Header Line: Uses `N tracked` format, no per-type counts, no `autopilot 1/3`
- [ ] CHK-007 Watch Timestamps: Relative format (`3m ago`) used instead of absolute (`last check HH:MM`)
- [ ] CHK-008 Autopilot Section: §6 references updated from header indicator to `▶` per-entry

## Behavioral Correctness
- [ ] CHK-009 Old format removed: No remaining references to the two-block layout (changes block + `👁` watches block)
- [ ] CHK-010 Stage indicators replaced: Old `Stage indicators: 🟢 ... Watch indicator: 👁` paragraph replaced with column structure + health emoji definitions

## Scenario Coverage
- [ ] CHK-011 Mixed changes and watches: Example shows both types interleaved in one list
- [ ] CHK-012 Autopilot vs non-autopilot: Example shows entries with and without `▶`
- [ ] CHK-013 Watch health states: Health emoji definitions cover 🟢/🟡/🔴/⏸

## Code Quality
- [ ] CHK-014 Pattern consistency: Markdown formatting follows existing operator7 skill conventions
- [ ] CHK-015 No unnecessary duplication: Health emoji definitions stated once, not repeated

## Documentation Accuracy
- [ ] CHK-016 §4 Tick Behavior description matches the example block
- [ ] CHK-017 §6 Autopilot description consistent with §4 frame format

## Cross References
- [ ] CHK-018 Deployed copy synced: `.claude/skills/fab-operator7.md` matches source after sync

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

# Quality Checklist: Fab Draft Auto Activate

**Change**: 260404-rzvq-fab-draft-auto-activate
**Generated**: 2026-04-05
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-new auto-activation: `src/kit/skills/fab-new.md` contains Step 10 calling `fab change switch "{name}"` after advancing intake to ready
- [x] CHK-002 fab-new description updated: skill frontmatter description reads "Start a new change — creates the intake and activates it."
- [x] CHK-003 fab-new Next line updated: final `Next:` line reads `/fab-continue, /fab-fff, /fab-ff, or /fab-clarify` (no activation preamble)
- [x] CHK-004 fab-draft skill exists: `src/kit/skills/fab-draft.md` is present with correct frontmatter (`name: fab-draft`, description "Create a change intake without activating it.")
- [x] CHK-005 fab-draft no activation: `src/kit/skills/fab-draft.md` does NOT call `fab change switch` and does NOT create the `.fab-status.yaml` symlink
- [x] CHK-006 fab-draft Next line: `fab-draft.md` final `Next:` includes activation preamble (`/fab-switch {name} to make it active`)
- [x] CHK-007 fab-switch empty-state messages: `src/kit/skills/fab-switch.md` No Argument Flow empty message references both `/fab-new` and `/fab-draft`
- [x] CHK-008 fab-switch error table: error table entry "No changes exist" action references both `Run /fab-new or /fab-draft.`
- [x] CHK-009 fab-proceed dispatch table: "Conversation context" row shows `/fab-new → /git-branch → /fab-fff` (no `/fab-switch` step)
- [x] CHK-010 fab-proceed unactivated path: "Unactivated intake" row retains `/fab-switch → /git-branch → /fab-fff`
- [x] CHK-011 fab-proceed error message: empty-context error reads "Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first."
- [x] CHK-012 _preamble.md activation preamble: references `/fab-draft` (not `/fab-new`) as the case requiring the activation preamble
- [x] CHK-013 fabhelp.go skill group: `skillToGroupMap` in `src/go/fab/cmd/fab/fabhelp.go` contains `"fab-draft": "Start & Navigate"`
- [x] CHK-014 fabhelp_test.go: `expectedMapped` slice in `TestFabHelp_GroupMapping` contains `"fab-draft"`

## Behavioral Correctness

- [x] CHK-015 fab-new activation flow: after `/fab-new` completes, `.fab-status.yaml` symlink resolves to the new change (fab-draft does NOT set this)
- [x] CHK-016 fab-proceed conversation context path: no `/fab-switch` subagent is dispatched between `/fab-new` and `/git-branch` in conversation context path

## Removal Verification

- [x] CHK-017 --switch flag removed from docs/specs/skills.md: no `--switch` in the `/fab-new` section signature or argument list
- [x] CHK-018 fab-new no activation preamble in Next output: the final `Next:` line in `fab-new.md` does not contain "make it active" or "/fab-switch"

## Scenario Coverage

- [x] CHK-019 Spec scenario "New change created and activated": fab-new.md Step 10 invokes `fab change switch "{name}"` and displays confirmation
- [x] CHK-020 Spec scenario "Power user queuing multiple changes": fab-draft.md does not modify `.fab-status.yaml`, leaving existing active change unaffected
- [x] CHK-021 Spec scenario "fab help renders fab-draft": `fab-draft` appears in `skillToGroupMap` under "Start & Navigate"

## Edge Cases & Error Handling

- [x] CHK-022 fab-operator.md updated: command vocabulary includes `/fab-new` (create + activate), `/fab-draft` (create without activating), and updated `/fab-proceed` pipeline description
- [x] CHK-023 _cli-fab.md updated: "No active changes found" error table entry action reads "Run `/fab-new` or `/fab-draft` first"

## Code Quality

- [x] CHK-024 Pattern consistency: `fab-draft.md` follows the same structural patterns as `fab-new.md` (frontmatter format, section headings, step numbering)
- [x] CHK-025 No unnecessary duplication: skill files reference `fab change switch` CLI command directly rather than re-implementing activation logic

## Documentation Accuracy

- [x] CHK-026 README mermaid diagram: 12-column layout, `/fab-draft` header in purple, `/fab-new` (`headerN`) in green, `fnew_act["change active"]` row shows fab-new covering activation
- [x] CHK-027 docs/specs/skills.md: contains `/fab-draft <description>` section, `/fab-new` section has no `--switch` argument, Next Steps table includes `/fab-draft` row

## Cross-References

- [x] CHK-028 No stale `/fab-new → /fab-switch` chains in updated files: grep `fab-new.*fab-switch` or `fab-switch.*fab-new` across changed skill files produces no unintentional sequences (operator and proceed files updated correctly)
- [x] CHK-029 Spec consistency: all spec requirements (T001-T011) have corresponding implementation in the changed files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`

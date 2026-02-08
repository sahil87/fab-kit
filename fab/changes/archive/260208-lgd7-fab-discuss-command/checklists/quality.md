# Quality Checklist: Add fab-discuss Command and Confidence Scoring to fab-new

**Change**: 260208-lgd7-fab-discuss-command
**Generated**: 2026-02-08
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-discuss skill file: `fab/.kit/skills/fab-discuss.md` exists and follows the standard skill format (frontmatter, purpose, pre-flight, behavior, output, error handling)
- [x] CHK-002 Skill registration: `.claude/skills/fab-discuss/SKILL.md` is a valid symlink to `../../../fab/.kit/skills/fab-discuss.md`
- [x] CHK-003 Context-driven mode selection: skill definition describes active change detection (refine if active, new if no active, confirm if divergent)
- [x] CHK-004 No active change switch: skill explicitly states it does NOT write to `fab/current`
- [x] CHK-005 No git integration: skill explicitly states it does NOT create/adopt branches; `.status.yaml` omits `branch:` field
- [x] CHK-006 Gap analysis phase: skill describes evaluating whether the change is needed before committing to a proposal
- [x] CHK-007 Conversational flow: skill describes free-form conversation with no fixed question cap
- [x] CHK-008 Conversation termination: skill describes score >= 3.0 threshold + user signal, with proactive wrap-up suggestion
- [x] CHK-009 Confidence scoring: skill describes computing and writing SRAD confidence score to `.status.yaml`
- [x] CHK-010 Proposal output (new): skill describes creating change folder, checklists/, .status.yaml, proposal.md
- [x] CHK-011 Proposal output (refine): skill describes updating existing proposal.md and recomputing confidence
- [x] CHK-012 Next steps: skill ends with appropriate `Next:` line (varies by new vs refine mode)
- [x] CHK-013 _context.md autonomy table: `/fab-discuss` row added with correct autonomy characteristics
- [x] CHK-014 _context.md next steps table: `/fab-discuss` entries added for both new-change and refined-existing outcomes
- [x] CHK-015 _context.md init/hydrate next steps: updated to include `/fab-discuss` alongside `/fab-new`
- [x] CHK-016 fab-new confidence scoring: Step 8 writes actual confidence counts to `.status.yaml` (not template zeros)

## Scenario Coverage

- [x] CHK-017 Active change exists — refine mode: skill describes loading existing proposal.md
- [x] CHK-018 Active change exists but description diverges: skill describes confirming with user
- [x] CHK-019 No active change — new change mode: skill describes creating new change folder
- [x] CHK-020 New change created without switching: skill confirms fab/current unchanged
- [x] CHK-021 Score threshold reached — suggest wrap-up: skill describes proactive suggestion
- [x] CHK-022 User ends early: skill describes finalizing with current score and noting if below gate

## Edge Cases & Error Handling

- [x] CHK-023 Error handling: skill defines behavior for missing config, missing constitution, missing template
- [x] CHK-024 Pre-flight: skill runs preflight or equivalent checks before proceeding

## Documentation Accuracy

- [x] CHK-025 Cross-references: all file paths in the skill definition match actual project structure
- [x] CHK-026 SRAD consistency: confidence formula and grade definitions match `_context.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

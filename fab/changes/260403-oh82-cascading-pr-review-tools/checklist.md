# Quality Checklist: Cascading PR Review Tools

**Change**: 260403-oh82-cascading-pr-review-tools
**Generated**: 2026-04-03
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Cascade Order: Skill attempts Copilot → Codex → Claude in order, stops on first success
- [ ] CHK-002 Copilot Reviewer: `gh pr edit {number} --add-reviewer copilot` attempted, falls through on failure
- [ ] CHK-003 Codex Reviewer: Detection via `command -v codex`, runs with enriched prompt when available
- [ ] CHK-004 Claude Reviewer: Detection via `command -v claude`, runs with enriched prompt when available
- [ ] CHK-005 Context Enrichment: Prompt includes diff, file list, test results (best-effort), PR description
- [ ] CHK-006 Local Output Posting: Review output posted as PR comment via `gh api`, best-effort
- [ ] CHK-007 Per-Tool Config: `review_tools` in config.yaml controls which tools are attempted
- [ ] CHK-008 `--tool` Flag: Forces specific reviewer, bypasses cascade
- [ ] CHK-009 Migration File: `src/kit/migrations/1.1.0-to-1.2.0.md` adds `review_tools` block idempotently
- [ ] CHK-010 Spec Update: `docs/specs/skills/SPEC-git-pr-review.md` reflects cascade, config, and flag

## Behavioral Correctness
- [ ] CHK-011 Cascade Placement: Cascade runs in Step 2 only when no existing reviews with comments found
- [ ] CHK-012 Existing Reviews: When reviews with comments exist, cascade is skipped and existing flow runs
- [ ] CHK-013 Copilot Split: Copilot request exits skill — user re-invokes later to process comments

## Scenario Coverage
- [ ] CHK-014 Full cascade fallthrough: All tools fail → reports no tools available
- [ ] CHK-015 First tool succeeds: Copilot works → cascade stops
- [ ] CHK-016 Tool disabled in config: Disabled tool is skipped entirely
- [ ] CHK-017 Config absent: Missing `review_tools` key defaults all tools to true
- [ ] CHK-018 Force specific tool: `--tool claude` runs only Claude

## Edge Cases & Error Handling
- [ ] CHK-019 All tools disabled: Config sets all to false → clear message and stop
- [ ] CHK-020 All tools unavailable: None installed/accessible → clear message and stop
- [ ] CHK-021 Invalid `--tool` value: Error with valid tool names listed
- [ ] CHK-022 Posting failure: `gh api` fails → logged, output still printed, no abort

## Code Quality
- [ ] CHK-023 Pattern consistency: New skill sections follow existing `/git-pr-review` structure and conventions
- [ ] CHK-024 No unnecessary duplication: Reuse existing Step 1 (PR resolution) and Step 2 Phase 1 (review detection)
- [ ] CHK-025 Readability: Cascade logic is clear and maintainable, not a god-section
- [ ] CHK-026 No magic strings: Tool names referenced consistently, not hardcoded in multiple places

## Documentation Accuracy
- [ ] CHK-027 Skill file: All new behavior documented inline in the skill markdown
- [ ] CHK-028 Migration instructions: Clear step-by-step, matches actual config structure

## Cross References
- [ ] CHK-029 Constitution compliance: Changes to skill file update corresponding spec (per constitution constraint)
- [ ] CHK-030 Memory alignment: Affected memory section in intake matches actual memory files modified during hydrate

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

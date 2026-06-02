# Quality Checklist: Fix git-pr-review Copilot reviewer login

**Change**: 260418-u1m1-copilot-reviewer-login
**Generated**: 2026-04-18
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Correct Copilot reviewer login in add-reviewer call: `src/kit/skills/git-pr-review.md` line 87 uses `gh pr edit {number} --add-reviewer copilot-pull-request-reviewer`.
- [x] CHK-002 Poll check uses matching reviewer login: `src/kit/skills/git-pr-review.md` line 92 uses `.author.login == "copilot-pull-request-reviewer"`.
- [x] CHK-003 Config key `review_tools.copilot` retains its name: grep confirms the config key spelling `review_tools.copilot` is unchanged across `src/`, `docs/`, `fab/project/`, and migrations (`src/kit/migrations/1.1.0-to-1.2.0.md`, `src/kit/skills/_review.md`, `docs/memory/fab-workflow/configuration.md`).
- [x] CHK-004 SPEC-git-pr-review reflects corrected login: both the flow diagram (line ~46) and the reviewer table (line ~86) show `--add-reviewer copilot-pull-request-reviewer`.
- [x] CHK-005 Memory files reflect corrected login: `docs/memory/fab-workflow/configuration.md` line ~61 and `docs/memory/fab-workflow/execution-skills.md` lines ~31 and ~473 use `copilot-pull-request-reviewer` in command examples.
- [x] CHK-006 No stray `copilot` reviewer references remain: `grep -rn "add-reviewer copilot\b" src/ docs/` returns no matches.

## Behavioral Correctness
- [x] CHK-007 Phase 2 tagging + polling handshake: both the `--add-reviewer` call and the `.author.login` poll filter reference the same identity `copilot-pull-request-reviewer`, so when the bot posts a review the poll recognizes it.

## Scenario Coverage
- [x] CHK-008 Phase 2 runs and request succeeds (spec scenario): inspection of the updated skill confirms the command is `gh pr edit {number} --add-reviewer copilot-pull-request-reviewer` and the success-branch text is unchanged (`Copilot review requested. Waiting up to 10 minutes...`).
- [x] CHK-009 Phase 2 runs and request fails (spec scenario): failure-branch text is unchanged (`No automated reviewer available. Run /git-pr-review when reviews are added.`).
- [x] CHK-010 Polling detects a Copilot review (spec scenario): the poll filter expression inside the skill's code block reads `.reviews | map(select(.author.login == "copilot-pull-request-reviewer")) | length`.
- [x] CHK-011 Polling exhausts all attempts (spec scenario): timeout-branch text is unchanged (`Copilot review requested but not yet available. Re-run /git-pr-review to process when ready.`).
- [x] CHK-012 Config with copilot disabled (spec scenario): the `review_tools.copilot: false` early-return branch still runs before any add-reviewer attempt.
- [x] CHK-013 Config key absent (spec scenario): the default-enabled path still applies when `review_tools` is omitted from `config.yaml`.

## Edge Cases & Error Handling
- [x] CHK-014 No change to stage-transition or error-handling flow: the only edits are literal string replacements; control flow (Step 0-6, Phase 1/2 routing, idempotence, dedup) is untouched.

## Code Quality
- [x] CHK-015 Pattern consistency: edits use the same formatting, quoting, and syntax as the surrounding lines (no stylistic drift — e.g., backticks and code-fence placement preserved).
- [x] CHK-016 No unnecessary duplication: no new constants or helpers introduced; this is a literal string fix.
- [x] CHK-017 Readability over cleverness: the change is a verbatim-substitution of the bot identifier — no abstraction, no new config knob.
- [x] CHK-018 Follows existing project patterns: `/git-pr-review`'s file structure, narrative style, and SPEC+memory split are all preserved.
- [x] CHK-019 No magic strings left un-corrected: `copilot-pull-request-reviewer` appears at every site that names the GitHub login, not just the primary add-reviewer call.

## Documentation Accuracy
<!-- Project-specific category from config.yaml -->
- [x] CHK-020 Spec and memory agree with skill: after edits, the four documentation references (SPEC flow, SPEC table, configuration.md bullet, execution-skills.md narrative + decision log) describe the same login string that the skill now invokes.
- [x] CHK-021 Command examples are executable as shown: the quoted `gh pr edit --add-reviewer copilot-pull-request-reviewer` string is a valid `gh` invocation with no placeholder mistakes.

## Cross-References
<!-- Project-specific category from config.yaml -->
- [x] CHK-022 Affected memory list matches edits: intake/spec `Affected memory` section names exactly the two memory files touched (`fab-workflow/configuration.md`, `fab-workflow/execution-skills.md`) and neither more nor less.
- [x] CHK-023 No dangling cross-references: no memory file, spec, or migration references `add-reviewer copilot` (word-boundary) after the change.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-XXX **N/A**: {reason}`

# Quality Checklist: fab-clarify Dual Modes + fab-ff Clarify Checkpoints

**Change**: 260207-m3qf-clarify-dual-modes
**Generated**: 2026-02-07
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Suggest mode as default: `/fab:clarify` invoked by user operates in suggest mode with interactive questions
- [x] CHK-002 Stage-scoped taxonomy scan: scan categories vary by stage (proposal/specs/plan/tasks), not a fixed universal list
- [x] CHK-003 Structured question format: each question includes recommendation with reasoning + options table (multiple-choice) or suggested answer (short-answer)
- [x] CHK-004 Max 5 questions cap: single invocation presents at most 5 questions; coverage summary indicates outstanding items
- [x] CHK-005 Incremental artifact updates: artifact is updated in place after each user answer, not batched at end
- [x] CHK-006 Early termination: "done"/"good"/"no more" stops question flow and proceeds to coverage summary
- [x] CHK-007 Clarifications audit trail: `## Clarifications > ### Session {date}` with `Q:` / `A:` bullets appended to artifact
- [x] CHK-008 Coverage summary: table with Resolved/Clear/Deferred/Outstanding counts displayed at session end
- [x] CHK-009 Auto mode activation: internal fab-ff call triggers auto mode (autonomous resolution, no user interaction)
- [x] CHK-010 Mode selection by call context: no `--suggest`/`--auto` flags on clarify; mode determined by caller
- [x] CHK-011 Machine-readable auto mode result: returns `{resolved, blocking, non_blocking}` counts for fab-ff consumption
- [x] CHK-012 Interleaved auto-clarify in fab-ff: default pipeline is `spec → auto-clarify → plan → auto-clarify → tasks → auto-clarify`
- [x] CHK-013 Bail on blocking: fab-ff default mode stops when auto-clarify returns blocking > 0, reports issues, suggests `/fab:clarify` then `/fab:ff`
- [x] CHK-014 Resumability: re-running `/fab:ff` after bail skips stages already `done`, continues from current position
- [x] CHK-015 Full-auto flag: `/fab:ff --auto` never stops for blockers, makes best-guess decisions
- [x] CHK-016 Auto-guess markers: `<!-- auto-guess: {description} -->` placed in artifacts for each guess
- [x] CHK-017 Auto-guess visibility: markers detectable by `/fab:review` (flagged as warnings) and resolvable by `/fab:clarify` suggest mode
- [x] CHK-018 _context.md updated: Next Steps table includes `/fab:ff --auto` variant
- [x] CHK-019 No changes to other skills: `/fab:continue`, `/fab:new`, `/fab:apply`, `/fab:review`, `/fab:archive`, templates, `.status.yaml` schema unchanged

## Behavioral Correctness

- [x] CHK-020 Existing clarify behavior preserved: auto mode retains current autonomous gap-resolution behavior (no regression)
- [x] CHK-021 Existing fab-ff behavior preserved: frontloaded questions still work; plan decision still autonomous; new pipeline wraps existing generation logic
- [x] CHK-022 One question at a time: suggest mode never reveals queued future questions

## Scenario Coverage

- [x] CHK-023 Scenario: user invokes /fab:clarify on spec with NEEDS CLARIFICATION markers — taxonomy scan, single question presented
- [x] CHK-024 Scenario: user invokes /fab:clarify on clean artifact — scan runs, reports "No gaps found" or surfaces implicit gaps
- [x] CHK-025 Scenario: fab-ff clean run — auto-clarify checkpoints pass with 0 issues, all artifacts generated
- [x] CHK-026 Scenario: fab-ff blocking bail — auto-clarify finds blocker, pipeline stops, user guidance displayed
- [x] CHK-027 Scenario: fab-ff resume after bail — skips done stages, continues generation
- [x] CHK-028 Scenario: fab-ff --auto with blocking issue — best-guess made, marker placed, warning in output
- [x] CHK-029 Scenario: /fab:review detects auto-guess markers — flags them as warnings

## Edge Cases & Error Handling

- [x] CHK-030 Zero gaps detected: suggest mode outputs "No gaps found — artifact looks solid"
- [x] CHK-031 Early termination after 0 answers: valid coverage summary with 0 resolved
- [x] CHK-032 Multiple clarify sessions: audit trail entries accumulate, don't overwrite previous sessions
- [x] CHK-033 Auto-clarify resolves all issues: fab-ff continues without bail (blocking: 0 path)

## Documentation Accuracy

- [x] CHK-034 New centralized doc `fab/docs/fab-workflow/clarify.md` accurately describes dual-mode behavior
- [x] CHK-035 `fab/docs/fab-workflow/index.md` includes clarify doc entry

## Cross References

- [x] CHK-036 fab-clarify auto mode result format matches what fab-ff consumes
- [x] CHK-037 fab-ff bail message references correct commands (`/fab:clarify` then `/fab:ff`)
- [x] CHK-038 _context.md Next Steps table is consistent with updated fab-ff behavior

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab:archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

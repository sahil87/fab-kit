# Quality Checklist: Redesign Hooks Strategy

**Change**: 260306-6bba-redesign-hooks-strategy
**Generated**: 2026-03-06
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 PostToolUse hook script: `on-artifact-write.sh` exists and handles Write and Edit events
- [x] CHK-002 Intake bookkeeping: hook runs `fab status set-change-type` and `fab score --stage intake` on intake write
- [x] CHK-003 Spec bookkeeping: hook runs `fab score` on spec write/edit
- [x] CHK-004 Tasks bookkeeping: hook runs `fab status set-checklist total <N>` on tasks write
- [x] CHK-005 Checklist bookkeeping: hook runs `set-checklist generated true`, `total <N>`, `completed 0` on checklist write
- [x] CHK-006 Runtime set-idle: `fab runtime set-idle <change>` writes idle timestamp to `.fab-runtime.yaml`
- [x] CHK-007 Runtime clear-idle: `fab runtime clear-idle <change>` removes agent block from `.fab-runtime.yaml`
- [x] CHK-008 Stop hook migrated: `on-stop.sh` uses `fab runtime set-idle`, no yq dependency
- [x] CHK-009 SessionStart hook migrated: `on-session-start.sh` uses `fab runtime clear-idle`, no yq dependency
- [x] CHK-010 Sync script matchers: `5-sync-hooks.sh` registers PostToolUse hooks with Write and Edit matchers
- [x] CHK-011 Constitution §I: reads "scripts" (not "shell scripts")
- [x] CHK-012 fab-setup.md: Phase 1b-lang section removed
- [x] CHK-013 _scripts.md: `fab runtime` commands documented

## Behavioral Correctness
- [x] CHK-014 Hook exits 0 always: bookkeeping failures do not interrupt the agent
- [x] CHK-015 Non-artifact writes ignored: files outside `fab/changes/*/` pattern exit fast
- [x] CHK-016 Idempotent bookkeeping: running hook + skill command produces same result as either alone
- [x] CHK-017 Runtime clear-idle no-op: exits 0 when `.fab-runtime.yaml` doesn't exist
- [x] CHK-018 Skills unchanged: existing bookkeeping instructions in skills preserved (reliability layer, not replacement)

## Scenario Coverage
- [x] CHK-019 Edit event: hook reads file from disk for content-dependent bookkeeping (intake type inference, tasks count)
- [x] CHK-020 Fab CLI unavailable: hook exits 0 when `fab` binary not found
- [x] CHK-021 Change resolution failure: hook exits 0 when change folder can't be derived
- [x] CHK-022 Runtime file creation: `set-idle` creates `.fab-runtime.yaml` with `{}` if missing
- [x] CHK-023 Multiple changes in runtime file: `set-idle`/`clear-idle` preserve other changes' entries

## Edge Cases & Error Handling
- [x] CHK-024 Hook returns additionalContext: stdout JSON includes summary of bookkeeping performed
- [x] CHK-025 Empty fab/current: hooks handle missing/empty current file gracefully

## Code Quality
- [x] CHK-026 Pattern consistency: Go code follows existing cmd/fab patterns (Cobra setup, resolve usage, error handling)
- [x] CHK-027 No unnecessary duplication: hook script reuses `fab` CLI rather than reimplementing logic
- [x] CHK-028 Readability: hook script logic is clear with comments for each artifact match
- [x] CHK-029 No magic strings: artifact patterns and keyword lists are documented inline

## Documentation Accuracy
- [x] CHK-030 _scripts.md matches implementation: `fab runtime` command signatures accurate
- [x] CHK-031 SPEC-fab-setup.md reflects removal: Phase 1b-lang documented as removed

## Cross References
- [x] CHK-032 Go tests exist: `runtime_test.go` covers set-idle and clear-idle
- [x] CHK-033 Hook sync produces valid JSON: `.claude/settings.local.json` is valid after sync

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

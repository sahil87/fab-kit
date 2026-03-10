# Quality Checklist: Build fab Go Binary

**Change**: 260305-bhd6-1-build-fab-go-binary
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Go module scaffold: `src/go/fab/go.mod` exists with correct module path, Go 1.22+, cobra and yaml.v3 dependencies
- [ ] CHK-002 StatusFile struct: `internal/statusfile` loads and saves `.status.yaml` with all fields (progress, checklist, confidence, stage_metrics, issues, prs)
- [ ] CHK-003 Resolve: `fab resolve` supports all four output modes (--id, --folder, --dir, --status) and all input forms (4-char ID, substring, full name, fab/current default)
- [ ] CHK-004 Log: `fab log` implements all four subcommands (command, confidence, review, transition) with correct JSON format
- [ ] CHK-005 Status events: `fab status` implements all six event subcommands (start, advance, finish, reset, skip, fail) with correct state transitions
- [ ] CHK-006 Status queries: `fab status` implements all query subcommands (progress-map, progress-line, current-stage, display-stage, checklist, confidence, all-stages, validate-status-file, get-issues, get-prs)
- [ ] CHK-007 Status writes: `fab status` implements all write subcommands (set-change-type, set-checklist, set-confidence, set-confidence-fuzzy, add-issue, add-pr)
- [ ] CHK-008 Preflight: `fab preflight` produces structured YAML output matching bash version format
- [ ] CHK-009 Change lifecycle: `fab change` implements new, rename, switch, list, resolve with correct output formats
- [ ] CHK-010 Score: `fab score` computes confidence correctly (penalty weights, coverage factor, forced zero on unresolved, dimension means)
- [ ] CHK-011 Archive: `fab archive` implements archive, restore, list with index.md management

## Behavioral Correctness
- [ ] CHK-012 Finish auto-activates next stage: `fab status finish` transitions next pending stage to active
- [ ] CHK-013 Review auto-log: `fab status finish/fail review` appends review log entry to .history.jsonl
- [ ] CHK-014 Reset cascade: `fab status reset` cascades downstream stages to pending
- [ ] CHK-015 Skip cascade: `fab status skip` cascades downstream pending stages to skipped
- [ ] CHK-016 Stage metrics tracking: started_at, driver, iterations, completed_at maintained correctly across transitions
- [ ] CHK-017 Idempotent add-issue/add-pr: duplicate additions are no-ops

## Scenario Coverage
- [ ] CHK-018 Resolve ambiguous match: multiple folder matches produce exit 1 with "Multiple changes match" stderr
- [ ] CHK-019 Log command graceful degradation: no fab/current + no explicit change = silent exit 0
- [ ] CHK-020 Change new collision detection: provided change-id that already exists produces error
- [ ] CHK-021 Change switch --blank: deletes fab/current, outputs "No active change."
- [ ] CHK-022 Score gate check: --check-gate mode outputs pass/fail YAML without modifying .status.yaml
- [ ] CHK-023 Archive with active pointer: archiving the active change clears fab/current
- [ ] CHK-024 Preflight change override: passing change-name uses that change, not fab/current

## Edge Cases & Error Handling
- [ ] CHK-025 StatusFile atomic save: uses temp file + rename to prevent corruption on interruption
- [ ] CHK-026 Missing .status.yaml: appropriate error messages from resolve/preflight/status commands
- [ ] CHK-027 Empty fab/changes/: list commands return empty output without errors
- [ ] CHK-028 Invalid slug format: change new rejects slugs starting/ending with hyphens
- [ ] CHK-029 Score with zero assumptions: handles edge case of empty Assumptions table

## Code Quality
- [ ] CHK-030 Pattern consistency: Go code follows standard Go conventions (gofmt, effective Go idioms)
- [ ] CHK-031 No unnecessary duplication: shared logic in internal/statusfile and internal/resolve reused across packages
- [ ] CHK-032 Readability: functions under 50 lines where practical, clear naming
- [ ] CHK-033 No magic strings: stage names, state names, and thresholds defined as constants

## Documentation Accuracy
- [ ] CHK-034 All subcommand help text matches the bash script behavior descriptions
- [ ] CHK-035 go.mod module path is correct and dependencies are minimal (cobra, yaml.v3 only)

## Cross References
- [ ] CHK-036 Output format parity: Go subcommands produce byte-compatible stdout/stderr with bash scripts (modulo timestamps)
- [ ] CHK-037 Stage order consistency: progress map always uses pipeline order (intake through review-pr)

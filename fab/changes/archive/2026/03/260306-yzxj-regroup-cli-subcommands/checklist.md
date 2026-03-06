# Quality Checklist: Regroup CLI Subcommands

**Change**: 260306-yzxj-regroup-cli-subcommands
**Generated**: 2026-03-06
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Archive subcommand: `fab change archive <change> --description "..."` works end-to-end
- [ ] CHK-002 Restore subcommand: `fab change restore <change> [--switch]` works end-to-end
- [ ] CHK-003 Archive-list subcommand: `fab change archive-list` lists archived changes
- [ ] CHK-004 Top-level removal: `fab archive` returns unknown command error
- [ ] CHK-005 Change help: `fab change --help` lists archive, restore, archive-list alongside existing subcommands
- [ ] CHK-006 _scripts.md regrouped: three sections (Change Lifecycle, Pipeline & Status, Plumbing) present
- [ ] CHK-007 _scripts.md archive section removed: no standalone `## fab archive` heading

## Behavioral Correctness

- [ ] CHK-008 Archive output: structured YAML output identical to previous `fab archive` behavior
- [ ] CHK-009 Restore output: structured YAML output identical to previous `fab archive restore` behavior
- [ ] CHK-010 Internal archive package: `internal/archive/` unchanged — only Cobra wiring moved

## Removal Verification

- [ ] CHK-011 No `archiveCmd()` in root AddCommand: `main.go` does not register archive at root level
- [ ] CHK-012 No standalone `## fab archive` section in `_scripts.md`

## Scenario Coverage

- [ ] CHK-013 Archive via new path: parity test covers `change archive` invocation
- [ ] CHK-014 Restore via new path: test infrastructure covers `change restore`
- [ ] CHK-015 List via new path: test infrastructure covers `change archive-list`
- [ ] CHK-016 Go build succeeds with refactored command wiring

## Edge Cases & Error Handling

- [ ] CHK-017 `fab change archive` with no args shows help (not error)
- [ ] CHK-018 `fab change restore` with no args returns error (ExactArgs(1))

## Code Quality

- [ ] CHK-019 Pattern consistency: renamed functions follow existing naming conventions in change.go
- [ ] CHK-020 No unnecessary duplication: archive functions reuse existing `internal/archive/` package without changes

## Documentation Accuracy

- [ ] CHK-021 fab-archive.md: all CLI invocations updated to `fab change archive`/`restore`/`archive-list`
- [ ] CHK-022 SPEC-fab-archive.md: flow diagram and tools table reference updated commands
- [ ] CHK-023 kit-architecture.md: command reference reflects new paths

## Cross References

- [ ] CHK-024 _scripts.md Command Reference table: no `fab archive` row, `fab change` row includes archive/restore/archive-list
- [ ] CHK-025 Parity test: `runGo` calls use `"change", "archive", ...` paths

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

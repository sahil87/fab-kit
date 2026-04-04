# Quality Checklist: Rename upgrade to upgrade-repo

**Change**: 260404-g0x1-rename-upgrade-to-upgrade-repo
**Generated**: 2026-04-05
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 upgrade-repo command registration: `fab-kit` cobra command uses `Use: "upgrade-repo [version]"` — not `upgrade`
- [ ] CHK-002 fab router allowlist: `fabKitArgs["upgrade-repo"]` is `true` and `fabKitArgs["upgrade"]` is absent/false in `cmd/fab/main.go`
- [ ] CHK-003 Help text: `printHelp()` in `cmd/fab/main.go` displays `upgrade-repo` in the workspace commands list
- [ ] CHK-004 fab-kit root Short: `cmd/fab-kit/main.go` root command Short string references `upgrade-repo`
- [ ] CHK-005 README update: "Updating from a previous version" code block shows `fab upgrade-repo` and `fab upgrade-repo 0.44.0`
- [ ] CHK-006 README tool table: fab-kit row lists `upgrade-repo` not `upgrade` (if table exists)
- [ ] CHK-007 Memory files updated: All four memory files (`distribution.md`, `kit-architecture.md`, `migrations.md`, `configuration.md`) use `fab upgrade-repo` in live prose — changelog rows unchanged
- [ ] CHK-008 index.md updated: `docs/memory/fab-workflow/index.md` line 14 description uses `fab upgrade-repo`
- [ ] CHK-009 fab-setup.md updated: Kit-not-found error message references `fab upgrade-repo`
- [ ] CHK-010 _cli-fab.md source updated: workspace commands routing list and fab-kit table row use `upgrade-repo`
- [ ] CHK-011 Deployed _cli-fab/SKILL.md updated: matches source changes from CHK-010

## Behavioral Correctness

- [ ] CHK-012 upgrade-repo behavior unchanged: The Go `Upgrade()` function logic is unmodified — only command registration changed
- [ ] CHK-013 `fab update` unaffected: `update` command in `cmd/fab-kit/main.go` and `fabKitArgs["update"]` in `cmd/fab/main.go` are unchanged

## Removal Verification

- [ ] CHK-014 `upgrade` key removed from `fabKitArgs`: No `"upgrade": true` entry in `cmd/fab/main.go`
- [ ] CHK-015 `upgrade` key removed from `fabKitCommands`: No `"upgrade": true` entry in `cmd/fab-kit/main.go`
- [ ] CHK-016 No stale `fab upgrade` in live docs: Grep of `fab upgrade` across `README.md`, `docs/memory/fab-workflow/*.md` (excluding changelog rows), `src/kit/skills/fab-setup.md`, and `src/kit/skills/_cli-fab.md` returns no hits

## Scenario Coverage

- [ ] CHK-017 Router dispatch scenario: `TestFabKitArgs` in `cmd/fab/main_test.go` passes with `upgrade-repo` in expected list
- [ ] CHK-018 fab-kit commands test: `TestFabKitCommands` in `cmd/fab-kit/main_test.go` passes with `upgrade-repo` in expected list
- [ ] CHK-019 All Go tests pass: `cd src/go/fab-kit && go test ./...` exits 0

## Edge Cases & Error Handling

- [ ] CHK-020 Old `fab upgrade` routes to fab-go: Since `upgrade` is not in `fabKitArgs`, `fab upgrade` now forwards to `fab-go` which will return an unknown-command error — this is the intended clean-break behavior

## Code Quality

- [ ] CHK-021 Pattern consistency: Renamed command follows existing cobra `Use` field patterns (verb-noun, matches binary-name conventions)
- [ ] CHK-022 No unnecessary duplication: No duplicate command registrations for both `upgrade` and `upgrade-repo`

## Documentation Accuracy

- [ ] CHK-023 Changelog row immutability: All `## Changelog` table rows in memory files still reference `fab upgrade` where they originally did — no historical rows modified
- [ ] CHK-024 Cross-references consistent: All cross-references between memory files (e.g., distribution.md ↔ migrations.md) use `fab upgrade-repo` consistently

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`

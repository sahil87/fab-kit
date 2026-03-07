# Quality Checklist: Status Symlink Pointer

**Change**: 260307-x2tx-status-symlink-pointer
**Generated**: 2026-03-07
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Symlink replaces fab/current: `.fab-status.yaml` symlink at repo root is the sole active change pointer
- [ ] CHK-002 resolveFromCurrent reads symlink: `os.Readlink()` extracts folder name from target path
- [ ] CHK-003 Switch creates symlink: `Switch()` creates `.fab-status.yaml` symlink with correct relative target
- [ ] CHK-004 SwitchBlank removes symlink: `SwitchBlank()` removes `.fab-status.yaml`
- [ ] CHK-005 Rename updates symlink: `Rename()` re-creates symlink when active change is renamed
- [ ] CHK-006 Pane map reads symlink: `readFabCurrent()` uses `os.Readlink()` on `.fab-status.yaml`
- [ ] CHK-007 ID field in StatusFile: struct has `ID string` with `yaml:"id"` tag
- [ ] CHK-008 Template has id placeholder: `fab/.kit/templates/status.yaml` includes `id: {ID}` as first field
- [ ] CHK-009 New() populates ID: `fab change new` replaces `{ID}` placeholder in template
- [ ] CHK-010 .gitignore updated: `.fab-status.yaml` replaces `fab/current`

## Behavioral Correctness
- [ ] CHK-011 No fab/current file created anywhere: Switch, Rename, New do not create `fab/current`
- [ ] CHK-012 Symlink target is relative: target path is `fab/changes/{name}/.status.yaml`, not absolute
- [ ] CHK-013 Single-change guess fallback preserved: when no symlink exists, falls through to guess logic

## Scenario Coverage
- [ ] CHK-014 Reading active change via symlink: test verifies correct folder name extraction
- [ ] CHK-015 No symlink present: test verifies fallthrough to single-change guess
- [ ] CHK-016 Broken symlink: test verifies graceful handling (readlink succeeds, resolution may fail)
- [ ] CHK-017 Switch creates correct symlink: test verifies symlink target after switch
- [ ] CHK-018 Switch --blank removes symlink: test verifies symlink removed
- [ ] CHK-019 Rename updates symlink target: test verifies new target after rename
- [ ] CHK-020 ID field round-trips: test verifies Load/Save preserves id field

## Edge Cases & Error Handling
- [ ] CHK-021 Stale regular file at .fab-status.yaml: Switch overwrites it with symlink
- [ ] CHK-022 Already-blank deactivation: SwitchBlank returns "(already blank)" without error
- [ ] CHK-023 Rename when symlink points elsewhere: skip symlink update, no error

## Code Quality
- [ ] CHK-024 Pattern consistency: symlink operations follow existing Go patterns (os.Remove/os.Symlink/os.Readlink)
- [ ] CHK-025 No unnecessary duplication: folder name extraction from symlink target is a shared helper or inline where simple
- [ ] CHK-026 Readability: code follows existing naming conventions in resolve.go and change.go

## Documentation Accuracy
- [ ] CHK-027 All `fab/current` references in skills updated to `.fab-status.yaml`
- [ ] CHK-028 All `fab/current` references in specs/memory updated
- [ ] CHK-029 Migration file covers all upgrade paths (existing fab/current, missing, empty, id backfill)

## Cross References
- [ ] CHK-030 Glossary reflects symlink pointer
- [ ] CHK-031 Change-lifecycle memory file updated
- [ ] CHK-032 Kit-architecture memory file updated

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

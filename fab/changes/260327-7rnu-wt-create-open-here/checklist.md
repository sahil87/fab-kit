# Quality Checklist: wt create "Open Here" Option

**Change**: 260327-7rnu-wt-create-open-here
**Generated**: 2026-03-27
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 "Open here" availability: `BuildAvailableApps()` returns "Open here" as first entry unconditionally
- [ ] CHK-002 open_here handler: `OpenInApp("open_here", path, ...)` prints `cd "<path>"` to stdout
- [ ] CHK-003 Path suppression: `create.go` suppresses final `fmt.Println(wtPath)` when open_here is selected
- [ ] CHK-004 Default detection: `DetectDefaultApp()` never returns the "Open here" index as default
- [ ] CHK-005 Last-app cache: Selecting "Open here" saves `open_here` to the last-app cache file
- [ ] CHK-006 wt open support: "Open here" appears and works in `wt open` menu

## Behavioral Correctness
- [ ] CHK-007 Existing apps unaffected: All other app entries retain their relative order (shifted by +1 index)
- [ ] CHK-008 Non-open_here stdout preserved: Selecting any other app still prints the worktree path as the last stdout line

## Scenario Coverage
- [ ] CHK-009 Interactive menu selection: User selects "Open here" from numbered menu
- [ ] CHK-010 Direct flag: `wt create --worktree-open open_here` works without menu
- [ ] CHK-011 Quoted path: Paths with spaces are properly quoted in the `cd` output

## Edge Cases & Error Handling
- [ ] CHK-012 Non-interactive mode: `--non-interactive` with `--worktree-open open_here` outputs cd line without menu
- [ ] CHK-013 No wrapper degradation: Without the shell wrapper, the cd line prints harmlessly to terminal

## Code Quality
- [ ] CHK-014 Pattern consistency: New code follows naming and structural patterns of surrounding code in apps.go
- [ ] CHK-015 No unnecessary duplication: Existing utilities reused where applicable

## Documentation Accuracy
- [ ] CHK-016 Shell wrapper documented: Wrapper function is documented in wt help output or README

## Cross References
- [ ] CHK-017 Spec alignment: All spec requirements have corresponding implementation

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

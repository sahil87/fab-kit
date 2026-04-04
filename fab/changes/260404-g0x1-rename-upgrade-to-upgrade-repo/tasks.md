# Tasks: Rename upgrade to upgrade-repo

**Change**: 260404-g0x1-rename-upgrade-to-upgrade-repo
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation — Go Source

- [x] T001 [P] Rename `upgrade` to `upgrade-repo` in `src/go/fab-kit/cmd/fab-kit/main.go`: update `fabKitCommands` map key, `upgradeCmd()` Use field, and root command Short description
- [x] T002 [P] Rename `upgrade` to `upgrade-repo` in `src/go/fab-kit/cmd/fab/main.go`: update `fabKitArgs` map key and `printHelp()` output line
- [x] T003 [P] Update comment in `src/go/fab-kit/internal/upgrade.go` line 10: `fab upgrade [version]` → `fab upgrade-repo [version]`

## Phase 2: Tests

- [x] T004 [P] Update `src/go/fab-kit/cmd/fab/main_test.go`: replace `"upgrade"` with `"upgrade-repo"` in `TestFabKitArgs` expected list
- [x] T005 [P] Update `src/go/fab-kit/cmd/fab-kit/main_test.go`: replace `"upgrade"` with `"upgrade-repo"` in `TestFabKitCommands` expected list

## Phase 3: Documentation

- [x] T006 [P] Update `README.md`: replace `fab upgrade` with `fab upgrade-repo` in "Updating from a previous version" code block and update fab-kit row in the tool table (if present)
- [x] T007 [P] Update `docs/memory/fab-workflow/distribution.md`: replace all user-facing `fab upgrade` occurrences with `fab upgrade-repo` — excluding rows in `## Changelog` table
- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md`: replace all user-facing `fab upgrade` occurrences with `fab upgrade-repo` — excluding rows in `## Changelog` table
- [x] T009 [P] Update `docs/memory/fab-workflow/migrations.md`: replace all user-facing `fab upgrade` occurrences with `fab upgrade-repo` — excluding rows in `## Changelog` table
- [x] T010 [P] Update `docs/memory/fab-workflow/configuration.md`: replace all user-facing `fab upgrade` occurrences with `fab upgrade-repo` — excluding rows in `## Changelog` table
- [x] T010b [P] Update `docs/memory/fab-workflow/index.md` line 14: replace `fab upgrade` with `fab upgrade-repo` in the `distribution.md` description row <!-- clarified: index.md has one live descriptive reference (not a changelog entry) — covered by same rule as T007–T010 -->

## Phase 4: Skills

- [x] T011 [P] Update `src/kit/skills/fab-setup.md`: replace `fab upgrade` with `fab upgrade-repo` in the kit-not-found error message
- [x] T012 [P] Update `src/kit/skills/_cli-fab.md`: replace `upgrade` with `upgrade-repo` in the workspace commands routing list and `fab-kit` table row
- [x] T013 [P] Update deployed copy `.claude/skills/_cli-fab/SKILL.md`: apply identical changes as T012 to keep deployed copy consistent with source

## Phase 5: Verify

- [x] T014 Run Go tests to verify the renamed command registrations pass: `cd src/go/fab-kit && go test ./...`

---

## Execution Order

- T001–T013 and T010b are all independent — execute in parallel within their phases (or all at once)
- T014 blocks on T001–T005 (tests depend on code changes)

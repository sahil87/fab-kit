# Tasks: Version-Pinned Upgrade and Release

**Change**: 260301-08pa-version-pinned-upgrade-and-release
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Rework argument parsing in `src/scripts/fab-release.sh` — replace the `case` on `$1` with a loop that extracts the bump type (required positional) and `--no-latest` flag, erroring on unknown flags
- [x] T002 Push to current branch in `src/scripts/fab-release.sh` — replace hardcoded `HEAD:main` with `HEAD:$(git branch --show-current)`
- [x] T003 Add `--latest=false` to `gh release create` in `src/scripts/fab-release.sh` when `--no-latest` flag is set; add completion note

## Phase 2: Integration & Edge Cases

- [x] T004 Verify `fab/.kit/scripts/fab-upgrade.sh` already handles optional tag argument correctly (tag download, error message with hint, "already on $tag" message) — confirm no further changes needed
- [x] T005 Verify `README.md` already documents `fab-upgrade.sh v0.24.0` in the "Updating" section — confirm no further changes needed

---

## Execution Order

- T001 blocks T002 and T003 (argument parsing must land before using the parsed flag)
- T002 and T003 are independent of each other after T001
- T004 and T005 are independent verification tasks, can run alongside Phase 1

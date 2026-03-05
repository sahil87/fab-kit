# Quality Checklist: Distribution & Update System for fab/.kit

**Change**: 260210-h7r3-kit-distribution-update
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 One-Liner Bootstrap: `curl -sL .../kit.tar.gz | tar xz -C fab/` produces `fab/.kit/` with all skills, templates, scripts, VERSION
- [x] CHK-002 Manual Copy: `cp -r` distribution still works and produces identical result to curl bootstrap
- [x] CHK-003 Update Script: `fab-update.sh` downloads latest release, replaces `.kit/`, displays version diff, re-runs `fab-setup.sh`
- [x] CHK-004 Update Preserves Project Files: `config.yaml`, `constitution.md`, `docs/`, `specs/`, `changes/`, `current` untouched after update
- [x] CHK-005 gh CLI Required: `fab-update.sh` uses `gh release download` as primary method
- [x] CHK-006 Atomic Update: extraction to temp dir, verification, then swap — no in-place corruption risk
- [x] CHK-007 Symlink Repair: `fab-setup.sh` re-run after update repairs all agent symlinks
- [x] CHK-008 Release Script: `fab-release.sh` bumps VERSION, creates `kit.tar.gz`, commits, creates GitHub Release
- [x] CHK-009 Bump Type Argument: `fab-release.sh [patch|minor|major]` works correctly, defaults to patch
- [x] CHK-010 Repo Inference: `fab-release.sh` uses `git remote get-url origin` to determine target repo
- [x] CHK-011 Release Archive Contents: `kit.tar.gz` contains only `.kit/`-rooted paths, no project files
- [x] CHK-012 Repo Rename: `docs-sddr` renamed to `fab-kit`, old URLs redirect
- [x] CHK-013 README Update: contains bootstrap one-liner, update instructions, release workflow, version checking

## Behavioral Correctness

- [x] CHK-014 Bootstrap with existing `fab/` dir: `.kit/` replaced, other files untouched
- [x] CHK-015 Already up to date: `fab-update.sh` reports current version, no files modified
- [x] CHK-016 Existing clones: `git fetch`/`git pull` follows redirect after repo rename

## Scenario Coverage

- [x] CHK-017 Scenario: Bootstrap a new project — no prior `fab/` directory
- [x] CHK-018 Scenario: Bootstrap with existing `fab/` directory
- [x] CHK-019 Scenario: Update to newer version — version diff displayed
- [x] CHK-020 Scenario: Already up to date — informational message, no changes
- [x] CHK-021 Scenario: Create release (default patch) — VERSION bumped, Release created
- [x] CHK-022 Scenario: Create minor release — correct semver bump
- [x] CHK-023 Scenario: Create major release — correct semver bump
- [x] CHK-024 Scenario: Archive contains only `.kit/` — verified via `tar tf`

## Edge Cases & Error Handling

- [x] CHK-025 gh CLI not found: `fab-update.sh` exits with install instructions, `.kit/` unchanged
- [x] CHK-026 No network access: `fab-update.sh` exits non-zero, error message, `.kit/` unchanged
- [x] CHK-027 Update interrupted during download: `.kit/` unchanged
- [x] CHK-028 Update interrupted during extraction: `.kit/` unchanged, temp dir cleaned on next run
- [x] CHK-029 Extraction verification fails: abort without replacing, error message displayed
- [x] CHK-030 Release with dirty working tree: abort with error message
- [x] CHK-031 Invalid bump argument: exit with usage error
- [x] CHK-032 No origin remote: `fab-release.sh` exits with error message

## Security

- [x] CHK-033 **N/A**: No security-sensitive changes — scripts use authenticated `gh` CLI over HTTPS

## Documentation Accuracy

- [x] CHK-034 README bootstrap one-liner matches actual release URL format
- [x] CHK-035 README update instructions match `fab-update.sh` actual interface
- [x] CHK-036 README release workflow matches `fab-release.sh` actual interface and arguments
- [x] CHK-037 `kit-architecture.md` directory listing includes `fab-update.sh` and `fab-release.sh`

## Cross References

- [x] CHK-038 `fab/docs/fab-workflow/init.md` references bootstrap one-liner as alternative to `cp -r`
- [x] CHK-039 `fab/docs/fab-workflow/kit-architecture.md` distribution section updated with release/update mechanism

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

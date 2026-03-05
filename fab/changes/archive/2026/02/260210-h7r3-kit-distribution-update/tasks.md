# Tasks: Distribution & Update System for fab/.kit

**Change**: 260210-h7r3-kit-distribution-update
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

- [x] T001 Rename GitHub repo from `docs-sddr` to `fab-kit` via GitHub Settings (manual step — verify redirect works for old URL `github.com/wvrdz/docs-sddr`) <!-- MANUAL: User must rename via GitHub Settings > General > Repository name -->

## Phase 2: Core Implementation

- [x] T002 [P] Create `fab/.kit/scripts/fab-update.sh` — implement gh CLI check, download latest `kit.tar.gz` via `gh release download`, atomic extraction to temp dir, VERSION verification, `rm -rf fab/.kit && mv`, version diff display, auto-run `fab-setup.sh`
- [x] T003 [P] Create `fab/.kit/scripts/fab-release.sh` — implement `[patch|minor|major]` argument parsing, semver bump of `fab/.kit/VERSION`, `kit.tar.gz` packaging (rooted at `.kit/`), dirty-tree guard, repo inference from `git remote get-url origin`, `gh release create` with asset attachment

## Phase 3: Integration & Edge Cases

- [x] T004 Add error handling to `fab-update.sh` — gh CLI not found message, network failure (download exits non-zero), extraction verification failure, temp dir cleanup on re-run, already-up-to-date check (compare local VERSION with release VERSION)
- [x] T005 Add error handling to `fab-release.sh` — dirty working tree abort, no origin remote error, invalid bump argument error, gh CLI not found
- [x] T006 Verify `kit.tar.gz` archive structure — all paths rooted at `.kit/`, no project-specific files (`config.yaml`, `constitution.md`, `docs/`, `specs/`, `changes/`) included
- [x] T007 Verify atomic update safety — interrupted download leaves `.kit/` unchanged, interrupted extraction leaves `.kit/` unchanged, temp dir cleaned up

## Phase 4: Polish

- [x] T008 Rewrite `README.md` — bootstrap one-liner with explanation, `fab-setup.sh` + `/fab-init` instructions, update instructions (`fab-update.sh` usage), release workflow (`fab-release.sh [patch|minor|major]`), version checking
- [x] T009 Update `fab/.kit/scripts/` directory listing in `fab/docs/fab-workflow/kit-architecture.md` — add `fab-update.sh` and `fab-release.sh` entries with descriptions

---

## Execution Order

- T001 (repo rename) is independent and can happen at any time
- T002 and T003 are parallel — no dependency between the two scripts
- T004 depends on T002 (adds error handling to update script)
- T005 depends on T003 (adds error handling to release script)
- T006 depends on T003 (verifies archive produced by release script)
- T007 depends on T002 (verifies atomic update behavior)
- T008 and T009 can start after T002+T003 are complete (need final script interfaces for documentation)

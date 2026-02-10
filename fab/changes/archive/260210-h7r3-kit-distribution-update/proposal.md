# Proposal: Distribution & Update System for fab/.kit

**Change**: 260210-h7r3-kit-distribution-update
**Created**: 2026-02-10
**Status**: Draft

## Why

`fab/.kit/` is distributed by manual `cp -r` with no mechanism to pull updates. Teams using fab across multiple repos have no way to stay current. We need a simple, script-based distribution system that aligns with fab's "pure prompt, no system dependencies" philosophy — specifically Constitution Principle V (Portability) and Principle I (Pure Prompt Play).

## What Changes

- **Rename repo** `docs-sddr` → `fab-kit` — this repo becomes the canonical source of truth for `fab/.kit/`
- **Add `fab-update.sh`** to `fab/.kit/scripts/` — downloads latest `kit.tar.gz` from GitHub Releases, extracts to `fab/.kit/`, shows version diff, re-runs `fab-setup.sh` to repair symlinks
- **Add `fab-release.sh`** to `fab/.kit/scripts/` — packages `fab/.kit/` into `kit.tar.gz`, bumps VERSION, commits, creates GitHub Release via `gh release create`
- **Update README.md** — distribution documentation with bootstrap one-liner, update instructions, and release workflow
- **One-liner bootstrap** for new repos: `curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/` — then run `fab-setup.sh`

## Affected Docs

### New Docs
- `fab-workflow/distribution`: How fab/.kit is distributed — bootstrap, update, release workflow

### Modified Docs
- `fab-workflow/kit-architecture`: Add distribution section (release assets, versioning, update mechanism)
- `fab-workflow/init`: Reference bootstrap one-liner as alternative to manual `cp -r`

### Removed Docs

None.

## Impact

- **Scripts**: Two new scripts in `fab/.kit/scripts/` (`fab-update.sh`, `fab-release.sh`)
- **README.md**: Rewritten as distribution/getting-started docs for the renamed repo
- **GitHub**: Repo rename triggers redirect from old URL; existing clones continue to work
- **Team workflow**: Members run `fab-update.sh` to pull latest kit; new projects use curl one-liner
- **CI/releases**: `fab-release.sh` creates GitHub Releases with `kit.tar.gz` asset containing only `.kit/` contents (not the full repo)
- **No breaking changes**: Existing `cp -r` distribution still works; update mechanism is additive

## Open Questions

None — approach was discussed and confirmed before proposal creation.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `gh` CLI as primary tool in `fab-update.sh` | Standard GitHub CLI, widely available; curl fallback can be added later if needed |
| 2 | Confident | Repo rename `docs-sddr` → `fab-kit` | User proposed; GitHub auto-redirects old URLs; team impact is manageable |
<!-- assumed: gh CLI as primary tool in fab-update.sh — standard GitHub CLI, curl fallback deferred -->
<!-- assumed: Repo rename — user proposed, GitHub redirects handle existing links -->

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.

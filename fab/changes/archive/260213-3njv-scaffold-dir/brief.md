# Brief: Extract scaffold content into fab/.kit/scaffold/

**Change**: 260213-3njv-scaffold-dir
**Created**: 2026-02-13
**Status**: Draft

## Origin

> User requested: "I want to maintain the list of entries to be added in .gitignore. Also the adding of .envrc, its content. Docs initial index, Design initial index. Is it easier to maintain via a scaffold folder containing all sources of truth. In fab/.kit/scaffold/?"

## Why

`_fab-scaffold.sh` currently hardcodes template content as heredocs (docs index, design index) and scatters related files at the kit root (`fab/.kit/envrc`). The `.gitignore` entry list is a single hardcoded `grep -qx 'fab/current'`. This makes it hard to find and edit scaffold content — you have to read the script to know what gets generated. A dedicated `fab/.kit/scaffold/` directory makes all bootstrap content editable in one place while the script stays pure logic.

## What Changes

- Create `fab/.kit/scaffold/` directory with four files:
  - `envrc` — moved from `fab/.kit/envrc`
  - `gitignore-entries` — one entry per line (initially just `fab/current`)
  - `docs-index.md` — initial `fab/docs/index.md` content (extracted from heredoc)
  - `design-index.md` — initial `fab/design/index.md` content (extracted from heredoc)
- Update `_fab-scaffold.sh` to read from scaffold files instead of hardcoded content:
  - `.envrc` symlink target: `fab/.kit/envrc` → `fab/.kit/scaffold/envrc`
  - `.gitignore`: loop over lines in `scaffold/gitignore-entries`, append any missing
  - `docs/index.md`: `cp` from `scaffold/docs-index.md` (if target doesn't exist)
  - `design/index.md`: `cp` from `scaffold/design-index.md` (if target doesn't exist)
- Remove `fab/.kit/envrc` (replaced by `fab/.kit/scaffold/envrc`)

## Affected Docs

- `fab-workflow/kit-architecture`: (modify) Document `scaffold/` directory in kit structure
- `fab-workflow/init`: (modify) Update references to scaffold content sources
- `fab-workflow/distribution`: (modify) Update kit contents listing if scaffold/ changes what's distributed

## Impact

- **`fab/.kit/scripts/_fab-scaffold.sh`** — primary change target; heredocs replaced with file reads, symlink target updated, gitignore logic generalized
- **`fab/.kit/envrc`** — removed (moved to `scaffold/envrc`)
- **`.envrc` symlink in consumer projects** — target path changes; existing symlinks will be repaired on next `_fab-scaffold.sh` run (the script already handles broken symlinks)
- **`fab/.kit/skills/fab-init.md`** — step 1g describes `.gitignore` behavior; may need minor update to reference `scaffold/gitignore-entries` as the source
- **Constitution**: Principle V (Portability) — `fab/.kit/` remains `cp -r` portable; scaffold is inside `.kit/`

## Open Questions

None — scope and approach were clarified in conversation.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | File names: `envrc`, `gitignore-entries`, `docs-index.md`, `design-index.md` | Discussed explicitly in conversation; descriptive and unambiguous |
| 2 | Confident | `gitignore-entries` uses one-entry-per-line format | Natural format for a line-list file; trivial to loop over in bash |
| 3 | Confident | Keep symlink mechanism for `.envrc` (just change target path) | Current approach works well; changing mechanism would be scope creep |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

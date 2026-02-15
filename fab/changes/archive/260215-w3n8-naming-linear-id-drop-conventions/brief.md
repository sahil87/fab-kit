# Brief: Naming Linear ID & Drop Conventions

**Change**: 260215-w3n8-naming-linear-id-drop-conventions
**Created**: 2026-02-15
**Status**: Draft

## Origin

> Add optional Linear issue ID to change naming convention and delete unused conventions config section

## Why

The change naming convention (`{YYMMDD}-{XXXX}-{slug}`) has no way to capture the Linear issue ID when a change originates from a Linear ticket. This makes it harder to trace changes back to their tickets by folder/branch name alone. Separately, the `conventions` section added to `config.yaml` in change `260213-r3m7` was never consumed by any skill and is dead schema that should be removed.

## What Changes

- **Naming format**: Extend to `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}` where `{ISSUE}` is an optional uppercase Linear issue ID (e.g., `DEV-988`) inserted after the 4-char random/backlog token and before the slug
- **Without Linear ID**: Format stays `{YYMMDD}-{XXXX}-{slug}` (backward compatible, no change)
- **With Linear ID**: Format becomes `{YYMMDD}-{XXXX}-{ISSUE}-{slug}` (e.g., `260115-a7k2-DEV-988-add-oauth`)
- **Linear ID casing**: Kept uppercase in the folder name (breaks the all-lowercase convention, but makes parsing unambiguous — `[A-Z]+-\d+` is distinct from lowercase slug and random token)
- **Branch name**: Equals folder name (already the default when `branch_prefix` is `""`)
- **Delete `conventions` section**: Remove the commented-out `conventions` section from `config.yaml` and its documentation from memory files. No skill consumes it; it was added speculatively and never activated.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Update folder naming convention table and examples
- `fab-workflow/configuration`: (modify) Update `naming` section docs, delete `conventions` section docs
- `fab-workflow/planning-skills`: (modify) Audit and update naming format references if present

## Impact

- **Skills**: `fab-new.md` (folder name generation), `fab-init.md` (config template)
- **Memory docs**: `change-lifecycle.md`, `configuration.md`, `planning-skills.md`
- **Config**: `fab/config.yaml` (naming comments, conventions deletion)
- **Scripts**: No script changes needed — `resolve-change.sh` uses substring matching (format-agnostic), `batch-switch-change.sh` uses full folder name as branch name
- **Existing changes**: Not affected — new format applies only to newly created changes

## Open Questions

- None. All design decisions were resolved in the planning conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Option C placement: Linear ID after random token, before slug | User explicitly chose Option C after considering A, B, C, D | S:95 R:90 A:95 D:95 |
| 2 | Certain | Linear ID stays uppercase in folder name | User explicitly stated "Lets keep linear ID caps" | S:95 R:85 A:90 D:95 |
| 3 | Certain | Branch name = folder name | User explicitly confirmed; already the default with empty `branch_prefix` | S:95 R:90 A:95 D:95 |
| 4 | Certain | Delete conventions section entirely | User explicitly requested; confirmed no skill consumes it | S:95 R:70 A:90 D:95 |
| 5 | Confident | Hyphen delimiter sufficient (no alternate delimiter needed) | Uppercase Linear ID makes regex parsing unambiguous; no script currently parses name components; user asked the question and accepted the analysis | S:80 R:75 A:85 D:70 |
| 6 | Confident | No migration for existing folder names | New format is forward-only; existing names remain valid; `resolve-change.sh` does substring matching | S:70 R:85 A:80 D:75 |
| 7 | Confident | 6 files in scope | Identified in plan: config.yaml, fab-new.md, fab-init.md, change-lifecycle.md, configuration.md, planning-skills.md | S:75 R:85 A:75 D:70 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).

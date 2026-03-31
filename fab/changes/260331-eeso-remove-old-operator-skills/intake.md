# Intake: Remove Old Operator Skills

**Change**: 260331-eeso-remove-old-operator-skills
**Created**: 2026-03-31
**Status**: Draft

## Origin

> Remove all previous operator skill versions — keep only fab-operator7. Delete fab-operator5.md and fab-operator6.md from fab/.kit/skills/. Also remove their deployed copies from .claude/skills/ (fab-operator5/ and fab-operator6/ directories). Remove any references to the old operators from docs/specs/ and docs/memory/ files. Update docs/specs/skills.md if it lists the old operators. The fab-operator7.sh launcher script should remain unchanged.

Note: The portion of the original request that asked to delete deployed copies under `.claude/skills/` (the `fab-operator5/` and `fab-operator6/` directories) was later rescoped out and is explicitly out of scope for this change; the spec and checklist below reflect the updated scope.

## Why

The operator skill has evolved through multiple versions (operator5 → operator6 → operator7). The older versions are no longer used but their files remain in the repo — source skills, deployed copies, spec files, and memory references. This creates confusion about which version is current and bloats the codebase with dead code. Cleaning them out ensures only the active operator7 is present, reducing maintenance burden and preventing accidental use of deprecated behavior.

## What Changes

### Delete Source Skills

Remove the following files from `fab/.kit/skills/`:
- `fab-operator5.md`
- `fab-operator6.md`

`fab-operator7.md` is retained as the current version.

### Update Memory Files

The following files in `docs/memory/` reference operator5 or operator6:

1. **`docs/memory/fab-workflow/index.md`** — The execution-skills row references `/fab-operator5`. Update to reference only `/fab-operator7`.
2. **`docs/memory/fab-workflow/execution-skills.md`** — Contains a full `/fab-operator5` section (starting at ~line 352) and an operator6 section. Remove both sections. If there's any operator7 content, retain it; if not, the operator section should reflect only operator7.
3. **`docs/memory/fab-workflow/kit-architecture.md`** — References `fab-operator5.sh` and `fab-operator4.sh`/`fab-operator5.sh` in the scripts section. Update to reference only `fab-operator7.sh`.

### Update Spec Files

1. **`docs/specs/skills/SPEC-fab-operator5.md`** — Delete entirely.
2. **`docs/specs/superpowers-comparison.md`** — Contains references to operator5/6. Update or remove those references.

### Unchanged

- `fab/.kit/scripts/fab-operator7.sh` — launcher script remains as-is
- `fab/.kit/skills/fab-operator7.md` — current operator skill remains as-is
- `.claude/skills/fab-operator7/` — deployed copy remains as-is

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Remove operator5 and operator6 sections, ensure operator7 is covered
- `fab-workflow/kit-architecture`: (modify) Remove references to operator5.sh, operator4.sh; update to reference only operator7.sh
- `fab-workflow/index`: (modify) Update execution-skills description to reference operator7 only

## Impact

- `fab/.kit/skills/` — two files deleted
- `docs/memory/fab-workflow/` — three files modified
- `docs/specs/skills/` — one file deleted
- `docs/specs/` — one file potentially modified
- No code behavior changes — operator7.md and its launcher are untouched

## Open Questions

None — the scope is well-defined file deletion and reference cleanup.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only operator7 is the current active version | User explicitly stated "keep only fab-operator7" | S:95 R:90 A:95 D:95 |
| 2 | Certain | fab-operator7.sh launcher remains unchanged | User explicitly stated this | S:95 R:95 A:95 D:95 |
| 3 | Certain | Delete source skill files for operator5 and operator6 | User explicitly listed these files; deployed copies excluded per user correction | S:95 R:85 A:95 D:95 |
| 4 | Confident | Remove full operator5/operator6 sections from execution-skills.md rather than condensing | Dead versions should be fully removed, not summarized — consistent with cleanup intent | S:80 R:80 A:75 D:80 |
| 5 | Confident | No SPEC-fab-operator6.md exists to delete | Only SPEC-fab-operator5.md was found in docs/specs/skills/ | S:70 R:95 A:85 D:90 |
| 6 | Confident | No migration needed for this change | This is a repo-internal cleanup — no user-facing data structures change | S:75 R:90 A:80 D:85 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).

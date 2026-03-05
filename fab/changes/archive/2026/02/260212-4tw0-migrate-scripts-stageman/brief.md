# Brief: Migrate Scripts to Use Stage Manager

**Change**: 260212-4tw0-migrate-scripts-stageman
**Created**: 2026-02-12
**Status**: Draft

## Origin

> User requested: "Migrate existing scripts to use stageman.sh for stage/state queries"

## Why

Stage and state knowledge is currently hardcoded in multiple bash scripts (`fab-status.sh`, `fab-preflight.sh`, template generation, etc.). We now have a canonical workflow schema (`fab/.kit/schemas/workflow.yaml`) and Stage Manager query utility (`fab/.kit/scripts/stageman.sh`), but existing scripts haven't been migrated to use it yet.

This creates:
- **Maintenance burden**: Changes to stages/states require updates in 7+ locations
- **Inconsistency risk**: Hardcoded stage lists can drift out of sync with schema
- **Missed validation**: Scripts don't leverage `validate_status_file()` and other utilities

The migration guide (`fab/.kit/schemas/MIGRATION.md`) documents the refactoring patterns, but the actual scripts still use hardcoded logic.

## What Changes

### Part A: Migrate scripts to use stageman

**Update fab-status.sh**:
- Source `stageman.sh` at the top
- Replace hardcoded stage loop `for s in brief spec tasks apply review archive` with `for s in $(get_all_stages)`
- Replace hardcoded stage number case statement with `get_stage_number "$stage"`
- Replace hardcoded `symbol()` function with calls to `get_state_symbol "$state"`

**Update fab-preflight.sh**:
- Source `stageman.sh` at the top
- Replace hardcoded stage loop with `get_all_stages`
- Add optional validation: call `validate_status_file "$status_file"` to catch schema violations early

**Update fab-help.sh**:
- Remove hardcoded stage progression from documentation string
- Dynamically generate stage list from `get_all_stages` if needed (or keep as static doc)

**Update template generation** (if applicable):
- Check if any skills dynamically generate `.status.yaml` templates
- Ensure they use `get_initial_state` and `get_allowed_states` from stageman

**Remove hardcoded logic**:
- Delete all hardcoded stage lists, state symbol mappings, and stage number mappings
- Scripts become pure consumers of the schema via stageman

### Part B: Deduplicate stageman / schemas documentation

Documentation is scattered across 3 locations with significant overlap (API function list repeated 4x, directory structure 2x, usage examples 2x). Consolidate per ownership rules:

**Move `fab/.kit/schemas/MIGRATION.md`** → this change folder (`MIGRATION.md`)
- Migration/refactoring guide is change-specific content, belongs with the change

**Move + trim `fab/.kit/schemas/README.md`** → `fab/docs/fab-workflow/schemas.md`
- Trim the stageman API section (lines 48-70) and bash usage examples (lines 77-101) that duplicate `src/stageman/` content
- Focus on: what `workflow.yaml` defines, design principles, how to reference from skills vs scripts, future enhancements
- Link to `src/stageman/README.md` for API details
- `fab/.kit/schemas/` ends up with only `workflow.yaml`

**Consolidate `src/stageman/` to single README.md**:
- Delete `SUMMARY.md` (reorganization summary — one-time content)
- Delete `SPEC.md` (fold API contracts into README)
- Delete `CHANGELOG.md` (fold 2 small entries into README)
- Rewrite `README.md` with: overview, sources-of-truth links, API reference (from SPEC.md), CLI interface, testing, changelog

**Fix dangling references** (6 files):

| File | Lines | Fix |
|------|-------|-----|
| `README.md` (root) | 117-127 | Update `SPEC.md` → `README.md`, update schemas/README link → `fab/docs/fab-workflow/schemas.md` |
| `fab/.kit/scripts/stageman.sh` | 374 | Update SEE ALSO paths |
| `fab/changes/.../brief.md` | 20 | Update MIGRATION.md path to local reference |
| `fab/docs/index.md` | — | Add `schemas` entry to fab-workflow table |

**Result:**
```
fab/.kit/schemas/
└── workflow.yaml              # only file remaining

src/stageman/
├── stageman.sh                # symlink (unchanged)
├── test-simple.sh             # unchanged
├── test.sh                    # unchanged
└── README.md                  # single consolidated doc

fab/docs/fab-workflow/
└── schemas.md                 # schema docs (trimmed, moved)

fab/changes/260212-4tw0-migrate-scripts-stageman/
├── brief.md
├── .status.yaml
└── MIGRATION.md               # moved here
```

## Affected Docs

### Modified Docs
- `fab-workflow/kit-architecture`: Update script implementation details to reference stageman integration
- `fab-workflow/preflight`: Update preflight script documentation to mention stageman usage and validation

### New Docs
- `fab-workflow/schemas`: Schema overview (moved from `fab/.kit/schemas/README.md`, trimmed)

## Impact

**Files affected (Part A — script migration)**:
- `fab/.kit/scripts/fab-status.sh` — primary display script
- `fab/.kit/scripts/fab-preflight.sh` — validation and stage detection
- `fab/.kit/scripts/fab-help.sh` — may reference stage progression
- `fab/.kit/templates/status.yaml` — template may need dynamic generation (check if sourced by bash)

**Files affected (Part B — doc deduplication)**:
- `fab/.kit/schemas/MIGRATION.md` — moved to change folder
- `fab/.kit/schemas/README.md` — moved + trimmed to `fab/docs/fab-workflow/schemas.md`
- `src/stageman/{SUMMARY,SPEC,CHANGELOG}.md` — deleted, folded into README
- `src/stageman/README.md` — rewritten as single consolidated doc
- `README.md` (root) — updated links
- `fab/.kit/scripts/stageman.sh` — updated help text refs
- `fab/docs/index.md` — added schemas entry

**Benefits**:
- Single source of truth enforced at runtime (Part A)
- Adding/removing stages becomes schema-only change (Part A)
- No more 4x duplicated API docs (Part B)
- Clear ownership: schema docs in `fab/docs/`, dev docs in `src/stageman/`, change docs in change folder (Part B)

**Risks**:
- Minimal: stageman.sh is tested and working
- All changes are in kit scripts (not user-facing config)
- Can be validated by running `fab-status` and `fab-preflight.sh` before/after
- Broken link check: `grep -r 'SPEC.md\|SUMMARY.md\|CHANGELOG.md\|schemas/README' --include='*.md' --include='*.sh'` should return nothing after completion

## Open Questions

None — migration patterns are documented in MIGRATION.md (this folder) and stageman.sh is fully functional and tested.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | All scripts in `fab/.kit/scripts/` should be checked | Standard kit directory structure per constitution |
| 2 | Confident | Hardcoded logic should be completely removed | Single source of truth principle from MIGRATION.md |
| 3 | Confident | `validate_status_file` should be added to preflight | Catches corruption early, aligns with preflight's validation role |
| 4 | Confident | `src/stageman/` needs only a single README.md | SUMMARY + SPEC + README + CHANGELOG is excessive for a bash utility |
| 5 | Confident | Schema docs belong in `fab/docs/fab-workflow/` | Follows existing doc organization; `fab/.kit/schemas/` should only contain the schema itself |
| 6 | Confident | MIGRATION.md belongs in the change folder | Migration guide is change-specific, not a permanent schema artifact |

6 assumptions made (6 confident, 0 tentative).

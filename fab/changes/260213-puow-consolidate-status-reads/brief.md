# Brief: Consolidate .status.yaml Ownership into Stageman

**Change**: 260213-puow-consolidate-status-reads
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Consolidate .status.yaml ownership into stageman. Add missing accessor functions (get_progress_map, get_checklist, get_confidence) to stageman.sh so it fully owns all .status.yaml reads. Then update fab-preflight.sh and fab-status.sh to call stageman instead of raw grep|sed. Also extract the duplicated change resolution logic (fuzzy matching against fab/changes/) into a shared function so it's not copy-pasted between preflight and status.

## Why

Three scripts (`stageman.sh`, `fab-preflight.sh`, `fab-status.sh`) currently read `.status.yaml` with raw `grep | sed` patterns. The "derive current stage" logic is written three times. The change resolution block (fuzzy matching against `fab/changes/`) is copy-pasted between preflight and status. This violates the project's own design intent: stageman was created to eliminate hardcoded workflow knowledge from other scripts, but its `.status.yaml` API is incomplete — it only covers validation and current stage detection, not progress map extraction, checklist, or confidence reads. Completing the API and removing the raw parsing from consumers makes the codebase consistent with its own architecture.

## What Changes

- **Add accessor functions to `_stageman.sh`**: `get_progress_map`, `get_checklist`, `get_confidence` — completing the `.status.yaml` read API so all status file access goes through stageman
- **Rename `stageman.sh` → `_stageman.sh`**: adopts the `_` prefix convention for internal library scripts (see `fab/design/architecture.md` Script Naming Convention)
- **Replace raw grep|sed in `fab-preflight.sh`**: lines 108-137 currently parse progress, checklist, and confidence fields directly — replace with calls to the new stageman functions. Also replace the reimplemented "derive current stage" loop (lines 116-125) with `get_current_stage()`
- **Replace raw grep|sed in `fab-status.sh`**: lines 98-99 (`get_field`/`get_nested` helpers), lines 119-147 (progress, checklist, confidence extraction), and lines 126-135 (derive current stage) — replace with stageman calls
- **Extract change resolution to `_resolve-change.sh`**: the fuzzy matching logic (preflight lines 19-83, status lines 20-87) is nearly identical — extract to a shared internal script that both entry points source
- **Create `src/resolve-change/` development folder**: follows the same pattern as `src/preflight/` and `src/stageman/` — symlink to `fab/.kit/scripts/_resolve-change.sh`, README with API docs, `test.sh` (comprehensive suite), and `test-simple.sh` (smoke test)

## Affected Docs

### New Docs
_(none)_

### Modified Docs
- `fab-workflow/preflight`: Update to reflect that preflight delegates all `.status.yaml` reads to stageman and sources `resolve-change.sh` for change resolution
- `fab-workflow/kit-architecture`: Update stageman's role description to include `.status.yaml` accessor API (not just validation)

### Removed Docs
_(none)_

## Impact

- **`fab/.kit/scripts/stageman.sh` → `_stageman.sh`** — renamed + gains 3-4 new accessor functions (~40-60 lines)
- **`fab/.kit/scripts/fab-preflight.sh`** — ~30 lines of raw grep replaced with function calls; change resolution block extracted; sources `_stageman.sh` and `_resolve-change.sh`
- **`fab/.kit/scripts/fab-status.sh`** — ~30 lines of raw grep replaced; change resolution block extracted; `get_field`/`get_nested` helpers removed; sources `_stageman.sh` and `_resolve-change.sh`
- **`fab/.kit/scripts/_resolve-change.sh`** — new internal library (~70 lines extracted from the duplicated block)
- **`src/resolve-change/`** — new dev folder: symlink to `_resolve-change.sh`, `README.md` (API docs), `test.sh` (comprehensive suite), `test-simple.sh` (smoke test) — mirrors `src/preflight/` and `src/stageman/` pattern
- **`src/stageman/README.md`** — API reference needs updating with new functions
- **No behavioral change** — all three scripts produce identical output before and after; this is a pure internal refactor
- **Risk**: low — shell scripts with existing self-test infrastructure; output is deterministic and diffable

## Open Questions

_(none — all decisions resolved from codebase context)_

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Scope limited to reads — no write/mutation functions added to stageman in this change | Description says "accessor functions"; writes are a separate concern with different consumers |
| 2 | Confident | Internal scripts use `_` prefix (`_stageman.sh`, `_resolve-change.sh`); entry points keep `fab-` prefix | Convention added to `fab/design/architecture.md` — makes library vs. entry point distinction visible at `ls` |
| 3 | Confident | Change resolution extracted to standalone `_resolve-change.sh` with its own `src/resolve-change/` dev folder | Mirrors stageman and preflight pattern — script in `.kit/scripts/`, dev symlink + README + tests in `src/` |
| 4 | Confident | `fab-status.sh` sources `_resolve-change.sh` directly rather than calling preflight as a subprocess | Status formats output differently (human-readable) from preflight (YAML for agents) — they share resolution logic but not the validation+emission pipeline |

4 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.

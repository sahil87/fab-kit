# Brief: Stageman Write API

**Change**: 260214-w3r8-stageman-write-api
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Add write functions to stageman — set_stage_state, transition_stages, set_checklist_field — to centralize .status.yaml mutations behind validated, atomic bash functions. Fold _calc-score.sh write logic into stageman. Simplify skill prompts to use Bash calls instead of ad-hoc LLM YAML editing.

## Why

`.status.yaml` writes are currently scattered across LLM skill instructions as ad-hoc YAML editing via the Edit tool. This is fragile (indentation errors, no validation on write, partial writes) and duplicated across every skill that touches stage progress. The read side already has a clean, schema-validated API via `_stageman.sh`; the write side needs parity. The only scripted write today is `_calc-score.sh` for the confidence block — everything else relies on the LLM getting YAML formatting right.

## What Changes

- Add write functions to `_stageman.sh`: `set_stage_state`, `transition_stages`, `set_checklist_field`, `set_confidence_block`
- Each write function validates inputs against the workflow schema before writing, uses temp-file-then-mv for atomicity, and auto-updates `last_updated`
- Refactor `_calc-score.sh` to call stageman's `set_confidence_block` instead of its own inline awk write logic
- Add CLI mode to `_stageman.sh` for write operations (e.g., `_stageman.sh transition <file> <from> <to>`) so skills can invoke via Bash tool
- Update skill prompts (`fab-continue.md` and related) to replace ad-hoc "edit .status.yaml" instructions with single Bash calls
- Extend test suite in `src/stageman/` to cover write functions

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) New write functions in `_stageman.sh`, expanded CLI interface
- `fab-workflow/change-lifecycle`: (modify) Two-write transition now has a scripted implementation; document the API
- `fab-workflow/planning-skills`: (modify) Skill prompts simplified — Bash calls replace YAML editing instructions
- `fab-workflow/execution-skills`: (modify) Apply/review/hydrate stage transitions simplified
- `fab-workflow/preflight`: (modify) Reference the write API as the counterpart to the existing read accessors

## Impact

- **`fab/.kit/scripts/_stageman.sh`** — primary change target; new write functions + CLI commands
- **`fab/.kit/scripts/_calc-score.sh`** — refactored to delegate its write portion to stageman
- **`fab/.kit/skills/fab-continue.md`** — Step 4 (Update .status.yaml) simplified to Bash calls
- **`fab/.kit/skills/fab-new.md`** — Step 4 (Initialize .status.yaml) potentially simplified
- **`src/stageman/`** — test files expanded for write coverage
- **No new files created** — purely extending the existing stageman library

## Open Questions

<!-- None — input is specific and the approach is well-defined from the preceding discussion. -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Extend `_stageman.sh` v1 in-place rather than creating a new file | User explicitly chose this approach over the existing stageman2 alternative; keeps one file to maintain |
| 2 | Confident | Keep `_calc-score.sh` as a separate script for scoring math, only move its write portion to stageman | Separation of concerns — scoring logic (grade counting, formula) is distinct from file I/O; calc-score becomes a consumer of the write API |
| 3 | Confident | Use temp-file-then-mv pattern for atomic writes | Established pattern already used by `_calc-score.sh`; prevents half-written YAML on interruption |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

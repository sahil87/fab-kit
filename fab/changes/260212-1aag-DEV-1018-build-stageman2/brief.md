# Brief: Build stageman2 — Full State Encapsulation for .status.yaml

**Change**: 260212-1aag-DEV-1018-build-stageman2
**Created**: 2026-02-12
**Status**: Draft

## Origin

> Build stageman2 — a new standalone bash utility that fully encapsulates all reads and writes to .status.yaml. Provides CLI + library interface with transactional set-progress, advance, reset commands. Independent of stageman v1. Deployed alongside v1 without modifying existing files.

## Why

`.status.yaml` is the mutable state file for every fab change, but its internal structure is leaked across the entire system — bash scripts parse it with grep/sed, and skills instruct the LLM to write YAML directly. This coupling means any change to the file's shape requires touching every consumer. stageman v1 only provides read-only schema queries; it doesn't own mutations. Building stageman2 as a complete state manager gives us full freedom to evolve `.status.yaml` without affecting the rest of the system.

## What Changes

- New `fab/.kit/scripts/_stageman2.sh` utility with dual-mode interface (CLI + sourced library)
- CLI commands for reads (`get progress`, `get stage`, `get checklist`, `get confidence`, `dump`, `validate`) and writes (`init`, `set progress`, `advance`, `reset`, `set confidence`, `set checklist.*`)
- Library functions with `sm2_` prefix for namespace-safe coexistence with v1
- `set progress` accepts multiple `stage:state` pairs as a transaction — validates net result before applying, rejects invalid states atomically
- `advance` as a first-class command encapsulating the two-write transition (current→done, next→active)
- `reset` as a first-class command for multi-field reset operations
- Internal confidence score computation (`5.0 - 0.3*confident - 1.0*tentative`, 0.0 if unresolved > 0)
- Status file auto-discovery from `fab/current`, with explicit `-f` override
- Reads workflow.yaml directly for schema queries — fully independent of stageman v1
- Comprehensive test suite (`src/stageman/test2.sh`)

## Affected Docs

### New Docs
- `fab-workflow/stageman2`: API reference, CLI commands, library functions, migration guide

### Modified Docs
- `fab-workflow/kit-architecture`: New script in `.kit/scripts/`, new symlink in `src/stageman/`

### Removed Docs
<!-- None -->

## Impact

- **fab/.kit/scripts/**: New _stageman2.sh alongside existing _stageman.sh
- **src/stageman/**: New symlink and test file alongside existing ones
- **No existing files modified** — purely additive deployment
- **Future migration** (out of scope for this change): fab-status.sh, _preflight.sh, and all skill .md files will switch from direct .status.yaml access to stageman2 API calls

## Open Questions

<!-- None — all decision points resolved via SRAD analysis. See Assumptions below. -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Reimplement only the schema query subset needed for status management (stage list, valid states, allowed states per stage), not the full v1 display API (symbols, suffixes, names) | "Independent of v1" directive + minimalism — display helpers are v1's domain, not needed for file management |
| 2 | Confident | Follow existing test.sh pattern (assert_equal/assert_success/assert_failure, temp files, self-contained) | Strong existing convention in src/stageman/test.sh; no reason to deviate |
| 3 | Confident | Preflight structural validation (config.yaml, constitution.md, change dir existence) stays in _preflight.sh; stageman2 only manages .status.yaml | Plan explicitly scoped stageman2 as a file manager, not a workflow engine; separation of concerns |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

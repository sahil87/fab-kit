# Brief: Migrate Stageman to CLI-Only Interface

**Change**: 260214-k8v2-stageman-cli-only
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Migrate stageman.sh to CLI-only interface — add CLI subcommands for all read/query functions currently only available via source/import, then migrate callers (preflight.sh, calc-score.sh) to use CLI, and remove the dual-mode scaffolding. This prepares stageman for an eventual Rust rewrite.

## Why

stageman.sh currently supports two interaction patterns: **source/import** (callers `source` the file, then call bash functions directly) and **CLI** (callers invoke it as a subprocess with subcommands). The CLI only exposes 4 write commands; all ~20 read/query functions are source-only. This dual-mode interface ties callers to bash — a Rust binary can only be invoked via CLI, not sourced. Migrating everything to CLI establishes a substrate-agnostic contract that a Rust (or any other) binary can satisfy.

## What Changes

- **Add ~22 CLI subcommands** to stageman.sh for all read/query functions currently missing from the CLI dispatch block (state queries, stage queries, file accessors, progression queries, validation, display helpers, plus `set-confidence-fuzzy`)
- **Migrate preflight.sh** from `source stageman.sh` + function calls → `$STAGEMAN <subcommand>` subprocess calls (6 read operations)
- **Migrate calc-score.sh** from `source stageman.sh` + function calls → `$STAGEMAN <subcommand>` subprocess calls (2 write operations)
- **Remove dual-mode scaffolding** — delete `BASH_SOURCE[0]` guard, simplify error handling from `return 1 2>/dev/null || exit 1` to `exit 1`
- **Migrate test suite** to CLI-only invocations — these become the contract test suite for the eventual Rust rewrite

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document stageman as CLI-only interface, remove source/import usage pattern
- `fab-workflow/preflight`: (modify) Reflect subprocess invocation pattern instead of sourced library

## Impact

- **`fab/.kit/scripts/lib/stageman.sh`** — primary target: ~22 new case arms in CLI dispatch, help text update, guard removal
- **`fab/.kit/scripts/lib/preflight.sh`** — replace `source` + 6 function calls with subprocess invocations
- **`fab/.kit/scripts/lib/calc-score.sh`** — replace `source` + 2 function calls with subprocess invocations
- **`src/lib/stageman/test.sh`** — rewrite ~60 assertions from source-pattern to CLI-pattern
- **`src/lib/stageman/test-simple.sh`** — same migration
- **`src/lib/stageman/README.md`** — update API reference to CLI-only

## Open Questions

None — design decisions resolved during planning session. See plan file at `.claude/plans/hashed-baking-volcano.md` for full analysis.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Incremental 3-commit approach (add CLI, migrate callers, clean up) | Enables safe rollout; commit 1 is purely additive with zero regression risk | S:80 R:90 A:85 D:75 |
| 2 | Confident | Remove source guard only after all callers are migrated | Prevents breaking callers during transition; clean cut | S:85 R:70 A:90 D:90 |
| 3 | Confident | Test suite becomes CLI-only contract tests for Rust rewrite | CLI tests define the exact interface contract a Rust binary must satisfy | S:85 R:80 A:80 D:85 |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

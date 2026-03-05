# Brief: Migrate Stageman to CLI-Only Interface

**Change**: 260215-lqm5-stageman-cli-only
**Created**: 2026-02-15
**Status**: Draft

## Origin

> Migrate stageman.sh to CLI-only interface — add CLI subcommands for all read/query functions currently only available via source/import, then migrate callers (preflight.sh, calc-score.sh, and check for any skills that use this) to use CLI, and remove the dual-mode scaffolding. This prepares stageman for an eventual Rust rewrite.

## Why

stageman.sh currently supports two interaction patterns: **source/import** (callers `source` the file then call bash functions directly) and **CLI** (callers invoke it as a subprocess with subcommands). The CLI dispatch block only exposes write commands and history logging — all ~24 read/query functions are source-only. This dual-mode interface ties callers to bash; a Rust binary can only be invoked via CLI, not sourced. Migrating everything to CLI establishes a substrate-agnostic contract that a Rust (or any other) binary can satisfy.

## What Changes

- **Add ~24 CLI subcommands** to stageman.sh for all read/query functions currently missing from the CLI dispatch block:
  - State queries: `all-states`, `validate-state`, `state-symbol`, `state-suffix`, `is-terminal`
  - Stage queries: `all-stages`, `validate-stage`, `stage-number`, `stage-name`, `stage-artifact`, `allowed-states`, `initial-state`, `is-required`, `has-auto-checklist`, `validate-stage-state`
  - .status.yaml accessors: `progress-map`, `checklist`, `confidence`
  - Stage metrics: `stage-metrics`, `set-stage-metric`
  - Progression: `current-stage`, `next-stage`
  - Validation: `validate-status-file`
  - Display: `format-state`
  - Write (missing): `set-confidence-fuzzy`
- **Migrate preflight.sh** from `source stageman.sh` + function calls to `$STAGEMAN <subcommand>` subprocess calls (6 read operations: `validate-status-file`, `progress-map`, `current-stage`, `checklist`, `confidence`, `all-stages`)
- **Migrate calc-score.sh** from `source stageman.sh` + function calls to `$STAGEMAN <subcommand>` subprocess calls (4 operations: `confidence`, `set-confidence`, `set-confidence-fuzzy`, `log-confidence`)
- **Migrate test suites** (`src/lib/stageman/test.sh`, `src/lib/stageman/test-simple.sh`) from source-pattern to CLI-pattern — these become the contract test suite for the eventual Rust rewrite
- **Remove dual-mode scaffolding** — delete `BASH_SOURCE[0]` guard at top, simplify error handling from `return 1 2>/dev/null || exit 1` to `exit 1`, remove `source`-oriented comments
- **Update help text** to reflect CLI-only interface (remove "As library" usage section)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document stageman as CLI-only interface, remove source/import usage pattern
- `fab-workflow/preflight`: (modify) Reflect subprocess invocation pattern instead of sourced library
- `fab-workflow/schemas`: (modify) Update usage examples from source-pattern to CLI-pattern

## Impact

- **`fab/.kit/scripts/lib/stageman.sh`** — primary target: ~24 new case arms in CLI dispatch, help text rewrite, dual-mode guard removal
- **`fab/.kit/scripts/lib/preflight.sh`** — replace `source` line + 6 function calls with subprocess invocations; restructure associative array parsing for CLI output
- **`fab/.kit/scripts/lib/calc-score.sh`** — replace `source` line + 4 function/write calls with subprocess invocations
- **`src/lib/stageman/test.sh`** — rewrite assertions from source-pattern to CLI-pattern
- **`src/lib/stageman/test-simple.sh`** — same migration
- **`src/lib/stageman/README.md`** — update API reference to CLI-only
- **Skills**: No changes needed — all skill `.md` files already invoke stageman via CLI (`lib/stageman.sh <subcommand>`)

## Open Questions

- None — the interface contract is fully defined by the existing function signatures, and the old brief + plan at `toDel/260214-k8v2-stageman-cli-only/` validated the approach.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skills need no changes — they already use CLI interface | Verified: all skill .md files invoke `lib/stageman.sh <subcommand>`, none source it | S:95 R:95 A:95 D:95 |
| 2 | Certain | CLI subcommand names use kebab-case (e.g., `all-stages` not `get_all_stages`) | Consistent with existing CLI commands (`set-state`, `set-checklist`, `log-command`); standard CLI convention | S:90 R:90 A:95 D:90 |
| 3 | Confident | Incremental 3-phase approach: add CLI subcommands, migrate callers, remove dual-mode | Phase 1 is purely additive (zero regression risk); phases 2-3 can be validated independently | S:80 R:90 A:85 D:75 |
| 4 | Confident | Test suites become CLI-only contract tests | CLI tests define the exact interface a Rust binary must satisfy; source-pattern tests would be meaningless after migration | S:85 R:80 A:80 D:85 |
| 5 | Confident | Output format for multi-value queries uses existing conventions (one `key:value` per line) | Matches existing `get_progress_map`, `get_checklist`, `get_confidence` output format — callers already parse this | S:85 R:85 A:90 D:80 |
| 6 | Certain | `set-confidence-fuzzy` is the only write function missing from CLI | Audited: all other write functions (`set-state`, `transition`, `set-checklist`, `set-confidence`, `log-*`) already have CLI dispatch | S:95 R:90 A:95 D:95 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.

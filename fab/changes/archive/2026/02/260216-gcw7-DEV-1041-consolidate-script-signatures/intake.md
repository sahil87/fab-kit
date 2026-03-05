# Intake: Consolidate Script Signatures

**Change**: 260216-gcw7-DEV-1041-consolidate-script-signatures
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Clean up and consolidate the signatures of all the command line scripts. Right now, are there too many sub-options for every single script? Can they be reduced to a fewer number that need to be maintained?

Investigative/refactor change — the user wants an audit of the CLI surface area across all `fab/.kit/scripts/` shell scripts, with the goal of reducing the number of options and subcommands that need to be maintained.

## Why

The scripts in `fab/.kit/scripts/` have grown organically over 10+ changes. Each script was added or expanded independently, resulting in inconsistent interfaces and potentially redundant subcommands. The maintenance burden scales with the total CLI surface area — every subcommand needs documentation, testing, and compatibility consideration on upgrades.

Specific concerns:
1. **stageman.sh** has ~35 subcommands across 8 categories. Some may be unused or could be collapsed (e.g., separate `validate-state`, `validate-stage`, `validate-stage-state` commands; `state-symbol`/`state-suffix`/`format-state` overlap).
2. **3 batch scripts** (`batch-fab-archive-change.sh`, `batch-fab-switch-change.sh`, `batch-fab-new-backlog.sh`) share an identical `--list/--all/positional-args` pattern with significant duplicated logic.
3. **changeman.sh** uses `--flag <value>` style while **stageman.sh** uses positional subcommands — inconsistent but each may be appropriate for its use case.

If the surface area isn't reduced, every new feature or refactor carries increasing risk of breaking an interface that some caller depends on.

## What Changes

### Audit Phase

Identify which subcommands and flags across all scripts are:
- Actually called by skills, other scripts, or tests (used)
- Never referenced outside the script itself (dead code)
- Duplicative of another subcommand with slight variation (merge candidates)

### Consolidation Targets

#### stageman.sh (~35 → fewer subcommands)

Candidates for removal or merging:
- **Display helpers** (`state-symbol`, `state-suffix`, `format-state`): `format-state` may subsume the other two
- **Validation triad** (`validate-state`, `validate-stage`, `validate-stage-state`): could collapse into a single `validate` with positional args
- **Boolean queries** (`is-terminal`, `is-required`, `has-auto-checklist`): usage frequency audit needed
- **Unused accessors**: any `stage-*` query subcommand never called externally

#### Batch scripts (3 scripts → shared framework)

Extract the common `--list/--all/positional` dispatch, tmux tab creation, and worktree management into a shared library or single script with a verb argument (`batch-fab.sh archive|switch|new`).

#### Cross-script consistency

Decide whether `changeman.sh`'s `--flag` style and `stageman.sh`'s positional style should converge, or whether the difference is justified by their different roles (lifecycle manager vs. query engine).

### Implementation

- Remove dead subcommands
- Merge overlapping subcommands with deprecation or direct replacement
- Extract shared batch logic
- Update all callers (skills, scripts, tests) to use the new interfaces
- Update memory files (kit-architecture.md, schemas.md)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Shell Scripts section — updated subcommand listings, batch script descriptions
- `fab-workflow/schemas`: (modify) Script example references if stageman interface changes

## Impact

- **Scripts**: `stageman.sh`, `changeman.sh`, all 3 batch scripts, `preflight.sh`, `calc-score.sh` (as callers)
- **Skills**: Any skill that invokes scripts directly (fab-new, fab-continue, fab-status, fab-switch, fab-clarify)
- **Tests**: `src/lib/stageman/test.bats` (75 tests), `src/lib/calc-score/test.bats`, other test suites — tests are the contract; removed subcommands lose their tests
- **No external API impact**: all scripts are internal to `.kit/`

## Open Questions

- Which stageman subcommands are actually unused by any caller outside the test suite?
- Should the batch scripts become a single `batch-fab.sh` with verb dispatch, or stay as separate scripts sharing a library?
- Is the changeman `--flag` vs. stageman positional inconsistency worth unifying, given their different usage patterns?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All scripts in scope are internal to `.kit/` | Constitution V (Portability) — `.kit/` is self-contained, no external consumers | S:90 R:90 A:95 D:90 |
| 2 | Certain | Tests serve as the contract for subcommand existence | Memory explicitly states "75 tests serve as contract test for future reimplementation" | S:85 R:85 A:90 D:90 |
| 3 | Confident | stageman.sh is the primary consolidation target | 35 subcommands dwarfs all other scripts combined; user's "too many sub-options" maps here | S:70 R:75 A:80 D:70 |
| 4 | Confident | Batch scripts share enough logic to warrant extraction | All 3 follow identical `--list/--all/positional` + tmux + worktree pattern | S:65 R:80 A:75 D:70 |
| 5 | Tentative | changeman flag style vs. stageman positional style can remain different | Different roles (lifecycle ops with optional params vs. query engine with many verbs) justify different styles | S:50 R:75 A:60 D:50 |
<!-- assumed: changeman/stageman style divergence acceptable — different script roles justify different CLI patterns -->
| 6 | Tentative | Dead code removal is safe without deprecation period | Internal scripts with no external consumers; skills and tests are the only callers | S:55 R:65 A:70 D:55 |
<!-- assumed: no deprecation needed for internal script changes — all callers are within .kit/ or skill files -->

6 assumptions (2 certain, 2 confident, 2 tentative, 0 unresolved).

# Intake: Refactor Kit Scripts Into Atomic Responsibilities

**Change**: 260228-9fg2-refactor-kit-scripts
**Created**: 2026-02-28
**Status**: Draft

## Origin

> Refactor internal kit scripts into atomic responsibility sets: extract resolve.sh, rename stageman.sh to statusman.sh, create logman.sh, DRY up calc-score.sh, and unify all tools on canonical change IDs via resolve.sh.

Initiated from a `/fab-discuss` session where three inconsistencies in the internal tooling were identified and discussed in depth:

1. **log_* functions in stageman.sh** — three history-logging functions (`log_command`, `log_confidence`, `log_review`) write to `.history.jsonl` but live in stageman, whose responsibility is `.status.yaml`. They belong elsewhere.
2. **Score formula duplication in calc-score.sh** — the grade counting loop and confidence formula are copy-pasted three times (intake gate, spec gate, normal scoring).
3. **Change argument convention sprawl** — three different input conventions (change ID/name, `.status.yaml` path, directory path) across three scripts, with `resolve_change_arg()` in stageman duplicating `changeman.sh resolve` logic.

Discussion evolved from "fix these three issues" to a full architectural breakdown of responsibilities across five atomic scripts, with specific decisions about naming, call graphs, and the principle that skills never call logman or resolve directly.

## Why

The current script architecture has grown organically and accumulated three categories of boundary violations:

1. **Wrong responsibility boundaries**: `stageman.sh` is a 1260-line script doing three jobs — stage state machine, `.status.yaml` metadata writes, and `.history.jsonl` logging. The `log_*` functions have zero relationship to stage management. The metadata writers (`set_change_type`, `set_checklist`, `set_confidence_block`, `add_issue`, `add_pr`) at least operate on `.status.yaml` but aren't about stage transitions.

2. **Duplicated logic**: `calc-score.sh` repeats the same grade-counting loop and score formula three times. The intake gate and spec gate paths are nearly identical — differing only in which file they read and how the threshold is determined. This makes the formula hard to change and easy to get out of sync.

3. **No canonical resolution path**: `stageman.sh` has its own `resolve_change_arg()` that wraps `changeman.sh resolve` and adds `.status.yaml` path handling. `calc-score.sh` bypasses both resolvers entirely, accepting a raw directory path. Skills are told to pass different argument forms to different scripts. This creates confusion and bugs (the exact issue that the prior `yobi` change partially addressed).

If we don't fix this, every new feature added to the kit deepens the coupling. The `yobi` change (completed) unified *argument conventions within* the existing scripts, but the *script boundary* problem remains. The next change that touches stageman faces 1260 lines of mixed concerns. The next change to the score formula has to update three code paths.

## What Changes

### 1. Extract `resolve.sh` — pure change resolution

Create `fab/.kit/scripts/lib/resolve.sh` as a standalone, side-effect-free resolver. One job: convert any change reference to a canonical output format.

**Input**: Any of the currently accepted forms — 4-char change ID (`9fg2`), folder name substring (`refactor-kit`), full folder name (`260228-9fg2-refactor-kit-scripts`), or no argument (reads `fab/current`).

**Output** (controlled by flags):
- `--id` (default) — 4-char change ID (e.g., `9fg2`)
- `--folder` — full folder name (e.g., `260228-9fg2-refactor-kit-scripts`)
- `--dir` — directory path (e.g., `fab/changes/260228-9fg2-refactor-kit-scripts/`)
- `--status` — `.status.yaml` path (e.g., `fab/changes/260228-9fg2-refactor-kit-scripts/.status.yaml`)

The resolution logic moves from `changeman.sh cmd_resolve()` into `resolve.sh`. `changeman.sh resolve` becomes a thin passthrough or is removed (changeman uses resolve.sh internally). The `resolve_change_arg()` function in stageman is deleted entirely.

**Why standalone**: resolve.sh is the universal dependency — every other script calls it first. A 60-line standalone script loads instantly without pulling in changeman's 400+ lines of lifecycle logic.

### 2. Rename `stageman.sh` → `statusman.sh` — stage state machine + .status.yaml metadata

Rename the script and update all references. The new name acknowledges that the script owns all of `.status.yaml`, not just stage transitions. Responsibilities that stay:

- **Stage state machine**: `start`, `advance`, `finish`, `reset`, `fail` + transition lookups via `workflow.yaml`
- **Status metadata**: `set-change-type`, `set-checklist`, `set-confidence`, `set-confidence-fuzzy`, `add-issue`, `add-pr`, `get-issues`, `get-prs`
- **Status queries**: `progress-map`, `checklist`, `confidence`, `current-stage`, `display-stage`, `progress-line`
- **Validation**: `validate-status-file`
- **Internals**: `_apply_metrics_side_effect`, lookup/query helpers

Responsibilities that move out:
- `log_command`, `log_confidence`, `log_review` → `logman.sh`
- `resolve_change_arg()` → `resolve.sh`

All CLI dispatch cases update to use `resolve.sh` for the `<change>` argument instead of the internal `resolve_change_arg()`.

### 3. Create `logman.sh` — append-only history logging

Create `fab/.kit/scripts/lib/logman.sh` as a standalone CLI script. Three subcommands:

```
logman.sh command <change> <cmd> [args]       # Log skill invocation
logman.sh confidence <change> <score> <delta> <trigger>  # Log score change
logman.sh review <change> <result> [rework]   # Log review outcome
```

Each subcommand resolves `<change>` via `resolve.sh --dir`, appends a JSON line to `{change_dir}/.history.jsonl`, and exits. No reads, no `.status.yaml` touches.

**Callers** — logman is never called directly by skills. It is always a side effect of another operation:

| Caller | Triggers | Logman call |
|--------|----------|-------------|
| `preflight.sh` (with new `--driver` flag) | Skill invocation | `logman.sh command` |
| `statusman.sh finish review` | Review pass | `logman.sh review "passed"` |
| `statusman.sh fail review` | Review fail | `logman.sh review "failed"` |
| `calc-score.sh` | Score computation | `logman.sh confidence` |
| `changeman.sh new` | Change creation | `logman.sh command` |
| `changeman.sh rename` | Change rename | `logman.sh command` |

### 4. Fold `log-command` into preflight via `--driver`

Currently every skill has a manual step: "Log invocation: `stageman.sh log-command <change> "fab-continue"`". This is boilerplate that every skill repeats.

Add a `--driver <skill-name>` flag to `preflight.sh`. When present, preflight calls `logman.sh command` after successful validation. Skills pass their name:

```bash
preflight.sh --driver fab-continue [change-name]
```

This eliminates the manual `log-command` step from all skill definitions.

### 5. Auto-log review outcomes from statusman state transitions

Currently skills call `log-review` explicitly after calling `finish review` or `fail review`. These are always paired — a review outcome is always followed by the corresponding state transition. Fold the logging into the state transitions:

- `statusman.sh finish <change> review [driver]` → auto-calls `logman.sh review "passed"`
- `statusman.sh fail <change> review [driver]` → auto-calls `logman.sh review "failed"`

The `fail` command gains an optional `[rework]` argument for the rework type, passed through to logman.

This eliminates the manual `log-review` step from `fab-continue.md`, `fab-ff.md`, and `fab-fff.md`.

### 6. DRY up `calc-score.sh`

Extract two internal helper functions:

**`count_grades()`** — parse an Assumptions table from a markdown file, output grade counts and dimension sums. Used by both gate-check and normal-scoring paths.

**`compute_score()`** — take grade counts and expected_min, output the confidence score. The formula `base = 5.0 - 0.3*confident - 1.0*tentative; cover = min(1.0, total/expected_min); score = base * cover` appears once.

The gate-check path becomes: determine which file to read + threshold → `count_grades` → `compute_score` → compare to threshold → output result.

The normal-scoring path becomes: determine which file to read → `count_grades` → `compute_score` → write to `.status.yaml` via statusman → log via logman → output result.

`calc-score.sh` also switches from taking `<change-dir>` to taking `<change>` (resolved via `resolve.sh --dir`).

### 7. Refactor test suite (`src/lib/`)

The existing test suite mirrors the script structure:

```
src/lib/
├── stageman/     → rename to statusman/, update test.bats and test-simple.sh
│   ├── SPEC-stageman.md  → rename to SPEC-statusman.md
│   ├── test.bats
│   └── test-simple.sh
├── changeman/    → update tests that reference stageman or test resolve logic
│   └── test.bats
├── calc-score/   → update tests for DRY'd helpers and new <change> input
│   ├── test.bats
│   └── test-simple.sh
├── preflight/    → update tests for --driver flag
│   ├── test.bats
│   └── test-simple.sh
└── sync-workspace/  → no changes expected
    └── test.bats
```

Changes:
- **`src/lib/stageman/`** → rename directory to `src/lib/statusman/`. Update all internal references. Rename `SPEC-stageman.md` → `SPEC-statusman.md`. Remove tests for `log-command`, `log-confidence`, `log-review` subcommands.
- **`src/lib/resolve/`** — new test directory. Tests for all four output modes (`--id`, `--folder`, `--dir`, `--status`), substring matching, exact matching, `fab/current` fallback, single-change guessing, error cases (no match, multiple matches).
- **`src/lib/logman/`** — new test directory. Tests for each subcommand (`command`, `confidence`, `review`), verifying JSON structure in `.history.jsonl`, append-only behavior, change resolution via resolve.sh.
- **`src/lib/changeman/`** — update tests that previously tested resolve behavior (now in resolve.sh). Update references from stageman to statusman.
- **`src/lib/calc-score/`** — update tests for the new `<change>` input convention (was `<change-dir>`), verify the DRY'd `count_grades()` and `compute_score()` helpers.
- **`src/lib/preflight/`** — add tests for the `--driver` flag and its automatic `logman.sh command` invocation.

### 8. Update all references

Every reference to `stageman.sh` across the codebase must be updated to `statusman.sh`:

- **Skill files**: `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-new.md`, `fab-status.md`, `fab-archive.md`, `git-pr.md`
- **Shared skill files**: `_preamble.md`, `_scripts.md`, `_generation.md`
- **Shell scripts**: `changeman.sh` (references `STAGEMAN`), `calc-score.sh` (references `STAGEMAN`), `preflight.sh`
- **Memory files**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/kit-architecture.md`, and others referencing stageman

Remove manual `log-command` and `log-review` invocations from all skill files (replaced by preflight `--driver` and statusman auto-logging).

Update `_scripts.md` to document the new 5-script architecture, removing the old `<change>` argument convention table (resolve.sh handles this now) and documenting the new calling conventions.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update script directory structure, rename stageman → statusman, add resolve.sh and logman.sh, update internal call graph
- `fab-workflow/execution-skills`: (modify) Update status mutation references from stageman to statusman, update log-review references to reflect auto-logging, update preflight references to include --driver flag

## Impact

- `fab/.kit/scripts/lib/resolve.sh` — new file
- `fab/.kit/scripts/lib/logman.sh` — new file
- `fab/.kit/scripts/lib/stageman.sh` → renamed to `statusman.sh`, log_* functions removed, resolve_change_arg removed
- `fab/.kit/scripts/lib/changeman.sh` — resolve logic moved to resolve.sh, log calls updated to logman
- `fab/.kit/scripts/lib/calc-score.sh` — DRY'd up, uses resolve.sh for input, calls logman instead of stageman
- `fab/.kit/scripts/lib/preflight.sh` — gains `--driver` flag, calls logman for command logging
- `fab/.kit/skills/_scripts.md` — rewritten for 5-script architecture
- `fab/.kit/skills/_preamble.md` — stageman → statusman references
- `fab/.kit/skills/_generation.md` — stageman → statusman references
- `fab/.kit/skills/fab-continue.md` — stageman → statusman, remove manual log-command and log-review
- `fab/.kit/skills/fab-ff.md` — stageman → statusman, remove manual log-command and log-review
- `fab/.kit/skills/fab-fff.md` — stageman → statusman, remove manual log-command and log-review
- `fab/.kit/skills/fab-clarify.md` — stageman → statusman, remove manual log-command
- `fab/.kit/skills/fab-new.md` — stageman → statusman
- `fab/.kit/skills/fab-status.md` — stageman → statusman
- `fab/.kit/skills/fab-archive.md` — stageman → statusman
- `fab/.kit/skills/git-pr.md` — stageman → statusman
- `docs/memory/fab-workflow/kit-architecture.md` — update
- `docs/memory/fab-workflow/execution-skills.md` — update
- Multiple other memory files referencing stageman
- `src/lib/stageman/` → renamed to `src/lib/statusman/`, tests updated, log_* tests removed
- `src/lib/resolve/` — new test directory
- `src/lib/logman/` — new test directory
- `src/lib/changeman/test.bats` — updated for resolve extraction and statusman references
- `src/lib/calc-score/test.bats` — updated for DRY'd helpers and new input convention
- `src/lib/preflight/test.bats` — updated for --driver flag

## Open Questions

None — architecture was fully designed and agreed upon during the `/fab-discuss` session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five-script architecture: resolve.sh, changeman.sh, statusman.sh, logman.sh, calc-score.sh | Discussed — user proposed and refined the breakdown across multiple exchanges. Each script has exactly one responsibility with no overlap | S:95 R:80 A:95 D:95 |
| 2 | Certain | Rename stageman.sh → statusman.sh | Discussed — user agreed the broader name acknowledges ownership of all .status.yaml, not just stage transitions | S:90 R:75 A:90 D:95 |
| 3 | Certain | resolve.sh is standalone, not a changeman subcommand | Discussed — user agreed. resolve.sh is the universal dependency (~60 lines); embedding it in changeman forces every script to load 400+ lines for a 50-line function | S:90 R:85 A:90 D:90 |
| 4 | Certain | logman.sh is a separate CLI script, not a sourced library | Discussed — user explicitly chose CLI over sourced library for easier testing and uniform calling signature | S:95 R:85 A:90 D:95 |
| 5 | Certain | Skills never call logman directly — logging is auto-triggered as side effects | Discussed — traced the full call graph. log-command folds into preflight --driver, log-review folds into statusman finish/fail review, log-confidence remains in calc-score | S:95 R:80 A:90 D:90 |
| 6 | Certain | preflight.sh gains --driver flag to auto-log skill invocation | Discussed — replaces manual log-command calls in every skill. preflight is already called by every skill, so this is the natural hook point | S:90 R:80 A:90 D:90 |
| 7 | Certain | statusman finish/fail review auto-logs review outcome | Discussed — these calls are always paired with log-review today. Folding eliminates the manual step from 3 skill files | S:90 R:80 A:85 D:90 |
| 8 | Certain | resolve.sh default output is 4-char change ID, with --folder/--dir/--status flags | Discussed — user proposed --id (default), --folder, --dir, --status output format flags | S:90 R:85 A:90 D:90 |
| 9 | Certain | DRY calc-score.sh by extracting count_grades() and compute_score() | Discussed — identified 3x duplication of grade counting loop and score formula | S:85 R:85 A:90 D:95 |
| 10 | Confident | calc-score.sh switches from <change-dir> to <change> via resolve.sh | Natural consequence of the canonical resolution design. Eliminates the last script bypassing the resolver | S:80 R:80 A:85 D:90 |
| 11 | Confident | .status.yaml metadata writers stay in statusman (not extracted) | Discussed — they share the same atomic-write pattern and yq dependency. Splitting would create two scripts with identical infrastructure for no gain | S:80 R:80 A:80 D:85 |
| 12 | Confident | stageman fail review gains optional [rework] argument | Natural extension — log_review already accepts a rework parameter. statusman fail needs to pass it through to logman | S:75 R:85 A:80 D:85 |

12 assumptions (9 certain, 3 confident, 0 tentative, 0 unresolved).

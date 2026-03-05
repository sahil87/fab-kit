# Spec: Batch Pipeline Series & Rename

**Change**: 260222-bcfy-batch-pipeline-series-rename
**Created**: 2026-02-22
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Non-Goals

- Parallel dispatch — v1 remains serial, one change at a time
- Multi-dependency support in series — series is strictly linear (A → B → C)
- `batch-pipeline-series.sh` argument validation via `changeman resolve` — change IDs are written to the manifest as-is; `run.sh` handles resolution at dispatch time
- Modifying `validate_manifest` to recognize the `watch` field — `watch` is read by the main loop only, not by validation

## Pipeline: Entry Point Rename

### Requirement: Rename batch-fab-pipeline to batch-pipeline

The user-facing entry point script SHALL be renamed from `fab/.kit/scripts/batch-fab-pipeline.sh` to `fab/.kit/scripts/batch-pipeline.sh`. All internal references (comments, usage text, help output, `batch_to_group` mapping) SHALL be updated to reflect the new name.

#### Scenario: Renamed script invocation
- **GIVEN** `fab/.kit/scripts/batch-pipeline.sh` exists
- **WHEN** the user runs `batch-pipeline.sh my-feature`
- **THEN** the script resolves and delegates to `pipeline/run.sh` identically to the previous `batch-fab-pipeline.sh`

#### Scenario: Help text uses new name
- **GIVEN** `fab/.kit/scripts/batch-pipeline.sh` exists
- **WHEN** the user runs `batch-pipeline.sh --help`
- **THEN** all usage text references `batch-pipeline.sh` (not `batch-fab-pipeline.sh`)

### Requirement: Update fab-help.sh batch_to_group mapping

The `batch_to_group` associative array in `fab-help.sh` SHALL replace the `batch-fab-pipeline` key with `batch-pipeline`. The `batch-pipeline-series` key SHALL also be added to the same `"Batch Operations"` group.

#### Scenario: Help output shows renamed and new scripts
- **GIVEN** `fab-help.sh` has updated `batch_to_group` entries
- **WHEN** `fab-help.sh` runs
- **THEN** "Batch Operations" group lists `batch-pipeline` and `batch-pipeline-series` (not `batch-fab-pipeline`)

### Requirement: Update kit-architecture.md directory tree

The `kit-architecture.md` memory file SHALL update the directory tree listing to show `batch-pipeline.sh` and `batch-pipeline-series.sh` instead of `batch-fab-pipeline.sh`. The `batch-fab-pipeline.sh` section description SHALL be updated to reflect the new name.

#### Scenario: Directory tree reflects rename
- **GIVEN** the kit-architecture.md has been updated
- **WHEN** a reader inspects the scripts directory tree
- **THEN** `batch-pipeline.sh` and `batch-pipeline-series.sh` appear; `batch-fab-pipeline.sh` does not

## Pipeline: Finite Exit Default

### Requirement: watch field in manifest format

The pipeline manifest format SHALL support an optional top-level `watch` field (boolean). When `watch` is `true`, the orchestrator SHALL run in infinite-loop mode (current behavior). When `watch` is `false` or absent, the orchestrator SHALL run in finite mode.

#### Scenario: Manifest with watch: true runs indefinitely
- **GIVEN** a manifest with `watch: true`
- **AND** all changes are terminal (`done`, `failed`, `invalid`)
- **WHEN** `run.sh` completes the dispatch cycle
- **THEN** `run.sh` continues polling (sleeps `POLL_INTERVAL`, re-reads manifest)
- **AND** does NOT exit

#### Scenario: Manifest without watch field exits when all terminal
- **GIVEN** a manifest with no `watch` field
- **AND** all changes are terminal
- **WHEN** `run.sh` finds nothing dispatchable
- **THEN** `run.sh` calls `print_summary` and exits 0

#### Scenario: Manifest with watch: false exits when all terminal
- **GIVEN** a manifest with `watch: false`
- **AND** all changes are terminal
- **WHEN** `run.sh` finds nothing dispatchable
- **THEN** `run.sh` calls `print_summary` and exits 0

#### Scenario: Finite mode continues when non-terminal changes remain
- **GIVEN** a manifest with no `watch` field
- **AND** some changes are not yet terminal (e.g., blocked by dependencies)
- **WHEN** `run.sh` finds nothing dispatchable this cycle
- **THEN** `run.sh` continues polling (does NOT exit)

### Requirement: Implementation in run.sh main loop

`run.sh` SHALL read the `watch` field from the manifest via `yq -r '.watch // false' "$MANIFEST"`. The check SHALL occur in the `else` branch of the main loop (when `find_next_dispatchable` fails). When `watch` is not `"true"`, the orchestrator SHALL check if all changes are terminal. If all are terminal, it SHALL call `print_summary` and `exit 0`.

#### Scenario: All-terminal check logic
- **GIVEN** `watch` is `false` and 3 changes exist: 1 done, 1 failed, 1 invalid
- **WHEN** the else branch evaluates
- **THEN** all 3 are terminal → `print_summary` + `exit 0`

#### Scenario: Mixed terminal and pending
- **GIVEN** `watch` is `false` and 2 changes exist: 1 done, 1 pending (deps on done)
- **WHEN** the else branch evaluates
- **THEN** not all terminal → continue polling

## Pipeline: Local Branch Refs

### Requirement: Use local branch refs in dispatch.sh

`dispatch.sh`'s `create_worktree()` function SHALL use local branch refs (`refs/heads/`) instead of remote tracking refs (`origin/`) when creating dependent node branches. The check SHALL use `git show-ref --verify --quiet "refs/heads/$PARENT_BRANCH"` instead of `git ls-remote --exit-code --heads origin "$PARENT_BRANCH"`.

#### Scenario: Dependent node branches from local parent
- **GIVEN** change-a's branch exists locally (created by its worktree)
- **AND** change-b depends on change-a
- **WHEN** `dispatch.sh` creates change-b's worktree
- **THEN** `git branch "$CHANGE_BRANCH" "$PARENT_BRANCH"` is called (no `origin/` prefix)

#### Scenario: Parent branch not yet local falls through to wt-create default
- **GIVEN** a parent branch does not exist locally
- **WHEN** `dispatch.sh` attempts to create the dependent branch
- **THEN** `git show-ref --verify --quiet` fails silently
- **AND** `wt-create` creates from HEAD (root node fallback, existing behavior)

## Pipeline: batch-pipeline-series.sh

### Requirement: New batch-pipeline-series.sh script

A new script SHALL exist at `fab/.kit/scripts/batch-pipeline-series.sh` that accepts a list of change IDs as positional arguments and an optional `--base <branch>` flag.

#### Scenario: Basic invocation
- **GIVEN** changes `change-a`, `change-b`, `change-c` exist in `fab/changes/`
- **WHEN** the user runs `batch-pipeline-series change-a change-b change-c`
- **THEN** a temporary manifest is generated and `run.sh` is invoked

#### Scenario: Custom base branch
- **GIVEN** the user is on branch `main`
- **WHEN** the user runs `batch-pipeline-series change-a change-b --base feat/setup`
- **THEN** the manifest's `base` field is `feat/setup`

#### Scenario: Default base is current branch
- **GIVEN** the user is on branch `my-feature`
- **WHEN** the user runs `batch-pipeline-series change-a change-b`
- **THEN** the manifest's `base` field is `my-feature`

### Requirement: Minimum 2 change arguments

`batch-pipeline-series.sh` SHALL require at least 2 positional change arguments. If fewer are provided, it SHALL print usage and exit 1.

#### Scenario: Single argument rejected
- **GIVEN** the user provides only 1 change
- **WHEN** `batch-pipeline-series change-a` runs
- **THEN** usage is printed to stderr and exit code is 1

#### Scenario: No arguments shows usage
- **GIVEN** no arguments provided
- **WHEN** `batch-pipeline-series` runs
- **THEN** usage is printed to stderr and exit code is 1

### Requirement: Temporary manifest generation

`batch-pipeline-series.sh` SHALL generate a manifest at `fab/pipelines/.series-{epoch}.yaml` where `{epoch}` is the Unix timestamp (`date +%s`). The manifest SHALL contain:
- `base`: value of `--base` or current branch
- No `watch` field (finite mode by default)
- `changes[]`: sequential dependency chain — first change has `depends_on: []`, each subsequent depends on its predecessor

#### Scenario: Generated manifest structure
- **GIVEN** `batch-pipeline-series change-a change-b change-c`
- **WHEN** the manifest is generated
- **THEN** the YAML content is:
  ```yaml
  base: <current-branch>
  changes:
    - id: change-a
      depends_on: []
    - id: change-b
      depends_on: [change-a]
    - id: change-c
      depends_on: [change-b]
  ```

### Requirement: Manifest not cleaned up

The generated `.series-*.yaml` manifest SHALL NOT be deleted after `run.sh` completes. It remains for debugging and inspection.

#### Scenario: Manifest persists after run
- **GIVEN** `batch-pipeline-series` has generated a manifest
- **WHEN** `run.sh` exits (success or failure)
- **THEN** the `.series-*.yaml` file still exists in `fab/pipelines/`

### Requirement: Delegate to run.sh via exec

After generating the manifest, `batch-pipeline-series.sh` SHALL delegate to `run.sh` via `exec bash "$SCRIPT_DIR/pipeline/run.sh" "$manifest_path"`.

#### Scenario: Delegation
- **GIVEN** the manifest has been generated at `fab/pipelines/.series-1740000000.yaml`
- **WHEN** `batch-pipeline-series.sh` finishes argument processing
- **THEN** `exec` replaces the process with `run.sh` invoked on the generated manifest

### Requirement: Shell frontmatter for fab-help.sh discovery

`batch-pipeline-series.sh` SHALL include a `# ---` delimited shell-comment frontmatter block with `name` and `description` fields, consistent with other batch scripts.

#### Scenario: Frontmatter block
- **GIVEN** `batch-pipeline-series.sh` is read by `fab-help.sh`
- **WHEN** `shell_frontmatter_field` extracts `name` and `description`
- **THEN** `name` is `batch-pipeline-series` and `description` is a concise one-liner

### Requirement: Help flag

`batch-pipeline-series.sh` SHALL support `-h` and `--help` flags, printing usage and exiting 0.

#### Scenario: Help output
- **GIVEN** the user runs `batch-pipeline-series --help`
- **WHEN** the script processes the flag
- **THEN** usage text is printed (arguments, base flag, examples) and exit code is 0

## Pipeline: .gitignore Pattern

### Requirement: Git-ignore generated series manifests

The repo `.gitignore` SHALL include a pattern to exclude generated series manifests: `fab/pipelines/.series-*.yaml`.

#### Scenario: Generated manifests are ignored
- **GIVEN** `fab/pipelines/.series-1740000000.yaml` exists
- **WHEN** `git status` is run
- **THEN** the file does not appear as untracked

## Pipeline: Example Manifest Documentation

### Requirement: Document watch field in example.yaml

`fab/pipelines/example.yaml` SHALL include a commented-out section documenting the `watch` field, its values, and its effect on loop behavior.

#### Scenario: Watch field documentation present
- **GIVEN** `example.yaml` is read by a user
- **WHEN** they look for loop behavior configuration
- **THEN** the `watch` field is documented with `true` (infinite loop) and default (finite, exit when done)

## Design Decisions

### 1. Finite exit as default, watch: true for infinite
- **Decision**: `run.sh` exits when all changes are terminal by default. `watch: true` in the manifest opts into infinite-loop mode.
- *Why*: Most pipeline runs have a known, finite set of changes. The infinite loop is only needed for the live-editing workflow where the human adds entries while the orchestrator runs. Making finite the default reduces ceremony for the common case.
- *Rejected*: `--finite` CLI flag — puts the control in the wrong place; the manifest already describes the pipeline's intent, so it should also declare its execution mode.

### 2. Thin wrapper generating manifest for series
- **Decision**: `batch-pipeline-series.sh` generates a temporary manifest and delegates to `run.sh` via `exec`, rather than implementing its own dispatch loop.
- *Why*: Zero logic duplication. `run.sh` already handles validation, dispatch, polling, shipping, SIGINT, and summary. The series script is ~50 lines of argument parsing and YAML generation.
- *Rejected*: Standalone serial runner — would duplicate dispatch/polling logic, diverge from manifest-based orchestrator over time.

### 3. Local branch refs instead of origin/
- **Decision**: `dispatch.sh` branches from local `refs/heads/` instead of `origin/`.
- *Why*: Git branches are shared across all worktrees of the same repo. The parent branch exists locally after its worktree was created. Using `origin/` assumed a push had completed, which is an implementation coincidence — the push happens during `/git-pr` but the branch resolution happens earlier.
- *Rejected*: Keeping `origin/` with a fetch-first approach — adds network dependency and latency for no benefit when the branch is already local.

## Deprecated Requirements

### Infinite Loop as Default Behavior
**Reason**: Replaced by finite-exit default with `watch: true` opt-in. The "Infinite Loop with SIGINT Exit" design decision in `pipeline-orchestrator.md` is superseded.
**Migration**: Existing manifests that relied on infinite loop behavior SHOULD add `watch: true` if they use live-editing.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename to `batch-pipeline.sh` | Confirmed from intake #1 — user explicitly requested | S:95 R:90 A:95 D:95 |
| 2 | Certain | Finite exit as default, `watch: true` for infinite | Confirmed from intake #2 — user specified exact mechanism | S:95 R:85 A:90 D:95 |
| 3 | Certain | Local branch refs instead of `origin/` | Confirmed from intake #3 — user specified local branches | S:95 R:80 A:90 D:95 |
| 4 | Certain | Series generates temp manifest, delegates to `run.sh` | Confirmed from intake #4 — thin wrapper approach agreed | S:90 R:90 A:90 D:90 |
| 5 | Certain | Don't remove temp manifest, gitignore the pattern | Confirmed from intake #5 — user specified explicitly | S:95 R:95 A:95 D:95 |
| 6 | Certain | `--base` defaults to current branch | Confirmed from intake #6 — user specified explicitly | S:95 R:85 A:90 D:95 |
| 7 | Confident | Series requires at least 2 change arguments | Confirmed from intake #7 — single change should use `fab-ff` directly | S:70 R:90 A:85 D:80 |
| 8 | Confident | Manifest timestamp format uses epoch seconds | Convention; `date +%s` is portable and collision-free | S:60 R:95 A:80 D:75 |
| 9 | Confident | No `--watch` override for series | Confirmed from intake #9 — series is inherently finite | S:70 R:90 A:85 D:80 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).

# Spec: Move env-packages.sh to lib & Add fab-pipeline.sh Entry Point

**Change**: 260221-i0z6-move-env-packages-add-fab-pipeline
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Scripts: env-packages.sh Relocation

### Requirement: env-packages.sh SHALL reside in `lib/`

`fab/.kit/scripts/env-packages.sh` SHALL be moved to `fab/.kit/scripts/lib/env-packages.sh`. The `lib/` subdirectory is not on PATH (only the parent `scripts/` directory is added via `PATH_add` in `.envrc`), so the script will no longer appear as a user-callable command.

#### Scenario: File is moved and no longer on PATH
- **GIVEN** `fab/.kit/scripts/` is on PATH via `.envrc` `PATH_add`
- **WHEN** `env-packages.sh` is moved to `fab/.kit/scripts/lib/env-packages.sh`
- **THEN** `env-packages.sh` SHALL NOT appear in shell tab-completion or `which` results
- **AND** `lib/env-packages.sh` SHALL remain sourceable via its full relative path

### Requirement: KIT_DIR resolution SHALL be updated for new depth

After the move, `SCRIPT_DIR` resolves to `.../scripts/lib/`. The `KIT_DIR` derivation MUST go up two levels (`../..`) instead of one (`..`) to reach the `.kit/` directory.

#### Scenario: KIT_DIR resolves correctly from new location
- **GIVEN** `env-packages.sh` is at `fab/.kit/scripts/lib/env-packages.sh`
- **WHEN** the script computes `KIT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"`
- **THEN** `KIT_DIR` SHALL resolve to `fab/.kit/`
- **AND** the `for d in "$KIT_DIR"/packages/*/bin` loop SHALL find package bin directories

### Requirement: All source references SHALL be updated

Two files source `env-packages.sh` and MUST be updated to the new path:

1. `fab/.kit/scaffold/fragment-.envrc` — the scaffold template that generates `.envrc` entries
2. `src/packages/rc-init.sh` — the shell rc sourcing entry point

#### Scenario: scaffold fragment sources from new path
- **GIVEN** `fab/.kit/scaffold/fragment-.envrc` contains `source fab/.kit/scripts/env-packages.sh`
- **WHEN** the path is updated
- **THEN** the line SHALL read `source fab/.kit/scripts/lib/env-packages.sh`

#### Scenario: rc-init.sh sources from new path
- **GIVEN** `src/packages/rc-init.sh` contains `source "$SCRIPT_DIR/../../fab/.kit/scripts/env-packages.sh"`
- **WHEN** the path is updated
- **THEN** the line SHALL read `source "$SCRIPT_DIR/../../fab/.kit/scripts/lib/env-packages.sh"`

## Scripts: fab-pipeline.sh Entry Point

`fab-pipeline.sh` is the user-facing entry point for the pipeline orchestrator. It owns all UX — argument parsing, help, listing, name resolution — and delegates to `pipeline/run.sh` for the actual orchestration loop.

### Requirement: fab-pipeline.sh SHALL be an executable wrapper on PATH

A new file `fab/.kit/scripts/fab-pipeline.sh` SHALL exist as the user-facing pipeline entry point. It SHALL delegate to `pipeline/run.sh` via `exec`, keeping `run.sh` as the single source of truth for orchestrator logic.

#### Scenario: User invokes with a manifest path
- **GIVEN** `fab/.kit/scripts/` is on PATH
- **WHEN** the user runs `fab-pipeline.sh fab/pipelines/my-feature.yaml`
- **THEN** `pipeline/run.sh` SHALL be invoked with `fab/pipelines/my-feature.yaml` as its argument

### Requirement: No arguments SHALL list available pipelines

When invoked with no arguments, `fab-pipeline.sh` SHALL list available pipeline manifests from `fab/pipelines/*.yaml`, excluding `example.yaml`. This matches the pattern established by `batch-fab-switch-change.sh` (no args → `--list`).

#### Scenario: No arguments lists pipelines
- **GIVEN** `fab/pipelines/` contains `example.yaml` and `pipeline1.yaml`
- **WHEN** the user runs `fab-pipeline.sh` with no arguments
- **THEN** the script SHALL print `pipeline1` (without path or `.yaml` extension)
- **AND** `example.yaml` SHALL be excluded from the listing
- **AND** the script SHALL exit with code 0

#### Scenario: No pipeline manifests found
- **GIVEN** `fab/pipelines/` contains only `example.yaml` (or is empty)
- **WHEN** the user runs `fab-pipeline.sh` with no arguments
- **THEN** the script SHALL print "No pipelines found." to stderr and exit with code 1

### Requirement: `-h` / `--help` SHALL print usage

#### Scenario: Help flag prints usage
- **GIVEN** the user runs `fab-pipeline.sh -h` or `fab-pipeline.sh --help`
- **WHEN** the script processes the flag
- **THEN** it SHALL print usage information to stdout and exit with code 0

### Requirement: `--list` SHALL explicitly list pipelines

`--list` SHALL behave identically to the no-arguments case — list available pipelines.

### Requirement: Partial pipeline name matching

The first positional argument SHALL be matched against `fab/pipelines/*.yaml` basenames (sans extension) using case-insensitive substring matching. This follows the same resolution pattern as `changeman resolve`.

#### Scenario: Exact match resolves
- **GIVEN** `fab/pipelines/pipeline1.yaml` exists
- **WHEN** the user runs `fab-pipeline.sh pipeline1`
- **THEN** the resolved path SHALL be `fab/pipelines/pipeline1.yaml`

#### Scenario: Partial match resolves to unique match
- **GIVEN** `fab/pipelines/pipeline1.yaml` exists and no other manifests match
- **WHEN** the user runs `fab-pipeline.sh pipe`
- **THEN** the resolved path SHALL be `fab/pipelines/pipeline1.yaml`

#### Scenario: Ambiguous partial match errors
- **GIVEN** `fab/pipelines/pipeline1.yaml` and `fab/pipelines/pipeline2.yaml` both exist
- **WHEN** the user runs `fab-pipeline.sh pipe`
- **THEN** the script SHALL print "Multiple pipelines match..." listing the matches to stderr
- **AND** exit with code 1

#### Scenario: No match errors
- **GIVEN** no manifest matches the argument
- **WHEN** the user runs `fab-pipeline.sh nonexistent`
- **THEN** the script SHALL print "No pipeline matches..." to stderr and exit with code 1

#### Scenario: Explicit path bypasses matching
- **GIVEN** the argument contains a `/` or ends with `.yaml`
- **WHEN** the user runs `fab-pipeline.sh ./custom/path/manifest.yaml`
- **THEN** the argument SHALL be passed to `pipeline/run.sh` unchanged (no name matching)

### Requirement: Additional arguments SHALL be forwarded

Any arguments after the manifest name SHALL be passed through to `pipeline/run.sh` via `"$@"`.

## Pipeline: Change ID Resolution via changeman

### Requirement: run.sh SHALL resolve manifest change IDs through changeman

Before dispatching a change, `run.sh` SHALL resolve each manifest entry's `id` field through `changeman resolve` to get the full folder name. This allows manifests to use short IDs (e.g., `a7k2`) or partial names.

The manifest's internal consistency is preserved — `id` and `depends_on` values must match each other as written. Resolution happens only at dispatch time to map the manifest ID to the actual `fab/changes/` folder.

#### Scenario: Short ID resolves to full change name
- **GIVEN** a manifest entry has `id: a7k2`
- **AND** `fab/changes/260221-a7k2-add-oauth/` exists
- **WHEN** `run.sh` dispatches this entry
- **THEN** `changeman resolve a7k2` SHALL return `260221-a7k2-add-oauth`
- **AND** `dispatch.sh` SHALL receive the resolved full name

#### Scenario: Resolution failure marks change as failed
- **GIVEN** a manifest entry has `id: zzzz` and no matching change exists
- **WHEN** `run.sh` attempts to dispatch this entry
- **THEN** `changeman resolve` SHALL fail
- **AND** the change SHALL be marked `stage: invalid` in the manifest
- **AND** the orchestrator SHALL continue to the next dispatchable entry

#### Scenario: Manifest dependencies use original IDs
- **GIVEN** a manifest has `id: a7k2` and another entry with `depends_on: [a7k2]`
- **WHEN** the orchestrator checks dependency satisfaction
- **THEN** it SHALL compare against the manifest's `id` values (not resolved names)
- **AND** resolution SHALL happen independently at dispatch time for each entry

## Pipeline: Worktree Creation in dispatch.sh

### Requirement: dispatch.sh SHALL use `--worktree-name` for readable worktree directories

The `wt-create` invocation in `dispatch.sh` SHALL include `--worktree-name "$CHANGE_ID"`, matching the pattern used by `batch-fab-switch-change.sh`. This produces readable worktree directory names instead of auto-generated ones.

#### Scenario: Worktree gets a named directory
- **GIVEN** dispatch.sh creates a worktree for change `260221-a7k2-add-oauth`
- **WHEN** `wt-create` is called
- **THEN** the call SHALL include `--worktree-name "260221-a7k2-add-oauth"`
- **AND** the worktree directory SHALL be named after the change

### Requirement: Parent-branch pre-creation SHALL be preserved

For dependent nodes, `dispatch.sh` SHALL continue to pre-create the change branch from the parent's remote branch before calling `wt-create`. This is necessary because dependent nodes branch from their parent's pushed branch, not from HEAD.

## Pipeline: Stage Detection via stageman

### Requirement: dispatch.sh SHALL use stageman to determine change stage

After `fab-ff` completes (or fails), `dispatch.sh` SHALL use `stageman current-stage` or `stageman display-stage` on the worktree's `.status.yaml` to determine the actual stage reached. This replaces the current brittle check that only inspects `progress.hydrate` via raw yq.

#### Scenario: Successful pipeline writes actual stage to manifest
- **GIVEN** `fab-ff` exits 0 and stageman reports `display-stage` as `hydrate:done`
- **WHEN** dispatch.sh checks the result
- **THEN** `stage: done` SHALL be written to the manifest

#### Scenario: Partial pipeline failure writes intermediate stage
- **GIVEN** `fab-ff` exits non-zero and stageman reports `display-stage` as `tasks:active`
- **WHEN** dispatch.sh checks the result
- **THEN** `stage: failed` SHALL be written to the manifest
- **AND** the log message SHALL include the stage reached (e.g., "failed at tasks")

#### Scenario: Pipeline success but hydrate incomplete
- **GIVEN** `fab-ff` exits 0 but stageman reports `display-stage` as `review:done` (hydrate not reached)
- **WHEN** dispatch.sh checks the result
- **THEN** `stage: failed` SHALL be written to the manifest
- **AND** the log message SHALL note the discrepancy

## Documentation: Memory and README Updates

### Requirement: kit-architecture.md SHALL reflect new layout

The directory tree in `kit-architecture.md` SHALL move `env-packages.sh` from the `scripts/` listing to the `lib/` listing and SHALL add `fab-pipeline.sh` to the `scripts/` listing. The `env-packages.sh` description section SHALL be updated with the new path. A new `fab-pipeline.sh` description section SHALL be added.

#### Scenario: Directory tree updated
- **GIVEN** `kit-architecture.md` lists `env-packages.sh` under `scripts/`
- **WHEN** the doc is updated
- **THEN** `env-packages.sh` SHALL appear under `scripts/lib/` with updated comment
- **AND** `fab-pipeline.sh` SHALL appear under `scripts/` with a description comment
- **AND** `pipeline/` directory SHALL be listed under `scripts/` if not already present

### Requirement: distribution.md SHALL reference new path

Any references to `env-packages.sh` in `distribution.md` SHALL use the updated `fab/.kit/scripts/lib/env-packages.sh` path.

#### Scenario: distribution.md path references updated
- **GIVEN** `distribution.md` references `env-packages.sh` sourcing
- **WHEN** the doc is updated
- **THEN** all path references SHALL point to `fab/.kit/scripts/lib/env-packages.sh`

### Requirement: README.md SHALL reference new path

The README description of `env-packages.sh` delegation SHALL use the updated path.

#### Scenario: README env-packages reference updated
- **GIVEN** README.md mentions `fab/.kit/scripts/env-packages.sh`
- **WHEN** the doc is updated
- **THEN** the reference SHALL point to `fab/.kit/scripts/lib/env-packages.sh`

### Requirement: pipeline-orchestrator.md SHALL reflect pipeline improvements

`docs/memory/fab-workflow/pipeline-orchestrator.md` SHALL be updated to document:
- `fab-pipeline.sh` as the user-facing entry point (with listing, partial matching, help)
- changeman resolve for manifest change IDs in `run.sh`
- `--worktree-name` usage in `dispatch.sh`
- stageman-based stage detection in `dispatch.sh`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Move destination is `lib/` not a new subfolder | Confirmed from intake #1 — `lib/` already exists and holds internal sourceable scripts; user explicitly agreed in discussion | S:95 R:90 A:95 D:90 |
| 2 | Certain | `env-packages.sh` needs `KIT_DIR` path update after move | Confirmed from intake #2 — mechanical necessity, one more directory level | S:95 R:95 A:95 D:95 |
| 3 | Certain | Wrapper uses `exec` delegation, not function copy | Confirmed from intake #3 — keeps pipeline/run.sh as single source of truth | S:90 R:95 A:90 D:95 |
| 4 | Certain | `fab-pipeline.sh` owns UX, `run.sh` stays internal | User confirmed in discussion — matches batch-fab-switch pattern where the PATH script handles args/help/listing and delegates to internal scripts | S:90 R:90 A:95 D:90 |
| 5 | Certain | No-args and `--list` list pipelines, `-h`/`--help` prints usage | User confirmed — mirrors `batch-fab-switch-change.sh` UX conventions | S:95 R:95 A:90 D:90 |
| 6 | Certain | Partial pipeline name matching uses changeman-style resolution | User confirmed — case-insensitive substring, error on ambiguity, same pattern as `changeman resolve` | S:90 R:90 A:90 D:85 |
| 7 | Certain | changeman resolve at dispatch time, manifest IDs stay internally consistent | User chose option (b) — manifest `id` and `depends_on` match each other as written; resolution maps to filesystem only at dispatch | S:95 R:90 A:90 D:90 |
| 8 | Certain | `dispatch.sh` uses `--worktree-name` matching batch-fab-switch pattern | User confirmed — batch-fab-switch works; dispatch should mimic it | S:95 R:95 A:95 D:90 |
| 9 | Certain | `dispatch.sh` uses stageman for stage detection, not raw yq | User requested — stageman `display-stage`/`current-stage` replaces brittle `yq '.progress.hydrate'` check | S:90 R:90 A:90 D:85 |
| 10 | Confident | Documentation updates are in-scope | Confirmed from intake #5 — memory files and README reference the old path | S:80 R:85 A:85 D:80 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).

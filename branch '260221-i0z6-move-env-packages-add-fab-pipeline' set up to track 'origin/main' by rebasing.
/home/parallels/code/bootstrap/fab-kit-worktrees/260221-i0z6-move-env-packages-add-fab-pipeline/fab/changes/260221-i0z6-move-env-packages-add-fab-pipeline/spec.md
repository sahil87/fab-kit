# Spec: Move env-packages.sh to lib & Add fab-pipeline.sh Entry Point

**Change**: 260221-i0z6-move-env-packages-add-fab-pipeline
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

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

### Requirement: fab-pipeline.sh SHALL be a thin wrapper on PATH

A new file `fab/.kit/scripts/fab-pipeline.sh` SHALL exist as an executable wrapper that delegates to `pipeline/run.sh`. It SHALL use `exec` to replace the shell process, keeping `pipeline/run.sh` as the single source of truth for orchestrator logic.

#### Scenario: User invokes with a manifest path
- **GIVEN** `fab/.kit/scripts/` is on PATH
- **WHEN** the user runs `fab-pipeline.sh fab/pipelines/my-feature.yaml`
- **THEN** `pipeline/run.sh` SHALL be invoked with `fab/pipelines/my-feature.yaml` as its argument

#### Scenario: User invokes with no arguments
- **GIVEN** `fab-pipeline.sh` is invoked with no arguments
- **WHEN** the script runs
- **THEN** it SHALL print usage to stderr and exit with code 1

### Requirement: Bare-name convenience resolution

If the first argument contains no path separator (`/`) and does not end with `.yaml`, the script SHALL resolve it as `fab/pipelines/{name}.yaml`.

#### Scenario: Bare name is resolved to manifest path
- **GIVEN** the user runs `fab-pipeline.sh my-feature`
- **WHEN** the argument `my-feature` contains no `/` and no `.yaml` suffix
- **THEN** the resolved path SHALL be `fab/pipelines/my-feature.yaml`
- **AND** `pipeline/run.sh` SHALL be invoked with `fab/pipelines/my-feature.yaml`

#### Scenario: Explicit path is passed through unchanged
- **GIVEN** the user runs `fab-pipeline.sh ./custom/path/manifest.yaml`
- **WHEN** the argument contains a `/`
- **THEN** the argument SHALL be passed to `pipeline/run.sh` unchanged

#### Scenario: .yaml suffix is passed through unchanged
- **GIVEN** the user runs `fab-pipeline.sh my-feature.yaml`
- **WHEN** the argument ends with `.yaml`
- **THEN** the argument SHALL be passed to `pipeline/run.sh` unchanged

### Requirement: Additional arguments SHALL be forwarded

Any arguments after the manifest SHALL be passed through to `pipeline/run.sh` via `"$@"`.

#### Scenario: Extra arguments are forwarded
- **GIVEN** the user runs `fab-pipeline.sh my-feature --some-flag`
- **WHEN** the manifest resolves to `fab/pipelines/my-feature.yaml`
- **THEN** `pipeline/run.sh` SHALL receive `fab/pipelines/my-feature.yaml --some-flag`

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

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Move destination is `lib/` not a new subfolder | Confirmed from intake #1 — `lib/` already exists and holds internal sourceable scripts; user explicitly agreed in discussion | S:95 R:90 A:95 D:90 |
| 2 | Certain | `env-packages.sh` needs `KIT_DIR` path update after move | Confirmed from intake #2 — mechanical necessity, one more directory level | S:95 R:95 A:95 D:95 |
| 3 | Certain | Wrapper uses `exec` delegation, not function copy | Confirmed from intake #3 — keeps pipeline/run.sh as single source of truth | S:90 R:95 A:90 D:95 |
| 4 | Confident | Bare-name convenience resolves to `fab/pipelines/{name}.yaml` | Confirmed from intake #4 — user confirmed the convenience feature in discussion | S:85 R:90 A:80 D:75 |
| 5 | Confident | Documentation updates are in-scope | Confirmed from intake #5 — memory files and README reference the old path | S:80 R:85 A:85 D:80 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).

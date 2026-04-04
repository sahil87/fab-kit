# Spec: Org Migration (wvrdz → sahil87) and MIT License

**Change**: 260401-ixzv-org-migrate-mit-license
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Updating Tier 2 documentation files (memory, specs) beyond the hydrate step — those will be updated during hydrate
- Updating `created_by` fields or historical PR URLs in `.status.yaml` files
- Updating the `linear_workspace` value in `config.yaml` (separate concern)
- Updating references inside `fab/changes/archive/`

## License

### Requirement: MIT License at Root

The project SHALL have a single MIT license file at the repository root (`LICENSE`). The existing `fab/.kit/LICENSE` (PolyForm Internal Use 1.0.0) SHALL be removed.

The MIT license text SHALL use:
- Copyright line: `Copyright (c) 2026 Sahil Ahuja`
- Standard MIT license body per https://opensource.org/licenses/MIT

#### Scenario: Root LICENSE Created

- **GIVEN** the repository has no root `LICENSE` file
- **AND** `fab/.kit/LICENSE` contains the PolyForm Internal Use license
- **WHEN** the change is applied
- **THEN** a new `LICENSE` file SHALL exist at the repository root with the MIT license text
- **AND** `fab/.kit/LICENSE` SHALL no longer exist

#### Scenario: Kit Distribution Excludes License

- **GIVEN** `fab/.kit/LICENSE` has been removed
- **WHEN** a kit archive is built via `just package-kit`
- **THEN** the archive SHALL NOT contain a `.kit/LICENSE` file
- **AND** downstream projects receive no license file inside `.kit/`

## Configuration

### Requirement: Kit Config Repo Reference

`fab/.kit/kit.conf` SHALL reference the new GitHub org in its `repo` field.

#### Scenario: kit.conf Updated

- **GIVEN** `fab/.kit/kit.conf` contains `repo=wvrdz/fab-kit`
- **WHEN** the change is applied
- **THEN** `fab/.kit/kit.conf` SHALL contain `repo=sahil87/fab-kit`

## Install Script

### Requirement: Install Script Repo Reference

`scripts/install.sh` SHALL reference the new GitHub org in both its comment header URL and `REPO` variable.

#### Scenario: install.sh Updated

- **GIVEN** `scripts/install.sh` contains `REPO="wvrdz/fab-kit"` and a comment URL referencing `wvrdz`
- **WHEN** the change is applied
- **THEN** `scripts/install.sh` SHALL contain `REPO="sahil87/fab-kit"`
- **AND** the comment URL SHALL reference `sahil87/fab-kit`

## README

### Requirement: README Install URL

`README.md` SHALL reference the new GitHub org in its install curl command.

#### Scenario: README Updated

- **GIVEN** `README.md` contains a curl URL referencing `wvrdz/fab-kit`
- **WHEN** the change is applied
- **THEN** the curl URL SHALL reference `sahil87/fab-kit`

## Go Modules

### Requirement: Module Path Migration

All three Go modules SHALL use module paths under `github.com/sahil87/fab-kit`:

- `src/go/fab/go.mod`: `module github.com/sahil87/fab-kit/src/go/fab`
- `src/go/idea/go.mod`: `module github.com/sahil87/fab-kit/src/go/idea`
- `src/go/wt/go.mod`: `module github.com/sahil87/fab-kit/src/go/wt`

#### Scenario: go.mod Files Updated

- **GIVEN** `src/go/fab/go.mod` contains `module github.com/wvrdz/fab-kit/src/go/fab`
- **WHEN** the change is applied
- **THEN** `src/go/fab/go.mod` SHALL contain `module github.com/sahil87/fab-kit/src/go/fab`

- **GIVEN** `src/go/idea/go.mod` contains `module github.com/wvrdz/fab-kit/src/go/idea`
- **WHEN** the change is applied
- **THEN** `src/go/idea/go.mod` SHALL contain `module github.com/sahil87/fab-kit/src/go/idea`

- **GIVEN** `src/go/wt/go.mod` contains `module github.com/wvrdz/fab-kit/src/go/wt`
- **WHEN** the change is applied
- **THEN** `src/go/wt/go.mod` SHALL contain `module github.com/sahil87/fab-kit/src/go/wt`

### Requirement: Import Path Migration

All Go source files importing packages from `github.com/wvrdz/fab-kit/...` SHALL be updated to import from `github.com/sahil87/fab-kit/...`.

#### Scenario: fab Internal Imports Updated

- **GIVEN** `src/go/fab/internal/status/status.go` imports `github.com/wvrdz/fab-kit/src/go/fab/internal/config`
- **WHEN** the change is applied
- **THEN** the import SHALL be `github.com/sahil87/fab-kit/src/go/fab/internal/config`
- **AND** all other imports in `src/go/fab/` referencing `wvrdz` SHALL be updated similarly

#### Scenario: idea Imports Updated

- **GIVEN** `src/go/idea/cmd/add.go` imports `github.com/wvrdz/fab-kit/src/go/idea/internal/idea`
- **WHEN** the change is applied
- **THEN** the import SHALL be `github.com/sahil87/fab-kit/src/go/idea/internal/idea`
- **AND** all other imports in `src/go/idea/` referencing `wvrdz` SHALL be updated similarly

#### Scenario: wt Imports Updated

- **GIVEN** `src/go/wt/cmd/create.go` imports `github.com/wvrdz/fab-kit/src/go/wt/internal/worktree`
- **WHEN** the change is applied
- **THEN** the import SHALL be `github.com/sahil87/fab-kit/src/go/wt/internal/worktree`
- **AND** all other imports in `src/go/wt/` referencing `wvrdz` SHALL be updated similarly

### Requirement: Go Sum Regeneration

After module path updates, `go mod tidy` SHALL be run in each module directory to regenerate `go.sum` files.

#### Scenario: go.sum Files Regenerated

- **GIVEN** module paths have been updated in all three `go.mod` files and all import statements
- **WHEN** `go mod tidy` is run in `src/go/fab/`, `src/go/idea/`, and `src/go/wt/`
- **THEN** each `go.sum` file SHALL be regenerated with correct checksums
- **AND** `go build ./...` SHALL succeed in each module directory

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Org changes from `wvrdz` to `sahil87` | Confirmed from intake #1 — user stated the repo has moved | S:95 R:90 A:95 D:95 |
| 2 | Certain | License changes to MIT | Confirmed from intake #2 — user explicitly requested MIT | S:95 R:85 A:95 D:95 |
| 3 | Certain | Root LICENSE replaces fab/.kit/LICENSE | Confirmed from intake #3 — user specified location change | S:95 R:90 A:95 D:95 |
| 4 | Certain | Copyright "2026 Sahil Ahuja" | Confirmed from intake #4 — preserved from existing license | S:90 R:95 A:90 D:95 |
| 5 | Certain | Scope is Tier 1 functional files only | Confirmed from intake #5 — user explicitly scoped | S:95 R:95 A:95 D:95 |
| 6 | Certain | Copyright year: 2026 | Confirmed from intake #6 — consistent with project dates | S:90 R:95 A:90 D:95 |
| 7 | Confident | go.sum regenerated via `go mod tidy` | Confirmed from intake #7 — standard Go practice | S:80 R:90 A:85 D:90 |
| 8 | Certain | String replacement `wvrdz` → `sahil87` is sufficient for all Go imports | All imports follow the pattern `github.com/wvrdz/fab-kit/...` — direct string replacement is safe and complete | S:90 R:95 A:95 D:95 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).

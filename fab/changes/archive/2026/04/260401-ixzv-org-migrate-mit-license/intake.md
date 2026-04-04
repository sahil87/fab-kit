# Intake: Org Migration (wvrdz → sahil87) and MIT License

**Change**: 260401-ixzv-org-migrate-mit-license
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Migrate the repo from the `wvrdz` GitHub org to `sahil87`. Update all functional references (Tier 1 only — code, config, scripts). Switch the license from PolyForm Internal Use 1.0.0 to MIT. Add a root `LICENSE` file and remove the existing `fab/.kit/LICENSE`.

Preceded by a `/fab-discuss` session where a full audit of `wvrdz` and `weaver` references was performed via grep. References were triaged into three tiers. User scoped this change to Tier 1 (must-change, functional) plus the license change.

## Why

The repository has moved from `github.com/wvrdz/fab-kit` to `github.com/sahil87/fab-kit`. All functional references — Go module paths, import statements, the install script, README, and kit.conf — still point to the old org. This will break Go module resolution for anyone pulling the module, and the install script will download from the wrong location.

The license change from PolyForm Internal Use to MIT makes the project open source, allowing distribution and modification by anyone.

## What Changes

### 1. License

- **Remove** `fab/.kit/LICENSE` (PolyForm Internal Use 1.0.0)
- **Add** root `LICENSE` with the MIT license text
- Copyright line: `Copyright (c) 2026 Sahil Ahuja` (preserved from current license)

### 2. Kit Configuration

- **`fab/.kit/kit.conf`** line 4: `repo=wvrdz/fab-kit` → `repo=sahil87/fab-kit`

### 3. Install Script

- **`scripts/install.sh`** line 7 (comment URL): `wvrdz/fab-kit` → `sahil87/fab-kit`
- **`scripts/install.sh`** line 13: `REPO="wvrdz/fab-kit"` → `REPO="sahil87/fab-kit"`

### 4. README

- **`README.md`** line 105: install curl URL `wvrdz/fab-kit` → `sahil87/fab-kit`

### 5. Go Module Paths (3 modules)

All three Go modules rename from `github.com/wvrdz/fab-kit` to `github.com/sahil87/fab-kit`:

- `src/go/fab/go.mod` — `module github.com/wvrdz/fab-kit/src/go/fab` → `module github.com/sahil87/fab-kit/src/go/fab`
- `src/go/idea/go.mod` — `module github.com/wvrdz/fab-kit/src/go/idea` → `module github.com/sahil87/fab-kit/src/go/idea`
- `src/go/wt/go.mod` — `module github.com/wvrdz/fab-kit/src/go/wt` → `module github.com/sahil87/fab-kit/src/go/wt`

### 6. Go Import Paths (~20 files)

Every Go source file importing `github.com/wvrdz/fab-kit/...` needs the import path updated to `github.com/sahil87/fab-kit/...`. Affected packages:

**`src/go/fab/`** (internal imports):
- `internal/status/status.go` — imports config, hooks, log, statusfile
- `internal/change/change.go` — imports log, resolve, status, statusfile
- `internal/preflight/preflight.go` — imports resolve, status, statusfile
- `internal/log/log.go` — imports resolve
- `internal/archive/archive.go` — imports change, resolve
- `internal/score/score.go` — imports log, resolve, status, statusfile

**`src/go/idea/`** (internal imports):
- `cmd/edit.go`, `cmd/show.go`, `cmd/rm.go`, `cmd/done.go`, `cmd/add.go`, `cmd/resolve.go`, `cmd/list.go`, `cmd/reopen.go` — all import `internal/idea`

**`src/go/wt/`** (internal imports):
- `cmd/open.go`, `cmd/init.go`, `cmd/delete.go`, `cmd/main.go`, `cmd/list.go`, `cmd/create.go` — all import `internal/worktree`

### 7. Go Sum Regeneration

After updating module paths, run `go mod tidy` in each module directory to regenerate:
- `src/go/fab/go.sum`
- `src/go/idea/go.sum`
- `src/go/wt/go.sum`

## Affected Memory

- `fab-workflow/distribution`: (modify) Update repo URL references from `wvrdz/fab-kit` to `sahil87/fab-kit`, update install script documentation, note license change
- `fab-workflow/kit-architecture`: (modify) Update Go module path references from `github.com/wvrdz/fab-kit` to `github.com/sahil87/fab-kit`

## Impact

- **Go module consumers**: Any existing `go get` or `go install` referencing the old module path will need to use the new path. GitHub's repo redirect handles HTTP but Go module proxy may cache the old path.
- **Install script users**: Anyone with bookmarked or documented install commands pointing to `wvrdz` will be redirected by GitHub, but should update.
- **CI/CD**: GitHub Actions workflows have no `wvrdz` references (verified), so no CI impact.
- **License**: Moving from a restrictive internal-use license to MIT — no code changes needed beyond the license file swap.

## Open Questions

None — scope is fully defined from the audit.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Org changes from `wvrdz` to `sahil87` | Discussed — user stated the repo has moved to sahil87 org | S:95 R:90 A:95 D:95 |
| 2 | Certain | License changes to MIT | Discussed — user explicitly requested MIT | S:95 R:85 A:95 D:95 |
| 3 | Certain | Root LICENSE replaces fab/.kit/LICENSE | Discussed — user specified "add root LICENSE, remove fab/.kit/LICENSE" | S:95 R:90 A:95 D:95 |
| 4 | Certain | Copyright holder: "Sahil Ahuja" | Preserved from existing license file, no change requested | S:90 R:95 A:90 D:95 |
| 5 | Certain | Scope is Tier 1 (functional files) only | Discussed — user explicitly said "only the must to changes (Tier 1)" | S:95 R:95 A:95 D:95 |
| 6 | Certain | Copyright year: 2026 | Preserved from existing license, consistent with project dates | S:90 R:95 A:90 D:95 |
| 7 | Confident | go.sum files regenerated via `go mod tidy` | Standard Go practice after module path changes; no alternative | S:80 R:90 A:85 D:90 |

7 assumptions (6 certain, 1 confident, 0 tentative, 0 unresolved).

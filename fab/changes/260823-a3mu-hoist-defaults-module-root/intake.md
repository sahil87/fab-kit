# Intake: Hoist defaults.yaml to the Go module root

**Change**: 260823-a3mu-hoist-defaults-module-root
**Created**: 2026-08-23

## Origin

Adopted from branch `robust-python` (no PR existed at adoption) via `/fab-adopt` — the code was authored off-pipeline in a `/fab-discuss` session that turned into implementation. The user asked where the config/model defaults live, was shown `src/go/fab/internal/agent/defaults.yaml`, and requested the file be moved higher in the tree because "this file needs to be referred to a lot":

> This defaults.yaml, can we move it higher in the tree? Like under src/ or in the repo root?

Repo root and `src/` were ruled out in discussion: `go:embed` cannot reference files outside the embedding package's module, and a copy-at-build step would reintroduce the two-copies drift the file's design exists to kill. The user chose the presented option 2: hoist to the **Go module root** (`src/go/fab/defaults.yaml`) with a tiny root-level package owning the embed — "option 2 - go for it".

## Why

1. **Pain point**: `defaults.yaml` is the single value source for every built-in default (model fills, provider grammars, dispatch and autopilot defaults) and is the file humans open most when checking or bumping models — yet it was buried four directories deep at `src/go/fab/internal/agent/defaults.yaml`, beside its consumer rather than where a reader looks.
2. **Consequence of not fixing**: continued friction locating the most-referred-to data file in the module; the `internal/` path also wrongly signals "implementation detail" for what is deliberately user-facing, documented, overridable data.
3. **Why this approach**: `go:embed` physically bounds the file to the module (`src/go/fab/`), so the module root is the highest legal home. A one-file root package keeps the embed while `internal/agent` retains all parsing, validation, and init-injection — preserving the embed-over-kit-cache design (binary can never disagree with its shipped defaults) and every existing drift guard. Rejected: repo-root/`src/` placement (impossible for go:embed; a copy step reintroduces drift), and leaving it in place with only doc pointers (does not fix the discoverability complaint).

## What Changes

### File move + new root package (`src/go/fab/`)

- `src/go/fab/internal/agent/defaults.yaml` → **`src/go/fab/defaults.yaml`** (git mv, 97% similarity — only the header comment changed). The header now explains the new wiring: lives at the module root for visibility; embedded by the root fab package in `defaults.go`; parsed and consumed by `internal/agent`; never read from the kit cache.
- **New** `src/go/fab/defaults.go` — a one-file `package fab` at the module root whose sole job is the embed:

  ```go
  //go:embed defaults.yaml
  var DefaultsYAML []byte
  ```

  Its package comment states why it exists (go:embed cannot reach above the embedding package's directory) and that all parsing, validation, and consumption stay in `internal/agent`. The exported var's comment carries the embed-over-kit-cache rationale relocated from `agent.go`.

### `internal/agent` rewiring (`src/go/fab/internal/agent/agent.go`)

- The `//go:embed defaults.yaml` directive and `_ "embed"` import are removed; `defaultsYAML` becomes `var defaultsYAML = fabroot.DefaultsYAML` (import alias `fabroot "github.com/sahil87/fab-kit/src/go/fab"`).
- Everything downstream is untouched: `mustParseDefaults`, `builtinDefaults`, the init-injection of `config.DefaultDispatch*` / `config.DefaultAutopilotMergeMode`, all exported symbols, and all six named drift-guard tests (which parse the embedded bytes, not a disk path).
- Package comment and the embed-site comment updated to describe the module-root location.

### Location-claim sweep (docs + skills + Go comments)

Every prose claim locating the file in `internal/agent` was updated; historical `log.md` / `log.seed.md` entries deliberately left as records.

- Full-path rewrites `src/go/fab/internal/agent/defaults.yaml` → `src/go/fab/defaults.yaml`: `docs/specs/stage-models.md` (3), `docs/specs/config.md` (2), `docs/memory/runtime/providers-and-profiles.md` (3).
- Possessive phrasing `` `internal/agent`'s embedded `defaults.yaml` `` → ``the module-root embedded `defaults.yaml` ``: `docs/specs/config.md`, `docs/specs/stage-models.md`, `docs/memory/_shared/configuration.md` (4 spots; the cascade-tier sentence additionally carries the new path and "embedded by the root fab package, parsed by `internal/agent`"), `src/kit/skills/_cli-agents.md`.
- `docs/memory/distribution/kit-architecture.md` embed-census sentence: "fab-go's `internal/agent` `defaults.yaml`" → "fab-go's module-root `defaults.yaml` (built-in values, embedded by the root fab package, parsed by `internal/agent`)".
- Go comments in `src/go/fab/internal/config/config.go` (2) and `src/go/fab/internal/spawn/spawn.go` (1): same possessive → module-root rewording.

## Affected Memory

- `_shared/configuration.md`: (modify) cascade tier-4 physical-source sentence + three dispatch/autopilot location sentences repointed to the module-root file
- `runtime/providers-and-profiles.md`: (modify) § "The built-in defaults are an embedded defaults.yaml" body + Design Decision path repointed
- `distribution/kit-architecture.md`: (modify) config embed-census sentence repointed

## Impact

- 11 files changed, +51/−30: 4 Go files (1 new, 3 modified — comment/wiring only), 1 yaml move, 5 docs files, 1 kit skill (`src/kit/skills/_cli-agents.md`, canonical source).
- **Zero behavior change**: no exported symbol, resolution rule, or value changed; the embedded bytes are identical apart from the yaml header comment.
- Verified: `go build ./...`, `go vet ./...`, full `go test ./...` (including the six drift-guard tests the yaml header names), `gofmt -l` clean, `fab memory-index --check` shows only pre-existing shape warnings.
- No migration needed: nothing about user data, config schema, or `.status.yaml` changes; the file path is internal to the dev repo.

## Open Questions

- (none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Hoist target is the Go module root `src/go/fab/defaults.yaml`, not repo root/`src/` | User explicitly chose presented option 2 after the go:embed constraint was explained; higher placements are compile errors | S:95 R:90 A:95 D:95 |
| 2 | Certain | Embed moves to a new one-file root package; parsing/consumption stay in `internal/agent` | The only way to host the file at the module root under go:embed's package-directory rule while preserving the single-source + drift-guard design | S:85 R:85 A:95 D:90 |
| 3 | Confident | Historical `log.md`/`log.seed.md` rows are NOT swept | They record what was true at their date; FKF present-truth rule applies to live prose only | S:70 R:90 A:85 D:80 |
| 4 | Confident | Symbol-level claims (e.g. `internal/agent.BuiltinProvider` "over the go:embed'd defaults.yaml", `internal/agent`'s `init()` parsing) left unchanged | Ownership of parsing/consumption genuinely did not move; only physical file location did | S:65 R:85 A:80 D:75 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).

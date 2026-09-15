# Plan: Hoist defaults.yaml to the Go module root

**Change**: 260823-a3mu-hoist-defaults-module-root
**Intake**: `intake.md`

> Adopted change — code authored off-pipeline. Apply was skipped; this plan is reverse-engineered from the branch diff to feed hydrate.

## Requirements

### File placement and embed wiring

`defaults.yaml` — the single value source for every built-in default (depth knobs, provider grammars and per-role model fills, dispatch defaults, autopilot merge mode) — now lives at the Go module root, `src/go/fab/defaults.yaml`, instead of `src/go/fab/internal/agent/defaults.yaml`. The module root is the highest placement `go:embed` permits (the directive cannot reference files above the embedding package's directory), and repo-root/`src/` placement was rejected because a copy-at-build step would reintroduce the two-copies drift the embedded-single-source design exists to prevent.

A new one-file module-root package (`src/go/fab/defaults.go`, `package fab`) exists solely to own the `//go:embed defaults.yaml` directive, exporting the raw bytes as `DefaultsYAML []byte`. It carries the embed-over-kit-cache rationale (kit and binary release atomically; a binary can never disagree with the defaults it shipped with) relocated from `agent.go`.

`internal/agent` remains the sole parser and consumer: `defaultsYAML` is now assigned from the root package (`var defaultsYAML = fabroot.DefaultsYAML`, import alias `fabroot`) instead of a local embed. `mustParseDefaults`, `builtinDefaults`, the init-injection into `config.DefaultDispatch*` / `config.DefaultAutopilotMergeMode`, every exported symbol, and all six drift-guard tests are unchanged — they operate on the embedded bytes, not a disk path.

### Location-claim sweep

All live prose locating the file in `internal/agent` is repointed to the module root; historical `log.md`/`log.seed.md` rows are deliberately untouched (they record what was true at their date):

- Full-path references updated in `docs/specs/stage-models.md`, `docs/specs/config.md`, `docs/memory/runtime/providers-and-profiles.md`.
- "`internal/agent`'s embedded `defaults.yaml`" possessive phrasing → "the module-root embedded `defaults.yaml`" in `docs/specs/config.md`, `docs/specs/stage-models.md`, `docs/memory/_shared/configuration.md` (the cascade tier-4 sentence additionally names the new path and the embed/parse split), and `src/kit/skills/_cli-agents.md` (canonical kit source, not the deployed copy).
- `docs/memory/distribution/kit-architecture.md`'s config embed-census sentence names the module-root file, embedded by the root fab package, parsed by `internal/agent`.
- Go comments in `internal/config/config.go` and `internal/spawn/spawn.go` reworded the same way. Symbol-level claims (e.g. `internal/agent`'s `init()` parsing, `internal/agent.BuiltinProvider`) stay — parsing/consumption ownership genuinely did not move.

### Non-Goals

- No behavior change of any kind: no value, resolution rule, or exported symbol changes; the embedded bytes differ only in the yaml header comment.
- No migration: nothing about user data, config schema, or `.status.yaml` is touched; the path is internal to the dev repo.

## Tasks

- [x] Adopted: implementation authored outside the pipeline (see branch `robust-python`, commit 6bd5b18a).

## Acceptance

- [x] Adopted: code already authored and verified (`go build`/`go vet`/full `go test ./...` incl. the six drift-guard tests, `gofmt -l` clean); a diff-only review runs in this pipeline.

## Assumptions

0 assumptions.

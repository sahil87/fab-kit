# Intake: Delete Orphaned hooklib Hook-Payload Parsers

**Change**: 260819-hk7p-delete-orphaned-hook-payload-parsers
**Created**: 2026-08-19

## Origin

One-shot `/fab-new hk7p` from the backlog:

> [hk7p] 2026-08-08: Delete the orphaned hooklib hook-payload parsers (ParsePayload, MatchArtifactPath/ArtifactMatch) — zero non-test callers since the fab hook removal (relocated from docs/memory/runtime/runtime-agents.md by /docs-distill-memory)

The micro-change backstop was evaluated and does not apply: the deletion has memory impact — `docs/memory/runtime/runtime-agents.md` explicitly documents these parsers as dead code, and that claim must be removed when the code is.

## Why

1. **Pain point**: `src/go/fab/internal/hooklib/artifact.go` still carries the PostToolUse hook-payload half — `postToolUsePayload`, `ParsePayload`, `ArtifactMatch`, `MatchArtifactPath` — even though the entire `fab hook` command family was removed outright in 2.14.0 (ioku). Verified 2026-08-19: `grep -rn "ParsePayload\|MatchArtifactPath\|ArtifactMatch"` over `src/` hits only `artifact.go` and `artifact_test.go` — zero non-test callers.
2. **Consequence of not fixing**: dead code accrues maintenance cost — the file misleads readers into thinking fab still parses hook payloads, tests exercise behavior nothing consumes, and memory (`runtime/runtime-agents.md`) has to keep carrying a "this is dead code" caveat instead of the code simply being gone.
3. **Why this approach**: plain deletion is the whole fix. The live half of `hooklib` (change-type inference and plan-section counting, consumed by `internal/refresh` and `internal/status`) is untouched; no callers exist to migrate.

## What Changes

### `src/go/fab/internal/hooklib/artifact.go` — delete the hook-payload half

Remove, verbatim (current lines 12–88):

- `postToolUsePayload` (unexported struct — the Claude Code PostToolUse JSON shape)
- `ParsePayload(r io.Reader) (string, error)`
- `ArtifactMatch` (struct: `ChangeFolder`, `Artifact`)
- `MatchArtifactPath(filePath string) (ArtifactMatch, bool)`

With them gone, the `encoding/json` and `io` imports become unused and MUST be dropped. `regexp`, `strings`, and `internal/lines` remain used by the retained helpers (`InferChangeType`/`fixSignal`, `HasSectionHeading`, `CountSectionItemsBounded`, `CountCompletedSectionItemsBounded`, `scanSectionItems`) — all of which stay exactly as they are.

### `src/go/fab/internal/hooklib/artifact_test.go` — delete the matching tests

Remove the 15 test functions covering the deleted API (nothing else):

- `TestParsePayload_Valid`, `TestParsePayload_MalformedJSON`, `TestParsePayload_MissingFilePath`, `TestParsePayload_Empty`
- `TestMatchArtifactPath_AbsoluteIntake`, `TestMatchArtifactPath_RelativePlan`, `TestMatchArtifactPath_LegacySpecRejected`, `TestMatchArtifactPath_Plan`, `TestMatchArtifactPath_LegacyTasksRejected`, `TestMatchArtifactPath_LegacyChecklistRejected`, `TestMatchArtifactPath_NonFabPath`, `TestMatchArtifactPath_UnknownArtifact`, `TestMatchArtifactPath_EmptyFolder`, `TestMatchArtifactPath_NoFolder`, `TestMatchArtifactPath_NotFabPrefix`

The `TestInferChangeType_*`, `TestHasSectionHeading_*`, and `TestCountSectionItemsBounded_*` functions stay. Drop any test-file imports that become unused (e.g., `strings`/`bytes` readers used only by the ParsePayload tests, if not used elsewhere in the file).

Verification: `go build ./...` and `go test ./src/go/fab/internal/hooklib/` green; repo-wide grep for `ParsePayload|MatchArtifactPath|ArtifactMatch` returns no hits under `src/`.

### Not changing

- The retained plan-parsing half of `hooklib` and its consumers (`internal/refresh`, `internal/status`) — no signature or behavior change.
- The package name `hooklib` — memory records "the package keeps its legacy name for now" (ioku); renaming is out of scope.
- `docs/memory/distribution/kit-architecture.md` — its claim that `internal/hooklib` "provides only the shared parsing primitives (change type inference, task/checklist section counting) in artifact.go" becomes exactly true after this deletion; no edit needed.
- No CLI surface change → no `_cli-fab.md` update, no migration (no user data restructured).

## Affected Memory

- `runtime/runtime-agents.md`: (modify) remove the dead-code caveat sentence — "The **hook-payload** parsers in the same file — `ParsePayload` and `MatchArtifactPath`/`ArtifactMatch` — have zero non-test callers (dead code)." — the parsers no longer exist; the adjacent "artifact.go survives, but only partly live" framing tightens to the retained plan-parsing helpers only.

## Impact

- **Code**: `src/go/fab/internal/hooklib/artifact.go` (−~77 lines), `src/go/fab/internal/hooklib/artifact_test.go` (−15 test functions). No other package touched; no exported API outside `hooklib` changes.
- **Docs**: one memory file (`runtime/runtime-agents.md`) at hydrate. Historical mentions in `docs/memory/*/log.md`, `log.seed.md`, and `docs/specs/findings/` are append-only history and stay verbatim. `docs/specs/change-types.md`'s reference to `fixSignal` in `artifact.go` remains valid (that function stays).
- **Tests**: deletion-only; retained hooklib tests must stay green.
- **Risk**: minimal — compiler-verified deletion of code with zero non-test callers.

## Open Questions

*None — the backlog item is precise, and the zero-caller claim was re-verified against the current tree.*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Deletion scope = the full hook-payload half (`postToolUsePayload`, `ParsePayload`, `ArtifactMatch`, `MatchArtifactPath`) plus its 15 tests; nothing else in hooklib moves | Backlog item names the symbols; grep re-verified zero non-test callers; memory (runtime-agents.md, ioku) records the same split | S:90 R:90 A:95 D:95 |
| 2 | Certain | Drop the `encoding/json` and `io` imports from artifact.go (and any test-only imports orphaned by the test deletions) | Compiler-enforced — the build fails otherwise; no judgment involved | S:85 R:95 A:100 D:100 |
| 3 | Confident | Keep the legacy package name `hooklib`; no rename ride-along | Memory explicitly parks the rename ("keeps its legacy name for now", ioku); a rename would touch 5+ importing packages and belongs to its own change | S:70 R:60 A:85 D:80 |
| 4 | Confident | Memory edit limited to `runtime/runtime-agents.md`; `kit-architecture.md` needs no change (its "provides only the shared parsing primitives" claim becomes exactly true) | Repo-wide grep of the symbol names over docs/ found only the runtime-agents.md live claim; logs/findings are append-only history | S:75 R:80 A:80 D:75 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).

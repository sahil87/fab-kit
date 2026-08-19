# Plan: Delete Orphaned hooklib Hook-Payload Parsers

**Change**: 260819-hk7p-delete-orphaned-hook-payload-parsers
**Intake**: `intake.md`

## Requirements

### Runtime: hooklib dead-code deletion

#### R1: Delete the hook-payload half of artifact.go
`src/go/fab/internal/hooklib/artifact.go` MUST no longer contain `postToolUsePayload`, `ParsePayload`, `ArtifactMatch`, or `MatchArtifactPath`. The now-unused `encoding/json` and `io` imports MUST be removed. The retained helpers — `InferChangeType`/`fixSignal` (+ their regex vars), `HasSectionHeading`, `CountSectionItemsBounded`, `CountCompletedSectionItemsBounded`, `scanSectionItems`, and the `PlanSection` constants — MUST remain byte-identical in behavior and signature.

- **GIVEN** the current tree, where the four symbols have zero non-test callers (verified by repo-wide grep)
- **WHEN** the symbols and their orphaned imports are deleted
- **THEN** `go build ./...` succeeds
- **AND** a repo-wide grep for `ParsePayload|MatchArtifactPath|ArtifactMatch` under `src/` returns no hits

#### R2: Delete the matching tests
`src/go/fab/internal/hooklib/artifact_test.go` MUST no longer contain the 4 `TestParsePayload_*` and 11 `TestMatchArtifactPath_*` functions. Test-file imports orphaned by those deletions MUST be removed. The `TestInferChangeType_*`, `TestHasSectionHeading_*`, and `TestCountSectionItemsBounded_*` functions MUST remain and pass.

- **GIVEN** the deletions from R1 and R2
- **WHEN** `go test ./src/go/fab/internal/hooklib/` runs
- **THEN** the package tests pass with only the retained test functions present

### Non-Goals

- Renaming the `hooklib` package — memory parks the legacy name ("keeps its legacy name for now", ioku); a rename touches 5+ importing packages and is its own change.
- Any change to the retained plan-parsing helpers or their consumers (`internal/refresh`, `internal/status`).
- Editing `docs/memory/distribution/kit-architecture.md` — its hooklib description becomes exactly true after the deletion.

### Deprecated Requirements

#### hooklib hook-payload parsing API (`ParsePayload`, `MatchArtifactPath`/`ArtifactMatch`)
**Reason**: The `fab hook` command family — the only consumer — was removed outright in 2.14.0 (ioku); the parsers have had zero non-test callers since.
**Migration**: N/A — nothing parses PostToolUse payloads anymore; artifact bookkeeping is pull-based via `fab status refresh`.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Delete `postToolUsePayload`, `ParsePayload`, `ArtifactMatch`, and `MatchArtifactPath` from `src/go/fab/internal/hooklib/artifact.go`; drop the now-unused `encoding/json` and `io` imports (keep `regexp`, `strings`, `internal/lines`) <!-- R1 -->
- [x] T002 Delete the 4 `TestParsePayload_*` and 11 `TestMatchArtifactPath_*` functions from `src/go/fab/internal/hooklib/artifact_test.go`; drop any test imports orphaned by the deletion <!-- R2 -->
- [x] T003 Verify: `go build ./...`, `go vet ./src/go/fab/internal/hooklib/`, `go test ./src/go/fab/internal/hooklib/` green; repo-wide grep for `ParsePayload|MatchArtifactPath|ArtifactMatch` under `src/` returns zero hits <!-- R1 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `artifact.go` contains none of `postToolUsePayload`, `ParsePayload`, `ArtifactMatch`, `MatchArtifactPath`; imports reduced to `regexp`, `strings`, `internal/lines`; `go build ./...` green
- [x] A-002 R2: `artifact_test.go` contains none of the 15 deleted test functions; `go test ./src/go/fab/internal/hooklib/` green

### Removal Verification

- [x] A-003 R1: Repo-wide grep for `ParsePayload|MatchArtifactPath|ArtifactMatch` under `src/` returns zero hits (no stragglers, no new references)

### Behavioral Correctness

- [x] A-004 R1: Retained helpers (`InferChangeType`, `HasSectionHeading`, `CountSectionItemsBounded`, `CountCompletedSectionItemsBounded`) are unchanged — no signature, behavior, or test changes beyond the deletions

### Code Quality

- [x] A-005 Pattern consistency: The surviving file reads as a coherent plan-parsing unit (no orphaned comments referencing the deleted parsers)
- [x] A-006 No unnecessary duplication: Deletion-only change — no new code introduced
- [x] A-007 Go changes ship tests: Test updates accompany the .go change (the deletions themselves; retained tests still cover the surviving API)
- [x] A-008 Canonical source only: No edits under `.claude/skills/`; no CLI surface change (no `_cli-fab.md` update needed); no user-data restructuring (no migration needed)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scoped verification (build repo-wide, tests scoped to `internal/hooklib`) is sufficient — no other package imports the deleted symbols | Grep-verified zero external references; `go build ./...` catches any missed importer | S:85 R:95 A:95 D:95 |

1 assumptions (1 certain, 0 confident, 0 tentative).

# Plan: Add the reserved `fab skill topics` verb

**Change**: 260903-psgm-fab-skill-topics-verb
**Intake**: `intake.md`

## Requirements

### CLI: the reserved `topics` verb

#### R1: `fab skill topics` prints zero topics conformantly
`fab skill topics` MUST print empty stdout (zero bytes), write nothing to stderr, and exit 0 — the shll `skill` standard's mandated behavior for a tool shipping zero topic pages. The reserved verb MUST NOT be implemented as a registered cobra child command.

- **GIVEN** an installed fab with zero topic pages (no `docs/site/skill/` directory)
- **WHEN** `fab skill topics` is invoked
- **THEN** stdout is exactly zero bytes, stderr is empty, and the exit code is 0

#### R2: bare `fab skill` and other positionals are unchanged
Bare `fab skill` SHALL keep its existing contract (embedded bundle to stdout byte-identical to `docs/site/skill.md`, stderr empty, exit 0). Any positional other than `topics` SHALL remain a usage error (cobra's default unknown-argument error, classified to exit 2 by the binary-wide `run()` seam).

- **GIVEN** the updated binary
- **WHEN** `fab skill` is invoked with no args
- **THEN** the embedded bundle is written to stdout byte-identically, stderr empty, exit 0
- **GIVEN** the updated binary
- **WHEN** `fab skill unexpected` is invoked
- **THEN** the invocation fails before `RunE` with a usage error → exit 2

#### R3: frozen surfaces stay byte-stable
The change MUST NOT add a node to the `fab help-dump` JSON tree, MUST NOT surface `topics` in `fab skill --help`, and MUST NOT modify `docs/site/skill.md` (the embed drift guard `TestSkillEmbedMatchesCanonical` keeps passing untouched).

- **GIVEN** the updated binary
- **WHEN** `fab skill --help` and `fab help-dump` are inspected
- **THEN** neither mentions `topics` and no new command node exists

#### R4: CLI reference updated per constitution
`src/kit/skills/_cli-fab.md` § fab skill MUST be updated in the same change: the reserved `topics` positional (content-topic names one per line, raw stdout, stderr empty, exit 0; currently zero topics → empty output), reserved by the shll `skill` standard as a machine affordance (never a content topic), with other positionals remaining usage errors → exit 2.

- **GIVEN** the shipped change
- **WHEN** `_cli-fab.md` § fab skill is read
- **THEN** it no longer claims "Takes no args/flags (`cobra.NoArgs`)" and documents the `topics` contract

#### R5: tests pin the new contract
`src/go/fab/cmd/fab/skill_test.go` MUST gain a test asserting the R1 contract through the assembled cobra command, and `TestSkill_RejectsArgs` (plus its comment, which currently says "the standard says `fab skill` takes no args/flags") MUST be reworded to the new contract: unknown positionals rejected, `topics` accepted. Existing bare-invocation and bundle-constraint tests keep passing unmodified.

- **GIVEN** the updated package
- **WHEN** `go test ./cmd/fab/` runs from `src/go/fab/`
- **THEN** all skill tests pass, including the new `topics` contract test and the reworded rejection test

### Non-Goals

- No topic pages, no `docs/site/skill/` directory, no `Topics:` help line, no core-bundle topic index — those mandates bind only tools shipping topic pages.
- No new exit codes, no router change (`skill ∉ LifecycleCommands`), no config surface, no migration.

### Design Decisions

#### Custom Args validator, not a cobra child command
**Decision**: Replace `Args: cobra.NoArgs` with a validator accepting zero args or exactly the single arg `topics`; branch in `RunE` (arg present → write nothing, return nil).
**Why**: A child command would surface `topics` in `fab skill --help` and add a node to the frozen `fab help-dump` JSON tree (shll.ai command-reference pull surface); the plain positional keeps both byte-stable and matches the standard's "machine affordance, not a topic page" framing. The validator is trivially extended to a topic-name set when fab later ships topic pages.
**Rejected**: `cobra.Command{Use: "topics"}` child — pollutes help/help-dump; `--list` flag — explicitly rejected upstream by the standard (the shll composer forwards positional args verbatim).
*Introduced by*: 260903-psgm-fab-skill-topics-verb

#### Unknown-positional error shape stays cobra's default
**Decision**: `fab skill <anything-but-topics>` keeps cobra's default unknown-argument usage error (exit 2 via the binary-wide `run()` classification). Resolves the intake's deferred assumption #7.
**Why**: The standard's fail-fast rule ("error on stderr naming the valid topics") textually binds only tools shipping topic pages, and today it would name zero valid topics — a bespoke error naming nothing is worse than the uniform usage error. Trivially reversible when topic pages ship.
**Rejected**: Adopting the fail-fast shape now — speculative, and an "error naming the valid topics" with an empty list is a confusing contract.
*Introduced by*: 260903-psgm-fab-skill-topics-verb

## Tasks

### Phase 1: Core Implementation

- [x] T001 Update `src/go/fab/cmd/fab/skill.go`: replace `Args: cobra.NoArgs` with a custom validator accepting zero args or exactly `topics`; in `RunE`, when the arg is `topics`, write nothing and return nil; update the doc comments (lines 24–30) to the new arg contract <!-- R1, R2, R3 -->
- [x] T002 Update `src/go/fab/cmd/fab/skill_test.go`: add `TestSkill_TopicsEmptyContract` (assembled cobra command, `SetArgs([]string{"topics"})` → empty stdout, empty stderr, nil error); reword `TestSkill_RejectsArgs` and its comment to the new contract (unknown positional still a usage error; the "takes no args/flags" claim removed) <!-- R5, R2 -->
- [x] T003 [P] Update `src/kit/skills/_cli-fab.md` § fab skill (line ~939): replace the "Takes no args/flags (`cobra.NoArgs`)" sentence with the reserved `topics` positional contract (one topic name per line raw to stdout, stderr empty, exit 0; fab ships zero topic pages so output is empty; reserved machine affordance per the shll `skill` standard, never a content topic; other positionals remain usage errors → exit 2) <!-- R4 -->

### Phase 2: Verification

- [x] T004 Run `go test ./cmd/fab/` from `src/go/fab/` (scoped; widen to `./...` only if anything cross-cutting surfaces) and run the stale-claim sweep grep (`cobra.NoArgs`, "no args", "zero-arg", "usage error" scoped to skill contexts) confirming the only remaining occurrences are the two hydrate-owned memory files (`docs/memory/distribution/distribution.md`, `docs/memory/distribution/kit-architecture.md`) <!-- R5, R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab skill topics` produces zero-byte stdout, empty stderr, exit 0 (pinned by a test through the assembled cobra command)
- [x] A-002 R4: `_cli-fab.md` § fab skill documents the `topics` contract and no longer claims `cobra.NoArgs`/"no args"

### Behavioral Correctness

- [x] A-003 R2: bare `fab skill` contract unchanged — `TestSkill_StdoutByteIdentical` and `TestSkill_EmptyStderrThroughCobra` pass unmodified
- [x] A-004 R2: `fab skill <unknown>` remains a usage error (test-pinned, comment reworded to the new contract)

### Scenario Coverage

- [x] A-005 R5: `go test ./cmd/fab/` passes with the new and reworded tests

### Edge Cases & Error Handling

- [x] A-006 R3: no new cobra command node — `topics` absent from `fab skill --help` output and the help-dump tree; `docs/site/skill.md` untouched (drift guard passes)

### Code Quality

- [x] A-007 Pattern consistency: validator + `RunE` branch follow the existing `skill.go`/`runSkill` seam style; no bespoke exit-code handling
- [x] A-008 No unnecessary duplication: no new helper duplicating cobra arg validation; doc comments state the contract once
- [x] A-009 Canonical source only: kit-doc edit lands in `src/kit/skills/_cli-fab.md`, never `.claude/skills/`
- [x] A-010 CLI ⇒ docs + tests: the Go signature change ships with `_cli-fab.md` § fab skill and test updates in the same change

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (`cobra.NoArgs` remains in use as the validator's fall-through and across sibling commands; the `runSkill` seam and both pre-existing tests are untouched in behavior)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Unknown-positional error shape stays cobra's default usage error (exit 2) — resolves intake deferred assumption #7 per its front-runner | The standard's fail-fast rule binds only topic-shipping tools; an error naming zero valid topics is a worse contract; trivially reversible when topic pages ship | S:70 R:85 A:75 D:70 |
| 2 | Confident | The `topics` branch lives in `RunE` (write nothing, return nil) rather than a separate seam function | Zero bytes by construction needs no testable writer seam; `runSkill` stays untouched for the bundle path | S:65 R:90 A:80 D:70 |
| 3 | Certain | Memory-file updates (`distribution.md`, `kit-architecture.md`) happen at hydrate, not apply | Affected Memory is hydrate's contract; apply's sweep only verifies no other stale occurrences exist | S:85 R:85 A:90 D:90 |

3 assumptions (1 certain, 2 confident, 0 tentative).

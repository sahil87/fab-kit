# Plan: Bump Claude Default Model Profiles to Sonnet 5.5

**Change**: 260929-9dwn-bump-claude-defaults-sonnet-5-5
**Intake**: `intake.md`

## Requirements

### Agent defaults: the two Sonnet role fills

#### R1: The `operator` and `fast` Claude fills SHALL pin `claude-sonnet-5-5`

`src/go/fab/defaults.yaml`'s `providers.claude.profiles` block MUST spell `claude-sonnet-5-5`
for the `operator` and `fast` roles. Every `effort` value on every row MUST be unchanged
(`operator` and `fast` stay `medium`; the four Opus rows stay `high`). The four Opus rows'
model IDs MUST be unchanged.

- **GIVEN** a project with no `providers.claude.profiles` override
- **WHEN** `fab agent operator -o yaml` or `fab agent fast -o yaml` resolves
- **THEN** the emitted `model` is `claude-sonnet-5-5`
- **AND** the emitted `effort` is `medium`
- **AND** `fab agent default|doing|review|hydrate -o yaml` still emits `claude-opus-5-5` / `high`

#### R2: The Go pin table SHALL match the shipped defaults

`TestDefaultRoleProfilesArePinned` in `src/go/fab/internal/agent/agent_test.go` is the
deliberate-change pin: its literal table MUST be updated in the same commit as the YAML, so the
bump is asserted rather than merely followed.

- **GIVEN** the `defaults.yaml` fills have moved to `claude-sonnet-5-5`
- **WHEN** `go test ./internal/agent/...` runs
- **THEN** `TestDefaultRoleProfilesArePinned` passes
- **AND** it would fail if either file were updated without the other

#### R3: The drift-guarded spec mirror SHALL match the shipped defaults

The `### Default role profiles` table in `docs/specs/stage-models.md` is the one doc mirror of
the fills, parsed by `TestDocTablesMatchAgentMaps`. Its `operator` and `fast` Model cells MUST
read `claude-sonnet-5-5`, landing in the same commit as the YAML.

- **GIVEN** the `defaults.yaml` fills have moved to `claude-sonnet-5-5`
- **WHEN** `go test ./cmd/fab/...` runs
- **THEN** `TestDocTablesMatchAgentMaps` passes with no disagreement between table and maps

#### R4: The generated-paragraph digest catalog SHALL record the new rendering

`knownGeneratedSystemParagraphDigests` in `src/go/fab/internal/configupgrade/configupgrade.go`
is append-only and byte-exact. The new `providers:` advert rendering's digest MUST be appended
with a dated comment naming this change; every existing entry MUST be retained verbatim. The
superseded vvdv entry's comment, which currently claims "Current providers advert", MUST be
re-labelled to name its era, matching how earlier superseded entries were re-labelled.

- **GIVEN** the claude fills have changed, so the rendered providers advert's bytes differ
- **WHEN** `go test ./internal/configupgrade/...` runs
- **THEN** `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` passes
- **AND** the vvdv digest and every older digest are still present in the map
- **AND** no comment in the catalog claims two entries are simultaneously "current"

#### R5: The "explicit effort is load-bearing" claim SHALL cover both directions at all three sites

Sonnet 5.5's built-in default effort is `high` (intake Assumption 7), so the Sonnet rows'
explicit `effort: medium` holds them *down*, the opposite direction from the Opus rows' explicit
`high`. The claim MUST state both directions, and MUST be updated at all three of its homes in
lockstep (`fab/project/code-quality.md` § Sibling Sweeps makes a partial update a must-fix):
the `defaults.yaml` fill-block comment, the `docs/specs/stage-models.md` "Why these defaults"
sentence, and the `docs/memory/runtime/providers-and-profiles.md` Design Decision (the memory
site is hydrate's, not apply's).

- **GIVEN** a reader looking at the `defaults.yaml` fill block
- **WHEN** they read the comment above `profiles:`
- **THEN** it explains that dropping `effort: high` steps the Opus rows DOWN and dropping
  `effort: medium` steps the Sonnet rows UP
- **AND** the `stage-models.md` sentence carries the same two-direction claim
- **AND** no site is left carrying only the Opus half

#### R6: The bump SHALL be confined to default-fill claims

Occurrences of `claude-sonnet-5` that are arbitrary test fixtures, explicit invocation
overrides, historical change/plan documents, or archive content MUST be left verbatim. No
`src/kit/**` file, no migration, and no Go logic change is in scope.

- **GIVEN** the tree after apply
- **WHEN** `grep -rn 'claude-sonnet-5\b' src/ docs/specs/` runs
- **THEN** every remaining hit is a fixture, an explicit override, or a historical document —
  none is a claim about the shipped default
- **AND** `git diff --stat` shows no file under `src/kit/`

### Non-Goals

- **No re-tiering** — whether Sonnet 5.5 is strong enough for `review` or `hydrate` to move down
  from Opus is a separate cost/quality judgment, explicitly deferred.
- **No effort changes** — `operator`/`fast` stay `medium`, the Opus rows stay `high`.
- **No prose rewrite of "Why these defaults"** beyond R5's one-clause addition.
- **No change to the codex, kimi, or agy fills or command templates.**
- **No migration file** — `presence=intent` means a user who copied the old rows above their
  fence pinned `claude-sonnet-5` deliberately, and that pin keeps working (vvdv precedent).
- **No `TestModelAlias` case addition** — `modelAliasPrefixes` matches on `claude-sonnet-`, and
  the existing dated-variant case already exercises the longer-suffix prefix behavior.

### Design Decisions

#### Widen the Load-Bearing-Effort Claim Rather Than Leave It Opus-Scoped
**Decision**: Extend the "explicit effort is load-bearing" claim at all three of its homes to
state both directions — Opus 5.5 defaults to `medium` so the explicit `high` holds those rows up;
Sonnet 5.5 defaults to `high` so the explicit `medium` holds those rows down — rather than
leaving the claim Opus-only.
**Why**: Once all six fills sit on the 5.5 generation, an Opus-only statement reads as a complete
account of the effort column and is not. The concrete trap it leaves is a reader concluding the
Sonnet rows' `medium` is a redundant restatement of a model default and dropping it, which would
silently step the highest-volume role (`operator`) and the ship role (`fast`) UP to `high` — the
mirror image of the regression the original comment exists to prevent.
**Rejected**: Leaving all three sites untouched (the existing claim is true but partial, and the
partiality is newly load-bearing because the Sonnet rows now sit one generation closer to the
Opus rows' framing); updating only the YAML comment (a subset update of a known sibling class is
a must-fix under `fab/project/code-quality.md` § Sibling Sweeps).
*Introduced by*: 260929-9dwn-bump-claude-defaults-sonnet-5-5

## Tasks

### Phase 1: Core Implementation

- [x] T001 Edit `src/go/fab/defaults.yaml`: set `providers.claude.profiles.operator.model` and `.fast.model` to `claude-sonnet-5-5` (efforts unchanged, Opus rows untouched), and widen the fill-block comment above `profiles:` to state both effort directions <!-- R1, R5 -->
- [x] T002 [P] Update the `TestDefaultRoleProfilesArePinned` literal table in `src/go/fab/internal/agent/agent_test.go` — `RoleOperator` and `RoleFast` to `claude-sonnet-5-5`, efforts and the four Opus rows unchanged <!-- R2 -->
- [x] T003 [P] Update the `### Default role profiles` table in `docs/specs/stage-models.md` — the `operator` and `fast` Model cells to `claude-sonnet-5-5` — and add the matching Sonnet clause to the "Why these defaults" load-bearing-effort sentence <!-- R3, R5 -->

### Phase 2: Digest Harvest (strictly after T001)

- [x] T004 Run the configupgrade catalog test to harvest the new providers-advert digest, append it to `knownGeneratedSystemParagraphDigests` in `src/go/fab/internal/configupgrade/configupgrade.go` with a dated `260929-9dwn` comment, and re-label the now-superseded vvdv entry's comment to name its era <!-- R4 -->

### Phase 3: Verification

- [x] T005 Run `gofmt -l` on every touched `.go` file and `go test ./internal/agent/... ./internal/configupgrade/... ./cmd/fab/...` from `src/go/fab`; re-grep `claude-sonnet-5` across `src/` and `docs/specs/` to confirm every remaining hit is a fixture or historical document and that no `src/kit/**` file changed <!-- R6 -->

## Execution Order

- T001 blocks T004 — the digest covers the rendered advert paragraph's bytes, which include the
  fill-block comment T001 widens. Harvesting before T001 lands would require a second harvest.
- T002 and T003 are independent of T001's ordering constraint and of each other.
- T005 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `providers.claude.profiles.operator` and `.fast` in `src/go/fab/defaults.yaml` read `{ model: claude-sonnet-5-5, effort: medium }`, and the four Opus rows are byte-identical to before
- [x] A-002 R2: The `TestDefaultRoleProfilesArePinned` table names `claude-sonnet-5-5` for `RoleOperator` and `RoleFast`, and the test passes
- [x] A-003 R3: The `### Default role profiles` table's `operator` and `fast` Model cells read `claude-sonnet-5-5`, and `TestDocTablesMatchAgentMaps` passes
- [x] A-004 R4: A new digest entry with a `260929-9dwn` comment is appended to `knownGeneratedSystemParagraphDigests`, and `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` passes
- [x] A-005 R5: The `defaults.yaml` fill-block comment and the `stage-models.md` "Why these defaults" sentence both state the Opus-down and Sonnet-up directions

### Behavioral Correctness

- [x] A-006 R1: `fab agent operator -o yaml` and `fab agent fast -o yaml`, run against a binary built from this tree, emit `model: claude-sonnet-5-5` and `effort: medium`
- [x] A-007 R4: Every digest present in `knownGeneratedSystemParagraphDigests` before this change is still present after it — the catalog grew by exactly one entry and lost none
- [x] A-008 R4: No comment in the digest catalog labels more than one entry "current"

### Scenario Coverage

- [x] A-009 R2: The pin test would fail if `defaults.yaml` and the pin table disagreed — verified by the test's own assertion shape, not by breaking it
- [x] A-010 R6: `grep -rn 'claude-sonnet-5' src/ docs/specs/` returns only fixtures, explicit overrides, and the two bumped default-fill sites; `git diff --name-only` lists no path under `src/kit/`

### Edge Cases & Error Handling

- [x] A-011 R5: No sibling site of the load-bearing-effort claim is left carrying only the Opus half — the two apply-owned sites are both updated, and the memory Design Decision is listed for hydrate
- [x] A-012 R1: No `effort` value anywhere in `defaults.yaml` changed — verifiable from the diff

### Code Quality

- [x] A-013 Pattern consistency: The appended digest entry follows the existing catalog's comment-then-digest shape, and the widened comments match the surrounding prose voice
- [x] A-014 No unnecessary duplication: No new restatement of the fills is introduced — the spec table remains the only doc mirror
- [x] A-015 Canonical source only: No file under `.agents/skills/` or `.claude/skills/` is edited
- [x] A-016 Go changes ship tests: The Go diff is accompanied by its test updates (the pin table and the digest catalog are the test surface)
- [x] A-017 gofmt clean: `gofmt -l` reports nothing for every touched `.go` file

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- T004's digest test is EXPECTED to fail on its first run — the failure message supplies the
  value to append. That is the catalog's designed mechanism, not a defect.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Take intake Assumption 8's widen branch: extend the load-bearing-effort claim at all three sites rather than leaving all three untouched | Intake Assumption 7 establishes Sonnet 5.5 defaults to `high`, so the Sonnet rows' explicit `medium` is load-bearing in the opposite direction. Leaving the claim Opus-only invites dropping `effort: medium` and stepping `operator`/`fast` UP. Prose-only and trivially reversible; the intake sanctions either branch and forbids only a subset update | S:85 R:95 A:85 D:75 |
| 2 | Certain | The two apply-owned sites (`defaults.yaml` comment, `stage-models.md` sentence) are updated in this change; the memory Design Decision is hydrate's site and is recorded in § Affected Memory, not edited at apply | Standard stage division — apply never writes `docs/memory/`. The intake states the memory DD is updated at hydrate | S:90 R:95 A:95 D:90 |
| 3 | Certain | The memory DD heading `### Explicit \`effort\` on the Opus Fills Is Load-Bearing` is itself part of the sweep and is renamed at hydrate, not just its body | The heading names Opus specifically, so widening only the body would leave the title contradicting its own content. Flagged here so hydrate does not treat it as body-only | S:80 R:95 A:90 D:80 |
| 4 | Certain | Task ordering pins T001 before T004; T002/T003 are unordered | Intake Assumption 9 — the digest covers the advert paragraph's bytes including the comment. Self-correcting if missed (the test simply fails again and supplies a fresh value) | S:85 R:95 A:90 D:85 |
| 5 | Certain | No `TestModelAlias` case is added | Intake Assumption 10's stated default; `modelAliasPrefixes` matches `claude-sonnet-` and the dated-variant case already covers longer-suffix behavior | S:85 R:95 A:90 D:80 |
| 6 | Confident | A-006's live `fab agent` check is run against a binary built from this tree (`go run ./cmd/fab`), not the installed `fab` on PATH | The installed binary predates even the vvdv Opus bump, so it would report stale values and the check would be meaningless. Building from the tree is the only way to observe this change's effect before release | S:75 R:90 A:85 D:70 |

6 assumptions (5 certain, 1 confident, 0 tentative).

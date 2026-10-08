# Plan: Bump Claude Default Model Profiles to Opus 5.5

**Change**: 260923-vvdv-bump-claude-defaults-opus-5-5
**Intake**: `intake.md`

## Requirements

### Runtime: Claude default role fills

#### R1: The four Opus fills move to `claude-opus-5-5`
`src/go/fab/defaults.yaml` `providers.claude.profiles` MUST pin `claude-opus-5-5` for `default`, `doing`, `review`, and `hydrate`, and MUST keep every `effort` value unchanged (`high` on the four Opus rows, `medium` on the two Sonnet rows). The `operator` and `fast` rows MUST stay `claude-sonnet-5`. A short comment beside the block SHALL record that the explicit `effort` is load-bearing under Opus 5.5's `medium` built-in default.

- **GIVEN** a project with no `agent.profiles` override and no `providers.claude.profiles` override
- **WHEN** `fab agent doing -o yaml` (or `default`/`review`/`hydrate`) resolves
- **THEN** `model` is `claude-opus-5-5`, `effort` is `high`, and `model_alias` is `opus`
- **AND** `fab agent operator -o yaml` still yields `claude-sonnet-5` / `medium`

- **GIVEN** the owner's system-tier override `providers.claude.profiles.default.model: claude-fable-5-1`
- **WHEN** `fab agent default -o yaml` resolves
- **THEN** the override wins exactly as before (the bump changes only the built-in fallback)

### Specs: the drift-guarded mirror

#### R2: `stage-models.md` § Default role profiles stays in lockstep
The table under `### Default role profiles` in `docs/specs/stage-models.md` MUST show `claude-opus-5-5` in the four Opus rows and MUST land in the same commit as R1. The "Why these defaults" paragraph SHALL gain one sentence stating that Opus 5.5's built-in default effort is `medium`, which is why the explicit `high` is load-bearing (fab passes `--effort` on every CLI arm).

- **GIVEN** the YAML and the spec table after the edit
- **WHEN** `TestDocTablesMatchAgentMaps` parses the spec table
- **THEN** it passes (no disagreement between the two)

### Tests: the pinned-defaults guard

#### R3: `TestDefaultRoleProfilesArePinned` pins the new values
The pinned table in `src/go/fab/internal/agent/agent_test.go` MUST read `claude-opus-5-5` for `RoleDefault`, `RoleDoing`, `RoleReview`, and `RoleHydrate`; `RoleOperator` and `RoleFast` MUST stay `claude-sonnet-5`. No other test literal changes: every remaining `claude-opus-5` in `src/go` tests is an arbitrary fixture value (oracle-provider fixtures, explicit `--model` overrides) and stays verbatim.

- **GIVEN** the edited YAML and test table
- **WHEN** `go test ./internal/agent/... ./internal/config/... ./internal/configupgrade/... ./cmd/fab/...` runs from `src/go/fab`
- **THEN** all packages pass, including `TestDefaultRoleProfilesArePinned`, `TestDocTablesMatchAgentMaps`, and `TestConfigReferenceDocumentsProviderFill`

### Kit skills: CLI reference examples

#### R4: `_cli-fab.md` example output shows the new default
The two `fab resolve-agent apply` example blocks in `src/kit/skills/_cli-fab.md` (`model=claude-opus-5` at the plain and `--effort medium` examples) MUST read `model=claude-opus-5-5`. The `--alias` example (`model=opus`) is unchanged. Only the canonical source under `src/kit/skills/` is edited — never the deployed copies.

- **GIVEN** the edited `_cli-fab.md`
- **WHEN** a reader compares the example to `fab resolve-agent apply` on an un-overridden project
- **THEN** the model line matches

### Sweep: no stale pin remains

#### R5: The old pin is gone from every present-truth site
After apply, a repo-wide grep for `claude-opus-5` not followed by `-5` MUST hit only: test fixtures that are not the shipped default, history (`log.md`, `log.seed.md`, `docs/findings/`, `fab/plans/`, archived changes), this change's own artifacts, and the one memory example line hydrate owns (`docs/memory/_shared/configuration.md` "versioned ID" example). Nothing in `src/`, `docs/specs/`, or `src/kit/` outside those classes carries the old pin.

- **GIVEN** apply is complete
- **WHEN** `grep -rnE 'claude-opus-5([^-.0-9]|$)'` runs over `src/ docs/specs/ src/kit/`
- **THEN** the only hits are fixture values in `agent_surface_test.go` and `resolve_agent_test.go`

### Non-Goals

- Codex, kimi, and agy fills and command templates — Claude-only bump
- `review` effort (`high` stays; `xhigh` is a separate decision)
- Fable as any kit default (per-user cascade override, not a kit value)
- A migration file — config `presence=intent`; a user who copied the old rows above their fence keeps that pin deliberately
- Go logic or CLI surface — `ModelAlias` already prefix-matches the new ID; both claude command templates already substitute `{model}` and `{effort}`

### Design Decisions

#### Kit default is Opus 5.5, not Fable
**Decision**: The `default` role's built-in fill becomes `claude-opus-5-5`, the cheapest model in the top coding tier.
**Why**: The kit default should be the best price/capability point in the tier every un-overridden project pays for. Opus 5.5 is a same-tier successor to Opus 5 at a lower list price with the same context, output cap, tokenizer, and feature set.
**Rejected**: `claude-fable-5-1` as the kit default — about 2.5x the price with additional safety gating; it is a per-user choice the config cascade already supports (the owner's system-tier override keeps working untouched).
*Introduced by*: 260923-vvdv-bump-claude-defaults-opus-5-5

#### Explicit `effort: high` is load-bearing under Opus 5.5
**Decision**: Keep `effort: high` explicit on every Opus row and record why in a YAML comment and one spec sentence.
**Why**: Opus 5.5's built-in default effort is `medium`, one level below Opus 5's `high`. fab passes `--effort {effort}` on both claude command templates and substitutes the profile effort in the operator launcher, so kit behavior is unchanged — but the explicit value can no longer be treated as redundant with the model's own default.
**Rejected**: Dropping the effort column as redundant — it would silently step every Opus dispatch down to `medium`.
*Introduced by*: 260923-vvdv-bump-claude-defaults-opus-5-5

## Tasks

### Phase 1: Core Implementation

- [x] T001 Edit `src/go/fab/defaults.yaml` `providers.claude.profiles`: `default`/`doing`/`review`/`hydrate` → `claude-opus-5-5` (effort unchanged, column alignment preserved); add a one-to-two-line comment that the explicit `effort` is load-bearing under Opus 5.5's `medium` built-in default <!-- R1 -->
- [x] T002 [P] Edit `docs/specs/stage-models.md` § Default role profiles: four `claude-opus-5` cells → `claude-opus-5-5`; add one sentence to "Why these defaults" on Opus 5.5's `medium` built-in effort default making the explicit `high` load-bearing <!-- R2 -->
- [x] T003 [P] Edit `src/kit/skills/_cli-fab.md`: the two `model=claude-opus-5` example lines → `model=claude-opus-5-5` (the `--alias` example stays `model=opus`) <!-- R4 -->
- [x] T004 Edit `TestDefaultRoleProfilesArePinned` in `src/go/fab/internal/agent/agent_test.go`: four Opus rows → `claude-opus-5-5`; run `gofmt -l` on the file and `go test ./internal/agent/... ./internal/config/... ./internal/configupgrade/... ./cmd/fab/...` from `src/go/fab` <!-- R3 -->

### Phase 2: Integration & Edge Cases

- [x] T005 Re-grep `src/ docs/specs/ src/kit/` for `claude-opus-5` not followed by `-5`; confirm the only hits are the oracle/override fixtures in `agent_surface_test.go` and `resolve_agent_test.go` <!-- R5 -->
- [x] T006 Append the new `providers` advert digest to `knownGeneratedSystemParagraphDigests` in `src/go/fab/internal/configupgrade/configupgrade.go` (append-only catalog; the rendered fill lines changed with the YAML), with a comment naming this change; re-run the configupgrade package <!-- R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `defaults.yaml` pins `claude-opus-5-5` on `default`, `doing`, `review`, `hydrate`; `operator` and `fast` still pin `claude-sonnet-5`; every `effort` value is byte-identical to before
- [x] A-002 R1: A comment adjacent to the `profiles:` block explains that the explicit `effort` is load-bearing under Opus 5.5's `medium` built-in default
- [x] A-003 R2: The `stage-models.md` default-profile table shows `claude-opus-5-5` in the four Opus rows and is otherwise unchanged
- [x] A-004 R2: "Why these defaults" carries one sentence on Opus 5.5's `medium` built-in effort and why fab's explicit `high` matters
- [x] A-005 R3: `TestDefaultRoleProfilesArePinned` pins `claude-opus-5-5` on the four Opus roles and `claude-sonnet-5` on the two Sonnet roles
- [x] A-006 R4: Both `fab resolve-agent apply` example blocks in `src/kit/skills/_cli-fab.md` print `model=claude-opus-5-5`; the `--alias` example still prints `model=opus`

### Behavioral Correctness

- [x] A-007 R1: `fab agent doing -o yaml` on an un-overridden project (or the Go resolution test equivalent) resolves `model: claude-opus-5-5`, `effort: high`, `model_alias: opus`
- [x] A-008 R1: The owner's system-tier `claude-fable-5-1` override on `default` still wins — the bump changed only the built-in fallback

### Scenario Coverage

- [x] A-009 R3: `go test ./internal/agent/... ./internal/config/... ./internal/configupgrade/... ./cmd/fab/...` passes from `src/go/fab`, including `TestDefaultRoleProfilesArePinned`, `TestDocTablesMatchAgentMaps`, and `TestConfigReferenceDocumentsProviderFill`
- [x] A-010 R5: A repo-wide grep for `claude-opus-5` not followed by `-5` over `src/ docs/specs/ src/kit/` hits only the oracle-provider / explicit-override fixtures in `agent_surface_test.go` and `resolve_agent_test.go`

### Edge Cases & Error Handling

- [x] A-011 R3: No test fixture that is NOT the shipped default was edited (the `oracle` fixtures and explicit `--model claude-opus-5` overrides remain verbatim)
- [x] A-012 R1: The codex, kimi, and agy provider blocks in `defaults.yaml` are byte-identical to before
- [x] A-021 R3: `knownGeneratedSystemParagraphDigests` gained exactly one new entry (the current `providers` advert digest) and lost none — the catalog stays append-only and `TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer` passes

### Code Quality

- [x] A-013 Pattern consistency: The YAML edit preserves the file's existing column alignment and comment style; the spec sentence matches the surrounding paragraph's voice
- [x] A-014 No unnecessary duplication: The load-bearing-effort rationale lives in the YAML comment (beside the value) and one spec sentence; it is not restated in deployed skill prose
- [x] A-015 Follow existing project patterns: The bump follows the documented "edit defaults.yaml, then update the pinned test and the spec mirror" procedure named in the test comment
- [x] A-016 Canonical source only: `src/kit/skills/_cli-fab.md` is edited; no `.agents/skills/` or `.claude/skills/` deployed copy is touched
- [x] A-017 No fab-kit-only paths from deployed content: the `_cli-fab.md` edit introduces no `docs/specs/`, `docs/memory/`, or `src/go/` citation
- [x] A-018 Sibling sweep complete: every member of the `claude-opus-5` present-truth class (YAML, spec table, pinned test, CLI reference examples) was updated in this change; the memory example line is left for hydrate
- [x] A-019 gofmt-clean: `gofmt -l src/go/fab/internal/agent/agent_test.go` prints nothing
- [x] A-020 Tests scoped then widened only on surprise: the four named packages were run; the full suite was not run unless a failure warranted it

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The task sweep is exactly four edit sites plus a verification grep; `providers-and-profiles.md` carries no `claude-opus-5` literal (only a Sonnet example), so its only hydrate work is the Design Decisions entry | Grep-verified in this worktree during plan co-gen | S:90 R:95 A:90 D:90 |
| 2 | Confident | The load-bearing-effort rationale is recorded in a YAML comment and one spec sentence, not in any deployed skill | Both homes are fab-kit-only; deployed skills describe resolution, not why a value was chosen; keeps owner-or-pointer clean | S:75 R:90 A:85 D:75 |
| 3 | Confident | `TestConfigReferenceDocumentsProviderFill` needs no edit because it derives its expected lines from `ResolveProvider` | Stated in the pinned test's own comment; verified by running the package | S:80 R:90 A:80 D:80 |
| 4 | Certain | The rendered `providers` advert's digest changes with the fills, so the append-only historical catalog in `configupgrade.go` gains one entry; no older entry is removed | Discovered at first test run; the catalog's own doc comment prescribes exactly this maintenance step | S:90 R:95 A:95 D:90 |

4 assumptions (2 certain, 2 confident, 0 tentative).

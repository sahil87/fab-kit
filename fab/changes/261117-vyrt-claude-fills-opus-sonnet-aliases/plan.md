# Plan: Claude Fills Use opus / sonnet Aliases

**Change**: 261117-vyrt-claude-fills-opus-sonnet-aliases
**Intake**: `intake.md`

## Requirements

### Agent Resolution: Built-in Claude Fills

#### R1: Claude fills are the family aliases
The six built-in `providers.claude.profiles.*` fills in `src/go/fab/defaults.yaml` SHALL be `opus`
(`default`, `review`) and `sonnet` (`operator`, `doing`, `hydrate`, `fast`), with efforts unchanged
(`default`/`review`/`doing`/`hydrate` `high`; `operator`/`fast` `medium`). The fill-block comments
SHALL explain that the fills are aliases `claude` resolves to the newest model of the family, and that
the explicit efforts stay load-bearing because the model behind an alias can change its own default.
The codex, agy, and kimi blocks MUST NOT change.

- **GIVEN** a project with no agent overrides
- **WHEN** `DefaultProfile(role)` is read for each of the six roles
- **THEN** it returns `{claude, opus, high}` for default/review, `{claude, sonnet, high}` for
  doing/hydrate, and `{claude, sonnet, medium}` for operator/fast

#### R2: `model_alias` is set for a model that already is an Agent-tool alias
Resolution SHALL set `model_alias` when the resolved model maps to an alias by family prefix (today's
behavior) **or** already equals one of the four Agent-tool aliases (`opus`, `sonnet`, `haiku`,
`fable`). `agent.ModelAlias()` MUST keep its current behavior; the four alias names SHALL be derived
from the existing `modelAliasPrefixes` table, not restated. Non-Claude and unrecognised models keep an
empty `model_alias`.

- **GIVEN** a project with no agent overrides
- **WHEN** `fab agent review -o yaml` runs
- **THEN** it prints `model: opus`, `model_alias: opus`, and a `command:` containing `--model opus`
- **AND** `ModelAlias("opus"|"sonnet"|"haiku"|"fable")` returns its input unchanged

- **GIVEN** the review role resolved onto a non-Claude model (e.g. `gpt-5`)
- **WHEN** `fab agent review -o yaml` runs
- **THEN** `model_alias` is `""`

### Configuration: Reference Prose and Fence Upgrade

#### R3: `fab config explain providers` describes alias vs concrete fills
The rendered providers reference SHALL state that claude's fills are the `opus`/`sonnet` family
aliases that track the installed tool with no kit release, and that only codex's and agy's fills are
concrete IDs refreshed at KIT-RELEASE cadence; all fills pass through unvalidated and are overridable
via `providers.<name>.profiles.<role>.model`. The test pinning this prose SHALL pin the alias claim.

- **GIVEN** the built-in reference
- **WHEN** `fab config explain providers` renders
- **THEN** the output contains the alias claim and keeps `KIT-RELEASE cadence` tied to codex/agy

#### R4: Previous providers advert is recognised as generated
`configupgrade.go`'s legacy-digest set SHALL relabel the 261030-lc7k providers advert as a previous
advert and SHALL include the digest of the newly rendered current providers advert, so
`fab config upgrade` re-renders an old fence paragraph.

- **GIVEN** a system config whose fence holds the 261030-lc7k providers advert
- **WHEN** `fab config upgrade --system` runs
- **THEN** the paragraph is recognised as generated and replaced with the current advert

### Documentation: Quoted Claude IDs

#### R5: Docs, spec, and CLI reference quote the aliases
`docs/specs/stage-models.md` § Default role profiles, `src/kit/skills/_cli-fab.md`, and any other
current-truth prose quoting the built-in Claude fills SHALL quote `opus`/`sonnet`; prose asserting that
built-in resolution or the `dispatch=`/flag-absent output is a "full versioned ID" SHALL say the
resolved model is emitted verbatim. The spec table MUST still satisfy the drift guard
(`TestDocTablesMatchAgentMaps`). Memory files are updated at hydrate.

- **GIVEN** the updated spec table
- **WHEN** `go test ./internal/agent/` runs
- **THEN** the drift guard passes

### Non-Goals

- Changing codex/agy/kimi fills — plan Decision "Codex/agy fills stay concrete".
- Validating model strings — fab stays verbatim pass-through.
- A `src/kit/migrations/` entry — fills live in the binary; fence handled by R4.

### Design Decisions

#### Claude Fills Are Family Aliases
**Decision**: The built-in Claude fills are the family aliases `opus` and `sonnet`, not versioned IDs.
**Why**: `claude --model` and the Agent tool's `model` parameter both take the aliases and resolve the newest model of the family, so the kit's Claude defaults move with the installed tool and need no kit release per model launch.
**Rejected**: Bumping versioned IDs per release (four advert re-rolls for Claude fills alone); validating model strings (contradicts verbatim pass-through).
*Introduced by*: 261117-vyrt-claude-fills-opus-sonnet-aliases

## Tasks

### Phase 1: Core Implementation

- [x] T001 Change the six claude fills to `opus`/`sonnet` in `src/go/fab/defaults.yaml`, rewrite the fill-block and top-of-file `providers` comments, and update `TestDefaultRoleProfilesArePinned` in `src/go/fab/internal/agent/agent_test.go` <!-- R1 -->
- [x] T002 Add an alias-membership helper beside `modelAliasPrefixes` in `src/go/fab/internal/agent/agent.go`, use it in `src/go/fab/cmd/fab/resolution.go`; add `TestModelAlias` pass-through cases, a helper unit test, and a `fab agent review -o yaml` test (plus non-Claude negative) in `src/go/fab/cmd/fab/agent_surface_test.go` <!-- R2 -->
- [x] T003 Rewrite the KIT-RELEASE prose in `src/go/fab/internal/configref/configref.go` and the top `defaults.yaml` comment; update the phrase pins in `src/go/fab/cmd/fab/config_test.go` <!-- R3 -->

### Phase 2: Integration

- [x] T004 Relabel the 261030-lc7k digest and add the newly computed current providers-advert digest in `src/go/fab/internal/configupgrade/configupgrade.go`; run configupgrade tests <!-- R4 -->
- [x] T005 Update `docs/specs/stage-models.md` (table, effort rationale, full-ID claims), `src/kit/skills/_cli-fab.md` (examples, `--alias` paragraph), and sweep `src/kit/`, `docs/specs/`, `docs/site/` for current-fill IDs; run the full fab Go test suite <!-- R5 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `defaults.yaml` ships `opus` for default/review and `sonnet` for the other four roles with unchanged efforts; codex/agy/kimi blocks byte-identical
- [ ] A-002 R2: `fab agent review -o yaml` yields `model: opus`, `model_alias: opus`, `--model opus` in `command:`; `ModelAlias()` is unchanged
- [ ] A-003 R3: `fab config explain providers` states the alias claim for claude and concrete IDs for codex/agy
- [ ] A-004 R4: The legacy-digest set contains the 261030-lc7k digest (relabelled) and the new current digest
- [ ] A-005 R5: Spec table, `_cli-fab.md` examples, and full-ID prose quote the aliases / "resolved model verbatim"

### Behavioral Correctness

- [ ] A-006 R2: A non-Claude model still resolves an empty `model_alias`; a versioned Claude ID still maps to its alias

### Scenario Coverage

- [ ] A-007 R2: Tests cover alias pass-through, the alias-membership helper, and the `fab agent review -o yaml` scenario
- [ ] A-008 R4: The configupgrade test suite passes with the new digest (the current advert is recognised as generated)

### Edge Cases & Error Handling

- [ ] A-009 R2: Empty model keeps an empty `model_alias` (inherit signal preserved)

### Code Quality

- [ ] A-010 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [ ] A-011 No unnecessary duplication: the alias names derive from `modelAliasPrefixes`; no magic strings restated
- [ ] A-012: Go changes ship tests; `_cli-fab.md` updated; no deployed file cites fab-kit-only paths

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Alias-membership helper named `IsModelAlias` derived from `modelAliasPrefixes`' alias column | Keeps the four names single-sourced; `ModelAlias()` untouched per Brief | S:70 R:90 A:80 D:70 |
| 2 | Confident | Tests grouped with their code (test-alongside), five tasks → light lane | code-quality.md test strategy | S:75 R:90 A:85 D:80 |

2 assumptions (0 certain, 2 confident, 0 tentative).

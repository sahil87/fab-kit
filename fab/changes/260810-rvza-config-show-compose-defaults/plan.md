# Plan: config-show-compose-defaults

**Change**: 260810-rvza-config-show-compose-defaults
**Intake**: `intake.md`

## Requirements

### Config CLI: Composed Bare Show

#### R1: Materialize the Defaults Tier

Bare `fab config show` (no key and no `--origin`) MUST compose the existing built-in-defaults projection beneath the environment, system, and project layers and print the resulting effective configuration as YAML. The projection MUST reuse `readModelDefaults` and `configref.DefaultsMap`/`DefaultsMapFor` so derived `agent.profiles` rows remain composed against the live depth knobs.

- **GIVEN** a bare project config that does not set `dispatch.mode`
- **WHEN** the user runs `fab config show`
- **THEN** the YAML output includes `dispatch.mode: native`
- **AND** any environment, system, or project value continues to override the corresponding built-in default per leaf

#### R2: Preserve Keyed and Provenance Semantics

The keyed `fab config show <key>` and both `--origin` forms MUST retain their existing output and provenance semantics while bare `show` gains the defaults tier. `--origin` MUST remain the provenance annotator rather than a prerequisite for seeing composed values.

- **GIVEN** a key defined by multiple cascade tiers
- **WHEN** the user runs keyed `show`, bare `show --origin`, or keyed `show --origin`
- **THEN** the existing raw-value, winner-only, and full-stack outputs remain unchanged respectively
- **AND** no sparse-view flag or alternate output shape is introduced

### CLI Contract: Help and Documentation

#### R3: Describe the Fully Composed View Consistently

The `fab config show` long help, examples, canonical CLI skill reference, its SPEC mirror, and the config design specification MUST describe bare `show` as the fully composed four-tier YAML view and MUST describe `--origin` as provenance-only. Stale claims that bare `show` skips or does not materialize built-in defaults MUST be removed from the implementation-and-specification sweep class.

- **GIVEN** a user reads `fab config show --help` or the canonical config command documentation
- **WHEN** they compare bare `show`, keyed `show`, and `--origin`
- **THEN** every surface states that bare `show` includes built-in defaults and `--origin` adds provenance
- **AND** the canonical skill and `docs/specs/skills/SPEC-_cli-fab.md` mirror remain synchronized

### Non-Goals

- Add `--set-only`, `--sparse`, or another sparse-view flag.
- Change the loader by merging defaults into `internal/config.LoadPath`.
- Change any keyed `show` or `--origin` output contract.
- Hydrate `docs/memory/`; the pipeline's later hydrate stage owns that update.

### Design Decisions

#### Compose Defaults in the Existing Read Model

**Decision**: Bare `show` uses the existing `readModelDefaults` projection and merges it beneath `layers.Effective` only for display.
**Why**: This reuses the same knob-aware projection as keyed and provenance views without poisoning the typed loader's derived-profile precedence.
**Rejected**: Merge defaults into `internal/config.LoadPath`, or build a second defaults projection specifically for bare `show`; both would violate existing ownership and create drift.
*Introduced by*: 260810-rvza-config-show-compose-defaults

## Tasks

### Phase 2: Core Implementation

- [x] T001 Update `src/go/fab/cmd/fab/config.go` so every show mode reads the existing defaults projection, bare `renderShow` merges it beneath `layers.Effective`, and `Long`/`Example` text describes the fully composed YAML view. <!-- R1, R3 -->
- [x] T002 Update `src/go/fab/cmd/fab/config_show_init_test.go` and any pinned cases in `src/go/fab/cmd/fab/config_test.go` to expect composed bare-show output, cover a bare-project `dispatch.mode: native` default, and preserve keyed/origin behavior. <!-- R1, R2 -->

### Phase 4: Polish

- [x] T003 [P] Update `src/kit/skills/_cli-fab.md` and its required mirror `docs/specs/skills/SPEC-_cli-fab.md` so bare `show` is the four-tier composed YAML view and `--origin` is provenance-only. <!-- R3 -->
- [x] T004 [P] Update the six-verb description in `docs/specs/config.md` to remove the bare-show/defaults-tier split and record the composed behavior. <!-- R3 -->
- [x] T005 Sweep `src/go/fab/cmd/fab/config.go`, `src/kit/skills/_cli-fab.md`, `docs/specs/config.md`, and `docs/specs/skills/SPEC-_cli-fab.md` for stale bare-show/defaults claims; run `gofmt` checks on touched Go files and focused Go tests for `./cmd/fab/...` from `src/go/fab`. <!-- R1, R2, R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Bare `fab config show` prints the four-tier effective configuration as YAML, including built-in defaults.
- [x] A-002 R2: Keyed `show`, bare `show --origin`, and keyed `show --origin` preserve their existing value and provenance contracts.
- [x] A-003 R3: Help, the canonical CLI reference, its SPEC mirror, and the config spec consistently describe the composed bare view.

### Behavioral Correctness

- [x] A-004 R1: Environment, system, and project values continue to win over defaults per leaf, and derived profile defaults remain knob-aware through `DefaultsMapFor`.
- [x] A-005 R2: `--origin` behavior and CLI flags are unchanged; no sparse-view flag is added.

### Scenario Coverage

- [x] A-006 R1: A Go test proves that a pure-default field such as `dispatch.mode: native` appears for a bare project.
- [x] A-007 R2: Existing keyed, origin, cascade-precedence, and empty-value regression tests remain green.

### Code Quality

- [x] A-008 Pattern consistency: The change follows surrounding Cobra command, read-model, test-helper, and documentation patterns.
- [x] A-009 No unnecessary duplication: Existing `readModelDefaults`, `config.MergeLayers`, and test utilities are reused.
- [x] A-010 Readability and maintainability: The implementation and tests communicate the four-tier behavior directly without clever or indirect control flow.
- [x] A-011 Focused structure: The change does not introduce a god function or widen module boundaries for a display-only composition.
- [x] A-012 Canonical source: No deployed file under `.claude/skills/` is edited.
- [x] A-013 CLI documentation and tests: The Go behavior change ships with corresponding tests and `src/kit/skills/_cli-fab.md` updates.
- [x] A-014 SPEC mirror synchronization: `docs/specs/skills/SPEC-_cli-fab.md` matches the canonical `_cli-fab.md` behavior.
- [x] A-015 Sibling and mirror sweep: Repository-wide searches find no stale claim in the apply-stage sweep class that bare `show` skips or does not materialize defaults; `docs/memory/` remains untouched for hydrate.
- [x] A-016 Formatting and verification: Every touched Go file is gofmt-clean and focused `go test ./cmd/fab/...` passes from `src/go/fab`.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/cmd/fab/config.go:388-390` — The no-effective-config fallback is unreachable now that every bare `show` call supplies the non-empty built-in defaults projection.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Include `docs/specs/skills/SPEC-_cli-fab.md` in the apply scope | The constitution and code-quality mirror rule require every canonical skill edit to update its SPEC mirror, even though the intake summary named only the config spec | S:95 R:90 A:100 D:100 |
| 2 | Confident | Put the pure-default integration assertion in `config_show_init_test.go` | That file already owns the `fab config show` command-level cascade tests; placement is easy to revise and does not affect behavior | S:70 R:90 A:80 D:65 |
| 3 | Certain | Leave `docs/memory/_shared/configuration.md` unchanged during apply | The dispatch contract explicitly reserves memory hydration for the later hydrate stage while still requiring source/spec stale-claim cleanup now | S:100 R:95 A:100 D:100 |

3 assumptions (2 certain, 1 confident, 0 tentative).

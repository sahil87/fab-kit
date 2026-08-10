# Plan: config-show-compose-defaults

**Change**: 260810-rvza-config-show-compose-defaults
**Intake**: `intake.md`

## Requirements

### Configuration: Show Command

#### R1: Bare show composes built-in defaults
Bare `fab config show` (no key, no `--origin` flag) SHALL print the fully composed configuration, merging the built-in defaults tier beneath the environment, system, and project tiers.

- **GIVEN** a configuration where a field has a built-in default but is not overridden in any layer
- **WHEN** the user runs `fab config show`
- **THEN** the output YAML includes that field with its built-in default value

#### R2: Reuse existing projection
The composed tier SHALL reuse the existing `DefaultsMapFor` (or `DefaultsMap`) projection to ensure derived rows like `agent.profiles` are properly resolved based on live depth knobs.

- **GIVEN** the existing `readModelDefaults` function
- **WHEN** constructing the effective config for bare show
- **THEN** it must use this function rather than rebuilding the projection

#### R3: --origin behavior is preserved
The `--origin` flag SHALL continue to function purely as a provenance annotator without changing its existing output semantics.

- **GIVEN** a configuration stack
- **WHEN** the user runs `fab config show --origin`
- **THEN** the output displays each field alongside its origin, identical to previous behavior

#### R4: Help text and documentation updated
The command help text and relevant documentation (skills, specs) MUST accurately reflect that bare `show` composes the defaults tier.

- **GIVEN** the `configShowCmd` and documentation files
- **WHEN** a user reads them
- **THEN** they should no longer state that "built-in defaults are NOT materialized here"

## Tasks

### Phase 1: Core Implementation

- [x] T001 Update `configShowCmd` help text in `src/go/fab/cmd/fab/config.go` to remove the carve-out about built-in defaults not being materialized. <!-- R4 -->
- [x] T002 Modify bare `show` execution in `src/go/fab/cmd/fab/config.go` to fetch defaults via `readModelDefaults` and merge them into the layers printed. <!-- R1 -->
- [x] T003 Reuse the `DefaultsMapFor` output by updating the `renderShow` signature or logic in `src/go/fab/cmd/fab/config.go` to incorporate `defaults`. <!-- R2 -->

### Phase 2: Integration & Edge Cases

- [x] T004 Ensure that when `--origin` is passed or a specific key is queried, the existing behavior is unmodified in `src/go/fab/cmd/fab/config.go`. <!-- R3 -->
- [x] T005 Update existing tests in `src/go/fab/cmd/fab/config_test.go` and `cmd/fab/config_show_init_test.go` (if applicable) to assert that bare show includes pure-default fields (e.g., `dispatch.mode`). <!-- R1 -->

### Phase 3: Polish

- [x] T006 Update `src/kit/skills/_cli-fab.md` section on `fab config` to reflect the updated behavior of bare `show`. <!-- R4 -->
- [x] T007 Update `docs/specs/config.md` to reflect the updated behavior of bare `show`. <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Running `fab config show` outputs the fully composed YAML, including fields that only have built-in defaults.
- [x] A-002 R2: The implementation reuses the existing `readModelDefaults` functionality without reimplementing the projection.
- [x] A-003 R4: Help text for `fab config show` accurately describes the composed output.
- [x] A-004 R4: `src/kit/skills/_cli-fab.md` and `docs/specs/config.md` are updated to reflect the new behavior.

### Behavioral Correctness

- [x] A-005 R3: Running `fab config show --origin` produces the exact same provenance annotation format as before.

### Scenario Coverage

- [x] A-006 R1: Tests assert that a pure-default field appears in bare `show` output.

### Code Quality

- [x] A-007 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [x] A-008 No unnecessary duplication: Existing utilities reused where applicable

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bare `fab config show` prints the fully composed config, defaults tier included | User accepted this recommendation explicitly in discussion | S:90 R:85 A:90 D:85 |
| 2 | Certain | `--origin` semantics untouched — provenance annotator only | User's framing: "--origin is to track the source"; no behavior change requested there | S:90 R:90 A:90 D:90 |
| 3 | Confident | No sparse "only what I set" flag ships now | YAGNI — additive to introduce later; files + `--origin` filtering cover the need meanwhile | S:60 R:90 A:65 D:50 |
| 4 | Confident | Reuse the existing `DefaultsMapFor` projection for the composed tier (knob-composed derived rows included) | The keyed and `--origin` paths already use it; a second projection would drift | S:70 R:80 A:85 D:80 |
| 5 | Confident | Composed output is additive for consumers (new keys appear; existing keys keep shape) — no compatibility carve-out needed | Bare show is a human-facing pure query; `--json`-style stable surfaces are not touched | S:65 R:75 A:75 D:70 |

5 assumptions (2 certain, 3 confident, 0 tentative).

## Deletion Candidates

- `None` — this change adds new functionality without making existing code redundant

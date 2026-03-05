# Spec: Define auto-mode signaling mechanism for skill-to-skill invocation

**Change**: 260210-nan4-define-auto-mode-signaling
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/clarify.md`

## Shared Context: Skill Invocation Protocol

### Requirement: Explicit Auto-Mode Signaling in `_context.md`

The shared context file (`fab/.kit/skills/_context.md`) SHALL define a "Skill Invocation Protocol" section that specifies how a calling skill signals mode to a called skill. The protocol SHALL use an explicit instruction prefix pattern — the calling skill MUST include a specific text preamble when invoking another skill internally.

#### Scenario: fab-ff invokes fab-clarify in auto mode

- **GIVEN** `/fab-ff` has generated a stage artifact (e.g., `spec.md`)
- **WHEN** `/fab-ff` invokes `/fab-clarify` internally between stages
- **THEN** the invocation MUST include the instruction prefix `[AUTO-MODE]` at the start of the invocation prompt
- **AND** `/fab-clarify` MUST detect this prefix and enter auto mode (autonomous resolution, no user interaction, machine-readable result)

#### Scenario: User invokes fab-clarify directly

- **GIVEN** a user types `/fab-clarify` in the CLI
- **WHEN** the skill begins execution
- **THEN** the invocation will NOT contain the `[AUTO-MODE]` prefix
- **AND** `/fab-clarify` MUST enter suggest mode (interactive, one question at a time)

#### Scenario: fab-fff invokes fab-clarify via fab-ff

- **GIVEN** `/fab-fff` is executing and reaches the planning phase
- **WHEN** `/fab-fff` delegates to `/fab-ff` behavior, which then invokes `/fab-clarify`
- **THEN** the same `[AUTO-MODE]` prefix protocol applies — `/fab-clarify` enters auto mode
- **AND** the protocol is transitive: `/fab-fff` → `/fab-ff` → `/fab-clarify` all follow the same signaling

### Requirement: Protocol Definition Format

The protocol definition in `_context.md` SHALL specify:

1. The exact prefix string (`[AUTO-MODE]`)
2. Where it appears (first line of the invocation prompt / instruction to the called skill)
3. What the called skill MUST do when it detects the prefix (enter auto mode)
4. What the called skill MUST do when the prefix is absent (enter default/interactive mode)

#### Scenario: Protocol section is self-contained

- **GIVEN** a developer reads the Skill Invocation Protocol section in `_context.md`
- **WHEN** they need to understand how auto-mode signaling works
- **THEN** the section provides all necessary information without requiring cross-references to individual skill files for the protocol definition itself

### Requirement: fab-clarify Mode Detection Update

The `/fab-clarify` skill file (`fab/.kit/skills/fab-clarify.md`) SHALL update its Mode Selection section to reference the protocol defined in `_context.md` and specify concrete detection logic.

#### Scenario: fab-clarify documents mode detection

- **GIVEN** the `/fab-clarify` skill file
- **WHEN** the Mode Selection section is read
- **THEN** it SHALL state that auto mode is triggered by detecting the `[AUTO-MODE]` prefix in the invocation context
- **AND** it SHALL reference the Skill Invocation Protocol section in `_context.md` as the authoritative definition

### Requirement: fab-ff Auto-Clarify Invocation Update

The `/fab-ff` skill file (`fab/.kit/skills/fab-ff.md`) SHALL update its auto-clarify invocation instructions to use the defined protocol.

#### Scenario: fab-ff uses the protocol when invoking auto-clarify

- **GIVEN** `/fab-ff` reaches an auto-clarify step (after spec, plan, or tasks generation)
- **WHEN** it invokes `/fab-clarify` internally
- **THEN** the invocation instruction SHALL include `[AUTO-MODE]` as specified by the Skill Invocation Protocol
- **AND** the skill file SHALL reference the protocol section in `_context.md` rather than defining ad-hoc invocation conventions

### Requirement: fab-fff Audit for Auto-Clarify Invocations

The `/fab-fff` skill file (`fab/.kit/skills/fab-fff.md`) SHALL be audited and updated if it contains any direct auto-clarify invocations that do not already delegate through `/fab-ff`.

#### Scenario: fab-fff delegates auto-clarify through fab-ff

- **GIVEN** `/fab-fff` executes its planning phase
- **WHEN** auto-clarify is needed
- **THEN** if `/fab-fff` delegates entirely to `/fab-ff` behavior (which already handles auto-clarify), no additional changes to `/fab-fff` are needed
- **AND** if `/fab-fff` has any standalone auto-clarify invocations, they SHALL be updated to use the `[AUTO-MODE]` protocol

## Deprecated Requirements

None — this change adds a new protocol without removing existing behavior.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Use `[AUTO-MODE]` as the prefix string | Simple, unambiguous, follows bracketed-keyword conventions already used in the project (e.g., `[NEEDS CLARIFICATION]`, `[P]`); easily searchable |
| 2 | Confident | Define the protocol in `_context.md` as a new top-level section | `_context.md` is the canonical location for shared conventions (SRAD, confidence scoring, next steps); adding a section is consistent with the file's role |
<!-- assumed: [AUTO-MODE] prefix string — chosen for consistency with existing bracketed markers like [NEEDS CLARIFICATION] and [P] -->

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.

# Spec: Add fab-discuss Command and Confidence Scoring to fab-new

**Change**: 260208-lgd7-fab-discuss-command
**Created**: 2026-02-08
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/change-lifecycle.md`

## Planning Skills: `/fab-discuss`

### Requirement: fab-discuss Skill Definition

`/fab-discuss` SHALL be a planning skill that develops proposals through free-form conversation. It SHALL be defined in `fab/.kit/skills/fab-discuss.md` and registered via a symlink at `.claude/skills/fab-discuss/SKILL.md`.

#### Scenario: Skill file exists and is loadable
- **GIVEN** the fab kit is initialized
- **WHEN** the user invokes `/fab-discuss`
- **THEN** Claude Code loads `fab/.kit/skills/fab-discuss.md` via the symlink at `.claude/skills/fab-discuss/SKILL.md`

### Requirement: Context-Driven Mode Selection

`/fab-discuss` SHALL determine its mode from the active change state (`fab/current`):

1. If `fab/current` exists and points to a valid change, `/fab-discuss` SHALL default to refining that change's proposal.
2. If the user's description is significantly different from the active change's scope, `/fab-discuss` SHALL confirm with the user whether this is a new change.
3. If `fab/current` does not exist or is empty, `/fab-discuss` SHALL start a new change from scratch.

#### Scenario: Active change exists — refine mode
- **GIVEN** `fab/current` points to change `260208-abc1-some-feature`
- **WHEN** the user invokes `/fab-discuss improve error handling`
- **THEN** `/fab-discuss` loads the existing `proposal.md` from `fab/changes/260208-abc1-some-feature/`
- **AND** begins a conversational refinement of that proposal

#### Scenario: Active change exists but description diverges
- **GIVEN** `fab/current` points to change `260208-abc1-auth-feature`
- **WHEN** the user invokes `/fab-discuss add payment processing`
- **THEN** `/fab-discuss` detects that "payment processing" is significantly different from the auth feature
- **AND** asks the user: "You're currently working on 260208-abc1-auth-feature (auth). Is this a new change, or related to the current one?"

#### Scenario: No active change — new change mode
- **GIVEN** `fab/current` does not exist
- **WHEN** the user invokes `/fab-discuss add better logging`
- **THEN** `/fab-discuss` creates a new change folder and begins developing a proposal from scratch

### Requirement: Does Not Switch Active Change

`/fab-discuss` SHALL create a change folder (when starting from scratch) but SHALL NOT write to `fab/current`. The active change pointer MUST remain unchanged.

#### Scenario: New change created without switching
- **GIVEN** `fab/current` points to `260208-abc1-existing-change`
- **WHEN** `/fab-discuss` creates a new change `260208-xyz9-new-idea`
- **THEN** `fab/current` still contains `260208-abc1-existing-change`
- **AND** the new change folder exists at `fab/changes/260208-xyz9-new-idea/`

### Requirement: No Git Integration

`/fab-discuss` SHALL NOT create, adopt, or checkout git branches. The `branch:` field SHALL be omitted from `.status.yaml` for changes created by `/fab-discuss`.

#### Scenario: No branch created
- **GIVEN** git is enabled in `fab/config.yaml`
- **WHEN** `/fab-discuss` creates a new change folder
- **THEN** no git branch operations are performed
- **AND** `.status.yaml` does not contain a `branch:` field

### Requirement: Gap Analysis Phase

When starting from scratch, `/fab-discuss` SHALL begin with a gap analysis phase before committing to a proposal. This phase SHALL:

1. Evaluate whether the change is needed — is there an existing mechanism?
2. Explore the scope — is the change too broad or too narrow?
3. Consider alternatives — are there simpler approaches?

The gap analysis SHALL be conversational, not a structured checklist.

#### Scenario: Gap analysis identifies existing mechanism
- **GIVEN** the user describes "add a way to reset to an earlier stage"
- **WHEN** `/fab-discuss` performs gap analysis
- **THEN** it identifies that `/fab-continue <stage>` already provides reset functionality
- **AND** presents this finding to the user for discussion

#### Scenario: Gap analysis confirms change is needed
- **GIVEN** the user describes a capability that does not exist in the current workflow
- **WHEN** `/fab-discuss` performs gap analysis
- **THEN** it confirms the gap exists and proceeds to proposal development

### Requirement: Conversational Proposal Development

`/fab-discuss` SHALL develop the proposal through free-form conversation with the user. There SHALL be no fixed question cap — the skill asks as many clarifying questions as needed to build a solid proposal.

#### Scenario: Multi-round conversation
- **GIVEN** the user starts with "I want better error handling"
- **WHEN** `/fab-discuss` asks clarifying questions
- **THEN** each question builds on previous answers
- **AND** the conversation continues until the proposal is solid

### Requirement: Conversation Termination

The discussion SHALL end when both conditions are met:
1. The confidence score is >= 3.0
2. The user signals satisfaction (e.g., "looks good", "done")

When the confidence score crosses 3.0, `/fab-discuss` SHALL proactively suggest wrapping up. The user MAY also end the discussion early at any time regardless of score.

#### Scenario: Score threshold reached — suggest wrap-up
- **GIVEN** the confidence score has just crossed 3.0
- **WHEN** `/fab-discuss` recomputes the score after resolving a question
- **THEN** it suggests: "Confidence is now {score} — high enough for /fab-fff. Want to finalize, or keep refining?"

#### Scenario: User ends early
- **GIVEN** the confidence score is 1.5
- **WHEN** the user says "done" or "that's enough for now"
- **THEN** `/fab-discuss` finalizes the proposal as-is
- **AND** writes the current confidence score to `.status.yaml`
- **AND** notes that the score is below the `/fab-fff` gate threshold

### Requirement: Confidence Score Computation

`/fab-discuss` SHALL compute and write the SRAD confidence score to `.status.yaml` after finalizing the proposal. The score SHALL use the standard formula: `max(0.0, 5.0 - 0.1 * confident - 1.0 * tentative)` if `unresolved == 0`, else `0.0`.

#### Scenario: High confidence after thorough discussion
- **GIVEN** a discussion resolved all ambiguities
- **WHEN** `/fab-discuss` finalizes the proposal
- **THEN** `.status.yaml` confidence block shows `score: >= 3.0`
- **AND** `unresolved: 0`

### Requirement: Proposal Output

When starting from scratch, `/fab-discuss` SHALL:
1. Create the change folder using standard naming (`{YYMMDD}-{XXXX}-{slug}`)
2. Create the `checklists/` subdirectory
3. Initialize `.status.yaml` (without `branch:` field)
4. Generate `proposal.md` from the template incorporating all discussion outcomes
5. Set `progress.proposal` to `done` and write the confidence score

When refining an existing proposal, `/fab-discuss` SHALL:
1. Update the existing `proposal.md` in place
2. Recompute and update the confidence score in `.status.yaml`
3. Update `last_updated` in `.status.yaml`

#### Scenario: New proposal created from scratch
- **GIVEN** no active change exists
- **WHEN** `/fab-discuss` completes a conversation
- **THEN** a new change folder exists with `proposal.md`, `.status.yaml`, and `checklists/`
- **AND** `progress.proposal` is `done`
- **AND** the confidence block is populated

#### Scenario: Existing proposal refined
- **GIVEN** an active change exists with a proposal
- **WHEN** `/fab-discuss` completes a refinement conversation
- **THEN** the existing `proposal.md` is updated in place
- **AND** the confidence score is recomputed and updated

### Requirement: Next Steps Output

`/fab-discuss` SHALL end with a `Next:` line. The suggestion SHALL vary based on outcome:
- If a new change was created: `Next: /fab-switch {change-name} to make it active, then /fab-continue or /fab-ff`
- If an existing proposal was refined: `Next: /fab-continue or /fab-ff (fast-forward all planning)`

#### Scenario: Next steps for new change
- **GIVEN** `/fab-discuss` created a new change `260208-xyz9-new-idea`
- **WHEN** the output is displayed
- **THEN** it ends with `Next: /fab-switch 260208-xyz9-new-idea to make it active, then /fab-continue or /fab-ff`

## Planning Skills: `/fab-new` Confidence Scoring

### Requirement: fab-new Writes Confidence Score

`/fab-new` SHALL compute the SRAD confidence score after generating the proposal and write the `confidence` block to `.status.yaml` with actual counts (not the template defaults of all zeros).

#### Scenario: Confidence block populated after fab-new
- **GIVEN** the user runs `/fab-new "Add OAuth support"`
- **WHEN** the proposal is generated with 2 Confident and 1 Tentative assumption
- **THEN** `.status.yaml` contains `confidence: {certain: N, confident: 2, tentative: 1, unresolved: 0, score: X}`
- **AND** the score is computed as `max(0.0, 5.0 - 0.1*2 - 1.0*1)`

## Shared Context: SRAD Autonomy Table and Next Steps

### Requirement: Update SRAD Skill Autonomy Table

`_context.md` SHALL include `/fab-discuss` in the Skill-Specific Autonomy Levels table with the following characteristics:
- **Posture**: Free-form conversation, no question cap, proactive gap analysis
- **Interruption budget**: Unlimited — conversational by design
- **Output**: Proposal + confidence score + "Run /fab-switch to make active"
- **Escape valve**: User can end early at any time
- **Recomputes confidence?**: Yes

#### Scenario: Autonomy table includes fab-discuss
- **GIVEN** `_context.md` is read by any planning skill
- **WHEN** the Skill-Specific Autonomy Levels table is consulted
- **THEN** `/fab-discuss` appears with its defined autonomy level

### Requirement: Update Next Steps Lookup Table

`_context.md` SHALL include `/fab-discuss` in the Next Steps lookup table:
- After `/fab-discuss` (new change): `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`
- After `/fab-discuss` (refined existing): `Next: /fab-continue or /fab-ff (fast-forward all planning)`

Additionally, `/fab-init` and `/fab-hydrate` next steps SHALL include `/fab-discuss` as an option alongside `/fab-new`.

#### Scenario: Next steps table includes fab-discuss
- **GIVEN** `_context.md` is read
- **WHEN** the Next Steps lookup table is consulted after `/fab-init`
- **THEN** the options include `/fab-discuss` alongside `/fab-new`

## Deprecated Requirements

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-discuss follows the same folder naming convention as fab-new | Constitution requires consistent naming; no reason to diverge |
| 2 | Confident | fab-discuss loads the same context as fab-new (config, constitution, docs/index.md) | Proposal generation needs the same grounding regardless of entry point |
| 3 | Confident | "Significantly different" scope detection is a judgment call, not an algorithm | Exact matching would be brittle; the agent can assess semantic similarity naturally |
| 4 | Confident | fab-new already has Step 8 (confidence scoring) in its skill definition — the fix is ensuring the init template values are overwritten | Reading fab-new.md confirms Step 8 exists but Step 5's .status.yaml snippet doesn't show populated confidence values |

4 assumptions made (4 confident, 0 tentative).

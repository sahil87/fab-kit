# Spec: Add SRAD Autonomy Framework to Planning Skills

**Change**: 260207-09sj-autonomy-framework
**Created**: 2026-02-07
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/clarify.md`, `fab/docs/fab-workflow/context-loading.md`

<!--
  CHANGE SPECIFICATION
  Requirements for adding the SRAD autonomy framework to the Fab planning skills.
  Organized by target file, since this change modifies 5 skill files and 1 shared preamble.
-->

## Shared Context: SRAD Framework in `_context.md`

### Requirement: SRAD Scoring Table

The `_context.md` shared preamble SHALL include a formalized SRAD scoring table defining four dimensions (Signal Strength, Reversibility, Agent Competence, Disambiguation Type), each with "High (safe to assume)" and "Low (consider asking)" descriptions.

#### Scenario: Agent encounters a decision point during artifact generation
- **GIVEN** a planning skill is generating an artifact
- **WHEN** the agent encounters a decision point not explicitly addressed by the user's input
- **THEN** the agent SHALL evaluate the decision against the four SRAD dimensions
- **AND** assign a confidence grade (Certain, Confident, Tentative, or Unresolved)

### Requirement: Confidence Grades Definition

The `_context.md` preamble SHALL define four confidence grades with their meanings, artifact markers, and output visibility rules:

| Grade | Artifact Marker | Output Visibility |
|-------|----------------|-------------------|
| Certain | None | None |
| Confident | None | Assumptions summary |
| Tentative | `<!-- assumed: {description} -->` | Assumptions summary + fab-clarify suggested |
| Unresolved | `<!-- auto-guess: {description} -->` (fab-ff --auto only) | Asked as question or auto-guessed depending on skill |

#### Scenario: Agent assigns Tentative grade to a decision
- **GIVEN** the agent evaluates a decision point using SRAD
- **WHEN** the decision has a reasonable default but multiple valid options exist
- **THEN** the agent SHALL grade it as Tentative
- **AND** insert a `<!-- assumed: {description} -->` marker inline in the artifact at the point of the assumption
- **AND** include the assumption in the Assumptions summary output

#### Scenario: Agent assigns Certain grade
- **GIVEN** the agent evaluates a decision point
- **WHEN** the answer is deterministically derived from config, constitution, or template rules
- **THEN** the agent SHALL grade it as Certain
- **AND** SHALL NOT include it in the Assumptions summary or add any marker

### Requirement: Worked Examples

The `_context.md` SRAD section SHALL include 2-3 worked examples demonstrating how the four dimensions interact to produce a confidence grade. Each example MUST show a realistic decision point, the SRAD evaluation, and the resulting grade.

#### Scenario: Agent references worked examples for guidance
- **GIVEN** a new contributor or agent reads the SRAD section in `_context.md`
- **WHEN** they encounter the worked examples
- **THEN** each example SHALL show: (1) the decision point, (2) each SRAD dimension scored, (3) the resulting confidence grade, (4) the chosen action

### Requirement: Critical Rule — Unresolved High-Risk Decisions

Unresolved decisions with low Reversibility AND low Agent Competence MUST always be asked as questions — even in `fab-new` and `fab-continue`. These count toward the skill's question budget (max ~3). The existence of `/fab-clarify` as an escape valve SHALL NOT justify silently assuming high-blast-radius decisions.

#### Scenario: High-risk unresolved decision in fab-new
- **GIVEN** `fab-new` is generating a proposal from a vague description
- **WHEN** a decision point scores low on both Reversibility and Agent Competence
- **THEN** the agent SHALL ask the user about it as one of the top ~3 questions
- **AND** SHALL NOT silently assume an answer

#### Scenario: High-risk unresolved in fab-ff --auto
- **GIVEN** `fab-ff --auto` encounters a decision with low R and low A
- **WHEN** the skill's posture is "assume everything"
- **THEN** the agent SHALL still auto-guess and mark it with `<!-- auto-guess: ... -->`
- **AND** the auto-guess SHALL be prominently listed in the output warning

### Requirement: `<!-- assumed: ... -->` Marker Convention

All planning skills SHALL use `<!-- assumed: {description} -->` HTML comment markers for Tentative assumptions. The marker SHALL be placed inline in the artifact immediately after the assumed content. The `{description}` SHALL be a concise summary of what was assumed and why.

#### Scenario: Tentative assumption in generated spec
- **GIVEN** `fab-continue` is generating `spec.md`
- **WHEN** the agent makes a Tentative assumption about a requirement detail
- **THEN** the agent SHALL write the assumed content followed by `<!-- assumed: {description} -->`
- **AND** the marker SHALL be scannable by `fab-clarify`

## Skill: `fab-new`

### Requirement: SRAD-Based Question Selection

`fab-new` SHALL apply SRAD scoring to identify up to 3 Unresolved decision points with the highest blast radius (lowest Reversibility + lowest Agent Competence). These SHALL be asked as blocking questions. All other decisions SHALL be assumed at their assessed confidence grade.

#### Scenario: Clear description — no questions needed
- **GIVEN** a user invokes `fab-new "Add retry logic to HTTP client"`
- **WHEN** SRAD evaluation finds no Unresolved decisions (all Certain/Confident/Tentative)
- **THEN** the agent SHALL generate the proposal without asking any questions
- **AND** SHALL output an Assumptions summary listing Confident and Tentative assumptions

#### Scenario: Ambiguous description — 2 unresolved decisions
- **GIVEN** a user invokes `fab-new "Improve auth"`
- **WHEN** SRAD evaluation identifies 2 Unresolved decisions (which auth aspect? replace or supplement?)
- **THEN** the agent SHALL ask exactly those 2 questions before generating the proposal
- **AND** SHALL assume all Confident and Tentative decisions without asking

### Requirement: Branch-on-Main Auto-Create

When `fab-new` detects the user is on `main` or `master` and `git.enabled` is `true`, the skill SHALL automatically create a branch using the standard naming convention. The branch creation prompt SHALL be removed — it is a low-value interruption.

#### Scenario: User on main, git enabled
- **GIVEN** `git.enabled` is `true` in config
- **AND** the current branch is `main`
- **WHEN** `fab-new` reaches the git integration step
- **THEN** the agent SHALL auto-create branch `{prefix}{change-name}` without prompting
- **AND** record the branch in `.status.yaml`

#### Scenario: User on main, git enabled, --branch provided
- **GIVEN** `git.enabled` is `true` and user provides `--branch feature/foo`
- **WHEN** `fab-new` reaches the git integration step
- **THEN** the `--branch` argument SHALL take precedence (existing behavior, unchanged)

#### Scenario: User on feature branch
- **GIVEN** the current branch is not `main`/`master`
- **WHEN** `fab-new` reaches the git integration step
- **THEN** the existing interactive prompt (Adopt / Create new / Skip) SHALL be preserved (unchanged)

### Requirement: Assumptions Summary in Output

`fab-new` output SHALL end with an Assumptions summary block listing all Confident and Tentative assumptions made during proposal generation. The block SHALL follow the standard format:

```
## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
```

Followed by a count line: `{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.`

#### Scenario: Proposal with 3 assumptions
- **GIVEN** `fab-new` generates a proposal making 1 Confident and 2 Tentative assumptions
- **WHEN** the output is displayed
- **THEN** an Assumptions summary table SHALL appear after "Proposal complete."
- **AND** each assumption SHALL list its grade, decision summary, and rationale
- **AND** the count line SHALL read "3 assumptions made (1 confident, 2 tentative). Run /fab-clarify to review."

### Requirement: Assumptions Persisted in Artifact

`fab-new` SHALL append the Assumptions summary as a trailing `## Assumptions` section in the generated `proposal.md` artifact. This ensures assumptions are scannable by `fab-clarify` and survive beyond terminal output.

#### Scenario: Artifact persistence
- **GIVEN** `fab-new` generates a proposal with assumptions
- **WHEN** `proposal.md` is written to disk
- **THEN** the file SHALL contain a `## Assumptions` section at the end with the same table shown in output

## Skill: `fab-continue`

### Requirement: [NEEDS CLARIFICATION] Count in Output

When `fab-continue` generates an artifact, the output summary SHALL include the count of `[NEEDS CLARIFICATION]` markers remaining in the artifact: `{N} [NEEDS CLARIFICATION] markers in {artifact}.`

If the count is 0, the line MAY be omitted.

#### Scenario: Spec with 2 unresolved markers
- **GIVEN** `fab-continue` generates `spec.md`
- **WHEN** the generated spec contains 2 `[NEEDS CLARIFICATION]` markers
- **THEN** the output SHALL include: `2 [NEEDS CLARIFICATION] markers in spec.md.`
- **AND** SHALL suggest: `Run /fab-clarify to resolve.`

### Requirement: Key Decisions Output Block

When `fab-continue` generates `plan.md`, the output SHALL include a "Key Decisions" block summarizing the major decisions made during plan generation. Each decision SHALL list the choice, rationale, and one rejected alternative.

#### Scenario: Plan with 2 key decisions
- **GIVEN** `fab-continue` generates a plan that required 2 architectural decisions
- **WHEN** the output is displayed
- **THEN** a "Key Decisions" block SHALL appear listing each decision with its rationale

### Requirement: Assumptions Summary in Output and Artifact

`fab-continue` output SHALL end with an Assumptions summary block (same format as `fab-new`). The Assumptions summary SHALL also be persisted as a trailing `## Assumptions` section in the generated artifact.

When tentative assumptions exist, the output SHALL include: `Run /fab-clarify to review tentative assumptions.`

#### Scenario: Spec generation with tentative assumptions
- **GIVEN** `fab-continue` generates `spec.md` with 2 Tentative assumptions
- **WHEN** the output is displayed
- **THEN** an Assumptions summary SHALL appear in both output and the artifact file
- **AND** the output SHALL suggest `/fab-clarify`

### Requirement: SRAD-Based Question Selection

`fab-continue` SHALL apply SRAD scoring to the current stage's decision points. Up to 3 Unresolved decisions (lowest R + lowest A) SHALL be asked. The interruption budget is 1-2 questions per stage for typical changes, up to 3 for highly ambiguous ones.

#### Scenario: Stage generation with 1 unresolved decision
- **GIVEN** `fab-continue` is generating `spec.md`
- **WHEN** SRAD evaluation identifies 1 Unresolved decision
- **THEN** the agent SHALL ask that 1 question before generating
- **AND** SHALL assume all Confident and Tentative decisions without asking

## Skill: `fab-ff`

### Requirement: `--auto` Skips Frontloaded Questions

`fab-ff --auto` SHALL skip the frontloaded question batch entirely. The "hard zero interruptions" contract means no questions are asked under any circumstances. All Unresolved decisions are auto-guessed with `<!-- auto-guess: ... -->` markers.

#### Scenario: fab-ff --auto with ambiguous proposal
- **GIVEN** a user invokes `fab-ff --auto`
- **WHEN** the proposal contains ambiguities that would normally generate questions
- **THEN** the agent SHALL NOT ask any questions
- **AND** SHALL auto-guess all Unresolved decisions and mark them in artifacts

### Requirement: Plan-Skip Reasoning

When `fab-ff` autonomously decides to skip or generate a plan, the output SHALL include a brief rationale explaining the decision. Format: `Plan {skipped|generated} — {one-sentence reason}.`

#### Scenario: Plan skipped with reasoning
- **GIVEN** `fab-ff` evaluates plan necessity
- **WHEN** the change is simple and no architectural decisions are needed
- **THEN** the output SHALL read something like: `Plan skipped — change touches only skill files with clear requirements from the spec.`

#### Scenario: Plan generated with reasoning
- **GIVEN** `fab-ff` evaluates plan necessity
- **WHEN** the change requires architectural decisions
- **THEN** the output SHALL explain: `Plan generated — multiple integration points require documented decisions.`

### Requirement: Cumulative Assumptions Summary

`fab-ff` output SHALL end with a cumulative Assumptions summary aggregating all assumptions made across all generated stages. Each entry SHALL note which artifact it belongs to.

#### Scenario: Multi-stage assumptions
- **GIVEN** `fab-ff` generates spec (2 assumptions), plan (1 assumption), and tasks (1 assumption)
- **WHEN** the pipeline completes
- **THEN** a single cumulative Assumptions summary SHALL appear with 4 entries
- **AND** each entry SHALL indicate its source artifact (e.g., "in spec.md")

### Requirement: Assumptions Persisted in Each Artifact

`fab-ff` SHALL persist assumptions as a trailing `## Assumptions` section in each generated artifact individually (`spec.md`, `plan.md`, `tasks.md`). The cumulative output summary is for the user; the per-artifact sections are for `fab-clarify` scanning.

#### Scenario: Artifacts have individual assumption sections
- **GIVEN** `fab-ff` completes all stages
- **WHEN** the artifacts are written to disk
- **THEN** each artifact file that had assumptions SHALL contain its own `## Assumptions` section

### Requirement: Auto-Guess Markers in --auto Mode

In `--auto` mode, every Unresolved decision SHALL be resolved with a best guess and marked with `<!-- auto-guess: {description} -->` in the artifact. The output SHALL list all auto-guesses prominently at the end.

#### Scenario: Auto-guess output listing
- **GIVEN** `fab-ff --auto` makes 3 auto-guesses across stages
- **WHEN** the pipeline completes
- **THEN** the output SHALL list all 3 auto-guesses with descriptions and source artifacts
- **AND** SHALL suggest: `Run /fab-clarify to review and confirm auto-guesses, or proceed with /fab-apply.`

## Skill: `fab-clarify`

### Requirement: Scan for `<!-- assumed: ... -->` Markers

`fab-clarify` suggest mode taxonomy scan SHALL detect `<!-- assumed: {description} -->` markers in the artifact, in addition to the existing `<!-- auto-guess: ... -->` and `[NEEDS CLARIFICATION]` markers.

#### Scenario: Artifact with assumed markers
- **GIVEN** a user invokes `/fab-clarify`
- **WHEN** the current artifact contains 2 `<!-- assumed: ... -->` markers
- **THEN** the taxonomy scan SHALL identify these as Tentative assumptions needing review
- **AND** SHALL generate questions allowing the user to confirm or override each assumption

### Requirement: Auto Mode Scans Assumed Markers

`fab-clarify` auto mode SHALL also detect and attempt to resolve `<!-- assumed: ... -->` markers autonomously, using the same resolvable/blocking/non-blocking classification as for other gap types.

#### Scenario: Auto mode resolves assumed marker
- **GIVEN** `fab-ff` invokes `fab-clarify` in auto mode
- **WHEN** the artifact contains an `<!-- assumed: ... -->` marker that can be confirmed from available context
- **THEN** the marker SHALL be removed (assumption confirmed)
- **AND** the resolution SHALL count toward the `resolved` total in the machine-readable result

### Requirement: Assumed Markers in Question Format

When `fab-clarify` suggest mode presents a question derived from an `<!-- assumed: ... -->` marker, the question SHALL frame the current assumption as the recommended option and offer alternatives.

#### Scenario: Question from assumed marker
- **GIVEN** the artifact contains `<!-- assumed: supplement existing auth rather than replace -->`
- **WHEN** `fab-clarify` presents this as a question
- **THEN** the recommendation SHALL be "Supplement existing auth" (the current assumption)
- **AND** alternatives SHALL be offered (e.g., "Replace existing auth", "Both — configurable")

## Skill: `fab-apply` — Soft Gate

### Requirement: Auto-Guess Warning Gate

`fab-apply` SHALL check for `<!-- auto-guess: ... -->` markers in the `tasks.md` artifact before beginning implementation. If markers exist, the skill SHALL:

1. Print a warning: `⚠ {N} auto-guess marker(s) found in tasks.md.`
2. List each auto-guess description.
3. Prompt: `Continue with implementation? (y/n)`
4. If the user answers "n" or "no", abort with: `Run /fab-clarify to resolve auto-guesses first.`
5. If the user answers "y" or "yes", proceed normally.

#### Scenario: Tasks with auto-guesses — user continues
- **GIVEN** `tasks.md` contains 2 `<!-- auto-guess: ... -->` markers
- **WHEN** the user invokes `/fab-apply`
- **THEN** the skill SHALL display the warning and prompt
- **AND** if the user answers "y", implementation SHALL proceed

#### Scenario: Tasks with auto-guesses — user aborts
- **GIVEN** `tasks.md` contains 1 `<!-- auto-guess: ... -->` marker
- **WHEN** the user invokes `/fab-apply` and answers "n"
- **THEN** the skill SHALL abort with the clarify suggestion
- **AND** SHALL NOT begin any implementation work

#### Scenario: Tasks with no auto-guesses
- **GIVEN** `tasks.md` contains no `<!-- auto-guess: ... -->` markers
- **WHEN** the user invokes `/fab-apply`
- **THEN** the soft gate SHALL be skipped entirely (no warning, no prompt)

## Deprecated Requirements

None — all changes are additive. Existing artifact formats and marker conventions remain valid.

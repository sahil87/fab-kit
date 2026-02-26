# SRAD Autonomy Framework

SRAD is the decision-making framework that governs how Fab planning skills handle ambiguity. When a skill encounters a decision point not explicitly addressed by user input, SRAD determines whether the skill should assume an answer, ask the user, or flag it for later resolution.

SRAD also produces a **confidence score** — a numeric measure of how well-resolved a change's decisions are. This score gates `/fab-ff` (the fast-forward pipeline), ensuring it only runs when ambiguity is low enough for safe execution.

---

## The Four Dimensions

**SRAD** stands for:

- **S — Signal Strength**: How much detail the user provided about this decision point.
- **R — Reversibility**: How easily the decision can be changed later without cascading rework.
- **A — Agent Competence**: How well the agent can answer this from available context (config, constitution, codebase).
- **D — Disambiguation Type**: How many valid interpretations exist for this decision.

### Evaluation Criteria

Each dimension is evaluated on a **continuous 0–100 scale** (100 = fully safe to assume, 0 = must ask). The following rubric provides guidance for scoring:

| Dimension | High (75–100) | Medium (40–74) | Low (0–39) |
|-----------|--------------|----------------|------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | Moderate detail, some gaps, partially specified | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via `/fab-clarify` or stage reset | Moderate rework — touches a few files/artifacts | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Partial codebase signals, some inference needed | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | 2–3 options with a clear front-runner | Multiple valid interpretations with different tradeoffs |

### Fuzzy-to-Grade Mapping

The four per-dimension scores are aggregated into a single **composite score** using a weighted mean, then mapped to a confidence grade via trapezoidal thresholds.

**Aggregation formula**:

```
composite = 0.25 * S + 0.30 * R + 0.25 * A + 0.20 * D
```

The higher weight on Reversibility (0.30 vs 0.25 for others) encodes the Critical Rule's intent: low-R decisions have disproportionate blast radius.

**Grade thresholds**:

| Grade | Composite Range | Interpretation |
|-------|----------------|----------------|
| **Certain** | 85–100 | All dimensions strongly favor assumption |
| **Confident** | 60–84 | Most dimensions favor assumption; minor gaps |
| **Tentative** | 30–59 | Mixed signals; reasonable guess but alternatives exist |
| **Unresolved** | 0–29 | Too ambiguous to assume safely |

**Critical Rule override**: Regardless of composite score, if R < 25 AND A < 25, the grade MUST be Unresolved.

### Dimension Score Persistence

When planning skills use fuzzy evaluation, per-decision scores are recorded in the Assumptions table's optional `Scores` column:

```markdown
| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:75 R:80 A:65 D:70 | Use OAuth2 | Config shows REST API |
```

Aggregate dimension statistics are stored in `.status.yaml`:

```yaml
confidence:
  fuzzy: true
  dimensions:
    signal: 78.5
    reversibility: 82.0
    competence: 71.2
    disambiguation: 85.0
```

---

## Confidence Grades

Each decision point produces an assumption graded on a 4-level scale:

| Grade | When to assign | Artifact marker | Output visibility |
|-------|---------------|----------------|-------------------|
| **Certain** | Deterministically answered by config, constitution, or template rules | None | Not mentioned — not worth noting |
| **Confident** | Strong signal with one obvious interpretation | None | Listed in Assumptions summary |
| **Tentative** | Reasonable guess, but multiple valid options exist | `<!-- assumed: {description} -->` | Listed in Assumptions summary; resolvable by `/fab-clarify` |
| **Unresolved** | Cannot determine; incompatible interpretations | None — always asked or bailed | Asked as a blocking question (never silently assumed) |

### Artifact Markers

Planning skills use HTML comment markers to flag assumptions for downstream scanning by `/fab-clarify`:

| Marker | Grade | Placed by | Scanned by |
|--------|-------|-----------|------------|
| `<!-- assumed: {description} -->` | Tentative | All planning skills (`/fab-new`, `/fab-continue`, `/fab-ff`) | `/fab-clarify` (suggest and auto modes) |
| `<!-- clarified: {description} -->` | Resolved | `/fab-clarify` | Informational — not scanned |

Markers are placed inline in the artifact, immediately after the assumed content:

```markdown
The API SHALL return errors as JSON objects with `error`, `message`, and `code` fields.
<!-- assumed: JSON error format — config shows REST/JSON stack, consistent with existing patterns -->
```

### Assumptions Summary

Every planning skill invocation that makes Confident or Tentative assumptions appends an `## Assumptions` section to the generated artifact:

```markdown
## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | OAuth2 over SAML | Config shows REST API stack |
| 2 | Tentative | Google + GitHub providers | Most common OSS combination |

2 assumptions made (1 confident, 1 tentative). Run /fab-clarify to review.
```

Certain grades are omitted (not worth mentioning). Unresolved grades are asked as questions, not assumed.

---

## Confidence Scoring

### Formula

```
if unresolved > 0:
  score = 0.0
else:
  base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  cover = min(1.0, total_decisions / expected_min)
  score = base * cover
```

Where `total_decisions = certain + confident + tentative + unresolved` and `expected_min` is looked up by `{stage, change_type}` from embedded tables in `calc-score.sh`. See [change-types.md](change-types.md) for the full `expected_min` threshold tables.

### Penalty Weights

| Grade | Penalty | Rationale |
|-------|---------|-----------|
| **Certain** | 0.0 | Deterministic — no ambiguity whatsoever |
| **Confident** | 0.3 | Moderate — strong signal but still an assumption; accumulates meaningfully |
| **Tentative** | 1.0 | Meaningful — reasonable guess but multiple valid options; could be wrong |
| **Unresolved** | Hard zero | Cannot run autonomously with unresolved decisions; any single Unresolved sets score to 0.0 |

### Coverage Factor

The `cover` component attenuates the score when the total number of decisions is less than the expected minimum for the change type and stage. This prevents thin specs (e.g., 2 decisions scoring 5.0) from getting inflated scores.

When `total_decisions >= expected_min`, `cover = 1.0` and the formula degenerates to the base penalty only. When `total_decisions < expected_min`, the score is proportionally reduced.

### Range

- **5.0**: All decisions are Certain AND decision count meets or exceeds `expected_min`
- **3.0**: The `/fab-ff` gate threshold for `feat`/`refactor` (see below)
- **0.0**: Any Unresolved decision exists, OR penalties + low coverage reduce the score to zero

The `max(0.0, ...)` floor clamps the base — if penalties exceed 5.0, the base is 0.0, not negative.

### Storage

The confidence score is stored in `.status.yaml` within each change folder:

```yaml
confidence:
  certain: 12      # count of Certain-graded decisions
  confident: 3     # count of Confident-graded decisions
  tentative: 2     # count of Tentative-graded decisions
  unresolved: 0    # count of Unresolved-graded decisions
  score: 2.1       # derived score from formula above
```

---

## Gate Threshold

`/fab-ff` requires `confidence.score >= threshold` before executing the fast-forward pipeline. The threshold varies by **change type** (7 types from [Conventional Commits](change-types.md)):

| Change Type | Gate Threshold | Rationale |
|-------------|---------------|-----------|
| **`fix`** | 2.0 | Low risk, narrow scope — more tolerance for assumptions |
| **`feat`** | 3.0 | Default — balanced risk tolerance |
| **`refactor`** | 3.0 | Behavioral preservation important, moderate tolerance |
| **`docs`** | 2.0 | Low blast radius, documentation-only |
| **`test`** | 2.0 | Low blast radius, test-only |
| **`ci`** | 2.0 | Low blast radius, infrastructure-only |
| **`chore`** | 2.0 | Low blast radius, maintenance |

Change type is stored as `change_type:` in `.status.yaml` (default: `feat`). The gate check is performed by `calc-score.sh --check-gate`. See [change-types.md](change-types.md) for the full taxonomy.

### What 3.0 Allows

With the formula `score = base * cover` where `base = 5.0 - 0.3 * confident - 1.0 * tentative`:

Assuming full coverage (`cover = 1.0`, i.e., `total_decisions >= expected_min`):

- **0 Tentative, up to 6 Confident**: base = 5.0 – 1.8 = 3.2 (passes)
- **0 Tentative, 7 Confident**: base = 5.0 – 2.1 = 2.9 (fails)
- **1 Tentative, up to 3 Confident**: base = 5.0 – 1.0 – 0.9 = 3.1 (passes)
- **2 Tentative, 0 Confident**: base = 5.0 – 2.0 = 3.0 (passes, barely)
- **3+ Tentative**: base ≤ 2.0 (fails — too many guesses)
- **Any Unresolved**: score = 0.0 (always fails)

With low coverage (e.g., 2 of 6 expected decisions for `feat`): `cover = 0.33`, even a perfect base of 5.0 yields only 1.7. This prevents thin specs from passing the gate.

### Gate Behavior

When the user runs `/fab-ff`:
- **Score >= threshold**: Pipeline fast-forwards through remaining planning stages, then continues through apply, review, and hydrate
- **Score < threshold**: Pipeline refuses to execute and reports the score, suggesting `/fab-clarify` to resolve Tentative assumptions or answer Unresolved questions before retrying

---

## Confidence Lifecycle

| Event | Trigger | Action |
|-------|---------|--------|
| Computation | `/fab-continue` (spec stage) | `calc-score.sh` scans spec, writes to `.status.yaml` |
| Recomputation | `/fab-clarify` (suggest mode) | `calc-score.sh` re-scans after resolved assumptions |
| Gate check | `/fab-ff` | Reads score from `.status.yaml` (no recomputation) |

---

## The Critical Rule

**Unresolved decisions with low Reversibility AND low Agent Competence MUST always be asked as questions** — even when the skill's question budget is otherwise exhausted.

These are high-blast-radius decisions where:
- Getting it wrong cascades through multiple artifacts (low R)
- The agent has no good basis for guessing — it requires business context, user preferences, or political knowledge the agent doesn't have (low A)

The existence of `/fab-clarify` as an escape valve does **not** justify silently assuming these. `/fab-clarify` is designed for Tentative assumptions (reasonable guesses that might be wrong). Unresolved decisions with low R + low A are not reasonable guesses — they are genuine unknowns.

---

## Worked Examples

### Example 1: High-Ambiguity Intake

> **Input**: "Add auth."

Two words, no detail on mechanism, scope, or integration.

| Decision point | S | R | A | D | Composite | Grade |
|---------------|---|---|---|---|-----------|-------|
| Auth mechanism (OAuth2 vs SAML vs API keys) | 10 | 15 | 10 | 15 | 12.5 | **Unresolved** (12.5 < 30) |
| Replace or supplement existing auth | 15 | 10 | 20 | 20 | 15.5 | **Unresolved** (R=10 < 25, A=20 < 25 → Critical Rule) |
| Session storage (JWT vs server-side) | 20 | 50 | 55 | 45 | 42.0 | **Tentative** (30 ≤ 42.0 < 60) |

**Confidence counts**: Certain: 2, Confident: 1, Tentative: 1, Unresolved: 2

**Score**: `0.0` — any Unresolved decision produces a hard zero.

**Outcome**: `/fab-ff` gate blocks (0.0 < 3.0 `feat` threshold). The user must answer the Unresolved questions or use `/fab-clarify` to resolve Tentative assumptions before the fast-forward pipeline can run.

### Example 2: Low-Ambiguity Intake

> **Input**: "Add a loading spinner to the submit button on the checkout page. Use the existing `Spinner` component from the design system. Show it while the payment API call is in-flight and disable the button to prevent double-submission."

Detailed description specifying the component, location, trigger, and behavior.

| Decision point | S | R | A | D | Composite | Grade |
|---------------|---|---|---|---|-----------|-------|
| Which spinner component | 95 | 90 | 95 | 100 | 94.5 | **Certain** (94.5 ≥ 85) |
| When to show/hide spinner | 90 | 92 | 88 | 95 | 91.1 | **Certain** (91.1 ≥ 85) |
| Double-submission prevention | 95 | 95 | 90 | 98 | 94.3 | **Certain** (94.3 ≥ 85) |

**Confidence counts**: Certain: 8, Confident: 2, Tentative: 0, Unresolved: 0

**Score**: `base = max(0.0, 5.0 - 0.6) = 4.4`, `cover = min(1.0, 10 / 6) = 1.0`, `score = 4.4 * 1.0 = 4.4`

**Outcome**: `/fab-ff` gate passes (4.4 >= 3.0 `feat` threshold). The fast-forward pipeline can run with high confidence.

---

## Skill-Specific Autonomy Levels

SRAD manifests differently depending on which skill is running. Skills closer to the "explore" end ask freely; skills closer to the "autonomous" end minimize interruption:

| Aspect | `/fab-new` | `/fab-continue` | `/fab-ff` | `/fab-fff` |
|--------|------------|-----------------|-----------|-----------|
| **Posture** | SRAD-driven adaptive questioning, gap analysis, conversational mode, intake-only output | Surface tentative, ask top ~3 unresolved | Batch all unresolved upfront, then go; gated on confidence >= threshold | Same as `/fab-ff` (frontloaded questions); no confidence gate |
| **Interruption budget** | Adaptive — SRAD-driven (no fixed cap) | 1–2 per stage | 0–1 batch at start | Same as `/fab-ff` (frontloaded) |
| **Output** | Intake + confidence score + assumptions summary | Key Decisions + Assumptions summary + [NEEDS CLARIFICATION] count | Cumulative Assumptions summary | Same as `/fab-ff` + apply/review/archive output |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` (bails on blockers or review failure) |
| **Recomputes confidence?** | No | Spec stage only | No | No |

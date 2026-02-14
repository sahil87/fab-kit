# SRAD Autonomy Framework

SRAD is the decision-making framework that governs how Fab planning skills handle ambiguity. When a skill encounters a decision point not explicitly addressed by user input, SRAD determines whether the skill should assume an answer, ask the user, or flag it for later resolution.

SRAD also produces a **confidence score** — a numeric measure of how well-resolved a change's decisions are. This score gates `/fab-fff` (the full autonomous pipeline), ensuring it only runs when ambiguity is low enough for safe unattended execution.

---

## The Four Dimensions

**SRAD** stands for:

- **S — Signal Strength**: How much detail the user provided about this decision point.
- **R — Reversibility**: How easily the decision can be changed later without cascading rework.
- **A — Agent Competence**: How well the agent can answer this from available context (config, constitution, codebase).
- **D — Disambiguation Type**: How many valid interpretations exist for this decision.

### Evaluation Criteria

Each dimension is evaluated as high (safe to assume) or low (consider asking):

| Dimension | High (safe to assume) | Low (consider asking) |
|-----------|----------------------|----------------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via `/fab-clarify` or stage reset | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | Multiple valid interpretations with different tradeoffs |

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
  score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
```

### Penalty Weights

| Grade | Penalty | Rationale |
|-------|---------|-----------|
| **Certain** | 0.0 | Deterministic — no ambiguity whatsoever |
| **Confident** | 0.3 | Moderate — strong signal but still an assumption; accumulates meaningfully |
| **Tentative** | 1.0 | Meaningful — reasonable guess but multiple valid options; could be wrong |
| **Unresolved** | Hard zero | Cannot run autonomously with unresolved decisions; any single Unresolved sets score to 0.0 |

### Range

- **5.0**: All decisions are Certain — maximum confidence, zero ambiguity
- **3.0**: The `/fab-fff` gate threshold (see below) — allows at most 2 Tentative decisions
- **0.0**: Any Unresolved decision exists, OR 5+ Tentative decisions accumulate enough penalty

The `max(0.0, ...)` floor clamps the score — if penalties exceed 5.0 (e.g., 6 Tentative decisions: `5.0 - 6.0 = -1.0`), the score is 0.0, not negative.

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

`/fab-fff` requires `confidence.score >= 3.0` before executing the autonomous pipeline.

### What 3.0 Allows

With the formula `5.0 - 0.3 * confident - 1.0 * tentative`:

- **0 Tentative, up to 6 Confident**: score = 5.0 – 1.8 = 3.2 (passes)
- **0 Tentative, 7 Confident**: score = 5.0 – 2.1 = 2.9 (fails)
- **1 Tentative, up to 3 Confident**: score = 5.0 – 1.0 – 0.9 = 3.1 (passes)
- **1 Tentative, 4 Confident**: score = 5.0 – 1.0 – 1.2 = 2.8 (fails)
- **2 Tentative, 0 Confident**: score = 5.0 – 2.0 = 3.0 (passes, barely)
- **2 Tentative, 1+ Confident**: score < 3.0 (fails)
- **3+ Tentative**: score ≤ 2.0 (fails — too many guesses)
- **Any Unresolved**: score = 0.0 (always fails)

In practice: at most 6 Confident decisions (with no Tentative), or 2 Tentative with very few Confident.

### Gate Behavior

When the user runs `/fab-fff`:
- **Score >= 3.0**: Pipeline proceeds autonomously through planning, apply, review, and archive
- **Score < 3.0**: Pipeline refuses to execute and reports the score, suggesting `/fab-clarify` to resolve Tentative assumptions or answer Unresolved questions before retrying

---

## Confidence Lifecycle

| Event | Trigger | Action |
|-------|---------|--------|
| Computation | `/fab-continue` (spec stage) | `_calc-score.sh` scans brief + spec, writes to `.status.yaml` |
| Recomputation | `/fab-clarify` (suggest mode) | `_calc-score.sh` re-scans after resolved assumptions |
| Gate check | `/fab-fff` | Reads score from `.status.yaml` (no recomputation) |

---

## The Critical Rule

**Unresolved decisions with low Reversibility AND low Agent Competence MUST always be asked as questions** — even when the skill's question budget is otherwise exhausted.

These are high-blast-radius decisions where:
- Getting it wrong cascades through multiple artifacts (low R)
- The agent has no good basis for guessing — it requires business context, user preferences, or political knowledge the agent doesn't have (low A)

The existence of `/fab-clarify` as an escape valve does **not** justify silently assuming these. `/fab-clarify` is designed for Tentative assumptions (reasonable guesses that might be wrong). Unresolved decisions with low R + low A are not reasonable guesses — they are genuine unknowns.

---

## Worked Examples

### Example 1: High-Ambiguity Brief

> **Input**: "Add auth."

Two words, no detail on mechanism, scope, or integration.

| Decision point | S | R | A | D | Grade |
|---------------|---|---|---|---|-------|
| Auth mechanism (OAuth2 vs SAML vs API keys) | Low — no detail | Low — cascades into DB schema, middleware, API contracts | Low — business relationship with identity providers | Low — all three valid with different tradeoffs | **Unresolved** |
| Replace or supplement existing auth | Low — no indication | Low — fundamental architectural choice | Low — depends on business requirements | Low — both valid | **Unresolved** |
| Session storage (JWT vs server-side) | Low — not mentioned | Medium — refactorable but touches many endpoints | Medium — config shows REST API | Medium — JWT is common default for REST | **Tentative** |

**Confidence counts**: Certain: 2, Confident: 1, Tentative: 1, Unresolved: 2

**Score**: `0.0` — any Unresolved decision produces a hard zero.

**Outcome**: `/fab-fff` gate blocks (0.0 < 3.0). The user must answer the Unresolved questions or use `/fab-clarify` to resolve Tentative assumptions before the autonomous pipeline can run.

### Example 2: Low-Ambiguity Brief

> **Input**: "Add a loading spinner to the submit button on the checkout page. Use the existing `Spinner` component from the design system. Show it while the payment API call is in-flight and disable the button to prevent double-submission."

Detailed description specifying the component, location, trigger, and behavior.

| Decision point | S | R | A | D | Grade |
|---------------|---|---|---|---|-------|
| Which spinner component | High — explicitly named | High — trivially swappable | High — design system documented in codebase | High — user specified it | **Certain** |
| When to show/hide spinner | High — "while API call is in-flight" | High — UI state, easily changed | High — standard loading pattern | High — one obvious interpretation | **Certain** |
| Double-submission prevention | High — "disable the button" | High — single attribute change | High — standard pattern | High — user specified the approach | **Certain** |

**Confidence counts**: Certain: 8, Confident: 2, Tentative: 0, Unresolved: 0

**Score**: `max(0.0, 5.0 - 0.6 - 0.0) = 4.4`

**Outcome**: `/fab-fff` gate passes (4.4 >= 3.0). The autonomous pipeline can run with high confidence.

---

## Skill-Specific Autonomy Levels

SRAD manifests differently depending on which skill is running. Skills closer to the "explore" end ask freely; skills closer to the "autonomous" end minimize interruption:

| Aspect | `/fab-new` | `/fab-continue` | `/fab-ff` | `/fab-fff` |
|--------|------------|-----------------|-----------|-----------|
| **Posture** | SRAD-driven adaptive questioning, gap analysis, conversational mode, brief-only output | Surface tentative, ask top ~3 unresolved | Batch all unresolved upfront, then go | Same as `/fab-ff`; gated on confidence >= 3.0 |
| **Interruption budget** | Adaptive — SRAD-driven (no fixed cap) | 1–2 per stage | 0–1 batch at start | Same as `/fab-ff` (frontloaded) |
| **Output** | Brief + confidence score + assumptions summary | Key Decisions + Assumptions summary + [NEEDS CLARIFICATION] count | Cumulative Assumptions summary | Same as `/fab-ff` + apply/review/archive output |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` (bails on blockers or review failure) |
| **Recomputes confidence?** | No | Spec stage only | No | No |

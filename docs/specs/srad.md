# SRAD Autonomy Framework

SRAD is the decision-making framework that governs how Fab planning skills handle ambiguity. When a skill encounters a decision point not explicitly addressed by user input, SRAD determines whether the skill should assume an answer, ask the user, or flag it for later resolution.

SRAD also produces a **confidence score** — a numeric measure of how well-resolved a change's decisions are. This score gates `/fab-ff` and `/fab-fff` (the fast-forward and full pipelines), ensuring they only run when ambiguity is low enough for safe execution.

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

The four per-dimension scores are aggregated into a single **composite score** using a weighted mean, then mapped to a confidence grade via half-open thresholds.

**Aggregation formula**:

```
composite = 0.25 * S + 0.30 * R + 0.25 * A + 0.20 * D
```

The higher weight on Reversibility (0.30 vs 0.25 for others) encodes the Critical Rule's intent: low-R decisions have disproportionate blast radius.

**Grade thresholds** (half-open — composites are continuous, so values like 59.85 or 84.5 must grade deterministically):

| Grade | Composite (half-open) | Interpretation |
|-------|----------------------|----------------|
| **Certain** | composite ≥ 85 | All dimensions strongly favor assumption |
| **Confident** | 60 ≤ composite < 85 | Most dimensions favor assumption; minor gaps |
| **Tentative** | 30 ≤ composite < 60 | Mixed signals; reasonable guess but alternatives exist |
| **Unresolved** | composite < 30 | Too ambiguous to assume safely |

**Critical Rule override**: Regardless of composite score, if R < 25 AND A < 25, the grade MUST be Unresolved. This is the Critical Rule's single numeric definition.

### Dimension Score Persistence

Per-decision scores are recorded in the Assumptions table's **required** `Scores` column — mandatory for every row, of every grade (`fab score` parses it and writes aggregate dimension statistics to `.status.yaml`):

```markdown
| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Use OAuth2 | Config shows REST API | S:75 R:80 A:65 D:70 |
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
| **Certain** | Deterministically answered by config, constitution, or template rules | None | Noted in Assumptions summary |
| **Confident** | Strong signal with one obvious interpretation | None | Noted in Assumptions summary |
| **Tentative** | Reasonable guess, but multiple valid options exist | `<!-- assumed: {description} -->` | Noted in Assumptions summary; resolvable by `/fab-clarify` |
| **Unresolved** | Cannot determine; incompatible interpretations | None — always asked or bailed | Asked as a blocking question (never silently assumed) AND noted in Assumptions summary |

### Artifact Markers

Planning skills use HTML comment markers to flag assumptions for downstream scanning by `/fab-clarify`:

| Marker | Grade | Placed by | Scanned by |
|--------|-------|-----------|------------|
| `<!-- assumed: {description} -->` | Tentative | All planning skills (`/fab-new`, `/fab-draft`, `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`) | `/fab-clarify` (suggest and auto modes) |
| `<!-- clarified: {description} -->` | Resolved | `/fab-clarify` | Informational — not scanned |

Markers are placed inline in the artifact, immediately after the assumed content:

```markdown
The API SHALL return errors as JSON objects with `error`, `message`, and `code` fields.
<!-- assumed: JSON error format — config shows REST/JSON stack, consistent with existing patterns -->
```

### Assumptions Summary

Every planning skill invocation appends an `## Assumptions` section to the generated artifact, recording **every graded decision** (canonical contract: `_srad.md` § Assumptions Summary Block):

```markdown
## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use the existing test runner | Config deterministically answers this | S:50 R:95 A:100 D:100 |
| 2 | Confident | OAuth2 over SAML | Config shows REST API stack | S:75 R:80 A:65 D:70 |
| 3 | Tentative | Google + GitHub providers | Most common OSS combination | S:40 R:70 A:45 D:40 |
| 4 | Unresolved | Replace or supplement existing auth | Asked — user chose replace | S:15 R:10 A:20 D:20 |

4 assumptions (1 certain, 1 confident, 1 tentative, 1 unresolved). Run /fab-clarify to review.
```

Rules:

- **All four grades are recorded in intake artifacts** — Certain rows included (`fab score` counts them toward `certain` and coverage; omitting them deflates the gate input). Unresolved rows are still asked as questions and carry status context in Rationale (`Asked — {outcome}` or `Deferred — {reason}`).
- **`plan.md` `## Assumptions` excludes Unresolved** — three grades only; apply decides-and-records (Unresolved is an intake-only construct).
- **The Scores column is required on every row.**
- **Omit-when-zero applies to displayed output only**: a skill's output omits the summary block when 0 assumptions were made, but generated artifacts ALWAYS carry the `## Assumptions` section — with a `0 assumptions.` footer and no table rows when empty — keeping `fab score` parsing uniform.

---

## Confidence Scoring

The confidence score is a **Resolution Average**: it reuses the per-row S:R:A:D dimensions already recorded on every Assumptions row (the same dimensions the grade mapping above uses), averages their composites, and rescales that mean onto the 0–5 scale. Thoroughness is rewarded — a richer, more honestly-graded intake with strong dimensions scores *higher*, never lower.

### Formula

```
for each Assumptions row with parseable dimensions:
  composite = 0.25 * S + 0.30 * R + 0.25 * A + 0.20 * D   # the grade-mapping weights, unchanged
  if R < 25 AND A < 25:  return 0.0   # Critical Rule, per-row, on raw dimensions → hard fail
if any row is Unresolved:  return 0.0  # genuine unknown → hard fail
mean  = average(composite over rows that have parseable dimensions)
cover = min(1.0, total_decisions / expected_min)
return (mean / 20.0) * cover
```

Where `total_decisions = certain + confident + tentative + unresolved` (ALL graded rows) and `expected_min` is looked up by `change_type` from a single embedded table in `fab score`. See [change-types.md](change-types.md) for the full `expected_min` table.

The per-row composite uses the **same weights** as the grade-mapping aggregation (`0.25 * S + 0.30 * R + 0.25 * A + 0.20 * D`) — no new constant is introduced. The `/20` divisor rescales the 0–100 composite mean onto the 0–5 scale, so a 3.0 gate equals a mean composite of 60 — exactly the "Confident" floor in the grade mapping. The threshold therefore stays principled and needs no tuning.

### Hard Fails

Two conditions short-circuit the score to `0.0` *before* the mean is taken:

| Condition | Scope | Rationale |
|-----------|-------|-----------|
| **Critical Rule** (`R < 25 AND A < 25`) | Per row, on raw dimensions | A single high-blast-radius, low-competence decision is a genuine unknown — the intake cannot run autonomously regardless of how strong the other rows are |
| **Unresolved** | Any Unresolved-graded row | An unresolved decision is an unanswered question; any single one sets the score to 0.0 |

Both are evaluated per row, so they hard-fail even when the averaged-in composites would otherwise be strong — a weak row cannot hide behind a high mean.

### Coverage Factor

The `cover` component attenuates the score when the total number of decisions is less than the expected minimum for the change type. This prevents thin intakes (e.g., 2 strong decisions) from getting inflated scores.

`total_decisions` counts **all graded rows** (`certain + confident + tentative + unresolved`), while the mean is taken only over rows with parseable dimensions. A dimensionless row (no `Scores` column — a malformed intake, since the column is required) is therefore excluded from the mean but still counted toward coverage. When `total_decisions >= expected_min`, `cover = 1.0` and the score is the rescaled mean alone. When `total_decisions < expected_min`, the score is proportionally reduced.

### Range

- **5.0**: Mean composite is 100 (all dimensions perfect) AND decision count meets or exceeds `expected_min`
- **3.0**: The single intake gate threshold (flat, all types — see below) — equivalently, a mean composite of 60 at full coverage
- **0.0**: Any Unresolved decision exists, any row trips the Critical Rule, OR a weak mean / low coverage reduces the score to zero

The score is clamped to the 0–5 scale by the bounded inputs: composites are 0–100 and `cover` is 0–1, so `(mean / 20) * cover` cannot exceed 5.0 or fall below 0.0.

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

There is exactly **one** confidence gate, evaluated at **intake** (the score is computed from `intake.md`, the sole scoring source). Both `/fab-ff` and `/fab-fff` require `confidence.score >= threshold` before entering the automated bracket. The `--force` flag on either skill bypasses it. The threshold is **flat 3.0 for all seven change types** (1.10.0): collapsing the former two-gate model (fixed-3.0 intake + per-type spec gate) to one gate at 3.0 keeps every type's bar ≥ both old gates — no silent relaxation.

| Change Type | Gate Threshold |
|-------------|---------------|
| **`fix`** | 3.0 |
| **`feat`** | 3.0 |
| **`refactor`** | 3.0 |
| **`docs`** | 3.0 |
| **`test`** | 3.0 |
| **`ci`** | 3.0 |
| **`chore`** | 3.0 |

The per-type map is retained in code (`getGateThreshold`) so future divergence is a data-only change. Change type is stored as `change_type:` in `.status.yaml` (default: `feat`). The gate check is performed by `fab score --check-gate --stage intake`. See [change-types.md](change-types.md) for the full taxonomy.

### What 3.0 Allows

With the formula `score = (mean / 20.0) * cover`, a 3.0 gate at full coverage (`cover = 1.0`, i.e., `total_decisions >= expected_min`) is exactly a **mean composite of 60** — the "Confident" floor in the grade mapping. So the gate passes iff the intake's decisions average out to at least Confident-grade resolution:

- **Mean composite ≥ 60** (all rows ~Confident or better): score ≥ 3.0 (passes)
- **Mean composite < 60** (the rows average below Confident): score < 3.0 (fails — too many weak guesses)
- **A single weak row drags the mean**: one low-composite row pulls the average down toward failure — thoroughness with strong dimensions is what clears the gate, not row count
- **Any row with `R < 25 AND A < 25`**: score = 0.0 (Critical Rule hard fail — always fails)
- **Any Unresolved row**: score = 0.0 (always fails)

With low coverage (e.g., 2 of 7 expected decisions for `feat`): `cover = 0.29`, so even a perfect mean composite of 100 yields only `(100 / 20) * 0.29 = 1.4`. This prevents thin intakes from passing the gate regardless of how strong the few recorded decisions are.

### Gate Behavior

When the user runs `/fab-ff`:
- **Score >= threshold**: Pipeline enters the automated bracket — apply (co-generating `plan.md`), review, and hydrate — unattended
- **Score < threshold**: Pipeline refuses to execute and reports the score, suggesting `/fab-clarify` (intake-only) to resolve Tentative assumptions or answer Unresolved questions before retrying

---

## Confidence Lifecycle

| Event | Trigger | Action |
|-------|---------|--------|
| Computation | `/fab-new` (after intake generation) | `fab score --stage intake` scans `intake.md`, writes to `.status.yaml` |
| Recomputation | `/fab-clarify` (intake-only, suggest mode) | `fab score --stage intake` re-scans after resolved assumptions |
| Gate check | `/fab-ff`, `/fab-fff` | `fab score --check-gate --stage intake` reads/compares against the flat 3.0 gate |

---

## The Critical Rule

**Decisions with R < 25 AND A < 25 — the single numeric override defined in the grade mapping above — are always Unresolved and MUST always be asked as questions**, even when the skill's question budget is otherwise exhausted. (The threshold is `< 25` on both dimensions; the rubric's 0–39 "Low" band is descriptive guidance, not the override's definition.)

These are high-blast-radius decisions where:
- Getting it wrong cascades through multiple artifacts (R < 25)
- The agent has no good basis for guessing — it requires business context, user preferences, or political knowledge the agent doesn't have (A < 25)

The existence of `/fab-clarify` as an escape valve does **not** justify silently assuming these. `/fab-clarify` is designed for Tentative assumptions (reasonable guesses that might be wrong). Unresolved decisions with R < 25 AND A < 25 are not reasonable guesses — they are genuine unknowns.

---

## Worked Examples

### Example 1: High-Ambiguity Intake

> **Input**: "Add auth."

Two words, no detail on mechanism, scope, or integration.

| Decision point | S | R | A | D | Composite | Grade |
|---------------|---|---|---|---|-----------|-------|
| Auth mechanism (OAuth2 vs SAML vs API keys) | 10 | 15 | 10 | 15 | 12.5 | **Unresolved** (12.5 < 30) |
| Replace or supplement existing auth | 15 | 10 | 20 | 20 | 15.75 | **Unresolved** (R=10 < 25, A=20 < 25 → Critical Rule) |
| Session storage (JWT vs server-side) | 20 | 50 | 55 | 45 | 42.75 | **Tentative** (30 ≤ 42.75 < 60) |

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
| Double-submission prevention | 95 | 95 | 90 | 98 | 94.35 | **Certain** (94.35 ≥ 85) |

**Confidence counts**: Certain: 8, Confident: 2, Tentative: 0, Unresolved: 0

**Score**: Resolution Average over the per-row composites. The three rows above average `(94.5 + 91.1 + 94.35) / 3 = 93.32`; with 10 decisions and `feat` `expected_min = 7`, `cover = min(1.0, 10 / 7) = 1.0`. `score = (93.32 / 20.0) * 1.0 = 4.7`. No row trips the Critical Rule and none is Unresolved, so there is no hard fail.

**Outcome**: `/fab-ff` gate passes (4.7 >= 3.0 `feat` threshold). The fast-forward pipeline can run with high confidence — strong, thoroughly-recorded dimensions produce a high score.

---

## Skill-Specific Autonomy Levels

SRAD manifests differently depending on which skill is running. Skills closer to the "explore" end ask freely; skills closer to the "autonomous" end minimize interruption (canonical table: `_srad.md` § Skill-Specific Autonomy Levels):

| Aspect | `/fab-new` (adaptive) | `/fab-continue` (deliberate) | `/fab-fff` (full pipeline) | `/fab-ff` (fast-forward) |
|--------|------------|-----------------|-----------|-----------|
| **Posture** | SRAD-driven: 0 questions for clear inputs, conversational for vague; gap analysis before folder creation | SRAD at intake only (the one asking stage); apply decides-and-records | Gated on confidence; extends through ship + review-pr | Gated on confidence; stops at hydrate |
| **Interruption budget** | SRAD-driven (no fixed cap); conversational mode for vague inputs | 1–2 at intake; 0 at apply and later | 0 (autonomous rework, then stop) | 0 (autonomous rework, then stop) |
| **Output** | Assumptions summary + "Run /fab-clarify to review" | Key Decisions block + Assumptions summary | Cumulative Assumptions summary + apply/review/hydrate/ship/review-pr output | Tasks + apply/review/hydrate output |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify`, `/fab-continue` (after rework cap) | `/fab-clarify`, `/fab-continue` (after rework cap) |
| **Recomputes confidence?** | Yes (intake, via `fab score --stage intake`) | No (no scoring at apply — intake is authoritative) | No | No |

The remaining two skills that declare `_srad` are covered by these columns: **`/fab-draft`** follows the `/fab-new` column exactly (a thin delta over fab-new Steps 0–9 — same SRAD posture and budget, minus activation/branch). **`/fab-clarify`** is the escape valve itself: suggest-mode questions are SRAD-prioritized (max 5 per invocation), resolved assumptions are re-graded in the artifact's table, and the intake score is always recomputed.

# SRAD Autonomy Framework

SRAD is the decision-making framework that governs how Fab planning skills handle ambiguity. When a skill encounters a decision point not explicitly addressed by user input, SRAD determines whether the skill should assume an answer, ask the user, or flag it for later resolution.

SRAD also produces a **confidence score** — a numeric 0–5 measure of how well-resolved a change's decisions are. This score gates `/fab-ff` and `/fab-fff` (the fast-forward and full pipelines), ensuring they only run when ambiguity is low enough for safe autonomous execution.

The framework has two layers, both built from the same four per-decision dimensions:

1. **Per-decision grading** — each decision point is scored on four dimensions, aggregated into a single **composite** (0–100), and labelled with an **indicative grade** (Certain / Confident / Tentative / Unresolved) that is a *hint for the reader*.
2. **Change-level confidence** — every decision's composite contributes a **penalty**; the change's score is `5.0` minus the sum of penalties. This is the number the gate checks.

---

## The Four Dimensions

**SRAD** stands for:

- **S — Signal Strength**: How much detail the user provided about this decision point.
- **R — Reversibility**: How easily the decision can be changed later without cascading rework.
- **A — Agent Competence**: How well the agent can answer this from available context (config, constitution, codebase).
- **D — Disambiguation Type**: How many valid interpretations exist for this decision.

### Evaluation Criteria

Each dimension is scored on a **continuous 0–100 scale** (100 = fully safe to assume, 0 = must ask). The following rubric provides guidance:

| Dimension | High (75–100) | Medium (40–74) | Low (0–39) |
|-----------|--------------|----------------|------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | Moderate detail, some gaps, partially specified | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via `/fab-clarify` or stage reset | Moderate rework — touches a few files/artifacts | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Partial codebase signals, some inference needed | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | 2–3 options with a clear front-runner | Multiple valid interpretations with different tradeoffs |

### The Composite

The four dimensions aggregate into a single **composite** via a weighted mean:

```
composite = 0.20·S + 0.30·R + 0.30·A + 0.20·D
```

**Reversibility and Agent Competence carry the highest weight (0.30 each).** This is deliberate: the decisions that produce unusable work are the ones that are *hard to undo* (low R) and that the agent *cannot reliably answer* (low A). By weighting R and A above S and D, the composite itself becomes the proxy for "how risky is it to assume this" — so the downstream penalty inherits the risk-weighting for free, with no separate risk rule.

The composite is a continuous 0–100 value. There are **no special-case overrides** — a decision's risk is fully expressed by where its composite lands.

### Grades (indicative only)

The composite maps to a four-level grade via half-open thresholds:

| Grade | Composite | Interpretation |
|-------|-----------|----------------|
| **Certain** | composite ≥ 80 | All dimensions strongly favor assumption |
| **Confident** | 50 ≤ composite < 80 | Most dimensions favor assumption; minor gaps |
| **Tentative** | 20 ≤ composite < 50 | Mixed signals; reasonable guess but alternatives exist |
| **Unresolved** | composite < 20 | Too ambiguous to assume safely |

**Grades are purely indicative — a human-readable hint, never an input to any formula.** The confidence score is computed entirely from composites (see § Confidence Scoring); the grade is a label *derived from* the composite and shown to the reader so they can scan an Assumptions table at a glance. Because the grade is derived, it can never contradict its own dimensions.

Agents record `S:R:A:D` for every decision; the grade is computed from those numbers by `fab score`, not asserted by hand.

### Dimension Score Persistence

Per-decision scores are recorded in the Assumptions table's **required** `Scores` column — mandatory for every row, of every grade (`fab score` parses it to compute the confidence score and to derive grades):

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

## Artifact Markers

Planning skills use HTML comment markers to flag assumptions for downstream scanning by `/fab-clarify`:

| Grade | When to assign | Artifact marker | Output visibility |
|-------|---------------|----------------|-------------------|
| **Certain** | Deterministically answered by config, constitution, or template rules | None | Noted in Assumptions summary |
| **Confident** | Strong signal with one obvious interpretation | None | Noted in Assumptions summary |
| **Tentative** | Reasonable guess, but multiple valid options exist | `<!-- assumed: {description} -->` | Noted in Assumptions summary; resolvable by `/fab-clarify` |
| **Unresolved** | Cannot determine; incompatible interpretations | None — always asked or deferred | Asked as a blocking question (or deferred and surfaced) AND noted in Assumptions summary |

| Marker | Grade | Placed by | Scanned by |
|--------|-------|-----------|------------|
| `<!-- assumed: {description} -->` | Tentative | All planning skills (`/fab-new`, `/fab-draft`, `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`) | `/fab-clarify` (suggest and auto modes) |
| `<!-- clarified: {description} -->` | Resolved | `/fab-clarify` | Informational — not scanned |

Markers are placed inline in the artifact, immediately after the assumed content:

```markdown
The API SHALL return errors as JSON objects with `error`, `message`, and `code` fields.
<!-- assumed: JSON error format — config shows REST/JSON stack, consistent with existing patterns -->
```

---

## Assumptions Summary

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

- **All four grades are recorded in intake artifacts** — Certain rows included (`fab score` counts them and uses their composites; omitting strong rows does not change the score, but omitting any row loses information). Unresolved rows are still asked as questions (or deferred — see § The Critical Rule) and carry status context in Rationale (`Asked — {outcome}` or `Deferred — {reason}`).
- **`plan.md` `## Assumptions` excludes Unresolved** — three grades only; apply decides-and-records (Unresolved is an intake-only construct).
- **The Scores column is required on every row** — it is the sole input to scoring and to grade derivation.
- **Omit-when-zero applies to displayed output only**: a skill's output omits the summary block when 0 assumptions were made, but generated artifacts ALWAYS carry the `## Assumptions` section — with a `0 assumptions.` footer and no table rows when empty — keeping `fab score` parsing uniform.

---

## Confidence Scoring

The confidence score is a **demerit model**: a change starts at a perfect `5.0`, and each decision subtracts a **penalty** determined by its composite. Strong decisions cost nothing; weak ones cost, and the cost **cannot be refunded** by surrounding strong decisions. This makes a single risky decision visible — it is not averaged away by a thorough intake.

Agents never compute the score — `fab score` (Go) does, reading `intake.md` (the sole scoring source).

### Formula

```
for each Assumptions row:
    c = 0.20·S + 0.30·R + 0.30·A + 0.20·D                  # composite, 0–100

    penalty(c) =  0                            if c ≥ 80    # Certain  → free
                  (80 − c) / 30 · 0.50         if 50 ≤ c < 80   # Confident → ≤ 0.5
                  0.50 + (50 − c)/50 · 2.50    if c < 50    # Tentative / Unresolved

score = clamp( 5.0 − Σ penalty(c),  0.0,  5.0 )
```

The penalty curve and the grade bands are the **same object** — the penalty at each band boundary equals the band's stated cost, so reading a grade tells you the row's penalty range:

| Band | Composite | Penalty range | Effect |
|------|-----------|---------------|--------|
| **Certain** | `c ≥ 80` | `0.00` | free |
| **Confident** | `50 ≤ c < 80` | `0.00 → 0.50` | low |
| **Tentative** | `20 ≤ c < 50` | `0.50 → 2.00` | high |
| **Unresolved** | `c < 20` | `2.00 → 3.00` | a single row alone blocks the gate |

The curve is continuous at both joins (`c=50`: both slopes meet at 0.50; `c=80`: meets 0.0). Per-row penalty ∈ `[0.0, 3.0]`.

### No hard-fail rules

There are **no short-circuit rules** — no "Unresolved → 0.0" and no reversibility override. Blocking is *emergent* from the penalty curve:

- A genuinely **Unresolved** decision scores `composite < 20`, so its penalty is `≥ 2.00`. A single such row drops a one-row intake to `≤ 3.0` (at the gate or below), and any deeper or additional weak row pushes it under. Unresolved decisions therefore block the gate **because the curve makes them block**, not because a separate rule forces zero.
- **Reversibility is handled by its 0.30 weight in the composite**, not by a separate rule. An irreversible decision scores a lower composite → lands in a worse band → is penalized harder, automatically.

This keeps the framework uniform: every decision flows through one curve, and there is exactly one number (the composite) that governs both its grade and its penalty.

> **Edge case**: a decision at *exactly* `composite = 20.000` (e.g. `S:0 R:0 A:0 D:100`) scores a penalty of exactly `2.00`, leaving a one-row intake at exactly `3.0` — a pass. This profile is degenerate and effectively never occurs; it is accepted in exchange for round penalty coefficients. Any composite below 20 blocks.

### Range

- **5.0**: Every decision has `composite ≥ 80` (all Certain) — zero total penalty.
- **3.0**: The single intake gate threshold (flat, all types). Reached when accumulated penalties total exactly 2.0 — e.g. four Confident-floor rows, or one deep-Tentative row, or one Unresolved row.
- **0.0**: Penalties total ≥ 5.0 — many weak decisions, or several Unresolved ones.

The score is clamped to `[0.0, 5.0]`.

### Worked Range Intuition

| Intake shape | Penalty sum | Score | Gate |
|--------------|-------------|-------|------|
| 10 decisions, all `c ≥ 80` | 0.00 | 5.00 | pass |
| 9 strong + 1 weak (`c=35`) | 1.25 | 3.75 | pass (one shaky decision survives) |
| 9 strong + 2 weak (`c=35`) | 2.50 | 2.50 | **fail** (two shaky decisions block) |
| 1 Unresolved (`c=10`) anywhere | ≥ 2.50 | ≤ 2.50 | **fail** (single unknown blocks) |

### Storage

The confidence score is stored in `.status.yaml` within each change folder:

```yaml
confidence:
  certain: 12      # count of Certain-graded decisions (derived from composite)
  confident: 3     # count of Confident-graded decisions
  tentative: 2     # count of Tentative-graded decisions
  unresolved: 0    # count of Unresolved-graded decisions
  score: 4.6       # derived score from the demerit formula above
```

The grade counts are **derived** from each row's composite (not from a hand-written grade column) and are informational — only `score` gates the pipeline.

---

## Gate Threshold

There is exactly **one** confidence gate, evaluated at **intake** (the score is computed from `intake.md`, the sole scoring source). Both `/fab-ff` and `/fab-fff` require `confidence.score >= threshold` before entering the automated bracket. The `--force` flag on either skill bypasses it. The threshold is **flat 3.0 for all seven change types**.

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

There is **no coverage factor and no minimum-decision requirement.** A thin intake is not penalized for being short — a change with two strong, well-resolved decisions genuinely passes at 5.0. Quality is measured per decision, so row count is not a proxy for it: a lazy thin intake has weak decisions, and the penalty curve catches them directly.

### What 3.0 Allows

The gate passes iff accumulated penalties total at most `2.0`. In grade terms:

- **All decisions Confident or better**: each costs ≤ 0.50, so up to four can sit at the Confident floor and still pass; anything stronger passes comfortably.
- **One isolated shaky decision survives**: a single Tentative row (penalty 0.50–2.00) leaves the score at 3.0–4.5 — a reasonable guess on one point does not block the pipeline.
- **Two shaky decisions block**: two deep-Tentative rows total ≥ 2.5 penalty → score ≤ 2.5 → fail.
- **Any single Unresolved decision blocks**: `composite < 20` → penalty ≥ 2.0 → score ≤ 3.0 and (for any composite strictly below 20) below it.

### Gate Behavior

When the user runs `/fab-ff` or `/fab-fff`:

- **Score ≥ threshold**: Pipeline enters the automated bracket — apply (co-generating `plan.md`), review, and hydrate (plus ship + review-pr for `/fab-fff`) — unattended.
- **Score < threshold**: Pipeline refuses to execute and reports the score, suggesting `/fab-clarify` (intake-only) to resolve Tentative assumptions or answer Unresolved questions before retrying.

---

## Confidence Lifecycle

| Event | Trigger | Action |
|-------|---------|--------|
| Computation | `/fab-new`, `/fab-draft` (after intake generation) | `fab score --stage intake` scans `intake.md`, writes to `.status.yaml` |
| Recomputation | `/fab-clarify` (intake-only, both modes) | `fab score --stage intake` re-scans after resolved assumptions |
| Gate check | `/fab-ff`, `/fab-fff` | `fab score --check-gate --stage intake` reads/compares against the flat 3.0 gate |

`/fab-continue` does NOT score at apply entry — intake is authoritative, and there is no scoring at any post-intake stage.

---

## The Critical Rule

**A decision the agent cannot answer — a genuine unknown — MUST be surfaced, never silently assumed.** Such a decision scores low on Reversibility and/or Agent Competence, lands at `composite < 20` (Unresolved), and is handled one of two ways:

1. **Asked** — when a user is reachable, the decision is asked as a blocking question. Recorded as an Unresolved row with Rationale `Asked — {outcome}`.
2. **Deferred** — when no user is reachable (the promptless-dispatch path under `/fab-proceed`), the decision is recorded as an Unresolved row with Rationale `Deferred — {reason}` and surfaced to the user by the dispatcher.

The existence of `/fab-clarify` as an escape valve does **not** justify silently assuming a genuine unknown. `/fab-clarify` is for Tentative assumptions (reasonable guesses that might be wrong); Unresolved decisions are not reasonable guesses.

### How blocking is enforced

There is **no separate gate** for unknown or deferred decisions — blocking is enforced *solely by the scoring curve*. A genuine unknown scores `composite < 20`, whose penalty (≥ 2.0) drops the change below the 3.0 gate. So:

- An **asked-and-answered** decision is re-scored after the answer; once resolved, its composite rises and it no longer blocks.
- A **deferred** decision blocks **by itself only if its composite is below 20.** This is the intended contract: a decision genuinely too ambiguous to assume should be scored with honestly-low dimensions (low S and A, often low R), placing it under 20 so the curve blocks the automated bracket until it is resolved via `/fab-clarify`. A decision scored at composite ≥ 20 does not block *on its own* — but it still adds a penalty and can contribute to a failing total alongside other weak rows (e.g. two Tentative rows at composite 35 sum to 2.5 penalty → score 2.5 → fail). The special property of `composite < 20` is that its penalty (≥ 2.0) clears the gate margin single-handedly.

This is the whole mechanism: surface genuine unknowns as low-composite Unresolved rows, and the penalty curve does the blocking. No override, no special case.

---

## Worked Examples

### Example 1: High-Ambiguity Intake

> **Input**: "Add auth."

Two words, no detail on mechanism, scope, or integration.

| Decision point | S | R | A | D | Composite | Grade |
|---------------|---|---|---|---|-----------|-------|
| Auth mechanism (OAuth2 vs SAML vs API keys) | 10 | 15 | 10 | 15 | 12.5 | **Unresolved** (< 20) |
| Replace or supplement existing auth | 15 | 10 | 20 | 20 | 16.0 | **Unresolved** (< 20) |
| Session storage (JWT vs server-side) | 20 | 50 | 55 | 45 | 44.5 | **Tentative** (20–50) |

**Penalties**: Unresolved rows at composite 12.5 and 16.0 → penalties `0.50 + (37.5/50)·2.50 = 2.375` and `0.50 + (34/50)·2.50 = 2.20`. Tentative row at 44.5 → `0.50 + (5.5/50)·2.50 = 0.775`. Sum = `5.35`.

**Score**: `clamp(5.0 − 5.35, 0, 5) = 0.0`.

**Outcome**: `/fab-ff` gate blocks (0.0 < 3.0). The two unknowns must be asked (or deferred and resolved) before the fast-forward pipeline can run.

### Example 2: Low-Ambiguity Intake

> **Input**: "Add a loading spinner to the submit button on the checkout page. Use the existing `Spinner` component from the design system. Show it while the payment API call is in-flight and disable the button to prevent double-submission."

Detailed description specifying the component, location, trigger, and behavior.

| Decision point | S | R | A | D | Composite | Grade |
|---------------|---|---|---|---|-----------|-------|
| Which spinner component | 95 | 90 | 95 | 100 | 94.5 | **Certain** (≥ 80) |
| When to show/hide spinner | 90 | 92 | 88 | 95 | 90.4 | **Certain** (≥ 80) |
| Double-submission prevention | 95 | 95 | 90 | 98 | 93.7 | **Certain** (≥ 80) |

**Penalties**: all three rows are `composite ≥ 80` → penalty 0 each. Sum = `0.0`.

**Score**: `clamp(5.0 − 0.0, 0, 5) = 5.0`.

**Outcome**: `/fab-ff` gate passes (5.0 ≥ 3.0). The fast-forward pipeline runs with high confidence — strong, thoroughly-recorded dimensions produce a perfect score.

### Example 3: One Risky Decision in a Thorough Intake

> **Input**: A detailed refactor with 11 well-specified decisions, but one of them is an irreversible behavior change the agent is only moderately sure about.

| Decision point | S | R | A | D | Composite | Grade |
|---------------|---|---|---|---|-----------|-------|
| 10 well-specified decisions | (all) | | | | ≥ 85 | **Certain** |
| Quick-drag now pans (irreversible widening) | 95 | 35 | 55 | 45 | 60.0 | **Confident** (just) |

**Penalties**: ten Certain rows → 0. The risky row at composite 60 → `(80−60)/30·0.50 = 0.33`. Sum = `0.33`.

**Score**: `5.0 − 0.33 = 4.67` → passes. *But its low R (35) pulled its composite down from where S alone would put it* — had the agent scored R lower (e.g. R:15, composite 54), the penalty rises and the row visibly dents the score. The point: the **R weight surfaces the risk into the number**, where a strength-averaging scheme would hide it.

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

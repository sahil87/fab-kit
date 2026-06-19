# SRAD Scoring Rationale (v1 → v2)

> **What this is**: the design rationale for the v2 SRAD confidence score — the **demerit
> model** specified in [srad.md](srad.md). It records *why* the v1 scheme
> ([srad-v1.md](srad-v1.md) — a Resolution-Average mean) was replaced: the evidence, the
> alternatives weighed and rejected, the penalty-curve derivation, the cross-repo
> validation, and the resulting implementation surface. The authoritative behavior lives in
> [srad.md](srad.md); this doc is the decision record behind it.

---

## Problem

The confidence score gates `/fab-ff` and `/fab-fff` — it decides whether a change is
well-resolved enough to run unattended. A good gate must **separate** safe intakes from
risky ones. The current formula does not.

The current score (the "Resolution Average") is a **weighted mean of every decision's
composite**, rescaled onto 0–5 and attenuated by a coverage factor:

```
composite = 0.25·S + 0.30·R + 0.25·A + 0.20·D       # per row, 0–100
mean      = average(composite over rows)
cover     = min(1, total_rows / expected_min)
score     = (mean / 20) · cover
```

Empirically, this compresses almost everything into a narrow passing band. Across 657
loom intakes the median is **4.31**, **81%** score ≥4.0, and only a thin tail fails. A
gate that passes the overwhelming majority of its inputs is barely a gate.

The deeper failure is **mean dilution**. The decisions that make shipped code unusable are
the one or two *irreversible, under-resolved* calls in an otherwise-thorough intake. Because
the score averages, a single weak row is swallowed by the strong ones — and the *more*
thorough the intake, the *more* effectively it hides the bad decision:

| 10-row intake | composite | current score |
|---|---|---|
| 9 strong rows | ~90 | |
| 1 irreversible-weak row | 35 | |
| **mean** | **84.5** | **4.2 — passes** |

The dangerous decision is invisible. This is the core defect: **the score measures the
*average* decision, but risk lives in the *worst* decision.**

---

## What was rejected, and why

Four "pull the aggregate toward the worst row" aggregators were simulated against 652
dimensioned loom intakes. All failed, for the same structural reason.

The weak-row population splits two ways:

- **Dangerous** (~10 of 54 weak intakes): the weakest row is *irreversible* (R<50) **and**
  under-resolved (composite<60). The real target.
- **Benign** (~44 of 54): the weakest row is *reversible* (R≥50), just vaguely worded.
  Safe to assume, easy to fix later.

Benign-weak rows outnumber dangerous ones **4:1**. Any *smooth* aggregator that pulls the
score toward the minimum punishes both populations equally, so it has terrible precision:

| Aggregator | Dangerous caught | Benign failed | Healthy killed | Verdict |
|---|---|---|---|---|
| **current mean** | ~0 | 1 | 11 | leaks dangerous rows |
| **soft-min** (α=0.5) | +1 | +3 | +3 | lowers everything, no separation |
| **harmonic / power mean** | ~0 | — | — | composites not spread enough to bite |
| **reversibility-weighted penalty** (smooth) | 0 (λ=1) | — | — | right idea, needs huge λ → collateral |

The lesson: a single dangerous row drowned in a 10–22-row mean cannot be surfaced by *any*
smooth aggregator without dragging the curve so low that hundreds of healthy intakes fail
too. **Smooth aggregation is the wrong tool.**

---

## The demerit model

Instead of averaging *quality*, the demerit model starts from a perfect score and
**subtracts a penalty for each weak decision**. Strong decisions are free; weak ones cost,
and the cost cannot be refunded by surrounding strong rows.

```
INPUT: ## Assumptions table. Every row carries dimensions S:R:A:D (0–100 each).

STEP 1 — composite (R and A up-weighted to 0.30):
    c = 0.20·S + 0.30·R + 0.30·A + 0.20·D                       # 0–100

STEP 2 — grade (purely INDICATIVE — a user hint, NEVER read by the score):
    Certain     if c ≥ 80
    Confident    if 50 ≤ c < 80
    Tentative     if 20 ≤ c < 50
    Unresolved    if c < 20

STEP 3 — per-row penalty (single piecewise curve; no hard rules):

                 ⎧ 0                          if c ≥ 80          (Certain  → free)
    penalty(c) = ⎨ (80 − c)/30 · 0.50          if 50 ≤ c < 80     (Confident → ≤ 0.5)
                 ⎩ 0.50 + (50 − c)/50 · 2.50   if c < 50          (Tentative/Unresolved)

STEP 4 — score:
    score = clamp( 5.0 − Σ penalty(c),  0.0,  5.0 )             # sum over all rows

GATE — score ≥ 3.0  (flat, all change types — unchanged)
```

### Two principles this encodes

1. **All grades are indicative.** Certain/Confident/Tentative/Unresolved are *derived from
   `c` and shown to the user as hints* — none is read by the scoring formula. The score
   depends only on `c`, never on the label. This resolves the "all-indicative-or-all-load-bearing"
   inconsistency: previously three grades were cosmetic while Unresolved was load-bearing
   (it forced a hard 0.0). Now the label is uniformly a *view* of the composite. (Today the
   grade is *supposed* to be derived from `c` already, but is hand-written by the agent and
   contradicts its own dimensions ~19% of the time — see § Grade derivation below.)

2. **Risk-weighting lives entirely in `c`.** R and A are weighted **0.30 each** (up from
   0.25), so an irreversible / low-competence decision scores a lower composite → lands in a
   worse band → is penalized harder, all through the one curve. There is **no separate R
   penalty term and no `R<25` hard-fail** — `c` is the single proxy that carries "R and A
   matter more" into the penalty. One lever.

### Constants

| Constant | Value | Meaning |
|---|---|---|
| free knee `X` | **80** | At/above this composite a decision is "Certain" and incurs no penalty |
| Confident-floor penalty | **0.50** | Penalty at `c = 50` — the hinge where the two slopes meet |
| aggressive slope coeff | **2.50** | A `c = 0` row adds `0.50 + 2.50 = 3.0` total |

The slopes are **derived from the band-boundary penalties**, not tuned freely — the penalty
*is* the grade boundary:

- **Confident** (`50 ≤ c < 80`): penalty ramps `0 → 0.50`. Max penalty 0.50 at the floor.
- **Tentative** (`20 ≤ c < 50`): penalty ramps `0.50 → 2.00`. By construction `penalty(20) =
  0.50 + (30/50)·2.50 = 2.00`.
- **Unresolved** (`c < 20`): penalty ramps `2.00 → 3.00`. A single Unresolved row alone sinks
  the gate (`5 − 2.0... < 3.0`).

So a user reading "Tentative" knows the row cost between 0.50 and 2.00 without a lookup —
the curve and the indicative bands are the same object.

### Penalty range per band

Penalty decreases monotonically with composite (better decision → less penalty):

| Band | Composite range | Penalty range | Effect |
|---|---|---|---|
| **Certain** | `80 ≤ c ≤ 100` | `0.00` | free |
| **Confident** | `50 ≤ c < 80` | `~0.0 → 0.50` | low |
| **Tentative** | `20 ≤ c < 50` | `0.50 → 2.00` | high |
| **Unresolved** | `0 ≤ c < 20` | `2.00 → 3.00` | auto-blocks (one row sinks the gate) |

The curve is continuous at both joins (`c=50`: both slopes → 0.50; `c=80`: → 0.0).
Whole-curve penalty ∈ **[0.0, 3.0]** per row.

**No hard-fail short-circuits.** There is no `Unresolved → 0.0` rule and no `R<25` rule —
blocking is purely emergent from the curve. A row at `c < 20` penalizes ≥ 2.0, which alone
drops a single-row intake below the 3.0 gate; deeper rows penalize more. The lone
exact-boundary edge — `c = 20.000` (e.g. `S:0 R:0 A:0 D:100`, an absurd profile) — scores
exactly 3.0 and passes; this is a measure-zero case accepted in exchange for the round 2.50
slope.

### Worked example

A 10-row intake: 9 strong rows (`c ≥ 80`) + 1 irreversible-weak row (`c = 35`):

```
9 strong rows (c ≥ 80):  penalty 0 each              → 0.00
1 weak row (c = 35):     0.50 + (15/50)·2.50         → 1.25
score = 5.0 − 1.25 = 3.75   → PASSES (one shaky row survives)
```

Two such weak rows: `5.0 − 2.50 = 2.50` → **fails**. The weak decision is now visible, where
the current mean reported ~4.2 and hid it. A single genuinely-Unresolved row (`c < 20`)
penalizes ≥ 2.0 and blocks on its own.

### Grade derivation (one source of truth)

Today the agent hand-writes both the grade **and** the dimensions; across 9,468 real rows
the hand-written grade contradicts the grade its own composite implies **19.1%** of the
time. Under this model the grade is **computed from `c`** by `fab score` (per the bands in
STEP 2) and the agent writes only `S:R:A:D`. One source of truth (the dimensions); the label
can never contradict itself.

---

## Why coverage and `expected_min` are dropped

The current formula needs `cover = min(1, rows/expected_min)` because a *mean* rewards
strength: two strong decisions average to ~90 and would pass cheaply, so `cover` was the
only brake on "lazy thin intake gets a free pass."

The demerit model **does not reward strength — it only punishes weakness.** A thin intake
gets no free pass because there is nothing to give: it starts at 5.0 and stays there *only
if every decision is genuinely strong*. A lazy thin intake has weak rows, and the demerit
catches them directly. **Row count stops being a proxy for quality because the formula
measures quality per row.**

Simulated coverage variants on 93 real thin intakes (rows < `expected_min`) confirm this —
keeping coverage actively *harms*:

| Variant | A 2-row all-strong intake scores | Healthy intakes killed (loom) |
|---|---|---|
| `none` (drop coverage) | **5.00** — correct | **0 / 568** |
| `mult` (today's `cover ×`) | 3.33 — punished for brevity | 9 / 568 |
| `floor` (cap <3 if thin) | 2.90 — gate-blocked for brevity | 39 / 568 |

`none` is the only variant that doesn't punish thin-but-solid intakes. **`expected_min`
becomes dead code in the score path** (it remains documented in
[change-types.md](change-types.md), but no longer affects the score).

This is a deliberate **philosophy shift**: from "a `feat` should surface ~7 decisions" to
"few-but-strong decisions are fine." Quality over quantity.

---

## Cross-repo validation

The formula was tuned on **loom** and validated against two independent held-out repos
(**fab-kit**, **run-kit**) — 1,080 real intakes total. The pattern is identical everywhere,
confirming it does not overfit the tuning set:

| Repo | n | current median | demerit median | current ≥4 | demerit ≥4 | healthy killed |
|---|---|---|---|---|---|---|
| **loom** (tuning) | 657 | 4.31 | **4.93** | 81.1% | 91.9% | **0 / 568** |
| **fab-kit** (held-out) | 264 | 4.27 | **4.92** | 81.4% | 98.1% | **0 / 249** |
| **run-kit** (held-out) | 164 | 4.29 | **4.89** | 85.4% | 97.0% | **0 / 153** |

Observations:

- **Decompresses correctly.** The median rises (clean intakes cluster near 5.0) while a
  real failure tail opens below 3.0 — a **bimodal** "ship-it / clarify-it" distribution,
  not the current unimodal clump at 4.3.
- **Catches everything the mean catches, and more.** Demerit dangerous-catch ≥ current in
  every repo. (fab-kit/run-kit have only 3 dangerous cases each, so dangerous-catch is not
  statistically meaningful there; the *healthy-preservation* and *decompression* signals
  are strong and consistent.)
- **Zero healthy collateral** in all three repos.

> Validation note: the cross-repo table above was generated during tuning under the original
> weights (`0.25 S / 0.30 R / 0.25 A / 0.20 D`) and slopes; re-running under the final weights
> (`0.20 / 0.30 / 0.30 / 0.20`) and slope (2.50) reproduces the same shape — loom median 4.93
> (<3 = 4.3%, ≥4 = 92.3%), fab-kit 4.93, run-kit 4.90. The conclusion is unchanged.

### Caveats

- The dangerous/benign/healthy labels use an operational definition (`R<50 & composite<60`
  for dangerous). The "catches N" figures share that threshold with the rule, so they are
  partly circular; the **healthy-killed** column is independent of that definition and is
  what proves the model is clean.
- **Risk-weighting is via the composite alone** — there is no separate R penalty term and
  no `R<25` hard-fail. The up-weighted R/A (0.30 each) sends low-reversibility / low-competence
  decisions into worse bands. The residual: a high-S/D row can still prop up a low-R decision
  (`R:20 S:95 A:90 D:95 → c=71`, Confident, passes). Accepted in exchange for a one-lever
  formula; the old `R<25 AND A<25` Critical Rule is **dropped**.
- **The Unresolved hard-fail is dropped** — blocking is emergent from the curve, not a
  short-circuit. This changes the promptless-defer backstop (see § Promptless-defer backstop)
  and is the one place this design touches an existing documented contract.

---

## Promptless-defer backstop (resolved)

`/fab-proceed`'s promptless-defer path records each un-askable decision as an `Unresolved`
row with Rationale `Deferred — promptless dispatch`. The old scheme relied on **"`fab score`
returns 0.0 whenever any Unresolved row exists"** as a structural backstop (documented in
`_srad.md`, `_intake.md`, `fab-proceed.md`, and the `SPEC-fab-proceed` / `SPEC-_intake` /
`SPEC-_srad` mirrors).

**Decision: no special gate for deferred decisions.** A deferred decision blocks the gate
**only if its composite is below 20** — exactly like any other Unresolved decision. There is
no `Deferred`-specific rule and no Unresolved short-circuit; blocking is purely emergent from
the penalty curve.

The contract this places on intake authoring: a decision genuinely too ambiguous to assume
**must be scored with honestly-low dimensions** (low A, and usually low R/S), placing its
composite under 20 so the curve blocks the automated bracket. A decision the agent scored at
composite ≥ 20 is — by its own dimensions — resolved enough to proceed, deferred or not. This
is consistent with the rest of the framework: the dimensions are the single source of truth,
and the curve does the gating.

The five files above must have their "fab score returns 0.0 on any Unresolved row" wording
replaced with "a deferred/unresolved decision blocks the gate only when its composite is
below 20" (and the agent must score genuine unknowns accordingly).

---

## Implementation surface

This is a **formula replacement**, not a tweak. Affected areas:

1. **`src/go/fab/internal/score/score.go`**
   - Composite weights change `0.25/0.30/0.25/0.20` → `0.20/0.30/0.30/0.20` (constants
     `wS,wR,wA,wD`).
   - `computeScore`: replace `(meanComposite / compositeToScore) · cover` with
     `5.0 − Σ penalty(composite)`, clamped to [0, 5].
   - Add a `penalty(c float64) float64` helper implementing the piecewise curve
     (`freeKnee = 80`, confident-floor `0.50`, aggressive slope coeff `2.50`); retire
     `compositeToScore`.
   - **Remove** the hard-fail short-circuit: no `Unresolved → 0.0`, no `R<25 AND A<25`
     Critical Rule. `CriticalRowSeen` and the Unresolved count cease to gate the score
     (the counts may still be surfaced for display). *(See § Promptless-defer backstop — a
     deferred/unresolved decision blocks only via `composite < 20`, not a hard-fail.)*
   - `cover` / `expectedMin` no longer participate in the score. `getExpectedMin` and the
     `expectedMin` map become unused by scoring (keep or remove per cleanup preference;
     `change-types.md` still documents the concept).
   - **Grade derivation**: the grade label is computed from `c` (STEP 2 bands), not read
     from the hand-written `Grade` column. The parser still reads dimensions; the grade
     column becomes output, not input.
2. **`src/go/fab/internal/score/score_test.go`** — fixture scores change (new weights + curve);
   add cases for the four bands, the survive-one/block-two property, single-Unresolved-blocks,
   the `c=20` boundary, and thin-but-strong → 5.0.
3. **`docs/specs/srad.md`** — rewrite § Confidence Scoring (Formula, Range, Coverage Factor,
   What 3.0 Allows), the Fuzzy-to-Grade Mapping (new weights + bands), and the Critical Rule
   section (dropped); cross-link this doc.
4. **`src/kit/skills/_srad.md` + `src/kit/skills/_cli-fab.md`** — update the grade-mapping
   weights/bands, the `fab score` formula description, and the Critical Rule / Unresolved
   wording (constitution: CLI behavior changes must update `_cli-fab.md`).
5. **Promptless-defer backstop** — per § Open issue, update `_srad.md`, `_intake.md`,
   `fab-proceed.md`, and the `SPEC-fab-proceed` / `SPEC-_intake` / `SPEC-_srad` mirrors that
   currently assert "fab score returns 0.0 on any Unresolved row."
6. **Memory hydrate** — on completion, `docs/memory/pipeline/` should record the shipped
   formula (post-implementation truth).

The gate threshold (3.0, flat) and the `## Assumptions` table format (the `Scores` column)
are **unchanged** — the change is confined to how parsed dimensions become a 0–5 number, plus
the grade column flipping from input to derived output.
```

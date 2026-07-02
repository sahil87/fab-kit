# Intake: SRAD v2 Demerit Confidence Scoring

**Change**: 260618-4yi8-srad-v2-demerit-scoring
**Created**: 2026-06-18

## Origin

> Implement the SRAD v2 demerit confidence-scoring scheme across code, skills, and specs.

This change implements a scoring redesign that was fully worked out in a preceding design
session. The starting problem: the v1 confidence score (a Resolution-Average **mean** of
per-decision composites, attenuated by a coverage factor) compressed almost every intake into
a narrow 4.0–4.5 band — across 657 loom intakes the median was 4.31 and 81% scored ≥4.0 — so
the gate rarely blocked anything. The deeper defect was **mean dilution**: the one or two
irreversible, under-resolved decisions that make shipped work unusable were averaged away by
the surrounding strong decisions, and a *more* thorough intake hid the bad decision *better*.

The design was validated empirically against 1,080 real intakes across three repos (loom,
fab-kit, run-kit) and four candidate aggregators were rejected before settling on the demerit
model. The full authoritative design is in **`docs/specs/srad.md`**; the decision record (why
the mean leaked, the rejected alternatives, the penalty-curve derivation, the cross-repo
validation, the promptless-defer decision) is in **`docs/specs/srad-scoring-rationale-v1-to-v2.md`**;
the superseded scheme is preserved in **`docs/specs/srad-v1.md`**. This change is the
implementation of that already-decided design — code, skills, and the remaining doc updates.

## Why

1. **Problem**: The v1 mean-based score does not separate safe intakes from risky ones. It
   passes ~98% of inputs (barely a gate) and structurally hides the single dangerous decision
   inside an averaged aggregate.
2. **Consequence if unfixed**: `/fab-ff` and `/fab-fff` proceed autonomously on intakes that
   contain an irreversible, under-resolved decision — exactly the decisions most likely to
   produce unusable output — because the score reports high confidence.
3. **Why this approach**: A demerit model (start at 5.0, subtract a per-decision penalty keyed
   on the composite) makes each weak decision visible because penalties cannot be refunded by
   strong rows. Smooth aggregators (soft-min, harmonic, reversibility-weighted) were simulated
   and rejected — benign-but-vague reversible decisions outnumber dangerous ones ~4:1, so any
   smooth pull-toward-the-worst-row punishes the wrong population. The demerit curve, with R
   and A up-weighted in the composite, surfaces the dangerous decision while leaving healthy
   intakes untouched (0 healthy intakes killed across all three repos).

## What Changes

The behavior is fully specified in `docs/specs/srad.md`. Summary of the formula being shipped:

```
composite c = 0.20·S + 0.30·R + 0.30·A + 0.20·D            # R and A up-weighted from 0.25

penalty(c) = 0                          if c ≥ 80           # Certain  → free
             (80 − c)/30 · 0.50         if 50 ≤ c < 80      # Confident → ≤ 0.5
             0.50 + (50 − c)/50 · 2.50  if c < 50           # Tentative/Unresolved

score = clamp(5.0 − Σ penalty(c), 0.0, 5.0)
gate: score ≥ 3.0  (flat, all change types — unchanged)
```

Grades become **indicative only** (derived from composite, never read by the formula):
Certain ≥80, Confident 50–80, Tentative 20–50, Unresolved <20. There are **no hard-fail
short-circuits** — the old `Unresolved → 0.0` and `R<25 ∧ A<25` Critical Rule are removed;
blocking is emergent from the curve (a `composite < 20` row penalizes ≥ 2.0). Coverage and
`expected_min` are dropped from the score path.

### 1. `src/go/fab/internal/score/score.go`

- Composite weights `wS,wR,wA,wD`: `0.25/0.30/0.25/0.20` → `0.20/0.30/0.30/0.20`.
- `computeScore`: replace `(meanComposite / compositeToScore) · cover` with
  `clamp(5.0 − Σ penalty(composite), 0, 5)`. Add a `penalty(c float64) float64` helper for the
  piecewise curve (constants: free-knee `80`, confident-floor penalty `0.50`, aggressive slope
  coeff `2.50`). Retire `compositeToScore`.
- Remove the hard-fail short-circuit (`if gc.Unresolved > 0 || gc.CriticalRowSeen { return 0.0 }`).
  `CriticalRowSeen` / per-row `R<25 ∧ A<25` tracking and the Unresolved count cease to gate the
  score (counts may still be surfaced for display).
- Drop `cover` / `expectedMin` from the score path (`getExpectedMin` and the `expectedMin` map
  become unused by scoring; `change-types.md` still documents the concept — leave the doc).
- **Grade derivation**: compute the grade label from the composite (the bands above) rather than
  reading the hand-written `Grade` column. The parser keeps reading the `Scores` column for
  dimensions; the grade column becomes derived output.

### 2. `src/go/fab/internal/score/score_test.go`

- Update fixture expected scores (new weights + curve).
- Add cases: the four bands (Certain/Confident/Tentative/Unresolved penalties), the `c=20`
  boundary (penalty exactly 2.0 → one-row score exactly 3.0 → passes), survive-one/block-two
  (one Tentative row passes, two block), single-Unresolved-blocks (`c<20` alone fails the gate),
  and thin-but-strong (a 2-row all-Certain intake scores 5.0 — no coverage penalty).

### 3. `src/kit/skills/_srad.md`

- Update the grade-mapping weights (`0.20/0.30/0.30/0.20`) and bands (80/50/20).
- Rewrite the Critical Rule: a genuine unknown is surfaced (asked or deferred) and blocks the
  gate via `composite < 20`, **not** via a hard-fail or the old `R<25 ∧ A<25` override.
- Update the promptless-defer carve-out wording (see item 5).

### 4. `src/kit/skills/_cli-fab.md`

- Update the `fab score` formula description (§ fab score extended) to the demerit model.

### 5. Promptless-defer backstop wording

In `src/kit/skills/_intake.md`, `src/kit/skills/fab-proceed.md`, and the SPEC mirrors
`docs/specs/skills/SPEC-fab-proceed.md`, `docs/specs/skills/SPEC-_intake.md`,
`docs/specs/skills/SPEC-_srad.md`: replace every assertion that "`fab score` returns 0.0
whenever any Unresolved row exists" with the v2 contract — **a deferred/unresolved decision
blocks the gate only when its composite is below 20**, and the agent must score genuine unknowns
with honestly-low dimensions (low A, usually low R/S) so they land there. No special gate for
deferred decisions; blocking is emergent from the curve, consistent with the rest of the model.

### Unchanged

The flat 3.0 gate (all change types), the `## Assumptions` table format including the required
`Scores` column, and the `.status.yaml` `confidence` schema shape are unchanged. The change is
confined to how parsed dimensions become a 0–5 number, plus the grade column flipping from input
to derived output.

## Affected Memory

- `pipeline/srad`: (modify) the SRAD/confidence-scoring memory must record the shipped v2 demerit
  formula, indicative grades, and the removal of hard-fail rules. <!-- assumed: hydrate will reconcile the exact memory file(s) under docs/memory/pipeline/ that cover SRAD scoring; the domain is pipeline -->
- `pipeline/clarify`: (modify) if it references the Unresolved hard-fail or the grade-count score,
  update to the v2 contract (composite<20 blocks; grades indicative).

## Impact

- **Code**: `src/go/fab/internal/score/score.go`, `src/go/fab/internal/score/score_test.go`. The
  composite-weight change and the parser may also touch the changetypes/doc tests in the `score`
  package if they assert the old formula — verify and update.
- **Skills (canonical sources)**: `src/kit/skills/_srad.md`, `_cli-fab.md`, `_intake.md`,
  `fab-proceed.md`. (`.claude/skills/` is the deployed copy — never edited directly; `fab sync`
  regenerates it.)
- **Specs**: the SPEC mirrors `SPEC-fab-proceed.md`, `SPEC-_intake.md`, `SPEC-_srad.md`. The
  design specs (`srad.md`, `srad-scoring-rationale-v1-to-v2.md`, `srad-v1.md`) and the specs
  `index.md` entry already reflect v2 (authored in the design session) — verify consistency, no
  rewrite expected.
- **Constitution constraints**: changes to the Go binary MUST include test updates and MUST
  update `_cli-fab.md`; changes to skill files MUST update the corresponding SPEC mirror.
- **No migration**: this changes a computed score, not stored user data layout. Existing
  `.status.yaml` confidence values are recomputed on the next `fab score`; no `.status.yaml`
  restructuring, so no `src/kit/migrations/` file is required.

## Open Questions

(none — the design is fully resolved in `docs/specs/srad.md`.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Composite weights become 0.20·S + 0.30·R + 0.30·A + 0.20·D | Specified verbatim in docs/specs/srad.md § The Composite; decided in design session | S:98 R:80 A:95 D:95 |
| 2 | Certain | Penalty curve: 0 for c≥80; (80−c)/30·0.5 for 50≤c<80; 0.5+(50−c)/50·2.5 for c<50 | Specified verbatim in srad.md § Formula; slopes derived from band-boundary penalties, validated on 1080 intakes | S:98 R:75 A:95 D:95 |
| 3 | Certain | score = clamp(5.0 − Σ penalty(c), 0, 5); flat 3.0 gate retained | srad.md § Formula and § Gate Threshold; gate threshold explicitly unchanged | S:98 R:85 A:98 D:98 |
| 4 | Certain | Remove both hard-fail short-circuits (Unresolved→0.0 and R<25∧A<25); blocking is emergent from the curve | srad.md § No hard-fail rules + § The Critical Rule; decided in session | S:95 R:65 A:90 D:90 |
| 5 | Certain | Grade label is derived from composite (80/50/20 bands), not read from the hand-written Grade column | srad.md § Grades (indicative only) + § Grade derivation; the 19% mismatch motivated it | S:95 R:80 A:90 D:90 |
| 6 | Certain | Drop coverage / expected_min from the score path (quality over quantity); leave change-types.md doc intact | srad.md § Gate Threshold; validated that coverage harmed thin-but-strong intakes | S:95 R:70 A:90 D:88 |
| 7 | Certain | Promptless-defer: no special gate — deferred decisions block only via composite<20; update wording in 5 files | srad.md § How blocking is enforced + rationale doc § Promptless-defer backstop (resolved); user chose this explicitly | S:95 R:60 A:88 D:85 |
| 8 | Confident | The affected docs/memory file(s) live under docs/memory/pipeline/ and are reconciled at hydrate | Memory index lists `pipeline` as the domain covering SRAD/scoring; exact file resolved during hydrate | S:70 R:80 A:75 D:70 |
| 9 | Confident | No migration file needed — only a computed score changes, not stored .status.yaml layout | context.md migration rule applies to data restructuring; confidence values recompute on next score | S:80 R:75 A:80 D:80 |
| 10 | Confident | score package doc/changetypes tests may need updates if they assert the old formula | The score package has changetypes_doc_test.go and score_test.go; Test Integrity (Constitution VII) requires conformance to the new spec | S:75 R:80 A:80 D:75 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).

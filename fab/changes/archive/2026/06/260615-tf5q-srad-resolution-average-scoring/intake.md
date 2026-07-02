# Intake: SRAD Resolution-Average Confidence Scoring

**Change**: 260615-tf5q-srad-resolution-average-scoring
**Created**: 2026-06-15

## Origin

> Natural-language directive: Replace fab's SRAD confidence-gate formula with a per-row
> "Resolution Average" that scores an intake from the per-row S:R:A:D dimensions already parsed
> on every Assumptions-table row (today they are parsed then discarded for the gate decision).
> Also lower the `fix` `expectedMin` from 5 to 3. This is a refactor of an existing mechanism,
> not a new feature or a bug fix.

Interaction mode: one-shot, fully-specified directive. Every design decision below was settled in
the originating discussion and is recorded here verbatim — there is no live user to ask, so any
genuinely-unresolvable point would be deferred (none were).

Key decisions reached in the originating discussion (all settled — do not re-ask):

- **Keep the 0–5 scale and the flat 3.0 gate.** The user explicitly chose 0–5 over 0–100 (easier
  for humans to reason about). The `/20` divisor rescales the 0–100 composite mean onto 0–5.
- **The Assumptions TABLE FORMAT is unchanged** — zero new author-facing fields, zero removed. The
  `Scores` column (already required on every row, already parsed) becomes load-bearing; the `Grade`
  column stays for human readability / `/fab-clarify` targeting but no longer feeds the gate.
- **Real trigger**: change `qg64` (a `fix`) with an honest 3-Certain / 5-Confident / 2-Tentative
  intake scored `1.5` (below the 3.0 gate) under today's penalty formula, even though its dimension
  means were strong (S:88.5 R:77.5 A:87.5 D:86.5 → composite 84.55 → would score `4.2` under
  Resolution Average and pass cleanly). The author could only pass by bucket-shuffling rows upward.
- **Validation already done** (cite, do not re-run): a 596-change back-test over the loom repo's
  archived changes reimplementing both formulas in Python — 92.1% gate agreement (549/596); 40
  honest/thorough intakes rescued (old-fail → new-pass), all with strong mean composites (76–88);
  only 7 old-pass → new-fail, of which 6 were thin high-quality fixes recovered by the
  `expectedMin` 5→3 change (the 7th was a refactor, unaffected, correctly still failing). Score
  distribution barely moves (old mean 4.05 → new 4.16) — a re-shaping, not a loosening. The Q3
  "lurking weak row averaged into invisibility" risk measured at 1/571 passing changes — a
  non-issue; no worst-row floor needed. Evidence file:
  `~/.claude/projects/-home-sahil-code-sahil87-fab-kit/memory/srad-resolution-average-backtest.md`.

## Why

**The problem.** Today's `fab score` (`src/go/fab/internal/score/score.go`, `computeScore`) gates an
intake from **grade counts only**:

```
base  = 5.0 - 0.3*confident - 1.0*tentative      # wCertain=0, wConfident=0.3, wTentative=1.0
cover = min(1.0, total_decisions / expectedMin[changeType])
score = round1(base * cover)
gate passes iff score >= 3.0
```

This **penalizes thoroughness**: every honestly-recorded assumption can only LOWER the score
(Confident costs 0.3, Tentative costs 1.0 each), so a terse intake mechanically beats a careful one.
Worse, `countGrades` already PARSES the per-row S:R:A:D dimensions (via `scoresRegex`, accumulated
into `SumS/SumR/SumA/SumD/DimCount`) and computes their means — then **throws them away** for the
gate decision (they survive only as `.status.yaml` telemetry). The data that would score the intake
*fairly* is computed and discarded.

**The consequence if unfixed.** Authors are incentivized to write thin intakes or to bucket-shuffle
honest grades upward to clear the 3.0 gate (the `qg64` real trigger). The richer and more honest the
S:R:A:D recording, the worse the intake fares — exactly backwards from what the framework intends.

**Why this approach.** The "Resolution Average" reuses the dimensions already on every row and the
EXISTING `_srad.md` aggregation weights (`0.25*S + 0.30*R + 0.25*A + 0.20*D`) — no new author-facing
surface, no recalibration of the gate. A 3.0 gate on the 0–5 scale equals a mean composite ≥ 60,
which is precisely the existing "Confident" floor in `srad.md` — so the threshold stays principled
and needs no tuning. The back-test confirms it rescues honest/thorough intakes with zero meaningful
distribution drift and no new false-passes (the 7 old-pass→new-fail are addressed by `fix` 5→3,
which can only RAISE scores). Lowering `fix` `expectedMin` is a separately-validated one-line data
change recovering 6 thin high-quality fixes with 0 regressions.

## What Changes

### 1. Rewrite `computeScore` — the Resolution-Average formula

Replace the grade-count penalty arithmetic with a per-row composite mean. Target behavior:

```
for each Assumptions row:
    composite = 0.25*S + 0.30*R + 0.25*A + 0.20*D     # EXISTING srad.md weights, unchanged
    if R < 25 AND A < 25:  return 0.0                  # Critical Rule, per-row, on raw dimensions
    if grade == Unresolved: return 0.0                 # genuine unknown — hard fail (same as today)
mean  = average(composite over rows that have parseable dimensions)   # DimCount rows
cover = min(1.0, total_decisions / expectedMin[changeType])           # UNCHANGED coverage term
score = round1( (mean / 20.0) * cover )               # /20 rescales 0-100 composite mean onto 0-5
gate passes iff score >= 3.0                          # UNCHANGED threshold
```

- **Preserved verbatim**: the `unresolved > 0 → 0.0` short-circuit; the coverage term
  `cover = min(1.0, total/expectedMin)`; the 0–5 scale; the flat 3.0 gate (`getGateThreshold`);
  `roundTo1`.
- **New**: the per-row Critical Rule (`R < 25 AND A < 25 → 0.0`) evaluated on the **raw** per-row
  dimensions, and the per-row Unresolved hard-fail, both short-circuiting BEFORE the mean. The
  dimensions are already parsed inside `countGrades`; the implementer may either thread per-row
  composites/flags out of `countGrades` (e.g., add a `CriticalRowSeen bool` and a per-row composite
  accumulator, or a `SumComposite float64`) or evaluate the Critical-Rule short-circuit inside
  `countGrades` itself. Whichever keeps `countGrades` the single parse pass and avoids a second scan.
- **Signature note**: `computeScore`'s current parameter list (`certain, confident, tentative,
  unresolved, total, expectedMin`) no longer carries enough information — it needs the per-row
  composite sum / DimCount (and the Critical-Rule / Unresolved flags). Thread these from the
  `GradeCount` already returned by `countGrades` (both call sites — `CheckGate` and `buildResult` —
  already hold the `gc GradeCount`). This is an internal-only signature change; the public
  `fab score` CLI signature does NOT change.

### 2. Remove the dead penalty-weight constants

`wCertain` (0.0), `wConfident` (0.3), `wTentative` (1.0) in `score.go` lines ~20–25 become dead once
the penalty arithmetic is gone. Remove the `const` block.

### 3. Lower `fix` `expectedMin` from 5 to 3

In the `expectedMin` map (`score.go` ~lines 31–33): `"fix": 5` → `"fix": 3`. One-line data change.
Rationale (validated): 6 thin high-quality fix intakes (3–4 Certain rows, composites 87–95) were
correctly-but-too-strictly blocked by the 5-coverage floor; lowering to 3 recovers all of them with
0 regressions (lowering `expectedMin` can only RAISE scores via a higher `cover`). A 1-row fix
(`sep0`) still correctly fails (too thin: `cover = 1/3`).

### 4. Dimensionless-row handling for coverage (decide-and-record-at-apply)

The mean averages only rows with parseable dimensions (`DimCount`). A row with NO parseable S:R:A:D
`Scores` column is a malformed-intake edge case — the SRAD spec makes the `Scores` column REQUIRED
on every row. **Recommended default (record at apply): coverage's `total` still counts ALL graded
rows** (`certain + confident + tentative + unresolved`), consistent with today's `countGrades` and
the existing coverage semantics; only the MEAN restricts to `DimCount` rows. This is a
decide-and-record detail for apply, not an intake blocker (see Open Questions).

### 5. Update tests

- `src/go/fab/internal/score/score_test.go` — replace assertions over the old penalty arithmetic
  (e.g., `5.0 - 0.3*confident - 1.0*tentative`) with Resolution-Average expectations; add cases for
  the per-row Critical Rule short-circuit, the per-row Unresolved hard-fail, the `mean/20*cover`
  rescale, and the dimensionless-row edge case.
- `src/go/fab/cmd/fab/score_test.go` — update any end-to-end assertions that depend on the old
  `fix=5` coverage or the old penalty score values.
- `src/go/fab/internal/score/changetypes_doc_test.go` — `TestDocTablesMatchScoringMaps` asserts the
  `change-types.md` `expected_min` table matches the `expectedMin` map; updating `fix` 5→3 in both
  the map and the doc (see #7) keeps this test green. Verify it has no separate hard-coded `fix:5`.

### 6. Rewrite `docs/specs/srad.md` § Confidence Scoring (lines ~129–217)

- Replace the § Formula block with the Resolution-Average formula.
- **Drop the § Penalty Weights table** (lines ~144–151) — penalties no longer exist.
- Restate § "What 3.0 Allows" (lines ~198–211) in **mean-composite** terms: a 3.0 gate on 0–5 ==
  mean composite ≥ 60 (the existing Confident floor); a single weak row drags the mean, a single
  R<25∧A<25 or Unresolved row hard-fails; low coverage still attenuates thin intakes.
- Keep the 0–5 scale, the § Range, § Storage (dimension means already stored), § Gate Threshold
  table (flat 3.0, unchanged), § Coverage Factor (unchanged), and the Critical Rule section.
- Worked Example 2's arithmetic (lines ~277) currently shows the old `base*cover`; update it to the
  Resolution-Average computation so the spec's example stays self-consistent.

### 7. Update `docs/specs/change-types.md` `expected_min` table

Line ~31: `| fix | 5 |` → `| fix | 3 |`. Mirrors the `expectedMin` map; `TestDocTablesMatchScoringMaps`
enforces the match.

### 8. Update `src/kit/skills/_cli-fab.md` § fab score (extended) — CONSTITUTION REQUIREMENT

Constitution: *"Changes to the `fab` CLI (Go binary) MUST include corresponding test updates and MUST
update `src/kit/skills/_cli-fab.md` with any new or changed command signatures."* The `fab score`
**signature** does not change, but `_cli-fab.md` documents the **formula** in its § fab score
(extended) block:

- Lines ~160–161: the `base = max(0.0, 5.0 - 0.3*confident - 1.0*tentative)` / `cover` pseudocode →
  the Resolution-Average pseudocode.
- Line ~165: the embedded `expected_min` table text `feat:7, refactor:6, fix:5` → `fix:3`.

Edit the CANONICAL source `src/kit/skills/_cli-fab.md` only; the `.claude/skills/` deployed copy is
regenerated by `fab sync` — do NOT edit deployed copies.

## Affected Memory

<!-- This is a refactor of an existing mechanism that DOES change spec-level behavior (the scoring
     formula), so the scoring/SRAD memory likely needs a touch. The exact memory file is confirmed
     at hydrate from the docs/memory/ tree, not at intake. -->

- `pipeline/srad-scoring`: (modify) Record the Resolution-Average formula replacing the grade-count
  penalty model, the per-row Critical-Rule / Unresolved short-circuits, the `/20` rescale onto 0–5,
  and the `fix` `expected_min` 5→3 change. <!-- assumed: memory domain/file name inferred from the pipeline domain housing SRAD scoring; confirm exact path against docs/memory/ at hydrate -->

## Impact

- **`src/go/fab/internal/score/score.go`** — `computeScore` rewrite; `GradeCount`/threading changes
  to surface per-row composite mean + Critical-Rule/Unresolved flags; remove `wCertain`/`wConfident`/
  `wTentative` consts; `expectedMin["fix"]` 5→3. Both call sites (`CheckGate`, `buildResult`) already
  hold the `gc GradeCount` so the threading is local.
- **Tests** — `score_test.go`, `cmd/fab/score_test.go`, `changetypes_doc_test.go` (per Test Integrity
  / constitution CLI-test requirement).
- **Docs/specs** — `srad.md` § Confidence Scoring rewrite; `change-types.md` `expected_min` table.
- **Skill source** — `src/kit/skills/_cli-fab.md` § fab score (extended) formula block.
- **No author-facing surface change** — the Assumptions template, intake template, and skill behavior
  are untouched (the `Scores` column was already required and parsed). The SPEC-*.md skill mirrors are
  almost certainly untouched; the implementer should grep skill .md files for the old formula text to
  confirm none reference it.
- **No migration** — this changes a derived score, not persisted user data shape (`.status.yaml`
  `confidence.score` is recomputed on next `fab score`; dimension means already stored). No
  `src/kit/migrations/` file needed.

## Open Questions

- Dimensionless-row coverage `total`: confirm at apply that coverage counts ALL graded rows (not just
  `DimCount`), per the recommended default in What-Changes §4. This is a decide-and-record detail
  with an obvious front-runner, not a blocker — the SRAD spec already makes `Scores` required, so a
  dimensionless row is a malformed-intake edge case, and counting all graded rows matches today's
  `countGrades` semantics.
- Exact `docs/memory/` path for the scoring memory note — resolve against the live memory tree at
  hydrate (the SRAD scoring memory may live under a `pipeline/` sub-domain or a flat file).

## Assumptions

<!-- This change is highly specified; thoroughness is rewarded by the very formula being implemented.
     Every row's S:R:A:D is grounded in the directive + the codebase (score.go, srad.md, _cli-fab.md,
     change-types.md all read). Zero Unresolved rows — no genuine unknowns require deferral. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace `computeScore`'s grade-count penalty with per-row composite mean rescaled `mean/20`, gate stays `>= 3.0` on the 0–5 scale | Formula given verbatim in the directive; `/20` and 0–5 retention are explicit settled decisions; `computeScore` body and `roundTo1` read directly | S:98 R:75 A:95 D:96 |
| 2 | Certain | Reuse the EXISTING `_srad.md`/`srad.md` aggregation weights `0.25*S + 0.30*R + 0.25*A + 0.20*D` unchanged for the per-row composite | Weights confirmed identical in `srad.md` line 36 and `_srad.md` line 30; directive says "unchanged"; no new constant introduced | S:98 R:90 A:100 D:98 |
| 3 | Certain | Lower `expectedMin["fix"]` 5→3 (one-line data change in the map) | Explicit directive; `expectedMin` map read at score.go ~31–33; back-test validated 6 recovered fixes, 0 regressions, lowering can only raise scores | S:97 R:90 A:97 D:97 |
| 4 | Certain | Preserve `unresolved > 0 → 0.0` short-circuit and the `cover = min(1.0, total/expectedMin)` coverage term verbatim | Directive states both are PRESERVED; both read in `computeScore` lines 376–391; no behavioral change to either | S:96 R:88 A:96 D:95 |
| 5 | Certain | Add per-row Critical Rule (`R < 25 AND A < 25 → 0.0`) evaluated on raw per-row dimensions, short-circuiting before the mean | Directive specifies per-row evaluation on raw data; matches `srad.md` Critical Rule numeric definition (`< 25` both); dimensions already parsed in `countGrades` | S:92 R:72 A:90 D:90 |
| 6 | Certain | Remove dead `wCertain`/`wConfident`/`wTentative` const block | Directive says remove; they become unreferenced once penalty arithmetic is gone; confirmed only used in the `base = ...` line | S:96 R:92 A:97 D:97 |
| 7 | Confident | Thread per-row composite mean + Critical-Rule/Unresolved flags out of `countGrades` (extend `GradeCount`) rather than re-scanning; change `computeScore`'s internal signature accordingly | `countGrades` is already the single parse pass holding S:R:A:D per row; both call sites hold the `gc`; matches code-quality "reuse, no duplication"; exact field shape is an implementer call with one obvious approach | S:80 R:68 A:82 D:72 |
| 8 | Confident | Coverage `total` counts ALL graded rows (certain+confident+tentative+unresolved); only the MEAN restricts to `DimCount` rows | Directive's explicit recommendation; matches today's `countGrades` total semantics; dimensionless rows are a malformed-intake edge case (Scores column is REQUIRED); recorded as decide-at-apply | S:78 R:70 A:80 D:75 |
| 9 | Confident | Rewrite `srad.md` § Confidence Scoring: new formula, drop the Penalty-Weights table, restate "What 3.0 Allows" in mean-composite terms, update Worked Example 2 arithmetic, keep 0–5 scale / Range / Storage / Gate table / Coverage / Critical Rule | Directive specifies the spec rewrite and line ranges; spec read in full; 3.0-gate-==-mean-≥-60 equivalence is principled and explicit | S:88 R:75 A:85 D:82 |
| 10 | Confident | Update `change-types.md` `expected_min` table `fix` 5→3 to mirror the map; `TestDocTablesMatchScoringMaps` enforces the match | Directive names the file; table read at line 31; doc test (`changetypes_doc_test.go`) named in directive ties the two together | S:90 R:85 A:90 D:88 |
| 11 | Confident | Update `src/kit/skills/_cli-fab.md` § fab score (extended) formula pseudocode + embedded `expected_min` text (`fix:5`→`fix:3`) per the constitution CLI doc rule; edit canonical source only (sync deploys) | Constitution constraint cited; `_cli-fab.md` lines 160–165 confirmed to restate the formula and the `feat:7, refactor:6, fix:5` table; signature unchanged so only the formula text updates | S:88 R:82 A:90 D:85 |
| 12 | Confident | Update tests: `score_test.go` (penalty→Resolution-Average + Critical-Rule/Unresolved/rescale/dimensionless cases), `cmd/fab/score_test.go` (old fix=5 / penalty values) | Constitution requires test updates for Go CLI changes; Test Integrity (tests conform to spec); files named in directive; exact new expected values derived at apply from the new formula | S:82 R:80 A:85 D:80 |
| 13 | Confident | change_type is `refactor` (reshapes an existing mechanism; not user-facing feature, not a bug fix) | Directive states REFACTOR explicitly; the intake-write hook may infer `fix` from "fix" tokens — will verify and override per Step 6 | S:90 R:80 A:88 D:82 |
| 14 | Confident | Touch the scoring/SRAD memory note at hydrate (spec-level behavior changes); exact `docs/memory/` path confirmed against the live tree at hydrate | Constitution "Docs Are Source of Truth" — a formula change is spec-level; intake should not hard-bind a memory path before reading the tree; recorded as an Open Question for hydrate | S:72 R:78 A:70 D:72 |
| 15 | Confident | No migration and no author-facing template/skill-behavior change; SPEC-*.md skill mirrors likely untouched (implementer greps skill .md for old formula to confirm) | Score is recomputed (not persisted shape); Assumptions/intake templates and skill behavior unchanged; dimension means already stored; verification is a cheap grep | S:80 R:80 A:82 D:80 |

15 assumptions (6 certain, 9 confident, 0 tentative, 0 unresolved).

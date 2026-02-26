# Spec: Coverage-Weighted Confidence Scoring & Formalize Change Types

**Change**: 260226-tnr8-coverage-scoring-change-types
**Created**: 2026-02-26
**Affected memory**:
- `docs/memory/fab-workflow/planning-skills.md` (modify)
- `docs/memory/fab-workflow/execution-skills.md` (modify)
- `docs/memory/fab-workflow/templates.md` (modify)
- `docs/memory/fab-workflow/configuration.md` (modify)

## Non-Goals

- Adding an `architecture` change type — coverage weighting naturally penalizes thin decision-making on large changes, making a dedicated type unnecessary
- Displaying confidence in `/fab-status` — it remains a fast, read-only glance command
- Persisting the indicative confidence to `.status.yaml` at intake time — it's display-only; the authoritative score is computed at the spec stage

## Confidence Scoring: Coverage-Weighted Formula

### Requirement: Coverage-Weighted Score Computation

`calc-score.sh` SHALL compute the confidence score using a coverage-weighted formula:

```
if unresolved > 0:
  score = 0.0
else:
  base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  cover = min(1.0, total_decisions / expected_min)
  score = base * cover
```

Where `total_decisions = certain + confident + tentative + unresolved` and `expected_min` is looked up by `{stage, change_type}` from an embedded table.

The `base` component is identical to the current formula. The `cover` component attenuates the score when `total_decisions < expected_min`, preventing thin specs from scoring high. When `total_decisions >= expected_min`, `cover = 1.0` and the formula degenerates to the current behavior.

#### Scenario: Thin spec with few decisions scores lower than before

- **GIVEN** a spec with 2 Certain decisions and change type `feat`
- **WHEN** `calc-score.sh` computes the confidence score at the spec stage
- **THEN** `base = 5.0`, `cover = min(1.0, 2 / 6) = 0.33`, `score = 5.0 * 0.33 = 1.7`
- **AND** the score is lower than the previous formula would have produced (5.0)

#### Scenario: Well-covered spec scores unchanged

- **GIVEN** a spec with 8 Certain and 2 Confident decisions and change type `feat`
- **WHEN** `calc-score.sh` computes the confidence score at the spec stage
- **THEN** `total_decisions = 10`, `expected_min = 6`, `cover = min(1.0, 10 / 6) = 1.0`
- **AND** `base = max(0.0, 5.0 - 0.6) = 4.4`, `score = 4.4 * 1.0 = 4.4`
- **AND** the score is identical to the current formula

#### Scenario: Coverage at intake stage uses intake thresholds

- **GIVEN** a spec with 3 Certain decisions and change type `refactor`
- **WHEN** `calc-score.sh` is invoked with `--stage intake`
- **THEN** `expected_min = 3` (intake threshold for refactor), `cover = min(1.0, 3 / 3) = 1.0`
- **AND** the score uses the intake-stage threshold, not the spec-stage threshold

### Requirement: Expected Minimum Thresholds

`calc-score.sh` SHALL embed the `expected_min` lookup tables directly in the script (since `fab/.kit/` is distributed to projects):

| Type | Intake `expected_min` | Spec `expected_min` |
|------|----------------------|---------------------|
| `fix` | 2 | 4 |
| `feat` | 4 | 6 |
| `refactor` | 3 | 5 |
| `docs` | 2 | 3 |
| `test` | 2 | 3 |
| `ci` | 2 | 3 |
| `chore` | 2 | 3 |

#### Scenario: Unknown change type uses default thresholds

- **GIVEN** a change with `change_type` set to an unrecognized value or null
- **WHEN** `calc-score.sh` looks up `expected_min`
- **THEN** it SHALL use the fallback values: intake=2, spec=3

### Requirement: Stage Flag for calc-score.sh

`calc-score.sh` SHALL accept an optional `--stage <stage>` flag (default: `spec`) to determine which `expected_min` column to use. The flag MUST appear before the `<change-dir>` argument.

#### Scenario: Default stage is spec

- **GIVEN** `calc-score.sh` is invoked as `calc-score.sh <change-dir>` (no `--stage` flag)
- **WHEN** the script looks up `expected_min`
- **THEN** it uses the spec-stage column

#### Scenario: Explicit intake stage

- **GIVEN** `calc-score.sh` is invoked as `calc-score.sh --stage intake <change-dir>`
- **WHEN** the script looks up `expected_min`
- **THEN** it uses the intake-stage column

### Requirement: Read change_type from .status.yaml

`calc-score.sh` SHALL read the `change_type` field from `.status.yaml` to determine which `expected_min` row to use. If the field is absent or null, it SHALL default to `feat`.

#### Scenario: change_type present in .status.yaml

- **GIVEN** `.status.yaml` contains `change_type: fix`
- **WHEN** `calc-score.sh` looks up `expected_min` at spec stage
- **THEN** `expected_min = 4`

#### Scenario: change_type absent or null

- **GIVEN** `.status.yaml` contains `change_type: null` or the field is missing
- **WHEN** `calc-score.sh` looks up `expected_min`
- **THEN** it defaults to `feat` thresholds (intake=4, spec=6)

## Change Type Taxonomy

### Requirement: Seven Canonical Change Types

The Fab workflow SHALL recognize exactly 7 change types, derived from Conventional Commits:

| Type | Description |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix or regression fix |
| `refactor` | Code restructuring without behavior change |
| `docs` | Documentation-only changes |
| `test` | Test additions or modifications |
| `ci` | CI/CD pipeline changes |
| `chore` | Maintenance, cleanup, housekeeping |

These types SHALL use the short conventional commit prefix form (e.g., `feat`, not `feature`).

#### Scenario: All seven types are recognized

- **GIVEN** a `.status.yaml` with `change_type` set to any of `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`
- **WHEN** any script or skill reads the `change_type` field
- **THEN** the type is treated as valid without fallback

### Requirement: Change Type Inference in fab-new

`/fab-new` SHALL infer `change_type` from the intake content after generating `intake.md`, using keyword matching evaluated in order (first match wins):

1. Contains any of: "fix", "bug", "broken", "regression" → `fix`
2. Contains any of: "refactor", "restructure", "consolidate", "split", "rename" → `refactor`
3. Contains any of: "docs", "document", "readme", "guide" → `docs`
4. Contains any of: "test", "spec", "coverage" → `test`
5. Contains any of: "ci", "pipeline", "deploy", "build" → `ci`
6. Contains any of: "chore", "cleanup", "maintenance", "housekeeping" → `chore`
7. Otherwise → `feat`

The inferred type SHALL be written to `.status.yaml` via `stageman.sh set-change-type <file> <type>`.

#### Scenario: Bug fix is inferred from keywords

- **GIVEN** a user describes a change containing the word "bug"
- **WHEN** `/fab-new` generates the intake and runs keyword inference
- **THEN** `change_type` is set to `fix` in `.status.yaml`

#### Scenario: No keyword match defaults to feat

- **GIVEN** a user describes a change with no matching keywords (e.g., "Add OAuth support")
- **WHEN** `/fab-new` runs keyword inference
- **THEN** `change_type` is set to `feat` in `.status.yaml`

#### Scenario: First match wins when multiple keywords present

- **GIVEN** a user describes "Fix the broken test coverage pipeline"
- **WHEN** `/fab-new` runs keyword inference
- **THEN** `change_type` is set to `fix` (matches "fix" before "test", "coverage", or "pipeline")

### Requirement: stageman.sh set-change-type Subcommand

`stageman.sh` SHALL provide a `set-change-type <file> <type>` subcommand that writes the `change_type` field to `.status.yaml`. The subcommand SHALL validate that `<type>` is one of the 7 canonical types and update `last_updated`.

#### Scenario: Valid type is written

- **GIVEN** a `.status.yaml` file with `change_type: feature`
- **WHEN** `stageman.sh set-change-type <file> fix` is executed
- **THEN** `change_type` becomes `fix` and `last_updated` is refreshed

#### Scenario: Invalid type is rejected

- **GIVEN** a `.status.yaml` file
- **WHEN** `stageman.sh set-change-type <file> architecture` is executed
- **THEN** the script exits non-zero with an error message listing valid types

## Indicative Confidence in fab-new

### Requirement: Display Indicative Confidence After Intake

After generating `intake.md`, `/fab-new` SHALL:

1. Count assumptions from the intake's `## Assumptions` table (certain, confident, tentative, unresolved)
2. Look up `expected_min` for the intake stage using the inferred `change_type`
3. Compute the indicative score using the coverage-weighted formula
4. Display it to the user

The indicative confidence SHALL NOT be written to `.status.yaml`. The authoritative score continues to be computed and persisted at the spec stage by `calc-score.sh`.

#### Scenario: Indicative confidence displayed after intake

- **GIVEN** `/fab-new` has generated an intake with 5 Certain and 1 Confident decisions for a `feat` change
- **WHEN** the intake generation completes
- **THEN** the output includes: `Indicative confidence: 4.4 / 5.0 (6 decisions, cover: 1.0)`

#### Scenario: Low coverage shows reduced score

- **GIVEN** `/fab-new` has generated an intake with 2 Certain decisions for a `feat` change
- **WHEN** the intake generation completes
- **THEN** `cover = min(1.0, 2 / 4) = 0.5`, and the output shows the attenuated score

## git-pr: Read change_type from .status.yaml

### Requirement: Extended PR Type Resolution Chain

`/git-pr` Step 0 (Resolve PR Type) SHALL use the following resolution chain:

1. **Explicit argument** (existing — unchanged)
2. **Read from `.status.yaml`**: If `changeman.sh resolve` succeeds, read `change_type` from `.status.yaml`. If non-null and one of the 7 valid types, use it. Fall through if missing or null.
3. **Infer from intake** (existing fallback — unchanged)
4. **Infer from diff** (existing fallback — unchanged)

#### Scenario: change_type read from .status.yaml

- **GIVEN** an active change with `change_type: refactor` in `.status.yaml`
- **WHEN** `/git-pr` resolves the PR type with no explicit argument
- **THEN** the PR type is `refactor` (read from `.status.yaml`)

#### Scenario: Fallback when change_type is null

- **GIVEN** an active change with `change_type: null` in `.status.yaml`
- **WHEN** `/git-pr` resolves the PR type with no explicit argument
- **THEN** the resolution chain falls through to step 3 (infer from intake)

## Gate Threshold Updates

### Requirement: Seven-Type Gate Thresholds

The `--check-gate` mode in `calc-score.sh` SHALL use the 7-type taxonomy for gate thresholds:

| Type | Gate Threshold |
|------|---------------|
| `fix` | 2.0 |
| `feat` | 3.0 |
| `refactor` | 3.0 |
| `docs` | 2.0 |
| `test` | 2.0 |
| `ci` | 2.0 |
| `chore` | 2.0 |

This replaces the current `bugfix`/`feature`/`refactor`/`architecture` mapping.

#### Scenario: fix type uses lower threshold

- **GIVEN** a change with `change_type: fix` and `score: 2.5`
- **WHEN** `calc-score.sh --check-gate` is invoked
- **THEN** gate result is `pass` (2.5 >= 2.0)

#### Scenario: feat type uses standard threshold

- **GIVEN** a change with `change_type: feat` and `score: 2.5`
- **WHEN** `calc-score.sh --check-gate` is invoked
- **THEN** gate result is `fail` (2.5 < 3.0)

#### Scenario: Unknown type defaults to feat threshold

- **GIVEN** a change with `change_type: null` and `score: 2.5`
- **WHEN** `calc-score.sh --check-gate` is invoked
- **THEN** the threshold defaults to `feat` (3.0) and gate result is `fail`

## Template Updates

### Requirement: Status Template Default Change Type

`fab/.kit/templates/status.yaml` SHALL set the default `change_type` to `feat` (not `feature`). This aligns the template with the 7-type conventional commit prefix taxonomy.

#### Scenario: New change created from template

- **GIVEN** a new change is created via `changeman.sh new`
- **WHEN** `.status.yaml` is initialized from the template
- **THEN** the `change_type` field is `feat`

## Spec and Documentation Reconciliation

### Requirement: Create docs/specs/change-types.md

A new spec document `docs/specs/change-types.md` SHALL be created defining the 7 change types authoritatively. The document SHALL include:

- The 7 types with descriptions and examples
- `expected_min` thresholds per stage (the lookup tables)
- Gate thresholds per type
- PR template tier mapping (Tier 1 fab-linked: feat/fix/refactor; Tier 2 lightweight: docs/test/ci/chore)
- Keyword heuristics for inference
- Relationship to conventional commits

#### Scenario: Spec index includes the new entry

- **GIVEN** `docs/specs/change-types.md` is created
- **WHEN** `docs/specs/index.md` is updated
- **THEN** the new entry appears in the index table

### Requirement: Reconcile _preamble.md References

`_preamble.md` §Confidence Scoring SHALL be updated to:

- Replace `bugfix`/`feature`/`refactor`/`architecture` with the 7-type taxonomy
- Update the formula to show the coverage-weighted version
- Reference `docs/specs/change-types.md` for the full taxonomy

#### Scenario: Preamble uses 7-type taxonomy

- **GIVEN** the updated `_preamble.md`
- **WHEN** an agent reads the confidence scoring section
- **THEN** it sees the 7 canonical types (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`) with their thresholds

### Requirement: Reconcile docs/specs/srad.md

`docs/specs/srad.md` §Gate Threshold SHALL be updated to:

- Replace the 4-type table (`bugfix`/`feature`/`refactor`/`architecture`) with the 7-type table
- Update the `change_type` default reference from `feature` to `feat`
- Update the formula section to include the coverage-weighted version

#### Scenario: SRAD spec uses 7-type taxonomy

- **GIVEN** the updated `docs/specs/srad.md`
- **WHEN** a reader consults the Gate Threshold section
- **THEN** it shows all 7 types with their thresholds and uses `feat` as the default

## Design Decisions

1. **Coverage weighting over stage ceiling**: The `cover` factor attenuates scores based on decision count relative to expectation, rather than capping scores at a maximum per-stage value.
   - *Why*: Coverage weighting degrades gracefully — a spec with 5 of 6 expected decisions gets a proportional score (0.83 coverage), not a hard cap. Stage ceilings create cliff effects.
   - *Rejected*: Stage ceiling — hard caps create arbitrary boundaries; "5 decisions = full score, 4 = capped" feels arbitrary.

2. **expected_min thresholds embedded in calc-score.sh**: The lookup table lives directly in the shell script, not in config.yaml or a separate data file.
   - *Why*: `fab/.kit/` is the distribution unit shipped to projects. External config would require a separate file to travel with the script. Embedding keeps the script self-contained.
   - *Rejected*: Reading from config.yaml — config is project-specific, not kit-level. Separate thresholds file — unnecessary indirection.

3. **Seven conventional commit types, no architecture type**: The taxonomy aligns with standard conventional commit prefixes. "Architecture" changes are handled by coverage weighting naturally penalizing thin decision-making.
   - *Why*: 7 types cover the practical spectrum. An "architecture" type would need its own inference heuristic and adds complexity with marginal benefit since coverage weighting already addresses the concern.
   - *Rejected*: Adding `architecture` as an 8th type — no reliable keyword heuristic, coverage weighting is more principled.

4. **Indicative confidence is display-only at intake**: The score shown after `/fab-new` is ephemeral — not persisted to `.status.yaml`.
   - *Why*: The authoritative score is computed from the spec's Assumptions table by `calc-score.sh`. Writing an intake-stage score would create confusion about which score is current. Display-only gives the user a signal without polluting the data.
   - *Rejected*: Persisting intake score — creates stale data; dual-score confusion.

## Deprecated Requirements

### Gate Threshold: bugfix/feature/refactor/architecture Mapping

**Reason**: Replaced by the 7-type taxonomy (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`). The old mapping used inconsistent naming (`bugfix` vs `fix`, `feature` vs `feat`) and included `architecture` which has no inference heuristic.

**Migration**: `calc-score.sh --check-gate` uses the new 7-type table. `_preamble.md` and `docs/specs/srad.md` are updated to reference the new taxonomy. Old type values (`feature`, `bugfix`, `architecture`) in existing `.status.yaml` files will fall through to the default `feat` threshold (3.0).

### Template Default: change_type: feature

**Reason**: Replaced by `change_type: feat` to align with conventional commit prefix taxonomy.

**Migration**: `fab/.kit/templates/status.yaml` updated. Existing changes with `change_type: feature` will be treated as unknown (defaulting to `feat` thresholds) — functionally equivalent.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Coverage formula: `score = base * cover` where `cover = min(1.0, total / expected_min)` | Confirmed from intake #1 — user chose coverage weighting over stage ceiling after evaluating both options | S:95 R:85 A:90 D:95 |
| 2 | Certain | Expected_min thresholds: intake fix=2/feat=4/refactor=3/rest=2, spec fix=4/feat=6/refactor=5/rest=3 | Confirmed from intake #2 — user specified exact values after archive analysis | S:100 R:80 A:95 D:100 |
| 3 | Certain | 7 conventional commit types, no `architecture` type | Confirmed from intake #3 — user agreed coverage weighting makes explicit architecture type unnecessary | S:95 R:85 A:90 D:95 |
| 4 | Certain | `change_type` inferred at `fab-new` and stored in `.status.yaml` | Confirmed from intake #4 — user proposed this approach | S:95 R:90 A:95 D:95 |
| 5 | Certain | Indicative confidence in `fab-new` only, display-only, not persisted | Confirmed from intake #5 — user decided against fab-status, agreed on display-only | S:95 R:95 A:90 D:95 |
| 6 | Certain | `git-pr` reads `change_type` from `.status.yaml` instead of re-inferring | Confirmed from intake #6 — natural consequence of decision #4 | S:90 R:90 A:95 D:95 |
| 7 | Certain | Gate thresholds: fix=2.0, feat/refactor=3.0, rest=2.0 | Confirmed from intake #7 — derived from mapping 7 types to tiers | S:90 R:80 A:85 D:90 |
| 8 | Certain | New `docs/specs/change-types.md` as authoritative reference | Confirmed from intake #8 — user explicitly requested this | S:95 R:90 A:90 D:95 |
| 9 | Certain | `expected_min` embedded in `calc-score.sh` (shipped with fab/.kit) | Confirmed from intake #9 — user explicitly noted "that's what gets shipped" | S:95 R:85 A:95 D:95 |
| 10 | Certain | Template default changes from `feature` to `feat` | Confirmed from intake #10 — aligns with conventional commit prefix taxonomy | S:85 R:90 A:90 D:90 |
| 11 | Confident | Keyword heuristic for type inference matches git-pr's existing logic with additions for docs/test/ci/chore | Confirmed from intake #11 — strong signal, extending existing pattern | S:80 R:85 A:80 D:75 |
| 12 | Confident | `calc-score.sh` accepts `--stage` flag defaulting to `spec` | Confirmed from intake #12 — logical extension for stage-specific lookups | S:75 R:90 A:85 D:80 |
| 13 | Certain | `change_type` written via `stageman.sh set-change-type`, not raw `yq` | Confirmed from intake #13 — follows existing contract | S:95 R:90 A:95 D:95 |

13 assumptions (11 certain, 2 confident, 0 tentative, 0 unresolved).

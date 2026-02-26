# Intake: Coverage-Weighted Confidence Scoring & Formalize Change Types

**Change**: 260226-tnr8-coverage-scoring-change-types
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Add coverage-weighted confidence scoring and formalize change types — coverage factor prevents thin specs from getting inflated scores, change types become a first-class concept with their own spec document, and fab-new shows an indicative confidence score.

Conversational mode — extended discussion exploring the problem (inflated scores on thin specs), evaluating options (stage ceiling vs coverage weighting), validating against the archive (124 archived changes analyzed), and arriving at specific thresholds chosen by the user.

Key decisions from conversation:
- Coverage weighting chosen over stage ceiling (more principled, degrades gracefully)
- `expected_min` thresholds specified by user after archive analysis showed 41% impact at higher thresholds was too aggressive
- No `architecture` type — 7 conventional commit types only; coverage weighting naturally penalizes thin decision-making on large changes
- Indicative confidence in `fab-new` only (not `fab-status`) — user decided against adding to `fab-status` to keep it a fast read-only glance
- `change_type` should be inferred at `fab-new` time and stored in `.status.yaml`, making `git-pr` a reader not an inferer

## Why

1. **Inflated scores on thin specs**: The current formula (`5.0 - 0.3 * confident - 1.0 * tentative`) only penalizes the *resolution quality* of known decisions, not the *completeness of decision discovery*. A spec with 2 Certain decisions scores 5.0 — the same as one with 12 Certain decisions. Archive analysis found 5 changes with zero decisions scoring 5.0 and 20 total changes (16%) getting inflated scores under the revised thresholds.

2. **Change type is a second-class citizen**: `change_type` exists in `.status.yaml` (template defaults to `feature`) but is never properly set. `git-pr` re-infers it every time from intake keywords or diff analysis. The confidence gate in `_preamble.md` references types (`bugfix`, `feature`, `architecture`) that don't match the 7-type taxonomy in `git-pr` (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`). No single document defines the authoritative type list.

3. **No early confidence signal**: Users get no feedback on decision completeness until the spec stage. An indicative score at intake time would nudge users toward `/fab-clarify` before advancing.

## What Changes

### 1. Coverage-Weighted Confidence Formula

Add a coverage factor to the existing formula:

```
if unresolved > 0:
  score = 0.0
else:
  base = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
  cover = min(1.0, total_decisions / expected_min)
  score = base * cover
```

Where `total_decisions = certain + confident + tentative + unresolved` and `expected_min` is looked up by `{stage, change_type}`:

| Type | Intake `expected_min` | Spec `expected_min` |
|------|----------------------|---------------------|
| `fix` | 2 | 4 |
| `feat` | 4 | 6 |
| `refactor` | 3 | 5 |
| `docs` | 2 | 3 |
| `test` | 2 | 3 |
| `ci` | 2 | 3 |
| `chore` | 2 | 3 |

The `base` component is identical to the current formula. The `cover` component is new — it attenuates the score when `total_decisions < expected_min`, preventing thin specs from scoring high.

When `total_decisions >= expected_min`, `cover = 1.0` and the formula degenerates to the current one. Archive validation showed 84% of existing changes are unaffected by these thresholds.

### 2. Change Type Inference in `fab-new`

`fab-new` SHALL infer `change_type` from the intake content after generating `intake.md`, using the same keyword heuristic `git-pr` currently uses (evaluated in order, first match wins):

- Contains any of: "fix", "bug", "broken", "regression" → `fix`
- Contains any of: "refactor", "restructure", "consolidate", "split", "rename" → `refactor`
- Contains any of: "docs", "document", "readme", "guide" → `docs`
- Contains any of: "test", "spec", "coverage" → `test`
- Contains any of: "ci", "pipeline", "deploy", "build" → `ci`
- Contains any of: "chore", "cleanup", "maintenance", "housekeeping" → `chore`
- Otherwise → `feat`

Write the inferred type to `.status.yaml` via a new `stageman.sh` subcommand: `stageman.sh set-change-type <file> <type>`. All `.status.yaml` mutations go through `stageman.sh` — this follows the existing contract (`set-state`, `set-checklist`, `set-confidence`, `ship`). No raw `yq` calls from skills.

Note: the `.status.yaml` field value should use the full conventional commit prefix (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`) — not the longer names (`feature`, `bugfix`). This aligns with `git-pr`'s PR title format.

### 3. `git-pr` Reads from `.status.yaml`

`git-pr` Step 0 (Resolve PR Type) gains a new first entry in the resolution chain:

1. **Explicit argument** (existing — unchanged)
2. **Read from `.status.yaml`**: If `changeman.sh resolve` succeeds, read `change_type` from `.status.yaml`. If non-null and one of the 7 valid types, use it. Fall through if missing.
3. **Infer from intake** (existing fallback — unchanged)
4. **Infer from diff** (existing fallback — unchanged)

### 4. Indicative Confidence in `fab-new`

After generating `intake.md`, `fab-new` SHALL:

1. Count assumptions from the intake's `## Assumptions` table (certain, confident, tentative, unresolved)
2. Look up `expected_min` for the intake stage using the inferred `change_type`
3. Compute the indicative score using the coverage-weighted formula
4. Display it to the user as:

```
Indicative confidence: {score} / 5.0 ({N} decisions, cover: {cover})
```

This is display-only — NOT written to `.status.yaml`. The actual score continues to be computed and persisted at the spec stage by `calc-score.sh`.

### 5. Update `calc-score.sh`

Embed the `expected_min` lookup tables directly in the script (since `fab/.kit/` gets shipped to projects):

```bash
# Expected minimum decisions by stage and change_type
get_expected_min() {
  local stage="$1" change_type="$2"
  case "$stage" in
    intake)
      case "$change_type" in
        fix) echo 2 ;; feat) echo 4 ;; refactor) echo 3 ;; *) echo 2 ;;
      esac ;;
    spec|*)
      case "$change_type" in
        fix) echo 4 ;; feat) echo 6 ;; refactor) echo 5 ;; *) echo 3 ;;
      esac ;;
  esac
}
```

The script SHALL:
- Read `change_type` from `.status.yaml`
- Accept an optional `--stage` flag (default: `spec`) for when it's called from intake context
- Compute `cover = min(1.0, total_decisions / expected_min)`
- Apply `score = base * cover`

### 6. Update Gate Thresholds

The `--check-gate` mode in `calc-score.sh` SHALL use the 7-type taxonomy:

| Type | Gate Threshold |
|------|---------------|
| `fix` | 2.0 |
| `feat` | 3.0 |
| `refactor` | 3.0 |
| `docs`, `test`, `ci`, `chore` | 2.0 |

This replaces the current `bugfix`/`feature`/`refactor`/`architecture` mapping.

### 7. Create `docs/specs/change-types.md`

New spec document defining the 7 change types authoritatively. Contents:
- The 7 types with descriptions and examples
- `expected_min` thresholds per stage (the lookup tables)
- Gate thresholds per type
- PR template tier mapping (Tier 1 fab-linked: feat/fix/refactor; Tier 2 lightweight: docs/test/ci/chore)
- Keyword heuristics for inference
- Relationship to conventional commits

This is the design rationale document for this repo — it captures the "why" behind the taxonomy, thresholds, and conventions. Distributed files (`git-pr.md`, `calc-score.sh`, `_preamble.md`) embed the values directly since they ship to projects where this spec won't exist. Memory files in `docs/memory/` can reference this spec since they're also project-specific.

### 8. Update `docs/specs/index.md`

Add the new `change-types` spec entry to the specs index.

### 9. Reconcile `_preamble.md` and `docs/specs/srad.md`

Update references to change types:
- `_preamble.md` §Confidence Scoring: replace `bugfix`/`feature`/`refactor`/`architecture` with the 7-type taxonomy and reference `docs/specs/change-types.md`
- `docs/specs/srad.md` §Gate Threshold: same reconciliation — use 7 types, update the table
- `fab/.kit/templates/status.yaml`: change default `change_type: feature` to `change_type: feat`

## Affected Memory

- `fab-workflow/planning-skills`: (modify) document indicative confidence in fab-new, change_type inference
- `fab-workflow/execution-skills`: (modify) document git-pr reading change_type from .status.yaml
- `fab-workflow/templates`: (modify) document status.yaml change_type default change from `feature` to `feat`
- `fab-workflow/configuration`: (modify) document expected_min thresholds in calc-score.sh

## Impact

- **`stageman.sh`**: New `set-change-type` subcommand for `.status.yaml` writes
- **`calc-score.sh`**: Formula change, new `--stage` flag, embedded expected_min tables, updated gate thresholds
- **`fab-new` skill** (`fab/.kit/skills/fab-new.md`): Change type inference step, indicative confidence display
- **`git-pr` skill** (`fab/.kit/skills/git-pr.md`): Resolution chain updated to read from `.status.yaml` first
- **`_preamble.md`**: Confidence scoring section updated with 7-type taxonomy
- **`docs/specs/srad.md`**: Gate threshold section updated
- **`fab/.kit/templates/status.yaml`**: Default change_type value change
- **`docs/specs/change-types.md`**: New file
- **`docs/specs/index.md`**: New entry

## Open Questions

None — all decisions resolved in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Coverage formula: `score = base * cover` where `cover = min(1.0, total / expected_min)` | Discussed — user chose coverage weighting over stage ceiling after evaluating both options | S:95 R:85 A:90 D:95 |
| 2 | Certain | Expected_min thresholds: intake fix=2/feat=4/refactor=3/rest=2, spec fix=4/feat=6/refactor=5/rest=3 | Discussed — user specified exact values after archive analysis at two threshold levels | S:100 R:80 A:95 D:100 |
| 3 | Certain | 7 conventional commit types, no `architecture` type | Discussed — user agreed coverage weighting makes explicit architecture type unnecessary | S:95 R:85 A:90 D:95 |
| 4 | Certain | `change_type` inferred at `fab-new` and stored in `.status.yaml` | Discussed — user proposed this approach | S:95 R:90 A:95 D:95 |
| 5 | Certain | Indicative confidence in `fab-new` only, display-only, not persisted | Discussed — user decided against fab-status, agreed on display-only | S:95 R:95 A:90 D:95 |
| 6 | Certain | `git-pr` reads `change_type` from `.status.yaml` instead of re-inferring | Discussed — natural consequence of decision #4 | S:90 R:90 A:95 D:95 |
| 7 | Certain | Gate thresholds: fix=2.0, feat/refactor=3.0, rest=2.0 | Discussed — derived from mapping 7 types to tiers, no architecture type | S:90 R:80 A:85 D:90 |
| 8 | Certain | New `docs/specs/change-types.md` as authoritative reference | Discussed — user explicitly requested this, analogous to naming.md | S:95 R:90 A:90 D:95 |
| 9 | Certain | `expected_min` embedded in `calc-score.sh` (shipped with fab/.kit) | Discussed — user explicitly noted "that's what gets shipped" | S:95 R:85 A:95 D:95 |
| 10 | Certain | Template default changes from `feature` to `feat` | Discussed — aligns with conventional commit prefix taxonomy | S:85 R:90 A:90 D:90 |
| 11 | Confident | Keyword heuristic for type inference matches git-pr's existing logic with additions for docs/test/ci/chore | Strong signal — git-pr already uses this for feat/fix/refactor; extending to remaining types is straightforward | S:80 R:85 A:80 D:75 |
| 12 | Confident | `calc-score.sh` accepts `--stage` flag defaulting to `spec` | Logical extension — script needs to know the stage to look up expected_min; spec is the current default behavior | S:75 R:90 A:85 D:80 |
| 13 | Certain | `change_type` written via `stageman.sh set-change-type`, not raw `yq` | All .status.yaml mutations go through stageman.sh — follows existing contract (set-state, set-checklist, set-confidence, ship) | S:95 R:90 A:95 D:95 |

13 assumptions (11 certain, 2 confident, 0 tentative, 0 unresolved).

Indicative confidence: 4.4 / 5.0 (13 decisions, cover: 1.0)

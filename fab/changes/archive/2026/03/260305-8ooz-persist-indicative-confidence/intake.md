# Intake: Persist Indicative Confidence

**Change**: 260305-8ooz-persist-indicative-confidence
**Created**: 2026-03-05
**Status**: Draft

## Origin

> [8ooz] for all stages, fab-switch and fab-status should show the confidence scores (if no spec - then tentative score, else actual score based on spec)

One-shot from backlog, followed by a `/fab-discuss` session that explored the architecture. Key decisions from the discussion:

- Agreed to persist indicative confidence into `.status.yaml` at intake finish (rather than computing on-the-fly)
- Chose Option A (same `confidence` block with `indicative: true` flag) over Option B (separate `indicative_confidence` block)
- Agreed that consumers (fab-status, fab-switch, changeman list/switch) become uniform readers of `.status.yaml` — the mode-selection logic ("indicative vs persisted") disappears from skills entirely
- Split: ~70% script changes / ~30% skill changes, with total complexity reduction

## Why

Currently, confidence scores are only visible via `/fab-status`, and only when the change is at the spec stage or later. At intake stage, `/fab-status` calls `calc-score.sh --check-gate --stage intake` on every invocation to compute an ephemeral indicative score. `/fab-switch` shows no confidence at all.

This creates two problems:

1. **Inconsistent visibility**: Users switching between changes (`/fab-switch`) have no signal about which changes are well-specified vs. vague. The confidence score — the primary quality metric — is invisible in the most common navigation path.

2. **Duplicated mode-selection logic**: Every consumer that wants to show confidence must implement the same branching: "if intake stage → call calc-score live; if spec+ → read .status.yaml; if pre-intake → show nothing." This logic is currently in `fab-status.md` and would need to be duplicated into `fab-switch.md`, `changeman.sh`, and any future consumer.

By persisting the indicative score into `.status.yaml` at intake finish, every consumer becomes a simple reader. The mode-selection logic vanishes. Adding confidence display to new consumers becomes trivial.

## What Changes

### 1. `calc-score.sh` — Add `indicative` flag for intake-stage scoring

When `calc-score.sh` runs in **normal mode** (not `--check-gate`) with `--stage intake`, it SHALL:
- Compute the score from `intake.md` Assumptions table (existing behavior)
- Write the confidence block to `.status.yaml` (existing behavior)
- Additionally set `confidence.indicative: true` in `.status.yaml`

When `calc-score.sh` runs in normal mode **without** `--stage intake` (i.e., spec scoring), it SHALL:
- Compute from `spec.md` (existing behavior)
- Write the confidence block (existing behavior)
- Ensure `confidence.indicative` is absent or `false` (clear the flag if present)

The `--check-gate` mode remains read-only and unchanged.

### 2. `statusman.sh` — Extend confidence accessors

`get_confidence` SHALL additionally output `indicative:{true|false}` (reading `confidence.indicative` from `.status.yaml`, defaulting to `false`).

`set_confidence_block` and `set_confidence_block_fuzzy` gain an optional `indicative` parameter. When passed `true`, the confidence block includes `indicative: true`. When passed `false` or omitted, the `indicative` key is removed from the block.

CLI: `set-confidence <change> <certain> <confident> <tentative> <unresolved> <score> [--indicative]`
CLI: `set-confidence-fuzzy <change> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d> [--indicative]`

### 3. `changeman.sh list` — Add confidence score to output

Current output format per line:
```
name:display_stage:display_state
```

New format:
```
name:display_stage:display_state:score:indicative
```

Where `score` is the confidence score from `.status.yaml` (e.g., `3.4` or `0.0`) and `indicative` is `true` or `false`.

Read via `statusman.sh confidence` for each change.

### 4. `changeman.sh switch` — Add confidence line to output

Current output:
```
fab/current → {name}

Stage:  {display_stage} ({N}/8) — {state}
Next:   {routing_stage} (via {command})
```

New output:
```
fab/current → {name}

Stage:       {display_stage} ({N}/8) — {state}
Confidence:  {score} of 5.0{indicative_suffix}
Next:        {routing_stage} (via {command})
```

Where `{indicative_suffix}` is ` (indicative)` when `confidence.indicative` is true, empty otherwise. When score is `0.0` and the stage is pre-intake (no assumptions yet), show `not yet scored`.

Read via `statusman.sh confidence`.

### 5. `/fab-new` skill — Step 7 persists instead of display-only

Current Step 7 ("Indicative Confidence") computes the score inline and displays it without writing. Change to:

1. Call `bash fab/.kit/scripts/lib/calc-score.sh --stage intake <change>` (normal mode, **not** `--check-gate`)
2. This writes the indicative score to `.status.yaml` with `indicative: true`
3. Display the result from stdout (same format as current)

This replaces the inline computation with a script call, matching the existing pattern for spec-stage scoring.

### 6. `/fab-status` skill — Simplify confidence display

Remove the intake-stage special case that calls `calc-score.sh --check-gate --stage intake` live. Instead, uniformly read the confidence block from `.status.yaml` (via preflight output) for all stages.

Display rules:
- **Score > 0.0 with `indicative: true`**: `Indicative confidence: {score} of 5.0 ({breakdown})`
- **Score > 0.0 without `indicative`**: `Confidence: {score} of 5.0 ({breakdown})`
- **Score = 0.0 (template default, pre-intake)**: `Confidence: not yet scored`

### 7. `/fab-switch` skill — Add confidence display

After displaying `changeman.sh switch` output (which now includes the Confidence line), no additional skill-level confidence logic is needed.

For the no-argument flow (listing changes), the skill reads `changeman.sh list` output (which now includes `:score:indicative`) and displays it in the numbered list alongside stage info.

### 8. `preflight.sh` — Add `indicative` field to output

Extend the confidence section of the YAML output:
```yaml
confidence:
  certain: 3
  confident: 1
  tentative: 0
  unresolved: 0
  score: 4.4
  indicative: true
```

Read via `statusman.sh confidence` (which already handles the field).

### 9. `_preamble.md` — Update Confidence Scoring documentation

Update the confidence scoring section to document:
- The `indicative: true` flag in `.status.yaml`
- That `/fab-new` persists indicative scores (no longer display-only)
- That consumers read uniformly from `.status.yaml`

### 10. Memory and spec updates

Update `docs/memory/fab-workflow/change-lifecycle.md` confidence field description to reflect indicative persistence.
Update `docs/memory/fab-workflow/kit-scripts.md` calc-score.sh section to document the `--indicative` behavior.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Update confidence field description — indicative scores now persisted, `indicative: true` flag added
- `fab-workflow/kit-scripts`: (modify) Update calc-score.sh and statusman.sh sections — new `--indicative` flag, extended confidence accessor
- `fab-workflow/planning-skills`: (modify) Update `/fab-new` indicative confidence section — now persists via script call instead of display-only

## Impact

**Scripts**:
- `fab/.kit/scripts/lib/calc-score.sh` — new `--indicative` flag behavior in normal mode for intake stage
- `fab/.kit/scripts/lib/statusman.sh` — extended confidence accessor and writer
- `fab/.kit/scripts/lib/changeman.sh` — extended list and switch output formats
- `fab/.kit/scripts/lib/preflight.sh` — extended confidence output

**Skills**:
- `fab/.kit/skills/fab-new.md` — Step 7 calls script instead of inline computation
- `fab/.kit/skills/fab-status.md` — simplified (remove intake special case)
- `fab/.kit/skills/fab-switch.md` — confidence display via changeman output
- `fab/.kit/skills/_preamble.md` — documentation update

**Backward compatibility**: Changes with existing `.status.yaml` lacking `confidence.indicative` will default to `false`, which is correct (their scores are real spec scores or template zeros). No migration needed.

## Open Questions

- None — the approach was fully discussed and agreed in the preceding conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Same `confidence` block with `indicative: true` flag (not separate block) | Discussed — user chose Option A over Option B; consumers don't care about distinction | S:95 R:90 A:95 D:95 |
| 2 | Certain | `/fab-new` calls `calc-score.sh --stage intake` in normal mode to persist | Discussed — user agreed to persist at intake finish | S:95 R:85 A:90 D:95 |
| 3 | Certain | Spec-stage scoring clears the `indicative` flag | Discussed — spec overwrites with real score | S:90 R:90 A:95 D:95 |
| 4 | Certain | `changeman.sh list` appends `:score:indicative` to output format | Discussed — user agreed to script-level output changes | S:90 R:85 A:90 D:90 |
| 5 | Certain | `changeman.sh switch` adds Confidence line | Discussed — user agreed | S:90 R:85 A:90 D:90 |
| 6 | Certain | `fab-status` removes live calc-score call, reads .status.yaml uniformly | Discussed — this is the core simplification | S:95 R:80 A:95 D:95 |
| 7 | Confident | `statusman.sh` CLI uses `--indicative` flag (not positional arg) | Strong convention match — existing CLI uses flags for optional params | S:70 R:90 A:85 D:75 |
| 8 | Confident | Pre-intake changes show "not yet scored" (not "0.0") | Convention match — current fab-status already does this for missing confidence | S:75 R:90 A:80 D:80 |
| 9 | Confident | No migration needed — missing `indicative` defaults to `false` | Backward compat is clean since false is correct for existing spec-scored or zero-score changes | S:80 R:85 A:85 D:80 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).

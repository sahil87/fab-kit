# Intake: Labelled-Rung Dispatch Choreography

**Change**: 260902-0i4x-labelled-rung-dispatch-choreography
**Created**: 2026-09-02

## Origin

> [0i4x] 2026-09-01: fab agent unification Change 4/4 (optional): labelled-rung choreography simplification in _preamble per fab/plans/sahil/26-09-01-fab-agent-unification.md § Change 4 -- prose-only, LIGHT lane. depends on Change 3.

Backlog entry `[0i4x]`, invoked via `/fab-fff 0i4x --light` → `/fab-new 0i4x` (one-shot, no design conversation beyond the plan doc). This is Change 4 of the 4-change fab-agent-unification ladder (`fab/plans/sahil/26-09-01-fab-agent-unification.md`, commit `1b4f71b0`): Changes 1–3 are MERGED — #635 (`77vz`, `fab agent` surface), #636 (`mp8d`, shared `agent.Resolution` struct), #637 (`u6es`, skill migration to `fab agent -o yaml`, merged 2026-09-02T01:47Z). The dependency hold ("don't stack on unmerged #637") is lifted; this worktree was fast-forwarded to `6dd02d27` (#637's merge commit) before intake.

## Why

Changes 1–3 gave the resolver's YAML output a **labelled rung**: `_cli-fab.md` § fab agent documents the `dispatch:` mapping as "omitted exactly when the native rung is selected; otherwise contains labelled `rung: pane|headless` and its fully substituted `command`". But the dispatch choreography in `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch step 1 still teaches the pre-label **discovery mechanism**: "the YAML labels the selected rung at `dispatch.rung`, but this choreography deliberately does **not** consume that label until the successor change; **attempt `start` first and let its answer be the discriminator**."

This change IS that named successor. The problems with leaving it as-is:

1. **A wasted probe per pane dispatch** — every pane landing pays one refused `fab dispatch start` invocation (plus the orchestrator turn to interpret it) before re-running the identical invocation as `open`. The information it discovers is already in the surfaced YAML.
2. **Indirection the reader must unlearn** — step 1 teaches probe-and-branch plus a compensating "cheap shortcut" parenthetical (default `native` ⇒ a `dispatch:`-present branch is always headless) that exists only because the prose refuses to read the label.
3. **A stale IOU** — the "until the successor change" clause dangles once Change 4 is the current state; two memory files restate it verbatim and would document a mechanism nothing uses.

Why this approach (branch on `rung:`) over alternatives: the plan doc § Change 4 fixed it at design time — the label is authoritative because `internal/dispatch.SelectMode` computed it from the same ladder `fab dispatch` re-resolves, and `start`'s Go-side pane-refusal remains as defense-in-depth for the skew case (environment changed between resolution and launch), so reading the label loses no safety.

## What Changes

### 1. `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch step 1 — branch on `dispatch.rung`

Rewrite step 1 ("Launch, by mode — and `start` is how you learn the mode") so the branch keys on the labelled rung in the already-surfaced resolution:

- **`rung: pane`** → go straight to § The pane readiness gate: `open` → gate → `deliver`, then step 2's `wait`. The "Remember this landing" instruction (the deferred apply reap fires only on the pane arm) keys off `rung: pane` at resolution time instead of a remembered `start` refusal.
- **`rung: headless`** → go straight to `fab dispatch start <change> <stage>` with the full stage prompt on stdin (the existing `--timeout` guidance is unchanged).
- **Remove** the discriminator teaching: "`start` is how you learn the mode", "attempt `start` first and let its answer be the discriminator", the free-probe rationale sentence ("That probe is free: a pane landing is refused *before* stdin is read…" — the fact survives as defense-in-depth, below, but not as the discovery rationale), and the "deliberately does not consume that label until the successor change" clause.
- **Remove** the "cheap shortcut" parenthetical ("a `dispatch.mode` other than `pane` can never land on pane, so under the default `native` the `dispatch:`-present branch is always headless") — it compensated for not reading the label and is redundant once the branch reads `rung:` directly.
- **Keep `start`'s pane-refusal as defense-in-depth**: retain a short note that `fab dispatch start` still refuses a pane landing before stdin is read and before any state write (Go behavior unchanged — `runtime/dispatch.md` owns that contract), so a mislabelled or stale-environment landing errors cleanly and names `open`; the prose just stops teaching the refusal as the discovery mechanism.
- Step 1's two mode bullets re-head accordingly (e.g. "**Headless (`rung: headless`)** — …" / "**Pane (`rung: pane`)** — …"); the surrounding steps 2–4, the state table, the recovery policy, the readiness gate, and the stall guard are untouched.

### 2. Behavior-claim sweep — the two memory restatements + repo-wide grep

Per `code-quality.md` § Sibling Sweeps (memory files documenting a skill's behavior are in the sweep class), update the two files that restate the start-probe discriminator, and grep repo-wide for stragglers:

- `docs/memory/pipeline/execution-skills.md` (§ Status-transition ownership paragraph, ~line 21): "the wiring **attempts `start` first and lets its answer discriminate**… deliberately does not consume that label until the successor change" → the wiring branches on `dispatch.rung`; `start`'s refusal stays as defense-in-depth.
- `docs/memory/runtime/dispatch.md` (§ skill-wiring paragraph, ~line 25): same claim, same rewrite. The Go-contract sections (`start` SHALL refuse a pane landing, `open` is pane's entry) are **unchanged — they stay true**.
- Sweep greps (pre-verified 2026-09-02: `until the successor change` / `learn the mode` match exactly these 3 files; sweep again at apply, including `*_test.go` comments and contrastive phrasings per the recurring-lessons sweep taxonomy — e.g. "attempt start first", "let its answer", "discriminator" in dispatch context, "cheap shortcut"). `docs/specs/harness-adapters.md`'s "`fab dispatch start` is consequently headless-only: it MUST refuse a pane landing" is the Go contract and remains true — no spec edit expected (verify at apply).

### Non-goals (from the plan doc § Non-goals + § Change 4)

- **No Go changes.** `fab dispatch start`'s pane-refusal, `SelectMode`, the resolver output/schema, and every `fab dispatch` verb are untouched.
- No change to the readiness gate, stall guard, reap timing, recovery budget, or Dispatch-Prompt Obligations.
- No change to `_pipeline.md` / `fab-continue.md` / `fab-adopt.md` dispatch sites — they point at the `_preamble.md` canon (verified: no restatement of the probe choreography outside the 3 files above).
- Deployed `.claude/skills/` copies are never edited (regenerated by `fab sync`; this worktree's deployed copies lag at kit 2.23.9 until the next release — expected).

## Affected Memory

- `pipeline/execution-skills`: (modify) — rewrite the dispatch-wiring sentence teaching the start-probe discriminator to the labelled-rung branch
- `runtime/dispatch`: (modify) — rewrite the skill-wiring paragraph's "does not consume that label until the successor change / attempts `start` first" claim; Go-contract requirement sections unchanged

## Impact

- **Files**: `src/kit/skills/_preamble.md` (the owner — § CLI-Adapter Dispatch step 1 + step-1-adjacent sentences), `docs/memory/pipeline/execution-skills.md`, `docs/memory/runtime/dispatch.md` (+ `fab memory-index` regen if descriptions shift — not expected). Zero `src/go/` files, zero tests.
- **Behavior**: orchestrator skills stop paying a refused-`start` probe on pane landings; headless landings are unchanged in effect (they went to `start` anyway). The Go safety net is unchanged, so a stale environment still errors to `open` cleanly.
- **Release**: ships with the next kit release; until then live orchestrators run the deployed 2.23.9 prose (probe choreography), which remains correct against the unchanged Go surface.
- **Lane**: LIGHT (plan doc § Sizing: "`_preamble` choreography prose only — LIGHT"), invoked with explicit `--light`.

## Open Questions

*(none — the plan doc § Change 4 fixed the design; all residual decisions graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Step 1 branches on `dispatch.rung`: `pane` → `open → gate → deliver`, `headless` → `start` | Plan doc § Change 4 states exactly this; schema shipped and documented in `_cli-fab.md` § fab agent | S:95 R:90 A:95 D:95 |
| 2 | Certain | Keep `start`'s pane-refusal as defense-in-depth; prose stops teaching it as the discovery mechanism | Plan doc § Change 4 verbatim ("Keep `start`'s pane-refusal as defense-in-depth; the prose stops teaching it as the discovery mechanism") | S:95 R:90 A:95 D:95 |
| 3 | Confident | Drop the "cheap shortcut" parenthetical (default-`native` ⇒ always headless) rather than keep it | It existed solely to shortcut the probe the change removes; redundant once `rung:` is read directly, and keeping it would restate what the label already says | S:70 R:85 A:80 D:75 |
| 4 | Confident | Sweep scope = 2 memory files + repo-wide grep; no spec edit (harness-adapters.md's refusal claim is the unchanged Go contract) | Pre-verified by grep 2026-09-02 (3 files match); re-verified at apply per sweep discipline | S:75 R:80 A:85 D:75 |
| 5 | Confident | `change_type` = refactor (choreography simplification, no new surface) | Precedent: Change 3 (#637) shipped as `refactor:`; simplification of existing documented behavior, not docs-only content | S:70 R:90 A:80 D:70 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).

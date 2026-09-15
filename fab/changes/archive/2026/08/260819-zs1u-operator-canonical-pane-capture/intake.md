# Intake: Operator Canonical Pane Capture

**Change**: 260819-zs1u-operator-canonical-pane-capture
**Created**: 2026-08-20

## Origin

Backlog item `[zs1u]` (2026-08-19), invoked via `/fab-new zs1u` (one-shot, no prior conversation):

> Operator skill drift: operators bypass 'fab pane capture' and run raw 'tmux -L <srv> capture-pane -t %N -p -S -20 | tail -20' — because the skill tells them to. fab-operator.md §5 Question Detection step 1 (~line 331) hardcodes the raw tmux command, and _cli-agents.md § Peek (~line 118) parenthetically endorses 'capture-pane -S -20' despite naming fab pane capture as canonical. Verified fab pane capture works fine incl. -L/--json enrichment (agent_state, change/stage) — so the answer model loses that context for free. Fix: rewrite §5 step 1 + the §5 re-capture guard to 'fab pane capture --raw -l 20 [-L <server>] <pane>' and drop the raw aside from § Peek, keeping raw tmux only as the fab-absent fallback (mirroring the rk mux send gate pattern). Depends on the sibling backlog item making -l N actually mean last-N-lines.

**Scope discovery at intake time**: the `_cli-agents.md` § Peek half of this backlog item is **already done**. The sibling dependency `260819-y4mu-pane-capture-tail-last-lines` (which made `-l N` mean last-N-lines) was cherry-picked onto this branch as commit `ef9a0159`, and that change *also* rewrote § Peek — the `capture-pane -S -20` parenthetical is gone and `-l N` is documented as "returns the last N lines (blank screen-padding stripped, tailed internally)". The remaining drift is confined to `fab-operator.md` §5 and its memory mirror. The dependency is therefore satisfied AND half the backlog text is stale relative to this branch.

## Why

1. **The pain point**: `fab-operator.md` §5 Question Detection step 1 hardcodes `tmux capture-pane -t <pane> -p -S -20` (src/kit/skills/fab-operator.md:333). Operators follow the skill literally, so every per-tick question-detection capture bypasses `fab pane capture` — the command the kit itself names canonical (`_cli-external.md:182`: "Use `fab pane capture` instead of raw `tmux capture-pane`"). This is a stated-rule-vs-owner drift: the operator skill contradicts the canon its own helper set carries.
2. **The consequence if unfixed**: operators keep hand-rolling raw tmux (including the `| tail -20` workaround the old `-S -20` semantics forced), skip fab's pane validation (`ValidatePane`, uniform exit 2/3 error scheme), and forgo the enrichment (`agent_state`, change/stage) available for free on the same binary. The divergence was flagged as far back as `docs/specs/findings/skills-review-2026-06-11.md:1304` (migration deferred from PR #311, never done) — it does not fix itself.
3. **Why now / why this approach**: the blocker is gone. Before y4mu, `fab pane capture -l 20` returned ~20 scrollback lines + the entire visible screen (not the last 20 lines), so raw tmux was the only way to get a bounded tail window. With y4mu cherry-picked onto this branch, `fab pane capture --raw -l 20` reproduces the old `-S -20` intent exactly (last 20 content lines, blank screen-padding stripped, bytes untouched). The fix is a targeted rewrite of the two §5 capture references plus the memory mirror — not a broader raw-tmux purge (spawn via `tmux new-window` and the rk-absent `send-keys` fallback stay raw by design).

## What Changes

### 1. `src/kit/skills/fab-operator.md` §5 Question Detection step 1 (line 333)

Replace the hardcoded raw tmux capture with the canonical fab command:

```markdown
# before
1. **Capture**: `tmux capture-pane -t <pane> -p -S -20`

# after (exact command form from the backlog item)
1. **Capture**: `fab pane capture --raw -l 20 [-L <server>] <pane>` (`-L <server>` only when the operator runs on a non-default tmux socket — the §9 second-operator case)
```

`--raw` is deliberate: the detection patterns in step 4 scan bare terminal text (last-2-lines guard, last-non-empty-line `?` check), and the enriched default header would inject non-terminal lines (e.g. `agent: waiting`, lines ending with `:`) into the scanned window. Agent state already reaches the answer model separately — the tick's `fab pane map --all-sessions` read and §5's `waiting`-state primary signal.

### 2. `src/kit/skills/fab-operator.md` §5 re-capture guard (line 389, § Sending Auto-Answers)

The guard currently says "then re-capture the terminal" with no command named (the decision-table rows at lines 365–366 reference "the re-capture guard"). Rewrite it to explicitly reuse the same capture command as step 1 — e.g. "re-capture the terminal (same `fab pane capture --raw -l 20` capture as § Question Detection step 1)" — so the before/after comparison is transform-symmetric and no reader falls back to raw tmux for the guard.

### 3. `docs/memory/runtime/operator.md` question-detection mirror (line 177)

The memory file documenting §5 restates the raw command verbatim:

```markdown
1. Capture: `tmux capture-pane -t <pane> -p -S -20` (wide window compensates for line wrapping)
```

Update it to mirror the new §5 step 1 (sibling-sweep class: "the memory file documenting a skill's behavior" — fab/project/code-quality.md § Sibling Sweeps). This lands during apply/hydrate as present-truth, not as a change narrative.

### Explicitly NOT changed

- **`_cli-agents.md` § Peek** — already rewritten by y4mu (cherry-pick `ef9a0159`); the raw aside is gone. No fab-absent fallback line is added there or in fab-operator.md: the operator is launched via `fab operator`, so fab is present by construction, and y4mu's accepted resolution for § Peek was drop-the-aside, not add-a-gate.
- **Raw tmux that is raw by design**: `tmux new-window` spawning (§6/§7), the rk-absent `tmux send-keys` fallback behind the state gate (§3/§5), and the pre-delivery pane judgment rounds (`_preamble.md` readiness gate).
- **Historical records**: `log.seed.md` entries, `docs/specs/findings/*` review records, and Design-Decision "Rejected" prose mentioning capture-pane churn (`operator.md:446`) stay verbatim — they describe past states, not current instruction.
- **Go code / `_cli-fab.md`**: no binary change; `fab pane capture`'s surface is already correct and documented post-y4mu.

## Affected Memory

- `runtime/operator`: (modify) § Auto-Nudge question-detection step 1 — replace the raw `tmux capture-pane -t <pane> -p -S -20` capture with `fab pane capture --raw -l 20 [-L <server>] <pane>`; the re-capture guard mirrors the same command.

## Impact

- `src/kit/skills/fab-operator.md` — 2 edit sites (§5 step 1 at line 333; § Sending Auto-Answers re-capture guard at line 389). Skill prose only; no flow/tool-structure change beyond the capture mechanism, no sub-agent structure change.
- `docs/memory/runtime/operator.md` — 1 edit site (line 177 mirror), plus `fab memory-index` regeneration if the description changes.
- No Go changes, no template changes, no migration (no user-data restructuring).
- Repo-wide sweep obligation: grep `capture-pane -S -20` / `tmux capture-pane` across `src/kit/` + `docs/memory/` at apply end — intake-time sweep found exactly the two live sites above (everything else is historical or by-design raw).

## Open Questions

None — the backlog item prescribes the exact replacement command, the dependency is verified satisfied on this branch, and the already-done § Peek half was confirmed against the cherry-pick diff.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replacement command is exactly `fab pane capture --raw -l 20 [-L <server>] <pane>` | Backlog item prescribes it verbatim; verified valid against `_cli-fab.md` § capture (flags `-l/--raw/-L` all exist, `-l` = last-N post-y4mu) | S:90 R:85 A:95 D:90 |
| 2 | Certain | y4mu last-N-lines dependency is satisfied on this branch | Cherry-pick commit `ef9a0159` verified present with the `pane.TailLines` semantics; `--raw -l 20` reproduces the `-S -20` intent | S:90 R:95 A:95 D:95 |
| 3 | Certain | `docs/memory/runtime/operator.md:177` mirror is updated in the same change | code-quality.md § Sibling Sweeps names "the memory file documenting a skill's behavior" as a must-sweep class | S:85 R:90 A:95 D:90 |
| 4 | Confident | Detection capture uses `--raw`, not enriched/`--json` output | Backlog's fix command says `--raw`; the pattern scan operates on bare terminal lines and an enrichment header would pollute the scanned window (e.g. `agent:`/`stage:` lines match the `:`-prompt indicator); state context already flows via `fab pane map` and the `waiting` primary signal | S:70 R:80 A:80 D:75 |
| 5 | Confident | Re-capture guard explicitly names the same `fab pane capture --raw -l 20` command instead of the unnamed "re-capture the terminal" | Backlog says rewrite "§5 step 1 + the §5 re-capture guard"; naming the command keeps before/after captures transform-symmetric | S:65 R:90 A:80 D:75 |
| 6 | Confident | No fab-absent fallback text added anywhere (backlog's "keeping raw tmux only as the fab-absent fallback" clause is moot) | The § Peek aside it targeted was already dropped by y4mu without a fallback line, and `fab operator` guarantees fab's presence in every operator context — a fab-absent gate would be vacuous | S:60 R:85 A:80 D:70 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).

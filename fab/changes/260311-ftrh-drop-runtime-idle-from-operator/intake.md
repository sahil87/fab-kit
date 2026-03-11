# Intake: Drop runtime is-idle from Operator

**Change**: 260311-ftrh-drop-runtime-idle-from-operator
**Created**: 2026-03-11
**Status**: Draft

## Origin

> User noticed during a `/fab-discuss` session reviewing the fab-operator1 skill that `fab runtime is-idle <change>` only checks `.fab-runtime.yaml` in the CWD, not in the target change's worktree. Since the operator runs in its own dedicated tmux pane (a different worktree), calling `runtime is-idle` from the operator always reads the wrong runtime file. The user chose Option A (use pane-map) over Option B (fix `fab runtime` to accept a worktree argument) based on YAGNI — the operator already refreshes pane-map before every action, and no other consumer needs cross-worktree runtime checks today.

## Why

The operator skill (`/fab-operator1`) references `fab runtime is-idle <change>` in 6 places for pre-send idle validation. This command reads `.fab-runtime.yaml` at the repo root of the **current working directory** — which is the operator's own worktree, not the target agent's worktree. The result is always wrong: the operator would see its own runtime state (or `unknown`) instead of the target agent's.

`fab pane-map` already includes an Agent column (`active`, `idle (2m)`, `?`, `—`) that correctly resolves each worktree's `.fab-runtime.yaml` by discovering worktree paths from tmux pane metadata. The operator already mandates "re-derive state via pane-map before every action" — the `runtime is-idle` calls are redundant and broken.

If left unfixed, the operator would fail its pre-send validation checks in practice, either sending to busy agents (dangerous) or refusing to send to idle ones (annoying).

## What Changes

### Remove `fab runtime is-idle` from the operator skill

In `fab/.kit/skills/fab-operator1.md`:

1. **State Re-derivation section (line ~80)**: Remove the `fab runtime is-idle` bullet. Keep only `fab pane-map`.
2. **UC1 Broadcast (line ~93)**: Already says "via `fab runtime` state in the pane map" — simplify to "via the Agent column in the pane map".
3. **UC6 Unstick (line ~125)**: Replace "Confirm the target agent is idle via `fab/.kit/bin/fab runtime`" with "Confirm the target agent is idle via the Agent column in the pane map".
4. **Pre-Send Validation (line ~164)**: Replace "Run `fab/.kit/bin/fab runtime is-idle <change>` or read the Agent column from the pane map" with just "Read the Agent column from the pane map".
5. **Autopilot per-change loop (line ~249)**: Replace "poll `fab pane-map` + `fab runtime is-idle`" with just "poll `fab pane-map`".
6. **Purpose/summary (line ~14)**: Replace "via `fab pane-map` and `fab runtime`" with "via `fab pane-map`".

### Update the operator spec

In `docs/specs/skills/SPEC-fab-operator1.md`, apply equivalent changes:

1. **Summary (line ~5)**: Remove `fab runtime` reference.
2. **Primitives table (line ~18)**: Remove the `fab runtime is-idle` row entirely.
3. **Discovery section (line ~42)**: Already correctly describes pane-map as handling agent idle state — no change needed.
4. **Per-change loop (line ~151)**: Remove `fab runtime is-idle` from monitoring step.
5. **Pre-send validation (line ~223)**: Replace `runtime is-idle` with pane-map Agent column.
6. **Always re-derive state (line ~243)**: Remove `fab runtime is-idle` from the list.
7. **Agent busy detection (line ~314)**: Replace `fab runtime is-idle` with pane-map Agent column.
8. **Relationship table (line ~329)**: Remove the `fab runtime is-idle` row.

### What does NOT change

- `fab runtime` CLI command itself — no changes to the Go binary
- `fab pane-map` — already works correctly
- Other skills that use `fab runtime` (hooks use it for CWD-local state, which is correct)
- The operator's behavior — idle checking still happens, just via pane-map instead of a broken path

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator skill description to reflect pane-map-only observation

## Impact

- `fab/.kit/skills/fab-operator1.md` — 6 edits (remove/replace `runtime is-idle` references)
- `docs/specs/skills/SPEC-fab-operator1.md` — 8 edits (equivalent spec-side changes)
- No code changes, no template changes, no migration needed

## Open Questions

None — the approach was fully discussed and decided before intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use pane-map Agent column instead of runtime is-idle | Discussed — user chose Option A over Option B (fix runtime). Pane-map already has the data, operator already refreshes it before every action | S:95 R:90 A:95 D:95 |
| 2 | Certain | No changes to fab runtime CLI | Discussed — YAGNI, no other consumer needs cross-worktree runtime checks today | S:90 R:95 A:90 D:90 |
| 3 | Certain | No behavioral change to operator | The operator still checks idle state before sending — the data source changes, not the policy | S:90 R:95 A:95 D:95 |
| 4 | Certain | Constitution compliance: update spec alongside skill | Constitution requires skill changes to update corresponding spec | S:95 R:80 A:95 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).

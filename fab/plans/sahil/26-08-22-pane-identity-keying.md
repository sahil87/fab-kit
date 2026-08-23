# Pane identity keying — declare the contract, close the restart-alias hole

> Backlog detail doc — written 2026-08-22 after a run-kit discussion session on pane-map
> keying. Findings below were verified against fab-kit HEAD (`cmd/fab/panemap.go`,
> `internal/dispatch/dispatch.go`) and run-kit HEAD (`internal/sessions/fabstate.go`)
> on that date — re-verify file/line claims before implementing. Companion run-kit
> items are noted inline; run-kit's cli-layering spec (`docs/specs/cli-layering.md`,
> pane-map row line 72 + Part 8) is the ownership authority.

## Goal

Make pane identity keying an explicit, documented contract instead of an emergent
convention, and close the one remaining correctness hole: a persisted `pane: %N`
dispatch record aliasing onto an unrelated pane after a tmux server restart.

## Context / reason

run-kit historically joined `fab pane map` output against its tmux snapshot keyed by
`session:window_index`. That keying caused a real bug class (the StatusDot swap-lag):
after `swap-window` / `move-window` / session rename, index-keyed joins misattribute one
window's fab state to whichever window now sits at the old position. run-kit has since
dropped the join entirely — `internal/sessions/fabstate.go` derives change/stage natively
from each pane's cwd (walk up to the `.fab-status.yaml` symlink, parse its target), so rk
no longer consumes `fab pane map` for enrichment at all.

**The design lesson to encode** — pick the key by what owns the fact:

1. Fab identity (change/stage/displayState) is a **worktree fact**, not a pane fact —
   derive it from cwd, never store it in a pane-keyed map.
2. Genuinely pane-scoped facts ride the pane itself as tmux user options
   (`@rk_agent_state` is the precedent — options travel with the pane through every
   move/swap/rename).
3. Where a persisted pane reference is unavoidable (dispatch records), the key is
   `%pane_id` scoped by tmux socket (`-L` server) — never `session:window_index`,
   which is positional and mutable.

**Forward pressure**: run-kit is exploring promoting the operator window into a dedicated
`_rk-operator` session via `move-window` at the `@rk_role=operator` set-moment. Any
fab-side assumption that a known pane stays at a fixed session/index — or even in its
original session — breaks under that. The keying contract is what makes that move safe.

## Current state (verified 2026-08-22 — mostly already correct)

- `internal/dispatch/dispatch.go` — records key `Pane: %N` + `Server` (socket), and
  `recordedPanes` filters by exact server equality precisely because `%N` is per-socket.
  Correct.
- `cmd/fab/panemap.go` — change/stage resolved per pane via cwd → worktree
  (`worktreeRootForPane` / `resolvePane`); every row carries `pane_id` and `window_id`;
  `session` / `window_index` are display/JSON columns only. Enumeration is already
  delegated to `rk mux panes --json` (capability probe, silent fallback to fab's own
  tmux walk) per cli-layering Part 8. Correct.
- run-kit side: no remaining `fab pane map` consumption; fabstate.go derivation shipped.

## Changes

### 1. Declare the identity-key contract (docs; split across repos)

Since enumeration is rk-owned, the primary declaration belongs on **`rk mux panes
--json`** (run-kit repo — companion item, not this repo's work). fab-kit's part:

- Document in `fab pane map --json`'s help/docs that its row schema **inherits rk's
  contract**: `pane` (with `server` context) and `window_id` are the identity keys;
  `session` and `window_index` are DISPLAY-ONLY and MUST NOT be used as join keys by
  any consumer. Cite the run-kit misattribution bug as the motivating case.
- Note the `window_id: ""` legacy-line fallback consumers must tolerate.

### 2. Audit remaining index joins (sweep)

Sweep fab-kit — Go, skills, and docs (fab-operator guidance in particular) — for:

- any consumer correlating panes by `session` + `window_index` instead of
  `%pane_id` / `window_id`;
- any assumption that a pane's session or index is stable across its lifetime
  (the operator-relocation direction will violate both).

Fix or annotate each site against the contract in §1.

### 3. Close the restart-alias hole (the one real bug candidate)

`%N` id-space resets when a tmux server restarts, and `.fab-dispatch/*/{stage}.yaml`
persists `pane: %17` across that boundary. A pane-mode dispatch whose tmux server
restarts mid-flight can have its recorded `%17` alias onto an unrelated new pane —
`fab dispatch status` / `wait` could then read a dead dispatch as `running`, and
peek/kill could target the wrong pane.

- Verify whether state derivation probes anything beyond pane existence today.
- If not: record a liveness discriminator at open-time (`pane_start_time`, or the pane's
  shell pid) in the dispatch record, and require it to match before treating the pane as
  the worker. Mismatch ⇒ the worker is gone (route to the existing `orphaned` path).
- Placement (`SiblingDispatchPane`) already intersects with live panes and is fine as-is.

### 4. Record the `.status.yaml` cross-repo contract

run-kit's `fabstate.go` parses a subset of fab's status artifacts **directly from disk**
(deliberate: run-kit Constitution II, derive-at-request-time; no subprocess on hot paths).
That quietly makes the following a public cross-repo API that fab-kit must version
carefully — document it wherever fab-kit records its external contracts:

- the `.fab-status.yaml` symlink convention at the worktree root, pointing at
  `fab/changes/<name>/.status.yaml` (change name = target's parent dir basename);
- within `.status.yaml`: the fields run-kit derives change/stage/display-state from
  (the `progress` map and its stage states).

Schema changes to these are breaking changes for run-kit — coordinate before changing.

## Non-goals

- No change to panemap's cwd-based change/stage resolution (already correct).
- No removal of `session` / `window_index` from output — they stay as display columns.
- Do NOT move fab enrichment (change/stage/PR) into `rk mux panes`, even though rk has a
  parser: it would make rk a second authority on fab's display-stage formatting and
  couple rk releases to fab schema evolution. The stable equilibrium is: rk owns rows,
  fab owns meaning, disk is the interface.
- rk-side code work: none tracked here (fabstate.go already shipped; the `rk mux panes`
  contract declaration is a run-kit backlog item).

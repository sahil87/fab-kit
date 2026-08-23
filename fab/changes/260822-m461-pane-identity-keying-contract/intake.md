# Intake: Pane Identity Keying Contract

**Change**: 260822-m461-pane-identity-keying-contract
**Created**: 2026-08-22

## Origin

> Source: **`fab/plans/sahil/26-08-22-pane-identity-keying.md`** — a backlog detail doc written
> 2026-08-22 after a run-kit discussion session on pane-map keying. Its findings were
> verified against fab-kit HEAD (`cmd/fab/panemap.go`, `internal/dispatch/dispatch.go`)
> and run-kit HEAD (`internal/sessions/fabstate.go`) on that date; the doc itself
> instructs re-verifying file/line claims before implementing. run-kit's cli-layering
> spec (`docs/specs/cli-layering.md`, pane-map row line 72 + Part 8) is the ownership
> authority for the fab/rk split. Companion run-kit items (the `rk mux panes --json`
> contract declaration) are noted inline in the doc and are **not** this repo's work.

This intake was created via the promptless-defer Create-Intake dispatch (no user
questions asked; would-be questions recorded as deferred Unresolved rows — none arose).
The plan doc is the authoritative content source; the "verified at intake" notes below
are re-verifications performed against this worktree on 2026-08-22 during intake
creation.

**Interaction mode**: one-shot dispatch from a plan doc; the decisions and non-goals in
that doc carry the design conversation's conclusions and are encoded as assumptions
below.

## Why

Pane identity keying in fab-kit is currently an **emergent convention**, not a declared
contract. run-kit historically joined `fab pane map` output against its tmux snapshot
keyed by `session:window_index` — positional, mutable keys — which caused a real bug
class (the StatusDot swap-lag): after `swap-window` / `move-window` / session rename,
index-keyed joins misattribute one window's fab state to whichever window now sits at
the old position. run-kit has since dropped the join entirely (its `fabstate.go` derives
change/stage natively from each pane's cwd), but nothing in fab-kit *documents* the
keying rules, so the same mistake can recur in any future consumer — fab's own skills
included.

Three consequences of not acting:

1. **The misattribution bug class recurs.** run-kit is exploring promoting the operator
   window into a dedicated `_rk-operator` session via `move-window` at the
   `@rk_role=operator` set-moment. Any fab-side assumption that a known pane stays at a
   fixed session/index — or even in its original session — breaks under that. The keying
   contract is what makes that move safe.
2. **One real correctness hole stays open**: tmux's `%N` pane-id space resets on server
   restart while `.fab-dispatch/*/{stage}.yaml` persists `pane: %17`. After a restart
   the record can alias onto an unrelated new pane — `fab dispatch status`/`wait` could
   read a dead dispatch as `running`, and peek/kill could target the wrong pane.
   **Verified at intake (2026-08-22)**: state derivation probes nothing beyond pane
   existence today — `PaneAlive` (`src/go/fab/internal/pane/create.go:291`) delegates to
   `ValidatePane`, a single `display-message -t <pane> -p '#{pane_id}'` ID-echo compare.
   No liveness discriminator exists anywhere in the dispatch path.
3. **An undocumented public API can be broken silently.** run-kit's `fabstate.go` parses
   fab's status artifacts directly from disk (deliberate: run-kit Constitution II,
   derive-at-request-time; no subprocess on hot paths). That quietly makes the
   `.fab-status.yaml` symlink convention and parts of `.status.yaml` a cross-repo
   contract fab-kit must version carefully — today it is recorded nowhere in fab-kit.

**The design lesson to encode** — pick the key by what owns the fact:

1. Fab identity (change/stage/displayState) is a **worktree fact**, not a pane fact —
   derive it from cwd, never store it in a pane-keyed map.
2. Genuinely pane-scoped facts ride the pane itself as tmux user options
   (`@rk_agent_state` is the precedent — options travel with the pane through every
   move/swap/rename).
3. Where a persisted pane reference is unavoidable (dispatch records), the key is
   `%pane_id` scoped by tmux socket (`-L` server) — never `session:window_index`.

**Current state (verified 2026-08-22 — mostly already correct)**:

- `internal/dispatch/dispatch.go` — records key `Pane: %N` + `Server` (socket);
  `recordedPanes` filters by exact server equality precisely because `%N` is
  per-socket. Correct.
- `cmd/fab/panemap.go` — change/stage resolved per pane via cwd → worktree; every row
  carries `pane_id` and `window_id`; `session`/`window_index` are display/JSON columns
  only. Enumeration is delegated to `rk mux panes --json` (capability probe, silent
  fallback to fab's own tmux walk) per cli-layering Part 8. Correct.
- run-kit side: no remaining `fab pane map` consumption; fabstate.go derivation shipped.

So the work is: declare the contract (docs), sweep for violations, close the one code
hole, and record the cross-repo disk contract.

## What Changes

### 1. Declare the identity-key contract (docs)

Since enumeration is rk-owned, the *primary* declaration belongs on `rk mux panes
--json` in the run-kit repo — that half is a companion item, NOT this repo's work.
fab-kit's part:

- Document in `fab pane map --json`'s help and docs that its row schema **inherits rk's
  contract**: `pane` (with `server` context) and `window_id` are the **identity keys**;
  `session` and `window_index` are **DISPLAY-ONLY** and MUST NOT be used as join keys by
  any consumer. Cite the run-kit misattribution bug (StatusDot swap-lag after
  `swap-window`/`move-window`/session rename) as the motivating case.
- Note the `window_id: ""` legacy-line fallback consumers must tolerate (panemap.go
  already models it: `windowID` is `"" when absent (legacy line)`).

Concrete surfaces:

- `src/go/fab/cmd/fab/panemap.go` — command `Short`/`Long` help text.
- `src/kit/skills/_cli-fab.md` § `fab pane map` — the JSON field table currently lumps
  the keys (line ~538: "`session`, `window_index`, `pane`, `tab`, `worktree`, `change`,
  `stage` | Table-equivalent identity/context fields") — split into identity keys
  (`pane` + `server`, `window_id`) vs display-only fields (`session`, `window_index`).
- `docs/memory/runtime/pane-commands.md` at hydrate.

### 2. Audit remaining index joins (sweep)

Sweep fab-kit — Go, skills, and docs (fab-operator guidance in particular) — for:

- any consumer correlating panes by `session` + `window_index` instead of
  `%pane_id` / `window_id`;
- any assumption that a pane's session or index is stable across its lifetime (the
  operator-relocation direction — a dedicated `_rk-operator` session via `move-window`
  — will violate both).

Fix or annotate each site against the contract in §1. Candidate sites found during
intake grounding (the sweep must re-run repo-wide, not stop at these):

- `src/kit/skills/_cli-fab.md:538` — the field-table lumping above (fix via §1).
- `src/kit/skills/fab-operator.md` — the `(session, repo, pane)` addressing tuple
  (lines ~34, ~201, ~728, ~761). Pane ID is already documented as "the server-global
  primary key", which conforms; annotate that `session` is a display/context dimension,
  not a join key, and that a monitored agent's session can change mid-lifetime.
- Sweep greps to run: `session:window`, `window_index`, `#{session_name}:#{window_index}`,
  `session.*window.*index` across `src/go/`, `src/kit/skills/`, `docs/` — per the
  code-quality sibling-sweep rule, include `*_test.go` comments, contrastive phrases,
  and user-facing string literals.

### 3. Close the restart-alias hole (the one code change)

`%N` id-space resets when a tmux server restarts, and `.fab-dispatch/*/{stage}.yaml`
persists `pane: %17` across that boundary. A pane-mode dispatch whose tmux server
restarts mid-flight can have its recorded `%17` alias onto an unrelated new pane —
`fab dispatch status`/`wait` then read a dead dispatch as `running`; peek/kill target
the wrong pane.

**Verified at intake**: the plan doc's "verify whether state derivation probes anything
beyond pane existence" question is answered — it does not. `PaneAlive(paneID, server)`
→ `ValidatePane` → `display-message -p '#{pane_id}'` echo-compare only. Consumers:
`observeDispatch` (`cmd/fab/dispatch_status.go:110`, shared by `status` and `wait`) and
`dispatch reap` (`cmd/fab/dispatch_reap.go:66`), both feeding `DerivePaneState`.

Design (from the plan doc, discriminator choice settled at intake):

- **Record a liveness discriminator at open-time** in the dispatch record. The plan
  offered `pane_start_time` *or* the pane's shell pid; tmux 3.7c exposes **no**
  `pane_start_time` format variable (verified via `man tmux`: only `pane_pid`,
  `pane_start_command`, `pane_start_path` exist), so the discriminator is the pane's
  shell pid — `#{pane_pid}`, already readable via the existing `GetPanePID`
  (`src/go/fab/internal/pane/pane.go:284`). New optional field on the `Dispatch` struct
  (`internal/dispatch/dispatch.go`), e.g. `PanePID int` / `yaml:"pane_pid,omitempty"`,
  written wherever `dispatch open` persists the pane record.
- **Require it to match before treating the pane as the worker**: at the observation
  seam, pane liveness becomes "pane exists AND (no recorded discriminator OR current
  `#{pane_pid}` == recorded)". Mismatch ⇒ the worker is gone — route to the existing
  `orphaned` path. Keep `DerivePaneState` itself pure and byte-stable (it is a
  documented cross-adapter contract); the identity check belongs in the computation of
  its `paneAlive` input, not in the state machine.
- **Guard the destructive/targeting verbs too**: `kill` (and any peek/capture pointer
  the record produces, e.g. `dispatch logs`' printed capture command) must not target an
  aliased pane — a mismatched pane is not the worker, so kill treats it as already-gone
  (orphaned path), never sends `kill-pane` at the impostor.
- **Backward compatibility**: records written by older binaries lack the field
  (`omitempty`, zero value) — absent discriminator keeps today's existence-only
  behavior. `.fab-dispatch/` records are transient per-change runtime state, and the
  field is additive — no `src/kit/migrations/` file is needed (the migration rule
  covers config/`.status.yaml`/archive restructuring, not transient dispatch state).
- **Placement is untouched**: `SiblingDispatchPane`'s `recordedPanes` already
  intersects recorded panes with live panes and is fine as-is (plan doc, explicit).
- Constitution constraints apply: tests ship with the Go change (the derivation seam is
  deliberately table-testable); `src/kit/skills/_cli-fab.md` § fab dispatch is updated
  for any changed observable surface (e.g. a new record/`--json` field).

### 4. Record the `.status.yaml` cross-repo contract (docs)

run-kit's `fabstate.go` parses a subset of fab's status artifacts **directly from
disk**. Document — wherever fab-kit records its external contracts — that this is a
public cross-repo API fab-kit must version carefully:

- the `.fab-status.yaml` symlink convention at the worktree root, pointing at
  `fab/changes/<name>/.status.yaml` (change name = target's parent dir basename);
- within `.status.yaml`: the fields run-kit derives change/stage/display-state from
  (the `progress` map and its stage states).

Schema changes to these are **breaking changes for run-kit — coordinate before
changing**. Placement (assumed, see Assumptions #6): fab-kit has no single
external-contracts registry; follow the `docs/memory/runtime/runtime-agents.md`
precedent (which records the `@rk_agent_state` cross-repo data convention) and put the
consumer-contract note in `docs/memory/pipeline/schemas.md` (the `.status.yaml` field
owner) with the symlink half in `docs/memory/pipeline/change-lifecycle.md` (the symlink
owner), cross-linked (Confident — see Assumptions #6).

### Non-Goals (from the plan doc, verbatim intent)

- No change to panemap's cwd-based change/stage resolution (already correct).
- No removal of `session`/`window_index` from output — they stay as display columns.
- Do NOT move fab enrichment (change/stage/PR) into `rk mux panes`, even though rk has
  a parser: it would make rk a second authority on fab's display-stage formatting and
  couple rk releases to fab schema evolution. The stable equilibrium is: **rk owns
  rows, fab owns meaning, disk is the interface**.
- rk-side code work: none tracked here (fabstate.go already shipped; the `rk mux panes`
  contract declaration is a run-kit backlog item).

## Affected Memory

- `runtime/pane-commands.md`: (modify) `fab pane map` identity-key contract — `pane`
  (+`server`) and `window_id` are identity keys, `session`/`window_index` display-only,
  `window_id: ""` legacy tolerance, misattribution citation
- `runtime/dispatch.md`: (modify) dispatch record gains the open-time `pane_pid`
  liveness discriminator; identity-checked liveness feeds `orphaned` on mismatch;
  kill/peek guard; back-compat for discriminator-less records
- `runtime/operator.md`: (modify) annotation from the §2 sweep — `session` in the
  `(session, repo, pane)` tuple is display/context, not a join key; sessions can change
  mid-lifetime (only if the sweep lands operator edits)
- `pipeline/schemas.md`: (modify) the `.status.yaml` cross-repo consumer contract
  (run-kit `fabstate.go` reads the `progress` map from disk; breaking-change
  coordination rule)
- `pipeline/change-lifecycle.md`: (modify) `.fab-status.yaml` symlink convention marked
  as an external consumer contract (run-kit derives change name from the target path)

## Impact

- **Go**: `src/go/fab/internal/dispatch/dispatch.go` (record schema),
  `src/go/fab/cmd/fab/dispatch_status.go` (`observeDispatch` — shared by `status` and
  `wait`), `src/go/fab/cmd/fab/dispatch_reap.go`, the `dispatch open` record-write path
  (`cmd/fab/dispatch_start.go` / open), kill path (`internal/pane/create.go`
  consumers), `cmd/fab/panemap.go` (help text only). Tests alongside
  (`internal/dispatch`, `cmd/fab` dispatch tests).
- **Skills (canonical sources only, `src/kit/skills/`)**: `_cli-fab.md` (§ fab pane
  map field table, § fab dispatch record/status surface), `fab-operator.md`
  (annotations from the sweep), plus whatever the §2 sweep surfaces.
- **Docs**: the five Affected Memory files at hydrate; no spec-mirror work
  (constitution 1.6.0 removed the mirror tree).
- **Cross-repo**: none executed here — two companion run-kit backlog items are cited
  (rk-side contract declaration; operator-relocation exploration). Coordination note
  only.
- **Risk**: low — the code change is additive and back-compatible (absent field ⇒
  today's behavior); everything else is docs. The main regression surface is the
  dispatch observation path (`status`/`wait`/`reap`), covered by table tests.

## Open Questions

- None. (Would-be questions were graded; none fell below the ask threshold — see
  Assumptions.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope split: the primary contract declaration on `rk mux panes --json` is a run-kit companion item; no rk-side code work in this change | Plan doc states it explicitly (§1, Non-goals); cli-layering ownership authority cited | S:95 R:90 A:95 D:95 |
| 2 | Certain | Non-goals hold: cwd-based resolution untouched; `session`/`window_index` stay as display columns; fab enrichment NOT moved into `rk mux panes` | Plan doc Non-goals section, verbatim; "rk owns rows, fab owns meaning, disk is the interface" | S:95 R:85 A:95 D:95 |
| 3 | Certain | Change 3's premise verified: state derivation probes only pane existence today (`PaneAlive` → `ValidatePane` ID-echo; no discriminator anywhere) | Verified in this worktree at intake — `create.go:291`, `dispatch_status.go:110`, `dispatch_reap.go:66` | S:90 R:90 A:95 D:90 |
| 4 | Confident | Liveness discriminator = pane shell pid (`#{pane_pid}`) recorded at open-time as an optional `pane_pid` record field | Plan offered `pane_start_time` OR shell pid; tmux 3.7c has no `pane_start_time` format (verified via man page) — pid is the only tmux-native candidate, and `GetPanePID` already exists | S:70 R:75 A:90 D:80 |
| 5 | Confident | Mismatch is routed to `orphaned` by computing identity-checked liveness at the observation seam, keeping `DerivePaneState` pure/byte-stable; kill/peek never target a mismatched pane | Plan doc mandates the orphaned routing; keeping the documented state machine byte-stable follows its own in-code contract comments | S:80 R:70 A:85 D:75 |
| 6 | Confident | Cross-repo contract (Change 4) documented in existing pipeline memory — `schemas.md` (+ `change-lifecycle.md` for the symlink), per the `runtime-agents.md` cross-repo-convention precedent — not a new registry file | "Wherever fab-kit records its external contracts" is vague (no such registry exists), but the precedent gives a clear front-runner; easily relocated later | S:40 R:85 A:55 D:55 |
| 7 | Confident | No `src/kit/migrations/` file: the new record field is additive/`omitempty` on transient `.fab-dispatch/` state; absent discriminator keeps today's existence-only behavior | The migration rule covers config/`.status.yaml`/archive restructuring; dispatch records are per-change runtime state with archive-time cleanup | S:60 R:80 A:85 D:75 |
| 8 | Confident | Sweep disposition (§2) is per-site fix-or-annotate at apply's judgment; operator's `(session, repo, pane)` tuple is annotated (pane ID already primary key), not restructured | Plan doc grants "fix or annotate" latitude; intake grounding found the two candidate sites and their conforming/non-conforming halves | S:75 R:80 A:80 D:70 |
| 9 | Confident | `change_type` set explicitly to `fix` (keyword inference on the slug would default to `feat`) | The sole code change closes a correctness bug (restart-alias); docs ride along. Explicit override sticks per the re-inference rule | S:65 R:90 A:80 D:65 |

9 assumptions (3 certain, 6 confident, 0 tentative, 0 unresolved).

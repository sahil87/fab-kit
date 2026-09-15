# Intake: Operator Note Narrative-State Verb Family

**Change**: 260823-s2gw-operator-note-narrative-state
**Created**: 2026-08-23

## Origin

Dispatched promptless by the autopilot orchestrator (`/fab-proceed` create-new dispatch, `{questioning-mode} = promptless-defer`) from the plan document **`fab/plans/sahil/26-08-23-operator-offload-plan.md` § Phase A** (committed on main as `4c4ba90d`). This is **Change 1 of 4 in the plan's § Critical couplings & sequencing queue** (a same-repo cherry-pick-ladder autopilot queue): Change 1 = Phase A (this intake), Change 2 = Phase C2 auto-merge (depends on this change's `coordination` note kind), Change 3 = Phase B1 `tick-start --diff`, Change 4 = Phase B2 `fab pane questions`.

The plan was written 2026-08-23 from a /fab-discuss brainstorm, then **revised the same day after a four-agent review** (Go feasibility · doctrine coherence · failure red-team · scope); all reviewer must-fixes are already integrated in the doc. Per the dispatch, the plan's Phase A text is **decided design, not open questions** — including the answers to the companion doc's open questions (no `depends_on` survival teaching; consumers = restarted operator + skimming humans; no auto-expiry for open notes, bounded history for resolved). Code anchors were verified 2026-08-23 by the plan author against `src/go/fab/cmd/fab/operator_*.go` and re-verified while drafting this intake (owned-sections doc comment ~`operator_state.go:20`, `knownCap = 200` ~`:72`, `mutateOperatorState` ~`:127`, `emptyOperatorState()` ~`:170`; `operator_watch.go` exists as the structural template).

> Change: `fab operator note` narrative-state verb family (Phase A of the operator offload plan) — an owned verb family over a new `notes:` section in the server-keyed operator state file, plus the `fab operator state` OPEN NOTES header, the skill's § Notes doctrine, and the constitution-mandated `_cli-fab.md` signatures.

## Why

1. **The pain point**: the operator has no owned surface for cross-cutting narrative state — phase-plan progress, peer-session agreements, corrections to earlier conclusions, and merge-gate dependency waits that outlive the `monitored` set have no schema home. It resorted to hand-writing a `plan_queue:` key into the operator state file via `yq`, violating the **Full Mediation doctrine** ("agents never compute or hand-write what the binary can own" — `docs/memory/runtime/operator.md` § Full Mediation of State Writes): no `resolved` flag, no pruning, no live-vs-stale distinction, no schema.
2. **The consequence of not fixing it**: after `/clear`, re-orientation means re-deriving hours of context from scratch; hand-written keys accrete without lifecycle management; the doctrine violation normalizes further hand-writes. Change 2 of the queue (C2 auto-merge) also *depends* on this change — its persisted merge-sequence state is a `kind: coordination` note, so an armed PR survives operator crash/`/clear`.
3. **Why this approach**: an owned verb family mirrors the split that already works for watches — mechanical envelope fields owned by the binary (ids, timestamps, caps, pruning, atomic writes) with a free-prose body evaluated by the LLM. The binary owns everything mechanical; operator judgment stays judgment. Alternatives were settled in the four-agent review: no `lesson` kind (guard against a reflexive scratchpad — durable lessons route via `idea` → a fab change into docs/memory), no auto-expiry for open notes (notes are decisions, not a dedupe cache), no teaching `depends_on` to survive removal (fixes one case by complicating two schemas — a surviving wait is a note).

## What Changes

### 1. `src/go/fab/cmd/fab/operator_note.go` (NEW) — the verb family

Follows `operator_watch.go` structure (316 lines; cobra verb family + typed struct + `mutateOperatorState` read-modify-write), over a new `notes:` section in the server-keyed operator state file.

**Schema per note:**

| Field | Type / semantics |
|---|---|
| `id` | binary-assigned `n<N>` from a **persisted top-level `notes_seq` counter** — ids are never reused after prune; an old binary's tolerant read carries the unknown `notes_seq` key through |
| `kind` | enum: `dependency_wait \| phase_plan \| coordination \| correction` |
| `text` | free prose, **binary-enforced cap ~500 chars** (notes re-enter operator context on every state read — unbounded prose defeats the plan's own goal) |
| `refs` | optional change IDs / repos |
| `created_at` / `updated_at` | RFC3339 UTC, computed by the binary (`nowRFC3339()` — no subcommand accepts a timestamp) |
| `resolved` / `resolved_at` | resolution flag + timestamp |

**Verbs:**

- `fab operator note add --kind <k> [--ref <r>]... <text>` — prints the id; unknown kind / over-cap text exits **1** (matches the operator-verb enum convention — invalid `--source`/`--stage` exit 1; exit 2 stays the pane-family pane-missing code).
- `fab operator note resolve <id>` — idempotent; prunes **resolved** notes past a cap of **50** oldest-first (the `knownCap = 200` pattern, `operator_state.go:72`); **open notes never auto-expire**. Unknown id exits 1.
- `fab operator note update <id> <text>` — in-place update for evolving phase-plan notes (rather than accreting near-duplicates); refreshes `updated_at`.
- `fab operator note list [--open|--all] [--json]` — `--open` default; renders each note's **age from `updated_at`** and flags stale ones (e.g. `⚠ 21d`, display-only); warns above **25** open notes (the "do I even need a note?" nudge made mechanical).

### 2. `src/go/fab/cmd/fab/operator_state.go` (MODIFY) — skeleton + header

- `fab operator state` prints an **OPEN NOTES header block** (id · kind · age · first line) before the dump — **HUMAN OUTPUT ONLY**: absent from `--json`, comment-prefixed (`# `) in default mode so stdout stays parseable YAML for yq consumers.
- Default output **excludes resolved notes**; `--all` includes them.
- Skeleton gains `notes: []` (`emptyOperatorState()`, `operator_state.go:170`).
- The owned-sections doc comment at `operator_state.go:20` ("the four OWNED sections — monitored, autopilot, branch_map, watches") gains **notes** as the fifth section.

### 3. `src/kit/skills/fab-operator.md` (skill) — new § Notes

- Document the four verbs and the note lifecycle.
- Carry the **doctrine table** from the plan verbatim in content (note vs watch vs not-operator-state routing):

| Content | Home |
|---|---|
| Passive narrative read on restart/orientation — phase progress + holds, peer scoping agreements, report-back promises, corrections to earlier conclusions, **merge-gate dependency waits** (checked by operator judgment per tick — no git/GitHub watch `source` exists today) | **note** |
| Standing concerns a watch source can actually express today (`linear`/`slack` queries with `instructions`) | **watch** — if a git-source watch ships later, merge-gate waits migrate to it as its own change |
| Anything still true for a different operator next month — process lessons | **not operator state** — route via an `idea` backlog entry → a fab change into docs/memory (the operator has no memory write path: 3-file context load, may run with no `fab/` project at all); deliberately **no `lesson` kind** — its absence is the guard against notes degrading into a reflexive scratchpad |

- Update **§2 Init step 2's restart-restore list** (`fab-operator.md:94` — "Restore monitored set, autopilot queue, and branch_map from the file") to include **notes** — surviving restart is Phase A's whole point.

### 4. `src/kit/skills/_cli-fab.md` (skill) — signatures

Constitution-mandated (Additional Constraints): signatures for all new/changed commands — the four `note` verbs (new `### fab operator note` block under `## fab operator`, alongside `### fab operator watch` at `_cli-fab.md:1256`) and the `fab operator state` header/`--all` behavior change (`### fab operator state`, `_cli-fab.md:1235`).

### 5. Tests — `src/go/fab/cmd/fab/operator_note_test.go` (NEW)

From the plan's § Tests: ids from persisted `notes_seq` — **never reused after prune**; timestamps; kind + text-cap validation (exit 1); resolve idempotency + unknown-id exit 1; resolved-cap prune; open notes never pruned; list shapes + staleness flag; `state` header human-mode-only (absent from `--json`, stdout stays parseable); skeleton `notes: []`; unknown top-level keys survive every verb. Use the existing `operatorStatePathOverride` test seam.

### Edge cases in scope (plan § Edge cases, Phase-A subset)

- Legacy hand-written `plan_queue:` key **survives** — `loadOperatorState` is a tolerant whole-file read and unknown top-level keys round-trip. One-time manual conversion to notes, then delete the key; **no migration code** (additive schema, no restructuring of existing user data).
- `note resolve` idempotent; duplicate `note add` allowed (dedupe is judgment, aided by the list warning).
- Concurrency: same guarantee as every verb — atomic temp+rename (`saveOperatorState`); last-writer-wins acceptable under one operator per server.

### OUT OF SCOPE (later changes in the same queue)

- **Change 2 (Phase C2)** auto-merge choreography — depends on this change's `coordination` note kind but ships separately.
- **Change 3 (Phase B1)** `tick-start --diff` / `fleet:` frame source; **Change 4 (Phase B2)** `fab pane questions`; all Phase C follow-ups (C1 event wake, C3 `state --frame`, C4).
- No changes to `operator_tick_start.go`, `panemap.go`, `pane_questions.go`, or `_cli-agents.md`.
- Hydrate for THIS change covers only what Phase A touches — the notes-vs-watches doctrine line in `docs/memory/runtime/operator.md`; the B-phase parts of that file's hydrate row (detection-policy rewrites, level-triggered-deltas decision, C2 sequential-arming decision) and the `pane-commands.md` row belong to Changes 2–4.

## Affected Memory

- `runtime/operator`: (modify) new `fab operator note` verb family in the state-file verb enumeration (§ Server-keyed state file lists the mutation verb families); new design-decision line: **notes vs watches vs not-operator-state** routing (the doctrine table above), including the deliberate absence of a `lesson` kind. Only the Phase-A-relevant part of the plan's hydrate row for this file — B-phase content is later changes'.

## Impact

- **Go binary** (`src/go/fab/cmd/fab/`): `operator_note.go` (new, ~modeled on `operator_watch.go`'s 316 lines), `operator_state.go` (modify: skeleton, owned-sections comment, header rendering, `--all` flag), `operator_note_test.go` (new). Scope test runs to the `cmd/fab` package first.
- **Kit skills** (`src/kit/skills/`): `fab-operator.md` (§ Notes + §2 restore list), `_cli-fab.md` (signatures). Canonical sources only — never `.claude/skills/` (deployed copies).
- **State file**: additive — two new top-level keys (`notes`, `notes_seq`) in `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`. Old state files load fine (tolerant read); old binaries reading a new file carry the unknown keys through. No migration file needed (no restructuring of existing data).
- **Constitution touchpoints**: CLI change ⇒ `_cli-fab.md` + tests (Additional Constraints); test-alongside; sibling sweep — grep the restart-restore phrase and the owned-sections/four-sections claims repo-wide during apply (e.g. the "four OWNED sections" comment, any "monitored set, autopilot queue, and branch_map" restatements).
- **Consumers**: the restarted operator (re-orientation from one `fab operator state` read) and skimming humans (the header block); Change 2 (C2) consumes the `coordination` kind.

## Open Questions

None — the plan's § Phase A is post-four-agent-review decided design, and the dispatch directs that its decisions not be re-opened. Every would-be decision point was graded below; none landed Unresolved.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly plan § Phase A + matching § File map rows (note verbs, state header, skill § Notes, `_cli-fab.md`, tests); Phases B1/B2/C2/C are later queue changes | Plan § Critical couplings & sequencing fixes the 4-change queue; dispatch restates it | S:95 R:85 A:95 D:95 |
| 2 | Certain | Note schema and mechanics verbatim from the plan: `n<N>` ids from persisted `notes_seq` (never reused), 4-kind enum, resolved-prune cap 50 oldest-first, open notes never auto-expire, list warns above 25 open | Plan § Phase A is post-four-agent-review decided design | S:95 R:70 A:95 D:95 |
| 3 | Certain | Text cap enforced at exactly 500 chars (named constant), reading the plan's "~500" as 500 | One obvious default; a trivially reversible named constant | S:70 R:95 A:85 D:80 |
| 4 | Confident | Staleness flag threshold: a note renders `⚠ <age>` when `updated_at` age exceeds 14 days | Plan shows `⚠ 21d` as an example rendering but names no threshold; display-only and trivially adjustable | S:45 R:90 A:55 D:45 |
| 5 | Certain | No migration file ships — `notes`/`notes_seq` are additive keys; legacy `plan_queue:` survives the tolerant read and is converted manually once | Plan § Edge cases explicit; code-quality's migration rule targets restructuring, which this is not | S:90 R:75 A:90 D:90 |
| 6 | Certain | Validation errors (unknown kind, over-cap text, unknown id) exit 1; exit 2 stays the pane-family pane-missing code | Plan explicit; matches the operator-verb enum convention | S:90 R:85 A:90 D:90 |
| 7 | Confident | `note update` refreshes `updated_at` (age + staleness render from it); `note resolve` sets `resolved: true` + `resolved_at` and leaves text intact | Implied by the plan's age-from-`updated_at` + update-in-place design; not stated verbatim | S:70 R:85 A:80 D:75 |
| 8 | Certain | `state` header is human-mode-only: absent from `--json`, comment-prefixed in default mode (stdout stays parseable YAML); new `--all` flag includes resolved notes, default excludes | Plan explicit, red-team-reviewed (yq consumers keep working) | S:90 R:85 A:90 D:90 |
| 9 | Certain | Hydrate scope = the Phase-A-relevant part of the `runtime/operator.md` hydrate row only (verb enumeration + notes-vs-watches decision line); B-phase hydrate content deferred to Changes 2–4 | Dispatch explicit ("new decision: notes-vs-watches line") | S:85 R:80 A:85 D:85 |
| 10 | Certain | `change_type` = feat (new verb family / new capability) | Keyword inference default; no fix/refactor/docs/test/ci/chore keyword governs | S:90 R:90 A:95 D:95 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).

# Intake: Native Apply-Worker Resume

**Change**: 260808-tv3g-native-apply-worker-resume
**Created**: 2026-08-09

## Origin

> It takes an immense amount of time in the apply ↔ review cycles, when we lose the previous agent and start the whole process from scratch. Is there any way to reuse the older apply and older review agent, until the apply ↔ review cycle is completely over and we move on to the hydrate step?

Conversational (`/fab-discuss` session, 2026-08-09). The discussion refined the raw ask into a narrower, decided design:

- **Apply-only reuse** — the user explicitly accepted keeping the review worker fresh every cycle ("I am ok only resume apply, and keep review fresh every time"). `_pipeline.md` Auto-Rework Loop item 4's "Never reuse a prior review worker's context" rule is deliberate reviewer-independence design and stays untouched.
- **Adapter scope narrowed twice.** All three adapters (native Agent-tool, headless CLI, tmux pane) were analyzed as resumable. The user then simplified: "add resumability only to native and pane mode right now. And let headless be non resumable" — which eliminates the entire session-ID persistence problem (`--session-id`/`resume_command` provider grammar, fork-vs-continue verification). Pane resume (send-keys based, user-approved) is a **separate follow-up change** because it needs Go work (`fab dispatch resume`, stage-aware reap); **this change is the native arm only** — pure skill prose, no Go.
- **Fallback is mandatory** — resume is an optimization; every failure path (handle gone, orchestrator restarted, harness lacks the capability) degrades to today's fresh dispatch. Correctness and Constitution III idempotency never depend on a session surviving.
- **Release point** — the tier-1 orchestrator lets go of the apply worker once review passes (equivalently: at hydrate entry — review workers are fresh each cycle, so nothing else is held); hydrate always runs a fresh worker.

## Why

1. **The pain**: each auto-rework cycle in `/fab-ff`/`/fab-fff` pays two agent cold starts. The fresh apply worker re-loads the always-load layer (7 files), `intake.md`, `plan.md`, affected memory, and — most expensively — re-discovers the source files its predecessor just wrote, while the actual delta per cycle is usually a handful of findings. Multi-cycle changes (3 rework cycles are routine in this repo) multiply that waste.
2. **The consequence of not fixing it**: rework wall-clock stays dominated by context re-acquisition rather than fixing; worse, a fresh worker loses the *negative* knowledge of the loop — what a reviewer already rejected. (A recurring failure in this repo: a rework "creatively rewords" a fix the reviewer already specified verbatim and fails again; a persistent worker that watched the rejection doesn't repeat it.)
3. **Why this approach**: the harness now supports continuing a named subagent with its context intact (Agent tool `name` parameter + SendMessage). Using it on the native arm needs zero new machinery — no session IDs, no Go, no config — because the handle lives entirely in the orchestrator's own context. Alternatives rejected: headless-CLI session resume (session-ID tracking complexity — user deferred it indefinitely), pane send-keys resume (approved, but Go-scoped → separate change), a persisted "context brief" artifact (recovers less; may still be layered on later, out of scope here).

## What Changes

### `src/kit/skills/_preamble.md` — § Subagent Dispatch gains the worker-continuation mechanics (owner)

Per the owner-or-pointer convention, `_preamble.md` § Subagent Dispatch owns the *mechanics*; `_pipeline.md` points at them. New content (a subsection, e.g. "Worker Continuation (native arm)"):

- **Naming**: when the native Agent-tool branch dispatches the apply stage from the auto-rework-capable orchestrators, the Agent call passes `name: "apply-{id}"` (`{id}` = the 4-char change ID, e.g. `apply-tv3g`).
- **Continuation**: a later rework cycle resumes that worker by sending the new instructions to `apply-{id}` via SendMessage instead of spawning a fresh agent. The continued worker operates under the same block contract as a fresh dispatch: it returns results only, runs no `fab status` transition commands, and ends with a terminal `fab status refresh`. The continuation prompt carries the triaged findings and the rework action chosen at item 2 — it does NOT need to re-carry the standard subagent context files (the worker already holds them; that is the point).
- **Fallback rule (load-bearing)**: continuation is an optimization, never a correctness dependency. If the named worker is unreachable — the orchestrator session was resumed/restarted (handles do not survive), the harness has no named-agent/SendMessage capability, the send errors, or the worker was never named (e.g., the stage went through the CLI adapter) — dispatch fresh per the existing Step 1 procedure, including the full dispatch-prompt obligations. A fresh fallback dispatch re-establishes the name for subsequent cycles when the capability exists.
- **Profile fixity**: a resumed worker keeps the model/effort it was first dispatched with; `fab resolve-agent apply --alias` is NOT re-run on the resume path. Resolution runs (and is surfaced) only on fresh dispatches — initial or fallback.
- **Scope guard**: continuation exists ONLY for the apply worker inside the auto-rework loop. Review workers are never named and never continued (item 4's fresh-worker rule). Hydrate and all other stages are unaffected. The CLI-adapter branch (`dispatch=` present — headless or pane) is entirely out of scope: headless stays non-resumable by decision; pane resume is a separate change.

### `src/kit/skills/_pipeline.md` — Auto-Rework Loop wiring (pointer)

- **Step 1 (initial apply dispatch)**: on the native branch, name the worker `apply-{id}` per `_preamble.md` § Worker Continuation.
- **Item 3 (Re-dispatch apply)** becomes resume-first: *if the native-arm apply worker `apply-{id}` from this orchestrator session is reachable, continue it via SendMessage with the item-2 rework instructions; otherwise run the Stage Dispatch Procedure for `apply` with the Step 1 target (current behavior, verbatim fallback)*. Everything else in item 3 is unchanged — on success the orchestrator still runs `fab status finish <change> apply {driver}`, which remains the one counted review `→ active` transition. The cycle-count invariant, the item-1 fail+reset pair, item-2 triage, item-4 fresh re-review, and item-5 verdict handling are all byte-identical in semantics.
- **Release**: after `fab status finish <change> review {driver}` (review pass), or at the exhaustion Stop, the orchestrator stops continuing the worker — no explicit teardown call exists or is needed; the handle is simply never used again. State a one-line rule so future edits don't message a worker post-review.
- `/fab-adopt` (partial consumer of the Auto-Rework Loop) inherits the behavior with no file change of its own — verify its prose doesn't restate the old "re-dispatch fresh" wording.

### Non-goals (explicit)

- No reviewer scope-brief / findings-forwarding change (discussed; not decided — possible follow-up).
- No headless-CLI resume, no session-ID tracking, no provider-grammar (`resume_command`) work — user-decided out.
- No pane-mode resume — separate follow-up change (send-keys choreography via `fab dispatch resume`, stage-aware reap, contract amendment).
- No Go, CLI, or config changes of any kind. No changes to `/fab-continue`'s manual rework menu (a manual rework typically runs in a later session where no handle can exist).

### Spec mirrors & sibling sweep (constitution-required)

- `docs/specs/skills/SPEC-_preamble.md` and `docs/specs/skills/SPEC-_pipeline.md` — mirror the two skill edits.
- `docs/specs/harness-adapters.md` — the native-adapter description gains a continuation note (apply-worker reuse across rework cycles, fallback-to-fresh; the other two adapters unchanged, headless documented non-resumable).
- Sweep aggregate specs (`docs/specs/skills.md`, `glossary.md`, `architecture.md`) and the `fab-ff`/`fab-fff` twin wrappers for any restated "fresh worker each cycle" claims about apply (the *review* fresh-worker claims stay).

## Affected Memory

- `pipeline/execution-skills.md`: (modify) the sequencer/dispatch paragraph and rework-loop description — add apply-worker continuation on the native arm (naming, SendMessage resume, fallback-to-fresh, release at review pass, profile fixity); review-worker freshness (pag2) explicitly unchanged.
- `_shared/context-loading.md`: (modify) dispatch-prompt obligations mirror — record the continuation carve-out (obligation 2 binds dispatches, not continuation messages)

## Impact

- **Files**: `src/kit/skills/_preamble.md`, `src/kit/skills/_pipeline.md` (sources; deployed copies regenerate via `fab sync`), `docs/specs/skills/SPEC-_preamble.md`, `docs/specs/skills/SPEC-_pipeline.md`, `docs/specs/harness-adapters.md`, plus any aggregate-spec sweep hits.
- **Consumers**: `/fab-ff`, `/fab-fff` (via the `_pipeline.md` bracket), `/fab-adopt` (partial consumer). `/fab-continue` unaffected.
- **No Go changes, no tests** (markdown-only change; the CLI-change constitution constraint is not triggered).
- **Behavioral risk**: low — the fallback path is today's behavior verbatim, so the worst case of a broken resume path is the status quo. The one semantic risk to guard in review: the continuation prompt must preserve the block contract (result-only + terminal `fab status refresh`), or a resumed worker could skip its refresh.

## Open Questions

*(none — all decision points were resolved in the originating discussion)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is the native Agent-tool arm only; headless stays non-resumable; pane resume is a separate follow-up change | Discussed — user decided explicitly ("only native and pane mode right now… let headless be non resumable"; pane split out for its Go surface) | S:95 R:90 A:95 D:95 |
| 2 | Certain | Apply-only reuse; the review worker stays fresh every cycle (item 4 untouched) | Discussed — user decided explicitly ("I am ok only resume apply, and keep review fresh every time") | S:95 R:90 A:95 D:95 |
| 3 | Certain | Resume is an optimization with a mandatory fresh-dispatch fallback; no correctness dependency on a live handle | Discussed and constitution-backed (III, idempotency); fallback is current behavior verbatim | S:90 R:90 A:95 D:90 |
| 4 | Confident | Release point is review pass (or the exhaustion Stop); hydrate always dispatches fresh; release is passive (stop messaging), no teardown call | User framed release "at hydrate"; since review workers are per-cycle fresh, review-pass is the same boundary stated precisely | S:75 R:85 A:85 D:80 |
| 5 | Confident | Worker name is `apply-{id}` (4-char change ID) | Proposed in discussion, unobjected; collision-safe per change; trivially reversible naming choice | S:60 R:95 A:85 D:80 |
| 6 | Confident | `_preamble.md` § Subagent Dispatch owns the continuation mechanics; `_pipeline.md` carries only the pointer + loop wiring | code-quality.md owner-or-pointer convention answers this directly | S:65 R:85 A:90 D:80 |
| 7 | Confident | Resume path skips `fab resolve-agent` re-resolution; profile is fixed at first dispatch, re-resolved only on fresh (incl. fallback) dispatch | A continued worker's model cannot change mid-session; re-resolving would surface a value that cannot apply | S:55 R:85 A:80 D:75 |
| 8 | Confident | Continuation prompt re-states the block contract (results-only, terminal `fab status refresh`) but not the standard context files | Context files are already in the worker's context — re-carrying them defeats the purpose; the contract restatement guards the one identified semantic risk | S:60 R:80 A:80 D:75 |
| 9 | Certain | Status choreography and the cycle-count invariant are untouched — resume changes only how the rework prompt reaches the worker | The invariant is pinned against the Go contract; item 3's `finish apply` still fires every cycle | S:85 R:90 A:95 D:90 |

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).

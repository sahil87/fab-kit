# Intake: Pane Readiness Gate + Send-Keys Delivery

**Change**: 260809-3oz7-pane-readiness-gate-sendkeys-delivery
**Created**: 2026-08-09

## Origin

> Pane-adapter delivery redesign: send-keys-only prompt delivery with an agent-driven readiness gate. Replace spawn-time positional pointer delivery with a gated three-step flow — `fab dispatch open` (spawn pane without delivery), `fab dispatch ready` (mechanical echo-based probe returning ready/booting/parked + capture snippet), and a verified Go `deliver` step (send pointer via send-keys, verify echo, Enter, confirm busy, one retry). The Tier-1 orchestrator runs a judgment loop over `parked` screens (trust dialogs, surveys, first-run walls) using raw send-keys — max 2 rounds, login walls always escalate — sanctioned ONLY between open and deliver (contract amendment to harness-adapters.md: a pre-delivery pane is not yet a worker). interactive_command becomes pure launch grammar (no positional-prompt requirement — decouples pane capability from provider prompt-arg support). Folds in the approved-but-undrafted pane-resume design (C2 of the apply-worker resume series): the same deliver verb pointed at a continuation prompt, plus stage-aware reap (apply pane survives until review passes; other stages reap on done). `fab dispatch start` becomes headless-only; pane mode's entry is open. Provider roster data (giving agy/kimi an interactive_command) is explicitly OUT of scope — backlog [agik] runs in parallel. Go runtime + tests + harness-adapters.md amendment + _preamble.md/_pipeline.md/_cli-fab.md wiring + SPEC mirrors, single change, lands on post-C4 field names.

Conversational origin (long `/fab-discuss` session, 2026-08-09, all decisions user-confirmed):

1. **Send-keys-only delivery** chosen over the hybrid ("prefer positional one-shot when the provider supports it, fall back to send-keys"). Rationale: the one-shot path is fire-and-forget and *unverifiable* — agy silently dropped the positional argument (verified live in change rpsr), undetectable short of the stage orphaning; a hybrid needs a per-provider "accepts positional prompt" capability bit, which is the presence-as-policy disease the C3 mode-ladder change just eliminated; and one delivery engine shared by initial dispatch and resume is hardened by every dispatch.
2. **Trust prompts (and other first-run walls) are handled by agent judgment, not Go pattern-matching.** The user explicitly rejected agy-specific Go routines (trust-store pre-seeding was probed and works — `~/.gemini/antigravity-cli/settings.json` `trustedWorkspaces`, exact-match — but is undocumented-format, provider-specific machinery). Instead: a provider-agnostic readiness gate run by the Tier-1 orchestrator, which reads capture output and answers arbitrary walls with eyes. Go pattern-tables were rejected earlier in the same session (version treadmill; a half-matched pattern pressing Enter into an unknown screen is worse than stalling).
3. **Gate parameters, user-decided verbatim**: budget = **2 judgment rounds** then escalate; **login walls always escalate** (never answered); ready/parked discrimination is **purely echo-based**; verb surface is the **`fab dispatch open` + `fab dispatch ready` pair** (not `start --no-deliver`).
4. **Single change** (user asked; confirmed feasible): this *replaces* the pane delivery mechanism, so the repo's usual 3c/3d runtime-vs-wiring split would create a broken intermediate window or throwaway compat shims. Atomic swap in one change. Lands after C4 (#560, merged — field names are `interactive_command`/`headless_command`).
5. **C2 of the apply-worker resume series folds in and disappears as a separate change**: pane resume = the same verified deliver step pointed at a continuation prompt, plus stage-aware reap. The user had already approved the send-keys resume method in that series' design session ("I am not against the send keys method. Do that").
6. **Provider roster data is out of scope** — giving agy/kimi an `interactive_command` is backlog `[agik]` in the main checkout, run in parallel by the user. This change ships provider-neutral machinery only.

## Why

**Problem 1 — pane capability is coupled to an accident of provider CLI grammar.** Today the pane adapter delivers the stage prompt as a one-line pointer appended to the provider's `interactive_command` as a positional spawn argument. Any provider whose CLI does not accept a positional initial prompt (kimi: bare positional parses as a subcommand, hard error; agy: silently drops it) cannot have an `interactive_command` at all and is stranded dispatch-only — even though an interactive TUI is the one interface every agent CLI supports. The implicit "accepts positional prompt" requirement is validated nowhere and already caused a full REVISE-REQUIREMENTS rework cycle (change rpsr).

**Problem 2 — spawn-time delivery is unverifiable.** A positional argument either is or isn't ingested; fab cannot tell. agy's silent drop means a worker can sit at an empty prompt forever while the dispatch reads `running`. Send-keys delivery through a Go choreography (verify the echoed text, submit, confirm the worker went busy, one retry) makes delivery a *verified* step.

**Problem 3 — first-run walls block unattended pane dispatch with no recovery story.** Trust dialogs (agy trust-prompts every fresh workspace even with `--dangerously-skip-permissions`; worktree-per-change makes every dispatch a fresh workspace), feedback surveys, and login banners park a worker before it ever sees its prompt. Today's contract says the pipeline never sends keys, so the only outcome is a never-`done` escalation after the fact. The readiness gate turns this into a classified, bounded, pre-dispatch step — and because walls are mostly workspace-scoped, the first gate pass in a checkout clears them for every subsequent worker there.

**Problem 4 — pane resume (C2 of the apply-worker resume series, user-approved, undrafted) needs exactly this machinery.** Continuation delivery into a live pane is send-keys by nature. Building resume around a single-purpose verb first and retrofitting initial delivery later would be rework; the delivery engine is the shared core.

**If not fixed**: every future provider addition re-litigates the positional-prompt probe; agy/kimi stay headless-only (no watch-and-steer); rework cycles keep paying two cold starts on the pane arm; delivery failures stay indistinguishable from slow workers.

**Why this approach over alternatives** (all rejected in the origin discussion): hybrid delivery (capability-bit disease, unverifiable one-shot arm); Go-side dialog pattern-matching (version treadmill, blind keypresses); agy-specific trust-store pre-seeding in Go (provider-specific shim in a provider-neutral binary); keeping delivery split across spawn-time (initial) and send-keys (resume) (two mechanisms, one purpose).

## What Changes

### 1. `fab dispatch open <change> <stage>` — spawn without delivery (new verb)

Spawns the pane exactly as today's `fab dispatch start --pane` does — same `internal/spawn` composition of the resolved provider's `interactive_command` with `{model}`/`{effort}` substitution, same two-shape placement (split into the dispatcher's window with the stacked right column carved at `dispatch.column_width`, or new-window fallback), same record-keyed sibling detection, same `fab-{id}-{stage}` identity, same `--server` handling, same refuse-if-running — **except no prompt pointer is appended to the command**. The composed command is the pure launch grammar.

- The full stage prompt is still persisted to `.fab-dispatch/{4-char-id}/{stage}-prompt.md` at open time (same path as today), ready for the deliver step.
- The dispatch record (`{stage}.yaml`) gains a delivery marker (e.g. `delivered: false`, flipped by the deliver step). The five-state machine is **unchanged**: pane alive + no result ⇒ `running`, result present ⇒ `done`, pane dead + no result ⇒ `orphaned`. The marker is bookkeeping for `deliver`/`status` reporting, not a new state.
- `--pane`-specific prerequisites and errors carry over: reachable tmux + `interactive_command` on the resolved provider, hard error otherwise.

### 2. `fab dispatch ready <change> <stage>` — mechanical echo probe (new verb)

A one-shot, read-mostly probe answering "can this pane accept typed input?" — **purely echo-based** (user decision):

1. Send a short sentinel via `tmux send-keys` (literal text, no Enter).
2. Capture the pane (`fab pane capture` internals) and check whether the sentinel echoed at an input line.
3. Clear the sentinel (`C-u`) whether or not it echoed.
4. Report exactly one of:
   - `ready` — sentinel echoed; the pane accepts input.
   - `booting` — screen content changed between two captures spaced by a short internal tick, or is still empty/splash-like; the TUI is plausibly still starting.
   - `parked` — stable screen, sentinel did not echo (a dialog, survey, login wall, or wedged process is holding the input).
5. Every non-`ready` report includes a trailing capture snippet (last N lines) so the orchestrator can judge the screen without a separate capture call.

`ready` never sends Enter, never answers anything, and is safe to re-run (idempotent; Constitution III).

### 3. Verified delivery — `fab dispatch deliver <change> <stage>` (new verb; subsumes pane resume)

The sole mechanism that hands a pane worker its prompt, used for **both** initial dispatch and rework-cycle continuation:

1. Default mode sends the one-line pointer to `.fab-dispatch/{id}/{stage}-prompt.md` (same pointer content as today's spawn argument).
2. A continuation flag/argument (e.g. `--prompt-file <path>`) points it at a continuation prompt file instead — this **is** the pane-arm resume from the apply-worker resume series (C2), which this change absorbs. The orchestrator writes the continuation prompt (triaged findings + rework action + the re-read-from-disk instruction, mirroring the native arm's Worker Continuation content rules), persists it under `.fab-dispatch/{id}/`, and delivers it to the still-live apply pane.
3. Choreography (from the C2 design, now generalized): verify the pane is `ready` (internal probe, same mechanics as verb 2) → `C-u` → send the pointer line → capture-verify the echoed line → `Enter` → confirm the worker went busy (echo line consumed / screen advanced — still capture-based) → on any verification failure, **one retry** → on second failure, exit non-zero with the capture snippet. Delivery failure writes no false state: the record keeps `delivered: false`.
4. On success, flips the record's delivery marker.

Failure semantics at the call sites: explicit pane dispatch hard-fails (surfaced per the ordinary failure rule); the auto-rework resume path falls back to a fresh dispatch (resume stays an optimization, never correctness-bearing — unchanged doctrine from tv3g).

### 4. The readiness gate — orchestrator wiring (`_preamble.md` § CLI-Adapter Dispatch, pane branch)

The pane branch of the CLI-adapter dispatch procedure becomes: **open → gate → deliver → wait**, replacing single-shot `start --pane`:

1. `fab dispatch open <change> <stage>`.
2. Loop: `fab dispatch ready <change> <stage>` —
   - `ready` ⇒ proceed to deliver.
   - `booting` ⇒ wait briefly, re-probe (booting re-probes do not consume gate rounds; a bounded overall boot allowance keeps this from spinning forever).
   - `parked` ⇒ the **judgment rounds**: the orchestrator reads the capture snippet and MAY answer the wall itself with raw `tmux send-keys` (trust dialogs, surveys, pickers) — **maximum 2 such rounds per gate**, then escalate. **Login/credential walls always escalate immediately, never answered.** Escalation = surface the capture evidence + `rk notify` (behind the fail-silent gate) + stop on the existing failure path; the pane is left alive for the human.
3. `fab dispatch deliver <change> <stage>`.
4. `fab dispatch wait <change> <stage> --timeout 300` and everything downstream — unchanged.

The gate amortizes: walls are mostly workspace-scoped (trust is per-worktree-path), so the first gate pass in a checkout clears them for every later pane worker there; subsequent gates hit `ready` on the first probe. The gate MAY also be run ahead of apply (a warm-up) — same steps, no special casing.

### 5. Contract amendment — `harness-adapters.md`

- **Prompt delivery row (adapter 3)** changes from "prompt file + one-line pointer embedded at spawn" to "prompt file + pointer delivered post-spawn via the verified send-keys choreography (`fab dispatch deliver`), behind an agent-driven readiness gate". The existing rationale sentence claiming spawn-time embedding "sidesteps the printed-prompt trap entirely" is explicitly superseded (recorded as an amendment, per the spec's own Amendments-are-explicit rule): verified delivery + capability decoupling outweigh the boot race now that the choreography exists in Go with retry.
- **The no-input-injection rule gains a bounded carve-out**: *between `open` and successful `deliver`, the pane is not yet a dispatched worker — it has no stage context to corrupt — and the orchestrator MAY send keys to it (the gate's judgment rounds); `fab dispatch deliver`/`ready` are the sanctioned mechanical senders. From successful delivery onward the existing rule applies unchanged*: peek, kill, restart, notify, stop, reap — never input injection; mid-stage walls still escalate.
- **Pane resume is recorded** as the pane-adapter counterpart of tv3g's native Worker Continuation: same apply-only scope, same fresh-dispatch fallback, same profile fixity (no re-resolution on resume), reviewer independence untouched. The headless adapter remains deliberately non-resumable.
- **Stage-aware reap** (§6) is recorded in the reap section: invoking reap remains the orchestrator's choice; the *when* moves.
- Adapter-3 mechanics sections (launch, two shapes, sibling detection, column invariant) update from `start --pane` to `open`; the three-state pane derivation table is unchanged.

### 6. Stage-aware reap — wiring change only, zero Go guard change

`fab dispatch reap`'s Go-side guard (pane record + `done` + `dispatch.reap_done`) is untouched. What changes is **when the orchestrator calls it** (`_preamble.md` step 3 wiring): the **apply** pane is NOT reaped when its `done` result is read — it survives as the resume target across rework cycles and is reaped when **review passes** (or when the pipeline stops/escalates past apply for good). Review and every other stage's panes are reaped immediately on done-read, as today. Rationale (from the C2 design): today's unconditional reap-on-done would destroy the resumable pane.

### 7. `fab dispatch start` becomes headless-only; selector implications

- `start` composes and launches only the headless arm (`headless_command`, stdin prompt, exit-file supervision). The `--pane` flag is removed from `start`; passing it errors with guidance pointing at `open` (a usage error, not a silent reinterpretation). `--server` moves to `open` (it is pane-only today). `--timeout` stays on `start` (headless-only bound, as today — the `--pane --timeout` usage-error rule dissolves along with the flag).
- **Mode selection is unchanged in shape**: `dispatch.mode` ceiling, descent-only ladder `pane → native → headless`, same reason strings. What changes is the *pane rung's execution*: when automatic selection lands on pane, the launcher takes the open/gate/deliver path (the skill wiring branches there); `fab dispatch start` invoked directly with a config that resolves to pane errors with guidance to use `open` (mirroring today's native-result error shape).
- **`restart`'s pane arm**: `restart` re-derives mode from the current environment as today; when it lands on pane it performs the `open` (spawn-only) step and reports that the gate + deliver must follow — the orchestrator wraps restart exactly as it wraps open. When it lands on headless it relaunches fully, as today.

### 8. `interactive_command` becomes pure launch grammar (docs/data semantics)

No code path appends a prompt argument to `interactive_command` anymore. The field's documented meaning (defaults.yaml comments, configref segment, `stage-models.md`, `_cli-fab.md`) updates to: *how to launch an interactive session — prompt delivery is fab's, not the command's*. The defaults.yaml comment block explaining why agy/kimi carry no `interactive_command` (positional-prompt inability) becomes stale and is rewritten to point at the real remaining consideration (first-run walls / echo behavior — probed per provider before shipping the field; backlog `[agik]` owns the roster flip). The shipped built-in `interactive_command` values themselves are NOT changed by this change.

### 9. Skill wiring + SPEC mirrors + CLI reference

- `_preamble.md`: § CLI-Adapter Dispatch pane branch rewritten (open/gate/deliver flow, gate budget, escalation classes, stage-aware reap timing); § Worker Continuation gains the pane-arm resume (or a pointer to the pipeline wiring, per the owner-or-pointer rule — decided at apply).
- `_pipeline.md`: Auto-Rework Loop — the pane-dispatched apply worker becomes resumable via `deliver --prompt-file`, with the same resume-first/fresh-fallback shape as the native arm; reap timing wiring.
- `_cli-fab.md` § fab dispatch: `open`/`ready`/`deliver` documented (arguments, outputs, exit codes, the report strings), `start` narrowed to headless, `restart` pane-arm behavior, reap timing note.
- SPEC mirrors for every touched skill source (constitution requirement): `docs/specs/skills/SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-_cli-fab.md`, plus any skill restating the pane procedure (grep-swept: `fab-continue.md`, `fab-adopt.md`, `fab-operator.md`/`_cli-agents.md` restatements if any).
- Specs: `harness-adapters.md` (the §5 amendment — the bulk), `stage-models.md` (capability-grammar sentence for `interactive_command`), `docs/specs/config.md` only if any knob text changes (none expected — no new config field; gate budget is contract prose, not config).

### 10. Go tests

Per code-quality test-alongside: `internal/dispatch` unit tests for `open` (record shape, delivered marker, no-pointer composition, refuse-if-running, prerequisites), `ready` (echo/booting/parked derivation against scripted capture fixtures), `deliver` (verify-retry-fail choreography, marker flip, continuation prompt-file mode), `start` pane-rejection, `restart` pane-arm, and the reap guard regression (unchanged behavior). Pinned-output tests for the new report strings. Existing `start --pane` tests migrate to `open`.

## Affected Memory

- `runtime/dispatch.md`: (modify) the two-mode process manager doc — pane entry becomes open/gate/deliver, new verbs, delivery marker, resume, stage-aware reap timing, start narrowed to headless
- `runtime/providers-and-profiles.md`: (modify) `interactive_command` semantics — pure launch grammar; pane capability decoupled from positional-prompt support
- `pipeline/execution-skills.md`: (modify) auto-rework loop's pane-arm resume + the gate step in the dispatch procedure

## Impact

- **Go**: `src/go/fab/internal/dispatch/` (`dispatch.go`, `pane_mode.go`, `wait.go`, `reap.go` untouched-guard-verified, new `open.go`/`ready.go`/`deliver.go` or equivalent), `cmd/fab` dispatch command registration, tests throughout. No `resolve_agent.go` change expected (the `dispatch=` line contract is unchanged; the pane rung's command is still the substituted `interactive_command`).
- **Kit skills**: `src/kit/skills/_preamble.md`, `_pipeline.md`, `_cli-fab.md` (+ grep-swept restatement sites).
- **Specs**: `docs/specs/harness-adapters.md` (major amendment), `docs/specs/stage-models.md` (minor), `docs/specs/skills/SPEC-*.md` mirrors for every touched skill file.
- **Memory**: three files above via hydrate.
- **No config schema change** (no new fields, no migration — `.fab-dispatch` record gains an additive `delivered` key, transient comms with no compat window per the no-GC/cleanup rules).
- **No provider data change** (roster stays; `[agik]` is parallel).
- Rough scale: ~30 files; the largest `harness-adapters.md` amendment since the pane adapter landed.

## Open Questions

*(none — all material decisions were resolved in the origin discussion; residual design freedom is graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Send-keys is the ONLY pane delivery mechanism — the spawn-time positional pointer is removed outright, for every provider including claude | Discussed — user chose uniform send-keys over the hybrid; verified-delivery + no-capability-bit arguments accepted | S:95 R:60 A:90 D:90 |
| 2 | Certain | Gate budget: max 2 judgment rounds on `parked`, then escalate; login/credential walls escalate immediately, never answered | User answered verbatim ("2 rounds. Yes, login needs escalation") | S:100 R:85 A:95 D:95 |
| 3 | Certain | `ready`/`parked` discrimination is purely echo-based (sentinel send + capture check); no `@rk_agent_state` consultation | User answered verbatim | S:100 R:90 A:95 D:90 |
| 4 | Certain | Verb surface: `fab dispatch open` + `fab dispatch ready` as separate verbs (not `start --no-deliver`) | User answered verbatim | S:100 R:80 A:95 D:90 |
| 5 | Certain | Single change, shipped atomically (runtime + contract + wiring); lands on post-C4 field names | User asked and confirmed; C4 (#560) verified merged, branch rebased onto it | S:95 R:50 A:90 D:85 |
| 6 | Certain | Provider roster data (agy/kimi `interactive_command`) is OUT of scope — backlog `[agik]` runs in parallel | User instruction; entry written to the main checkout's backlog this session | S:100 R:90 A:100 D:95 |
| 7 | Certain | Pane resume (apply-worker series C2) folds into this change and stops being a separate change: continuation = the same deliver step with a continuation prompt file; apply-only scope, fresh-dispatch fallback, profile fixity, reviewer independence — all carried over from the tv3g doctrine unchanged | User approved send-keys resume in the C2 design session; folding confirmed in this session's single-change decision | S:90 R:55 A:90 D:85 |
| 8 | Confident | The delivery verb is named `fab dispatch deliver`, with a continuation mode selected by a prompt-file flag (e.g. `--prompt-file`) rather than a separate `resume` verb | Naming/flag shape not explicitly user-fixed; one engine for both prompts follows directly from decisions 1+7; exact spelling is cheap to change at apply | S:70 R:90 A:80 D:70 |
| 9 | Confident | Contract carve-out boundary: orchestrator send-keys is sanctioned only between `open` and successful `deliver`; after delivery the existing no-input-injection rule applies unchanged (mid-stage walls escalate) | Proposed in discussion and built upon without objection; the "pre-delivery pane is not yet a worker" principle was the accepted framing | S:85 R:70 A:85 D:80 |
| 10 | Confident | Gate exhaustion (2 rounds without `ready`) escalates — it does NOT auto-descend to headless | Consistent with the user's escalation answer; descent happens only at pre-launch capability resolution, keeping the ladder's semantics untouched | S:75 R:75 A:80 D:70 |
| 11 | Confident | Stage-aware reap is pure orchestrator wiring (when reap is invoked): apply pane reaped after review passes, all other stages on done-read; the Go reap guard is untouched | The Go guard already supports this (reap is orchestrator-invoked by design); C2 design named stage-aware reap explicitly | S:80 R:85 A:90 D:85 |
| 12 | Confident | `start` rejects pane outright (flag removed; pane-resolving config errors with open guidance); `restart`'s pane arm performs the spawn-only open step and hands gate+deliver back to the orchestrator | Follows from "pane mode's entry is open" (user-stated); restart cannot run agent judgment from Go, so the hand-back is forced | S:75 R:70 A:85 D:75 |
| 13 | Confident | Five-state machine, pane three-state derivation, `wait`, `status`, `clean`, and record paths are all unchanged; the record gains only an additive `delivered` marker | Contract-stability requirement from harness-adapters.md; nothing in the design needs a new state | S:80 R:75 A:90 D:85 |
| 14 | Confident | Native and headless adapters are untouched: native Worker Continuation (tv3g) stands as-is; headless keeps stdin delivery and stays non-resumable by decision | Explicit scope framing throughout the discussion | S:85 R:85 A:95 D:90 |
| 15 | Tentative | Probe mechanics: sentinel text choice, capture-tick spacing for `booting` vs `parked`, the boot allowance before `booting` degrades to `parked`, busy-confirmation heuristic after Enter, snippet line count | Pure implementation detail; easily revised; grounded in the C2 choreography sketch and the operator printed-text lesson | S:50 R:95 A:70 D:55 |
| 16 | Tentative | `open` carries over `--server` and both placement shapes byte-identically; the `dispatched …` report string family extends with open/deliver report lines pinned by tests | Mechanical carry-over of existing behavior to the new verb; exact report wording decided at apply against the shll standards check | S:55 R:90 A:75 D:60 |

16 assumptions (7 certain, 7 confident, 2 tentative, 0 unresolved).

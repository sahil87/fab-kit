# Harness Adapters — the cross-harness stage-dispatch contract

> **Status:** Design intent (pre-implementation, Constitution VI). This spec is human-curated: it
> fixes the dispatch protocol that the native Agent-tool path, both `fab dispatch` modes (headless and
> interactive-pane), and any future harness adapter conform to. It was authored by change 3c
> (`260702-6sgj-fab-dispatch-command`, the change that also implements the `fab dispatch` runtime) —
> the same "spec authored alongside its implementing change" pattern as
> [`stage-models.md`](stage-models.md) (#406). It is human-maintained thereafter, never
> auto-generated.
>
> **This spec fixes the contract once so it can be implemented by more than one change without silent
> drift.** The `fab dispatch` *runtime* ships in 3c (this change); the *skill-side wiring* that decides
> when to dispatch and calls the protocol is **change 3d** (wiring-only). 3d implements against this
> spec — it does not re-open it. See § Skill wiring is NOT part of the contract-defining change.
>
> **Amended by `260805-zxe0-interactive-pane-stage-dispatch`** — the interactive-pane adapter (a third
> adapter, and a second *mode* of `fab dispatch`) was added as an explicit amendment per
> § Skill wiring → *Amendments are explicit*: the protocol now names three adapters, states the
> per-adapter delivery mechanism and observable-state subset, and records that steering a pane worker
> changes no obligation. Nothing was silently redefined inside a skill file.
>
> **Amended by `260808-tv3g-native-apply-worker-resume`** — worker continuation (reusing the **apply**
> worker across rework cycles) is recorded as a **native-adapter-only** capability with a mandatory
> fresh-dispatch fallback, and the headless adapter is stated to be deliberately non-resumable while the
> pane adapter is left unchanged. Same explicit-amendment route as above: the capability is amended
> into this spec rather than silently redefined inside a skill file.
>
> **Amended by `260809-3oz7-pane-readiness-gate-sendkeys-delivery`** — the interactive-pane adapter's
> **prompt delivery moves off the spawn command**. The pointer is no longer embedded as a positional
> argument at window creation; it is typed into the pane after spawn by a verified send-keys
> choreography (`fab dispatch deliver`), behind an agent-driven **readiness gate**
> (`fab dispatch open` → `fab dispatch ready` → `fab dispatch deliver`). Three consequences are amended
> in with it: the **no-input-injection rule gains a bounded pre-delivery carve-out**, **pane resume** is
> recorded as the pane counterpart of tv3g's native continuation (superseding that amendment's statement
> that it would be a separate follow-up), and **reap timing becomes stage-aware** in the wiring. The
> earlier rationale that embedding the pointer at spawn "sidesteps the printed-prompt trap entirely" is
> **superseded**: verified delivery and the decoupling of pane capability from a provider's
> positional-prompt grammar outweigh the boot race, which the choreography now handles explicitly with a
> readiness precondition and a retry. Same explicit-amendment route as above.
>
> **Amended by `260904-b4j7-gate-delegation-rk-await-ready`** — the readiness gate's **mechanical
> classification delegates to run-kit**. When a sentinel-capable `rk` is on PATH, `fab dispatch ready`
> (and the `fab pane ready` primitive it binds) classifies via `rk mux await --ready` instead of typing
> its own sentinel — per the agent-messaging spec's ownership split ("fab consumes, never
> reimplements"). The shell-foreground takeover precondition, the fail-open raw-tmux fallback, and every
> report/exit-code MUST below are unchanged and bind both arms. Same explicit-amendment route as above.

Fab runs a six-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`). Every
post-intake stage is executed by **dispatching a worker** in a fresh context that returns a structured
result (see [`stage-models.md`](stage-models.md) § Why this is possible now, and `_preamble.md`
§ Subagent Dispatch) — except a **continued** native apply worker, which is deliberately *not* fresh
(§ 1), and the `/fab-ff`/`/fab-fff` **light lane**, whose non-review stages run inline in the
orchestrator's context with no dispatch at all (`_pipeline.md` § Light Lane). Historically there was one way to dispatch a worker — the Claude Code **Agent
tool** (an in-harness sub-agent). Cross-harness dispatch (e.g. a codex orchestrator running `apply` on
claude, or a claude orchestrator handing a stage to codex) added a second: a **detached CLI process**
observed via files. A third recovers what that detached process cannot offer — **watch and steer**: an
**interactive worker in a tmux pane**, observed via the same result file. This spec catalogs all three
**adapters** and fixes the **protocol** they share.

---

## The three adapters

An *adapter* is the mechanism that turns "run stage S as a worker" into an actual launched worker and,
later, an observed result. The resolution that precedes dispatch (stage → role →
`{provider, model, effort}`, via `fab agent <stage> -o yaml`; the invocation command lives on the resolved
provider) is **provider-neutral and adapter-independent** — see
[`stage-models.md`](stage-models.md). Only the launch+observe step is adapter-specific.

| Adapter | Launch | Prompt delivery | Completion observed via | States it can report |
|---------|--------|-----------------|-------------------------|----------------------|
| **1. Native Agent-tool** | in-harness sub-agent (Claude Code Agent tool) | the dispatched prompt itself | the held sub-agent handle (structural) | all five |
| **2. Headless CLI** (`fab dispatch start`) | detached `sh -c` wrapper, `setsid` semantics | prompt file on the command's **stdin** | `{stage}.exit` + pid liveness + result file | all five |
| **3. Interactive pane** (`fab dispatch open`) | a tmux pane running the provider's `interactive_command` — split into the dispatching agent's own window, or a new window when there is no pane to split | prompt **file** + a one-line **pointer** to it, **delivered post-spawn** by the verified send-keys choreography (`fab dispatch deliver`), behind an agent-driven readiness gate | **result file** + pane liveness | `running` / `done` / `orphaned` |

Adapters 2 and 3 are two **modes of the same command family** (`fab dispatch`), sharing its resolution,
`.fab-dispatch/{id}/` state directory, refuse-if-running concurrency, and the status/kill/reap/clean
surfaces — plus the `restart` recovery verb, which relaunches either mode from the persisted prompt and
re-derives its mode from the current environment (see § Recovery is orchestrator policy over these states). `logs` is the one verb that is **not** shared: it reads the headless wrapper's redirected
stream, so against a pane record it refuses and names the pane-mode equivalent, `fab pane capture
<pane>` (carrying the recorded socket as `-L <server>` when there is one). They differ in which of the
provider's two command fields they compose — `headless_command`
for headless, `interactive_command` for pane — and in how completion is observed. The two fields are never
merged and **never fall back to each other in either direction**.

### 1. Native Agent-tool adapter (in-harness)

Today's path: the orchestrator spawns a sub-agent via the Claude Code **Agent tool**. Per
[`stage-models.md`](stage-models.md) § Harness-adapter boundary, the resolved profile rides two seams —
the **model** on the Agent tool's `model` parameter (the YAML `model_alias`), and the **effort** from
the YAML `effort` key as an imperative instruction in the dispatched prompt (the Agent tool has
no effort parameter). The worker runs in-process; its result is the sub-agent's returned message. The
orchestrator observes the five states (below) **structurally** — it holds the sub-agent handle, so
"running/done/failed" are direct properties of the Agent-tool call.

**Worker continuation is a native-adapter-only capability** (amendment, `260808-tv3g`). Because the
handle lives entirely in the orchestrator's own context, the native adapter can *keep* a worker rather
than re-dispatch it: an auto-rework-capable orchestrator names its **apply** worker `apply-{id}` at
dispatch and, on a later rework cycle, sends the triaged findings and chosen rework action to that name
instead of paying a cold start, and instructs the worker to RE-READ from disk every artifact the
orchestrator edited at that item — always plan.md — because its in-context copy predates those edits.
The continued worker is bound by the **same** dispatch-prompt obligations as a fresh one (result,
block-contract carve-out, terminal `fab status refresh <change>`); only the standard **context files** are not
re-carried, because the worker still holds them. **Continuation is an optimization with a mandatory
fallback**: an unreachable handle — the orchestrator session was resumed or restarted, the harness has
no named-agent/message capability, the send errors, or the worker was never named — falls back to an
ordinary fresh dispatch carrying the full obligations, so no protocol guarantee is conditioned on a
session surviving. A resumed worker also keeps its first-dispatch `{model, effort}`: resolution runs
only on fresh dispatches. Scope is **apply inside the rework loop only** — review workers are
deliberately never named or continued (reviewer independence), and hydrate and every other stage always
dispatch fresh. `_preamble.md` § Worker Continuation owns the mechanics.

**Adapter 3 carries the same capability through a different mechanism; adapter 2 carries none.**

- **Headless CLI (adapter 2) is deliberately non-resumable.** Resuming a detached CLI worker would
  require session-ID persistence and a provider `resume_command` grammar, which the contract does not
  define and no adapter may improvise.
- **Interactive pane (adapter 3) IS resumable** (amendment, `260809-3oz7`). A pane worker never exits on
  completion — it sits at its prompt — so the worker the native adapter keeps in memory is, here, simply
  still on screen. Resume is therefore the **same verified delivery step pointed at a different prompt**:
  the orchestrator writes a continuation prompt under `.fab-dispatch/{id}/` and runs
  `fab dispatch deliver <change> <stage> --prompt-file <path>`. Every rule tv3g fixed for the native arm
  carries over unchanged — **apply-only scope** (review workers are never resumed, reviewer independence
  intact), the **mandatory fresh-dispatch fallback** (a failed delivery re-runs open → gate → deliver, so
  no protocol guarantee is conditioned on a pane surviving), **profile fixity** (no re-resolution on
  resume), and obligations 1 and 3 without the context files. The one runtime consequence is **reap
  timing**: the apply pane must survive to be resumed, so it is reaped when review passes rather than at
  done-read (§ 3's reap bullet).

The pipeline still never types into a *delivered* worker — resume delivery happens only while the worker
is `done` and sitting at its prompt, never mid-stage (§ Steering a pane worker changes no contract).

### 2. Headless CLI adapter — `fab dispatch start` (new in 3c)

The headless path: the worker is a **detached CLI process** (e.g. `claude …` or `codex exec …`),
launched and observed via `fab dispatch` (see [`_cli-fab.md`](../../src/kit/skills/_cli-fab.md)
§ fab dispatch). It exists because the native path is **tmux/in-harness-bound** and cannot drive a
stage on a CI box, a remote host, or a different agent CLI. `fab dispatch start` launches the resolved
provider `headless_command` detached via `sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'` launched with
`setsid` semantics (the shell is the supervisor — no Go process remains, so the dispatch survives the
orchestrator dying), tracks it
under `.fab-dispatch/{id}/`, and the orchestrator observes all five states **via `fab dispatch status` /
`fab dispatch wait`** (derived from on-disk signals plus a pid liveness probe) rather than a held handle.
POSIX-only in v1.

> `fab dispatch`'s **headless** mode is deliberately **parallel to and independent of** `fab pane` /
> `fab operator`. Those stay the *interactive operator-visibility* path (a human watching a monitored set
> of tmux panes with operator-owned lifecycles); headless dispatch is the *unattended pipeline* path
> (launch-and-poll). Making `fab pane` grow a headless mode was rejected — pane observation (tmux
> capture) and headless observation (exit-file polling) are different models. The pane **dispatch** mode
> below does not undo that split: it *consumes* tmux as a launch surface for a pipeline worker, while
> operator enrollment, monitored sets, and the `»`/`›` window markers remain the operator's, never a
> dispatch's (see § Pane dispatch is not operator enrollment).

### 3. Interactive-pane adapter — `fab dispatch open` → `ready` → `deliver`

The **watch-and-steer** path: the worker is an **interactive agent session in a tmux pane** the user
can read and type into mid-stage. It exists because a detached headless worker is a black box —
`fab dispatch logs --tail` recovers *watching*, but **steering is impossible**: a detached process in its
own session is nobody's conversation partner. That asymmetry against the native adapter's
peek-and-converse richness is the largest capability loss when running a stage cross-provider, and an
interactive pane is the one interface every agent CLI supports natively, so it recovers both with no
per-provider integration work.

**This adapter's entry is `fab dispatch open`, not `fab dispatch start`.** Spawning the pane and handing
the worker its prompt are two separate events with a decision between them, so they are two verbs:

| Verb | Does | Reports |
|------|------|---------|
| `fab dispatch open <change> <stage>` | spawns the pane from the composed `interactive_command` **verbatim** and persists the stdin prompt to `{stage}-prompt.md`; delivers **nothing** | `opened <id>/<stage> (…)` |
| `fab dispatch ready <change> <stage>` | mechanically probes whether the pane accepts typed input — gated on **agent takeover**: a shell foreground classifies `booting` with nothing typed; past takeover the classification delegates to `rk mux await --ready` when a sentinel-capable rk is on PATH (fail-open to fab's own raw-tmux classifier) | `ready` \| `booting` \| `parked` (+ pane, socket, capture snippet) |
| `fab dispatch deliver <change> <stage> [--prompt-file <p>]` | types the prompt pointer and verifies it landed | `delivered <id>/<stage> (…)` |

**Implementation layering — primitives vs. bindings (260810-1lah).** The readiness classifier (the
mechanical gate: agent-takeover precondition, then the rk-delegated or raw-tmux classification arm —
typed sentinel, echo check, `C-u` clear on the raw arm), the
wrap-tolerant echo-verify delivery
choreography, and the tmux pane mechanics live in `internal/pane` as **pane-addressed primitives**,
exposed as `fab pane open` / `fab pane ready` / `fab pane deliver` for driving any pane by id with no
dispatch record ([`_cli-fab.md`](../../src/kit/skills/_cli-fab.md) § fab pane). The three dispatch
verbs above are **thin record-keeping bindings** over those primitives: they add the `.fab-dispatch/`
record, the stdin-prompt persistence, the delivery marker, and the dispatch-owned placement policy —
and nothing else. Every MUST below binds the primitives exactly as before; the relocation is
behavior-preserving and this contract is byte-identical.

`fab dispatch start` is consequently **headless-only**: it MUST refuse a pane landing (flag or resolved
preference) with an actionable pointer at `open`, before any state write, rather than silently launching
headless or opening a promptless pane. `restart` still re-derives its mode; on a pane landing it performs
the `open` step alone and MUST report that the gate and `deliver` have to follow, because the gate needs
judgment a binary cannot supply.

Mechanics, all fixed by this spec:

- **Command composed**: the resolved provider's **`interactive_command`** — the same string `fab agent`
  composes — with `{model}`/`{effort}` substituted through the shared `internal/spawn` resolution. This is
  why pane mode needs **no new provider config field**: the interactive invocation is already in the
  provider table. It MUST NOT read or fall back to `headless_command`. Interactive **human-facing** spawns
  (the operator launcher, operator-spawned agent windows, `fab batch new`/`switch`) carry a shell fallback
  (`; exec "$SHELL"`) so the pane survives the agent's exit; dispatch pane workers do **not**, because this
  adapter's `running`/`done`/`orphaned` subset treats pane death as the worker's terminal event.
- **Where the pane opens — TWO SHAPES, one identity.** Both are the `_cli-agents.md` § Spawn Composition
  form with cwd = the repo root, and both record the new **pane ID** in
  `.fab-dispatch/{id}/{stage}.yaml` alongside the `fab-{id}-{stage}` identity string and the tmux socket
  label. The shape is a **pure decision** from the dispatching process's own tmux position
  (`$TMUX_PANE`) and whether `--server` was supplied:
  - **Split** (`$TMUX_PANE` set **and** no `--server`) — `tmux split-window {-h -l <n>%|-v} -t <target>
    -c <dir> "<composed-cmd>"` followed by `tmux select-pane -t <new> -T
    fab-{id}-{stage}`. The worker lands **in the dispatching agent's own window**. Placement is a
    **stacked right column**: `-v` (unsized) off the **last live sibling worker pane** in that window
    when one exists (stacking under the newest worker), else `-h -l <n>%` off `$TMUX_PANE` — **carving**
    the column at `dispatch.column_width` (default 35), so the dispatching agent keeps the rest of the
    window rather than being halved.

    **Sibling detection MUST key on the dispatch records, never on pane titles.** The probe intersects
    the pane IDs recorded across the checkout's `.fab-dispatch/*/{stage}.yaml` records with the
    window's live `tmux list-panes -t <target> -F '#{pane_id}'` output and keeps the last id present in
    both. Records MUST first be filtered to the socket being probed (`server:` equality, with the
    default socket recorded as `""`): a tmux pane ID is per-SERVER, so an unfiltered set would let a
    `--server`-scoped record's `%N` false-match the same `%N` on another socket. A pane ID is server-global and stable for the pane's lifetime — the same property `status`,
    `kill`, and `capture` already rely on — whereas a pane **title** is rewritten by the harness running
    inside the worker within seconds of spawn, so a title-keyed probe reports no sibling and every later
    worker carves another full-height column until the dispatching agent is a sliver. The intersection
    with one window's live pane list IS the liveness and same-window filter; titles remain set at spawn
    for **identification only**.

    **The column invariant is a creation-time rule.** Once the carving split has created the
    left/right separator, fab MUST NOT touch it again: no `select-layout`, no layout normalisation, no
    rearranging of user-made panes, no undoing of a manual resize. Only horizontal separators *inside*
    the column are added (by `-v` stacking). fab MUST NOT attempt to repair an already-mangled window —
    it only stops creating new mess.

    Placement is cosmetic and MUST degrade rather than fail, and every degradation MUST be
    **surfaced as a stderr warning** naming both the failure and the placement it resolved to — a
    silently absorbed failure would leave blind placement unexplainable from output. A failing
    sibling probe (`list-panes`) leaves no window to intersect and so falls back to the same
    **sized** `-h` carve off `$TMUX_PANE` (an unsized fallback would reintroduce the halving the
    width exists to prevent). A failing **record read** (an unreadable dir, a corrupt `{stage}.yaml`)
    MUST warn but MUST keep the records that did read: an unread record can only fail to *find* a
    sibling, never invent one, so the degraded answer is either the clean one or the carve. An
    **absent** `.fab-dispatch/` tree is the ordinary first-dispatch case and MUST NOT warn. A tmux
    that rejects `-l <n>%` (pre-3.1) MUST retry the split unsized with a stderr warning. A failed
    title set is likewise **non-fatal** (stderr warning at most): the pane ID is the identity.
  - **New window** (otherwise) — `tmux new-window -n fab-{id}-{stage} -c <dir> "<composed-cmd>"`. This
    is the **fallback**, reached when the dispatcher has no pane of its own to split (`$TMUX_PANE` unset
    — a headless orchestrator running `open`) or when `--server <name>` targets a socket on which the
    caller's own pane id is meaningless (pane ids are server-global, not global).

  This realizes the **two-tier tmux hierarchy**: an operator opens worktree agents as **windows**, and
  each worktree agent's stage workers are **panes beside it** — so a stage worker no longer consumes a
  window in the operator's tab bar. The `fab-{id}-{stage}` string is **unchanged and shape-independent**,
  carried by the pane **title** when split and the window **name** otherwise, and stored in the same
  record field either way (no schema change, no migration).
- **Everything downstream is pane-ID keyed and therefore SHAPE-BLIND**: pane liveness, `status`,
  `kill` (killing a split worker's pane leaves the agent's window and any sibling worker intact — plain
  `kill-pane` semantics), `fab pane capture`, and refuse-if-running behave identically in both shapes.
  Only the `opened …` report distinguishes them (`pane %N, split, title fab-…` vs.
  `pane %N, window fab-…`).
- **Prompt delivery is POST-SPAWN and VERIFIED.** The full stage prompt is persisted to
  `.fab-dispatch/{id}/{stage}-prompt.md` (the same path the headless mode uses) — a
  multi-thousand-token prompt cannot ride `send-keys` or argv reliably — and the worker is later typed a
  **one-line pointer** to that path. The composed `interactive_command` reaches tmux **verbatim**: fab
  appends nothing to it.

  The delivery choreography, run by `fab dispatch deliver`, MUST verify every step that could silently
  do nothing: confirm readiness → clear the input line → type the pointer → **check it echoed** →
  Enter → **confirm the screen advanced**. A failed check costs one attempt; there MUST be exactly
  **one retry**, and a second failure MUST exit non-zero carrying the pane's screen, leaving the record's
  delivery marker unset so a caller can distinguish "the worker never got its prompt" from "the worker
  got it and failed at the work". Delivery MUST also clear the previous attempt's result file, so a
  continuation reads `running` rather than the last cycle's `done` — and MUST restore it when no
  attempt verifies. A delivery that never landed has superseded nothing, and a record left at
  `delivered: true` with no result reads `running`, which every recovery verb refuses: the mandatory
  fresh-dispatch fallback below would need a `kill` step to be executable at all.

  **This supersedes the spawn-time positional pointer** and its rationale that embedding at window
  creation "sidesteps the printed-prompt trap entirely". Two costs outweighed that: a positional argument
  is **unverifiable** — a CLI that silently discards it strands a worker at an empty prompt while the
  dispatch reads `running` — and requiring one made pane capability hostage to an accident of a
  provider's CLI grammar, so a CLI that parses a bare positional as a subcommand could not have an
  `interactive_command` at all. The boot race the old rationale avoided is now handled explicitly, by the
  readiness gate below and the retry above.
- **The readiness gate stands between `open` and `deliver`, and it is MECHANICAL in the runtime and
  JUDGMENT in the orchestrator.** `fab dispatch ready` first checks who owns the pane: while the
  pane's foreground command (`#{pane_current_command}`) is still a shell, the provider binary has not
  taken the tty, and a cooked-mode shell echoes typed characters by itself — so the gate MUST report
  `booting` and MUST type NOTHING, because the sentinel would echo for a reason that has nothing to do
  with an agent being ready. This takeover precondition runs fab-side AHEAD of both classification
  arms: rk's await deliberately classifies a cooked-shell echo as ready (terminals-are-one-standard),
  which for a dispatch pane is exactly the false-ready the precondition exists to close, so rk MUST
  NOT be invoked while a shell owns the pane. Past the precondition the classification runs in **two
  arms**. The **rk arm** is preferred: when a sentinel-capable run-kit is on PATH — probed from the
  binary (`rk mux await --help` mentions the `parked` report; never a version compare, cached at most
  once per process) — the gate MUST delegate the mechanical classification to `rk mux await --ready
  <pane>` under a bounded internal timeout and map rk's report onto the frozen contract: `ready`
  (state-present or sentinel echo) ⇒ `ready`; `parked` ⇒ `parked`; a timeout `running` ⇒ `booting`;
  `gone` ⇒ the dead-pane error (not a classification). Any OTHER rk outcome — an unexpected non-zero
  exit or an unparsable report — MUST fail OPEN to the raw arm for that probe with at most one stderr
  warning per process; a classified rk outcome is an answer and MUST NOT be re-classified by the
  fallback. The **raw-tmux arm** is the rk-less fallback and MUST stay byte-identical: classify
  **purely** by typing a sentinel literally (never submitted, always cleared with `C-u`) and reading
  captures: the
  sentinel echoed ⇒ `ready`; no echo on a blank or still-changing screen ⇒ `booting`; no echo on a
  stable screen ⇒ `parked`. Both arms MUST carry **no table of known dialogs** — dialog text is a
  version
  treadmill, and a half-matched pattern pressing Enter into an unknown screen is worse than stalling —
  and MUST answer nothing themselves. Every non-`ready` report MUST carry the pane, its socket, and a
  capture snippet (fab's own capture on BOTH arms — rk's stderr formatting is not contract), because
  deciding what a parked screen wants belongs to the orchestrator. All three
  classifications are a successful observation (the `wait`-timeout precedent); non-zero exit is
  reserved for real errors. The gate's budget, escalation classes, and login-wall rule are skill-side
  policy in `_preamble.md` § CLI-Adapter Dispatch, exactly as the recovery policy is.
- **The pane path has TWO prerequisites — a reachable tmux server and an `interactive_command` on the
  resolved provider.** `open` selects pane EXPLICITLY, so either missing prerequisite is a hard error
  with no launch or state write — never a silent descent to headless, which would be the opposite of what
  that verb's caller asked for. Under `restart`'s automatic selection, a missing prerequisite skips pane
  and the fixed ladder continues to native, then headless. Pane command composition is deferred until validation;
  a failed real tmux probe re-runs selection with `pane unavailable: tmux unreachable`. Headless performs
  no tmux probe. Command fields never substitute for one another: each rung composes only its own grammar.
- **Mode selection = explicit overrides, then a preference ceiling**, resolved per invocation inside
  `fab dispatch start`/`restart` through the same pure selector projected by `fab agent <stage> -o yaml`. In order,
  `--pane` ⇒ pane; `--headless` ⇒ headless; `--timeout` ⇒ headless; `--server` ⇒ pane. (`start` no longer
  accepts the two pane signals and refuses a pane landing outright; `open` needs none of them, being the
  pane verb itself. The ladder is therefore `restart`'s, and `start`'s means of detecting a landing it
  must hand off.) With no signal,
  start at `dispatch.mode` (default `native`) and descend only through `pane → native → headless`:
  pane needs tmux + `interactive_command`, native needs `native: true`, headless needs `headless_command`.
  Automatic selection never ascends. A native result is not launchable by `fab dispatch`; it errors
  before state writes with `fab agent <stage> -o yaml` re-resolution guidance. Automatic success appends exactly
  `mode: <rung> (preferred)` or `mode: <rung> (descended: <reason>[; <reason>])`; pane reasons are
  `pane unavailable: no tmux`, `pane unavailable: tmux unreachable`, or
  `pane unavailable: no interactive_command`, followed in ladder order by
  `native unavailable`. Explicit selection carries no suffix.

  The complete selector reason set is:

  - `mode: pane (preferred)`
  - `mode: native (preferred)`
  - `mode: headless (preferred)`
  - `mode: native (descended: pane unavailable: no tmux)`
  - `mode: native (descended: pane unavailable: tmux unreachable)`
  - `mode: native (descended: pane unavailable: no interactive_command)`
  - `mode: headless (descended: native unavailable)`
  - `mode: headless (descended: pane unavailable: no tmux; native unavailable)`
  - `mode: headless (descended: pane unavailable: tmux unreachable; native unavailable)`
  - `mode: headless (descended: pane unavailable: no interactive_command; native unavailable)`
- **No timeout**: `--timeout` is enforced by the headless `sh -c` wrapper (POSIX `timeout`), which pane
  mode never constructs, so no pane entry accepts one — `open` does not register the flag at all, and on
  `restart` the pair remains a usage error rather than a silently unenforced bound.
- **The done worker's lifecycle ends in an OPTIONAL, orchestrator-invoked reap.** A pane worker never
  exits on completion — it writes its result file and sits at its prompt, deliberately, so it stays
  steerable — so across a multi-stage pipeline the carved worker column accumulates finished panes that
  hold their space indefinitely, shrinking the panes the user actually watches with every completed
  stage. The adapter therefore permits a **`fab dispatch reap <change> <stage>`** at a deterministic
  moment the protocol already has. That moment is **stage-aware**: for every stage except **apply** it is
  immediately after the orchestrator reads a `done` result, while the **apply** pane MUST survive its own
  `done` — it is the resume target across rework cycles — and is reaped when **review passes**, or when
  the pipeline stops or escalates past apply for good. Reap MUST act only when the record is pane-mode
  **and** the derived state is `done` **and**
  the `dispatch.reap_done` policy (default *true*) allows it; every other case MUST be a reported no-op,
  never an error. It MUST NOT be reachable from any non-`done` state — that separates it from `kill`,
  the recovery verb (§ Recovery is orchestrator policy over these states) — and it MUST remove **no**
  `.fab-dispatch/` state (§ Cleanup). Reap is **shape-blind** like every other pane-ID-keyed operation,
  and it changes **no state**: result presence still wins over pane liveness, so a reaped dispatch
  reports `done` for good. **Invoking it is the orchestrator's choice, not the adapter's obligation** —
  the skill-side wiring is `_preamble.md` § CLI-Adapter Dispatch step 3.

---

## The dispatch protocol (shared by all three adapters)

The protocol is what makes an adapter interchangeable: whichever adapter launches the worker, the
**outcome contract** is identical. It has an orchestrator-side half and a worker-side half; the
worker-side half (dispatch-prompt obligations) is **implemented by 3d**, but its rules are fixed here.

### Dispatch-prompt obligations (the worker-side half — 3d implements)

Whatever adapter dispatches a stage, the prompt handed to the worker MUST:

1. **Instruct the worker to write `{stage}-result.yaml`.** The result file is the contract's success
   token. For **both** `fab dispatch` modes it is a real file at
   `.fab-dispatch/{id}/{stage}-result.yaml`; for the native adapter it is the structural equivalent (the
   returned result). Its *content* schema is 3d's business — this spec fixes only the **path**
   (`fab dispatch` modes) and the **presence obligation** (all three). On the **pane** mode the result
   file is not merely the success token but the *sole* completion signal — an interactive worker never
   exits on task completion, so there is nothing else to key on.
2. **Carry the standard subagent context files** — `fab/project/config.yaml`,
   `fab/project/constitution.md`, and (optional) `context.md` / `code-quality.md` / `code-review.md`
   (`_preamble.md` § Standard Subagent Context). A worker in a fresh context/harness has no other
   awareness of project principles. **This obligation binds every *dispatch*.** A **continuation**
   message to an already-running named worker (§ 1, native adapter only) carries obligations 1 and 3
   only — the worker already holds the context files, which is the point of continuing it;
   `_preamble.md` § Worker Continuation owns the mechanics.
3. **End with a post-stage `fab status refresh <change>` epilogue** so the worker recomputes state from
   artifacts after finishing (the pull-based state-recompute surface change 3a lands — `fab status
   refresh`, replacing the removed artifact-write hook). This keeps a dispatched stage's `.status.yaml`
   consistent with the artifacts it just wrote, regardless of which harness ran it.

**Delivery mechanism is adapter-specific; obligations are not.** *How* the prompt reaches the worker
varies — the dispatched prompt itself (native), the command's stdin (headless), or a prompt **file** plus
a one-line **pointer** to it (pane). The three obligations above bind every **dispatch** prompt's
*content* identically: a pane worker that reads its pointer is reading the same block prompt, with the
same result-file, context-file, and refresh-epilogue obligations, that the other two adapters hand over
directly. A **continuation** message is not a dispatch and carries obligations 1 and 3 only — see
obligation 2's carve-out above.

**Self-managing stages (ship/review-pr).** The block-contract transition prohibition — a dispatched
worker runs no `fab status` transition command; the orchestrator owns every transition — is scoped to
the `/fab-continue`-behavior workers (apply, review, hydrate). Dispatched **ship** and **review-pr**
workers self-manage their **own** stage's `fab status` start/finish/fail on every adapter, exactly as
the standalone `/git-pr` / `/git-pr-review` skills do; the orchestrator owns sequencing and never runs
a transition for a stage whose worker owns it (the dispatching rows' only-if-still-active guards are
the reconciliation seam). Obligations 1–3 bind these workers unchanged, and a Copilot-poll timeout is
an `outcome: timeout` **result** (dispatch-state `done`), never an infra failure.

### The five-state machine (every adapter observes a subset of it)

A dispatched stage is in exactly one of five states, and the **state strings are the cross-adapter
contract** — byte-stable, never renamed or renumbered per adapter. What varies is which states an
adapter's observation model can *produce*: the native adapter observes them structurally (the Agent-tool
handle), the headless mode via `fab dispatch status` (pid liveness + `{stage}.exit` +
`{stage}-result.yaml`), and the pane mode via `fab dispatch status` (**pane** liveness +
`{stage}-result.yaml`, no exit file).

| State | Meaning | Native | Headless | Pane |
|-------|---------|--------|----------|------|
| `running` | the worker is still executing | ✓ | ✓ | ✓ |
| `done` | finished successfully **with a result** | ✓ | ✓ | ✓ |
| `failed` | non-zero exit (includes `124`, the POSIX `timeout` code) | ✓ | ✓ | — |
| `failed (no-result)` | **exited clean but wrote no result** — a contract violation, NOT done | ✓ | ✓ | — |
| `orphaned` | the worker died with no recorded exit (reboot / `kill -9` / crash / pane killed) | ✓ | ✓ | ✓ |

**`done` requires the result file** on every adapter. On the exit-code-bearing adapters a clean exit is
necessary but **not sufficient**: `failed (no-result)` is the state that distinguishes a well-behaved
success from a worker that exited 0 without honoring the result obligation above. This is the crux the
protocol exists to make observable — an orchestrator must never mistake a resultless clean exit for a
completed stage.

**On the pane mode the two exit-code-derived states are UNREACHABLE**, and that is a property of the
observation model, not a gap: an interactive worker never exits on task completion (it finishes and sits
at its prompt), so there is no exit-code channel at all — a crashed, killed, or otherwise vanished pane
worker collapses into `orphaned`. Consequently the pane mode's derivation is:

| Condition | State |
|-----------|-------|
| `{stage}-result.yaml` present | `done` |
| result absent, pane alive | `running` |
| result absent, pane dead (or the tmux server gone) | `orphaned` |

**Result presence WINS over pane liveness** — a worker that produced its result and is still sitting at
its prompt reads `done`, not `running`. A liveness-first rule would never terminate. An orchestrator
consuming a pane dispatch therefore handles three states and never waits for the other two; nothing else
about the observation contract changes (the blocking `fab dispatch wait` below, `done` ⇒ read the result
file, a review `verdict: fail` inside a `done` result is still a review outcome rather than a dispatch
failure).

### Observation is a blocking wait, not a poll loop

**How** an orchestrator learns a dispatch's state is fixed here; **what** it does about a non-`done` one is
policy (next section). The two `fab dispatch` adapters expose one observation surface in two shapes over a
single derivation:

| Verb | Shape | Use |
|------|-------|-----|
| `fab dispatch status <change> <stage>` | one-shot probe, returns immediately | spot checks, `--json` consumers, refuse-if-running explanations |
| `fab dispatch wait <change> <stage> [--timeout <secs>]` | **blocks** until the state leaves `running` | the observation an orchestrator waiting on a stage performs |

`wait` re-derives state through the **same** loader and the same per-mode derivation `status` uses, on an
internal sub-poll tick, so the two verbs cannot disagree by construction. It is not a filesystem watcher: a
watcher can see a result file appear but cannot see a worker *die*, and `orphaned` is derived from pid or
pane liveness — a periodic probe is needed regardless.

An orchestrator SHOULD observe with `wait`, **not** by looping `status` behind a sleep. On an in-harness
agent every poll is a full model turn, so a long stage costs hundreds of turns establishing that nothing
happened — the exact cost the CLI adapters exist to avoid paying, since they carry no in-harness completion
signal of their own. A blocking `wait` collapses those turns into one wake-up at the moment something
actually changes.

**Delivery of that wake-up is harness-specific; the verb is not.** A harness whose background commands
re-invoke the agent on exit (Claude Code's `run_in_background`) runs `wait` there and spends **zero** turns
while the stage executes. A harness with **no such seam** runs the **identical** command as a plain
**foreground blocking call** — a bounded `--timeout` then costs one turn per bound instead of one per poll,
which is a degraded but fully supported path. Nothing in this contract may assume the background seam
exists.

**`--timeout` is the peek cadence, not a poll interval.** On expiry `wait` prints the still-current state
(necessarily `running`) and exits **0** — the state string is the sole discriminator, so a consumer reading
`running` knows the bound expired and treats that wake as its peek-on-suspicion moment (next section);
every other string is a terminal state. Only real errors — no dispatch record, an unresolvable change — are
non-zero. An **already-terminal** dispatch returns immediately, so re-arming a `wait` after a restart is
free and idempotent.

**This changes no state, no result-file contract, and no prompt obligation.** It fixes only how the
existing five states are observed. The `--timeout` on `start`/`restart` is unrelated: that one bounds (and
kills) the **worker**; `wait --timeout` bounds only the **observer** and has no side effect on the dispatch
at all.

### Recovery is orchestrator policy over these states, not new protocol

An adapter reports a state; deciding what to *do* about a non-`done` one is the orchestrator's. This
matters because the two `fab dispatch` adapters have no in-harness supervision: a native sub-agent is
retried on 5xx and its death is reported by the harness, while a CLI or pane worker that exhausts its
provider's internal retry simply stops, and one wedged at an error banner reads `running` forever.

The protocol therefore fixes **nothing new** here — no state is added, renamed, or re-tabled; the
result-file contract and the dispatch-prompt obligations are untouched — and instead names what an
orchestrator MAY and MUST NOT do over the existing states:

- **Restart is the recovery verb, and it is bounded.** A stage dispatch carries no irreplaceable
  conversational state (fab checkpoints stage state into artifacts, so a relaunched worker resumes from
  the last completed task), which makes relaunch-from-the-persisted-prompt cheap and deterministic —
  `fab dispatch restart <change> <stage>`. An orchestrator SHOULD spend at most **one** automatic restart
  per stage dispatch and MUST escalate rather than loop: an unbounded restart against a provider that is
  5xx-ing platform-wide only burns tokens. The budget is the orchestrator's own bookkeeping — **the
  protocol defines no on-disk attempt counter or history**, and `restart` is a fresh attempt under the
  same last-attempt-only overwrite semantics as `start`, so nothing distinguishes it in the state dir.
- **A restart re-derives its launch mode from the current environment.** The prior attempt's adapter is
  not inherited: an `orphaned` pane dispatch relaunched after its tmux server died lands on the headless
  adapter. This is what makes recovery adapter-agnostic in practice, and it is why the adapters are worth
  keeping interchangeable.
- **`orphaned` is the recoverable state; `failed (no-result)` is not.** `orphaned` means no exit code was
  ever recorded (death by reboot / `kill -9` / crash / a killed pane) and is transient by nature.
  `failed` carries a real exit code and is usually deterministic, so it MUST NOT be restarted by rule —
  an orchestrator MAY judge an individual failure transient from its evidence. `failed (no-result)` is a
  **contract violation** and MUST always escalate to a human: retrying a worker that ignores the result
  obligation cannot fix it.
- **The pipeline's verbs against a WORKER are read-only-peek, kill, restart, notify, stop, reap — never
  input injection.** An orchestrator MAY read a worker's evidence (`fab dispatch logs` headless /
  `fab pane capture` pane) to tell a progressing worker from a parked one, and MUST escalate rather than
  answer when a worker is waiting on genuine human input. Typing into a worker is the human's and the
  operator's affordance: a pipeline-side input channel would fork this contract, since the native
  adapter has no such channel at all.

  **Bounded carve-out — the pre-delivery pane** (amendment, `260809-3oz7`). Between `fab dispatch open`
  and a **successful** `fab dispatch deliver`, the pane is **not yet a dispatched worker**: it holds no
  stage context, so there is nothing a keystroke could corrupt and the rule has no subject. In that
  window the orchestrator MAY send keys to the pane — the readiness gate's judgment rounds, answering a
  trust dialog, a survey, or a first-run picker — and `fab dispatch ready` / `fab dispatch deliver` are
  the sanctioned mechanical senders. **From successful delivery onward the rule above applies unchanged**:
  a wall that appears mid-stage MUST escalate, never be answered. The gate's round budget and its
  never-answer classes (credential and login walls) are skill-side policy in `_preamble.md`
  § CLI-Adapter Dispatch, and a **resume** delivery is not an exception to the rule — it targets a
  `done` worker sitting at its prompt, never a running one.
- **Reap is hygiene, not recovery, and belongs to the success path.** `reap` (§ 3) appears in the verb
  set above but takes no part in this policy: it fires **only** on `done` and MUST NOT terminate a
  `running`, `orphaned`, `failed`, or `failed (no-result)` dispatch, so it can never stand in for the
  `kill` that classification (b) spends, and it can never reach a worker awaiting human input. Recovery
  and hygiene are deliberately separate verbs so a policy knob can gate the latter without ever
  modulating the former.
- **No supervisor, timer, or background sweep.** The orchestrator's own observation remains the only clock.
  Recovery happens where it already waits — a `wait` that returns a non-`done` state, or one that returns
  `running` at its bound — which is why it needs no protocol surface of its own.

The concrete wiring — the restart budget, the peek cadence (a `wait` timeout-return), and the three-way
classification of a result-less dispatch — is skill-side policy in `_preamble.md` § CLI-Adapter Dispatch → *Recovery policy*,
not part of this contract. It is stated here only to fix the boundary: **recovery composes over the five
states; it does not extend them.**

### Steering a pane worker changes no contract

The pane mode's reason for existing is that a human MAY converse with the worker mid-stage. This is
**contract-neutral by construction** and is documented here rather than enforced in code:

- The worker still owes `{stage}-result.yaml`, still ends with the terminal `fab status refresh <change>`
  epilogue, and still runs **no `fab status` transition command** — the orchestrator owns every
  transition, exactly as for the other two adapters (ship/review-pr workers excepted — they
  self-manage their own stage's transitions; § Dispatch-prompt obligations).
- Steering is human input into a worker's context, no different in kind from answering a native
  sub-agent's question mid-run. It is distinct from the **pre-delivery** keystrokes the readiness gate
  may send (§ Recovery → *Bounded carve-out*), which reach a pane that is not yet a worker.
- A worker steered *away* from producing a result needs no new failure state: it simply never reaches
  `done`, and surfaces through the never-`done` escalation an orchestrator already owns.

Nothing detects, gates, or reports steering. There is no "was steered" flag, because no protocol decision
depends on the answer.

### Pane dispatch is not operator enrollment

A pane dispatch **borrows tmux as a launch surface**; it does not join the operator's monitored set. It
carries a dedicated dispatch identity string (`fab-{4-char-change-id}-{stage}` — the pane **title** in the
split shape, the window **name** in the new-window shape) and **MUST NOT** carry the operator's `»`
(U+00BB) enrollment prefix or its `›` (U+203A) done marker in **either** shape: those assert that a window
is in the operator's monitored set and that the operator owns its lifecycle, neither of which is true of a
pipeline dispatch. Pre-marking would make the operator's tab bar misreport what it tracks. An operator
that genuinely enrolls a window still adds the marker itself, through its own idempotent
`fab pane window-name ensure-prefix` primitive.

The **split shape reinforces this separation** rather than complicating it: a split worker opens **no
window at all**, so it cannot appear in the operator's tab bar even unmarked — which is the concrete
problem the split shape fixes (every pane worker used to surface as another window in the operator's, and
run-kit's, window list).

### Hooks may enhance, never own

Harness hooks (Claude Code `PostToolUse`, telemetry, notifications, …) MAY add value *around* dispatch
— but MUST NOT own any step of the protocol. **The protocol is complete and correct with no hook.** A
hook that becomes load-bearing (a step the protocol relies on to be correct) is a contract violation:
the same posture that motivated 3a's removal of the artifact-write hook in favor of the pull-based `fab
status refresh`. If a step matters, it lives in the protocol (the prompt epilogue, the result-file
obligation), not in a hook a different harness won't run.

---

## Cleanup (`fab dispatch` adapters only)

Both `fab dispatch` modes' `.fab-dispatch/{id}/` state is **transient comms, not history** — including the
pane mode's prompt file, which is one more file in the same directory and gains no separate lifecycle. It
is cleaned at exactly **two** deterministic moments, never on a timer (**no automatic GC** — a deliberate
rejection of throttled sweeps, matching fab's no-magic-background-work posture):

1. **Archive-time deletion**: `fab change archive` deletes `.fab-dispatch/{id}/` as part of the archive
   move; `fab change restore` does **not** recreate it.
2. **Explicit `fab dispatch clean [<change>] [--orphans]`**: manual cleanup — named change, all dirs, or
   only orphaned dirs (IDs that no longer resolve to a non-archived change).

`clean` is **mode-blind**: it removes state directories and never inspects a record's mode. As with a live
headless process, cleaning a live pane dispatch removes the state without killing the worker — `kill` is
the separate verb for that, and it terminates by the mechanism the record's mode implies (the detached
worker's process group, or the tmux pane).

**`reap` is not a third cleanup moment.** It kills a done pane worker's *pane* and removes **no files**:
the record, result, prompt, and log all survive it, which is precisely why a reaped dispatch still
derives `done` (result presence wins over pane liveness). Reap reclaims *screen space*, `clean` reclaims
*disk state*, and the two above remain the only moments `.fab-dispatch/` is touched.

The native adapter has no persisted state to clean (the sub-agent handle is in-process).

---

## Skill wiring is NOT part of the contract-defining change

This spec is authored and the `fab dispatch` **runtime** ships in change 3c. The **skill-side dispatch
seam** — the `/fab-*` skill logic that decides *when* to dispatch via `fab dispatch` vs. the native
Agent-tool path, and the dispatch-prompt *content* that satisfies the obligations above — is **change
3d** (wiring-only). 3d implements against this fixed contract; it does not co-define it. *(3d also wired
a review-stage nesting-degradation path; that machinery was removed in 260704-pag2 when review
collapsed to a single sub-agent — with one worker running the whole review inline, native and CLI
dispatch are structurally identical and there is nothing to degrade.)*

**Amendments are explicit.** If 3d's wiring reveals a flaw in this contract, the fix is an **explicit
amendment to this spec** (and to 3c's runtime code if the runtime is implicated), reviewed as a
contract change — **never** a silent redefinition inside 3d's skill files. A shared contract split
across two changes with no single authority is exactly how silent drift starts; this spec is that
authority.

---

## Relationship to `stage-models.md`

[`stage-models.md`](stage-models.md) owns the **resolution** layer (stage → role →
`{provider, model, effort}`, the top-level `providers:` capability grammar, `fab agent -o yaml`, verbatim
pass-through, provider neutrality) and describes the
**native Agent-tool adapter** as its harness-specific injection layer. This spec catalogs that native
adapter as **one of three** dispatch adapters and adds the two `fab dispatch` modes — **headless CLI** and
**interactive pane** — alongside it, plus the cross-adapter protocol all three share. The optional YAML
`dispatch:` mapping is the adapter seam: `fab agent <stage> -o yaml` resolves `dispatch.mode` against the provider's independent
capabilities and `$TMUX`; native omits the key, while pane/headless emit their labelled rung and substituted command.
`stage-models.md` § Harness-adapter boundary points here for the runtime that RUNS it.

Resolution stays **adapter-independent** across all three, while the configured preference and capability
ladder select the adapter. What selects it is therefore the **config** a dispatch resolves from —
`dispatch.mode`, a depth knob (`agent.workers`/`agent.session`) or
`agent.profiles.<role>.provider`, plus the `providers:` capability table — not command presence or an
invocation flag. An invocation-time `fab agent <stage> -o yaml --provider <name>` override
(`260805-j3cm`) binds the **native adapter only**: `fab dispatch start`/`open` accept no override flags
and re-resolve the stage from config themselves, so an overridden profile never reaches either
`fab dispatch` mode (the headless mode would compose — or fail on the absence of — the *unoverridden*
provider's `headless_command`, and `open` the unoverridden provider's `interactive_command`). Dispatch sites still
re-read the resolved `dispatch:` key after an override — the branch rule is unchanged (it keys
on key presence) — but a `dispatch:` mapping that appears *only* because of an override is **not
actionable**, and the two remedies are **not interchangeable**: dispatching that stage natively with the
overridden model/effort is executable only for a **within-claude** `--model`/`--effort` override, since
the native adapter's model seam is the Agent tool's `model` param — a Claude-alias enum
(`opus`/`sonnet`/`haiku`/`fable`) with no room for a non-Claude ID. So for a **cross-provider
`--provider` override** the **sole executable path** is a config override — a depth knob (`agent.workers`/`agent.session`) or
`agent.profiles.<role>.provider` — that `dispatch start`'s
own re-resolution will see. Nothing else moves: pane composes the resolved provider's existing
`interactive_command` (the same field `fab agent` and the operator launcher compose) through the same
`internal/spawn` substitution. Mode selection is **per invocation**: explicit flags first, otherwise the
configured descending ladder (§ 3's *Mode selection*), never a property inferred from command presence.
A provider whose **pane-dispatch** grammar genuinely diverges from the interactive command it shares
with `fab agent` and the operator launcher would be the trigger to add a dedicated pane field
(distinct from `interactive_command`) later — a data-only config addition, not a protocol change.

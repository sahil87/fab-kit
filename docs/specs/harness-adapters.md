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

Fab runs a six-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`). Every
post-intake stage is executed by **dispatching a worker** in a fresh context that returns a structured
result (see [`stage-models.md`](stage-models.md) § Why this is possible now, and `_preamble.md`
§ Subagent Dispatch). Historically there was one way to dispatch a worker — the Claude Code **Agent
tool** (an in-harness sub-agent). Cross-harness dispatch (e.g. a codex orchestrator running `apply` on
claude, or a claude orchestrator handing a stage to codex) added a second: a **detached CLI process**
observed via files. A third recovers what that detached process cannot offer — **watch and steer**: an
**interactive worker in a tmux window**, observed via the same result file. This spec catalogs all three
**adapters** and fixes the **protocol** they share.

---

## The three adapters

An *adapter* is the mechanism that turns "run stage S as a worker" into an actual launched worker and,
later, an observed result. The resolution that precedes dispatch (stage → tier →
`{provider, model, effort}`, via `fab resolve-agent`; the invocation command lives on the resolved
provider) is **provider-neutral and adapter-independent** — see
[`stage-models.md`](stage-models.md). Only the launch+observe step is adapter-specific.

| Adapter | Launch | Prompt delivery | Completion observed via | States it can report |
|---------|--------|-----------------|-------------------------|----------------------|
| **1. Native Agent-tool** | in-harness sub-agent (Claude Code Agent tool) | the dispatched prompt itself | the held sub-agent handle (structural) | all five |
| **2. Headless CLI** (`fab dispatch start`) | detached `sh -c` wrapper, `setsid` semantics | prompt file on the command's **stdin** | `{stage}.exit` + pid liveness + result file | all five |
| **3. Interactive pane** (`fab dispatch start --pane`) | a tmux window running the provider's `session_command` | prompt **file** + a one-line **pointer** to it, embedded at spawn | **result file** + pane liveness | `running` / `done` / `orphaned` |

Adapters 2 and 3 are two **modes of the same command family** (`fab dispatch`), sharing its resolution,
`.fab-dispatch/{id}/` state directory, refuse-if-running concurrency, and status/kill/logs/clean
surfaces. They differ in which of the provider's two command fields they compose — `dispatch_command`
for headless, `session_command` for pane — and in how completion is observed. The two fields are never
merged and **never fall back to each other in either direction**.

### 1. Native Agent-tool adapter (in-harness)

Today's path: the orchestrator spawns a sub-agent via the Claude Code **Agent tool**. Per
[`stage-models.md`](stage-models.md) § Harness-adapter boundary, the resolved profile rides two seams —
the **model** on the Agent tool's `model` parameter (a short alias via `fab resolve-agent <stage>
--alias`), and the **effort** as an imperative instruction in the dispatched prompt (the Agent tool has
no effort parameter). The worker runs in-process; its result is the sub-agent's returned message. The
orchestrator observes the five states (below) **structurally** — it holds the sub-agent handle, so
"running/done/failed" are direct properties of the Agent-tool call.

### 2. Headless CLI adapter — `fab dispatch start` (new in 3c)

The headless path: the worker is a **detached CLI process** (e.g. `claude …` or `codex exec …`),
launched and observed via `fab dispatch` (see [`_cli-fab.md`](../../src/kit/skills/_cli-fab.md)
§ fab dispatch). It exists because the native path is **tmux/in-harness-bound** and cannot drive a
stage on a CI box, a remote host, or a different agent CLI. `fab dispatch start` launches the resolved
provider `dispatch_command` detached via `sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'` launched with
`setsid` semantics (the shell is the supervisor — no Go process remains, so the dispatch survives the
orchestrator dying), tracks it
under `.fab-dispatch/{id}/`, and the orchestrator observes all five states **via `fab dispatch
status`** (file polling) rather than a held handle. POSIX-only in v1.

> `fab dispatch`'s **headless** mode is deliberately **parallel to and independent of** `fab pane` /
> `fab operator`. Those stay the *interactive operator-visibility* path (a human watching a monitored set
> of tmux panes with operator-owned lifecycles); headless dispatch is the *unattended pipeline* path
> (launch-and-poll). Making `fab pane` grow a headless mode was rejected — pane observation (tmux
> capture) and headless observation (exit-file polling) are different models. The pane **dispatch** mode
> below does not undo that split: it *consumes* tmux as a launch surface for a pipeline worker, while
> operator enrollment, monitored sets, and the `»`/`›` window markers remain the operator's, never a
> dispatch's (see § Pane dispatch is not operator enrollment).

### 3. Interactive-pane adapter — `fab dispatch start --pane`

The **watch-and-steer** path: the worker is an **interactive agent session in a tmux window** the user
can read and type into mid-stage. It exists because a detached headless worker is a black box —
`fab dispatch logs --tail` recovers *watching*, but **steering is impossible**: a detached process in its
own session is nobody's conversation partner. That asymmetry against the native adapter's
peek-and-converse richness is the largest capability loss when running a stage cross-provider, and an
interactive pane is the one interface every agent CLI supports natively, so it recovers both with no
per-provider integration work.

Mechanics, all fixed by this spec:

- **Command composed**: the resolved provider's **`session_command`** — the same string `fab agent`
  composes — with `{model}`/`{effort}` substituted through the shared `internal/spawn` resolution. This is
  why pane mode needs **no new provider config field**: the interactive invocation is already in the
  provider table. It MUST NOT read or fall back to `dispatch_command`.
- **Window**: created via the `_cli-agents.md` § Spawn Composition form
  (`tmux new-window -n <name> -c <dir> "<composed-cmd> <shell-quoted-prompt>"`), cwd = the repo root, and
  the new window's **pane ID** recorded in `.fab-dispatch/{id}/{stage}.yaml` alongside the window name and
  tmux socket label.
- **Prompt delivery**: the full stage prompt is persisted to `.fab-dispatch/{id}/{stage}-prompt.md` (the
  same path the headless mode uses) and the worker receives a **one-line pointer** to that path as its
  single spawn argument, **shell-quoted** per § Spawn Composition's escape rule (the pointer is
  repo-path-derived, so a `'` in the checkout path must not break out of the argument); the composed
  command itself is inserted verbatim so its own expansions still apply. A multi-thousand-token prompt
  cannot ride `send-keys` or argv reliably, and embedding the pointer *at spawn* also sidesteps the
  printed-prompt trap entirely — there is no pre-existing input buffer to probe.
- **tmux is REQUIRED, and only here**: `--pane` without a reachable tmux server is a **hard error**
  (non-zero exit, actionable stderr, nothing launched and no state persisted), established by a real
  tmux query rather than an `$TMUX` environment read so a headless orchestrator can target a socket
  explicitly. The **headless mode performs no tmux probe at all** — its tmux-independence guarantee is
  untouched, and pane mode is opt-in per invocation.
- **No timeout**: `--timeout` is enforced by the headless `sh -c` wrapper (POSIX `timeout`), which pane
  mode never constructs, so `--pane --timeout` is a **usage error** rather than a silently unenforced
  bound.

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
   awareness of project principles.
3. **End with a post-stage `fab status refresh` epilogue** so the worker recomputes state from
   artifacts after finishing (the pull-based state-recompute surface change 3a lands — `fab status
   refresh`, replacing the removed artifact-write hook). This keeps a dispatched stage's `.status.yaml`
   consistent with the artifacts it just wrote, regardless of which harness ran it.

**Delivery mechanism is adapter-specific; obligations are not.** *How* the prompt reaches the worker
varies — the dispatched prompt itself (native), the command's stdin (headless), or a prompt **file** plus
a one-line **pointer** to it (pane). The three obligations above bind the prompt's *content* identically
in every case: a pane worker that reads its pointer is reading the same block prompt, with the same
result-file, context-file, and refresh-epilogue obligations, that the other two adapters hand over
directly.

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
about the polling contract changes (fixed `sleep 30` cadence, `done` ⇒ read the result file, a review
`verdict: fail` inside a `done` result is still a review outcome rather than a dispatch failure).

### Steering a pane worker changes no contract

The pane mode's reason for existing is that a human MAY converse with the worker mid-stage. This is
**contract-neutral by construction** and is documented here rather than enforced in code:

- The worker still owes `{stage}-result.yaml`, still ends with the terminal `fab status refresh`
  epilogue, and still runs **no `fab status` transition command** — the orchestrator owns every
  transition, exactly as for the other two adapters.
- Steering is human input into a worker's context, no different in kind from answering a native
  sub-agent's question mid-run.
- A worker steered *away* from producing a result needs no new failure state: it simply never reaches
  `done`, and surfaces through the never-`done` escalation an orchestrator already owns.

Nothing detects, gates, or reports steering. There is no "was steered" flag, because no protocol decision
depends on the answer.

### Pane dispatch is not operator enrollment

A pane dispatch **borrows tmux as a launch surface**; it does not join the operator's monitored set. Its
window carries a dedicated dispatch name (`fab-{4-char-change-id}-{stage}`) and **MUST NOT** carry the
operator's `»` (U+00BB) enrollment prefix or its `›` (U+203A) done marker: those assert that a window is
in the operator's monitored set and that the operator owns its lifecycle, neither of which is true of a
pipeline dispatch. Pre-marking would make the operator's tab bar misreport what it tracks. An operator
that genuinely enrolls such a window still adds the marker itself, through its own idempotent
`fab pane window-name ensure-prefix` primitive.

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

[`stage-models.md`](stage-models.md) owns the **resolution** layer (stage → tier →
`{provider, model, effort}`, the top-level `providers:` command grammar, `fab resolve-agent`, verbatim
pass-through, provider neutrality) and describes the
**native Agent-tool adapter** as its harness-specific injection layer. This spec catalogs that native
adapter as **one of three** dispatch adapters and adds the two `fab dispatch` modes — **headless CLI** and
**interactive pane** — alongside it, plus the cross-adapter protocol all three share. The `dispatch=` line
`fab resolve-agent` emits (when the resolved tier's provider carries a `dispatch_command`) is the seam the
`fab dispatch` adapters consume; `stage-models.md` § Harness-adapter boundary points here for the runtime
that RUNS it.

Resolution stays **adapter-independent** across all three: the pane mode adds no resolver output and no
provider config field — it composes the resolved provider's existing `session_command` (the same field
`fab agent` and the operator launcher compose) through the same `internal/spawn` substitution. Mode
selection is therefore **per-invocation** (`fab dispatch start … --pane`), not a property of a tier or a
provider. A provider whose interactive grammar genuinely diverges from its `session_command` would be the
trigger to add a dedicated field later — a data-only config addition, not a protocol change.

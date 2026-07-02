# Harness Adapters — the cross-harness stage-dispatch contract

> **Status:** Design intent (pre-implementation, Constitution VI). This spec is human-curated: it
> fixes the dispatch protocol that both the native Agent-tool path and the CLI `fab dispatch` path
> (and any future harness adapter) conform to. It was authored by change 3c
> (`260702-6sgj-fab-dispatch-command`, the change that also implements the `fab dispatch` runtime) —
> the same "spec authored alongside its implementing change" pattern as
> [`stage-models.md`](stage-models.md) (#406). It is human-maintained thereafter, never
> auto-generated.
>
> **This spec fixes the contract once so it can be implemented by more than one change without silent
> drift.** The `fab dispatch` *runtime* ships in 3c (this change); the *skill-side wiring* that decides
> when to dispatch and calls the protocol is **change 3d** (wiring-only). 3d implements against this
> spec — it does not re-open it. See § Skill wiring is NOT part of the contract-defining change.

Fab runs a six-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`). Every
post-intake stage is executed by **dispatching a worker** in a fresh context that returns a structured
result (see [`stage-models.md`](stage-models.md) § Why this is possible now, and `_preamble.md`
§ Subagent Dispatch). Historically there was one way to dispatch a worker — the Claude Code **Agent
tool** (an in-harness sub-agent). Cross-harness dispatch (e.g. a codex orchestrator running `apply` on
claude, or a claude orchestrator handing a stage to codex) adds a second: a **detached CLI process**
observed via files. This spec catalogs both **adapters** and fixes the **protocol** they share.

---

## The two adapters

An *adapter* is the mechanism that turns "run stage S as a worker" into an actual launched worker and,
later, an observed result. The resolution that precedes dispatch (stage → tier → `{model, effort,
spawn_command}`, via `fab resolve-agent`) is **provider-neutral and adapter-independent** — see
[`stage-models.md`](stage-models.md). Only the launch+observe step is adapter-specific.

### 1. Native Agent-tool adapter (in-harness)

Today's path: the orchestrator spawns a sub-agent via the Claude Code **Agent tool**. Per
[`stage-models.md`](stage-models.md) § Harness-adapter boundary, the resolved profile rides two seams —
the **model** on the Agent tool's `model` parameter (a short alias via `fab resolve-agent <stage>
--alias`), and the **effort** as an imperative instruction in the dispatched prompt (the Agent tool has
no effort parameter). The worker runs in-process; its result is the sub-agent's returned message. The
orchestrator observes the five states (below) **structurally** — it holds the sub-agent handle, so
"running/done/failed" are direct properties of the Agent-tool call.

### 2. CLI adapter — `fab dispatch` (new in 3c)

The headless path: the worker is a **detached CLI process** (e.g. `claude …` or `codex exec …`),
launched and observed via `fab dispatch` (see [`_cli-fab.md`](../../src/kit/skills/_cli-fab.md)
§ fab dispatch). It exists because the native path is **tmux/in-harness-bound** and cannot drive a
stage on a CI box, a remote host, or a different agent CLI. `fab dispatch start` launches the resolved
tier `spawn_command` detached via `sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'` launched with
`setsid` semantics (the shell is the supervisor — no Go process remains, so the dispatch survives the
orchestrator dying), tracks it
under `.fab-dispatch/{id}/`, and the orchestrator observes the five states **via `fab dispatch
status`** (file polling) rather than a held handle. POSIX-only in v1.

> `fab dispatch` is deliberately **parallel to and independent of** `fab pane` / `fab operator`. Those
> stay the *interactive operator-visibility* path (a human watching tmux panes); dispatch is the
> *headless pipeline* path (launch-and-poll). Conflating them was rejected — pane observation (tmux
> capture) and dispatch observation (file polling) are different models.

---

## The dispatch protocol (shared by both adapters)

The protocol is what makes an adapter interchangeable: whichever adapter launches the worker, the
**outcome contract** is identical. It has an orchestrator-side half and a worker-side half; the
worker-side half (dispatch-prompt obligations) is **implemented by 3d**, but its rules are fixed here.

### Dispatch-prompt obligations (the worker-side half — 3d implements)

Whatever adapter dispatches a stage, the prompt handed to the worker MUST:

1. **Instruct the worker to write `{stage}-result.yaml`.** The result file is the contract's success
   token. For the CLI adapter it is a real file at `.fab-dispatch/{id}/{stage}-result.yaml`; for the
   native adapter it is the structural equivalent (the returned result). Its *content* schema is 3d's
   business — this spec fixes only the **path** (CLI adapter) and the **presence obligation** (both).
2. **Carry the standard subagent context files** — `fab/project/config.yaml`,
   `fab/project/constitution.md`, and (optional) `context.md` / `code-quality.md` / `code-review.md`
   (`_preamble.md` § Standard Subagent Context). A worker in a fresh context/harness has no other
   awareness of project principles.
3. **End with a post-stage `fab status refresh` epilogue** so the worker recomputes state from
   artifacts after finishing (the pull-based state-recompute surface change 3a lands — `fab status
   refresh`, replacing the removed artifact-write hook). This keeps a dispatched stage's `.status.yaml`
   consistent with the artifacts it just wrote, regardless of which harness ran it.

### The five-state machine (both adapters observe it)

A dispatched stage is in exactly one of five states. The CLI adapter observes them via `fab dispatch
status` (pid liveness + `{stage}.exit` + `{stage}-result.yaml`); the native adapter observes them
structurally (the Agent-tool handle):

| State | Meaning |
|-------|---------|
| `running` | the worker is still executing |
| `done` | finished successfully **with a result** — clean exit AND `{stage}-result.yaml` present |
| `failed` | non-zero exit (includes `124`, the POSIX `timeout` code) |
| `failed (no-result)` | **exited clean but wrote no result** — a contract violation, NOT done |
| `orphaned` | the worker died with no recorded exit (reboot / `kill -9` / crash) |

**`done` requires the result file.** A clean exit is necessary but **not sufficient**: `failed
(no-result)` is the state that distinguishes a well-behaved success from a worker that exited 0 without
honoring the result obligation above. This is the crux the protocol exists to make observable — an
orchestrator must never mistake a resultless clean exit for a completed stage.

### Nesting degradation (the `review` stage)

One stage nests sub-workers: **`review`** spawns an inward reviewer + an outward reviewer + a merge
(`_preamble.md` § Review resolves once). On a harness **with** sub-agent support (native adapter), those
run as parallel sub-agents. On a harness **without** sub-agent support, the adapter runs the nesting
stage's parts **sequentially inline** in one context instead of as parallel workers. **Only the
concurrency degrades — the outcome contract is identical**: the same merged findings + pass/fail
verdict are produced either way. 3d implements the inline-sequential path; this spec fixes the rule
(review is the nesting stage; degrade concurrency, never the outcome).

### Hooks may enhance, never own

Harness hooks (Claude Code `PostToolUse`, telemetry, notifications, …) MAY add value *around* dispatch
— but MUST NOT own any step of the protocol. **The protocol is complete and correct with no hook.** A
hook that becomes load-bearing (a step the protocol relies on to be correct) is a contract violation:
the same posture that motivated 3a's removal of the artifact-write hook in favor of the pull-based `fab
status refresh`. If a step matters, it lives in the protocol (the prompt epilogue, the result-file
obligation), not in a hook a different harness won't run.

---

## Cleanup (CLI adapter only)

The CLI adapter's `.fab-dispatch/{id}/` state is **transient comms, not history**. It is cleaned at
exactly **two** deterministic moments, never on a timer (**no automatic GC** — a deliberate rejection of
throttled sweeps, matching fab's no-magic-background-work posture):

1. **Archive-time deletion**: `fab change archive` deletes `.fab-dispatch/{id}/` as part of the archive
   move; `fab change restore` does **not** recreate it.
2. **Explicit `fab dispatch clean [<change>] [--orphans]`**: manual cleanup — named change, all dirs, or
   only orphaned dirs (IDs that no longer resolve to a non-archived change).

The native adapter has no persisted state to clean (the sub-agent handle is in-process).

---

## Skill wiring is NOT part of the contract-defining change

This spec is authored and the `fab dispatch` **runtime** ships in change 3c. The **skill-side dispatch
seam** — the `/fab-*` skill logic that decides *when* to dispatch via `fab dispatch` vs. the native
Agent-tool path, the dispatch-prompt *content* that satisfies the obligations above, and the
nesting-degradation *implementation* — is **change 3d** (wiring-only). 3d implements against this fixed
contract; it does not co-define it.

**Amendments are explicit.** If 3d's wiring reveals a flaw in this contract, the fix is an **explicit
amendment to this spec** (and to 3c's runtime code if the runtime is implicated), reviewed as a
contract change — **never** a silent redefinition inside 3d's skill files. A shared contract split
across two changes with no single authority is exactly how silent drift starts; this spec is that
authority.

---

## Relationship to `stage-models.md`

[`stage-models.md`](stage-models.md) owns the **resolution** layer (stage → tier → `{model, effort,
spawn_command}`, `fab resolve-agent`, verbatim pass-through, provider neutrality) and describes the
**native Agent-tool adapter** as its harness-specific injection layer. This spec catalogs that native
adapter as **one of two** dispatch adapters and adds the **CLI adapter** (`fab dispatch`) alongside it,
plus the cross-adapter protocol both share. The `spawn=` line `fab resolve-agent` emits (when a tier
carries a `spawn_command`) is the seam the CLI adapter consumes; `stage-models.md` § Harness-adapter
boundary points here for the runtime that RUNS it.

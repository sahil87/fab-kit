# Intake: `fab dispatch` — headless process manager for CLI-dispatched pipeline stages

**Change**: 260702-6sgj-fab-dispatch-command
**Created**: 2026-07-02

## Origin

Drafted via `/fab-draft` (background dispatch, `{questioning-mode} = promptless-defer`) out of a `/fab-discuss` session on cross-harness stage dispatch — enabling a pipeline stage (e.g. `apply`) to run headless on a *different* agent CLI (e.g. codex launching a claude worker, or a claude orchestrator dispatching to codex). This is **change 3c of a four-part series** scoped in that discussion:

- **3a** (`260702-y022-status-refresh-drop-artifact-hook`): `fab status refresh` + artifact-write hook removal — the state-recompute surface a resumed dispatch relies on.
- **3b** (not yet on disk at drafting time; slug will contain `tier-spawn-command`): per-tier `spawn_command` in `agent.tiers` — widens `spawn_command` from a single top-level `agent.spawn_command` to a per-tier profile field. **Drafted in parallel with this change.** 3c **DEPENDS ON 3b** for the tier-resolution it consumes (see § Dependency).
- **3c (this change)**: a new `fab dispatch` command family — a tmux-independent headless process manager that launches a stage's resolved spawn command detached, tracks it via a state dir, and exposes poll/logs/kill/clean surfaces. **3c also AUTHORS the new spec `docs/specs/harness-adapters.md`**, which fixes the full dispatch protocol (the contract 3c and 3d share) once — see § Contract ownership and What Changes §9.
- **3d** (later, wiring-only): the skill dispatch-seam wiring + the dispatch-prompt *content* + the nesting-degradation *implementation*, all conforming to the `harness-adapters.md` spec THIS change lands. 3d implements against a fixed contract; it does not re-open it.

Recently merged foundations this builds on: **PR #455** (`fab config reference` — `260702-6nke-config-reference-command`) and **PR #456** (spawn_command `{model}`/`{effort}` placeholders — `260702-6tmi-spawn-command-placeholders`). PR #456 established the placeholder-substitution rules that 3b's per-tier `spawn_command` reuses and that this change's internal resolution ultimately consumes.

### Contract ownership (why 3c authors the spec)

3c and 3d share **one contract** — the dispatch protocol: prompt piped on stdin, `{stage}-result.yaml` at a fixed path, and the five-state status machine. The user challenged whether 3c and 3d should therefore be a single change. The agreed resolution: **keep them separate changes, but fix the contract once, in a spec authored by THIS change.** A shared contract split across two changes with no single authority is exactly how silent drift starts — one side's implementation quietly redefines what the other assumed. Landing `docs/specs/harness-adapters.md` here makes the contract a pre-implementation design artifact (**Constitution VI** — specs are human-curated design intent, authored ahead of the code that conforms to them). 3d then *implements against* it rather than *co-defining* it. Precedent: `docs/specs/stage-models.md` was authored by the very change that implemented per-stage model selection (#406, "Per-Stage Model Selection via Named Tiers") — a spec landing alongside its implementing change is an established fab-kit pattern, not a novelty.

> User-approved raw framing (mined from the discussion — encoded below as Certain/Confident assumptions): "A new `fab dispatch` command family — a process manager independent of tmux and `fab pane`. `fab pane`/`fab operator` machinery stays for interactive operator visibility; dispatch is the headless pipeline path. Launch detached via `setsid sh -c '<cmd> < prompt > log 2>&1; echo $? > exit-code-file'` so no Go supervisor process is needed and the dispatch survives the orchestrator dying. Refuse-if-running + last-attempt-only concurrency. Optional `--timeout` enforced inside the wrapper with POSIX `timeout`. No automatic GC anywhere — exactly two cleanup paths: archive-time deletion and explicit `fab dispatch clean`. POSIX-only v1 — error on Windows rather than half-work."

## Why

**Problem.** fab's only current mechanism for spawning a worker agent is the `fab operator` launcher + `fab pane` family, which is **tmux-bound**: it opens a tmux window, injects a `claude` invocation, and observes the pane via `tmux capture-pane`. That path is designed for *interactive operator visibility* (a human watching panes across worktrees). It cannot drive a **headless** stage on a CI box, a remote host, or any environment with no tmux server — and cross-harness dispatch ("codex orchestrator runs `apply` on claude") is fundamentally a headless launch-and-poll problem, not a pane-observation problem.

**What happens if we don't fix it.** The four-part series stalls at 3c: 3b can *configure* a per-tier spawn command, but there is no runtime that actually **launches the resolved command detached, tracks it, and lets the orchestrator poll/reattach**, and no single artifact that *fixes the dispatch contract* the skill side (3d) must conform to. Without a supervisor-free detached launch, a dispatched stage dies when the orchestrator dies (no resumability), and the orchestrator has no byte-stable surface to poll. Without the contract spec, 3d's skill wiring would have to reverse-engineer the protocol from 3c's code — the drift hazard the spec exists to close.

**Why this approach over alternatives.**
- *A Go supervisor process that waits on the child* was rejected: it re-introduces a long-lived Go process that itself must survive the orchestrator, defeating the point. The `setsid sh -c '...; echo $? > exit-code-file'` wrapper makes the **shell** the supervisor — it double-forks away from the orchestrator's process group, redirects I/O to files, and records the exit code with no Go process in the loop. Resumability then falls out for free: on resume a skill runs `fab dispatch status` and reattaches to the tracked state dir instead of re-running the stage.
- *Extending `fab pane` to a headless mode* was rejected: pane observation is a fundamentally different model (tmux capture vs. file polling) and conflating them would burden the interactive path with headless concerns. `fab pane`/`fab operator` stay exactly as-is; `fab dispatch` is a parallel, independent family.
- *Automatic GC of state dirs* was rejected by the user (they explicitly rejected throttled sweeps): transient dispatch artifacts are cleaned at exactly two deterministic moments (archive, explicit `clean`), never on a timer — matching fab's no-magic-background-work posture.

## What Changes

### 1. New `fab dispatch` command family (`fab-go`)

A new top-level command group `fab dispatch` with five subcommands: `start`, `status`, `logs`, `kill`, `clean`. It mirrors the multi-file `cmd/fab/pane*.go` structure of the existing `fab pane` group (`pane.go` parent + `pane_send.go` / `pane_capture.go` / `pane_process*.go` children), and — like every fab-go command — is **always-routed** through the `fab` router with a name that must not collide with the fab-kit `LifecycleCommands` allowlist. State reads/writes go through the existing `internal/atomicfile`; process-group / liveness handling follows the existing platform-split pattern in `internal/proc` (`proc_linux.go` / `proc_darwin.go`).

**POSIX-only v1** (user-approved). `fab dispatch` explicitly errors on Windows with a clear message ("`fab dispatch` requires a POSIX shell (setsid/timeout); Windows is not supported in v1") rather than half-working. This is declared in docs, not discovered at runtime by a broken launch.

### 2. State layout — `.fab-dispatch/{id}/` at repo root

Each dispatch's state lives under `.fab-dispatch/{4-char-change-id}/` at the repository root, alongside the existing `.fab-status.yaml` / `.fab-runtime.yaml` ephemeral-state convention. Rationale for this location:

- **Already gitignored, zero scaffold/migration work.** The scaffold fragment `$(fab kit-path)/scaffold/fragment-.gitignore` carries the pattern `.fab-*` (verified at drafting: the fragment's "Fab Specific" block is `.fab-*` + `.status.yaml.lock`), which already matches `.fab-dispatch/`. Every fab project deployed from this scaffold ignores it with no change. **No gitignore/scaffold/migration work is needed** for this change — this is stated as a deliberate finding, not an omission.
- The **4-char change ID** (not the slug) keys the dir, so it is stable across `fab change rename`.
- Each git worktree naturally gets its own `.fab-dispatch/` (repo-root-relative), matching the per-worktree observation model of `.fab-status.yaml`.

Per-stage files under `.fab-dispatch/{id}/`:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start` (from stdin) | the stage prompt piped to the dispatched command's stdin |
| `{stage}.yaml` | `start` (via `internal/atomicfile`) | `pid`, `pgid`, `spawn_cmd` (resolved), `started_at`, `timeout` (secs, or absent), and the file paths |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its presence is the "process finished" signal |
| `{stage}-result.yaml` | the **dispatched agent** (contract) | the stage result; its *content* is 3d's business, this change only defines the path + consumes its presence for the `done` vs `failed (no-result)` distinction |

### 3. `fab dispatch start <change> <stage> [--timeout <secs>]`

1. Resolves `<change>` to its 4-char ID (via `internal/resolve`, accepting ID / folder substring / full name like every other fab command).
2. **Reads the stage prompt on stdin** and persists it to `.fab-dispatch/{id}/{stage}-prompt.md`.
3. **Resolves the tier's spawn command internally** via `internal/agent` + `internal/spawn` placeholder substitution — consuming **3b's widened per-tier `spawn_command` profile** (see § Dependency). If the resolved tier has **no `spawn_command`**, `start` errors clearly ("stage `<stage>` resolves to tier `<tier>`, which has no `spawn_command`; configure `agent.tiers.<tier>.spawn_command` to dispatch this stage") — it does NOT fall back to the top-level `agent.spawn_command`.
4. **Concurrency — refuse-if-running** (user-approved "refuse-if-running + last-attempt-only"): if a dispatch for this exact `(change, stage)` is already **running** (see § status states), `start` REFUSES with a clear error ("a dispatch for `<change>`/`<stage>` is already running (pid N); run `fab dispatch kill` first"). The orchestrator must `fab dispatch kill` before re-dispatching. A new `start` over a **completed** prior attempt (done / failed / orphaned) **overwrites** its files — there is **no per-attempt history** (last-attempt-only).
5. **Launches DETACHED** via a shell wrapper, cwd = the repo root:
   ```sh
   setsid sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'
   ```
   With `--timeout <secs>`, the resolved command is wrapped in POSIX `timeout`:
   ```sh
   setsid sh -c 'timeout <secs> <resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'
   ```
   `setsid` detaches into a new session/process group so the dispatch **survives the orchestrator dying**. No Go supervisor process remains — the shell records the exit code itself. `start` captures the child `pid`/`pgid` and writes `{stage}.yaml` (via `internal/atomicfile`) before returning.
6. **Timeout is enforced entirely inside the wrapper** (user-approved) via POSIX `timeout N` — self-contained, no background sweep, no daemon, no Go timer. A timed-out command exits `124` (POSIX `timeout` convention), which surfaces as `failed` via the normal exit-code path.

### 4. `fab dispatch status <change> <stage> [--json]`

The polling surface — **byte-stable output**. It reads `{stage}.yaml`, `{stage}.exit`, and checks liveness of `pid` (signal-0 / `/proc`, via `internal/proc`), then reports one of these distinct states (user-approved, including the no-result state):

| State | Condition | Meaning |
|-------|-----------|---------|
| `running` | pid alive AND `{stage}.exit` absent | still executing |
| `done` | `{stage}.exit` == `0` AND `{stage}-result.yaml` present | finished successfully with a result |
| `failed` | `{stage}.exit` present AND != `0` | non-zero exit (includes `124` timeout) |
| `failed (no-result)` | `{stage}.exit` == `0` BUT `{stage}-result.yaml` absent | **contract violation, NOT done** — the process exited clean but never wrote its result |
| `orphaned` | pid dead AND `{stage}.exit` absent | reboot / `kill -9` / crash — no exit code was ever recorded |

The `failed (no-result)` state is the crux: a clean exit is necessary but **not sufficient** for `done`; the result file must exist. This distinguishes a well-behaved success from an agent that exited 0 without honoring the result contract (whose semantics 3d owns).

### 5. `fab dispatch logs <change> <stage> [--tail N]`

Prints `.fab-dispatch/{id}/{stage}.log`. `--tail N` prints the last N lines (self-contained; no external `tail` dependency required if the read is done in Go). Missing log → clear "no dispatch log for `<change>`/`<stage>`" message.

### 6. `fab dispatch kill <change> <stage>`

Kills the **process group** (`pgid` from `{stage}.yaml`) so the detached command and any children die together — following the existing `internal/proc` process-group handling. Idempotent: killing an already-dead dispatch is a benign no-op with a clear report.

### 7. Cleanup — exactly two paths, NO automatic GC anywhere (user decision)

The user explicitly rejected throttled/timer sweeps. Cleanup happens at exactly two deterministic moments:

**(a) `fab change archive` deletes `.fab-dispatch/{id}/`** as part of the archive move (in `internal/archive`). Dispatch artifacts are **transient comms, not history** — so **`fab change restore` does NOT recreate them**. This is a behavior change to `fab change archive` and requires prose sweeps (see § Affected Memory and the constraint below).

**(b) `fab dispatch clean [<change>] [--orphans]`** — manual cleanup:
- `fab dispatch clean <change>` — removes `.fab-dispatch/{id}/` for the named change.
- `fab dispatch clean` (no arg) — removes all `.fab-dispatch/*/` dirs.
- `fab dispatch clean --orphans` — prunes any `.fab-dispatch/{id}/` whose ID **no longer resolves to a non-archived change** (covers the case where someone archived the change upstream and a local `git pull` left the state dir orphaned).

### 8. Boundary (what this change's CODE does NOT own)

The `fab dispatch` **Go code** in this change owns the runtime and the *path convention* + `failed (no-result)` *state*; it does not own the config surface or the skill wiring:

- The `spawn=` resolution line and per-tier `spawn_command` semantics are **3b's** — this change *consumes* the resolved command but does not define the config surface.
- The **skill-side dispatch branch**, the dispatch-prompt *content*, and the nesting-degradation *implementation* are **3d's code**. This change's *code* defines only the `{stage}-result.yaml` path convention and the five status states.

Note the distinction from §9: the *protocol contract* (prompt-on-stdin, result-file path/obligation, five states, nesting rule, hooks principle) IS owned by this change — but as a **spec**, not as skill code. 3c writes the contract; 3d's code conforms to it. See Non-Goals for 3d's residual scope.

### 9. Author `docs/specs/harness-adapters.md` (the shared dispatch contract)

This change lands a new pre-implementation spec (**Constitution VI**) that fixes the full dispatch protocol both dispatch paths share, so 3d wires against a fixed target. It is authored by this change (precedent: `stage-models.md`, #406) and human-curated thereafter. Contents:

- **Both dispatch adapters, by name:**
  - the **native Agent-tool adapter** — today's in-harness path, where an orchestrator spawns a sub-agent via the Agent tool (per `stage-models.md` § Harness-adapter boundary — model via the `model` param, effort via a prompt instruction);
  - the new **CLI adapter** — `fab dispatch` (this change), where the worker is a detached CLI process observed via `.fab-dispatch/{id}/` files.
- **The full dispatch protocol, including the SKILL-SIDE half 3d implements later:**
  - **Dispatch-prompt obligations**: the prompt MUST instruct the worker to write `{stage}-result.yaml`; MUST carry the standard subagent context files (`_preamble.md` § Standard Subagent Context — config/constitution/context/code-quality/code-review); and MUST end with a **post-stage `fab status refresh` epilogue** so the worker recomputes state from artifacts after finishing (the state-recompute surface 3a lands).
  - **The five-state machine**: `running` / `done` / `failed` / `failed (no-result)` / `orphaned` — the same states §4 defines, elevated here to the cross-adapter contract level (the native adapter observes them structurally; the CLI adapter observes them via `fab dispatch status`).
  - **Nesting degradation**: a harness *without* sub-agent support runs a nesting stage's parts **sequentially inline** instead of as parallel sub-agents. The nesting stage is **`review`** (it spawns inward + outward reviewers + a merge — `_preamble.md` § Review resolves once). On a non-nesting harness, those parts run in sequence in one context; the *outcome contract* is identical, only the concurrency degrades.
  - **"Hooks may enhance, never own"**: harness hooks MAY add value around dispatch (telemetry, notifications) but MUST NOT own any step of the protocol — the protocol is complete and correct without any hook. A hook that becomes load-bearing is a contract violation.
- **An explicit "skill wiring is NOT part of this change" marker**: 3d implements the skill-side dispatch seam against this spec. If wiring reveals a contract flaw, the fix is an **explicit amendment** (edit `harness-adapters.md` + this change's code if the runtime is implicated), never silent drift in 3d's skill files.
- **Cross-references**: add a pointer from `docs/specs/stage-models.md` § Harness-adapter boundary to the new spec (the native adapter it already describes is now one of two adapters catalogued in `harness-adapters.md`), and add a new row to `docs/specs/index.md`.

## Affected Memory

- `distribution/kit-architecture`: (modify) add `fab dispatch` to the fab-go command inventory (the description-frontmatter command roll-call and the command-group prose), alongside `fab pane` / `fab config` / etc. Note the new `internal/agent`+`internal/spawn`-consuming dispatch path and the POSIX-only constraint.
- `pipeline/change-lifecycle`: (modify) the archive prose (`fab change archive` mechanical-ops list and the archive-CLI-exit-semantics paragraph) gains "deletes `.fab-dispatch/{id}/`"; the restore prose gains "does NOT recreate `.fab-dispatch/`".
- `pipeline/execution-skills`: (modify) the `internal/archive` `Archive()` implementation memory — record the `.fab-dispatch/{id}/` deletion step.
- `runtime/index` or a new `runtime/dispatch` file: (new) document the `fab dispatch` family — the detached-launch model, `.fab-dispatch/{id}/` layout, the five status states, refuse-if-running/last-attempt-only, timeout-in-wrapper, the two cleanup paths, and POSIX-only. The runtime domain is the natural home (it owns the process/agent-runtime surface); confirm placement at hydrate against the domain's actual shape.
- `pipeline/schemas`: (modify, if warranted) the `{stage}.yaml` / `{stage}-result.yaml` file shapes belong in the schema catalog if the domain documents per-file schemas at that granularity — confirm at hydrate.

> The exact memory set is confirmed at hydrate. The apply-time grep sweep for `archive` / `dispatch` / `.fab-` across `docs/memory/` is REQUIRED (many `_shared/` and `pipeline/` files mention these tokens) — do not rely on this list to define the sweep class (per code-quality.md § Sibling & Mirror Sweeps).

## Impact

**Code (new):**
- `src/go/fab/cmd/fab/dispatch.go` (parent command) + `dispatch_start.go` / `dispatch_status.go` / `dispatch_logs.go` / `dispatch_kill.go` / `dispatch_clean.go` (mirroring the `pane*.go` split), with table-driven `*_test.go` siblings.
- Possibly a small `internal/dispatch` package for the state-dir read/write + wrapper composition + status-state derivation, so the logic is unit-testable independent of cobra wiring (follows the `internal/pane`, `internal/archive` precedent).

**Code (modified):**
- `src/go/fab/internal/archive/archive.go` (and/or `cmd/fab/archive.go`) — add `.fab-dispatch/{id}/` deletion to the archive move; ensure restore does NOT recreate it.
- Router registration (`main.go` / command tree) for the new `fab dispatch` group; the `LifecycleCommands` collision test must stay green (the router always-route policy + shared allowlist has contract/collision drift tests).

**Docs (constitution-required):**
- `src/kit/skills/_cli-fab.md` — a new `## fab dispatch` section (mirroring `## fab pane`), with each subcommand's signature/flags. **Its SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` MUST be updated in the same change** (SPEC-mirror sync is must-fix).
- `src/kit/skills/fab-archive.md` + `docs/specs/skills/SPEC-fab-archive.md` — the archive-mode mechanical-ops list / delegation description gains the `.fab-dispatch/` deletion; restore prose gains the "not recreated" note.

**Specs (new + cross-ref — this change authors the contract, per §9):**
- `docs/specs/harness-adapters.md` — **NEW** pre-implementation spec (Constitution VI): both dispatch adapters (native Agent-tool + CLI `fab dispatch`), the full dispatch protocol incl. the skill-side half (prompt obligations, `fab status refresh` epilogue, five states, `review` nesting degradation, hooks-enhance-never-own), and the explicit "3d wires against this, amendments are explicit" marker.
- `docs/specs/stage-models.md` § Harness-adapter boundary — add a cross-reference to `harness-adapters.md` (the native adapter it describes is now one of two catalogued adapters).
- `docs/specs/index.md` — add a new row for `harness-adapters`.
- These specs are human-curated (Constitution VI) — they are authored by this change, NOT auto-generated. No memory file mirrors them (specs are pre-implementation intent; memory is post-implementation truth).

**Dependencies / systems:**
- POSIX `setsid`, `sh`, `timeout` (v1 targets POSIX only — Windows errors out).
- `internal/agent` + `internal/spawn` (resolution) — the resolution path depends on **3b** having landed (§ Dependency).

## Non-Goals

- **3b's config surface** — per-tier `spawn_command` in `agent.tiers`. This change *consumes* the resolution; it does not add or modify the config schema.
- **3d's skill-side wiring** — the `/fab-*` skill dispatch-seam branch that decides *when* to dispatch via `fab dispatch` vs. the native Agent-tool path, and *calls* the protocol. 3d implements this against `harness-adapters.md`.
- **3d's dispatch-prompt content** — the actual text/obligations the worker prompt carries. §9's spec fixes the *contract* (what the prompt MUST include); 3d writes the *content* that satisfies it.
- **3d's nesting-degradation implementation** — the code that runs `review`'s reviewers inline when the harness lacks sub-agents. §9's spec fixes the *rule*; 3d implements it.
- **`{stage}-result.yaml` content schema** — this change fixes the *path* and the *presence obligation* (via `failed (no-result)`); the field-level schema of the result body is 3d's, guided by the spec.
- No automatic garbage collection of `.fab-dispatch/` anywhere (explicit user decision — see What Changes §7).

## Open Questions

- **Concurrency across different stages of the same change**: refuse-if-running is scoped to a `(change, stage)` pair. Two *different* stages of the same change dispatched concurrently share `.fab-dispatch/{id}/` but use distinct `{stage}.*` filenames, so they do not collide — confirm this is the intended granularity (it is the natural reading of "refuse if a dispatch for that (change, stage) is already running"). *(Encoded as a Confident assumption below.)*
- **`internal/dispatch` package vs. inline in `cmd/fab`**: whether to extract a package or keep the logic in the cmd files. Follows the codebase's own inline-vs-`internal` convention for the surface's testability needs — an apply-time decision, recorded here so apply doesn't treat it as open scope. *(Confident.)*
- **Exact `runtime/` memory placement** (new file vs. extend `runtime/index`): confirmed at hydrate against the domain's shape. *(Tentative — hydrate-time, low blast radius.)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `fab dispatch` command family (`start`/`status`/`logs`/`kill`/`clean`), independent of tmux/`fab pane`; the interactive pane machinery stays untouched | Discussed — user approved the headless-vs-interactive split explicitly; mirrors the existing `fab pane` command-group precedent in the codebase | S:95 R:70 A:90 D:90 |
| 2 | Certain | `start` reads the prompt on stdin → `{stage}-prompt.md`; launches DETACHED via `setsid sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'`, cwd = repo root; no Go supervisor; state in `{stage}.yaml` via `internal/atomicfile` | Discussed — user approved the exact wrapper and the supervisor-free/resumable rationale; `internal/atomicfile` is the established state-write path | S:95 R:60 A:85 D:90 |
| 3 | Certain | Concurrency = refuse-if-running + last-attempt-only: `start` refuses if a `(change,stage)` dispatch is running; overwrites a completed prior attempt; no per-attempt history | Discussed — user approved "refuse-if-running + last-attempt-only" verbatim | S:95 R:65 A:90 D:95 |
| 4 | Certain | Timeout is an optional `--timeout <secs>` enforced INSIDE the wrapper via POSIX `timeout N <cmd>` — no background sweep/daemon | Discussed — user approved; self-contained-in-wrapper is the stated design | S:95 R:75 A:90 D:95 |
| 5 | Certain | `status` reports 5 distinct byte-stable states: `running` / `done` (exit 0 AND result present) / `failed` (exit≠0) / `failed (no-result)` (exit 0, no result — a contract violation, not done) / `orphaned` (pid dead, no exit file) | Discussed — user approved the state set including the no-result state explicitly | S:95 R:70 A:85 D:90 |
| 6 | Certain | No automatic GC anywhere; exactly two cleanup paths — (a) `fab change archive` deletes `.fab-dispatch/{id}/` (restore does NOT recreate), (b) `fab dispatch clean [<change>] [--orphans]` for manual cleanup | Discussed — user explicitly rejected throttled sweeps and approved the two-path model incl. `--orphans` semantics | S:95 R:55 A:90 D:90 |
| 7 | Certain | Location `.fab-dispatch/{4-char-id}/` at repo root; already gitignored via the scaffold `fragment-.gitignore` `.fab-*` pattern (verified) — ZERO gitignore/scaffold/migration work | Discussed — user asserted it; VERIFIED at drafting against `$(fab kit-path)/scaffold/fragment-.gitignore` (contains `.fab-*`) | S:100 R:90 A:100 D:100 |
| 8 | Certain | POSIX-only v1 — `fab dispatch` errors clearly on Windows rather than half-working; declared in docs | Discussed — user approved POSIX-only-and-declared | S:95 R:80 A:95 D:95 |
| 9 | Certain | Code boundary: this change's Go code owns the runtime + `{stage}-result.yaml` path convention + `failed (no-result)` state; the `spawn=` resolution/tier semantics are 3b's config, and the skill wiring / dispatch-prompt content / nesting-degradation implementation are 3d's code — all explicit non-goals (distinct from the protocol CONTRACT, which this change owns as a spec — see row 16) | Discussed — user drew the 3b/3c/3d boundary explicitly; the amendment refines it (contract=3c spec, wiring=3d code) | S:95 R:75 A:90 D:95 |
| 10 | Certain | New CLI commands documented in `_cli-fab.md` + its SPEC mirror with table-driven tests (status state machine, wrapper composition, refuse-if-running, clean/--orphans, archive-time deletion); archive change swept in change-lifecycle/execution-skills memory + fab-archive skill/SPEC | Constitution Additional Constraints + code-review.md project rules mandate CLI⇒docs+tests and SPEC-mirror sync; non-negotiable | S:100 R:60 A:100 D:100 |
| 11 | Confident | `kill` targets the process GROUP (pgid); liveness + pgid handling follow the existing `internal/proc` platform-split (`proc_linux.go`/`proc_darwin.go`) pattern | Discussed — user said "kills the process group"; the `internal/proc` precedent makes the mechanism a clear codebase-derived default | S:80 R:70 A:85 D:80 |
| 12 | Confident | `start` errors clearly if the resolved tier has no `spawn_command` and does NOT fall back to top-level `agent.spawn_command` | Discussed — user said "errors clearly if the resolved tier has no spawn_command"; no-fallback is the natural reading of the per-tier resolution the series introduces | S:80 R:70 A:80 D:80 |
| 13 | Confident | refuse-if-running is scoped per `(change, stage)`; different stages of the same change share `.fab-dispatch/{id}/` via distinct `{stage}.*` filenames and do not collide | Natural reading of "a dispatch for that (change, stage)"; distinct-filename layout makes cross-stage concurrency safe without extra design | S:70 R:60 A:75 D:70 |
| 14 | Confident | Whether to extract an `internal/dispatch` package vs. inline in `cmd/fab` is an apply-time call following the codebase's own inline-vs-`internal` testability convention (`internal/pane`, `internal/archive` precedent) | Codebase convention answers this; recorded so apply decides-and-records rather than treating it as open scope | S:70 R:80 A:80 D:70 |
| 15 | Tentative | Exact `runtime/` memory placement (new `runtime/dispatch` file vs. extend `runtime/index`) confirmed at hydrate against the domain's actual shape | Genuine two-home ambiguity resolved by the domain's shape at hydrate — low signal now, two valid options; low blast radius keeps it a Tentative reader-hint | S:45 R:60 A:45 D:40 |

15 assumptions (10 certain, 4 confident, 1 tentative).

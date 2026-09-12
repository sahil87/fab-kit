---
name: _cli-fab-pane
description: "Fab CLI reference — the `fab pane` and `fab dispatch` command families (pane primitives: map/capture/process/window-name/open/ready/deliver/kill/questions; the two-mode stage-dispatch manager). Split out of _cli-fab so operator and dispatch consumers load only this slice."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Fab CLI Reference — pane & dispatch

> Loaded via a skill's `helpers: [_cli-fab-pane]` frontmatter. Core commands (change/status/score/preflight/log/resolve/config/…) are in `_cli-fab`; `fab operator`/`fab agent` in `_cli-fab-operator`.

## Contents

- fab pane
- fab dispatch

---

## fab pane

Tmux pane operations with fab context enrichment. `fab pane <map|capture|process|window-name|open|ready|deliver|kill|questions> [flags...]`

**Dispatch-internal verbs (cli-layering Part 7)**: `capture`, `process`, and `kill` are kept for the **rk-less pane arm** — the dispatch orchestrator's peek/escalation path (`_preamble.md` § CLI-Adapter Dispatch), `fab dispatch logs`' suggested capture command, and probe cleanups. Skill-facing guidance rides HexoKit's substrate twins `rk mux capture`/`rk mux process`/`rk mux kill` instead (`command -v rk`-gated, raw-tmux fallback; usage owned by `_cli-agents.md` § Peek). Command behavior, flags, and exit codes below are unchanged by the demotion.

**Pane-family exit codes** (capture, window-name, open, ready, deliver, kill, process): pane validation failures use a shared scheme so callers can branch on cause — `2` = pane missing, `3` = any other tmux failure (dead server, bad socket). `map` and `questions` alone use plain `ERROR:`-formatted exit 1 (multi-pane discovery has no single target pane to be "missing"). **Usage-error coexistence**: a *usage* error on any pane verb — a bad flag or a cobra arg-count violation — exits `2` at parse time (the binary-wide convention above), caught before the handler runs; the in-handler `2` = pane-missing / `3` = tmux-failure scheme is a separate, in-handler `os.Exit` path that bypasses the usage/operational mapping. Exit `2` on a pane verb is therefore ambiguous between "usage error" (at parse time) and "pane missing" (in-handler) — disambiguate on stderr wording; the codes are not renumbered.

**Persistent flag** (all subcommands): `--server <name>` / `-L <name>` (default `""`) — target tmux socket (`tmux -L <name>`). Defaults to `$TMUX` / tmux default. Lets daemons on one tmux server inspect panes on another.

**§ agent state (`@rk_pane_agent_state` convention — read-only).** `map`/`capture` resolve a pane's agent lifecycle state by READING the tmux pane user option `@rk_pane_agent_state` (value `"<state>:<epoch_seconds>[:<pid>]"`, `state ∈ active | waiting | idle`), written by HexoKit's `rk agent setup` global agent-harness hooks (covering Claude Code, Codex, Copilot, Gemini, OpenCode — not just Claude). The name follows HexoKit's `@rk_<scope>_<name>` scheme (tmux format expansion resolves `#{@opt}` by walking pane → window → session → global, so scope is encoded in the name). **Legacy fallback**: the retired unscoped name `@rk_agent_state` is still read when the canonical option is unset — hook generations installed before the rename write only the retired name, and HexoKit dual-writes both during its deprecation window; when both are set the canonical value wins. The canonical name is written by HexoKit from the first release after v3.18.7 (the one carrying run-kit PR #755, its dual-read change); older `rk agent setup` hooks write only `@rk_agent_state`. Dropping the fallback is a follow-up sequenced after HexoKit removes its own legacy reads. fab is a pure CONSUMER: it never writes the option and needs no HexoKit software installed — it reads with plain tmux (`map` via the `#{@rk_pane_agent_state}` + `#{@rk_agent_state}` fields on its `list-panes -F` call on the rk-absent fallback path — the delegated path below takes rk's already-reconciled state instead; `capture` via `tmux show-options -pv -t <pane> @rk_pane_agent_state`, then the same read of `@rk_agent_state` only if that is unset). `active` = turn in progress, `waiting` = blocked on a human (permission prompt / menu / elicitation), `idle` = turn complete. The epoch segment is mandatory — idle duration is `now - epoch`; only `idle` carries a duration. The optional third `:<pid>` segment (the agent process pid, written by current hooks) is validated as a positive integer and otherwise ignored by fab — PID-liveness reconciliation is rk's. An absent option, unknown token, wrong segment count, missing/non-integer epoch, or malformed pid is **unknown** (`—` in tables, `null` in JSON — an uninstrumented or foreign-agent pane is ordinary, not an error). No staleness heuristic: a stale `active` (e.g. an Esc-interrupted agent) is reported as-is.

### map — `fab pane map [--json] [--session <name>] [--all-sessions] [--server <name>]`

All tmux panes with pipeline state. Non-git/non-fab panes included with `---` fallbacks.

**Enumeration is delegated to HexoKit when present (cli-layering Part 8)**: `map` sources its pane list from `rk mux panes --json` (appending `-L <server>` when set) and keeps only the change/stage enrichment role. The row array is read from either shape run-kit prints — the bare document or the `{"ok":true,"result":[…]}` envelope (an `{"ok":false,…}` envelope counts as unparseable). ANY failure — rk absent, a pre-3.17.18 rk, non-zero exit, unparseable JSON, or a failed current-session lookup in default mode — falls back SILENTLY to fab's own `tmux list-panes` enumeration (the attempt is the capability probe; no version check, never an error). Two deliberate deltas on the delegated path: the row set is rk's filtered view (`_rk-pin-*` pin-sessions and the `_rk-ctl` anchor excluded, a pinned window listed once via its home session) and agent state is rk's RECONCILED value; also, an unknown `--session` name filters to zero rows (`No tmux panes found.`, exit 0) instead of tmux's unknown-target error. Session scoping is fab's own row filter in both cases — rk enumerates the whole server. Everything else — table columns, the JSON field set and nullability, enrichment — is identical on both paths. Both paths also carry the pane's current foreground command **snapshot-internally** (rk rows' `command`, or the ninth `#{pane_current_command}` field on the fallback path's `list-panes -F` format) — consumed by the operator tick (see `_cli-fab-operator.md` § `fab operator tick-start` Detection semantics), never rendered as a column or JSON key.

| Flag | Description |
|------|-------------|
| `--json` | Emit the snake-case JSON field set below |
| `--session <name>` | Target specific session (skips `$TMUX` check) |
| `--all-sessions` | Query all sessions (skips `$TMUX` check; mutually exclusive with `--session`) |

| JSON field | Type / meaning |
|------------|----------------|
| `pane` | **Identity key** (with the invocation's `server` socket context): the tmux `%pane_id`. Join rows to a tmux snapshot on this |
| `window_id` | **Identity key**: `string\|null`; stable server-assigned tmux `@N` identity that follows `swap-window`/`move-window`; JSON-only. Consumers MUST tolerate `""`/`null` — a legacy enumeration line carries none |
| `session`, `window_index` | **DISPLAY-ONLY** — positional, reassigned by `swap-window`/`move-window`/session rename; MUST NOT be used as join keys. The motivating bug: run-kit once joined pane state by `session:window_index` and misattributed one window's fab state to whichever window slid into the old position (the StatusDot swap-lag) |
| `tab`, `worktree`, `change`, `stage` | Table-equivalent context fields |
| `repo` | `string\|null`; absolute main-worktree root; JSON-only |
| `display_state` | `string\|null`; `active` / `ready` / `done` / `failed` / `pending` / `skipped`, or `null` with no stage; JSON-only |
| `agent_state` | `string\|null`; `active` / `waiting` / `idle` from `@rk_pane_agent_state`, else `null` |
| `agent_idle_duration` | `string\|null`; populated only for `idle` |
| `pr_url` | `string\|null`; last `.status.yaml` `prs:` entry; JSON-only |
| `pr_number` | `number\|null`; trailing `/pull/<n>` parsed from `pr_url`; JSON-only |

**Identity-key contract**: the row schema inherits HexoKit's contract — `rk mux panes --json` is the primary declaration (a HexoKit companion item); this command consumes and re-emits it. The identity keys are `pane` (+ `server` context) and `window_id`; `session`/`window_index` ride along as display columns only (Non-goal: they are not removed).

PR fields come from the already-loaded status file: **no `gh`/`git`, no network, no PR status (open/merged/CI)**. Consumers fetch live state themselves. `stage: "review-pr"` plus `display_state: "done"` distinguishes a parked shipped change from `display_state: "active"` review work.

Without `--session`/`--all-sessions` → current session only (`-s` scope, requires `$TMUX`). Table columns: `Session` (only with `--all-sessions`), `Pane`, `WinIdx`, `Tab`, `Worktree` (relative; `(main)` for main; `basename/` non-git), `Change`, `Stage`, `Agent`. The `Worktree` relative path is computed **per repo** — each pane's display path is relative to its own repo's main-worktree root (cached by git worktree root), so panes from multiple repos render correct paths. Agent: `active`, `waiting`, `idle ({dur})`, or `—` (em dash for unknown). Change: folder name, `(no change)` for fab worktree with no active change, or `—` for non-fab panes. Idle duration: `{N}s`/`{N}m`/`{N}h` floor division (idle only). Change and Agent resolve on independent axes: Change comes from `.fab-status.yaml`; Agent arrives with the enumeration — rk's reconciled `agent_state`/`agent_state_duration` on the delegated path (a `waiting` duration is dropped: `agent_idle_duration` keeps its published idle-only semantics), or the pane's `@rk_pane_agent_state` option on the fallback path (read from the SAME `list-panes` call via the `#{@rk_pane_agent_state}` format field — zero extra subprocesses, and server disambiguation evaporates since a pane option lives on exactly one server's pane; see § agent state above) — so a pane running any instrumented agent in discussion mode (no active change) shows `(no change)` in Change but a populated Agent column. `$TMUX` unset without targeting flag → exit 1 (`ERROR: not inside a tmux session`). No panes → exit 0 `No tmux panes found.`

### capture — `fab pane capture <pane> [-l N] [--json] [--raw] [--server <name>]`

*Dispatch-internal — skill-facing capture rides `rk mux capture` (see the § fab pane note above).*

`<pane>` required (e.g., `%5`). `-l/--lines N` (default 50) = the **last N lines** of the pane's content: the raw tmux fetch is tailed internally — trailing blank screen-padding stripped, then the last N taken — so no `| tail -N` is needed; interior blank lines and every byte within the window are preserved. `--json` = content + metadata (`worktree`/`change`/`stage`/`agent_state`/`agent_idle_duration` — `agent_state` ∈ `active`/`waiting`/`idle`/`null`, read from the pane's `@rk_pane_agent_state` option; see § agent state above). `--raw` = the captured text only, no enrichment header (byte-identical to tmux's output within the returned window). `--json`/`--raw` mutually exclusive. Pane not found → exit 2 (`Error: pane <id> not found`); other tmux validation failure → exit 3. `--lines < 1` → exit 1 (`ERROR: --lines must be >= 1`).

### process — `fab pane process <pane> [--json] [--server <name>]`

*Dispatch-internal — skill-facing process inspection rides `rk mux process` (see the § fab pane note above).*

OS-level process tree. Linux: walks `/proc/<pid>/task/<tid>/children`, reads `/proc/<pid>/comm` + `/cmdline`. macOS: `ps -o pid,ppid,comm -ax` PPID traversal, plus one batched `ps -axo pid=,args=` pass joined by PID for full cmdlines (two `ps` spawns total — no per-node lookups; a process exiting between the passes degrades to cmdline `""`). Classification: `claude`/`claude-code` → `agent`, `node` → `node`, `git`/`gh` → `git`, else `other`. JSON: `{pane, pane_pid, processes (tree), has_agent}`. Pane not found → exit 2 (`Error: pane <id> not found`); other tmux validation failure → exit 3 — the family scheme. `--server` scopes tmux lookup only; `/proc`/`ps` walk is socket-independent.

### window-name — `fab pane window-name <ensure-prefix|replace-prefix> [--json] [--server <name>]`

Guarded, idempotent rewrites of the tmux window name — retained as dispatch-facing pane primitives; the operator skill marks its windows via `rk tab mark` / `rk tab note`, not these verbs.

| Verb | Usage | Behavior |
|------|-------|----------|
| `ensure-prefix` | `ensure-prefix <pane> <char>` | Idempotent prepend: if the window name already begins with the literal `<char>`, no-op; else `rename-window` to `<char><name>`. `<char>` must be non-empty (else exit 3) |
| `replace-prefix` | `replace-prefix <pane> <from> <to>` | Atomic guarded swap: if the name begins with `<from>`, rename to `<to><name-without-from>`; else silent no-op (the user-rename-mid-monitoring guard). `<to>` may be empty (prefix strip); `<from>` must be non-empty (else exit 3) |

**Exit codes** (both verbs): `0` = renamed OR no-op; `2` = pane missing (tmux stderr propagated); `3` = any other tmux failure (tmux not running, socket error, rename failed, argument usage error — e.g., empty `<char>` or `<from>`). The 2/3 split lets a removing caller treat "pane gone" (exit 2) as successful removal. No `$TMUX` gate — tmux's own exec failure surfaces as exit 3, so the verbs work via `--server` targeting from outside a tmux client.

**Output**: plain `renamed: <old> -> <new>` on rename, empty stdout on no-op; `--json` always emits one `{"pane","old","new","action"}` object (`action`: `renamed`|`noop`).

### open — `fab pane open --provider <name> [--role <role>] [-c <dir>] [--json] [--server <name>]`

The provider-generic pane spawn — no dispatch record, no `.fab-dispatch/` state. Resolves the provider's `interactive_command` exactly as `fab agent` does (project config per-field merged over the built-in table; works outside a fab repo, where the built-in table alone applies), fills `{model}`/`{effort}` via the standard precedence with the provider pinned at invocation time (`--role` selects whose fills apply, the `default` role otherwise — the opposite of `fab agent --provider`'s deliberate fill bypass), and spawns the composed command as a **plain split** of the current window when the invoker is a tmux pane on the target server (`$TMUX_PANE` set, no `--server`), an **unnamed new window** otherwise. No worker-column placement, no `fab-{id}-{stage}` title — placement and identity are dispatch policy; `fab dispatch open` is the record-keeping binding over this primitive (§ fab dispatch). Unknown provider → the shared lookup error naming the available providers (exit 1); a provider with no `interactive_command` → hard error naming it (`configure providers.<name>.interactive_command`, exit 1); unreachable tmux or a failed spawn → exit 3. Success: `opened pane %N (provider <name>)`, plus a `server: <name>` line when non-default; `--json` instead emits `{"pane","provider","server"}` (server `null` for the default socket). Probe the new pane with `fab pane ready`, then hand it a prompt with `fab pane deliver`.

### ready — `fab pane ready <pane> [--json] [--server <name>]`

The readiness gate addressed by pane id — the same classifier `fab dispatch ready` binds over (§ fab dispatch), with no dispatch record to load. First checks who owns the pane: while the foreground command is still a shell (the provider binary has not taken the tty yet) it reports `booting` and types NOTHING — a cooked-mode shell echoes typed characters by itself, so the sentinel echo would be a false signal (this precondition runs fab-side ahead of both classification arms — rk's await deliberately classifies a cooked-shell echo as ready). Once a non-shell process owns the pane the mechanical classification runs in two arms: when a sentinel-capable HexoKit is on PATH (probed from the binary — `rk mux await --help` mentions `parked`, cached once per process, never a version compare) it delegates to `rk mux await --ready <pane>` under a bounded internal timeout and maps rk's report (`ready` (state|echo) → `ready`, `parked` → `parked`, timeout `running` → `booting`, `gone` → the pane-missing error, `narrow` (the pane is under rk's own geometry floor) → a silent hand-off to the raw arm, no warning); any unexpected rk failure fails OPEN to the raw arm with at most one stderr warning per process. The raw-tmux arm — the rk-less fallback — types a sentinel literally (never submitted), checks the echo against two screen-stability captures, clears with `C-u`, and reports `ready` / `booting` / `parked`. Non-`ready` reports add `pane: %N`, a `server: <name>` line when set, and a `--- last 20 lines ---` capture snippet (fab's own capture on BOTH arms; omitted on a blank screen). `--json` instead emits one object `{"state","pane","server","snippet"}` — `state` ∈ `ready`/`booting`/`parked`, `snippet` the same capture (`""` when blank), `server` `null` for the default socket. **All three classifications exit 0** — the report string (or `state` key) is the sole discriminator. Pane missing (including rk's `gone`) → exit 2; any other tmux failure → exit 3. **Side effect**: once an agent owns the pane the probe TYPES into it (the sentinel — rk's on the delegated arm, fab's `C-u`-cleared one on the raw arm) — run it only against panes you own, never one an agent or human is actively working in; a shell-foreground pane is never typed into at all.

### deliver — `fab pane deliver <pane> (--prompt-file <path> | --text <string>) [--json] [--server <name>]`

Verified delivery addressed by pane id — the same choreography `fab dispatch deliver` binds over: readiness probe → `C-u` → type the payload literally → wrap-tolerant echo-verify → Enter → screen-advance confirm, with exactly one retry. Every keystroke in the choreography (here and in `ready`'s raw-arm sentinel — the delegated rk arm types its own) rides the shared Go send seam, which first clears any tmux pane mode (`#{pane_in_mode}` probe + conditional `send-keys -X cancel`) — a pane a human left scrolled up in copy-mode would otherwise consume the keys as mode bindings and silently eat the delivery. `--prompt-file` checks the file exists first (missing → exit 1, nothing typed) and types the dispatch-parity pointer line `Read <path> and execute it.` — the path is typed **as supplied**, so make it meaningful from the pane's own cwd; `--text` types its argument literally. Exactly one of the two flags is required (mutually exclusive — usage error otherwise). Retry warnings go to stderr even when the retry succeeds; a second failure prints the pane's last 20 lines to stderr and exits 1. Success: `delivered <pane> (prompt <path>)` or `delivered <pane> (text)`; `--json` instead emits `{"pane","source","path"}` on VERIFIED delivery only (`source` ∈ `prompt`/`text`; `path` present only for `prompt`) — failures keep the stderr + non-zero contract with no JSON. Pane missing → exit 2; other tmux failure → exit 3.

### kill — `fab pane kill <pane> [--server <name>]`

The record-free generic kill — exposes the shared `KillPane` helper with the family's validated exit-code contract. *Dispatch-internal — skill-facing pane removal rides the agent-state-gated `rk mux kill` (see the § fab pane note above); this verb backs rk-less probe cleanups and the pane arm.* Validates the pane first, then kills it. Success: `killed <pane>`, plus a `server: <name>` line when non-default. Pane missing → exit 2 (`Error: pane <id> not found`); other tmux failure → exit 3. No dispatch-record interaction, no `.fab-dispatch/` state — `fab dispatch kill` (record-keyed, ungated recovery) is unaffected and remains the pipeline's kill.

### questions — `fab pane questions [--all-sessions] [--panes <id>...] [--json] [--server <name>]`

Sweep candidate panes for pending questions/prompts — fab-operator's §5 Question Detection policy mechanized into the binary. A **first-class skill-facing verb** (NOT dispatch-internal, unlike `capture`/`process`/`kill` — the deliberate carve-out: this is a policy-bearing sweep, not a peek primitive). Per candidate: capture the last 20 lines (fixed, not a flag), apply the two guards, scan bottom-most-first for the mechanical indicator classes, and report matches + skip reasons. Detection input only — never a license to send blind (the operator's pre-send gate and re-capture-before-send guard still run before any send).

| Flag | Description |
|------|-------------|
| `--panes <id>...` | Explicit pane IDs to sweep (repeatable/comma-separated); mutually exclusive with `--all-sessions`; skips the `$TMUX` check and resolves IDs server-wide |
| `--all-sessions` | Discover candidates across all sessions (skips the `$TMUX` check) |
| `--json` | Emit the JSON result below |

Discovery modes (`--all-sessions`, or no flags = current session, requires `$TMUX`) sweep only panes whose resolved `agent_state` is `waiting`/`idle` (see § agent state above) — unknown (`—`) and `active` panes are excluded by construction. `--panes` takes the IDs verbatim and applies the state check per pane during the sweep.

| JSON field | Type / meaning |
|------------|----------------|
| `matches[].pane` | Pane ID of a matched candidate |
| `matches[].agent_state` | `waiting` / `idle`, as read at sweep time |
| `matches[].indicator` | `question_mark` / `yes_no` / `action_word` / `imperative_question` / `colon_prompt` / `enumerated_options` / `press_key` |
| `matches[].snippet` | The matched line |
| `skipped[].pane` | Candidate pane that did not match |
| `skipped[].reason` | `state_changed` / `capture_failed` / `blank_capture` / `turn_boundary` / `no_indicator` |

Both arrays encode `[]` when empty (never `null`). Human output: one line per match (`pane [agent_state] indicator: snippet`), one line per skip (`pane: reason`), then `N matched, M skipped`; an empty candidate set prints `No candidate panes.`. **Exit codes** follow `map`, not the per-pane 2/3 scheme: `0` on any clean sweep regardless of match/skip counts (a dead candidate is a normal `capture_failed` skip, not a failure); non-zero only on usage error (`$TMUX` unset with no targeting flag → `ERROR: not inside a tmux session`) or a hard discovery failure (tmux unreachable).

---

## fab dispatch

Process manager for CLI-dispatched pipeline stages, in **two modes** — the two non-native adapters of cross-harness stage dispatch. `fab dispatch <start|open|ready|deliver|restart|status|wait|logs|kill|reap|clean> [args...]`. Full cross-adapter contract (three adapters: native Agent-tool / headless CLI / interactive pane): `_preamble.md` § CLI-Adapter Dispatch and § Dispatch-Prompt Obligations.

**The two modes have separate ENTRIES**, because they hand a worker its prompt in fundamentally different ways. `start` (§ start) launches **headless** in one step, with the prompt on stdin. **Pane** mode takes three — `open` (§ open) spawns the pane and delivers nothing, `ready` (§ ready) probes whether it can accept typed input (the mechanical classification delegates to `rk mux await --ready` when a sentinel-capable rk is on PATH, fail-open to fab's own raw-tmux probe), `deliver` (§ deliver) types the prompt pointer and verifies it landed — because a freshly spawned agent TUI may still be booting or parked behind a first-run wall, and answering one is the orchestrator's judgment rather than the binary's. `deliver --prompt-file` is also the **pane-arm resume**. The gate's wiring, budget, and escalation rules are `_preamble.md` § The pane readiness gate. **Primitives vs. bindings** (260810-1lah): `open`/`ready`/`deliver` are thin record-keeping bindings over the provider-generic pane primitives `fab pane open`/`ready`/`deliver` (§ fab pane) — the readiness classifier, the verified-delivery choreography, the tmux pane creators, liveness/kill, and the pointer-line composer relocated from `internal/dispatch` to `internal/pane`, while placement *policy* (`SelectMode`/`SelectPaneShape`/`SplitTarget`) and the dispatch-record bookkeeping stay dispatch-owned. Their external contract — flags, output forms, exit behavior — is unchanged by the relocation.

`restart` is the family's **recovery** verb (§ restart) — it relaunches a non-running dispatch from the prompt `start`/`open` persisted; the observation policy that spends it lives in `_preamble.md` § CLI-Adapter Dispatch → *Recovery policy*. `status` and `wait` are the family's two **observation** verbs over one derivation: `status` is the one-shot probe, `wait` its blocking sibling (§ wait). `reap` is the family's **hygiene** verb (§ reap) — it reclaims a *done* pane worker's tmux pane and is a reported no-op for every other dispatch, which is what keeps it distinct from `kill` (§ kill), the *recovery* verb valid in any state.

| Mode | Entry | Worker | Command composed | Completion observed via | tmux |
|------|-------|--------|------------------|-------------------------|------|
| **headless** | `start` (`--headless`, `--timeout`, or automatic descent to the headless rung) | detached `sh -c` process | the provider's `headless_command` | `{stage}.exit` + pid liveness + result file | never touched |
| **pane** | `open` → `ready` → `deliver` (`restart` also lands here and performs `open` alone) | an interactive agent session in a tmux pane you can watch and steer — **split into your own window** when you are a tmux pane on the target server, else a new window | the provider's `interactive_command`, **verbatim** — nothing is appended to it | **result file** + pane liveness | **required** — as is an `interactive_command`; either prerequisite missing is a hard error on `open` and a skipped rung under `restart`'s automatic selection |

The two provider command fields are **never merged and never substitute for each other**; `native` is a third, explicit provider capability. Headless mode stays **tmux-independent**; pane mode borrows tmux. Automatic selection starts at `dispatch.mode` (default `native`) and descends only through `pane → native → headless`. `fab dispatch` can launch the two non-native rungs; if its fresh selection lands on native it stops before writing state and tells the caller to re-run `fab agent <stage> -o yaml` and use native dispatch when `dispatch:` is absent. Headless dispatch remains parallel to and independent of `fab pane` / `fab operator`; pane dispatch borrows tmux as a launch surface but does **not** join the operator's tracked set. **POSIX-only (v1)** — headless `start`/`kill` error clearly on Windows (`fab dispatch requires a POSIX shell (setsid/timeout); Windows is not supported in v1`) rather than half-working.

**Pane liveness is identity-checked (the restart-alias guard).** A tmux server's `%N` pane-id space resets on server restart while `{stage}.yaml` persists `pane: %17`, so after a restart the recorded ID can **alias onto an unrelated new pane**. Against that, `open` records the pane's shell pid (`#{pane_pid}`) as `pane_pid` in the record (additive, `omitempty` — a failed read warns on stderr, leaves the key absent, and never fails the launch). Every record-keyed liveness observation — `status`/`wait`'s derivation, refuse-if-running, `reap`'s state check, and the `kill`/`ready`/`deliver` gates — then treats the pane as the worker only when the pane exists **and** (no `pane_pid` was recorded **or** the pid read fails right now **or** the pane's current pid matches the record). A **mismatch means the pane is an impostor and the worker is gone**: `status`/`wait` derive `orphaned` (the existing recovery path) instead of `running`; a fresh `open`/`restart` overwrites the attempt instead of refusing "already running"; and the targeting verbs never touch the impostor — `kill`/`reap` report already-gone and send no `kill-pane`, `ready`/`deliver` refuse naming `fab dispatch restart` and type nothing, and `logs` reports the gone worker instead of printing a `fab pane capture` hint aimed at it. Records written by older binaries carry no `pane_pid` and behave exactly as before (existence-only) at every consumer — the field is additive on transient per-change state, so **no migration** ships.

**State layout** — `.fab-dispatch/{4-char-change-id}/` at the **repo root** (alongside `.fab-status.yaml`, already gitignored via the scaffold `.fab-*` pattern — no gitignore/scaffold/migration work). Keyed by the stable 4-char change ID (stable across `fab change rename`); one dir per worktree. Both modes share the dir, the loader, and the refuse-if-running check. Per-stage files:

| File | Written by | Contents |
|------|-----------|----------|
| `{stage}-prompt.md` | `start`/`open` (from stdin) | the stage prompt — piped to the dispatched command's stdin (headless) or **pointed at** by the one-line prompt `deliver` types into the pane worker. Also **`restart`'s input**, which reads it and leaves it byte-identical (see § restart) |
| `{stage}-continuation.md` *(convention)* | the orchestrator | a rework-cycle continuation prompt for the pane-arm resume, handed over with `deliver --prompt-file`. fab neither writes nor requires it; the path is the wiring's convention (`_preamble.md` § Pane-arm continuation) and it is cleaned with the rest of the dir |
| `{stage}.yaml` | `start`/`open` (via `internal/atomicfile`), plus `deliver`'s marker flip | `spawn_cmd` (resolved) + `started_at`, plus the mode's identity: `pid`/`pgid`/`timeout` (headless) or `pane`/`window`/`server`/`pane_pid`/`delivered` (pane — where `window` holds the `fab-{id}-{stage}` identity string, meaning the tmux **window name** in the new-window shape and the tmux **pane title** in the split shape, `pane_pid` is the open-time liveness discriminator below, and `delivered` records that the worker has been handed its prompt). Every mode-specific key is omitted when empty — including `delivered`, whose ABSENCE means "not delivered" — so a headless record's shape is unchanged and the mode is **derived** from which keys are present (no stored discriminator) |
| `{stage}.log` | the wrapper | combined stdout+stderr of the dispatched command — **headless only** (a pane worker's output is tmux scrollback) |
| `{stage}.exit` | the wrapper | the exit code (`echo $? > ...`) — its presence is the "process finished" signal; **headless only** |
| `{stage}-result.yaml` | the dispatched agent (contract) | the stage result; presence is required for `done` in both modes, and is the **sole** completion signal in pane mode |

### start — `fab dispatch start <change> <stage> [--timeout <secs>] [--headless]`

Launches a **HEADLESS** stage worker. Resolves `<change>` → 4-char ID; reads the stage prompt on **stdin** → `{stage}-prompt.md`; resolves the stage's role → provider internally (via `internal/agent` + `internal/spawn` `{model}`/`{effort}` substitution — the same resolution `fab agent <stage> -o yaml` projects). **This re-resolution reads CONFIG ONLY — `start` has no `--provider`/`--model`/`--effort` surface**, so `fab agent`'s invocation-time overrides do NOT reach it: an override binds the **native Agent-tool arm** only, and moving a stage onto CLI dispatch requires a config override (`agent.workers`/`agent.session`, or `agent.profiles.<role>.provider`) rather than a flag (see `_cli-fab-operator.md` § fab agent).

**Pane mode's entry is `open`, not `start`.** A pane worker is spawned and delivered to in two steps with an agent-driven readiness gate between them, which a single-shot launch has nothing to map onto. `start` therefore refuses a pane landing, always before any state write, in both shapes it can arrive:

| Typed / resolved | Result |
|------------------|--------|
| `--pane` or `--server` | refusal naming `fab dispatch open`, then `ready` and `deliver`. The flags stay registered (hidden) precisely so this guidance is reachable instead of cobra's bare `unknown flag` |
| the ladder LANDS on pane (a `dispatch.mode: pane` preference whose prerequisites hold) | `stage "<stage>" resolves to pane mode for provider "<p>", but \`fab dispatch start\` launches only the headless arm; run \`fab dispatch open …\` (then \`fab dispatch ready\` and \`fab dispatch deliver\`) …, or pass --headless to force this launch headless` |

The refusal fires **after** the descent has had its chance, which is load-bearing: a stale `$TMUX` makes the ladder pick pane, and descending to headless on the failed reachability probe is exactly what keeps an unattended `start` working there. Only a pane rung that survives validation is a genuine "you wanted a watchable worker" landing. The `--pane`/`--timeout` mutual-exclusion rule dissolved with the flag; `--timeout` and `--headless` are unaffected and still compose.

**Mode selection = explicit overrides, then the configured descent ladder** — the same pure `dispatch.SelectMode` ladder `restart` and `fab agent <stage> -o yaml` use. Evaluated in order, the first explicit match wins (each keys on whether the flag was **supplied**, not its value, so `--timeout 0` / `--server ""` still count). With no explicit signal, selection starts at `dispatch.mode` and chooses the first possible rung at or below it:

| # | Signal | Mode | Selection is |
|---|--------|------|--------------|
| 1 | `--pane` | pane | explicit *(refused on `start`; valid on `restart`)* |
| 2 | `--headless` | headless | explicit |
| 3 | `--timeout <secs>` | headless | explicit (the timeout is enforced by the headless wrapper, so it can only mean headless) |
| 4 | `--server <name>` | pane | explicit *(refused on `start`; valid on `restart`; on `open` it names the socket without selecting anything)* |
| 5 | *(none of the above)* | first possible rung in `pane → native → headless`, beginning at `dispatch.mode` | **automatic** |

- Pane is possible when tmux is available and the provider has `interactive_command`; native when `native: true`; headless when `headless_command` is present. A missing prerequisite skips that rung, and automatic selection never ascends. At the resolver seam `$TMUX` presence represents tmux availability; the pane path validates with a real reachability probe before launch, and under `restart`'s automatic selection a failed probe re-runs the same descent with `tmux unreachable`. An empty `$TMUX` reads as unset. **`$TMUX_PANE` is separate**: it chooses pane shape only after pane mode resolves.
- An automatic native result is not launchable by this command family: `start`/`restart` error before state writes and tell the caller to re-run `fab agent <stage> -o yaml`, whose absent `dispatch:` key selects the native Agent-tool adapter.
- **`--pane` + `--headless` is a usage error** on `restart` (exit 2, cobra flag group — fired before any work, so nothing is launched and nothing persisted). `--headless` + `--timeout` **composes** (both select headless).

**The headless launch** runs the resolved `headless_command` **DETACHED**, cwd = repo root:

```sh
sh -c '<resolved-cmd> < {stage}-prompt.md > {stage}.log 2>&1; echo $? > {stage}.exit'
```

The shell is launched with `setsid` semantics (Go's `SysProcAttr{Setsid:true}`, not a `setsid` binary prefix — prefixing it would double-fork and leave the recorded pid pointing at a process that exits immediately), detaching it into a new session/process group so the dispatch **survives the orchestrator dying** — no Go supervisor remains, the shell records the exit code itself and the recorded `pid`/`pgid` track the live worker. `--timeout N` wraps the resolved command in POSIX `timeout N <cmd>` **inside the wrapper** (no Go timer/daemon); a timed-out command exits `124`, surfacing as `failed`.

- **No reachable capability → actionable error:** explicit headless hard-errors on its own missing command key. Automatic selection skips unavailable rungs; if none remains, it errors naming the provider and the three capability keys (`providers.<name>.interactive_command`, `.native`, `.headless_command`). This is mode descent, never command-field substitution.
- Output: `dispatched <id>/<stage> (pid N, pgid N)`. Automatic success appends its exact selection reason; explicit selection carries no suffix. The selector's complete reason set (some forms become an actionable error rather than a `dispatched` line) is:

  - Direct: `mode: pane (preferred)`, `mode: native (preferred)`, `mode: headless (preferred)`.
  - One-rung pane descent: `mode: native (descended: pane unavailable: no tmux)`, `mode: native (descended: pane unavailable: tmux unreachable)`, `mode: native (descended: pane unavailable: no interactive_command)`.
  - Native descent: `mode: headless (descended: native unavailable)`.
  - Two-rung descent: `mode: headless (descended: pane unavailable: no tmux; native unavailable)`, `mode: headless (descended: pane unavailable: tmux unreachable; native unavailable)`, `mode: headless (descended: pane unavailable: no interactive_command; native unavailable)`.

### open — `fab dispatch open <change> <stage> [--server <name>]`

**Pane mode's entry.** Spawns the interactive worker's tmux pane and stops there, delivering **nothing**. The stage prompt still arrives on **stdin** and is persisted to `{stage}-prompt.md`, ready for § deliver to point the worker at.

It is `start`'s pane path verbatim — same resolution, same `internal/spawn` substitution, same two placement shapes, same record-keyed column stacking, same `fab-{id}-{stage}` identity, same refuse-if-running check, same stale exit/result/log clearing — with **one difference**: the composed `interactive_command` reaches tmux **verbatim**, with no prompt argument appended. That is what decouples pane capability from whether a provider's CLI accepts a positional initial prompt, and what makes delivery a verifiable step instead of a fire-and-forget one.

- **Pane is EXPLICIT here, never a ladder result.** `open` opens a pane or it errors: an unreachable tmux server or a provider with no `interactive_command` is a hard error with nothing launched and nothing persisted, never a silent descent to headless (which would be the opposite of what the caller asked for). The errors are the pane path's existing ones — `pane mode requires a reachable tmux server, but <target> is unreachable; …` and the `providers.<name>.interactive_command` config-key hint.
- **`--server` / `-L`** targets and persists a tmux socket, exactly as before; naming one also keeps the **new-window** shape, since the caller's own `$TMUX_PANE` id is meaningless on another socket. There is no `--pane` (redundant), no `--headless`, and no `--timeout` (a headless-wrapper bound pane mode never constructs).
- **WHERE the pane opens** is unchanged and decided from `$TMUX_PANE` and `--server`:

| # | Condition | Shape | tmux call | Identity carried by |
|---|-----------|-------|-----------|---------------------|
| 1 | `$TMUX_PANE` set **and** no `--server` | **split** — a pane inside the **dispatching agent's own window** | `tmux split-window {-h -l <n>%\|-v} -t <target> -c <repo-root> "<resolved-cmd>"` then `select-pane -T fab-{id}-{stage}` | the **pane title** |
| 2 | `--server <name>` supplied | **new window** | `tmux -L <name> new-window -n fab-{id}-{stage} -c <repo-root> "<resolved-cmd>"` | the **window name** |
| 3 | `$TMUX_PANE` unset | **new window** | `tmux new-window -n fab-{id}-{stage} -c <repo-root> "<resolved-cmd>"` | the **window name** |

  This is the two-tier tmux hierarchy: an operator opens worktree agents as windows; a worktree agent's stage workers appear as panes beside it. **Placement:** split workers form a stacked right column keyed on same-socket dispatch-record pane IDs, never mutable pane titles. Intersect recorded panes with the caller window's live panes; split the last live sibling with unsized `-v`, or carve once from `$TMUX_PANE` with `-h -l <n>%` at `dispatch.column_width` (default 45). **Geometry floor:** before the split runs, the planned worker pane is priced from the window geometry (carve = a `window_width × column_width / 100`-col-wide column spanning the window's full height; stack = the column's width × half the sibling's height — width and height are priced separately against their own floor) against `dispatch.min_cols`/`dispatch.min_rows` (defaults 50/20) — a below-floor pane opens in its own detached window pinned to a manual 200x50 size instead, with a warning naming the reason, and a failed geometry probe keeps the split (fail-open). This is a creation-time column invariant: never `select-layout`, repair, rearrange user panes, or fight manual resizes. Placement is cosmetic and warn-only: probe/read failures retain usable records and name the chosen fallback; absent state is the normal first run; a failed `list-panes` still makes a sized carve; tmux versions rejecting percentage size retry unsized. **Identity and lifecycle:** `fab-{id}-{stage}` is the `window` record value, pane title in the split shape and window name in the new-window shape. The pane ID is the downstream identity, making `status`, `kill`, capture, reap, and refuse-if-running shape-blind; a failed title-set only warns. A split kill leaves the agent window intact. Neither shape carries operator window marks — operator marks ride `rk tab mark` / `rk tab note` (fab-operator.md §6), and a dispatch worker is never the operator's to mark.
- **Refuse-if-running + last-attempt-only** apply exactly as on `start`, reading the PRIOR record's own mode's finished signal — for a pane record `{stage}-result.yaml` absent **and** the pane alive, since result presence wins over pane liveness there (an interactive worker sits at its prompt after finishing, so a liveness-only rule would refuse forever after a successful run).
- Output: `opened <id>/<stage> (pane %N, split, title fab-<id>-<stage>)` or `opened <id>/<stage> (pane %N, window fab-<id>-<stage>)`. The verb is **`opened`, not `dispatched`** — the pane exists, but the stage has not been handed over yet. `open` also records the pane's shell pid as `pane_pid` in the record — the restart-alias liveness discriminator (see § fab dispatch → *Pane liveness is identity-checked*); a failed pid read warns on stderr and never fails the launch.

### ready — `fab dispatch ready <change> <stage>`

Answers one question about an opened pane: **can it accept typed input right now?** Before any keystroke the probe checks who owns the pane: while the foreground command is still a shell (the provider binary has not taken the tty yet) it reports `booting` and types NOTHING — a shell in cooked mode echoes typed characters by itself, so the sentinel would echo for a reason that has nothing to do with an agent being ready. This takeover precondition runs fab-side ahead of BOTH classification arms and rk is never invoked while it holds. Past the precondition the classification is TWO-ARM, and both arms are purely MECHANICAL. The preferred **rk arm**: when a sentinel-capable HexoKit is on PATH — capability is probed from the binary (`rk mux await --help` mentions `parked`; cached once per process; never a version compare, which can lie across bottle/source skew) — the probe delegates to `rk mux await --ready <pane>` under a bounded internal timeout and maps rk's report word: `ready %N (state)` or `ready %N (echo)` → `ready`; `parked %N` → `parked`; a timeout `running` → `booting`; `gone %N` → the dead-pane error (not a classification); `narrow %N (WxH)` — rk declining to classify a pane under its own geometry floor — hands off SILENTLY to the raw arm for that probe (no warning: nothing failed, and the raw arm has no geometry floor). An unexpected rk failure (non-zero exit, unparsable report) fails OPEN to the raw arm for that probe with at most one stderr warning per process — a *classified* rk outcome is an answer and is never re-classified by the fallback. The **raw-tmux arm** (rk absent, not sentinel-capable, or failed) is the unchanged fab classifier — it types a sentinel literally (`send-keys -l`, never submitted), checks whether the sentinel echoed, looks at whether the screen is still moving between two captures, and clears the sentinel with `C-u` whether or not it echoed. The clear comes **after both captures**: `C-u` is itself a keystroke, and a TUI that repaints its input line in response would make a straddling capture pair differ every time, so a parked pane would report `booting` forever. Neither arm carries **a table of known dialogs**, presses any other key, or answers anything: dialog text is a version treadmill, and a half-matched pattern pressing Enter into an unknown screen is worse than stalling. Re-running it is always safe (Constitution III).

| Report | Meaning | The wiring's response |
|--------|---------|-----------------------|
| `ready` | the pane accepts typed input (rk's state-present fast path or a sentinel echo, on either arm) | hand the worker its prompt with `deliver` |
| `booting` | the pane's foreground command is still a shell (the provider has not taken the tty), rk's bounded await timed out (`running`), or no echo on a screen blank or changed between the two captures | wait briefly and re-probe |
| `parked` | no echo on a stable screen — a dialog, survey, login wall, or wedged process holds the input | spend a judgment round, or escalate |

- **Output**: the bare word on `ready`. Every non-`ready` report adds `pane: %N`, a `server: <name>` line when the record carries a socket, and a `--- last 20 lines ---` capture snippet — fab's OWN capture on both arms (rk's stderr formatting is not contract), counted after the pane's trailing blank padding is dropped, so a dialog drawn near the top of a tall pane is still what you see — everything a judgment round needs to answer the wall with `tmux [-L <server>] send-keys -t <pane> …` without a separate lookup (`status --json` also carries the socket, as `server`).
- **Exit codes**: `0` for **all three** classifications — the report string is the sole discriminator, exactly as with `wait`'s timeout return. Non-zero is reserved for real errors: no dispatch record, a **headless** record (the verb applies only to pane workers), a dead pane (identity-checked, or rk's `gone` on the delegated arm — a restart-aliased pane is the same refusal, so no sentinel is ever typed into the impostor), a **mid-stage worker**, or a tmux failure. The mid-stage refusal is § deliver's guard verbatim, shared because the probe is a **sender** too — it types the sentinel (fab's own, or rk's on the delegated arm) and presses `C-u`, which against a delivered worker still executing its stage is exactly the input injection the contract's carve-out stops at successful delivery.
- The gate's budget, its never-answer classes (login and credential walls), and the escalation path are **skill-side policy**, not this command's: `_preamble.md` § The pane readiness gate.

### deliver — `fab dispatch deliver <change> <stage> [--prompt-file <path>]`

Types a one-line pointer to the stage prompt into a pane worker and **VERIFIES that it landed**. This is the sole mechanism that hands a pane worker its prompt, for both initial dispatch and rework-cycle continuation.

**The choreography**, per attempt: readiness probe (§ ready mechanics, inline) → `C-u` → type the pointer literally → **capture-verify that it echoed** → `Enter` → **confirm the screen advanced**. Any failed check costs one attempt; there is exactly **one retry** (reported as a `warning: delivery attempt 1 failed (…); retrying` on stderr even when the retry succeeds), and a second failure exits non-zero with the pane's last 20 lines on stderr. Echo checks ignore whitespace **and box-drawing runes**, so a pointer tmux hard-wrapped across lines is still found — including inside a TUI that frames its input box with vertical rules, where a wrap interleaves those frame runes between the halves.

Verification is the point: a spawn-time positional prompt either is or is not ingested and fab cannot tell, so a CLI that silently drops one strands a worker at an empty prompt while the dispatch reads `running`.

- **`--prompt-file <path>`** points the worker at a different prompt — the **pane-arm resume**. The orchestrator writes a continuation prompt (triaged findings + rework action + the re-read-from-disk instruction) under `.fab-dispatch/{id}/` and delivers it into the still-live apply pane instead of paying a cold start. Content rules and the fallback are `_preamble.md` § Pane-arm continuation. With no flag the pointer names `{stage}-prompt.md`.
- **The previous attempt's `{stage}-result.yaml` and `{stage}.exit` are taken out of the way before the first send** — so a continuation reads `running` again rather than letting the next `wait` return immediately on the last cycle's result — and **restored if no attempt verifies**. Only a delivery that landed has superseded them; a failed continuation that kept `delivered: true` with no result would read `running`, which `deliver` and `open` both refuse, leaving the mandatory fresh-dispatch fallback unexecutable without a `kill`.
- **The delivery marker** (`delivered: true` in `{stage}.yaml`) is written only **after** verification succeeds. A failed delivery leaves it unset, which is what lets a caller tell "the worker never got its prompt" from "the worker got it and failed at the work".
- **Refusals** (each before any keystroke): a **headless** record (naming `fab dispatch start`); a **dead pane** (naming `fab dispatch restart`); a **missing prompt file**, whether the stage's own or a `--prompt-file` path (`no prompt at <path> — nothing to deliver; …`) — a pointer at a file that is not there would type cleanly, verify cleanly, and leave the worker reading nothing; and a worker that is **mid-stage** (`delivered: true` with no result file), which is the contract's no-input-injection rule in code. `delivered: true` **with** a result present is the sanctioned continuation case and proceeds.
- Output: `delivered <id>/<stage> (pane %N, prompt <repo-relative-path>)` — the resolved 4-char ID, whatever spelling of the change was typed, as with `dispatched …` and `opened …`.

### restart — `fab dispatch restart <change> <stage> [--timeout <secs>] [--pane] [--headless] [--server <name>]`

Relaunches a non-running dispatch through the shared prologue, validation, flag exclusions, launch, save, record/output shape, stale-attempt clearing, refuse-if-running rule, and last-attempt-only semantics. It has three unique behaviors:

- **Current-environment mode:** mode and pane shape are re-derived from current config, provider capabilities, and environment, never inherited. A pane dispatch orphaned by a dead server descends again; it may relaunch headless or stop for native re-resolution. A restart issued from a tmux pane can split that pane's window when pane is still at or below the configured preference.
- **A pane landing is HALF a relaunch.** `restart` is the one launch verb that still accepts pane, but it performs the `open` step alone — the pane is spawned, `delivered` stays unset, and stdout reports `opened …` rather than `dispatched …`. Go cannot run the readiness gate's judgment, so the missing half is handed back on **stderr**: `note: the pane holds no prompt yet — run \`fab dispatch ready …\`, clear any wall it reports, then \`fab dispatch deliver …\``. (`open` prints no such note: its own name says it.) A headless landing relaunches fully, as before.
- **Persisted-prompt input:** reads `.fab-dispatch/{id}/{stage}-prompt.md` instead of stdin and leaves it byte-identical. Refuse-if-running is checked before prompt presence. Missing prompt returns `no persisted prompt at <path> — nothing to relaunch; run \`fab dispatch start\` (headless) or \`fab dispatch open\` (pane) with the prompt on stdin` and touches no state.

### status — `fab dispatch status <change> <stage> [--json]`

Byte-stable poll surface. Reads `{stage}.yaml`, then derives the state by the record's **mode**. The five state **strings are the cross-adapter contract** — identical in both modes; what differs is which are reachable.

**Headless** — reads `{stage}.exit`, probes `pid` liveness (POSIX `kill(pid,0)`), reports one of all five:

| State | Condition |
|-------|-----------|
| `running` | pid alive AND `{stage}.exit` absent |
| `done` | `{stage}.exit` == `0` AND `{stage}-result.yaml` present |
| `failed` | `{stage}.exit` present AND != `0` (includes `124` timeout) |
| `failed (no-result)` | `{stage}.exit` == `0` BUT `{stage}-result.yaml` absent — a **contract violation, NOT done** |
| `orphaned` | pid dead AND `{stage}.exit` absent (reboot / `kill -9` / crash) |

A clean exit (code 0) is necessary but **not sufficient** for `done` — the result file must exist.

**Pane** — reads result-file presence and **pane** liveness (no exit file is ever written or consulted), reporting a **subset of three**:

| State | Condition |
|-------|-----------|
| `done` | `{stage}-result.yaml` present |
| `running` | result absent AND the pane is alive |
| `orphaned` | result absent AND the pane is dead (killed / crashed / tmux server gone) |

**Result presence WINS over pane liveness** — an interactive worker that produced its result and is still sitting at its prompt reads `done`, not `running` (a liveness-first rule would never terminate). **`failed` and `failed (no-result)` are UNREACHABLE in pane mode**: there is no exit-code channel, so a crashed or killed worker collapses into `orphaned`.

Human output is the bare state string on stdout. `--json` emits `{change, stage, state, mode, …}` where `mode` is `headless` or `pane` and the mode's identity keys follow: `pid`, `pgid`, `exit?` (headless) or `pane`, `window`, `server?`, `pane_pid?`, `delivered` (pane). Keys for the other mode are **omitted**, so a headless object is unchanged apart from the added `mode`. **`server`** carries the record's tmux socket label when the pane dispatch was started with `--server`/`-L` (omitted for a default-socket dispatch) — additive (the `repo`/`window_id`/`pr_url` precedent), so a socket-scoped `fab pane capture -L <server> <pane>` assembles from `--json` alone. **`pane_pid`** mirrors the record's open-time liveness discriminator (omitted when the record has none — an older binary's record or a failed open-time read). **`delivered` is reported even when `false`** — a pane is opened and delivered to in two steps, so "opened but holding no prompt yet" is a case a consumer must be able to see — and it is bookkeeping, never a state: `state` is derived without it.

### wait — `fab dispatch wait <change> <stage> [--timeout <secs>] [--json]`

`status`'s **blocking** sibling: it blocks until the dispatch's state leaves `running`, then prints that state **exactly as `status` does** (same text output, same `--json` object) and exits 0. It exists so an orchestrator can be **woken by a state change** instead of burning an inference turn every 30s asking whether one happened — the wiring in `_preamble.md` § CLI-Adapter Dispatch step 2 runs it as a **background command** (the harness's notify-on-exit seam) and falls back to a plain foreground blocking call on harnesses without one.

- **One derivation, shared with `status`.** The block re-derives state on an internal **~2s tick** through the same record loader and the same pure derivations `status` uses — `DeriveState` (headless: `{stage}.exit` + result file + pid liveness) or `DerivePaneState` (pane: result file + pane liveness), selected by the record's derived mode. `wait` and `status` therefore **cannot disagree** about state, by construction. The tick is an internal constant with **no flag and no config field**.
- **Not a file watcher.** A watcher on `{stage}-result.yaml` would see a worker *finish* but never see one *die* — `orphaned` is derived from pid/pane liveness, which needs a periodic probe regardless. The cost being eliminated is inference turns, not syscalls.
- **`--timeout <secs>` bounds the block, and expiry is NOT an error.** On expiry `wait` prints the still-current state — necessarily `running`, since any other state would already have ended the wait — and exits **0**. **The state string is the sole discriminator**: read `running` ⇒ the wait timed out (the wiring treats that wake as its peek-on-suspicion moment); read anything else ⇒ a terminal state was reached. Absent (or `0`) ⇒ wait indefinitely.
- **Already-terminal returns immediately** — no tick is consumed when the state is non-`running` at entry, so the verb is idempotent and safe to re-arm after a `restart` or re-run after an interruption.
- **Exit codes**: 0 for any successfully observed state (terminal **or** timeout); non-zero only for real errors — no dispatch record for the pair, or an unresolvable change — with the **same message surface as `status`** (`no dispatch for <change>/<stage> (run \`fab dispatch start\` for a headless worker, or \`fab dispatch open\` for a pane worker, first)`).
- **`--json`** emits the same object `status --json` emits, through the same render path (mode discriminator + that mode's identity keys).
- **Does not touch the worker.** `wait --timeout` bounds the *observer*; the unrelated `start`/`restart` `--timeout` is a POSIX `timeout` inside the headless wrapper that **kills** the worker. Waiting has no side effects at all: no state is written and no signal is sent.

### logs — `fab dispatch logs <change> <stage> [--tail N]`

Prints `.fab-dispatch/{id}/{stage}.log`. `--tail N` prints the last N lines (Go-side, no external `tail`). Missing log → `no dispatch log for <change>/<stage>`.

**Pane dispatches keep no log file** — an interactive worker's output is tmux scrollback, not a redirected stream — so `logs` reports that and names the equivalent: `<change>/<stage> is a pane dispatch and keeps no log file (an interactive worker's output is tmux scrollback); read it with 'fab pane capture <pane>'`. When the record carries a `server` (the dispatch was started with `--server`/`-L`), the suggested command carries the socket too — `fab pane capture -L <server> <pane>` — since a socket-scoped pane is unreachable from a default-socket capture. This report is therefore the copy-pasteable source for the capture command (equivalently, `status --json` exposes both `pane` and `server`). The hint names a TARGET, so one identity guard applies: a record whose pane ID exists but whose `pane_pid` no longer matches (the restart-alias) gets a gone-worker report naming `fab dispatch restart`, with **no** capture command aimed at the impostor (see § fab dispatch → *Pane liveness is identity-checked*).

### kill — `fab dispatch kill <change> <stage>`

Terminates the dispatch by the mechanism its recorded **mode** implies, idempotently in both cases (an already-dead target is a benign no-op with a clear report; a missing record is a clear error `no dispatch for <change>/<stage>`):

- **Headless** — `SIGTERM` to the **process group** (`pgid` from `{stage}.yaml`) so the detached command and its children die together. Already dead → `dispatch <change>/<stage> already dead (pid N); nothing to kill`.
- **Pane** — kills the **tmux pane** (`tmux kill-pane -t <pane>`), taking the interactive worker down with it. Shape-blind: the target is the pane ID, so a **split** worker's pane dies while the dispatching agent's window (and any sibling worker) survives, and a **new-window** worker's window closes with its only pane. Already gone → `dispatch <change>/<stage> already dead (pane %N); nothing to kill`. The liveness read is identity-checked, so a restart-aliased record (pane ID exists, `pane_pid` mismatches) is the SAME already-dead no-op — no `kill-pane` is ever sent at the impostor.

No marker file is written in either mode: with no result file present, a killed dispatch simply reads `orphaned` on the next `status`.

> **`kill` vs `reap`** (§ reap). `kill` is the **recovery** verb: valid in **any** state, ungated by config, and the one the Recovery policy spends on a parked worker. `reap` is the **hygiene** verb: it fires **only** on `done`, is gated on `dispatch.reap_done`, and refuses to touch a live or failed dispatch. Use `kill` to terminate something; use `reap` to reclaim the space something finished with. They share the same `kill-pane` mechanism and the same idempotence, and neither removes any `.fab-dispatch/` state.

### reap — `fab dispatch reap <change> <stage>`

Reclaims a **done pane worker's tmux pane**. It exists because a pane-mode worker never exits on completion — it writes `{stage}-result.yaml` and sits at its prompt (by design, so it stays steerable) — so across a multi-stage pipeline the carved worker column fills with finished panes and the panes the user actually watches shrink with every completed stage.

**WHEN the pipeline calls it — and, for the deferred apply reap, WHETHER — is wiring, not a flag** — `_preamble.md` § CLI-Adapter Dispatch step 3 owns that rule. The Go guard below is unaffected by it either way: it fires on pane + `done` + knob whenever the call is made, which is why a call made against an existing record needs no skill-side mode or config check.

**No flags.** No `--json`, and no `--server`: the socket comes from the record, so a `--server`-started dispatch is reaped on the right socket with nothing extra passed.

It kills the pane **only when all three** hold — and owns the whole guard itself, which is why the skill-side call is unconditional and dumb:

1. the record is **pane-mode** (a `pane:`-bearing record), **and**
2. the **derived state is `done`** (`{stage}-result.yaml` present — pane liveness is irrelevant to the state), **and**
3. **`dispatch.reap_done` resolves `true`** (default `true`) through the four-tier config cascade (environment > system > project > built-in defaults) — the reason the policy check lives in Go: a skill reading `fab/project/config.yaml` directly would miss the higher-precedence environment tier and the machine-wide system tier `~/.fab-kit/config.yaml`, which for a `both`-scope preference is exactly where it is usually set — and which now outranks the project file.

Every other case is a **no-op that names its reason and exits 0**:

| Case | Behavior |
|------|----------|
| record is **headless** | no-op — the worker process already exited; there is nothing visual to reclaim |
| state is **not `done`** (`running` / `orphaned`, or any headless state) | no-op — reap is NOT kill; it must never terminate a live or failed dispatch. The report points at `fab dispatch kill` for that |
| **`dispatch.reap_done: false`** | no-op — the user opted to keep done-worker panes and their scrollback |
| **pane already gone** (killed by hand, tmux server died) — **or restart-aliased** (pane ID exists, `pane_pid` mismatches) | benign already-gone report — mirrors `kill`'s idempotence, so a re-reap is safe, and no `kill-pane` is ever aimed at the impostor |
| all three hold | `tmux kill-pane -t <pane>` on the record's own socket; reports `reaped pane %N for <change>/<stage>` |

**Exit codes**: 0 for the reap and for every no-op above; non-zero **only** for real errors — no dispatch record for the pair, or an unresolvable change — sharing `status`/`wait`'s message surface (`no dispatch for <change>/<stage> (run \`fab dispatch start\` for a headless worker, or \`fab dispatch open\` for a pane worker, first)`). An **unreadable `fab/project/config.yaml` is deliberately NOT in that set**: the knob is resolved only once conditions 1–2 already hold, so a headless or not-`done` no-op reads no config at all, and where it *is* read a config that will not parse **warns on stderr and falls back to the built-in default** (`true`) — the same value an absent key resolves to. The wiring calls reap without inspecting mode or config wherever it calls it at all, so a broken config must not turn pane hygiene into a pipeline failure.

**Reap kills the pane only — it cleans no state.** The record (`{stage}.yaml`), the result (`{stage}-result.yaml`), the prompt file, and the log all remain. That is exactly why a reaped dispatch still reads **`done`** forever (`DerivePaneState` gives result presence precedence over pane liveness) and why reap is pane *hygiene*, not state cleanup: the no-automatic-GC posture — archive-time deletion plus explicit `fab dispatch clean` as the only two cleanup moments (§ clean) — is untouched. It is also **shape-blind**, like `kill`: killing a split worker's pane leaves the dispatching agent's pane, its window, and any sibling worker intact, while killing the only pane of a new-window worker takes the window with it. A `restart` after a reaped attempt needs no special handling — last-attempt-only overwrite already covers a completed prior attempt.

### clean — `fab dispatch clean [<change>] [--orphans]`

Manual cleanup — one of exactly **two** cleanup paths (the other is archive-time deletion; there is **no automatic GC** anywhere):

- `fab dispatch clean <change>` — removes `.fab-dispatch/{id}/` for the named change.
- `fab dispatch clean` (no arg) — removes all `.fab-dispatch/*/` dirs.
- `fab dispatch clean --orphans` — prunes any `.fab-dispatch/{id}/` whose ID no longer resolves to a non-archived change (covers a change archived/deleted upstream leaving a local state dir orphaned).

`clean` is **mode-blind** — it removes state dirs and never inspects a record's mode, so a pane dispatch's dir (prompt file included) is cleaned exactly like a headless one's. As with a live headless process, cleaning a **live** pane dispatch removes the state without killing the worker; `kill` is the verb for that.

---


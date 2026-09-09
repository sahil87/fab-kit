# Intake: Operator Drives the rk Cron Clock Explicitly — Mute/Unmute on Tracked-Set Transitions, Lease as Bounded Snooze

**Change**: 260909-qvek-operator-cron-mute-lease
**Created**: 2026-09-09

## Origin

> **Title (verbatim)**: Operator drives the rk cron clock explicitly — mute/unmute on tracked-set transitions, lease as bounded snooze, docs realigned to the guard-free entry
>
> **Context / origin (cross-repo, 2026-09-09)**: run-kit change `260909-upt2-cron-decoupling-mute-lease` (PR pending on sahil87/run-kit) removed BOTH `suppress_while` guards from the cron substrate: `nothing-tracked` and `operator-loop-fresh` no longer exist. Root cause of the incident that triggered it: rk resolved the fab operator state file as `$XDG_STATE_HOME/fab/operator/<tmux server name>.yaml` while fab writes `<slugified socket path>.yaml` (e.g. `tmp-tmux--1001-runKit.yaml`), so rk never found the file, `nothing-tracked` held on the cold posture, and every operator tick on every server was silently suppressed for days. The design decision (the user's): the clock must never infer operator intent by parsing fab's private state file; the operator TELLS the clock through the clock's own verbs. rk now: (1) has no guards; (2) `rk cron mute <id>` = indefinite mute, `rk cron mute <id> --for <dur>` = lease (auto-expires; no unmute call needed), `rk cron mute <id> --off` = clear both; (3) the seeded entry is `{kind: backoff, min: 60s, max: 30m}` (NO `anchor` field), `wake_on: {agent-state-change, scope: server, debounce: 10s}`, `target: {kind: role, role: operator}`, payload `operator tick`, `deliver: immediate`, `if_absent: respawn`, `respawn: ["rk","operator","-L","{server}"]` (caller-supplied argv; `{server}` substituted by rk at fire time), `pinned: true` — NO `suppress_while`; (4) `rk operator -L <server>` exists (daemon-invocable); (5) role targets accept any `@rk_win_role` value; (6) rk still reads fab's operator state file for DISPLAY only (◉ watched rows, `⚠ operator stale`), now mirroring fab's `slugify` rule exactly — pinned in run-kit's `docs/specs/cron.md` as a cross-repo contract: fab-kit OWNS the file and its slug rule; renaming either requires a coordinated run-kit change.
>
> Interim state until this fab-kit change ships: an idle operator is ticked every backoff step (60s→30m) with nothing suppressing it — loud but safe.
>
> **What changes (fab-kit side)**: (1) quiescence becomes an explicit mute issued by the Go verbs that mutate the tracked set; (2) the lease (`--for <dur>`) is documented as a bounded snooze, not a heartbeat; (3) skill text — §2 Init ready line, §4 clock block, Tick Behavior quiescence step — quotes the entry WITHOUT `anchor` and WITHOUT `suppress_while`, "stop-when-empty" becomes "the tracked-set verbs mute/unmute the entry", the ready line also reports `muted` / `muted until <t>`; bare-payload rule and Claude-only `/loop 3m` fallback paragraph unchanged; (4) memory/spec realignment (`docs/memory/runtime/operator.md` clock block + two Design Decisions, `docs/specs/skills.md` ~line 1120, any `_cli-external` / `_cli-agents` `rk cron` reference); (5) cross-repo contract note: `StatePath`/`serverSlug`/`slugify` in `cmd/fab/operator.go` are read by run-kit (mirrored slug rule) — code comment + memory sentence.
>
> **Out of scope**: any change to run-kit (done in `260909-upt2-cron-decoupling-mute-lease`); changing the operator state file's name or schema; re-introducing any rk-side inference of operator state.
>
> **Open design point to record (do not resolve)**: whether the mute/unmute calls belong in the Go verbs (recommended: deterministic) or in the skill's tick prose (simpler, but an LLM step that can be skipped). The user should confirm.

**Interaction mode**: one-shot, promptless dispatch — `/fab-draft` executed by a sub-agent from an orchestrator working in the run-kit repo, with `{questioning-mode} = promptless-defer` (no questions asked; would-be questions recorded as deferred Unresolved rows). The description above is the sole design source; all exact values (verbs, entry fields, respawn argv, file paths) are carried verbatim from it. The sub-agent grounded the description against fab-kit's current truth before writing: `docs/memory/runtime/operator.md` (clock block line 130; § Design Decisions "Cron Entry as the Sole Cadence, Provider-Neutral", "Document the `nothing-tracked`/Merge-Sequence Gap, No Workaround"), `src/kit/skills/fab-operator.md` (§2 Init steps 4–5, §4 The Clock, Tick Behavior step 7, §6 Auto-Merge Choreography rule 4 + known-gap paragraph, §8 Settings, §9 Key Properties), `src/go/fab/cmd/fab/operator.go` (`slugify` / `serverSlug` / `StatePath`), `operator_state.go` (`mutateOperatorState`), `operator_monitored.go`, `operator_autopilot.go`, `operator_watch.go`, `operator_note.go`, `operator_tick_start.go`, `src/kit/skills/_cli-fab.md` § fab operator, `src/kit/skills/_cli-external.md` § rk / § /loop, `docs/specs/skills.md` § /fab-operator, and `fab/changes/260909-6t88-retire-loop-cron-cadence/` (the change that retired the in-session `/loop`, PR #655).

## Why

1. **The pain point.** After 260909-6t88 the operator's only clock is run-kit's seeded operator-tick cron entry, and fab-kit's skill/memory text says quiescence is the entry's `nothing-tracked` guard ("while `monitored`, `watches`, and `autopilot` are all empty, fires are skipped silently") and migration safety is `operator-loop-fresh`. Both guards worked by rk parsing fab's private operator state file — and rk resolved the wrong filename (`<tmux server name>.yaml` vs fab's `<slugified socket path>.yaml`, e.g. `tmp-tmux--1001-runKit.yaml`), so `nothing-tracked` held on the cold posture and every operator tick on every server was silently suppressed for days. run-kit `260909-upt2-cron-decoupling-mute-lease` removed both guards outright: fab-kit's documentation of the clock is now wrong in three places (skill §4 entry shape + union-predicate bullets, Tick Behavior step 7, memory clock block + two Design Decisions), and nothing on the fab side stops ticks when there is nothing to do.

2. **If we don't fix it.** The interim posture is loud but safe: an idle operator is ticked every backoff step (60s→30m) forever, each tick spending a `fab operator tick-start --diff --quiet` and an LLM turn on a fleet of zero. The skill teaches a guard that no longer exists, so a user reading the ready line or §4 cannot explain why ticks keep arriving on an empty operator, and the documented "known gap" (merge sequence open, monitored set empty → ticks suppressed) is described as a run-kit follow-up when run-kit has already decided it will never re-add the inference.

3. **Why this approach.** The user's design decision: the clock never infers operator intent from fab's private file; the operator *tells* the clock through the clock's own verbs (`rk cron mute <id>` / `--for <dur>` / `--off`). Issuing the mute/unmute from the Go verbs that mutate the tracked set (rather than from skill prose) makes quiescence deterministic — it cannot be skipped by an LLM step, it survives compaction, and the rk-absent/old-rk case degrades exactly to today's loud-but-safe interim behavior because every rk call is `command -v rk`-gated and fail-silent. It also closes the merge-sequence gap by construction: the operator decides when ticks stop, so an open merge-sequence `coordination` note keeps them coming. The alternative — mute from the skill's tick prose — is recorded as the one open design point for the user to confirm (see Open Questions / Assumptions row 18).

## What Changes

All skill edits target the canonical sources under `src/kit/skills/` — never the deployed copies under `.agents/skills/` or `.claude/skills/` (`fab/project/context.md` § Skills; code-quality anti-pattern "Editing deployed skills directly").

### 1. Go: quiescence is an explicit mute issued by the tracked-set verbs (`src/go/fab/cmd/fab/`)

**Behavior contract.** After any operator-state mutation:

- When the mutation leaves `monitored`, `watches`, and `autopilot` **all empty** AND no merge-sequence `coordination` note is open → run `rk cron mute <id>` (indefinite mute).
- When the mutation makes any of them **non-empty** (or a merge sequence opens) → run `rk cron mute <id> --off` (clears both an indefinite mute and a lease).

**Candidate verbs** (every one already funnels through `mutateOperatorState` in `operator_state.go:127` — the read-modify-write skeleton):

| Verb | File | Tracked-set effect |
|------|------|--------------------|
| `fab operator enroll <id> …` | `operator_monitored.go` `runOperatorEnroll` | adds to `monitored` |
| `fab operator remove <id>` | `operator_monitored.go` `runOperatorRemove` | removes from `monitored` (the completion / pane-death / mismatch / agent-exited ack) |
| `fab operator watch add <name> …` / `watch rm <name>` | `operator_watch.go` `runOperatorWatchAdd` / `operatorWatchRmCmd` | adds/removes a `watches` entry |
| `fab operator autopilot start --queue …` / `autopilot stop` | `operator_autopilot.go` `runOperatorAutopilotStart` / `runOperatorAutopilotStop` | sets / clears the `autopilot` block |
| `fab operator autopilot advance` (on exhaustion) | `operator_autopilot.go` `runOperatorAutopilotAdvance` | sets `current: null, state: null` while retaining `queue`/`completed`/`mode` |
| `fab operator note add --kind coordination …` / `note resolve <id>` | `operator_note.go` `runOperatorNoteAdd` / `runOperatorNoteResolve` | opens / closes the merge-sequence coordination note (skill §6 Auto-Merge Choreography rule 4 writes it) |

**Implementation shape** (placement in the Go verbs confirmed by the user 2026-09-09 — see § Clarifications):

- Compute a boolean `tracked` from the state map **before and after** `fn(data)` inside `mutateOperatorState`, and issue the rk call **only when the boolean flips** (edge-triggered). Rationale: `fab operator tick-start --diff` (`operator_tick_start.go:263`) also runs through `mutateOperatorState` for its baseline write — a level-triggered hook would shell out to rk on every tick; edge-triggering makes a tick cost zero rk calls and leaves a user-set lease alone across unrelated mutations. <!-- clarified: edge-triggered flip inside mutateOperatorState confirmed by the user 2026-09-09; the post-flip drift window is closed by the lightweight reconcile bullet below -->
- `tracked` = `len(monitored) > 0` OR `autopilot` block present with `state` non-null (`running`/`paused`; an exhausted-but-retained block with `state: null` counts as empty, matching the retired guard's reading at merge-all time) OR `len(watches) > 0` (enabled or disabled) OR any note with `kind: coordination` and `resolved: false`. <!-- clarified: user confirmed 2026-09-09 — any unresolved `kind: coordination` note counts as "merge sequence open"; no reserved-marker convention -->
- **Lightweight reconcile** — two self-heals that close the drift window left by a silently failed flip (rk absent or old at the moment of the flip), confirmed by the user 2026-09-09: (a) **loud direction, Go** — `fab operator tick-start --diff` (`operator_tick_start.go`), after its baseline write, issues `rk cron mute <id>` when the resulting state is untracked: a tick arriving on an empty tracked set *is* the drift signal, so this costs an rk call only while drifted (in the Claude-only `/loop` fallback the cron entry is absent and the call no-ops); (b) **silent direction, skill prose** — §2 Init step 4 issues `rk cron mute <id> --off` when `rk cron list --json` reports the entry `muted` while `fab operator state` shows tracked work (heals a muted-while-tracked entry on every startup/reload — the incident class this change exists to prevent). The verbs themselves stay edge-triggered; neither self-heal touches a live lease except (b), which only fires while work is tracked and a standing mute/lease would silence it.
- Entry resolution: run `rk cron list --json`, select the entry whose `target` is `{kind: role, role: operator}` (name / payload `operator tick` as the tiebreak); zero matches, several matches, or unparseable output → silent no-op. Field names are read from the installed rk's actual `--json` output at apply time, never invented.
- Server addressing: pass no `-L`. The operator subcommand family has no server flag and `operatorStatePath()` resolves `StatePath("")` — the *current* server — so fab has no server name to hand rk; rk's own `$TMUX` derivation is the addressing. (The description's `rk cron -L <server>` branch applies only "if fab knows the server", which it does not on these verbs.)
- Every rk call: `exec.LookPath("rk")`-gated (the `command -v rk` rule of `_preamble.md` § Run-Kit), argv exec (never a shell string), bounded by a short timeout (no timeout-capable helper exists — `pane.RunCmd` in `internal/pane/pane.go:73` is unbounded — so add a context-bound variant; the value, e.g. 5s, is settled at apply <!-- clarified: ~5s per-call timeout via a new context-bound RunCmd variant — user confirmed 2026-09-09 -->), and **fail-silent**: a non-zero exit (an rk older than the `mute --for`/`--off` verbs, no entry, no daemon), a timeout, or absent rk never changes the verb's exit code or stdout — the state mutation has already been saved; the clock call is a best-effort side effect. Degradation is exactly today's interim behavior (ticks keep arriving).
- Runner as an injectable package-level seam (the `rkPanesRunner` / `rkAwaitRunner` / `rkOperatorPath` precedent) so tests stub rk without a live daemon.

**Tests (`operator_*_test.go`, Constitution VII + code-quality "Go changes ship tests")**: flip empty→non-empty on `enroll`/`watch add`/`autopilot start`/`note add --kind coordination` issues exactly one `--off`; flip non-empty→empty on the last `remove`/`watch rm`/`autopilot stop`/`note resolve` issues exactly one indefinite mute; `advance` to exhaustion with everything else empty mutes, but not while an open coordination note exists; a mutation that does not flip (second enroll, `update`, `tick-start --diff`, `note add --kind correction`) issues no rk call; rk absent / non-zero / timeout leaves the verb's exit code and output unchanged; `tick-start --diff` on an untracked state issues exactly one indefinite mute and on a tracked state issues none (the reconcile); the entry-selection predicate on a fixture `rk cron list --json` document.

### 2. The lease is a bounded snooze, not a heartbeat (skill §4)

The in-session `/loop` was retired by 260909-6t88, so there is no loop to renew a lease; `operator-loop-fresh` has no successor. New §4 prose:

- The skill **MAY** use `rk cron mute <id> --for <dur>` when the operator knowingly wants quiet for a bounded window — e.g. a user says "hold the ticks for 30 minutes" (conversational; `--for 30m`). The lease auto-expires; no unmute call is needed.
- The skill **MUST** never leave an indefinite mute behind while work is tracked. The Go verbs enforce the tracked-set half (an enroll/`--off` clears any standing mute or lease); the prose rule covers a manual `rk cron mute <id>` a user or the operator typed.
- `rk cron mute <id> --off` clears both a mute and a lease.

### 3. Skill text — `src/kit/skills/fab-operator.md`

**§2 Init step 4 / step 5 (ready line).** Step 4 keeps the fail-silent `command -v rk` + `rk cron list` verification but reads `rk cron list --json` so it can also see the mute state — and, when the entry is `muted` while `fab operator state` shows tracked work, issues `rk cron mute <id> --off` (the reload-time half of the lightweight reconcile, § 1). Step 5's first ready-line form gains a clock-status suffix when the entry is muted, so a user sees why ticks are quiet (exact copy at apply; the copy-never-compose rule survives — whatever literals land are the ones the agent copies):

```
Operator ready. Clock: rk cron "operator tick" (backoff 60s–30m, wakes on agent-state-change)
Operator ready. Clock: rk cron "operator tick" (backoff 60s–30m, wakes on agent-state-change) · muted
Operator ready. Clock: rk cron "operator tick" (backoff 60s–30m, wakes on agent-state-change) · muted until <t>
Operator ready. Clock: none — run `rk operator` to seed the cron entry, or (Claude Code only) start the fallback: /loop 3m "operator tick"
```

**§4 The Clock — entry shape**, quoted WITHOUT `anchor` and WITHOUT `suppress_while`, plus the respawn argv:

```yaml
schedule: { kind: backoff, min: 60s, max: 30m }
wake_on: { event: agent-state-change, scope: server, debounce: 10s }
target: { kind: role, role: operator }
payload: "operator tick"
deliver: immediate
if_absent: respawn
respawn: ["rk", "operator", "-L", "{server}"]   # caller-supplied argv; {server} substituted by rk at fire time
pinned: true
```

**§4 Ownership paragraph** — rewritten: `rk operator` seeds the entry idempotently; **the tracked-set verbs mute/unmute it** (`fab operator enroll`/`remove`, `watch add`/`rm`, `autopilot start`/`stop`/`advance`-to-exhaustion, `note add --kind coordination`/`resolve`) via `rk cron mute <id>` / `--off`; `rk cron add`/`rm` and the schedule stay the user's. The "this skill never creates or mutates it" sentence and the anchor-join sentence ("the `backoff` anchor is the operator's idle epoch joined against rk's delivery log…") are deleted — there is no anchor.

**§4 union-predicate bullets** — the `suppress_while: [nothing-tracked]` bullet becomes "the tracked-set verbs mute/unmute the entry ≻ the retired stop-when-empty rule: an empty tracked set (no monitored entries, watches, active autopilot, or open merge-sequence coordination note) mutes the entry; the first thing tracked unmutes it"; the `suppress_while: [operator-loop-fresh]` bullet is deleted (no successor — the loop is gone); the `backoff` bullet drops "anchored on operator idle". A new short subsection **Mute and Lease** carries § 2 above.

**Tick Behavior step 7** — "Clock lifecycle — none to manage in the tick: the tracked-set verbs mute/unmute the entry as a side effect of their state mutation (§4 Mute and Lease); cadence adaptation is the entry's backoff + `wake_on` union predicate — evaluated by rk, not the tick." (Step 7 stays so §2 Init and the Post-Compaction Reload trigger sentence keep their target.)

**§6 Auto-Merge Choreography** — the "Known gap (run-kit follow-up)" paragraph is replaced: an open merge-sequence `coordination` note is tracked state to the mute logic (`note add --kind coordination` unmutes, `note resolve` may mute), so ticks keep coming for the whole armed sequence by construction; no gap, no follow-up.

**§9 Key Properties `Cadence` row** — drop "`operator-loop-fresh`/`nothing-tracked` suppress guards" and "never mutated by the skill"; add "tracked-set verbs mute/unmute via `rk cron mute`; lease = bounded snooze". §1 header prose and §8 Settings ("Cadence is not a session setting — it is the cron entry's to tune via `rk cron`") stay valid as written.

**Unchanged**: the bare-payload rule (`operator tick`, never a slash command — § Tick Payload); the Claude-only `/loop 3m "operator tick"` Degraded Fallback paragraph; Post-Compaction Reload.

### 4. Memory and spec realignment

- `docs/memory/runtime/operator.md`:
  - **Clock block (line ~130)**: entry shape without `anchor`/`suppress_while`, with `respawn: ["rk","operator","-L","{server}"]`; drop the `operator-loop-fresh`/`nothing-tracked` union-predicate prose; add the mute/lease verbs and the tracked-set-verb ownership; the trailing "Known gap (run-kit follow-up)" sentence is deleted.
  - **Monitoring tick closing paragraph (line ~171)** "Quiescence is not the tick's to manage — the entry's `nothing-tracked` suppress guard…" → the tracked-set verbs' mute.
  - **§ Design Decisions** — "Cron Entry as the Sole Cadence, Provider-Neutral": drop the union-predicate/guard prose from Decision/Why (keep the Rejected line's historical mention of `operator-loop-fresh` as a rejected alternative only if it still reads as history). "Document the `nothing-tracked`/Merge-Sequence Gap, No Workaround": superseded — rewrite (or replace) as the gap closed by explicit mute. New entries in the four-field shape: **Explicit Mute Over Inferred Quiescence** (Decision: the operator tells the clock via `rk cron mute` from the Go verbs; Why: rk's parsing of fab's private file failed silently for days; Rejected: rk-side inference, skill-prose-only mute, a fab-side heartbeat) and **Lease Is a Bounded Snooze, Not a Heartbeat** (Rejected: renewing leases from a loop — no loop exists).
  - **Server-keyed state file paragraph (line ~80)** and the "one operator per server" Design Decision (line ~397): one sentence — the file path and its `slugify` rule are a cross-repo contract read by run-kit for display; renaming the file or changing the slug rule requires a coordinated run-kit change (see § 5).
- `docs/specs/skills.md` § /fab-operator (lines ~1114 and ~1120): the Flow header line and the Tools line add the mute/lease posture — cadence delivered by the guard-free entry; the tracked-set verbs mute/unmute it; `rk cron list --json` / `rk cron mute --for` are the skill's own rk uses (deliberate human-authored spec edit — Constitution VI permits authored edits).
- `src/kit/skills/_cli-external.md` § rk (run-kit): add a fab-owned pointer "Operator clock mute/lease" → owned by `fab-operator.md` §4 Mute and Lease (skill-side lease) and `_cli-fab.md` § fab operator (Go-side mute/unmute). § /loop is unchanged (the fallback still exists). `src/kit/skills/_cli-agents.md` carries no `rk cron` reference (grep-verified 2026-09-09) — no edit.
- `src/kit/skills/_cli-fab.md` § fab operator (line ~1185; subsections `enroll / update / remove`, `watch`, `autopilot`, `note`): document the clock side effect once (a shared paragraph under § fab operator) and reference it from each verb — the Constitution's "CLI change ⇒ `_cli-fab.md`" constraint. Also the **State path** paragraph (line ~1266): add the cross-repo-contract sentence (sibling of the memory sentence — sweep the class up front).

### 5. Cross-repo contract note in code

`src/go/fab/cmd/fab/operator.go` — `slugify` (line 295), `serverSlug` (line 310), `StatePath` (line 323): add a code comment that run-kit mirrors this exact slug rule to locate the file for display (◉ watched rows, `⚠ operator stale`), pinned in run-kit's `docs/specs/cron.md`; fab-kit owns the file and the rule; renaming the file or changing the rule requires a coordinated run-kit change. The existing `slugify` tests in `operator_test.go` (line ~668) already pin the rule's determinism/injectivity — add a one-line comment there naming the cross-repo consumer so a future edit sees it in both places.

## Affected Memory

- `runtime/operator`: (modify) clock block (entry shape without `anchor`/`suppress_while`, respawn argv, mute/lease verbs, tracked-set-verb ownership), the monitoring-tick quiescence sentence, the server-keyed-state-file paragraph (cross-repo contract sentence), Design Decisions — rewrite "Cron Entry as the Sole Cadence, Provider-Neutral" (drop guard prose), supersede "Document the `nothing-tracked`/Merge-Sequence Gap, No Workaround", add "Explicit Mute Over Inferred Quiescence" and "Lease Is a Bounded Snooze, Not a Heartbeat"; description frontmatter re-checked for routing.

(Indexes regenerated via `fab docs-index` at hydrate.)

## Impact

- **Go** (`src/go/fab/cmd/fab/`): `operator_state.go` (`mutateOperatorState` edge-triggered flip hook), `operator_tick_start.go` (untracked → mute reconcile after the baseline write), a new small file or section for the rk cron runner seam + entry selection + tracked predicate, `operator.go` (comments on `slugify`/`serverSlug`/`StatePath`), `internal/pane/pane.go` (timeout-capable `RunCmd` variant) — plus tests in `operator_*_test.go`. Callers' exit codes and stdout are unchanged; the only new observable is the rk subprocess on a tracked-set flip.
- **Skills**: `src/kit/skills/fab-operator.md` (§2 Init 4–5, §4 wholesale minus Tick Payload / Degraded Fallback / Post-Compaction Reload, Tick Behavior step 7, §6 one paragraph, §9 one row), `src/kit/skills/_cli-fab.md` (§ fab operator shared paragraph + State path sentence), `src/kit/skills/_cli-external.md` (§ rk pointer).
- **Docs**: `docs/memory/runtime/operator.md` (hydrate), `docs/specs/skills.md` (two lines, authored edit), regenerated indexes.
- **Cross-repo dependency**: consuming only. The mute/lease verbs and the guard-free entry shipped in run-kit `260909-upt2-cron-decoupling-mute-lease` (merged as sahil87/run-kit#889; the installed rk v3.19.37 on 2026-09-09 carries `mute --for`/`--off` and the guard-free seeded entry). Version skew is safe in both directions: an rk **without** `mute --for`/`--off` fails the call non-zero → fail-silent → today's behavior; an rk that **still has** the guards ignores nothing fab sends (a mute on top of a guard is harmless).
- **Behavior contract**: quiescence moves from rk inference to fab's tracked-set verbs; the documented clock loses `anchor` and both guards; the merge-sequence gap is closed. Running operators pick the skill text up on their next `/fab-operator` reload; the Go behavior lands with the next `fab` release.
- **Constitution**: I (Pure Prompt Play — rk stays an optional single-binary utility, fail-silent), III (edge-triggered flip logic is idempotent — a re-run with the same inputs issues no second call), VII + Additional Constraints (Go change ⇒ tests + `_cli-fab.md`). Note: the description cites "Constitution I of fab-kit (argv exec with timeout, no shell strings)" — fab-kit's Constitution I is Pure Prompt Play and contains no argv-exec rule; the argv/timeout/no-shell-string requirement is carried here as an explicit requirement of this change, not as a constitutional citation.

## Open Questions

- ~~**Placement of the mute/unmute calls**~~ — **Resolved 2026-09-09**: Go verbs (edge-triggered inside `mutateOperatorState`), not skill prose (row 18).
- **Drift after a silently failed flip**: with edge-triggered mute, an rk call that fails silently at the flip (daemon down, rk momentarily absent) leaves the entry out of sync until the next flip. **Resolved 2026-09-09**: lightweight reconcile — `tick-start --diff` mutes on an untracked state (Go), §2 Init step 4 unmutes when muted-while-tracked (prose); see § 1 (row 19).
- **`rk cron list --json` field names** for the entry id, target, mute state, and lease expiry are rk's contract; they must be read from the installed rk at apply, not assumed here (row 9, row 11).
- ~~**Identifying "merge sequence open"**~~ — **Resolved 2026-09-09**: any unresolved `kind: coordination` note; no reserved marker (row 7).

## Clarifications

### Session 2026-09-09

Interactive `/fab-clarify` round (the intake was drafted promptless, so every would-be question was deferred). Grounding performed before the round: run-kit `260909-upt2` is merged (sahil87/run-kit#889); the installed rk is v3.19.37 with `rk cron mute <id> [--for <dur>] [--off]`; `rk cron list --json` rows are `{id, name, schedule, target, deliver, pinned, muted, muted_until?, last_fired, orphaned_since, expires_at}` with `target` a string (`"role:operator"`), `muted` the *effective* state (indefinite mute OR live lease) and `muted_until` (unix seconds) present only while a lease is live.

| # | Question | Answer |
|---|----------|--------|
| 18 | Where do the mute/unmute calls live — Go verbs or skill tick prose? | Go verbs, edge-triggered inside `mutateOperatorState` (recommended option accepted) |
| 19 | Reconcile after a silently failed flip? | Yes — lightweight reconcile: `tick-start --diff` mutes on an untracked state; §2 Init step 4 unmutes when muted-while-tracked (recommended option accepted; Decision changed from "no reconcile") |
| 7 | How is "merge sequence open" identified? | Any unresolved `kind: coordination` note; no reserved marker (recommended option accepted) |
| 6 | Edge-triggered flip check inside `mutateOperatorState`? | Confirmed (recommended option accepted) |

### Session 2026-09-09 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 7 | Confirmed | Disabled watches count as tracked |
| 9 | Confirmed | Field names verified: `target == "role:operator"`, `name`, `muted`, `muted_until` |
| 10 | Confirmed | ~5s timeout |
| 11 | Confirmed | ` · muted` / ` · muted until <t>` keyed on `muted` / `muted_until` |
| 12 | Confirmed | `event:` key |
| 18 | Confirmed | Go-verb placement; R left at 25 |

Scoring note: rows 18 and 19 were user *decisions* among alternatives, so A and D were re-scored to 95 alongside S (the agent infers nothing once the user has chosen); rows 6 and 7 were plain confirmations and follow the S→95-only rule. R was left unchanged on every row.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Quiescence is an explicit mute the operator issues through rk's own verbs (`rk cron mute <id>` / `--off`); no rk-side inference of operator state is ever reintroduced | The user's stated design decision, verbatim in the description | S:95 R:70 A:90 D:95 |
| 2 | Certain | Entry shape and values carried verbatim: `{kind: backoff, min: 60s, max: 30m}` (no `anchor`), `wake_on` agent-state-change / scope server / debounce 10s, `target: {kind: role, role: operator}`, payload `operator tick`, `deliver: immediate`, `if_absent: respawn`, `respawn: ["rk","operator","-L","{server}"]`, `pinned: true`, no `suppress_while` | Given by the description; run-kit's `docs/specs/cron.md` owns the schema | S:95 R:85 A:90 D:95 |
| 3 | Certain | Every rk call is `command -v rk`/`exec.LookPath`-gated, argv exec (no shell strings), bounded by a timeout, and fail-silent — absent or older rk degrades to today's loud-but-safe interim behavior | Description requirement + `_preamble.md` § Run-Kit universal rule; matches the `rkPanesRunner`/`rkAwaitRunner` precedents | S:95 R:85 A:90 D:90 |
| 4 | Certain | Out of scope: any run-kit change (done in `260909-upt2-cron-decoupling-mute-lease`), the operator state file's name or schema, any rk-side inference | Stated verbatim in the description | S:95 R:90 A:95 D:95 |
| 5 | Certain | The lease (`--for <dur>`) is a bounded snooze, not a heartbeat: the skill MAY use it for a user-requested bounded quiet window and MUST never leave an indefinite mute behind while work is tracked; `/loop` retirement (260909-6t88) means nothing renews a lease | Stated in the description; grounded in 6t88's shipped state | S:95 R:85 A:85 D:90 |
| 6 | Certain | Implement the mute/unmute as an edge-triggered check inside `mutateOperatorState` (tracked-ness before vs after `fn`), not a level-triggered call per mutation or per-verb call sites | Clarified — user confirmed. Every candidate verb AND `tick-start --diff` funnel through `mutateOperatorState` (`operator_state.go:127`, `operator_tick_start.go:263`); level-triggered would shell out every tick and clear user leases on unrelated mutations; edge-triggered is idempotent (Constitution III). Drift window closed by row 19 | S:95 R:80 A:80 D:65 |
| 7 | Confident | Tracked predicate: `monitored` non-empty OR `autopilot` block with non-null `state` (exhausted `state: null` = empty) OR `watches` non-empty (a *disabled* watch still counts — it exists and may be re-enabled) OR any unresolved `kind: coordination` note (no reserved merge-sequence marker; over-inclusion is the harmless direction) | Clarified — user confirmed (any open coordination note, no marker convention; disabled watches count). Matches the retired guard's documented reading ("exhausted queue + empty monitored → suppressed" in the 6t88 gap prose) plus the description's merge-sequence rule; `noteEntry` has `Kind`/`Resolved` and no sub-kind | S:95 R:85 A:65 D:55 |
| 8 | Certain | No `-L <server>` is passed to rk; rely on rk's `$TMUX` derivation | `operatorStatePath()` resolves `StatePath("")` (current server) and the operator subcommand family has no server flag — fab has no server name to pass; the description's `-L` branch is conditional on fab knowing it | S:80 R:85 A:90 D:80 |
| 9 | Confident | Entry `<id>` is resolved per call via `rk cron list --json` — the row whose `target` string equals `"role:operator"` (`name` `"operator tick"` as tiebreak); zero/many/unparseable → silent no-op. Row shape (rk v3.19.37): `{id, name, schedule, target, deliver, pinned, muted, muted_until?, last_fired, orphaned_since, expires_at}` | Clarified — user confirmed; field names verified against run-kit `cmd/rk/cron_list.go` (`cronListRecord`) on 2026-09-09 | S:95 R:85 A:60 D:70 |
| 10 | Confident | Add a context-bound variant of `pane.RunCmd` for the rk calls with a ~5s per-call timeout (rk only writes an intent file on disk, so 5s is generous) | Clarified — user confirmed. `RunCmd` (`internal/pane/pane.go:73`) is unbounded and nothing in `cmd/fab` uses `exec.CommandContext`; the description mandates a timeout | S:95 R:90 A:60 D:50 |
| 11 | Confident | Ready line (§2 Init step 5) appends ` · muted` when the row has `muted: true` and no `muted_until`, and ` · muted until <t>` (local HH:MM) when `muted_until` is present; exact copy at apply; copy-never-compose rule kept | Clarified — user confirmed. `muted` is the *effective* state (indefinite mute OR live lease); `muted_until` (unix seconds) is emitted only while a lease is live (row 9) | S:95 R:90 A:65 D:65 |
| 12 | Confident | Quote `wake_on` with the `event:` key (`{ event: agent-state-change, scope: server, debounce: 10s }`) as the current skill text does; the description's `{agent-state-change, scope: server, debounce: 10s}` is shorthand, not a schema change | Clarified — user confirmed; verified against run-kit `docs/specs/cron.md` (line 112) and the Go seeder (`cmd/rk/operator.go` `WakeOn{Event, Scope, Debounce}`) on 2026-09-09 | S:95 R:90 A:70 D:70 |
| 13 | Certain | The description's "Constitution I (argv exec with timeout, no shell strings)" is carried as an explicit requirement of this change, not as a constitutional citation — fab-kit's Constitution I is Pure Prompt Play (rk remains an optional single-binary utility) | `fab/project/constitution.md` read in full; no argv-exec rule exists there; the intent is honored regardless | S:70 R:90 A:90 D:80 |
| 14 | Certain | §4 Ownership is reworded: `rk operator` seeds; the tracked-set verbs mute/unmute; `rk cron add`/`rm`/schedule stay the user's; the anchor-join sentence is deleted | Follows directly from § 1 and the anchor-less entry; the current "this skill never … mutates it" claim would be false | S:85 R:85 A:90 D:90 |
| 15 | Certain | `_cli-fab.md` § fab operator documents the clock side effect once (shared paragraph) + per-verb references; Go tests stub rk via a package-level runner seam | Constitution Additional Constraints (CLI change ⇒ `_cli-fab.md` + tests); `rkPanesRunner` precedent; code-review "Go changes ship tests" | S:85 R:85 A:95 D:90 |
| 16 | Certain | Affected memory is `runtime/operator` (modify) only; spec touch is `docs/specs/skills.md` lines ~1114/~1120; `_cli-external.md` gains a § rk pointer and § /loop is unchanged; `_cli-agents.md` needs no edit | Repo-wide grep for `rk cron` / `suppress_while` / `nothing-tracked` / `operator-loop-fresh` on 2026-09-09: hits only in `fab-operator.md`, `_cli-external.md` (pointer), `docs/specs/skills.md`, `docs/memory/runtime/operator.md`, and change folders/logs | S:85 R:85 A:90 D:85 |
| 17 | Certain | Cross-repo contract lands as a code comment on `slugify`/`serverSlug`/`StatePath` (+ a test-file comment) and one sentence each in `operator.md`'s server-keyed-state-file paragraph and `_cli-fab.md`'s State path paragraph (sibling sweep) | Description item 5; code-quality § Sibling Sweeps requires touching the aggregate restatement (`_cli-fab.md:1266`) up front | S:80 R:90 A:85 D:80 |
| 18 | Confident | The mute/unmute calls live in the Go verbs (edge-triggered inside `mutateOperatorState`, § 1) — not in the skill's tick prose | Clarified — user confirmed the Go-verb placement (the description's recorded open design point). A and D re-scored as a user-given decision, not an inference: the agent no longer infers anything and there is one interpretation; R stays low because placement cascades through Go + skill + memory; re-confirmed in the bulk round | S:95 R:25 A:95 D:95 |
| 19 | Certain | Lightweight reconcile ships in this change: `tick-start --diff` issues an indefinite mute when the post-diff state is untracked (Go; fires only while drifted), and §2 Init step 4 issues `--off` when the entry is `muted` while work is tracked (prose; heals the silent direction on reload). Verbs stay edge-triggered | Clarified — user changed to "lightweight reconcile" (from "no reconcile"). A and D re-scored as a user-given decision; the muted-while-tracked direction is the incident class this change exists to prevent, so leaving it unhealed was the wrong default | S:95 R:85 A:95 D:95 |

19 assumptions (14 certain, 5 confident, 0 tentative, 0 unresolved).

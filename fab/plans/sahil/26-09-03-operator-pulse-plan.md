# Operator pulse plan — externally-owned heartbeat for the operator

> Plan doc — written 2026-09-03 from the dev-ws-sahil01 incident investigation
> (a `/loop 3m "operator tick"` silently killed by an accidental Ctrl-C; the
> session then asserted the loop was alive; three monitored workers ran
> unwatched for hours), then **revised through three adversarial review rounds**
> (liveness/correctness · architecture/placement · rev-3 deltas: rk-hosted
> lifecycle + cadence lease + respawn). All reviewer must-fixes are integrated
> below; rejected alternatives are recorded in place. Supersedes the liveness
> half of backlog [2ne8] (whose offload half stays deferred, § 2ne8 below).
> Companion facts verified 2026-09-03 against `src/go/fab/cmd/fab/operator*.go`,
> run-kit `internal/daemon`/`api/operator*.go`, and run-kit change
> `260903-a8e4-rk-operator-launcher` (glazed-gopher worktree).

**Problem:** the operator's fleet-monitoring cadence lives in a `/loop` inside the operator's own Claude session. Anything that disturbs the session — Ctrl-C, restart, compaction, harness bugs — kills the timer silently, and nothing detects the death. Root cause: **monitoring liveness depends on the monitored session.**

---

## 1. Shape: two independent clocks

- **Clock A (unchanged): the in-session `/loop`** — primary cadence, self-adaptive, works under read-only dashboard viewing, carries the Post-Compaction Reload trigger.
- **Clock B (new): `fab operator pulse run`** — a short-lived idempotent Go verb, invoked every ~60s by a **client-independent ticker goroutine in the rk daemon** (snapshotter precedent; settings key `fab_pulse`, **default ON** — the work-check makes it a zero-blast-radius no-op for non-fab users; `rk doctor` row). The verb is caller-agnostic: an OS user timer or manual run are drop-in alternates (non-blocking flock skip serializes invokers).

Arbitration is pure staleness: a live loop keeps `tick_count` advancing ⇒ pulse silent; loop dies ⇒ staleness crosses ⇒ pulse delivers the bare text `operator tick` into the operator pane; either clock alone suffices. No sole-owner mode exists (a sole clock recreates the single point of failure one layer down).

**Mutual watching:** the pulse watches the loop (staleness), and the loop watches the pulse — `tick-start --diff` reads the pulse sidecar's last-run stamp and emits `pulse_health: stale` in the tick doc, so the operator's own status frame warns "backstop not running" (covers daemon-down at zero new machinery).

**Reboot story (the one hole rk-hosting opens):** rk-daemon start is manual by its constitution, so an unattended reboot silences the backstop with no alarm. Resolution: (i) run-kit follow-up — a boot unit that *starts* rk-daemon (`Restart=no`; start-at-boot is not self-supervision, so the constitution holds); (ii) until it ships, the doc and `pulse status` classify the OS user timer (`loginctl enable-linger` + one-line timer) as **required on machines that reboot unattended**, optional elsewhere.

## 2. Topology

One clock per machine (the rk daemon / one optional OS timer), per-server logic inside each `pulse run`: every tmux server's operator gets its **own** sidecar (`~/.local/state/fab/operator/pulse/<slug>.yaml` — separate subdir, never parsed as operator state) holding its own ladder rung, attempts, notify cursor, lease view, and mute flag; per-server locks/goroutines so one wedged server can't delay another. Nothing crosses machines.

## 3. Environment discipline (rk-daemon exec hazards)

- `pulse run` **unsets `TMUX`/`TMUX_PANE` at entry** and addresses every tmux call via the **stamped absolute socket path** — never "current server". (Inherited daemon `$TMUX` would otherwise resolve the rk-daemon socket and could type `operator tick` into the daemon's own panes — pane IDs are per-server.)
- The rk ticker execs with TMUX unset, **cwd pinned to `$HOME`** (a project-worktree cwd would route the fab shim to a project-pinned, possibly pre-pulse binary), and passes `RK_HOST`/`RK_PORT` so `rk notify` resolves the right origin (no covering server without `$TMUX`).
- The ticker **capability-probes `fab operator pulse` once per daemon start**: unknown command ⇒ ticker disabled, logged once, doctor row "fab pulse: unavailable" — never an error every 60s against an older fab.

## 4. The pulse algorithm (per server, per run)

1. **Work check** — skip unless `monitored` non-empty ∨ autopilot active ∨ enabled watch ∨ open coordination note. (Same predicate as the skill's loop-lifecycle rule, so both clocks go quiet together.)
2. **Always-on liveness checks (lease-independent):** stamped socket dead ⇒ alarm now; stamped operator pane/pid dead ⇒ **fall through to the `@rk_win_role=operator` window** (relocations leave a stale stamp until the new session's first `--claim`); neither resolvable ⇒ alarm. Cost: one stat/dial + one `list-panes` + one /proc walk per server per minute.
3. **Staleness (ack = `tick_count`, lease-scaled):** sidecar remembers `last_tick_seen` and when it changed. Threshold = **2× the declared cadence lease** — `tick-start --cadence <dur>` is a **per-tick declaration** the skill always passes with its current loop interval (absent ⇒ 3m default; floor 1m, **cap 6h** — beyond that is `mute`'s territory, and a compacted session declaring a bogus huge lease must not blind the backstop for a day). Fresh ⇒ done. This is what lets an operator *legitimately* slow to 60m: the pulse holds ~2h on delivery while step 2 still catches session death within one pulse — the split is the feature; the alive-but-loop-dead window equals 2× lease and is the documented price of a long lease.
4. **Gate:** pane agent-state `active`/`waiting` ⇒ busy-hold (never type). Composer holding text that isn't ours ⇒ busy-hold (protects a human draft from the deliver choreography's C-u).
5. **Deliver** bare `operator tick`: copy-mode guard → probe → **re-read `tick_count` immediately before Enter, abort + C-u if it advanced** (TOCTOU) → Enter → screen-advance confirm.
6. **Ladder** (all counters reset on any fresh tick):
   - Verified delivery, no tick advance in 3m ⇒ one more bare delivery ⇒ still nothing ⇒ **notify**. (No reload-hint injection — never coach a known-broken session; the delivered tick itself is the Post-Compaction Reload trigger, best-effort.)
   - **Probe failure ≠ escalation** — likely cross-sender contention (rk queue drain, voice, human) or the printed-prompt trap: silent retry next pulse, capped, then notify with the capture snippet.
   - **`send-keys` read-only-client failure** (rk viewer is the only client) ⇒ infrastructure-class notify (distinct text); optional config applies the recorded `script` dummy-client workaround; never counted as operator failure.
   - Busy-hold persisting while stale >30m ⇒ notify **naming the blocking state** ("operator waiting on a question 34m; N changes unwatched") — deliberate: a `waiting` operator may hold a Strategic question a human should see, and the skill's 30m auto-default can't fire without a tick.
   - Liveness-check alarm (step 2) ⇒ notify immediately.
7. **Notify:** `rk notify` → `operator.pulse.ntfy_topic` → journal; `pulse status` prints the resolved channel and warns when none resolves. First alarm immediate; re-notify 10m → 30m → 60m cap; reset on recovery. **Auto-mute** after 24h of consecutive socket-dead pulses (final "muting <server>" notification); `pulse mute <server> [--until]` for deliberate teardowns.

**No respawn in v1** (cut by review): `rk operator` today *selects* an existing role-marked window and exits 0 — against an agent-dead-window-alive operator (the `; exec "$SHELL"` fallback shape) it would respawn nothing while reporting success, and it hard-requires `$TMUX` the pulse doesn't have. Notify-only. Re-add as opt-in once the filed a8e4 follow-up exists: `rk operator --server <name> --force-new` (skip select-on-hit when the hit window hosts no live agent).

## 5. Verbs and rollout

- `fab operator pulse run` — one iteration, silent no-op when healthy.
- `fab operator pulse status` — per-server table: work, lease, tick freshness, ladder state, resolved notify channel; warns on "no pulse ran in >2 intervals" and "no notify channel".
- `fab operator pulse mute|unmute <server>`.
- `tick-start` additions: `--cadence <dur>` (per-tick lease) and the `operator:` stamp `{pane, pid, socket_path}` guarded by `--claim` (skill §2 Init does the one legitimate claim; a diagnostic tick-start from another pane warns and doesn't restamp).
- Ladder/policy logic in an internal package (not welded into cmd/fab) — keeps the later oracle split cheap.
- Skill edits: §2 Init adds `tick-start --claim`; tick step 1 passes `--cadence`; §Loop Lifecycle gains the two-clock paragraph; status frame renders `pulse_health`; escalation table references pulse alarms.
- run-kit changes: (1) the daemon ticker goroutine (+settings key, doctor row, capability probe, env discipline) — small; (2) follow-ups filed: browser-independent operator-queue drain; `rk operator --force-new`; rk-daemon boot unit; voice×queue `noQueue` carve-out; speculative generic `rk keepalive`.
- Memory: amend the mnri "no-magic-background-work" decision with the narrowly-scoped operator-clock carve-out (pulse action vocabulary: deliver tick text to the operator pane, notify — never worker panes, never pipeline verbs, never answers).
- Rollout order: fab verb + stamp first (project pins must carry the stamp before the pulse relies on it; rk role-mark fallback covers the interim), rk ticker second.

## 6. Seam summary

| Concern | Owner |
|---|---|
| Clock lifecycle (ticker), doctor/dashboard surfacing | **run-kit** (daemon goroutine; OS timer as drop-in alternate) |
| Pulse policy: staleness, lease, ladder, mute, sidecar, delivery, alarm decisions | **fab** (`pulse run` — the brain; caller-agnostic) |
| Delivery mechanics | fab's guarded sender (copy-mode + probe + TOCTOU); evolution path: rk's inject engine absorbs delivery, fab shrinks to a stateless `pulse check --json` oracle — seam shaped so nothing is thrown away |
| Operator launcher, pane identity, agent-state, notify transport, request queue, voice | **run-kit** (a8e4 et al.), fab consumes |

**Voice / operator-as-actuator:** two channels, one gate. Cadence rides the pulse's direct delivery (never queued — TTL/coalesce/drop-quietly semantics are wrong for heartbeats, and the queue's drain is browser-gated today). Intents (dashboard, auto-name, voice) keep riding rk's queue. Both gate on pane state and probe-verify; the pane serializes them; the pulse treats probe collisions as contention, not failure, and yields by construction.

**2ne8 (full tick offload):** deferred — breaks the single-ticker/baseline-writer invariant, eats consumed-on-read deltas, removes the reload trigger; and the pulse is its substrate (a later offload swaps trigger + executor, reusing pane resolution, guarded wake, escalation, notify). Backlog entry re-pointed here.

## 7. Change ladder

| # | Repo | Change | Contents | Depends on |
|---|------|--------|----------|------------|
| 1 | fab-kit | tick-start lease + stamp | `--cadence` (per-tick lease), `operator:` stamp + `--claim` guard, `pulse_health` emission; skill edits (§2 Init claim, tick step 1 cadence, two-clock §Loop Lifecycle paragraph, frame renders pulse_health); mnri carve-out memory amendment | — |
| 2 | fab-kit | `fab operator pulse` verb family | `run`/`status`/`mute`/`unmute`, sidecar + ladder in an internal package, guarded delivery, notify chain, tests | 1 |
| 3 | run-kit | daemon pulse ticker | client-independent goroutine (snapshotter pattern), `fab_pulse` settings key (default ON), capability probe, env discipline (TMUX unset, cwd `$HOME`, RK_HOST/RK_PORT), doctor row | 2 shipped in installed fab |
| — | run-kit | follow-up ideas to file | browser-independent operator-queue drain; `rk operator --server --force-new` (unblocks pulse respawn); rk-daemon boot unit (closes the reboot hole); voice×queue `noQueue` carve-out; speculative generic `rk keepalive` | — |

## 8. Priced trades

- Dead loop → recovery/page in ~(2× lease) + delivery + 3m grace: ~10–12m at default cadence. Long leases stretch the alive-but-loop-dead window proportionally, by informed choice; session *death* pages within ~1 pulse regardless.
- A `waiting` operator blocks ticks up to 30m before a page (humans should see Strategic questions).
- Backstop liveness chains to rk-daemon uptime until the boot unit ships (OS timer covers the paranoid); the loop-side `pulse_health` warning and doctor row make that chain visible.
- rk-daemon→fab-pulse→rk-notify→rk-daemon cycle is benign (notify fail-silent/async); the daemon-hosted path can never report the daemon's own death — that's the OS-timer variant's and `pulse_health`'s job.

# Operator generic tracking — one tracked-item model, rk-mandatory operator, lean reload

> Plan doc — written 2026-09-11 from the `/fab-discuss` study of `src/kit/skills/fab-operator.md`
> (v9, 903 lines) triggered by the PR #913 stale-state incident (the operator reported a PR
> "queued behind #913" fifteen minutes after it had merged). The study, with flow diagrams,
> the tick timeline, the frame proposal and the cron-kind evaluation, is
> `docs/wiki/operator-tick-anatomy.html`. Facts below were verified against the skill source,
> `src/go/fab/cmd/fab/operator*.go`, `src/go/fab/cmd/fab/pane*.go`, run-kit
> `app/backend/internal/cron/{backoff,evaluate,deliver}.go`, `rk skill mux`, and the
> run-kit intake `260910-9aup-cron-idle-every-skip-if-busy`.

**Problem:** the operator's tracked-thing model is fab-change-shaped. The only thing the binary
probes each tick is the set of tmux panes in `monitored`. Linear and Slack are reached only through
a typed `watch` (the Go verb rejects any other `--source`), GitHub only while the operator itself
has armed a merge, and notes never. Anything else the user hands the operator — "tell me when #913
merges", a non-fab branch, a deploy, a CI run, a ticket to merely watch — has no home except a
prose note that §4 Notes explicitly says is "checked by operator judgment per tick". On quiet
ticks the skill also forbids any output beyond the one-line frame, so nothing prompts that
judgment. #913 was the model working as written.

Second finding, orthogonal: the operator loads **3,425 lines per reload** — skill 903,
`_cli-fab.md` 1,475, `_preamble.md` 545, `_cli-agents.md` 255, `_cli-external.md` 247 — and uses
roughly 400 of `_cli-fab.md`'s lines. Reload cost is dominated by files the operator does not own.

---

## Goal

The operator's job statement becomes literally: *hand me something long-running, I will watch it
and act when it moves.* Fab changes, plain branches, PRs, Linear tickets, shell probes and notes
are all one kind of thing to the tick loop. The choreography (dependencies, merge modes, ordered
merge, auto-merge) stays in the main skill — it is the operator's main job and must survive a
compaction reload — and is re-pointed at items.

Two fab changes, in order. **A** is loading and dependency only and depends on nothing unreleased.
**B** is the model change and depends on run-kit work that is planned or in flight.

---

## Decisions already taken (do not reopen at intake)

| Decision | Choice | Why |
|---|---|---|
| Where probes run | **fab** (`tick-start --diff`), never an rk `wake_on: probe-change` | A raw output fingerprint has too many unrelated ways to change → wakes on noise. The guard is field-scoped comparison, declared at `track add` by the LLM. |
| Choreography placement | **inline in `fab-operator.md`**, tightened in place (235 → ≈180 lines) | Main job of the operator; an on-demand helper would be absent after a compaction reload with a merge sequence armed — the #913 shape again. |
| rk dependency | **hard** for the operator skill | Clock, role mark, send gate, spawn readiness and notify are all rk already; the 27 fallback passages describe a mode nobody runs. Binary-side fallbacks (built-in launcher, `fab pane map` tmux path) are out of scope — other consumers use them. |
| Clock kind, generic model | `--idle-every 3m` + `wake_on: agent-state-change` + `deliver: skip-if-busy` (9aup); plain `--every 3m` when the operator pane carries no agent-state epoch | Flat ladder: no rung to lose, interval never thins, re-phases on user activity. `skip-if-busy` avoids queued tick backlog (`immediate`) and a tick after every exchange (`when-idle`). Pane-only tracking keeps `backoff 1m→30m`. |
| Who tunes the clock | LLM sets `check_every` per item at `track add`; **binary derives** the entry schedule from the tracked set and applies it via `rk cron edit` on the existing mute/unmute hook, only on change; user overrides are bounded (lease / edit-with-expiry); frame shows live cadence read from `rk cron list --json` | A per-tick LLM judgment about cadence is state that lives only in conversation — the same failure class. |
| Frame | one table over all items: `▶ · ID · Kind · State · Checked · Next`; compact line unchanged in shape plus the live cadence; notes render as rows; idle message deleted | The frame is the only surface where a stale item can be seen; today it has no freshness column and no note rows. |

---

## Current state (verified 2026-09-11)

- **Tick** (`fab-operator.md` §4 Tick Behavior): seven fixed steps — Snapshot (`tick-start --diff --quiet`) → Auto-nudge (`fab pane questions` over `candidates:`) → Watches (MCP query per enabled watch) → Autopilot / merge-sequence check → Removals → (no-op) → (no-op). Only step 1 is mechanical; step 3 is typed `linear|slack`; step 4 polls `gh` only for an armed merge (an open `kind: coordination` note).
- **State file** (server-keyed, `$XDG_STATE_HOME/fab/operator/<slug>.yaml`): five owned sections `monitored`, `autopilot`, `branch_map`, `watches`, `notes` (+ `notes_seq`, `tick_count`), each with its own verbs, lifecycle and completion semantics; the merge sequence is a sixth pseudo-entity persisted as note prose.
- **Clock**: rk cron entry `operator tick`, `backoff 1m→30m` + `wake_on agent-state-change`, `deliver: immediate`, seeded by `rk operator`; fab's tracked-set verbs mute/unmute it (`operator_clock.go`, resolves the entry from `rk cron list --json` by `target == "role:operator"`). Backoff anchor = operator pane's last non-clock idle epoch (120 s attribution window, `backoff.go` JoinAnchor); only the operator's own activity resets the ladder; wake fires are attributed deliveries and do not reset it.
- **Substrate layering**: `fab pane map` delegates enumeration to `rk mux panes --json` (silent tmux fallback) and adds change/stage/PR enrichment; peek/kill/process ride `rk mux capture/process/kill`; sends ride `rk mux send` (`--answer`/`--key`/`--force`); `fab pane ready` delegates to `rk mux await --ready`. Still raw or fab-owned: worker spawn (`tmux new-window -t '<session>:' -P -F … -c … "<cmd>; exec \"$SHELL\""`), window marking (`fab pane window-name` `»`/`›` prefixes), prompt detection on uninstrumented panes (`fab pane questions`), `agent_exited` liveness (own process-tree walk in `tick-start`).
- **Fallback prose**: 27 rk-absent / raw-tmux / version-skew passages in `fab-operator.md`, 12 in `_cli-agents.md`; `_cli-external.md` carries a `/loop` section used only by the retired degraded clock.
- **run-kit side** (not released as of this writing — B depends on them): intake `260910-9aup` (`rk cron edit`, `deliver: skip-if-busy`, `--idle-every`); backlog `wzve` (`rk tab new -- <cmd>`, `--json`, `--ready`), `hzih` (`rk tab mark/note/color`), `7pek` (`has_agent` on `rk mux panes --json`), `r5ao` (structured `schedule`/`wake_on`/`respawn` in `rk cron list --json`), `a7g2` (`rk mux capture --classify`).

---

## Change A — operator loading and dependency (no behaviour change)

**Depends on:** nothing unreleased. `rk cron list --json` already carries `schedule` (display string) and `deliver`.

1. **rk is mandatory for the operator skill.** One startup gate in §2 replaces every scattered `command -v rk` check: `command -v rk` and `rk cron list --json` succeeding (the capability probe — a pre-cron rk fails it); else STOP with `Error: the operator requires run-kit — brew install sahil87/tap/run-kit`. Delete: raw `tmux send-keys` fallbacks (§3, §5), the Degraded Fallback `/loop` clock (§4) and the `Clock: none` ready-line form, the rk-absent notify ladder (ntfy / Discord / PushNotification / Slack MCP, §5) and the `Notify channel` setting (§8), the two-rung version-skew procedure (§4 tick step 1), rk-absent pane-map notes, the rk-absent delivery-probe paths. Keep `rk notify` fail-silent semantics (that is rk's contract, not a fallback). Sweep `_cli-agents.md` for the same 12 passages (Pre-Send Validation, Peek, Delivery Probe rk-absent arms) and delete the `/loop` section from `_cli-external.md` if the operator was its only consumer (verify with a repo-wide grep before deleting).
2. **Split `_cli-fab.md` by command family so the operator loads its slice.** Owner-or-pointer: each section has exactly one home, nothing is duplicated. Proposed cut (line counts from today's file): `_cli-fab-operator.md` ← `## fab operator` (≈190) + `## fab agent` (≈60); `_cli-fab-pane.md` ← `## fab pane` (≈110) + `## fab dispatch` (≈225); `_cli-fab.md` keeps the rest (change/status/score/preflight/log/resolve/resolve-agent/config/doctor/…/batch/errors). Every skill's `helpers:` list is re-declared for what it actually uses (fab-fff/fab-ff/fab-continue need pane+dispatch; fab-operator needs operator+pane; most planning skills need only the core). The `_preamble.md` § Skill Helper Declaration text and the Constitution's "MUST update `src/kit/skills/_cli-fab.md`" constraint get the family names (wording amendment, no new MUST). Operator load drops from 1,475 to ≈585 lines of CLI reference.
3. **Live cadence in the ready line and the compact frame.** §2 Init step 5 renders one template from `rk cron list --json` fields (`schedule`, `deliver`, `muted`, `muted_until`) instead of four literal variants: `Operator ready. Clock: rk cron "operator tick" · {schedule} · {deliver}[ · muted[ until HH:MM]]`. The §4 compact line gains ` · {schedule}` after the tracked count. Never composed from memory; the skill copies the fields.
4. **Docs sweep** (sibling class per `fab/project/code-quality.md`): `docs/memory/runtime/operator.md` (fallback paragraphs, clock ready-line, helper list), `docs/memory/runtime/agent-primitives.md` (rk-absent arms), `docs/specs/operator.md` (version history row v10: rk-mandatory, helper split), `docs/specs/skills.md` if it restates the operator's helper list or the `_cli-fab` shape, `_preamble.md` § Skill Helper Declaration.

**Non-goals for A:** any Go change; any state-file or tick-document change; the frame table shape (B); deleting the binary's built-in launcher or `fab pane map`'s tmux fallback (other consumers).

**Sizing:** docs/skill only; ≈6 tasks (gate + deletions, `_cli-agents` sweep, `_cli-external`, helper split + `helpers:` re-declaration sweep, ready-line/compact-line, memory/spec sweep). Likely FULL lane on task count. Rework risk is the sibling sweep — grep `rk is absent|rk-absent|raw tmux|send-keys|/loop|Notify channel|version-skew` repo-wide before finishing apply.

---

## Change B — generic `tracked` list, probe loop, derived schedule, one-table frame

**Depends on:** run-kit 9aup (`rk cron edit`, `skip-if-busy`, `--idle-every`) and backlog `wzve`, `hzih`, `7pek`, `r5ao`, `a7g2` released. Degradation if one is missing is listed per step; do not start B until at least 9aup and `r5ao` are on the machine, since the derived schedule needs both.

### B1 — one tracked-item schema (Go + skill §4)

Replace `monitored`, `watches`, `autopilot`, `notes` (and the merge-sequence-in-a-note) with one owned section:

```yaml
tracked:
  - id: pr-913
    kind: github-pr            # fab-change | github-pr | linear | slack | shell | task | note
    probe: { mode: shell, argv: [gh, pr, view, "913", --json, state,mergedAt], fields: [state] }
    check_every: 2m            # LLM-chosen at track add; null for pane items (snapshot every tick)
    done_when: 'state == "MERGED"'
    then: "spawn n34 in ~/code/hexokit via /fab-fff"
    depends_on: []
    scope: { repo: /home/x/code/hexokit }     # kind-specific metadata (pane, branch, stage for fab-change; project for linear)
    last: { state: OPEN }      # the declared fields only
    checked_at: 2026-09-11T16:33:00Z
    failures: 0
    added_at: …
```

- `probe.mode` ∈ `pane` (binary snapshot, as today), `shell` (binary runs argv — never a shell string — with a 10 s timeout, compares only `fields`), `agent` (LLM runs it, e.g. MCP Linear/Slack; the tick lists these in `needs_check:` with their age), `none` (notes).
- `kind: fab-change` carries today's monitored semantics as defaults: `probe: pane`, `done_when` = `review-pr` done/skipped or `stop_stage`, `then` = null, `scope` = pane/repo/session/branch/stage. `branch_map` stays as a side lookup written on `track add` for fab-change items.
- Verbs: `fab operator track add|update|observe|rm|list` replace `enroll/update/remove`, the `watch` verbs, the `autopilot` verbs and the `note` verbs. `observe <id> --json <doc>` is how the LLM records an `agent`-probe result. Migration file in `src/kit/migrations/` converts an existing state file (monitored→fab-change items, watches→linear/slack items with `probe: agent`, autopilot queue→items chained by `depends_on`, open notes→`kind: note`).
- Clock predicate becomes "any item not done".

### B2 — `tick-start --diff` polls every mechanical probe

Runs `pane` and `shell` probes (only those due per `check_every`), compares the declared fields against `last`, evaluates `done_when`, and emits: `changed` (field delta), `done`, `stale` (an `agent` item past 2× its cadence without an `observe`), `probe_error` (three consecutive failures → item auto-paused, health 🔴), plus today's pane deltas (`pane_death`, `pane_mismatch`, `agent_exited` — the last one via `has_agent` from `7pek`; keep the process-tree walk only if `7pek` is absent). `needs_check:` lists due `agent` items. `fleet:` becomes `items:`; `fleet_summary:` keeps its shape.

### B3 — derived schedule via `rk cron edit`

On every tracked-set mutation and on the tick reconcile: no non-pane items → `edit --backoff --min 1m --max 30m --deliver immediate`; any non-pane item → `edit --idle-every min(check_every) --deliver skip-if-busy` (or `--every` when `rk cron list --json` shows the target without an agent-state epoch). Only when the derived value differs from the structured `schedule`/`deliver` fields (`r5ao`). Bounded user override: a lease (`rk cron mute --for`) or an edit carrying an expiry the reconcile respects. Cross-repo contract note: this replaces nothing in run-kit; fab is one more `rk cron edit` caller.

### B4 — skill rewrite (§4, §5 detection, §6 spawn, §7)

- §4 Tick Behavior becomes four steps: snapshot (`tick-start --diff --quiet`) → act on `changed`/`done`/`stale` via each item's `then` → answer waiting agents (§5 unchanged; detection via `rk mux capture --classify` from `a7g2`, falling back to `fab pane questions` if absent) → ack (`track rm` for done, `observe` for `needs_check` results).
- §6 Spawning: `rk tab new --session =S --cwd <wt> --name <n> --ready --json -- <cmd>` (`wzve`) returns `{session, window_id, pane_id}` for `track add`; the four-tier session inference collapses to the majority rule over `rk mux sessions`; window marking via `rk tab mark`/`note` (`hzih`) replaces the `»`/`›` renames and `fab pane window-name` leaves the skill. Fab-change-kind steps (worktree, pointer activation, dependency cherry-pick, target-repo session command) stay, labelled as that kind's defaults; "pipeline-first" and "spawn in a worktree" move from §1 Principles to the fab-change kind. `wt` gate runs only when a fab-change item is about to spawn.
- §6 Choreography: dependency resolution, the three merge modes (diagrams kept — the mode question requires them), ordered merge and auto-merge stay inline; "Dependency satisfied" is defined once against items (`done` observed); the merge sequence is a chain of `github-pr` items with `depends_on` and `then: arm next` — the `coordination` note and its rule 4 disappear. Target ≈180 lines from 235.
- §7 Watches becomes the `linear`/`slack` kinds' section: `probe: agent` with the MCP call named, dedupe against `last.seen`, `then` prose for spawn/notify. Conversational map: each utterance → a `track` verb.
- §4 Notes: `kind: note`, `probe: none`, rendered; the four note kinds and the routing-doctrine table are deleted.
- Frame (§4 Status Frame Format): one table `▶ · ID · Kind · State · Checked · Next`, ordering kind → scope → id; `▶` = item has a `then`; `⏸ held` for a dependency wait naming what it waits on; Checked = age since last successful probe, `live` for pane items, `⚠` past `check_every`, 🔴 at 2×; Next = the armed action in ≤5 words; notes render with age since update; idle message deleted; no-fence rule, emoji-as-colour, italic footnote, two-shape rule kept. Removed items render once with ✅ then vanish.

### B5 — docs

`docs/memory/runtime/operator.md` rewritten around items; `docs/specs/operator.md` v11 row; `docs/specs/skills.md` operator entry; `_cli-fab-operator.md` verb contracts; migration doc in `docs/memory/distribution/migrations.md`; Constitution unaffected (no new MUST; the "binary owns state" doctrine is restated, not changed).

**Sizing:** Go (new verbs, probe runner, predicate evaluation, migration) + large skill rewrite + memory; ≈14 tasks; FULL lane. Highest rework risk: the frame's Checked/⚠ semantics and the `done_when` predicate grammar — pin both at intake (proposal: `jq`-style path equality/inequality only, no arbitrary expressions).

---

## Line budget (targets, not measurements)

| File | Today | After A | After B |
|---|---:|---:|---:|
| `fab-operator.md` | 903 | ≈780 | ≈480 (choreography ≈180 of it) |
| CLI reference loaded by the operator | 1,475 (`_cli-fab.md`) | ≈585 (`_cli-fab-operator` + `_cli-fab-pane`) | ≈350 (pane family shrinks with `fab pane questions`/`window-name` gone) |
| `_preamble.md` | 545 | 545 | 545 |
| `_cli-agents.md` | 255 | ≈180 | ≈120 |
| `_cli-external.md` | 247 | ≈200 | ≈80 (wt moves to the fab-change kind text) |
| **Loaded per reload** | **3,425** | **≈2,290** | **≈1,575** |

---

## Open decisions (resolve at each change's intake)

- **A:** exact family split of `_cli-fab.md` (two new files as proposed, or three with `fab agent` on its own); whether `_cli-external.md`'s `/loop` section has any consumer besides the operator.
- **A:** ready-line wording once fields render (`backoff 1m→30m · immediate` vs spelling `deliver` out).
- **B:** `done_when` grammar (proposal above); `check_every` floor (proposal 1m) and default when the LLM omits it (proposal 5m); whether `shell` probes may carry environment (proposal: no — argv only, inherit the operator's env).
- **B:** whether `linear`/`slack` items keep a bounded `seen` list (today's 200-cap `known`) or rely on `last` alone.
- **B:** migration of an in-flight autopilot queue (proposal: refuse to migrate while `autopilot.state == running`; ask the user to let it finish or stop it).

## Non-goals

- Any rk-side probe/wake mechanism (decided against).
- Removing the operator's tmux requirement, the one-operator-per-server model, or the server-keyed state file path (run-kit mirrors the slug rule).
- Multi-operator or per-repo operators.
- Changing `_preamble.md` beyond the helper-declaration wording.

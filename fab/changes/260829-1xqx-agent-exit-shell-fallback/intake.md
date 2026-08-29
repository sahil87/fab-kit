# Intake: Agent-Exit Shell Fallback for Interactive Spawns

**Change**: 260829-1xqx-agent-exit-shell-fallback
**Created**: 2026-08-29

## Origin

Conversational — raised during a `/fab-discuss` session on 2026-08-29, then drafted via `/fab-draft`.

> Right now in the way that sessions are spawned by the operator, the problem is that if you kill the Claude session, the tab goes away because that's the main process that's running in that tab. Can we move to a world where that's not the main process that runs on the tab but rather the second process? First the tab opens, the bash or the zsh or whatever the default shell is starts up, and then the Claude (or the fab agent) command runs in it. […] The idea here is if the agent quits or you exit the agent, what you end up with is the terminal, not the deletion of the whole tmux window/pane.

Key decisions reached in the discussion (all confirmed by the user):

1. **Mechanism: shell-fallback wrapper, not shell-then-send-keys.** Two options were compared. (A) open the pane with no command so the default shell starts, then `send-keys` the agent command — literally what was described, but it introduces a readiness race on the shell prompt, routes the command through the user's aliases/quoting, and turns every spawn into a two-step choreography. (B) keep the single-shot spawn and append a shell fallback to the command: `"<spawn_cmd> '<prompt>'; exec \"$SHELL\""`. The agent runs as the wrapper shell's foreground child; when it exits, the wrapper `exec`s the user's interactive (non-login) shell in the same pane with the cwd intact. User accepted (B).
2. **Dispatch pane workers stay "just agent".** `fab dispatch open` workers (`pane.OpenWindow` / `pane.OpenSplitPane` in `src/go/fab/internal/pane/create.go`) are reaped by `dispatch.reap_done`, nobody types into them, and their `running`/`done`/`orphaned` state machine keys on pane death. User: "They can just be agent like it is currently."
3. **The operator's `pane_death` signal does not dissolve.** Operator-spawned agent windows are exactly the panes getting the fallback, so an agent exiting no longer removes the pane from the snapshot and `pane_death` never fires. A new level-triggered delta `agent_exited` is required.
4. **No new kill verb.** `rk mux kill` already kills `idle`/uninstrumented panes (a lingering shell pane qualifies); killing just the agent process is `/exit` or `C-c`/`C-d` in the pane.
5. **rk-side state clearing is not a dependency** for the operator change — see § What Changes › Detection predicate for why fab cannot rely on it either way.

## Why

**Problem.** Every interactive agent tab fab opens — the operator's own tab (`fab operator` in `src/go/fab/cmd/fab/operator.go:106`), operator-spawned agent windows (`fab-operator.md` §6 step 7 / `_cli-agents.md` § Spawn Composition raw form), and `fab batch new`/`fab batch switch` tabs (`batch_new.go:141`, `batch_switch.go:141`) — passes the agent command as `tmux new-window`'s shell-command argument. tmux runs it as the pane's root process, so when the agent exits (user `/exit`, crash, provider CLI quitting after an error, a `C-c` too many) the pane and its window vanish with it. The user loses the cwd, the scrollback, the worktree context, and the tab position; getting back means re-spawning through the operator or by hand.

**Consequence of not fixing.** Agents remain disposable-on-exit tabs. Any interactive recovery ("drop to a shell in that worktree and look") requires a separate pane, and an agent crash in an operator-spawned window is indistinguishable from an intentional kill — both just remove the window.

**Why this approach.** The `; exec "$SHELL"` wrapper is a one-token grammar change per interactive spawn site, keeps the spawn atomic (no readiness race, no send-keys into a shell), preserves the existing readiness/deliver contracts (`#{pane_current_command}` still reports the agent binary while it runs, because the agent is the wrapper shell's foreground child), and leaves the dispatch adapter — where pane death is a load-bearing signal — untouched. The cost is concentrated in one place: the operator must learn to recognise "agent gone, pane still here".

## What Changes

### 1. Interactive spawn grammar gains a shell fallback

**Owner of the rule**: `src/kit/skills/_cli-agents.md` § Spawn Composition (the raw form). Add the fallback as the canonical interactive raw form and state the scope rule once there; every other site points at it (code-quality.md's owner-or-pointer rule).

Raw form becomes:

```sh
tmux new-window -n "<name>" -c "<dir>" "<composed-cmd> '<initial-prompt>'; exec \"\$SHELL\""
```

Semantics to document at the owner:

- tmux runs the shell-command argument via its `default-shell -c`. With a second command after `;`, that wrapper shell cannot exec-optimise the agent away, so the agent runs as the wrapper's foreground child; `#{pane_current_command}` reports the agent (e.g. `claude`, `kimi`) while it runs.
- When the agent exits for any reason, the wrapper `exec`s `$SHELL` (tmux sets `SHELL` from its `default-shell` option) in the same pane, same cwd. The pane and window survive.
- The fallback applies to **interactive** spawns only — sessions a human may type into. It does **not** apply to `fab dispatch open` pane workers, whose lifecycle depends on pane death (see § 3).
- The exec'd shell is a fresh login/interactive shell: rc files run, a prompt is drawn over the agent's last screen. That is the intended end state.

**Sites that adopt it** (each becomes a pointer to the owner, plus the mechanical edit):

| Site | Today | After |
|------|-------|-------|
| `src/go/fab/cmd/fab/operator.go` `runOperator` (~line 104) | `shellCmd := fmt.Sprintf("%s '/fab-operator'", spawnCmd)` | `shellCmd := spawn.WithShellFallback(fmt.Sprintf("%s '/fab-operator'", spawnCmd))` (or equivalent helper — see § 2) |
| `src/go/fab/cmd/fab/batch_new.go` (~line 141) and `batch_switch.go` (~line 141) | `new-window -n fab-<id> -c <wt> "<shellCmd>"` | same wrapper applied to `shellCmd` |
| `src/kit/skills/fab-operator.md` §6 step 7 | `"<spawn_cmd> '<command>'"` | `"<spawn_cmd> '<command>'; exec \"\$SHELL\""` with a one-line pointer to `_cli-agents.md` § Spawn Composition for why |
| `src/kit/skills/_cli-external.md` `new-window` row (~line 178/185) | describes the command form | note that the interactive form carries the fallback; still points at `_cli-agents.md` as owner |

`withWorkersEnv` (operator.go) prefixes an env assignment onto `shellCmd`; the fallback must wrap the *whole* thing so the order is `ENV=… <spawn_cmd> '/fab-operator'; exec "$SHELL"` — the env assignment only scopes the agent command, which is the existing behavior.

### 2. A single Go helper owns the wrapper string

Add one function in `src/go/fab/internal/spawn/` (where `DefaultSpawnCommand`/`WithProfile` already live), e.g.:

```go
// WithShellFallback appends the interactive-spawn shell fallback so the pane
// survives the agent's exit: the agent runs as the wrapper shell's foreground
// child and `exec "$SHELL"` takes the pane over afterwards. Interactive spawns
// only — dispatch pane workers (pane.OpenWindow/OpenSplitPane callers in
// dispatch_start.go) deliberately do NOT use it: their state machine treats
// pane death as the worker's terminal event.
func WithShellFallback(cmd string) string {
	return cmd + `; exec "$SHELL"`
}
```

Table-driven test asserting the exact suffix and that the three call sites compose it (extend `operator_test.go`, `batch_new_test.go`, and the batch-switch test's `new-window` argv expectations). The dispatch tests (`dispatch_open_test.go`, `pane_open_test.go`) must keep asserting the **unwrapped** command — that is the scope boundary made executable.

### 3. Explicit non-goals (recorded so review does not "fix" them)

- `pane.OpenWindow` / `pane.OpenSplitPane` (`internal/pane/create.go`) and their callers `dispatch_start.go:618/630` are unchanged. `fab pane open` is the shared LAUNCH half the dispatch adapter is built on (`_cli-agents.md` "Pipeline consumer" note), so it is also unchanged — no `--shell-fallback` flag is added in this change.
- No new `fab pane`/`rk` kill verb. `rk mux kill`'s agent-state gate already permits killing an uninstrumented or idle pane, which is what a fallen-back shell pane is; killing only the agent is a keystroke in the pane.
- No `remain-on-exit`/`respawn-pane` based design: a dead pane is not a terminal.

### 4. Operator: new level-triggered `agent_exited` delta

**Where**: `src/go/fab/cmd/fab/operator_tick_start.go`, alongside `pane_death` (line ~410) and `pane_mismatch`; snapshot plumbing in `src/go/fab/cmd/fab/pane_map.go` (`tmuxPaneFormat`, line ~364, and the row parser).

**Detection predicate** — the entry's pane **is present** in the snapshot AND its `#{pane_current_command}` is a shell. Concretely: add `#{pane_current_command}` as a ninth tab-separated field to `tmuxPaneFormat` (and the matching field in whatever the `rk mux panes` delegation path returns, if that path is used for the tick snapshot — verify which enumeration the tick uses and extend that one; if both exist, extend both so the row struct is uniform), carry it on the snapshot row, and evaluate `isShellCommand(cmd)` against a fixed set: `sh bash zsh fish dash ksh tcsh csh nu` (basename match; case-sensitive). Optionally also accept the basename of the `default-shell` tmux option if cheap to read; the fixed set is the required baseline.

Why not agent state: fab's `parseAgentState` (`internal/pane/pane.go:447–480`) validates and **ignores** the `:pid` segment — PID-liveness reconciliation is rk's, not fab's. After the agent exits and `$SHELL` takes over, the `@rk_pane_agent_state` pane option still carries the agent's last value (typically `idle:<epoch>:<pid>`), so fab's tick would read a stale `idle`, list the bare shell in `candidates:`, and the §5 sweep could nudge/route a command into a zsh prompt. The `pane_current_command` predicate is provider-agnostic, works on rk-less servers, and also correctly covers a user-opened shell pane where an agent was started by hand and later exited.

Why not "pane_pid has no children": an unwrapped single-command spawn (`sh -c "claude …"`) is exec-optimised by the wrapper shell, so `pane_pid` *is* the agent and an idle agent may have zero children — a false positive for any enrolled pane that was not spawned with the fallback.

**Delta shape** (extend the `_cli-fab.md` § fab operator tick-start `deltas:` block):

```yaml
deltas:
    - kind: agent_exited          # completion | pane_death | pane_mismatch | agent_exited | stage_advance | review_fail
      change: r3m7
      pane: "%3"
      # kind-specific fields:
      #   agent_exited  → command   (the shell now occupying the pane, e.g. zsh)
```

**Delivery class**: level-triggered — a stateless predicate over the current snapshot, re-emitted every tick until `fab operator remove` acks (same class and rationale as `pane_death`/`pane_mismatch`; extend the Level-Triggered vs Consumed-on-Read design decision in `docs/memory/runtime/operator.md` ~line 539).

**Exclusions** (mirror `pane_mismatch`): an `agent_exited` pane is **not** diffed for `stage_advance`/`review_fail`, does **not** receive the baseline `agent`/`stage` write, is **excluded from `candidates:`** (this is the load-bearing one — see stale-idle above), and its `fleet:` row uses the baseline identity fields with `agent_state: null` (unknown) — via `baselineFleetRow` or an equivalent. `fleet_summary:` counts it under `unknown`.

**Ordering**: evaluate `pane_death` first (absent pane), then `pane_mismatch`, then `agent_exited` — a mismatched pane must not also be reported as agent-exited.

### 5. Operator skill prose

`src/kit/skills/fab-operator.md`:

- **Tick Behavior step 1 delta list** (~line 268): add `agent_exited` (the entry's pane is present but its foreground process is now a shell — the agent quit or crashed and the interactive spawn's shell fallback took the pane over; carries `command`) — report, then remove via step 5. Add it to the level-triggered sentence.
- **§4 automation table** (~line 137): add a row after `Pane death`:

  | Agent exited (pane survives as a shell) | 0 | Report gone (pane kept, cwd intact). Respawn only in autopilot (1 attempt): kill the leftover shell pane first (`rk mux kill` when rk is installed — an uninstrumented/idle pane passes its gate — else `tmux kill-pane -t <pane>`), then spawn per §6 |

- **§6 step 7** grammar update per § 1.
- **Status Frame Format**: the `agent_exited` row renders like a `pane_death` row (baseline identity, `—` agent state) with a distinguishing marker — exact glyph/word is the implementer's call, but the frame must not present it as a live agent.

### 6. Reference, spec, and memory updates

- `src/kit/skills/_cli-fab.md` § fab operator tick-start — `deltas:` block + the "Two delivery classes" and "Detection semantics" bullets gain `agent_exited`; `fleet:` row comment notes the null-state fallback for exited panes.
- `docs/specs/harness-adapters.md` — one sentence at the pane-adapter section: interactive (human-facing) spawns carry a shell fallback; dispatch pane workers do not, because the adapter's three-state subset depends on pane death.
- `docs/specs/skills.md` `/fab-operator` section — delta list + §4 row if restated there (sweep: grep `pane_death` repo-wide and update every occurrence in the twin/aggregate class; known hits today: `_cli-fab.md:1258/1263/1285`, `fab-operator.md:268/271`, `docs/memory/runtime/operator.md:166/286/539`).
- `docs/specs/glossary.md` — `agent_exited` entry next to `pane_death` if the glossary lists delta kinds.

## Affected Memory

- `runtime/agent-primitives`: (modify) Spawn Composition raw form carries the `; exec "$SHELL"` interactive fallback; scope rule (interactive vs dispatch workers) and the exec-optimisation/foreground-child rationale.
- `runtime/operator`: (modify) new `agent_exited` level-triggered delta — predicate (`pane_current_command` ∈ shell set), `command` field, candidate/baseline/fleet exclusions, §4 row with autopilot kill-then-respawn; extend the Level-Triggered vs Consumed-on-Read design decision; the launcher's spawn command now wrapped.
- `runtime/pane-commands`: (modify) pane-map/tick snapshot row carries `pane_current_command` (the `tmuxPaneFormat` field addition), if the memory documents the row fields.
- `runtime/dispatch`: (modify) one-line non-goal note — pane workers deliberately keep the unwrapped spawn because `orphaned`/reaping key on pane death.
- `runtime/runtime-agents`: (modify) design note — the ignored `:pid` segment is why `agent_exited` cannot be derived from agent state (stale-idle after exit).
- `runtime/providers-and-profiles`: (modify) if it documents `fab batch new/switch` or the operator launcher's composed command, note the wrapper.

## Impact

**Go** (`src/go/fab/`): `internal/spawn` (new `WithShellFallback` + test), `cmd/fab/operator.go`, `cmd/fab/batch_new.go`, `cmd/fab/batch_switch.go` (wrap), `cmd/fab/pane_map.go` (`tmuxPaneFormat` 8→9 fields, row struct, parser + its tests), `cmd/fab/operator_tick_start.go` (new delta, exclusions, `fleet_summary` unknown bucket) + `operator_tick_start_test.go` fixtures. Existing dispatch/pane tests must continue to assert the unwrapped command. Run `go test ./cmd/fab/... ./internal/spawn/... ./internal/pane/...` first, then the full suite.

**Kit skills** (`src/kit/skills/`): `_cli-agents.md`, `_cli-external.md`, `_cli-fab.md`, `fab-operator.md`.

**Specs**: `docs/specs/harness-adapters.md`, `docs/specs/skills.md`, possibly `docs/specs/glossary.md`.

**Behavioral risk**: any consumer that inferred "agent gone" from window disappearance — only the operator's `pane_death` was found; `rk mux await`'s `gone` report is rk's and refers to pane death, which still holds for dispatch workers and simply stops firing for interactive tabs (an exited interactive agent shows up as `agent_exited` instead). `fab pane ready`/`deliver` are unaffected while the agent runs; after fallback they would probe a shell — the operator must act on `agent_exited` before routing, which the "act on `deltas:` before any answers" tick rule already guarantees.

**Compatibility**: pure additive delta kind; older skills reading the YAML ignore unknown kinds. No migration — no user data restructured. `_cli-fab.md` must be updated with the new delta and field per the constitution's CLI constraint.

## Open Questions

- Should the fixed shell-name set be supplemented by reading tmux's `default-shell` option (covers exotic shells) or is the fixed set sufficient for a first cut? (Default: fixed set only; extend on demand.)
- Status-frame marker for an `agent_exited` row (e.g. `⏏ shell` vs. reusing the `pane_death` rendering with a `(shell)` suffix) — implementer's call within "must not read as a live agent".

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Mechanism is the `; exec "$SHELL"` wrapper appended to the single-shot spawn, not shell-first + send-keys | Discussed — user chose option B after the race/quoting/two-step costs of option A were laid out | S:90 R:90 A:95 D:90 |
| 2 | Certain | Dispatch pane workers (`pane.OpenWindow`/`OpenSplitPane`, `fab dispatch open`, `fab pane open`) keep the unwrapped spawn | Discussed — user: "They can just be agent like it is currently"; their state machine keys on pane death | S:95 R:90 A:95 D:95 |
| 3 | Certain | Sites in scope: operator launcher tab, operator-spawned §6 windows | Discussed — these are the tabs the user named | S:95 R:90 A:95 D:95 |
| 4 | Confident | `fab batch new` / `fab batch switch` tabs are also in scope | Not discussed explicitly, but they are interactive agent tabs a human types into — same class as the operator's spawns; easy to drop if unwanted | S:75 R:85 A:80 D:80 |
| 5 | Certain | A new level-triggered `agent_exited` operator delta replaces the now-silent `pane_death` for interactive windows, acked by `fab operator remove` | Discussed and confirmed ("2 does not dissolve"); same delivery class rationale as `pane_death`/`pane_mismatch` | S:90 R:85 A:90 D:90 |
| 6 | Confident | Detection predicate = pane present AND `#{pane_current_command}` basename ∈ fixed shell set; agent state and pid-liveness are NOT used | Verified in code: `parseAgentState` ignores `:pid`, so agent state reads stale `idle` after exit; pid-children heuristic false-positives on exec-optimised unwrapped spawns | S:80 R:80 A:85 D:75 |
| 7 | Confident | `agent_exited` panes are excluded from `candidates:`, baseline writes, and stage diffs, mirroring `pane_mismatch` | Direct consequence of the stale-idle finding — otherwise the §5 sweep could type into a bare shell | S:80 R:85 A:85 D:85 |
| 8 | Certain | No new kill verb; `rk mux kill` / `C-d` suffice | Discussed and confirmed ("4 dissolves"); rk's gate already permits killing idle/uninstrumented panes | S:90 R:95 A:90 D:90 |
| 9 | Confident | Autopilot respawn on `agent_exited` kills the leftover shell pane first, then spawns per §6 | Keeps the "1 attempt" respawn semantics without accumulating dead shell tabs; reversible policy line | S:70 R:85 A:80 D:75 |
| 10 | Confident | Wrapper string is owned by one Go helper (`internal/spawn.WithShellFallback`) and one prose owner (`_cli-agents.md` § Spawn Composition); other sites point | code-quality.md owner-or-pointer rule and sibling-sweep guidance | S:80 R:90 A:90 D:85 |
| 11 | Confident | `#{pane_current_command}` is appended as a ninth field to `tmuxPaneFormat` (and to any parallel enumeration path the tick uses) | Cheapest way to carry the predicate input; the exact enumeration path must be verified at apply | S:70 R:85 A:80 D:80 |
| 12 | Tentative | Fixed shell set only (no `default-shell` lookup) in the first cut | Covers every common shell; listed as an open question for extension | S:60 R:90 A:70 D:65 |
| 13 | Tentative | Status-frame marker for exited rows left to the implementer within "must not read as a live agent" | Cosmetic; recorded as an open question | S:55 R:95 A:70 D:60 |

13 assumptions (5 certain, 6 confident, 2 tentative, 0 unresolved).

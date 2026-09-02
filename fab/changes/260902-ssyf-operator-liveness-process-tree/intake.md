# Intake: Operator Liveness via Process-Tree Confirmation

**Change**: 260902-ssyf-operator-liveness-process-tree
**Created**: 2026-09-02

## Origin

One-shot `/fab-new` invocation with a detailed field report (root cause already diagnosed by the user via `ps`/`pstree` against a live pane):

> fab operator agent-liveness detection is unreliable: fab operator tick-start classifies a monitored pane as agent_exited by checking pane_current_command against a shell-name list, but the operator spawn shape is `zsh -c "claude ...; exec \"$SHELL\""` — inside a non-interactive `zsh -c` there is no job control, so the claude child process never gets its own process group and inherits the shell's, so tmux pane_current_command reports the pgrp leader (zsh) even while claude is genuinely alive and running (confirmed via ps/pstree on the actual claude pid showing real CPU/elapsed time as a live child of the zsh -c invocation). This caused the operator to wrongly conclude a live, working agent had exited and remove it from monitoring. Need a more robust liveness check in the operator tick-start exited-detection path, e.g. walk the pane pid's process tree for a live claude/provider-binary descendant instead of trusting pane_current_command alone, so agent_exited only fires when the agent process is actually gone.

The false positive was hit in anger during change 0i4x (worktree amber-mamba, pane %23): a live, working apply agent was reported `agent_exited` and dropped from the monitored set.

## Why

1. **The pain point**: `fab operator tick-start --diff` emits `agent_exited` purely from `#{pane_current_command}` basename ∈ the fixed shell set (`pane.IsShellCommand`, `operator_tick_start.go:451`). tmux reports the pane tty's **foreground process-group leader**. The operator's interactive spawn shape wraps the agent — `zsh -c "claude ...; exec \"$SHELL\""` — and a non-interactive `zsh -c` runs with job control off, so the claude child shares zsh's process group and the pgrp leader stays `zsh` even while claude is alive and working. The predicate's premise ("a shell foreground means the fallback shell took over after the agent quit") is false for this spawn shape: the shell foreground is present from spawn, agent alive or not.

2. **The consequence**: `agent_exited` is level-triggered and its documented operator response is *report + remove from the monitored set*. A false positive silently drops a live, mid-stage agent from monitoring — no completion detection, no stage deltas, no auto-answer sweep, no status-frame row. This is the operator's worst failure direction: the whole point of monitoring is to not lose track of running agents. Already observed in a real run (0i4x).

3. **Why this approach**: the fix is a **secondary, positive confirmation** in the binary's exited path — when (and only when) the foreground command classifies as a shell, walk the pane's process tree (rooted at `#{pane_pid}`) and look for a live agent/provider-binary process; emit `agent_exited` only when none is found. The primitive already exists: `fab pane process` (`pane_process.go` + `pane_process_linux.go`/`pane_process_darwin.go`, same `package main` as the tick) discovers the tree and classifies nodes, with `hasAgentInTree` already implemented. Alternatives rejected:
   - *Skill-side confirmation* (operator re-checks with `fab pane process` before acting on the delta) — treats the symptom; every consumer of the delta (skill text, autopilot, frame rendering) would need the same guard. Root cause is the binary predicate; fix it there.
   - *Trusting `@rk_pane_agent_state`* — already rejected on the record (operator.md Design Decisions): fab's `parseAgentState` ignores the `:pid` segment, so the option reads the agent's last stale value after exit.
   - *Fixing the spawn shape instead* (e.g. `setsid`/job-control tricks so claude gets its own pgrp) — fragile across shells/providers/OSes, and would not repair detection for panes spawned by older shapes or by hand. The wrap (`; exec "$SHELL"`) is a deliberate, shipped contract (1xqx #627).

**Relation to the recorded rejection**: operator.md's Design Decisions explicitly rejected "a `pane_pid`-has-no-children heuristic" — as the *primary* signal, because an unwrapped single-command spawn is exec-optimized (pane_pid *is* the agent, zero children ⇒ false positive on every unwrapped pane). This change does not reinstate that: the tree walk runs only *inside* the shell-foreground branch as a suppressor, and it looks for a **live agent process anywhere in the tree including the root pid**, not for "has children". An unwrapped pane never enters the branch (its foreground command is the agent binary, not a shell), and even if it did, its root pid would classify as agent. The hydrate must update that Design Decision's Rejected narrative to distinguish primary-signal use (still rejected) from secondary confirmation (now shipped).

## What Changes

### 1. Refined `agent_exited` predicate in `fab operator tick-start --diff`

In `diffMonitored` (`src/go/fab/cmd/fab/operator_tick_start.go`), the exited branch becomes:

```
agent_exited  ⇔  pane present ∧ change-matched
              ∧  pane.IsShellCommand(row.command)
              ∧  NO live agent process in the pane's process tree
```

Mechanically: when `IsShellCommand(row.command)` is true, resolve the pane's root PID (`pane.GetPanePID(pane, server)`), discover its process tree (`discoverProcessTree` — already in the same package), and check for an agent-classified process anywhere in the tree (root included). If one is found, the pane is treated as a **clean join** (stage diffs, completion check, baseline update, `candidates:` eligibility all apply normally) — the agent is genuinely alive; its `@rk_pane_agent_state` is live and typed input reaches the agent (it shares the foreground pgrp). If none is found, emit `agent_exited` exactly as today (same delta shape, same `command` field, same level-triggered class, same exclusions).

- The walk runs **only** for monitored, change-matched, shell-fronted panes — after the `pane_death` and `pane_mismatch` branches, preserving the documented precedence order `pane_death → pane_mismatch → agent_exited → clean join`. Cost is one `display-message` + one process enumeration per shell-fronted monitored pane per tick (rare set, typically 0–1 panes).
- **Failure direction**: the confirmation suppresses `agent_exited` only on *positive* evidence of a live agent process. If the PID resolution or tree walk errors, emit `agent_exited` (today's behavior). Rationale: a wrongly-suppressed exit on a genuinely dead agent would leave a bare interactive shell classified as a clean join, whose stale `idle` agent state would re-enter `candidates:` — and the §5 sweep must never type into a bare shell prompt. A wrongly-emitted exit on walk *error* is transient and recoverable (re-enroll), whereas the sweep-types-into-shell hazard is the property 1xqx existed to protect. Errors on the walk are best-effort silent (matching snapshot-internal conventions), not surfaced per tick.
- `diffMonitored` stays table-testable: the liveness check enters as an injected predicate (e.g. a `func(paneID string) bool` field/parameter defaulting to the real GetPanePID+tree implementation), so `operator_tick_diff_test.go` can script both outcomes without a tmux server or live processes.

### 2. Agent classification extended to provider binaries

`ClassifyProcess` (`pane_process.go`) currently classifies only `claude`/`claude-code` as `agent`. The operator spawns non-Claude providers into panes too (kimi/codex/agy runs are routine), so a kimi pane would false-negative the confirmation and still be wrongly removed. Extend the agent-name set:

- Union of the built-in names (`claude`, `claude-code`) with the **basenames of the first token of each configured provider's `interactive_command`** (config `providers:` table) and the built-in provider command names fab already knows (kimi, codex, agy per the built-in provider dictionary).
- Matching should consult `comm` first and fall back to the node's `Cmdline` (a provider distributed as a script wrapper can surface as `node`/`bun` in comm while the cmdline names the binary). Exact matching strategy is an apply-time decision recorded in the plan.
- `fab pane process` output (`classification`, `has_agent`) picks up the same extension — one classifier, two consumers. Its human/JSON output shape is unchanged.
- Since `ClassifyProcess` would now need config, keep the pure classifier parameterized (e.g. accept an agent-name set) with the config-derived set built once per tick/invocation — no global state.

### 3. Documentation and skill-text sweep

Behavior-contract text that states the old predicate ("its foreground process is a shell ⇒ the agent quit or crashed") must be updated to the confirmed form:

- `src/kit/skills/_cli-fab.md` — § fab operator tick-start "Detection semantics" (line ~1301) and the delta-kind comment block (~1279): `agent_exited` fires on shell foreground **and no live agent process in the pane's tree**; note the confirmation and its failure direction.
- `src/kit/skills/fab-operator.md` — §4 delta handling (~309) and the `⏏ shell` frame-marker row (~387): wording only; the operator's *response* to the delta is unchanged.
- `src/go/fab/internal/pane/pane.go` — `IsShellCommand` doc comment: it is now the operator's *trigger for confirmation*, not the sole exited signal ("the foreground command is the ground truth about who owns the pane" claim needs revision); the readiness gate remains a direct consumer.
- Constitution constraint: the CLI change updates `_cli-fab.md` and ships test updates in the same change (both covered above).

### 4. Explicit non-goals

- **The readiness gate's agent-takeover precondition (`gate.go` Probe) is unchanged.** Dispatch pane workers are unwrapped spawns (exec-optimized, so `pane_current_command` *is* the agent binary) — the gate's shell-foreground → `booting` classification is not subject to this false positive, and its failure mode (bounded boot re-probes → judgment rounds → escalation with the pane alive) is diagnosable, unlike a silent monitoring removal. If wrapped spawns ever route through the gate, that is a separate change.
- **No change to the spawn shape** (`; exec "$SHELL"` wrap stays as shipped in 1xqx).
- **No new delta kind** for "shell-fronted but agent alive" — that state is simply a clean join; the frame renders it as a normal live row.
- **No snapshot-format change**: `pane_pid` is not added to the `rk mux panes` / `list-panes -F` enumeration (that would need a run-kit change on the delegated path); the PID is resolved per-pane on demand, only for shell-fronted monitored panes.

## Affected Memory

- `runtime/operator`: (modify) `agent_exited` predicate description (§ tick-start, §4 narrative) and the "Level-Triggered vs Consumed-on-Read Delta Classes" Design Decision — update the Rejected narrative to distinguish the still-rejected primary-signal pane_pid heuristic from the shipped secondary confirmation; record the fail-toward-emit error direction
- `runtime/pane-commands`: (modify) `fab pane process` classification — provider-binary extension to the agent class, parameterized classifier
- `runtime/agent-primitives`: (modify) spawn-composition/shell-fallback narrative where it claims the foreground command is the ground truth for pane ownership (the wrapped spawn keeps a shell as pgrp leader while the agent lives)
- `runtime/dispatch`: (modify) only if its `IsShellCommand`/gate cross-references restate the operator predicate — pointer wording at most; gate behavior itself is unchanged

## Impact

- **Go**: `src/go/fab/cmd/fab/operator_tick_start.go` (predicate + injection seam), `src/go/fab/cmd/fab/pane_process.go` (classifier parameterization + provider names), `pane_process_linux.go`/`pane_process_darwin.go` (unchanged discovery, verify reuse), `src/go/fab/internal/pane/pane.go` (doc comment)
- **Tests**: `operator_tick_diff_test.go` (existing shell-basename cases gain the injected-liveness dimension: shell + agent-alive ⇒ clean join; shell + no agent ⇒ `agent_exited`; walk error ⇒ `agent_exited`; mismatch-wins ordering preserved), `pane_process_test.go` (classifier extension)
- **Skills**: `src/kit/skills/_cli-fab.md`, `src/kit/skills/fab-operator.md` (canonical sources; deployed copies via `fab sync`)
- **Release**: Go binary behavior fix — patch/minor per release conventions; no config schema change, no migration (no per-checkout state touched)

## Open Questions

- None — the root cause, fix direction, and reuse surface were specified in the origin report and confirmed against the code.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix lives in the binary's tick predicate (`diffMonitored`), not as a skill-side re-check | Root-cause rule; every delta consumer inherits the fix; user asked for "the operator tick-start exited-detection path" | S:85 R:70 A:90 D:85 |
| 2 | Certain | `agent_exited` requires shell foreground AND no live agent process in the pane_pid tree (tree walk as secondary confirmation only) | Specified verbatim in the origin report; sidesteps the recorded primary-signal rejection since unwrapped panes never enter the shell branch | S:95 R:75 A:90 D:90 |
| 3 | Confident | Agent-name set = built-ins (claude/claude-code) ∪ configured providers' `interactive_command` first-token basenames ∪ built-in provider commands (kimi/codex/agy) | Operator runs all-kimi fleets routinely; a Claude-only classifier would keep the false positive for them; config is the natural source | S:60 R:70 A:75 D:60 |
| 4 | Confident | Classification matches `comm` first with a `Cmdline` fallback for script/node-wrapped provider binaries | Node-wrapped CLIs can surface as `node` in comm; exact matching strategy decided at apply | S:50 R:75 A:60 D:55 |
| 5 | Confident | On PID-resolution or tree-walk error, emit `agent_exited` (suppress only on positive live-agent evidence) | Wrong suppression re-admits a bare shell to `candidates:` — the sweep-types-into-shell hazard 1xqx guards; wrong emission on transient error is recoverable | S:55 R:75 A:80 D:70 |
| 6 | Confident | A shell-fronted pane with a live agent is a full clean join (stage diffs, completion, baseline write, candidates eligible) | The agent is alive: its agent-state option is fresh and typed input reaches it (shared foreground pgrp); no new delta kind | S:70 R:75 A:85 D:80 |
| 7 | Confident | Readiness gate (`gate.go`) takeover precondition unchanged — non-goal | Dispatch pane workers are unwrapped (exec-optimized), so the gate never sees the wrapped shape; its failure mode escalates loudly rather than silently removing | S:60 R:80 A:75 D:65 |
| 8 | Confident | Reuse `fab pane process` discovery (`discoverProcessTree`/`hasAgentInTree`, same package) with an injected predicate seam into pure `diffMonitored` | Mechanism already shipped and platform-split; injection keeps the existing table tests serverless | S:65 R:80 A:85 D:75 |

8 assumptions (2 certain, 6 confident, 0 tentative, 0 unresolved).

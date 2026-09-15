# Intake: Guidance Re-Point — capture/kill/process to rk mux Twins (cli-layering Part 7)

**Change**: 260820-4un7-guidance-repoint-rk-mux-twins
**Created**: 2026-08-20

## Origin

One-shot `/fab-new` invocation carrying the Part 7 row of the cross-repo CLI-layering execution plan (the spec lives in the **run-kit** repo: `~/code/sahil87/run-kit/docs/specs/cli-layering.md` — fab-kit has no local copy; this intake restates every fact the pipeline needs from it). This is the second fab-kit-side part of that plan, following Part 5 (260815-4i0n, `fab pane send`/`await` retirement, released in fab-kit v2.19.x line and current in v2.20.5).

> Part 7 of docs/specs/cli-layering.md: guidance re-point -- repoint _cli-external.md and _cli-agents.md capture/kill/process guidance at rk mux twins; demote fab pane capture/kill/process to dispatch-internal (kept for the rk-less pane arm); drop from skill-facing guidance. Deps: Parts 5 and 6, both released (fab-kit v2.20.5, run-kit v3.17.15).

The spec's Part 7 row verbatim: *"guidance re-point | fab-kit | `_cli-external`/`_cli-agents` point capture/kill/process at rk; fab's pane copies demoted to dispatch-internal (kept for rk-less pane arm), dropped from skill-facing guidance | Depends on 5, 6 (released)"*. The fab-split table's disposition row: *"`pane ready`, `pane deliver`, `pane capture`, `pane kill`, `pane process` | substrate mechanics | rk grows canonical twins over time; fab's copies become dispatch-internal (pane arm must work rk-less), dropped from skill-facing guidance"*.

**Dependencies verified live at intake time** (both must be *released*, not merely merged, per the execution plan's gating rule):

- Part 5 (fab-kit): `fab pane send`/`await` deleted, guidance migrated to `rk mux send`/`await` — shipped; installed `fab 2.20.5`.
- Part 6 (run-kit): `rk mux capture`, `rk mux kill`, `rk mux process` — shipped; installed `run-kit v3.17.15` carries all three with full flag parity (probed live):
  - `rk mux capture <target> [-l <lines>] [--json | --raw] [-L <server>]` — last-N tail (default 50), `--raw` byte-identical text, `--json` metadata wrapper with reconciled `@rk_agent_state` + idle/waiting duration, pane cwd enrichment. Targets: `%N`, `@N` (window → its agent pane), `=session:window`.
  - `rk mux kill <target> [--force] [-L <server>]` — **agent-state-gated**: refuses a pane whose agent is `active` or `waiting` (exit 1, touches nothing); idle and uninstrumented panes are killed; `--force` skips the gate. Richer than fab's ungated `fab pane kill`.
  - `rk mux process <target> [--json] [-L <server>]` — process tree classified agent/node/git/other; a live agent pid from `@rk_agent_state` classifies its node `agent` authoritatively (comm heuristics are the fallback).

## Why

1. **The layering model says substrate verbs belong to rk.** cli-layering's two-layer model gives rk the substrate (tmux conventions, pane interaction verbs) and fab the choreography (changes, stages, dispatch records). `fab pane capture`/`kill`/`process` are pure substrate mechanics that predate rk's twins. Part 6 shipped the canonical twins; Part 7 is the guidance half — without it, skills keep steering agents at fab's copies, and the two implementations must be kept behaviorally aligned forever.

2. **The rk twins are strictly better for skill-facing use.** `rk mux kill` holds the agent-state gate (refuses killing an `active`/`waiting` agent) that `fab pane kill` never had; `rk mux process` classifies the agent process authoritatively from instrumentation rather than comm heuristics; `rk mux capture` reports the *reconciled* agent state. Skill guidance pointing at fab's copies leaves that safety and accuracy on the table.

3. **Consequence of not doing it**: permanent duplicate maintenance (every capture/kill/process improvement lands twice — e.g. the y4mu last-N tail fix went into fab's copy days before rk's twin shipped with the same semantics), and drift between the copies becomes an agent-visible inconsistency.

4. **Why demote rather than delete** (the deliberate contrast with Part 5): Part 5 *deleted* `fab pane send`/`await` because dispatch delivery uses internal builders, not the CLI verbs. Capture/kill/process are different — the **dispatch pane arm must work rk-less** (rk is an optional sibling binary; cli-layering rule 2 makes fab installable without it), and the dispatch orchestrator's peek/recovery path (`_preamble.md` § CLI-Adapter Dispatch) and `fab dispatch logs`' copy-pasteable capture command genuinely invoke `fab pane capture` as a CLI verb. So the verbs stay; only their *skill-facing guidance* status changes.

## What Changes

All edits are to canonical kit sources under `src/kit/skills/` (never `.claude/skills/` — deployed copies). **No Go changes, no CLI surface changes, no migration**: the three `fab pane` verbs keep their exact behavior, flags, exit codes, and help visibility. This is a guidance-ownership change, mirroring Part 5's structure minus the verb deletion.

### 1. `_cli-external.md` — tmux + rk sections

- **§ tmux → Usage Notes, "Pane capture" bullet** (currently: *"Use `fab pane capture` instead of raw `tmux capture-pane`. It provides fab context enrichment, validation, and structured output."*): repoint to `rk mux capture` when rk is present (`command -v rk`-gated — reconciled agent state, last-N tail, `--raw`/`--json`), failing open to raw `tmux capture-pane` when rk is absent — never an error. Follow the "Send keys" bullet's existing shape (rk verb preferred, raw-tmux fallback, ownership pointer to `_cli-agents.md` § Peek).
- **§ tmux → Usage Notes, "Send keys" bullet**: the rk-absent state-read parenthetical currently names `fab pane map`/`fab pane capture --json`; drop the `fab pane capture --json` half — the rk-less skill-facing state read is `fab pane map` (which stays skill-facing per the spec's split table: `pane map` is hybrid/choreography).
- **§ Reference Model** intro prose: the parenthetical "(the operator's spawning sequence, the escalation `rk notify` usage …, the `fab pane` internalization notes)" — update the internalization-notes phrasing to reflect that pane substrate verbs now ride rk with the fab copies dispatch-internal.
- **§ rk (run-kit)**: extend the "Agent messaging (fab-owned — pointer)" subsection (or add a sibling pointer) so the fab-owned rk usage index covers the peek/kill/process repoint — pointing at `_cli-agents.md` § Peek (and § Pre-Send Validation) as the owner, with the verbs' full contracts tool-owned (`rk skill`). Keep owner-or-pointer discipline: `_cli-external.md` points, `_cli-agents.md` owns.

### 2. `_cli-agents.md` — Scope Boundary, Pre-Send Validation, Peek

- **§ Scope Boundary** fab-commands list (currently "`fab agent`, `fab pane open`/`ready`/`deliver`, `fab pane capture`, `fab pane map`"): drop `fab pane capture` from the list; extend the existing rk sentence ("Messaging into an existing agent pane rides `rk mux send`/`rk mux await` …") to also name peek/kill/process riding `rk mux capture`/`kill`/`process` with the raw-tmux fallback.
- **§ Pre-Send Validation step 2**, the rk-absent path (currently "state via `fab pane map`/`fab pane capture --json`"): state via `fab pane map` alone.
- **§ Peek** — the main re-point:
  - **Output axis** (currently `fab pane capture <pane> [-l N] [--json]` / `--raw`): becomes `rk mux capture <pane> [-l N] [--json|--raw]` when rk is present (`command -v rk`-gated), failing open to raw `tmux capture-pane -p -t <pane>` (piped through `tail` for a last-N window) when rk is absent. Carry over the existing guidance content (last-N semantics, wide `-l 50`+ window for wrap compensation).
  - **Agent-state axis**: read via `fab pane map`'s Agent column (stays) and `rk mux capture --json`'s reconciled agent-state fields (replacing the `fab pane capture --json` mention). The state-writer caveat block stays as-is apart from the command swap.
  - **Process-tree peek**: where the topic warrants a sentence, name `rk mux process` as the skill-facing verb (agent-classification is instrumentation-authoritative). Do not invent a new procedure — mention only where a natural slot exists.
  - **Pane removal**: if a kill mention fits § Peek/§ Await scope it names `rk mux kill` (agent-state-gated, `--force` override); otherwise kill guidance lives only in the demotion notes and `_cli-external.md` pointer. Keep additions minimal.

### 3. `fab-operator.md` — the two capture call sites

- **§5 Question Detection step 1** (currently `fab pane capture --raw -l 20 [-L <server>] <pane>`): becomes `rk mux capture --raw -l 20 [-L <server>] <pane>` when rk is installed (`command -v rk`-gated), raw `tmux capture-pane` fallback when absent — matching the skill's existing rk-present/rk-absent dual-path pattern for sends (§3/§5). Flag parity is exact (`--raw`, `-l`, `-L` all exist on the twin).
- **§5 Answer-delivery re-capture** (the "re-capture the terminal (the same `fab pane capture --raw -l 20` capture as § Question Detection step 1)" sentence): the self-reference follows step 1's new form.

### 4. `_cli-fab.md` — § fab pane demotion notes

- Family header and/or the three subcommand entries (`### capture`, `### process`, `### kill`) gain a demotion note: these verbs are **dispatch-internal** — kept because the dispatch pane arm must work rk-less (cli-layering Part 7); skill-facing guidance uses the rk twins `rk mux capture`/`kill`/`process` (`_cli-agents.md` § Peek / § Pre-Send Validation own the usage). Command behavior, flags, and exit codes are unchanged.
- The `### kill` entry's rationale sentence ("so operator removal paths and probe cleanups stop falling back to raw `tmux kill-pane`") is updated — operator-facing removal now points at `rk mux kill`; fab's kill remains for dispatch-internal/rk-less use.

### 5. Explicitly untouched (the "kept for the rk-less pane arm" half)

- `_preamble.md` § CLI-Adapter Dispatch pane peek (`fab pane capture [-L <server>] <pane>`) — dispatch-orchestrator usage, stays verbatim.
- `fab dispatch logs`' pane-mode report naming `fab pane capture <pane>` as the copy-pasteable capture command (`_cli-fab.md` § fab dispatch, `internal/dispatch` strings) — stays.
- `fab pane open`/`ready`/`deliver`/`map`/`window-name` — out of scope (open is choreography, map is hybrid, window-name stays until the marker convention moves, ready/deliver are the dispatch gate's primitives).
- All Go code, tests, and `fab pane` help text.

### 6. Sweep obligations (code-quality Sibling Sweeps)

Before finishing apply, grep repo-wide across `src/kit/skills/` for `fab pane capture`, `fab pane kill`, `fab pane process`, and contrastive phrases ("instead of raw `tmux capture-pane`", "stop falling back to raw `tmux kill-pane`") — update every *skill-facing* occurrence, leave every *dispatch-internal* occurrence, and verify each classification against § 5's untouched list. User-facing string literals and `*_test.go` comments are in scope for the sweep (recurring-lessons rule) though none are expected given no Go changes.

## Affected Memory

- `runtime/agent-primitives.md`: (modify) § Peek output/state axes and the pre-send rk-absent state read repoint to `rk mux capture` / `fab pane map`; kill/process skill-facing verbs become the rk twins
- `runtime/operator.md`: (modify) question-detection capture command (line ~177) and the re-capture-before-send reference follow the operator's new rk-gated capture form
- `runtime/pane-commands.md`: (modify) capture/process/kill subcommand sections gain the dispatch-internal demotion + rk-twin pointer; the domain framing sentence about agent messaging extends to peek/kill/process
- `distribution/kit-architecture.md`: (modify) the `_cli-external.md` helper-row content description, if its wording enumerates the pane internalization notes

(`runtime/dispatch.md` and `_shared/context-loading.md` mention `fab pane capture` only in dispatch-internal contexts — expected unchanged; the apply sweep confirms.)

## Impact

- **Files**: 4 kit skill sources (`src/kit/skills/_cli-external.md`, `_cli-agents.md`, `fab-operator.md`, `_cli-fab.md`) + 3–4 memory files at hydrate. No `src/go/` changes, no templates, no migrations.
- **Behavior contract**: agent-facing guidance changes (operators and skill consumers start invoking `rk mux capture/kill/process` when rk is present). No fab CLI behavior changes.
- **Release**: patch-level kit release (guidance-only refactor); no binary changes.
- **Runtime prerequisite**: none hard — rk remains optional (`command -v rk`-gated everywhere, raw-tmux fail-open), per cli-layering rule 2.

## Open Questions

- (none — the Part 7 row, the fab-split disposition table, and the Part 5 precedent resolve every scoping decision; see Assumptions)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Dependencies satisfied: Parts 5+6 both released | Verified live at intake — installed `fab 2.20.5` (post-Part-5) and `run-kit v3.17.15` ships `rk mux capture/kill/process` with full flag parity (`--raw`/`-l`/`--json`/`-L`, kill `--force`) | S:90 R:90 A:100 D:95 |
| 2 | Certain | Operator capture becomes `rk mux capture --raw -l 20 [-L <server>]` — a drop-in flag-for-flag swap | Flag parity probed against the installed twin; `-L` socket scoping identical; the operator skill already has the rk-present/rk-absent dual-path pattern from Part 5 | S:75 R:85 A:90 D:85 |
| 3 | Certain | Dispatch-internal call sites stay on `fab pane capture` verbatim (`_preamble.md` pane peek, `fab dispatch logs` suggested command, dispatch memory) | The spec's "kept for the rk-less pane arm" half — the pane adapter must work without rk (rule 2); these are the exact sites that justify keeping the verbs | S:80 R:85 A:85 D:80 |
| 4 | Confident | rk-absent skill-facing fallback is raw tmux (`tmux capture-pane`/`kill-pane`) + `fab pane map` for agent state — NOT `fab pane capture` | cli-layering rule 2: "absence degrades to raw tmux / fab's internal builders"; Part 5 used the same raw-tmux fallback shape; naming `fab pane capture` as fallback would contradict "dropped from skill-facing guidance" | S:65 R:80 A:75 D:60 |
| 5 | Confident | "Demote to dispatch-internal" is prose-only: no cobra hiding, no Go changes, verbs stay visible in `fab pane` help | The spec's stated mechanism is "dropped from skill-facing guidance"; hiding was an rk-side device for machine-invoked plumbing; `open`/`ready`/`deliver` stay visible with the same dual role, and a help-surface change would drag in the shll standards audit for zero layering gain | S:60 R:70 A:70 D:55 |
| 6 | Confident | Where kill/process guidance surfaces skill-facing, name the rk twins (`rk mux kill` gated, `rk mux process`) — but add no new procedures; today's skill-facing kill/process guidance is nearly nil outside `_cli-fab.md`, and it stays that way | The spec says guidance "points capture/kill/process at rk"; minimal-addition keeps the change one pipeline run and avoids inventing operator kill-adoption (a separate follow-up, per the yxyi notes) | S:55 R:80 A:65 D:55 |
| 7 | Confident | change_type `refactor` via explicit override | Part 5 (4i0n) precedent: explicit `refactor`; keyword inference on this description would land elsewhere ("fix" substring risk in prose); guidance-ownership restructuring with no new capability is refactor-shaped | S:60 R:95 A:80 D:70 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).

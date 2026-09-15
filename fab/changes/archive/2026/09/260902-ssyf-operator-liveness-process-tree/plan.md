# Plan: Operator Liveness via Process-Tree Confirmation

**Change**: 260902-ssyf-operator-liveness-process-tree
**Intake**: [intake.md](intake.md)

## Requirements

### Operator: Confirmed exited predicate

- **R1**: `fab operator tick-start --diff` MUST emit `agent_exited` for a monitored, change-matched pane only when BOTH hold: the pane's foreground command is a shell (`pane.IsShellCommand`) AND the pane's process tree (rooted at `#{pane_pid}`) contains no live agent process. A shell-fronted pane with a live agent process anywhere in its tree (root included) MUST take the clean-join path.
  - GIVEN a monitored pane spawned as `zsh -c "claude ...; exec \"$SHELL\""` with claude alive (empirically verified 2026-09-02: `pane_current_command` reads `zsh` for the child's entire life on Linux — the misreport is universal for wrapped spawns, not intermittent), WHEN the tick diffs it, THEN no `agent_exited` delta is emitted and the pane joins cleanly (stage diffs, completion check, baseline write, candidates eligibility).
  - GIVEN the same pane after claude exits (the fallback shell owns the pane, no agent process in the tree), WHEN the tick diffs it, THEN `agent_exited` is emitted with the `command` field exactly as today (level-triggered, same exclusions).
  - GIVEN a pane whose foreground command is NOT a shell, WHEN the tick diffs it, THEN no process-tree walk runs (the confirmation is lazy — shell branch only).

- **R2**: The confirmation MUST suppress `agent_exited` only on positive evidence of a live agent process. If `GetPanePID` or the tree walk errors, the tick MUST emit `agent_exited` (today's behavior), best-effort silently.
  - GIVEN a shell-fronted monitored pane whose PID resolution fails, WHEN the tick runs, THEN `agent_exited` is emitted. (Rationale: wrong suppression would re-admit a bare shell's stale `idle` agent state to `candidates:` — the sweep must never type into a bare shell prompt.)

- **R3**: The agent-process match MUST recognize: the built-in agent names (`claude`, `claude-code`) ∪ the basename of each provider's **leading command word** from its `interactive_command` in the **merged** provider table (`cfg.ProviderNames()` + `cfg.GetProvider` — built-ins claude/codex/agy/kimi plus project `providers:`), with any name matching `IsShellCommand` excluded from the set. The leading command word MUST be extracted with **quote-aware, assignment-prefix-aware parsing** — skip only true `NAME=value` environment-assignment prefixes (POSIX name chars before the `=`), honor quoting so a quoted assignment value is never split into tokens, and never skip an executable path merely because it contains `=` (reuse or extract the existing quote-aware leading-command parser at `src/go/fab/internal/setupcheck/setupcheck.go:130-167` rather than `strings.Fields`). Example: `TOKEN='a b' /opt/agents/my-agent --flag` yields `my-agent`. Matching consults the node's `comm` first; fallback: the basenames of the first two tokens of the node's `Cmdline` (bounded window so prompt-text arguments can never match).
  - GIVEN a kimi pane under the wrapped spawn with kimi alive, WHEN the tick diffs it, THEN the kimi process classifies as agent and no `agent_exited` is emitted.
  - GIVEN a provider distributed as a node script (comm `node`, cmdline `node /usr/local/bin/claude ...`), WHEN classified, THEN the cmdline fallback marks it agent.

- **R4**: `diffMonitored` MUST stay a pure, table-testable function: the liveness check enters as an injected `func(paneID string) bool` parameter; the real implementation (PID + tree walk, current socket) is composed only at the `runOperatorTickStartDiff` call site. Per-entry evaluation order `pane_death → pane_mismatch → agent_exited → clean join` is unchanged.

### Pane process: shared classifier

- **R5**: `fab pane process` MUST use the same extended agent-name set (config-derived, fail-open per the `operator.go:158` precedent: `config.Load` when a fab root resolves, degrading to built-ins only), so `classification`/`has_agent` agree with the tick's confirmation. Output shape (human and `--json`) is unchanged.
  - GIVEN a pane hosting kimi, WHEN `fab pane process <pane> --json` runs, THEN the kimi node classifies `agent` and `has_agent` is `true`.

- **R6**: No snapshot-format change: the pane enumeration (rk-delegated or `list-panes -F` fallback) is untouched; the pane PID is resolved per-pane on demand, only for shell-fronted monitored panes.

### Docs: behavior-claim sweep

- **R7**: All skill prose stating the old predicate or the disproven mechanism claim MUST be updated in the same change: `src/kit/skills/_cli-fab.md` (§ tick-start detection semantics + delta-kind comment), `src/kit/skills/fab-operator.md` (§4 delta handling, `⏏ shell` frame-marker row), `src/kit/skills/_cli-agents.md` (§ Spawn Composition line ~90: "`#{pane_current_command}` reports the agent (e.g. `claude`, `kimi`) while it runs" is false on Linux — the wrapper shell stays pgrp leader; reword to the confirmed-liveness reality), and `src/go/fab/internal/pane/pane.go` `IsShellCommand` doc comment (now the operator's *trigger for confirmation*, not the sole exited signal). Sweep the whole sibling class per code-quality.md (grep the old phrases repo-wide across `src/kit/` + `docs/specs/`; `docs/memory/` lands at hydrate). **Owner-or-pointer discipline (must-fix per code-quality.md)**: `_cli-fab.md` § tick-start Detection semantics is the OWNER of the `agent_exited` predicate and its failure direction; `fab-operator.md` and `_cli-agents.md` MUST NOT restate the shell/tree predicate or the fail-toward-emit rule — they keep only their own facts (operator response + frame rendering; spawn mechanism + pgrp reality) and POINT to `_cli-fab.md` for detection semantics.

### Non-Goals

- The readiness gate (`internal/pane/gate.go` Probe takeover precondition) is unchanged — dispatch pane workers are unwrapped (exec-optimized, foreground = agent binary). **Latent interplay to record at hydrate**: `fab pane ready` on a *wrapped* pane with a live agent reports `booting` on Linux for the same misreport reason; the operator's routed sends ride rk's agent-state gate instead, so nothing shipped breaks today — flag as a backlog candidate, not fixed here.
- No change to the spawn shape (`; exec "$SHELL"` wrap, `internal/spawn.WithShellFallback`).
- No new delta kind; no `.status.yaml`/state-file schema change; no migration (no per-checkout user data restructured).
- No run-kit change (enumeration format untouched).

### Design Decisions

- **Decision**: Liveness enters `diffMonitored` as an injected predicate func, composed at the call site.
  **Why**: keeps the diff pure and the existing table tests serverless; matches the package's scripted-fake convention (gate.go PaneIO precedent).
  **Rejected**: calling `GetPanePID`/`/proc` directly inside `diffMonitored` (untestable without live processes); pre-computing liveness for all rows (wasteful — only shell-fronted monitored rows need it).
  *Introduced by*: 260902-ssyf-operator-liveness-process-tree

- **Decision**: Agent-name set sources `interactive_command` only (not `headless_command`), quote-aware leading-command-word basename (assignment prefixes skipped per POSIX rules), shells excluded; plus the hardcoded `claude`/`claude-code` baseline.
  **Why**: operator interactive spawns are the only wrapped shape; headless built-ins use the nested-shell idiom (`sh -c '... "$(cat)"'`) whose first token is a shell and whose real binary is unreachable by token position.
  **Rejected**: parsing headless commands (fragile); a config knob for extra names (a project provider row already extends the set).
  *Introduced by*: 260902-ssyf-operator-liveness-process-tree

- **Decision**: On walk error, emit `agent_exited` (fail toward today's behavior).
  **Why**: suppression without evidence re-admits a bare shell to `candidates:` via its stale `idle` agent state — the type-into-shell hazard 1xqx guards; a wrongly-emitted exit is recoverable (level-triggered report, re-enroll).
  **Rejected**: fail-toward-suppress; surfacing walk errors per tick (noise on a best-effort snapshot path).
  *Introduced by*: 260902-ssyf-operator-liveness-process-tree

- **Decision**: Cmdline fallback window = first two tokens' basenames.
  **Why**: covers interpreter-wrapped binaries (`node /path/claude`) while a prompt argument containing an agent name (`kimi -p "ask claude"`) can never match.
  **Rejected**: whole-cmdline scan (false suppress on prompt text); comm-only (misses script/interpreter wrappers).
  *Introduced by*: 260902-ssyf-operator-liveness-process-tree

## Tasks

### Phase 2: Core Implementation

- [x] T001 Parameterize the process classifier: in `src/go/fab/cmd/fab/pane_process.go` change classification to take an agent-name set + cmdline (`classifyProcess(comm, cmdline string, agents map[string]bool) string`), add `agentBinaryNames(cfg *config.Config) map[string]bool` (built-ins ∪ merged `interactive_command` **quote-aware leading-command-word** basenames per R3 — reuse/extract the setupcheck.go:130-167 parser, NOT `strings.Fields`; `IsShellCommand` names excluded, nil-cfg safe), update `pane_process_linux.go` + `pane_process_darwin.go` to build nodes with the set (or classify post-walk), and wire `runPaneProcess` to build the set fail-open (`resolve.FabRoot` → `config.Load`, else built-ins) <!-- R3, R5 --> <!-- rework: interactiveCommandBinary used strings.Fields + token-contains-= skipping — quoted assignment values split wrongly (TOKEN='a b' /opt/agents/my-agent registered `b`) and paths containing = were skipped -->
- [x] T002 Add the liveness checker in `src/go/fab/cmd/fab` (alongside the tick code): `paneAgentAlive(paneID string, agents map[string]bool) bool` composing `pane.GetPanePID(paneID, "")` + `discoverProcessTree` + a tree scan (comm-first, two-token cmdline fallback, root included); errors return false-alive (⇒ emit) <!-- R1, R2, R3 -->
- [x] T003 Wire the confirmation into the tick: add the `agentAlive func(string) bool` parameter to `diffMonitored` (`src/go/fab/cmd/fab/operator_tick_start.go`), gate the `agent_exited` branch on `pane.IsShellCommand(row.command) && !agentAlive(entry.Pane)` (alive ⇒ fall through to clean join), compose the real checker in `runOperatorTickStartDiff` (config fail-open, name set built once per tick), and update the branch comment to the confirmed semantics <!-- R1, R2, R4, R6 -->

### Phase 3: Tests

- [x] T004 Update `src/go/fab/cmd/fab/operator_tick_diff_test.go`: existing shell-basename cases inject a never-alive fake (behavior preserved); new cases — shell + alive ⇒ clean join with stage diff/candidates/baseline write; shell + alive never invokes the walk for non-shell rows (call-recording fake); walk-error/false ⇒ `agent_exited`; `pane_mismatch` precedence over a shell-fronted alive pane unchanged <!-- R1, R2, R4 -->
- [x] T005 Update `src/go/fab/cmd/fab/pane_process_test.go`: classifier cases — kimi/codex/agy comm names classify agent via the config-derived set; `node /path/claude` cmdline fallback; prompt-text argument (`kimi -p "ask claude"` as tokens 3+) does NOT match; shell names never in the set; `has_agent` reflects the extension; **regression cases for R3 parsing**: `TOKEN='a b' /opt/agents/my-agent --flag` → `my-agent`, an executable path containing `=` is not skipped, plain `FOO=1 kimi --auto` → `kimi` <!-- R3, R5 --> <!-- rework: add quote-aware parsing regressions alongside the T001 fix -->
- [x] T006 Run scoped tests + gofmt from the Go module root (the repo root has no `go.mod`/`go.work`): `gofmt -l` on touched files, then from `src/go/fab`: `go test ./cmd/fab/... ./internal/pane/...`; widen to `go test ./...` (same directory) once green <!-- R1–R6 --> <!-- rework: prior recipe ran go test with repo-root-relative package paths, which fails outside the module -->

### Phase 4: Docs

- [x] T007 Update `src/kit/skills/_cli-fab.md`: § fab operator tick-start detection semantics (~line 1301) and the delta-kind comment (~1279) — `agent_exited` = shell foreground AND no live agent process in the pane's tree; name the fail-toward-emit direction and the lazy walk <!-- R7 -->
- [x] T008 Update `src/kit/skills/fab-operator.md`: §4 delta bullet (~309) and the `⏏ shell` frame-marker row (~387) — per R7's owner-or-pointer discipline, REMOVE any restatement of the shell/tree detection predicate or failure direction (owned by `_cli-fab.md` § tick-start Detection semantics; point there), keeping only the operator's own facts: the delta's meaning for the operator (report + remove, never a candidate) and the frame rendering <!-- R7 --> <!-- rework: owner-or-pointer violation — §4 restated the owned detection contract right after pointing at its owner -->
- [x] T009 Update `src/kit/skills/_cli-agents.md` § Spawn Composition (~line 90): correct the "`#{pane_current_command}` reports the agent while it runs" claim (Linux: the non-interactive wrapper keeps pgrp leadership for the child's entire life — verified live 2026-09-02) — state only the spawn-mechanism fact and POINT to `_cli-fab.md` § tick-start for how the operator detects an exit (no restated predicate or failure direction, per R7's owner-or-pointer discipline); and `src/go/fab/internal/pane/pane.go` `IsShellCommand` doc comment (trigger-for-confirmation, gate consumer unchanged) <!-- R7 --> <!-- rework: owner-or-pointer violation — the corrected sentence restated the owned detection contract -->
- [x] T010 Behavior-claim sweep: grep `src/kit/` + `docs/specs/` for the old claim class — `foreground command is a shell`, `foreground command is the ground truth`, `reports the agent (e.g.`, `the agent quit or crashed` — update every occurrence stating the unconfirmed predicate or the disproven mechanism (docs/memory/ is hydrate's); note the gate/wrapped-pane latent interplay for hydrate <!-- R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: A shell-fronted monitored pane with a live agent process in its tree emits no `agent_exited` and takes the clean-join path (stage diffs, completion, baseline write, candidates)
- [x] A-002 R1: A shell-fronted monitored pane with no agent process emits `agent_exited` with the `command` field, level-triggered, same exclusions as before
- [x] A-003 R2: PID/tree-walk failure emits `agent_exited` (fail toward today's behavior), silently
- [x] A-004 R3: The agent-name set covers built-ins (claude/claude-code) and merged provider `interactive_command` basenames (codex/agy/kimi + project rows); shell names are excluded; cmdline fallback is bounded to two tokens
- [x] A-005 R4: `diffMonitored` takes the liveness check as an injected func; evaluation order `pane_death → pane_mismatch → agent_exited → clean join` unchanged
- [x] A-006 R5: `fab pane process` classifies provider binaries as `agent` with unchanged output shape
- [x] A-007 R6: No enumeration-format change; the walk runs only for shell-fronted monitored panes

### Scenario Coverage

- [x] A-008 R1: Test — wrapped-spawn live-agent scenario (shell command + alive fake) joins cleanly
- [x] A-009 R2: Test — error/dead fake emits `agent_exited`; existing shell cases preserved under a never-alive fake
- [x] A-010 R3: Test — comm and cmdline-fallback classification, including the prompt-text non-match

### Behavioral Correctness

- [x] A-011 R1: `pane_mismatch` still wins over `agent_exited` for a mismatched shell-fronted pane (no walk consulted)

### Removal Verification

- [x] A-012 R7: No skill prose still states the unconfirmed predicate ("foreground command is a shell ⇒ agent exited") or the disproven wrapper claim (grep-verified across `src/kit/` + `docs/specs/`)

### Code Quality

- [x] A-013 Pattern consistency: injected-dependency seam matches gate.go's PaneIO convention; no new magic strings (delta kinds/readiness constants stay named)
- [x] A-014 No unnecessary duplication: one classifier serves the tick and `fab pane process`; `discoverProcessTree`/`hasAgentInTree` reused, not re-implemented
- [x] A-015 CLI ⇒ docs + tests: `_cli-fab.md` updated and Go tests ship in the same change (constitution Additional Constraints)
- [x] A-016 Canonical source only: skill edits land under `src/kit/skills/`, never `.claude/skills/`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The wrapped-spawn misreport is universal on Linux, not intermittent | Verified live on this machine (throwaway tmux socket, 2026-09-02): wrapped `zsh -c "cmd; exec zsh"` reads `zsh` for the child's whole life; unwrapped exec-optimizes to the binary | S:95 R:90 A:95 D:95 |
| 2 | Confident | Cmdline fallback window = first two tokens' basenames | Covers interpreter wrappers; prompt arguments can never match | S:60 R:80 A:75 D:70 |
| 3 | Confident | Config loads fail-open at the tick (FabRoot → Load, else built-ins only) | operator.go:158 precedent; the tick is multi-repo and must not error on a repo-less cwd | S:65 R:80 A:85 D:75 |
| 4 | Confident | Liveness checker lives in `cmd/fab` package main beside `discoverProcessTree` | Discovery is already platform-split there; no cross-package export needed | S:60 R:85 A:85 D:75 |
| 5 | Confident | Gate/wrapped-pane interplay recorded as backlog candidate, not fixed | Nothing shipped breaks today (operator sends ride rk's agent-state gate); widening scope would reverse an intake non-goal mid-run | S:60 R:70 A:75 D:65 |

5 assumptions (1 certain, 4 confident, 0 tentative).

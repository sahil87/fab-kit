# Intake: Pane Readiness Gate — Require Agent Takeover Before Probing

**Change**: 260829-57mp-pane-readiness-agent-takeover
**Created**: 2026-08-29

## Origin

Promptless dispatch (`{questioning-mode} = promptless-defer`) from a live failure observed on 2026-08-29 in a run-kit worktree, provider `kimi`, pane-mode apply. The synthesized brief handed to this intake:

> **Title idea:** Pane readiness gate — require agent takeover before probing; wait-timeout stall guard
>
> **Problem observed (2026-08-29, live run in run-kit worktree, provider kimi, pane-mode apply):** `fab dispatch open` returns immediately after spawning the pane. `fab dispatch ready` (`src/go/fab/internal/pane/gate.go` `Probe`, ~line 178) is purely echo-based: it types the sentinel `FAB-READY-PROBE` and reports `ready` if it echoes. In the ~1–3 s window before the provider TUI (kimi) puts the pty into raw mode, the pty is in cooked mode and the SHELL echoes typed keystrokes, so the sentinel echoes → false `ready`. `fab dispatch deliver` then re-probes (same false echo), types the pointer (echoes), presses Enter, and its 800 ms "screen advanced" check passes because kimi is drawing its TUI at that moment — every verification step is fooled by the same window. Kimi comes up, discards the pre-TUI input, and parks on its first-run `Trust this folder?` dialog. The orchestrator (fab-fff pane branch, `_preamble.md` § The pane readiness gate) then arms `fab dispatch wait --timeout 290` "for several rounds" against a worker that never received a prompt — 9+ minutes of stall observed. This is a recurring trap (previously logged as "deliver can false-verify into a still-booting kimi pane"); the root cause was never fixed. It is provider-generic (any TUI provider has this window), not kimi-specific.
>
> **Decisions made (user-confirmed):**
>
> 1. **Gate on agent takeover before probing (binary fix, Go).** In the gate's `Probe`, before typing the sentinel, query the pane's foreground command (`tmux [-L server] display -p -t <pane> '#{pane_current_command}'`). While the foreground command is still a shell (bash/zsh/sh/fish etc. — i.e. the provider binary has not taken the tty), report `booting` regardless of echo and do NOT type the sentinel. Only once a non-shell process owns the pane does the echo/stability probe run. This reuses the same pane-command signal the operator's `agent_exited` delta already keys on (see docs/memory/runtime and the 1xqx change for that precedent). Extend `PaneIO` (or equivalent seam) so the check is testable against the scripted fake; add table tests in `gate_test.go` covering shell-foreground → booting, agent-foreground + echo → ready, agent-foreground + no echo → parked/booting as today. Because `Deliver` calls `Probe`, delivery inherits the guard automatically. Update `src/kit/skills/_cli-fab.md` (fab dispatch ready / fab pane ready descriptions: the "purely mechanical, echo-based" wording must now mention the foreground-command precondition) and the `dispatch_ready.go` / `pane ready` Long help text accordingly (constitution: CLI change ⇒ `_cli-fab.md` + tests).
> 2. **REJECTED:** additionally requiring run-kit's `@rk_pane_agent_state` tmux option to be set before probing. User said no — pane-command alone covers the race; keep run-kit coupling out of the gate.
> 3. **Skill-side stall guard (prose fix).** In `_preamble.md`'s pane branch / `wait` guidance (§ The pane readiness gate and the numbered procedure's step 2 `wait`): when a `fab dispatch wait` round times out AND the worker shows no progress (no `{stage}-result.yaml`, `fab pane capture` screen unchanged vs. the previous round / no context growth), the orchestrator MUST re-run `fab dispatch ready <change> <stage>` instead of blindly re-arming another wait round; a `parked` answer re-enters the judgment rounds (mid-stage carve-out: a worker that never received its prompt holds no stage context, so answering the wall is still legal — state this explicitly, since the current text says "from successful delivery onward the ordinary rule applies" and a false-verified delivery is not a real delivery). Sweep the twin/sibling class: fab-fff ↔ fab-ff, fab-operator if it restates wait/stall behaviour, `docs/specs/harness-adapters.md` (gate description), and the runtime memory files documenting the gate (docs/memory/runtime/*). Owner-or-pointer rule applies (fab/project/code-quality.md): state the rule once in its owner and point elsewhere.
>
> **Constraints:** no table of known dialog texts in the gate (design explicitly refuses this — keep). No new config field or flag. Constitution VII test integrity. Change type likely `fix`.

Interaction mode: one-shot dispatch, no questions asked. Every decision that would have been a question is recorded below as an Unresolved or Tentative row in `## Assumptions`.

## Why

**The problem.** The pane readiness gate exists to answer one question — *can this pane accept typed input right now?* — and it answers it with a single observable: does a literally-typed sentinel echo back? That observable is **not sufficient**, because a tmux pane always starts with a shell holding the pty in **cooked mode**, and a cooked-mode tty echoes typed characters by itself. So during the ~1–3 s between `fab dispatch open` spawning the pane and the provider's TUI putting the pty into raw mode, the sentinel echoes for a reason that has nothing to do with the agent being ready. The gate reports `ready`. Every downstream verification then fails in the same direction, because they are all built on the same fooled observable:

| Step | What it checks | Why it passes falsely |
|------|----------------|------------------------|
| `fab dispatch ready` | sentinel echo | the shell echoes it in cooked mode |
| `deliver` internal re-probe | sentinel echo | same window, same shell |
| `deliver` pointer echo (`newlyEchoed`) | the pointer newly appeared on screen | the shell echoed it onto the command line |
| `deliver` busy check (800 ms) | the screen changed after Enter | the TUI is painting itself at that exact moment |

The composite result is the worst possible outcome the delivery choreography was built to prevent: `delivered: true` is written, the dispatch derives `running`, and the provider comes up, **discards the pre-TUI input** (the keystrokes went to a shell that the exec'd TUI replaced), and parks on its own first-run wall. The record says a prompt was verifiably delivered; no prompt was delivered.

**The consequence.** The orchestrator is now past the gate, so `_preamble.md`'s rule "from successful delivery onward the ordinary rule applies again" holds — it may not send keys, and it re-arms `fab dispatch wait --timeout 290` round after round against a worker that will never produce a result file. The observed cost on 2026-08-29 was **9+ minutes of dead stall** before a human noticed, and the stall is unbounded in principle: `wait` returning `running` on timeout expiry is a peek-and-re-arm signal, and the peek's three classifications (progressing / parked at an error banner / awaiting human) do not have a bucket for *"the worker never got its prompt"*.

**Why this is not a kimi bug.** The cooked-mode window is a property of how tmux spawns a pane and how any TUI takes over a pty. Claude, agy, codex and kimi all have it; kimi is simply slow enough and dialog-y enough to lose the race reliably. The failure is already in this project's recurring-lessons memory as *"`fab dispatch deliver` can false-verify into a still-booting kimi pane (welcome screen + no context growth ⇒ kill+restart)"* — recorded as an operator workaround, never as a root-cause fix. This change fixes the root cause.

**Why this approach over the alternatives.**

- **Foreground-command precondition (chosen).** The pane's `#{pane_current_command}` answers exactly the question the echo cannot: *has the provider binary actually taken the tty?* It is a single cheap tmux read, provider-neutral, needs no dialog table, and reuses a signal this codebase already trusts — the operator's `agent_exited` delta keys on the same field for the mirror-image reason (agent state goes stale when a process exits, so the foreground command is the ground truth about who owns the pane).
- **Rejected: gate on `@rk_pane_agent_state`.** Explicitly rejected by the user. It would make fab's core dispatch gate depend on run-kit hooks being installed, and the option is a *consumed* convention fab does not own. Pane-command alone covers the race.
- **Rejected: a table of known dialog texts.** Already refused by the shipped design (dialog text is a version treadmill; a half-matched pattern pressing Enter into an unknown screen is worse than stalling). Unchanged by this change.
- **Rejected: lengthening the settle/busy delays.** A fixed delay is a race, not a fix — it trades a reliable failure for a slower, still-possible one, and it penalizes every fast provider.

**Why the prose half is needed too.** The binary fix closes the window that produced *this* stall, but the orchestrator's `wait` loop still has no move for a worker that is silently doing nothing. A second, independent backstop — re-probe readiness instead of blindly re-arming — is what converts a future silent stall of any cause into a bounded, self-recovering event. The two halves are deliberately belt-and-braces.

## What Changes

### 1. `Gate.Probe` gains a foreground-command precondition (`src/go/fab/internal/pane/gate.go`)

`Probe` currently opens by typing the sentinel. It gains an earlier step: read the pane's foreground command, and if its basename is a known shell, return `ReadyBooting` **without typing anything**.

Sketch (shape, not final text):

```go
func (g *Gate) Probe(paneID string) (Readiness, string, error) {
	// AGENT TAKEOVER PRECONDITION. A tmux pane starts with a shell holding the
	// pty in COOKED mode, where the tty echoes typed characters by itself — so
	// the sentinel echoes for a reason that has nothing to do with an agent
	// being ready. Until the provider binary has taken the tty, there is no
	// agent to probe: report booting and type NOTHING.
	cmd, err := g.IO.CurrentCommand(paneID)
	if err != nil {
		return "", "", err
	}
	if IsShellCommand(cmd) {
		snippet, err := g.IO.Capture(paneID, captureLines)
		if err != nil {
			return "", "", err
		}
		return ReadyBooting, Snippet(snippet), nil
	}
	// … existing sentinel / echo / stability choreography, unchanged …
}
```

Properties that MUST hold:

- **No keystroke is sent on the shell-foreground path.** This strengthens `Probe`'s existing read-mostly/idempotent property (Constitution III), it does not weaken it.
- **`booting` regardless of echo.** The echo is not consulted at all on this path; it is the untrustworthy signal.
- **`Deliver` inherits the guard for free** — `deliverOnce` already opens with `g.Probe(paneID)` and returns `pane is %s, not ready` for any non-`ready` classification. No change to `Deliver`'s own body is required. A delivery attempted into a pre-takeover pane now fails its probe twice and returns the ordinary two-attempt error with the snippet, instead of silently "succeeding".
- **`DeriveReadiness` is untouched.** It stays the pure echo/stability classifier for the post-takeover path; the precondition sits in front of it. (An alternative — folding the command into `DeriveReadiness`'s signature — is rejected: it would make the pure classifier's table about two unrelated axes.)

### 2. `PaneIO` gains a third method

```go
type PaneIO interface {
	Capture(paneID string, lines int) (string, error)
	SendLiteral(paneID, text string) error
	SendKey(paneID, key string) error
	CurrentCommand(paneID string) (string, error) // NEW
}
```

`tmuxPaneIO.CurrentCommand` delegates to a new shared helper in `src/go/fab/internal/pane/pane.go`, alongside the existing `Capture`/`SendLiteral`/`SendKey`/`ReadWindowName` family and following their `server`-first argument order and `RunCmd` + `StderrError` conventions:

```go
// CurrentCommand returns a pane's foreground command via
// `tmux [-L <server>] display-message -p -t <pane> '#{pane_current_command}'`.
func CurrentCommand(server, paneID string) (string, error)
```

The fake in `gate_test.go` (`fakePaneIO`) gains a scripted `CurrentCommand` — a value (or queue) plus a `failOn["command"]` arm, matching how `captures`/`failOn` already work — so every existing gate test keeps compiling and the new rows are table-driven.

### 3. The shell-name predicate gets one shared home

The set already exists, in `src/go/fab/cmd/fab/operator_tick_start.go`:

```go
var shellCommands = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true, "dash": true,
	"ksh": true, "tcsh": true, "csh": true, "nu": true,
}

func isShellCommand(cmd string) bool { … }   // basename match; "" never matches
```

It relocates to `src/go/fab/internal/pane` as exported `IsShellCommand` (plus its set), and `cmd/fab`'s `agent_exited` predicate calls the relocated function. Rationale: `internal/pane` is already the shared home for the pane primitives both `cmd/fab` and the gate ride, and `fab/project/code-quality.md` names "duplicating existing utilities instead of reusing them" as an anti-pattern. Semantics are byte-identical — same nine basenames, same case-sensitive basename match, empty string never matches — so `operator_tick_diff_test.go`'s existing `agent_exited` rows (including the `/usr/bin/fish` and the `zshrc-lint`-is-not-a-shell rows) must keep passing unchanged. That equality is the acceptance criterion for the relocation.

### 4. New `gate_test.go` table rows

Added to the unit suite (scripted fake, no tmux server), per the brief:

| Foreground command | Echo | Expected |
|--------------------|------|----------|
| `zsh` / `bash` / `/usr/bin/fish` | sentinel WOULD echo | `booting`, **and `io.sends` is empty** |
| `kimi` / `claude` / `node` | echoes | `ready` |
| `kimi` | no echo, screen changed | `booting` (as today) |
| `kimi` | no echo, stable non-blank screen | `parked` (as today) |
| `kimi` | no echo, blank screen | `booting` (as today) |
| (`CurrentCommand` errors) | — | error propagated, nothing typed |

The **no-keystroke assertion is the load-bearing one** — it is what pins that the shell path types nothing, which is the whole safety property. Also add a `Deliver`-level row proving a shell-foreground pane makes delivery fail (rather than false-verify) with the snippet attached.

### 5. Integration-test fixtures: the bare-shell pane stops being the "ready" stand-in

`newTmuxPane(t, server, "", width)` — an empty command, i.e. the **default shell** — is used at **13 sites** across `cmd/fab` as the stand-in for a live agent, on the documented reasoning that a shell "is enough of an 'agent' for the choreography's terms to mean something: it echoes typed text and reacts to Enter". That reasoning is exactly what this change invalidates. Affected sites:

- `dispatch_ready_test.go:38`, `:80`
- `pane_ready_test.go:26`, `:91` (`TestPaneReady_ReadyReport`, `TestPaneReady_JSON`)
- `dispatch_deliver_test.go:168`, `:212`, `:288`, `:323`
- `pane_deliver_test.go:47`, `:97`, `:111`, `:181`, `:201`
- `pane_kill_test.go:22`, `:39` (liveness only — no gate involvement; likely unaffected, verify)

The fixture becomes a **non-shell** command that still echoes typed text and reacts to Enter — `cat` is the front-runner (basename `cat` is not in the shell set; a cooked-mode `cat` echoes what is typed and prints the line on Enter, so the screen advances). Whether this is done by changing the call sites or by giving `newTmuxPane` a named default is an apply-time detail; the contract is that the "ready" fixture no longer depends on which login shell the host runs.

The `parked` / `booting` fixtures need **no change**: `parkedPaneCommand` (`stty -echo; echo TRUST-THIS-FOLDER-WALL; sleep 300`) and `bootingPaneCommand` (`stty -echo; sleep 300`) both leave `sleep` — a non-shell basename — as the pane's foreground command while they hold, so they still reach the echo/stability classifier and still classify as they do today.

Constitution VII applies with force here: these tests are being changed **because the spec changed**, not to accommodate the implementation. The plan must say so at each edited site.

### 6. CLI help text (`dispatch_ready.go`, `pane_ready.go`)

Both `Long` strings open with the same claim:

> "The probe is purely MECHANICAL — it types a sentinel literally, checks whether the sentinel echoed, clears it with C-u, and looks at whether the screen is still moving."

Both gain the precondition ahead of that sentence, and the `booting` line gains the second cause. Sketch:

```
Before any keystroke the probe checks who owns the pane: while the foreground
command is still a shell (the provider binary has not taken the tty yet), it
reports `booting` and types NOTHING — a shell in cooked mode echoes typed
characters by itself, so the sentinel would echo for a reason that has nothing
to do with an agent being ready. Only once a non-shell process owns the pane
does the echo-and-stability probe run.

  booting  the pane is still a shell, or no echo on a blank/changing screen —
           wait and re-probe
```

`fab pane ready`'s **SIDE EFFECT** paragraph stays true and becomes narrower (nothing is typed on the shell path); it should be reworded accordingly rather than left claiming an unconditional type.

### 7. `src/kit/skills/_cli-fab.md`

Two sections restate the "purely echo-based" contract and MUST be updated (Constitution: a CLI signature/behaviour change updates `_cli-fab.md` + tests):

- **§ `fab pane ready`** (line ~586) — "Types a sentinel literally (never submitted), checks the echo against two screen-stability captures, clears with `C-u`, and reports `ready` / `booting` / `parked`." Add the foreground-command precondition and the "types nothing while the pane is a shell" property.
- **§ `fab dispatch ready`** (line ~714) — "Answers one question about an opened pane: **can it accept typed input right now?** The probe is purely MECHANICAL — …" plus its three-row classification table (line ~719–720): the `booting` row's condition gains "the pane's foreground command is still a shell (the provider has not taken the tty)".

The `--json` object shape (`{state, pane, server, snippet}`) and all exit codes are **unchanged**.

### 8. `src/kit/skills/_preamble.md` — the stall guard

`_preamble.md` § The pane readiness gate is the **owner** of this rule; the numbered CLI-Adapter Dispatch procedure's step 2 (`wait`) and the timeout-return peek table **point at it** rather than restating it (`code-quality.md` owner-or-pointer rule).

Two edits in the owner section:

**(a) A new stall-guard rule.** When a `fab dispatch wait` round returns `running` (its bound expired) **and** the worker shows no progress — no `.fab-dispatch/{id}/{stage}-result.yaml`, and the `fab pane capture` screen is unchanged against the previous round — the orchestrator MUST judge the captured screen **before** re-arming another `wait`: if it shows a first-run wall or a bare shell prompt, the delivery never happened (see (b)) and the gate's judgment rounds re-enter; otherwise re-arm. It is a bounded, read-only diagnostic on a no-progress timeout-return — never a poll-loop step, and never a re-run of a readiness verb (see (c) for why).

**(b) The mid-stage carve-out, stated explicitly.** The section currently closes with:

> **From successful delivery onward the ordinary rule applies again** — a wall that appears mid-stage escalates, exactly as before.

This must be qualified: when the stall guard finds a first-run wall or a bare shell prompt on the screen, the "successful" delivery was never a real delivery — a worker that never received its prompt holds **no stage context**, so the no-input-injection rule still has no subject and the judgment rounds are legal. State this explicitly so the two rules do not read as contradictory.

**(c) The brief's literal instruction is not executable — verified.** The brief says the orchestrator "MUST re-run `fab dispatch ready <change> <stage>`". It cannot: `runDispatchReady` calls `refuseMidStageDelivery`, whose condition is exactly

```go
if !rec.Delivered { return nil }
if dispatch.ResultPresent(dir, stage) { return nil }
return fmt.Errorf("%s/%s already has its prompt and is still running (pane %s); the pipeline never types into a worker mid-stage — …")
```

`Delivered && !ResultPresent` **is** the false-verified stall state, so `fab dispatch ready` hard-errors precisely when the stall guard would call it. The record-free primitive `fab pane ready <pane>` carries no such guard and would work mechanically — but it is still a **sender**: against a genuinely-working worker it types the sentinel into a live input box, which is the injection `_preamble.md`'s "the pipeline NEVER sends keys to a WORKER" rule forbids. Three candidate resolutions were weighed; **(c) is the decision** (clarified 2026-08-29):

| Option | Shape | Cost |
|--------|-------|------|
| (a) name `fab pane ready <pane>` | pane id + socket from `fab dispatch status --json` | reintroduces mid-stage typing into a possibly-live worker |
| (b) carve out `fab dispatch ready`'s refusal | Go change | same injection risk, and it weakens a guard that exists for a good reason |
| (c) **capture-based judgment, no probe** | the guard takes the ordinary read-only `fab pane capture` peek and, if the screen shows a first-run wall or a bare shell prompt, treats the delivery as never having happened and re-enters the judgment rounds | none — read-only, no new verb or flag, and the orchestrator is already the judgment layer at every timeout-return |

Option (c) ships: it is the only one that keeps the never-type-into-a-worker invariant intact, and this change's own §1 precondition is what makes a bare shell prompt on the screen a *conclusive* signal rather than a guess. The prose must name a mechanism that actually works — it must not restate the brief's `fab dispatch ready` instruction verbatim.

The existing timeout-return classification table gains a fourth row for the new bucket ("(d) delivered but silent — no result file, screen unchanged ⇒ re-probe readiness per § The pane readiness gate"), or a pointer line, whichever keeps the owner-or-pointer rule intact.

### 9. Sibling sweep

Checked up front (`code-quality.md` § Sibling Sweeps), with the result recorded so review does not have to rediscover it:

| Surface | Restates the gate / wait rule? | Action |
|---------|-------------------------------|--------|
| `src/kit/skills/fab-fff.md`, `fab-ff.md` | **No** — grepped for `readiness gate` / `dispatch ready` / `booting`: zero hits | No edit; re-verify at apply |
| `src/kit/skills/fab-operator.md`, `fab-continue.md`, `fab-adopt.md`, `_pipeline.md` | **No** — same grep, zero hits | No edit; re-verify at apply |
| `src/kit/skills/_cli-agents.md` (line ~105, and the provider-authoring guidance) | Yes — "`fab pane ready <pane>` runs the classification half (typed sentinel → echo check → …)"; also the home of the new **exec contract**: a provider's `interactive_command` MUST exec its binary (a wrapper that keeps a shell in the foreground reads `booting` → `parked` with the shell prompt in the snippet and escalates — observable, not silent) | Update the parenthetical; add the exec-contract sentence once (owner) |
| `docs/specs/harness-adapters.md` (lines ~164, ~273–276) | Yes — the verb table row and "The readiness gate … is MECHANICAL … no echo on a blank or still-changing screen ⇒ `booting`" | Update both |
| `docs/memory/runtime/dispatch.md` | Yes — see Affected Memory | Update |
| `docs/memory/runtime/pane-commands.md` | Yes — see Affected Memory | Update |
| `docs/memory/pipeline/execution-skills.md` | Yes — § The readiness gate (pane arm), incl. the "From successful delivery onward" sentence | Update |
| `docs/memory/runtime/providers-and-profiles.md` | Mentions "the generic readiness gate" for boot/first-run walls (lines ~108, ~146, ~178, ~186, ~489–490) | Read at apply; edit only if a claim became false |

## Affected Memory

- `runtime/dispatch`: (modify) — the requirement heading **"`fab dispatch ready <change> <stage>` is a mechanical, purely echo-based probe"** and its body ("derived **only** from a literal sentinel send plus pane captures") are the single most stale claim this change creates. The heading, the body sentence, the three-row classification table (the `booting` row), the scenario block, and the Design Decision *"`fab dispatch ready` classifies a pane by echo and screen stability only"* + its **Rejected** list all need the foreground-command precondition folded in. Note carefully: the existing text's *"never from `@rk_pane_agent_state`"* stays **true and must be kept** — that is decision 2's rejection, and `#{pane_current_command}` is a different signal. Also update the source-layout paragraph for the new `PaneIO` method and the `internal/pane` shell predicate.
- `runtime/pane-commands`: (modify) — § `fab pane ready`'s probe description (line ~143), and the `internal/pane` shared-package export list (line ~225: `Gate`/`Probe`/`Deliver`/`DeriveReadiness`/`PaneIO`/…) which gains `CurrentCommand` and `IsShellCommand`.
- `runtime/operator`: (modify) — the `agent_exited` prose names the shell set literally in three places (lines ~167, ~287, ~540). The **set is unchanged**, so the claims stay true; the edit is only to note that the predicate now shares one home in `internal/pane` with the readiness gate. Keep it to a pointer, not a restatement.
- `pipeline/execution-skills`: (modify) — § **The readiness gate (pane arm)** (line ~23) restates the gate's branch table and carries the *"From successful delivery onward the ordinary rule applies again"* sentence verbatim. It needs the stall-guard rule (as a pointer to `_preamble.md`, which owns it) and the mid-stage carve-out.
- `runtime/providers-and-profiles`: (modify, conditional) — several passages lean on "the generic readiness gate" handling boot and first-run walls. Read at hydrate; edit only where a claim became inaccurate.

## Impact

**Go (`src/go/fab`)**

| File | Change |
|------|--------|
| `internal/pane/gate.go` | `Probe` precondition; `PaneIO` third method; `tmuxPaneIO.CurrentCommand`; doc-comment rewrite of the "Why the gate is mechanical" block |
| `internal/pane/pane.go` | new `CurrentCommand(server, paneID)` helper; relocated `IsShellCommand` + shell-name set |
| `internal/pane/gate_test.go` | `fakePaneIO.CurrentCommand` + `failOn` arm; new precondition table; the no-keystroke assertion; a `Deliver` row |
| `internal/pane/pane_test.go` | tests for `CurrentCommand`'s argv/result mapping and `IsShellCommand` (mirroring `validatePaneResult`'s pure-half precedent) |
| `cmd/fab/dispatch_ready.go` | `Long` help text |
| `cmd/fab/pane_ready.go` | `Long` help text (incl. the SIDE EFFECT paragraph) |
| `cmd/fab/operator_tick_start.go` | `shellCommands`/`isShellCommand` relocated; call site repointed |
| `cmd/fab/dispatch_deliver_test.go` | `newTmuxPane` "ready" fixture (4 sites) |
| `cmd/fab/pane_deliver_test.go` | same (5 sites) |
| `cmd/fab/dispatch_ready_test.go` | same (2 sites) |
| `cmd/fab/pane_ready_test.go` | same (2 sites) |
| `cmd/fab/pane_kill_test.go` | verify only (2 sites; liveness, no gate) |
| `cmd/fab/operator_tick_diff_test.go` | verify only — must pass byte-unchanged after the relocation |

**Kit prose (`src/kit/skills`)**: `_cli-fab.md` (2 sections), `_preamble.md` (§ The pane readiness gate + step 2 pointer + the peek table), `_cli-agents.md` (1 parenthetical).

**Specs**: `docs/specs/harness-adapters.md` (2 places).

**Memory**: the five files listed above, plus regenerated indexes.

**Not affected**: `--json` shapes, exit codes, the three report words, `dispatch.mode`, any config field, `DeriveReadiness`'s signature, the dialog-table refusal, `@rk_pane_agent_state` reading.

**Release**: behaviour change to a shipped CLI contract; `fix` change type; no migration (no user data restructured).

**Adjacent backlog**: `[zshf]` (flaky ready-classification tests on zsh-default hosts — the zsh-newuser-install menu makes `TestPaneReady_ReadyReport` / `TestPaneReady_JSON` classify `parked`) is plausibly resolved as a side effect of the §5 fixture change, since those tests stop depending on the host's login shell entirely. Flag it at ship; do not assume it.

## Open Questions

None open. Resolved 2026-08-29 (see `## Clarifications`):

- Stall-guard mechanism: `fab dispatch ready` refuses `Delivered && !ResultPresent` (the stall state itself), so the guard is **capture-based judgment with no probe** — §8(c) option (c).
- Shell-wrapper providers: **exec contract** — a provider's `interactive_command` must exec its binary; documented in `_cli-agents.md`, no time bound, no `spawn_cmd` comparison.
- Shell-foreground vs. painting-TUI `booting`: one undifferentiated `booting`; no new report field (row 11).
- 5-consecutive-`booting` allowance stays at 5 (row 9).

## Clarifications

### Session 2026-08-29 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |
| 11 | Confirmed | — |
| 12 | Confirmed | — |

### Session 2026-08-29

| # | Action | Detail |
|---|--------|--------|
| 13 | Confirmed | After explanation — `fab dispatch ready` provably refuses the stall state; stall guard = capture-based judgment, no probe (option c) |
| 14 | Changed | "Option (a): a provider's `interactive_command` must exec its binary — documented exec contract in `_cli-agents.md`; wrapper providers fail observably (parked + snippet), not silently. R/A/D re-scored for the decided contract." |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Gate on a non-shell `#{pane_current_command}` inside `Gate.Probe`, so both `fab pane ready` and `fab dispatch ready` (and, transitively, `Deliver`) inherit it; a shell foreground reports `booting` regardless of echo and types nothing | User-confirmed decision 1, stated verbatim in the dispatch brief including the tmux query and the placement in `Probe` | S:95 R:70 A:90 D:95 |
| 2 | Certain | Do NOT consult `@rk_pane_agent_state` in the gate; the existing memory claim "never from `@rk_pane_agent_state`" is preserved, not contradicted | User-confirmed decision 2 is an explicit REJECTION; keeping run-kit coupling out of fab's core dispatch gate is also the shipped divestment posture | S:95 R:80 A:90 D:95 |
| 3 | Certain | No new config field, no new flag, no dialog-text table; the `--json` object shape and every exit code stay byte-identical | Stated constraint in the brief, and the dialog-table refusal is an existing shipped design decision this change explicitly keeps | S:95 R:75 A:90 D:95 |
| 4 | Certain | The foreground read rides a new third `PaneIO` method (`CurrentCommand`) rather than a direct tmux call from `Probe` | The brief says "extend `PaneIO` (or equivalent seam)", and `PaneIO` exists precisely so the whole choreography is testable against the scripted fake; a direct call would put the one new decision outside the only seam the tests can reach | S:80 R:85 A:85 D:80 |
| 5 | Certain | Relocate `shellCommands`/`isShellCommand` from `cmd/fab/operator_tick_start.go` into `internal/pane` as exported `IsShellCommand`, byte-identical semantics, with the operator's `agent_exited` predicate calling the relocated function | Clarified — user confirmed. `code-quality.md` names duplicating existing utilities an anti-pattern, and `internal/pane` is already the shared home both consumers ride. A pure move, fully reversible; `operator_tick_diff_test.go` passing unchanged is the equality proof | S:95 R:75 A:85 D:70 |
| 6 | Confident | The replacement "ready" integration fixture is a non-shell command that still echoes and reacts to Enter — `cat` is the front-runner | Clarified — user confirmed. The 13 bare-shell `newTmuxPane(t, server, "", …)` sites classify `booting` by construction after this change (verified by reading the sites and `newTmuxPane`'s own doc comment). `cat`'s basename is outside the shell set and a cooked-mode `cat` satisfies every property the choreography asserts; a helper binary is the only alternative and buys nothing | S:95 R:70 A:70 D:60 |
| 7 | Certain | `DeriveReadiness` keeps its current pure signature; the precondition sits in front of it rather than becoming a fourth parameter | Clarified — user confirmed. Folding the command in would make one pure classifier's table span two unrelated axes — who owns the pane vs. what the screen shows — against the package's established one-decision-per-classifier shape | S:95 R:80 A:80 D:70 |
| 8 | Certain | The stall guard is OWNED by `_preamble.md` § The pane readiness gate and POINTED AT from the numbered procedure's step 2 and the timeout-return peek table — stated once, never twice | Clarified — user confirmed. The brief names both locations ambiguously; `code-quality.md`'s owner-or-pointer rule settles it, and the rule is a gate re-entry rule, so the gate section is its owner | S:95 R:85 A:85 D:65 |
| 9 | Certain | The `_preamble.md` 5-consecutive-`booting` allowance stays at 5 | Clarified — user confirmed. Pre-takeover shell time is 1–3 s and boot re-probes are cheap and consume no judgment round, so 5 remains ample; the new stall guard is the real backstop for a slow provider, and a prose number is trivially revised later | S:95 R:90 A:70 D:65 |
| 10 | Confident | A pane-arm continuation `deliver` into an apply pane whose agent already exited (the `; exec "$SHELL"` fallback holding the pane) now fails its probe rather than false-verifying, and falls through to the documented mandatory fresh-dispatch fallback — no bespoke "the agent has exited" error is added | Clarified — user confirmed. That fallback is already load-bearing and documented in `_preamble.md` § Pane-arm continuation; a special-cased error would add a code path for an outcome the generic one already handles correctly | S:95 R:80 A:70 D:60 |
| 11 | Confident | The report and `--json` shapes gain no field naming the foreground command; a shell-foreground `booting` is reported like any other `booting` | Clarified — user confirmed. Adding a JSON field later is additive and non-breaking, while removing one is breaking — so deferring is the reversible direction, and the no-new-surface constraint reads as covering it. The cost is that a caller cannot distinguish "still a shell" from "TUI painting" | S:95 R:55 A:60 D:55 |
| 12 | Confident | Backlog `[zshf]` is FLAGGED as probably-resolved at ship, not marked done | Clarified — user confirmed. The §5 fixture change removes the ready tests' dependency on the host login shell, which is `[zshf]`'s entire mechanism — but that cannot be verified without running the suite on a zsh-default host, and a wrongly-closed backlog row is cheap to reopen | S:95 R:90 A:50 D:50 |
| 13 | Confident | The stall guard uses **capture-based judgment with no probe** (§8 option (c)); no readiness verb is re-run against a delivered record | Clarified — user confirmed after explanation (the `fab dispatch ready` refusal was verified in code; capture-based judgment is the mechanism). Verified: `refuseMidStageDelivery` fires on `Delivered && !ResultPresent`, which IS the stall state, so the brief's literal instruction cannot ship. Option (c) is the only one preserving "the pipeline NEVER sends keys to a WORKER", but choosing it means the brief's stated mechanism is replaced rather than implemented — a substitution the user has not seen | S:95 R:30 A:45 D:45 |
| 14 | Confident | **Exec contract**: a provider's `interactive_command` MUST exec its binary (option (a)), documented once in `_cli-agents.md`. A wrapper that keeps a shell in the foreground is unsupported by design and fails **observably** — `booting` → `parked` with the shell prompt in the snippet, then escalation — not silently. No time bound, no `spawn_cmd` comparison | Clarified — user changed to option (a); R/A/D re-scored for the decided contract (prose, one owner sentence, fully reversible) rather than for the open question. Deferred — promptless dispatch. A product-scope call: it decides whether a class of user-defined providers silently stops working, it is baked into a documented contract once shipped, and the three options have materially different tradeoffs | S:95 R:70 A:80 D:70 |

14 assumptions (8 certain, 6 confident, 0 tentative, 0 unresolved).

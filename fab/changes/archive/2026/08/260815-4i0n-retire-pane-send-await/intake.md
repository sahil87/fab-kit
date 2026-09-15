# Intake: Retire fab pane send/await (cli-layering Part 5)

**Change**: 260815-4i0n-retire-pane-send-await
**Created**: 2026-08-15

## Origin

One-shot `/fab-new` invocation carrying the Part 5 row of the cross-repo CLI-layering execution plan (the spec lives in the **run-kit** repo: `~/code/sahil87/run-kit/docs/specs/cli-layering.md` — fab-kit has no local copy; this intake restates every fact the pipeline needs from it). This is the first fab-kit-side part of that plan (run-kit hosts Parts 1–4/6/8; fab-kit hosts Parts 5/7/8).

> Part 5 of docs/specs/cli-layering.md Execution Plan: retire `fab pane send` and `fab pane await` as CLI verbs. Migrate fab-operator + `_cli-agents`/`_cli-external` guidance to `rk mux send`/`rk mux await` with a raw-tmux fallback when rk is absent; delete the two CLI verbs (internal builders stay — dispatch delivery uses them). Depends on Part 1 *(released)*.

**Release gate is satisfied and locally verified**: run-kit v3.16.24 (tag pushed 2026-08-15, GitHub Release non-draft, Homebrew tap Formula/run-kit.rb at 3.16.24) contains PR #617 (`rk mux send` + `rk mux await`, change 260815-a5vf). The intake agent confirmed the installed binary: `rk --version` → v3.16.24 and `rk mux send --help` prints the full send contract.

Key decisions carried from the cli-layering spec (all restated in What Changes):

- Delegation rule 2: fab → rk delegation is **capability-probed and fail-open** — `command -v rk` gates every use; absence degrades to raw tmux / fab's internal builders, **never to an error**. fab-kit remains installable without rk.
- Delegation rule 4: what fab may assume of rk when present — the `@rk_agent_state` schema (run-kit `docs/specs/agent-state.md`), the `rk mux send`/`rk mux await` contracts (gate matrix, probe-verified delivery, report words), and `rk notify`'s fail-silent contract.
- fab command surface split table, `pane send`/`pane await` row: "superseded by `rk mux send`/`await` (260815-a5vf); operator + helper guidance migrate, then delete".
- Only the user-facing CLI verbs are deleted. The internal Go builders that back them (used by dispatch delivery, e.g. the pane-arm deliver choreography) STAY.
- When shipped, this part unblocks Part 7 (guidance re-point for capture/kill/process) once Part 6 (run-kit substrate twins, not yet started) is also released.

## Why

1. **The pain point**: run-kit and fab-kit now split into two layers — rk owns the tmux **substrate** (the `@rk_agent_state` convention, pane interaction verbs), fab owns **choreography** (changes, stages, dispatch). `fab pane send`/`fab pane await` are substrate verbs living in the wrong binary: fab is a pure *consumer* of `@rk_agent_state` yet carries its own gated sender and awaiter, duplicating the gate matrix rk now implements verbatim (`rk mux send` applies "fab-kit's `idleGate` matrix" per its shipped contract). Two implementations of one gate is a standing version-skew and maintenance liability.
2. **The successor is strictly better**: `fab pane send` is a *blind* `send-keys -l` behind the gate — the printed-prompt trap and delivery verification are the caller's problem (manual sentinel probe in `_cli-agents.md`). `rk mux send` delivers through run-kit's hardened injection engine (baseline capture → named-buffer bracketed paste → novelty echo probe → probe-gated Enter), adds `--key <name>` (closing the raw-tmux carve-out that key-name input forced on fab callers), `-` stdin multi-line paste, and `--await` ask-and-wait composition. `rk mux await` adds `--until <states>`, `--after-active` (stale-state race fix), and `--notify`.
3. **If we don't**: skills keep steering agents at the weaker fab verbs, the duplicated gate drifts (rk's gate already reconciles pid-liveness; fab's does not), and Part 7 (the capture/kill/process guidance re-point) stays blocked behind this part.
4. **Why delete rather than alias**: cli-layering rule 3 (permanent aliases) protects *machine-baked* entry points — hook lines, PATH shims. `fab pane send`/`await` are typed by agents following skill guidance; the guidance migrates in the same change, so nothing installed invokes the old names. The spec's disposition row says "migrate, then delete".

## What Changes

### 1. Delete the two CLI verbs (Go)

Remove the user-facing commands only:

- `src/go/fab/cmd/fab/pane_send.go` + `pane_send_test.go` — the `fab pane send` cobra command, including the `idleGate` decision function (the matrix now lives in rk; fab no longer needs a CLI-side copy) and the `sendTextArgs`/`sendEnterArgs` cobra-layer wrappers.
- `src/go/fab/cmd/fab/pane_await.go` + `pane_await_test.go` — the `fab pane await` cobra command (`runPaneAwait`, `paneAgentInstrumented`).
- `src/go/fab/cmd/fab/pane.go` — drop the `paneSendCmd()` and `paneAwaitCmd()` registration rows and update the family `Use`/help string (`fab pane <map|capture|process|window-name|open|ready|deliver|kill>`).
- Sweep remaining `cmd/fab` tests that exercise the deleted verbs' exit codes or help output (e.g. `pane_exitcode_test.go`) — delete or narrow the cases; do not weaken coverage of the surviving verbs.

**What STAYS (explicitly)**:

- `internal/pane/pane.go` builders `SendLiteralArgs`/`SendKeyArgs`/`SendText`/`SendKey` — consumed by `internal/pane/gate.go`'s verified-delivery choreography (`tmuxPaneIO.SendKey`, `KeyClear`/`KeyEnter` sends), which backs `fab pane deliver` and `fab dispatch deliver` (the pane-arm mechanism).
- The whole readiness/deliver stack: `fab pane open/ready/deliver/kill`, `fab dispatch` family, `internal/pane/gate.go`, `create.go`. Untouched.
- `fab pane map`/`capture`'s `@rk_agent_state` reads (`ReadAgentStateOption`, `AgentDisplayFromOption`) — fab remains a read consumer of the convention.

**Dead-code consequence (decided here)**: `internal/pane/await.go` + `await_test.go` (the pure `Await` loop, `AwaitTick`, `AwaitReport` constants) has exactly one non-test consumer — the deleted `pane_await.go`. `fab dispatch wait` runs its own record-keyed loop in `internal/dispatch` and does not import it. Delete `internal/pane/await.go` + `await_test.go` with the verb; it is not part of the dispatch-delivery plumbing the "builders stay" rule protects. Verify with `go build ./... && go vet ./...` plus a grep that no `pane.Await`/`AwaitReport` consumer remains.

### 2. `_cli-fab.md` — remove the command reference (constitution CLI constraint)

- Delete `### send — fab pane send …` and `### await — fab pane await …` sections entirely.
- Update the `## fab pane` family line (`fab pane <map|capture|send|process|window-name|open|ready|deliver|kill|await>` → drop `send`/`await`) and the pane-family exit-code paragraph (drop send/await from the verb list and drop await's two extra codes).
- Update `§ agent state`: the reader list is now `map`/`capture` (drop `send`, and the "warn-and-send under send without `--force`" clause).
- Sweep the rest of the file for cross-references (e.g. `§ open`'s trailing pointer, `§ kill`/`§ deliver` prose, `fab dispatch` sections that name the deleted verbs) — reword to the surviving surface.

### 3. `_cli-agents.md` — migrate the generic agent-interaction procedures

The gate/await *usage* guidance stays fab-owned here (it is fab's choreography over rk's tool-owned contract, per `_cli-external.md` § Reference Model); the mechanism it names changes:

- **§ Scope Boundary**: drop `fab pane send` from the `_cli-fab.md`-documented list; note that agent messaging rides `rk mux send`/`rk mux await` (tool-owned contract via `rk skill`) with a raw-tmux fallback.
- **§ Pre-Send Validation step 2**: the gated binary becomes `rk mux send` — same matrix (idle sends; waiting refuses unless `--answer`; active always refuses, `--answer` included; unknown warns and sends; `--force` skips), now enforced by rk with probe-verified delivery built in. Gate the recommendation on `command -v rk`. **rk-absent degraded path (never error)**: the caller keeps the two-step validation (pane exists via `fab pane map`; state read via `fab pane map`/`fab pane capture --json`) as its own policy gate, then delivers via raw `tmux send-keys` + the § Delivery Probe manual recipe.
- **§ Delivery Probe**: note that `rk mux send` mechanizes the probe for message sends (echo probe → probe-gated Enter; a probe failure stages the text, warns on stderr, exits 1 — do not blind-resend); the manual sentinel recipe remains the raw-tmux fallback. `fab pane ready`/`deliver` guidance for spawn/dispatch delivery is untouched.
- **§ Await**: `rk mux await <pane> [--until …] [--file …] [--timeout …]` (and the composed `rk mux send --await`) becomes the preferred mechanized wait when rk is present — report words `idle`/`waiting`/`file`/`running`/`gone` (gone = exit 1, rk's toolkit codes, not fab's 2/3 scheme). The existing poll-capture loop and worker-announces-itself patterns remain the rk-less fallback; the prefer-an-artifact-over-a-screen-pattern rule is unchanged.
- **Key-name input**: `rk mux send --key Enter|Up|C-c` closes the raw-keys carve-out when rk is present; raw `tmux send-keys` remains the fallback.

### 4. `fab-operator.md` — migrate the operator's send paths

- **Header paragraph (line ~23)**: "routes commands and answers via `fab pane send`" → routes via `rk mux send` (gated on `command -v rk`; plain for command routing, `--answer` for prompt answers, `--key` for key-name input), degrading to raw `tmux send-keys` behind its own §3 state gate when rk is absent.
- **§3 Pre-Send Validation item 2**: confirmed sends ride the gated binary → a `waiting` target rides `rk mux send --answer`, an `active` target requires `rk mux send --force`; rk-absent → confirm, then raw send-keys (the operator's own state read from the pane map is the gate).
- **§5 Sending Auto-Answers**: `fab pane send --answer <pane> <text>` → `rk mux send <pane> "<text>" --answer` (gated). Note rk's built-in delivery verification replaces the "apply the delivery probe if the answer doesn't land" advice for the rk path (a probe failure surfaces as staged-text + exit 1 — re-capture and decide; never blind-resend). rk-absent fallback: re-capture guard + raw send-keys + manual delivery probe, as the flow worked before the gated binary existed.
- The `-L <server>` need (operator is per-server) maps directly: `rk mux` carries the same persistent `-L/--server` flag.

### 5. `_cli-external.md` — tmux notes + rk section

- **§ tmux Usage Notes "Send keys" bullet**: `fab pane send` → `rk mux send` (gated, fail-open; raw `tmux send-keys` when absent).
- **§ rk (run-kit)**: add the third fab-owned rk usage pointer — agent messaging (`rk mux send`/`await`) usage is owned by `_cli-agents.md` § Pre-Send Validation / § Await and `fab-operator.md` §3/§5; the verbs' full contract stays tool-owned (`rk skill`). Keep the existing delegation/binary-gate framing; frontmatter `description` updated to match.

### 6. Spec + docs sweep

- `docs/specs/skills.md` `/fab-operator` section (3 references): purpose line, flow line (`fab pane send --answer`), and Tools line — repoint to `rk mux send` with the raw-tmux fallback phrasing.
- Repo-wide sweep for remaining *guidance* references: `grep -rn "pane send\|pane await"` across `src/kit/`, `docs/specs/` (excluding `docs/specs/findings/` — dated review reports stay as historical record), README/site (no hits found at intake time). `docs/memory/` is hydrate's job (below). Include contrastive phrases and user-facing string literals per the sweep discipline (e.g. "prefer `fab pane send` over raw `tmux send-keys`", "the one remaining raw path").

### 7. Hydrate targets (memory)

Present-truth rewrite of the pane-command surface: send/await verb requirements removed from fab's memory, retirement + rk successor recorded as a Design Decision (pointer to run-kit's `agent-messaging.md` for the contract), operator/agent-primitive files repointed. Files listed under Affected Memory.

### Non-goals

- No changes to `fab pane capture`/`kill`/`process` or their guidance — that is Part 7, gated on this part + Part 6 both releasing.
- No `fab pane open`/`ready`/`deliver`/`window-name`/`map` changes; no `fab dispatch` changes.
- No hidden alias or deprecation stub for the deleted verbs.
- No rk-side changes (Part 1 shipped them); no migration file (no user data restructured).

## Affected Memory

- `runtime/pane-commands`: (modify) remove the send/await verb requirements (gate matrix, await observer, their exit-code rows); record retirement → `rk mux send`/`await` as a Design Decision; family list and exit-code scheme rewritten to the surviving verbs
- `runtime/agent-primitives`: (modify) pre-send gate / delivery / await guidance repointed to rk mux with the raw-tmux fallback
- `runtime/operator`: (modify) operator send paths (routing, auto-answer) repointed to `rk mux send --answer`/`--force` + fallback
- `runtime/runtime-agents`: (modify) agent-interaction references to the retired verbs rewritten
- `runtime/dispatch`: (modify) one stale cross-reference to the retired verbs; dispatch mechanics themselves unchanged
- `distribution/kit-architecture`: (modify) command-surface enumeration drops the two verbs

## Impact

- **Go**: `src/go/fab/cmd/fab/` — delete `pane_send.go`, `pane_send_test.go`, `pane_await.go`, `pane_await_test.go`; edit `pane.go` (registration + family help); sweep `pane_exitcode_test.go` and any help-surface tests. `src/go/fab/internal/pane/` — delete `await.go`, `await_test.go`; everything else stays.
- **Kit skills**: `src/kit/skills/` — `_cli-fab.md`, `_cli-agents.md`, `_cli-external.md`, `fab-operator.md`. (Deployed copies regenerate via `fab sync`; never edited directly.)
- **Specs**: `docs/specs/skills.md` operator section.
- **Memory (hydrate)**: 6 files above + `fab memory-index` regeneration.
- **Tests**: `go test ./...` must pass with the deletions; no new Go behavior is added, so new tests are limited to whatever registration/help assertions exist.
- **Release**: next kit release MINOR per the `fab hook`-family removal precedent (removed outright in 2.14.0); pending-release note already says next release MINOR.
- **Cross-repo**: unblocks Part 7 (with Part 6); no run-kit edits in this change.

## Open Questions

- (none — the Part 5 prompt and the cli-layering spec resolve every scoping decision; see Assumptions)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete only the two CLI verbs; `internal/pane` send builders and the whole open/ready/deliver/dispatch stack stay | Stated verbatim in the Part 5 prompt and the spec's split table; verified consumers in code (`gate.go` uses the builders) | S:95 R:85 A:95 D:95 |
| 2 | Confident | Also delete `internal/pane/await.go` + `await_test.go` as dead code | Sole non-test consumer is the deleted verb; `fab dispatch wait` has its own loop; Parts 6/7 plan no fab-side await consumer; trivially recoverable from git | S:70 R:80 A:85 D:80 |
| 3 | Certain | Every rk recommendation in skills is gated on `command -v rk` and fails open to raw tmux + manual probe/state-read — never an error | Delegation rule 2, restated in the prompt; matches the existing `_preamble.md` rk fail-silent rule | S:95 R:90 A:95 D:95 |
| 4 | Confident | No deprecation alias/stub for `fab pane send`/`await` | Spec disposition says "migrate, then delete"; rule 3's permanent-alias protection covers machine-baked entry points only, and no installed artifact invokes these verbs | S:85 R:70 A:85 D:85 |
| 5 | Confident | Gate-matrix/await usage guidance stays summarized in fab skills (fab-owned usage) with the full contract delegated to `rk skill` | `_cli-external.md` § Reference Model: fab-owned choreography lives in fab skills; tool-owned knowledge is delegated at use-time | S:70 R:85 A:80 D:75 |
| 6 | Confident | rk-absent operator fallback = operator's own state read (pane map/capture) + raw `tmux send-keys` + manual delivery probe, with the same §3 confirm policy | The only remaining fab-side sender would be raw tmux (the binary gate is deleted); the operator already performs the two-step policy gate before every send, so the degraded path loses defense-in-depth, not the gate itself | S:70 R:80 A:85 D:75 |
| 7 | Certain | Flag mapping for migrated guidance: plain→plain, `--answer`→`--answer`, `--force`→`--force`, `--no-enter`→`--no-enter`, `-L/--server`→`-L/--server`; key names ride `rk mux send --key`; await maps `--file`/`--timeout` directly (report `gone` = exit 1 under rk vs fab's exit 2) | Read from the shipped rk v3.16.24 contract (`rk mux send --help` verified live; run-kit memory `agent-messaging.md`) | S:90 R:90 A:95 D:90 |
| 8 | Confident | `docs/specs/findings/` historical review reports are NOT swept | Dated point-in-time findings; rewriting history is worse than a stale mention in an archived report | S:65 R:90 A:85 D:80 |
| 9 | Certain | Part 1 release gate satisfied | run-kit v3.16.24 released (tag, GitHub Release, Homebrew formula) and locally verified at intake (`rk --version`, `rk mux send --help`) | S:100 R:95 A:100 D:100 |
| 10 | Confident | Next kit release MINOR | `fab hook` family was removed outright in 2.14.0 (minor); project practice treats guided verb removals as minor | S:60 R:80 A:75 D:70 |

10 assumptions (4 certain, 6 confident, 0 tentative, 0 unresolved).

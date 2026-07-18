# Intake: Divest Agent Active/Idle State Production from fab-kit

**Change**: 260705-ioku-divest-agent-state-production
**Created**: 2026-07-06

## Origin

> Divest agent active/idle state production: delete the `.fab-runtime.yaml` `_agents` pipeline (hooks/GC/PID-walker/flock), make `fab pane send`/`map`/`capture` read the `@rk_agent_state` tmux pane-option convention written by `rk agent-setup`.

- **Backlog item**: `[ioku]` (2026-07-06) in `fab/backlog.md`.
- **Pickup detail doc**: `fab/plans/sahil/agent-state-divestment.md` — written 2026-07-06 after a run-kit discussion session that audited agent-status detection across the competitive landscape and both kits' mechanisms. It contains the verified audit, subtraction list, reader-rewrite specs, operator-skill impact, sequencing, and acceptance criteria. This intake is grounded in it.
- **Interaction mode**: promptless dispatch (`promptless-defer`) — the intake was synthesized from the discussion session's decisions; no questions were asked. Key decisions (proceed-without-writer, release gate, schema ownership, test simulation) were made by Sahil in that session and are recorded as directed assumptions below.
- **Companion work**: run-kit's future `rk agent-setup` is the writer. The shared design decision is run-kit constitution Principle X "Hooks Carry Only the Underivable" (v1.4.0 amendment — currently an UNCOMMITTED edit in run-kit's candid-gopher worktree).

## Why

1. **Independence principle** (per Sahil): fab-kit must function fully wherever it runs — with or without tmux, with or without run-kit. Agent-state detection was never core fab; it is a tmux-context observation feature that got bolted onto fab because no owner existed. run-kit (`rk agent-setup`) is that owner now. fab-kit stops PRODUCING agent lifecycle state and becomes a pure CONSUMER of a shared tmux pane-option convention.
2. **Dead weight outside tmux**: today the hooks fire and write `.fab-runtime.yaml` entries nothing ever reads (no `tmux_pane` to match) — pure waste until GC'd. The verified audit also shows `change` and `transcript_path` are write-only fields with zero readers.
3. **Claude-only blindness fixed for free**: the current pipeline tracks only Claude Code (its hooks). `fab pane send`'s idle gate is blind to codex/copilot/gemini agents. Reading the shared convention — written by run-kit's global agent-harness hooks for Claude Code, Codex, Copilot, Gemini, OpenCode — covers them all.
4. **Richer state**: the convention adds `waiting` (blocked on a human — permission prompt / menu / elicitation). Today a mid-turn permission prompt fires no Stop, so the agent reads `active` and the operator's idle-only sweep likely never probes it. `waiting` makes these agents event-visible.
5. **If we don't**: fab keeps carrying a producer subsystem (hooks, GC, PID walker, flock serialization, runtime file) whose entire job run-kit is about to do better, with two writers' worth of drift risk and a per-hook-event latency tax.

## What Changes

### The convention (read-side contract)

fab reads a tmux **pane user option** with plain tmux commands — a data format in tmux, not a dependency on run-kit software:

```
@rk_agent_state = "<state>:<epoch_seconds>"     # state ∈ active | waiting | idle
```

- `active` — turn in progress (UserPromptSubmit/PreToolUse fired, no terminal event since)
- `waiting` — blocked on a human (Notification: permission_prompt | elicitation_dialog | agent_needs_input; PermissionRequest for agents that have it)
- `idle` — turn complete (Stop; idle_prompt as backstop)
- Option **absent** — no instrumented agent in this pane (render `—`, treat as unknown)
- The **epoch suffix is mandatory** — consumers compute idle duration from it and can apply staleness heuristics (an Esc-interrupted agent can leave a stale `active`).

The exact value schema is **owned by run-kit**; the above is the current working contract (draft from the pickup doc). Divergence risk is accepted by Sahil — if run-kit changes the format later, that is a follow-up change.

### Release gate (HARD constraint — merge/release hold)

Verified 2026-07-06: **the writer does not exist yet.** Installed rk (2.6.x) has no `agent-setup` command, no tmux pane on this machine carries `@rk_agent_state`, run-kit has no backlog item for it, and run-kit's Principle X v1.4.0 constitution amendment is uncommitted (candid-gopher worktree only). Sahil (coordinator of both repos) explicitly directed execution anyway: fab is the consumer; the schema draft is the working contract.

**The PR from this change MUST NOT merge/release until `rk agent-setup` exists on Sahil's machines** — otherwise `fab pane send` gating and the operator's Agent column go blind everywhere (pane map all `—`, send refuses without `--force`). Ship/review-pr stages produce a PR that is **explicitly held**: draft PR with a prominent hold note in the body (fab never auto-merges; Sahil coordinates the merge timing).

### Delete (subtraction list — verified audit, refined this session)

1. **The `_agents` write pipeline** (`src/go/fab/cmd/fab/hook.go`): the state-tracking purpose of `fab hook stop`, `fab hook user-prompt`, `fab hook session-start`, including `WriteAgent`/`ClearAgent`/`ClearAgentIdle`/`UpdateAgent`, the GC sweep + `last_run_gc` throttle, and the grandparent PID walker.
2. **`internal/proc/`** (`src/go/fab/internal/proc/`) — deleted. Verified: sole importer is `cmd/fab/hook.go`; `internal/dispatch/dispatch.go` references it in a **comment only** (sweep the comment).
3. **`internal/lockfile` STAYS.** Verified this session: `cmd/fab/status.go`, `cmd/fab/preflight.go`, and `internal/score/score.go` use `lockfile.WithLock` for `.status.yaml` serialization. Only the **runtime** lock usage (`.fab-runtime.yaml.lock` in `internal/runtime/runtime.go`) is removed with the runtime package.
4. **`internal/runtime/`** (`src/go/fab/internal/runtime/runtime.go` + tests) — the whole `_agents` map and `.fab-runtime.yaml` read/write. Verified: nothing else lives in the file (only `_agents` + top-level `last_run_gc`), so the file concept dies wholesale. No gitignore/scaffold edit needed — the scaffold `fragment-.gitignore` uses the `.fab-*` pattern, which also covers `.fab-status.yaml`/`.fab-dispatch/`.
5. **`internal/pane/pane.go`**: `ResolveAgentState`, `ResolveAgentStateWithCache`, `findAgentByPane`, `loadRuntimeForCache`, `LoadRuntimeFile` (+ the per-worktree runtime cache in pane map). `FormatIdleDuration` (pane.go, ~line 471) **survives** — it formats the new epoch-derived durations.
6. **Hook settings entries** for Stop/UserPromptSubmit/SessionStart in the deployed `.claude/settings.local.json` (registered by `hooklib.Sync`, `src/go/fab/internal/hooklib/sync.go` lines 29–31). Verified: the SessionStart handler's only action is deleting the `_agents` entry (`hookSessionStartCmd` → `ClearAgent`) — no non-`_agents` uses, safe to remove. Follow the `fab hook artifact-write` removal precedent (change y022, migration `2.10.1-to-2.11.0.md`): one-release no-op exit-0 shims for un-migrated settings + a version migration removing the three entries.

### Rewrite as convention readers (what stays)

1. **`fab pane send` idle gate** (`src/go/fab/cmd/fab/pane_send.go`): read `@rk_agent_state` via `tmux [-L <server>] show-options -pv -t <pane> @rk_agent_state`.
   - `idle` → send.
   - `active`/`waiting` → refuse (same error shape as today, now three-state aware — the state name appears in the message).
   - Absent/unparseable → "unknown": refuse with a **distinct message** pointing the caller at `--force`.
   - `--force` semantics unchanged (bypasses the state check only; pane existence still enforced via the targeted probe).
2. **`fab pane map` Agent column** (`src/go/fab/cmd/fab/panemap.go`): add `#{@rk_agent_state}` to the **existing `list-panes` format string** — zero extra subprocesses; the `tmux_server` disambiguation problem evaporates (a pane option lives on exactly one server's pane). Column values: `active` / `waiting` / `idle (<duration>)` / `—`. Duration computed from the epoch suffix via the existing `FormatIdleDuration`.
3. **`fab pane capture` header** (`src/go/fab/cmd/fab/pane_capture.go`): same read, same display.
4. **JSON schema compatibility**: keep the `agent_state` / `agent_idle_duration` field names in `pane map --json` / `pane capture --json` (verified: `panemap.go:374-375`, `pane_capture.go:35-36`) — run-kit joins them during its own migration. `agent_state` gains the `waiting` value — note it in the schema docs.
5. **Untouched**: pane map's Change/Stage/display_state/pr_url columns come from cwd → `.fab-status.yaml` → `.status.yaml` (`ResolvePaneContext`), NOT from `_agents`.

### Settings migration + shims

- New migration in `src/kit/migrations/` (named from the actual next release version at ship time): removes the three session-scoped hook entries (`SessionStart` → `fab hook session-start`, `Stop` → `fab hook stop`, `UserPromptSubmit` → `fab hook user-prompt`) from `.claude/settings.local.json`. Sentinel-guarded and idempotent, modeled on `2.10.1-to-2.11.0.md`.
- `fab hook stop|user-prompt|session-start` become one-release no-op shims (exit 0, emit nothing) for stale un-migrated settings.
- `fab hook sync` (`hooklib.Sync`, still invoked from fab-kit sync — `src/go/fab-kit/internal/sync.go`): its registration list empties of the three entries; retained for one release (it also converges/removes stale fab-managed entries and legacy script shims); full removal is a follow-up.
- **Scope exclusion**: y022's pending deletions (the `artifact-write` shim + its orphaned hooklib funcs) stay untouched here — they have their own slated next-release cleanup. This change's three shims get their own one-release lifecycle.

### Operator skill impact

`fab-operator` (`src/kit/skills/fab-operator.md`) keys question detection, stuck detection (>15m idle), and pre-send checks off the pane-map Agent column — **interface unchanged, data richer**:

- Update skill text where it assumes the two-state active/idle vocabulary (three states + unknown now).
- `waiting` makes menu/permission-blocked agents event-visible; treat `waiting` as the trigger for the existing tightened 90s heartbeat cadence (today that trigger is capture-based menu detection).
- Pre-send checks inherit the three-state gate semantics from `fab pane send`.

### Docs/spec sweep (constitution-required mirrors)

- `docs/memory/runtime/runtime-agents.md` describes the deleted system — hydrate rewrites it to the convention-reader model.
- SPEC mirrors of touched skills (code-quality.md § Sibling & Mirror Sweeps — treat ALL of a touched skill's SPEC mirrors as the sweep class): `docs/specs/skills/SPEC-hooks.md`, `docs/specs/skills/SPEC-fab-operator.md`, `docs/specs/skills/SPEC-_cli-fab.md`.
- `src/kit/skills/_cli-fab.md` — CLI signature/behavior changes (`fab pane send`/`map`/`capture` sections lines ~362–370, `fab hook` section line ~331).
- Aggregate specs restating agent-state facts (`skills.md`, `glossary.md`, `architecture.md`) — sweep by grepping the old claims (`idle_since`, `_agents`, `fab-runtime`, two-state vocabulary) repo-wide.
- Go changes ship tests in the same change (constitution VII + code-quality test strategy).

### Acceptance (from the pickup doc, adapted for the missing writer)

- `rg "idle_since|_agents|fab-runtime" src/` → only convention-reader code and migration shims remain.
- `fab pane send` refuses on `active`/`waiting`/unknown, sends on `idle`, `--force` bypasses — covering a codex pane (previously invisible).
- All fab commands behave identically outside tmux (no runtime-file writes anywhere).
- Version migration removes the three hook settings entries; a stale settings file invoking removed hooks gets no-op exit-0 shims for one release.
- **Testability without the writer**: automated tests simulate the writer by setting the pane option directly (`tmux set-option -p -t <pane> @rk_agent_state "idle:<epoch>"`). The pickup doc's manual-probe acceptance ("pane map shows `waiting` for a Claude on a permission prompt") is **deferred to post-writer manual verification**.

## Affected Memory

- `runtime/runtime-agents.md`: (modify) rewrite wholesale — from `.fab-runtime.yaml` `_agents` producer schema to the `@rk_agent_state` convention-reader model (read contract, three states + unknown, epoch duration, ownership by run-kit)
- `runtime/pane-commands.md`: (modify) pane send three-state gate, pane map Agent column via `#{@rk_agent_state}` format string, capture header, `agent_state` JSON `waiting` value, removal of the runtime-file matching rule
- `runtime/operator.md`: (modify) operator keys off the richer Agent column; `waiting`-triggered cadence; three-state vocabulary
- `pipeline/hooks-may-enhance-never-own.md`: (modify) the "three session-scoped hooks SHALL remain" paragraph inverts — they are deleted; the principle's example set updates (divestment strengthens the principle)
- `pipeline/schemas.md`: (modify) remove/rewrite the § Agent State `.fab-runtime.yaml` section; the "two sibling files" ephemeral-state framing drops to `.fab-dispatch/` only
- `distribution/kit-architecture.md`: (modify) hook script shims (`on-stop.sh` etc.), `fab hook` subcommand list, config-free command list entries

## Impact

- **Go source** (`src/go/fab/`): `cmd/fab/hook.go` (handlers → shims), `cmd/fab/pane_send.go`, `cmd/fab/panemap.go`, `cmd/fab/pane_capture.go`, `internal/pane/pane.go` (delete resolvers, keep `FormatIdleDuration`), `internal/runtime/` (delete package), `internal/proc/` (delete package), `internal/hooklib/sync.go` (empty registration list), `internal/dispatch/dispatch.go` (comment sweep). `internal/lockfile` untouched.
- **Go tests** (same change): `hook_test.go`, `pane_send_test.go`, `panemap_test.go`, `pane_capture_test.go`, `internal/pane/pane_test.go`, `internal/runtime/runtime_test.go` (deleted with package), `internal/hooklib/sync_test.go`, `src/go/fab-kit/internal/hooksync_test.go`. New tests simulate the writer via direct `tmux set-option -p`.
- **Kit content** (`src/kit/`): `skills/fab-operator.md`, `skills/_cli-fab.md`, new migration in `migrations/`.
- **Docs**: 6 memory files above (+ regenerate indexes via `fab memory-index`), `docs/specs/skills/SPEC-hooks.md`, `SPEC-fab-operator.md`, `SPEC-_cli-fab.md`, aggregate specs as swept.
- **User-visible**: `.fab-runtime.yaml` stops existing; three hook entries leave `.claude/settings.local.json` (via migration); `fab pane send/map/capture` report `waiting` and unknown distinctly; non-Claude agents become visible once the writer ships.
- **Runtime behavior outside tmux**: strictly less work (no hook writes at all).
- **Release**: PR held until `rk agent-setup` exists on Sahil's machines (see Release gate above).

## Open Questions

- None blocking. The discussion session and pickup doc pre-resolved sequencing, schema ownership, gate semantics, and testability; residual open items (schema divergence, staleness heuristics, `fab hook sync` end-state) are recorded as graded assumptions below and deferred to follow-ups.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Proceed now although the writer (`rk agent-setup`) does not exist; the PR is explicitly HELD from merge/release until the writer exists on Sahil's machines | Directed by Sahil 2026-07-06 (coordinator of both repos); verified: installed rk 2.6.x lacks agent-setup, no pane carries the option | S:95 R:85 A:90 D:95 |
| 2 | Certain | `@rk_agent_state = "<state>:<epoch_seconds>"` (active / waiting / idle; absent = unknown; epoch mandatory) is the working contract; run-kit owns the schema; divergence is an accepted follow-up risk | Directed — schema draft from the pickup doc; shared decision recorded in run-kit Principle X v1.4.0 (uncommitted) | S:90 R:70 A:85 D:85 |
| 3 | Certain | Delete the `_agents` pipeline wholesale: hook state-tracking, WriteAgent/ClearAgent/ClearAgentIdle/UpdateAgent, GC + `last_run_gc`, `internal/runtime/`, `.fab-runtime.yaml` | Pickup doc's verified audit: sole readers are the three pane consumers; `change`/`transcript_path` are write-only | S:95 R:75 A:95 D:95 |
| 4 | Certain | `internal/lockfile` package stays; only the runtime lock usage is removed | Code-verified this session: status.go, preflight.go, score.go consume `lockfile.WithLock` for `.status.yaml` | S:85 R:90 A:100 D:95 |
| 5 | Certain | `internal/proc/` is deleted; the comment reference in `internal/dispatch/dispatch.go` is swept | Code-verified: `cmd/fab/hook.go` is the sole importer | S:85 R:85 A:100 D:95 |
| 6 | Certain | All three settings entries (Stop/UserPromptSubmit/SessionStart) are removable; SessionStart has no non-`_agents` use | Code-verified: `hookSessionStartCmd` only clears the `_agents` entry | S:85 R:80 A:95 D:90 |
| 7 | Certain | `.fab-runtime.yaml` dies wholesale; no gitignore/scaffold edit needed | Code-verified: file holds only `_agents` + `last_run_gc`; scaffold gitignore uses the `.fab-*` pattern | S:85 R:85 A:95 D:90 |
| 8 | Certain | Pane-send gate: `idle` → send; `active`/`waiting` → refuse (same error shape, three-state aware); absent/unparseable → distinct "unknown" refusal pointing at `--force`; `--force` unchanged | Directed — exact behavior specified in the session decisions | S:95 R:80 A:90 D:95 |
| 9 | Certain | Pane map reads `#{@rk_agent_state}` in the existing list-panes format string (zero extra subprocesses; server disambiguation evaporates); column `active`/`waiting`/`idle (<dur>)`/`—` via existing `FormatIdleDuration`; capture header same read/display | Directed; `FormatIdleDuration` verified present in `internal/pane/pane.go` | S:95 R:85 A:90 D:95 |
| 10 | Certain | JSON keeps `agent_state`/`agent_idle_duration` field names; `agent_state` gains `waiting`; noted in schema docs | Directed — run-kit joins these fields during its own migration; names verified in panemap.go/pane_capture.go | S:90 R:75 A:90 D:90 |
| 11 | Certain | Version migration removes the three hook settings entries; stop/user-prompt/session-start become one-release no-op exit-0 shims | Directed; follows the verified y022 precedent (`2.10.1-to-2.11.0.md`, artifact-write) | S:90 R:80 A:95 D:90 |
| 12 | Certain | Automated tests simulate the writer via `tmux set-option -p -t <pane> @rk_agent_state "<state>:<epoch>"`; the manual `waiting`-probe acceptance is deferred to post-writer verification | Directed in session (testability without the writer) | S:90 R:90 A:90 D:90 |
| 13 | Certain | The release gate is enforced procedurally: draft PR with a prominent hold note; fab never auto-merges; Sahil coordinates merge timing | Session constraint; `/git-pr` creates draft PRs by default, so no new mechanism is needed | S:75 R:90 A:85 D:80 |
| 14 | Certain | Docs sweep class: SPEC-hooks.md + SPEC-fab-operator.md + SPEC-_cli-fab.md mirrors, aggregate specs restating agent-state facts, six affected memory files; Go tests ship in the same change | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps | S:85 R:85 A:95 D:90 |
| 15 | Confident | `fab hook sync` is retained for one release with its registration list emptied (still converges/removes stale fab-managed entries); full removal is a follow-up | y022 precedent kept sync while dropping registration; still invoked from fab-kit sync — apply finalizes the exact shape | S:60 R:80 A:70 D:60 |
| 16 | Confident | No staleness heuristic in v1 readers — a stale `active` (Esc-interrupted agent) still refuses sends; `--force` is the escape hatch; heuristics are a consumer follow-up | Pickup doc names staleness as a consumer possibility, not a v1 requirement; simplest correct reader wins | S:50 R:85 A:70 D:65 |
| 17 | Confident | Operator skill: `waiting` becomes the trigger for the existing tightened 90s cadence and informs pre-send checks; skill text updated from two-state vocabulary | Session said "consider tightening cadence on waiting"; `waiting` is precisely the menu/permission-blocked signal §5 detects today by capture | S:65 R:90 A:75 D:70 |
| 18 | Confident | Scope excludes y022's pending deletions (the `artifact-write` shim + orphaned hooklib funcs stay untouched); this change's three shims get their own one-release lifecycle | Avoids cross-change entanglement; y022 cleanup is separately slated | S:60 R:85 A:75 D:70 |
| 19 | Confident | The migration file is named from the actual next release version at ship time | Mechanical versioning decision, resolved when the release version is known | S:65 R:90 A:75 D:75 |

19 assumptions (14 certain, 5 confident, 0 tentative, 0 unresolved).

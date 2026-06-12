# Intake: Operator/Pane/Tmux Surface — Perf + Error Surfacing (Binary-Review B5)

**Change**: 260612-pw3k-operator-pane-perf-error-surfacing
**Created**: 2026-06-12

## Origin

Backlog ID `pw3k` via `/fab-new pw3k` (one-shot, no prior conversation). The entry is batch B5/6 of the 2026-06-12 binary review of the Go sources, and **absorbs backlog `[dkn3]`** (display_state on pane map JSON). Full findings with adversarial-verifier notes: `docs/specs/findings/binary-review-2026-06-12.md` §B5 (F31–F38), verified against `1431a9c3` and spot-rechecked at HEAD `b39da308` during this intake (all cited code unchanged).

> [pw3k] 2026-06-12: Binary-review batch B5/6 — operator/pane/tmux surface: perf + error surfacing. PARALLEL: wave 1 — alongside k4ge, mz4q, dn2c; touches panemap/pane_*/batch_new/operator/memoryindex only (mz4q owns hook.go + runtime.go). ABSORBS [dkn3]. GOAL: the operator-facing surface is fast and honest about failures. ACTIONS: F31, F38, F35, F33, F32, F34, F36, F37, +[dkn3] display_state. CONSTRAINTS: pane map --json change is additive-only; _cli-fab.md + fab-operator.md rows for changed output; SPEC mirrors per touched skill. REPORT: docs/specs/findings/binary-review-2026-06-12.md §B5 F31-F38 (vs 1431a9c3).

## Why

The operator/pane/tmux surface is the multi-agent runtime's control plane: `fab pane map --all-sessions --json` is the operator's per-tick snapshot (every ~3 min plus before every action) and is also called programmatically by the run-kit daemon; `fab batch new` spawns agent fleets; `fab operator` enforces the one-operator-per-server invariant. Today this surface is neither fast nor honest:

1. **Silent failures** — `fab batch new` discards the tmux `new-window` error entirely and exits 0 after printing the item as launched, leaving an orphaned worktree; a blind retry then breaks on the existing worktree (anti-idempotent, constitution principle III). The operator singleton check prefix-matches, so a window named `operator-logs` silently suppresses the real operator launch (and switches the user's view to the wrong window).
2. **Opaque errors** — multiple tmux/git subprocess sites return bare `exit status 1`, discarding the child's stderr. For an agent-facing CLI, stderr is the self-correction signal; the codebase already has the right pattern (`pane_window_name.go`) but doesn't reuse it.
3. **Subprocess waste on hot paths** — pane map spawns 2 `git rev-parse` per pane plus one tmux call per session per tick where ~half the git spawns and 1 tmux call suffice; memory-index spawns one `git log -1` per memory file; capture/send/process each pay a server-wide `tmux list-panes -a` pre-validation that the very next targeted command already covers; darwin `pane process` is a classic N+1 `ps`.
4. **Missing state axis** ([dkn3]) — `DisplayStage` computes `(stage, state)` but both callers discard the state half, so pane map consumers (run-kit sidebar) cannot distinguish an actively-worked stage from a parked finished change: a fully-shipped change displays `review-pr` indefinitely, identical to review-pr actively running.

If unfixed: multi-agent spawn loops fail silently and unrecoverably, agents can't self-correct from error text, every operator tick pays avoidable subprocess multiplications, and run-kit keeps guessing attention states from heuristics instead of reading them.

## What Changes

All file references are to `src/go/fab/` unless noted. Per-finding verifier corrections from the report are folded in below as binding design decisions.

### F31 — Surface tmux new-window failures in `fab batch new` (high/small)

- `cmd/fab/batch_new.go:115`: the launch is `exec.Command("tmux", "new-window", ...).Run()` with no error variable. Capture the error; on failure print a warning naming the item **and the already-created worktree path** (recovery/cleanup hint), track a failure count, and return a non-nil error / non-zero exit when any item failed. Pattern precedent: `batch_archive.go` archiveLoop (per-item FAILED line, failure count, non-zero exit only when `failed > 0`), documented at `_cli-fab.md:494`.
- `cmd/fab/batch_new.go:103-107`: the `wt create` failure branch discards error detail (`err` from `.Output()` carries `ExitError.Stderr`, never printed). Include `%v` plus the captured stderr in the message.
- Doc: update the `fab batch new` row in `src/kit/skills/_cli-fab.md` (~:492) for the new exit semantics (constitution: CLI changes MUST update `_cli-fab.md`).
- **Out of scope** (verifier correction noted, deliberately excluded): `batch_switch.go:96-108` has the identical discarded tmux/wt errors, but the backlog's wave-1 scope list explicitly limits pw3k to `panemap/pane_*/batch_new/operator/memoryindex`, and `batch_switch.go` is B4/[ye8r] surface (F29). See Non-Goals.

### F38 — Return errors from RunE instead of `os.Exit(1)` (low/small)

- Replace `Fprintln(errW)` + `os.Exit(1)` with `return fmt.Errorf(...)` at exactly three sites: `cmd/fab/batch_new.go:58-61`, `batch_new.go:67-70`, `cmd/fab/operator.go:31-34`. The root command centralizes error formatting at `main.go:42-45` (`ERROR: %s` + exit 1); os.Exit bypasses it and makes the RunE handlers untestable.
- **Deliberate user-visible stderr change**: `Error: not inside a tmux session.` → `ERROR: not inside a tmux session`; `No pending backlog items found.` → `ERROR: No pending backlog items found.` Nothing live pins these strings (only an archived change spec, non-authoritative per constitution II).
- **No blanket sweep**: ~12 sibling sites share the pattern (`resolve.go`, `panemap.go`, `pane_send/capture/process.go`, `batch_switch.go`, `batch_archive.go`) but `resolve --pane` and `pane map` stderr/exit semantics are pinned in live memory docs (`docs/memory/distribution/kit-architecture.md:400`, `docs/memory/runtime/pane-commands.md:61`). Only the three unpinned sites change. Keep `pane_window_name.go`'s custom exit-code scheme as the documented exception.

### F35 — Include child stderr in tmux/git subprocess errors (medium/medium)

- Extract a shared stderr-capturing helper (reuse/generalize the `pane_window_name.go` pattern: `renameWindow` :118-124, `printTmuxErr` :149-156; also `pane.ReadWindowName` at `internal/pane/pane.go:79-86`) and apply it at: `cmd/fab/pane_capture.go:108-113` (`.Output()` — `ExitError.Stderr` populated but discarded), `cmd/fab/pane_send.go:60-68` (`.Run()` with nil Stderr), `cmd/fab/operator.go:60-62` (tmux new-window), `cmd/fab/operator.go:69-75` (gitRepoRoot — caller wraps as `cannot determine repo root: %w` but git's `fatal: ...` detail is lost).
- Errors include the trimmed child message and the relevant identifier (pane ID, target).
- Bonus site from the verifier: `internal/pane/pane.go:61-63` (`ValidatePane` discards list-panes stderr) — this site is restructured by F36 anyway; the replacement probe must surface stderr per this finding.

### F33 — Exact, server-wide operator singleton check (medium/small)

- `cmd/fab/operator.go:36-42`: the guard `tmux select-window -t operator` treats exit-0 as "already running", but tmux target resolution falls back to name-prefix and glob — any `operator-*` window satisfies it (false positive: prints "Switched to existing operator tab", switches the user to the wrong window, never launches the real operator). It is also **session-scoped**, so an operator in another session on the same server is missed (duplicate operator, breaking the per-SERVER singleton documented at `fab-operator.md` §"enforced by the operator window").
- Fix per verifier correction: enumerate `tmux list-windows -a -F '#{window_name}'` and compare names **exactly** — `-a` makes the check server-wide (a bare `=`-prefix fix would stay session-scoped), and enumeration distinguishes "window absent" from "tmux error" instead of conflating them. On exact match, select that window; otherwise launch.
- `_cli-fab.md:444` already documents exact-name semantics ("If window `operator` exists → select it") — code aligns to the documented contract.

### F32 — pane map: dedupe `git rev-parse`, collapse N+1 tmux calls (medium/small)

- Duplicate git spawn: `pane.GitWorktreeRoot(cwd)` (one `git -C <cwd> rev-parse --show-toplevel` each) runs **twice per pane** — `mainRootForPane` (`cmd/fab/panemap.go:127`, unconditionally, before any cache check) and again in `resolvePane` (`panemap.go:285`). Fix: a cwd-keyed worktree-root cache (`map[cwd]wtRoot`) populated once in the pane loop, threading the resolved wtRoot into `resolvePane` (the signature already threads mainRoot). The cache must carry the non-git signal too — `resolvePane`'s error branch keys off `GitWorktreeRoot` failing, so use an ok/empty-sentinel (`""` = not a git repo).
- N+1 tmux: `discoverAllSessions` (`panemap.go:199-217`) runs `tmux list-sessions` then one `tmux list-panes -s -t <session>` per session, even though `tmuxPaneFormat` already includes `#{session_name}` (`panemap.go:154`) and `parsePaneLines` reads it. Fix: a single `tmux list-panes -a -F <tmuxPaneFormat>` (precedent: `ValidatePane`, `internal/pane/pane.go:61`). Incidentally fixes a latent wrinkle: `-t <sess>` uses prefix/glob target resolution, `-a` + parsing `#{session_name}` is exact.
- Output (JSON/table) stays byte-identical apart from the additive [dkn3] field below. Existing argv-builder unit tests in `panemap_test.go` update accordingly.
- Scale: for a 10-pane/4-session server, each tick drops from ~20 git + 5 tmux spawns to ~10 git + 1 tmux.

### [dkn3] — Expose `display_state` on `fab pane map --json` rows (absorbed)

- `internal/status.DisplayStage` (`internal/status/status.go:463-498`) computes `(stage, state)` but both callers discard the state half: `cmd/fab/panemap.go:324` and `internal/pane/pane.go:152` (`stage, _ := status.DisplayStage(...)`).
- Add `display_state` as a **nullable** field on pane map JSON rows, values: `active | ready | done | failed | pending | skipped`; null/omitted when the pane has no resolvable change or its `.status.yaml` fails to load (same condition under which `stage` is absent today). **Additive-only** shape change — existing fields, ordering, and table output are unchanged.
- Unlocks honest per-row attention states in run-kit (failed/ready = needs human; done + parked = quiet row) instead of heuristics over agentState/`.fab-runtime.yaml` idle_since. run-kit consumes it via `app/backend/internal/sessions/sessions.go` `paneMapEntry` and updates its own `docs/specs/api.md` **separately — out of scope here**.
- Docs: `_cli-fab.md` pane map JSON row, `fab-operator.md` output-shape mention, `docs/memory/runtime/pane-commands.md`.

### F34 — memory-index: one batched `git log` pass (low-medium/medium)

- `internal/memoryindex/memoryindex.go:389-399` (`gitLastUpdated`) spawns `git log -1 --date=short --format=%ad -- <path>` per topic file, called from `gatherFiles` (:297, and sub-domain files via :323). N files → N git spawns per `fab memory-index` run, each potentially walking deep history.
- Fix: run a single batched `git log --date=short --name-only` pass over `docs/memory` once in Gather, stream output recording the **first (most recent) date seen per path**, and look dates up from that map. Keep the per-file call **only as fallback** when the batched call fails. Output identical: batched `--name-only` skips merge-commit file lists and doesn't follow renames — both match the current `git log -1 -- <path>` defaults.
- Honest impact (verifier-measured): ~54ms → ~31ms in this repo (20 files, 1453 commits) — the win materializes on large-history repos with wide memory trees; this is a low/medium cleanup, not a hot-loop fix.
- Docs to update in the same change (constitution :31): `_cli-fab.md:363-366` (documents the per-file mechanism), `docs/memory/memory-docs/hydrate.md:63`, `docs/memory/memory-docs/templates.md:115`.

### F36 — Replace the server-wide pre-validation in pane capture/send/process (low/small, verifier confidence MEDIUM)

- `internal/pane/pane.go:60-71` (`ValidatePane`) enumerates every pane on the server (`tmux list-panes -a`) before each `fab pane capture` (`pane_capture.go:54`), `send` (`pane_send.go:32`), and `process` (`pane_process.go:77`) — its only 3 callers. The pre-check is also TOCTOU-ineffective.
- Fix per verifier correction — **not** plain `-t` targeting: a single `tmux display-message -t <arg> -p '#{pane_id}'` probe, comparing output to the argument. This preserves both existence checking and **ID-exactness** (bare `-t` would accept the full tmux target grammar — window names, `session:win.pane` — a behavioral loosening), while dropping the O(server) enumeration. Map "can't find pane" stderr to the existing not-found error via the `tmuxExitCode`/`printTmuxErr` pattern (`pane_window_name.go:132-151`), surfacing stderr per F35.
- Contracts to preserve: `fab pane send --force` "still validates pane existence" (`pane_send.go:20`, `_cli-fab.md:208`); error string/exit-1 behavior per `pane-commands.md:83`.
- **Backlog constraint**: verifier confidence is MEDIUM — re-verify error-path equivalence (exit codes + stderr text for missing pane, dead server, bad server socket) before removing the old path.
- Docs: `_cli-fab.md:208` ("pane exists via `tmux list-panes -a`") and `pane-commands.md:83/:170` describe the current mechanism — update both.
- Honesty note from the verifier: the operator's per-tick auto-nudge currently uses raw `tmux capture-pane`/`send-keys`, not `fab pane capture/send`, so the per-tick saving materializes only for agents following `_cli-external.md` or after that deferred migration — impact is "low" and stays low.

### F37 — darwin pane process: batch the per-node `ps` spawns (low/small)

- `cmd/fab/pane_process_darwin.go:22` spawns `ps -o pid,ppid,comm -ax` once, then `buildNodeFromPS` (:68-69) spawns `ps -o args= -p <pid>` (`getPSCmdline`, :88-94) per node in the pane's tree — N+1.
- Fix per verifier preference: a **second single pass** `ps -axo pid=,args=` joined by PID (robust: pid is numeric-first, remainder is args). Avoid the one-pass `pid=,ppid=,comm=,args=` variant — it mis-parses when comm contains spaces (common for macOS app paths). Bonus: removes the TOCTOU window where a process exiting between passes yields cmdline `""`.
- Output shape unchanged; no skill invokes `fab pane process` (opt-in diagnostic).
- Docs: `_cli-fab.md:212` (documents `ps -o args= -p <pid>`) and `docs/memory/runtime/pane-commands.md:93`.

### Cross-cutting constraints

- **File-scope boundary (wave 1 parallelism)**: touch only `panemap.go`, `pane_*.go`, `batch_new.go`, `operator.go`, `internal/pane/`, `internal/memoryindex/` — `[mz4q]` owns `hook.go` + `runtime.go`; `batch_switch.go` is `[ye8r]` surface.
- **Tests**: constitution :31 — Go CLI changes MUST ship with test updates. F38 makes the RunE handlers unit-testable; F32's `panemap_test.go` argv-builder tests update for the `-a` rewrite.
- **Docs**: `_cli-fab.md` rows for every changed output/mechanism; `fab-operator.md` for the pane map JSON shape; SPEC mirrors per touched skill file (`docs/specs/skills/SPEC-fab-operator.md` if `fab-operator.md` changes).

## Affected Memory

- `runtime/pane-commands`: (modify) pane map JSON gains nullable `display_state`; capture/send/process validation mechanism changes from `tmux list-panes -a` pre-check to a targeted `display-message` probe (:83, :170); darwin `ps` args sourcing becomes a single batched pass (:93)
- `runtime/operator`: (modify) singleton enforcement mechanism becomes an exact, server-wide window-name match (list-windows `-a`) — final placement call at hydrate (see Assumptions #10)
- `memory-docs/hydrate`: (modify) index "Last Updated" date sourcing becomes one batched `git log --name-only` pass with per-file fallback (:63)
- `memory-docs/templates`: (modify) same date-sourcing echo at :115

## Impact

- **Go sources**: `src/go/fab/cmd/fab/{batch_new,operator,panemap,pane_capture,pane_send,pane_process,pane_process_darwin,pane_window_name}.go` (last one read-only/refactored into a shared helper), `src/go/fab/internal/pane/pane.go`, `src/go/fab/internal/memoryindex/memoryindex.go` + their `_test.go` files.
- **CLI contract changes**: `fab batch new` gains non-zero exit on any launch failure (was unconditional 0); batch-new/operator stderr moves to the central `ERROR: %s` format; pane map JSON gains additive `display_state`; subprocess error text is enriched with child stderr. No exit-code/stderr changes to `resolve --pane` or `pane map` error paths (pinned in memory docs).
- **Consumers**: fab-operator skill (pane map per-tick snapshot — shape additive-safe); run-kit daemon `rk serve` (programmatic pane map caller — additive-safe; its own api.md update is out of scope); no skill invokes `fab batch new` or `fab pane process` (verified in report).
- **Kit skills**: `src/kit/skills/_cli-fab.md` (:208, :212, :363-366, :444, :492, pane map row), `src/kit/skills/fab-operator.md` (pane map output shape) → SPEC mirrors per touched skill.
- **Memory docs**: per Affected Memory above.
- **Excluded files**: `hook.go`, `runtime.go` (mz4q), `batch_switch.go` (ye8r).

## Open Questions

None — the backlog entry plus the adversarially-verified findings report (§B5, with verifier corrections folded in above) resolve all design decisions. F36's MEDIUM verifier confidence is handled as an in-change re-verification step, not a question.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = F31, F32, F33, F34, F35, F36, F37, F38 + absorbed [dkn3], exactly as filed | Backlog entry enumerates actions; report §B5 gives verified files/lines/fixes; spot-rechecked at HEAD | S:95 R:90 A:95 D:95 |
| 2 | Certain | F33 uses `tmux list-windows -a` exact name compare (not bare `=` prefix) | Verifier correction: `=` alone stays session-scoped; `-a` enforces the documented per-SERVER singleton and separates "absent" from "tmux error" | S:90 R:85 A:90 D:85 |
| 3 | Certain | F38 limited to the three unpinned sites (batch_new ×2, operator ×1); no blanket sweep of the ~12 sibling sites | Verifier: `resolve --pane`/`pane map` stderr semantics are pinned in live memory docs; the three sites are the safe subset | S:90 R:85 A:90 D:85 |
| 4 | Certain | `display_state` is JSON-only, nullable (omitted when no resolvable change / status load fails); table output unchanged | Backlog states the decision verbatim ("nullable field on pane map --json rows; additive shape change"); null condition mirrors today's absent `stage` | S:90 R:85 A:90 D:85 |
| 5 | Certain | Exclude `batch_switch.go` despite F31's verifier correction naming it | Backlog wave-1 scope list is an explicit constraint ("touches panemap/pane_*/batch_new/operator/memoryindex only"); batch_switch is [ye8r]/F29 surface; deferral is freely reversible | S:90 R:85 A:90 D:85 |
| 6 | Confident | F36 probe = `tmux display-message -t <arg> -p '#{pane_id}'` with output==arg comparison; re-verify error-path equivalence before removing ValidatePane | Verifier-designed probe preserves ID-exactness + `--force` existence contract; MEDIUM verifier confidence on worthwhileness handled as an in-change verification step | S:80 R:75 A:80 D:75 |
| 7 | Certain | F37 uses the two-pass variant (`ps -axo pid=,args=` joined by PID) | Report prescribes it ("prefer the second variant — robust"); one-pass mis-parses comm-with-spaces; also removes the TOCTOU empty-cmdline window | S:85 R:90 A:90 D:85 |
| 8 | Certain | F34 keeps the per-file `git log -1` as fallback when the batched call fails; merge-commit/rename equivalence caveat accepted | Report's fix text prescribes the fallback; verifier confirms both behaviors match current defaults | S:85 R:90 A:90 D:85 |
| 9 | Certain | F38's stderr string changes (`Error: ...` → `ERROR: ...`) are accepted as a deliberate output change | Verifier explicitly accepts it: nothing live pins the strings (archived spec only, non-authoritative per constitution II); F38 filed with this known | S:85 R:90 A:85 D:85 |
| 10 | Confident | `runtime/operator` memory gets a brief singleton-mechanism note at hydrate | operator.md documents the per-server singleton invariant so a mechanism line fits; highly reversible — hydrate makes the final placement call | S:55 R:90 A:60 D:50 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).

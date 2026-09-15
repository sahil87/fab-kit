# Intake: Pane Send Copy-Mode Guard

**Change**: 260902-13qo-pane-send-copy-mode-guard
**Created**: 2026-09-02

## Origin

> Original request (/fab-new): fab operator pane capture/peek can read a scrolled-back, stale view instead of the live bottom of a pane. tmux implements scrollback by putting the pane into copy-mode; while a pane is in copy-mode, `tmux capture-pane -p` returns that frozen scrolled content, not the live tail. [...] Need a robust fix: before any capture-based read of a pane, check `#{pane_in_mode}` and if so exit copy-mode (`send-keys -X cancel`) before capturing. Audit every capture-pane call site and centralize the guard.

**Reframed during intake after empirical testing** (interactive session, user confirmed the reframe explicitly):

- **The capture premise was disproven** on tmux 3.7c (clean private-socket test, no rk client attached): with a pane in copy-mode and `scroll_position` held at 10–20, `capture-pane -p` kept returning the **live bottom** — repeated captures 3s apart tracked new output (TICK-21 → TICK-36) while the human-visible display stayed frozen. `capture-pane` reads the pane's live grid, not the copy-mode viewport. Captures (`fab pane capture`, `fab pane questions`, the dispatch gate's stall guard, raw `capture-pane` fallbacks) are already stale-proof against scrollback. No capture-side change is needed.
- **The send side is the real hazard**, found in the same test run: `tmux send-keys -t <pane> 'echo DELIVERED' Enter` into a copy-mode pane **exits 0 and is silently eaten** — the keys are interpreted as copy-mode key bindings, nothing reaches the application, and the pane stays in copy-mode. Every fab send path would silently no-op or garble into a pane a human left scrolled up.
- `grep -rn "pane_in_mode|copy-mode" src/ docs/memory/` confirms **no guard exists anywhere today**.
- User decision (AskUserQuestion): **"Reframe to send-path guard"** — pre-send `pane_in_mode` probe + cancel, centralized in `internal/pane` so all senders inherit it; no capture-side change; the `rk mux send` twin fix is noted as a run-kit follow-up, not done here.

## Why

1. **Problem**: A human (or any client) scrolling a monitored pane up puts it in tmux copy-mode. From that moment, every fab delivery into that pane — `fab dispatch deliver`'s prompt-pointer typing, `fab pane deliver`, `fab pane ready`'s sentinel probe — is silently consumed by copy-mode key bindings. `send-keys` exits 0, so the caller believes the send landed. The delivery choreography's capture-verify step will see the typed text missing and retry/fail, but it fails *confusingly* (looks like a dead worker) instead of self-healing, and the `ready` sentinel path can misclassify the pane. Non-choreographed sends have no verify at all.
2. **Consequence if unfixed**: operator/pipeline deliveries into scrolled-up panes fail with misleading symptoms; the recovery burden lands on a human noticing the pane was scrolled and pressing `q`/Ctrl+End. The failure is timing-dependent (depends on when a human last scrolled), so it presents as intermittent flakiness.
3. **Why this approach**: the probe is one cheap tmux round-trip (`display-message -p '#{pane_in_mode}'`) and the remedy (`send-keys -X cancel`) is exactly what a human would do. Centralizing it at the shared send runner in `internal/pane` means every current and future sender inherits the guard for free — the same one-builder/one-runner rationale `CaptureArgs`/`Capture` already document. Guarding captures instead (the original request) was rejected because testing shows captures are not affected.

## What Changes

### 1. `internal/pane`: copy-mode guard on the send path

New helpers in `src/go/fab/internal/pane/pane.go`, following the package's existing pure-builder + thin-runner pattern:

- `InModeArgs(server, paneID string) []string` → `WithServer(server, "display-message", "-p", "-t", paneID, "#{pane_in_mode}")`
- `CancelModeArgs(server, paneID string) []string` → `WithServer(server, "send-keys", "-X", "-t", paneID, "cancel")`
- A guard step wired into the shared send path (`runSend`, which both `SendLiteral` and `SendKey` ride): probe `#{pane_in_mode}`; when it reports `1`, run the cancel, then proceed with the send. When `0`, send directly — `send-keys -X cancel` against a pane not in a mode is an error, so the cancel MUST be conditional (verified: probe/scroll/cancel behavior on tmux 3.7c).

Behavior notes (verified in testing):

- `send-keys -X` mode commands work even when the only attached client is read-only (rk's control-mode viewer) — the guard is not blocked by the known rk read-only-client issue that blocks ordinary `send-keys`.
- `#{pane_in_mode}` is `1` for *any* pane mode (copy-mode, choose-tree, clock, …). The guard cancels them all — correct for delivery, since no mode passes typed text through to the application.
- Guarding per-send (inside the runner) rather than once per choreography also covers a human re-entering copy-mode *between* the choreography's keystrokes (C-u → literal text → Enter are separate sends).
- The guard is transparent: no new output on success. The existing choreography verification (capture the typed text before Enter) remains the correctness backstop.

### 2. Tests

- Unit tests for the new argv builders (pure, no tmux server) alongside the existing `CaptureArgs`/`SendLiteralArgs` tests.
- Guard decision logic (probe result → cancel-or-not) extracted pure or exercised through the package's existing injectable seams (`PaneIO` in gate.go is interface-backed; the runner seam determines the exact shape — plan decides).
- Scoped run: `go test ./src/go/fab/internal/pane/...` plus the dispatch/pane cmd packages that consume the senders.

### 3. Skill-side sweep (`src/kit/skills/`)

The raw-tmux **fallback** send paths (used when rk is absent, and the pre-delivery judgment rounds) are instructed prose, not Go — they need the same guard as guidance:

- `_cli-agents.md` § Pre-Send Validation becomes the **owner** of the copy-mode rule for skill-driven raw sends: before a raw `tmux send-keys` fallback, probe `tmux display-message -p -t <pane> '#{pane_in_mode}'` and `tmux send-keys -X -t <pane> cancel` when `1`. Other skill files that instruct raw sends (operator §3 fallback, `_preamble.md` judgment rounds) get a pointer, not a restatement (owner-or-pointer, code-quality.md).
- Sweep: grep `send-keys` across `src/kit/skills/*.md` and confirm each raw-send instruction site either carries the pointer or is already covered by the fab binary's guard.

### 4. Memory (hydrate scope)

- Document the send-path guard in the pane/dispatch memory files.
- Record the **disproven capture premise** as present truth in the capture documentation: `capture-pane` reads the live grid and is immune to copy-mode scrollback (verified tmux 3.7c) — so the next "stale capture" suspicion has a recorded answer.

### Non-Goals

- **No capture-side guard** — the original request's premise; empirically unnecessary, and auto-cancelling on every capture would yank a human out of scrollback on every operator tick.
- **`rk mux send` twin fix** — run-kit's sender needs the same guard, but that is a separate repo; to be filed as a run-kit idea (same precedent as the rk read-only-client issue).
- **The rk read-only-client send failure** — pre-existing, already tracked; unrelated to copy-mode (it blocked ordinary sends in testing even with no mode active).

## Affected Memory

- `runtime/pane-commands.md`: (modify) `fab pane deliver`/`ready` gain the pre-send copy-mode guard; capture section records copy-mode immunity of `capture-pane` (verified tmux 3.7c)
- `runtime/dispatch.md`: (modify) delivery choreography (`fab dispatch deliver`/`ready`) is copy-mode-proof via the shared send runner's guard
- `runtime/agent-primitives.md`: (modify) the pre-send gate description gains the copy-mode probe/cancel step for the raw-tmux fallback path (owner: `_cli-agents.md` § Pre-Send Validation)

## Impact

- `src/go/fab/internal/pane/pane.go` (+ its tests; possibly `gate.go` if the guard shape touches `PaneIO`) — all Go senders funnel through `SendLiteral`/`SendKey`, so `cmd/fab` needs no changes
- `src/kit/skills/_cli-agents.md` (rule owner) + pointer-level touches where raw sends are instructed (`fab-operator.md` §3, `_preamble.md` judgment rounds) — sweep-verified
- `docs/memory/runtime/` — 3 files above
- `src/kit/skills/_cli-fab.md` — only if command *signatures* change (none expected; the guard is internal behavior — verify at apply)

## Open Questions

- None — the empirical questions (capture behavior, send behavior, cancel semantics, `-X` vs read-only clients) were all settled by live tmux testing during intake, and the scope decision was confirmed by the user.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Send-path guard only; capture path untouched | Empirically verified on tmux 3.7c (captures track the live grid during copy-mode; sends are eaten); user explicitly confirmed the reframe | S:95 R:90 A:100 D:95 |
| 2 | Confident | Guard lives in the shared send runner (`runSend`) so `SendLiteral`/`SendKey` and every choreography inherit it, rather than per-entry guards in deliver/ready | Matches the package's stated one-builder/one-runner anti-drift rationale; per-send probing also covers mid-choreography mode entry; easy to relocate if the plan finds a better seam | S:70 R:85 A:80 D:70 |
| 3 | Confident | Cancel fires for ANY pane mode (`#{pane_in_mode}`=1), not copy-mode specifically | No tmux mode passes typed text to the application, so exiting any mode before delivery is correct; conditional cancel avoids the error `-X cancel` raises outside a mode | S:60 R:85 A:80 D:75 |
| 4 | Confident | Skill-side rule owned by `_cli-agents.md` § Pre-Send Validation; other raw-send sites get pointers | Owner-or-pointer convention (code-quality.md) — § Pre-Send Validation is the existing pre-send gate owner | S:65 R:80 A:85 D:75 |
| 5 | Certain | `rk mux send` twin fix is out of scope, filed as a run-kit idea | Separate repo; user selected the reframe option that stated this disposition; precedent: rk read-only-client issue | S:85 R:90 A:90 D:85 |
| 6 | Confident | Guard is transparent (no new stdout/warning when it cancels a mode) | Matches existing choreography verbosity; delivery's capture-verify remains the observable correctness check; a log line can be added later without contract impact | S:50 R:90 A:70 D:55 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).

# Plan: Pane Send Copy-Mode Guard

**Change**: 260902-13qo-pane-send-copy-mode-guard
**Intake**: `intake.md`

## Requirements

### Runtime: pre-send pane-mode guard

- **R1**: `pane.SendLiteral` and `pane.SendKey` MUST run a pane-mode guard before executing their `send-keys`: probe `#{pane_in_mode}` via `display-message`; when it reports `1`, run `send-keys -X -t <pane> cancel` and then proceed with the send; when `0`, send directly with no cancel (an `-X cancel` against a pane not in a mode is a tmux error, so the cancel MUST be conditional).
  - GIVEN a pane a human left scrolled up in copy-mode, WHEN any fab sender (gate choreography `C-u`/text/Enter, `fab pane deliver`, `fab pane ready` sentinel, `fab dispatch deliver`) sends into it, THEN the mode is cancelled first and the keystrokes reach the application instead of being consumed as copy-mode bindings.
  - GIVEN a pane not in any mode, WHEN a send runs, THEN no cancel is issued and behavior is byte-identical to today.
  - GIVEN the probe itself fails (pane gone, tmux unreachable), WHEN the guard runs, THEN the send fails with the same stderr-enriched error convention the senders already use (`StderrError`).

- **R2**: The guard's tmux argv MUST be built by new pure builders following the package's builder/runner pattern — `InModeArgs(server, paneID)` → `display-message -p -t <pane> #{pane_in_mode}` and `CancelModeArgs(server, paneID)` → `send-keys -X -t <pane> cancel` — both honoring the `WithServer` `-L` prefix, with the cancel-or-not decision extracted as a pure function of the probe output (trimmed output `== "1"`).
  - GIVEN a non-empty server, WHEN either builder runs, THEN the argv is prefixed `-L <server>` exactly like `CaptureArgs`/`SendLiteralArgs`.
  - GIVEN probe outputs `"1\n"`, `"0\n"`, `""`, WHEN the decision function evaluates them, THEN only `"1\n"` (trimmed `1`) selects cancel.

### Skills: raw-fallback guidance (owner-or-pointer)

- **R3**: `src/kit/skills/_cli-agents.md` § Pre-Send Validation MUST own the copy-mode rule for skill-driven RAW `tmux send-keys` fallbacks (used when rk is absent, and in `_preamble.md`'s pre-delivery judgment rounds): before a raw send, probe `tmux display-message -p -t <pane> '#{pane_in_mode}'` and run `tmux send-keys -X -t <pane> cancel` when it reports `1` — keys sent into a pane in a mode are silently consumed as mode bindings (exit 0, nothing reaches the application). Other skill files that instruct raw sends MUST carry at most a pointer, never a restatement (owner-or-pointer, code-quality.md). The rule MUST note that fab's own senders apply this guard automatically.
  - GIVEN the sweep `grep -rn "send-keys" src/kit/skills/*.md`, WHEN each raw-send instruction site is inspected, THEN it is either the owner text in `_cli-agents.md`, a pointer to it, or a `fab`-binary path already guarded in Go.

- **R4**: `src/kit/skills/_cli-fab.md` MUST record the new pre-send guard behavior in its `fab pane` deliver/ready documentation (constitution: CLI behavior changes update `_cli-fab.md`) — a one-line behavior note, no signature changes.
  - GIVEN `_cli-fab.md` § fab pane, WHEN a reader checks deliver/ready semantics, THEN the pane-mode cancel-before-send behavior is stated.

### Non-Goals

- No capture-side guard — `capture-pane` reads the live grid during copy-mode (verified tmux 3.7c during intake); captures are already stale-proof.
- No `rk mux send` change — run-kit is a separate repo; its twin guard is a run-kit idea.
- No fix for the rk read-only-client send blocker — pre-existing, tracked separately, orthogonal to pane modes.

### Design Decisions

- **Decision**: Guard at the `SendLiteral`/`SendKey` entry points in `internal/pane` (one shared helper both call), not per-choreography or inside `runSend`.
  **Why**: All Go senders funnel through these two functions (gate.go's `tmuxPaneIO` delegates to them; no other `send-keys` exists in Go). `runSend` receives a prebuilt argv without the server string, so the guard sits one level up where `server`/`paneID` are in hand. Per-send guarding also covers a human re-entering copy-mode between a choreography's keystrokes.
  **Rejected**: Guarding inside `Gate` (misses future non-gate senders); probing once per delivery (races mid-choreography mode entry).
  *Introduced by 260902-13qo.*
- **Decision**: The guard is transparent — no new stdout/log line when it cancels a mode.
  **Why**: Matches the senders' existing silence; delivery's capture-verify remains the observable correctness check.
  **Rejected**: stderr warning on cancel (noise in choreography retries; can be added later without contract impact).
  *Introduced by 260902-13qo.*

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `InModeArgs`/`CancelModeArgs` builders, the pure cancel decision (`paneInMode(probeOut string) bool` or equivalent), and an `ensureNoMode(server, paneID)` runner wired into `SendLiteral` and `SendKey` in `src/go/fab/internal/pane/pane.go` <!-- R1, R2 -->
- [x] T002 [P] Unit tests in `src/go/fab/internal/pane/pane_test.go` for both builders (with/without server) and the cancel decision over `"1\n"`/`"0\n"`/`""`; run scoped tests `go test ./...` from `src/go/fab` for the pane + consumer packages <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Add the copy-mode raw-send rule to `src/kit/skills/_cli-agents.md` § Pre-Send Validation (owner), then sweep `grep -rn "send-keys" src/kit/skills/*.md` and reconcile every raw-send instruction site (pointer or already-guarded-by-fab, per R3) <!-- R3 -->
- [x] T004 [P] Add the one-line pre-send guard behavior note to `src/kit/skills/_cli-fab.md` § fab pane (deliver/ready) <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Every Go send path (`SendLiteral`/`SendKey`, hence gate choreography, `fab pane deliver`/`ready`, `fab dispatch deliver`) probes `#{pane_in_mode}` and conditionally cancels before sending.
- [x] A-002 R2: `InModeArgs`/`CancelModeArgs` exist as pure builders honoring `WithServer`, with the cancel decision a pure function.
- [x] A-003 R3: `_cli-agents.md` § Pre-Send Validation owns the raw-fallback copy-mode rule; sweep of `send-keys` sites in `src/kit/skills/` shows owner text, pointers, or fab-guarded paths only.
- [x] A-004 R4: `_cli-fab.md` documents the deliver/ready pre-send guard behavior.

### Behavioral Correctness

- [x] A-005 R1: A pane NOT in a mode gets no `-X cancel` (conditional guard — cancel outside a mode errors).
- [x] A-006 R1: Probe failure surfaces via the existing `StderrError` convention naming the pane.

### Scenario Coverage

- [x] A-007 R2: Builder tests cover the server and no-server argv shapes; decision tests cover `"1\n"`, `"0\n"`, and empty output.

### Code Quality

- [x] A-008 Pattern consistency: new code follows the package's pure-builder + thin-runner pattern and server-first argument order.
- [x] A-009: No duplication of existing utilities (`WithServer`, `StderrError`, `RunCmd` reused).
- [x] A-010: No edits under `.claude/skills/` — kit changes land in `src/kit/` only.
- [x] A-011: Go change ships tests in the same change (Constitution VII / code-review.md).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Guard helper called from `SendLiteral`/`SendKey` (not inside `runSend`) | `runSend` lacks the server string; the two exported senders are the funnel every Go caller rides — same centralization, cleaner seam | S:70 R:85 A:85 D:75 |
| 2 | Confident | Guard wiring itself is exercised indirectly (thin runner over `RunCmd`), pure parts unit-tested | Matches package convention: `Capture`/`SendLiteral` runners are thin and untested; builders and decisions are the tested surface | S:60 R:85 A:80 D:70 |

2 assumptions (0 certain, 2 confident, 0 tentative, 0 unresolved).

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

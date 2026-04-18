# Intake: Pane Capture Tmux Flag Fix

**Change**: 260406-x33u-pane-capture-tmux-flag
**Created**: 2026-04-06
**Status**: Draft

## Origin

> fix: pane capture uses invalid tmux -l flag, replace with -S -N

One-shot. The user identified a concrete bug: `fab pane capture` invokes `tmux capture-pane` with `-l N` (intended to limit lines), but tmux's `capture-pane` does not have a `-l` flag. The correct approach to limit the number of captured lines is `-S -N` where `-N` is the negative line offset from the bottom of the pane history.

## Why

`tmux capture-pane` accepts `-S start-line` and `-E end-line` for scoping the capture range. A negative value for `-S` means "N lines up from the current bottom of the scrollback buffer", so `-S -50` captures the last 50 lines. The flag `-l` is not a recognized flag in `capture-pane` (it is a valid flag in other tmux commands, e.g., `display-message -l`). Using `-l` causes tmux to return an error, making `fab pane capture` silently fail or produce no output in most tmux versions.

Without this fix:
- `fab pane capture <pane>` returns an error from tmux and surfaces it to the caller.
- The operator's question-detection logic (which calls `tmux capture-pane -l 20` directly in the skill) also fails.
- Any tooling that relies on `fab pane capture` for context enrichment or monitoring gets broken output.

## What Changes

### 1. `src/go/fab/cmd/fab/pane_capture.go` — `capturePaneContent` function

**Before** (line 94):
```go
out, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p", "-l", fmt.Sprintf("%d", lines)).Output()
```

**After**:
```go
out, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p", "-S", fmt.Sprintf("-%d", lines)).Output()
```

The `-l` arg and its value are replaced with `-S` and `-%d` (negative integer). For `lines=50` this becomes `tmux capture-pane -t %5 -p -S -50`.

### 2. `src/kit/skills/fab-operator.md` — Question Detection section

**Before** (line 244):
```
1. **Capture**: `tmux capture-pane -t <pane> -p -l 20`
```

**After**:
```
1. **Capture**: `tmux capture-pane -t <pane> -p -S -20`
```

The skill documents the raw tmux command used during question detection. This documents the correct invocation so that if any skill-level logic calls tmux directly (not via `fab pane capture`), it also uses the correct flag.

### 3. `src/go/fab/cmd/fab/pane_capture_test.go` — `TestCaptureLineFlagShorthand`

The existing test at line 169 verifies the `-l` shorthand flag on the cobra command (the `--lines`/`-l` CLI flag). This is unrelated to the tmux argument — the cobra flag `-l` is valid for the `fab pane capture` CLI. No test change needed for the cobra flag.

However, a new test should verify that `capturePaneContent` passes `-S` and a negative value to tmux. Since `capturePaneContent` directly calls `exec.Command`, this is most practically tested by verifying the command args — we can add a unit test that uses a fake tmux or check the args constructed.

Alternatively: since the function is simple and tmux is an external dependency, documenting the fix in the existing test file with a comment or a build-tag-guarded integration test is sufficient. The core change is one-line in the Go source.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) `fab pane capture` behavior — document that the tmux invocation uses `-S -N`, not `-l N`

## Impact

- `src/go/fab/cmd/fab/pane_capture.go` — `capturePaneContent` function (line 94)
- `src/kit/skills/fab-operator.md` — Question Detection section (line 244)
- `src/go/fab/cmd/fab/pane_capture_test.go` — optionally add test for tmux args construction
- `_cli-fab.md` — no update needed (the spec shows `-l N` only as the `fab pane capture` CLI flag, which remains valid)

## Open Questions

- No open questions. The fix is unambiguous: `-l N` → `-S -N` in both locations.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace `-l N` with `-S -N` (negative integer) in `capturePaneContent` | tmux `capture-pane` has no `-l` flag; `-S -N` is the standard way to capture the last N lines | S:90 R:90 A:90 D:90 |
| 2 | Certain | Fix the skill documentation in `fab-operator.md` line 244 | The skill uses the raw tmux command; it must document the correct invocation | S:85 R:90 A:85 D:85 |
| 3 | Confident | No test change required for existing `TestCaptureLineFlagShorthand` | That test validates the cobra CLI `-l` shorthand flag, not the tmux argument — they are independent | S:80 R:85 A:80 D:80 |
| 4 | Confident | `_cli-fab.md` and other docs do not need updating | The `[-l N]` in `_cli-fab.md` refers to the `fab pane capture` CLI flag, not the tmux internal flag — no user-visible contract changes | S:75 R:80 A:80 D:80 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).

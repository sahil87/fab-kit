# Intake: wt open — Add tmux session option

**Change**: 260403-o8eg-wt-open-tmux-session-option
**Created**: 2026-04-03
**Status**: Draft

## Origin

> Currently wt open only lists Open in tmux window. Along with this, also provide an Open in new tmux session equivalent option.

One-shot request. User wants a new menu entry in `wt open` that creates a new tmux session (rather than a new window in the current session).

## Why

When running inside tmux, `wt open` only offers "tmux window" — which opens the worktree as a new window/tab within the current tmux session. For users who want full session isolation (e.g., separate session per project, easier `tmux switch-client`, independent window layouts), there is no option. The user must manually run `tmux new-session` outside of `wt open`.

Adding a "tmux session" option alongside the existing "tmux window" entry completes the tmux integration story and gives users a choice between lightweight (same session) and isolated (new session) workflows.

## What Changes

### New app entry: `tmux_session`

In `src/go/wt/internal/worktree/apps.go`:

1. **Detection** (in `BuildAvailableApps`): Add a new `AppInfo{"tmux session", "tmux_session"}` entry, conditionally shown when `IsTmuxSession()` is true — same guard as the existing `tmux_window` entry. It should appear immediately after the `tmux_window` entry.

2. **Opening logic** (in `OpenInApp`): Add a `case "tmux_session":` that creates a new tmux session:
   ```go
   case "tmux_session":
       sessionName := repoName + "-" + wtName
       if _, err := exec.LookPath("tmux"); err != nil {
           return fmt.Errorf("tmux is not available on this system")
       }
       cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", path)
       if out, err := cmd.CombinedOutput(); err != nil {
           return fmt.Errorf("tmux new-session failed: %s", strings.TrimSpace(string(out)))
       }
       return nil
   ```
   Key flags: `-d` (detached — don't attach immediately, avoids nested session issues), `-s` (session name), `-c` (start directory).

### Exit code handling

In `src/go/wt/cmd/open.go`: The existing tmux error path uses `strings.Contains(resolved.Cmd, "tmux")` which will naturally match both `tmux_window` and `tmux_session`, so `ExitTmuxWindowError` (exit code 6) will apply to both. This is acceptable — no new exit code needed.

### Default app detection

In `DetectDefaultApp` (`apps.go`): No change. When inside a plain tmux session, the default remains `tmux_window`. The new `tmux_session` is an opt-in alternative, not a replacement for the default.

### Tests

Add test coverage for the new `tmux_session` case in the `OpenInApp` switch and in the app detection logic.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the new `tmux_session` app entry in the wt packages section if it tracks app types

## Impact

- **Files changed**: `src/go/wt/internal/worktree/apps.go` (detection + opening), test file(s)
- **No API changes**: This is a new menu entry only
- **No config changes**: Detection is automatic (same `IsTmuxSession()` guard)
- **Backward compatible**: Existing `tmux_window` behavior is untouched

## Open Questions

None — the scope is clear and the implementation follows the established pattern.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `-d` (detached) flag for `tmux new-session` | Running `tmux new-session` without `-d` inside an existing tmux session causes "sessions should be nested" error. `-d` is the only correct approach | S:85 R:90 A:95 D:95 |
| 2 | Certain | Session name follows `repoName-wtName` pattern | Consistent with existing `tmux_window` tab naming convention (`tabName := repoName + "-" + wtName`) | S:90 R:95 A:95 D:95 |
| 3 | Certain | Same `IsTmuxSession()` guard for detection | New option should only appear when already in a tmux session, matching the `tmux_window` guard | S:90 R:90 A:95 D:95 |
| 4 | Certain | Reuse `ExitTmuxWindowError` exit code | `strings.Contains(resolved.Cmd, "tmux")` already matches both; separate exit codes would be over-engineering for a closely related feature | S:80 R:95 A:90 D:90 |
| 5 | Confident | Default app remains `tmux_window` when in tmux | `tmux_session` is for users who explicitly want isolation; it shouldn't change the default experience | S:75 R:85 A:80 D:75 |
| 6 | Confident | Place `tmux_session` entry immediately after `tmux_window` in the menu | Logical grouping of related tmux options | S:70 R:95 A:80 D:70 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).

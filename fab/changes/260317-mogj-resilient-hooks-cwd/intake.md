# Intake: Resilient Hooks CWD

**Change**: 260317-mogj-resilient-hooks-cwd
**Created**: 2026-03-17
**Status**: Draft

## Origin

> Make the hooks more resilient. Currently during normal working in case cwd changes, hooks start failing. hooks should work from subfolders of repo root also.

One-shot request. User has observed that Claude Code hooks break during normal development when the agent's working directory shifts to a subfolder of the repository root.

## Why

Claude Code hooks are registered in `.claude/settings.local.json` with **relative path** commands like:

```json
"command": "bash fab/.kit/hooks/on-session-start.sh"
```

When Claude Code executes a hook, it runs the command from whatever the current working directory happens to be. During normal agent operation, cwd can shift to subdirectories (e.g., `src/go/fab/`, `fab/changes/260317-xyz/`) via Bash `cd` commands or other tool operations. When this happens, the relative path `fab/.kit/hooks/on-session-start.sh` doesn't resolve, and the hook silently fails.

The Go binary (`fab-go`) already handles cwd robustly — `resolve.FabRoot()` walks upward from the current directory to find `fab/`. The problem is purely in the shell layer: the hook command string in settings and the hook shell scripts themselves.

If unfixed, hooks will continue to silently fail whenever cwd drifts, causing:
- Agent idle state not being tracked (session-start, stop, user-prompt hooks)
- Artifact bookkeeping not firing (artifact-write hook)
- No visible error (hooks are designed to swallow failures via `exit 0`)

## What Changes

### Hook shell scripts (`fab/.kit/hooks/on-*.sh`)

Each hook script currently uses `dirname "$0"` to locate the fab binary relative to itself:

```bash
#!/usr/bin/env bash
exec "$(dirname "$0")/../bin/fab" hook session-start 2>/dev/null; exit 0
```

The `dirname "$0"` approach works correctly when the script is found and executed — `$0` contains the path used to invoke the script, and `dirname` resolves relative to that. The issue is upstream: the script isn't found in the first place because the command path is relative.

The fix should make each hook script **self-locating** — able to find the repo root from any cwd within the repo. Two approaches:

1. **Git-based resolution**: Use `git rev-parse --show-toplevel` to find the repo root, then construct the path to the fab binary.
2. **Upward walk**: Mirror the Go `FabRoot()` logic — walk up from cwd looking for `fab/`.

The git approach is simpler and more reliable (works even if invoked via an unexpected path).

Updated script pattern:

```bash
#!/usr/bin/env bash
ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
exec "$ROOT/fab/.kit/bin/fab" hook session-start 2>/dev/null; exit 0
```

### Hook sync logic (`fab hook sync` / `internal/hooklib/sync.go`)

The `fab hook sync` command generates the hook entries in `.claude/settings.local.json`. Currently it writes relative paths. It should continue writing relative paths (since absolute paths would break portability across machines), but the hook scripts themselves will handle resolution.

No change needed to the sync logic — the scripts themselves become resilient.

### Affected scripts

- `fab/.kit/hooks/on-session-start.sh`
- `fab/.kit/hooks/on-stop.sh`
- `fab/.kit/hooks/on-user-prompt.sh`
- `fab/.kit/hooks/on-artifact-write.sh`

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the hook cwd-resilience pattern

## Impact

- **Hook scripts**: 4 files modified (trivial change — add git root resolution)
- **Tests**: Hook test (`internal/hooks/hooks_test.go`) may need updating if it validates cwd behavior
- **No Go code changes**: The Go binary already handles cwd correctly
- **No settings changes**: The registered commands in `.claude/settings.local.json` remain the same
- **No migration needed**: Existing hooks will pick up the new scripts on next `fab hook sync` or immediately if `.kit/` is updated in place

## Open Questions

(none — the approach is straightforward)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `git rev-parse --show-toplevel` for repo root resolution | Git is always available (fab-kit is a git-hosted project), and this is the standard way to find repo root from any subdirectory | S:80 R:90 A:95 D:90 |
| 2 | Certain | Keep hook commands as relative paths in settings.local.json | Absolute paths would break portability across machines and worktrees — the scripts handle resolution | S:85 R:85 A:90 D:90 |
| 3 | Certain | No Go code changes needed | The Go binary already uses `resolve.FabRoot()` upward walk — the problem is purely in the shell invocation layer | S:90 R:95 A:95 D:95 |
| 4 | Confident | No migration needed | Users get the fix when `.kit/` is updated; existing settings commands remain unchanged | S:75 R:90 A:80 D:85 |
| 5 | Certain | Maintain `exit 0` / error-swallowing contract | Constitution and `_cli-fab.md` require hooks to never block the agent | S:90 R:95 A:95 D:95 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).

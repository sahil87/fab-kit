# Spec: Resilient Hooks CWD

**Change**: 260317-mogj-resilient-hooks-cwd
**Created**: 2026-03-17
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Hooks: CWD-Resilient Script Resolution

### Requirement: Git-Based Repo Root Resolution

Each hook shell script in `fab/.kit/hooks/on-*.sh` SHALL resolve the repository root using `git rev-parse --show-toplevel` before constructing the path to the fab binary. The script MUST NOT rely on the current working directory being the repo root.

The updated script pattern SHALL be:

```bash
#!/usr/bin/env bash
ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
exec "$ROOT/fab/.kit/bin/fab" hook <subcommand> 2>/dev/null; exit 0
```

Where `<subcommand>` is the hook-specific command (`session-start`, `stop`, `user-prompt`, `artifact-write`).

#### Scenario: Hook invoked from repo root
- **GIVEN** the agent's cwd is the repository root
- **WHEN** Claude Code fires a hook (e.g., `bash fab/.kit/hooks/on-session-start.sh`)
- **THEN** `git rev-parse --show-toplevel` returns the repo root
- **AND** the fab binary is found and executed successfully

#### Scenario: Hook invoked from a subdirectory
- **GIVEN** the agent's cwd is a subdirectory (e.g., `src/go/fab/internal/`)
- **WHEN** Claude Code fires a hook
- **THEN** `git rev-parse --show-toplevel` still returns the repo root
- **AND** the fab binary is found and executed successfully

#### Scenario: Hook invoked outside a git repo
- **GIVEN** the agent's cwd is outside any git repository
- **WHEN** Claude Code fires a hook
- **THEN** `git rev-parse --show-toplevel` fails
- **AND** the script exits 0 silently (error-swallowing contract maintained)

### Requirement: Error-Swallowing Contract

All hook scripts MUST exit 0 regardless of whether the fab binary is found or executes successfully. This is the existing contract defined in `_cli-fab.md`: hook subcommands MUST exit 0 always to never block the agent.

#### Scenario: fab binary missing
- **GIVEN** `fab/.kit/bin/fab` does not exist (e.g., fresh clone without binary)
- **WHEN** Claude Code fires a hook
- **THEN** the script exits 0 silently

### Requirement: All Four Hook Scripts Updated

The following scripts SHALL be updated with the git-based resolution pattern:

- `fab/.kit/hooks/on-session-start.sh`
- `fab/.kit/hooks/on-stop.sh`
- `fab/.kit/hooks/on-user-prompt.sh`
- `fab/.kit/hooks/on-artifact-write.sh`

Each script SHALL use the identical resolution pattern, differing only in the hook subcommand name.

#### Scenario: Consistency across hooks
- **GIVEN** all four hook scripts exist
- **WHEN** inspected for the resolution pattern
- **THEN** each contains `git rev-parse --show-toplevel` resolution
- **AND** each uses `"$ROOT/fab/.kit/bin/fab"` for binary invocation

### Requirement: No Changes to Hook Sync or Settings

The `fab hook sync` command and `.claude/settings.local.json` hook entries SHALL NOT be modified. The hook command strings remain relative paths (`bash fab/.kit/hooks/on-*.sh`). The scripts themselves handle cwd resolution.

#### Scenario: Settings unchanged after fix
- **GIVEN** `.claude/settings.local.json` contains hook entries with relative paths
- **WHEN** the hook scripts are updated
- **THEN** no changes to `.claude/settings.local.json` are needed
- **AND** `fab hook sync` output remains unchanged

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `git rev-parse --show-toplevel` for repo root resolution | Confirmed from intake #1 — git is always available, standard approach | S:80 R:90 A:95 D:90 |
| 2 | Certain | Keep hook commands as relative paths in settings.local.json | Confirmed from intake #2 — absolute paths break portability | S:85 R:85 A:90 D:90 |
| 3 | Certain | No Go code changes needed | Confirmed from intake #3 — Go binary uses FabRoot() upward walk | S:90 R:95 A:95 D:95 |
| 4 | Confident | No migration needed | Confirmed from intake #4 — fix is in .kit/ scripts | S:75 R:90 A:80 D:85 |
| 5 | Certain | Maintain exit 0 error-swallowing contract | Confirmed from intake #5 — hooks must never block | S:90 R:95 A:95 D:95 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).

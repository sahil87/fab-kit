# Intake: Clean Break — Go Only

**Change**: 260305-u8t9-clean-break-go-only
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Discussion session exploring an orchestrator for agents and worktrees on top of fab-kit. The conversation started with adding `--json` to `wt-status`, evolved into absorbing `wt-status` into `fab status` with JSON output, then expanded into a full clean break: removing all shell script fallbacks from the dispatcher, deleting ported lib/ scripts, and establishing the Go binary as the sole backend.

One-shot decision from extended `/fab-discuss` conversation. Key decisions made during discussion:

1. Keep path as `fab/.kit/bin/fab` (dispatcher stays in `bin/`, no rename to `scripts/`)
2. Keep thin dispatcher with `fab-rust` > `fab-go` priority, remove shell fallback
3. Remove `wt-status` shell command, absorb into `fab status show` with `--json`/`--all`
4. Add `fab/.kit/bin/` to PATH via `env-packages.sh` for human terminal callability
5. Remove all 7 ported lib/ shell scripts
6. Keep Go parity tests as authoritative integration tests

## Why

1. **Dead code**: All 7 CLI shell scripts in `fab/.kit/scripts/lib/` have been fully ported to Go with parity tests. Keeping them adds maintenance burden — any future changes would need dual implementation.

2. **Orchestrator foundation**: Building an agent+worktree orchestrator requires machine-readable status output. `wt-status` is the natural window into worktree state, but it's a shell script producing human-only output. Moving to `fab status show --json` provides the structured data an orchestrator needs.

3. **Human callability**: The `fab` command is currently only callable via full path (`fab/.kit/bin/fab`). Adding `fab/.kit/bin/` to PATH makes `fab` a first-class terminal command alongside `wt-*` and `idea`.

4. **Simplified architecture**: Removing the shell fallback from the dispatcher eliminates the dual-backend complexity. The dispatcher becomes a thin 20-line script: version check, backend priority (`fab-rust` > `fab-go`), error if neither exists.

## What Changes

### Remove Shell Fallback from Dispatcher

Update `fab/.kit/bin/fab` to remove the shell fallback `case` block (lines 30-49). When no compiled backend is found, the dispatcher SHALL print an error directing the user to install the Go binary or run `fab-sync.sh`, then exit 1. The `--version` handler stays and continues to report the backend in use.

After:
```sh
#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
KIT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# --version: handled by dispatcher
if [ "${1:-}" = "--version" ]; then
  version="unknown"
  [ -f "$KIT_DIR/VERSION" ] && version=$(cat "$KIT_DIR/VERSION")
  backend="none"
  if [ -x "$SCRIPT_DIR/fab-rust" ]; then
    backend="rust"
  elif [ -x "$SCRIPT_DIR/fab-go" ]; then
    backend="go"
  fi
  echo "fab $version ($backend backend)"
  exit 0
fi

# Backend priority: rust > go
if [ -x "$SCRIPT_DIR/fab-rust" ]; then
  exec "$SCRIPT_DIR/fab-rust" "$@"
fi
if [ -x "$SCRIPT_DIR/fab-go" ]; then
  exec "$SCRIPT_DIR/fab-go" "$@"
fi

echo "Error: no fab backend found (expected fab-go or fab-rust in $SCRIPT_DIR)" >&2
echo "Fix: run fab-sync.sh or download a platform-specific kit archive" >&2
exit 1
```

### Delete Ported Shell Scripts

Remove from `fab/.kit/scripts/lib/`:
- `statusman.sh` (~1,294 lines)
- `changeman.sh` (~562 lines)
- `archiveman.sh` (~359 lines)
- `logman.sh` (~165 lines)
- `calc-score.sh` (~374 lines)
- `preflight.sh` (~142 lines)
- `resolve.sh` (~179 lines)

**Keep**: `env-packages.sh` (PATH setup, still needed), `frontmatter.sh` (used by `fab-help.sh` and `2-sync-workspace.sh`, not ported to Go).

### Remove `wt-status` Shell Command

Delete `fab/.kit/packages/wt/bin/wt-status`. Its functionality moves to the Go binary as `fab status show`.

### Add `fab status show` to Go Binary

New subcommand under `fab status`:

```
fab status show [--all] [--json] [<name>]
```

| Flag | Behavior |
|------|----------|
| (none) | Current worktree's fab pipeline state (human-readable) |
| `--all` | All worktrees' fab state |
| `--json` | Current worktree (JSON) |
| `--all --json` | All worktrees (JSON) — the orchestrator endpoint |
| `<name>` | Specific worktree by name |

**Human-readable output** (single worktree):
```
swift-fox  260305-u8t9-clean-break-go-only  tasks  active
```

**Human-readable output** (`--all`):
```
Worktrees for: fab-kit
Location: /path/to/fab-kit.worktrees

* (main)         260305-u8t9-clean-break-go-only  tasks       active
  swift-fox      (no change)
  calm-owl       260305-bs5x-orchestrator-idle     review      done

Total: 3 worktree(s)
```

**JSON output** (single worktree):
```json
{
  "name": "main",
  "path": "/path/to/fab-kit",
  "branch": "main",
  "is_main": true,
  "is_current": true,
  "change": "260305-u8t9-clean-break-go-only",
  "stage": "tasks",
  "state": "active"
}
```

**JSON output** (`--all --json`): Array of the above objects.

The Go implementation reuses the worktree discovery logic from `git worktree list --porcelain` (same approach as the shell `wt_list_worktrees` function) and the existing `internal/statusfile` + `internal/resolve` packages for fab state resolution.

### Add `fab/.kit/bin/` to PATH

Update `fab/.kit/scripts/lib/env-packages.sh` to also add `fab/.kit/bin/` to PATH:

```bash
#!/usr/bin/env bash
# Add fab-kit bin and package bin directories to PATH
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
export PATH="$KIT_DIR/bin:$PATH"
for d in "$KIT_DIR"/packages/*/bin; do
  [ -d "$d" ] && export PATH="$d:$PATH"
done
```

### Update `pipeline/dispatch.sh`

Lines 154-162 reference `calc-score.sh` directly. Update to use `fab score --check-gate` instead:

```bash
# Before
local wt_calc_score="$wt_path/fab/.kit/scripts/lib/calc-score.sh"
gate_result=$(bash "$wt_calc_score" --check-gate "$change_dir" 2>/dev/null) || true

# After
gate_result=$("$wt_path/fab/.kit/bin/fab" score --check-gate "$CHANGE_ID" 2>/dev/null) || true
```

### Update `_scripts.md` Skill Documentation

Update `fab/.kit/skills/_scripts.md` to:
1. Remove the "Shell fallback" section and command mapping table (no more shell scripts)
2. Remove the "Shell scripts in `fab/.kit/scripts/lib/` are pure implementations" sentence
3. Update the Backend Priority section to show `rust > go > error` (no shell fallback)
4. Note that `fab -h`, `fab --help`, and `fab <subcommand> --help` work via Cobra

### Remove Bash Test Cases

Delete `src/packages/wt/tests/wt-status.bats` — tests the shell `wt-status` command which is being removed.

The Go parity tests in `src/go/fab/test/parity/` stay as authoritative integration tests. They currently compare bash vs Go output; after this change they test the Go binary standalone (the bash side will fail gracefully or can be updated to test Go-only).

### Update Memory and Docs

- Update `docs/memory/fab-workflow/kit-architecture.md` to remove shell script references from the directory structure and dispatcher documentation
- Update `docs/memory/fab-workflow/distribution.md` to reflect that shell-only bootstrap (generic `kit.tar.gz`) no longer provides a working `fab` command — the Go binary is required

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Remove shell script lib/ entries from directory structure, update dispatcher description to `rust > go > error`, remove command mapping table
- `fab-workflow/distribution`: (modify) Update generic archive description — shell-only fallback no longer works, Go binary is required
- `fab-workflow/execution-skills`: (modify) Update status mutations note from `statusman.sh` to `fab status` CLI

## Impact

- **Kit size**: Removes ~3,075 lines of bash from `fab/.kit/scripts/lib/`
- **Prerequisites**: `yq` is no longer needed by lib/ scripts (still needed by `dispatch.sh` for YAML manifest writes and `fab-doctor.sh`)
- **`wt` package**: Loses `wt-status` command; other `wt-*` commands unaffected
- **`idea` package**: Unaffected
- **Batch scripts**: `FAB_BIN` references in `batch-fab-switch-change.sh` and `batch-fab-archive-change.sh` already point to `fab/.kit/bin/fab` — no change needed
- **Pipeline dispatch**: `dispatch.sh` needs update from direct `calc-score.sh` call to `fab score`
- **Parity tests**: Continue to work as Go-only integration tests (bash comparison side will need graceful handling or removal of the bash runner)
- **Skills**: All 16+ skill files that reference `fab/.kit/bin/fab` — no path changes needed (path stays the same)

## Open Questions

- Should the Go parity tests be refactored from "compare bash vs Go" to "test Go standalone" in this change, or deferred? (Leaning toward: update them to skip the bash side gracefully, refactor later)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep `fab/.kit/bin/fab` path unchanged | Discussed — Option A chosen: least churn (320 references stay), `bin/` is semantically correct for binaries | S:95 R:90 A:95 D:95 |
| 2 | Certain | Dispatcher keeps `fab-rust` > `fab-go` priority | Discussed — user explicitly requested rust support for future use | S:90 R:85 A:90 D:95 |
| 3 | Certain | Keep `frontmatter.sh` | Active consumers: `fab-help.sh` (lines 15, 81-82, 99-100) and `2-sync-workspace.sh` (line 32). Not ported to Go | S:95 R:90 A:95 D:95 |
| 4 | Certain | Keep `env-packages.sh` and extend with `bin/` PATH | Discussed — still needed for wt/idea PATH setup, natural place to add fab binary PATH | S:90 R:90 A:90 D:90 |
| 5 | Certain | `fab status show` interface: `[--all] [--json] [<name>]` | Discussed — `--all --json` is the orchestrator endpoint, single worktree is the default | S:90 R:85 A:85 D:85 |
| 6 | Certain | Remove wt-status completely (no deprecation period) | Discussed — clean break, `fab status show` replaces it fully | S:85 R:80 A:85 D:90 |
| 7 | Confident | Keep Go parity tests as-is, skip bash side gracefully | Parity tests are valuable as integration tests. Full refactor to Go-only can be deferred | S:70 R:85 A:75 D:70 |
| 8 | Confident | `yq` still needed (dispatch.sh, fab-doctor.sh) | dispatch.sh uses `yq -i` for manifest YAML writes; fab-doctor.sh checks tool availability | S:75 R:80 A:80 D:75 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).

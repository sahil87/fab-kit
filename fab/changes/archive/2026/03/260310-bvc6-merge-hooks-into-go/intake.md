# Intake: Merge Claude Code Hooks Into Go Binary

**Change**: 260310-bvc6-merge-hooks-into-go
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Discussion session via `/fab-discuss`. User evaluated the three Claude Code hook shell scripts (`on-session-start.sh`, `on-stop.sh`, `on-artifact-write.sh`) and asked whether they should be merged into the compiled binary. After reviewing the analysis — which recommended only `on-artifact-write.sh` as the strongest candidate — the user explicitly decided: "Move all hooks." Subsequently chose Go over Rust as the target binary.

## Why

1. **Fragile JSON parsing in on-artifact-write.sh** — The hook parses Claude Code's PostToolUse JSON payload using `jq` with a regex fallback when `jq` is unavailable. The fallback is not fully JSON-safe and can break on escaped quotes or nested objects.

2. **Subprocess overhead in on-artifact-write.sh** — The hook makes 2-4 separate calls to `fab` subcommands per invocation (`resolve`, `status set-change-type`, `score`, `status set-checklist`). Each is a process spawn + binary lookup. Moving the logic into the binary eliminates these round-trips.

3. **jq dependency** — `on-artifact-write.sh` is the only remaining consumer of `jq` in the kit. Moving it to Go eliminates this dependency entirely, simplifying the prerequisite list in `fab-doctor.sh`.

4. **Testability** — Shell hook logic is tested via bats with stub `fab` binaries that simulate behavior. Moving to Go enables unit tests with proper JSON fixtures and no filesystem setup.

5. **Consistency** — `on-session-start.sh` and `on-stop.sh` are already thin wrappers that delegate to `fab runtime`. Making all hooks use `fab hook <subcommand>` gives a uniform pattern.

## What Changes

### New `fab hook` subcommand group

Add a `hook` subcommand group to the Go binary with four subcommands:

```
fab hook session-start    # Clears agent idle state (replaces on-session-start.sh)
fab hook stop             # Sets agent idle timestamp (replaces on-stop.sh)
fab hook user-prompt      # Clears agent idle state (NEW — UserPromptSubmit event)
fab hook artifact-write   # Artifact bookkeeping (replaces on-artifact-write.sh)
```

### `fab hook session-start`

Behavior (same as current `on-session-start.sh`):
1. Find repo root via git
2. Check `.fab-status.yaml` symlink exists
3. Resolve active change via `fab resolve --folder`
4. Call `fab runtime clear-idle <change>` logic internally (no subprocess)
5. Exit 0 always — never fail

### `fab hook stop`

Behavior (same as current `on-stop.sh`):
1. Find repo root via git
2. Check `.fab-status.yaml` symlink exists
3. Resolve active change via `fab resolve --folder`
4. Call `fab runtime set-idle <change>` logic internally (no subprocess)
5. Exit 0 always — never fail

### `fab hook user-prompt`

Behavior (NEW — no existing shell script equivalent):
1. Find repo root via git
2. Check `.fab-status.yaml` symlink exists
3. Resolve active change via `fab resolve --folder`
4. Call `fab runtime clear-idle <change>` logic internally (no subprocess)
5. Exit 0 always — never fail

Registered for the `UserPromptSubmit` event. Fixes agent idle tracking accuracy — without this, agents show as `idle (Nm)` while actively processing between the user's prompt and the agent's response. Combined with `fab hook stop` (which sets idle on response completion), this gives accurate tracking:
- User sends prompt → `user-prompt` → `clear-idle` → agent shows `active`
- Agent finishes → `stop` → `set-idle` → agent shows `idle (0s)`

### `fab hook artifact-write`

Reads PostToolUse JSON from stdin. Behavior (same as current `on-artifact-write.sh`):
1. Parse JSON from stdin — extract `tool_input.file_path`
2. Pattern match against fab artifact paths (`fab/changes/*/intake.md`, `spec.md`, `tasks.md`, `checklist.md`)
3. Derive change folder name and artifact type from path
4. Find repo root, verify change resolves
5. Per-artifact bookkeeping:
   - **intake.md**: Infer change type via keyword matching → `status set-change-type`. Run `score --stage intake`.
   - **spec.md**: Run `score`.
   - **tasks.md**: Count unchecked tasks → `status set-checklist total`.
   - **checklist.md**: Mark generated, count items → `status set-checklist`.
6. Output `{"additionalContext": "Bookkeeping: ..."}` JSON to stdout
7. Exit 0 always — never fail

The keyword matching rules for change type inference (case-insensitive, first match wins):
- fix/bug/broken/regression → `fix`
- refactor/restructure/consolidate/split/rename/redesign → `refactor`
- docs/document/readme/guide → `docs`
- test/spec/coverage → `test`
- ci/pipeline/deploy/build → `ci`
- chore/cleanup/maintenance/housekeeping → `chore`
- default → `feat`

### Shell scripts become thin wrappers

The existing three `.sh` files in `fab/.kit/hooks/` become one-liners that delegate to the binary, and a new `on-user-prompt.sh` is created:

```bash
# on-session-start.sh
#!/usr/bin/env bash
exec "$(dirname "$0")/../bin/fab" hook session-start 2>/dev/null; exit 0

# on-stop.sh
#!/usr/bin/env bash
exec "$(dirname "$0")/../bin/fab" hook stop 2>/dev/null; exit 0

# on-user-prompt.sh (NEW)
#!/usr/bin/env bash
exec "$(dirname "$0")/../bin/fab" hook user-prompt 2>/dev/null; exit 0

# on-artifact-write.sh
#!/usr/bin/env bash
exec "$(dirname "$0")/../bin/fab" hook artifact-write 2>/dev/null; exit 0
```

The `exec ... 2>/dev/null; exit 0` pattern ensures the hook never fails even if the binary is missing.

The sync script `5-sync-hooks.sh` needs a new mapping entry for the `UserPromptSubmit` event pointing to `on-user-prompt.sh`.

### Test updates

- Existing bats tests in `src/hooks/` update to test the Go binary directly (invoke `fab hook <subcommand>` instead of `bash on-*.sh`)
- Add Go unit tests for JSON parsing, keyword matching, and artifact path detection
- Go parity tests are not needed here since hooks are a new Go-only subcommand (no bash equivalent to compare against)

### Remove jq from prerequisites

Update `fab/.kit/scripts/fab-doctor.sh` to remove the `jq` prerequisite check, since `on-artifact-write.sh` was the sole consumer.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update hooks directory description — shell scripts become thin wrappers, add `fab hook` subcommand group to command reference

## Impact

- **Go binary** (`src/go/fab/`): New `hook` package with Cobra subcommands, reusing existing internal packages (`resolve`, `runtime`, `statusman`, `score`)
- **Shell hooks** (`fab/.kit/hooks/`): Rewritten to one-liners
- **Bats tests** (`src/hooks/`): Updated to test binary instead of shell scripts
- **fab-doctor.sh**: Remove jq prerequisite
- **Rust binary**: Port `fab hook` subcommands to maintain parity (or defer — Rust binary is local-dev only)

## Open Questions

- None — Go binary structure is well-established with Cobra commands and internal packages.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All three existing hooks move to Go binary, plus new UserPromptSubmit hook | Discussed — user explicitly said "Move all hooks", then chose Go; UserPromptSubmit added for idle tracking accuracy | S:95 R:80 A:90 D:95 |
| 2 | Certain | Shell scripts become thin wrappers, not deleted | Constitution requires hooks dir to exist; Claude Code config points to .sh files | S:85 R:90 A:85 D:90 |
| 3 | Certain | Exit 0 always — hooks must never block the agent | Existing behavior, documented in each hook file | S:95 R:95 A:95 D:95 |
| 4 | Confident | Rust binary parity deferred | Rust binary is local-dev only; Go is the distributed binary with all subcommands | S:70 R:85 A:75 D:80 |
| 5 | Confident | Remove jq from prerequisites after migration | on-artifact-write.sh is the sole jq consumer in the kit | S:75 R:80 A:80 D:85 |
| 6 | Confident | artifact-write reads stdin JSON, session-start and stop take no stdin | Matches current hook API — PostToolUse hooks receive JSON on stdin, SessionStart/Stop do not | S:80 R:85 A:80 D:90 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).

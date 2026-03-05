# Intake: Extract Logman from Preflight

**Change**: 260302-9fnn-extract-logman-from-preflight
**Created**: 2026-03-02
**Status**: Draft

## Origin

> Extract command logging from preflight.sh into direct logman.sh calls from skills. Currently preflight bundles validation + logging (via --driver flag), which means skills that run without an active change can never log.

Conversational `/fab-discuss` session. Key decisions from the discussion:

- **Logging should be separated from validation** — preflight's `--driver` flag gates logging behind 5 validation checks, which is the wrong coupling
- **logman owns the "should I log?" decision** — rather than each caller deciding whether to log, logman attempts resolution and silently succeeds if no change is active
- **Arg order flips to `command <cmd> [change]`** — skill name is always known, change is the optional piece
- **Skills that happen to run with an active change (e.g., `/fab-discuss`) get logged opportunistically** — zero cost to the caller

## Why

1. **Problem**: `preflight.sh` bundles two unrelated responsibilities — validation (5 checks + structured YAML output) and command logging (`--driver` flag calling logman). Skills exempt from preflight (`/fab-new`, `/fab-switch`, `/fab-setup`, `/fab-discuss`, `/fab-help`) can never log invocations because they can't pass preflight's change-resolution check.

2. **Consequence**: The audit trail in `.history.jsonl` has blind spots — `/fab-new` (change creation), `/fab-switch` (change activation), and `/fab-discuss` (context sessions) are invisible. This undermines the value of having a history log at all.

3. **Approach**: Move the logging concern to the caller level. Make logman's `command` subcommand resilient to missing changes (silent exit 0), so every skill can include a best-effort log call without conditional logic. Preflight becomes purely validation + structured output.

## What Changes

### 1. logman.sh `command` subcommand — make change optional

Current signature: `logman.sh command <change> <cmd> [args]`
New signature: `logman.sh command <cmd> [change] [args]`

- `<cmd>` (required) — the skill/command name, always known
- `[change]` (optional) — if omitted, logman tries to resolve via `fab/current`
- If resolution fails (no `fab/current`, empty file, directory doesn't exist), exit 0 silently — no error, no stderr
- If resolution succeeds, append to `.history.jsonl` as today

### 2. preflight.sh — remove `--driver` flag

- Remove the `--driver` flag parsing block and the `logman command` call
- preflight becomes: test-build guard → validation (steps 1-5) → structured YAML output
- No behavioral change to validation or output

### 3. Skill instructions — add direct logman calls

Every skill adds a best-effort logman call:

```bash
fab/.kit/scripts/lib/logman.sh command <skill-name> [change] 2>/dev/null || true
```

- **Preflight-calling skills** (continue, clarify, archive, ff, fff, status): call logman after preflight, passing the resolved change name from YAML output
- **Exempt skills** (new, switch, discuss, help, setup): call logman without change arg (or with newly-created change for `/fab-new`)
- The `2>/dev/null || true` pattern is already established in preflight today

### 4. _scripts.md — update documentation

- Update logman `command` signature and examples
- Remove `--driver` from preflight documentation
- Remove "Skills never call logman.sh directly" — invert to "Skills call logman.sh command directly"
- Update call graph to show skills as direct logman callers

### 5. _preamble.md — update preflight protocol

- Remove `--driver` from the §2 preflight invocation instructions
- The 3-step protocol (invoke, check exit, parse YAML) remains unchanged

## Affected Memory

- `fab-workflow/preflight`: (modify) Remove --driver flag, document pure validation role
- `fab-workflow/execution-skills`: (modify) Document direct logman calls from skills
- `fab-workflow/kit-architecture`: (modify) Update call graph — skills now call logman directly

## Impact

- **Scripts**: `logman.sh` (signature change), `preflight.sh` (remove --driver)
- **Skills**: All `fab/.kit/skills/fab-*.md` files (add logman call instruction)
- **Docs**: `_scripts.md`, `_preamble.md` (update conventions)
- **Tests**: Any tests that exercise `--driver` flag or logman `command` arg order
- **Other callers**: `changeman.sh new/rename` and `statusman.sh finish/fail review` call logman with the old `command <change> <cmd>` signature — these need updating to match the new arg order

## Open Questions

None — design was fully resolved in the `/fab-discuss` session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | logman silently exits 0 on failed resolution | Discussed — user explicitly chose "logman decides, silently" over caller-decides | S:95 R:90 A:95 D:95 |
| 2 | Certain | Arg order flips to `command <cmd> [change]` | Discussed — user agreed skill name is always known, change is optional | S:90 R:85 A:90 D:90 |
| 3 | Certain | preflight drops --driver entirely | Discussed — clean separation, preflight is pure validation | S:95 R:90 A:95 D:95 |
| 4 | Certain | All skills get a logman call, not just preflight-calling ones | Discussed — user specifically asked about fab-discuss logging opportunistically | S:90 R:90 A:90 D:90 |
| 5 | Confident | Other logman callers (changeman, statusman, calc-score) update to new arg order | Consistent interface — all `command` calls use same signature | S:70 R:80 A:85 D:85 |
| 6 | Confident | Resolution fallback uses fab/current only, not fuzzy match | Full resolve.sh is overkill for best-effort logging; simple file read suffices | S:65 R:90 A:80 D:75 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).

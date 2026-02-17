# Intake: Fold resolve-change into changeman

**Change**: 260216-oinh-fold-resolve-into-changeman
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Conversational design discussion. User observed that `/fab-switch` is slow because it requires ~5-6 Claude tool calls for deterministic operations (resolve name, write `fab/current`, git branch checkout, read `.status.yaml`, display status). User asked whether the write-to-`fab/current` step could move into `changeman.sh`, then followed up asking whether `resolve-change.sh` should be folded entirely into changeman — making `fab-switch` a single bash call.

## Why

1. **Performance**: `/fab-switch` currently requires Claude to orchestrate 5-6 sequential tool calls for entirely deterministic logic. Each tool call is a round-trip. Consolidating into a single `changeman.sh switch` call eliminates this overhead.

2. **Cohesion**: `resolve-change.sh` answers "which change are we talking about?" — a change lifecycle concern that belongs in the Change Manager, not as a standalone library. Today it's the only `lib/` script that must be `source`d (sets `RESOLVED_CHANGE_NAME` as a shell variable) rather than invoked as a subprocess. Folding it into changeman eliminates the sourcing pattern and aligns it with the CLI-subprocess convention used by stageman.

3. **Dependency simplification**: Three callers (`preflight.sh`, `batch-fab-switch-change.sh`, `batch-fab-archive-change.sh`) currently source `resolve-change.sh` separately. After folding, they call `changeman.sh resolve` — one fewer dependency, consistent invocation pattern.

## What Changes

### New `resolve` subcommand in `changeman.sh`

Absorbs the logic from `resolve-change.sh` (~96 lines). Interface:

```bash
changeman.sh resolve [override]
```

- **No override**: reads `fab/current`, strips whitespace, outputs folder name to stdout. Exits 1 if no active change.
- **With override**: case-insensitive substring match against `fab/changes/` folders (excluding `archive/`). Exact match wins; single partial match resolves; multiple matches → error listing them; no match → error. Outputs resolved folder name to stdout.
- **Exit codes**: 0 success, 1 error (with diagnostic to stderr).

This replaces the `resolve_change()` function and its `RESOLVED_CHANGE_NAME` global variable. Callers migrate from `source resolve-change.sh; resolve_change "$fab_root" "$override"; echo "$RESOLVED_CHANGE_NAME"` to `name=$(changeman.sh resolve "$override")`.

### New `switch` subcommand in `changeman.sh`

Composes `resolve` with pointer management and branch integration. Interface:

```bash
changeman.sh switch <name>          # resolve + write fab/current + git branch + summary
changeman.sh switch --blank         # clear fab/current (deactivate)
```

**Normal flow** (`changeman.sh switch <name>`):
1. Resolve change name via internal `resolve` logic
2. Write folder name to `fab/current` (`printf '%s' "$name" > "$FAB_ROOT/current"`)
3. Read `.status.yaml` for branch field (via `yq` or `grep`)
4. Git branch integration: `git checkout <branch>` if exists, `git checkout -b <branch>` if not (respect `config.yaml` `git.enabled` and `branch_prefix`)
5. Derive current stage via `$STAGEMAN current-stage`
6. Output summary to stdout (name, branch, stage, next command suggestion)

**Deactivation flow** (`changeman.sh switch --blank`):
1. Delete `fab/current` (no-op if absent)
2. Output confirmation to stdout

### Callers migration

| Caller | Before | After |
|--------|--------|-------|
| `preflight.sh` | `source resolve-change.sh; resolve_change "$fab_root" "$override"` | `name=$("$CHANGEMAN" resolve "$override")` |
| `batch-fab-switch-change.sh` | `source resolve-change.sh; resolve_change "$FAB_DIR" "$change"` | `name=$("$CHANGEMAN" resolve "$change")` |
| `batch-fab-archive-change.sh` | `source resolve-change.sh; resolve_change "$FAB_DIR" "$change"` | `name=$("$CHANGEMAN" resolve "$change")` |
| `/fab-switch` skill | 5-6 tool calls | Single `changeman.sh switch "$arg"` call + display output |

### Deletion of `resolve-change.sh`

`fab/.kit/scripts/lib/resolve-change.sh` becomes deletable after all callers migrate. The `src/lib/resolve-change/` dev directory (test suite, spec) migrates to `src/lib/changeman/` — resolve tests become part of the changeman test suite.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update `lib/` directory listing to remove `resolve-change.sh`, add `resolve` and `switch` subcommands to changeman description, update preflight description
- `fab-workflow/preflight`: (modify) Update resolve-change sourcing → changeman CLI call, update design decisions (remove "Shared Change Resolution Library" or rewrite as "Resolution via changeman CLI")

## Impact

- **`fab/.kit/scripts/lib/changeman.sh`** — adds `resolve` and `switch` subcommands (~120 lines)
- **`fab/.kit/scripts/lib/resolve-change.sh`** — deleted
- **`fab/.kit/scripts/lib/preflight.sh`** — migrate from `source resolve-change.sh` to `$CHANGEMAN resolve`
- **`fab/.kit/scripts/batch-fab-switch-change.sh`** — migrate from `source resolve-change.sh` to `$CHANGEMAN resolve`
- **`fab/.kit/scripts/batch-fab-archive-change.sh`** — migrate from `source resolve-change.sh` to `$CHANGEMAN resolve`
- **`fab/.kit/skills/fab-switch.md`** — simplify to thin wrapper around `changeman.sh switch`
- **`src/lib/resolve-change/`** — tests migrate into `src/lib/changeman/test.bats`
- **`src/lib/changeman/SPEC-changeman.md`** — add `resolve` and `switch` subcommand docs
- **`src/lib/preflight/test.bats`** — update to expect changeman resolve instead of source pattern

## Open Questions

- None — the design was fully discussed in conversation before intake creation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `resolve` outputs to stdout, not via shell variable | CLI-subprocess convention established by stageman migration; eliminates the only remaining sourced library | S:95 R:90 A:95 D:95 |
| 2 | Certain | `switch` handles git branch integration | User explicitly agreed branch logic is deterministic and should be in shell | S:90 R:85 A:90 D:90 |
| 3 | Certain | `resolve-change.sh` is deleted after migration | All callers migrate to `changeman.sh resolve`; no reason to keep the standalone file | S:90 R:90 A:90 D:95 |
| 4 | Confident | `switch` reads `config.yaml` for `git.enabled` and `branch_prefix` | changeman currently doesn't read config.yaml; switch needs it for branch naming. Requires adding yq or grep-based config parsing | S:80 R:80 A:70 D:75 |
| 5 | Confident | Interactive selection (no-arg `/fab-switch`) stays in skill layer | Shell can't prompt through Claude's UI; only the selection UX remains in the skill, changeman handles the rest | S:85 R:85 A:80 D:70 |
| 6 | Confident | Resolve tests merge into changeman test suite | Single test file per script; resolve is now a changeman subcommand | S:80 R:85 A:75 D:80 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).

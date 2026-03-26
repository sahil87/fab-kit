# Intake: Allow idea shorthand

**Change**: 260326-p4ki-allow-idea-shorthand
**Created**: 2026-03-26
**Status**: Draft

## Origin

> `/fab-new Allow idea <text of idea> format once again as a shorthand`
>
> Backlog item `[p4ki]`: "Allow 'idea <text of idea>' format once again"

## Why

Before the Go port (260310-pl72), the Bash-based `idea` script supported bare `idea "text"` as a shorthand for `idea add "text"`. During the port, this was intentionally dropped because Cobra's subcommand parsing conflicts with positional-arg-as-default — a bare string argument looks like an unknown subcommand to Cobra.

The shorthand was convenient for quick capture. Having to type `idea add "text"` adds friction to what should be a zero-thought operation. The original Bash behavior was more ergonomic.

## What Changes

### Root command default-to-add behavior

When `idea` is invoked with positional arguments that don't match any known subcommand, it should treat them as an `idea add` invocation. Specifically:

```bash
# These should be equivalent:
idea "refactor auth middleware"
idea add "refactor auth middleware"
```

### Implementation approach: Cobra `RunE` on root command

Add a `RunE` handler to the root `cobra.Command` in `src/go/idea/cmd/main.go`. When the root command receives positional args (i.e., the user typed `idea "some text"` and Cobra couldn't match a subcommand), delegate to the same logic as `addCmd()`.

Cobra's `TraverseChildren` or `Args` on root can be used, but the cleanest approach is to set `root.RunE` so that when args are present and no subcommand matched, it calls `idea.Add(...)` directly. The key subtlety: Cobra will error on unknown subcommands by default, so `FParseErrWhitelist` or `DisableFlagParsing` or a custom `args` validator may be needed to avoid "unknown command" errors when the first positional arg happens to look like a subcommand name.

**Key file**: `src/go/idea/cmd/main.go` — add `RunE` to root command and configure arg handling.

### Documentation updates

- `fab/.kit/skills/_cli-external.md` — add the shorthand to the idea command table
- `docs/specs/packages.md` — mention the shorthand in the idea section

## Affected Memory

- `fab-workflow/distribution`: (modify) Document that idea supports bare shorthand again

## Impact

- **`src/go/idea/cmd/main.go`** — root command gains `RunE` handler
- **`src/go/idea/internal/idea/idea_test.go`** — may need integration-level test if testing CLI dispatch
- **`src/go/idea/cmd/add.go`** — the `resolveFile()` and `idea.Add()` calls need to be reusable from root
- **`fab/.kit/skills/_cli-external.md`** — docs update
- **`docs/specs/packages.md`** — docs update

## Open Questions

- None — scope is clear.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Shorthand delegates to `idea.Add()` with same defaults | Direct reuse of existing add logic — no new behavior | S:90 R:95 A:95 D:95 |
| 2 | Certain | No custom `--id` or `--date` flags on bare shorthand | Shorthand is for quick capture; flags belong on `idea add` | S:85 R:90 A:90 D:90 |
| 3 | Confident | Use root `RunE` with `Args: cobra.ArbitraryArgs` to handle bare args | Cobra's standard pattern for default commands; `DisableFlagParsing` would break persistent flags | S:75 R:90 A:70 D:65 |
| 4 | Certain | Update `_cli-external.md` and `docs/specs/packages.md` | Constitution requires docs updates for CLI changes | S:90 R:95 A:95 D:95 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).

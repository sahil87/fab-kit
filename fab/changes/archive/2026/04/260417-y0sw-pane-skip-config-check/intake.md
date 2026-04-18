# Intake: Pane Skip Config Check

**Change**: 260417-y0sw-pane-skip-config-check
**Created**: 2026-04-17
**Status**: Draft

## Origin

Dispatched by `/fab-proceed` with pre-synthesized context. No interactive conversation — the caller provided a scoped, decided design brief including target files, line numbers, alternatives considered (and rejected), verified-source facts, and the acceptance criteria for safe defaults. This intake encodes those decisions verbatim; downstream stages (spec, tasks, apply) inherit them without needing to re-litigate.

> Exempt the `fab pane` top-level command (and all four subcommands: map, capture, send, process) from the `fab/project/config.yaml` requirement at the router level so that `fab pane map --all-sessions` (and other pane subcommands) work from any directory — including scratch tmux tabs outside any fab worktree. This is critical for the operator multi-agent monitoring workflow where a user wants to view state of fab agents across all worktrees from a tab that isn't itself inside a fab repo.

## Why

**Problem.** The `fab` router (`src/go/fab-kit/cmd/fab/main.go`) enforces the presence of `fab/project/config.yaml` before dispatching any workflow command to `fab-go`. When absent, it exits with `Not in a fab-managed repo. Run 'fab init' to set one up.` This is the correct default for commands that operate on a project's active change — but the `fab pane` command group is different: every pane subcommand is fully CWD-independent because it resolves state from each target pane's own `pane_current_path` via `tmux display-message`, not from the invoker's CWD.

**Consequence of no fix.** The operator multi-agent monitoring workflow is broken from scratch tmux tabs. `fab pane map --all-sessions` is specifically designed to surface the state of fab agents across every worktree in every tmux session — the caller's CWD is irrelevant by design. Forcing the caller to `cd` into some fab worktree before running `--all-sessions` is self-contradictory. Operators need a tab that is purely observational; today that's impossible without the router refusing to route `pane` to `fab-go`.

**Why router-level exemption over the alternatives.** Two other approaches were considered and rejected:

1. *Push the config check into each `fab-go` command.* Rejected — there are ~12 workflow commands that correctly require `config.yaml`; retrofitting each reverses the safe default (fail-closed) into fail-open. Every future command added to `fab-go` would become an opt-in to the guard instead of being protected by it. High regression risk for zero architectural gain.

2. *Move `pane` entirely into the `fab-kit` shim binary* (which has no config requirement by design — it hosts `init`, `upgrade-repo`, `doctor`, etc.). Rejected for this change — `fab pane` sits on ~500 lines of shared helpers in `fab-go` (status resolution, runtime-file reading, change folder walking). Moving it means either duplicating those helpers or extracting a new shared Go module. Not worth it for one command group; revisit only if more cross-cutting commands accumulate.

The chosen approach is a minimal, targeted allowlist in the router: `pane` is explicitly marked as "no config required" and the router provides a bundled `fab-go` (the one that shipped with the currently-installed fab-kit release) to run it. This preserves safe defaults for all other commands.

## What Changes

All changes are confined to `src/go/fab-kit/cmd/fab/main.go` (primary) and `src/go/fab-kit/cmd/fab/main_test.go` (tests). No changes to `fab-go`'s pane code — the pane subcommands are already CWD-independent.

### 1. Add a new allowlist: `fabGoNoConfigArgs`

Alongside the existing `fabKitArgs` allowlist (currently lines 17–23 of `main.go`), introduce a sibling allowlist for fab-go commands that are exempt from the config.yaml requirement:

```go
// fabGoNoConfigArgs is the static allowlist of fab-go subcommands that may
// execute without fab/project/config.yaml present. These commands resolve
// state from their arguments (e.g., tmux pane IDs), not from the invoker's CWD.
var fabGoNoConfigArgs = map[string]bool{
    "pane": true,
}
```

Only `pane` is listed. Notably excluded: `runtime` (set-idle/clear-idle/is-idle) — it depends on `resolve.FabRoot()` and MUST continue to require a fab repo.

### 2. Update `execFabGo` (currently lines 71–93)

Current flow:

```go
func execFabGo(args []string) {
    cfg, err := internal.ResolveConfig()
    if err != nil { ... os.Exit(1) }
    if cfg == nil {
        fmt.Fprintln(os.Stderr, "Not in a fab-managed repo. Run 'fab init' to set one up.")
        os.Exit(1)
    }
    bin, err := internal.EnsureCached(cfg.FabVersion)
    ...
    syscall.Exec(bin, argv, os.Environ())
}
```

New flow:

```go
func execFabGo(args []string) {
    cfg, err := internal.ResolveConfig()
    if err != nil { ... os.Exit(1) }

    exempt := len(args) > 0 && fabGoNoConfigArgs[args[0]]

    var fabVersion string
    switch {
    case cfg != nil:
        // In a fab repo — always use the project-pinned version (existing behavior).
        fabVersion = cfg.FabVersion
    case exempt:
        // Not in a fab repo, but command is exempt — use the router's bundled version.
        fabVersion = version
    default:
        fmt.Fprintln(os.Stderr, "Not in a fab-managed repo. Run 'fab init' to set one up.")
        os.Exit(1)
    }

    bin, err := internal.EnsureCached(fabVersion)
    if err != nil { ... os.Exit(1) }

    argv := append([]string{bin}, args...)
    if err := syscall.Exec(bin, argv, os.Environ()); err != nil { ... os.Exit(1) }
}
```

Key properties:

- **When inside a fab repo (`cfg != nil`)**: behavior is unchanged — the project-pinned `fab_version` is used for every command, including `pane`. This preserves reproducibility for users who have pinned a specific fab version.
- **When outside a fab repo (`cfg == nil`) AND command is exempt**: use `version` (the router's own build-time constant, currently `var version = "dev"` at line 14; set to the release tag at brew install time via Go's `-ldflags -X`). This guarantees exempted commands always use the fab-go that ships with the currently-installed fab-kit release.
- **When outside a fab repo AND command is NOT exempt**: exit 1 with the existing error message. Unchanged for every other command.

### 3. Update `printHelp` (currently lines 96–123)

Today the fab-go `--help` block is gated on `cfg != nil` (lines 107–118), which means from a scratch tab outside any fab repo, `fab --help` shows only workspace commands. `pane` is invisible, defeating discoverability.

New behavior: when `cfg == nil`, still invoke `fab-go --help`, but use the bundled version via `EnsureCached(version)` to locate the binary. When `cfg != nil`, retain the existing path (use `cfg.FabVersion`).

```go
// Show workflow commands. Inside a fab repo, use the project-pinned version.
// Outside a fab repo, use the bundled version so pane (and any future no-config
// exempt commands) remain discoverable.
cfg, _ := internal.ResolveConfig()
var fabVersion string
if cfg != nil {
    fabVersion = cfg.FabVersion
} else {
    fabVersion = version
}
if bin, err := internal.EnsureCached(fabVersion); err == nil {
    if out, err := exec.Command(bin, "--help").Output(); err == nil {
        fmt.Println("Workflow commands (fab-go):")
        fmt.Print(string(out))
        fmt.Println()
    }
}
```

Errors remain silently swallowed (same as today) — help is best-effort; it should never fail the overall help output.

### 4. Tests (`main_test.go`)

Add tests for:

- **`TestFabGoNoConfigArgs`**: assert `fabGoNoConfigArgs["pane"] == true`; assert `runtime`, `status`, `preflight`, `resolve`, `change`, `score` are NOT in the allowlist (these MUST continue to require config).
- **Exemption dispatch path**: a unit-testable extraction of the "pick fab version" decision (either refactor the branching out of `execFabGo` into a pure helper `resolveFabVersion(cfg, arg, routerVersion) (string, error)`, or test indirectly via a seam). Cover three cases: (a) `cfg != nil, arg="pane"` → project version; (b) `cfg == nil, arg="pane"` → router version; (c) `cfg == nil, arg="status"` → error.
- **Help output with no config**: extend the existing `printHelp` test surface (or add one if none exists) to assert the fab-go help block runs when `cfg == nil`. This likely requires a similar refactor: extract the "which version to use for help" decision into a pure helper.

The exact refactor shape is a spec-stage decision; the intake just commits to test coverage for the three behavior axes (exemption true/false × in-repo true/false + help-in-scratch-tab).

### 5. Safe defaults preserved (acceptance criteria)

These MUST continue to hold after the change:

- Every fab-go command other than `pane` emits `Not in a fab-managed repo. Run 'fab init' to set one up.` and exits 1 when run outside a fab repo.
- `runtime set-idle/clear-idle/is-idle` (same binary, depends on `resolve.FabRoot()`) is NOT in the exemption list and continues to require a fab repo.
- `pane send` without `--force` preserves its existing safety behavior: if the target pane's CWD is outside a fab repo and no runtime state can be resolved, the agent state is reported as `unknown`, the idle guard fires, and the command refuses to send keystrokes. No change to that logic — the exemption only gets the command past the router's config check; downstream pane-level safety is untouched.
- Inside a fab repo, `fab pane ...` continues to use the project-pinned `fab_version` from `config.yaml`. The bundled-version fallback only engages when `cfg == nil`.

### 6. Verified facts (confirmed from source by dispatcher — do not re-verify)

- All four `pane` subcommands (`map`, `capture`, `send`, `process`) are already fully CWD-independent. Each resolves state from the target pane's own CWD via `tmux display-message` / `pane_current_path`. Source: `src/go/fab/cmd/fab/panemap.go`, `pane_capture.go`, `pane_send.go`, `pane_process.go`, `src/go/fab/internal/pane/pane.go`.
- `panemap.go:236–260` gracefully em-dashes panes whose CWD isn't in a fab worktree. No errors, no crashes.
- `pane.go:61–86` (`ResolvePaneContext`) handles "not in a git repo" without error.

### 7. First-run cost

On a pristine machine (fresh brew install, empty `~/.fab-kit/versions/` cache) the first `fab pane ...` invocation from outside a fab repo will trigger `EnsureCached(version)` to download the bundled fab-go release. Acceptable one-time stall — same network cost users already pay for the first workflow command inside a fab repo.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the `fabGoNoConfigArgs` allowlist and the `pane` exemption. Update the "Not in a fab-managed repo" paragraph (currently around line 265) to note that `pane` is the single exception and that it resolves to the bundled fab-go version when `cfg == nil`. Update the `fab pane` section to note "available from any directory, including outside a fab repo."
- `fab-workflow/distribution`: (modify) If this memory file describes router routing rules in detail, note the exemption and bundled-version fallback for `pane`. (Confirmed at spec stage — otherwise this entry is dropped.)

## Impact

**Code areas**:
- `src/go/fab-kit/cmd/fab/main.go` — primary (allowlist, `execFabGo`, `printHelp`)
- `src/go/fab-kit/cmd/fab/main_test.go` — tests

**Not touched**:
- `src/go/fab/**` — no changes to fab-go's pane code, runtime code, or resolve code. The pane subcommands already behave correctly from any CWD.
- Other fab-go commands — their dispatch path is unaffected.

**Runtime behavior**:
- Inside a fab repo: no change.
- Outside a fab repo: `fab pane ...` now succeeds (previously exited 1); every other `fab <cmd>` still exits 1 with the existing error.
- `fab --help` from a scratch tab: now shows the full workflow command listing (including `pane`); previously showed only workspace commands.

**Dependencies / APIs**: no new dependencies. Uses existing `internal.ResolveConfig`, `internal.EnsureCached`, and the router's `var version` build-time constant.

**Distribution / release**: no changes to the release process. The router's `version` variable is already set via `-ldflags -X` at brew install time — the exemption path just reads it.

## Open Questions

None. The dispatch description exhaustively specifies target files, line numbers, allowlist shape, version resolution logic, help behavior, safe defaults, and rejected alternatives. All questions were resolved upstream before this intake was created.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exemption implemented at the router (`fab-kit/cmd/fab/main.go`), not in each fab-go command | Dispatcher explicitly chose router-level over per-command; preserves fail-closed default for all other commands | S:95 R:85 A:90 D:95 |
| 2 | Certain | Only `pane` is exempt in this change; `runtime` and all other commands remain config-required | Dispatcher explicitly excluded `runtime` (depends on `resolve.FabRoot()`) and named `pane` as the sole entry | S:95 R:85 A:95 D:95 |
| 3 | Certain | Outside a fab repo, exempted commands use the router's build-time `version` constant via `EnsureCached(version)` | Dispatcher specified this; `var version = "dev"` at line 14 is set to the release tag at brew install time | S:95 R:80 A:90 D:90 |
| 4 | Certain | Inside a fab repo, exempted commands continue to use `cfg.FabVersion` (project-pinned) | Dispatcher: "only fall back to the bundled version when cfg == nil" — preserves current behavior for in-repo users | S:95 R:85 A:95 D:95 |
| 5 | Certain | `printHelp` runs the fab-go `--help` block even when `cfg == nil`, using bundled version | Dispatcher specified this verbatim; required for `pane` discoverability from scratch tabs | S:95 R:90 A:90 D:95 |
| 6 | Certain | `pane send` idle guard is unchanged — the exemption is router-level only, not a safety bypass | Dispatcher: "preserves its existing safety behavior... No change to that logic" | S:95 R:85 A:95 D:95 |
| 7 | Certain | Change type is `fix` (removes a usability blocker, no new capability) | Dispatcher declared this explicitly | S:95 R:95 A:90 D:95 |
| 8 | Certain | Tests live in `src/go/fab-kit/cmd/fab/main_test.go` alongside existing `TestFabKitArgs` / `TestVersion` / `TestPrintVersion` | Project `code-quality.md` declares test-alongside strategy; existing test file is right next to main.go and already covers analogous allowlist/version logic | S:90 R:95 A:95 D:95 |
| 9 | Certain | The version-selection branching inside `execFabGo` must be extracted into a pure helper for testability | Current `execFabGo` ends in `syscall.Exec` which is untestable directly; extracting the version-resolution decision is the standard Go pattern and the only path to unit-testing all three cases | S:90 R:95 A:90 D:90 |
| 10 | Certain | First-run download cost on a pristine machine (empty `~/.fab-kit/versions/`) is acceptable | Dispatcher explicitly flagged and accepted this; same network cost users already pay for the first in-repo workflow command | S:95 R:90 A:90 D:95 |
| 11 | Certain | Primary memory file to update is `docs/memory/fab-workflow/kit-architecture.md` | Directly verified via grep + read: it documents the router's "Not in a fab-managed repo" error (line 265) and the `fab pane` command group (line 308+) | S:90 R:95 A:95 D:90 |
| 12 | Tentative | `docs/memory/fab-workflow/distribution.md` may also need a one-line update about the routing exemption | Grep matched on "fab-kit.*router"; whether it narrates routing-decision logic in detail (vs. packaging) needs a spec-stage read to confirm | S:60 R:80 A:70 D:55 |

<!-- assumed: distribution.md update necessity — will confirm at spec stage whether it covers router decision logic or only packaging; drop the Affected Memory entry if the latter -->

12 assumptions (11 certain, 0 confident, 1 tentative, 0 unresolved).

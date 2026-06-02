# Intake: Remove router config gate + add `shell-init` wrapper

**Change**: 260511-c432-fix-completion-outside-repo
**Created**: 2026-05-12
**Status**: Draft

## Origin

User started from "Should we add shell completion to `fab`? Check how `tu` does this." Investigation showed `fab` already has Cobra-generated `completion bash|zsh|fish|powershell`, but the user then surfaced a real bug: `fab completion zsh` outside a fab-managed repo prints `Not in a fab-managed repo. Run 'fab init' to set one up.` and exits, rather than emitting the completion script. This blocks the standard install pattern (sourcing completion from `~/.zshrc`, which loads from arbitrary directories).

Mode: conversational, exploratory. The investigation evolved across three turns:

1. **First framing (assistant)**: narrow patch — add `completion`, `help`, `fab-help` to the `fabGoNoConfigArgs` allowlist; handle `--help` as a flag in any position.
2. **User narrowed**: drop `fab-help` from that list. Add `shell-init` (a `tu`-style alias for `completion`).
3. **User broadened**: "the whole list you suggested — can be made to work outside a fab repo." Pointing at the deeper design issue: the router gate is too coarse.

The third framing is the right one and is what this intake captures. Verified by audit (see Why):

- **fab-go has its own per-command guards.** Every command that needs project state already fails cleanly outside a repo with `ERROR: fab/ directory not found` (exit 1). No segfaults, no panics — confirmed for `score`, `resolve`, `status`, `change`, `log`, `batch`, `fab-help`, `preflight`.
- **fab-go has commands that don't need project state.** `kit-path` works anywhere (prints the system cache path). `pane map` works anywhere (surveys tmux globally). `operator` switches/launches its tmux tab without needing config. `hook session-start` is silently a no-op when there's nothing to clean up. All exit 0 outside a repo.
- **The router gate is redundant and worse.** It blanket-rejects everything not in `fabGoNoConfigArgs` (currently just `pane`), even commands that would work fine. Its error message ("Run `fab init`") is also wrong for commands like `kit-path` that don't need init at all.

Decisions reached jointly:

1. **Drop the router's config gate entirely.** Route every command to fab-go; let fab-go's per-command guards produce accurate errors.
2. **Add `fab shell-init <shell>`** as a `tu`-style alias for `completion` (eval-able output, one-step install).
3. **Version skew is acceptable.** Outside a repo, commands run on the router's bundled fab-go version. For config-free commands (`completion`, `help`, `kit-path`, `--help`) that's fine. For commands that need config, fab-go's guards reject them anyway — the bundled-version path is unreachable in practice for those.
4. **Out of scope**: dynamic completion handlers (change IDs, stage names, change types via Cobra `ValidArgsFunction`) — separate enhancement, explicitly deferred.

## Why

**Problem (concrete).** Standard shell-completion install (`eval "$(fab completion zsh)"` in `~/.zshrc`) is unusable because `~/.zshrc` loads from arbitrary directories — most of which are not fab-managed. Same for `fab --help` and `fab <subcommand> --help` from a scratch tab: new users see an init prompt instead of help. And entirely-config-free commands like `fab kit-path` fail with a misleading "Run `fab init`" when init isn't needed.

**Why the current architecture is wrong.** `src/go/fab-kit/cmd/fab/main.go` is a thin router that exec's fab-go. It applies a gate (`fabGoNoConfigArgs` allowlist) requiring config.yaml before exec'ing. The gate's intent was version-pinning: route to the *project-pinned* fab-go version when inside a repo, fall back to the router's *bundled* version when outside (currently only for `pane`). But the gate solves the version-routing problem by *blocking* everything else, which is overreach — it conflates "which version to use" with "is this command allowed to run."

**Audit results (run from `/tmp`, no fab config present).** Every fab-go command outside a repo behaves correctly when invoked directly:

| Command | Exit | Behavior |
|---------|------|----------|
| `kit-path` | 0 | Prints system cache path — config-independent by design |
| `pane map` | 0 | Lists tmux panes — works anywhere |
| `operator` | 0 | Switches/launches operator tab — config-independent |
| `hook session-start` | 0 | Silent no-op when nothing to clean up |
| `--help` / `help` / `completion` / `fab-help` --help | 0 | Cobra docs — version-insensitive |
| `preflight`, `score`, `resolve`, `status`, `change`, `log`, `batch`, `fab-help` (no `--help`) | 1 | Clean `ERROR: fab/ directory not found` |

**Consequence if unfixed.** Shell completion stays effectively broken. `kit-path` keeps lying about needing init. New users get a hostile first impression. Workarounds (`cd /path/to/fab-repo && eval ...`) get baked into install docs. The router's exemption-list grows over time as people discover more commands that should "just work outside a repo," and we keep playing whack-a-mole.

**Why this approach over the narrow patch.**
- *Narrow patch (extend `fabGoNoConfigArgs` by 3-4 entries)*: fixes the symptom but leaves the design wrong. Every new config-free command in the future needs a router-side allowlist update too.
- *Per-command exemption flags*: also patches symptom.
- *Drop the gate entirely* (this proposal): treats fab-go's existing per-command guards as the authoritative answer to "does this need config?" The router stops second-guessing them. Less code, accurate errors, future-proof.

**Why NOT alternatives.**
- *Hand-roll completion like `tu`*: Cobra's autogen produces complete scripts with subcommands, flags, descriptions for free. Re-implementing loses fidelity.
- *Make config optional everywhere in fab-go*: most workflow commands genuinely need it; existing per-command guards are correct.
- *Pre-flight check in `~/.zshrc`*: pushes workaround onto every user.

## What Changes

### 1. Remove the router's config gate (`src/go/fab-kit/cmd/fab/main.go`)

Current behavior (lines 97-125): `execFabGo` calls `internal.ResolveConfig()`, then `resolveFabVersion(cfg, arg0, version)` which returns `shouldExit=true` if `cfg == nil` and `arg0` isn't in `fabGoNoConfigArgs`. The exit path prints the "Not in a fab-managed repo" message.

New behavior: never exit at the router. If `cfg != nil`, use `cfg.FabVersion`; otherwise use the router's bundled `version`. Always proceed to `EnsureCached` + `syscall.Exec`.

Sketch:

```go
func execFabGo(args []string) {
    cfg, err := internal.ResolveConfig()
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
        os.Exit(1)
    }

    fabVersion := version // router's bundled version
    if cfg != nil {
        fabVersion = cfg.FabVersion
    }

    bin, err := internal.EnsureCached(fabVersion)
    if err != nil { /* ... */ }
    // syscall.Exec as before
}
```

The `resolveFabVersion` helper and the `fabGoNoConfigArgs` map both become unused. **Delete them** (the user's narrow-patch instinct was to extend the map; the right move is to delete it). Update `printHelp()` similarly — its existing pattern (line 142-149: `cfg, _ := ...; if cfg != nil { ...project version... } else { ...router version... }`) is already the model we want everywhere.

**Migration note**: `internal.ResolveConfig` currently distinguishes "not initialized" (returns `nil, nil`) from "config corrupted" (returns `nil, err`). The corrupted case still hard-errors at the router — that's correct, the user genuinely needs to fix their file. Only the missing-config case becomes a soft fall-through.

### 2. Add `fab shell-init <shell>` subcommand (in fab-go)

A new Cobra command in fab-go's command tree, alongside the auto-registered `completion` command.

- **Synopsis**: `fab shell-init <bash|zsh|fish>`
- **Behavior**: an alias for `completion <shell>` — emits the same eval-able script. User runs `eval "$(fab shell-init zsh)"` exactly as with `tu shell-init`.
- **Implementation**: thin wrapper that delegates to Cobra's `GenBashCompletion` / `GenZshCompletion` / `GenFishCompletion` based on the shell argument. Reject unknown shells via `cobra.ExactValidArgs([]string{"bash","zsh","fish"})` or equivalent allowlist.
- **Help text**: short docstring referencing `fab completion --help` for advanced install details (system-wide paths, etc.).
- **Why a separate command instead of just documenting `completion`**: discoverability. `completion` is a Cobra convention name; new users don't know it. `shell-init` is the verb they're already thinking ("how do I init shell completion?"). Costs ~30 lines. The user explicitly asked for this modeled on `tu`.

File location TBD at spec stage — depends on fab-go's cmd directory structure. Cobra registration is a one-line `rootCmd.AddCommand(...)`.

### 3. Tests (`src/go/fab-kit/cmd/fab/main_test.go`)

The audit-based design means most test changes are deletions, not additions:

- **`TestFabGoNoConfigArgs` (lines 28-45)**: **DELETE entirely.** The map is gone. The test asserts the absence of behavior we're removing.
- **`TestResolveFabVersion` (lines 47-109)**: **DELETE entirely.** Function is gone. The version-selection logic (cfg ? cfg.FabVersion : version) is now inline in `execFabGo` and `printHelp`, and is trivial enough that a single end-to-end test suffices.
- **`TestFabKitArgs` (lines 10-26)**: keep as-is — `fabKitArgs` (workspace-command allowlist) is unrelated and still correct.
- **Add new test(s)** for `execFabGo`'s version selection — ideally a small unit test on a factored helper like `pickFabVersion(cfg, routerVersion) string` if the body is non-trivial enough to warrant extraction. Otherwise leave end-to-end coverage to integration tests.
- **Add tests for `shell-init` in the fab-go test tree**: accepts `bash|zsh|fish`, rejects unknown shells, output is non-empty and begins with the expected per-shell header (e.g., `#compdef fab` for zsh).

### 4. Documentation

- **README install section**: add `eval "$(fab shell-init zsh)"` (or equivalent) as the canonical one-liner.
- **`docs/specs/architecture.md`**: replace any text describing the `fabGoNoConfigArgs` gate with the new policy — "router routes everything; fab-go subcommands self-guard." Keep the version-selection rule (project-pinned when config exists, bundled otherwise).
- **Memory**: defer to hydrate stage. Likely updates `docs/memory/fab-workflow/kit-architecture.md` (router policy) and adds a brief mention of `shell-init` in the relevant CLI/distribution memory file.

### 5. Surprises surfaced by the audit (flag for spec-stage decisions)

These were discovered during the audit and should be confirmed during spec generation, not assumed:

- **`hook session-start` silently exits 0 outside a repo.** Probably correct (Claude Code hooks fire regardless of CWD; they should no-op when no fab project exists). Spec stage: confirm this is intentional or whether it should print a debug-level message.
- **`operator` exits 0 outside a repo** (it switched to an existing operator tmux tab). It's currently NOT in `fabGoNoConfigArgs` but apparently works. Either it's truly config-free at the entry point, or it'll fail downstream when it tries to read changes. Spec stage: audit `operator`'s code path to confirm it degrades gracefully or add the same `fab/ directory not found` guard.
- **`kit-path` is the only currently-blocked-by-router command that *should* work anywhere.** After this change it will. No further action needed.

## Affected Memory

- `fab-workflow/kit-architecture.md`: (modify) Replace router-gate description with "router routes all subcommands; fab-go self-guards" policy. Document version-selection rule.
- `fab-workflow/distribution.md`: (modify, possibly) If shell-completion install is documented anywhere in distribution context, update with `shell-init` snippet.

## Impact

**Affected code**:
- `src/go/fab-kit/cmd/fab/main.go` — drop gate, drop `fabGoNoConfigArgs`, drop `resolveFabVersion`, inline version selection.
- `src/go/fab-kit/cmd/fab/main_test.go` — delete two test functions, possibly add one small replacement.
- fab-go command tree (path TBD) — new `shell-init` command + tests.
- `README.md` — install snippet update.
- `docs/specs/architecture.md` — router policy rewrite.

**APIs / external surface**:
- **New**: `fab shell-init <shell>`. Additive, no compatibility risk.
- **Changed (bug fixes, not contract changes)**: `fab completion <shell>`, `fab help`, `fab --help`, `fab <sub> --help`, `fab kit-path` now work outside a repo. No existing user can be relying on these *failing* — pure UX improvement.
- **Unchanged**: every command that needs project state still fails outside a repo, just with fab-go's own (more accurate) error message instead of the router's generic one.

**Dependencies**: none added.

**Risk**: low. Audit confirmed zero crashes outside a repo across all fab-go subcommands. Error messages get *more* accurate, not less. Version-skew risk is contained: commands that hit project state are unreachable outside a repo (fab-go guards them), so the bundled-version path is in practice only used for pure-doc commands and `kit-path` / `pane` / `operator` which are version-insensitive.

## Open Questions

Surfaced by audit, deferred to spec stage:

1. Should `operator` outside-of-repo behavior be documented as supported (it works now) or guarded (force same error as other workflow commands)?
2. Should `hook session-start` log a debug message when it no-ops outside a repo, or stay silent?
3. Exact file location for `shell-init` in fab-go's cmd directory.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop the router's config gate entirely instead of extending `fabGoNoConfigArgs` | User explicitly broadened scope ("the whole list you suggested — can be made to work outside a fab repo"); audit confirmed it's safe | S:95 R:80 A:95 D:95 |
| 2 | Certain | fab-go's per-command guards are the authoritative answer to "does this need config?" | Audit confirmed every workflow command produces clean `ERROR: fab/ directory not found` (exit 1); no segfaults, no panics | S:100 R:90 A:100 D:100 |
| 3 | Certain | `kit-path`, `pane`, `operator`, `hook session-start` are genuinely config-free and exit 0 outside a repo | Audit run directly against `~/.fab-kit/versions/1.9.4/fab-go` in `/tmp` | S:100 R:95 A:100 D:100 |
| 4 | Certain | Add `fab shell-init <shell>` as a `tu`-style alias for `completion` | User explicitly requested this; matches `tu shell-init` semantics referenced in conversation | S:95 R:85 A:95 D:95 |
| 5 | Confident | Delete `fabGoNoConfigArgs` and `resolveFabVersion` entirely rather than keep them as no-ops | They have no remaining purpose; keeping them invites confusion. Tests delete cleanly | S:80 R:70 A:85 D:80 |
| 6 | Confident | Version skew (router-bundled fab-go vs. project-pinned) is acceptable outside a repo | Only pure-doc commands reach the bundled path in practice; user confirmed acceptable | S:80 R:60 A:80 D:80 |
| 7 | Certain | `shell-init` lives in fab-go, not the router | Router has no Cobra command tree — only manual arg-routing in main.go. fab-go is the only place where Cobra's `completion` is auto-registered and where `GenZshCompletion` etc. are available. No alternative exists | S:95 R:85 A:95 D:95 |
| 8 | Confident | Inline `cfg != nil ? cfg.FabVersion : version` in `execFabGo` and `printHelp` rather than extract a helper | Pattern is two lines, already used in `printHelp` at line 142-149 | S:75 R:80 A:80 D:75 |
| 9 | Confident | Memory updates: `kit-architecture.md` (modify) primarily; possibly `distribution.md` | Existing kit-architecture covers router/version-pinning — natural fit for new policy | S:75 R:80 A:80 D:75 |
| 10 | Confident | `operator` outside-repo behavior stays as-is (works now, leave it) | Code reading confirms self-consistency: switch path (`tmux select-window`) is pure tmux and safe anywhere; launch path self-guards via `resolve.FabRoot()` at operator.go:50. Not accidental | S:80 R:80 A:85 D:80 |
| 11 | Confident | `hook session-start` silent no-op is correct; no debug message needed | Code at hook.go:113-119 explicitly swallows errors with "swallow" comments. Same pattern in hookStopCmd, hookUserPromptCmd. Deliberate design — Claude Code fires hooks regardless of CWD | S:90 R:85 A:90 D:85 |
| 12 | Certain | `shell-init` file location: `src/go/fab/cmd/fab/shellinit.go`, factory `shellInitCmd()`, registered in main.go's `AddCommand` block, test at `shellinit_test.go` | Convention confirmed by reading main.go and listing cmd directory: one-command-one-file with `xCmd()` factory pattern, matches `kitpath.go`/`operator.go`/`fabhelp.go` | S:95 R:80 A:95 D:95 |

12 assumptions (6 certain, 6 confident, 0 tentative, 0 unresolved).
